package picturall

import (
	"testing"
)

func TestParseMap(t *testing.T) {
	data := parseMap(`object name="source1",description=""\ninfo media_file="/picturall/media/assy_wide_2.mp4",play_state=0,timecode=31866666666,media_length=60032000000\nmetadata width=8000,height=1600,has_video=1,supported_video=1,fps=60,aspect_ratio=5,video_codec="hevc",media_container="mp4",media_type="media",has_audio=1,supported_audio=1,frequency=48000,channels=2,audio_codec="aac",media_in=0,media_out=1\nselection slot=2,collection=4\ncontrol media_end_action=4,play_state_req=0,seek=0.958128\nsync source=0,timecode=31866666666,input_offset=0,output_offset=0\ntime fps=30,relative_fps=1,fps_mode=0,effective_fps=60,fps_control_allowed=1\ncrossfade type=1,duration=1,smoothing=1,progress=0\nfade in_type=0,in=1,out_type=0,out=1\nframe_blending mode=0\ntools step=0\n`)
	if len(data) != 11 {
		t.Errorf("Wrong number of sections parsed, got: %d expected 11", len(data))
	}
	if len(data["control"]) != 3 {
		t.Errorf("Wrong number of entries in control section, got %d expected 3", len(data["control"]))
	}

	if data["info"]["media_file"] != `/picturall/media/assy_wide_2.mp4` {
		t.Errorf("Wrong media file name: %s", data["info"]["media_file"])
	}

	data = parseMap(`object name="source1 \n\"",description=""\ninfo media_file="/picturall/media/assy_wide_2.mp4",play_state=0,timecode=31866666666,media_length=60032000000\nmetadata width=8000,height=1600,has_video=1,supported_video=1,fps=60,aspect_ratio=5,video_codec="hevc",media_container="mp4",media_type="media",has_audio=1,supported_audio=1,frequency=48000,channels=2,audio_codec="aac",media_in=0,media_out=1\nselection slot=2,collection=4\ncontrol media_end_action=4,play_state_req=0,seek=0.958128\nsync source=0,timecode=31866666666,input_offset=0,output_offset=0\ntime fps=30,relative_fps=1,fps_mode=0,effective_fps=60,fps_control_allowed=1\ncrossfade type=1,duration=1,smoothing=1,progress=0\nfade in_type=0,in=1,out_type=0,out=1\nframe_blending mode=0\ntools step=0\n`)
	if len(data) != 11 {
		t.Errorf("Wrong number of sections parsed, got: %d expected 11", len(data))
	}
	if len(data["control"]) != 3 {
		t.Errorf("Wrong number of entries in control section, got %d expected 3", len(data["control"]))
	}

	if data["info"]["media_file"] != `/picturall/media/assy_wide_2.mp4` {
		t.Errorf("Wrong media file name: %s", data["info"]["media_file"])
	}

	data = parseMap(`info media_file="",play_state=6,timecode=0,media_length=0`)
	if i, ok := data["info"]; ok {
		if len(i) != 4 {
			t.Errorf("Wrong number of info message attributes, got %d expected 4", len(i))
		}
	} else {
		t.Errorf("Failed to parse info message")
	}

	data = parseMap(`info media_file="Test ASD",play_state=6,timecode=0,media_length=0`)
	if data["info"]["media_file"] != `Test ASD` {
		t.Errorf("Failed to parse string with spaces")
	}

	data = parseMap(`info media_file="Test \"ASD",play_state=6,timecode=0,media_length=0`)
	if data["info"]["media_file"] != `Test \"ASD` {
		t.Errorf("Failed to parse string with escapes: %s", data["info"]["media_file"])
	}
}

