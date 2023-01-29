package clock

import (
	"fmt"
	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	// "gitlab.com/clock-8001/clock-8001/v4/oscutil"
	"github.com/chabad360/go-osc/osc"

	"image/color"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Listen for OSC messages
func (engine *Engine) listen() {
	engine.wg.Add(1)
	defer engine.wg.Done()

	oscChan := engine.clockServer.Listen()
	engine.messageTimer.Stop()

	engine.ltcTimer = timer.NewTimer(engine.timeout)
	engine.ltcTimer.Stop() // Needed to prevent a timeout at the start

	engine.mittiTimer = timer.NewTimer(updateTimeout)
	engine.milluminTimer = timer.NewTimer(updateTimeout)
	engine.flashTimer = timer.NewTimer(flashDuration)
	engine.mediaTimeouts = make([]*time.Time, numCounters)

	engine.cueTimer = timer.NewTimer(engine.cueDuration)
	engine.cueTimer.Stop()

	mediaTicker := time.NewTicker(mediaTickRate)

	stateTicker := time.NewTicker(stateTimer)
	udpTicker := time.NewTicker(udpTimer)

	for {
		select {
		case <-engine.ctx.Done():
			log.Printf("engine.listen() quitting")
			return
		case message := <-oscChan:
			// New OSC message received

			debug.Printf("Got new osc data: %v\n", message)

			engine.handleOSC(&message)

			// We have received a osc command, so stop the version display
			engine.initialized = true

		case <-engine.flashTimer.C:
			engine.screenFlash = false
		case <-engine.mittiTimer.C:
			engine.mittiCounter.ResetMedia()
		case <-engine.milluminTimer.C:
			engine.milluminCounter.ResetMedia()
		case <-engine.messageTimer.C:
			// OSC message timeout
			engine.message = ""
			engine.oscTally = false
		case <-engine.ltcTimer.C:
			// LTC message timeout
			engine.ltcTimeout = true
		case <-stateTicker.C:
			// Send OSC feedback
			state := engine.State()
			if err := engine.sendState(state); err != nil {
				log.Printf("Error sending osc state: %v", err)
			}
		case <-udpTicker.C:
			engine.sendUDPTimers()
		case <-mediaTicker.C:
			for i := range engine.mediaTimeouts {
				if t := engine.mediaTimeouts[i]; t != nil && time.Now().After(*t) {
					engine.Counters[i].ResetMedia()
					engine.mediaTimeouts[i] = nil
				}
			}
		case cs := <-engine.cueChan:
			cs.Show = true
			engine.cueState = cs
			if cs.Blank {
				engine.cueTimer.Stop()
			} else {
				engine.cueTimer.Reset(engine.cueDuration)
			}
			engine.sendCue(cs)
		case <-engine.cueTimer.C:
			engine.cueState.Show = false
		}
	}
}

func (engine *Engine) handleOSC(message *Message) {
	switch message.Type {
	case "timerStart":
		time := time.Duration(message.CountdownMessage.Seconds) * time.Second
		engine.StartCounter(message.Counter, message.Countdown, time)
	case "timerModify":
		time := time.Duration(message.CountdownMessage.Seconds) * time.Second
		engine.ModifyCounter(message.Counter, time)
	case "timerStop":
		engine.StopCounter(message.Counter)
	case "timerTarget":
		engine.TargetCounter(message.Counter, message.Data, message.Countdown)
	case "timerPause":
		engine.PauseCounter(message.Counter)
	case "timerResume":
		engine.ResumeCounter(message.Counter)
	case "timerMedia":
		m := message.MediaMessage
		engine.Counters[message.Counter].SetMedia(m.hours, m.minutes, m.seconds, m.frames, time.Duration(m.remaining)*time.Second, float64(m.progress), m.paused, m.looping)
		t := time.Now().Add(mediaTimeout)
		engine.mediaTimeouts[message.Counter] = &t
	case "timerResetMedia":
		engine.Counters[message.Counter].ResetMedia()
		engine.mediaTimeouts[message.Counter] = nil
	case "display":
		msg := message.DisplayMessage
		log.Printf("Setting tally message to: %s", msg.Text)

		engine.message = msg.Text
		engine.messageColor = &color.RGBA{
			R: uint8(msg.ColorRed),
			G: uint8(msg.ColorBlue),
			B: uint8(msg.ColorGreen),
			A: 255,
		}

		engine.messageBG = &color.RGBA{
			R: 0,
			G: 0,
			B: 0,
			A: 255,
		}

		// Mark the OSC message state as active
		engine.oscTally = true

		// Reset the timer that will clear the message when it expires
		engine.messageTimer.Reset(engine.timeout)

	case "displayText":
		msg := message.DisplayTextMessage
		log.Printf("Displaying text: %v", msg)

		engine.message = msg.text
		engine.messageColor = &color.RGBA{
			R: uint8(msg.r),
			G: uint8(msg.g),
			B: uint8(msg.b),
			A: uint8(msg.a),
		}
		engine.messageBG = &color.RGBA{
			R: uint8(msg.bgR),
			G: uint8(msg.bgG),
			B: uint8(msg.bgB),
			A: uint8(msg.bgA),
		}
		engine.oscTally = true
		if msg.time != 0 {
			engine.messageTimer.Reset(time.Duration(msg.time) * time.Second)
		}
	case "pause":
		engine.Pause()
	case "resume":
		engine.Resume()
	case "hideAll":
		engine.hideAll()
	case "showAll":
		engine.showAll()
	case "secondsOff":
		engine.displaySeconds = false
	case "secondsOn":
		engine.displaySeconds = true
	case "setTime":
		engine.setTime(message.Data)
	case "LTC":
		if engine.ltcEnabled {
			engine.setLTC(message.Data)
			engine.ltcTimer.Reset(engine.timeout)
		}
	case "dualText":
		engine.message = fmt.Sprintf("%-.8s", message.Data)
	case "mitti":
		engine.mittiTimer.Reset(updateTimeout)

		m := message.MediaMessage
		engine.mittiCounter.SetMedia(m.hours, m.minutes, m.seconds, m.frames, time.Duration(m.remaining)*time.Second, float64(m.progress), m.paused, m.looping)
	case "mittiReset":
		engine.mittiCounter.ResetMedia()
	case "millumin:":
		engine.milluminTimer.Reset(updateTimeout)

		m := message.MediaMessage
		engine.milluminCounter.SetMedia(m.hours, m.minutes, m.seconds, m.frames, time.Duration(m.remaining)*time.Second, float64(m.progress), m.paused, m.looping)
	case "milluminReset":
		engine.milluminCounter.ResetMedia()
	case "background":
		// FIXME: non semantic ugliness
		engine.background = message.Counter
	case "sourceHide":
		if message.Counter >= 0 &&
			message.Counter < len(engine.sources) {

			engine.sources[message.Counter].hidden = true
		}
	case "sourceShow":
		if message.Counter >= 0 &&
			message.Counter < len(engine.sources) {
			engine.sources[message.Counter].hidden = false
		}
	case "sourceTitle":
		if message.Counter >= 0 &&
			message.Counter < len(engine.sources) {

			engine.sources[message.Counter].title = message.Data
		}
	case "showInfo":
		engine.showInfo = true
		engine.infoTimer.Reset(time.Duration(message.Counter) * time.Second)
	case "sourceColors":
		if message.Counter >= 0 &&
			message.Counter < len(engine.sources) &&
			len(message.Colors) == 2 {

			log.Printf("Setting source %d colors: %v - %v", message.Counter+1, message.Colors[0], message.Colors[1])
			engine.SetSourceColors(message.Counter, message.Colors[0], message.Colors[1])
		}
	case "titleColors":
		if len(message.Colors) == 2 {
			log.Printf("Setting title colors: %v - %v", message.Colors[0], message.Colors[1])
			engine.SetTitleColors(message.Colors[0], message.Colors[1])
		}
	case "screenFlash":
		engine.screenFlash = true
		engine.flashTimer.Reset(flashDuration)
	case "timerSignal":
		if message.Counter >= 0 &&
			message.Counter < len(engine.sources) &&
			len(message.Colors) == 1 {
			engine.Counters[message.Counter].signalColor = message.Colors[0]
		}
	case "hardwareSignal":
		if message.Counter == engine.signalHardware && len(message.Colors) == 1 {
			engine.signalHardwareColor = message.Colors[0]
		}
	case "signalAutomation":
		for i := range engine.Counters {
			engine.Counters[i].autoSignals = message.Countdown
		}
	case "limitimer":
		engine.Counters[message.Counter].parseLimitimer(message.LimitimerMessage)
	case "cueRight":
		cs := cueState{
			RightArrow: true,
			Show:       true,
		}
		engine.cueState = &cs
		engine.cueTimer.Reset(engine.cueDuration)
	case "cueLeft":
		cs := cueState{
			LeftArrow: true,
			Show:      true,
		}
		engine.cueState = &cs
		engine.cueTimer.Reset(engine.cueDuration)
	case "cueBlank":
		cs := cueState{
			Blank: message.Countdown,
			Show:  message.Countdown,
		}
		engine.cueState = &cs
		engine.cueTimer.Stop()
	}
}

// Sends the OSC feedback messages
func (engine *Engine) sendState(state *State) error {
	if engine.oscDests == nil {
		// No osc connection
		return nil
	}
	t := time.Now()
	engine.sendLegacyState(state)

	bundle := osc.NewBundleWithTime(time.Now())

	for i, s := range state.Clocks {
		addr := fmt.Sprintf("/clock/source/%d/state", i+1)
		state := sourceStateMessageFromClock(s, engine.uuid)

		packet := state.MarshalOSC(addr)
		bundle.Append(packet)
	}

	for i, c := range engine.Counters {
		addr := fmt.Sprintf("/clock/timer/%d/state", i)
		out := c.Output(t)
		state := timerStateMessageFromOutput(out, engine.uuid)

		packet := state.MarshalOSC(addr)
		bundle.Append(packet)
	}

	data, err := bundle.MarshalBinary()
	if err != nil {
		return err
	}
	engine.oscSendChan <- data
	return nil
}

// Send the clock state as /clock/state
func (engine *Engine) sendLegacyState(state *State) error {
	if engine.oscDests == nil {
		// No osc connection
		return nil
	}

	var hours, minutes, seconds string

	if !state.Clocks[0].Expired {
		// HH:MM:SS(:FF for LTC)
		parts := strings.Split(state.Clocks[0].Text, ":")
		if len(parts) > 2 {
			hours = parts[0]
			minutes = parts[1]
			seconds = parts[2]
		}
	} else {
		hours = "00"
		minutes = "00"
		seconds = "00"
	}
	mode := state.Clocks[0].Mode
	pause := int32(0)

	if state.Clocks[0].Paused {
		pause = 1
	}

	packet := osc.NewMessage("/clock/state", int32(mode), hours, minutes, seconds, state.Tally, pause)

	data, err := packet.MarshalBinary()
	if err != nil {
		return err
	}
	engine.oscSendChan <- data

	return nil
}

// sendCue sends the engine cue events as OSC messages
func (engine *Engine) sendCue(cs *cueState) {
	var msg *osc.Message

	if cs.RightArrow {
		msg = osc.NewMessage("/clock/cue/right", engine.uuid)
	} else if cs.LeftArrow {
		msg = osc.NewMessage("/clock/cue/left", engine.uuid)
	} else {
		msg = osc.NewMessage("/clock/cue/blank", engine.uuid, cs.Blank)
	}

	data, err := msg.MarshalBinary()
	if err != nil {
		log.Printf("sendCue: error marshalling osc message: %v", err)
		return
	}

	engine.oscSendChan <- data
}

/*
 * OSC Message handlers
 */

// StartCounter starts a counter
func (engine *Engine) StartCounter(counter int, countdown bool, timer time.Duration) {
	if counter < 0 || counter >= numCounters {
		log.Printf("engine.StartCounter: illegal counter number %d (have %d counters)\n", counter, numCounters)
	}

	engine.Counters[counter].Start(countdown, timer)
	engine.activateSourceByCounter(counter)
}

// ModifyCounter adds or removes time from a counter
func (engine *Engine) ModifyCounter(counter int, delta time.Duration) {
	if counter < 0 || counter >= numCounters {
		log.Printf("engine.StartCounter: illegal counter number %d (have %d counters)\n", counter, numCounters)
	}

	engine.Counters[counter].Modify(delta)
}

// StopCounter stops a given counter
func (engine *Engine) StopCounter(counter int) {
	if counter < 0 || counter >= numCounters {
		log.Printf("engine.StartCounter: illegal counter number %d (have %d counters)\n", counter, numCounters)
	}

	engine.Counters[counter].Stop()
	if engine.Counters[counter].autoSignals {
		engine.Counters[counter].signalColor = color.RGBA{R: 0, G: 0, B: 0, A: 0}
	}
}

// PauseCounter pauses a given counter
func (engine *Engine) PauseCounter(counter int) {
	if counter < 0 || counter >= numCounters {
		log.Printf("engine.StartCounter: illegal counter number %d (have %d counters)\n", counter, numCounters)
	}
	engine.Counters[counter].Pause()
}

// ResumeCounter resumes a paused counter
func (engine *Engine) ResumeCounter(counter int) {
	if counter < 0 || counter >= numCounters {
		log.Printf("engine.StartCounter: illegal counter number %d (have %d counters)\n", counter, numCounters)
	}
	engine.Counters[counter].Resume()
}

// TargetCounter sets the target time and date for a counter
func (engine *Engine) TargetCounter(counter int, target string, countdown bool) {
	if counter < 0 || counter >= numCounters {
		log.Printf("engine.TargetCounter: illegal counter number %d (have %d counters)\n", counter, numCounters)
	}

	match, err := regexp.MatchString("^([0-1]?[0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])$", target)
	if match && err == nil {
		tz := engine.sources[0].tz
		now := time.Now().In(tz)
		t, err := time.ParseInLocation("15:04:05", target, tz)
		if err != nil {
			log.Printf("TargetCounter error: %v", err)
			return
		}
		target := time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			t.Hour(),
			t.Minute(),
			t.Second(),
			0,
			tz)

		if target.Before(now) && countdown {
			target = target.Add(24 * time.Hour)
		} else if target.After(now) && !countdown {
			target = target.Add(-24 * time.Hour)
		}

		engine.Counters[counter].Target(target)
		engine.activateSourceByCounter(counter)
		debug.Printf("Counter target set!")
	} else {
		log.Printf("Illegal timer target string, string: %v, err: %v", target, err)
	}
}

