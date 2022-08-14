package picturall

import (
	"fmt"
	"log"
	"time"
)

// haveEnums returns true if Picturall object enums list has been retrie
func (s *State) haveEnums() bool {
	return len(state.enums) != 0
}

// reset the internal picturall state
func (s *State) reset() {
	s.sources = make(map[string]*Source)
	s.enums = nil
	s.mediaChan = make(chan *Media)
}

func (s *State) initConnection() error {
	log.Printf("Picturall setting initial connection state")
	_, err := s.conn.Write([]byte("loglevel none\n"))
	if err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)

	_, err = s.conn.Write([]byte("receiving all\n"))
	if err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)

	_, err = s.conn.Write([]byte("enum_objects\n"))
	if err != nil {
		return err
	}

	go s.keepAlive()

	return nil
}

func (s *State) keepAlive() {
	defer s.conn.Close()

	for {
		_, err := s.conn.Write([]byte("\n"))
		if !s.haveEnums() {
			_, err = s.conn.Write([]byte("enum_objects\n"))
			if err != nil {
				return
			}
		}

		if err != nil {
			log.Printf("Picturall keepAlive() error writing: %v", err)
			return
		}
		time.Sleep(time.Second)
	}
}

func (s *State) getStatus(ctrl string) {
	cmd := fmt.Sprintf("ctrl_status %s\n", ctrl)
	s.conn.Write([]byte(cmd))
}
