package clock

import (
	"github.com/tarm/serial"
	"io"
	"log"
	"time"
)

const (
	cueRightArrow = 0x0F
	cueLeftArrow  = 0x1F
	cueBlankOff   = 0x2F
	cueBlankOn    = 0x3F
)

// cueState contains the state of DSAN Perfect Cue
type cueState struct {
	RightArrow bool
	LeftArrow  bool
	Blank      bool
	Show       bool
}

type cueOptions struct {
	CueEnabled  bool   `long:"cue-enabled" description:"Enable DSAN Perfect Cue support"`
	CueSerial   string `long:"cue-serial" description:"Serial port for DSAN Perfect Cue" default:"/dev/ttyAMA0"`
	CueDuration int    `long:"cue-duration" description:"Time to show cue changes on screen, in seconds" default:"2"`
}

func (engine *Engine) initPerfectCue(options *EngineOptions) {
	engine.cueDuration = time.Duration(options.CueDuration) * time.Second
	engine.cueState = &cueState{}
	engine.cueChan = make(chan *cueState)

	if !options.CueEnabled {
		return
	}
	go engine.listenPerfectCue(options)
}

func (engine *Engine) listenPerfectCue(options *EngineOptions) {
	engine.wg.Add(1)
	defer engine.wg.Done()

	log.Printf("Perfect Cue: Opening serial: %s", options.CueSerial)

	c := &serial.Config{
		Name:        options.CueSerial,
		Baud:        19200,
		ReadTimeout: 500 * time.Millisecond,
	}

	port, err := serial.OpenPort(c)
	if err != nil {
		if err != io.EOF {
			log.Printf("Error opening Perfect Cue serial port")
			return
		}
	}

	buff := make([]byte, 100)

	for {
		select {
		case <-engine.ctx.Done():
			log.Printf("Perfect Cue: stopping listener on request")
			close(engine.cueChan)
			return
		default:
		}
		n, err := port.Read(buff)
		if err != nil {
			if err == io.EOF {
				continue
			}
			log.Printf("Perfect cue: error reading serial port: %v", err)
			return
		}

		if n == 0 {
			continue
		}

		for _, b := range buff[:n] {
			switch b {
			case cueRightArrow:
				s := cueState{
					RightArrow: true,
				}
				engine.cueChan <- &s
			case cueLeftArrow:
				s := cueState{
					LeftArrow: true,
				}
				engine.cueChan <- &s
			case cueBlankOff:
				s := cueState{
					Blank: false,
				}
				engine.cueChan <- &s
			case cueBlankOn:
				s := cueState{
					Blank: true,
				}
				engine.cueChan <- &s
			}
		}
	}
}
