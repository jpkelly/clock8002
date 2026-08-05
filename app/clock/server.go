package clock

import (
	"github.com/jpkelly/clock8002/app/debug"
	"github.com/jpkelly/clock8002/app/oscutil"
	// "gitlab.com/Depili/go-osc/osc"
	"github.com/chabad360/go-osc/osc"

	"context"
	"image/color"
	"log"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	timerPattern     = `/clock/timer/(\d)/`
	sourcePattern    = `/clock/source/([1-4])/`
	signalPattern    = `/clock/signal/(\d)`
	limitimerPattern = `/clock/limitimer/([1-4]|active)`
)

// MakeServer creates a clock.Server instance from osc.Server instance
func MakeServer(oscServer *osc.Server, d *oscutil.RegexpDispatcher, uuid string) *Server {
	var server = Server{
		listeners:       make(map[chan Message]struct{}),
		Debug:           false,
		dispatcher:      d,
		osc:             oscServer,
		timerRegexp:     regexp.MustCompile(timerPattern),
		sourceRegexp:    regexp.MustCompile(sourcePattern),
		signalRegexp:    regexp.MustCompile(signalPattern),
		limitimerRegexp: regexp.MustCompile(limitimerPattern),
		uuid:            uuid,
	}

	server.setupDispatch(d)
	return &server
}

// Server is a clock osc server and listens for incoming osc messages
type Server struct {
	listeners       map[chan Message]struct{}
	Debug           bool
	dispatcher      *oscutil.RegexpDispatcher
	osc             *osc.Server
	timerRegexp     *regexp.Regexp
	sourceRegexp    *regexp.Regexp
	signalRegexp    *regexp.Regexp
	limitimerRegexp *regexp.Regexp
	lastMedia       time.Time
	uuid            string
}

