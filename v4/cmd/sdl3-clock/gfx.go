package main

// Ported primitives from sdl gfx library

import (
	"github.com/Zyko0/go-sdl3/sdl"
)

const ellipseOverscan = 4

func drawQuadrants(x, y, dx, dy int, filled bool) error {
	var err error
	if dx == 0 {
		if dy == 0 {
			err = pixel(x, y)
		} else {
			if filled {
				err = vline(x, y-dy, y+dy)
			} else {
				err = pixel(x, y+dy)
				if err == nil {
					err = pixel(x, y-dy)
				}
			}
		}
	} else {
		xpdx, xmdx := x+dx, x-dx
		ypdy, ymdy := y+dy, y-dy
		if filled {
			err = vline(xpdx, ymdy, ypdy)
			if err == nil {
				err = vline(xmdx, ymdy, ypdy)
			}
		} else {
			err = pixel(xpdx, ypdy)
			if err == nil {
				err = pixel(xmdx, ypdy)
			}
			if err == nil {
				err = pixel(xpdx, ymdy)
			}
			if err == nil {
				err = pixel(xmdx, ymdy)
			}
		}
	}
	return err
}

func ellipseRGBA(x, y, rx, ry int, c sdl.Color, filled bool) error {
	if rx < 0 || ry < 0 {
		return nil
	}

	// Set Blend Mode
	bm := sdl.BLENDMODE_NONE
	if c.A < 255 {
		bm = sdl.BLENDMODE_BLEND
	}
	renderer.SetDrawBlendMode(bm)
	setDrawSdlColor(c)

	// Special cases
	if rx == 0 {
		if ry == 0 {
			return pixel(x, y)
		}
		return vline(x, y-ry, y+ry)
	} else if ry == 0 {
		return hline(x-rx, x+rx, y)
	}

	var oldX, oldY int = 0, ry
	drawQuadrants(x, y, 0, ry, filled)

	// Midpoint algorithm with overscan
	rx32, ry32 := rx*ellipseOverscan, ry*ellipseOverscan
	rx2 := rx32 * rx32
	rx22 := rx2 + rx2
	ry2 := ry32 * ry32
	ry22 := ry2 + ry2

	var curX int = 0
	var curY int = ry32
	var deltaX int = 0
	deltaY := rx22 * curY

	// Segment 1
	errorVal := ry2 - rx2*ry32 + rx2/4
	for deltaX <= deltaY {
		curX++
		deltaX += ry22
		errorVal += deltaX + ry2
		if errorVal >= 0 {
			curY--
			deltaY -= rx22
			errorVal -= deltaY
		}

		scrX, scrY := curX/ellipseOverscan, curY/ellipseOverscan
		if (scrX != oldX && scrY == oldY) || (scrX != oldX && scrY != oldY) {
			drawQuadrants(x, y, scrX, scrY, filled)
			oldX, oldY = scrX, scrY
		}
	}

	// Segment 2
	if curY > 0 {
		curXp1 := curX + 1
		curYm1 := curY - 1
		errorVal = ry2*curX*curXp1 + ((ry2 + 3) / 4) + rx2*curYm1*curYm1 - rx2*ry2
		for curY > 0 {
			curY--
			deltaY -= rx22
			errorVal += rx2 - deltaY

			if errorVal <= 0 {
				curX++
				deltaX += ry22
				errorVal += deltaX
			}

			scrX, scrY := curX/ellipseOverscan, curY/ellipseOverscan
			if (scrX != oldX && scrY == oldY) || (scrX != oldX && scrY != oldY) {
				oldY--
				for ; oldY >= scrY; oldY-- {
					drawQuadrants(x, y, scrX, oldY, filled)
					if filled {
						oldY = scrY - 1
					}
				}
				oldX, oldY = scrX, scrY
			}
		}
		// Remaining vertical points
		if !filled {
			oldY--
			for ; oldY >= 0; oldY-- {
				drawQuadrants(x, y, oldX, oldY, filled)
			}
		}
	}
	return nil
}

// FilledCircleRGBA is the public wrapper for circles
func filledCircleRGBA(x, y, rad int, c sdl.Color) error {
	return ellipseRGBA(x, y, rad, rad, c, true)
}

func filledCircle(x, y, rad float32, c sdl.Color) {
	filledCircleRGBA(int(x), int(y), int(rad), c)
}
