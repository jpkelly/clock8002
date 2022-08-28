package vmix

import (
	"encoding/xml"
	"testing"
)

func TestParseXML(t *testing.T) {
	// vMix example API xml
	xml1 := `<vmix><version>11.0.0.16</version><inputs><input key="26cae087-b7b6-4d45-98e4-de03ab4feb6b" number="1" type="Xaml" title="NewsHD.xaml" state="Paused" position="0" duration="0" muted="True" loop="False" selectedIndex="0">NewsHD.xaml<text index="0" name="Headline">Hello</text><text index="1" name="Description">Hello</text></input><input key="55cbe357-a801-4d54-8ff2-08ee68766fae" number="2" type="VirtualSet" title="LateNightNews" state="Paused" position="0" duration="0" muted="True" loop="False" selectedIndex="0">LateNightNews<overlay index="0" key="2fe8ff9d-e400-4504-85ab-df7c17a1edd4"/><overlay index="1" key="20e4ee9a-05cc-4f58-bb69-cd179e1c1958"/><overlay index="2" key="94b88db0-c5cd-49d8-98a2-27d83d4bf3fe"/></input></inputs><overlays><overlay number="1"/><overlay number="2">1</overlay><overlay number="3"/><overlay number="4"/><overlay number="5"/><overlay number="6"/></overlays><preview>1</preview><active>2</active><fadeToBlack>False</fadeToBlack><transitions><transition number="1" effect="Fade" duration="500"/><transition number="2" effect="Wipe" duration="500"/><transition number="3" effect="Fly" duration="500"/><transition number="4" effect="CubeZoom" duration="3000"/></transitions><recording>False</recording><external>False</external><streaming>False</streaming><playList>False</playList><multiCorder>False</multiCorder></vmix>`
	// video playing
	xml2 := `<?xml version="1.0" encoding="UTF-8"?><vmix><version>25.0.0.34</version><edition>4K</edition><inputs><input key="f542009d-6186-4ca7-b7ac-9c907376ce08" number="1" type="Video" title="RW_2021_TF3_C2.mp4" shortTitle="RW_2021_TF3_C2.mp4" state="Running" position="426080" duration="4679792" loop="True" muted="False" volume="100" balance="0" solo="False" audiobusses="M" meterF1="0.1882018" meterF2="0.1882018" gainDb="0">RW_2021_TF3_C2.mp4</input><input key="49130aca-f0d8-4177-bc57-5b24e530bcbc" number="2" type="Blank" title="Blank" shortTitle="Blank" state="Paused" position="0" duration="0" loop="False">Blank</input></inputs><overlays><overlay number="1" /><overlay number="2" /><overlay number="3" /><overlay number="4" /><overlay number="5" /><overlay number="6" /><overlay number="7" /><overlay number="8" /></overlays><preview>2</preview><active>1</active><fadeToBlack>False</fadeToBlack><transitions><transition number="1" effect="Fade" duration="500" /><transition number="2" effect="Merge" duration="1000" /><transition number="3" effect="Wipe" duration="1000" /><transition number="4" effect="CubeZoom" duration="1000" /></transitions><recording>False</recording><external>False</external><streaming>False</streaming><playList>False</playList><multiCorder>False</multiCorder><fullscreen>False</fullscreen><audio><master volume="100" muted="False" meterF1="0.2188829" meterF2="0.2188829" headphonesVolume="100" /></audio><dynamic><input1 /><input2 /><input3 /><input4 /><value1 /><value2 /><value3 /><value4 /></dynamic></vmix>`
	// video list
	xml3 := `<vmix><version>25.0.0.34</version><edition>4K</edition><inputs><input key="a8208de9-69ae-4bec-b548-deb980a2d54a" number="1" type="VideoList" title="List - RW_2021_TF3_B3.mp4" shortTitle="List" state="Running" position="32860" duration="3913259" loop="False" muted="False" volume="100" balance="0" solo="False" audiobusses="M" meterF1="0.2572337" meterF2="0.2572337" gainDb="0" selectedIndex="1">List - RW_2021_TF3_B3.mp4<list><item selected="true">C:\Users\genadmin\Desktop\RW_2021_TF3_B3.mp4</item><item>C:\Users\genadmin\Desktop\RW_2021_TF3_C2.mp4</item></list></input><input key="ce42dabe-a9ae-44ba-819a-06eaf1c90646" number="2" type="Blank" title="Blank" shortTitle="Blank" state="Paused" position="0" duration="0" loop="False">Blank</input></inputs><overlays><overlay number="1" /><overlay number="2" /><overlay number="3" /><overlay number="4" /><overlay number="5" /><overlay number="6" /><overlay number="7" /><overlay number="8" /></overlays><preview>2</preview><active>1</active><fadeToBlack>False</fadeToBlack><transitions><transition number="1" effect="Fade" duration="500" /><transition number="2" effect="Merge" duration="1000" /><transition number="3" effect="Wipe" duration="1000" /><transition number="4" effect="CubeZoom" duration="1000" /></transitions><recording>False</recording><external>False</external><streaming>False</streaming><playList>False</playList><multiCorder>False</multiCorder><fullscreen>False</fullscreen><audio><master volume="100" muted="False" meterF1="0.2710195" meterF2="0.2710195" headphonesVolume="100" /></audio><dynamic><input1></input1><input2></input2><input3></input3><input4></input4><value1></value1><value2></value2><value3></value3><value4></value4></dynamic></vmix>`

	s := &status{}
	err := xml.Unmarshal([]byte(xml1), s)

	if err != nil {
		t.Errorf("Error parsing example XML: %v", err)
	}

	if s.PGM() != 2 {
		t.Errorf("Input 2 should be PGM")
	}

	if !s.Live(1) {
		t.Errorf("Input 1 should be live as overlay 1")
	}

	if !s.Overlay(1) {
		t.Errorf("Input 1 should be overlay")
	}

	if s.Overlay(2) {
		t.Errorf("Input 2 shouldn't be an overlay, it is in PGM")
	}

	s = &status{}
	err = xml.Unmarshal([]byte(xml2), s)

	if err != nil {
		t.Errorf("Error parsing looping video XML: %v", err)
	}

	if s.PGM() != 1 {
		t.Errorf("Input 1 should be PGM")
	}

	i := s.Inputs[0]

	if !i.Playing() {
		t.Errorf("Input 1 should be playing video")
	}

	if !i.Loop {
		t.Errorf("Input 1 should be looping")
	}

	s = &status{}
	err = xml.Unmarshal([]byte(xml3), s)

	if err != nil {
		t.Errorf("Error parsing videoList XML: %v", err)
	}

	if s.PGM() != 1 {
		t.Errorf("Input 1 should be PGM")
	}

	i = s.Inputs[0]

	if !i.Playing() {
		t.Errorf("Input 1 should be playing video")
	}

	if i.Loop {
		t.Errorf("Input 1 shouldn't be looping")
	}

	if len(i.Items) != 2 {
		t.Errorf("Input 1 should have a playlist of 2 items")
	}

	if i.Selected != 1 {
		t.Errorf("Input 1 should be playing index 1")
	}
}
