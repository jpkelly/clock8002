package clock

import (
	"fmt"
	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/vmix"
	"image/color"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type vmixState struct {
	counter        *Counter
	ignoreOverlays map[int]bool
	noLoop         bool
	liveOnly       bool
	pgmOnly        bool
	mediaName      bool
	mediaColor     *color.RGBA
	mediaBG        *color.RGBA
	timeout        time.Duration
}

func (engine *Engine) initVmix(options *EngineOptions) {
	engine.vmix = make([]*vmixState, NumCounters)

	for i, o := range options.TimerOptions {
		if !o.VmixEnabled {
			continue
		}

		s := vmixState{
			counter:   engine.Counters[i],
			noLoop:    !o.VmixLoops,
			pgmOnly:   o.VmixPGMOnly,
			mediaName: o.VmixMediaName,
			liveOnly:  !o.VmixPVM,
			timeout:   time.Duration(options.VmixTimeout) * time.Millisecond,
		}
		var err error
		s.mediaColor, err = parseColor(o.MediaColor)
		if err != nil {
			log.Fatalf("Error parsing media color: %v", err)
		}

		s.mediaBG, err = parseColor(o.MediaBG)
		if err != nil {
			log.Fatalf("Error parsing media background color: %v", err)
		}

		s.ignoreOverlays = make(map[int]bool)
		re := regexp.MustCompile(`^(?:\d+,?)+$`)
		if re.MatchString(o.VmixIgnoreOverlays) {
			layers := strings.Split(o.VmixIgnoreOverlays, ",")
			for _, l := range layers {
				n, _ := strconv.Atoi(l)
				s.ignoreOverlays[n] = true
			}
		}

		addr := fmt.Sprintf("%s:%d", o.VmixAddress, o.VmixPort)

		interval := time.Duration(options.VmixInterval) * time.Millisecond

		log.Printf("vMix: connecting to %s", addr)
		c := vmix.Connect(engine.ctx, addr, interval)
		engine.vmix[i] = &s

		go engine.vmixListen(c, i)
		log.Printf("vMix: Initialization done")
	}
}

func (engine *Engine) vmixListen(c chan *vmix.State, counter int) {
	log.Printf("vMix: Listening")
	t := timer.NewTimer(engine.vmix[counter].timeout)
	for {
		select {
		case s, ok := <-c:
			if !ok {
				log.Printf("vMix: listener channel closed, shutting down.")
				return
			}
			video := engine.vmixSelect(s, counter)
			if video != nil {
				// Video data to show
				engine.vmixSet(video, counter)
				t.Reset(engine.vmix[counter].timeout)
			} else {
				// No playing video matching the current filter
				engine.Counters[counter].ResetMedia()
			}
		case <-t.C:
			// Timeout on communications, reset the output
			engine.Counters[counter].ResetMedia()
		}
	}
}

func (engine *Engine) vmixSelect(s *vmix.State, i int) *vmix.Video {
	var video *vmix.Video
	highestOverlay := -10
	for _, v := range s.Videos {
		if engine.vmix[i].liveOnly && !v.Live {
			// PVM videos are ignored
			continue
		}
		if engine.vmix[i].noLoop && v.Looping {
			// Looping videos are ignored
			continue
		}
		if engine.vmix[i].pgmOnly && !v.PGM {
			// Only care about PGM videos
			continue
		} else {
			if v.PGM {
				// Prefer the PGM video
				video = v
				break
			}

			if v.Overlay > highestOverlay {
				if video != nil && video.Playing && !v.Playing {
					// Prefer playing media
					continue
				}
				highestOverlay = v.Overlay
				video = v
			}
		}
	}
	return video
}

func (engine *Engine) vmixSet(video *vmix.Video, i int) {

	engine.Counters[i].mediaUpdate(video)

	if engine.vmix[i].mediaName {
		engine.message = video.Name
		engine.messageColor = engine.vmix[i].mediaColor
		engine.messageBG = engine.vmix[i].mediaBG
		engine.oscTally = true
		engine.messageTimer.Reset(engine.vmix[i].timeout)
	}
}
