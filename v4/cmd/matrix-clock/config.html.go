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

					<label for="Format12h">
						<span>Use 12 hour format for time-of-day display</span>
						<input type="checkbox" id="Format12h" name="Format12h" {{if .EngineOptions.Format12h}} checked {{end}}/>
					</label>

					<label for="Debug">
						<span>Output verbose debug information. This will impact performance</span>
						<input type="checkbox" id="Debug" name="Debug" {{if .Debug}} checked {{end}}/>
					</label>

					<label for="Font">
						<span>Font filename for round clocks</span>
						<input type="text" id="Font" name="Font" value="{{.Font}}" />
					</label>

					<label for="Flash">
						<span>Flashing interval in milliseconds for ellapsed countdowns</span>
						<input type="number" min="0" id="Flash" name="Flash" value="{{.EngineOptions.Flash}}" />
					</label>

					<label for="Timeout">
						<span>Timeout for clearing OSC text display messages, milliseconds</span>
						<input type="number" min="0" id="Timeout" name="Timeout" value="{{.EngineOptions.Timeout}}" />
					</label>

					<label for="ShowInfo">
						<span>Time to show clock information on startup, seconds</span>
						<input type="number" min="0" id="ShowInfo" name="ShowInfo" value="{{.EngineOptions.ShowInfo}}" />
					</label>

					<label for="TextColor">
						<span>Text color</span>
						<input type="color" id="TextColor" name="TextColor" value="{{.TextColor}}" />
					</label>

					<label for="CountdownColor">
						<span>Aux countdown color</span>
						<input type="color" id="CountdownColor" name="CountdownColor" value="{{.CountdownColor}}" />
					</label>

					<label for="BackgroundColor">
						<span>Background color</span>
						<input type="color" id="BackgroundColor" name="BackgroundColor" value="{{.BackgroundColor}}" />
					</label>

					<label for="Matrix">
						<span>Led matrix UPD port</span>
						<input type="text" id="Matrix" name="Matrix" value="{{.Matrix}}" />
					</label>

					<label for="SerialName">
						<span>Led ring serial port</span>
						<input type="text" id="SerialName" name="SerialName" value="{{.SerialName}}" />
					</label>
					<label for="SerialBaud">
						<span>Led ring serial port baud rate</span>
						<input type="text" id="SerialBaud" name="SerialBaud" value="{{.SerialBaud}}" />
					</label>

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

					<label for="source1-ltc">
						<span>Enable LTC input on this source</span>
						<input type="checkbox" id="source1-ltc" name="source1-ltc" {{if .EngineOptions.Source1.LTC}} checked {{end}} />
					</label>

					<label for="source1-timer">
						<span>Enable input from the associated timer</span>
						<input type="checkbox" id="source1-timer" name="source1-timer" {{if .EngineOptions.Source1.Timer}} checked {{end}} />
					</label>

					<label for="source1-timer-target">
						<span>Display timer end time, if available, instead of the time remaining</span>
						<input type="checkbox" id="source1-timer-target" name="source1-timer-target" {{if .EngineOptions.Source1.TimerTarget}} checked {{end}} />
					</label>

					<label for="source1-counter">
						<span>Timer number to use (0-9)</span>
						<input type="number" min="0" max="9" id="source1-counter" name="source1-counter" value="{{.EngineOptions.Source1.Counter}}" />
					</label>

					<label for="source1-tod">
						<span>Enable time of day input on this source</span>
						<input type="checkbox" id="source1-tod" name="source1-tod" {{if .EngineOptions.Source1.Tod}} checked {{end}} />
					</label>

					<label for="source1-timezone">
						<span>Timezone for the time of day input</span>
						{{$selected := .EngineOptions.Source1.TimeZone}}
						<select id="source1-timezone" name="source1-timezone" >
							{{ range $tz := .Timezones }}
								<option {{if eq $selected $tz}} selected {{end}}>{{$tz}}</option>
							{{ end }}
						</select>
					</label>

					<label for="source1-hidden">
						<span>Initially hide this source. Can be toggled by OSC on runtime.</span>
						<input type="checkbox" id="source1-hidden" name="source1-hidden" {{if .EngineOptions.Source1.Hidden}} checked {{end}} />
					</label>

					<label for="source1-overtime-color">
						<span>Background color for overtime countdowns</span>
						<input type="color" id="source1-overtime-color" name="source1-overtime-color" value="{{.EngineOptions.Source1.OvertimeColor}}" />
					</label>

				</fieldset>
				<fieldset>
					<legend>Source 2</legend>

					<label for="source2-text">
						<span>Text label for time source</span>
						<input type="text" id="source2-text" name="source2-text" value="{{.EngineOptions.Source2.Text}}" />
					</label>

					<label for="source2-ltc">
						<span>Enable LTC input on this source</span>
						<input type="checkbox" id="source2-ltc" name="source2-ltc" {{if .EngineOptions.Source2.LTC}} checked {{end}} />
					</label>

					<label for="source2-timer">
						<span>Enable input from the associated timer</span>
						<input type="checkbox" id="source2-timer" name="source2-timer" {{if .EngineOptions.Source2.Timer}} checked {{end}} />
					</label>

					<label for="source2-timer-target">
						<span>Display timer end time, if available, instead of the time remaining</span>
						<input type="checkbox" id="source2-timer-target" name="source2-timer-target" {{if .EngineOptions.Source2.TimerTarget}} checked {{end}} />
					</label>

					<label for="source2-counter">
						<span>Timer number to use (0-9)</span>
						<input type="number" min="0" max="9" id="source2-counter" name="source2-counter" value="{{.EngineOptions.Source2.Counter}}" />
					</label>

					<label for="source2-tod">
						<span>Enable time of day input on this source</span>
						<input type="checkbox" id="source2-tod" name="source2-tod" {{if .EngineOptions.Source2.Tod}} checked {{end}} />
					</label>

					<label for="source2-timezone">
						<span>Timezone for the time of day input</span>
						{{$selected = .EngineOptions.Source2.TimeZone}}
						<select id="source2-timezone" name="source2-timezone" >
							{{ range $tz := .Timezones }}
								<option {{if eq $selected $tz}} selected {{end}}>{{$tz}}</option>
							{{ end }}
						</select>
					</label>

					<label for="source2-hidden">
						<span>Initially hide this source. Can be toggled by OSC on runtime.</span>
						<input type="checkbox" id="source2-hidden" name="source2-hidden" {{if .EngineOptions.Source2.Hidden}} checked {{end}} />
					</label>

					<label for="source2-overtime-color">
						<span>Background color for overtime countdowns</span>
						<input type="color" id="source2-overtime-color" name="source2-overtime-color" value="{{.EngineOptions.Source2.OvertimeColor}}" />
					</label>
				</fieldset>
				<fieldset>
					<legend>Source 3</legend>

					<label for="source3-text">
						<span>Text label for time source</span>
						<input type="text" id="source3-text" name="source3-text" value="{{.EngineOptions.Source3.Text}}" />
					</label>

					<label for="source3-ltc">
						<span>Enable LTC input on this source</span>
						<input type="checkbox" id="source3-ltc" name="source3-ltc" {{if .EngineOptions.Source3.LTC}} checked {{end}} />
					</label>

					<label for="source3-timer">
						<span>Enable input from the associated timer on this source</span>
						<input type="checkbox" id="source3-timer" name="source3-timer" {{if .EngineOptions.Source3.Timer}} checked {{end}} />
					</label>

					<label for="source3-timer-target">
						<span>Display timer end time, if available, instead of the time remaining</span>
						<input type="checkbox" id="source3-timer-target" name="source3-timer-target" {{if .EngineOptions.Source3.TimerTarget}} checked {{end}} />
					</label>

					<label for="source3-counter">
						<span>Timer number to use (0-9)</span>
						<input type="number" min="0" max="9" id="source3-counter" name="source3-counter" value="{{.EngineOptions.Source3.Counter}}" />
					</label>

					<label for="source3-tod">
						<span>Enable time of day input on this source</span>
						<input type="checkbox" id="source3-tod" name="source3-tod" {{if .EngineOptions.Source3.Tod}} checked {{end}} />
					</label>

					<label for="source3-timezone">
						<span>Timezone for the time of day input</span>
						{{$selected = .EngineOptions.Source3.TimeZone}}
						<select id="source3-timezone" name="source3-timezone" >
							{{ range $tz := .Timezones }}
								<option {{if eq $selected $tz}} selected {{end}}>{{$tz}}</option>
							{{ end }}
						</select>
					</label>

					<label for="source3-hidden">
						<span>Initially hide this source. Can be toggled by OSC on runtime.</span>
						<input type="checkbox" id="source3-hidden" name="source3-hidden" {{if .EngineOptions.Source3.Hidden}} checked {{end}} />
					</label>

					<label for="source3-overtime-color">
						<span>Background color for overtime countdowns</span>
						<input type="color" id="source3-overtime-color" name="source3-overtime-color" value="{{.EngineOptions.Source3.OvertimeColor}}" />
					</label>
				</fieldset>
				<fieldset>
					<legend>Source 4</legend>

					<label for="source4-text">
						<span>Text label for time source</span>
						<input type="text" id="source4-text" name="source4-text" value="{{.EngineOptions.Source4.Text}}" />
					</label>

					<label for="source4-ltc">
						<span>Enable LTC input on this source</span>
						<input type="checkbox" id="source4-ltc" name="source4-ltc" {{if .EngineOptions.Source4.LTC}} checked {{end}} />
					</label>

					<label for="source4-timer">
						<span>Enable input from the associated timer on this source</span>
						<input type="checkbox" id="source4-timer" name="source4-timer" {{if .EngineOptions.Source4.Timer}} checked {{end}} />
					</label>

					<label for="source4-timer-target">
						<span>Display timer end time, if available, instead of the time remaining</span>
						<input type="checkbox" id="source4-timer-target" name="source4-timer-target" {{if .EngineOptions.Source4.TimerTarget}} checked {{end}} />
					</label>

					<label for="source4-counter">
						<span>Timer number to use (0-9)</span>
						<input type="number" min="0" max="9" id="source4-counter" name="source4-counter" value="{{.EngineOptions.Source4.Counter}}" />
					</label>

					<label for="source4-tod">
						<span>Enable time of day input on this source</span>
						<input type="checkbox" id="source4-tod" name="source4-tod" {{if .EngineOptions.Source4.Tod}} checked {{end}} />
					</label>

					<label for="source4-timezone">
						<span>Time zone for the time of day input</span>
						{{$selected = .EngineOptions.Source4.TimeZone}}
						<select id="source4-timezone" name="source4-timezone" >
							{{ range $tz := .Timezones }}
								<option {{if eq $selected $tz}} selected {{end}}>{{$tz}}</option>
							{{ end }}
						</select>
					</label>

					<label for="source4-hidden">
						<span>Initially hide this source. Can be toggled by OSC on runtime.</span>
						<input type="checkbox" id="source4-hidden" name="source4-hidden" {{if .EngineOptions.Source4.Hidden}} checked {{end}} />
					</label>

					<label for="source4-overtime-color">
						<span>Background color for overtime countdowns</span>
						<input type="color" id="source4-overtime-color" name="source4-overtime-color" value="{{.EngineOptions.Source4.OvertimeColor}}" />
					</label>
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
				<label for="auto-signals">
					<span>Automatically set signal color per timer state</span>
					<input type="checkbox" id="auto-signals" name="auto-signals" {{if .EngineOptions.AutoSignals}} checked {{end}} />
				</label>

				<label for="signal-start">
					<span>In automation mode, set a color on timer start</span>
					<input type="checkbox" id="signal-start" name="signal-start" {{if .EngineOptions.SignalStart}} checked {{end}} />
				</label>

				<label for="signal-color-start">
					<span>Start signal color</span>
					<input type="color" id="signal-color-start" name="signal-color-start" value="{{.EngineOptions.SignalColorStart}}" />
				</label>

				<label for="signal-threshold-warning">
					<span>Time threshold for warning color, in seconds. Set to 0 to disable.</span>
					<input type="number" min="0" id="signal-threshold-warning" name="signal-threshold-warning" value="{{.EngineOptions.SignalThresholdWarning}}" />
				</label>


				<label for="signal-color-warning">
					<span>Warning signal color</span>
					<input type="color" id="signal-color-warning" name="signal-color-warning" value="{{.EngineOptions.SignalColorWarning}}" />
				</label>

				<label for="signal-threshold-end">
					<span>Time threshold for end color, in seconds.</span>
					<input type="number" min="0" id="signal-threshold-end" name="signal-threshold-end" value="{{.EngineOptions.SignalThresholdEnd}}" />
				</label>

				<label for="signal-color-end">
					<span>End signal color</span>
					<input type="color" id="signal-color-end" name="signal-color-end" value="{{.EngineOptions.SignalColorEnd}}" />
				</label>

				<label for="signal-hw-group">
					<span>Hardware signal group</span>
					<input type="number" min="0" id="signal-hw-group" name="signal-hw-group" value="{{.EngineOptions.SignalHardware}}" />
				</label>

				<label for="signal-hw-follow">
					<span>Hardware signal follows source 1 color</span>
					<input type="checkbox" id="signal-hw-follow" name="signal-hw-follow" {{if .SignalFollow}} checked {{end}} />
				</label>

				<label for="SignalToBG">
					<span>Use hardware signal color as clock background</span>
					<input type="checkbox" id="SignalToBG" name="SignalToBG" {{if .SignalToBG}} checked {{end}} />
				</label>


			</fieldset>


			<fieldset>
				<legend>Mitti and Millumin</legend>

				<label for="mitti">
					<span>Timer number for OSC feedback from Mitti</span>
					<input type="number" min="0" max="9" id="mitti" name="mitti" value="{{.EngineOptions.Mitti}}" />
				</label>

				<label for="millumin">
					<span>Timer number for OSC feedback from Millumin</span>
					<input type="number" min="0" max="9" id="millumin" name="millumin" value="{{.EngineOptions.Millumin}}" />
				</label>


				<label for="millumin-ignore">
					<span>Regexp for ignoring media layers from the Millumin OSC feedback</span>
					<input type="text" id="millumin-ignore" name="millumin-ignore" value="{{.EngineOptions.Ignore}}" />
				</label>
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

				<label for="limitimer-serial">
					<span>Serial device for limitimer communication</span>
					<input type="text" id="limitimer-serial" name="limitimer-serial" value="{{.EngineOptions.LimitimerSerial}}" />
				</label>

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

				<label for="upd-timer-1">
					<span>Timer number for StageTimer2 UDP timer 1 from port 36700</span>
					<input type="number" min="0" max="9" id="udp-timer-1" name="udp-timer-1" value="{{.EngineOptions.UDPTimer1}}" />
				</label>

				<label for="upd-timer-2">
					<span>Timer number for StageTimer2 UDP timer 2 from port 36701</span>
					<input type="number" min="0" max="9" id="udp-timer-2" name="udp-timer-2" value="{{.EngineOptions.UDPTimer2}}" />
				</label>
			</fieldset>
			<fieldset>
			<fieldset>
				<legend>OSC</legend>

				<label for="DisableOSC">
					<span>Disable remote OSC commands</span>
					<input type="checkbox" id="DisableOSC" name="DisableOSC" {{if .EngineOptions.DisableOSC}} checked {{end}}/>
				</label>

				<label for="DisableOSC">
					<span>Disable sending of OSC state feedback</span>
					<input type="checkbox" id="DisableFeedback" name="DisableFeedback" {{if .EngineOptions.DisableFeedback}} checked {{end}}/>
				</label>


				<label for="ListenAddr">
					<span>Address and port to listen for osc commands. 0.0.0.0 defaults to all network interfaces</span>
					<input type="text" id="ListenAddr" name="ListenAddr" value="{{.EngineOptions.ListenAddr}}" />
				</label>

				<label for="Connect">
					<span>Address and port to send OSC feedback to. 255.255.255.255 broadcasts to all network interfaces</span>
					<input type="text" id="Connect" name="Connect" value="{{.EngineOptions.Connect}}" />
				</label>
			</fieldset>
			<fieldset>
				<legend>Config interface</legend>

				<label for="HTTPUser">
					<span>Username for the web configuration interface</span>
					<input type="text" id="HTTPUser" name="HTTPUser" value="{{.HTTPUser}}" />
				</label>

				<label for="HTTPUser">
					<span>Password for the web configuration interface</span>
					<input type="text" id="HTTPPassword" name="HTTPPassword" value="{{.HTTPPassword}}" />
				</label>

				<label for="DisableHTTP">
					<span>Disable this web configuration interface. Undoing this needs access to the SD-card</span>
					<input type="checkbox" id="DisableHTTP" name="DisableHTTP" {{if .DisableHTTP}} checked {{end}}/>
				</label>

				<label for="HTTPPort">
					<span>Port to listen for the web configuration. Needs to be in format of ":1234"</span>
					<input type="text" id="HTTPPort" name="HTTPPort" value="{{.HTTPPort}}" />
				</label>
			</fieldset>
			<fieldset>
				<legend>LTC</legend>

				<label for="DisableLTC">
					<span>Disable LTC display</span>
					<input type="checkbox" id="DisableLTC" name="DisableLTC" {{if .EngineOptions.DisableLTC}} checked {{end}}/>
				</label>

				<label for="LTCSeconds">
					<span>Controls what is displayed on the clock ring in LTC mode, unchecked = frames, checked = seconds</span>
					<input type="checkbox" id="LTCSeconds" name="LTCSeconds" {{if .EngineOptions.LTCSeconds}} checked {{end}}/>
				</label>

				<label for="LTCFollow">
					<span>Continue on internal clock if LTC signal is lost. If unset display will blank when signal is gone</span>
					<input type="checkbox" id="LTCFollow" name="LTCFollow" {{if .EngineOptions.LTCFollow}} checked {{end}}/>
				</label>

			</fieldset>

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
