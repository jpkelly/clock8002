package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/Zyko0/go-sdl3/ttf"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
)

var colors struct {
	static     sdl.Color
	sec        sdl.Color
	text       sdl.Color
	countdown  sdl.Color
	tally      sdl.Color
	tallyBG    sdl.Color
	row        [4]sdl.Color
	rowBG      [4]sdl.Color
	icon       [4]sdl.Color
	signal     [4]sdl.Color
	label      sdl.Color
	labelBG    sdl.Color
	background sdl.Color
}

var window *sdl.Window

// fullscreenMode, when set, pins the exclusive fullscreen video mode used under
// the kmsdrm backend so the output is a real 1080p signal rather than the
// panel's native (possibly 4K) mode. nil on desktop backends.
var fullscreenMode *sdl.DisplayMode

var renderer *sdl.Renderer

var textureSource sdl.FRect

var staticTexture *sdl.Texture
var secTexture *sdl.Texture
var backgroundTexture *sdl.Texture
var clockTextures []*sdl.Texture

var infoTexture *sdl.Texture
var infoFont *ttf.Font

var cueTexture *sdl.Texture

// initSDL initializes the SDL library, creates a window and a hw accelerated renderer
func initSDL() {
	var err error
	sdl.SetAppMetadata("clock-8001", clock.Version, "com.clock-8001")

	sdl.SetHint(sdl.HINT_APP_NAME, "clock-8001")

	if err = sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO | sdl.INIT_EVENTS); err != nil {
		log.Fatalf("Failed to initialize SDL: %s\n", err)
	}

	title := fmt.Sprintf("clock-8001 v%s", clock.Version)

	log.Printf("SDL video driver: %s", sdl.GetCurrentVideoDriver())

	if sdl.GetCurrentVideoDriver() == "kmsdrm" {
		display := sdl.GetPrimaryDisplay()

		// Force 1080p output. The clock canvas is authored at 1920x1080 (see
		// setupScaling); the hardware must never drive a larger CRTC mode. On a
		// 4K display SDL would otherwise select the largest advertised mode and
		// emit an upscaled 4K signal. Pick the closest 1920x1080@60 mode the
		// display advertises and pin it as the exclusive fullscreen mode below.
		mode, modeErr := display.ClosestFullscreenDisplayMode(1920, 1080, 60, false)
		if modeErr != nil || mode == nil {
			// Fall back to the display's largest advertised mode.
			modes, listErr := display.FullscreenDisplayModes()
			if listErr == nil && len(modes) > 0 {
				mode = modes[0]
			}
		}
		if mode != nil {
			winWidth = int(mode.W)
			winHeight = int(mode.H)
			fullscreenMode = mode
		}
		log.Printf("-> kmsdrm detected, using %d x %d for screen resolution", winWidth, winHeight)
	}

	window, renderer, err = sdl.CreateWindowAndRenderer(title, winWidth, winHeight, sdl.WINDOW_RESIZABLE)
	if err != nil {
		log.Fatalf("Failed to create window and renderer: %v", err)
	}

	// Pin the exclusive fullscreen mode so SetFullscreen(true) performs a real
	// modeset to 1080p instead of a fullscreen-desktop pass at the panel's
	// native (possibly 4K) resolution.
	if fullscreenMode != nil {
		if err = window.SetFullscreenMode(fullscreenMode); err != nil {
			log.Printf("Warning: failed to pin %d x %d fullscreen mode: %v", winWidth, winHeight, err)
		}
	}

	renderer.SetVSync(1)
	window.SetSurfaceVSync(1)

	err = sdl.HideCursor() // Hide mouse cursor
	check(err)

	err = renderer.Clear()
	check(err)

	err = ttf.Init()
	check(err)

	log.Printf("SDL init done\n")

	name, err := renderer.Name()
	check(err)
	log.Printf("Renderer: %v\n", name)

	infoFont, err = ttf.OpenFont(options.LabelFont, 50)
	check(err)

	// Do not exit full screen mode on focus loss in multi-monitor systems
	sdl.SetHint(sdl.HINT_VIDEO_MINIMIZE_ON_FOCUS_LOSS, "0")
}

