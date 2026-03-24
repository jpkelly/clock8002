package clock

import (
	"fmt"
	"image/color"
	"log"
	"runtime/debug"
	"time"

	"github.com/chabad360/go-osc/osc"
	"gitlab.com/clock-8001/clock-8001/v4/oscutil"
)

/*
 * Generic utility functions that aren't part of any struct
 */

func oscListenerSetup(listenAddr string) *Server {
	oscDispatcher := oscutil.NewRegexpDispatcher()
	oscServer := osc.Server{
		Addr:    listenAddr,
		Handler: oscDispatcher.Dispatch,
	}

	// clock.Server
	var server = Server{
		listeners:  make(map[chan Message]struct{}),
		Debug:      false,
		dispatcher: oscDispatcher,
		osc:        &oscServer,
	}

	log.Printf("OSC: listening on %v", oscServer.Addr)
	return &server
}

func formatDuration(diff time.Duration) string {
	hours, minutes, seconds := splatDuration(diff)
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func splatDuration(diff time.Duration) (hours, minutes, seconds int) {
	hours = int(diff.Truncate(time.Hour).Hours())
	minutes = int(diff.Truncate(time.Minute).Minutes()) - (hours * 60)
	seconds = int(diff.Truncate(time.Second).Seconds()) - (((hours * 60) + minutes) * 60)
	return
}

// parseColor parses a string "#XXX or #XXXXXX to a color.RGBA"
func parseColor(s string) (c *color.RGBA, err error) {
	c = &color.RGBA{
		A: 255,
	}
	switch len(s) {
	case 7:
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	case 4:
		_, err = fmt.Sscanf(s, "#%1x%1x%1x", &c.R, &c.G, &c.B)
		// Double the hex digits:
		c.R *= 17
		c.G *= 17
		c.B *= 17
	default:
		err = fmt.Errorf("parseColor(): invalid length, must be 7 or 4: %v", s)
	}
	return
}

// VersionInfo returns the clock engine version and git commit as a string
func VersionInfo() string {
	version := Version
	if gitTag != "" && gitTag != "v0.0.1" && gitTag != "Unknown" {
		version = gitTag
	}

	commit := gitCommit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	if commit == "" || commit == "Unknown" {
		commit = "unknown"
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				commit = s.Value[:7]
				break
			}
		}
	}
	return fmt.Sprintf("Clock-8002 version %s (%s)", version, commit)
}
