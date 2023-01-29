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
	if options.CueFullScreen {

		rect := sdl.Rect{
			W: 1920 - 20,
			H: 1080 - 20,
			X: 10,
			Y: 10,
		}
		copyIntoRect(cueTexture, rect)
	} else {
		rect := sdl.Rect{
			H: 150,
			W: 150,
			X: 5,
			Y: 1080 - 155,
		}
		copyIntoRect(cueTexture, rect)
	}

}
