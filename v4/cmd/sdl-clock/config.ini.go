package main

const configTemplate = `# Example configuration file for clock-8001
# Lines starting with '#' are comments and
# are ignored by clock-8001

# Clock face to use. (countdown, round, dual-round, text, single or small)
Face={{.Face}}

# Username and password for the web configuration interface
HTTPUser={{.HTTPUser}}
HTTPPassword={{.HTTPPassword}}

# Set to true to use 12 hour format for time-of-day display.
Format12h={{.EngineOptions.Format12h}}

# Set to true to disable detection of official raspberry pi display for aspect ratio correction
NoARCorrection={{.NoARCorrection}}

# Start in full screen mode (excluding raspberry pi images)
FullScreen={{.FullScreen}}

# Background image support. You need to provide the background in
# the correct resolution as a png or jpeg image.
Background={{.Background}}

# Background image path for OSC background selection
BackgroundPath={{.BackgroundPath}}

# Background color, used if no image is provided
BackgroundColor={{.BackgroundColor}}

# Truetype fonts for the text clock face

# Font for numbers
NumberFont={{.NumberFont}}

# Font for label texts
LabelFont={{.LabelFont}}

# Font for icons
IconFont={{.IconFont}}

# Enable beeps for countdowns, currently produces the BBC bips on 5 last seconds of a countdown
AudioEnabled={{.AudioEnabled}}

# Enable voice sample triggereing for countdowns and media durations.
VoiceEnabled={{.VoiceEnabled}}

# Directory to load voice samples from. Samples should be named 1234.wav where the numbers are the trigger time in seconds
VoiceDir={{.VoiceDir}}

# Enable audio beeps on each full hour
TODBeep={{.TODBeep}}

# Show clock info for X seconds at startup
info-timer={{.EngineOptions.ShowInfo}}

# InterSpace Industries Countdown2 UDP timer protocol
# (also supported by Stagetimer2 and Irisdown):
# set to off, send or receive
udp-time={{.EngineOptions.UDPTime}}

# Timer to send / receive as UDP timer 1 (port 36700)
udp-timer-1={{.EngineOptions.UDPTimer1}}

# Timer to send / receive as UDP timer 1 (port 36700)
udp-timer-2={{.EngineOptions.UDPTimer2}}

# Target for the countdown clock face (YYYY-MM-DD HH:MM:SS)
CountdownTarget={{.CountdownTarget}}

# Countdown face counts up instead if set to true
Countup={{.Countup}}

# Time sources
#
# The single round clock uses source 1 as the main display and source 2 as a secondary timer
# The dual round clock mode uses all four sources, with 1 and 2 in the left clock and 3 and 4 in the right
#
# The round clocks only support timers as the secondary display source, as others can't be compacted to 4 characters
#
# The sources choose their displayed time in the following priority if enabled:
# 1. LTC
# 2. Associated timer if running
# 3. Time of day
# 4. Blank display

# Text label for time source
source1.text={{.EngineOptions.Source1.Text}}
# Set to true to enable LTC input on this source
source1.ltc={{.EngineOptions.Source1.LTC}}
# Set to true for countdown / count up timer input on this source
source1.timer={{.EngineOptions.Source1.Timer}}
# Set to true to show counter end time instead of time remaining (if available)
source1.timer-target={{.EngineOptions.Source1.TimerTarget}}
# Counter number for timer support (0-9)
source1.counter={{.EngineOptions.Source1.Counter}}
# Set to true to enable time of day input on this source
source1.tod={{.EngineOptions.Source1.Tod}}
# Time zone for the time of day input
source1.timezone={{.EngineOptions.Source1.TimeZone}}
# Initially hide this source, can be toggled via OSC
source1.hidden={{.EngineOptions.Source1.Hidden}}
# Background color for overtime countdown timers
source1.overtime-color={{.EngineOptions.Source1.OvertimeColor}}

source2.text={{.EngineOptions.Source2.Text}}
source2.ltc={{.EngineOptions.Source2.LTC}}
source2.timer={{.EngineOptions.Source2.Timer}}
source2.timer-target={{.EngineOptions.Source2.TimerTarget}}
source2.counter={{.EngineOptions.Source2.Counter}}
source2.tod={{.EngineOptions.Source2.Tod}}
source2.timezone={{.EngineOptions.Source2.TimeZone}}
source2.hidden={{.EngineOptions.Source2.Hidden}}
source2.overtime-color={{.EngineOptions.Source2.OvertimeColor}}

source3.text={{.EngineOptions.Source3.Text}}
source3.ltc={{.EngineOptions.Source3.LTC}}
source3.timer={{.EngineOptions.Source3.Timer}}
source3.timer-target={{.EngineOptions.Source3.TimerTarget}}
source3.counter={{.EngineOptions.Source3.Counter}}
source3.tod={{.EngineOptions.Source3.Tod}}
source3.timezone={{.EngineOptions.Source3.TimeZone}}
source3.hidden={{.EngineOptions.Source3.Hidden}}
source3.overtime-color={{.EngineOptions.Source3.OvertimeColor}}

source4.text={{.EngineOptions.Source4.Text}}
source4.ltc={{.EngineOptions.Source4.LTC}}
source4.timer={{.EngineOptions.Source4.Timer}}
source4.timer-target={{.EngineOptions.Source4.TimerTarget}}
source4.counter={{.EngineOptions.Source4.Counter}}
source4.tod={{.EngineOptions.Source4.Tod}}
source4.timezone={{.EngineOptions.Source4.TimeZone}}
source4.hidden={{.EngineOptions.Source4.Hidden}}
source4.overtime-color={{.EngineOptions.Source4.OvertimeColor}}


# Overtime behaviour

# Countdown readout for overtime timers
# zero - show 00:00:00
# blank - show nothing
# continue - continue counting up
overtime-count-mode={{.EngineOptions.OvertimeCountMode}}

# Extra visibility for overtime timers
# none - no extra visibility
# blink - blink the timer readout
# background - change the background color to the overtime color
# both - change background and blink
overtime-visibility={{.EngineOptions.OvertimeVisibility}}

# Timer color signals

# Automatically set the signal color based on time remaining
auto-signals={{.EngineOptions.AutoSignals}}

# In automation mode, set a color on timer start
signal-start={{.EngineOptions.SignalStart}}

# Start signal color (in HTML color format #FFFFFF)
signal-color-start={{.EngineOptions.SignalColorStart}}

# Time threshold for warning color, in seconds. Set to 0 to disable
signal-threshold-warning={{.EngineOptions.SignalThresholdWarning}}

# Warning signal color (in HTML color format #FFFFFF)
signal-color-warning={{.EngineOptions.SignalColorWarning}}

# Time threshold for end color, in seconds.
signal-threshold-end={{.EngineOptions.SignalThresholdEnd}}

# End signal color (in HTML color format #FFFFFF)
signal-color-end={{.EngineOptions.SignalColorEnd}}

# Signal hardware type: supported: none, unicorn-hd
signal-hw-type={{.SignalType}}

# Signal hardware group number
signal-hw-group={{.EngineOptions.SignalHardware}}

# Signal hardware brighness adjustment 0-255, 0 == disabled, 255 == full brightness
signal-hw-brightness={{.SignalBrightness}}

# Hardware signal follows source 1 signal color
signal-hw-follow={{.SignalFollow}}

# Use hardware signal color as clock background
SignalToBG={{.SignalToBG}}

# Mitti and Millumin

# Counter number for Mitti OSC feedback
mitti={{.EngineOptions.Mitti}}

# Counter number for Millumin OSC feedback
millumin={{.EngineOptions.Millumin}}

# Millumin layer ignore regexp
millumin-ignore-layer={{.EngineOptions.Ignore}}

# Picturall

# Enable picturall integration
picturall-enabled={{.EngineOptions.PicturallEnabled}}

# Picturall address. Leave blank for autodiscovery. Autodiscovery will use the first instance that responds.
picturall-address={{.EngineOptions.PicturallAddress}}

# Picturall port
picturall-port={{.EngineOptions.PicturallPort}}

# Pictural destination timer
picturall-timer={{.EngineOptions.PicturallTimer}}

# Picturall show looping media
picturall-loops={{.EngineOptions.PicturallLoops}}

# Picturall show streaming media
picturall-streams={{.EngineOptions.PicturallStreams}}

# Picturall show media name as tally message
picturall-media-name={{.EngineOptions.PicturallMediaName}}

# Picturall message timeout, in milliseconds
picturall-timeout={{.EngineOptions.PicturallTimeout}}

# Picturall layer ignore list, comma separated
picturall-ignore-layers={{.EngineOptions.PicturallIgnoreLayers}}

# Picturall media name color & background
picturall-media-color={{.EngineOptions.PicturallMediaColor}}
picturall-media-bg={{.EngineOptions.PicturallMediaBG}}

# Vmix

# Set to true to enable vMix integration
vmix-enabled={{.EngineOptions.VmixEnabled}}

# vMix ip address
vmix-address={{.EngineOptions.VmixAddress}}

# vMix http port
vmix-port={{.EngineOptions.VmixPort}}

# vMix timer number to use
vmix-timer={{.EngineOptions.VmixTimer}}

# vMix show looping media
vmix-loops={{.EngineOptions.VmixLoops}}

# vMix only show media in PGM, no overlays
vmix-pgm-only={{.EngineOptions.VmixPGMOnly}}

# vMix also show media on preview
vmix-show-pvm={{.EngineOptions.VmixPVM}}

# vMix ignore media on given overlays, comma separated list of overlay numbers, eg. 1,4,6
vmix-ignore-overlays={{.EngineOptions.VmixIgnoreOverlays}}

# vMix show playing media name as clock tally message
vmix-media-name={{.EngineOptions.VmixMediaName}}

# vMix media name color and background, html colors
vmix-media-color={{.EngineOptions.VmixMediaColor}}
vmix-media-bg={{.EngineOptions.VmixMediaBG}}

# vMix timers, in milliseconds:
# Polling interval, adjust if there is excessive load on the vMix machine.
vmix-interval={{.EngineOptions.VmixInterval}}
# Communication timeout
vmix-timeout={{.EngineOptions.VmixTimeout}}

# Limitimer
# see https://gitlab.com/clock-8001/clock-8001/-/blob/master/limitimer.md

# Mode, off, send or receive
limitimer-mode={{.EngineOptions.LimitimerMode}}
# Serial device to use for limitimer communications
limitimer-serial={{.EngineOptions.LimitimerSerial}}

# Receive and broadcast flags per limitimer program:

# Limitimer program 1 as timer 1
limitimer-receive-timer1={{.EngineOptions.LimitimerReceive1}}
# Limitimer program 2 as timer 2
limitimer-receive-timer2={{.EngineOptions.LimitimerReceive2}}
# Limitimer program 3 as timer 3
limitimer-receive-timer3={{.EngineOptions.LimitimerReceive3}}
# Limitimer session (program 4) as timer 4
limitimer-receive-timer4={{.EngineOptions.LimitimerReceive4}}
# Limitimer program selected on controller as timer 5
limitimer-receive-timer5={{.EngineOptions.LimitimerReceive5}}

# Send program 1 as timer 1 to other clocks via OSC
limitimer-broadcast-timer1={{.EngineOptions.LimitimerBroadcast1}}
# Send program 2 as timer 2 to other clocks via OSC
limitimer-broadcast-timer2={{.EngineOptions.LimitimerBroadcast2}}
# Send program 3 as timer 3 to other clocks via OSC
limitimer-broadcast-timer3={{.EngineOptions.LimitimerBroadcast3}}
# Send session (program 4) as timer 4 to other clocks via OSC
limitimer-broadcast-timer4={{.EngineOptions.LimitimerBroadcast4}}
# Send program selected on controller as timer 5 to other clocks via OSC
limitimer-broadcast-timer5={{.EngineOptions.LimitimerBroadcast5}}


# Font to use
Font={{.Font}}

# Colors, in hex format, #XXX or #XXXXXX

# Round clocks

# Color for text
text-color={{.TextColor}}

# Color for the 12 static "hour" markers
static-color={{.StaticColor}}

# Color for the second ring dots
second-color={{.SecondColor}}

# Color for the secondary countdown display
countdown-color={{.CountdownColor}}

# Text clock

# Color for labels
label-color={{.LabelColor}}
# Alpha for labels
label-alpha={{.LabelAlpha}}

# Timer row 1 color
row1-color={{.Row1Color}}
# Timer row 1 alpha
row1-alpha={{.Row1Alpha}}

# Timer row 2 color
row2-color={{.Row2Color}}
# Timer row 2 alpha
row2-alpha={{.Row2Alpha}}

# Timer row 3 color
row3-color={{.Row3Color}}
# Timer row 3 alpha
row3-alpha={{.Row3Alpha}}

# Draw background boxes for timers and labels
draw-boxes={{.DrawBoxes}}

# Timer background color
timer-bg-color={{.TimerBG}}
# Timer background alpha
timer-bg-alpha={{.TimerBGAlpha}}

# Label background color
label-bg-color={{.LabelBG}}
# Label background alpa
label-bg-alpha={{.LabelBGAlpha}}

# Numbers font size
numbers-size={{.NumberFontSize}}

# Engine internals

# Set to true to output verbose debug information
Debug={{.Debug}}

# Flashing interval for ellapsed countdowns, in milliseconds
Flash={{.EngineOptions.Flash}}

# Set to true to disable remote osc commands
DisableOSC={{.EngineOptions.DisableOSC}}

# Address to listen for osc commands. 0.0.0.0 defaults to all network interfaces
ListenAddr={{.EngineOptions.ListenAddr}}

# Timeout for clearing OSC text display messages
Timeout={{.EngineOptions.Timeout}}

# Set to true to disable sending of the OSC feedback messages
DisableFeedback={{.EngineOptions.DisableFeedback}}

# Address to send OSC feedback to. 255.255.255.255 broadcasts to all network interfaces
Connect={{.EngineOptions.Connect}}

# Set to true to disable the web configuration interface
DisableHTTP={{.DisableHTTP}}

# Port to listen for the web configuration. Needs to be in format of ":1234".
HTTPPort={{.HTTPPort}}

# Set to true to disable LTC timecode display mode
DisableLTC={{.EngineOptions.DisableLTC}}

# Controls what is displayed on the clock ring in LTC mode, false = frames, true = seconds
LTCSeconds={{.EngineOptions.LTCSeconds}}

# Continue on internal clock if LTC signal is lost. If unset display will blank when signal is gone.
LTCFollow={{.EngineOptions.LTCFollow}}
`
