package main

import (
	"fmt"
	"image/color"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/Zyko0/go-sdl3/ttf"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
)

type outputLine struct {
	icon          string
	text          string
	label         string
	iconTex       *sdl.Texture
	textTex       *sdl.Texture
	labelTex      *sdl.Texture
	signalTex     *sdl.Texture
	timeFragments [10]*sdl.Texture
	fragmentRect  sdl.FRect
	colonTex      *sdl.Texture
	colonRect     sdl.FRect
}

var textClock struct {
	numberFont  *ttf.Font
	labelFont   *ttf.Font
	iconFont    *ttf.Font
	r           [4]outputLine
	glyphRegexp *regexp.Regexp
	tally       string
	tallyTex    *sdl.Texture
}

// Font sizes. Rpi <4 is limited to 2048x2048 texture size.
const (
	labelSize = 200
	iconSize  = 200
)

func initTextClock() {
	if textClock.numberFont != nil {
		textClock.numberFont.Close()
	}
	textClock.numberFont = openFont(options.NumberFont, options.NumberFontSize)

	if textClock.labelFont != nil {
		textClock.labelFont.Close()
	}
	textClock.labelFont = openFont(options.LabelFont, labelSize)

	if textClock.iconFont != nil {
		textClock.iconFont.Close()
	}
	textClock.iconFont = openFont(options.IconFont, iconSize)

	textClock.glyphRegexp = regexp.MustCompile(`^[\d:]+$`)
	preRenderFonts()

	if options.singleLine {
		numAudioSources = 1
	} else if options.Face == "text2" {
		numAudioSources = 2
	} else if options.Face == "text4" {
		numAudioSources = 4
	} else {
		numAudioSources = 3
	}

	log.Printf("Text clock face intialized.")
}

func openFont(file string, size int) *ttf.Font {
	if f, err := ttf.OpenFont(file, float32(size)); err != nil {
		panic(err)
	} else {
		return f
	}
}

func drawTextClock(state *clock.State) {
	colors.labelBG = toSDLColor(state.TitleBGColor)

	for i := 0; i < numAudioSources; i++ {
		clk := state.Clocks[i]
		colors.rowBG[i] = toSDLColor(clk.BGColor)

		if clk.Hidden {
			continue
		}

		text := clk.Text
		if options.HideHours && clk.Mode != clock.Normal {
			parts := strings.Split(text, ":")
			if len(parts) == 3 {
				text = parts[1] + ":" + parts[2]
			}
		}

		if clk.Expired && clk.Mode == clock.Countdown {
			if !state.Flash {
				text = " "
			}
		}

		renderNumbers(i, text, toSDLColor(clk.TextColor))
		titleColor := toSDLColor(state.TitleColor)
		if colors.label != titleColor {
			for row := range textClock.r {
				textClock.r[row].label = ""
			}
		}
		renderLabel(i, fmt.Sprintf("%.10s", clk.Label), titleColor)
		if !options.IconsDisable {
			renderIcon(i, clk.Icon, colors.row[i])
		}
		renderSignal(i, clk.SignalColor)
	}

	// Clear output and setup background
	prepareCanvas()

	if options.Face == "max" {
		drawMaxClock(state)
	} else if options.singleLine && !state.Clocks[0].Hidden {
		drawSingleLineClock(state)
	} else if options.Face == "text2" {
		draw2TextClocks(state)
	} else if options.Face == "text4" {
		draw4TextClocks(state)
	} else if !options.singleLine {
		draw3TextClocks(state)
	}

	drawTally(state)
}

func drawMaxClock(state *clock.State) {
	w, h, _ := renderer.CurrentOutputSize()

	topMargin := float32(10)
	rightMargin := float32(10)
	labelH := float32(h * 150 / 1080)
	labelW := float32(w * 900 / 1920)

	labelR := sdl.FRect{X: topMargin, Y: rightMargin, H: labelH, W: labelW}
	textR := sdl.FRect{X: topMargin, Y: rightMargin, H: float32(h) - topMargin, W: float32(w) - (rightMargin * 2)}

	if options.DrawBoxes {
		rectColor(&labelR, colors.labelBG)
	}

	copyIntoRect(textClock.r[0].labelTex, labelR)
	copyIntoRect(textClock.r[0].textTex, textR)
}

