package mitti

import (
	"gitlab.com/Depili/clock-8001/v4/debug"
	"gitlab.com/Depili/go-osc/osc"
	"log"
)

// MakeListener creates a Mitti OSC message listener
func MakeListener(d *osc.StandardDispatcher) *Listener {
	var listener = Listener{
		listeners: make(map[chan State]struct{}),
	}
	var state State
	state.Paused = true
	state.Loop = false
	state.Remaining = 0
	listener.state = &state
	listener.setup(d)
	return &listener
}

// Listener is a Mitti OSC message receiver
type Listener struct {
	state     *State
	listeners map[chan State]struct{}
}

// Listen registers a new listener for the decoded Mitti messages
func (listener *Listener) Listen() chan State {
	var listenChan = make(chan State)

	listener.listeners[listenChan] = struct{}{}

	return listenChan
}

func (listener *Listener) update() {
	state := listener.state.Copy()

	for listenChan := range listener.listeners {
		listenChan <- state
	}
	debug.Printf("mitti state update: %v\n", listener.state)
}

func (listener *Listener) handleTogglePlay(msg *osc.Message) {
	var playing int32

	if err := msg.UnmarshalArgument(0, &playing); err != nil {
		log.Printf("mitti togglePlay unmarshal %v: %v\n", msg, err)
	}

	listener.state.TogglePlay(playing)
	listener.update()
}

func (listener *Listener) handleToggleLoop(msg *osc.Message) {
	var loop int32

	if err := msg.UnmarshalArgument(0, &loop); err != nil {
		log.Printf("mitti toggleLoop unmarshal %v: %v\n", msg, err)
	}

	listener.state.ToggleLoop(loop)
	listener.update()
}

func (listener *Listener) handleCueTimeLeft(msg *osc.Message) {
	var cueTimeLeft string

	if err := msg.UnmarshalArgument(0, &cueTimeLeft); err != nil {
		log.Printf("mitti cueTimeLeft unmarshal %v: %v\n", msg, err)
	}

	listener.state.CueTimeLeft(cueTimeLeft)
	listener.update()
}

func (listener *Listener) handleCueTimeElapsed(msg *osc.Message) {
	var cueTimeElapsed string

	if err := msg.UnmarshalArgument(0, &cueTimeElapsed); err != nil {
		log.Printf("mitti cueTimeLeft unmarshal %v: %v\n", msg, err)
	}

	listener.state.CueTimeElapsed(cueTimeElapsed)
}

func registerHandler(d *osc.StandardDispatcher, addr string, handler osc.HandlerFunc) {
	if err := d.AddMsgHandler(addr, handler); err != nil {
		panic(err)
	}
}

func (listener *Listener) setup(d *osc.StandardDispatcher) {
	registerHandler(d, "/mitti/cueTimeLeft", listener.handleCueTimeLeft)
	registerHandler(d, "/mitti/cueTimeElapsed", listener.handleCueTimeElapsed)
	registerHandler(d, "/mitti/togglePlay", listener.handleTogglePlay)
	registerHandler(d, "/mitti/toggleLoop", listener.handleToggleLoop)
	registerHandler(d, "/mitti/current/toggleLoop", listener.handleToggleLoop)
}
