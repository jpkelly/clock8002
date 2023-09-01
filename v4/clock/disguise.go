package clock

import (
	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/disguise"
	"log"
	"time"
)

type DisguiseOptions struct {
	DisguiseTimeout int `long:"disguise-timeout" description:"Hyperdeck communication timeout, in milliseconds" default:"1000"`
}

type DisguiseTimerOptions struct {
	DisguiseEnabled   bool `long:"disguise-enabled" description:"Enable Disguise OSC integration"`
	DisguisePort      int  `long:"disguise-port" description:"Disguise: port number to listen for this timer"`
	DisguiseLoops     bool `long:"disguise-loops" description:"Disguise: Show looping sections"`
	DisguisePaused    bool `long:"disguise-paused" description:"Disguise: Show paused timers"`
	DisguiseNextName  bool `long:"disguise-next-name" description:"Show the name of next cue instead of current cue"`
	DisguiseMediaName bool `long:"disguise-media-name" description:"Disguise: Show section / note name"`
}

type disguiseState struct {
	timeout time.Duration
}

func (engine *Engine) disguiseInit(options *EngineOptions) {
	for i, o := range options.TimerOptions {
		if !o.DisguiseEnabled {
			continue
		}

		s := disguiseState{
			timeout: time.Duration(options.DisguiseTimeout) * time.Millisecond,
		}

		engine.disguise = s

		go engine.disguiseListen(o, i)
	}
}

func (engine *Engine) disguiseListen(o *TimerOptions, i int) {
	engine.wg.Add(1)
	defer engine.wg.Done()

	for {
		c, err := disguise.Listen(engine.ctx, engine.wg, o.DisguisePort, o.DisguiseNextName)
		if err != nil {
			log.Printf("Error starting disguise listener: %v", err)
			return
		}

		t := timer.NewTimer(engine.disguise.timeout)

		for {
			select {
			case <-engine.ctx.Done():
				log.Printf("Disguise: exiting on request")
				return
			case <-t.C:
				engine.Counters[i].ResetMedia()
			case m, ok := <-c:
				if !ok {
					log.Printf("Disguise: listener chan closed, retrying")
					time.Sleep(5 * time.Second)
					break
				}

				if !o.DisguiseLoops && m.Loop() {
					continue
				}

				if !o.DisguisePaused && !m.Play() {
					continue
				}

				hours, minutes, seconds := splitTime(m)
				progress := progress(m)
				frames := int32(0)
				engine.Counters[i].SetMedia(hours, minutes, seconds, frames, m.Remaining(), progress, !m.Play(), m.Loop())

				if o.DisguiseMediaName {
					engine.message = m.MediaName()
					engine.messageColor = engine.Counters[i].mediaColor
					engine.messageBG = engine.Counters[i].mediaBG
					engine.oscTally = true
					engine.messageTimer.Reset(engine.disguise.timeout)
				}
			}
		}
	}
}
