package picturall

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

// 34 bytes for media request, client -> picturall
var picaPktMediaReq = []byte{0x50, 0x49,
	0x43, 0x41, 0x50, 0x4b, 0x54, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x04,
	0x00, 0x00, 0x00, 0x03}

// Reply to media xml request, picturall -> client
var picaPktMediaReply = []byte{0x50, 0x49,
	0x45, 0x41, 0x50, 0x4b, 0x54, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x05, 0x46, 0x3e,
	0x00, 0x00, 0x00, 0x06} // Followed by the utf16 xml

// 48 bytes for a handshake, msgLen 18 picturall -> client
var picaPktHandshake = []byte{0x50, 0x49,
	0x43, 0x41, 0x50, 0x4b, 0x54, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x12,
	0x0b, 0x00, 0x00, 0x00,
	0x33, 0x00, 0x2e, 0x00, 0x34, 0x00, 0x2e, 0x00,
	0x31, 0x00, 0x0a, 0x00, 0x00, 0x00} // "3.4.1 in UTF16, version string"

// 38 bytes, second possible handshaking packet, client -> picturall, results in status XML
var picaPktHandshake2 = []byte{0x50, 0x49,
	0x43, 0x41, 0x50, 0x4b, 0x54, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x08,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x0b}

// 670 bytes total, server info xml, picturall -> client
var picaPktServerXMLReply = []byte{0x50, 0x49,
	0x43, 0x41, 0x50, 0x4b, 0x54, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x02, 0x80,
	0x00, 0x00, 0x00, 0x01} // Followed by the <?xml version="1.1" encoding="UTF8"...><server xmlns="http://picturall.com/showtime/serverInfo/2.0" ...

// PicaPkt is a packet used by picturall on port 11001, total lenght of the header is 30 bytes?
type PicaPkt struct {
	Magic      [7]byte // "PICAPKT" no null at end
	Padding    [3]byte
	Something1 uint32 // 5 on the XML requests
	Something2 uint32 // zero
	Something3 uint32 // 4 on the xml requests
	Something4 uint32 // zero
	Length     uint32
}

// PicaPktMagic is the magic identifier at the start of all packets
func PicaPktMagic() []byte {
	return []byte{0x50, 0x49, 0x43, 0x41, 0x50, 0x4b, 0x54}
}

func (p *PicaPkt) String() string {
	return fmt.Sprintf("%s: 1: %d 2: %d 3: %d 4: %d length: %d", p.Magic, p.Something1, p.Something2, p.Something3, p.Something4, p.Length)
}

// MediaXML returns true if this is a packet that should contain the media collection xml
func (p *PicaPkt) MediaXML() bool {
	return p.Something1 == 5 && p.Something2 == 0 && p.Something3 == 4 && p.Something4 == 0
}

// ParsePicaPkt parses a byte slice into a PicaPkt header struct
func ParsePicaPkt(data []byte) (*PicaPkt, error) {
	header := &PicaPkt{}
	if bytes.Equal(data[:7], PicaPktMagic()) && len(data) >= 30 {
		err := binary.Read(bytes.NewReader(data), binary.BigEndian, header)
		if err != nil {
			return nil, fmt.Errorf("Binary read for header failed: %w", err)
		}
		return header, nil

	}
	return nil, fmt.Errorf("Start magic missmatch, expected %s got %s", PicaPktMagic(), data[:7])
}

// FetchMedia connects to a picturall and tries to fetch the media collection info via port 11001
func FetchMedia(ip string) (*MediaCollections, error) {
	addr := fmt.Sprintf("%s:11001", ip)
	c := make(chan *MediaCollections)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	go xmlListener(conn, c)

	time.Sleep(50 * time.Millisecond)
	_, err = conn.Write(picaPktMediaReq)
	if err != nil {
		log.Fatalf("Failed to write: %v", err)
	}

	timer := time.NewTimer(time.Second)
	select {
	case mc, ok := <-c:
		if ok {
			return mc, nil
		}
		return nil, fmt.Errorf("Listener closed the channel")
	case <-timer.C:
		return nil, fmt.Errorf("Timeout waiting for the XML data")
	}
}

func xmlListener(conn net.Conn, c chan *MediaCollections) {
	buffer := bytes.NewBuffer(nil)
	reader := bufio.NewReader(buffer)
	read := make([]byte, 30)

	var header *PicaPkt

	for {
		// Calculate the size we want to read
		buffSize(header, read, buffer.Len())

		nRead, err := conn.Read(read)
		if err != nil {
			log.Printf("Read failed: %v", err)
			close(c)
			return
		}
		buffer.Write(read[:nRead])

		if header == nil && buffer.Len() >= 30 {
			start, err := reader.Peek(30)
			if err != nil {
				conn.Close()
				log.Printf("Failed to peek")
				close(c)
				return
			}
			header, err = ParsePicaPkt(start)
			if err == nil {
				// Header found
				reader.Discard(30)
				buffer.Truncate(buffer.Len())
			} else {
				reader.Discard(1)
			}
		} else if header != nil {
			// Have header, waiting for payload
			if buffer.Len() >= int(header.Length) {
				// Have the payload
				var n uint32
				var other []byte

				if header.Length >= 4 {
					err := binary.Read(reader, binary.LittleEndian, &n)
					if err != nil {
						log.Printf("Error parsing payload 1: %v", err)
					}
				}

				if header.Length > 4 {
					other = make([]byte, header.Length-4)
					err := binary.Read(reader, binary.LittleEndian, &other)
					if err != nil {
						log.Printf("Error parsing payload 2: %v", err)
					}
				}

				if header.MediaXML() {
					mc, err := parseMediaCollections(other)

					if err != nil {
						log.Printf("Error parsing media collections %v", err)
						continue
					}
					c <- mc
					return
				}
				header = nil
				buffer.Truncate(buffer.Len())
			}
		}
	}
}

func buffSize(header *PicaPkt, read []byte, pending int) {
	if header == nil {
		read = make([]byte, 30)
	} else {
		missing := int(header.Length) - pending
		if missing > 1460 {
			missing = 1460
		}
		if len(read) != missing {
			read = make([]byte, missing)
		}
	}
}
