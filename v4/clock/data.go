package clock

import (
	"context"

	"github.com/desertbit/timer"
	"gitlab.com/clock-8001/clock-8001/v4/oscutil"

	// "gitlab.com/Depili/go-osc/osc"
	"github.com/chabad360/go-osc/osc"

	"image/color"
	"regexp"
	"sync"
	"time"
)

// Version is the current clock engine version
const Version = "1.1.8"

// State feedback timer
const stateTimer = time.Second / 2
const udpTimer = time.Second / 10
const flashDuration = 200 * time.Millisecond

// Will get overridden by ldflags in Makefile
var gitCommit = "Unknown"
var gitTag = "v0.0.1"

// Icons
const (
	// Icon for overtime timers
	IconOvertime = "+"
	// Icon for paused media readouts
	IconPaused = "Ⅱ"
	// Icon for looping media readouts
	IconLooping = "⇄"
	// Icon for playing media readouts
	IconPlaying = "▶"
	// Icon for countdown timers
	IconCountdown = "↓"
	// Icon for upwards counting timers
	IconCountup = "↑"
	// Icon for countdown target times
	IconTarget = "⇒"
	// Icon for record
	IconRecord = "⏺"
	// Icon for PlayPause, used when playback ends after one cue
	IconPlayPause = "⏯"
	// Icon for negative external times
	IconNegative = "-"
)

const (
	colorStart   = 0
	colorWarning = 1
	colorEnd     = 2
)

const mediaTimeout = 1000 * time.Millisecond
const mediaTickRate = 50 * time.Millisecond

// SourceOptions contains all options for clock display sources.
type SourceOptions struct {
	Text          string `long:"text" description:"Title text for the time source"`
	Counter       string `long:"counter" description:"Counter numbers to associate with this source, leave empty to disable it as a suorce" default:"0"`
	TimerTarget   bool   `long:"timer-target" description:"Show end time of the timer instead of time remaining"`
	LTC           bool   `long:"ltc" description:"Enable LTC as a source"`
	LTCChannel    int    `long:"ltc-channel" description:"LTC channel" default:"0" choice:"0" choice:"1"`
	Timer         bool   `long:"timer" description:"Enable timer counter as a source"`
	Tod           bool   `long:"tod" description:"Enable time-of-day as a source"`
	TimeZone      string `long:"timezone" description:"Time zone to use for ToD display" default:"Europe/Helsinki"`
	Hidden        bool   `long:"hidden" description:"Hide this time source"`
	OvertimeColor string `long:"overtime-color" description:"Background color for overtime countdowns, in HTML format #FFFFFF" default:"#FF0000"`
}

// TimerOptions contains all per-timer options
type TimerOptions struct {
	MediaColor string `long:"media-color" description:"CSS color for media name" default:"#FF8000"`
	MediaBG    string `long:"media-bg" description:"CSS color for media name background" default:"#101010"`

	ListenPort        int  `long:"port" description:"Port to listen to for mitti and millumin osc messages"`
	MittiBroadcast    bool `long:"mitti-broadcast" description:"Broadcast received mitti state to other clocks"`
	MilluminBroadcast bool `long:"millumin-broadcast" description:"Broadcast received millumin state to other clocks"`

	VmixEnabled        bool   `long:"vmix-enabled" description:"Enable vMix integration"`
	VmixAddress        string `long:"vmix-address" description:"vMix IP address"`
	VmixPort           int    `long:"vmix-port" description:"vMmix http port" default:"8088"`
	VmixLoops          bool   `long:"vmix-loops" description:"vMix show looping media"`
	VmixPGMOnly        bool   `long:"vmix-pgm-only" description:"Only show PGM media information, ignore all overlays"`
	VmixPVM            bool   `long:"vmix-show-pvm" description:"Do not ingore media playing in vMix preview"`
	VmixIgnoreOverlays string `long:"vmix-ignore-overlays" description:"vMix: Comma separated list of overlay numbers to ignore"`
	VmixMediaName      bool   `long:"vmix-media-name" description:"Show vMix media name as clock text"`

	PicturallEnabled      bool   `long:"picturall-enabled" descriptin:"Enable picturall media info support"`
	PicturallAddress      string `long:"picturall-address" description:"Address for picturall media server to connect to"`
	PicturallPort         int    `long:"picturall-port" description:"Picturall telnet port" default:"11000"`
	PicturallTimer        int    `long:"picturall-timer" description:"Timer for picturall video state" default:"7"`
	PicturallLoops        bool   `long:"picturall-loops" description:"Show looping content"`
	PicturallMediaName    bool   `long:"picturall-media-name" description:"Show picturall media name as clock text"`
	PicturallIgnoreLayers string `long:"picturall-ignore-layers" description:"Comma separated list of layers to ignore"`
	PicturallStreams      bool   `long:"picturall-streams" description:"Show streaming content"`

	DisguiseTimerOptions
	OSC OSCOptions

	StopAtEnd     bool `long:"stop-at-end" description:"Stop the timer when it expires"`
	EndRestart    bool `long:"end-restart" description:"Restart the timer specified by restart-target when this countdown reaches its end"`
	EndOSC        bool `long:"end-osc" description:"Send a OSC message when this countdown reaches its end"`
	RestartTarget int  `long:"restart-target" description:"Timer target for the end action"`
	Duration      int  `long:"duration" description:"Default duration, in seconds" default:"600"`
	Countup       bool `long:"countup" description:"Default to countup instead of countdown"`
	Mute          bool `long:"mute" description:"Ignore this timer for audio cues"`
}

