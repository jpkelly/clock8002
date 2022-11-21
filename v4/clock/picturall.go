package clock

/*
 * Integration for Picturall media servers made by Analog Way
 */

import (
	"fmt"
	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/picturall"
	"image/color"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type picturallState struct {
	counter      *Counter
	noLoop       bool
	noStreams    bool
	timeout      time.Duration
	mediaName    bool
	mediaColor   *color.RGBA
	mediaBG      *color.RGBA
	ignoreLayers map[int]bool
	lastLayer    int
	lastHead     map[int]time.Duration
}

// Initialize the picturall connection.
func (engine *Engine) initPicturall(options *EngineOptions) {
	var err error

	engine.picturall = make([]*picturallState, numCounters)

	if options.PicturallDebug {
		picturall.DumpTraffic = true
	}

	for i, o := range options.TimerOptions {
		if !o.PicturallEnabled {
			continue
		}

		p := picturallState{
			counter:      engine.Counters[i],
			ignoreLayers: make(map[int]bool),
			noLoop:       !o.PicturallLoops,
			noStreams:    !o.PicturallStreams,
			mediaName:    o.PicturallMediaName,
			timeout:      time.Duration(options.PicturallTimeout) * time.Millisecond,
			lastHead:     make(map[int]time.Duration),
		}

		p.mediaColor, err = parseColor(o.MediaColor)
		if err != nil {
			log.Fatalf("Error parsing media color: %s %v", o.MediaColor, err)
		}
		p.mediaBG, err = parseColor(o.MediaBG)
		if err != nil {
			log.Fatalf("Error parsingbg color: %s %v", o.MediaBG, err)
		}

		re := regexp.MustCompile(`^(?:\d+,?)+$`)
		if re.MatchString(o.PicturallIgnoreLayers) {
			layers := strings.Split(o.PicturallIgnoreLayers, ",")
			for _, l := range layers {
				n, _ := strconv.Atoi(l)
				p.ignoreLayers[n] = true
			}
		}

		addr := fmt.Sprintf("%s:%d", o.PicturallAddress, o.PicturallPort)

		log.Printf("Timer %d: Connecting to picturall: %s", i, addr)

		c := picturall.Connect(engine.ctx, addr)
		go engine.picturallListen(c, i)
		engine.picturall[i] = &p
	}
	log.Printf("Picturall init done!")
}

// Listener go routine, retries the connection on errors
func (engine *Engine) picturallListen(c chan *picturall.Media, counter int) {
	engine.wg.Add(1)
	defer engine.wg.Done()

	p := engine.picturall[counter]

	timer := timer.NewTimer(p.timeout)

	for {
		select {
		case m := <-c:
			if engine.picturallAccept(m, p) {
				engine.picturallHandle(m, p)
				timer.Reset(p.timeout)
			}
		case <-timer.C:
			p.counter.ResetMedia()
			p.lastLayer = 0
		case <-engine.ctx.Done():
			log.Printf("Pictural listener done")
			return
		}
	}
}

func (engine *Engine) picturallAccept(m *picturall.Media, p *picturallState) bool {
	if m.Layer < p.lastLayer {
		// Media playing on a layer bellow the current highest
		return false
	}

	if p.noLoop && m.Loop() {
		return false
	}

	// Check for ignored layers
	if _, ok := p.ignoreLayers[m.Layer]; ok {
		return false
	}

	if m.Length == 0 {
		if m.PlayState == picturall.Stop {
			// Stopped media playback
			p.counter.ResetMedia()
			p.lastLayer = 0
			return false
		}
		if p.noStreams {
			return false
		}
	} else if m.Head > m.Length {
		// Clear up one buggy case on picturall media start:
		// The media will be reported as being past it's end....
		log.Printf("Picturall bug: discarding playhead past media length on non-stream")
		return false
	}

	if lastHead, ok := p.lastHead[m.Layer]; ok {
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

	p.lastHead[m.Layer] = m.Head
	p.lastLayer = m.Layer
	return true
}

func (engine *Engine) picturallHandle(m *picturall.Media, p *picturallState) {
	diff := m.Length - m.Head

	// Get the absolute value
	if diff < 0 {
		diff = -diff
	}
	hours := int32(diff.Truncate(time.Hour).Hours())
	minutes := int32(diff.Truncate(time.Minute).Minutes()) - (hours * 60)
	seconds := int32(diff.Truncate(time.Second).Seconds()) - (((hours * 60) + minutes) * 60)
	progress := m.Head.Seconds() / m.Length.Seconds()

	p.counter.SetMedia(hours, minutes, seconds, 0, diff, progress, !m.Play(), m.Loop())

	if p.mediaName {
		parts := strings.Split(m.Name, "/")
		name := parts[len(parts)-1]
		engine.message = name
		engine.messageColor = p.mediaColor
		engine.messageBG = p.mediaBG
		engine.oscTally = true
		engine.messageTimer.Reset(p.timeout)
	}

}