// initColors takes the color options from flags and translates them to sdl.Color variables
func initColors() {
	var err error

	colors.text, err = parseColor(options.TextColor)
	check(err)

	colors.static, err = parseColor(options.StaticColor)
	check(err)

	colors.sec, err = parseColor(options.SecondColor)
	check(err)

	colors.countdown, err = parseColor(options.CountdownColor)
	check(err)

	colors.row[0], err = parseColor(options.Row1Color)
	check(err)
	colors.row[0].A = options.Row1Alpha

	colors.row[1], err = parseColor(options.Row2Color)
	check(err)
	colors.row[1].A = options.Row1Alpha

	colors.row[2], err = parseColor(options.Row3Color)
	check(err)
	colors.row[2].A = options.Row3Alpha

	colors.row[3], err = parseColor(options.Row4Color)
	check(err)
	colors.row[3].A = options.Row4Alpha

	for i := 0; i < 4; i++ {
		colors.icon[i] = colors.row[i]
	}

	for i := 0; i < 4; i++ {
		colors.signal[i] = sdl.Color{R: 0, G: 0, B: 0, A: 0}
	}

	colors.label, err = parseColor(options.LabelColor)
	check(err)
	colors.label.A = options.LabelAlpha

	colors.labelBG, err = parseColor(options.LabelBG)
	check(err)
	colors.labelBG.A = options.LabelBGAlpha

	timerBG, err := parseColor(options.TimerBG)
	check(err)
	timerBG.A = options.TimerBGAlpha
	for i := 0; i < 4; i++ {
		colors.rowBG[i] = timerBG
	}

	colors.background, err = parseColor(options.BackgroundColor)
	check(err)

	colors.tally = sdl.Color{R: 0, G: 0, B: 0, A: 0}
}

// initTextures initializes the circle textures for seconds and static "hour" markers
func initTextures() {
	var textureSize int = 40
	var textureCoord float32 = 20
	var textureRadius float32 = 19
	var err error

	debug.Printf("Rendering circle textures -> Size: %v, Coord: %v, Radius: %v", textureSize, textureCoord, textureRadius)

	// Constants for the small 192x192 px clock
	if options.small {
		textureSize = 5
		textureCoord = 3
		textureRadius = 3
		if options.Face == "144" {
			gridStartX = 24
			gridStartY = 24
			gridSize = 2
			gridSpacing = 3
		} else {
			// 192x192px clock
			gridStartX = 32
			gridStartY = 32
			gridSize = 3
			gridSpacing = 4
		}
	}

	// Texture for 12 static circles
	renderer.SetDrawColor(0, 0, 0, 0)
	if staticTexture != nil {
		staticTexture.Destroy()
	}
	staticTexture, err = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, textureSize, textureSize)
	check(err)
	err = staticTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	check(err)

	renderTarget(staticTexture)
	renderer.Clear()

	if !options.small {
		filledCircle(textureCoord, textureCoord, textureRadius, colors.static)
	} else {
		setDrawSdlColor(colors.static)
		check(err)

		for _, point := range circlePixels {
			pixel(point[0], point[1])
		}
	}

	// Texture for the second marker circles
	renderer.SetDrawColor(0, 0, 0, 0)
	if secTexture != nil {
		secTexture.Destroy()
	}
	secTexture, err = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, textureSize, textureSize)
	check(err)
	err = secTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	check(err)

	renderTarget(secTexture)
	renderer.Clear()

	if !options.small {
		filledCircle(textureCoord, textureCoord, textureRadius, colors.sec)
	} else {
		setDrawSdlColor(colors.sec)

		for _, point := range circlePixels {
			pixel(point[0], point[1])
		}
	}

	cueTexture, err = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, 500, 500)
	check(err)
	err = cueTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	check(err)

	renderTarget(nil)

	textureSource = sdl.FRect{X: 0, Y: 0, W: float32(textureSize), H: float32(textureSize)}
}

