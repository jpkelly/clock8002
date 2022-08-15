package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"gitlab.com/clock-8001/clock-8001/v4/picturall"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
)

var picaPktMediaReq = []byte{0x50, 0x49,
	0x43, 0x41, 0x50, 0x4b, 0x54, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x04,
	0x00, 0x00, 0x00, 0x03}

func main() {
	log.Printf("Attempting to connect to picturall and dump the media collection xml...")

	ip, found := picturall.Discover(time.Second)
	if !found {
		log.Fatalf("Autodiscovery failed")
	}

	log.Printf("Found picturall at %s", ip)

	logName := fmt.Sprintf("picturall_raw_%s.bin", time.Now().Format("2006-01-02T15:04:05"))
	f, err := os.OpenFile(logName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.Printf("Picturall log file open: %s", logName)
	logFile := bufio.NewWriter(f)
	defer f.Close()

	addr := fmt.Sprintf("%s:11001", ip)

	conn, err := net.Dial("tcp", addr)

	go listener(conn, logFile)

	time.Sleep(50 * time.Millisecond)

	for {
		_, err := conn.Write(picaPktMediaReq)
		if err != nil {
			log.Fatalf("Failed to write: %v", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func listener(conn net.Conn, logFile *bufio.Writer) {
	buffer := bytes.NewBuffer(nil)
	reader := bufio.NewReader(buffer)
	read := make([]byte, 1024)

	var header *picturall.PicaPkt

	// Make an tranformer that converts MS-Win default to UTF8:
	win16be := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	// Make a transformer that is like win16be, but abides by BOM:
	utf16bom := unicode.BOMOverride(win16be.NewDecoder())

	xmlName := fmt.Sprintf("picturall_xml_%s.txt", time.Now().Format("2006-01-02T15:04:05"))
	f, err := os.OpenFile(xmlName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.Printf("Picturall xml log file open: %s", xmlName)
	xmlFile := bufio.NewWriter(f)
	defer f.Close()

	for {
		if header == nil {
			read = make([]byte, 30)
		} else {
			missing := int(header.Length) - buffer.Len()
			if missing > 1460 {
				missing = 1460
			}
			read = make([]byte, missing)
		}

		log.Printf("Trying to read %d bytes...", len(read))
		nRead, err := conn.Read(read)
		if err != nil {
			log.Printf("Read failed: %v", err)
			conn.Close()
			return
		}
		buffer.Write(read[:nRead])
		logFile.Write(read[:nRead])

		if header == nil && buffer.Len() > 7 {
			log.Printf("Checking for start")
			start, err := reader.Peek(7)
			if err != nil {
				conn.Close()
				log.Printf("Failed to peek")
				return
			}
			if bytes.Equal(start, picturall.PicaPktMagic()) && reader.Buffered() >= 30 {
				// Start found
				header = &picturall.PicaPkt{}
				err := binary.Read(reader, binary.BigEndian, header)
				if err != nil {
					log.Printf("Binary read for header failed: %v", err)
					header = nil
					continue
				}
				log.Printf("Got packet: %s", header.String())
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
					// xml, err := DecodeUTF16(other)

					// Make a Reader that uses utf16bom:
					unicodeReader := transform.NewReader(bytes.NewReader(other), utf16bom)

					// decode and print:
					x, err := ioutil.ReadAll(unicodeReader)

					if err != nil {
						log.Printf("Error decoding UTF16 XML %v", err)
					} else {
						log.Printf("Got XML")
						xmlFile.Write(x[:len(x)-1])
						xmlFile.Flush()

						mc := picturall.MediaCollections{}
						xml.Unmarshal(x[:len(x)-1], &mc)

						log.Printf("Unmarshal xml: collections: %d", len(mc.Collections))
						for i, c := range mc.Collections {
							if len(c.Medias) == 0 {
								continue
							}
							log.Printf(" -> Collection %d: %s", i, c.Name)
							for _, m := range c.Medias {
								log.Printf("   -> %d Media: %s type: %s play mode: %d", m.Index, m.Name, m.Type, m.PlayMode)
							}
						}

						os.Exit(0)
					}

				} else {
					log.Printf(" -> Payload: %d %X", n, other)
				}
				header = nil
				buffer.Truncate(buffer.Len())
			}
		}
	}
}