// OSCOptions contains the options for a osc message target
type OSCOptions struct {
	IP      string `long:"osc-ip" description:"IP address to send a OSC message to"`
	Port    string `long:"osc-port" description:"Port for the OSC message"`
	Command string `long:"osc-command" description:"OSC command to send"`
	Params  string `long:"osc-params" description:"Parameter list, i for integer, b for bool, s for string, f for float (eg. s test i 123 f 1.23 b true)"`
}

// EngineOptions contains all common options for clock.Engines
type EngineOptions struct {
	Flash           int    `long:"flash" description:"Flashing interval when countdown reached zero (ms), 0 disables" default:"500"`
	ListenAddr      string `long:"osc-listen" description:"Address to listen for incoming osc messages" default:"0.0.0.0:1245"`
	Timeout         int    `short:"d" long:"timeout" description:"Timeout for OSC message updates in milliseconds" default:"1000"`
	Connect         string `short:"o" long:"osc-dest" description:"Address to send OSC feedback to" default:"255.255.255.255:1245"`
	DisableOSC      bool   `long:"disable-osc" description:"Disable OSC control and feedback"`
	DisableFeedback bool   `long:"disable-feedback" description:"Disable OSC feedback"`
	DisableLTC      bool   `long:"disable-ltc" description:"Disable LTC display mode"`
	LTCSeconds      bool   `long:"ltc-seconds" description:"Show seconds on the ring in LTC mode"`
	//lint:ignore SA5008 go-flags uses multiple choice
	UDPTime   string `long:"udp-time" description:"Stagetimer2 UDP protocol support" choice:"off" choice:"send" choice:"receive" default:"receive"`
	UDPTimer1 int    `long:"udp-timer-1" description:"Timer to send as UDP timer 1 (port 36700)" default:"1"`
	UDPTimer2 int    `long:"udp-timer-2" description:"Timer to send as UDP timer 2 (port 36701)" default:"2"`
	LTCFollow bool   `long:"ltc-follow" description:"Continue on internal clock if LTC signal is lost. If unset display will blank when signal is gone."`
	Format12h bool   `long:"format-12h" description:"Use 12 hour format for time-of-day display"`
	Mitti     int    `long:"mitti" description:"Counter number for Mitti OSC feedback" default:"8"`
	Millumin  int    `long:"millumin" description:"Counter number for Millumin OSC feedback" default:"9"`
	Ignore    string `long:"millumin-ignore-layer" value-name:"REGEXP" description:"Ignore matching millumin layers (case-insensitive regexp)" default:"ignore"`
	ShowInfo  int    `long:"info-timer" description:"Show clock status for x seconds on startup" default:"30"`
	//lint:ignore SA5008 go-flags uses multiple choice
	OvertimeCountMode string `long:"overtime-count-mode" description:"Behaviour for expired countdown timer counts" default:"zero" choice:"zero" choice:"blank" choice:"continue"`
	//lint:ignore SA5008 go-flags uses multiple choice
	OvertimeVisibility string `long:"overtime-visibility" description:"Extra visibility for overtime timers" default:"blink" choice:"blink" choice:"background" choice:"both" choice:"none"`
	ToDHideSeconds     bool   `long:"tod-hide-seconds" description:"Hide seconds on time-of-day displays"`
	HighResolution     bool   `long:"high-resolution" description:"1/100th second resolution timers"`

	AutoSignals            bool   `long:"auto-signals" description:"Automatic signal colors based on timer state"`
	SignalStart            bool   `long:"signal-start" description:"Set signal color on timer start"`
	SignalColorStart       string `long:"signal-color-start" description:"Signal colors for timers above thresholds" default:"#00FF00"`
	SignalColorWarning     string `long:"signal-color-warning" description:"Signal colors for timers between thresholds" default:"#FFFF00"`
	SignalColorEnd         string `long:"signal-color-end" description:"Signal colors for timers bellow thresholds" default:"#FF0000"`
	SignalThresholdWarning int    `long:"signal-threshold-warning" description:"Threshold for medium color transition (seconds)" default:"180"`
	SignalThresholdEnd     int    `long:"signal-threshold-end" description:"Threshold for medium color transition (seconds)" default:"60"`
	SignalHardware         int    `long:"signal-hw-group" description:"Hardware signal group number" default:"1"`

	//lint:ignore SA5008 go-flags uses multiple choice
	LimitimerMode       string `long:"limitimer-mode" description:"Listen for limitimer messages on the serial device and update sources based on them" choice:"off" choice:"send" choice:"receive" default:"off"`
	LimitimerSerial     string `long:"limitimer-serial" description:"Serial device for limitimer communication" default:"/dev/ttyAMA0"`
	LimitimerReceive1   bool   `long:"limitimer-receive-timer1" description:"Receive limitimer program 1 as timer 1"`
	LimitimerReceive2   bool   `long:"limitimer-receive-timer2" description:"Receive limitimer program 2 as timer 2"`
	LimitimerReceive3   bool   `long:"limitimer-receive-timer3" description:"Receive limitimer program 3 as timer 3"`
	LimitimerReceive4   bool   `long:"limitimer-receive-timer4" description:"Receive limitimer program 4 as timer 4"`
	LimitimerReceive5   bool   `long:"limitimer-receive-timer5" description:"Receive limitimer active program as timer 5"`
	LimitimerBroadcast1 bool   `long:"limitimer-broadcast-timer1" description:"Broadcast limitimer program 1 to other clocks as timer 1"`
	LimitimerBroadcast2 bool   `long:"limitimer-broadcast-timer2" description:"Broadcast limitimer program 2 to other clocks as timer 2"`
	LimitimerBroadcast3 bool   `long:"limitimer-broadcast-timer3" description:"Broadcast limitimer program 3 to other clocks as timer 3"`
	LimitimerBroadcast4 bool   `long:"limitimer-broadcast-timer4" description:"Broadcast limitimer program 4 to other clocks as timer 4"`
	LimitimerBroadcast5 bool   `long:"limitimer-broadcast-timer5" description:"Broadcast limitimer active program to other clocks as timer 5"`

	PicturallTimeout int  `long:"picturall-timeout" description:"Timeout for communication, the display will be blanked after this time, in milliseconds" default:"1000"`
	PicturallDebug   bool `long:"picturall-debug" description:"Write picturall traffic to a logfile"`

	VmixInterval int `long:"vmix-interval" description:"vMix polling interval, in milliseconds" default:"200"`
	VmixTimeout  int `long:"vmix-timeout" description:"Timeout for vMix communication, in milliseconds" default:"1000"`

	TricasterOptions
	HyperdeckOptions
	DisguiseOptions

	Source1 *SourceOptions `group:"1st clock display source" namespace:"source1"`
	Source2 *SourceOptions `group:"2nd clock display source" namespace:"source2"`
	Source3 *SourceOptions `group:"3rd clock display source" namespace:"source3"`
	Source4 *SourceOptions `group:"4th clock display source" namespace:"source4"`

	SourceOptions []*SourceOptions

	Timer0 *TimerOptions `group:"Timer 0" namespace:"timer0"`
	Timer1 *TimerOptions `group:"Timer 1" namespace:"timer1"`
	Timer2 *TimerOptions `group:"Timer 2" namespace:"timer2"`
	Timer3 *TimerOptions `group:"Timer 3" namespace:"timer3"`
	Timer4 *TimerOptions `group:"Timer 4" namespace:"timer4"`
	Timer5 *TimerOptions `group:"Timer 5" namespace:"timer5"`
	Timer6 *TimerOptions `group:"Timer 6" namespace:"timer6"`
	Timer7 *TimerOptions `group:"Timer 7" namespace:"timer7"`
	Timer8 *TimerOptions `group:"Timer 8" namespace:"timer8"`
	Timer9 *TimerOptions `group:"Timer 9" namespace:"timer9"`

	TimerOptions []*TimerOptions

	cueOptions

	// Deprecated options, to be removed on next major release
	VmixEnabled           bool   `long:"vmix-enabled" description:"Enable vMix integration" hidden:"true"`
	VmixAddress           string `long:"vmix-address" description:"vMix IP address" hidden:"true"`
	VmixPort              int    `long:"vmix-port" description:"vMmix http port" default:"8088" hidden:"true"`
	VmixTimer             int    `long:"vmix-timer" description:"Timer for vMix video state" default:"6" hidden:"true"`
	VmixLoops             bool   `long:"vmix-loops" description:"vMix show looping media" hidden:"true"`
	VmixPGMOnly           bool   `long:"vmix-pgm-only" description:"Only show PGM media information, ignore all overlays" hidden:"true"`
	VmixPVM               bool   `long:"vmix-show-pvm" description:"Do not ingore media playing in vMix preview" hidden:"true"`
	VmixIgnoreOverlays    string `long:"vmix-ignore-overlays" description:"vMix: Comma separated list of overlay numbers to ignore" hidden:"true"`
	VmixMediaName         bool   `long:"vmix-media-name" description:"Show vMix media name as clock text" hidden:"true"`
	VmixMediaColor        string `long:"vmix-media-color" description:"CSS color for vMix media name" default:"#FF8000" hidden:"true"`
	VmixMediaBG           string `long:"vmix-media-bg" description:"CSS color for vMix media name background" default:"#101010" hidden:"true"`
	PicturallEnabled      bool   `long:"picturall-enabled" descriptin:"Enable picturall media info support" hidden:"true"`
	PicturallAddress      string `long:"picturall-address" description:"Address for picturall media server to connect to" hidden:"true"`
	PicturallPort         int    `long:"picturall-port" description:"Picturall telnet port" default:"11000" hidden:"true"`
	PicturallTimer        int    `long:"picturall-timer" description:"Timer for picturall video state" default:"7" hidden:"true"`
	PicturallLoops        bool   `long:"picturall-loops" description:"Show looping content" hidden:"true"`
	PicturallMediaName    bool   `long:"picturall-media-name" description:"Show picturall media name as clock text" hidden:"true"`
	PicturallIgnoreLayers string `long:"picturall-ignore-layers" description:"Comma separated list of layers to ignore" hidden:"true"`
	PicturallStreams      bool   `long:"picturall-streams" description:"Show streaming content" hidden:"true"`
	PicturallMediaColor   string `long:"picturall-media-color" description:"CSS color for picturall media name" default:"#FF8000" hidden:"true"`
	PicturallMediaBG      string `long:"picturall-media-bg" description:"CSS color for picturall media name background" default:"#101010" hidden:"true"`
}

