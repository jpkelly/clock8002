package limitimer

import (
	"errors"
	"fmt"
	"github.com/sigurn/crc16"
	"log"
)

// Different beep tones selectable via the dip switches
const (
	BeepNone  = 0
	BeepBuzz  = 1
	BeepRing  = 2
	BeepChime = 3
)

// State bitfield
const (
	stateOff       = 0
	stateRun       = 1
	stateBlink     = 2
	stateBeep      = 4
	stateSecAdjust = 8
)

// Message structure constants
const (
	crcHighByte    = 52
	crcLowByte     = 53
	crcLen         = 52
	msgLen         = 55
	timerStartByte = 6
	timerLen       = 11
	configByteHigh = 2
	configByteLow  = 3
	sequenceByte   = 4
)

// Config byte low
const (
	configBeep2        = 0x40
	configBeepLoud     = 0x20
	configContinue     = 0x10
	configCountUp      = 0x08
	configProgHours    = 0x04
	configSessionHours = 0x02
)

// Config byte high
const (
	configBeep1         = 0x01
	configPermitChanges = 0x20
)

// State is the representation of limitimer system state packet
type State struct {
	Sequence          int
	ProgramMinutes    bool // Programs 1-3 uses minutes:seconds instead of hours:minutes
	SessionMinutes    bool // Program 4, sessions uses minutes:seconds
	CountDirection    bool // True - count down, false - count up
	ContinueAfterZero bool // True - countinue counting on expired timers, false - stop at zero
	BeepLoud          bool
	BeepType          int
	PermitChanges     bool
	SelectedTimer     int
	Timers            [4]Program
}

// Program is the state of single limitimer program / timer
type Program struct {
	State    int
	Total    Time
	Sumup    Time
	Elapsed  Time
	unparsed byte // Still unknown byte..
}

// Time represents the time on the timer
type Time struct {
	H int
	M int
	S int
}

// Sync is the small 6 byte sync packet
type Sync struct {
}

var crcModbus *crc16.Table

func init() {
	crcModbus = crc16.MakeTable(crc16.CRC16_MODBUS)
}

// ParseStateMessage parses a given byte slice into a State struct
func ParseStateMessage(msg []byte) (*State, error) {
	s := State{}
	err := s.Unmarshal(msg)
	return &s, err
}

// Unmarshal updates the given state struct contents from the parsed packet from the byte slice
func (s *State) Unmarshal(msg []byte) error {
	if len(msg) != msgLen {
		return errors.New("Missmatched size")
	}

	if err := validate(msg); err != nil || Type(msg) != 0x00 {
		return fmt.Errorf("Invalid status message: %v", err)
	}

	s.Sequence = int(msg[sequenceByte])

	s.SelectedTimer = int(msg[5])

	cb := msg[configByteLow]
	s.ProgramMinutes = cb&configProgHours != 0
	s.SessionMinutes = cb&configSessionHours != 0
	s.CountDirection = cb&configCountUp != 0
	s.ContinueAfterZero = cb&configContinue != 0
	s.BeepLoud = cb&configBeepLoud != 0

	cb = msg[configByteHigh]
	s.PermitChanges = cb&configPermitChanges != 0

	beep := msg[configByteHigh]&configBeep1<<1 + msg[configByteLow]&configBeep2>>6
	s.BeepType = int(beep)

	for i := range s.Timers {
		offset := timerStartByte + (i * timerLen)
		s.Timers[i].parse(msg, offset)
		if !s.ProgramMinutes && i < 3 {
			s.Timers[i].hourAdjust()
		} else if !s.SessionMinutes && i == 3 {
			s.Timers[i].hourAdjust()
		}
	}

	return nil
}

// String outputs a string representation of the given State
func (s *State) String() (out string) {
	out = fmt.Sprintf(" -> Seq: %d Sel: P%d Timers: ", s.Sequence, s.SelectedTimer+1)
	for i := range s.Timers {
		out += fmt.Sprintf("P%d %s", i+1, s.Timers[i].String())
	}

	beepType := "Unknown"
	switch s.BeepType {
	case BeepNone:
		beepType = "None"
	case BeepBuzz:
		beepType = "Buzz"
	case BeepRing:
		beepType = "Ring"
	case BeepChime:
		beepType = "Chime"
	}

	out += fmt.Sprintf("Beep type: %s ", beepType)

	return
}

// Marshal convers a State struct into a byte array representing the limitimer state packet
func (s *State) Marshal() []byte {
	ret := make([]byte, 0)
	ret = append(ret, 0x81, 0x00, 0x00, 0x01, byte(s.Sequence), byte(s.SelectedTimer))

	// Config high byte
	if s.PermitChanges {
		ret[configByteHigh] += configPermitChanges
	}
	if s.BeepType&0x02 != 0 {
		ret[configByteHigh] += configBeep1
	}

	// Config low byte

	if s.BeepType&0x01 != 0 {
		ret[configByteLow] += configBeep2
	}

	if s.BeepLoud {
		ret[configByteLow] += configBeepLoud
	}

	if s.ContinueAfterZero {
		ret[configByteLow] += configContinue
	}

	if s.CountDirection {
		ret[configByteLow] += configCountUp
	}

	if s.ProgramMinutes {
		ret[configByteLow] += configProgHours
	}

	if s.SessionMinutes {
		ret[configByteLow] += configSessionHours
	}

	for i := range s.Timers {
		ret = append(ret, s.Timers[i].encode()...)
	}

	ret = append(ret, 0x00, 0x83)

	crc := crc16.Checksum(ret, crcModbus)

	crc1 := byte(crc >> 8)
	crc2 := byte(crc & 0xFF)

	ret = append(ret, crc1, crc2, 0xFF)

	return ret
}

