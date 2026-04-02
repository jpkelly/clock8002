package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
)

type secondDisplayCueVisual string

const (
	secondDisplayCueUnset secondDisplayCueVisual = "unset"
	secondDisplayCueOff   secondDisplayCueVisual = "off"
	secondDisplayCueRight secondDisplayCueVisual = "right"
	secondDisplayCueLeft  secondDisplayCueVisual = "left"
	secondDisplayCueBlank secondDisplayCueVisual = "blank"
)

var lastSecondDisplayCueVisual = secondDisplayCueUnset

var mirrorPixels []byte

const secondDisplayCueAssetVersion = "v2"

// probeSecondDisplayOutput applies runtime second-display policy for issue #23.
// It toggles fbcon bind state for live enable/disable behavior and logs HDMI-2 status.
func probeSecondDisplayOutput() {
	log.Printf("Info: second-display probe invoked (cue-second-display=%v)", options.CueSecondDisplay)

	if runtime.GOOS != "linux" {
		log.Printf("Warning: cue-second-display is only supported on linux; current OS is %s", runtime.GOOS)
		return
	}

	if options.CueSecondDisplay {
		destroySecondDisplayMirror()
		if err := setFramebufferConsoleBound(false); err != nil {
			log.Printf("Warning: could not disable framebuffer console binding: %v", err)
		} else {
			log.Printf("Info: framebuffer console unbound for second display icon mode")
		}
	} else {
		// Mirror mode: stop any running fbi, reset cue state, unbind fbcon, start DRM mirror.
		stopSecondDisplayImageProcesses()
		lastSecondDisplayCueVisual = secondDisplayCueUnset
		if err := setFramebufferConsoleBound(false); err != nil {
			log.Printf("Warning: could not disable framebuffer console binding for mirror mode: %v", err)
		} else {
			log.Printf("Info: framebuffer console unbound for second display mirror mode")
		}
		if !isSpareHDMIConnected() {
			log.Printf("Info: no spare HDMI connected, mirror mode unavailable")
			return
		}
		if err := initDRMMirror(); err != nil {
			log.Printf("Warning: DRM mirror init failed: %v", err)
		}
		return
	}

	statusPath := findHDMI2StatusPath()
	if statusPath == "" {
		log.Printf("Warning: cue-second-display enabled, but no HDMI-A-2 status path found under /sys/class/drm")
		return
	}

	statusBytes, err := os.ReadFile(statusPath)
	if err != nil {
		log.Printf("Warning: cue-second-display enabled, but unable to read %s: %v", statusPath, err)
		return
	}

	status := strings.TrimSpace(string(statusBytes))
	log.Printf("Info: cue-second-display probe: %s=%s", statusPath, status)
	if status != "connected" {
		log.Printf("Warning: cue-second-display enabled, but HDMI-A-2 is not connected")
		return
	}

	log.Printf("Info: HDMI-A-2 is connected and ready for second display rendering")
	showSecondDisplaySplashTest()
}

