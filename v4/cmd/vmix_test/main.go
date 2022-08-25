package main

import (
	"context"
	"gitlab.com/clock-8001/clock-8001/v4/vmix"
	"log"
	"os"
	"time"
)

func main() {
	log.Printf("Starting listener")
	ctx := context.TODO()
	c := vmix.Connect(ctx, os.Args[1], 200*time.Millisecond)
	log.Printf("Waiting")
	for s := range c {
		log.Printf("%v", s)
	}
}
