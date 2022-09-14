package clock

import (
	"context"
	"fmt"
	"github.com/denisbrodbeck/machineid"
	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	"gitlab.com/clock-8001/clock-8001/v4/oscutil"
	"gitlab.com/clock-8001/clock-8001/v4/udptime"
	// "gitlab.com/Depili/go-osc/osc"
	"github.com/chabad360/go-osc/osc"

	"image/color"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	db "runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MakeEngine creates a clock engine
func MakeEngine(options *EngineOptions) (*Engine, error) {
	var engine = Engine{
		mode:               Normal,
		displaySeconds:     true,
		oscTally:           false,
		timeout:            time.Duration(options.Timeout) * time.Millisecond,
		initialized:        false,
		oscDests:           nil,
		ltcShowSeconds:     options.LTCSeconds,
		ltcFollow:          options.LTCFollow,
		ltcEnabled:         !options.DisableLTC,
		ltcActive:          false,
		format12h:          options.Format12h,
		off:                false,
		messageColor:       &color.RGBA{255, 255, 155, 255},
		signalHardware:     options.SignalHardware,
		limitimer:          options.LimitimerMode,
		limitimerSerial:    options.LimitimerSerial,
		wg:                 &sync.WaitGroup{},
		flashPeriod:        options.Flash,
		overtimeVisibility: options.OvertimeVisibility,
	}
	engine.ctx, engine.cancelFunc = context.WithCancel(context.Background())
	uuid, err := machineid.ProtectedID("clock-8001")
	if err != nil {
		log.Fatalf("Failed to generate unique identifier: %v", err)
	}
	engine.uuid = uuid

	log.Printf("Source1: %v", options.Source1)
	log.Printf("Source2: %v", options.Source2)
	log.Printf("Source3: %v", options.Source3)
	log.Printf("Source4: %v", options.Source4)

	ltc := ltcData{hours: 0}
	engine.ltc = &ltc

	engine.messageTimer = timer.NewTimer(engine.timeout)
	engine.messageTimer.Stop()

	engine.printVersion()
	err = engine.initCounters(options)
	if err != nil {
		log.Fatalf("Failed to initialize counters: %v", err)
	}

	sources := make([]*SourceOptions, 4)
	sources[0] = options.Source1
	sources[1] = options.Source2
	sources[2] = options.Source3
	sources[3] = options.Source4

	if err := engine.initSources(sources); err != nil {
		log.Printf("Error initializing engine clock sources: %v", err)
		return nil, err
	}
	engine.initOSC(options)

	// Millumin ignore regexp
	regexp, err := regexp.Compile("(?i)" + options.Ignore)
	if err != nil {
		log.Fatalf("Invalid --millumin-ignore-layer=%v: %v", options.Ignore, err)
	}
	engine.ignoreRegexp = regexp

	engine.prepareInfo()

	engine.infoTimer = timer.NewTimer(time.Duration(options.ShowInfo) * time.Second)
	go engine.infoTimeout()
	engine.showInfo = true
	fmt.Printf(engine.info)

	engine.initPicturall(options)
	engine.initVmix(options)
	engine.initTricaster(options)
	engine.initUDPTime(options)
	engine.initLimitimer(options)

	return &engine, nil
}

// Close stops the engine and closes all open connections and files
func (engine *Engine) Close() {
	log.Printf("Stopping the clock engine on request...")
	engine.cancelFunc()
	engine.wg.Wait()
	log.Printf("Clock engine stopped")
}

func (engine *Engine) listenUDPTime() {
	engine.wg.Add(1)
	defer engine.wg.Done()
	chan1, err := udptime.Listen(engine.ctx, "0.0.0.0:36700", engine.wg)
	if err != nil {
		log.Printf("UDPTime listen error: %v", err)
		return
	}
	chan2, err := udptime.Listen(engine.ctx, "0.0.0.0:36701", engine.wg)
	if err != nil {
		log.Printf("UDPTime listen error: %v", err)
		return
	}

	for {
		var msg *udptime.Message
		var t int
		select {
		case msg = <-chan1:
			t = 0
		case msg = <-chan2:
			t = 1
		case <-engine.ctx.Done():
			return
		}
		icon := ""
		if msg.OverTime {
			icon = "+"
		}
		engine.udpCounters[t].SetSlave(0, msg.Minutes, msg.Seconds, true, icon)
	}
}

func (engine *Engine) prepareInfo() {
	info := fmt.Sprintf("Clock-8001 version: %v\n\n", gitTag)
	info += fmt.Sprintf("ID: %v\n", engine.uuid[len(engine.uuid)-8:])
	info += fmt.Sprintf("IP-addresses:\n%s", clockAddresses())

	if engine.oscServer.Addr != "" {
		info += fmt.Sprintf("OSC-listen: %s\n", engine.oscServer.Addr)
	}

	engine.info = info
}

func (engine *Engine) infoTimeout() {
	for range engine.infoTimer.C {
		engine.showInfo = false
	}
}

func (engine *Engine) runOSC() {
	engine.wg.Add(1)
	defer engine.wg.Done()

	err := engine.oscBridge()
	if err != nil {
		panic(err)
	}

	for {
		err = engine.oscServer.ListenAndServe()
		if err != nil {
			if e, ok := err.(*net.OpError); ok {
				if e.Temporary() {
					log.Printf("OSC-listen: Temporary error: %v. Retrying", e)
				} else {
					log.Printf("OSC-listen fatal error: %v. Giving up", e)
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

// Listen for OSC messages
func (engine *Engine) listen() {
	engine.wg.Add(1)
	defer engine.wg.Done()

	oscChan := engine.clockServer.Listen()
	engine.messageTimer.Stop()
	ltcTimer := timer.NewTimer(engine.timeout)
	ltcTimer.Stop() // Needed to prevent a timeout at the start

	mittiTimer := timer.NewTimer(updateTimeout)
	milluminTimer := timer.NewTimer(updateTimeout)
	stateTicker := time.NewTicker(stateTimer)
	udpTicker := time.NewTicker(udpTimer)
	flashTimer := timer.NewTimer(flashDuration)

	for {
		select {
		case <-engine.ctx.Done():
			log.Printf("engine.listen() quitting")
			err := engine.oscServer.Close()
			if err != nil {
				log.Printf("-> error: %v", err)
			}
			return
		case message := <-oscChan:
			// New OSC message received
			debug.Printf("Got new osc data: %v\n", message)
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
					ltcTimer.Reset(engine.timeout)
				}
			case "dualText":
				engine.message = fmt.Sprintf("%-.8s", message.Data)
			case "mitti":
				mittiTimer.Reset(updateTimeout)

				m := message.MediaMessage
				engine.mittiCounter.SetMedia(m.hours, m.minutes, m.seconds, m.frames, time.Duration(m.remaining)*time.Second, m.progress, m.paused, m.looping)
			case "mittiReset":
				engine.mittiCounter.ResetMedia()
			case "millumin:":
				milluminTimer.Reset(updateTimeout)

				m := message.MediaMessage
				engine.milluminCounter.SetMedia(m.hours, m.minutes, m.seconds, m.frames, time.Duration(m.remaining)*time.Second, m.progress, m.paused, m.looping)
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
				flashTimer.Reset(flashDuration)
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
			}
			// We have received a osc command, so stop the version display
			engine.initialized = true

		case <-flashTimer.C:
			engine.screenFlash = false
		case <-mittiTimer.C:
			engine.mittiCounter.ResetMedia()
		case <-milluminTimer.C:
			engine.milluminCounter.ResetMedia()
		case <-engine.messageTimer.C:
			// OSC message timeout
			engine.message = ""
			engine.oscTally = false
		case <-ltcTimer.C:
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
		}
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

func (engine *Engine) sendUDPTimers() {
	t := time.Now()
	for i, conn := range engine.udpDests {
		c := engine.udpCounters[i].Output(t)
		mins := c.Minutes
		secs := c.Seconds

		if c.Hours != 0 {
			// If timer is over an hour send hours and minutes
			mins = c.Hours
			secs = c.Minutes
		}
		msg := udptime.Message{
			Minutes: mins,
			Seconds: secs,
			Red:     c.Countdown,
			Green:   !c.Countdown,
		}
		data := udptime.Encode(&msg)
		conn.Write(data)
	}
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

func (engine *Engine) flash(t time.Time) bool {
	if t.Nanosecond() < engine.flashPeriod*1000000 {
		return true
	}
	return false
}

// State creates a snapshot of the clock state for display on clock faces
func (engine *Engine) State() *State {
	t := time.Now()
	var clocks []*Clock
	for _, s := range engine.sources {
		c := Clock{
			Text:        "",
			Compact:     "",
			Label:       s.title,
			Icon:        "",
			Expired:     false,
			Hidden:      s.hidden,
			TextColor:   s.textColor,
			BGColor:     s.bgColor,
			HideSeconds: !engine.displaySeconds,
			SignalColor: color.RGBA{R: 0, G: 0, B: 0, A: 0},
		}

		if s.timer {
			c.SignalColor = s.counter.signalColor
		}

		if s.ltc && engine.ltcActive {
			engine.ltcState(&c, s)
		} else if out := s.counter.Output(t); s.timer && out.Active {
			engine.timerState(&c, s, out)
		} else if s.tod {
			engine.todState(&c, s, t)
		}

		clocks = append(clocks, &c)
	}
	state := State{
		Initialized:         engine.initialized,
		Clocks:              clocks,
		Flash:               engine.flash(t),
		TallyColor:          &color.RGBA{},
		Background:          engine.background,
		TitleColor:          engine.titleTextColor,
		TitleBGColor:        engine.titleBGColor,
		ScreenFlash:         engine.screenFlash,
		HardwareSignalColor: engine.signalHardwareColor,
	}

	if engine.showInfo {
		engine.prepareInfo()
		state.Info = engine.info
	}

	if engine.oscTally {
		state.Tally = engine.message
		state.TallyColor = engine.messageColor
		state.TallyBG = engine.messageBG
	}

	return &state
}

func (engine *Engine) todState(c *Clock, s *source, t time.Time) {
	// Time of day
	c.Mode = Normal
	if engine.format12h {
		c.Text = t.In(s.tz).Format("03:04:05")
		c.Hours = t.In(s.tz).Hour() % 12
	} else {
		c.Text = t.In(s.tz).Format("15:04:05")
		c.Hours = t.In(s.tz).Hour()
	}
	c.Minutes = t.In(s.tz).Minute()
	c.Seconds = t.In(s.tz).Second()

	// Hide seconds if requested
	if !engine.displaySeconds {
		c.Text = c.Text[0:5]
	}
}

func (engine *Engine) ltcState(c *Clock, s *source) {
	c.Expired = engine.ltcTimeout
	c.Mode = LTC
	ltc := engine.ltc
	if !engine.ltcTimeout {
		// We have LTC time, so display it
		// engine.initialized = true
		c.Text = fmt.Sprintf("%02d:%02d:%02d:%02d", ltc.hours, ltc.minutes, ltc.seconds, ltc.frames)
		c.Hours = ltc.hours
		c.Minutes = ltc.minutes
		c.Seconds = ltc.seconds
		c.Frames = ltc.frames
	} else if engine.ltcFollow {
		// Follow the LTC time when signal is lost
		// Todo: must be easier way to print out the duration...
		t := time.Now()
		diff := t.Sub(engine.ltc.target)
		c.Text = fmt.Sprintf("%s:%02d", formatDuration(diff), 0)
		c.Hours, c.Minutes, c.Seconds = splatDuration(diff)
	} else {
		// Timeout without follow mode
		c.Text = ""
	}
}

func (engine *Engine) timerState(c *Clock, s *source, out *CounterOutput) {
	// Active timer
	if s.timerTarget && out.Target != nil {
		engine.todState(c, s, *out.Target)
		c.Icon = IconTarget
	} else {
		c.Text = out.Text
		c.Hours = out.Hours
		c.Minutes = out.Minutes
		c.Seconds = out.Seconds
		c.Compact = out.Compact
		c.Expired = out.Expired
		c.Paused = out.Paused
		c.Progress = out.Progress
		c.Icon = out.Icon
		c.SignalColor = out.SignalColor
	}

	if out.Mode == "slave" {
		c.Mode = Slave
	} else if out.Mode == "media" {
		c.Mode = Media
	} else if out.Countdown {
		c.Mode = Countdown
		if out.Expired {
			switch engine.overtimeVisibility {
			case "none":
				c.Expired = false
			case "blink":
			case "background":
				c.Expired = false
				c.BGColor = s.overtime
			case "both":
				c.BGColor = s.overtime
			}
		}
	} else {
		c.Mode = Countup
	}
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

// printVersion prints to stdout the clock version and dependency versions
func (engine *Engine) printVersion() {
	clockModule, ok := db.ReadBuildInfo()
	if ok {
		for _, mod := range clockModule.Deps {
			log.Printf("Dep: %s: version %s", mod.Path, mod.Version)
			if mod.Path == "gitlab.com/clock-8001/clock-8001" {
				// gitTag = mod.Version
			}
		}
	} else {
		log.Printf("Error reading BuildInfo, version data unavailable")
	}
	log.Printf("Clock-8001 engine version %s git: %s\n", gitTag, gitCommit)
}

// initCounters initializes the countdown and count up timers
func (engine *Engine) initCounters(options *EngineOptions) (err error) {
	engine.Counters = make([]*Counter, numCounters)
	for i := 0; i < numCounters; i++ {
		engine.Counters[i] = &Counter{
			active:           false,
			state:            &counterState{},
			autoSignals:      options.AutoSignals,
			signalStart:      options.SignalStart,
			warningThreshold: time.Duration(options.SignalThresholdWarning) * time.Second,
			endThreshold:     time.Duration(options.SignalThresholdEnd) * time.Second,
			overtimeMode:     options.OvertimeCountMode,
		}
	}

	for i, s := range []string{options.SignalColorStart, options.SignalColorWarning, options.SignalColorEnd} {
		c := color.RGBA{A: 255}
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
		if err != nil {
			return
		}
		for j := range engine.Counters {
			engine.Counters[j].signalColors[i] = c
		}
	}

	engine.mittiCounter = engine.Counters[options.Mitti]
	engine.milluminCounter = engine.Counters[options.Millumin]
	log.Printf("Media counters - Mitti: %d, Millumin %d", options.Mitti, options.Millumin)

	log.Printf("Initialized %d timer counters", len(engine.Counters))
	return nil
}

func (engine *Engine) initUDPTime(options *EngineOptions) {
	// Stagetimer2 UDP time reception
	if options.UDPTime != "off" {
		engine.udpCounters = make([]*Counter, 2)
		engine.udpCounters[0] = engine.Counters[options.UDPTimer1]
		engine.udpCounters[1] = engine.Counters[options.UDPTimer2]

		if options.UDPTime == "send" {
			log.Printf("Initializing UDP time sender")
			// Send timers
			engine.udpDests = make([]*feedbackDestination, 2)
			engine.udpDests[0] = initFeedback(engine.ctx, "255.255.255.255:36700")
			engine.udpDests[1] = initFeedback(engine.ctx, "255.255.255.255:36701")
		} else {
			log.Printf("Initializing UDP time receiver")
			// Receive timers
			go engine.listenUDPTime()
		}
	}

}

func (engine *Engine) initLimitimer(options *EngineOptions) {
	engine.limitimerReceive[0] = options.LimitimerReceive1
	engine.limitimerReceive[1] = options.LimitimerReceive2
	engine.limitimerReceive[2] = options.LimitimerReceive3
	engine.limitimerReceive[3] = options.LimitimerReceive4
	engine.limitimerReceive[4] = options.LimitimerReceive5

	engine.limitimerBroadcast[0] = options.LimitimerBroadcast1
	engine.limitimerBroadcast[1] = options.LimitimerBroadcast2
	engine.limitimerBroadcast[2] = options.LimitimerBroadcast3
	engine.limitimerBroadcast[3] = options.LimitimerBroadcast4
	engine.limitimerBroadcast[4] = options.LimitimerBroadcast5

	if options.LimitimerMode == "send" {
		go engine.limitimerSend()
	} else if options.LimitimerMode == "receive" {
		go engine.limitimerListen()
	}

}

func (engine *Engine) initSources(sources []*SourceOptions) error {
	// Todo get the map from options...
	engine.sources = make([]*source, len(sources))
	for i, s := range sources {
		// Time zone
		tz, err := time.LoadLocation(s.TimeZone)
		if err != nil {
			return err
		}

		c := color.RGBA{A: 255}
		_, err = fmt.Sscanf(s.OvertimeColor, "#%02x%02x%02x", &c.R, &c.G, &c.B)
		if err != nil {
			return err
		}

		engine.sources[i] = &source{
			counter:     engine.Counters[s.Counter],
			tod:         s.Tod,
			timer:       s.Timer,
			ltc:         s.LTC,
			tz:          tz,
			title:       s.Text,
			hidden:      s.Hidden,
			overtime:    c,
			timerTarget: s.TimerTarget,
		}
	}
	log.Printf("Initialized %d clock display sources", len(engine.sources))
	return nil
}

// initOSC Sets up the OSC listener and feedback
func (engine *Engine) initOSC(options *EngineOptions) {
	if !options.DisableOSC {
		engine.oscDispatcher = oscutil.NewRegexpDispatcher()
		engine.oscServer = osc.Server{
			Addr:    options.ListenAddr,
			Handler: engine.oscDispatcher.Dispatch,
		}
		engine.clockServer = MakeServer(&engine.oscServer, engine.oscDispatcher, engine.uuid)
		log.Printf("OSC control: listening on %v", engine.oscServer.Addr)

		go engine.runOSC()

		// process osc commands
		go engine.listen()

		if options.DisableFeedback {
			engine.oscDests = nil
			log.Printf("OSC feedback disabled")
		} else {
			// OSC feedback
			engine.oscDests = initFeedback(engine.ctx, options.Connect)
		}
	} else {
		log.Printf("OSC control and feedback disabled.\n")
	}
	engine.oscSendChan = make(chan []byte)
	go engine.oscSender()
}

func (engine *Engine) oscSender() {
	for data := range engine.oscSendChan {
		if engine.oscDests != nil {
			engine.oscDests.Write(data)
		}
	}
}

func (engine *Engine) activateSourceByCounter(c int) {
	for _, s := range engine.sources {
		if s.counter == engine.Counters[c] {
			s.hidden = false
		}
	}
}

// SetSourceColors sets the source output colors
func (engine *Engine) SetSourceColors(source int, text, bg color.RGBA) {
	engine.sources[source].textColor = text
	engine.sources[source].bgColor = bg
}

// SetTitleColors sets the source title colors
func (engine *Engine) SetTitleColors(text, bg color.RGBA) {
	engine.titleTextColor = text
	engine.titleBGColor = bg
}

func clockAddresses() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("Failed to get interface addresses")
		return ""
	}
	var ret string
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		if ip.IsLoopback() {
			continue
		} else if ip.To4() != nil {
			ret += fmt.Sprintf("    %v\n", ip)
		}
	}
	return ret
}

func formatDuration(diff time.Duration) string {
	hours, minutes, seconds := splatDuration(diff)
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func splatDuration(diff time.Duration) (hours, minutes, seconds int) {
	hours = int(diff.Truncate(time.Hour).Hours())
	minutes = int(diff.Truncate(time.Minute).Minutes()) - (hours * 60)
	seconds = int(diff.Truncate(time.Second).Seconds()) - (((hours * 60) + minutes) * 60)
	return
}

// parseColor parses a string "#XXX or #XXXXXX to a color.RGBA"
func parseColor(s string) (c *color.RGBA, err error) {
	c = &color.RGBA{
		A: 255,
	}
	switch len(s) {
	case 7:
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	case 4:
		_, err = fmt.Sscanf(s, "#%1x%1x%1x", &c.R, &c.G, &c.B)
		// Double the hex digits:
		c.R *= 17
		c.G *= 17
		c.B *= 17
	default:
		err = fmt.Errorf("parseColor(): invalid length, must be 7 or 4: %v", s)
	}
	return
}
