package main

import (
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
)

type cue struct {
	right bool
	left  bool
	blank bool
}

var cueState cue

func updateCue(state *clock.State) {
	newCue := cue{
		right: state.CueRight,
		left:  state.CueLeft,
		blank: state.CueBlank,
	}

	if cueState == newCue {
		return
	}
	cueState = newCue

	err := renderer.SetRenderTarget(cueTexture)
	check(err)
	renderer.SetDrawColor(0, 0, 0, 0)
	renderer.Clear()

	vy := make([]int16, 3)
	vy[0] = 1
	vy[1] = 250
	vy[2] = 499

	vx := make([]int16, 3)
	c := sdl.Color{
		R: 0,
		G: 0,
		B: 0,
		A: 255,
	}

	if cueState.right {
		vx[0] = 1
		vx[1] = 499
		vx[2] = 1
		c.G = 255
		gfx.FilledPolygonColor(renderer, vx, vy, c)
	} else if cueState.left {
		vx[0] = 499
		vx[1] = 1
		vx[2] = 499
		c.R = 255
		gfx.FilledPolygonColor(renderer, vx, vy, c)
	} else if cueState.blank {
		c = sdl.Color{
			R: 255,
			G: 239,
			B: 0,
			A: 255,
		}
		gfx.FilledCircleColor(renderer, 249, 249, 245, c)
	}

	renderer.SetRenderTarget(nil)
}

func drawCue() {
	canvasW, canvasH := renderer.GetLogicalSize()
	if canvasW <= 0 || canvasH <= 0 {
		canvasW, canvasH, _ = renderer.GetOutputSize()
	}

	clamp := func(v, min, max int32) int32 {
		if v < min {
			return min
		}
		if v > max {
			return max
		}
		return v
	}

	computeRect := func() sdl.Rect {
		x := int32(options.CuePosX)
		y := int32(options.CuePosY)
		s := int32(options.CueSize)
		w := s
		h := s

		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		if w > canvasW {
			w = canvasW
		}
		if h > canvasH {
			h = canvasH
		}

		x = clamp(x, 0, canvasW-w)
		y = clamp(y, 0, canvasH-h)

		return sdl.Rect{X: x, Y: y, W: w, H: h}
	}

	if options.CueFullScreen {

		// Keep legacy fullscreen behavior unchanged when cue-fullscreen is enabled.
		rect := sdl.Rect{X: 10, Y: 10, W: canvasW - 20, H: canvasH - 20}
		if rect.W < 1 {
			rect.W = 1
		}
		if rect.H < 1 {
			rect.H = 1
		}
		copyIntoRect(cueTexture, rect)
	} else {
		rect := computeRect()
		copyIntoRect(cueTexture, rect)
	}

}
