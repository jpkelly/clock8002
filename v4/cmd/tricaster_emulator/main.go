package main

// A really dumb emulator for testing tricaster integration
import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/v1/dictionary", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		fmt.Println("key =>", key)

		switch key {
		case "ddr_timecode":
			io.WriteString(w, ddrTimecode)
		case "ddr_playlist":
			io.WriteString(w, playlist)
		case "switcher":
			io.WriteString(w, switcher)
		}
	})

	http.HandleFunc("/v1/datalink", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, datalink)
	})

	http.ListenAndServe(":3000", nil)
}

var datalink = `
<datalink_values>
<data>
<key>Session Title Name</key>
<value>Company/Team name here ...</value>
</data>
<data>
<key>Session Title Description</key>
<value>Company/Team description here ...</value>
</data>
<data>
<key>Session Title Image</key>
<value>
C:\ProgramData\NewTek\Media\Stills\Default\Default Title Image.png
</value>
</data>
<data>
<key>WebKey 01</key>
<value>
Key Name Image Key Name Image 01   07   02   08   03   09   04   10   05   11   06
</value>
</data>
<data>
<key>WebKey 02</key>
<value/>
</data>
<data>
<key>WebKey 03</key>
<value/>
</data>
<data>
<key>WebKey 04</key>
<value/>
</data>
<data>
<key>WebKey 05</key>
<value/>
</data>
<data>
<key>WebKey 06</key>
<value/>
</data>
<data>
<key>WebKey 07</key>
<value>
D:\Sessions\2021-12-20\Stills\Received\Web\Web File.png
</value>
</data>
<data>
<key>WebKey 08</key>
<value/>
</data>
<data>
<key>WebKey 09</key>
<value/>
</data>
<data>
<key>WebKey 10</key>
<value/>
</data>
<data>
<key>WebKey 11</key>
<value/>
</data>
<data>
<key>WebKey 12</key>
<value/>
</data>
<data>
<key>PTZ PGM Alias</key>
<value/>
</data>
<data>
<key>PTZ PGM Comment</key>
<value/>
</data>
<data>
<key>PTZ PREV Alias</key>
<value/>
</data>
<data>
<key>PTZ PREV Comment</key>
<value/>
</data>
<data>
<key>PGM Source Name</key>
<value>input 1</value>
</data>
<data>
<key>PGM Source Comment</key>
<value>Enter a Comment for INPUT 25</value>
</data>
<data>
<key>ME1 A Source Name</key>
<value>REMOTE 3</value>
</data>
<data>
<key>ME1 A Source Comment</key>
<value>Enter a Comment for INPUT 11</value>
</data>
<data>
<key>ME1 B Source Name</key>
<value>STUDIO PC B</value>
</data>
<data>
<key>ME1 B Source Comment</key>
<value>Enter a Comment for INPUT 8</value>
</data>
<data>
<key>ME1 C Source Name</key>
<value/>
</data>
<data>
<key>ME1 C Source Comment</key>
<value/>
</data>
<data>
<key>ME1 D Source Name</key>
<value/>
</data>
<data>
<key>ME1 D Source Comment</key>
<value/>
</data>
<data>
<key>ME2 A Source Name</key>
<value>REMOTE 1</value>
</data>
<data>
<key>ME2 A Source Comment</key>
<value>Enter a Comment for INPUT 9</value>
</data>
<data>
<key>ME2 B Source Name</key>
<value>STUDIO PC A</value>
</data>
<data>
<key>ME2 B Source Comment</key>
<value>Enter a Comment for INPUT 7</value>
</data>
<data>
<key>ME2 C Source Name</key>
<value/>
</data>
<data>
<key>ME2 C Source Comment</key>
<value/>
</data>
<data>
<key>ME2 D Source Name</key>
<value/>
</data>
<data>
<key>ME2 D Source Comment</key>
<value/>
</data>
<data>
<key>ME3 A Source Name</key>
<value>camera 1</value>
</data>
<data>
<key>ME3 A Source Comment</key>
<value>Enter a Comment for INPUT 1</value>
</data>
<data>
<key>ME3 B Source Name</key>
<value>STUDIO PC B</value>
</data>
<data>
<key>ME3 B Source Comment</key>
<value>Enter a Comment for INPUT 8</value>
</data>
<data>
<key>ME3 C Source Name</key>
<value/>
</data>
<data>
<key>ME3 C Source Comment</key>
<value/>
</data>
<data>
<key>ME3 D Source Name</key>
<value/>
</data>
<data>
<key>ME3 D Source Comment</key>
<value/>
</data>
<data>
<key>Session Name</key>
<value>2021-12-20</value>
</data>
<data>
<key>Session Type</key>
<value>1080/59.94p</value>
</data>
<data>
<key>Session Encoding</key>
<value>NTSC</value>
</data>
<data>
<key>Session Aspect Ratio</key>
<value>16:9</value>
</data>
<data>
<key>NtkHomeScore</key>
<value>0</value>
</data>
<data>
<key>NtkClock</key>
<value>0:00</value>
</data>
<data>
<key>NtkHomeTeamName</key>
<value>Home</value>
</data>
<data>
<key>NtkGuestScore</key>
<value>0</value>
</data>
<data>
<key>NtkGuestTeamName</key>
<value>Guest</value>
</data>
<data>
<key>ME4 A Source Name</key>
<value>INPUT 2</value>
</data>
<data>
<key>ME4 A Source Comment</key>
<value>Enter a Comment for INPUT 2</value>
</data>
<data>
<key>ME4 B Source Name</key>
<value>camera 1</value>
</data>
<data>
<key>ME4 B Source Comment</key>
<value>Enter a Comment for INPUT 1</value>
</data>
<data>
<key>ME4 C Source Name</key>
<value/>
</data>
<data>
<key>ME4 C Source Comment</key>
<value/>
</data>
<data>
<key>ME4 D Source Name</key>
<value/>
</data>
<data>
<key>ME4 D Source Comment</key>
<value/>
</data>
<data>
<key>ME5 A Source Name</key>
<value>STUDIO PC B</value>
</data>
<data>
<key>ME5 A Source Comment</key>
<value>Enter a Comment for INPUT 8</value>
</data>
<data>
<key>ME5 B Source Name</key>
<value>REMOTE 1</value>
</data>
<data>
<key>ME5 B Source Comment</key>
<value>Enter a Comment for INPUT 9</value>
</data>
<data>
<key>ME5 C Source Name</key>
<value/>
</data>
<data>
<key>ME5 C Source Comment</key>
<value/>
</data>
<data>
<key>ME5 D Source Name</key>
<value/>
</data>
<data>
<key>ME5 D Source Comment</key>
<value/>
</data>
<data>
<key>ME6 A Source Name</key>
<value>camera 3</value>
</data>
<data>
<key>ME6 A Source Comment</key>
<value>Enter a Comment for INPUT 3</value>
</data>
<data>
<key>ME6 B Source Name</key>
<value>BRAIN SDI 2</value>
</data>
<data>
<key>ME6 B Source Comment</key>
<value>Enter a Comment for INPUT 22</value>
</data>
<data>
<key>ME6 C Source Name</key>
<value/>
</data>
<data>
<key>ME6 C Source Comment</key>
<value/>
</data>
<data>
<key>ME6 D Source Name</key>
<value/>
</data>
<data>
<key>ME6 D Source Comment</key>
<value/>
</data>
<data>
<key>ME7 A Source Name</key>
<value>INPUT 2</value>
</data>
<data>
<key>ME7 A Source Comment</key>
<value>Enter a Comment for INPUT 2</value>
</data>
<data>
<key>ME7 B Source Name</key>
<value>BLACK</value>
</data>
<data>
<key>ME7 B Source Comment</key>
<value>Enter a Comment for BLACK</value>
</data>
<data>
<key>ME7 C Source Name</key>
<value/>
</data>
<data>
<key>ME7 C Source Comment</key>
<value/>
</data>
<data>
<key>ME7 D Source Name</key>
<value/>
</data>
<data>
<key>ME7 D Source Comment</key>
<value/>
</data>
<data>
<key>ME8 A Source Name</key>
<value>DDR 3</value>
</data>
<data>
<key>ME8 A Source Comment</key>
<value>Enter a Comment for DDR 3</value>
</data>
<data>
<key>ME8 B Source Name</key>
<value>M/E 1</value>
</data>
<data>
<key>ME8 B Source Comment</key>
<value>Enter a Comment for M/E 1</value>
</data>
<data>
<key>ME8 C Source Name</key>
<value/>
</data>
<data>
<key>ME8 C Source Comment</key>
<value/>
</data>
<data>
<key>ME8 D Source Name</key>
<value/>
</data>
<data>
<key>ME8 D Source Comment</key>
<value/>
</data>
<data>
<key>SCRIPT_ShowTitle</key>
<value>Springdale Morning Mic</value>
</data>
<data>
<key>SCRIPT_Title</key>
<value>Springdale Morning Mic</value>
</data>
<data>
<key>SCRIPT_ShowDescription</key>
<value>Springdale Institute of Technology’s Daily Show</value>
</data>
<data>
<key>SCRIPT_Heading1</key>
<value>Springdale Institute of Technology’s Daily Show</value>
</data>
<data>
<key>SCRIPT_ShowSegment</key>
<value/>
</data>
<data>
<key>SCRIPT_Heading2</key>
<value/>
</data>
<data>
<key>SCRIPT_CueName</key>
<value/>
</data>
<data>
<key>SCRIPT_Heading3</key>
<value/>
</data>
<data>
<key>SCRIPT_Info</key>
<value/>
</data>
<data>
<key>SCRIPT_Subtitle</key>
<value/>
</data>
<data>
<key>SCRIPT_CueDescription</key>
<value/>
</data>
<data>
<key>SCRIPT_Heading4</key>
<value/>
</data>
<data>
<key>SCRIPT_Heading5</key>
<value/>
</data>
<data>
<key>SCRIPT_Heading6</key>
<value/>
</data>
<data>
<key>SCRIPT_Emphasis</key>
<value/>
</data>
<data>
<key>DDR4 Clip Alias</key>
<value>EEC placeholder 5994_4224</value>
</data>
<data>
<key>DDR4 Clip Comment</key>
<value/>
</data>
<data>
<key>DDR3 Clip Alias</key>
<value>River Rocks</value>
</data>
<data>
<key>DDR3 Clip Comment</key>
<value/>
</data>
<data>
<key>DDR1 Clip Alias</key>
<value>tricaster spin</value>
</data>
<data>
<key>DDR1 Clip Comment</key>
<value/>
</data>
<data>
<key>SOUND Clip Alias</key>
<value/>
</data>
<data>
<key>SOUND Clip Comment</key>
<value/>
</data>
<data>
<key>DDR2 Clip Alias</key>
<value>Lot of fun</value>
</data>
<data>
<key>DDR2 Clip Comment</key>
<value>paina tata jos uskallat</value>
</data>
<data>
<key>Time</key>
<value>4:20:46 PM</value>
</data>
<data>
<key>Time (filename)</key>
<value>042046 ip.</value>
</data>
<data>
<key>Seconds (short)</key>
<value>46</value>
</data>
<data>
<key>Seconds (long)</key>
<value>46</value>
</data>
<data>
<key>Minutes (short)</key>
<value>20</value>
</data>
<data>
<key>Minutes (long)</key>
<value>20</value>
</data>
<data>
<key>Hours (short)</key>
<value>4</value>
</data>
<data>
<key>Hours (long)</key>
<value>04</value>
</data>
<data>
<key>Hours (short, 24h)</key>
<value>16</value>
</data>
<data>
<key>Hours (long, 24h)</key>
<value>16</value>
</data>
<data>
<key>AM/PM (short)</key>
<value>i</value>
</data>
<data>
<key>AM/PM (long)</key>
<value>ip.</value>
</data>
<data>
<key>Date</key>
<value>09.09.2022</value>
</data>
<data>
<key>Date (filename)</key>
<value>2022-09-09</value>
</data>
<data>
<key>Date (MM/DD/YY)</key>
<value>09.09.22</value>
</data>
<data>
<key>Date (MM/DD/YYYY)</key>
<value>09.09.2022</value>
</data>
<data>
<key>Date (DD/MM/YY)</key>
<value>09.09.22</value>
</data>
<data>
<key>Date (DD/MM/YYYY)</key>
<value>09.09.2022</value>
</data>
<data>
<key>SystemDate</key>
<value>9.9.2022</value>
</data>
<data>
<key>Day (short, numeric)</key>
<value>9</value>
</data>
<data>
<key>Day (long, numeric)</key>
<value>09</value>
</data>
<data>
<key>Day (short)</key>
<value>pe</value>
</data>
<data>
<key>Day (long)</key>
<value>perjantai</value>
</data>
<data>
<key>Month (short, numeric)</key>
<value>9</value>
</data>
<data>
<key>Month (long, numeric)</key>
<value>09</value>
</data>
<data>
<key>Month (short)</key>
<value>syys</value>
</data>
<data>
<key>Month (long)</key>
<value>syyskuu</value>
</data>
<data>
<key>DayMonth (short, long)</key>
<value>1 syyskuuta</value>
</data>
<data>
<key>MonthDay (long, short)</key>
<value>syyskuuta 1</value>
</data>
<data>
<key>Year (short)</key>
<value>22</value>
</data>
<data>
<key>Year (full)</key>
<value>2022</value>
</data>
<data>
<key>Next Event</key>
<value>END</value>
</data>
<data>
<key>Time Until Next Event</key>
<value> 01:39:13</value>
</data>
<data>
<key>NtkHomeFouls</key>
<value>0</value>
</data>
<data>
<key>NtkGuestFouls</key>
<value>0</value>
</data>
<data>
<key>NtkHomeTOTotal</key>
<value>3</value>
</data>
<data>
<key>NtkPossession</key>
<value>Home</value>
</data>
<data>
<key>NtkGuestTOTotal</key>
<value>3</value>
</data>
<data>
<key>NtkShotClock</key>
<value>25</value>
</data>
<data>
<key>NtkHomeShotsOnGoal</key>
<value>0</value>
</data>
<data>
<key>NtkGuestShotsOnGoal</key>
<value>0</value>
</data>
<data>
<key>NtkHockeyHomePenaltyClock1</key>
<value>2:00</value>
</data>
<data>
<key>NtkHockeyGuestPenaltyClock1</key>
<value>2:00</value>
</data>
<data>
<key>NtkHockeyHomePenaltyClock2</key>
<value>2:00</value>
</data>
<data>
<key>NtkHockeyGuestPenaltyClock2</key>
<value>2:00</value>
</data>
</datalink_values>
`

