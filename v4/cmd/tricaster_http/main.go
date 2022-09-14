package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/icholy/digest"
	"gitlab.com/clock-8001/clock-8001/v4/tricaster"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	log.Printf("Tricaster test code starting")

	c := tricaster.Connect(context.TODO(), "127.0.0.1:3000", "admin", "admin", 100*time.Millisecond)

	for s := range c {
		log.Printf("Got state: %s", s)
	}
}

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

func (s *switcher) Live(source string) bool {
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

func (d *dataLink) String() string {
	ret := ""

	for _, v := range d.Data {
		if strings.Index(v.Key, "DDR") != -1 {
			ret += fmt.Sprintf("%s -> %v", v.Key, v.Value)
		}
	}
	return ret
}

type ddrStatus struct {
	name      string
	remaining time.Duration
	length    time.Duration
	live      bool
}

func (d *dataLink) Alias(ddr int) string {
	a := fmt.Sprintf("DDR%d Clip Alias", ddr)
	for _, d := range d.Data {
		if d.Key == a {
			return d.Value
		}
	}
	return ""
}

func (tc *ddrTimecode) DDR(ddr int) *ddrState {
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

func (d *dataLink) NextEvent() (string, string) {
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

func oldMain() {
	log.Printf("foo")
	for {
		tc := getTimecode()
		sw := getSwitcher()
		dl := getDataLink()

		log.Printf("DDR1 remaining %.2f", tc.DDR1.Remaining)
		log.Printf("DDR2 remaining %.2f", tc.DDR2.Remaining)
		log.Printf("DDR3 remaining %.2f", tc.DDR3.Remaining)
		log.Printf("DDR4 remaining %.2f", tc.DDR4.Remaining)
		log.Printf("SW PGM: %s", sw.PGM)
		for _, o := range sw.Overlays {
			if o.Tbar.Position != 0 {
				log.Printf("Live overlay: %s", o.Source)
			}
		}

		for i := 1; i < 5; i++ {
			ddr := fmt.Sprintf("DDR%d", i)
			if sw.Live(ddr) {
				log.Printf("%s is live", ddr)
			}
		}

		ddr := make([]ddrStatus, 4)

		for i := 0; i < 4; i++ {
			n := fmt.Sprintf("DDR%d", i+1)
			ddr[i].live = sw.Live(n)
			ddr[i].name = dl.Alias(i + 1)
			s := tc.DDR(i + 1)
			if s == nil {
				log.Printf("adsas")
				continue
			}
			ddr[i].remaining = time.Duration(s.Remaining) * time.Second
			ddr[i].length = time.Duration(s.Duration) * time.Second
		}

		log.Printf("")
		for _, d := range ddr {
			log.Printf("DDR: %v %s %s (%s)", d.live, d.name, d.remaining, d.length)
		}

		t, n := dl.NextEvent()
		log.Printf("Next event: %s %s", n, t)

		time.Sleep(100 * time.Millisecond)
	}
}

func getDataLink() *dataLink {
	url := "http://192.168.77.20/v1/datalink"

	client := &http.Client{
		Transport: &digest.Transport{
			Username: "admin",
			Password: "admin",
		},
	}
	res, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	dl := &dataLink{}

	xml.Unmarshal(body, dl)

	return dl
}

func getSwitcher() *switcher {
	url := "http://192.168.77.20/v1/dictionary?key=switcher"

	client := &http.Client{
		Transport: &digest.Transport{
			Username: "admin",
			Password: "admin",
		},
	}
	res, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	sw := &switcher{}

	xml.Unmarshal(body, sw)

	return sw
}

func getTimecode() *ddrTimecode {
	url := "http://192.168.77.20/v1/dictionary?key=ddr_timecode"

	client := &http.Client{
		Transport: &digest.Transport{
			Username: "admin",
			Password: "admin",
		},
	}
	res, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	tc := &ddrTimecode{}

	xml.Unmarshal(body, tc)

	return tc
}
