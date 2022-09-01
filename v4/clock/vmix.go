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
	if !options.VmixEnabled {
		return
	}

	s := vmixState{
		counter:   engine.Counters[options.VmixTimer],
		noLoop:    !options.VmixLoops,
		pgmOnly:   options.VmixPGMOnly,
		mediaName: options.VmixMediaName,
		liveOnly:  !options.VmixPVM,
		timeout:   time.Duration(options.VmixTimeout) * time.Millisecond,
	}
	var err error
	s.mediaColor, err = parseColor(options.VmixMediaColor)
	if err != nil {
		log.Fatalf("Error parsing vMix media color: %v", err)
	}

	s.mediaBG, err = parseColor(options.VmixMediaBG)
	if err != nil {
		log.Fatalf("Error parsing vMix media background color: %v", err)
	}

	s.ignoreOverlays = make(map[int]bool)
	re := regexp.MustCompile(`^(?:\d+,?)+$`)
	if re.MatchString(options.VmixIgnoreOverlays) {
		layers := strings.Split(options.VmixIgnoreOverlays, ",")
		for _, l := range layers {
			n, _ := strconv.Atoi(l)
			s.ignoreOverlays[n] = true
		}
	}

	addr := fmt.Sprintf("%s:%d", options.VmixAddress, options.VmixPort)

	interval := time.Duration(options.VmixInterval) * time.Millisecond

	log.Printf("vMix: connecting to %s", addr)
	c := vmix.Connect(engine.ctx, addr, interval)
	engine.vmix = s

	go engine.vmixListen(c)
	log.Printf("vMix: Initialization done")
}

func (engine *Engine) vmixListen(c chan *vmix.State) {
	log.Printf("vMix: Listening")
	t := timer.NewTimer(engine.vmix.timeout)
	for {
		select {
		case s, ok := <-c:
			if !ok {
				log.Printf("vMix: listener channel closed, shutting down.")
				return
			}
			video := engine.vmixSelect(s)
			if video != nil {
				// Video data to show
				engine.vmixSet(video)
				t.Reset(engine.vmix.timeout)
			} else {
				// No playing video matching the current filter
				engine.vmix.counter.ResetMedia()
			}
		case <-t.C:
			// Timeout on communications, reset the output
			engine.vmix.counter.ResetMedia()
		}
	}
}

func (engine *Engine) vmixSelect(s *vmix.State) *vmix.Video {
	var video *vmix.Video
	highestOverlay := -10
	for _, v := range s.Videos {
		if engine.vmix.liveOnly && !v.Live {
			// PVM videos are ignored
			continue
		}
		if engine.vmix.noLoop && v.Looping {
			// Looping videos are ignored
			continue
		}
		if engine.vmix.pgmOnly && !v.PGM {
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

func (engine *Engine) vmixSet(video *vmix.Video) {
	diff := video.Length - video.Head
	if diff < 0 {
		diff = -diff
	}

	hours := int32(diff.Truncate(time.Hour).Hours())
	minutes := int32(diff.Truncate(time.Minute).Minutes()) - (hours * 60)
	seconds := int32(diff.Truncate(time.Second).Seconds()) - (((hours * 60) + minutes) * 60)
	progress := video.Head.Seconds() / video.Length.Seconds()
	frames := int32(0)
	engine.vmix.counter.SetMedia(hours, minutes, seconds, frames, diff, progress, !video.Playing, video.Looping)

	if engine.vmix.mediaName {
		engine.message = video.Name
		engine.messageColor = engine.vmix.mediaColor
		engine.messageBG = engine.vmix.mediaBG
		engine.oscTally = true
		engine.messageTimer.Reset(engine.vmix.timeout)
	}
}
