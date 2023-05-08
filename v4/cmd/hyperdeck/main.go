package main

import (
	"context"
	"gitlab.com/clock-8001/clock-8001/v4/hyperdeck"
)

func main() {
	ip := "192.168.2.109:9993"

	ctx, cancel := context.WithCancel(context.Background())

	c, err := hyperdeck.Listen(ctx, ip, true)
	if err != nil {
		panic(err)
	}

	for range c {

	}

	cancel()
}
