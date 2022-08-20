package main

import (
	"fmt"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
	"log"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/experimental/conn/gpio/gpioutil"
	"periph.io/x/periph/host"
	"time"
)

const pollSleep = 300 * time.Millisecond
const denoise = 50 * time.Millisecond
const debounce = 30 * time.Millisecond

type gpioPulser struct {
	secPins  *gpioPins
	minPins  *gpioPins
	hourPins *gpioPins
	duration time.Duration
	polarity gpio.Level
}

type gpioPins struct {
	a         gpio.PinIO
	pulse     gpio.PinIO
	trigger   gpio.PinIO
	direction gpio.Level
	c         chan bool
}

var pulser *gpioPulser

func init() {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Printf("Error initializing GPIO: %v", err)
		return
	}

	p := &gpioPulser{
		duration: time.Duration(options.PulseDuration) * time.Millisecond,
		polarity: gpio.Level(options.InvertPolarity),
	}

	pulser = p
	outputModules = append(outputModules, pulser)
}

func (p *gpioPulser) Options() {
	p.duration = time.Duration(options.PulseDuration) * time.Millisecond
	p.polarity = gpio.Level(options.InvertPolarity)

	p.secPins = &gpioPins{
		a:       gpioreg.ByName(options.SecA),
		pulse:   gpioreg.ByName(options.SecPulse),
		trigger: gpioreg.ByName(options.SecTrigger),
	}

	p.minPins = &gpioPins{
		a:       gpioreg.ByName(options.MinA),
		pulse:   gpioreg.ByName(options.MinPulse),
		trigger: gpioreg.ByName(options.MinTrigger),
	}

	p.hourPins = &gpioPins{
		a:       gpioreg.ByName(options.HourA),
		pulse:   gpioreg.ByName(options.HourPulse),
		trigger: gpioreg.ByName(options.HourTrigger),
	}

	if err := p.secPins.init(p.polarity); err != nil {
		log.Printf("Error initializing second pulse pins: %v", err)
	}

	if err := p.minPins.init(p.polarity); err != nil {
		log.Printf("Error initializing minute pulse pins: %v", err)
	}

	if err := p.hourPins.init(p.polarity); err != nil {
		log.Printf("Error initializing hour pulse pins: %v", err)
	}
}

func (g *gpioPins) sendPulse(polarity gpio.Level, duration time.Duration) {
	g.pulse.Out(polarity)
	time.Sleep(duration)
	g.pulse.Out(!polarity)
	time.Sleep(duration)
	g.direction = !g.direction
	g.a.Out(g.direction)
}

func (g *gpioPins) listen(polarity gpio.Level, duration time.Duration) {
	for range g.c {
		g.sendPulse(polarity, duration)
	}
}

func (g *gpioPins) init(polarity gpio.Level) error {
	if g.a == nil {
		return fmt.Errorf("Alternating pin missing")
	}

	if g.pulse == nil {
		return fmt.Errorf("Pulsing pin missing")
	}

	if g.trigger == nil {
		return fmt.Errorf("Triggering pin missing")
	}

	g.a.Out(g.direction)
	g.pulse.Out(!polarity)
	g.c = make(chan bool)
	return nil
}

func (p *gpioPulser) secTrigger() {
	pin, _ := gpioutil.Debounce(p.secPins.trigger, denoise, debounce, gpio.RisingEdge)
	for {
		pin.WaitForEdge(-1)
		time.Sleep(debounce)
		if pin.Read() {
			log.Printf("Second trigger pin active")
			p.secPins.c <- true
		}
		time.Sleep(pollSleep)
	}
}

func (p *gpioPulser) minTrigger() {
	pin, _ := gpioutil.Debounce(p.minPins.trigger, denoise, debounce, gpio.RisingEdge)
	for {
		pin.WaitForEdge(-1)
		time.Sleep(debounce)
		if pin.Read() {
			log.Printf("Minute trigger pin active")
			p.minPins.c <- true
		}
		time.Sleep(pollSleep)
	}
}

func (p *gpioPulser) hourTrigger() {
	pin, _ := gpioutil.Debounce(p.hourPins.trigger, denoise, debounce, gpio.RisingEdge)
	for {
		pin.WaitForEdge(-1)
		time.Sleep(debounce)
		if pin.Read() {
			log.Printf("Hour trigger pin active")
			for i := 0; i < 60; i++ {
				p.minPins.c <- true
			}
			p.hourPins.c <- true
		}
		time.Sleep(pollSleep)
	}
}

// Run is executed as goroutine for output module
func (p *gpioPulser) Run() {
	secTimer := time.NewTimer(secondDuration())
	minTimer := time.NewTimer(minuteDuration())
	hourTimer := time.NewTimer(hourDuration())

	go p.secPins.listen(p.polarity, p.duration)
	go p.minPins.listen(p.polarity, p.duration)
	go p.hourPins.listen(p.polarity, p.duration)
	go p.secTrigger()
	go p.minTrigger()
	go p.hourTrigger()

	for {
		select {
		case <-secTimer.C:
			p.secPins.c <- true
			secTimer.Reset(secondDuration())
			log.Printf("Second pulse sent")
		case <-minTimer.C:
			p.minPins.c <- true
			minTimer.Reset(minuteDuration())
			log.Printf("Minute pulse sent")
		case <-hourTimer.C:
			p.hourPins.c <- true
			hourTimer.Reset(hourDuration())
			log.Printf("Hour pulse sent")
		}
	}
}

// Update processes the clock state. In this case we don't do anything
func (p *gpioPulser) Update(state *clock.State) {}

func secondDuration() time.Duration {
	return time.Until(time.Now().Truncate(time.Second).Add(time.Second))
}

func minuteDuration() time.Duration {
	return time.Until(time.Now().Truncate(time.Minute).Add(time.Minute))
}

func hourDuration() time.Duration {
	return time.Until(time.Now().Truncate(time.Hour).Add(time.Hour))
}
