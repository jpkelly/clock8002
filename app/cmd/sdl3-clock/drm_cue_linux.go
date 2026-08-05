//go:build linux

package main

// updateCueDRMBuffer renders the cue icon for visual directly into the DRM dumb buffer.
func updateCueDRMBuffer(visual secondDisplayCueVisual) {
	m := drmMirrorState
	if m == nil {
		return
	}
	img := renderCueVisualImage(visual, int(m.width), int(m.height))
	pitch := int(m.pitch)
	for y := 0; y < int(m.height); y++ {
		for x := 0; x < int(m.width); x++ {
			c := img.RGBAAt(x, y)
			off := y*pitch + x*4
			m.mmapBuf[off+0] = c.B // XRGB8888 little-endian
			m.mmapBuf[off+1] = c.G
			m.mmapBuf[off+2] = c.R
			m.mmapBuf[off+3] = 0
		}
	}
}