func drawSingleLineClock(state *clock.State) {
	labelR := sdl.FRect{X: 25, Y: 115, H: 150, W: 900}
	// 25px margin bellow label
	numberBox := sdl.FRect{X: 25, Y: 290, H: 440, W: 1920 - 50}
	iconR := sdl.FRect{X: 25, Y: 290, H: 440, W: 300}
	textR := sdl.FRect{X: 375, Y: 290, H: 440, W: 1920 - 425}

	if options.IconsDisable {
		textR = numberBox
	}

	signalR := sdl.FRect{X: 1920 - 170, Y: 115, H: 150, W: 150}

	if options.DrawBoxes {
		// Draw the placeholder boxes for timers and labels
		rectColor(&numberBox, colors.rowBG[0])
		rectColor(&labelR, colors.labelBG)
	}

	copyIntoRect(textClock.r[0].labelTex, labelR)
	copyIntoRect(textClock.r[0].signalTex, signalR)

	if state.Clocks[0].Mode != clock.LTC {
		// Clock time

		copyIntoRect(textClock.r[0].textTex, textR)
		if textClock.r[0].iconTex != nil {
			copyIntoRect(textClock.r[0].iconTex, iconR)
		}
	} else {
		// LTC
		// Maintain little spacing with the box borders
		numberBox.Y = numberBox.Y + 10
		numberBox.W = numberBox.W - 20

		copyIntoRect(textClock.r[0].textTex, numberBox)
	}
}

func draw3TextClocks(state *clock.State) {
	var x, y float32

	for i := 0; i < 3; i++ {
		if state.Clocks[i].Hidden {
			// Row is hidden
			continue
		}
		y = float32(25 + (365 * i))
		x = 530
		numberBox := sdl.FRect{X: x, Y: y, W: 1380, H: 300}
		textR := sdl.FRect{X: x + 300, Y: y, W: 1380 - 300, H: 300}
		if options.IconsDisable {
			textR = numberBox
		}
		iconR := sdl.FRect{X: x, Y: y, W: 300, H: 300}
		x = 10
		labelR := sdl.FRect{X: x, Y: y, W: 500, H: 100}
		signalR := sdl.FRect{X: iconR.X - 175, Y: y + 125, W: 150, H: 150}
		if options.DrawBoxes {
			// Draw the placeholder boxes for timers and labels
			rectColor(&numberBox, colors.rowBG[i])
			rectColor(&labelR, colors.labelBG)
		}

		copyIntoRect(textClock.r[i].signalTex, signalR)
		copyIntoRect(textClock.r[i].labelTex, labelR)

		if state.Clocks[i].Mode != clock.LTC {
			// Clock time

			copyIntoRect(textClock.r[i].textTex, textR)
			if textClock.r[i].iconTex != nil {
				copyIntoRect(textClock.r[i].iconTex, iconR)
			}

		} else {
			// LTC

			// Maintain little spacing with the box borders
			numberBox.Y = numberBox.Y + 10
			numberBox.W = numberBox.W - 20

			copyIntoRect(textClock.r[i].textTex, numberBox)
		}
	}
}

