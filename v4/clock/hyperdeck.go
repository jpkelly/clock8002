package clock

import (
	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/hyperdeck"
	"image/color"
	"log"
	"time"
)

// HyperdeckOptions contains the options for go-flags for the hyperdeck module
type HyperdeckOptions struct {
	HyperdeckEnabled    bool   `long:"hyperdeck-enabled" description:"Enable hyperdeck integration"`
	HyperdeckAddress    string `long:"hyperdeck-address" desctiption:"IP address of the hyperdeck"`
	HyperdeckRelay      bool   `long:"hyperdeck-relay" description:"Relay commands to this hyperdeck on port 9993"`
	HyperdeckTimer      int    `long:"hyperdeck-timer" description:"Timer to use for hyperdeck status" default:"8"`
	HyperdeckMediaName  bool   `long:"hyperdeck-media-name" description:"Show hyperdeck media name as clock text"`
	HyperdeckMediaColor string `long:"hyperdeck-media-color" description:"CSS color for hyperdeck media name" default:"#FF8000"`
	HyperdeckMediaBG    string `long:"hyperdeck-media-bg" description:"CSS color for hyperdeck media name background" default:"#101010"`
	HyperdeckTimeout    int    `long:"hyperdeck-timeout" description:"Hyperdeck communication timeout, in milliseconds" default:"1000"`
}

type hyperdeckUpdate struct {
	head             time.Duration
	duration         time.Duration
	playlistHead     time.Duration
	playlistDuration time.Duration
	name             string
	play             bool
	loop             bool
	record           bool
	playlist         bool
	currentClip      int
	totalClips       int
}

type hyperdeckState struct {
	ip         string
	relay      bool
	counter    *Counter
	mediaName  bool
	mediaColor *color.RGBA
	mediaBG    *color.RGBA
	timeout    time.Duration
}

func (engine *Engine) hyperdeckInit(options *EngineOptions) {
	var err error

	if !options.HyperdeckEnabled {
		return
	}

	s := hyperdeckState{
		ip:        options.HyperdeckAddress,
		relay:     options.HyperdeckRelay,
		counter:   engine.Counters[options.HyperdeckTimer],
		mediaName: options.HyperdeckMediaName,
		timeout:   time.Duration(options.HyperdeckTimeout) * time.Millisecond,
	}
	s.mediaColor, err = parseColor(options.HyperdeckMediaColor)
	if err != nil {
		log.Fatalf("Error parsing hyperdeck media color: %v", err)
	}
	s.mediaBG, err = parseColor(options.HyperdeckMediaBG)
	if err != nil {
		log.Fatalf("Error parsing hyperdeck media background color: %v", err)
	}
	engine.hyperdeck = s
	go engine.hyperdeckListen()
}

func (engine *Engine) hyperdeckListen() {
	engine.wg.Add(1)
	for {
		log.Printf("Hyperdeck: connecting to %s relay: %v", engine.hyperdeck.ip, engine.hyperdeck.relay)
		addr := engine.hyperdeck.ip + ":9993"
		c, err := hyperdeck.Listen(engine.ctx, addr, engine.hyperdeck.relay)
		if err != nil {
			log.Printf("Hyperdeck listen error: %v", err)
			continue
		}

		t := timer.NewTimer(engine.hyperdeck.timeout)

		for {
			select {
			case <-engine.ctx.Done():
				log.Printf("Hyperdeck: terminating listener on request")
				engine.wg.Done()
				return
			case s, ok := <-c:
				if !ok {
					log.Printf("Hyperdeck: listener channel closed, retrying")
					engine.hyperdeck.counter.ResetMedia()
					break
				}
				hours, minutes, seconds := SplitTime(s)
				progress := Progress(s)

				frames := int32(0)
				engine.hyperdeck.counter.SetMedia(hours, minutes, seconds, frames, s.Remaining(), progress, !s.Play(), s.Loop())

				if engine.hyperdeck.mediaName {
					engine.message = s.MediaName()
					engine.messageColor = engine.hyperdeck.mediaColor
					engine.messageBG = engine.hyperdeck.mediaBG
					engine.oscTally = true
					engine.messageTimer.Reset(engine.hyperdeck.timeout)
				}

				t.Reset(engine.hyperdeck.timeout)
			case <-t.C:
				engine.hyperdeck.counter.ResetMedia()
			}
		}
	}
}
