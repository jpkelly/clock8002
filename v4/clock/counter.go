package clock

import (
	"fmt"
	"gitlab.com/Depili/limitimer"
	"image/color"
	"strings"
	"time"
)

/*
 * Counters representing generic countdowns / ups or external sources
 */

const (
	autoColorOff   = iota
	autoColorStart = iota
	autoColorWarn  = iota
	autoColorEnd   = iota
)

const (
	priorityNormal    = 0
	priorityLimitimer = iota
	priorityMedia     = iota
	prioritySlave     = iota
)

// Counter abstracts a generic counter counting up or down
type Counter struct {
	state            *counterState
	externalState    externalState
	active           bool // Is this counter active?
	countdown        bool // Count up / down from the target
	paused           bool // Is the counter paused?
	signalColor      color.RGBA
	autoColorState   int
	endThreshold     time.Duration // Seconds for the "end" warning color
	warningThreshold time.Duration // Seconds for the warning color
	signalColors     [3]color.RGBA
	signalStart      bool
	autoSignals      bool
	overtimeMode     string
	mediaColor       *color.RGBA
	mediaBG          *color.RGBA
	number           int
	mute             bool
}

// CounterOutput the data structure returned by Counter.Output() and contains the static state of the counter at that time
type CounterOutput struct {
	Active      bool          // True if the counter is active
	Media       bool          // True if counter represents a playing media file
	Countdown   bool          // True if counting down, false if counting up
	Paused      bool          // True if counter has been paused
	Looping     bool          // True if the playing media is looping in the player
	Expired     bool          // Has the countdown timer expired?
	Hours       int           // Hour part of the timer
	Minutes     int           // Minutes of the timer, 0-60
	Seconds     int           // Seconds of the timer, 0-60
	Text        string        // HH:MM:SS string representation
	Icon        string        // Single unicode glyph to use as an icon for the timer
	Compact     string        // Compact 4-character output
	Progress    float64       // Percentage of total time elapsed of the countdown, 0-1
	Diff        time.Duration // raw difference
	Duration    time.Duration // Total duration of the count, if available
	SignalColor color.RGBA
	Mode        string
	Target      *time.Time // End time for counter, if available
}

type externalState interface {
	output() *CounterOutput
	priority() int
}

type slaveState struct {
	hours     int
	minutes   int
	seconds   int
	icon      string
	hideHours bool
}

type mediaState struct {
	paused    bool
	looping   bool
	hours     int32
	minutes   int32
	seconds   int32
	frames    int32
	progress  float64
	remaining time.Duration
	target    time.Time
	icon      string
}

// MediaUpdate is the interface for updating external media player state
type MediaUpdate interface {
	Remaining() time.Duration
	Duration() time.Duration
	Play() bool
	Loop() bool
	Record() bool
	MediaName() string
	IconOverride() string
}

type counterState struct {
	target   time.Time     // Target timestamp for main countdown
	duration time.Duration // Total duration of main countdown, used to scale the leds
	left     time.Duration // Duration left when paused
}

type limitimerState struct {
	hours      int
	minutes    int
	seconds    int
	duration   time.Duration
	elapsed    time.Duration
	icon       string
	blink      bool
	led        int
	paused     bool
	expired    bool
	warning    bool
	minuteMode bool
	countdown  bool
	target     time.Time
}

func (slave *slaveState) output() *CounterOutput {
	hours := slave.hours
	minutes := slave.minutes
	seconds := slave.seconds

	text := fmt.Sprintf("%02d:%02d:%02d", hours, abs(minutes), abs(seconds))
	if slave.hideHours {
		text = text[3:8]
	}

	out := &CounterOutput{
		Active:    true,
		Countdown: true,
		Paused:    false,
		Expired:   false,
		Hours:     hours,
		Minutes:   minutes,
		Seconds:   seconds,
		Text:      text,
		Compact:   "",
		Icon:      slave.icon,
		Progress:  0,
		Diff:      0,
		Duration:  0,
		Mode:      "slave",
	}

	return out
}