var ddrTimecode = `
<timecode>
<ddr2 clip_seconds_elapsed="5.005" clip_seconds_remaining="-0.005" clip_embedded_timecode="5.005" clip_in="0" clip_out="5" file_duration="5" play_speed="1" clip_framerate="59.940059940059939" playlist_seconds_elapsed="5.005" playlist_seconds_remaining="0" preset_index="0" clip_index="1" num_clips="4"/>
<ddr1 clip_seconds_elapsed="1.333333" clip_seconds_remaining="4.633334" clip_embedded_timecode="1.3333334" clip_in="0" clip_out="5.966667" file_duration="5.966667" play_speed="1" clip_framerate="30" playlist_seconds_elapsed="1.334667" playlist_seconds_remaining="4.6046" preset_index="11" clip_index="0" num_clips="1"/>
<ddr3 clip_seconds_elapsed="4.2" clip_seconds_remaining="15.82" clip_embedded_timecode="4.2" clip_in="0" clip_out="20.02" file_duration="20.02" play_speed="1" clip_framerate="50" playlist_seconds_elapsed="4.2042" playlist_seconds_remaining="15.8158" preset_index="0" clip_index="0" num_clips="10"/>
<ddr4 clip_seconds_elapsed="58.425033" clip_seconds_remaining="1.685017" clip_embedded_timecode="58.4250334" clip_in="0" clip_out="60.11005" file_duration="60.11005" play_speed="1" clip_framerate="59.94006" playlist_seconds_elapsed="58.425033" playlist_seconds_remaining="1.685017" preset_index="0" clip_index="4" num_clips="5"/>
</timecode>
`

