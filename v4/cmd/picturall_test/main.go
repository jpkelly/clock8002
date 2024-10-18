package main

import (
	"gitlab.com/clock-8001/clock-8001/v4/picturall"
	"log"
)

func main() {
	log.Printf("starting")
	picturall.Connect("10.100.3.114:11000")
	for {
	}

}