func (slave *slaveState) priority() int {
	return prioritySlave
}

func (m *mediaState) output() *CounterOutput {
	var icon string
	var seconds int64

	if m.icon != "" {
		icon = m.icon
	} else if m.paused {
		icon = IconPaused
	} else if m.looping {
		icon = IconLooping
	} else {
		icon = IconPlaying
	}

	seconds = int64(m.hours) * 60
	seconds = (int64(m.minutes) + seconds) * 60
	seconds = seconds + int64(m.seconds)

	text := fmt.Sprintf("%02d:%02d:%02d", m.hours, m.minutes, m.seconds)
	compact := fmt.Sprintf("%s%s", icon, secsToCompact(seconds))

	dur := time.Duration(seconds)*time.Second + m.remaining

	out := &CounterOutput{
		Active:   true,
		Media:    true,
		Expired:  false,
		Icon:     icon,
		Paused:   m.paused,
		Looping:  m.looping,
		Hours:    int(m.hours),
		Minutes:  int(m.minutes),
		Seconds:  int(m.seconds),
		Text:     text,
		Compact:  compact,
		Progress: m.progress,
		Diff:     m.remaining,
		Duration: dur,
		Mode:     "media",
		Target:   &m.target,
	}

	return out
}

func (m *mediaState) priority() int {
	return priorityMedia
}

func (lt *limitimerState) output() *CounterOutput {
	out := CounterOutput{
		Active:    true,
		Countdown: lt.countdown,
		Paused:    lt.paused,
		Expired:   lt.expired,
		Hours:     lt.hours,
		Minutes:   lt.minutes,
		Seconds:   lt.seconds,
		Icon:      lt.icon,
		Diff:      lt.duration - lt.elapsed,
		Progress:  1 - lt.elapsed.Seconds()/lt.duration.Seconds(),
		Duration:  lt.duration,
		Mode:      "limitimer",
	}

	if lt.countdown {
		out.Target = &lt.target
	}

	if out.Expired && out.Countdown {
		out.Progress = 1
	}

	if lt.minuteMode {
		out.Text = fmt.Sprintf("%0.2d:%0.2d", out.Minutes, out.Seconds)
	} else {
		out.Text = fmt.Sprintf("%0.2d:%0.2d", out.Hours, out.Minutes)
	}

	secs := int64(out.Hours)*360 + int64(out.Minutes)*60 + int64(out.Seconds)
	out.Compact = fmt.Sprintf("%s%s", lt.icon, secsToCompact(secs))

	return &out
}

func (lt *limitimerState) priority() int {
	return priorityLimitimer
}

func (counter *Counter) parseLimitimer(p *LimitimerMessage) {
	expired := p.Total-p.Elapsed < 1
	warning := p.Total-p.Elapsed <= p.Sumup

	t := int32(0)
	var hour, min, sec int32

	if p.Countdown {
		// Count down
		t = p.Total - p.Elapsed
	} else {
		t = p.Elapsed
	}

	if p.Minutes {
		hour = 0
		min = t / 60
		sec = t % 60
	} else {
		hour = t / 60 / 60
		min = t / 60
		sec = t % 60
	}

	if min > 99 {
		min = 99
		sec = 59
	}

	if expired {
		if hour < 0 {
			hour = -hour
		}
		if min < 0 {
			min = -min
		}
		if sec < 0 {
			sec = -sec
		}
	}

	ltState := limitimerState{
		hours:      int(hour),
		minutes:    int(min),
		seconds:    int(sec),
		duration:   time.Duration(p.Total) * time.Second,
		elapsed:    time.Duration(p.Elapsed) * time.Second,
		blink:      p.Blink,
		paused:     !p.Run,
		expired:    expired,
		warning:    warning,
		minuteMode: p.Minutes,
		countdown:  p.Countdown,
	}

	counter.setLimitimerState(&ltState)
}

