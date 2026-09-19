package main

import (
	"testing"

	"github.com/Zyko0/go-sdl3/sdl"
)

// contains reports whether outer fully encloses inner.
func contains(outer, inner sdl.FRect) bool {
	return inner.X >= outer.X && inner.Y >= outer.Y &&
		inner.X+inner.W <= outer.X+outer.W &&
		inner.Y+inner.H <= outer.Y+outer.H
}

// TestScaleRowKeepsNesting is the regression test for the bug that scaling each
// rect about its own centre introduced: the icon walked out of the number box
// and the digits overflowed its right edge, which is visible whenever
// draw-boxes is on. A row must stay nested at every supported scale.
func TestScaleRowKeepsNesting(t *testing.T) {
	// Row geometry of each text clock face, as the draw functions build it.
	faces := []struct {
		name                    string
		numberBox, iconR, textR sdl.FRect
	}{
		{
			"text/text3-row",
			sdl.FRect{X: 530, Y: 10, W: 1380, H: 300},
			sdl.FRect{X: 530, Y: 10, W: 300, H: 300},
			sdl.FRect{X: 830, Y: 10, W: 1080, H: 300},
		},
		{
			"text2-row",
			sdl.FRect{X: 530, Y: 25, W: 1380, H: 440},
			sdl.FRect{X: 530, Y: 25, W: 300, H: 440},
			sdl.FRect{X: 830, Y: 25, W: 1080, H: 440},
		},
		{
			"text4-row",
			sdl.FRect{X: 505, Y: 40, W: 1380, H: 210},
			sdl.FRect{X: 505, Y: 40, W: 300, H: 210},
			sdl.FRect{X: 805, Y: 40, W: 1080, H: 210},
		},
		{
			"single-line",
			sdl.FRect{X: 25, Y: 290, W: 1870, H: 440},
			sdl.FRect{X: 25, Y: 290, W: 300, H: 440},
			sdl.FRect{X: 375, Y: 290, W: 1495, H: 440},
		},
	}

	for _, f := range faces {
		for _, scale := range []float32{1.0, 0.95, 0.8, 0.75, 0.5} {
			box, icon, text := f.numberBox, f.iconR, f.textR
			scaleRow(scale, &box, &text, &icon)

			if !contains(box, icon) {
				t.Errorf("%s at scale %v: icon %+v escaped number box %+v", f.name, scale, icon, box)
			}
			if !contains(box, text) {
				t.Errorf("%s at scale %v: text %+v overflows number box %+v", f.name, scale, text, box)
			}
		}
	}
}

// TestScaleRowPreservesProportions checks that scaling is a true similarity
// transform: every gap and size shrinks by exactly the scale factor, so the row
// looks identical apart from its size.
func TestScaleRowPreservesProportions(t *testing.T) {
	const scale = 0.5
	box := sdl.FRect{X: 530, Y: 25, W: 1380, H: 440}
	icon := sdl.FRect{X: 530, Y: 25, W: 300, H: 440}
	text := sdl.FRect{X: 830, Y: 25, W: 1080, H: 440}
	gapBefore := text.X - icon.X

	scaleRow(scale, &box, &text, &icon)

	if got, want := text.X-icon.X, gapBefore*scale; got != want {
		t.Errorf("icon-to-text gap: expected %v, got %v", want, got)
	}
	if got, want := box.W, float32(1380*scale); got != want {
		t.Errorf("box width: expected %v, got %v", want, got)
	}
	// The row must stay centred in the space it occupied.
	if got, want := box.X+box.W/2, float32(530+1380/2); got != want {
		t.Errorf("row centre moved: expected %v, got %v", want, got)
	}
}

// TestScaleRowNoOps checks the cases that must leave geometry untouched, so the
// default scale of 1.0 renders exactly as previous versions did.
func TestScaleRowNoOps(t *testing.T) {
	for _, scale := range []float32{1.0, 1.5, 0, -0.5} {
		box := sdl.FRect{X: 530, Y: 25, W: 1380, H: 440}
		icon := sdl.FRect{X: 530, Y: 25, W: 300, H: 440}
		want := box
		wantIcon := icon

		scaleRow(scale, &box, &icon)

		if box != want || icon != wantIcon {
			t.Errorf("scale %v changed geometry: box %+v icon %+v", scale, box, icon)
		}
	}
}

// TestTextClockScaleClamps checks that a hand-edited clock.ini cannot push the
// scale outside the range the web UI enforces, and that an unset option renders
// the historical layout rather than being clamped up to the minimum.
func TestTextClockScaleClamps(t *testing.T) {
	tests := []struct {
		configured float64
		want       float32
	}{
		{1.0, 1.0},
		{0.5, 0.5},
		{0.75, 0.75},
		{2.0, maxTextClockScale},
		{0.1, minTextClockScale},
		{0, maxTextClockScale},  // unset: must not shrink anything
		{-1, maxTextClockScale}, // nonsense: must not shrink anything
	}

	original := options.TextClockScale
	defer func() { options.TextClockScale = original }()

	for _, tc := range tests {
		options.TextClockScale = tc.configured
		if got := textClockScale(); got != tc.want {
			t.Errorf("TextClockScale=%v: expected %v, got %v", tc.configured, tc.want, got)
		}
	}
}

// TestTextClockScaleNormalisation documents the contract computeDerivedOptions
// relies on: whatever is in clock.ini, the value handed to the web form and
// written back must be inside the range the form's min/max accepts, or the
// browser rejects the entire config form on submit.
func TestTextClockScaleNormalisation(t *testing.T) {
	original := options.TextClockScale
	defer func() { options.TextClockScale = original }()

	for _, configured := range []float64{1.5, 2.0, 0.1, 0, -1, 0.5, 0.75, 1.0} {
		options.TextClockScale = configured
		normalised := float64(textClockScale())
		if normalised < minTextClockScale || normalised > maxTextClockScale {
			t.Errorf("TextClockScale=%v normalised to %v, outside %v-%v",
				configured, normalised, minTextClockScale, maxTextClockScale)
		}
	}
}
