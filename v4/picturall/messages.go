package picturall

import (
	"fmt"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	"log"
	"strconv"
	"strings"
	"time"
)

func parseMsg(msg string) *Msg {
	if m := msgRe.FindStringSubmatch(msg); len(m) > 1 {
		p := &Msg{
			Content: m[4],
		}
		p.Target, _ = strconv.Atoi(m[1])
		p.Source, _ = strconv.Atoi(m[2])
		p.Type, _ = strconv.Atoi(m[3])
		return p
	}
	return nil
}

// Loop returns true if the media is looping
func (m *Media) Loop() bool {
	return (m.PlayState == PlayDefault && m.MediaEndAction == 4) || m.PlayState == PlayLoop
}

// Progress returns the media progress, between 0 and 1
func (m *Media) Progress() float64 {
	return 1 - m.Head.Seconds()/m.Length.Seconds()
}

// Play returns true if media is playing
func (m *Media) Play() bool {
	ps := m.PlayState
	return (ps != Stop && ps != Pause)
}

// String formats the PicturallMedia struct as a string
func (m *Media) String() string {
	v, ok := state.enums[m.Source]
	if ok {
		return fmt.Sprintf("Source: %s Target: %d Media: %s state: %d layer: %d looping: %v (%0.4fs / %0.4fs)", v, m.Target, m.Name, m.PlayState, m.Layer, m.Loop(), m.Head.Seconds(), m.Length.Seconds())
	}
	return fmt.Sprintf("Source: %d Target: %d Media: %s state: %d layer: %d looping: %v (%0.4fs / %0.4fs)", m.Source, m.Target, m.Name, m.PlayState, m.Layer, m.Loop(), m.Head.Seconds(), m.Length.Seconds())
}

func (m *Media) combineSource(source string) {
	if p := sourceNumberRe.FindStringSubmatch(source); len(p) == 2 {
		m.Layer, _ = strconv.Atoi(p[1])
	} else {
		m.Layer = -1
	}

	if s, ok := state.sources[source]; ok {
		m.MediaEndAction = s.MediaEndAction
		m.PlayStateReq = s.PlayStateReq
		m.PlayState = s.PlayStateReq
	} else {
		m.MediaEndAction = -1
	}
}

