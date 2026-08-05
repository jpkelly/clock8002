package unicorn

import (
	"errors"
	"image/color"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

const SOH = 0x72
const SPI_speed = 9000000 * physic.Hertz

type unicorn struct {
	c      spi.Conn
	buffer []byte
}

func Init() (*unicorn, error) {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		return nil, err
	}

	// Use spireg SPI port registry to find the first available SPI bus.
	p, err := spireg.Open("")
	if err != nil {
		return nil, err
	}

	// Convert the spi.Port into a spi.Conn so it can be used for communication.
	c, err := p.Connect(SPI_speed, spi.Mode0, 8)
	if err != nil {
		return nil, err
	}

	uni := &unicorn{
		c:      c,
		buffer: make([]byte, 256*3),
	}
	return uni, nil
}

// Close turns the leds off and closes the SPI connection
func (u *unicorn) Close() error {
	data := make([]byte, 256*3+1)
	data[0] = SOH
	return u.c.Tx(data, nil)
}

func (u *unicorn) Update() error {
	data := make([]byte, 1)
	data[0] = SOH
	data = append(data, u.buffer...)
	return u.c.Tx(data, nil)
}

func (u *unicorn) SetPixel(x, y int, c color.RGBA) error {
	if x < 0 || x > 15 || y < 0 || y > 15 {
		return errors.New("Index out of bounds")
	}

	offset := (y*16 + x) * 3

	u.buffer[offset] = byte(c.R)
	u.buffer[offset+1] = byte(c.G)
	u.buffer[offset+2] = byte(c.B)

	return nil
}

func (u *unicorn) Fill(c color.RGBA) {
	for i := 0; i < 256; i++ {
		u.buffer[i*3] = c.R
		u.buffer[(i*3)+1] = c.G
		u.buffer[(i*3)+2] = c.B
	}
}
