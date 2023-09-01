package disguise

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Media implements the clock mediaUpdate interface for state transfer
type Media struct {
	head        time.Duration // Time since last cue/note
	remaining   time.Duration // Time to next cue/note
	sectionName string        // Name of the section, not sure how to set in in disguise gui
	note        string        // Current note / cue name
	playMode    string        // Play, PlaySection, LoopSection, Stop, HoldSection, HoldEnd
}

// Listen creates a new listener instance for disguise OSC communications
func Listen(ctx context.Context, wg *sync.WaitGroup, port int, nextName bool) (chan *Media, error) {
	log.Printf("Disguise: Creating new listener: %d", port)

	addr := fmt.Sprintf("0.0.0.0:%d", port)

	c := make(chan *Media)
	s := state{
		ctx:      ctx,
		wg:       wg,
		c:        c,
		nextName: nextName,
	}

	s.setup(addr)

	go s.run()

	return c, nil
}

// Remaining gives the time remaining on the playing section
func (m *Media) Remaining() time.Duration {
	return m.remaining
}

// Duration gives the length of the current section
func (m *Media) Duration() time.Duration {
	return m.head + m.remaining
}

// Play tells if the section is playing or stopped
func (m *Media) Play() bool {
	return m.playMode != "Stop" && m.playMode != "HoldEnd" && m.playMode != "HoldSection"
}

// Loop tells if the section is looping
func (m *Media) Loop() bool {
	return m.playMode == "LoopSection"
}

// Record always returns false, as disguise doesn't do recording
func (m *Media) Record() bool {
	return false
}

// MediaName gives the section name if defined, or the latest note name
func (m *Media) MediaName() string {
	if m.sectionName != "" {
		return m.sectionName
	}
	return m.note
}

func (m *Media) String() string {
	return fmt.Sprintf("Disguise: -%s (%s) Play: %v Loop: %v MediaName: %v", m.Remaining(), m.Duration(), m.Play(), m.Loop(), m.MediaName())
}