//ParseEnum parses an enum_objects reply message into a id -> name mapping
func (p *Msg) ParseEnum() map[int]string {
	// Example message:
	// MSG(100002, 1, 15, 142:magewell2_0:\n145:magewell4_0:\n147:magewell4_1:\n226:audio_driver1:\n228:canvas1:\n232:artnet1:\n233:encoder:\n234:mtc:\n236:tcg1:\n237:tcg2:\n238:tcg3:\n239:tcg4:\n240:tcg5:\n241:tcg6:\n242:tcg7:\n243:tcg8:\n244:wallclock:\n245:lcd_writer:\n284:fx_info:\n323:layer1:\n324:source1:\n325:audio1:\n330:fx_l1_fx1:\n331:fx_l1_fx2:\n332:layer2:\n333:source2:\n334:audio2:\n339:fx_l2_fx1:\n340:fx_l2_fx2:\n341:layer3:\n342:source3:\n343:audio3:\n348:fx_l3_fx1:\n349:fx_l3_fx2:\n350:layer4:\n351:source4:\n352:audio4:\n357:fx_l4_fx1:\n358:fx_l4_fx2:\n359:layer5:\n360:source5:\n361:audio5:\n366:fx_l5_fx1:\n367:fx_l5_fx2:\n368:layer6:\n369:source6:\n370:audio6:\n375:fx_l6_fx1:\n376:fx_l6_fx2:\n377:layer7:\n378:source7:\n379:audio7:\n384:fx_l7_fx1:\n385:fx_l7_fx2:\n386:layer8:\n387:source8:\n388:audio8:\n393:fx_l8_fx1:\n394:fx_l8_fx2:\n395:layer9:\n396:source9:\n397:audio9:\n402:fx_l9_fx1:\n403:fx_l9_fx2:\n404:layer10:\n405:source10:\n406:audio10:\n411:fx_l10_fx1:\n412:fx_l10_fx2:\n413:layer11:\n414:source11:\n415:audio11:\n420:fx_l11_fx1:\n421:fx_l11_fx2:\n422:layer12:\n423:source12:\n424:audio12:\n429:fx_l12_fx1:\n430:fx_l12_fx2:\n431:layer13:\n432:source13:\n433:audio13:\n438:fx_l13_fx1:\n439:fx_l13_fx2:\n440:layer14:\n441:source14:\n442:audio14:\n447:fx_l14_fx1:\n448:fx_l14_fx2:\n449:layer15:\n450:source15:\n451:audio15:\n456:fx_l15_fx1:\n457:fx_l15_fx2:\n458:layer16:\n459:source16:\n460:audio16:\n465:fx_l16_fx1:\n466:fx_l16_fx2:\n467:layer17:\n468:source17:\n469:audio17:\n474:fx_l17_fx1:\n475:fx_l17_fx2:\n476:layer18:\n477:source18:\n478:audio18:\n483:fx_l18_fx1:\n484:fx_l18_fx2:\n485:layer19:\n486:source19:\n487:audio19:\n492:fx_l19_fx1:\n493:fx_l19_fx2:\n494:layer20:\n495:source20:\n496:audio20:\n501:fx_l20_fx1:\n502:fx_l20_fx2:\n503:layer21:\n504:source21:\n505:audio21:\n510:fx_l21_fx1:\n511:fx_l21_fx2:\n512:layer22:\n513:source22:\n514:audio22:\n519:fx_l22_fx1:\n520:fx_l22_fx2:\n521:layer23:\n522:source23:\n523:audio23:\n528:fx_l23_fx1:\n529:fx_l23_fx2:\n530:layer24:\n531:source24:\n532:audio24:\n537:fx_l24_fx1:\n538:fx_l24_fx2:\n539:layer25:\n540:source25:\n541:audio25:\n546:fx_l25_fx1:\n547:fx_l25_fx2:\n548:layer26:\n549:source26:\n550:audio26:\n555:fx_l26_fx1:\n556:fx_l26_fx2:\n557:layer27:\n558:source27:\n559:audio27:\n564:fx_l27_fx1:\n565:fx_l27_fx2:\n566:layer28:\n567:source28:\n568:audio28:\n573:fx_l28_fx1:\n574:fx_l28_fx2:\n575:layer29:\n576:source29:\n577:audio29:\n582:fx_l29_fx1:\n583:fx_l29_fx2:\n584:layer30:\n585:source30:\n586:audio30:\n591:fx_l30_fx1:\n592:fx_l30_fx2:\n593:layer31:\n594:source31:\n595:audio31:\n600:fx_l31_fx1:\n601:fx_l31_fx2:\n602:layer32:\n603:source32:\n604:audio32:\n609:fx_l32_fx1:\n610:fx_l32_fx2:\n611:citp:\n612:monitor:\n613:file_watch:\n614:alsamixer:\n615:stack1:\n617:stack2:\n619:stack3:\n621:stack4:\n623:stack5:\n625:stack6:\n627:stack7:\n629:stack8:\n631:cue1:\n632:gpu1:\n643:gpu2:\n654:ltc1:\n656:ltc2:\n658:ltc3:\n660:ltc4:\n662:clocksv:\n663:timebasesv:\n664:net1:\n665:net2:\n666:net3:\n667:net4:\n668:performance_watch:\n669:io_monitor:\n)

	enum := make(map[int]string)
	lines := strings.Split(p.Content, "\\n")
	for _, l := range lines {
		parts := strings.Split(l, ":")
		e, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Printf("Picturall ParseEnum() error: %v", err)
			continue
		}
		if parts[1][len(parts[1])-1] == ':' {
			enum[e] = parts[1][:len(parts[1])-1]
		} else {
			enum[e] = parts[1]
		}
	}
	return enum
}

// SourceString gets the string representation of the source enum
func (p *Msg) SourceString() (string, bool) {
	if source, ok := state.enums[p.Source]; ok {
		return source, true
	}
	return "", false
}

// ParseMedia parses the message payload into a struct representing playing media
func (p *Msg) ParseMedia() *Media {
	var msgMap map[string](map[string]string)

	if p.Type != CtrlStatus {
		return nil
	}

	msgMap = parseMap(p.Content)

	payload, ok := msgMap["info"]
	if !ok {
		debug.Printf("Picturall ParseMedia() missing info section")
		return nil
	}

	debug.Printf("Picturall ParseMedia() payload: %v", payload)

	timecode, err := strconv.Atoi(payload["timecode"])
	if err != nil {
		debug.Printf("Picturall ParseMedia() error parsing timecode %s - %v", payload["timecode"], err)
		return nil
	}

	mediaLength, err := strconv.Atoi(payload["media_length"])
	if err != nil {
		debug.Printf("Picturall ParseMedia() error parsing media length %s - %v", payload["media_length"], err)
		return nil
	}

	playState, err := strconv.Atoi(payload["play_state"])
	if err != nil {
		debug.Printf("Picturall ParseMedia() error parsing play state %s - %v", payload["play_state"], err)
		return nil
	}

	media := payload["media_file"]
	if len(media) > 1 && media[0] == '"' {
		// Strip "" limiters from media name
		media = media[1 : len(media)-1]
	}

	head := time.Duration(timecode) * time.Nanosecond
	length := time.Duration(mediaLength) * time.Nanosecond

	m := &Media{
		Name:      media,
		PlayState: playState,
		Head:      head,
		Length:    length,
	}

	m.Target = p.Target
	m.Source = p.Source
	m.Type = p.Type
	m.Content = p.Content

	debug.Printf("  -> Media: %s state: %d (%0.4fs / %0.4fs)\n\n", media, playState, head.Seconds(), length.Seconds())

	return m
}

