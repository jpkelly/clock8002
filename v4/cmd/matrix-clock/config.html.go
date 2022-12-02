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
        <div class="tabs">
          <div class="tab">
            <input type="checkbox" id="gen_settings" class="tab_ctrl">
            <label class="tab-label" for="gen_settings">General Settings</label>
            <div class="tab-content">
              <fieldset>
                <legend>General settings</legend>
                {{checkbox "Format12h" "Use 12 hour format for time-of-day display" .EngineOptions.Format12h }}
                {{checkbox "Debug" "Output verbose debug information. This will impact performance" .Debug}}

                {{text "Font" "Font filename for round clocks." .Font}}

                <datalist id="FontList">
                  {{ range $font := .Fonts }}
                    <option>{{$font}}</option>
                  {{ end }}
                </datalist>

                {{number "Flash" "Flashing interval in milliseconds for ellapsed countdowns." .EngineOptions.Flash}}
                {{number "Timeout" "Timeout for clearing OSC text display messages, in milliseconds." .EngineOptions.Timeout}}
                {{number "ShowInfo" "Time to show clock information on startup, in seconds." .EngineOptions.ShowInfo}}

                {{color "TextColor" "Text color" .TextColor}}
                {{color "CountdownColor" "Aux countdown color" .CountdownColor}}
                {{color "BackgroundColor" "Background color, used if no background image is provided." .BackgroundColor}}

                {{text "Matrix" "Led matrix UDP port" .Matrix}}
                {{text "SerialName" "Led ring serial port" .SerialName}}
                {{number "SerialBaud" "Led ring serial port baud rate" .SerialBaud}}
              </fieldset>
            </div>
          </div>

          <div class="tab">
            <input type="checkbox" id="time_sources" class="tab_ctrl">
            <label class="tab-label" for="time_sources">Time sources</label>
            <div class="tab-content">
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

                <div class="tabs">
                  {{range $i, $source := .EngineOptions.SourceOptions }}
                    {{$n := add $i 1}}
                    <div class="tab">
                      <input type="radio" id="source{{$n}}" name="sources" class="tab_ctrl">
                      <label class="tab-label" for="source{{$n}}">Source {{$n}}</label>
                      <div class="tab-content">
                        <fieldset>
                          <legend>Source {{$n}}</legend>

                          <label for="source{{$n}}-text">
                            <span>Text label for time source</span>
                            <input type="text" id="source{{$n}}-text" name="source{{$n}}-text" value="{{$source.Text}}" />
                          </label>

                          {{checkbox (printf "source%d-ltc" $n) "Enable LTC input on this source" $source.LTC}}
                          {{checkbox (printf "source%d-timer" $n) "Enable input from the associated timer" $source.Timer}}
                          {{checkbox (printf "source%d-timer-target" $n) "Display timer end time, if available, instead of the time remaining" $source.TimerTarget}}
                          {{counter (printf "source%d-counter" $n) "Timer number to use (0-9)." $source.Counter}}
                          {{checkbox (printf "source%d-tod" $n) "Enable time of day input on this source" $source.Tod}}

                          <label for="source{{$n}}-timezone">
                            <span>Timezone for the time of day input</span>
                             <select id="source{{$n}}-timezone" name="source{{$n}}-timezone" >
                              {{ range $tz := $.Timezones }}
                                <option {{if eq $source.TimeZone $tz}} selected {{end}}>{{$tz}}</option>
                              {{ end }}
                            </select>
                          </label>

                          {{checkbox (printf "source%d-hidden" $n) "Initially hide this source. Can be toggled by OSC on runtime." $source.Hidden}}
                          {{color (printf "source%d-overtime-color" $n) "Background color for overtime countdowns." $source.OvertimeColor}}
                        </fieldset>
                      </div>
                    </div>
                  {{end}}
                </div>
              </fieldset>
            </div>
          </div>

          <div class="tab">
            <input type="checkbox" id="timers" class="tab_ctrl">
            <label class="tab-label" for="timers">Timers</label>
            <div class="tab-content">
              <fieldset>
                <legend>Timers</legend>
                <div class="tabs">

                  {{range $i, $timer := .EngineOptions.TimerOptions }}
                    <div class="tab">
                      <input type="radio" id="timer{{$i}}" name="timers" class="tab_ctrl">
                      <label class="tab-label" for="timer{{$i}}">Timer {{$i}}</label>
                      <div class="tab-content">
                        <fieldset>
                          <legend>Timer {{$i}}</legend>
                          <h3>Millumin & Mitti</h3>

                          {{number (printf "timer%d-port" $i) "Millumin and Mitti OSC port, 0 disables" $timer.ListenPort}}
                          {{checkbox (printf "timer%d-millumin-broadcast" $i) "Rebroadcast received Millumin state" $timer.MilluminBroadcast}}
                          {{checkbox (printf "timer%d-mitti-broadcast" $i) "Rebroadcast received Mitti state" $timer.MittiBroadcast}}

                          <h3>vMix<h3>

                          {{checkbox (printf "timer%d-vmix-enabled" $i) "Enable vMix integration." $timer.VmixEnabled}}
                          {{text (printf "timer%d-vmix-address" $i) "Address to connect to." $timer.VmixAddress}}
                          {{number (printf "timer%d-vmix-port" $i) "Port to connect to" $timer.VmixPort}}
                          {{checkbox (printf "timer%d-vmix-loops" $i) "Show looping media" $timer.VmixLoops}}
                          {{checkbox (printf "timer%d-vmix-pgm-only" $i) "Show only PGM media (no overlays)." $timer.VmixPGMOnly}}
                          {{checkbox (printf "timer%d-vmix-show-pvm" $i) "Show media in preview." $timer.VmixPVM}}
                          {{text (printf "timer%d-vmix-ignore-overlays" $i) "List of overlay numbers to ignore, comma separated." $timer.VmixIgnoreOverlays}}
                          {{checkbox (printf "timer%d-vmix-media-name" $i) "Show media name as tally message on the clock." $timer.VmixMediaName}}

                          <h3>Analog Way Picturall</h3>

                          <p>Picturall integration connects to a single picturall per timer in local network. Tested against
                          picturall pro mk2 with software version 3.4.1.</p>

                          <p>By default the media information from the highest playing layer is displayed and
                          looping and streaming media is ignored.</p>

                          <p>A known caveat is that the media default playmode changes are updated every 10 seconds so loop
                          detection may be erronous just after changes to the settings.</p>

                          {{checkbox (printf "timer%d-picturall-enabled" $i) "Enable Picturall integration." $timer.PicturallEnabled}}
                          {{text (printf "timer%d-picturall-address" $i) "Address to connect to. Leave blank for autodiscovery. Autodiscovery will use the first picturall that responds." $timer.PicturallAddress}}
                          {{number (printf "timer%d-picturall-port" $i) "Port to connect to." $timer.PicturallPort}}
                          {{checkbox (printf "timer%d-picturall-loops" $i) "Show looping media." $timer.PicturallLoops}}
                          {{checkbox (printf "timer%d-picturall-streams" $i) "Show streaming media." $timer.PicturallStreams}}
                          {{checkbox (printf "timer%d-picturall-media-name" $i) "Show media name as tally message on the clock." $timer.PicturallMediaName}}
                          {{text (printf "timer%d-picturall-ignore-layers" $i) "List of layer numbers to ignore, comma separated." $timer.PicturallIgnoreLayers}}

                          <h3>Common</h3>

                          {{color (printf "timer%d-media-color" $i) "Media name text color." $timer.MediaColor}}
                          {{color (printf "timer%d-media-bg" $i) "Media name background color" $timer.MediaBG}}
                        </fieldset>
                      </div>
                    </div>
                  {{end}}
                </div>
              </fieldset>
            </div>
          </div>

          <div class="tab">
            <input type="checkbox" id="overtime" class="tab_ctrl">
            <label class="tab-label" for="overtime">Overtime behaviour</label>
            <div class="tab-content">
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
            </div>
          </div>

          <div class="tab">
            <input type="checkbox" id="signals" class="tab_ctrl">
            <label class="tab-label" for="signals">Timer signals</label>
            <div class="tab-content">
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
                {{byte "signal-hw-brightness" "Hardware signal master brightness, 0 = off, 255 = maximum brightness." .SignalBrightness}}
                {{checkbox "signal-hw-follow" "Hardware signal follows source 1 color." .SignalFollow}}
                {{checkbox "SignalToBG" "Use hardware signal color as clock background." .SignalToBG}}
              </fieldset>
            </div>
          </div>

          <div class="tab">
            <input type="checkbox" id="integrations" class="tab_ctrl">
            <label class="tab-label" for="integrations">Integrations</label>
            <div class="tab-content">

              <fieldset>
                <legend>Mitti and Millumin</legend>
                <p>Legacy settings for listening for mitti and millumin on the general OSC port.
                For finer control use the timer config and per timer OSC ports.</p>

                {{counter "mitti" "Timer number for OSC feedback from Mitti" .EngineOptions.Mitti}}
                {{counter "millumin" "Timer number for OSC feedback from Millumin" .EngineOptions.Millumin}}
                {{text "millumin-ignore" "Regexp for ignoring media layers from the Millumin OSC feedback" .EngineOptions.Ignore}}
              </fieldset>

              <fieldset>
                <legend>Analog Way Picturall</legend>

                <p>General picturall settings. See also the timer settings for per
                timer options on connecting to Picturall media servers.</p>

                {{number "picturall-timeout" "Message timeout for clearing the clock display, in milliseconds." .EngineOptions.PicturallTimeout}}
              </fieldset>

              <fieldset>
                <legend>vMix integration</legend>

                <p>By default the clock displays the data about the first video found in list PGM - overlay 8... overlay 1.<p>

                {{number "vmix-interval" "Polling interval, in milliseconds. Adjust if the clock causes too high load on the vMix machine or if you need better accuracy." .EngineOptions.VmixInterval}}
                {{number "vmix-timeout" "Message timeout for clearing the clock display, in milliseconds." .EngineOptions.VmixTimeout}}
              </fieldset>

              <fieldset>
                <legend>Tricaster integration</legend>

                <p>Show data from the first live DDR with media on a tricaster system.
                By default only PGM and overlays are considered. For playlists
                the shown time left is the total playlist remaining time.</p>

                <p>Caveats: Tricaster doesn't provide play/pause or loop information in the
                API.</p>

                {{checkbox "tricaster-enabled" "Enable tricaster integration." .EngineOptions.TricasterEnabled}}
                {{text "tricaster-address" "Address to connect to" .EngineOptions.TricasterAddress}}
                {{text "tricaster-username" "Username for the tricaster system, default: admin." .EngineOptions.TricasterUser}}
                {{text "tricaster-password" "Password for the tricaster system, default: admin." .EngineOptions.TricasterPassword}}
                {{counter "tricaster-timer" "Timer number for feedback from tricaster." .EngineOptions.TricasterTimer}}
                {{checkbox "tricaster-show-pvm" "Show media playing in preview also." .EngineOptions.TricasterPVM}}
                {{checkbox "tricaster-event-enabled" "Show time until next tricaster event on a timer." .EngineOptions.TricasterEvent}}
                {{counter "tricaster-event-timer" "Timer number for the event timer." .EngineOptions.TricasterEventTimer}}
                {{checkbox "tricaster-media-name" "Show media name as tally message on the clock." .EngineOptions.TricasterMediaName}}
                {{color "tricaster-media-color" "Media name text color." .EngineOptions.TricasterMediaColor}}
                {{color "tricaster-media-bg" "Media name background color" .EngineOptions.TricasterMediaBG}}
                {{number "tricaster-interval" "Polling interval, in milliseconds. Adjust if the clock causes too high load on the tricaster or if you need better accuracy." .EngineOptions.TricasterInterval}}
                {{number "tricaster-timeout" "Message timeout for clearing the clock display, in milliseconds." .EngineOptions.TricasterTimeout}}
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
            </div>
          </div>

          <div class="tab">
            <input type="checkbox" id="misc" class="tab_ctrl">
            <label class="tab-label" for="misc">Misc settings</label>
            <div class="tab-content">
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

              {{if .Raspberry}}
                <fieldset>
                  <legend>Raspberry pi configuration</legend>

                  <label for="configtxt">
                    <span>Raspberry pi /boot/config.txt. Changing this will reboot the raspberry pi</span>
                    <textarea id="configtxt" name="configtxt" rows="20" cols="50">{{.ConfigTxt}}</textarea>
                  </label>
                </fieldset>
              {{end}}
            </div>
          </div>
        </div>
        <input type="submit" value="Save config and restart clock" />
      </form>
    </div>

    {{$color := "#8E1047"}}
    {{$oldColor := "F072A9"}}
    {{$background := "#FFF4F4"}}
    {{$tabInactive := "#EB3383"}}
    {{$tabActive := "#E3166F"}}
    {{$tabText := "white"}}

    <style type="text/css">
    h1 {
      color: {{$color}};
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
      color: {{$color}};
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
      color: {{$color}};
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
      color: {{$color}};
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
    .config-form fieldset legend {
      color: {{$color}};
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
    .config-form h3{
      color: {{$color}};
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
      color: {{$color}};
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
      background: {{$tabInactive}};
      border: 1px solid #C94A81;
      padding: 5px 15px 5px 15px;
      color: {{$tabText}};
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
      cursor: pointer;
    }
    .config-form input[type=submit]:hover,
    .config-form input[type=button]:hover{
      background: {{$tabActive}};
    }
    .config-form table th{
      text-align: right;
      color: {{$color}};
      font-weight: bold;
      font-size: 13px;
      text-shadow: 1px 1px 1px #fff;
      padding-right: 2em;
    }
    .config-form table td{
      color: {{$color}};
      font-weight: normal;
      font-size: 13px;
      text-shadow: 1px 1px 1px #fff;
    }
    .required{
      color:red;
      font-weight:normal;
    }


    input.tab_ctrl {
      position: absolute;
      opacity: 0;
      z-index: -1;
    }


    .tabs {
      border-radius: 8px;
      overflow: hidden;
      box-shadow: 0 4px 4px -2px rgba(0, 0, 0, 0.5);
    }

    .tab {
      width: 100%;
      color: {{$tabText}};
      overflow: hidden;
    }
    label.tab-label {
      display: flex;
      justify-content: space-between;
      padding: 1em;
      background: {{$tabInactive}};
      font-weight: bold;
      cursor: pointer;
      /* Icon */
    }
    .tab-label:hover {
      background: {{$tabActive}};
    }
    .tab-label::after {
      content: "❯";
      width: 1em;
      height: 1em;
      text-align: center;
      transition: all 0.35s;
    }
    .tab-content {
      max-height: 0;
      padding: 0 1em;
      color: #F072A9;
      background: white;
      transition: all 0.35s;
    }
    .tab-close {
      display: flex;
      justify-content: flex-end;
      padding: 1em;
      font-size: 0.75em;
      background: #F072A9;
      cursor: pointer;
    }
    .tab-close:hover {
      background: #eb448d;
    }

    input:checked + .tab-label {
      background: {{$tabActive}};
    }
    input:checked + .tab-label::after {
      transform: rotate(90deg);
    }
    input:checked ~ .tab-content {
      max-height: 500em;
      padding: 1em;
    }
    </style>
  </body>
</html>
`
