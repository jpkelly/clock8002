package hyperdeck

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"time"
)

/*
TODO:
- Timeout on connection activity?
- Simulator?

*/

// 00:07:25:53 or 00:07:25;53 depending on the field...
var timeRegexp = regexp.MustCompile(`^(\d\d):(\d\d):(\d\d)[;:](\d\d)$`)

// "208 transport info:" for a response with parameters
var responseRegexp = regexp.MustCompile(`^((?:\d){3}) ((?:[a-z ])*):$`)

// "display timecode: 00:00:26:37"
var paramRegexp = regexp.MustCompile(`^(.+): (.*)$`)

// Capture0004.mov 02:02:03;48 00:00:05;43
var clipRegexp = regexp.MustCompile(`^((?:\\.|.)+) ((?:\d\d[:;]){3}(?:\d\d)) ((?:\d\d[:;]){3}(?:\d\d))$`)

const pingPeriod = 2 * time.Second
const pollPeriod = 250 * time.Millisecond
const relayAddr = "0.0.0.0:9993"

type state struct {
	ctx             context.Context
	clips           []*clip
	conn            net.Conn
	mediaChan       chan *Media
	rxChan          chan string
	txChan          chan string
	clipID          int
	timecode        time.Duration
	displayTimecode time.Duration
	loop            bool
	singleClip      bool
	speed           int
	slotID          int
	status          string // stopped, preview, record, play
	recordingTime   time.Duration
	playlistLength  time.Duration
	listeners       []chan string
	addListener     chan chan string
	removeListener  chan (<-chan string)
	listenConn      net.Listener
	banner          string
	pingTicker      *time.Ticker
}

type clip struct {
	name          string
	length        time.Duration
	playlistStart time.Duration
}

// Media is the transport data to outside world
type Media struct {
	name        string
	remaining   time.Duration
	length      time.Duration
	loop        bool
	pause       bool
	record      bool
	singleClip  bool
	currentClip int
	totalClips  int
}

type message struct {
	code   int
	name   string
	params map[string]string
}

// Listen open a connection to a hyperdeck and listen for state changes.
// Optionally listen on port 9993 and relay data to connecting clients to and from
// the hyperdeck
func Listen(ctx context.Context, ip string, relay bool) (chan *Media, error) {
	var err error
	log.Printf("Hyperdeck: listen() %s", ip)

	s := &state{
		ctx:            ctx,
		clips:          make([]*clip, 0),
		mediaChan:      make(chan *Media),
		rxChan:         make(chan string),
		txChan:         make(chan string),
		listeners:      make([]chan string, 0),
		addListener:    make(chan chan string),
		removeListener: make(chan (<-chan string)),
		pingTicker:     time.NewTicker(pingPeriod),
	}

	s.conn, err = net.Dial("tcp", ip)
	if err != nil {
		return nil, err
	}

	go s.sender()
	go s.listen()
	if relay {
		go s.relay()
	}

	return s.mediaChan, nil
}

func (s *state) listen() {
	var buffer bytes.Buffer
	reader := bufio.NewReader(s.conn)
	state := "start"

	m := &message{
		params: make(map[string]string),
	}

	for {
		msg, isPrefix, err := reader.ReadLine()
		if err != nil {
			log.Printf("Hyperdeck: state.listen() err: %v", err)
			close(s.rxChan)
			close(s.mediaChan)
			s.conn.Close()
			return
		}
		buffer.Write(msg)

		if isPrefix {
			// Did not get a complete line
			continue
		}

		str := buffer.String()
		buffer.Reset()
		s.rxChan <- (str + "\r\n")

		if str == "" {
			// Got empty line, end of message parameters
			state = "complete"
		}

		switch state {
		case "start":
			if p := responseRegexp.FindStringSubmatch(str); len(p) == 3 {
				code, err := strconv.Atoi(p[1])
				if err != nil {
					// This should newer happen due to the regexp
					log.Fatalf("hyperdeck state.listen() fatal conversion: %v", err)
				}

				state = "params"

				m = &message{
					code:   code,
					name:   p[2],
					params: make(map[string]string),
				}
			}
		case "params":
			if p := paramRegexp.FindStringSubmatch(str); len(p) == 3 {
				m.params[p[1]] = p[2]
			}
		case "complete":
			switch m.code {
			case 202:
				s.parseSlot(m)
			case 205:
				s.parseClips(m)
			case 208:
				s.parseTransport(m)
				s.sendMedia()
			case 500:
				s.parseConnectionInfo(m)
			}
			state = "start"
		}
	}
}