func setupScaling() {
	x, y, _ := renderer.CurrentOutputSize()
	if y > x {
		options.vertical = true
	}

	if options.dualClock || options.textClock || options.countdown {
		// FIXME: rpi display scaling fix
		// Dual clock
		log.Printf("SDL output size: %v, %v", x, y)
		if options.Face == "max" {
			// No logical size, we might have different aspect ratio
			return
		} else if !options.vertical {
			var lw, lh int32 = 1920, 1080
			if options.Face == "text4" {
				// 5% scale-up for text4: shrink logical canvas so SDL upscales content
				lw, lh = 1829, 1029
			}
			err := renderer.SetLogicalPresentation(lw, lh, sdl.LOGICAL_PRESENTATION_LETTERBOX)
			check(err)
		} else {
			// rotated display
			var lw, lh int32 = 1080, 1920
			if options.Face == "text4" {
				lw, lh = 1029, 1829
			}
			err := renderer.SetLogicalPresentation(lw, lh, sdl.LOGICAL_PRESENTATION_LETTERBOX)
			check(err)
		}
	} else if !options.NoARCorrection {
		rpiDisplayCorrection()
	}
}

// rpiDisplayCorrection detects the official 7" rpi display and applies aspect ratio correction.
// The official display has non-square pixels...
func rpiDisplayCorrection() {
	// the official raspberry pi display has weird pixels
	// We detect it by the unusual 800 x 480 resolution
	// We will eventually support rotated displays also
	x, y, _ := renderer.RenderOutputSize()
	log.Printf("SDL renderer size: %v x %v", x, y)
	scaleX, scaleY, _ := renderer.Scale()
	log.Printf("Scaling: x: %v, y: %v\n", scaleX, scaleY)

	if (x == 800) && (y == 480) {
		// Official display, rotated 0 or 180 degrees
		// The display has non-square pixels and needs correction:
		// Y scale = 1
		// Scale for x is ((9*800) / (16*480)) = 0.9375
		err := renderer.SetScale(0.9375, 1)
		check(err)
		log.Printf("Detected official raspberry pi display, correcting aspect ratio\n")
		check(err)
	} else if (y == 800) && (x == 480) {
		// Official display rotated 90 or 270 degrees
		err := renderer.SetScale(1, 0.9375)
		check(err)
		log.Printf("Detected official raspberry pi display (rotated 90 or 270 deg), correcting aspect ratio.\n")
		log.Printf("Moving clock to top corner of the display.\n")
	}
}

// drawSecondCircles draws the requested amount of the second marker circles on the ring
func drawSecondCircles(seconds int) {
	// Clamp the array index
	if seconds > 59 {
		seconds = 59
	} else if seconds < 0 {
		seconds = 0
	}
	// Draw second circles
	for i := 0; i <= int(seconds); i++ {
		dest := sdl.FRect{X: float32(secCircles[i].X - 20), Y: float32(secCircles[i].Y - 20), W: 40, H: 40}
		if options.small {
			dest = sdl.FRect{X: float32(secCircles[i].X - 3), Y: float32(secCircles[i].Y - 3), W: 5, H: 5}
		}
		err := renderer.RenderTexture(secTexture, &textureSource, &dest)
		check(err)
	}
}

// drawStaticCircles draws the 12 static "hour" marker circles
func drawStaticCircles() {
	// Draw static indicator circles
	for _, p := range staticCircles {
		if options.small {
			dest := sdl.FRect{X: float32(p.X - 3), Y: float32(p.Y - 3), W: 5, H: 5}
			err := renderer.RenderTexture(staticTexture, &textureSource, &dest)
			check(err)
		} else {
			dest := sdl.FRect{X: float32(p.X - 20), Y: float32(p.Y - 20), W: 40, H: 40}
			err := renderer.RenderTexture(staticTexture, &textureSource, &dest)
			check(err)
		}
	}
}

// drawDots draws the two dots between hours and minutes on the clock
func drawDots(y int, x int, c sdl.Color) {
	// Draw the dots between hours and minutes
	setMatrix(y, x, c)
	setMatrix(y, x+1, c)
	setMatrix(y+1, x, c)
	setMatrix(y+1, x+1, c)

	setMatrix(y+4, x, c)
	setMatrix(y+4, x+1, c)
	setMatrix(y+5, x, c)
	setMatrix(y+5, x+1, c)
}

