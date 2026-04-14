package gp9002

import (
	"github.com/stianeikeland/go-rpio/v4"
	"time"
)

const (
	ScreenOff       = 0x00
	Screen1On       = 0x01
	Screen2On       = 0x02
	AddrIncrement   = 0x04
	AddrHold        = 0x05
	ClearScreen     = 0x06
	PowerControl    = 0x07
	WriteData       = 0x08
	ReadData        = 0x09
	Screen1LowAddr  = 0x0A
	Screen1HighAddr = 0x0B
	Screen2LowAddr  = 0x0C
	Screen2HighAddr = 0x0D
	DataLowAddr     = 0x0E
	DataHighAddr    = 0x0F
	DisplayOR       = 0x10
	DisplayXOR      = 0x11
	DisplayAND      = 0x12
	LuminanceAdj    = 0x13
	DisplayMode     = 0x14
	IntMode         = 0x15
	DisplayChar     = 0x20
	CharLocation    = 0x21
	CharSize        = 0x22
	CharBrightness  = 0x24
)

const (
	ModeOR = iota
	ModeXOR
	ModeAND
)

// Hat pin mapping
const (
	pinD0  = 16
	pinD1  = 17
	pinD2  = 18
	pinD3  = 19
	pinD4  = 20
	pinD5  = 21
	pinD6  = 22
	pinD7  = 23
	pinWR  = 24
	pinRD  = 25
	pinCS  = 26
	pinCD  = 27
	pinInt = 13
)

type GP9002 struct {
	dataPins  []rpio.Pin
	wr        rpio.Pin
	rd        rpio.Pin
	cs        rpio.Pin
	cd        rpio.Pin
	interrupt rpio.Pin
}

func Make() (*GP9002, error) {
	err := rpio.Open()

	pins := []int{pinD0, pinD1, pinD2, pinD3, pinD4, pinD5, pinD6, pinD7}

	if err != nil {
		return nil, err
	}

	ret := GP9002{}
	ret.dataPins = make([]rpio.Pin, 8)

	for i := range pins {
		ret.dataPins[i] = rpio.Pin(pins[i])
		ret.dataPins[i].Mode(rpio.Output)
		ret.dataPins[i].Write(rpio.Low)
	}

	ret.wr = rpio.Pin(pinWR)
	ret.wr.Mode(rpio.Output)
	ret.wr.Write(rpio.High)

	ret.rd = rpio.Pin(pinRD)
	ret.rd.Mode(rpio.Output)
	ret.rd.Write(rpio.High)

	ret.cs = rpio.Pin(pinCS)
	ret.cs.Mode(rpio.Output)
	ret.cs.Write(rpio.High)

	ret.cd = rpio.Pin(pinCD)
	ret.cd.Mode(rpio.Output)
	ret.cd.Write(rpio.High)

	ret.interrupt = rpio.Pin(pinInt)
	ret.interrupt.Mode(rpio.Input)

	ret.Init()

	return &ret, nil
}

func (gp *GP9002) Init() {
	gp.Greyscale(false)

	gp.Clear()

	gp.ScreenOn(1)
	gp.Increment(true)

	gp.ScreenAddr(1, 0)
}

func (gp *GP9002) WriteBitmap(bitmap []byte) {
	gp.WriteCommand(WriteData)
	gp.WriteBytes(bitmap)
}

func (gp *GP9002) ScreenMode(mode int) {
	switch mode {
	case ModeOR:
		gp.WriteCommand(DisplayOR)
	case ModeXOR:
		gp.WriteCommand(DisplayXOR)
	case ModeAND:
		gp.WriteCommand(DisplayAND)
	}
}

func (gp *GP9002) Luminance(value byte) {
	gp.WriteCommand(LuminanceAdj)
	gp.WriteBytes([]byte{value})
}

func (gp *GP9002) Greyscale(enable bool) {
	gp.WriteCommand(DisplayMode)
	if enable {
		gp.WriteBytes([]byte{0x14})
	} else {
		gp.WriteBytes([]byte{0x10})
	}
}

func (gp *GP9002) Increment(mode bool) {
	if mode {
		gp.WriteCommand(AddrIncrement)
	} else {
		gp.WriteCommand(AddrHold)
	}
}

func (gp *GP9002) Power(pwr bool) {
	gp.WriteCommand(PowerControl)
	if pwr {
		gp.WriteBytes([]byte{0})
	} else {
		gp.WriteBytes([]byte{1})
	}
}

func (gp *GP9002) Off() {
	gp.WriteCommand(ScreenOff)
}

func (gp *GP9002) ScreenOn(screen int) {
	cmd := byte(Screen1On)
	if screen == 2 {
		cmd = Screen2On
	}
	gp.WriteCommand(cmd)
}

func (gp *GP9002) ScreenAddr(screen int, addr uint16) {
	cmdLow := byte(Screen1LowAddr)
	cmdHigh := byte(Screen1HighAddr)
	if screen == 2 {
		cmdLow = Screen2LowAddr
		cmdHigh = Screen2HighAddr
	}

	low := byte(addr & 0xFF)
	high := byte(addr >> 8)

	gp.WriteCommand(cmdLow)
	gp.WriteBytes([]byte{low})

	gp.WriteCommand(cmdHigh)
	gp.WriteBytes([]byte{high})
}

func (gp *GP9002) DataLocation(addr uint16) {
	low := byte(addr & 0xFF)
	high := byte(addr >> 8)
	gp.WriteCommand(DataLowAddr)
	gp.WriteBytes([]byte{low})

	gp.WriteCommand(DataHighAddr)
	gp.WriteBytes([]byte{high})
	gp.WriteCommand(WriteData)
}

func (gp *GP9002) WriteBytes(data []byte) {
	for _, b := range data {
		gp.cd.Write(rpio.Low)
		gp.writeByte(b)
	}
}

func (gp *GP9002) WriteCommand(cmd byte) {
	gp.cd.Write(rpio.High)
	gp.writeByte(cmd)
}

func (gp *GP9002) writeByte(data byte) {
	gp.cs.Write(rpio.Low)
	time.Sleep(50 * time.Nanosecond)
	gp.wr.Write(rpio.Low)

	for i := 0; i < 8; i++ {
		if (data>>i)&1 == 1 {
			gp.dataPins[i].Write(rpio.High)
		} else {
			gp.dataPins[i].Write(rpio.Low)
		}
	}

	gp.wr.Write(rpio.High)
	time.Sleep(20 * time.Nanosecond)
	gp.cs.Write(rpio.High)
}

func (gp *GP9002) CharSize(x, y byte) {
	gp.WriteCommand(CharSize)
	gp.WriteBytes([]byte{x, y})
}

func (gp *GP9002) CharLocation(x, y byte) {
	gp.WriteCommand(CharLocation)
	gp.WriteBytes([]byte{x, 0, y})
}

func (gp *GP9002) WriteChars(text string) {
	gp.WriteCommand(DisplayChar)
	gp.WriteBytes([]byte(text))
}

func (gp *GP9002) Clear() {
	gp.WriteCommand(ClearScreen)
}
