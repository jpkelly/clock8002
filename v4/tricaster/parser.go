package tricaster

import (
	"encoding/xml"
	"fmt"
)

// /v1/dictionary?key=ddr_timecode format
type ddrTimecode struct {
	XMLName xml.Name  `xml:"timecode"`
	DDR1    *ddrState `xml:"ddr1"`
	DDR2    *ddrState `xml:"ddr2"`
	DDR3    *ddrState `xml:"ddr3"`
	DDR4    *ddrState `xml:"ddr4"`
}

type ddrState struct {
	XMLName           xml.Name
	Elapsed           float64 `xml:"clip_seconds_elapsed,attr"`
	Remaining         float64 `xml:"clip_seconds_remaining,attr"`
	EmbededTimecode   float64 `xml:"clip_embedded_timecode,attr"`
	In                float64 `xml:"clip_in,attr"`
	Out               float64 `xml:"clip_out,attr"`
	Duration          float64 `xml:"file_duration,attr"`
	PlaySpeed         float64 `xml:"play_speed,attr"`
	Framerate         float64 `xml:"clip_framerate,attr"`
	PlaylistElapsed   float64 `xml:"playlist_seconds_elapsed,attr"`
	PlaylistRemaining float64 `xml:"playlist_seconds_remaining,attr"`
	PresetIndex       int     `xml:"preset_index,attr"`
	ClipIndex         int     `xml:"clip_index,attr"`
	Clips             int     `xml:"num_clips,attr"`
}

// /v/dictionary?key=switcher format
type switcher struct {
	XMLName  xml.Name  `xml:"switcher_update"`
	PGM      string    `xml:"main_source,attr"`
	PVM      string    `xml:"preview_source,attr"`
	Tbar     *tbar     `xml:"tbar"`
	Overlays []overlay `xml:"switcher_overlays>overlay"`
}

type overlay struct {
	XMLName xml.Name `xml:"overlay"`
	Z       int      `xml:"z_order_position,attr"`
	Source  string   `xml:"source,attr"`
	Tbar    *tbar    `xml:"tbar"`
}

type tbar struct {
	XMLName  xml.Name `xml:"tbar"`
	Position float64  `xml:"position,attr"`
}

// /v/datalink format
type dataLink struct {
	XMLName xml.Name `xml:"datalink_values"`
	Data    []data   `xml:"data"`
}

type data struct {
	XMLName xml.Name `xml:"data"`
	Key     string   `xml:"key"`
	Value   string   `xml:"value"`
}

func (d *ddrState) String() string {
	return fmt.Sprintf("->Remaining %.2f (%02.2f) Playlist remaining: %.2f", d.Remaining, d.Duration, d.PlaylistRemaining)
}

func (s *switcher) live(source string) bool {
	if s.PGM == source {
		return true
	}

	for _, o := range s.Overlays {
		if o.Source == source && o.Tbar.Position != 0 {
			return true
		}
	}
	return false
}

func (s *switcher) preview(source string) bool {
	return s.PVM == source
}

func (tc *ddrTimecode) ddr(ddr int) *ddrState {
	switch ddr {
	case 1:
		return tc.DDR1
	case 2:
		return tc.DDR2
	case 3:
		return tc.DDR3
	case 4:
		return tc.DDR4
	}
	return nil
}

func (d *dataLink) String() string {
	ret := ""

	for _, v := range d.Data {
		ret += fmt.Sprintf("\t%s -> %v\n", v.Key, v.Value)
	}
	return ret
}

func (d *dataLink) alias(ddr int) string {
	a := fmt.Sprintf("DDR%d Clip Alias", ddr)
	for _, d := range d.Data {
		if d.Key == a {
			return d.Value
		}
	}
	return ""
}

func (d *dataLink) nextEvent() (string, string) {
	time := "00:00:00"
	name := ""
	for _, d := range d.Data {
		if d.Key == "Time Until Next Event" {
			time = d.Value
		}
		if d.Key == "Next Event" {
			name = d.Value
		}
	}
	return time, name
}
