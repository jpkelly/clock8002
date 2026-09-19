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

// TestLabelFontSizeFallback checks that a non-positive label-size falls back to
// the default instead of reaching openFont, which panics on a size SDL_ttf
// rejects — a hand-edited label-size=0 must not take the clock down at startup.
func TestLabelFontSizeFallback(t *testing.T) {
	tests := []struct {
		configured int
		want       int
	}{
		{200, 200},
		{120, 120},
		{1, 1},
		{0, defaultLabelSize},
		{-50, defaultLabelSize},
	}

	original := options.LabelFontSize
	defer func() { options.LabelFontSize = original }()

	for _, tc := range tests {
		options.LabelFontSize = tc.configured
		if got := labelFontSize(); got != tc.want {
			t.Errorf("LabelFontSize=%d: expected %d, got %d", tc.configured, tc.want, got)
		}
	}
}

// TestOverrideLabelRect covers the label rect overrides: label-w gates the
// whole feature, height falls back rather than collapsing, and Y is refused on
// the multi-row faces where it would stack every label on the first row.
func TestOverrideLabelRect(t *testing.T) {
	// The built-in text2 label rect, as draw2TextClocks builds it for row 1.
	base := sdl.FRect{X: 10, Y: 555, W: 500, H: 150}

	tests := []struct {
		name       string
		x, y, w, h int
		useY       bool
		want       sdl.FRect
	}{
		{
			name: "disabled when label-w is 0",
			x:    999, y: 999, w: 0, h: 999, useY: true,
			want: base,
		},
		{
			name: "width and height applied",
			x:    0, y: 0, w: 700, h: 200, useY: false,
			want: sdl.FRect{X: 0, Y: 555, W: 700, H: 200},
		},
		{
			name: "height of 0 keeps the built-in height",
			x:    40, y: 0, w: 700, h: 0, useY: false,
			want: sdl.FRect{X: 40, Y: 555, W: 700, H: 150},
		},
		{
			name: "x of 0 means flush left, not unset",
			x:    0, y: 0, w: 600, h: 0, useY: false,
			want: sdl.FRect{X: 0, Y: 555, W: 600, H: 150},
		},
		{
			name: "y ignored on multi-row faces",
			x:    10, y: 300, w: 500, h: 150, useY: false,
			want: base,
		},
		{
			name: "y applied on the single-line face",
			x:    10, y: 300, w: 500, h: 150, useY: true,
			want: sdl.FRect{X: 10, Y: 300, W: 500, H: 150},
		},
		{
			// Consistent with x: once label-w enables the override, a y of 0
			// means the top edge, not "leave the built-in position alone".
			name: "y of 0 means flush top, not unset",
			x:    10, y: 0, w: 500, h: 150, useY: true,
			want: sdl.FRect{X: 10, Y: 0, W: 500, H: 150},
		},
	}

	origX, origY := options.LabelX, options.LabelY
	origW, origH := options.LabelW, options.LabelH
	defer func() {
		options.LabelX, options.LabelY = origX, origY
		options.LabelW, options.LabelH = origW, origH
	}()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			options.LabelX, options.LabelY = tc.x, tc.y
			options.LabelW, options.LabelH = tc.w, tc.h
			got := base
			overrideLabelRect(&got, tc.useY)
			if got != tc.want {
				t.Errorf("expected %+v, got %+v", tc.want, got)
			}
		})
	}
}

// TestOverrideLabelRectDefaultsAreInert is the compatibility guarantee: the
// shipped defaults must leave every face's label exactly where it was.
func TestOverrideLabelRectDefaultsAreInert(t *testing.T) {
	origX, origY := options.LabelX, options.LabelY
	origW, origH := options.LabelW, options.LabelH
	defer func() {
		options.LabelX, options.LabelY = origX, origY
		options.LabelW, options.LabelH = origW, origH
	}()
	options.LabelX, options.LabelY, options.LabelW, options.LabelH = 0, 0, 0, 0

	for _, base := range []sdl.FRect{
		{X: 25, Y: 115, W: 900, H: 150}, // single
		{X: 10, Y: 10, W: 500, H: 100},  // text
		{X: 10, Y: 25, W: 500, H: 150},  // text2
		{X: 10, Y: 40, W: 500, H: 80},   // text4
	} {
		for _, useY := range []bool{true, false} {
			got := base
			overrideLabelRect(&got, useY)
			if got != base {
				t.Errorf("defaults changed %+v to %+v (useY=%v)", base, got, useY)
			}
		}
	}
}
