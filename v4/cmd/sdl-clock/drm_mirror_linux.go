//go:build linux

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// DRM ioctl numbers (from drm.h / drm_mode.h)
const (
	// DRM_IOCTL_MODE_GETRESOURCES = DRM_IOWR(0xA0, struct drm_mode_card_res)
	drmIoctlModeGetResources = 0xC04064A0

	// DRM_IOCTL_MODE_GETCONNECTOR = DRM_IOWR(0xA7, struct drm_mode_get_connector)
	drmIoctlModeGetConnector = 0xC05064A7

	// DRM_IOCTL_MODE_GETENCODER = DRM_IOWR(0xA6, struct drm_mode_get_encoder)
	drmIoctlModeGetEncoder = 0xC01464A6

	// DRM_IOCTL_MODE_GETCRTC = DRM_IOWR(0xA1, struct drm_mode_crtc)
	drmIoctlModeGetCrtc = 0xC06864A1

	// DRM_IOCTL_MODE_SETCRTC = DRM_IOWR(0xA2, struct drm_mode_crtc)
	drmIoctlModeSetCrtc = 0xC06864A2

	// DRM_IOCTL_MODE_CREATE_DUMB = DRM_IOWR(0xB2, struct drm_mode_create_dumb)
	drmIoctlModeCreateDumb = 0xC02064B2

	// DRM_IOCTL_MODE_MAP_DUMB = DRM_IOWR(0xB3, struct drm_mode_map_dumb)
	drmIoctlModeMapDumb = 0xC01064B3

	// DRM_IOCTL_MODE_DESTROY_DUMB = DRM_IOWR(0xB4, struct drm_mode_destroy_dumb)
	drmIoctlModeDestroyDumb = 0xC00464B4

	// DRM_IOCTL_MODE_ADDFB = DRM_IOWR(0xAE, struct drm_mode_fb_cmd)
	drmIoctlModeAddFB = 0xC01C64AE

	// DRM_IOCTL_MODE_RMFB = DRM_IOWR(0xAF, unsigned int)
	drmIoctlModeRmFB = 0xC00464AF

	// DRM connector types
	drmModeConnectorHDMIA = 11

	// DRM connector status
	drmModeConnected = 1
)

// DRM structures matching kernel ABI (linux/drm_mode.h).
// Field order and sizes must match exactly for ioctl marshalling.

type drmModeCardRes struct {
	FbIdPtr         uint64
	CrtcIdPtr       uint64
	ConnectorIdPtr  uint64
	EncoderIdPtr    uint64
	CountFbs        uint32
	CountCrtcs      uint32
	CountConnectors uint32
	CountEncoders   uint32
	MinWidth        uint32
	MaxWidth        uint32
	MinHeight       uint32
	MaxHeight       uint32
}

type drmModeModeInfo struct {
	Clock      uint32
	Hdisplay   uint16
	HsyncStart uint16
	HsyncEnd   uint16
	Htotal     uint16
	Hskew      uint16
	Vdisplay   uint16
	VsyncStart uint16
	VsyncEnd   uint16
	Vtotal     uint16
	Vscan      uint16
	Vrefresh   uint32
	Flags      uint32
	Type       uint32
	Name       [32]byte
}

type drmModeGetConnector struct {
	EncodersPtr     uint64
	ModesPtr        uint64
	PropsPtr        uint64
	PropValuesPtr   uint64
	CountModes      uint32
	CountProps      uint32
	CountEncoders   uint32
	EncoderID       uint32 // current encoder
	ConnectorID     uint32
	ConnectorType   uint32
	ConnectorTypeID uint32
	Connection      uint32
	MmWidth         uint32
	MmHeight        uint32
	Subpixel        uint32
	Pad             uint32
}

type drmModeGetEncoder struct {
	EncoderID      uint32
	EncoderType    uint32
	CrtcID         uint32
	PossibleCrtcs  uint32
	PossibleClones uint32
}

type drmModeCrtc struct {
	SetConnectorsPtr uint64
	CountConnectors  uint32
	CrtcID           uint32
	FbID             uint32
	X                uint32
	Y                uint32
	GammaSize        uint32
	ModeValid        uint32
	Mode             drmModeModeInfo
}

type drmModeCreateDumb struct {
	Height uint32
	Width  uint32
	Bpp    uint32
	Flags  uint32
	Handle uint32
	Pitch  uint32
	Size   uint64
}

type drmModeMapDumb struct {
	Handle uint32
	Pad    uint32
	Offset uint64
}

type drmModeDestroyDumb struct {
	Handle uint32
}

type drmModeFBCmd struct {
	FbID   uint32
	Width  uint32
	Height uint32
	Pitch  uint32
	Bpp    uint32
	Depth  uint32
	Handle uint32
}

