package picturall

import (
	"context"
	"net"
	"regexp"
	"time"
)

const msgRegexp = `^MSG\((\d+), (\d+), (\d+), (.+)\)`
const sourceRegexp = `^object name="(source\d+)"`
const sourceNumberRegexp = `source(\d+)`
const messageMapRegexp = `^(\S+) ((?:\S+=\S+,?)+)`

var msgRe = regexp.MustCompile(msgRegexp)
var sourceRe = regexp.MustCompile(sourceRegexp)
var sourceNumberRe = regexp.MustCompile(sourceNumberRegexp)
var messageMapRe = regexp.MustCompile(messageMapRegexp)

// DumpTraffic controls if all traffic gets dumped to a log file
var DumpTraffic = false

const logTimestamp = "2006-01-02T15:04:05.000"

const (
	keepAliveSleep = time.Second * 10
	fetchTimeout   = time.Second * 20
)

// Msg is a struct for generic Picturall messages
type Msg struct {
	Target  int
	Source  int
	Type    int
	Content string
}

// Media is a struct for Picturall media status messages
type Media struct {
	Msg
	Name            string
	PlayState       int
	Head            time.Duration
	Length          time.Duration
	MediaEndAction  int
	Layer           int
	PlayStateReq    int
	DefaultPlayMode int
}

// Source is a struct for Picturall source state
type Source struct {
	Msg
	Media          *Media
	Name           string
	MediaEndAction int
	PlayStateReq   int
	Map            map[string](map[string]string)
	Collection     int
	Slot           int
}

// State is a struct for complete Picturall state
type State struct {
	enums     map[int]string
	sources   map[string]*Source
	conn      net.Conn
	mediaChan chan *Media
	ctx       context.Context
	ip        string
	mc        *MediaCollections
}

var state *State

// Message types
const (
	CtrlStatus           = 13
	CtrlInfo             = 14
	EnumObjects          = 15
	Overflow             = 20
	ModelChangeAdd       = 24
	CmdSystemResults     = 33
	CmdSystemLineResults = 38
)

// Media modes
const (
	PlayError      = -1
	PlayDefault    = 0 // Loop
	PlayNext       = 1
	PlayStop       = 2
	PlayPause      = 3
	PlayLoop       = 4
	Pause          = 5
	Stop           = 6
	LoopCollection = 10
	Random         = 11
)