// SetLimitimer set the counter state from a limitimer status message
func (counter *Counter) SetLimitimer(p *limitimer.State, i int) {
	h, m, s := p.TimerDisplay(i)

	ltState := limitimerState{
		hours:      h,
		minutes:    m,
		seconds:    s,
		duration:   time.Duration(p.Timers[i].Total.Seconds()) * time.Second,
		elapsed:    time.Duration(p.Timers[i].Elapsed.Seconds()) * time.Second,
		blink:      p.Timers[i].Blink(),
		paused:     !p.Timers[i].Run(),
		expired:    p.Timers[i].Expired(),
		warning:    p.Timers[i].Warning(),
		minuteMode: p.Minutes(i),
		countdown:  p.CountDirection,
	}

	if ltState.countdown {
		target := time.Now().Add(ltState.duration - ltState.elapsed)
		ltState.target = target
	}

	counter.setLimitimerState(&ltState)
}

func (counter *Counter) setLimitimerState(ltState *limitimerState) {
	icon := "Ⅱ"
	if !ltState.paused && ltState.countdown {
		icon = "↓"
	} else if !ltState.paused {
		icon = "↑"
	}
	ltState.icon = icon

	led := autoColorOff

	if !ltState.paused {
		if ltState.expired {
			led = autoColorEnd
		} else if ltState.warning {
			led = autoColorWarn
		} else {
			led = autoColorStart
		}
	}
	ltState.led = led

	// Uglyish hack for the hours mode
	if !ltState.minuteMode && ltState.hours == 0 && ltState.minutes == 0 {
		ltState.expired = true
	}

	switch ltState.led {
	case autoColorOff:
		counter.signalColor = color.RGBA{R: 0, G: 0, B: 0, A: 0}
	case autoColorStart:
		counter.signalColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	case autoColorWarn:
		counter.signalColor = color.RGBA{R: 255, G: 200, B: 0, A: 255}
	case autoColorEnd:
		if !ltState.blink || int(ltState.elapsed.Seconds())%2 == 0 {
			counter.signalColor = color.RGBA{R: 255, G: 0, B: 0, A: 255}
		} else {
			counter.signalColor = color.RGBA{R: 0, G: 0, B: 0, A: 0}
		}
	}

	counter.setExternal(ltState)
}

func (counter *Counter) setExternal(ext externalState) {
	if counter.externalState != nil {
		if counter.externalState.priority() <= ext.priority() {
			counter.externalState = ext
		}
	} else {
		counter.externalState = ext
	}
}

// Output generates the static output of the counter for use in clock displays
func (counter *Counter) Output(t time.Time) *CounterOutput {
	var out *CounterOutput

	if counter.externalState != nil {
		out = counter.externalState.output()
	} else {
		out = counter.normalOutput(t)
	}

	// Signal color
	out.SignalColor = counter.signalColor

	if counter.autoSignals && out.Active && out.Countdown {
		if out.Countdown {
			if out.Diff < counter.endThreshold {
				counter.setAutoColor(counter.signalColors[colorEnd], autoColorEnd)
			} else if out.Diff < counter.warningThreshold {
				counter.setAutoColor(counter.signalColors[colorWarning], autoColorWarn)
			} else if counter.signalStart {
				counter.setAutoColor(counter.signalColors[colorStart], autoColorStart)
			} else {
				counter.setAutoColor(color.RGBA{R: 0, G: 0, B: 0, A: 0}, autoColorOff)
			}
		} else if counter.signalStart {
			counter.setAutoColor(counter.signalColors[colorStart], autoColorStart)
		} else {
			counter.setAutoColor(color.RGBA{R: 0, G: 0, B: 0, A: 0}, autoColorOff)
		}
		out.SignalColor = counter.signalColor
	}

	if out.Expired {
		// FIXME make "continue" the default on counters, and just zero here if needed
		// FIXME same logic needs to apply to counters, move logic deeper?
		switch counter.overtimeMode {
		case "zero":
			out.zero()
		case "blank":
			out.blank()
		case "continue":
			out.Icon = IconOvertime
		}

	}

	return out
}