func (s *state) relay() {
	l, err := net.Listen("tcp", relayAddr)
	if err != nil {
		log.Printf("hyperdeck relay error: %v", err)
		return
	}
	s.listenConn = l

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		conn, err := l.Accept()
		if err != nil {
			log.Printf("hyperdeck relay accept conn err: %v", err)
			return
		}

		ch := make(chan string)
		s.addListener <- ch
		go s.handleConn(conn, ch)
	}
}

func (s *state) handleConn(conn net.Conn, tx chan string) {
	go func() {
		for m := range tx {
			_, err := conn.Write([]byte(m))
			if err != nil {
				return
			}
		}
	}()

	var buffer bytes.Buffer
	reader := bufio.NewReader(conn)

	for {
		msg, isPrefix, err := reader.ReadLine()
		if err != nil {
			log.Printf("Hyperdeck: state.handleConn() err: %v", err)
			s.removeListener <- tx
			conn.Close()
			return
		}
		buffer.Write(msg)

		if isPrefix {
			// Did not get a complete line
			continue
		}

		str := buffer.String()
		buffer.Reset()

		// Filter out "quit"
		if str == "quit" {
			s.removeListener <- tx
			conn.Close()
			return
		}

		s.txChan <- (str + "\r\n")
	}
}

func (s *state) sender() {
	t := time.NewTicker(pollPeriod)
	clipsCMD := []byte("clips get\n")
	transportCMD := []byte("transport info\n")
	slotCMD := []byte("slot info\n")
	pingCMD := []byte("ping\n")

	for loop := true; loop; {
		select {
		case <-s.ctx.Done():
			loop = false
		case send := <-s.txChan:
			if send == "ping" {
				s.pingTicker.Reset(pingPeriod)
			}
			_, err := s.conn.Write([]byte(send))
			if err != nil {
				loop = false
			}
		case m, ok := <-s.rxChan:
			if !ok {
				loop = false
			}
			for _, c := range s.listeners {
				// Need to test for TCP breakage?
				c <- m
			}
		case <-t.C:
			// Request a status update
			_, err := s.conn.Write(clipsCMD)
			if err != nil {
				loop = false
			}
			_, err = s.conn.Write(transportCMD)
			if err != nil {
				loop = false
			}
			_, err = s.conn.Write(slotCMD)
			if err != nil {
				loop = false
			}
		case <-s.pingTicker.C:
			_, err := s.conn.Write(pingCMD)
			if err != nil {
				loop = false
			}
		case c := <-s.addListener:
			s.listeners = append(s.listeners, c)
			c <- s.banner
		case c := <-s.removeListener:
			for i, ch := range s.listeners {
				if ch == c {
					s.listeners[i] = s.listeners[len(s.listeners)-1]
					s.listeners = s.listeners[:len(s.listeners)-1]
					close(ch)
					break
				}
			}

		}
	}
	log.Printf("Hyperdeck: listener exiting on request")
	if s.conn != nil {
		s.conn.Close()
	}
}

func (s *state) sendMedia() {
	var name string

	if s.status == "stopped" {
		return
	}

	current_clip := s.clipID - 1

	if current_clip < 0 {
		// Special case for recording...
		current_clip = 0
	}

	if len(s.clips) > current_clip {
		name = s.clips[current_clip].name
	}

	length := s.playlistLength

	m := Media{
		name:        name,
		remaining:   length - s.timecode,
		length:      length,
		loop:        s.loop,
		singleClip:  s.singleClip,
		currentClip: s.clipID,
		totalClips:  len(s.clips),
	}

	switch s.status {
	case "record":
		m.remaining = s.recordingTime
		m.length = s.recordingTime
		m.record = true
	case "playing":
	case "preview":
		m.pause = true
	}

	s.mediaChan <- &m
}

func (h *Media) Remaining() time.Duration {
	return h.remaining
}

func (h *Media) Duration() time.Duration {
	return h.length
}

func (h *Media) Play() bool {
	return !h.pause
}

func (h *Media) Loop() bool {
	return h.loop
}

func (h *Media) Record() bool {
	return h.record
}

func (h *Media) MediaName() string {
	name := ""
	if h.totalClips < 2 {
		name = h.name
	} else {
		name = fmt.Sprintf("%d/%d %s", h.currentClip, h.totalClips, h.name)
	}
	return name
}

