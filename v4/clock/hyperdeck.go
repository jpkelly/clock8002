package clock

import (
	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/hyperdeck"
	"log"
	"time"
)

// HyperdeckOptions contains the options for go-flags for the hyperdeck module
type HyperdeckOptions struct {
	HyperdeckEnabled   bool   `long:"hyperdeck-enabled" description:"Enable hyperdeck integration"`
	HyperdeckAddress   string `long:"hyperdeck-address" desctiption:"IP address of the hyperdeck"`
	HyperdeckRelay     bool   `long:"hyperdeck-relay" description:"Relay commands to this hyperdeck on port 9993"`
	HyperdeckTimer     int    `long:"hyperdeck-timer" description:"Timer to use for hyperdeck status" default:"8"`
	HyperdeckMediaName bool   `long:"hyperdeck-media-name" description:"Show hyperdeck media name as clock text"`
	HyperdeckTimeout   int    `long:"hyperdeck-timeout" description:"Hyperdeck communication timeout, in milliseconds" default:"1000"`
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
	ip        string
	relay     bool
	counter   *Counter
	mediaName bool
	timeout   time.Duration
}

func (engine *Engine) hyperdeckInit(options *EngineOptions) {
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
				hours, minutes, seconds := splitTime(s)
				progress := progress(s)

				frames := int32(0)
				engine.hyperdeck.counter.SetMedia(hours, minutes, seconds, frames, s.Remaining(), progress, !s.Play(), s.Loop())

				if engine.hyperdeck.mediaName {
					engine.message = s.MediaName()
					engine.messageColor = engine.hyperdeck.counter.mediaColor
					engine.messageBG = engine.hyperdeck.counter.mediaBG
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