func (server *Server) run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	go server.closer(ctx)

	for {
		err := server.osc.ListenAndServe()
		if err != nil {
			if e, ok := err.(*net.OpError); ok {
				if e.Temporary() {
					log.Printf("OSC-listen: Temporary error: %v. Retrying", e)
					server.osc.Close()
				} else {
					log.Printf("OSC-listen fatal error: %v. Giving up", e)
					server.osc = nil
					return
				}
			} else {
				log.Printf("OSC-listen error: %T %v", err, err)
				log.Printf("Retrying...")
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (server *Server) closer(ctx context.Context) {
	<-ctx.Done()
	if server.osc != nil {
		log.Printf("Closing osc listener for %s", server.osc.Addr)
		server.osc.Close()
	} else {
		log.Printf("Closing osc listener after fatal error")
	}
}

// Listen adds a new listener for the decoded incoming osc messages
func (server *Server) Listen() chan Message {
	var listenChan = make(chan Message)
	server.listeners[listenChan] = struct{}{}
	return listenChan
}

func (server *Server) update(message Message) {
	debug.Printf("update: %#v", message)

	for listenChan := range server.listeners {
		listenChan <- message
	}
}

/*
 * Timer related handlers
 */

func (server *Server) handleTimerSet(msg *osc.Message) {
	debug.Printf("handleTimerSet: %v", msg)
	server.sendTimerMessage("timerSet", false, msg)
}

func (server *Server) handleTimerRestart(msg *osc.Message) {
	debug.Printf("handleTimerRestart: %v", msg)
	server.sendTimerCommand("timerRestart", msg)
}

func (server *Server) handleTimerModify(msg *osc.Message) {
	if msg.Address == "/clock/countdown/modify" {
		msg.Address = "/clock/timer/0/modify"
	} else if msg.Address == "/clock/countdown2/modify" {
		msg.Address = "/clock/timer/1/modify"
	} else if msg.Address == "/clock/countup/modify" {
		msg.Address = "/clock/timer/0/modify"
	}
	server.sendTimerMessage("timerModify", false, msg)
}

func (server *Server) handleTimerStop(msg *osc.Message) {
	debug.Printf("countdownStop: %#v", msg)
	if msg.Address == "/clock/countdown/stop" {
		msg.Address = "/clock/timer/0/stop"
	} else if msg.Address == "/clock/countdown2/stop" {
		msg.Address = "/clock/timer/1/stop"
	}

	server.sendTimerCommand("timerStop", msg)
}

func (server *Server) handleTimerPause(msg *osc.Message) {
	debug.Printf("handleTimerPause: %v", msg)
	server.sendTimerCommand("timerPause", msg)
}

func (server *Server) handleTimerResume(msg *osc.Message) {
	debug.Printf("handlTimerResume: %v", msg)
	server.sendTimerCommand("timerResume", msg)
}

func (server *Server) handleCountdownTarget(msg *osc.Message) {
	server.sendTargetMessage(msg, true)
}

func (server *Server) handleCountupTarget(msg *osc.Message) {
	server.sendTargetMessage(msg, false)
}

func (server *Server) handleLimitimer(msg *osc.Message) {
	if matches := server.limitimerRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		var counter int
		var err error

		if matches[1] == "active" {
			counter = 5
		} else {
			counter, err = strconv.Atoi(matches[1])
			if err != nil {
				log.Printf("handleLimitimer error: %v", err)
				return
			}
		}
		ltMsg := LimitimerMessage{}
		ltMsg.UnmarshalOSC(msg)
		if ltMsg.UUID == server.uuid {
			// Discard our own messages
			return
		}

		m := Message{
			Type:             "limitimer",
			Counter:          counter,
			LimitimerMessage: &ltMsg,
		}

		server.update(m)
	}
}

func (server *Server) sendTargetMessage(msg *osc.Message, countdown bool) {
	debug.Printf("sendTargetMessage: %v %v", countdown, msg)
	if matches := server.timerRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		counter, _ := strconv.Atoi(matches[1])
		var target string
		err := oscutil.UnmarshalArguments(msg, &target)
		if err != nil {
			log.Printf("handleTimerTarget error: %v", err)
			return
		}
		m := Message{
			Type:      "timerTarget",
			Countdown: countdown,
			Counter:   counter,
			Data:      target,
		}
		server.update(m)
	}
}

func (server *Server) sendTimerCommand(cmd string, msg *osc.Message) {
	if matches := server.timerRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		counter, _ := strconv.Atoi(matches[1])

		if counter < 0 && counter > NumCounters {
			log.Printf("sendTimerCommand: Invalid counter: %d", counter)
			return
		}

		msg := Message{
			Type:    cmd,
			Counter: counter,
		}
		server.update(msg)

	} else {
		log.Printf("matches: %v", matches)
		log.Printf("invalid timer message: %v\n", msg)
	}
}

func (server *Server) sendTimerMessage(cmd string, countdown bool, msg *osc.Message) {
	if matches := server.timerRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		counter, _ := strconv.Atoi(matches[1])

		if counter < 0 && counter > NumCounters {
			log.Printf("sendTimerCommand: Invalid counter: %d", counter)
			return
		}

		var message CountdownMessage

		if err := message.UnmarshalOSC(msg); err != nil {
			log.Printf("Unmarshal %v: %v", msg, err)
		} else {
			debug.Printf("%s: %#v", cmd, message)
			msg := Message{
				Type:             cmd,
				Counter:          counter,
				Countdown:        countdown,
				CountdownMessage: &message,
			}
			server.update(msg)
		}
	} else {
		log.Printf("matches: %v", matches)
		log.Printf("invalid timer message: %v\n", msg)
	}
}

func (server *Server) handlePause(msg *osc.Message) {
	debug.Printf("pause: %#v", msg)
	message := Message{
		Type: "pause",
	}
	server.update(message)
}

func (server *Server) handleResume(msg *osc.Message) {
	debug.Printf("resume: %#v", msg)
	message := Message{
		Type: "resume",
	}
	server.update(message)
}

func (server *Server) handleTimerSignal(msg *osc.Message) {
	debug.Printf("handleTimerSignal: %v", msg)
	if matches := server.timerRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		counter, _ := strconv.Atoi(matches[1])
		var r, g, b, a int32
		err := oscutil.UnmarshalArguments(msg, &r, &g, &b, &a)
		if err != nil {
			log.Printf("handleTimerSignal: %v %v", err, msg)
			return
		}

		colors := make([]color.RGBA, 1)
		colors[0] = color.RGBA{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
			A: uint8(a),
		}
		message := Message{
			Type:    "timerSignal",
			Counter: counter,
			Colors:  colors,
		}

		server.update(message)
	} else {
		log.Printf("handleTimerSignal: Invalid message: %v", msg)
	}
}

