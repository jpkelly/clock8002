package main

import (
	"fmt"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/Zyko0/go-sdl3/ttf"

	// "gitlab.com/Depili/clock-8001/v4/clock"
	"log"
	"math"
	"time"

	"github.com/jpkelly/clock8002/app/debug"
)

var countdown struct {
	largeFont *ttf.Font
	smallFont *ttf.Font
	bgColor   sdl.Color
	color     sdl.Color
	target    time.Time
	loc       *time.Location
}

func initCountdown() {
	log.Printf("Initializing countdown display...")

	var f *ttf.Font
	var err error

	if countdown.smallFont != nil {
		countdown.smallFont.Close()
	}

	if f, err = ttf.OpenFont(options.NumberFont, 250); err != nil {
		panic(err)
	}
	countdown.smallFont = f

	if countdown.largeFont != nil {
		countdown.largeFont.Close()
	}

	if f, err = ttf.OpenFont(options.NumberFont, 350); err != nil {
		panic(err)
	}
	countdown.largeFont = f

	countdown.loc, err = time.LoadLocation(options.EngineOptions.Source1.TimeZone)
	check(err)

	countdown.target, err = time.ParseInLocation("2006-01-02 15:04:05", options.CountdownTarget, countdown.loc)
	check(err)
	log.Printf("Target: %v", countdown.target)

	countdown.color = sdl.Color{R: 255, G: 255, B: 255, A: 255}
	countdown.bgColor = sdl.Color{R: 0, G: 0, B: 0, A: 255}

	log.Printf("Countdown face intialized.")
}

func drawCountdown() {
	debug.Printf("drawCountdown")
	var t time.Duration

	if options.Countup {
		t = time.Now().In(countdown.loc).Sub(countdown.target)
	} else {
		t = countdown.target.Sub(time.Now().In(countdown.loc))
	}

	hours := t.Truncate(time.Hour).Hours()
	minutes := t.Truncate(time.Minute).Minutes() - (hours * 60)
	seconds := t.Truncate(time.Second).Seconds() - (hours * 60 * 60) - (minutes * 60)
	days := math.Floor(hours / 24)
	hours -= days * 24

	if t.Seconds() < 0 {
		days = 0.0
		hours = 0.0
		minutes = 0.0
		seconds = 0.0
	}

	years := 0

	for i := 0; true; {
		if y := time.Now().Add(time.Duration(-i*24) * time.Hour).Year(); y%400 == 0 || y%4 == 0 && y%100 != 0 {
			i = 366
		} else {
			i = 365
		}
		if int(days) > i {
			years++
			days -= float64(i)
		} else {
			break
		}
	}

	yearString := fmt.Sprintf("%d", years)
	if years == 0 {
		yearString = " "
	}

	yearTex := renderText(yearString, countdown.largeFont, countdown.color)
	defer yearTex.Destroy()

	dayTex := renderText(fmt.Sprintf("%.0f", days), countdown.largeFont, countdown.color)
	defer dayTex.Destroy()

	lineTex := renderText(fmt.Sprintf("%02.0f:%02.0f:%02.0f", hours, minutes, seconds), countdown.smallFont, countdown.color)
	defer lineTex.Destroy()

	prepareCanvas()

	yearRect := sdl.FRect{Y: 5, H: 400, X: (1920 / 2) - 300, W: 600}
	dayRect := sdl.FRect{Y: (1080 / 2) - 200, H: 400, X: (1920 / 2) - 300, W: 600}
	lineRect := sdl.FRect{X: 10, Y: 755, H: 300, W: 1920 - 20}
	copyIntoRect(yearTex, yearRect)
	copyIntoRect(dayTex, dayRect)
	copyIntoRect(lineTex, lineRect)
}