/*
clip count: 5
1: Capture0000.mov 00:00:00;00 00:00:10;06
2: Capture0001.mov 00:00:10;06 01:01:45;08
3: Capture0002.mov 01:01:55;14 00:00:01;39
4: Capture0003.mov 01:01:56;53 01:00:06;51
5: Capture0004.mov 02:02:03;48 00:00:05;43
*/

func (s *state) parseClips(m *message) {
	var err error
	numClips := 1
	if clipCount, ok := m.params["clip count"]; ok {
		numClips, err = strconv.Atoi(clipCount)
		if err != nil {
			log.Printf("hyperdeck parseClips illegal clip count")
			return
		}
	}

	clips := make([]*clip, numClips)
	for i := 0; i < numClips; i++ {
		v, ok := m.params[fmt.Sprintf("%d", i+1)]
		if !ok {
			continue
		}
		parts := clipRegexp.FindStringSubmatch(v)
		if len(parts) != 4 {
			continue
		}
		c := clip{
			name:          parts[1],
			playlistStart: parseDuration(parts[2]),
			length:        parseDuration(parts[3]),
		}
		clips[i] = &c
	}

	var total time.Duration
	for _, c := range clips {
		total += c.length
	}

	s.playlistLength = total
	s.clips = clips
}

/*
500 connection info:
protocol version: 1.8
model: HyperDeck Studio 12G
*/

func (s *state) parseConnectionInfo(m *message) {
	// Store the banner for retransmission

	s.banner = "500 connection info:\r\n"
	for k, v := range m.params {
		s.banner += k + ": " + v + "\r\n"
	}
	s.banner += "\r\n"
}

// 12:34:56;12
func parseDuration(str string) time.Duration {
	var dur time.Duration
	if p := timeRegexp.FindStringSubmatch(str); len(p) == 5 {
		hours, _ := strconv.Atoi(p[1])
		mins, _ := strconv.Atoi(p[2])
		secs, _ := strconv.Atoi(p[3])

		dur += time.Duration(hours) * time.Hour
		dur += time.Duration(mins) * time.Minute
		dur += time.Duration(secs) * time.Second
	}
	return dur
}

/*
202 slot info:
slot id: 1
status: mounted
volume name: DiskHFS
recording time: 4029
video format: 1080p60
device serial: S3F4NY0J405733
device model: SAMSUNG MZ7KM480HMHQ-00005
firmware revision: GXM510
*/

func (s *state) parseSlot(m *message) {
	// Other users of the API might request info on inactive slots
	// So filter out any that aren't the current active slot
	if val, ok := m.params["slot id"]; ok {
		slot, err := strconv.Atoi(val)
		if err != nil || slot != s.slotID {
			return
		}
	} else {
		return
	}

	// Get the estimated recording seconds left
	if val, ok := m.params["recording time"]; ok {
		t, err := strconv.Atoi(val)
		if err != nil {
			log.Printf("hyperdeck: parseSlot illegal recording time: %v", err)
			return
		}
		s.recordingTime = time.Duration(t) * time.Second
	}
}

/*
status: play
speed: 100
slot id: 1
clip id: 1
single clip: false
display timecode: 00:00:05;38
timecode: 00:00:05;38
video format: 1080p5994
loop: false
*/

func (s *state) parseTransport(m *message) {
	if val, ok := m.params["loop"]; ok {
		s.loop = (val == "true")
	}

	if val, ok := m.params["single clip"]; ok {
		s.singleClip = (val == "true")
	}

	if val, ok := m.params["display timecode"]; ok {
		s.displayTimecode = parseDuration(val)
	}

	if val, ok := m.params["timecode"]; ok {
		s.timecode = parseDuration(val)
	}

	if val, ok := m.params["clip id"]; ok {
		s.clipID, _ = strconv.Atoi(val)
	}

	if val, ok := m.params["slot id"]; ok {
		s.slotID, _ = strconv.Atoi(val)
	}

	if val, ok := m.params["speed"]; ok {
		s.speed, _ = strconv.Atoi(val)
	}

	if val, ok := m.params["status"]; ok {
		s.status = val
	}
}

func (s *state) String() string {
	ret := fmt.Sprintf("Hyperdeck: status: %s timecode: %s playlist: %v looping: %v single clip: %v clip id: %d slot id: %d recording time: %v\n",
		s.status, s.displayTimecode, s.playlistLength, s.loop, s.singleClip, s.clipID, s.slotID, s.recordingTime)
	for i, c := range s.clips {
		ret += fmt.Sprintf("\tClip %d: %s %v\n", i+1, c.name, c.length)
	}
	return ret
}
