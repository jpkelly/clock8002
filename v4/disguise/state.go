package disguise

import (
	"context"
	"github.com/chabad360/go-osc/osc"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	"gitlab.com/clock-8001/clock-8001/v4/oscutil"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var sectionHintRe *regexp.Regexp
var timeRe *regexp.Regexp

func init() {
	sectionHintRe = regexp.MustCompile(`^(.*) \+(\d\d:\d\d:\d\d.\d\d) (.*) -(\d\d:\d\d:\d\d.\d\d)`)
	timeRe = regexp.MustCompile(`(\d\d):(\d\d):(\d\d).(\d\d)`)
}

type state struct {
	ctx             context.Context
	ipFilter        *regexp.Regexp
	server          *osc.Server
	d               *oscutil.RegexpDispatcher
	c               chan *Media
	playMode        string
	head            time.Duration
	remaining       time.Duration
	sectionName     string
	nextSectionName string
	noteName        string
	nextNote        string
	wg              *sync.WaitGroup
	nextName        bool
}

func (s *state) setup(addr string) {
	s.d = oscutil.NewRegexpDispatcher()
	s.registerHandler("^/d3/showcontrol/sectionhint$", s.handleSectionHint)
	s.registerHandler("^/d3/showcontrol/currentsectionname$", s.handleSectionName)
	s.registerHandler("^/d3/showcontrol/nextsectionname$", s.handleNextSectionName)
	s.registerHandler("^/d3/showcontrol/playmode$", s.handlePlayMode)

	s.server = &osc.Server{
		Addr:    addr,
		Handler: s.d.Dispatch,
	}
}

func (s *state) run() {
	s.wg.Add(1)
	defer s.wg.Done()
	go s.closer()

	for {
		err := s.server.ListenAndServe()
		if err != nil {
			if e, ok := err.(*net.OpError); ok {
				if e.Temporary() {
					log.Printf("Disguise: Temporary error: %v. Retrying", e)
					s.server.Close()
				} else {
					log.Printf("Disguise: Fatal error: %v. Giving up", e)
					s.server.Close()
					s.server = nil
					return
				}
			} else {
				log.Printf("Disguise: Error: %T %v", err, err)
				log.Printf("Retrying...")
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *state) closer() {
	<-s.ctx.Done()
	if s.server != nil {
		log.Printf("Disguise: Closing OSC listener: %s", s.server.Addr)
		s.server.Close()
	} else {
		log.Printf("Disguise: Closing OSC listener after fatal error")
	}
}

func (s *state) handleSectionHint(msg *osc.Message) {
	debug.Printf("Disguise: handleSectionHint: %v", msg)
	var payload string
	err := oscutil.UnmarshalArguments(msg, &payload)
	if err != nil {
		log.Printf("Disguise: handleSectionHint UnmarshalArguments error: %v", err)
		return
	}

	if matches := sectionHintRe.FindStringSubmatch(payload); matches != nil && len(matches) == 5 {
		currentNote := strings.TrimSpace(matches[1])
		head := parseTime(matches[2])
		nextNote := strings.TrimSpace(matches[3])
		remaining := parseTime(matches[4])

		length := head + remaining

		debug.Printf("Disguise: SectionHint: -%s (%s / %s) %s %s", remaining, head, length, currentNote, nextNote)
		s.head = head
		s.remaining = remaining
		s.noteName = currentNote
		s.nextNote = nextNote
	}

	s.sendUpdate()
}

func (s *state) sendUpdate() {
	m := Media{
		head:        s.head,
		remaining:   s.remaining,
		playMode:    s.playMode,
		sectionName: s.sectionName,
		note:        s.noteName,
	}

	if s.nextName {
		m.sectionName = s.nextSectionName
		m.note = s.nextNote
	}

	s.c <- &m
}

func (s *state) handleSectionName(msg *osc.Message) {
	debug.Printf("Disguise: handleSectionName: %v", msg)
	var name string
	err := oscutil.UnmarshalArguments(msg, &name)
	if err != nil {
		log.Printf("Disguise: handleSectionName error: %v", err)
		return
	}

	s.sectionName = name
}

func (s *state) handleNextSectionName(msg *osc.Message) {
	debug.Printf("Disguise: handleNextSectionName: %v", msg)
	var name string
	err := oscutil.UnmarshalArguments(msg, &name)
	if err != nil {
		log.Printf("Disguise: handleNextSectionName error: %v", err)
		return
	}

	s.nextSectionName = name
}

func (s *state) handlePlayMode(msg *osc.Message) {
	debug.Printf("Disguise: handlePlayMode: %v", msg)
	var mode string
	err := oscutil.UnmarshalArguments(msg, &mode)
	if err != nil {
		log.Printf("Disguise: handlePlayMode rror: %v", err)
		return
	}

	s.playMode = mode
}

func (s *state) registerHandler(addr string, handler osc.MethodFunc) {
	if err := s.d.AddMsgHandler(addr, handler); err != nil {
		panic(err)
	}
}

func parseTime(str string) time.Duration {
	dur := time.Duration(0)
	if matches := timeRe.FindStringSubmatch(str); len(matches) == 5 {
		hours, _ := strconv.Atoi(matches[1])
		minutes, _ := strconv.Atoi(matches[2])
		seconds, _ := strconv.Atoi(matches[3])

		dur += time.Duration(hours) * time.Hour
		dur += time.Duration(minutes) * time.Minute
		dur += time.Duration(seconds) * time.Second
	}

	return dur
}

func isCue(currentNote string) bool {
	return strings.HasPrefix(currentNote, "MIDI ") || strings.HasPrefix(currentNote, "TC ") || strings.HasPrefix(currentNote, "CUE ")
}
