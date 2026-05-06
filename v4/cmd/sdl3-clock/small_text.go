package main

import (
	"fmt"
	"github.com/Zyko0/go-sdl3/sdl"
	"gitlab.com/Depili/go-rgb-led-matrix/bdf"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
	"regexp"
)

var smallTextClock struct {
	font        *bdf.Bdf
	r           [3]outputLine
	glyphRegexp *regexp.Regexp
	tally       string
}

const (
	smallTextPixelSize    = 2
	smallTextPixelSpacing = 3
	smallTextLineSpacing  = 21
	smallTextTop          = 5
	smallTextLabelY       = 4
	smallTextSignalY      = 29
	smallTextIconY        = smallTextSignalY + 7
	smallTextClockY       = smallTextIconY + 7 + 3
	smallTextColonW       = 3 * 4
	smallTextH            = 9 * 3
)

func initSmallTextClock() {
	var err error

	smallTextClock.font, err = bdf.Parse(options.Font)
	if err != nil {
		panic(err)
	}

	gridStartX = 0
	gridStartY = 0
	gridSize = smallTextPixelSize
	gridSpacing = smallTextPixelSpacing
}

func drawSmallTextClock(state *clock.State) {
	colors.labelBG = toSDLColor(state.TitleBGColor)
	font := smallTextClock.font
	// A circle
	signalBitmap := [][]bool{
		{false, false, false, false, false, false, false},
		{false, false, false, false, false, false, false},
		{false, false, false, false, false, false, false},
		{false, false, false, false, false, false, false},
		{false, false, true, true, true, true, false},
		{false, true, true, true, true, true, true},
		{false, true, true, true, true, true, true},
		{false, true, true, true, true, true, true},
		{false, true, true, true, true, true, true},
		{false, false, true, true, true, true, false},
		{false, false, false, false, false, false, false},
	}

	// Ugly global variables for the bitmap drawing functions
	gridStartY = smallTextTop
	gridStartX = 1

	// Clear output and setup background
	prepareCanvas()

	for i := range smallTextClock.r {
		clk := state.Clocks[i]
		if clk.Hidden {
			continue
		}

		colors.rowBG[i] = toSDLColor(clk.BGColor)
		labelColor := toSDLColor(state.TitleColor)
		labelBG := toSDLColor(state.TitleBGColor)
		textColor := toSDLColor(clk.TextColor)

		ltc, hours, minutes, seconds := normalizeFields(clk, state)

		// Generate the text bitmaps
		labelBitmap := font.TextBitmap(fmt.Sprintf("%.4s", clk.Label))
		hourBitmap := font.TextBitmap(hours)
		minuteBitmap := font.TextBitmap(minutes)
		secondBitmap := font.TextBitmap(seconds)
		ltcBitmap := font.TextBitmap(ltc)
		iconBitmap := font.TextBitmap(clk.Icon)

		if options.DrawBoxes {
			h := float32(12*3 + 2)
			titleW := float32(2 + (7 * smallTextPixelSpacing * 4))
			numX := float32(smallTextIconY*smallTextPixelSpacing) + gridStartX + 4

			if clk.Mode == clock.LTC {
				numX -= 7 * smallTextPixelSpacing
			}

			rect := sdl.FRect{X: gridStartX - 2, Y: gridStartY - 1, H: h, W: titleW}

			rectColor(&rect, labelBG)

			rect.X = numX - 2
			rect.W = 288

			rectColor(&rect, colors.rowBG[i])
		}

		drawBitmask(labelBitmap, labelColor, 0, 0)
		drawBitmask(hourBitmap, textColor, 0, smallTextClockY)
		drawBitmask(minuteBitmap, textColor, 0, smallTextClockY+14+3)
		drawBitmask(secondBitmap, textColor, 0, smallTextClockY+28+6)
		if ltc != "" {
			drawBitmask(ltcBitmap, textColor, 0, smallTextSignalY)
		} else {
			drawBitmask(signalBitmap, toSDLColor(clk.SignalColor), 0, smallTextSignalY)
			drawBitmask(iconBitmap, textColor, 0, smallTextIconY)
		}

		// Dots between hours, minutes and seconds (+ franes for LTC)
		if (!clk.Paused || state.Flash) && (clk.Mode != clock.Off) {
			dots := 2
			firstY := smallTextClockY + 1
			if clk.Mode == clock.LTC {
				dots = 3
				firstY = smallTextSignalY + 1
			}
			firstY += 14
			for i = 0; i < dots; i++ {
				drawDots(4, firstY, textColor)
				firstY += 14 + 3
			}
		}

		gridStartY += smallTextLineSpacing + smallTextH
	}

	// Get the last line coordinates, undoing the last addition in loop
	gridStartY -= smallTextLineSpacing + smallTextH

	if state.Tally != "" {
		tallyColor := toSDLColor(*state.TallyColor)
		bgColor := toSDLColor(*state.TallyBG)

		h := float32(12*3 + 2)

		rect := sdl.FRect{X: 0, Y: gridStartY - 1, W: 288, H: gridStartY + h}
		rectColor(&rect, bgColor)

		tallyBitmap := font.TextBitmap(state.Tally)
		drawBitmask(tallyBitmap, tallyColor, 0, 0)
	}
}

// Get the string representation of all the clock fields, normalizing fields over 99 to 99
func normalizeFields(clk *clock.Clock, state *clock.State) (ltc string, hours string, minutes string, seconds string) {
	// Normalize timers over 100 hours
	if clk.Hours > 99 {
		clk.Hours = 99
		clk.Minutes = 59
		clk.Seconds = 59
	}

	if clk.Text != "" {
		if clk.Mode == clock.LTC {
			ltc = fmt.Sprintf(" %02d", clk.Hours)
			hours = fmt.Sprintf("%02d", clk.Minutes)
			minutes = fmt.Sprintf("%02d", clk.Seconds)
			seconds = fmt.Sprintf("%02d", clk.Frames)

		} else {
			// Non-LTC clocks
			hours = fmt.Sprintf("%02d", clk.Hours)

			minutes = fmt.Sprintf("%02d", clk.Minutes)
			seconds = fmt.Sprintf("%02d", clk.Seconds)

			if clk.HideSeconds && clk.Mode == clock.Normal {
				seconds = ""
			}

			if clk.Mode == clock.Countdown ||
				clk.Mode == clock.Media {
				if clk.Expired {
					// TODO: Multiple different options of expired timers?
					if !state.Flash {
						hours = ""
						minutes = ""
						seconds = ""
					}
				}
			} else if clk.Mode == clock.Countup {
				if clk.Expired {
					hours = "00"
					minutes = "00"
					seconds = "00"
				}
			}
		}
	}
	return
}
