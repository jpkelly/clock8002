package tricaster

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/icholy/digest"
	"io"
	"log"
	"net/http"
	"time"
)

var state struct {
	username string
	password string
	address  string
	ctx      context.Context
	interval time.Duration
	c        chan *Status
}

const (
	datalinkPath = "/v1/datalink"
	switcherPath = "/v1/dictionary?key=switcher"
	timecodePath = "/v1/dictionary?key=ddr_timecode"
)

// Status is the complete tricaster state for the clock
type Status struct {
	NextEventName string
	NextEventTime string
	DDR           []DDR
}

// DDR contains the state of one tricaster video player
type DDR struct {
	Name              string
	Left              time.Duration
	Length            time.Duration
	PlaylistRemaining time.Duration
	PlaylistLength    time.Duration
	ClipIndex         int
	Clips             int
	Live              bool
	Preview           bool
	Static            bool
}

// Remaining is the time remaining on the clip or playlist, if active
func (d *DDR) Remaining() time.Duration {
	if d.ClipIndex == 0 {
		return d.Left
	}
	return d.PlaylistRemaining
}

// Duration is the duration of the clip or playlist, if active
func (d *DDR) Duration() time.Duration {
	if d.ClipIndex == 0 {
		return d.Length
	}
	return d.PlaylistRemaining
}

// Play is always true for tricaster
func (d *DDR) Play() bool {
	return true
}

// Loop is always false for tricaster
func (d *DDR) Loop() bool {
	return false
}

// Record is always false for tricaster
func (d *DDR) Record() bool {
	return false
}

// MediaName is the name of the clip
func (d *DDR) MediaName() string {
	if d.ClipIndex == 0 {
		return d.Name
	}
	return fmt.Sprintf("%d/%d %s", d.ClipIndex, d.Clips, d.Name)
}

// IconOverride is always none for tricaster
func (d *DDR) IconOverride() string {
	return ""
}

func (s *Status) String() string {
	ret := fmt.Sprintf("Tricaster state:\n -> Next event: %s\n -> In: %s\n", s.NextEventName, s.NextEventTime)
	ret += fmt.Sprintf(" -> DDRs: %d\n", len(s.DDR))
	for i, d := range s.DDR {
		ret += fmt.Sprintf(" -> DDR%d Live: %v Name: %s Remaining: %s (%s)\n", i+1, d.Live, d.Name, d.Left, d.Length)
		ret += fmt.Sprintf("  -> Playlist: %d/%d Remaining: %s (%s)\n", d.ClipIndex, d.Clips, d.PlaylistRemaining, d.PlaylistLength)
	}
	return ret
}

// Connect starts monitoring a tricaster mixer for DDR video state
func Connect(ctx context.Context, addr string, username string, password string, interval time.Duration) chan *Status {
	state.c = make(chan *Status)
	state.username = username
	state.password = password
	state.address = addr
	state.interval = interval
	state.ctx = ctx

	log.Printf("Tricaster: Connecting to: %s", state.address)
	go listen()

	return state.c
}

func listen() {
	t := time.NewTicker(state.interval)

	for {
		select {
		case <-t.C:
			tc := getTimecode()
			if tc == nil {
				continue
			}
			sw := getSwitcher()
			if sw == nil {
				continue
			}
			dl := getDataLink()
			if dl == nil {
				continue
			}

			nextTime, nextName := dl.nextEvent()
			s := Status{
				NextEventName: nextName,
				NextEventTime: nextTime,
				DDR:           make([]DDR, 0),
			}

			for i := 0; i < 4; i++ {
				n := fmt.Sprintf("DDR%d", i+1)
				ddrtc := tc.ddr(i + 1)
				if ddrtc == nil {
					continue
				}
				plLen := ddrtc.PlaylistRemaining + ddrtc.PlaylistElapsed
				ddr := DDR{
					Live:              sw.live(n),
					Preview:           sw.preview(n),
					Name:              dl.alias(i + 1),
					Left:              time.Duration(ddrtc.Remaining) * time.Second,
					Length:            time.Duration(ddrtc.Duration) * time.Second,
					ClipIndex:         ddrtc.ClipIndex,
					Clips:             ddrtc.Clips,
					PlaylistRemaining: time.Duration(ddrtc.PlaylistRemaining) * time.Second,
					PlaylistLength:    time.Duration(plLen) * time.Second,
					Static:            ddrtc.Remaining < 0,
				}
				s.DDR = append(s.DDR, ddr)
			}
			state.c <- &s
		case <-state.ctx.Done():
			close(state.c)
			log.Printf("Tricaster: Listener exiting on context loss.")
			return
		}
	}
}

func getDataLink() *dataLink {
	url := "http://" + state.address + datalinkPath
	body, err := get(url)
	if err != nil {
		return nil
	}
	dl := &dataLink{}
	xml.Unmarshal(body, dl)

	return dl
}

func getSwitcher() *switcher {
	url := "http://" + state.address + switcherPath
	body, err := get(url)
	if err != nil {
		return nil
	}
	sw := &switcher{}
	xml.Unmarshal(body, sw)

	return sw
}

func getTimecode() *ddrTimecode {
	url := "http://" + state.address + timecodePath
	body, err := get(url)
	if err != nil {
		return nil
	}
	tc := &ddrTimecode{}
	xml.Unmarshal(body, tc)

	return tc
}

// Need to do http digest auth :(
func get(url string) ([]byte, error) {
	client := &http.Client{
		Transport: &digest.Transport{
			Username: state.username,
			Password: state.password,
		},
	}
	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	return body, err
}