func draw2TextClocks(state *clock.State) {
	var x, y float32

	for i := 0; i < 2; i++ {
		if state.Clocks[i].Hidden {
			continue
		}
		y = float32(25 + (530 * i))
		x = 530
		numberBox := sdl.FRect{X: x, Y: y, W: 1380, H: 440}
		textR := sdl.FRect{X: x + 300, Y: y, W: 1380 - 300, H: 440}
		if options.IconsDisable {
			textR = numberBox
		}
		iconR := sdl.FRect{X: x, Y: y, W: 300, H: 440}
		x = 10
		labelR := sdl.FRect{X: x, Y: y, W: 500, H: 150}
		signalR := sdl.FRect{X: iconR.X - 175, Y: y + 170, W: 150, H: 150}
		if options.DrawBoxes {
			rectColor(&numberBox, colors.rowBG[i])
			rectColor(&labelR, colors.labelBG)
		}

		copyIntoRect(textClock.r[i].signalTex, signalR)
		copyIntoRect(textClock.r[i].labelTex, labelR)

		if state.Clocks[i].Mode != clock.LTC {
			copyIntoRect(textClock.r[i].textTex, textR)
			if textClock.r[i].iconTex != nil {
				copyIntoRect(textClock.r[i].iconTex, iconR)
			}
		} else {
			numberBox.Y = numberBox.Y + 10
			numberBox.W = numberBox.W - 20
			copyIntoRect(textClock.r[i].textTex, numberBox)
		}
	}
}

func draw4TextClocks(state *clock.State) {
	var x, y float32

	for i := 0; i < 4; i++ {
		if state.Clocks[i].Hidden {
			continue
		}
		y = float32(5 + (268 * i))
		x = 530
		numberBox := sdl.FRect{X: x, Y: y, W: 1380, H: 255}
		textR := sdl.FRect{X: x + 300, Y: y, W: 1380 - 300, H: 255}
		if options.IconsDisable {
			textR = numberBox
		}
		iconR := sdl.FRect{X: x, Y: y, W: 300, H: 255}
		x = 10
		labelR := sdl.FRect{X: x, Y: y, W: 500, H: 85}
		signalR := sdl.FRect{X: iconR.X - 175, Y: y + 68, W: 120, H: 120}
		if options.DrawBoxes {
			rectColor(&numberBox, colors.rowBG[i])
			rectColor(&labelR, colors.labelBG)
		}

		copyIntoRect(textClock.r[i].signalTex, signalR)
		copyIntoRect(textClock.r[i].labelTex, labelR)

		if state.Clocks[i].Mode != clock.LTC {
			copyIntoRect(textClock.r[i].textTex, textR)
			if textClock.r[i].iconTex != nil {
				copyIntoRect(textClock.r[i].iconTex, iconR)
			}
		} else {
			numberBox.Y = numberBox.Y + 10
			numberBox.W = numberBox.W - 20
			copyIntoRect(textClock.r[i].textTex, numberBox)
		}
	}
}

func copyIntoRect(t *sdl.Texture, r sdl.FRect) {
	if t == nil {
		return
	}
	w, h, err := t.Size()
	if err != nil {
		debug.Printf("copyIntoRect: %v", err)
		return
	}
	dest := centerRect(w, h, r)
	if dest.W <= 0 || dest.H <= 0 {
		return
	}
	renderer.RenderTexture(t, nil, &dest)
}

func renderText(text string, font *ttf.Font, color sdl.Color) *sdl.Texture {
	if text == "" {
		text = " "
	}

	t, err := font.RenderTextBlended(text, color)
	if err != nil {
		log.Printf("renderText RenderUTF8Blended error: %v", err)
		log.Printf("rendering error text")
		t, err = font.RenderTextBlended("INVALID TEXT", color)
		check(err)
	}

	tex, err := renderer.CreateTextureFromSurface(t)
	if err != nil {
		log.Printf("renderText CreateTextureFromSurface error: %v", err)
		log.Printf("rendering error text")
		t.Destroy()
		t, err = font.RenderTextBlended("INVALID TEXT", color)
		check(err)

		tex, err = renderer.CreateTextureFromSurface(t)
		check(err)
	}
	t.Destroy()
	t = nil
	tex.SetAlphaMod(color.A)
	tex.SetBlendMode(sdl.BLENDMODE_BLEND)
	return tex
}

func preRenderFonts() {
	log.Printf("Precalcs!")
	for row := range textClock.r {
		preRenderRowFont(row)
	}
}