// ParseSource parses the ctrl_status sourceNN command reply message
func (p *Msg) ParseSource() *Source {
	// Example message:
	// object name="source1",description=""\ninfo media_file="/picturall/media/assy_wide_2.mp4",play_state=0,timecode=31866666666,media_length=60032000000\nmetadata width=8000,height=1600,has_video=1,supported_video=1,fps=60,aspect_ratio=5,video_codec="hevc",media_container="mp4",media_type="media",has_audio=1,supported_audio=1,frequency=48000,channels=2,audio_codec="aac",media_in=0,media_out=1\nselection slot=2,collection=4\ncontrol media_end_action=4,play_state_req=0,seek=0.958128\nsync source=0,timecode=31866666666,input_offset=0,output_offset=0\ntime fps=30,relative_fps=1,fps_mode=0,effective_fps=60,fps_control_allowed=1\ncrossfade type=1,duration=1,smoothing=1,progress=0\nfade in_type=0,in=1,out_type=0,out=1\nframe_blending mode=0\ntools step=0\n

	var err error
	if p.Type == CtrlStatus {
		if source := sourceRe.FindStringSubmatch(p.Content); len(source) == 2 {
			s := &Source{
				Name: source[1],
			}

			s.Target = p.Target
			s.Source = p.Source
			s.Type = p.Type
			s.Content = p.Content

			payload := parseMap(s.Content)

			s.Map = payload

			s.MediaEndAction, err = strconv.Atoi(payload["control"]["media_end_action"])
			if err != nil {
				debug.Printf("Pictural ParseSource() error parsing media end action %s - %v", payload["media_end_action"], err)
				return nil
			}

			s.PlayStateReq, err = strconv.Atoi(payload["control"]["play_state_req"])
			if err != nil {
				debug.Printf("Pictural ParseSource() error parsing play state req %s - %v", payload["play_state_req"], err)
				return nil
			}

			if m := p.ParseMedia(); m != nil {
				// Got also the media state in this message
				m.combineSource(s.Name)
				s.Media = m
			}

			return s
		}
	}
	return nil
}

func parseMap(data string) map[string](map[string]string) {
	ret := make(map[string](map[string]string))
	debug.Printf("Pictural parseMap -> data %s", data)
	sections := strings.Split(data, "\\n")
	for _, section := range sections {
		matches := messageMapRe.FindStringSubmatch(section)
		m := make(map[string]string)
		if len(matches) != 3 {
			continue
		}
		sectionName := matches[1]

		attrs := strings.Split(matches[2], ",")
		for _, a := range attrs {
			s := strings.Split(a, "=")
			if len(s) < 2 {
				debug.Printf("Pictural parseMap -> error splitting %s", a)
			} else {
				m[s[0]] = strings.Join(s[1:], "=")
			}
		}
		ret[sectionName] = m

	}
	if debug.Enabled {
		log.Printf("Picturall parseMap:")
		for k, v := range ret {
			log.Printf(" -> %s: %v", k, v)
		}
	}
	return ret
}

// PlayState converts the play state enum field to string
func PlayState(s int) string {
	switch s {
	case PlayError:
		return "Play error"
	case PlayDefault:
		return "Play default (loop)"
	case PlayNext:
		return "Play next"
	case PlayStop:
		return "Play stop"
	case PlayPause:
		return "Play pause"
	case PlayLoop:
		return "Play loop"
	case Pause:
		return "Pause"
	case Stop:
		return "Stop"
	case LoopCollection:
		return "Loop collection"
	case Random:
		return "Random"
	default:
		return "Unkown"
	}
}

// String formats the Msg struct as a string
func (p *Msg) String() string {
	return fmt.Sprintf("Target: %d Source: %d Type: %d  Content: %s",
		p.Target, p.Source, p.Type, p.Content)
}