func TestParseEnum(t *testing.T) {
	msg := `MSG(100002, 1, 15, 142:magewell2_0:\n145:magewell4_0:\n147:magewell4_1:\n226:audio_driver1:\n228:canvas1:\n232:artnet1:\n233:encoder:\n234:mtc:\n236:tcg1:\n237:tcg2:\n238:tcg3:\n239:tcg4:\n240:tcg5:\n241:tcg6:\n242:tcg7:\n243:tcg8:\n244:wallclock:\n245:lcd_writer:\n284:fx_info:\n323:layer1:\n324:source1:\n325:audio1:\n330:fx_l1_fx1:\n331:fx_l1_fx2:\n332:layer2:\n333:source2:\n334:audio2:\n339:fx_l2_fx1:\n340:fx_l2_fx2:\n341:layer3:\n342:source3:\n343:audio3:\n348:fx_l3_fx1:\n349:fx_l3_fx2:\n350:layer4:\n351:source4:\n352:audio4:\n357:fx_l4_fx1:\n358:fx_l4_fx2:\n359:layer5:\n360:source5:\n361:audio5:\n366:fx_l5_fx1:\n367:fx_l5_fx2:\n368:layer6:\n369:source6:\n370:audio6:\n375:fx_l6_fx1:\n376:fx_l6_fx2:\n377:layer7:\n378:source7:\n379:audio7:\n384:fx_l7_fx1:\n385:fx_l7_fx2:\n386:layer8:\n387:source8:\n388:audio8:\n393:fx_l8_fx1:\n394:fx_l8_fx2:\n395:layer9:\n396:source9:\n397:audio9:\n402:fx_l9_fx1:\n403:fx_l9_fx2:\n404:layer10:\n405:source10:\n406:audio10:\n411:fx_l10_fx1:\n412:fx_l10_fx2:\n413:layer11:\n414:source11:\n415:audio11:\n420:fx_l11_fx1:\n421:fx_l11_fx2:\n422:layer12:\n423:source12:\n424:audio12:\n429:fx_l12_fx1:\n430:fx_l12_fx2:\n431:layer13:\n432:source13:\n433:audio13:\n438:fx_l13_fx1:\n439:fx_l13_fx2:\n440:layer14:\n441:source14:\n442:audio14:\n447:fx_l14_fx1:\n448:fx_l14_fx2:\n449:layer15:\n450:source15:\n451:audio15:\n456:fx_l15_fx1:\n457:fx_l15_fx2:\n458:layer16:\n459:source16:\n460:audio16:\n465:fx_l16_fx1:\n466:fx_l16_fx2:\n467:layer17:\n468:source17:\n469:audio17:\n474:fx_l17_fx1:\n475:fx_l17_fx2:\n476:layer18:\n477:source18:\n478:audio18:\n483:fx_l18_fx1:\n484:fx_l18_fx2:\n485:layer19:\n486:source19:\n487:audio19:\n492:fx_l19_fx1:\n493:fx_l19_fx2:\n494:layer20:\n495:source20:\n496:audio20:\n501:fx_l20_fx1:\n502:fx_l20_fx2:\n503:layer21:\n504:source21:\n505:audio21:\n510:fx_l21_fx1:\n511:fx_l21_fx2:\n512:layer22:\n513:source22:\n514:audio22:\n519:fx_l22_fx1:\n520:fx_l22_fx2:\n521:layer23:\n522:source23:\n523:audio23:\n528:fx_l23_fx1:\n529:fx_l23_fx2:\n530:layer24:\n531:source24:\n532:audio24:\n537:fx_l24_fx1:\n538:fx_l24_fx2:\n539:layer25:\n540:source25:\n541:audio25:\n546:fx_l25_fx1:\n547:fx_l25_fx2:\n548:layer26:\n549:source26:\n550:audio26:\n555:fx_l26_fx1:\n556:fx_l26_fx2:\n557:layer27:\n558:source27:\n559:audio27:\n564:fx_l27_fx1:\n565:fx_l27_fx2:\n566:layer28:\n567:source28:\n568:audio28:\n573:fx_l28_fx1:\n574:fx_l28_fx2:\n575:layer29:\n576:source29:\n577:audio29:\n582:fx_l29_fx1:\n583:fx_l29_fx2:\n584:layer30:\n585:source30:\n586:audio30:\n591:fx_l30_fx1:\n592:fx_l30_fx2:\n593:layer31:\n594:source31:\n595:audio31:\n600:fx_l31_fx1:\n601:fx_l31_fx2:\n602:layer32:\n603:source32:\n604:audio32:\n609:fx_l32_fx1:\n610:fx_l32_fx2:\n611:citp:\n612:monitor:\n613:file_watch:\n614:alsamixer:\n615:stack1:\n617:stack2:\n619:stack3:\n621:stack4:\n623:stack5:\n625:stack6:\n627:stack7:\n629:stack8:\n631:cue1:\n632:gpu1:\n643:gpu2:\n654:ltc1:\n656:ltc2:\n658:ltc3:\n660:ltc4:\n662:clocksv:\n663:timebasesv:\n664:net1:\n665:net2:\n666:net3:\n667:net4:\n668:performance_watch:\n669:io_monitor:\n)`
	p := parseMsg(msg)
	if p == nil {
		t.Errorf("Error parsing picturall message")
	}

	enums := p.ParseEnum()

	if enums == nil {
		t.Errorf("Error parsing enums")
	}

	if len(enums) != 206 {
		t.Errorf("Wrong number of enums parsed, got %d expected 206", len(enums))
	}

	if enums[359] != "layer5" {
		t.Errorf("Wrong enum number 359: got %s expected layer5", enums[359])
	}
}

func TestParseCollection(t *testing.T) {
	state = &State{}
	state.reset()
	msg := `MSG(100003, 1, 39, collection=21, slot=29, file="/picturall/media/compo/photocompo.mp4", name="assembly photocompo.mp4", description=")`
	p := parseMsg(msg)
	if p == nil {
		t.Errorf("Error parsing picturall message")
	}

	p.parseCollection()

	if len(state.mcTelnet[21]) != 1 {
		t.Errorf("Didn't get media in collection")
	}

	if state.mcTelnet[21][29] == nil {
		t.Errorf("Media not in right slot")
	}

	if state.mcTelnet[21][29].Name != `assembly photocompo.mp4` {
		t.Errorf("Media name not correct")
	}
}
