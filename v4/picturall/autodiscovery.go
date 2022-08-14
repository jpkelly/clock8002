package picturall

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/desertbit/timer"
	"log"
	"net"
	"time"
)

const (
	maxDatagramSize    = 8192
	picturallDiscovery = "224.0.0.180:11009"
)

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
func Discover(timeout time.Duration) *Ident {
	log.Printf("Picturall autodiscovery starting...")
	c := make(chan *Ident)

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

	conn, err = newBroadcaster(picturallDiscovery)
	if err != nil {
		log.Fatal(err)
	}

	conn.Write([]byte("HELLO"))
	timer := timer.NewTimer(timeout)
	select {
	case p := <-c:
		return p
	case <-timer.C:
		return nil
	}
}

func autoDiscover(conn *net.UDPConn, c chan *Ident) {

	// Loop forever reading from the socket
	for {
		buffer := make([]byte, maxDatagramSize)
		numBytes, src, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal("Picturall discovery: ReadFromUDP failed:", err)
		}

		log.Printf("Picturall discovery: Got %d bytes from %v: %X", numBytes, src, buffer[:numBytes])
		reader := bytes.NewReader(buffer[:numBytes])

		picturall := Ident{}

		err = binary.Read(reader, binary.LittleEndian, &picturall)
		if err != nil {
			log.Printf("Picturall discovery: binary.Read failed: %v", err)
			continue
		}
		log.Printf("Read pictural data: %v", picturall.String())
		c <- &picturall
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
