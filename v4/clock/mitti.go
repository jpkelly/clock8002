package clock

import (
	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	"gitlab.com/clock-8001/clock-8001/v4/mitti"
	// "gitlab.com/Depili/go-osc/osc"
	"github.com/chabad360/go-osc/osc"
	"gitlab.com/clock-8001/clock-8001/v4/oscutil"

	"log"
	"time"
)

func (engine *Engine) mittiListen(listenAddr string, counter int, bridge bool) {
	server := oscListenerSetup(listenAddr)
	go server.run(engine.ctx, engine.wg)

	log.Printf("Mitti listening on %s for counter %d", listenAddr, counter)
	var mittiListener = mitti.MakeListener(server.dispatcher)
	go engine.runMittiClockClient(mittiListener.Listen(), engine.Counters[counter], bridge)
}

func oscListenerSetup(listenAddr string) *Server {
	oscDispatcher := oscutil.NewRegexpDispatcher()
	oscServer := osc.Server{
		Addr:    listenAddr,
		Handler: oscDispatcher.Dispatch,
	}

	// clock.Server
	var server = Server{
		listeners:  make(map[chan Message]struct{}),
		Debug:      false,
		dispatcher: oscDispatcher,
		osc:        &oscServer,
	}

	log.Printf("OSC: listening on %v", oscServer.Addr)
	return &server
}

func (engine *Engine) updateMittiClock(state mitti.State, mittiCounter *Counter, bridge bool) error {
	// FIXME: need to fudge this by one second to get the displays to agree?
	remaining := time.Duration(state.Remaining) * time.Second
	total := remaining + (time.Duration(state.Elapsed) * time.Second)
	progress := float64(remaining) / float64(total)

	hours := int32(state.Hours)
	minutes := int32(state.Minutes)
	seconds := int32(state.Seconds)
	frames := int32(state.Frames)

	log.Printf("Mitti update, remaining: %v total: %v\n", remaining.Seconds(), total.Seconds())

	log.Printf(" -> update state: %02d:%02d:%02d", state.Hours, state.Minutes, state.Seconds)
	mittiCounter.SetMedia(hours, minutes, seconds, frames, remaining, progress, state.Paused, state.Loop)
	if bridge {
		engine.sendMedia("mitti", hours, minutes, seconds, frames, int32(state.Remaining), progress, state.Paused, state.Loop)
	}

	// TODO: cue name

	return nil
}

func (engine *Engine) runMittiClockClient(listenChan chan mitti.State, mittiCounter *Counter, bridge bool) {
	timeout := timer.NewTimer(updateTimeout)
	for {
		select {
		case state := <-listenChan:
			timeout.Reset(updateTimeout)
			// TODO: also refresh on tick
			if err := engine.updateMittiClock(state, mittiCounter, bridge); err != nil {
				log.Fatalf("Mitti: update clock: %v", err)
			} else {
				debug.Printf("Mitti: update clock: %v\n", state)
			}
		case <-timeout.C:
			mittiCounter.ResetMedia()
			engine.sendResetMedia("mitti")
		}
	}
}