func (out *CounterOutput) blank() {
	out.Icon = ""
	out.Text = ""
	out.Compact = ""
	out.Hours = 0
	out.Minutes = 0
	out.Seconds = 0
}

func (out *CounterOutput) zero() {
	switch strings.Count(out.Text, ":") {
	case 0:
		out.Text = "00"
	case 1:
		out.Text = "00:00"
	case 2:
		out.Text = "00:00:00"
	case 3:
		out.Text = "00:00:00:00"
	}
	out.Compact = fmt.Sprintf("%s%s", out.Icon, secsToCompact(0))
	out.Hours = 0
	out.Minutes = 0
	out.Seconds = 0
}

func (counter *Counter) normalOutput(t time.Time) *CounterOutput {
	if !counter.active {
		return &CounterOutput{Active: false}
	}

	var icon string
	diff := counter.Diff(t)
	expired := diff.Seconds() < 1

	// Fudge dual-00:00:00 due to rounding direction change
	if expired {
		diff -= time.Second
	}

	hours := int(diff.Truncate(time.Hour).Hours())
	minutes := int(diff.Truncate(time.Minute).Minutes()) - (hours * 60)
	seconds := int(diff.Truncate(time.Second).Seconds()) - (((hours * 60) + minutes) * 60)

	progress := (float64(diff) / float64(counter.state.duration))

	if expired {
		hours = -hours
		minutes = -minutes
		seconds = -seconds
		progress = 1
	}

	if progress >= 1 {
		progress = 1
	} else if progress < 0 {
		progress = 0
	}

	if counter.paused {
		icon = IconPaused
	} else if counter.countdown {
		icon = IconCountdown
	} else {
		icon = IconCountup
	}

	rawSecs := int64((counter.Diff(t).Truncate(time.Second) + time.Second).Seconds())
	c := secsToCompact(rawSecs)

	target := counter.state.target

	out := &CounterOutput{
		Active:    counter.active,
		Countdown: counter.countdown,
		Paused:    counter.paused,
		Expired:   expired,
		Hours:     hours,
		Minutes:   minutes,
		Seconds:   seconds,
		Text:      fmt.Sprintf("%02d:%02d:%02d", abs(hours), abs(minutes), abs(seconds)),
		Compact:   fmt.Sprintf("%s%s", icon, c),
		Icon:      icon,
		Progress:  progress,
		Diff:      diff,
		Duration:  counter.state.duration,
		Target:    &target,
	}

	return out
}

func secsToCompact(rawSecs int64) string {
	for _, unit := range clockUnits {
		if rawSecs/int64(unit.seconds) >= 100 {
			continue
		}
		count := rawSecs / int64(unit.seconds)
		return fmt.Sprintf("%02d%1s", count, unit.unit)
	}
	return "+++"
}

// Start begins counting time up or down
func (counter *Counter) Start(countdown bool, timer time.Duration) {
	s := counterState{
		target:   time.Now().Add(timer).Truncate(time.Second),
		duration: timer,
		left:     timer,
	}
	counter.state = &s

	counter.countdown = countdown

	t := time.Now()

	if counter.countdown {
		counter.state.left = counter.state.target.Sub(t).Truncate(time.Second)
	} else {
		counter.state.left = t.Sub(counter.state.target).Truncate(time.Second)
	}

	counter.active = true
}

// Target sets the target date and time for a counter
func (counter *Counter) Target(target time.Time) {
	timer := time.Until(target)
	fmt.Printf("target: %v timer: %v\n", target, timer)
	if timer < 0 {
		counter.Start(false, timer)
	} else {
		counter.Start(true, timer)
	}
}

// SetSlave sets the counter state as a slave from external source
func (counter *Counter) SetSlave(hours, minutes, seconds int, hideHours bool, icon string) {
	s := &slaveState{
		hours:     hours,
		minutes:   minutes,
		seconds:   seconds,
		hideHours: hideHours,
		icon:      icon,
	}
	counter.setExternal(s)
}