// drmMirror holds state for the direct-DRM mirror output on HDMI-A-1.
type drmMirror struct {
	fd          int
	fbID        uint32
	crtcID      uint32
	connectorID uint32
	dumbHandle  uint32
	width       uint32
	height      uint32
	pitch       uint32
	mmapBuf     []byte
	savedCrtc   *drmModeCrtc
}

var drmMirrorState *drmMirror

func drmIoctl(fd int, request uint, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(request), uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

// findDRICard finds an open DRI card device that has HDMI connectors.
func findDRICard() (int, string, error) {
	// Try card1 first (Pi 5 typically uses card1 for vc4-drm), then card0.
	for _, card := range []string{"/dev/dri/card1", "/dev/dri/card0"} {
		fd, err := syscall.Open(card, syscall.O_RDWR|syscall.O_CLOEXEC, 0)
		if err != nil {
			continue
		}

		// Check if this card has connectors.
		var res drmModeCardRes
		if err := drmIoctl(fd, drmIoctlModeGetResources, unsafe.Pointer(&res)); err != nil {
			syscall.Close(fd)
			continue
		}
		if res.CountConnectors > 0 {
			return fd, card, nil
		}
		syscall.Close(fd)
	}
	return -1, "", fmt.Errorf("no DRI card found with connectors")
}

// findHDMI1Connector scans connectors to find HDMI-A-1 (type=HDMIA, type_id=1) that is connected.
// It returns the connector ID, encoder ID, and mode.
func findHDMI1Connector(fd int) (connID, encID uint32, mode drmModeModeInfo, err error) {
	// First pass: get connector count.
	var res drmModeCardRes
	if err = drmIoctl(fd, drmIoctlModeGetResources, unsafe.Pointer(&res)); err != nil {
		return 0, 0, mode, fmt.Errorf("getResources: %w", err)
	}

	connIDs := make([]uint32, res.CountConnectors)
	res.ConnectorIdPtr = uint64(uintptr(unsafe.Pointer(&connIDs[0])))

	// We also need CRTCs for later.
	crtcIDs := make([]uint32, res.CountCrtcs)
	res.CrtcIdPtr = uint64(uintptr(unsafe.Pointer(&crtcIDs[0])))

	if err = drmIoctl(fd, drmIoctlModeGetResources, unsafe.Pointer(&res)); err != nil {
		return 0, 0, mode, fmt.Errorf("getResources (fill): %w", err)
	}

	for _, cid := range connIDs {
		var conn drmModeGetConnector
		conn.ConnectorID = cid

		// First call: get counts.
		if err = drmIoctl(fd, drmIoctlModeGetConnector, unsafe.Pointer(&conn)); err != nil {
			continue
		}

		// We want HDMI-A type (11) with type_id 1 (i.e., HDMI-A-1).
		if conn.ConnectorType != drmModeConnectorHDMIA || conn.ConnectorTypeID != 1 {
			continue
		}
		if conn.Connection != drmModeConnected {
			continue
		}

		// Second call: get modes.
		modes := make([]drmModeModeInfo, conn.CountModes)
		if conn.CountModes > 0 {
			conn.ModesPtr = uint64(uintptr(unsafe.Pointer(&modes[0])))
		}
		encoders := make([]uint32, conn.CountEncoders)
		if conn.CountEncoders > 0 {
			conn.EncodersPtr = uint64(uintptr(unsafe.Pointer(&encoders[0])))
		}

		if err = drmIoctl(fd, drmIoctlModeGetConnector, unsafe.Pointer(&conn)); err != nil {
			continue
		}

		if conn.CountModes == 0 {
			return 0, 0, mode, fmt.Errorf("HDMI-A-1 connected but no modes available")
		}

		// Use the first preferred mode, or fall back to the first mode.
		selectedMode := modes[0]
		for _, m := range modes {
			if m.Type&(1<<3) != 0 { // DRM_MODE_TYPE_PREFERRED
				selectedMode = m
				break
			}
		}

		return cid, conn.EncoderID, selectedMode, nil
	}

	return 0, 0, mode, fmt.Errorf("HDMI-A-1 connector not found or not connected")
}

// getCrtcForEncoder resolves the CRTC currently assigned to an encoder.
func getCrtcForEncoder(fd int, encoderID uint32) (uint32, error) {
	var enc drmModeGetEncoder
	enc.EncoderID = encoderID
	if err := drmIoctl(fd, drmIoctlModeGetEncoder, unsafe.Pointer(&enc)); err != nil {
		return 0, fmt.Errorf("getEncoder(%d): %w", encoderID, err)
	}
	if enc.CrtcID == 0 {
		return 0, fmt.Errorf("encoder %d has no CRTC assigned", encoderID)
	}
	return enc.CrtcID, nil
}

// initDRMMirror sets up a direct DRM framebuffer on HDMI-A-1 for mirror output.
func initDRMMirror() error {
	fd, cardPath, err := findDRICard()
	if err != nil {
		return fmt.Errorf("findDRICard: %w", err)
	}

	connID, encID, mode, err := findHDMI1Connector(fd)
	if err != nil {
		syscall.Close(fd)
		return fmt.Errorf("findHDMI1: %w", err)
	}

	crtcID, err := getCrtcForEncoder(fd, encID)
	if err != nil {
		syscall.Close(fd)
		return fmt.Errorf("getCrtcForEncoder: %w", err)
	}

	w := uint32(mode.Hdisplay)
	h := uint32(mode.Vdisplay)

	log.Printf("Info: DRM mirror: %s connector=%d encoder=%d crtc=%d mode=%dx%d@%dHz",
		cardPath, connID, encID, crtcID, w, h, mode.Vrefresh)

	// Save current CRTC state for cleanup.
	saved := drmModeCrtc{CrtcID: crtcID}
	if err := drmIoctl(fd, drmIoctlModeGetCrtc, unsafe.Pointer(&saved)); err != nil {
		log.Printf("Warning: DRM mirror: could not save CRTC state: %v", err)
	}

	// Create dumb buffer (XRGB8888 = 32bpp, depth 24).
	create := drmModeCreateDumb{
		Width:  w,
		Height: h,
		Bpp:    32,
	}
	if err := drmIoctl(fd, drmIoctlModeCreateDumb, unsafe.Pointer(&create)); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("createDumb: %w", err)
	}

	// Add framebuffer.
	fb := drmModeFBCmd{
		Width:  w,
		Height: h,
		Pitch:  create.Pitch,
		Bpp:    32,
		Depth:  24,
		Handle: create.Handle,
	}
	if err := drmIoctl(fd, drmIoctlModeAddFB, unsafe.Pointer(&fb)); err != nil {
		// Clean up dumb buffer.
		destroy := drmModeDestroyDumb{Handle: create.Handle}
		drmIoctl(fd, drmIoctlModeDestroyDumb, unsafe.Pointer(&destroy))
		syscall.Close(fd)
		return fmt.Errorf("addFB: %w", err)
	}

	// Map dumb buffer for CPU access.
	mapReq := drmModeMapDumb{Handle: create.Handle}
	if err := drmIoctl(fd, drmIoctlModeMapDumb, unsafe.Pointer(&mapReq)); err != nil {
		drmIoctl(fd, drmIoctlModeRmFB, unsafe.Pointer(&fb.FbID))
		destroy := drmModeDestroyDumb{Handle: create.Handle}
		drmIoctl(fd, drmIoctlModeDestroyDumb, unsafe.Pointer(&destroy))
		syscall.Close(fd)
		return fmt.Errorf("mapDumb: %w", err)
	}

	mmapBuf, err := syscall.Mmap(fd, int64(mapReq.Offset), int(create.Size),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		drmIoctl(fd, drmIoctlModeRmFB, unsafe.Pointer(&fb.FbID))
		destroy := drmModeDestroyDumb{Handle: create.Handle}
		drmIoctl(fd, drmIoctlModeDestroyDumb, unsafe.Pointer(&destroy))
		syscall.Close(fd)
		return fmt.Errorf("mmap: %w", err)
	}

	// Clear to black.
	for i := range mmapBuf {
		mmapBuf[i] = 0
	}

	// Set CRTC to display our framebuffer.
	setCrtc := drmModeCrtc{
		CrtcID:           crtcID,
		FbID:             fb.FbID,
		X:                0,
		Y:                0,
		SetConnectorsPtr: uint64(uintptr(unsafe.Pointer(&connID))),
		CountConnectors:  1,
		ModeValid:        1,
		Mode:             mode,
	}
	if err := drmIoctl(fd, drmIoctlModeSetCrtc, unsafe.Pointer(&setCrtc)); err != nil {
		syscall.Munmap(mmapBuf)
		drmIoctl(fd, drmIoctlModeRmFB, unsafe.Pointer(&fb.FbID))
		destroy := drmModeDestroyDumb{Handle: create.Handle}
		drmIoctl(fd, drmIoctlModeDestroyDumb, unsafe.Pointer(&destroy))
		syscall.Close(fd)
		return fmt.Errorf("setCrtc: %w", err)
	}

	drmMirrorState = &drmMirror{
		fd:          fd,
		fbID:        fb.FbID,
		crtcID:      crtcID,
		connectorID: connID,
		dumbHandle:  create.Handle,
		width:       w,
		height:      h,
		pitch:       create.Pitch,
		mmapBuf:     mmapBuf,
		savedCrtc:   &saved,
	}

	log.Printf("Info: DRM mirror: initialized %dx%d XRGB8888 framebuffer on HDMI-A-1 (fb=%d)", w, h, fb.FbID)
	return nil
}

