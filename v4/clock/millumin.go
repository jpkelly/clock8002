package clock

import (
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	"gitlab.com/clock-8001/clock-8001/v4/millumin"
	// "gitlab.com/Depili/go-osc/osc"

	"log"
	"time"
)

func (engine *Engine) milluminListen(server *Server, counter int, bridge bool) {
	log.Printf("Millumin listening on %s for counter %d", server.osc.Addr, counter)
	var milluminListener = millumin.MakeListener(server.dispatcher)
	go engine.runMilluminClockClient(milluminListener.Listen(), engine.Counters[counter], bridge)
}

func (engine *Engine) updateMilluminClock(state millumin.State, milluminCounter *Counter, bridge bool) error {
	var err error

	// XXX: select named layer, not first playing?
	for _, layerState := range state {
		if !layerState.Playing {
			continue
		} else if engine.ignoreRegexp.MatchString(layerState.Layer) {
			debug.Printf("Ignored layer update\n")
			continue
		} else if layerState.Updated.Before(time.Now().Add(-1*time.Second)) == true {
			debug.Printf("Layer information stale, ignored")
			continue
		}

		// Fix one second offset with millumin time remaining...
		remaining := (time.Duration(layerState.Remaining()) + 1) * time.Second
		total := time.Duration(layerState.Duration) * time.Second

		hours := int32(remaining.Truncate(time.Hour).Hours())
		minutes := int32(remaining.Truncate(time.Minute).Minutes()) - (hours * 60)
		seconds := int32(remaining.Truncate(time.Second).Seconds()) - (((hours * 60) + minutes) * 60)

		progress := float64(remaining) / float64(total)

		milluminCounter.SetMedia(hours, minutes, seconds, 0, remaining, progress, layerState.Paused, false)
		if bridge {
			engine.sendMedia("millumin", hours, minutes, seconds, 0, int32(layerState.Remaining()+1), progress, layerState.Paused, false)
		}

		return nil
	}
	// No playing media found
	milluminCounter.ResetMedia()
	engine.sendResetMedia("millumin")

	return err
}

func (engine *Engine) runMilluminClockClient(listenChan chan millumin.State, milluminCounter *Counter, bridge bool) {
	for state := range listenChan {
		// TODO: also refresh on tick
		if err := engine.updateMilluminClock(state, milluminCounter, bridge); err != nil {
			log.Fatalf("Millumin: update clock: %v", err)
		} else {
			debug.Printf("Millumin: update clock: %v\n", state)
		}
	}
}