// showAll returns main display to normal clock
func (engine *Engine) showAll() {
	for _, s := range engine.sources {
		s.hidden = false
	}
}

// hideAll blanks the clock display
func (engine *Engine) hideAll() {
	for _, s := range engine.sources {
		s.hidden = true
	}
}

// Pause pauses all timers
func (engine *Engine) Pause() {
	for _, c := range engine.Counters {
		c.Pause()
	}
}

// Resume resumes all timers
func (engine *Engine) Resume() {
	for _, c := range engine.Counters {
		c.Resume()
	}
}

/* Legacy handlers */

// DisplaySeconds returns true if the clock should display seconds
func (engine *Engine) DisplaySeconds() bool {
	return engine.displaySeconds
}

func (engine *Engine) setTime(time string) {
	debug.Printf("Set time: %#v", time)
	_, lookErr := exec.LookPath("date")
	if lookErr != nil {
		debug.Printf("Date binary not found, cannot set system date: %s\n", lookErr.Error())
		return
	}
	// Validate the received time
	match, _ := regexp.MatchString("^(2[0-3]|[01][0-9]):([0-5][0-9]):([0-5][0-9])$", time)
	if match {
		// Set the system time
		dateString := fmt.Sprintf("2019-01-01 %s", time)
		tzString := fmt.Sprintf("TZ=%s", engine.sources[0].tz.String())
		debug.Printf("Setting system date to: %s\n", dateString)
		args := []string{"--set", dateString}
		cmd := exec.Command("date", args...) // #nosec we have strict validation in place
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, tzString)
		if err := cmd.Run(); err != nil {
			panic(err)
		}
	} else {
		debug.Printf("Invalid time provided: %v\n", time)
	}
}

// LtcActive returns true if the clock is displaying LTC timecode
func (engine *Engine) LtcActive() bool {
	return engine.mode == LTC
}

func (engine *Engine) setLTC(timestamp string) {
	match, _ := regexp.MatchString("^([0-9][0-9]):([0-5][0-9]):([0-5][0-9]):([0-9][0-9])$", timestamp)
	if match {
		parts := strings.Split(timestamp, ":")
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.Atoi(parts[2])
		frames, _ := strconv.Atoi(parts[3])
		var ltcTarget time.Time
		if frames == 0 {
			// Update the LTC countdown target
			ltcDuration := time.Duration(hours) * time.Hour
			ltcDuration += time.Duration(minutes) * time.Minute
			ltcDuration += time.Duration(seconds) * time.Second
			ltcTarget = time.Now().Add(-ltcDuration)
		} else {
			ltcTarget = engine.ltc.target
		}
		engine.mode = LTC
		engine.ltcActive = true
		engine.oscTally = true
		engine.ltcTimeout = false
		engine.ltc = &ltcData{
			hours:   hours,
			minutes: minutes,
			seconds: seconds,
			frames:  frames + 1,
			target:  ltcTarget,
		}
	}
}