var playlist = `
<playlist>
<sound custom_short_name="SOUND" custom_long_name="SOUND">
<playlist contains_playhead="true" is_selected="true" preset_index="0" current_clip="-1"/>
</sound>
<ddr2 custom_short_name="DDR 2" custom_long_name="DDR 2">
<playlist contains_playhead="true" is_selected="true" preset_index="0" current_clip="1">
<clip alias="Lot of fun" comment="kokkelis hokkelis mitas lahdit uu jee" frame_rate="59.940059940059939" file_length="5" filename="D:\Sessions\2021-12-20\presets\DDR2\Preset0\2201051055.cgxml2" audio_level="0" clip_length="5"/>
<clip alias="Lot of fun" comment="paina tata jos uskallat" frame_rate="59.940059940059939" file_length="5" filename="D:\Sessions\2021-12-20\presets\DDR2\Preset0\2201251139-0002.cgxml2" audio_level="0" clip_length="5"/>
<clip alias="Heat 001" frame_rate="59.940059940059939" file_length="5" filename="D:\Sessions\2021-12-20\presets\DDR2\Preset0\2201251139.cgxml2" audio_level="0" clip_length="5"/>
<clip alias="Pentti Aaraspelto" comment="paina tata jos uskallat" frame_rate="59.940059940059939" file_length="5" filename="D:\Sessions\2021-12-20\presets\DDR2\Preset0\2201030205.cgxml2" audio_level="0" clip_length="5"/>
</playlist>
</ddr2>
<ddr1 custom_short_name="DDR 1" custom_long_name="DDR 1">
<playlist contains_playhead="true" is_selected="true" preset_index="11" current_clip="0">
<clip alias="tricaster spin" frame_rate="30" file_length="5.966667" filename="d:\media\clips\newtek (ntsc)\logos\tricaster spin.mov" audio_level="0" clip_length="5.966667"/>
</playlist>
</ddr1>
<ddr3 custom_short_name="DDR 3" custom_long_name="DDR 3">
<playlist contains_playhead="true" is_selected="true" preset_index="0" current_clip="0">
<clip alias="River Rocks" frame_rate="50" file_length="20.02" filename="D:\Media\Clips\NewTek (PAL)\Clips\River Rocks.mov" audio_level="0" clip_length="20.02"/>
<clip alias="River Roots" frame_rate="50" file_length="15.02" filename="D:\Media\Clips\NewTek (PAL)\Clips\River Roots.mov" audio_level="0" clip_length="15.02"/>
<clip alias="River Ducks" frame_rate="50" file_length="20.48" filename="D:\Media\Clips\NewTek (PAL)\Clips\River Ducks.mov" audio_level="0" clip_length="20.48"/>
<clip alias="River Bridge" frame_rate="50" file_length="20.02" filename="D:\Media\Clips\NewTek (PAL)\Clips\River Bridge.mov" audio_level="0" clip_length="20.02"/>
<clip alias="River Flow" frame_rate="50" file_length="15.02" filename="D:\Media\Clips\NewTek (PAL)\Clips\River Flow.mov" audio_level="0" clip_length="15.02"/>
<clip alias="Rustling Trees" frame_rate="50" file_length="19.06" filename="D:\Media\Clips\NewTek (PAL)\Clips\Rustling Trees.mov" audio_level="0" clip_length="19.06"/>
<clip alias="Vegas Bellagio Pool" frame_rate="50" file_length="20.02" filename="D:\Media\Clips\NewTek (PAL)\Clips\Vegas Bellagio Pool.mov" audio_level="0" clip_length="20.02"/>
<clip alias="Vegas Night Lights" frame_rate="50" file_length="20.02" filename="D:\Media\Clips\NewTek (PAL)\Clips\Vegas Night Lights.mov" audio_level="0" clip_length="20.02"/>
<clip alias="Vegas Pool Lights" frame_rate="50" file_length="20.02" filename="D:\Media\Clips\NewTek (PAL)\Clips\Vegas Pool Lights.mov" audio_level="0" clip_length="20.02"/>
<clip alias="Waterfall" frame_rate="50" file_length="16" filename="D:\Media\Clips\NewTek (PAL)\Clips\Waterfall.mov" audio_level="0" clip_length="16"/>
</playlist>
</ddr3>
<ddr4 custom_short_name="DDR 4" custom_long_name="DDR 4">
<playlist contains_playhead="true" is_selected="true" preset_index="0" current_clip="4">
<clip alias="1920x1080_5994p alpha 4220" frame_rate="59.94006" file_length="20.036683" filename="D:\Sessions\2021-12-20\Clips\testi\1920x1080_5994p alpha 4220.avi" audio_level="0" clip_length="20.036683"/>
<clip alias="1920x1080_5994p alpha 4204" frame_rate="59.94006" file_length="20.036683" filename="D:\Sessions\2021-12-20\Clips\testi\1920x1080_5994p alpha 4204.avi" audio_level="0" clip_length="20.036683"/>
<clip alias="Audio Video Sync Test 60 FPS" frame_rate="59.94006" file_length="120.0199" filename="D:\Sessions\2021-12-20\Clips\testi\Audio Video Sync Test 60 FPS.avi" audio_level="0" clip_length="120.0199"/>
<clip alias="EEC placeholder 5994 4204" frame_rate="59.94006" file_length="60.11005" filename="D:\Sessions\2021-12-20\Clips\testi\EEC placeholder 5994 4204.avi" audio_level="0" clip_length="60.11005"/>
<clip alias="EEC placeholder 5994_4224" frame_rate="59.94006" file_length="60.11005" filename="D:\Sessions\2021-12-20\Clips\testi\EEC placeholder 5994_4224.avi" audio_level="0" clip_length="60.11005"/>
</playlist>
</ddr4>
</playlist>
`