// syncDRMMirror copies pixel data from the SDL primary renderer into the DRM mirror framebuffer.
// pixelData must be XRGB8888 (or ARGB8888 — alpha is ignored by the display).
// srcWidth/srcHeight are the source dimensions; the image is centered/scaled into the mirror FB.
func syncDRMMirror(pixelData []byte, srcWidth, srcHeight, srcPitch int) {
	m := drmMirrorState
	if m == nil {
		return
	}

	dstPitch := int(m.pitch)
	dstW := int(m.width)
	dstH := int(m.height)

	if srcWidth == dstW && srcHeight == dstH && srcPitch == dstPitch {
		// Fast path: dimensions match exactly, direct copy.
		copy(m.mmapBuf, pixelData[:dstH*dstPitch])
		return
	}

	// Scale/center: compute letterbox/pillarbox offsets.
	scaleX := float64(dstW) / float64(srcWidth)
	scaleY := float64(dstH) / float64(srcHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	scaledW := int(float64(srcWidth) * scale)
	scaledH := int(float64(srcHeight) * scale)

	if scale == 1.0 && scaledW == srcWidth && scaledH == srcHeight {
		// Same resolution, possibly different pitch — copy row by row.
		offsetX := (dstW - scaledW) / 2
		offsetY := (dstH - scaledH) / 2

		// Clear the buffer first (letterbox bars).
		for i := range m.mmapBuf[:dstH*dstPitch] {
			m.mmapBuf[i] = 0
		}

		copyW := scaledW * 4
		for y := 0; y < scaledH; y++ {
			dstOff := (offsetY+y)*dstPitch + offsetX*4
			srcOff := y * srcPitch
			copy(m.mmapBuf[dstOff:dstOff+copyW], pixelData[srcOff:srcOff+copyW])
		}
		return
	}

	// Nearest-neighbor scale for mismatched resolutions.
	offsetX := (dstW - scaledW) / 2
	offsetY := (dstH - scaledH) / 2

	// Clear buffer.
	for i := range m.mmapBuf[:dstH*dstPitch] {
		m.mmapBuf[i] = 0
	}

	for dy := 0; dy < scaledH; dy++ {
		sy := dy * srcHeight / scaledH
		dstOff := (offsetY+dy)*dstPitch + offsetX*4
		srcRowOff := sy * srcPitch
		for dx := 0; dx < scaledW; dx++ {
			sx := dx * srcWidth / scaledW
			sp := srcRowOff + sx*4
			dp := dstOff + dx*4
			// Copy 4 bytes (XRGB8888).
			m.mmapBuf[dp] = pixelData[sp]
			m.mmapBuf[dp+1] = pixelData[sp+1]
			m.mmapBuf[dp+2] = pixelData[sp+2]
			m.mmapBuf[dp+3] = pixelData[sp+3]
		}
	}
}

// destroyDRMMirror cleans up the DRM mirror resources.
func destroyDRMMirror() {
	m := drmMirrorState
	if m == nil {
		return
	}
	drmMirrorState = nil

	// Restore saved CRTC state if possible.
	if m.savedCrtc != nil && m.savedCrtc.ModeValid != 0 {
		drmIoctl(m.fd, drmIoctlModeSetCrtc, unsafe.Pointer(m.savedCrtc))
	}

	if m.mmapBuf != nil {
		syscall.Munmap(m.mmapBuf)
	}

	fbID := m.fbID
	drmIoctl(m.fd, drmIoctlModeRmFB, unsafe.Pointer(&fbID))

	destroy := drmModeDestroyDumb{Handle: m.dumbHandle}
	drmIoctl(m.fd, drmIoctlModeDestroyDumb, unsafe.Pointer(&destroy))

	syscall.Close(m.fd)
	log.Printf("Info: DRM mirror: destroyed")
}

// isDRMMirrorActive returns whether the DRM mirror is currently initialized.
func isDRMMirrorActive() bool {
	return drmMirrorState != nil
}

// drmMirrorSize returns the mirror framebuffer dimensions.
func drmMirrorSize() (width, height uint32) {
	m := drmMirrorState
	if m == nil {
		return 0, 0
	}
	return m.width, m.height
}

// findHDMI1StatusPath returns the sysfs path for HDMI-A-1 status.
func findHDMI1StatusPath() string {
	const explicitPath = "/sys/class/drm/card1-HDMI-A-1/status"
	if _, err := os.Stat(explicitPath); err == nil {
		return explicitPath
	}

	matches, err := filepath.Glob("/sys/class/drm/*HDMI-A-1/status")
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// isHDMI1Connected checks if HDMI-A-1 is physically connected.
func isHDMI1Connected() bool {
	statusPath := findHDMI1StatusPath()
	if statusPath == "" {
		return false
	}
	statusBytes, err := os.ReadFile(statusPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(statusBytes)) == "connected"
}
