package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/tarm/serial"
	"gitlab.com/Depili/go-rgb-led-matrix/bdf"
	"gitlab.com/Depili/go-rgb-led-matrix/matrix"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
	"image/color"
	"log"
	"math"
	"os"
	"os/signal"
	"strconv"
	"time"
)

var options clockOptions
var newOptions clockOptions

var colors struct {
	tally      color.RGBA
	text       color.RGBA
	countdown  color.RGBA
	background color.RGBA
}

var parser = flags.NewParser(&options, flags.Default)
var confChan chan bool

func main() {
	options.Config = func(s string) error {
		ini := flags.NewIniParser(parser)
		options.configFile = s
		return ini.ParseFile(s)
	}

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			panic(err)
		}
	}
	initColors()
	// Serial connection for the led ring
	serialConfig := serial.Config{
		Name: options.SerialName,
		Baud: options.SerialBaud,
		// ReadTimeout:    options.SerialTimeout,
	}

	serial, err := serial.OpenPort(&serialConfig)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Serial open.\n")

	// Parse font for clock text
	font, err := bdf.Parse(options.Font)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Fonts loaded.\n")

	// Initialize the led matrix library
	m := matrix.Init(options.Matrix, 32, 32)
	defer m.Close()

	// Trap SIGINT aka Ctrl-C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Create initial bitmaps for clock text
	hourBitmap := font.TextBitmap("15")
	minuteBitmap := font.TextBitmap("04")
	secondBitmap := font.TextBitmap("05")
	tallyBitmap := font.TextBitmap("  ")

	updateTicker := time.NewTicker(time.Millisecond * 10)
	send := make([]byte, 1)

	engine, err := clock.MakeEngine(options.EngineOptions)
	if err != nil {
		panic(err)
	}

	// Channel to notify for config changes
	confChan = make(chan bool, 1)

	if !options.DisableHTTP {
		go runHTTP()
	}

	fmt.Printf("Entering main loop\n")

	for {
		select {
		case <-sigChan:
			// SIGINT received, shutdown gracefully
			m.Close()
			os.Exit(1)
		case <-confChan:
			log.Printf("Reloading config!")
			engine.Close()
			log.Printf("Clock engine closed")
			newOptions.configFile = options.configFile
			options = newOptions

			initColors()
			log.Printf("Color init done")

			// Create main clock engine
			engine, err = clock.MakeEngine(options.EngineOptions)
			check(err)
			log.Printf("Config reloaded!")

		case <-updateTicker.C:
			state := engine.State()
			mainClock := state.Clocks[0]
			auxClock := state.Clocks[1]
			tally := ""
			hours := ""
			minutes := ""
			seconds := ""
			leds := 0

			// Normalize timers over 100 hours
			if mainClock.Hours > 99 {
				mainClock.Hours = 99
				mainClock.Minutes = 59
				mainClock.Seconds = 59
			}

			if mainClock.Text != "" {
				if mainClock.Mode == clock.LTC {
					tally = fmt.Sprintf(" %02d", mainClock.Hours)
					hours = fmt.Sprintf("%02d", mainClock.Minutes)
					minutes = fmt.Sprintf("%02d", mainClock.Seconds)
					seconds = fmt.Sprintf("%02d", mainClock.Frames)
					if options.EngineOptions.LTCSeconds {
						leds = mainClock.Seconds
					} else {
						leds = mainClock.Frames
					}
					colors.tally = colors.text

				} else if !mainClock.Hidden {
					// Non-LTC clocks
					hours = fmt.Sprintf("%02d", mainClock.Hours)

					minutes = fmt.Sprintf("%02d", mainClock.Minutes)
					seconds = fmt.Sprintf("%02d", mainClock.Seconds)
					leds = mainClock.Seconds

					if mainClock.HideSeconds && mainClock.Mode == clock.Normal {
						seconds = ""
					}

					// Shift counters with zero hours up on fields
					if mainClock.Mode != clock.Normal &&
						hours == "00" {
						hours = minutes
						minutes = seconds
						seconds = ""
					}

					// UDPTime overtime icon
					if mainClock.Icon == "+" {
						tmp := []rune(hours)
						tmp[0] = '+'
						hours = string(tmp)
					}

					if mainClock.Mode == clock.Countdown ||
						mainClock.Mode == clock.Media {
						if mainClock.Expired {
							// TODO: Multiple different options of expired timers?
							if !state.Flash {
								hours = ""
								minutes = ""
								seconds = ""
								leds = 59
							}
						} else {
							leds = int(math.Floor(mainClock.Progress * 59))
						}
					} else if mainClock.Mode == clock.Countup {
						leds, _ = strconv.Atoi(minutes)
					}
				}
			}

			if mainClock.Mode != clock.LTC {
				if state.Tally != "" {
					tally = fmt.Sprintf("%-.4s", state.Tally)
					colors.tally = color.RGBA{R: state.TallyColor.R, G: state.TallyColor.G, B: state.TallyColor.B, A: 255}

				} else if auxClock.Mode != clock.Normal && !auxClock.Hidden {
					if auxClock.Expired && auxClock.Mode == clock.Countdown {
						if state.Flash {
							tally = " 00"
						}
					} else {
						tally = auxClock.Compact
						colors.tally = colors.countdown
					}
				}
			}

			hourBitmap = font.TextBitmap(hours)
			minuteBitmap = font.TextBitmap(minutes)
			secondBitmap = font.TextBitmap(seconds)

			tallyBitmap = font.TextBitmap(tally)

			// Hardware signal color handling
			c := state.HardwareSignalColor
			if options.SignalFollow {
				// HW follows source 1
				c = state.Clocks[0].SignalColor
			}

			if options.SignalToBG {
				colors.background = c
			}

			tallyColor := [3]byte{byte(colors.tally.R), byte(colors.tally.G), byte(colors.tally.B)}
			textColor := [3]byte{byte(colors.text.R), byte(colors.text.G), byte(colors.text.B)}
			bgColor := [3]byte{byte(colors.background.R), byte(colors.background.G), byte(colors.background.B)}

			// Clear the matrix buffer
			m.Fill(bgColor)

			haveDisplay := (hours != "") && (minutes != "")
			if haveDisplay && (!mainClock.Paused || state.Flash) && (mainClock.Mode != clock.Off) {
				// Draw the dots between hours and minutes
				m.SetPixel(14, 15, textColor)
				m.SetPixel(14, 16, textColor)
				m.SetPixel(15, 15, textColor)
				m.SetPixel(15, 16, textColor)

				m.SetPixel(18, 15, textColor)
				m.SetPixel(18, 16, textColor)
				m.SetPixel(19, 15, textColor)
				m.SetPixel(19, 16, textColor)
			}
			// Draw the text
			m.Scroll(hourBitmap, textColor, 10, 0, 0, 14)
			m.Scroll(minuteBitmap, textColor, 10, 17, 0, 14)
			m.Scroll(secondBitmap, textColor, 21, 8, 0, 14)
			m.Scroll(tallyBitmap, tallyColor, 0, 2, 0, 30)

			// Update the led ring via serial
			send[0] = byte(leds)
			_, err := serial.Write(send)
			if err != nil {
				panic(err)
			}
			m.Send()
		}
	}
}

func initColors() {
	var err error
	colors.text, err = parseColor(options.TextColor)
	if err != nil {
		panic(err)
	}

	colors.countdown, err = parseColor(options.CountdownColor)
	if err != nil {
		panic(err)
	}

	colors.background, err = parseColor(options.BackgroundColor)
	if err != nil {
		panic(err)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// parseColor parses a string "#XXX or #XXXXXX to a sdl.Color"
func parseColor(s string) (c color.RGBA, err error) {
	c.A = 0xff
	switch len(s) {
	case 7:
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	case 4:
		_, err = fmt.Sscanf(s, "#%1x%1x%1x", &c.R, &c.G, &c.B)
		// Double the hex digits:
		c.R *= 17
		c.G *= 17
		c.B *= 17
	default:
		err = fmt.Errorf("parseColor(): invalid length, must be 7 or 4: %v", s)
	}
	return
}