func findHDMI2StatusPath() string {
	const explicitPath = "/sys/class/drm/card1-HDMI-A-2/status"
	if _, err := os.Stat(explicitPath); err == nil {
		return explicitPath
	}

	matches, err := filepath.Glob("/sys/class/drm/*HDMI-A-2/status")
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func fbconBound() bool {
	entries := framebufferVtconEntries()
	if len(entries) == 0 {
		return false
	}

	for _, vtcon := range entries {
		bindBytes, err := os.ReadFile(filepath.Join(vtcon, "bind"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(bindBytes)) == "1" {
			return true
		}
	}

	return false
}

func framebufferVtconEntries() []string {
	entries, err := filepath.Glob("/sys/class/vtconsole/vtcon*")
	if err != nil || len(entries) == 0 {
		return nil
	}

	framebufferEntries := make([]string, 0, len(entries))

	for _, vtcon := range entries {
		nameBytes, err := os.ReadFile(filepath.Join(vtcon, "name"))
		if err != nil {
			continue
		}
		if !strings.Contains(strings.ToLower(string(nameBytes)), "frame buffer") {
			continue
		}
		framebufferEntries = append(framebufferEntries, vtcon)
	}

	return framebufferEntries
}

func setFramebufferConsoleBound(enable bool) error {
	entries := framebufferVtconEntries()
	if len(entries) == 0 {
		return nil
	}

	desired := "0"
	if enable {
		desired = "1"
	}

	for _, vtcon := range entries {
		bindPath := filepath.Join(vtcon, "bind")
		currentBytes, err := os.ReadFile(bindPath)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(currentBytes)) == desired {
			continue
		}

		if err := os.WriteFile(bindPath, []byte(desired), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func stopSecondDisplayImageProcesses() {
	pkillPath, err := exec.LookPath("pkill")
	if err != nil {
		return
	}

	cmd := exec.Command(pkillPath, "-x", "fbi")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return
		}
		log.Printf("Warning: failed to stop fbi while restoring console: %v (%s)", err, strings.TrimSpace(string(out)))
		return
	}

	log.Printf("Info: stopped fbi processes for second display icon mode")
}

func showSecondDisplaySplashTest() {
	fbiPath, err := exec.LookPath("fbi")
	if err != nil {
		log.Printf("Warning: unable to run second display splash test, fbi not found: %v", err)
		return
	}

	splashPath := firstExistingPath(
		"/opt/clock8002/bootsplash.png",
		"/opt/clock8002/splash/bootsplash.png",
	)
	if splashPath == "" {
		log.Printf("Warning: unable to run second display splash test, no splash image found")
		return
	}

	go func() {
		cmd := exec.Command(fbiPath, "-T", "1", "-d", "/dev/fb0", "-a", "-noverbose", "-1", "-t", "6", splashPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Warning: second display splash test failed: %v (%s)", err, strings.TrimSpace(string(out)))
			return
		}
		log.Printf("Info: second display splash test displayed from %s", splashPath)
	}()
}

func syncSecondDisplayCueDisplay(state *clock.State) {
	if runtime.GOOS != "linux" {
		return
	}
	if !options.CueSecondDisplay {
		return
	}

	desired := desiredSecondDisplayCueVisual(state)
	if desired == lastSecondDisplayCueVisual {
		return
	}

	// No cue received yet since icon mode started — keep the initial splash visible
	// rather than overwriting it with black. Once the first cue fires, normal
	// black-between-cues behaviour applies.
	if desired == secondDisplayCueOff && lastSecondDisplayCueVisual == secondDisplayCueUnset {
		return
	}

	assetPath, ok := secondDisplayCueAssetPath(desired)
	if !ok {
		log.Printf("Warning: unable to resolve second display cue asset for state=%s", desired)
		lastSecondDisplayCueVisual = desired
		return
	}

	fbiPath, err := exec.LookPath("fbi")
	if err != nil {
		log.Printf("Warning: unable to update second display cue image, fbi not found: %v", err)
		lastSecondDisplayCueVisual = desired
		return
	}

	cmd := exec.Command(fbiPath, "-T", "1", "-d", "/dev/fb0", "-a", "-noverbose", "-1", assetPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Warning: second display cue update failed (%s): %v (%s)", desired, err, strings.TrimSpace(string(out)))
		lastSecondDisplayCueVisual = desired
		return
	}

	log.Printf("Info: second display cue update: state=%s image=%s", desired, assetPath)
	lastSecondDisplayCueVisual = desired
}

func destroySecondDisplayMirror() {
	destroyDRMMirror()
	mirrorPixels = nil
}

func syncSecondDisplayMirrorDisplay() {
	if runtime.GOOS != "linux" || options.CueSecondDisplay || renderer == nil {
		return
	}
	if !isDRMMirrorActive() {
		return
	}

	w, h, err := renderer.GetOutputSize()
	if err != nil || w <= 0 || h <= 0 {
		return
	}

	srcPitch := int(w) * 4
	needed := int(h) * srcPitch
	if len(mirrorPixels) != needed {
		mirrorPixels = make([]byte, needed)
	}

	// ReadPixels as ARGB8888 (matches XRGB8888 dumb buffer — alpha byte ignored by display).
	if err := renderer.ReadPixels(nil, sdl.PIXELFORMAT_ARGB8888, unsafe.Pointer(&mirrorPixels[0]), srcPitch); err != nil {
		log.Printf("Warning: second display mirror ReadPixels failed: %v", err)
		return
	}

	syncDRMMirror(mirrorPixels, int(w), int(h), srcPitch)
}

func desiredSecondDisplayCueVisual(state *clock.State) secondDisplayCueVisual {
	if state == nil {
		return secondDisplayCueOff
	}

	if !options.CueSecondDisplay || !isHDMI2Connected() || !state.CueShow {
		return secondDisplayCueOff
	}

	if state.CueRight {
		return secondDisplayCueRight
	}
	if state.CueLeft {
		return secondDisplayCueLeft
	}
	if state.CueBlank {
		return secondDisplayCueBlank
	}

	return secondDisplayCueOff
}

func isHDMI2Connected() bool {
	statusPath := findHDMI2StatusPath()
	if statusPath == "" {
		return false
	}

	statusBytes, err := os.ReadFile(statusPath)
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(statusBytes)) == "connected"
}

func secondDisplayCueAssetPath(visual secondDisplayCueVisual) (string, bool) {
	baseDir := "/tmp/clock8002-cue"
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		log.Printf("Warning: unable to create second display cue asset directory: %v", err)
		return "", false
	}

	assetPath := filepath.Join(baseDir, string(visual)+"-"+secondDisplayCueAssetVersion+".png")
	if _, err := os.Stat(assetPath); err == nil {
		return assetPath, true
	}

	img := renderCueVisualImage(visual, 1920, 1080)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Printf("Warning: unable to encode second display cue image: %v", err)
		return "", false
	}
	if err := os.WriteFile(assetPath, buf.Bytes(), 0o644); err != nil {
		log.Printf("Warning: unable to write second display cue image: %v", err)
		return "", false
	}

	return assetPath, true
}