// TimerDisplay formats the given timers data into hours, minutes and seconds useful as a display data
func (s *State) TimerDisplay(i int) (hour, min, sec int) {
	minutes := false
	timer := s.Timers[i]
	t := 0

	if i < 3 && s.ProgramMinutes {
		minutes = true
	} else if i == 3 && s.SessionMinutes {
		minutes = true
	}

	if s.CountDirection {
		// Count down
		t = timer.Total.Seconds() - timer.Elapsed.Seconds()
	} else {
		t = timer.Elapsed.Seconds()
	}

	if minutes {
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

	if timer.Expired() {
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

	return
}

// Minutes gets the minutes/hours setting for a given timer
func (s *State) Minutes(i int) bool {
	if i < 3 && s.ProgramMinutes {
		return true
	}
	if i == 3 && s.SessionMinutes {
		return true
	}
	return false
}

// Run gets the timer run/pause flag
func (t *Program) Run() bool {
	return t.State&stateRun != 0
}

// SetRun sets the timer run/pause flag
func (t *Program) SetRun(s bool) {
	if s {
		t.State |= stateRun
	} else {
		t.State &= ^stateRun
	}
}

// Blink gets the timer blink flag
func (t *Program) Blink() bool {
	return t.State&stateBlink != 0
}

// SetBlink sets the timer blink flag
func (t *Program) SetBlink(s bool) {
	if s {
		t.State |= stateBlink
	} else {
		t.State &= ^stateBlink
	}
}

// Beep gets the timer beep flag
func (t *Program) Beep() bool {
	return t.State&stateBeep != 0
}

// SetBeep sets the timer beep flag
func (t *Program) SetBeep(s bool) {
	if s {
		t.State |= stateBeep
	} else {
		t.State &= ^stateBeep
	}
}

// SecAdjust gets the timer seconds adjust flag
func (t *Program) SecAdjust() bool {
	return t.State&stateSecAdjust != 0
}

// SetSecAdjust sets the timer seconds adjust flag
func (t *Program) SetSecAdjust(s bool) {
	if s {
		t.State |= stateSecAdjust
	} else {
		t.State &= ^stateSecAdjust
	}
}

// String formats the given timer data as a string
func (t *Program) String() (out string) {
	state := ""

	if t.Run() {
		state += "R"
	} else {
		state += " "
	}

	if t.Blink() {
		state += " B"
	} else {
		state += "  "
	}

	if t.Beep() {
		state += " SND"
	} else {
		state += "   "
	}

	if t.SecAdjust() {
		state += " ADJ"
	} else {
		state += "    "
	}

	out = fmt.Sprintf("%-11s %0.2X %s / %s (%s) ", state, t.unparsed, t.Elapsed.String(), t.Total.String(), t.Sumup.String())
	return
}

func (t *Program) parse(msg []byte, offset int) {
	t.State = int(msg[offset])
	t.unparsed = msg[offset+1]

	t.Total.M, t.Total.S = parseTime(msg, offset+2)
	t.Sumup.M, t.Sumup.S = parseTime(msg, offset+5)
	t.Elapsed.M, t.Elapsed.S = parseTime(msg, offset+8)
}

func (t *Program) hourAdjust() {
	t.Total.H = t.Total.M / 60
	t.Total.M = t.Total.M % 60

	t.Sumup.H = t.Sumup.M / 60
	t.Sumup.M = t.Sumup.M % 60

	t.Elapsed.H = t.Elapsed.M / 60
	t.Elapsed.M = t.Elapsed.M % 60
}

func (t *Program) encode() []byte {
	r := make([]byte, 0)
	r = append(r, byte(t.State), t.unparsed)
	r = append(r, t.Total.encode()...)
	r = append(r, t.Sumup.encode()...)
	r = append(r, t.Elapsed.encode()...)

	return r
}

// Expired returns the expiration flag for a timer
func (t *Program) Expired() bool {
	return t.Total.Seconds()-t.Elapsed.Seconds() < 1
}

// Warning returns a flag telling if the timer is over the sumup time
func (t *Program) Warning() bool {
	if t.Total.Seconds()-t.Elapsed.Seconds() <= t.Sumup.Seconds() {
		return !t.Expired()
	}
	return false
}

// Seconds returns the raw seconds of a given Time object as integer
func (t *Time) Seconds() int {
	return t.H*60*60 + t.M*60 + t.S
}

// SetSeconds sets the time from raw integer seconds
func (t *Time) SetSeconds(s int) {
	t.M = s / 60
	t.S = s % 60
}

// String gets the string representation of a Time
func (t *Time) String() string {
	return fmt.Sprintf("%0.2d:%0.2d:%0.2d", t.H, t.M, t.S)
}

func (t *Time) encode() []byte {
	ret := make([]byte, 3)
	secs := t.S + 60*t.M + 60*60*t.H

	ret[0] = byte(secs / 128 / 128 % 128)
	ret[1] = byte(secs / 128 % 128)
	ret[2] = byte(secs % 128)

	return ret
}

func parseTime(msg []byte, i int) (m int, s int) {
	t := parseTriple(msg, i)
	s = t % 60
	m = t / 60
	return
}

func parseTriple(msg []byte, i int) int {
	return int(msg[i])*128*128 + int(msg[i+1])*128 + int(msg[i+2])
}

func parseDouble(msg []byte, i int) int {
	return int(msg[i])*128 + int(msg[i+1])
}

func outputRaw(msg []byte) {
	out := " -> "
	out += fmt.Sprintf(" [%02d] ", len(msg))
	for _, b := range msg {
		out += fmt.Sprintf("%0.2X ", b)
	}
	out = out[:len(out)-1]
	log.Printf(out)
}
