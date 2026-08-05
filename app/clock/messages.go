package clock

import (
	"github.com/jpkelly/clock8002/app/oscutil"
	// "gitlab.com/Depili/go-osc/osc"
	"fmt"
	"github.com/chabad360/go-osc/osc"
	"image/color"
	"strings"
)

var clockUnits = []struct {
	unit    string
	seconds float32
}{
	{"s", 1},
	{"m", 60},
	{"h", 60 * 60},
	{"d", 24 * 60 * 60},
}

// Message is a generic clock message for decoded osc data
type Message struct {
	Type               string
	Counter            int
	Countdown          bool
	Data               string
	CountdownMessage   *CountdownMessage
	DisplayMessage     *DisplayMessage
	MediaMessage       *MediaMessage
	DisplayTextMessage *displayTextMessage
	LimitimerMessage   *LimitimerMessage
	Colors             []color.RGBA
}

// MediaMessage contains data from media players
type MediaMessage struct {
	hours     int32
	minutes   int32
	seconds   int32
	frames    int32
	remaining int32
	progress  float32
	paused    bool
	looping   bool
	timeStamp osc.Timetag
	uuid      string
}

// UnmarshalOSC converts a osc.Message to MediaMessage
func (message *MediaMessage) UnmarshalOSC(msg *osc.Message) error {
	return oscutil.UnmarshalArguments(
		msg,
		&message.hours,
		&message.minutes,
		&message.seconds,
		&message.frames,
		&message.remaining,
		&message.progress,
		&message.paused,
		&message.looping,
		&message.timeStamp,
		&message.uuid,
	)
}

// MarshalOSC converts a MediaMessage to osc.Message
func (message MediaMessage) MarshalOSC(addr string) *osc.Message {
	return osc.NewMessage(addr,
		message.hours,
		message.minutes,
		message.seconds,
		message.frames,
		message.remaining,
		message.progress,
		message.paused,
		message.looping,
		message.timeStamp,
		message.uuid,
	)
}

// CountdownMessage is for /clock/countdown/start
type CountdownMessage struct {
	Seconds int32
}

// DisplayMessage is for /clock/display
type DisplayMessage struct {
	ColorRed   float32
	ColorGreen float32
	ColorBlue  float32
	Text       string
}

// TimeMessage is for /clock/settime and /clock/ltc
type TimeMessage struct {
	Time string
}

type displayTextMessage struct {
	r    int32
	g    int32
	b    int32
	a    int32
	bgR  int32
	bgG  int32
	bgB  int32
	bgA  int32
	time int32
	text string
}

func (message *displayTextMessage) UnmarshalOSC(msg *osc.Message) error {
	return oscutil.UnmarshalArguments(
		msg,
		&message.r,
		&message.g,
		&message.b,
		&message.a,
		&message.bgR,
		&message.bgG,
		&message.bgB,
		&message.bgA,
		&message.time,
		&message.text,
	)
}

// LimitimerMessage is for relaying limitimer program states over OSC
type LimitimerMessage struct {
	Total     int32
	Sumup     int32
	Elapsed   int32
	Minutes   bool
	Countdown bool
	Run       bool
	Blink     bool
	Beep      bool
	UUID      string
}

// UnmarshalOSC converts a osc.Message to LimitimerMessage
func (message *LimitimerMessage) UnmarshalOSC(msg *osc.Message) error {
	return oscutil.UnmarshalArguments(
		msg,
		&message.Total,
		&message.Sumup,
		&message.Elapsed,
		&message.Minutes,
		&message.Countdown,
		&message.Run,
		&message.Blink,
		&message.Beep,
		&message.UUID,
	)
}

// MarshalOSC converts a LimitimerMessage to osc.Message
func (message *LimitimerMessage) MarshalOSC(addr string) *osc.Message {
	return osc.NewMessage(addr,
		message.Total,
		message.Sumup,
		message.Elapsed,
		message.Minutes,
		message.Countdown,
		message.Run,
		message.Blink,
		message.Beep,
		message.UUID,
	)
}

// TextMessage is for text only messages like /clock/dual/text
type TextMessage struct {
	Text string
}

// UnmarshalOSC converts a osc.Message to TextMessage
func (message *TextMessage) UnmarshalOSC(msg *osc.Message) error {
	return oscutil.UnmarshalArguments(
		msg,
		&message.Text,
	)
}

// MarshalOSC converts a TextMessage to osc.Message
func (message TextMessage) MarshalOSC(addr string) *osc.Message {
	return osc.NewMessage(addr,
		message.Text,
	)
}

// UnmarshalOSC converts a osc.Message to TimeMessage
func (message *TimeMessage) UnmarshalOSC(msg *osc.Message) error {
	return oscutil.UnmarshalArguments(
		msg,
		&message.Time,
	)
}

// MarshalOSC converts a TimeMessage to osc.Message
func (message TimeMessage) MarshalOSC(addr string) *osc.Message {
	return osc.NewMessage(addr,
		message.Time,
	)
}

// UnmarshalOSC converts a osc.Message to DisplayMessage
func (message *DisplayMessage) UnmarshalOSC(msg *osc.Message) error {
	return oscutil.UnmarshalArguments(
		msg,
		&message.ColorRed,
		&message.ColorGreen,
		&message.ColorBlue,
		&message.Text,
	)
}