/*
 * Source related handlers
 */
func (server *Server) parseSourceMsg(msg *osc.Message, cmd string) {
	if matches := server.sourceRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		counter, _ := strconv.Atoi(matches[1])

		msg := Message{
			Type:    cmd,
			Counter: counter - 1,
		}
		server.update(msg)
	} else {
		log.Printf("matches: %v", matches)
		log.Printf("invalid source message: %v\n", msg)
	}
}

func (server *Server) handleHideAll(msg *osc.Message) {
	debug.Printf("handleHide: %#v", msg)

	message := Message{
		Type: "hideAll",
	}
	server.update(message)
}

func (server *Server) handleShowAll(msg *osc.Message) {
	debug.Printf("handleShowAll: %#v", msg)
	message := Message{
		Type: "showAll",
	}
	server.update(message)
}

func (server *Server) handleHide(msg *osc.Message) {
	debug.Printf("handleHide: %v", msg)
	server.parseSourceMsg(msg, "sourceHide")
}

func (server *Server) handleShow(msg *osc.Message) {
	debug.Printf("handleShow: %v", msg)
	server.parseSourceMsg(msg, "sourceShow")
}

func (server *Server) handleSourceTitle(msg *osc.Message) {
	debug.Printf("handleSourceTitle: %v", msg)

	if matches := server.sourceRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		counter, _ := strconv.Atoi(matches[1])

		var label string
		err := oscutil.UnmarshalArguments(msg, &label)
		if err != nil {
			log.Printf("handleSourceTitle error: %v", err)
			return
		}

		msg := Message{
			Type:    "sourceTitle",
			Counter: counter - 1,
			Data:    label,
		}
		server.update(msg)
	} else {
		log.Printf("matches: %v", matches)
		log.Printf("invalid source message: %v\n", msg)
	}
}

func (server *Server) handleSourceColor(msg *osc.Message) {
	debug.Printf("handleSourceColor: %v", msg)
	if matches := server.sourceRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		counter, _ := strconv.Atoi(matches[1])

		cm := ColorMessage{}
		err := cm.UnmarshalOSC(msg)
		if err != nil {
			log.Printf("colors unmarshal: %v - %v", err, msg)
			return
		}

		m := Message{
			Type:    "sourceColors",
			Counter: counter - 1,
			Colors:  cm.ToRGBA(),
		}
		server.update(m)
	}
}

func (server *Server) handleTitleColors(msg *osc.Message) {
	debug.Printf("handleTitleColor: %v", msg)
	cm := ColorMessage{}
	err := cm.UnmarshalOSC(msg)
	if err != nil {
		log.Printf("colors unmarshal: %v - %v", err, msg)
		return
	}

	m := Message{
		Type:   "titleColors",
		Colors: cm.ToRGBA(),
	}
	server.update(m)
}

/*
 * Clock sync handlers
 */

func (server *Server) handleMedia(msg *osc.Message) {
	debug.Printf("handleMedia: %v", msg)
	message := Message{}
	if msg.Address == "/clock/media/mitti" {
		message.Type = "mitti"
	} else if msg.Address == "/clock/media/millumin" {
		message.Type = "millumin"
	} else if matches := server.timerRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		message.Counter, _ = strconv.Atoi(matches[1])
		message.Type = "timerMedia"
	} else {
		log.Printf("Unknown media message: %v", msg)
		return
	}
	mm := MediaMessage{}
	err := mm.UnmarshalOSC(msg)
	if err != nil {
		log.Printf("error unmarshaling media message: %v", err)
		return
	}

	if mm.uuid == server.uuid {
		// Our own message, ignore
		return
	}

	if server.lastMedia.Before(mm.timeStamp.Time()) {
		server.lastMedia = mm.timeStamp.Time()
		message.MediaMessage = &mm
		server.update(message)
	}
}

