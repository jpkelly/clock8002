package clock

import (
	"gitlab.com/Depili/go-osc/osc"
	"math/rand"
	"testing"
	"time"
)

func TestLimitimerMessage(t *testing.T) {
	m := LimitimerMessage{
		Total:     rand.Int31(),
		Sumup:     rand.Int31(),
		Elapsed:   rand.Int31(),
		Minutes:   rand.Intn(1) == 1,
		Countdown: rand.Intn(1) == 1,
		Run:       rand.Intn(1) == 1,
		Blink:     rand.Intn(1) == 1,
		Beep:      rand.Intn(1) == 1,
		UUID:      "loremIpsum",
	}

	enc := m.MarshalOSC("/clock/test")

	m2 := LimitimerMessage{}
	err := m2.UnmarshalOSC(enc)
	if err != nil {
		t.Errorf("Error unmarshalling OSC message: original: %v, error: %v", m, err)
	}

	if m != m2 {
		t.Errorf("Source message and encode-decode message don't match")
	}

}

func TestMediaMessage(t *testing.T) {
	m := MediaMessage{
		hours:     rand.Int31(),
		minutes:   rand.Int31(),
		seconds:   rand.Int31(),
		frames:    rand.Int31(),
		remaining: rand.Int31(),
		progress:  rand.Float64(),
		paused:    rand.Intn(1) == 1,
		looping:   rand.Intn(1) == 1,
		timeStamp: osc.NewTimetagFromTime(time.Now()),
		uuid:      "loremIpsum",
	}

	enc := m.MarshalOSC("/clock/test")

	m2 := MediaMessage{}
	err := m2.UnmarshalOSC(enc)
	if err != nil {
		t.Errorf("Error unmarshalling OSC message: original: %v, error: %v", m, err)
	}

	if m != m2 {
		t.Errorf("Source message and encode-decode message don't match")
	}
}

func TestSourceStateMessage(t *testing.T) {
	m := SourceStateMessage{
		UUID:     "loremIpsum",
		Hidden:   randBool(),
		Text:     "00:01:02",
		Compact:  "",
		Icon:     "▶",
		Progress: rand.Float32(),
		Expired:  randBool(),
		Paused:   randBool(),
		Label:    "Foobar",
		Mode:     rand.Int31(),
	}

	enc := m.MarshalOSC("/clock/test")

	m2 := SourceStateMessage{}
	err := m2.UnmarshalOSC(enc)
	if err != nil {
		t.Errorf("Error unmarshalling OSC message: original: %v, error: %v", m, err)
	}

	if m != m2 {
		t.Errorf("Source message and encode-decode message don't match")
	}
}

func TestTimerStateMessage(t *testing.T) {
	m := TimerStateMessage{
		UUID:     "loremIpsum",
		Active:   randBool(),
		Text:     "00:01:02",
		Compact:  "",
		Icon:     "▶",
		Progress: rand.Float32(),
		Expired:  randBool(),
		Paused:   randBool(),
	}

	enc := m.MarshalOSC("/clock/test")

	m2 := TimerStateMessage{}
	err := m2.UnmarshalOSC(enc)
	if err != nil {
		t.Errorf("Error unmarshalling OSC message: original: %v, error: %v", m, err)
	}

	if m != m2 {
		t.Errorf("Source message and encode-decode message don't match")
	}
}

func randBool() bool {
	return rand.Intn(1) == 1
}
