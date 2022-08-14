package clock

/*
 * Integration for Picturall media servers made by Analog Way
 */

import (
	"fmt"
	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/picturall"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Initialize the picturall connection.
func (engine *Engine) initPicturall(options *EngineOptions) {
	var err error

	if !options.PicturallEnabled {
		return
	}
	engine.picturall.counter = engine.Counters[options.PicturallTimer]
	engine.picturall.ignoreLayers = make(map[int]bool)
	engine.picturall.noLoop = !options.PicturallLoops
	engine.picturall.noStreams = !options.PicturallStreams
	engine.picturall.mediaName = options.PicturallMediaName
	engine.picturall.timeout = time.Duration(options.PicturallTimeout) * time.Millisecond
	engine.picturall.lastHead = make(map[int]time.Duration)
	engine.picturall.mediaColor, err = parseColor(options.PicturallMediaColor)
	if err != nil {
		log.Fatal("Error parsing picturall media color: %s %v", options.PicturallMediaColor, err)
	}
	engine.picturall.mediaBG, err = parseColor(options.PicturallMediaBG)
	if err != nil {
		log.Fatal("Error parsing picturall media bg color: %s %v", options.PicturallMediaBG, err)
	}

	if options.PicturallDebug {
		picturall.DumpTraffic = true
	}

	re := regexp.MustCompile(`^(?:\d+,?)+$`)
	if re.MatchString(options.PicturallIgnoreLayers) {
		layers := strings.Split(options.PicturallIgnoreLayers, ",")
		for _, l := range layers {
			n, _ := strconv.Atoi(l)
			engine.picturall.ignoreLayers[n] = true
		}
	}

	addr := fmt.Sprintf("%s:%d", options.PicturallAddress, options.PicturallPort)

	log.Printf("Connecting to picturall: %s", addr)

	c := picturall.Connect(engine.ctx, addr)
	go engine.picturallListen(c)

	log.Printf("Picturall init done!")
}

// Listener go routine, retries the connection on errors
func (engine *Engine) picturallListen(c chan *picturall.Media) {
	engine.wg.Add(1)
	defer engine.wg.Done()

	timer := timer.NewTimer(engine.picturall.timeout)

	for {
		select {
		case m := <-c:
			if engine.picturallAccept(m) {
				engine.picturallHandle(m)
				timer.Reset(engine.picturall.timeout)
			}
		case <-timer.C:
			engine.picturall.counter.ResetMedia()
			engine.picturall.lastLayer = 0
		case <-engine.ctx.Done():
			return
		}
	}

	log.Printf("Pictural listener done")
}

func (engine *Engine) picturallAccept(m *picturall.Media) bool {
	if m.Layer < engine.picturall.lastLayer {
		// Media playing on a layer bellow the current highest
		return false
	}

	if engine.picturall.noLoop && m.Loop() {
		return false
	}

	// Check for ignored layers
	if _, ok := engine.picturall.ignoreLayers[m.Layer]; ok {
		return false
	}

	if m.Length == 0 {
		if m.PlayState == picturall.Stop {
			// Stopped media playback
			engine.picturall.
				counter.ResetMedia()
			engine.picturall.lastLayer = 0
			return false
		}
		if engine.picturall.noStreams {
			return false
		}
	} else if m.Head > m.Length {
		// Clear up one buggy case on picturall media start:
		// The media will be reported as being past it's end....
		log.Printf("Picturall bug: discarding playhead past media length on non-stream")
		return false
	}

	if lastHead, ok := engine.picturall.lastHead[m.Layer]; ok {
		if lastHead == 0 {
			// There is a bug in picturall where on media start
			// the playhead goes to 0, <end of file>, 0
			// This is a workaround for that...
			if m.Length-m.Head < time.Millisecond*50 {
				log.Printf("Picturall bug: discarding end-of-file message at playback start")
				return false
			}
		}
	} else {
		if m.Length == m.Head {
			log.Printf("Picturall bug: ignoring end-of-file media with no knowledge of previous state")
			return false
		}
	}

	engine.picturall.lastHead[m.Layer] = m.Head
	engine.picturall.lastLayer = m.Layer
	return true
}

func (engine *Engine) picturallHandle(m *picturall.Media) {
	diff := m.Length - m.Head

	// Get the absolute value
	if diff < 0 {
		diff = -diff
	}
	hours := int32(diff.Truncate(time.Hour).Hours())
	minutes := int32(diff.Truncate(time.Minute).Minutes()) - (hours * 60)
	seconds := int32(diff.Truncate(time.Second).Seconds()) - (((hours * 60) + minutes) * 60)
	progress := m.Head.Seconds() / m.Length.Seconds()

	engine.picturall.counter.SetMedia(hours, minutes, seconds, 0, diff, progress, !m.Play(), m.Loop())

	if engine.picturall.mediaName {
		parts := strings.Split(m.Name, "/")
		name := parts[len(parts)-1]
		engine.message = name
		engine.messageColor = engine.picturall.mediaColor
		engine.messageBG = engine.picturall.mediaBG
		engine.oscTally = true
		engine.messageTimer.Reset(engine.picturall.timeout)
	}

}