func (server *Server) handleResetMedia(msg *osc.Message) {
	var uuid string
	var timeStamp *osc.Timetag

	debug.Printf("handleResetMedia: %v", msg)
	message := Message{}
	if msg.Address == "/clock/resetmedia/mitti" {
		message.Type = "mittiReset"
	} else if msg.Address == "/clock/resetmedia/millumin" {
		message.Type = "milluminReset"
	} else if matches := server.timerRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		message.Counter, _ = strconv.Atoi(matches[1])
		message.Type = "timerResetMedia"
	} else {
		log.Printf("Unknown resetMedia message: %v", msg)
		return
	}

	err := oscutil.UnmarshalArguments(msg, &timeStamp, &uuid)
	if err != nil {
		log.Printf("Unmarshal %v: %v", msg, err)
		return
	}

	if uuid == server.uuid {
		return
	}

	if server.lastMedia.Before(timeStamp.Time()) {
		server.lastMedia = timeStamp.Time()
		server.update(message)
	}
}

func (server *Server) handleLTC(msg *osc.Message) {
	var message TimeMessage

	msgType := "LTC"

	if msg.Address == "/clock/ltc2" {
		msgType = "LTC2"
	}

	if err := message.UnmarshalOSC(msg); err != nil {
		log.Printf("Unmarshal %v: %v", msg, err)
	} else {
		debug.Printf("LTC: %v\n", message.Time)
		m := Message{
			Type: msgType,
			Data: message.Time,
		}
		server.update(m)
	}
}

/*
 * Misc commands
 */

func (server *Server) handleSecondsOff(msg *osc.Message) {
	debug.Printf("Second display off: %v\n", msg)
	message := Message{
		Type: "secondsOff",
	}
	server.update(message)
}

func (server *Server) handleSecondsOn(msg *osc.Message) {
	debug.Printf("Second display on: %v\n", msg)
	message := Message{
		Type: "secondsOn",
	}
	server.update(message)
}

func (server *Server) handleTimeSet(msg *osc.Message) {
	var message TimeMessage

	if err := message.UnmarshalOSC(msg); err != nil {
		log.Printf("Unmarshal %v: %v", msg, err)
	} else {
		debug.Printf("Set time: %v\n", message.Time)
		m := Message{
			Type: "setTime",
			Data: message.Time,
		}
		server.update(m)
	}
}

func (server *Server) handleBackground(msg *osc.Message) {
	debug.Printf("background: %v", msg)
	var bg int32
	err := oscutil.UnmarshalArguments(msg, &bg)
	if err != nil {
		log.Printf("Background msg error: %v", err)
		return
	}
	m := Message{
		Type:    "background",
		Counter: int(bg),
	}
	server.update(m)
}

func (server *Server) handleInfo(msg *osc.Message) {
	debug.Printf("handleInfo")
	var bg int32
	err := oscutil.UnmarshalArguments(msg, &bg)
	if err != nil {
		log.Printf("Info msg error: %v", err)
		return
	}
	m := Message{
		Type:    "showInfo",
		Counter: int(bg),
	}
	server.update(m)
}

func (server *Server) handleFlash(msg *osc.Message) {
	debug.Printf("handleFlash")
	m := Message{
		Type: "screenFlash",
	}
	server.update(m)
}

func (server *Server) handleHardwareSignal(msg *osc.Message) {
	debug.Printf("handleHardwareSignal: %v", msg)
	if matches := server.signalRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		counter, _ := strconv.Atoi(matches[1])
		var r, g, b int32
		err := oscutil.UnmarshalArguments(msg, &r, &g, &b)
		if err != nil {
			log.Printf("handleHardwareSignal: %v %v", err, msg)
			return
		}

		colors := make([]color.RGBA, 1)
		colors[0] = color.RGBA{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
		}
		message := Message{
			Type:    "hardwareSignal",
			Counter: counter,
			Colors:  colors,
		}

		server.update(message)
	} else {
		log.Printf("handleHardwareSignal: Invalid message: %v", msg)
	}
}