// Fills the screen with white
func drawWhiteScreen() {
	renderTarget(nil)

	err := renderer.SetDrawColor(255, 255, 255, 255)
	check(err)

	err = renderer.Clear()
	check(err)
}

// setMatrix draws a "led matrix" pixel
func setMatrix(cy, cx int, color sdl.Color) {
	x := gridStartX + float32(cx*gridSpacing)
	y := gridStartY + float32(cy*gridSpacing)
	rect := sdl.FRect{X: x, Y: y, W: gridSize, H: gridSize}

	rectColor(&rect, color)
}

// setPixel sets a generic "pixel" on a grid
func setPixel(cy, cx int, color sdl.Color, startX, startY, spacing, pixelSize int32) {
	x := startX + int32(cx)*spacing
	y := startY + int32(cy)*spacing
	rect := sdl.FRect{X: float32(x), Y: float32(y), W: float32(pixelSize), H: float32(pixelSize)}

	rectColor(&rect, color)
}

// drawBitmask draws a 2d boolean array
func drawBitmask(bitmask [][]bool, color sdl.Color, r int, c int) {
	for y, row := range bitmask {
		for x, b := range row {
			if b {
				setMatrix(r+y, c+x, color)
			}
		}
	}
}

// clearCanvas fills the whole SDL window with black
func clearCanvas() {
	err := renderer.SetDrawColor(0, 0, 0, 0)
	check(err)

	err = renderer.Clear()
	check(err)
}

// prepare the main window canvas with the background
func prepareCanvas() {
	renderTarget(nil)

	setDrawSdlColor(colors.background)
	renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
	err := renderer.Clear()
	check(err)

	// Copy the background image as needed
	if showBackground {
		h, w, _ := backgroundTexture.Size()
		rect := sdl.FRect{X: 0, Y: 0, H: h, W: w}
		renderer.RenderTexture(backgroundTexture, &rect, nil)
	} else {
		x, y, _ := renderer.CurrentOutputSize()
		rect := sdl.FRect{X: 0, Y: 0, W: float32(x), H: float32(y)}
		renderer.RenderFillRect(&rect)
	}
}

// parseColor parses a string "#XXX or #XXXXXX to a sdl.Color"
func parseColor(s string) (c sdl.Color, err error) {
	c.A = 0xff
	switch len(s) {
	case 7:
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	case 4:
		_, err = fmt.Sscanf(s, "#%1x%1x%1x", &c.R, &c.G, &c.B)
		// Double the hex digits:
		c.R *= 17
		c.G *= 17
		c.B *= 17
	default:
		err = fmt.Errorf("parseColor(): invalid length, must be 7 or 4: %v", s)
	}
	return
}

func setDrawSdlColor(c sdl.Color) {
	renderer.SetDrawColor(c.R, c.G, c.B, c.A)
}

func setDrawColor(c color.RGBA) {
	renderer.SetDrawColor(c.R, c.G, c.B, c.A)
}

func toSDLColor(in color.RGBA) sdl.Color {
	return sdl.Color{
		R: in.R,
		G: in.G,
		B: in.B,
		A: in.A,
	}
}

func toRGBA(in sdl.Color) color.RGBA {
	return color.RGBA{
		R: in.R,
		G: in.G,
		B: in.B,
		A: in.A,
	}
}

func rectColor(rect *sdl.FRect, color sdl.Color) {
	setDrawSdlColor(color)

	err := renderer.RenderFillRect(rect)
	check(err)
}

func renderTarget(t *sdl.Texture) {
	err := renderer.SetRenderTarget(t)
	if err != nil {
		panic(err)
	}
}

func pixel(x, y int) error {
	return renderer.RenderPoint(float32(x), float32(y))
}

func vline(x, y1, y2 int) error {
	return renderer.RenderLine(float32(x), float32(y1), float32(x), float32(y2))
}

func hline(x1, x2, y int) error {
	return renderer.RenderLine(float32(x1), float32(y), float32(x2), float32(y))
}
