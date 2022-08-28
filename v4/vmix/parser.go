package vmix

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"time"
)

type vmixBool bool

// Status is a struct for the vMix status xml
type status struct {
	XMLName     xml.Name  `xml:"vmix"`
	Version     string    `xml:"version"`
	Edition     string    `xml:"edition"`
	Preview     string    `xml:"preview"`
	Active      string    `xml:"active"`
	Inputs      []input   `xml:"inputs>input"`
	Overlays    []overlay `xml:"overlays>overlay"`
	Recording   vmixBool  `xml:"recording"`
	External    vmixBool  `xml:"external"`
	Streaming   vmixBool  `xml:"streaming"`
	Playlist    vmixBool  `xml:"playlist"`
	MultiCorder vmixBool  `xml:"multiCorder"`
	Fullscreen  vmixBool  `xml:"fullscreen"`
}

type input struct {
	XMLName    xml.Name `xml:"input"`
	Key        string   `xml:"key,attr"`
	Number     int      `xml:"number,attr"`
	Type       string   `xml:"type,attr"`
	Title      string   `xml:"tittle,attr"`
	ShortTitle string   `xml:"shortTitle,attr"`
	State      string   `xml:"state,attr"`
	Position   int      `xml:"position,attr"`
	Duration   int      `xml:"duration,attr"`
	Loop       vmixBool `xml:"loop,attr"`
	Content    string   `xml:",chardata"`
	Selected   int      `xml:"selectedIndex,attr"`
	Items      []item   `xml:"list>item"`
}

type overlay struct {
	XMLName xml.Name `xml:"overlay"`
	Number  int      `xml:"number,attr"`
	Input   string   `xml:",chardata"`
}

type item struct {
	XMLName  xml.Name `xml:"item"`
	Selected bool     `xml:"selected,attr"`
	File     string   `xml:",chardata"`
}

// State converts the raw parsed xml to a State message
func (s status) State() *State {
	state := &State{
		External:    bool(s.External),
		Recording:   bool(s.Recording),
		MultiCorder: bool(s.MultiCorder),
		Playlist:    bool(s.Playlist),
		Videos:      make([]*Video, 0),
	}

	overlays := make(map[int]int)
	for _, o := range s.Overlays {
		if i, err := strconv.Atoi(o.Input); err != nil {
			overlays[i] = o.Number
		}
	}

	for _, i := range s.Inputs {
		if i.File() {
			v := &Video{
				Name:     i.ShortTitle,
				Looping:  bool(i.Loop),
				Playing:  i.Playing(),
				Live:     s.Live(i.Number),
				PGM:      s.PGM() == i.Number,
				Length:   i.Length(),
				Head:     i.Head(),
				Progress: i.Progress(),
				Overlay:  -1,
			}
			if o, ok := overlays[i.Number]; ok {
				v.Overlay = o
			}

			if i.Type == "VideoList" {
				v.Name = fmt.Sprintf("%d/%d %s", i.Selected, len(i.Items), i.Content)
			}
			state.Videos = append(state.Videos, v)
		}
	}

	return state
}

// PGM is the current program input
func (s *status) PGM() int {
	p, _ := strconv.Atoi(s.Active)
	return p
}

// Overlay returns true if the input is an active overlay
func (s *status) Overlay(input int) bool {
	for _, o := range s.Overlays {
		if input == o.Source() {
			return true
		}
	}
	return false
}

// Live returns true if the input is live as PGM or overlay
func (s *status) Live(input int) bool {
	if s.PGM() == input || s.Overlay(input) {
		return true
	}
	return false
}

// File returns true if the input is a file
func (i *input) File() bool {
	return i.Type == "Video" || i.Type == "VideoList"
}

// Playing returns true if the input is playing
func (i *input) Playing() bool {
	return i.State == "Running"
}

// Length gives the input media length as time.Duration
func (i *input) Length() time.Duration {
	return time.Duration(i.Duration) * time.Millisecond
}

// Head gives the input media playhead position as time.Duration
func (i *input) Head() time.Duration {
	return time.Duration(i.Position) * time.Millisecond
}

// Progress return the playback progress as a float, 1-0
func (i *input) Progress() float64 {
	return 1 - i.Head().Seconds()/i.Length().Seconds()
}

// Source returns the active input of an overlay. Returns -1 if no input is active
func (o *overlay) Source() int {
	i, err := strconv.Atoi(o.Input)
	if err == nil {
		return i
	}
	return -1
}

// UnmarshalXMLAttr converts vmix xml True/Flase field into boolean
func (b *vmixBool) UnmarshalXMLAttr(attr xml.Attr) (err error) {
	if attr.Value == "True" {
		*b = true
	} else {
		*b = false
	}
	return nil
}

func (s *status) String() string {
	ret := fmt.Sprintf("Version: %s Edition %s PVM: %s PGM %s ", s.Version, s.Edition, s.Preview, s.Active)
	ret += fmt.Sprintf("Recording: %v External: %v Streaming %v Playlist %v\n", s.Recording, s.External, s.Streaming, s.Playlist)
	for _, i := range s.Inputs {
		ret += fmt.Sprintf(" -> %s\n", i.String())
	}
	for _, o := range s.Overlays {
		ret += fmt.Sprintf(" -> %s\n", o.String())
	}
	return ret
}

func (i *input) String() string {
	return fmt.Sprintf("Input %d (%s) Type: %s State: %s Loop: %v Position: %s Duration: %s Progress: %0.2f Title: %s", i.Number, i.Key, i.Type, i.State, i.Loop, i.Head(), i.Length(), i.Progress(), i.ShortTitle)
}

func (o *overlay) String() string {
	return fmt.Sprintf("Overlay %d: %s", o.Number, o.Input)
}