// Clock engine state constants
const (
	Normal    = iota // Display current time
	Countdown = iota // Display countdown timer only
	Countup   = iota // Count time up
	Off       = iota // (Mostly) blank screen
	Paused    = iota // Paused countdown timer(s)
	LTC       = iota // LTC display
	Media     = iota // Playing media counter
	Slave     = iota // Displaying slaved output
)

// Misc constants
const (
	NumCounters      = 10 // Number of distinct counters to initialize
	NumSources       = 4
	PrimaryCounter   = 0 // Main counter that replaces the ToD display on the round clock when active
	SecondaryCounter = 1 // Secondary counter that is displayed in the tally message space on the round clock
)

type ltcData struct {
	hours   int
	minutes int
	seconds int
	frames  int
	target  time.Time
}

// Engine contains the state machine for clock-8001
type Engine struct {
	mode                int        // Main display mode
	Counters            []*Counter // Timer counters
	sources             []*source  // Time sources for 1-3 displays
	displaySeconds      bool
	flashPeriod         int
	clockServer         *Server
	oscServer           osc.Server
	oscDispatcher       *oscutil.RegexpDispatcher
	oscDests            *feedbackDestination // udp connections to send osc feedback to
	oscSendChan         chan []byte
	oscTally            bool          // Tally text was from osc event
	timeout             time.Duration // Timeout for osc tally events
	message             string        // Full tally message as received from OSC
	messageColor        *color.RGBA   // Tally message color from OSC
	messageBG           *color.RGBA
	messageTimer        *timer.Timer
	udpDests            []*feedbackDestination // Stagetimer2 udp time destinations
	udpCounters         []*Counter
	initialized         bool        // Show version on startup until ntp synced or receiving OSC control
	ltc                 [2]*ltcData // LTC time code status
	ltcShowSeconds      bool        // Toggles led display on LTC mode between seconds and frames
	ltcFollow           bool        // Continue on internal timer if LTC signal is lost
	ltcEnabled          bool        // Toggle LTC mode on or off
	ltcTimeout          [2]bool     // Set to true if LTC signal is lost by the ltc timer
	ltcActive           [2]bool     // Do we have a active LTC to display?
	format12h           bool        // Use 12 hour format for time-of-day
	off                 bool        // Is the engine output off?
	ignoreRegexp        *regexp.Regexp
	mittiCounter        *Counter
	milluminCounter     *Counter
	picturall           []*picturallState
	vmix                []*vmixState
	tricaster           tricasterState
	hyperdeck           hyperdeckState
	disguise            disguiseState
	background          int
	info                string // Version, ip address etc
	showInfo            bool
	infoTimer           *timer.Timer
	uuid                string // Clock unique id
	titleTextColor      color.RGBA
	titleBGColor        color.RGBA
	screenFlash         bool
	signalHardwareColor color.RGBA
	signalHardware      int
	limitimer           string
	limitimerSerial     string
	limitimerBroadcast  [5]bool
	limitimerReceive    [5]bool
	ctx                 context.Context
	cancelFunc          context.CancelFunc
	wg                  *sync.WaitGroup
	overtimeVisibility  string
	cueChan             chan *cueState
	cueState            *cueState
	cueDuration         time.Duration
	highResolution      bool

	mediaTimeouts []*time.Time
	flashTimer    *timer.Timer
	ltcTimer      [2]*timer.Timer
	cueTimer      *timer.Timer
	// FIXME: Are these used anymore?
	mittiTimer    *timer.Timer
	milluminTimer *timer.Timer
}