func (server *Server) handleSignalAutomation(msg *osc.Message) {
	var v bool
	debug.Printf("handleSignalAutomation: %v", msg)
	err := oscutil.UnmarshalArguments(msg, &v)
	if err != nil {
		log.Printf("handleSignalAutomation: %v %v", err, msg)
		return
	}
	message := Message{
		Type:      "signalAutomation",
		Countdown: v, // FIXME: Ugly hack!
	}
	server.update(message)
}

/*
 * Deprecated message handlers awaiting removal
 */

func (server *Server) handleDisplayText(msg *osc.Message) {
	debug.Printf("handleText")
	var message displayTextMessage
	if err := message.UnmarshalOSC(msg); err != nil {
		log.Printf("handleText unmarshal: %v: %v", msg, err)
	} else {
		m := Message{
			Type:               "displayText",
			DisplayTextMessage: &message,
		}
		server.update(m)
	}
}

func (server *Server) handleCountupStart(msg *osc.Message) {
	debug.Printf("countup start: %#v", msg)

	if len(msg.Arguments) != 0 {
		log.Printf("handleCountupStart: too many arguments")
		return
	}

	if msg.Address == "/clock/countup/start" {
		msg.Address = "/clock/timer/0/countup"
	}

	if matches := server.timerRegexp.FindStringSubmatch(msg.Address); len(matches) == 2 {
		counter, _ := strconv.Atoi(matches[1])

		msg := Message{
			Type:             "timerStart",
			Counter:          counter,
			Countdown:        false,
			CountdownMessage: &CountdownMessage{Seconds: 0},
		}
		server.update(msg)

	} else {
		log.Printf("matches: %v", matches)
		log.Printf("invalid timer message: %v\n", msg)
	}
}

func (server *Server) handleCountdownStart(msg *osc.Message) {
	log.Printf("handleCountdownStart: %v", msg)
	if msg.Address == "/clock/countdown/start" {
		msg.Address = "/clock/timer/0/start"
	} else if msg.Address == "/clock/countdown2/start" {
		msg.Address = "/clock/timer/1/start"
	}
	server.sendTimerMessage("timerStart", true, msg)
}

func (server *Server) handleDualText(msg *osc.Message) {
	var message TextMessage
	if err := message.UnmarshalOSC(msg); err != nil {
		log.Printf("Unmarshal %v: %v", msg, err)
	} else {
		debug.Printf("Dual clock text: %v\n", message.Text)
		m := Message{
			Type: "dualText",
			Data: message.Text,
		}
		server.update(m)
	}
}

func (server *Server) handleDisplay(msg *osc.Message) {
	var message DisplayMessage

	if err := message.UnmarshalOSC(msg); err != nil {
		log.Printf("Unmarshal %v: %v", msg, err)
	} else {
		debug.Printf("display: %#v", message)
		msg := Message{
			Type:           "display",
			DisplayMessage: &message,
		}
		server.update(msg)
	}
}

func (server *Server) handleCue(msg *osc.Message) {
	debug.Printf("handleCue: %v", msg)

	var uuid string
	var blank bool
	var m Message

	err := oscutil.UnmarshalArgument(msg, 0, &uuid)
	if err != nil {
		log.Printf("handleCue error: %v msg: %v", err, msg)
		return
	}

	if uuid == server.uuid {
		return
	}

	switch msg.Address {
	case "/clock/cue/right":
		m.Type = "cueRight"
	case "/clock/cue/left":
		m.Type = "cueLeft"
	case "/clock/cue/blank":
		err = oscutil.UnmarshalArgument(msg, 1, &blank)
		if err != nil {
			log.Printf("handleCue blank error: %v msg: %v", err, msg)
			return
		}
		m.Type = "cueBlank"
		m.Countdown = blank
	default:
		log.Printf("handleCue error: Unknown message address: %s", msg.Address)
	}

	server.update(m)

}

