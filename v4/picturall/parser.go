package picturall

import (
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	"log"
	"strconv"
)

func (m *Media) combineSource(source string, state *State) {
	m.DefaultPlayMode = -100
	m.MediaEndAction = -100
	m.Layer = -1

	if p := sourceNumberRe.FindStringSubmatch(source); len(p) == 2 {
		m.Layer, _ = strconv.Atoi(p[1])
	}

	if s, ok := state.sources[source]; ok {
		m.MediaEndAction = s.MediaEndAction
		m.PlayStateReq = s.PlayStateReq

		if state.telnetHasDefaultPlayMode {
			if c, ok := state.mcTelnet[s.Collection]; ok {
				if media, ok := c[s.Slot]; ok {
					m.DefaultPlayMode = media.PlayMode
				}
			}
		} else if state.mc != nil {
			// Try to get the default playmode....
			if len(state.mc.Collections) >= s.Collection {
				for _, media := range state.mc.Collections[s.Collection].Medias {
					if media.Index == s.Slot {
						m.DefaultPlayMode = media.PlayMode
					}
				}
			}
		}

	}
}

func parseMap(data string) map[string](map[string]string) {
	ret := make(map[string](map[string]string))
	debug.Printf("Pictural parseMap -> data %s", data)
	sections := sectionRe.FindAllStringSubmatch(data, -1)
	for _, section := range sections {
		m := make(map[string]string)

		sectionName := section[1]

		attrs := attrRe.FindAllStringSubmatch(section[2], -1)

		for _, a := range attrs {
			debug.Printf(" -> Attrs: %v", a)
			if a[2] != "" {
				m[a[1]] = a[2]
			} else {
				m[a[1]] = a[3]
			}
		}
		ret[sectionName] = m

	}
	if debug.Enabled {
		log.Printf("Picturall parseMap:")
		for k, v := range ret {
			log.Printf(" -> %s:", k)
			for kk, vv := range v {
				log.Printf("   -> %s: %s", kk, vv)
			}
		}
	}
	return ret
}
