package oscutil

import (
	"fmt"
	"github.com/jpkelly/clock8002/app/debug"
	// "gitlab.com/Depili/go-osc/osc"
	"errors"
	"github.com/chabad360/go-osc/osc"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// ParamRE is a regexp for validating ParseParams arguments
	ParamRE = `^(?: ?i \d+| ?[fd] \d+(?:.\d+)?| ?s (?:\".*?\"|\S+)| ?b (?:true|false))+ *$`
)

// RegexpDispatcher is a dispatcher for OSC packets. It handles the dispatching of
// received OSC packets to Handlers for their given address.
type RegexpDispatcher struct {
	handlers       []*oscHandler
	defaultHandler osc.MethodFunc
}

type oscHandler struct {
	handler osc.MethodFunc
	exp     *regexp.Regexp
}

// NewRegexpDispatcher returns an RegexpDispatcher.
func NewRegexpDispatcher() *RegexpDispatcher {
	return &RegexpDispatcher{handlers: make([]*oscHandler, 0)}
}

// AddMsgHandler adds a new message handler for the given OSC address.
func (s *RegexpDispatcher) AddMsgHandler(addr string, handler osc.MethodFunc) error {
	if addr == "*" {
		s.defaultHandler = handler
		return nil
	}

	s.handlers = append(s.handlers, &oscHandler{
		handler: handler,
		exp:     getRegEx(addr),
	})
	return nil
}

// Dispatch dispatches OSC packets. Implements the Dispatcher interface.
func (s *RegexpDispatcher) Dispatch(packet osc.Packet, a net.Addr) {
	switch p := packet.(type) {
	default:
		return

	case *osc.Message:
		msg, _ := packet.(*osc.Message)
		debug.Printf("Dispatching osc message %v %v", msg.Address, msg.Arguments)
		for _, h := range s.handlers {
			if h.exp.MatchString(msg.Address) {
				debug.Printf("-> found handler")
				h.handler.HandleMessage(msg)
				break
			}
		}
		if s.defaultHandler != nil {
			s.defaultHandler.HandleMessage(p)
		}

	case *osc.Bundle:
		timer := time.NewTimer(p.Timetag.ExpiresIn())

		go func() {
			<-timer.C
			for _, message := range p.Elements {
				switch m := message.(type) {
				case *osc.Message:
					debug.Printf("Dispatching osc message %v %v", m.Address, m.Arguments)
					for _, h := range s.handlers {
						if h.exp.MatchString(m.Address) {
							debug.Printf("-> found handler")
							h.handler.HandleMessage(m)
							break
						}
					}
					if s.defaultHandler != nil {
						s.defaultHandler.HandleMessage(m)
					}
				case *osc.Bundle:
					s.Dispatch(m, a)
				}
			}
		}()
	}
}

// ParseParams parses a string for OSC message arguments
func ParseParams(msg *osc.Message, params string) error {
	re := regexp.MustCompile(ParamRE)

	if !re.MatchString(params) {
		return errors.New("Invalid parameter format")
	}

	state := "seek"
	buffer := ""
	quote := false

	params += " "

	for _, c := range params {
		if len(buffer) == 0 && c == ' ' {
			continue
		}
		switch state {
		case "seek":
			switch c {
			case 'i':
				state = "int"
			case 'f':
				state = "float"
			case 'd':
				state = "double"
			case 's':
				state = "string"
			case 'b':
				state = "bool"
			}
			buffer = ""
			continue
		case "int":
			if c == ' ' {
				val, _ := strconv.Atoi(buffer)
				msg.Append(int32(val))
				state = "seek"
			}
		case "float", "double":
			if c == ' ' {
				val, _ := strconv.ParseFloat(buffer, 64)
				if state == "float" {
					msg.Append(float32(val))
				} else {
					msg.Append(float64(val))
				}
				state = "seek"
			}
		case "string":
			if !quote && c == ' ' {
				msg.Append(buffer)
				state = "seek"
			} else if c == '"' {
				if len(buffer) == 0 || buffer[len(buffer)-1:] != "\\" {
					quote = !quote
					continue
				}
			}
		case "bool":
			if c == ' ' {
				msg.Append(strings.ToLower(buffer) == "true")
				state = "seek"
			}
		}
		buffer += string(c)
	}
	return nil
}

// UnmarshalArgument tries to parse a given value from the osc message arguments at given index
func UnmarshalArgument(msg *osc.Message, argIndex int, value interface{}) error {
	var valueType = reflect.TypeOf(value)

	if valueType.Kind() != reflect.Ptr {
		panic("value is not a pointer")
	}

	if len(msg.Arguments) <= argIndex {
		return fmt.Errorf("missing argument %d", argIndex)
	}

	var arg = msg.Arguments[argIndex]
	var argValue = reflect.ValueOf(arg)
	var valueValue = reflect.ValueOf(value)

	if argValue.Type().Kind() != valueType.Elem().Kind() {
		return fmt.Errorf("invalid arugment %d: expected %v, got %T: %#v", argIndex, valueType.Elem(), arg, arg)
	}

	valueValue.Elem().Set(argValue)

	return nil
}

// UnmarshalArguments tries to match the given variables to osc message arguments
func UnmarshalArguments(msg *osc.Message, values ...interface{}) error {
	for i, value := range values {
		if err := UnmarshalArgument(msg, i, value); err != nil {
			return err
		}
	}

	return nil
}

func getRegEx(pattern string) *regexp.Regexp {
	for _, trs := range []struct {
		old, new string
	}{
		{".", `\.`}, // Escape all '.' in the pattern
		{"(", `\(`}, // Escape all '(' in the pattern
		{")", `\)`}, // Escape all ')' in the pattern
		{"*", ".*"}, // Replace a '*' with '.*' that matches zero or more chars
		{"{", "("},  // Change a '{' to '('
		{",", "|"},  // Change a ',' to '|'
		{"}", ")"},  // Change a '}' to ')'
		{"?", "."},  // Change a '?' to '.'
	} {
		pattern = strings.Replace(pattern, trs.old, trs.new, -1)
	}

	return regexp.MustCompile(pattern)
}