// MarshalOSC converts a DisplayMessage to osc.Message
func (message DisplayMessage) MarshalOSC(addr string) *osc.Message {
	return osc.NewMessage(addr,
		message.ColorRed,
		message.ColorGreen,
		message.ColorBlue,
		message.Text,
	)
}

// UnmarshalOSC converts a osc.Message to CountdownMessage
func (message *CountdownMessage) UnmarshalOSC(msg *osc.Message) error {
	return oscutil.UnmarshalArguments(
		msg,
		&message.Seconds,
	)
}

// MarshalOSC converts a CountdownMessage to osc.Message
func (message CountdownMessage) MarshalOSC(addr string) *osc.Message {
	return osc.NewMessage(addr,
		message.Seconds,
	)
}

// ColorMessage holds text and background colors with alpha
type ColorMessage struct {
	r   int32
	g   int32
	b   int32
	a   int32
	bgR int32
	bgG int32
	bgB int32
	bgA int32
}

// UnmarshalOSC creates a colormessage from OSC message
func (message *ColorMessage) UnmarshalOSC(msg *osc.Message) error {
	return oscutil.UnmarshalArguments(
		msg,
		&message.r,
		&message.g,
		&message.b,
		&message.a,
		&message.bgR,
		&message.bgG,
		&message.bgB,
		&message.bgA,
	)
}

// ToRGBA converts the message contents to a slice of color.RGBA
func (message *ColorMessage) ToRGBA() []color.RGBA {
	text := color.RGBA{
		R: uint8(message.r),
		G: uint8(message.g),
		B: uint8(message.b),
		A: uint8(message.a),
	}
	bg := color.RGBA{
		R: uint8(message.bgR),
		G: uint8(message.bgG),
		B: uint8(message.bgB),
		A: uint8(message.bgA),
	}
	ret := make([]color.RGBA, 2)
	ret[0] = text
	ret[1] = bg
	return ret
}

// SourceStateMessage is a representation of
// the clock source state OSC messages
type SourceStateMessage struct {
	UUID     string
	Hidden   bool
	Text     string
	Compact  string
	Icon     string
	Progress float32
	Expired  bool
	Paused   bool
	Label    string
	Mode     int32
}

// MarshalOSC converts SourceStateMessage into osc.Message
func (ssm *SourceStateMessage) MarshalOSC(addr string) *osc.Message {
	return osc.NewMessage(addr,
		ssm.UUID,
		ssm.Hidden,
		ssm.Text,
		ssm.Compact,
		ssm.Icon,
		ssm.Progress,
		ssm.Expired,
		ssm.Paused,
		ssm.Label,
		ssm.Mode,
	)
}

// UnmarshalOSC converts osc.Message into SourceStateMessage
func (ssm *SourceStateMessage) UnmarshalOSC(msg *osc.Message) error {
	return oscutil.UnmarshalArguments(
		msg,
		&ssm.UUID,
		&ssm.Hidden,
		&ssm.Text,
		&ssm.Compact,
		&ssm.Icon,
		&ssm.Progress,
		&ssm.Expired,
		&ssm.Paused,
		&ssm.Label,
		&ssm.Mode,
	)
}

func sourceStateMessageFromClock(s *Clock, uuid string) *SourceStateMessage {
	state := SourceStateMessage{
		UUID:     uuid,
		Hidden:   s.Hidden,
		Text:     s.Text,
		Compact:  s.Compact,
		Icon:     s.Icon,
		Progress: float32(s.Progress),
		Expired:  s.Expired,
		Paused:   s.Paused,
		Label:    s.Label,
		Mode:     int32(s.Mode),
	}
	if strings.Count(state.Text, ":") < 2 {
		state.Text = fmt.Sprintf("%02d:%02d:%02d", s.Hours, s.Minutes, s.Seconds)
	}

	return &state
}

// TimerStateMessage is a representation of timer
// state OSC messages
type TimerStateMessage struct {
	UUID     string
	Active   bool
	Text     string
	Compact  string
	Icon     string
	Progress float32
	Expired  bool
	Paused   bool
}

// MarshalOSC converts a TimerStateMessage to osc.Message
func (tsm *TimerStateMessage) MarshalOSC(addr string) *osc.Message {
	return osc.NewMessage(addr,
		tsm.UUID,
		tsm.Active,
		tsm.Text,
		tsm.Compact,
		tsm.Icon,
		tsm.Progress,
		tsm.Expired,
		tsm.Paused,
	)
}

// UnmarshalOSC converts osc.Message into TimerStateMessage
func (tsm *TimerStateMessage) UnmarshalOSC(msg *osc.Message) error {
	return oscutil.UnmarshalArguments(
		msg,
		&tsm.UUID,
		&tsm.Active,
		&tsm.Text,
		&tsm.Compact,
		&tsm.Icon,
		&tsm.Progress,
		&tsm.Expired,
		&tsm.Paused,
	)
}

func timerStateMessageFromOutput(out *CounterOutput, uuid string) *TimerStateMessage {
	state := TimerStateMessage{
		UUID:     uuid,
		Active:   out.Active,
		Text:     out.Text,
		Compact:  out.Compact,
		Icon:     out.Icon,
		Progress: float32(out.Progress),
		Expired:  out.Expired,
		Paused:   out.Paused,
	}

	if strings.Count(state.Text, ":") < 3 {
		state.Text = fmt.Sprintf("%02d:%02d:%02d", out.Hours, out.Minutes, out.Seconds)
	}

	return &state
}
