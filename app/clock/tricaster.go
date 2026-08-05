package clock

import (
	"github.com/desertbit/timer"
	"github.com/jpkelly/clock8002/app/debug"
	"github.com/jpkelly/clock8002/app/tricaster"
	"image/color"
	"log"
	"strconv"
	"strings"
	"time"
)

// TricasterOptions contains the options for go-flags
// for the tricaster module.
type TricasterOptions struct {
	TricasterEnabled    bool   `long:"tricaster-enabled" description:"Enable the tricaster integration"`
	TricasterAddress    string `long:"tricaster-address" description:"Tricaster address to connect to."`
	TricasterUser       string `long:"tricaster-username" description:"Tricaster http username" default:"admin"`
	TricasterPassword   string `long:"tricaster-password" description:"Tricaster http password" default:"admin"`
	TricasterTimer      int    `long:"tricaster-timer" description:"Timer for tricaster DDR state" default:"5"`
	TricasterPVM        bool   `long:"tricaster-show-pvm" description:"Do not ingore media playing in tricaster preview"`
	TricasterEvent      bool   `long:"tricaster-event-enabled" description:"Show time until next tricaster event on a timer"`
	TricasterEventTimer int    `long:"tricaster-event-timer" description:"Timer for tricaster event time" default:"5"`
	TricasterMediaName  bool   `long:"tricaster-media-name" description:"Show tricaster media name as clock text"`
	TricasterMediaColor string `long:"tricaster-media-color" description:"CSS color for tricaster media name" default:"#FF8000"`
	TricasterMediaBG    string `long:"tricaster-media-bg" description:"CSS color for tricaster media name background" default:"#101010"`
	TricasterInterval   int    `long:"tricaster-interval" description:"Tricaster state polling interval, in milliseconds" default:"100"`
	TricasterTimeout    int    `long:"tricaster-timeout" description:"Tricaster communication timeout, in milliseconds" default:"1000"`
}

type tricasterState struct {
	counter      *Counter
	eventCounter *Counter
	event        bool
	pvm          bool
	mediaName    bool
	mediaColor   *color.RGBA
	mediaBG      *color.RGBA
	interval     time.Duration
	timeout      time.Duration
}

func (engine *Engine) initTricaster(options *EngineOptions) {
	if !options.TricasterEnabled {
		return
	}
	var err error
	log.Printf("Tricaster: Initializing")

	s := tricasterState{
		counter:      engine.Counters[options.TricasterTimer],
		eventCounter: engine.Counters[options.TricasterEventTimer],
		event:        options.TricasterEvent,
		pvm:          options.TricasterPVM,
		mediaName:    options.TricasterMediaName,
		interval:     time.Duration(options.TricasterInterval) * time.Millisecond,
		timeout:      time.Duration(options.TricasterTimeout) * time.Millisecond,
	}
	s.mediaColor, err = parseColor(options.TricasterMediaColor)
	if err != nil {
		log.Fatalf("Error parsing tricaster media color: %v", err)
	}
	s.mediaBG, err = parseColor(options.TricasterMediaBG)
	if err != nil {
		log.Fatalf("Error parsing tricaster media background color: %v", err)
	}
	engine.tricaster = s

	log.Printf("Tricaster: connecting to %s", options.TricasterAddress)
	c := tricaster.Connect(engine.ctx, options.TricasterAddress, options.TricasterUser, options.TricasterPassword, s.interval)
	go engine.tricasterListen(c)
}

func (engine *Engine) tricasterListen(c chan *tricaster.Status) {
	log.Printf("Tricaster: listening")
	t := timer.NewTimer(engine.tricaster.timeout)
	for {
		select {
		case s, ok := <-c:
			if !ok {
				log.Printf("Tricaster: listener channel closed, shutting down.")
				engine.tricaster.counter.ResetMedia()
				if engine.tricaster.event {
					engine.tricaster.eventCounter.ResetSlave()
				}
				return
			}
			var playing *tricaster.DDR
			for _, d := range s.DDR {
				if !d.Static && (d.Live || (d.Preview && engine.tricaster.pvm)) {
					playing = &d
					break
				}
			}

			if playing != nil {
				engine.tricasterSet(playing)
			} else {
				// No playing video matching the current filter
				engine.tricaster.counter.ResetMedia()
			}
			if engine.tricaster.event {
				parts := strings.Split(s.NextEventTime, ":")
				if len(parts) != 3 {
					continue
					debug.Printf("Tricaster: error splitting next event time")
				}
				hours, err := strconv.Atoi(parts[0][1:])
				if err != nil {
					debug.Printf("Tricaster: error converting event time hours: %v", err)
					continue
				}
				minutes, err := strconv.Atoi(parts[1])
				if err != nil {
					debug.Printf("Tricaster: error converting event time minutes: %v", err)
					continue
				}
				seconds, err := strconv.Atoi(parts[2])
				if err != nil {
					debug.Printf("Tricaster: error converting event time seconds: %v", err)
					continue
				}
				engine.tricaster.eventCounter.SetSlave(hours, minutes, seconds, false, IconCountdown)
			}
		case <-t.C:
			// Timeout on communications, reset the output
			engine.tricaster.counter.ResetMedia()
			if engine.tricaster.event {
				engine.tricaster.eventCounter.ResetSlave()
			}
		}
	}
}

func (engine *Engine) tricasterSet(playing *tricaster.DDR) {
	engine.tricaster.counter.mediaUpdate(playing)

	if engine.tricaster.mediaName {
		name := playing.MediaName()
		if name == "" {
			return
		}

		engine.message = name
		engine.messageColor = engine.tricaster.mediaColor
		engine.messageBG = engine.tricaster.mediaBG
		engine.oscTally = true
		engine.messageTimer.Reset(engine.tricaster.timeout)
	}
}
