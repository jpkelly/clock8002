package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gitlab.com/clock-8001/clock-8001/v4/picturall"
	"log"
	"net"
	"time"
)

const (
	maxDatagramSize    = 8192
	picturallDiscovery = "224.0.0.180:11009"
)

type picturallIdent struct {
	Magic     [16]byte // "PICTURALL SERVER" (no null termination)
	BcVersion uint32   // bcast protocol version network byte order
	IP        [32]byte // null terminated IP-address
	Name      [32]byte // null terminated host name
	Version   [32]byte // null terminated server version string
}

func (p *picturallIdent) String() string {
	return fmt.Sprintf("Magic: %s BcVersion: %d Ip: %s Name: %s Version %s", p.Magic, p.BcVersion, p.IP, p.Name, p.Version)
}

func main() {
	p := picturall.Discover(time.Second)
	log.Printf("Got: %s", p.String())
}

func oldMain() {
	go listen(picturallDiscovery)

	conn, err := newBroadcaster(picturallDiscovery)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn.Write([]byte("HELLO"))
		time.Sleep(1 * time.Second)
	}

}

func listen(address string) {
	// Parse the string address
	addr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		log.Fatal(err)
	}

	// Open up a connection
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		log.Fatal(err)
	}

	conn.SetReadBuffer(maxDatagramSize)

	// Loop forever reading from the socket
	for {
		buffer := make([]byte, maxDatagramSize)
		numBytes, src, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal("ReadFromUDP failed:", err)
		}

		log.Printf("Got %d bytes from %v: %X", numBytes, src, buffer[:numBytes])
		reader := bytes.NewReader(buffer[:numBytes])

		picturall := picturallIdent{}

		err = binary.Read(reader, binary.LittleEndian, &picturall)
		if err != nil {
			log.Printf("binary.Read failed: %v", err)
		}
		log.Printf("Read pictural data: %v", picturall.String())
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