func preRenderRowFont(row int) {
	log.Printf("Updating row %d glyphs", row)
	for i := range textClock.r[row].timeFragments {
		text := fmt.Sprintf("%01d", i)
		if textClock.r[row].timeFragments[i] != nil {
			textClock.r[row].timeFragments[i].Destroy()
		}

		textClock.r[row].timeFragments[i] = renderText(text, textClock.numberFont, colors.row[row])
		textClock.r[row].timeFragments[i].SetAlphaMod(colors.row[row].A)
	}
	w, h, _ := textClock.r[row].timeFragments[0].Size()
	textClock.r[row].fragmentRect = sdl.FRect{X: 0, Y: 0, W: w, H: h}

	if textClock.r[row].colonTex != nil {
		textClock.r[row].colonTex.Destroy()
	}
	textClock.r[row].colonTex = renderText(":", textClock.numberFont, colors.row[row])
	textClock.r[row].colonTex.SetAlphaMod(colors.row[row].A)
	w, h, _ = textClock.r[row].colonTex.Size()
	textClock.r[row].colonRect = sdl.FRect{X: 0, Y: 0, W: w, H: h}
}

func createRowTexture(i int, text string) {
	var texW, texH float32
	var err error

	texH = textClock.r[i].fragmentRect.H
	// Calculate string width
	for _, ch := range text {
		if ch == ':' {
			texW += textClock.r[i].colonRect.W
		} else {
			texW += textClock.r[i].fragmentRect.W
		}
	}

	if textClock.r[i].textTex != nil {
		textClock.r[i].textTex.Destroy()
	}

	textClock.r[i].textTex, err = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, int(texW), int(texH))
	textClock.r[i].textTex.SetBlendMode(sdl.BLENDMODE_BLEND)
	check(err)
	renderTarget(textClock.r[i].textTex)
	renderer.SetDrawColor(0, 0, 0, 0)
	renderer.Clear()
	renderTarget(nil)
}

func renderFromGlyphs(i int, text string) {
	target := sdl.FRect{}
	target.H = textClock.r[i].fragmentRect.H
	renderTarget(textClock.r[i].textTex)

	for _, ch := range text {
		if num, err := strconv.Atoi(string(ch)); err == nil {
			target.W = textClock.r[i].fragmentRect.W
			renderer.RenderTexture(textClock.r[i].timeFragments[num], nil, &target)
			target.X += target.W
		} else {
			target.W = textClock.r[i].colonRect.W
			renderer.RenderTexture(textClock.r[i].colonTex, nil, &target)
			target.X += target.W
		}
	}
	renderTarget(nil)
}

func renderNumbers(i int, text string, textColor sdl.Color) {
	if textColor != colors.row[i] {
		colors.row[i] = textColor
		preRenderRowFont(i)
		// Force redrawing of the text
		textClock.r[i].text = " "
	}

	if textClock.r[i].text != text {
		textClock.r[i].text = text
		if textClock.r[i].textTex != nil {
			textClock.r[i].textTex.Destroy()
		}

		if textClock.glyphRegexp.MatchString(text) {
			// Fast text with prerendered glyphs
			createRowTexture(i, text)
			renderFromGlyphs(i, text)
		} else {
			textClock.r[i].textTex = renderText(text, textClock.numberFont, colors.row[i])
		}
	}
}

func renderIcon(row int, icon string, textColor sdl.Color) {
	var err error
	icon = materialIcon(icon)
	if textClock.r[row].icon != icon || colors.icon[row] != textColor {
		colors.icon[row] = textColor
		textClock.r[row].icon = icon
		if textClock.r[row].iconTex != nil {
			textClock.r[row].iconTex.Destroy()
		}
		if icon != "" {
			textClock.r[row].iconTex = renderText(icon, textClock.iconFont, colors.icon[row])
		} else {
			renderer.SetDrawColor(0, 0, 0, 0)
			textClock.r[row].iconTex, err = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, 1, 1)
			check(err)
			err = textClock.r[row].iconTex.SetBlendMode(sdl.BLENDMODE_BLEND)
			check(err)
			err = textClock.r[row].iconTex.SetAlphaMod(0)
			check(err)
		}
	}
}