func renderCueVisualImage(visual secondDisplayCueVisual, width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fillImage(img, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	square := cueSquareBounds(width, height)

	switch visual {
	case secondDisplayCueRight:
		drawRightTriangle(img, square, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	case secondDisplayCueLeft:
		drawLeftTriangle(img, square, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	case secondDisplayCueBlank:
		drawFilledCircle(img, square, color.RGBA{R: 255, G: 239, B: 0, A: 255})
	}

	return img
}

func cueSquareBounds(width, height int) image.Rectangle {
	size := minInt(width, height) - 40
	if size < 1 {
		size = minInt(width, height)
	}
	x := (width - size) / 2
	y := (height - size) / 2
	return image.Rect(x, y, x+size, y+size)
}

func fillImage(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawRightTriangle(img *image.RGBA, b image.Rectangle, c color.RGBA) {
	w := b.Dx()
	h := b.Dy()
	half := h / 2
	for y := 0; y < h; y++ {
		delta := y
		if y > half {
			delta = h - 1 - y
		}
		if half == 0 {
			continue
		}
		xMax := (delta * (w - 1)) / half
		for x := 0; x <= xMax; x++ {
			img.SetRGBA(b.Min.X+x, b.Min.Y+y, c)
		}
	}
}

func drawLeftTriangle(img *image.RGBA, b image.Rectangle, c color.RGBA) {
	w := b.Dx()
	h := b.Dy()
	half := h / 2
	for y := 0; y < h; y++ {
		delta := y
		if y > half {
			delta = h - 1 - y
		}
		if half == 0 {
			continue
		}
		xMin := w - 1 - ((delta * (w - 1)) / half)
		for x := xMin; x < w; x++ {
			img.SetRGBA(b.Min.X+x, b.Min.Y+y, c)
		}
	}
}

func drawFilledCircle(img *image.RGBA, b image.Rectangle, c color.RGBA) {
	w := b.Dx()
	h := b.Dy()
	cx := b.Min.X + (w / 2)
	cy := b.Min.Y + (h / 2)
	r := (minInt(w, h) / 2) - 20
	if r < 1 {
		r = 1
	}
	r2 := r * r
	for y := b.Min.Y; y < b.Max.Y; y++ {
		dy := y - cy
		for x := b.Min.X; x < b.Max.X; x++ {
			dx := x - cx
			if dx*dx+dy*dy <= r2 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstExistingPath(paths ...string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
