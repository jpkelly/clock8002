package main

const configHTML = `
<html>
<head>
	<title>Clock-8001 configuration</title>
</head>
	<body>
		<h1>Clock configuration editor</h1>
		{{if .Errors}}
			<div class="errors">
				<p>
					Following errors prevented the configuration from being saved:
					{{.Errors}}
				</p>
			</div>
		{{end}}
		<div class="config-form">
			<form action="/import" method="post" enctype="multipart/form-data">
				<fieldset>
					<legend>Project links</legend>
					<ul>
						<li><a href="https://gitlab.com/clock-8001/clock-8001/">View the project on gitlab</a></li>
						<li><a href="https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=XUMXUL5RX5MWJ&currency_code=EUR">Support development of clock-8001 via Paypal</a></li>
					</ul>
				</fieldset>

				<fieldset>
					<legend>Config Import / Export</legend>
					<p><a href="/export">Download current configuration.</a></p>
					<label for="import"><span>Import configurations file</span>
						<input type="file" id="import" name="import" />
					</label>
					<input type="submit" value="upload" />
				</fieldset>
			</form>

			<form action="/save" method="post">
				<fieldset>
					<legend>General settings</legend>
					<label for="Face">
						<span>Select the clock face to use</span>
						<select name="Face" id="Face">
							<option value="round" {{if eq .Face "round"}} selected {{end}}>Single round clock</option>
							<option value="dual-round" {{if eq .Face "dual-round"}} selected {{end}}>Dual round clocks</option>
							<option value="text" {{if eq .Face "text"}} selected {{end}}>Text clock with 3 timers</option>
							<option value="single" {{if eq .Face "single"}} selected {{end}}>Text clock with 1 timer</option>
							<option value="192" {{if eq .Face "192"}} selected {{end}}>Small 192x192px round clock</option>
							<option value="144" {{if eq .Face "144"}} selected {{end}}>Small 144x144px round clock</option>
							<option value="288x144" {{if eq .Face "288x144"}} selected {{end}}>Small 288x144px text clock</option>
							<option value="countdown" {{if eq .Face "countdown"}} selected {{end}}>Countdown to static date and time</option>
						</select><br />
					</label>

					{{checkbox "Format12h" "Use 12 hour format for time-of-day display" .EngineOptions.Format12h }}
					{{checkbox "NoARCorrection" "Disable detection of official raspberry pi display for aspect ratio correction" .NoARCorrection}}
					{{checkbox "FullScreen" "Start in full screen mode (ignored on raspberry pi images)" .FullScreen }}
					{{checkbox "Debug" "Output verbose debug information. This will impact performance" .Debug}}

					{{text "Font" "Font filename for round clocks." .Font}}

					<datalist id="FontList">
						{{ range $font := .Fonts }}
							<option>{{$font}}</option>
						{{ end }}
					</datalist>

					<label for="NumberFont">
						<span>Font filename for text clock numbers</span>
						<input list="FontList" type="text" id="NumberFont" name="NumberFont" value="{{.NumberFont}}" />
					</label>

					<label for="LabelFont">
						<span>Font filename for text clock labels</span>
						<input list="FontList" type="text" id="LabelFont" name="LabelFont" value="{{.LabelFont}}" />
					</label>

					<label for="IconFont">
						<span>Font filename for text clock icons</span>
						<input list="FontList" type="text" id="IconFont" name="IconFont" value="{{.IconFont}}" />
					</label>

					{{number "Flash" "Flashing interval in milliseconds for ellapsed countdowns." .EngineOptions.Flash}}
					{{number "Timeout" "Timeout for clearing OSC text display messages, in milliseconds." .EngineOptions.Timeout}}
					{{number "ShowInfo" "Time to show clock information on startup, in seconds." .EngineOptions.ShowInfo}}
					{{text "Background" "Background image filename." .Background}}

					<p>The image needs to be in the correct resolution and either png or jpeg file. Place the
					image in the fat partition and refer to it as /boot/imagename.png</p>

					{{text "BackgroundPath" "Path for OSC selectable background images" .BackgroundPath}}

					<p>The OSC command /clock/background/ can be used to select a numbered background image from this path.
					Files should be named with the number (eg 1.png or 01.jpeg). Supported filetypes are BMP, PNG and JPEG.</p>

					{{color "BackgroundColor" "Background color, used if no background image is provided." .BackgroundColor}}

					{{checkbox "AudioEnabled" "Enable audio cues for expiring countdown timers." .AudioEnabled}}
					{{checkbox "VoiceEnabled" "Enable voice cues for expiring countdown timers and media durations." .VoiceEnabled}}
					{{text "VoiceDir" "Directory to load the voice samples from. Files should be named 1234.wav where 1234 is the time in seconds to trigger the sample." .VoiceDir}}

					{{checkbox "TODBeep" "Enable audio cues for time of day displays on each full hour." .TODBeep}}
					{{text "CountdownTarget" "Target for the countdown clock face (YYYY-MM-DD HH:MM:SS)" .CountdownTarget}}
					{{checkbox "Countup" "Countdown face counts up instead." .Countup}}

				</fieldset>
				<fieldset>
					<legend>Time sources</legend>

				<p>The single round clock uses source 1 as the main display and source 2 as a secondary timer.
				The dual round clock mode uses all four sources, with 1 and 2 in the left clock and 3 and 4 in the right clock.</p>

				<p>The round clocks only support timers as the secondary display source, as others can't be compacted to 4 characters.</p>

				<p>The sources choose their displayed time in the following priority if enabled:
					<ol>
						<li>LTC</li>
						<li>Associated timer if it is running</li>
						<li>Time of day in the selected time zone</li>
						<li>Blank display</li>
					</ol>
				</p>

				<fieldset>
					<legend>Source 1</legend>

					<label for="source1-text">
						<span>Text label for time source</span>
						<input type="text" id="source1-text" name="source1-text" value="{{.EngineOptions.Source1.Text}}" />
					</label>

					{{checkbox "source1-ltc" "Enable LTC input on this source" .EngineOptions.Source1.LTC}}
					{{checkbox "source1-timer" "Enable input from the associated timer" .EngineOptions.Source1.Timer}}
					{{checkbox "source1-timer-target" "Display timer end time, if available, instead of the time remaining" .EngineOptions.Source1.TimerTarget}}
					{{counter "source1-counter" "Timer number to use (0-9)." .EngineOptions.Source1.Counter}}
					{{checkbox "source1-tod" "Enable time of day input on this source" .EngineOptions.Source1.Tod}}

					<label for="source1-timezone">
						<span>Timezone for the time of day input</span>x
						{{$selected := .EngineOptions.Source1.TimeZone}}
						<select id="source1-timezone" name="source1-timezone" >
							{{ range $tz := .Timezones }}
								<option {{if eq $selected $tz}} selected {{end}}>{{$tz}}</option>
							{{ end }}
						</select>
					</label>

					{{checkbox "source1-hidden" "Initially hide this source. Can be toggled by OSC on runtime." .EngineOptions.Source1.Hidden}}
					{{color "source1-overtime-color" "Background color for overtime countdowns." .EngineOptions.Source1.OvertimeColor}}
				</fieldset>
				<fieldset>
					<legend>Source 2</legend>

					<label for="source2-text">
						<span>Text label for time source</span>
						<input type="text" id="source2-text" name="source2-text" value="{{.EngineOptions.Source2.Text}}" />
					</label>

					{{checkbox "source2-ltc" "Enable LTC input on this source" .EngineOptions.Source2.LTC}}
					{{checkbox "source2-timer" "Enable input from the associated timer" .EngineOptions.Source2.Timer}}
					{{checkbox "source2-timer-target" "Display timer end time, if available, instead of the time remaining" .EngineOptions.Source2.TimerTarget}}
					{{counter "source2-counter" "Timer number to use (0-9)." .EngineOptions.Source2.Counter}}
					{{checkbox "source2-tod" "Enable time of day input on this source" .EngineOptions.Source2.Tod}}

					<label for="source2-timezone">
						<span>Timezone for the time of day input</span>
						{{$selected = .EngineOptions.Source2.TimeZone}}
						<select id="source2-timezone" name="source2-timezone" >
							{{ range $tz := .Timezones }}
								<option {{if eq $selected $tz}} selected {{end}}>{{$tz}}</option>
							{{ end }}
						</select>
					</label>

					{{checkbox "source2-hidden" "Initially hide this source. Can be toggled by OSC on runtime." .EngineOptions.Source2.Hidden}}
					{{color "source2-overtime-color" "Background color for overtime countdowns." .EngineOptions.Source2.OvertimeColor}}
				</fieldset>
				<fieldset>
					<legend>Source 3</legend>

					<label for="source3-text">
						<span>Text label for time source</span>
						<input type="text" id="source3-text" name="source3-text" value="{{.EngineOptions.Source3.Text}}" />
					</label>

					{{checkbox "source3-ltc" "Enable LTC input on this source" .EngineOptions.Source3.LTC}}
					{{checkbox "source3-timer" "Enable input from the associated timer" .EngineOptions.Source3.Timer}}
					{{checkbox "source3-timer-target" "Display timer end time, if available, instead of the time remaining" .EngineOptions.Source3.TimerTarget}}
					{{counter "source3-counter" "Timer number to use (0-9)." .EngineOptions.Source3.Counter}}
					{{checkbox "source3-tod" "Enable time of day input on this source" .EngineOptions.Source3.Tod}}

					<label for="source3-timezone">
						<span>Timezone for the time of day input</span>
						{{$selected = .EngineOptions.Source3.TimeZone}}
						<select id="source3-timezone" name="source3-timezone" >
							{{ range $tz := .Timezones }}
								<option {{if eq $selected $tz}} selected {{end}}>{{$tz}}</option>
							{{ end }}
						</select>
					</label>

					{{checkbox "source3-hidden" "Initially hide this source. Can be toggled by OSC on runtime." .EngineOptions.Source3.Hidden}}
					{{color "source3-overtime-color" "Background color for overtime countdowns." .EngineOptions.Source3.OvertimeColor}}
				</fieldset>
				<fieldset>
					<legend>Source 4</legend>

					<label for="source4-text">
						<span>Text label for time source</span>
						<input type="text" id="source4-text" name="source4-text" value="{{.EngineOptions.Source4.Text}}" />
					</label>

					{{checkbox "source4-ltc" "Enable LTC input on this source" .EngineOptions.Source4.LTC}}
					{{checkbox "source4-timer" "Enable input from the associated timer" .EngineOptions.Source4.Timer}}
					{{checkbox "source4-timer-target" "Display timer end time, if available, instead of the time remaining" .EngineOptions.Source4.TimerTarget}}
					{{counter "source4-counter" "Timer number to use (0-9)." .EngineOptions.Source4.Counter}}
					{{checkbox "source4-tod" "Enable time of day input on this source" .EngineOptions.Source4.Tod}}

					<label for="source4-timezone">
						<span>Time zone for the time of day input</span>
						{{$selected = .EngineOptions.Source4.TimeZone}}
						<select id="source4-timezone" name="source4-timezone" >
							{{ range $tz := .Timezones }}
								<option {{if eq $selected $tz}} selected {{end}}>{{$tz}}</option>
							{{ end }}
						</select>
					</label>

					{{checkbox "source4-hidden" "Initially hide this source. Can be toggled by OSC on runtime." .EngineOptions.Source4.Hidden}}
					{{color "source4-overtime-color" "Background color for overtime countdowns." .EngineOptions.Source4.OvertimeColor}}
				</fieldset>
			</fieldset>

			<fieldset>
				<legend>Overtime behaviour</legend>

				<label for="overtime-count-mode">
					<span>Countdown readout for overtime timers</span>
					<select name="overtime-count-mode" id="overtime-count-mode">
						<option value="zero" {{if eq .EngineOptions.OvertimeCountMode "zero"}} selected {{end}}>Show 00:00:00</option>
						<option value="blank" {{if eq .EngineOptions.OvertimeCountMode "blank"}} selected {{end}}>Blank display</option>
						<option value="continue" {{if eq .EngineOptions.OvertimeCountMode "continue"}} selected {{end}}>Continue counting up</option>
					</select>
				</label>

				<label for="overtime-visibility">
					<span>Extra visibility for overtime timers</span>
					<select name="overtime-visibility" id="overtime-visibility">
						<option value="blink" {{if eq .EngineOptions.OvertimeVisibility "blink"}} selected {{end}}>Blink readout</option>
						<option value="none" {{if eq .EngineOptions.OvertimeVisibility "none"}} selected {{end}}>No extra visibility</option>
						<option value="background" {{if eq .EngineOptions.OvertimeVisibility "background"}} selected {{end}}>Change background color</option>
						<option value="both" {{if eq .EngineOptions.OvertimeVisibility "both"}} selected {{end}}>Change background + blink</option>
					</select>
				</label>
			</fieldset>
			<fieldset>
				<legend>Timer signal colors</legend>

				{{checkbox "auto-signals" "Automatically set signal color per timer state." .EngineOptions.AutoSignals}}
				{{checkbox "signal-start" "In automation mode, set a color on timer start." .EngineOptions.SignalStart}}
				{{color "signal-color-start" "Timer start signal color." .EngineOptions.SignalColorStart}}
				{{number "signal-threshold-warning" "Time threshold for warning color, in seconds. Set to 0 to disable." .EngineOptions.SignalThresholdWarning}}
				{{color "signal-color-warning" "Color for warning signals." .EngineOptions.SignalColorWarning}}
				{{number "signal-threshold-end" "Time threshold for end color, in seconds." .EngineOptions.SignalThresholdEnd}}
				{{color "signal-color-end" "End signal color." .EngineOptions.SignalColorEnd}}

				<label for="signal-hw-type">
						<span>Signal hardware type</span>
						<select name="signal-hw-type" id="signal-hw-type">
						<option value="unicorn-hd" {{if eq .SignalType "unicorn-hd"}} selected {{end}}>Pimoroni Unicorn HD or Ubercorn</option>
						<option value="none" {{if eq .SignalType "none"}} selected {{end}}>None</option>
					</select><br />
				</label>

				{{number "signal-hw-group" "Hardware signal group." .EngineOptions.SignalHardware}}
				{{byte "signal-hw-brightness" "ardware signal master brightness, 0 = off, 255 = maximum brightness." .SignalBrightness}}
				{{checkbox "signal-hw-follow" "Hardware signal follows source 1 color." .SignalFollow}}
				{{checkbox "SignalToBG" "Use hardware signal color as clock background." .SignalToBG}}
			</fieldset>


			<fieldset>
				<legend>Mitti and Millumin</legend>

				{{counter "mitti" "Timer number for OSC feedback from Mitti" .EngineOptions.Mitti}}
				{{counter "millumin" "Timer number for OSC feedback from Millumin" .EngineOptions.Millumin}}
				{{text "millumin-ignore" "Regexp for ignoring media layers from the Millumin OSC feedback" .EngineOptions.Ignore}}
			</fieldset>

			<fieldset>
				<legend>Analog Way Picturall</legend>

				<p>Picturall integration connects to a single picturall in local network. Tested against
				picturall pro mk2 with software version 3.4.1.</p>
				<p>By default the media information from the highest playing layer is displayed and
				looping and streaming media is ignored.</p>
				<p>A known caveat is that the media default playmode changes are updated every 10 seconds so loop
				detection may be erronous just after changes to the settings.</p>

				{{checkbox "picturall-enabled" "Enable Picturalll integration." .EngineOptions.PicturallEnabled}}
				{{text "picturall-address" "Address to connect to. Leave blank for autodiscovery. Autodiscovery will use the first picturall that responds." .EngineOptions.PicturallAddress}}
				{{number "picturall-port" "Port to connect to." .EngineOptions.PicturallPort}}
				{{counter "picturall-timer" "Timer number for feedback from Picturall" .EngineOptions.PicturallTimer}}
				{{checkbox "picturall-loops" "Show looping media." .EngineOptions.PicturallLoops}}
				{{checkbox "picturall-streams" "Show streaming media." .EngineOptions.PicturallStreams}}
				{{checkbox "picturall-media-name" "Show media name as tally message on the clock." .EngineOptions.PicturallMediaName}}
				{{number "picturall-timeout" "Message timeout for clearing the clock display, in milliseconds." .EngineOptions.PicturallTimeout}}
				{{text "picturall-ignore-layers" "List of layer numbers to ignore, comma separated." .EngineOptions.PicturallIgnoreLayers}}
				{{color "picturall-media-color" "Media name text color." .EngineOptions.PicturallMediaColor}}
				{{color "picturall-media-bg" "Media name background color." .EngineOptions.PicturallMediaBG}}
			</fieldset>

			<fieldset>
				<legend>vMix integration</legend>
				<p>By default the clock displays the data about the first video found in list PGM - overlay 8... overlay 1.<p>

				{{checkbox "vmix-enabled" "Enable vMix integration." .EngineOptions.VmixEnabled}}
				{{text "vmix-address" "Address to connect to." .EngineOptions.VmixAddress}}
				{{number "vmix-port" "Port to connect to" .EngineOptions.VmixPort}}
				{{counter "vmix-timer" "Timer number for feedback from vMix." .EngineOptions.VmixTimer}}
				{{checkbox "vmix-loops" "Show looping media" .EngineOptions.VmixLoops}}
				{{checkbox "vmix-pgm-only" "Show only PGM media (no overlays)." .EngineOptions.VmixPGMOnly}}
				{{checkbox "vmix-show-pvm" "Show media in preview." .EngineOptions.VmixPVM}}
				{{text "vmix-ignore-overlays" "List of overlay numbers to ignore, comma separated." .EngineOptions.VmixIgnoreOverlays}}
				{{checkbox "vmix-media-name" "Show media name as tally message on the clock." .EngineOptions.VmixMediaName}}
				{{color "vmix-media-color" "Media name text color." .EngineOptions.VmixMediaColor}}
				{{color "vmix-media-bg" "Media name background color" .EngineOptions.VmixMediaBG}}
				{{number "vmix-interval" "Polling interval, in milliseconds. Adjust if the clock causes too high load on the vMix machine or if you need better accuracy." .EngineOptions.VmixInterval}}
				{{number "vmix-timeout" "Message timeout for clearing the clock display, in milliseconds." .EngineOptions.VmixTimeout}}
			</fieldset>

			<fieldset>
				<legend>DSAN Limitimer</legend>
				<p>See <a href="https://gitlab.com/clock-8001/clock-8001/-/blob/master/limitimer.md">https://gitlab.com/Depili/clock-8001/-/blob/master/limitimer.md</a> for documentation.</p>
				<label for="limitimer-mode">
						<span>Limitimer mode</span>
						<select name="limitimer-mode" id="limitimer-mode">
							<option value="off" {{if eq .EngineOptions.LimitimerMode "off"}} selected {{end}}>Off</option>
							<option value="send" {{if eq .EngineOptions.LimitimerMode "send"}} selected {{end}}>Send timers</option>
							<option value="receive" {{if eq .EngineOptions.LimitimerMode "receive"}} selected {{end}}>Receive timers</option>
						</select>
				</label>

				{{text "limitimer-serial" "Serial device for limitimer communication." .EngineOptions.LimitimerSerial}}

				<p>RS-485 reception and OSC broadcast controls for individual limitimer source programs:</p>
				<table>
					<tr>
						<th>Timer</th>
						<th>Receive</th>
						<th>Broadcast</th>
					</tr>
					<tr>
						<td>Program 1</td>
						<td><input type="checkbox" id="limitimer-receive-timer1" name="limitimer-receive-timer1" {{if .EngineOptions.LimitimerReceive1}} checked {{end}}/></td>
						<td><input type="checkbox" id="limitimer-broadcast-timer1" name="limitimer-broadcast-timer1" {{if .EngineOptions.LimitimerBroadcast1}} checked {{end}}/></td>
					</tr>
					<tr>
						<td>Program 2</td>
						<td><input type="checkbox" id="limitimer-receive-timer2" name="limitimer-receive-timer2" {{if .EngineOptions.LimitimerReceive2}} checked {{end}}/></td>
						<td><input type="checkbox" id="limitimer-broadcast-timer2" name="limitimer-broadcast-timer2" {{if .EngineOptions.LimitimerBroadcast2}} checked {{end}}/></td>
					</tr>
					<tr>
						<td>Program 3</td>
						<td><input type="checkbox" id="limitimer-receive-timer3" name="limitimer-receive-timer3" {{if .EngineOptions.LimitimerReceive3}} checked {{end}}/></td>
						<td><input type="checkbox" id="limitimer-broadcast-timer3" name="limitimer-broadcast-timer3" {{if .EngineOptions.LimitimerBroadcast3}} checked {{end}}/></td>
					</tr>
					<tr>
						<td>Session / Program 4</td>
						<td><input type="checkbox" id="limitimer-receive-timer4" name="limitimer-receive-timer4" {{if .EngineOptions.LimitimerReceive4}} checked {{end}}/></td>
						<td><input type="checkbox" id="limitimer-broadcast-timer4" name="limitimer-broadcast-timer4" {{if .EngineOptions.LimitimerBroadcast4}} checked {{end}}/></td>
					</tr>
					<tr>
						<td>Selected program</td>
						<td><input type="checkbox" id="limitimer-receive-timer5" name="limitimer-receive-timer5" {{if .EngineOptions.LimitimerReceive5}} checked {{end}}/></td>
						<td><input type="checkbox" id="limitimer-broadcast-timer5" name="limitimer-broadcast-timer5" {{if .EngineOptions.LimitimerBroadcast5}} checked {{end}}/></td>
					</tr>
				</table>

			</fieldset>

			<fieldset>
				<legend>InterSpace Industries Countdown2 UDP</legend>
				<p>StageTimer2 and Irisdown also support sending data with this protocol.</p>
				<label for="udp-time">
						<span>Countdown2 UDP mode</span>
						<select name="udp-time" id="udp-time">
							<option value="off" {{if eq .EngineOptions.UDPTime "off"}} selected {{end}}>Off</option>
							<option value="send" {{if eq .EngineOptions.UDPTime "send"}} selected {{end}}>Send timers</option>
							<option value="receive" {{if eq .EngineOptions.UDPTime "receive"}} selected {{end}}>Receive timers</option>
						</select>
				</label>

				{{counter "udp-timer-1" "Timer number for StageTimer2 UDP timer 1 from port 36700" .EngineOptions.UDPTimer1}}
				{{counter "udp-timer-2" "Timer number for StageTimer2 UDP timer 2 from port 36701" .EngineOptions.UDPTimer2}}
			</fieldset>
			<fieldset>
				<legend>Colors</legend>

				<fieldset>
					<legend>Round clocks</legend>

					{{color "TextColor" "Color for text." .TextColor}}
					{{color "SecColor" "Color for the second ring circles." .SecondColor}}
					{{color "StaticColor" "Color for 12 static \"hour\" markers." .StaticColor}}
					{{color "CountdownColor" "Color for secondary countdown display." .CountdownColor}}
				</fieldset>
				<fieldset>
					<legend>Text clock</legend>

					{{color "Row1Color" "Color for timer row 1." .Row1Color}}
					{{uint8 "row1-alpha" "Alpha for timer row 1." .Row1Alpha}}

					{{color "Row2Color" "Color for timer row 2." .Row2Color}}
					{{uint8 "row2-alpha" "Alpha for timer row 2." .Row2Alpha}}

					{{color "Row3Color" "Color for timer row 3." .Row3Color}}
					{{uint8 "row3-alpha" "Alpha for timer row 3." .Row3Alpha}}

					{{color "LabelColor" "Color for timer titles." .LabelColor}}
					{{uint8 "label-alpha" "Alpha for timer titles." .LabelAlpha}}

					{{ checkbox "DrawBoxes" "Draw background boxes for labels and timers." .DrawBoxes}}

					{{color "LabelBG" "Background color for time titles." .LabelBG}}
					{{uint8 "label-bg-alpha" "Alpha for timer title backgrounds." .LabelBGAlpha}}

					{{color "TimerBG" "Background color for timers." .TimerBG}}
					{{uint8 "timer-bg-alpha" "Alpha for timer backgrounds." .TimerBGAlpha}}

					{{number "NumberFontSize" "Size used to render number tect, higher results in smoother letters, but going too high will crash on the rpi." .NumberFontSize}}
				</fieldset>
			</fieldset>
			<fieldset>
				<legend>OSC</legend>

				{{checkbox "DisableOSC" "Disable remote OSC commands." .EngineOptions.DisableOSC}}
				{{checkbox "DisableFeedback" "Disable sendinf of OSC state feedback." .EngineOptions.DisableFeedback}}
				{{text "ListenAddr" "Address and port to listen for osc commands. 0.0.0.0 defaults to all network interfaces." .EngineOptions.ListenAddr}}
				{{text "Connect" "Address and port to send OSC feedback to. 255.255.255.255 broadcasts to all network interfaces" .EngineOptions.Connect}}
			</fieldset>
			<fieldset>
				<legend>Config interface</legend>

				{{text "HTTPUser" "Username for web configuration interface." .HTTPUser}}
				{{text "HTTPPassword" "Password for web configuration interface." .HTTPPassword}}
				{{checkbox "DisableHTTP" "Disable this web configuration interface. Undoing this needs editing of the config.ini file." .DisableHTTP}}
				{{text "HTTPPort" "Port to listen for the web configuration. Needs to be in format of \":1234\"" .HTTPPort}}
			</fieldset>
			<fieldset>
				<legend>LTC</legend>

				{{checkbox "DisableLTC" "Disable LTC display and reception." .EngineOptions.DisableLTC}}
				{{checkbox "LTCSeconds" "Controls what is displayed on the clock ring in LTC mode, unchecked = frames, checked = seconds" .EngineOptions.LTCSeconds}}
				{{checkbox "LTCFollow" "Continue on internal clock if LTC signal is lost. If unset display will blank when signal is gone." .EngineOptions.LTCFollow}}
			</fieldset>
			<fieldset>
				<legend>GPIO Pulser, raspberry pi only.</legend>

				<p>Provides pulses on GPIO pins every second, minute and hour, and
				alternating polarity on another pin. Suitable for acting as central clock.<\p>

				<p>Pins are given as raspberry pi GPIO numbers, which are not the same as pin numbers
				on the gpio connector.</p>

				{{checkbox "gpio-enabled" "Enable GPIO pulses based on time-of-day." .GpioEnabled}}

				{{text "gpio-seconds-a-pin" "Seconds alternating pin." .SecA}}
				{{text "gpio-seconds-pulse-pin" "Seconds pulse pin." .SecPulse}}
				{{text "gpio-seconds-trigger" "Seconds manual advance trigger pin." .SecTrigger}}

				{{text "gpio-minutes-a-pin" "Minutes alternating pin." .MinA}}
				{{text "gpio-minutes-pulse-pin" "Minutes pulse pin." .MinPulse}}
				{{text "gpio-minutes-trigger" "Minutes manual advance trigger pin." .MinTrigger}}

				{{text "gpio-hours-a-pin" "Hours alternating pin." .HourA}}
				{{text "gpio-hours-pulse-pin" "Hours pulse pin." .HourPulse}}
				{{text "gpio-hours-trigger" "Hours manual advance trigger pin." .HourTrigger}}

				{{number "gpio-pulse-duration" "Pulse duration, in milliseoncds." .PulseDuration}}

				{{checkbox "gpio-invert-polarity" "Invert the pulse pin polarity." .InvertPolarity}}
			</fieldset>


			{{if .Raspberry}}
				<fieldset>
					<legend>Raspberry pi configuration</legend>

					<label for="configtxt">
						<span>Raspberry pi /boot/config.txt. Changing this will reboot the raspberry pi</span>
						<textarea id="configtxt" name="configtxt" rows="20" cols="50">{{.ConfigTxt}}</textarea>
					</label>
				</fieldset>
			{{end}}

			<input type="submit" value="Save config and restart clock" />
		</form>
	</div>


	<style type="text/css">
		h1 {
			color: F072A9;
			font-weight: bold;
			text-shadow: 1px 1px 1px #fff;
		}
		.errors {
			border-radius: 10px;
			-webkit-border-radius: 10px;
			-moz-border-radius: 10px;
			margin: 0px 0px 10px 0px;
			border: 1px solid red;
			padding: 20px;
			background: #FFF4F4;
			box-shadow: inset 0px 0px 15px #FFE5E5;
			-moz-box-shadow: inset 0px 0px 15px #FFE5E5;
			-webkit-box-shadow: inset 0px 0px 15px #FFE5E5;
			max-width: 760px;

		}
		.config-form{
			max-width: 800px;
			font-family: "Lucida Sans Unicode", "Lucida Grande", sans-serif;
		}
		p{
			color: #F072A9;
			font-weight: bold;
			font-size: 13px;
			text-shadow: 1px 1px 1px #fff;
		}
		.errors p{
			color: red;
			font-weight: bold;
			font-size: 13px;
			text-shadow: 1px 1px 1px #fff;
		}
		.config-form li{
			color: #F072A9;
			font-weight: bold;
			font-size: 13px;
			text-shadow: 1px 1px 1px #fff;
		}
		.errors li{
			color: red;
			font-weight: bold;
			font-size: 13px;
			text-shadow: 1px 1px 1px #fff;
		}
		.config-form label{
			display:block;
			margin-bottom: 10px;
			overflow: auto;
		}
		.config-form label > span{
			float: left;
			width: 300px;
			color: #F072A9;
			font-weight: bold;
			font-size: 13px;
			text-shadow: 1px 1px 1px #fff;
		}
		.config-form fieldset{
			border-radius: 10px;
			-webkit-border-radius: 10px;
			-moz-border-radius: 10px;
			margin: 0px 0px 10px 0px;
			border: 1px solid #FFD2D2;
			padding: 20px;
			background: #FFF4F4;
			box-shadow: inset 0px 0px 15px #FFE5E5;
			-moz-box-shadow: inset 0px 0px 15px #FFE5E5;
			-webkit-box-shadow: inset 0px 0px 15px #FFE5E5;
		}
		.config-form fieldset legend{
			color: #FFA0C9;
			border-top: 1px solid #FFD2D2;
			border-left: 1px solid #FFD2D2;
			border-right: 1px solid #FFD2D2;
			border-radius: 5px 5px 0px 0px;
			-webkit-border-radius: 5px 5px 0px 0px;
			-moz-border-radius: 5px 5px 0px 0px;
			background: #FFF4F4;
			padding: 0px 8px 3px 8px;
			box-shadow: -0px -1px 2px #F1F1F1;
			-moz-box-shadow:-0px -1px 2px #F1F1F1;
			-webkit-box-shadow:-0px -1px 2px #F1F1F1;
			font-weight: normal;
			font-size: 12px;
		}
		.config-form textarea{
			width:250px;
			height:100px;
		}
		.config-form input,
		.config-form select,
		.config-form textarea{
			border-radius: 3px;
			-webkit-border-radius: 3px;
			-moz-border-radius: 3px;
			border: 1px solid #FFC2DC;
			outline: none;
			color: #F072A9;
			padding: 5px 8px 5px 8px;
			box-shadow: inset 1px 1px 4px #FFD5E7;
			-moz-box-shadow: inset 1px 1px 4px #FFD5E7;
			-webkit-box-shadow: inset 1px 1px 4px #FFD5E7;
			background: #FFEFF6;
			width:50%;
		}
		.config-form  input[type=checkbox]{
			width:20px;
		}
		.config-form  input[type=submit],
		.config-form  input[type=button]{
			background: #EB3B88;
			border: 1px solid #C94A81;
			padding: 5px 15px 5px 15px;
			color: #FFCBE2;
			box-shadow: inset -1px -1px 3px #FF62A7;
			-moz-box-shadow: inset -1px -1px 3px #FF62A7;
			-webkit-box-shadow: inset -1px -1px 3px #FF62A7;
			border-radius: 3px;
			border-radius: 3px;
			-webkit-border-radius: 3px;
			-moz-border-radius: 3px;
			font-weight: bold;
			max-width: 800px;
			width: 100%;
		}
		.config-form table th{
			text-align: right;
			color: #F072A9;
			font-weight: bold;
			font-size: 13px;
			text-shadow: 1px 1px 1px #fff;
			padding-right: 2em;
		}
		.config-form table td{
			color: #F072A9;
			font-weight: normal;
			font-size: 13px;
			text-shadow: 1px 1px 1px #fff;
		}
		.required{
			color:red;
			font-weight:normal;
		}
	</style>
</body>
</html>
`