// Le huge registerHandler block
func (server *Server) setupDispatch(d *oscutil.RegexpDispatcher) {
	// Sync messages
	registerHandler(d, "^/clock/media/*", server.handleMedia)
	registerHandler(d, "^/clock/resetmedia/*", server.handleResetMedia)
	registerHandler(d, "^/clock/ltc*", server.handleLTC)

	// Timer related
	registerHandler(d, "^/clock/timer/*/countdown/target$", server.handleCountdownTarget)
	registerHandler(d, "^/clock/timer/*/countdown$", server.handleCountdownStart)
	registerHandler(d, "^/clock/timer/*/countup/target$", server.handleCountupTarget)
	registerHandler(d, "^/clock/timer/*/countup$", server.handleCountupStart)
	registerHandler(d, "^/clock/timer/*/modify$", server.handleTimerModify)
	registerHandler(d, "^/clock/timer/*/signal$", server.handleTimerSignal)
	registerHandler(d, "^/clock/timer/*/stop$", server.handleTimerStop)
	registerHandler(d, "^/clock/timer/*/pause$", server.handleTimerPause)
	registerHandler(d, "^/clock/timer/*/resume$", server.handleTimerResume)
	registerHandler(d, "^/clock/timer/*/media$", server.handleMedia)
	registerHandler(d, "^/clock/timer/*/resetmedia$", server.handleResetMedia)
	registerHandler(d, "^/clock/timer/*/restart$", server.handleTimerRestart)
	registerHandler(d, "^/clock/timer/*/set", server.handleTimerSet)
	registerHandler(d, "^/clock/pause$", server.handlePause)
	registerHandler(d, "^/clock/resume$", server.handleResume)
	registerHandler(d, "^/clock/limitimer/*", server.handleLimitimer)

	// Source related
	registerHandler(d, "^/clock/source/*/hide$", server.handleHide)
	registerHandler(d, "^/clock/source/*/show$", server.handleShow)
	registerHandler(d, "^/clock/source/*/title$", server.handleSourceTitle)
	registerHandler(d, "^/clock/source/*/color$", server.handleSourceColor)
	registerHandler(d, "^/clock/hide$", server.handleHideAll)
	registerHandler(d, "^/clock/show$", server.handleShowAll)

	// Misc commands
	registerHandler(d, "^/clock/background$", server.handleBackground)
	registerHandler(d, "^/clock/info$", server.handleInfo)
	registerHandler(d, "^/clock/text$", server.handleDisplayText)
	registerHandler(d, "^/clock/titlecolors$", server.handleTitleColors)
	registerHandler(d, "^/clock/seconds/off$", server.handleSecondsOff)
	registerHandler(d, "^/clock/seconds/on$", server.handleSecondsOn)
	registerHandler(d, "^/clock/time/set$", server.handleTimeSet)
	registerHandler(d, "^/clock/flash$", server.handleFlash)
	registerHandler(d, "^/clock/signal/*", server.handleHardwareSignal)
	registerHandler(d, "^/clock/automation$", server.handleSignalAutomation)

	// Cues
	registerHandler(d, "^/clock/cue/right$", server.handleCue)
	registerHandler(d, "^/clock/cue/left$", server.handleCue)
	registerHandler(d, "^/clock/cue/blank$", server.handleCue)

	// Deprecated
	registerHandler(d, "^/clock/dual/text$", server.handleDualText)
	registerHandler(d, "^/clock/kill$", server.handleHideAll)
	registerHandler(d, "^/clock/normal$", server.handleShowAll)
	registerHandler(d, "^/clock/countup/start$", server.handleCountupStart)
	registerHandler(d, "^/clock/countup/modify$", server.handleTimerModify)
	registerHandler(d, "^/clock/display$", server.handleDisplay)
	registerHandler(d, "^/clock/countdown/start$", server.handleCountdownStart)
	registerHandler(d, "^/clock/countdown2/start$", server.handleCountdownStart)
	registerHandler(d, "^/clock/countdown/modify$", server.handleTimerModify)
	registerHandler(d, "^/clock/countdown2/modify$", server.handleTimerModify)
	registerHandler(d, "^/clock/countdown/stop$", server.handleTimerStop)
	registerHandler(d, "^/clock/countdown2/stop$", server.handleTimerStop)
}

func registerHandler(d *oscutil.RegexpDispatcher, addr string, handler osc.MethodFunc) {
	if err := d.AddMsgHandler(addr, handler); err != nil {
		panic(err)
	}
}
