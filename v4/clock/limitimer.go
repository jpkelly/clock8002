package clock

import (
	"github.com/tarm/serial"
	"gitlab.com/Depili/limitimer"
	"log"
)

func (engine *Engine) limitimerListen() {
	c := &serial.Config{Name: engine.limitimerSerial, Baud: 19200}

	port, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalf("Error opening limitimer serial port %s %v", engine.limitimerSerial, err)
		return
	}

	buff := make([]byte, 100)
	decoder := limitimer.Decoder{}

	for {
		n, err := port.Read(buff)
		if err != nil {
			log.Printf("Error reading from limitimer serial port %v", err)
			return
		}
		if n == 0 {
			log.Printf("Limitimer serial: EOF")
			return
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
						engine.Counters[i+1].SetLimitimer(p, i)
					}
					engine.Counters[5].SetLimitimer(p, p.SelectedTimer)
				}
			case limitimer.PING_MSG:
			default:
				log.Printf(" -> UNKNOWN limitimer message %v\n", msg)
			}
		}
	}
}
