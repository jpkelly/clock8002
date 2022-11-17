package clock

import (
	"fmt"
	"gitlab.com/clock-8001/clock-8001/v4/millumin"
	"gitlab.com/clock-8001/clock-8001/v4/mitti"
	// "gitlab.com/Depili/go-osc/osc"
	"github.com/chabad360/go-osc/osc"
	"gitlab.com/clock-8001/clock-8001/v4/oscutil"

	"log"
	"time"
)

const (
	clockRemainingThreshold = 20
	updateTimeout           = 1000 * time.Millisecond
)

func (engine *Engine) oscBridge(dispatcher *oscutil.RegexpDispatcher) error {
	var milluminListener = millumin.MakeListener(dispatcher)
	var mittiListener = mitti.MakeListener(dispatcher)

	if err := engine.startClockClient(milluminListener, mittiListener); err != nil {
		return fmt.Errorf("start clock client: %v", err)
	}

	log.Printf("Clock bridge: listening on %v", engine.oscServer.Addr)
	return nil
}

func (engine *Engine) startClockClient(milluminListener *millumin.Listener, mittiListener *mitti.Listener) error {
	go engine.runMilluminClockClient(milluminListener.Listen(), engine.milluminCounter, true)
	go engine.runMittiClockClient(mittiListener.Listen(), engine.mittiCounter, true)

	return nil
}

func (engine *Engine) sendMedia(player string, hours, minutes, seconds, frames, remaining int32, progress float64, paused, looping bool) error {
	if engine.oscDests == nil {
		return nil
	}
	address := fmt.Sprintf("/clock/media/%s", player)
	packet := osc.NewMessage(address, hours, minutes, seconds, frames, remaining, progress, paused, looping, osc.NewTimetagFromTime(time.Now()), engine.uuid)

	data, err := packet.MarshalBinary()
	if err != nil {
		log.Printf("sendMedia error: %v", err)
		return err
	}
	engine.oscSendChan <- data

	return nil
}

func (engine *Engine) sendResetMedia(player string) error {
	if engine.oscDests == nil {
		switch player {
		case "millumin":
			engine.milluminCounter.ResetMedia()
		case "mitti":
			engine.mittiCounter.ResetMedia()
		}
		return nil
	}

	address := fmt.Sprintf("/clock/resetmedia/%s", player)
	packet := osc.NewMessage(address, osc.NewTimetagFromTime(time.Now()), engine.uuid)

	data, err := packet.MarshalBinary()
	if err != nil {
		log.Printf("sendResetMedia error: %v", err)
		return err
	}
	engine.oscSendChan <- data

	return nil
}