// Clock contains the state of a single component clock / timer
type Clock struct {
	Text        string     // Normal clock representation HH:MM:SS(:FF)
	Hours       int        // Hours on the clock
	Minutes     int        // Minutes on the clock
	Seconds     int        // Seconds on the clock
	Frames      int        // Frames, only on LTC
	Label       string     // Label text
	Icon        string     // Icon for the clock type
	Compact     string     // 4 character condensed output
	Expired     bool       // true if asscociated timer is expired
	Mode        int        // Display type
	Paused      bool       // Is the clock/timer paused?
	Progress    float64    // Progress of the total timer 0-1
	Hidden      bool       // The timer should not be rendered if true
	TextColor   color.RGBA // Color for text
	BGColor     color.RGBA // Background color
	HideHours   bool       // Should the hour field of the time be displayed for this clock.
	HideSeconds bool       // Should seconds be shown for this clock
	ActiveTimer int        // If a timer is active, give its number, -1 otherwise
	SignalColor color.RGBA
	Mute        bool
}

// State is a snapshot of the clock representation on the time State() was called
type State struct {
	Initialized         bool        // Does the clock have valid time or has it received an osc command?
	Clocks              []*Clock    // All configured clocks / timers
	Tally               string      // Tally message text
	TallyColor          *color.RGBA // Tally message color
	TallyBG             *color.RGBA // Tally message background color
	Flash               bool        // Flash cycle state
	Background          int         // User selected background number
	Info                string      // Clock information, version, ip-address etc. Should be displayed if not empty
	TitleColor          color.RGBA  // Color for the clock title text
	TitleBGColor        color.RGBA  // Background color for clock title text
	ScreenFlash         bool        // Set to true if the screen should be flashed white
	HardwareSignalColor color.RGBA

	CueRight bool
	CueLeft  bool
	CueBlank bool
	CueShow  bool
}
