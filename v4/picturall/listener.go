package picturall

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// Connect connects to a Picturall media server.
// ip is a combination of host and port
func Connect(ctx context.Context, ip string) chan *Media {
	mainChan := make(chan *Media)
	s := &State{
		sources: make(map[string]*Source),
		enums:   make(map[int]string),
		ctx:     ctx,
	}
	go s.openConnection(ip, mainChan)

	return mainChan
}

func (state *State) openConnection(ip string, mainChan chan *Media) {
	defer log.Printf("Picturall listener exiting on request...")
	var err error

	for {
		state.reset()
		state.mediaChan = make(chan *Media)

		select {
		case <-state.ctx.Done():
			return
		default:
		}

		// Autodiscover if needed
		addr := resolveAddr(ip)
		if addr == "" {
			log.Printf("Picturall autodiscovery failed, retrying...")
			time.Sleep(time.Second)
			continue
		}

		parts := strings.Split(addr, ":")
		state.ip = parts[0]

		log.Printf("Starting picturall listener for %s", addr)
		state.conn, err = net.Dial("tcp", addr)
		if err != nil {
			log.Printf("Error connecting to picturall %v", err)
			time.Sleep(time.Second)
			continue
		}

		go state.listen()

		err = state.initConnection()
		if err != nil {
			log.Printf("Picturall connection init error: %v", err)
			close(state.mediaChan)
			continue
		}

		for end := false; !end; {
			select {
			case p, ok := <-state.mediaChan:
				if ok {
					mainChan <- p
				} else {
					log.Printf("Picturall listener connection lost, retrying...")
					end = true
				}
			case <-state.ctx.Done():
				state.conn.Close()
				return
			}
		}
		time.Sleep(time.Second)
	}
}

func resolveAddr(ip string) (addr string) {
	if parts := strings.Split(ip, ":"); len(parts[0]) == 0 {
		// Autodiscovery time
		p, found := Discover(time.Second)
		if found {
			addr = fmt.Sprintf("%s:%s", p, parts[1])
		} else {
			// Didn't find anything...
			return ""
		}
	} else {
		addr = ip
	}
	log.Printf("Picturall address resolved to: '%s' %X", addr, []byte(addr))
	return
}

func (state *State) listen() {
	defer close(state.mediaChan)

	var buffer bytes.Buffer
	var logFile *bufio.Writer
	c := state.mediaChan

	log.Printf("Picturall listener starting...")

	reader := bufio.NewReader(state.conn)

	if DumpTraffic {
		logName := fmt.Sprintf("picturall_log_%s.txt", time.Now().Format("2006-01-02T15:04:05"))
		f, err := os.OpenFile(logName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		log.Printf("Picturall log file open: %s", logName)
		logFile = bufio.NewWriter(f)
		defer f.Close()
	}

	for {
		msg, isPrefix, err := reader.ReadLine()
		if err != nil {
			log.Printf("Picturall listen() read error: %v", err)
			break
		}
		buffer.Write(msg)
		if isPrefix {
			continue
		}

		str := buffer.String()
		buffer.Reset()
		p := state.parseMsg(str)

		if DumpTraffic {
			logFile.WriteString(str)
			logFile.WriteString("\n")
		}

		if p != nil {
			if s := p.ParseSource(state); s != nil {
				state.sources[s.Name] = s
				if m := s.Media; m != nil {
					debug.Printf("Picturall got media: %s", m.String())
					c <- m
				}
			} else if m := p.ParseMedia(state); m != nil {
				// We got a media playback message, but can't really
				// do too much with it alone, as it lacks information on
				// media play mode.
				//
				// We need to request the source state

				if source, ok := state.enums[p.Source]; ok {
					// We have the source string representation from the enum list
					state.getStatus(source)
				} else {
					debug.Printf("Picturall got partial media: %s", m.String())
					c <- m
				}

			} else if p.Type == EnumObjects {
				// fmt.Printf("-> Parsed: %s\n", p.String())
				state.enums = p.ParseEnum()
			} else if p.collectionMsg() {
				p.parseCollection(state)
			} else {
				//fmt.Printf("Unkown: %s\n", p.String())
			}
		} else if str != "" {
			fmt.Printf("Picturall: failed to parse: %s\n", str)
		}
	}
}