var switcher = `
<switcher_update main_source="DDR3" preview_source="INPUT6" effect="" iso_label="INPUT 6" button_label="INPUT 6">
<tbar position="0" speed="0" current_speed="0.40099999999999991" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
<switcher_overlays>
<overlay z_order_position="0" source="INPUT15" effect="" iso_label="VMIX 1" button_label="VMIX 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="1" source="INPUT4" effect="" iso_label="PPT" button_label="PPT">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="2" source="INPUT26" effect="" iso_label="INPUT 26" button_label="INPUT 26">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="3" source="INPUT15" effect="" iso_label="VMIX 1" button_label="VMIX 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
</switcher_overlays>
<inputs>
<physical_input physical_input_number="Input1" iso_label="camera 1" button_label="camera 1">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input2" iso_label="INPUT 2" button_label="INPUT 2">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input3" iso_label="camera 3" button_label="camera 3">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input4" iso_label="PPT" button_label="PPT">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input5" iso_label="INPUT 5" button_label="INPUT 5">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input6" iso_label="INPUT 6" button_label="INPUT 6">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input7" iso_label="STUDIO PC A" button_label="STUDIO PC A">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input8" iso_label="STUDIO PC B" button_label="STUDIO PC B">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input9" iso_label="REMOTE 1" button_label="REMOTE 1">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input10" iso_label="REMOTE 2" button_label="REMOTE 2">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input11" iso_label="REMOTE 3" button_label="REMOTE 3">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input12" iso_label="REMOTE 4" button_label="REMOTE 4">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input13" iso_label="DEMO 1" button_label="DEMO 1">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input14" iso_label="DEMO 2" button_label="DEMO 2">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input15" iso_label="VMIX 1" button_label="VMIX 1">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input16" iso_label="NC2" button_label="NC2">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input17" iso_label="CUSTOMER 1" button_label="CUSTOMER 1">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input18" iso_label="REMOTE 5" button_label="REMOTE 5">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input19" iso_label="LOGO PC" button_label="LOGO PC">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input20" iso_label="NOTES" button_label="NOTES">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input21" iso_label="BRAIN SDI 1" button_label="BRAIN SDI 1">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input22" iso_label="BRAIN SDI 2" button_label="BRAIN SDI 2">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input23" iso_label="INPUT 23" button_label="INPUT 23">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input24" iso_label="INPUT 24" button_label="INPUT 24">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input25" iso_label="ddr1" button_label="ddr1">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input26" iso_label="INPUT 26" button_label="INPUT 26">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input27" iso_label="INPUT 27" button_label="INPUT 27">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input28" iso_label="INPUT 28" button_label="INPUT 28">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input29" iso_label="INPUT 29" button_label="INPUT 29">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input30" iso_label="INPUT 30" button_label="INPUT 30">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input31" iso_label="INPUT 31" button_label="INPUT 31">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="Input32" iso_label="INPUT 32" button_label="INPUT 32">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR1" iso_label="BUFFER 1" button_label="BUFFER 1">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR2" iso_label="BUFFER 2" button_label="BUFFER 2">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR3" iso_label="BUFFER 3" button_label="BUFFER 3">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR4" iso_label="BUFFER 4" button_label="BUFFER 4">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR5" iso_label="BUFFER 5" button_label="BUFFER 5">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR6" iso_label="BUFFER 6" button_label="BUFFER 6">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR7" iso_label="BUFFER 7" button_label="BUFFER 7">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR8" iso_label="BUFFER 8" button_label="BUFFER 8">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR9" iso_label="BUFFER 9" button_label="BUFFER 9">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR10" iso_label="BUFFER 10" button_label="BUFFER 10">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR11" iso_label="BUFFER 11" button_label="BUFFER 11">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR12" iso_label="BUFFER 12" button_label="BUFFER 12">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR13" iso_label="BUFFER 13" button_label="BUFFER 13">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR14" iso_label="BUFFER 14" button_label="BUFFER 14">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="BFR15" iso_label="BUFFER 15" button_label="BUFFER 15">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="DDR1_A" iso_label="DDR 1_A" button_label="DDR 1_A">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="DDR1_B" iso_label="DDR 1_B" button_label="DDR 1_B">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="DDR2_A" iso_label="DDR 2_A" button_label="DDR 2_A">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="DDR2_B" iso_label="DDR 2_B" button_label="DDR 2_B">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="DDR3_A" iso_label="DDR 3_A" button_label="DDR 3_A">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="DDR3_B" iso_label="DDR 3_B" button_label="DDR 3_B">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="DDR4_A" iso_label="DDR 4_A" button_label="DDR 4_A">
<playback speed="0" position="0"/>
</physical_input>
<physical_input physical_input_number="DDR4_B" iso_label="DDR 4_B" button_label="DDR 4_B">
<playback speed="0" position="0"/>
</physical_input>
<simulated_input simulated_input_number="DDR1" effect=" ">
<source_a a="47"/>
<source_b b="48"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="1" source="DDR2" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="2" source="BFR1" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="3" source="BFR2" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0"/>
</simulated_input>
<simulated_input simulated_input_number="DDR2" effect=" ">
<source_a a="49"/>
<source_b b="50"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="1" source="DDR2" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="2" source="BFR1" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="3" source="BFR2" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0"/>
</simulated_input>
<simulated_input simulated_input_number="DDR3" effect=" ">
<source_a a="51"/>
<source_b b="52"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="1" source="DDR2" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="2" source="BFR1" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="3" source="BFR2" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0"/>
</simulated_input>
<simulated_input simulated_input_number="DDR4" effect=" ">
<source_a a="53"/>
<source_b b="54"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="1" source="DDR2" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="2" source="BFR1" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="3" source="BFR2" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0"/>
</simulated_input>
<simulated_input simulated_input_number="V1" effect="">
<source_a a="10"/>
<source_b b="7"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" effect="" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="1" source="DDR2" effect="" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="2" source="BFR1" effect="" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="3" source="BFR2" effect="" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</simulated_input>
<simulated_input simulated_input_number="V2" effect="">
<source_a a="8"/>
<source_b b="6"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" effect="" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="1" source="DDR2" effect="" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="2" source="BFR1" effect="" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="3" source="BFR2" effect="" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</simulated_input>
<simulated_input simulated_input_number="V3" effect="">
<source_a a="0"/>
<source_b b="7"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" effect="" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="1" source="DDR2" effect="" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="2" source="BFR1" effect="" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="3" source="BFR2" effect="" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</simulated_input>
<simulated_input simulated_input_number="V4" effect="">
<source_a a="1"/>
<source_b b="0"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" effect="" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="1" source="DDR2" effect="" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="2" source="BFR1" effect="" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="3" source="BFR2" effect="" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</simulated_input>
<simulated_input simulated_input_number="V5" effect="">
<source_a a="7"/>
<source_b b="8"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" effect="" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="1" source="DDR2" effect="" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="2" source="BFR1" effect="" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="3" source="BFR2" effect="" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</simulated_input>
<simulated_input simulated_input_number="V6" effect="">
<source_a a="2"/>
<source_b b="21"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="INPUT21" effect="" iso_label="BRAIN SDI 1" button_label="BRAIN SDI 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="1" source="INPUT22" effect="" iso_label="BRAIN SDI 2" button_label="BRAIN SDI 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="2" source="INPUT8" effect="" iso_label="STUDIO PC B" button_label="STUDIO PC B">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="3" source="INPUT31" effect="" iso_label="INPUT 31" button_label="INPUT 31">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</simulated_input>
<simulated_input simulated_input_number="V7" effect="">
<source_a a="1"/>
<source_b b="-1"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" effect="" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="1" source="DDR2" effect="" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="2" source="BFR1" effect="" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="3" source="BFR2" effect="" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</simulated_input>
<simulated_input simulated_input_number="V8" effect="">
<source_a a="-1"/>
<source_b b="0"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="INPUT1" effect="" iso_label="camera 1" button_label="camera 1">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="1" source="INPUT2" effect="" iso_label="INPUT 2" button_label="INPUT 2">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="2" source="INPUT5" effect="" iso_label="INPUT 5" button_label="INPUT 5">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="3" source="INPUT6" effect="" iso_label="INPUT 6" button_label="INPUT 6">
<tbar position="0" speed="0" current_speed="1" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0" current_speed="1.9340000000000002" slow_speed="2" medium_speed="1" fast_speed="0.5"/>
</simulated_input>
<simulated_input simulated_input_number="preview" effect=" ">
<source_a a="5"/>
<source_b b="24"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="INPUT15" iso_label="VMIX 1" button_label="VMIX 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="1" source="INPUT4" iso_label="PPT" button_label="PPT">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="2" source="INPUT26" iso_label="INPUT 26" button_label="INPUT 26">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="3" source="INPUT15" iso_label="VMIX 1" button_label="VMIX 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0"/>
</simulated_input>
<simulated_input simulated_input_number="me_preview" effect=" ">
<source_a a="0"/>
<source_b b="-1"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="INPUT1" iso_label="camera 1" button_label="camera 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="1" source="INPUT2" iso_label="INPUT 2" button_label="INPUT 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="2" source="INPUT5" iso_label="INPUT 5" button_label="INPUT 5">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="3" source="INPUT6" iso_label="INPUT 6" button_label="INPUT 6">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0"/>
</simulated_input>
<simulated_input simulated_input_number="me_follow" effect=" ">
<source_a a="v11"/>
<source_b b="v1"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR1" iso_label="DDR 1" button_label="DDR 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="1" source="DDR2" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="2" source="BFR1" iso_label="BUFFER 1" button_label="BUFFER 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="3" source="BFR2" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0"/>
</simulated_input>
<simulated_input simulated_input_number="previz" effect=" ">
<source_a a="v0"/>
<source_b b="v1"/>
<source_c c="v2"/>
<source_d d="v3"/>
<overlay z_order_position="0" source="DDR2" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="1" source="DDR2" iso_label="DDR 2" button_label="DDR 2">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="2" source="INPUT13" iso_label="DEMO 1" button_label="DEMO 1">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="3" source="BFR2" iso_label="BUFFER 2" button_label="BUFFER 2">
<tbar position="1" speed="0"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0"/>
</simulated_input>
<simulated_input simulated_input_number="web_follow" effect=" ">
<source_a a="v4"/>
<source_b b="-1"/>
<source_c c="-1"/>
<source_d d="-1"/>
<overlay z_order_position="0" source="Black" iso_label="BLACK" button_label="BLACK">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="1" source="Black" iso_label="BLACK" button_label="BLACK">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="2" source="Black" iso_label="BLACK" button_label="BLACK">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="3" source="Black" iso_label="BLACK" button_label="BLACK">
<tbar position="0" speed="0"/>
</overlay>
<overlay z_order_position="4" source="">
<tbar position="0" speed="0"/>
</overlay>
<tbar position="0" speed="0"/>
</simulated_input>
</inputs>
</switcher_update>
`
