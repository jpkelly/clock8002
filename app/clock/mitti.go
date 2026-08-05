package clock

import (
	"github.com/desertbit/timer"
	"github.com/jpkelly/clock8002/app/debug"
	"github.com/jpkelly/clock8002/app/mitti"
	"log"
)

func (engine *Engine) mittiListen(server *Server, counter int, bridge bool) {
	log.Printf("Mitti listening on %s for counter %d", server.osc.Addr, counter)
	var mittiListener = mitti.MakeListener(server.dispatcher)
	go engine.runMittiClockClient(mittiListener.Listen(), engine.Counters[counter], bridge)
}

func (engine *Engine) updateMittiClock(state mitti.State, mittiCounter *Counter, bridge bool) error {
	// FIXME: need to fudge this by one second to get the displays to agree?

	hours, minutes, seconds := splitTime(&state)

	debug.Printf("Mitti update, remaining: %v total: %v\n", state.Remaining(), state.Duration())
	debug.Printf(" -> update state: %02d:%02d:%02d", hours, minutes, seconds)

	mittiCounter.mediaUpdate(&state)
	if bridge {
		engine.sendMedia("mitti", hours, minutes, seconds, 0, int32(state.Remaining().Seconds()), progress(&state), !state.Play(), !state.Loop())
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
