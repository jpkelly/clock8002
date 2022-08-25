package vmix

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	api = "/api"
)

// State represents the vMix video playback state
type State struct {
	External    bool
	Recording   bool
	MultiCorder bool
	Playlist    bool
	Videos      []*Video
}

// Video represents a media file being played on vMix
type Video struct {
	Name     string
	Looping  bool
	Playing  bool
	Live     bool
	PGM      bool
	Overlay  int
	Length   time.Duration
	Head     time.Duration
	Progress float64
}

func (v *Video) String() string {
	return fmt.Sprintf("Video: %v (%v) Remaining: %2.2f%% PGM: %v Overlay: %v Live %v Playing: %v Looping: %v Name: %s",
		v.Head, v.Length, v.Progress*100, v.PGM, v.Overlay, v.Live, v.Playing, v.Looping, v.Name)
}

func (s *State) String() string {
	ret := fmt.Sprintf("vMix State: Recording: %v External %v MultiCorder: %v Playlist: %v\n",
		s.Recording, s.External, s.MultiCorder, s.Playlist)
	for _, v := range s.Videos {
		ret += fmt.Sprintf(" -> %v\n", v)
	}
	return ret
}

// Connect connects to a vMix server and polls for the Status
func Connect(ctx context.Context, addr string, timeout time.Duration) chan *State {
	log.Printf("vMix: Connecting to: %s", addr)

	url := fmt.Sprintf("http://%s%s", addr, api)

	c := make(chan *State)
	go listen(ctx, url, timeout, c)
	return c
}

func listen(ctx context.Context, url string, timeout time.Duration, c chan *State) {
	t := time.NewTicker(timeout)
	for {
		select {
		case <-t.C:
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("vMix: Error getting XML: %v", err)
				continue
			}
			defer resp.Body.Close()

			bytes, _ := io.ReadAll(resp.Body)
			s := &status{}
			err = xml.Unmarshal(bytes, s)
			if err != nil {
				log.Printf("vMix: Error parsing XML: %v", err)
				continue
			}
			state := s.State()
			c <- state
		case <-ctx.Done():
			close(c)
			return
		}
	}
}
