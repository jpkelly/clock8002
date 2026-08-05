//go:build !linux

package main

func initDRMMirror() error {
	return nil
}

func syncDRMMirror(pixelData []byte, srcWidth, srcHeight, srcPitch int) {}

func destroyDRMMirror() {}

func isDRMMirrorActive() bool {
	return false
}

func drmMirrorSize() (width, height uint32) {
	return 0, 0
}

func isSpareHDMIConnected() bool {
	return false
}
