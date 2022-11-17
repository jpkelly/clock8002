package clock

import (
	"context"
	"fmt"
	"github.com/chabad360/go-osc/osc"
	"github.com/denisbrodbeck/machineid"
	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/oscutil"
	"gitlab.com/clock-8001/clock-8001/v4/udptime"
	"image/color"
	"log"
	"regexp"
	db "runtime/debug"
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

	if len(options.TimerOptions) != 10 {
		options.TimerOptions = make([]*TimerOptions, 10)
		options.TimerOptions[0] = options.Timer0
		options.TimerOptions[1] = options.Timer1
		options.TimerOptions[2] = options.Timer2
		options.TimerOptions[3] = options.Timer3
		options.TimerOptions[4] = options.Timer4
		options.TimerOptions[5] = options.Timer5
		options.TimerOptions[6] = options.Timer6
		options.TimerOptions[7] = options.Timer7
		options.TimerOptions[8] = options.Timer8
		options.TimerOptions[9] = options.Timer9
	}

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

	for i, o := range options.TimerOptions {
		if o.ListenPort != 0 {
			addr := fmt.Sprintf("0.0.0.0:%d", o.ListenPort)

			server := oscListenerSetup(addr)
			go server.run(engine.ctx, engine.wg)

			engine.mittiListen(server, i, o.MittiBroadcast)
			engine.milluminListen(server, i, o.MilluminBroadcast)
		}
	}

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
	// OSC feedback channel and processor
	// We start this every time to consume the data even with
	// OSC feedback disabled
	engine.oscSendChan = make(chan []byte)
	go engine.oscSender()

	if options.DisableOSC {
		log.Printf("OSC control and feedback disabled.\n")
		return
	}

	engine.oscDispatcher = oscutil.NewRegexpDispatcher()
	engine.oscServer = osc.Server{
		Addr:    options.ListenAddr,
		Handler: engine.oscDispatcher.Dispatch,
	}
	engine.clockServer = MakeServer(&engine.oscServer, engine.oscDispatcher, engine.uuid)
	log.Printf("OSC control: listening on %v", engine.oscServer.Addr)

	go engine.clockServer.run(engine.ctx, engine.wg)

	err := engine.oscBridge(engine.clockServer.dispatcher)
	if err != nil {
		panic(err)
	}

	// process osc commands
	go engine.listen()

	if options.DisableFeedback {
		log.Printf("OSC feedback disabled")
		return
	}

	// OSC feedback destinations
	engine.oscDests = initFeedback(engine.ctx, options.Connect)
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
