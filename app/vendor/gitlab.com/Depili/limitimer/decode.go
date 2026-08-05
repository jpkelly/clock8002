package limitimer

import (
	"fmt"
	"github.com/sigurn/crc16"
	"log"
)

const (
	seeking    = 0
	startFound = iota
	checksum   = iota
	endFound   = iota
)

const (
	msgStart      = 0x81
	checksumStart = 0x83
	msgEnd        = 0xFF
)

// Message types for different known limitimer packets
const (
	StatusMsg = 0x00 // Message type for limitimer status messages
	PingMsg   = 0x10 // Message type for the short unknown message, assumed as a ping for slave controllers
)

// Decoder contains the decode logic and state for parsing limitimer communications
type Decoder struct {
	buff          [256]byte
	decodeState   int
	checksumBytes int
	msgLen        int
}

func (d *Decoder) reset() {
	d.decodeState = seeking
	d.msgLen = 0
	d.checksumBytes = 0
}

// Feed is used to input bytes to the limitimer decode logic. Returns an array of limitimer message frames if any are found
func (d *Decoder) Feed(in []byte) [][]byte {

	ret := make([][]byte, 0)

	for _, data := range in {

		if d.decodeState == seeking && data == msgStart {
			d.reset()
			d.decodeState = startFound
		} else if d.decodeState == seeking {
			continue
		} else if d.decodeState == startFound && data == checksumStart {
			d.decodeState = checksum
		} else if d.decodeState == checksum && d.checksumBytes < 2 {
			// 2 byte checksum
			d.checksumBytes++
		} else if d.decodeState == checksum && d.checksumBytes == 2 && data == msgEnd {
			// Complete message
			d.decodeState = endFound
		} else if d.checksumBytes == 2 {
			// checksum error
			log.Printf("Limitimer decode - Checksum end marker missing!")
			d.reset()
			continue
		} else if data&0x80 != 0 {
			if data == msgStart {
				log.Printf("Limitimer decode - Start marker in the middle of message, restarting decode")
				d.reset()
				d.decodeState = startFound
			} else {
				log.Printf("Limitimer decode - Unknown control byte %0.2X in message: %x", data, d.buff[0:d.msgLen])
				log.Printf("Limitimer decode - Discarding for now...")
				d.reset()
				continue
			}
		} else if d.msgLen >= len(d.buff) {
			// Too many bytes received without valid message
			d.reset()
			continue
		}

		d.buff[d.msgLen] = data
		d.msgLen++

		if d.decodeState == endFound {
			// Process a complete message

			if err := validate(d.buff[0:d.msgLen]); err == nil {
				ret = append(ret, d.buff[0:d.msgLen])
			} else {
				log.Printf("Limitimer: Message parsing error: %v", err)
				log.Printf("Limitimer: Buffer contents: %x", d.buff)
			}
			d.reset()
		}
	}
	return ret
}

// Type extracts the message type from a byte array containing a limitimer message
func Type(msg []byte) byte {
	return msg[1]
}

func validate(msg []byte) error {
	if msg[0] != msgStart {
		return fmt.Errorf("Message start marker missing")
	}

	if msg[len(msg)-1] != msgEnd {
		return fmt.Errorf("Message end marker missing")
	}

	if msg[len(msg)-4] != checksumStart {
		return fmt.Errorf("Message lacks checksum start marker")
	}

	crcLen := len(msg) - 3

	crc := crc16.Checksum(msg[:crcLen], crcModbus)
	msgChecksum := uint16(msg[crcLen])<<8 + uint16(msg[crcLen+1])

	if msgChecksum != crc && msgChecksum != 0 {
		return fmt.Errorf("Message checksum error, message: %x calculated: %x", msgChecksum, crc)
	}

	return nil
}
