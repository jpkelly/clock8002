package main

import (
	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/jpkelly/clock8002/app/clock"
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

	renderTarget(cueTexture)
	renderer.SetDrawColor(0, 0, 0, 0)
	renderer.Clear()

	vy := make([]int16, 3)
	vy[0] = 1
	vy[1] = 250
	vy[2] = 499

	vx := make([]int16, 3)

	if cueState.right {
		vx[0] = 1
		vx[1] = 499
		vx[2] = 1
		c := sdl.FColor{
			R: 0,
			G: 1,
			B: 0,
			A: 1,
		}

		verts := make([]sdl.Vertex, 3)
		for i := range verts {
			verts[i].Color = c
			verts[i].Position = sdl.FPoint{X: float32(vx[i]), Y: float32(vy[i])}
		}

		renderer.RenderGeometry(nil, verts, nil)
	} else if cueState.left {
		vx[0] = 499
		vx[1] = 1
		vx[2] = 499
		c := sdl.FColor{
			R: 1,
			G: 0,
			B: 0,
			A: 1,
		}

		verts := make([]sdl.Vertex, 3)
		for i := range verts {
			verts[i].Color = c
			verts[i].Position = sdl.FPoint{X: float32(vx[i]), Y: float32(vy[i])}
		}

		renderer.RenderGeometry(nil, verts, nil)
	} else if cueState.blank {
		c := sdl.Color{
			R: 255,
			G: 239,
			B: 0,
			A: 255,
		}
		filledCircle(249, 249, 245, c)
	}

	renderTarget(nil)
}

func drawCue() {
	if options.CueFullScreen {
		// Match the logical canvas size set in setupScaling(); text4
		// shrinks the logical size 5% to upscale content, and the full-
		// screen cue rect must track it or it clips at the edges.
		var lw, lh float32 = 1920, 1080
		if options.Face == "text4" {
			lw, lh = 1829, 1029
		}
		if options.vertical {
			lw, lh = lh, lw
		}
		rect := sdl.FRect{
			X: 10,
			Y: 10,
			W: lw - 20,
			H: lh - 20,
		}
		copyIntoRect(cueTexture, rect)
	} else {
		size := float32(options.CueSize)
		if size < 1 {
			size = 150
		}
		posX := float32(options.CuePosX)
		posY := float32(options.CuePosY)
		if posX < 0 {
			posX = 0
		}
		if posY < 0 {
			posY = 0
		}
		rect := sdl.FRect{
			H: size,
			W: size,
			X: posX,
			Y: posY,
		}
		copyIntoRect(cueTexture, rect)
	}
}