func renderSignal(i int, newColor color.RGBA) {
	var err error
	c := toSDLColor(newColor)
	if c != colors.signal[i] {
		if textClock.r[i].signalTex != nil {
			textClock.r[i].signalTex.Destroy()
		}
		renderer.SetDrawColor(0, 0, 0, 0)
		textClock.r[i].signalTex, err = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, 150, 150)
		check(err)
		err = textClock.r[i].signalTex.SetBlendMode(sdl.BLENDMODE_BLEND)
		check(err)
		renderTarget(textClock.r[i].signalTex)
		renderer.Clear()
		filledCircle(75, 75, 74, c)
		colors.signal[i] = c
	}
}

func renderLabel(i int, label string, textColor sdl.Color) {
	if textClock.r[i].label != label {

		colors.label = textColor
		textClock.r[i].label = label
		if textClock.r[i].labelTex != nil {
			textClock.r[i].labelTex.Destroy()
		}

		textClock.r[i].labelTex = renderText(label, textClock.labelFont, colors.label)
	}
}

func drawTally(state *clock.State) {
	// Draw possible OSC text message
	if state.Tally != "" {
		tallyColor := toSDLColor(*state.TallyColor)
		bgColor := toSDLColor(*state.TallyColor)

		if textClock.tally != state.Tally ||
			colors.tally != tallyColor ||
			colors.tallyBG != bgColor {
			if textClock.tallyTex != nil {
				textClock.tallyTex.Destroy()
			}
			textClock.tally = state.Tally
			colors.tally = tallyColor
			colors.tallyBG = bgColor
			tally := fmt.Sprintf("%.16s", state.Tally)

			textClock.tallyTex = renderText(tally, textClock.labelFont, colors.tally)
			textClock.tallyTex.SetBlendMode(sdl.BLENDMODE_BLEND)
			textClock.tallyTex.SetAlphaMod(colors.tally.A)
		}

		tallyRect := sdl.FRect{X: 10, Y: 25 + (365 * 2), W: 1920 - 20, H: 300}
		if options.Face == "text2" {
			tallyRect = sdl.FRect{X: 10, Y: 25 + (530 * 1), W: 1920 - 20, H: 440}
		} else if options.Face == "text4" {
			tallyRect = sdl.FRect{X: 10, Y: 10 + (265 * 3), W: 1920 - 20, H: 240}
		} else if options.singleLine {
			tallyRect.X = 25
			tallyRect.W = 1920 - 50
		}

		rectColor(&tallyRect, colors.tallyBG)
		copyIntoRect(textClock.tallyTex, tallyRect)
	}
}

func centerRect(w, h float32, r sdl.FRect) sdl.FRect {
	if r.W <= 0 || r.H <= 0 || w <= 0 || h <= 0 {
		return sdl.FRect{}
	}
	if r.W <= 0 || r.H <= 0 || w <= 0 || h <= 0 {
		return sdl.FRect{}
	}
	dest := sdl.FRect{}
	rSource := float64(w) / float64(h)
	rDest := float64(r.W) / float64(r.H)
	if rSource < rDest {
		dest.W = w * r.H / h
		dest.H = r.H
	} else {
		dest.W = r.W
		dest.H = h * r.W / w
	}
	dest.X = r.X + ((r.W - dest.W) / 2)
	dest.Y = r.Y + ((r.H - dest.H) / 2)
	return dest
}

// Substitute unicode glyphs used for icons to material design icon font private glyphs
func materialIcon(icon string) string {
	switch icon {
	case clock.IconPaused:
		return "\ue034"
	case clock.IconCountdown:
		return "\ue5db"
	case clock.IconCountup:
		return "\ue5d8"
	case clock.IconLooping:
		return "\ue040"
	case clock.IconPlaying:
		return "\ue037"
	case clock.IconOvertime:
		return "\ue145"
	case clock.IconTarget:
		return "\ue044"
	case clock.IconPlayPause:
		return "\ue044"
	case clock.IconRecord:
		return "\ue061"
	case clock.IconNegative:
		return "\ue15b"
	}
	return ""
}
