package picturall

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/desertbit/timer"
	"github.com/jpkelly/clock8002/app/debug"
	"log"
	"net"
	"time"
)

const (
	maxDatagramSize    = 8192
	picturallDiscovery = "224.0.0.180:11009"
)

var magic = [16]byte{0x50, 0x49, 0x43, 0x54, 0x55, 0x52, 0x41, 0x4c, 0x4c, 0x20, 0x53, 0x45, 0x52, 0x56, 0x45, 0x52}

// Ident is the autodiscovery reply message sent by Picturall media servers
type Ident struct {
	Magic     [16]byte // "PICTURALL SERVER" (no null termination)
	BcVersion uint32   // bcast protocol version network byte order
	IP        [32]byte // null terminated IP-address
	Name      [32]byte // null terminated host name
	Version   [32]byte // null terminated server version string
}

func (p *Ident) String() string {
	return fmt.Sprintf("Magic: %s BcVersion: %d Ip: %s Name: %s Version %s", p.Magic, p.BcVersion, p.IP, p.Name, p.Version)
}

// Discover performs autodiscovery for Picturalls via multicast.
// Returns the info for the first unit discovered or nil for timeout
func Discover(timeout time.Duration) (string, bool) {
	log.Printf("Picturall autodiscovery starting...")
	c := make(chan string)

	addr, err := net.ResolveUDPAddr("udp4", picturallDiscovery)
	if err != nil {
		log.Fatal(err)
	}

	// Open up a connection
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		log.Fatal(err)
	}

	conn.SetReadBuffer(maxDatagramSize)
	go autoDiscover(conn, c)

	bc, err := newBroadcaster(picturallDiscovery)
	if err != nil {
		log.Printf("Picturall error opening port: %v", err)
	}

	bc.Write([]byte("HELLO"))
	timer := timer.NewTimer(timeout)
	select {
	case p := <-c:
		conn.Close()
		bc.Close()
		return p, true
	case <-timer.C:
		conn.Close()
		bc.Close()
		return "", false
	}
}

func autoDiscover(conn *net.UDPConn, c chan string) {

	// Loop forever reading from the socket
	for {
		buffer := make([]byte, maxDatagramSize)
		numBytes, src, err := conn.ReadFromUDP(buffer)
		if err != nil {
			debug.Printf("Picturall discovery: ReadFromUDP failed: %v", err)
			close(c)
			return
		}

		debug.Printf("Picturall discovery: Got %d bytes from %v: %X", numBytes, src, buffer[:numBytes])

		if numBytes == 5 {
			continue
		}

		reader := bytes.NewReader(buffer[:numBytes])

		picturall := Ident{}

		err = binary.Read(reader, binary.LittleEndian, &picturall)
		if err != nil {
			debug.Printf("Picturall discovery: binary.Read failed: %v", err)
			continue
		}

		if picturall.Magic != magic {
			log.Printf("Picturall discovery: wrong magic identifier")
		}

		log.Printf("Read pictural data: %v", picturall.String())
		c <- src.IP.String()
		close(c)
		return
	}
}

func newBroadcaster(address string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, err
	}

	return conn, nil

}
