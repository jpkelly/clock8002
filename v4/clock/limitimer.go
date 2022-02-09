package clock

import (
	"fmt"
	"github.com/tarm/serial"
	"gitlab.com/Depili/limitimer"
	"log"
	"time"
)

func (engine *Engine) limitimerListen() {
	engine.wg.Add(1)
	defer engine.wg.Done()

	c := &serial.Config{
		Name:        engine.limitimerSerial,
		Baud:        19200,
		ReadTimeout: 500 * time.Millisecond,
	}

	port, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalf("Error opening limitimer serial port %s %v", engine.limitimerSerial, err)
		return
	}
	defer port.Close()

	buff := make([]byte, 100)
	decoder := limitimer.Decoder{}

	for {
		select {
		case <-engine.ctx.Done():
			log.Printf("Limitimer listener quiting")
			return
		default:
		}

		n, err := port.Read(buff)
		if err != nil {
			log.Printf("Error reading from limitimer serial port %v", err)
			return
		}
		if n == 0 {
			continue
		}

		messages := decoder.Feed(buff[:n])
		for _, msg := range messages {
			switch limitimer.Type(msg) {
			case limitimer.STATUS_MSG:
				p, err := limitimer.ParseStateMessage(msg)
				if err != nil {
					log.Printf(" -> Invalid limitimer message %v", err)
				} else {
					for i := range p.Timers {
						if engine.limitimerReceive[i] {
							engine.Counters[i+1].SetLimitimer(p, i)
							if engine.limitimerBroadcast[i] {
								msg := makeLimitimerMessage(p, i, engine.uuid)
								addr := fmt.Sprintf("/clock/limitimer/%d", i+1)
								b, err := msg.MarshalOSC(addr).MarshalBinary()
								if err != nil {
									log.Printf("Limitimer message osc marshal error: %v", err)
								} else {
									engine.oscSendChan <- b
								}
							}
						}
					}
					if engine.limitimerReceive[4] {
						engine.Counters[5].SetLimitimer(p, p.SelectedTimer)
						if engine.limitimerBroadcast[4] {
							msg := makeLimitimerMessage(p, p.SelectedTimer, engine.uuid)
							b, err := msg.MarshalOSC("/clock/limitimer/active").MarshalBinary()
							if err != nil {
								log.Printf("Limitimer message osc marshal error: %v", err)
							} else {
								engine.oscSendChan <- b
							}
						}
					}
				}
			case limitimer.PING_MSG:
			default:
				log.Printf(" -> UNKNOWN limitimer message %v\n", msg)
			}
		}
	}
}

const (
	limitimerSendMinutes  = true
	limitimerPeriod       = 250 * time.Millisecond
	limitimerCountupTotal = (99 * 360) + (59 * 60) + 59
)

func (engine *Engine) limitimerSend() {
	engine.wg.Add(1)
	defer engine.wg.Done()

	c := &serial.Config{Name: engine.limitimerSerial, Baud: 19200}

	port, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalf("Error opening limitimer serial port %s %v", engine.limitimerSerial, err)
		return
	}
	defer port.Close()

	// Base config for the limitimer state packet
	p := limitimer.State{
		CountDirection:    true,
		ProgramMinutes:    limitimerSendMinutes,
		SessionMinutes:    limitimerSendMinutes,
		ContinueAfterZero: engine.overtimeCountMode == "continue",
		SelectedTimer:     0,
	}

	ticker := time.NewTicker(limitimerPeriod)

	for {
		select {
		case <-engine.ctx.Done():
			log.Printf("Limitimer sender quitting")
			return
		case <-ticker.C:
			t := time.Now()
			p.Sequence = time.Now().Nanosecond() / 100000000

			for i := 0; i < 4; i++ {
				c := engine.Counters[i+1].Output(t)
				p.Timers[i].SetRun(!c.Paused)
				p.Timers[i].SetBlink(engine.overtimeVisibility == "blink" || engine.overtimeVisibility == "both")

				if !c.Active {
					// Inactive, show 00:00
					p.Timers[i].Total.SetSeconds(0)
					p.Timers[i].Sumup.SetSeconds(0)
					p.Timers[i].Elapsed.SetSeconds(0)
					p.Timers[i].SetRun(false)
				} else if c.Countdown {
					p.Timers[i].Total.SetSeconds(int(c.Duration.Seconds()))
					p.Timers[i].Sumup.SetSeconds(int(engine.signalThresholdWarning.Seconds()))
					p.Timers[i].Elapsed.SetSeconds(int(c.Duration.Seconds()-c.Diff.Seconds()) + 1)
				} else {
					p.Timers[i].Total.SetSeconds(limitimerCountupTotal)
					p.Timers[i].Sumup.SetSeconds(0)
					p.Timers[i].Elapsed.SetSeconds(limitimerCountupTotal - int(c.Diff.Seconds()))
				}
			}

			_, err := port.Write(p.Marshal())
			if err != nil {
				log.Printf("Error writing to limitimer serial %v", err)
				return
			}

			// Send the ping packet???
		}
	}
}

func makeLimitimerMessage(p *limitimer.State, i int, uuid string) LimitimerMessage {
	msg := LimitimerMessage{
		Total:     int32(p.Timers[i].Total.Seconds()),
		Sumup:     int32(p.Timers[i].Sumup.Seconds()),
		Elapsed:   int32(p.Timers[i].Elapsed.Seconds()),
		Minutes:   minutes(p, i),
		Countdown: p.CountDirection,
		Run:       p.Timers[i].Run(),
		Blink:     p.Timers[i].Blink(),
		Beep:      p.Timers[i].Beep(),
		UUID:      uuid,
	}
	return msg
}

func minutes(p *limitimer.State, i int) bool {
	if i < 4 {
		return p.ProgramMinutes
	}
	return p.SessionMinutes
}
