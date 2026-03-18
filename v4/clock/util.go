package clock

import (
	"fmt"
	"github.com/chabad360/go-osc/osc"
	"gitlab.com/clock-8001/clock-8001/v4/oscutil"
	"image/color"
	"log"
	"net"
	"time"
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

func clockAddresses() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("Failed to get interface addresses")
		return ""
	}
	var ret string
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		if ip.IsLoopback() {
			continue
		} else if ip.To4() != nil {
			ret += fmt.Sprintf("    %v\n", ip)
		}
	}
	return ret
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
	version := fmt.Sprintf("Clock-8002 version %s", Version)
	return version
}
