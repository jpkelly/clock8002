package mitti

import (
	"fmt"
	"github.com/jpkelly/clock8002/app/debug"
	"log"
	"time"
)

// State is the Mitti playback state from osc messages
type State struct {
	remaining int
	elapsed   int
	hours     int
	minutes   int
	seconds   int
	frames    int
	progress  float64
	paused    bool
	loop      bool
	updated   time.Time
	mediaName string
}

// Remaining tells the remaining time on the cue being played
func (state *State) Remaining() time.Duration {
	return time.Duration(state.remaining) * time.Second
}

// Duration gives the total cue duration
func (state *State) Duration() time.Duration {
	return time.Duration(state.remaining+state.elapsed) * time.Second
}

// Play tells if the cue is playing
func (state *State) Play() bool {
	return !state.paused
}

// Loop tells if the cue is looping
func (state *State) Loop() bool {
	return state.loop
}

// Record is always false for Mitti
func (state *State) Record() bool {
	return false
}

// MediaName is the cue name
func (state *State) MediaName() string {
	return state.mediaName
}

// IconOverride is always empty
func (state *State) IconOverride() string {
	return ""
}

func (state *State) String() string {
	return fmt.Sprintf("Mitti state updated %.2fs ago: left=%d elapsed=%d paused=%v loop=%v",
		time.Since(state.updated).Seconds(),
		state.remaining,
		state.elapsed,
		state.paused,
		state.loop,
	)
}

// CueTimeLeft gets the time left on current Mitti cue
func (state *State) CueTimeLeft(cueTimeLeft string) {
	var hours, min, sec, cs int

	n, err := fmt.Sscanf(cueTimeLeft, "-%2d:%2d:%2d:%2d", &hours, &min, &sec, &cs)
	if err != nil || n != 4 {
		log.Printf("Error parsing cueTimeLeft string: %v, error: %v", cueTimeLeft, err)
		return
	}

	state.hours = hours
	state.minutes = min
	state.seconds = sec
	state.updated = time.Now()

	min += hours * 60
	sec += min * 60
	cs += sec * 100

	state.remaining = sec
}

// CueTimeElapsed gets the elapsed time on current Mitti cue
func (state *State) CueTimeElapsed(cueTimeElapsed string) {
	var hours, min, sec, cs int

	n, err := fmt.Sscanf(cueTimeElapsed, "%2d:%2d:%2d:%2d", &hours, &min, &sec, &cs)
	if err != nil || n != 4 {
		log.Printf("Error parsing cueTimeElapsed string: \"%v\", error: %v", cueTimeElapsed, err)
		return
	}

	min += hours * 60
	sec += min * 60
	cs += sec * 100

	state.updated = time.Now()
	state.elapsed = sec
	debug.Printf("Mitti: elpased: %v\n", state.elapsed)
}

// TogglePlay toggles the play/pause state
func (state *State) TogglePlay(i int32) {
	state.updated = time.Now()
	if i == 0 {
		state.paused = true
	} else {
		state.paused = false
	}

	debug.Printf("Mitti: togglePlay: %d", i)
}

// ToggleLoop toggles the loop state
func (state *State) ToggleLoop(i int32) {
	state.updated = time.Now()
	if i == 0 {
		state.loop = false
	} else {
		state.loop = true
	}

	debug.Printf("Mitti: toggleLoop: %d", i)
}

// Copy creates a new copy of the Mitti state
func (state *State) Copy() State {
	s := State{
		remaining: state.remaining,
		elapsed:   state.elapsed,
		hours:     state.hours,
		minutes:   state.minutes,
		seconds:   state.seconds,
		frames:    state.frames,
		progress:  state.progress,
		paused:    state.paused,
		updated:   state.updated,
		loop:      state.loop,
		mediaName: state.mediaName,
	}

	return s
}
