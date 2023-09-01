package main

import (
	"context"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	"gitlab.com/clock-8001/clock-8001/v4/disguise"
	"log"
	"sync"
)

func main() {

	debug.Enabled = false

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	c, _ := disguise.Listen(ctx, &wg, 7400)

	for m := range c {
		log.Printf("%s", m)
	}

	cancel()
	wg.Wait()
}