// ResetSlave removes slave state from external source
func (counter *Counter) ResetSlave() {
	if _, ok := counter.externalState.(*slaveState); ok {
		counter.externalState = nil
		counter.active = false
	}
}

// SetMedia sets the counter state from a playing media file
func (counter *Counter) SetMedia(hours, minutes, seconds, frames int32, remaining time.Duration, progress float64, paused bool, looping bool) {
	// FIXME: .truncate(time.Second) and mitti timers cause blinking on second changes!
	m := mediaState{
		hours:     hours,
		minutes:   minutes,
		seconds:   seconds,
		frames:    frames,
		paused:    paused,
		looping:   looping,
		progress:  progress,
		remaining: remaining,
		target:    time.Now().Add(remaining),
	}
	counter.setExternal(&m)
	counter.active = true
}

func (counter *Counter) mediaUpdate(m MediaUpdate) {
	hours, minutes, seconds := splitTime(m)
	progress := progress(m)
	frames := int32(0)

	ms := mediaState{
		hours:     hours,
		minutes:   minutes,
		seconds:   seconds,
		frames:    frames,
		paused:    !m.Play(),
		looping:   m.Loop(),
		progress:  progress,
		remaining: m.Remaining(),
		target:    time.Now().Add(m.Remaining()),
	}

	if m.IconOverride() != "" {
		ms.icon = m.IconOverride()
	}
	counter.setExternal(&ms)
	counter.active = true
}

// ResetMedia removes the media state from a counter
func (counter *Counter) ResetMedia() {
	if _, ok := counter.externalState.(*mediaState); ok {
		counter.active = false
		counter.externalState = nil
	}
}

// Modify alters the counter target on a running counter
func (counter *Counter) Modify(delta time.Duration) {
	if !counter.active {
		return
	}
	if !counter.countdown {
		// Invert delta if counting up
		delta = -delta
	}

	s := counterState{
		target:   counter.state.target.Add(delta),
		duration: counter.state.duration + delta,
		left:     counter.state.left + delta,
	}
	counter.state = &s
}

// Stop stops and deactivates the counter
func (counter *Counter) Stop() {
	counter.active = false
	counter.paused = false

	s := counterState{
		target:   time.Now(),
		duration: time.Millisecond,
		left:     time.Millisecond,
	}
	counter.state = &s
}

// Pause pauses a running counter
func (counter *Counter) Pause() {
	// TODO: atomic replace
	if counter.paused {
		return
	}
	t := time.Now()
	if counter.countdown {
		counter.state.left = counter.state.target.Sub(t).Truncate(time.Second)
	} else {
		counter.state.left = t.Sub(counter.state.target).Truncate(time.Second)
	}
	counter.paused = true
}

// Resume resumes a paused counter
func (counter *Counter) Resume() {
	// TODO: atomic replace
	if !counter.paused {
		return
	}
	t := time.Now()
	if counter.countdown {
		counter.state.target = t.Add(counter.state.left).Truncate(time.Second)
	} else {
		counter.state.target = t.Add(-counter.state.left).Truncate(time.Second)
	}
	counter.paused = false
}

// Diff gives a time difference to current time that can be used to format clock output strings
func (counter *Counter) Diff(t time.Time) time.Duration {
	if counter.paused {
		return counter.state.left
	}
	if counter.countdown {
		return counter.state.target.Sub(t)
	}
	return t.Sub(counter.state.target)
}

func (counter *Counter) setAutoColor(c color.RGBA, state int) {
	if counter.autoColorState != state {
		counter.autoColorState = state
		counter.signalColor = c
	}
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// splitTime splits the time in a mediaUpdate to components
func splitTime(m MediaUpdate) (hours, minutes, seconds int32) {
	diff := m.Remaining()
	seconds = int32(diff.Seconds()) % 60
	minutes = int32(diff.Minutes()) % 60
	hours = int32(diff.Hours())
	return
}

// progress calculates the playhead progress as 0 - 1.0 float
func progress(m MediaUpdate) float64 {
	return 1 - (m.Remaining().Seconds() / m.Duration().Seconds())
}
