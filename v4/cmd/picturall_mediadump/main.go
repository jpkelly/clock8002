package main

import (
	"gitlab.com/clock-8001/clock-8001/v4/picturall"
	"log"
	"time"
)

func main() {
	log.Printf("Attempting to connect to picturall and dump the media collection xml...")

	ip, found := picturall.Discover(time.Second)
	if !found {
		log.Fatalf("Autodiscovery failed")
	}

	log.Printf("Found picturall at %s", ip)
	mc, err := picturall.FetchMedia(ip)
	if err != nil {
		log.Fatalf("Error fetching XML: %v", err)
	}
	log.Printf("Got XML:\n%s", mc.String())
}
