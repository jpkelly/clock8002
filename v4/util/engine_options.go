package util

import (
	"fmt"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
	"net/http"
	"regexp"
	"strconv"
)

// ParseEngineOptions takes a http request and uses the form data to fill out clock.EngineOptions struct
func ParseEngineOptions(r *http.Request) (*clock.EngineOptions, string) {
	var err error
	eo := &clock.EngineOptions{}
	errors := ""

	eo.Source1 = &clock.SourceOptions{}
	eo.Source2 = &clock.SourceOptions{}
	eo.Source3 = &clock.SourceOptions{}
	eo.Source4 = &clock.SourceOptions{}

	// Booleans, no validation on them
	eo.DisableOSC = r.FormValue("DisableOSC") != ""
	eo.DisableFeedback = r.FormValue("DisableFeedback") != ""
	eo.DisableLTC = r.FormValue("DisableLTC") != ""
	eo.LTCSeconds = r.FormValue("LTCSeconds") != ""
	eo.LTCFollow = r.FormValue("LTCFollow") != ""
	eo.Format12h = r.FormValue("Format12h") != ""

	eo.Source1.LTC = r.FormValue("source1-ltc") != ""
	eo.Source1.Timer = r.FormValue("source1-timer") != ""
	eo.Source1.TimerTarget = r.FormValue("source1-timer-target") != ""
	eo.Source1.Tod = r.FormValue("source1-tod") != ""
	eo.Source1.Hidden = r.FormValue("source1-hidden") != ""

	eo.Source2.LTC = r.FormValue("source2-ltc") != ""
	eo.Source2.Timer = r.FormValue("source2-timer") != ""
	eo.Source2.TimerTarget = r.FormValue("source2-timer-target") != ""
	eo.Source2.Tod = r.FormValue("source2-tod") != ""
	eo.Source2.Hidden = r.FormValue("source2-hidden") != ""

	eo.Source3.LTC = r.FormValue("source3-ltc") != ""
	eo.Source3.Timer = r.FormValue("source3-timer") != ""
	eo.Source3.TimerTarget = r.FormValue("source3-timer-target") != ""
	eo.Source3.Tod = r.FormValue("source3-tod") != ""
	eo.Source3.Hidden = r.FormValue("source3-hidden") != ""

	eo.Source4.LTC = r.FormValue("source4-ltc") != ""
	eo.Source4.Timer = r.FormValue("source4-timer") != ""
	eo.Source4.TimerTarget = r.FormValue("source4-timer-target") != ""
	eo.Source4.Tod = r.FormValue("source4-tod") != ""
	eo.Source4.Hidden = r.FormValue("source4-hidden") != ""

	eo.LimitimerReceive1 = r.FormValue("limitimer-receive-timer1") != ""
	eo.LimitimerReceive2 = r.FormValue("limitimer-receive-timer2") != ""
	eo.LimitimerReceive3 = r.FormValue("limitimer-receive-timer3") != ""
	eo.LimitimerReceive4 = r.FormValue("limitimer-receive-timer4") != ""
	eo.LimitimerReceive5 = r.FormValue("limitimer-receive-timer5") != ""

	eo.LimitimerBroadcast1 = r.FormValue("limitimer-broadcast-timer1") != ""
	eo.LimitimerBroadcast2 = r.FormValue("limitimer-broadcast-timer2") != ""
	eo.LimitimerBroadcast3 = r.FormValue("limitimer-broadcast-timer3") != ""
	eo.LimitimerBroadcast4 = r.FormValue("limitimer-broadcast-timer4") != ""
	eo.LimitimerBroadcast5 = r.FormValue("limitimer-broadcast-timer5") != ""

	eo.PicturallEnabled = r.FormValue("picturall-enabled") != ""
	eo.PicturallLoops = r.FormValue("picturall-loop") != ""
	eo.PicturallStreams = r.FormValue("picturall-streams") != ""
	eo.PicturallMediaName = r.FormValue("picturall-media-name") != ""

	eo.VmixEnabled = r.FormValue("vmix-enabled") != ""
	eo.VmixLoops = r.FormValue("vmix-loops") != ""
	eo.VmixPGMOnly = r.FormValue("vmix-pgm-only") != ""
	eo.VmixPVM = r.FormValue("vmix-show-pvm") != ""
	eo.VmixMediaName = r.FormValue("vmix-media-name") != ""

	eo.AutoSignals = r.FormValue("auto-signals") != ""
	eo.SignalStart = r.FormValue("signal-start") != ""

	// Strings, will not be validated
	eo.Source1.Text = r.FormValue("source1-text")
	eo.Source2.Text = r.FormValue("source2-text")
	eo.Source3.Text = r.FormValue("source3-text")
	eo.Source4.Text = r.FormValue("source4-text")

	eo.LimitimerSerial = r.FormValue("limitimer-serial")

	eo.PicturallAddress = r.FormValue("picturall-address")

	eo.VmixAddress = r.FormValue("vmix-address")

	// Limitimer mode
	eo.LimitimerMode = r.FormValue("limitimer-mode")
	if t := eo.LimitimerMode; (t != "off") && (t != "send") && (t != "receive") {
		errors += fmt.Sprintf("<li>Limitimer mode is invalid (%s)</li>", eo.LimitimerMode)
	}

	// UDPTime
	eo.UDPTime = r.FormValue("udp-time")
	if f := eo.UDPTime; (f != "off") && (f != "send") && (f != "receive") {
		errors += fmt.Sprintf("<li>UDP time selection is invalid (%s)</li>", eo.UDPTime)
	}

	// Overtime count mode
	eo.OvertimeCountMode = r.FormValue("overtime-count-mode")
	if f := eo.OvertimeCountMode; (f != "zero") && (f != "blank") && (f != "continue") {
		errors += fmt.Sprintf("<li>Overtime count mode selection is invalid (%s)</li>", eo.OvertimeCountMode)
	}

	// Overtime visibility
	eo.OvertimeVisibility = r.FormValue("overtime-visibility")
	if f := eo.OvertimeVisibility; (f != "blink") && (f != "none") && (f != "background") && (f != "both") {
		errors += fmt.Sprintf("<li>Overtime visibility selection is invalid (%s)</li>", eo.OvertimeVisibility)
	}

	// Addresses
	eo.ListenAddr = r.FormValue("ListenAddr")
	errors += ValidateAddr(eo.ListenAddr, "OSC listen address")
	eo.Connect = r.FormValue("Connect")
	errors += ValidateAddr(eo.Connect, "OSC feedback address")

	// Timezones
	eo.Source1.TimeZone = r.FormValue("source1-timezone")
	errors += ValidateTZ(eo.Source1.TimeZone, "Source 1 timezone")
	eo.Source2.TimeZone = r.FormValue("source2-timezone")
	errors += ValidateTZ(eo.Source2.TimeZone, "Source 2 timezone")
	eo.Source3.TimeZone = r.FormValue("source3-timezone")
	errors += ValidateTZ(eo.Source3.TimeZone, "Source 3 timezone")
	eo.Source4.TimeZone = r.FormValue("source4-timezone")
	errors += ValidateTZ(eo.Source4.TimeZone, "Source 4 timezone")

	// Regexp
	eo.Ignore = r.FormValue("millumin-ignore")
	_, err = regexp.Compile("(?i)" + eo.Ignore)
	if err != nil {
		errors += fmt.Sprintf("<li>Millumin layer ignore regexp: %v</li>", err)
	}

	// Integers
	eo.Flash, err = strconv.Atoi(r.FormValue("Flash"))
	errors += ValidateNumber(err, "Flash time")
	eo.Timeout, err = strconv.Atoi(r.FormValue("Timeout"))
	errors += ValidateNumber(err, "Tally message timeout")
	eo.Source1.Counter, err = strconv.Atoi(r.FormValue("source1-counter"))
	errors += ValidateNumber(err, "Source 1 timer")
	errors += ValidateTimer(eo.Source1.Counter, "Source 1 timer")
	eo.Source2.Counter, err = strconv.Atoi(r.FormValue("source2-counter"))
	errors += ValidateNumber(err, "Source 2 timer")
	errors += ValidateTimer(eo.Source2.Counter, "Source 2 timer")
	eo.Source3.Counter, err = strconv.Atoi(r.FormValue("source3-counter"))
	errors += ValidateNumber(err, "Source 3 timer")
	errors += ValidateTimer(eo.Source3.Counter, "Source 3 timer")
	eo.Source4.Counter, err = strconv.Atoi(r.FormValue("source4-counter"))
	errors += ValidateNumber(err, "Source 4 timer")
	errors += ValidateTimer(eo.Source4.Counter, "Source 4 timer")
	eo.Mitti, err = strconv.Atoi(r.FormValue("mitti"))
	errors += ValidateNumber(err, "Mitti destination timer")
	errors += ValidateTimer(eo.Mitti, "Mitti destination timer")
	eo.Millumin, err = strconv.Atoi(r.FormValue("millumin"))
	errors += ValidateNumber(err, "Millumin destination timer")
	errors += ValidateTimer(eo.Millumin, "Millumin destination timer")
	eo.ShowInfo, err = strconv.Atoi(r.FormValue("ShowInfo"))
	errors += ValidateNumber(err, "Time to show clock info on startup")

	eo.UDPTimer1, err = strconv.Atoi(r.FormValue("udp-timer-1"))
	errors += ValidateNumber(err, "UDP Timer 1")
	errors += ValidateTimer(eo.UDPTimer1, "UDP Timer 1")
	eo.UDPTimer2, err = strconv.Atoi(r.FormValue("udp-timer-2"))
	errors += ValidateNumber(err, "UDP Timer 2")
	errors += ValidateTimer(eo.UDPTimer1, "UDP Timer 2")

	eo.SignalThresholdWarning, err = strconv.Atoi(r.FormValue("signal-threshold-warning"))
	errors += ValidateNumber(err, "Warning signal threshold")

	eo.SignalThresholdEnd, err = strconv.Atoi(r.FormValue("signal-threshold-end"))
	errors += ValidateNumber(err, "End signal threshold")

	eo.SignalHardware, err = strconv.Atoi(r.FormValue("signal-hw-group"))
	errors += ValidateNumber(err, "Signal hardware group")

	eo.PicturallPort, err = strconv.Atoi(r.FormValue("picturall-port"))
	errors += ValidateNumber(err, "Picturall port")
	eo.PicturallTimer, err = strconv.Atoi(r.FormValue("picturall-timer"))
	errors += ValidateNumber(err, "Picturall timer")
	eo.PicturallTimeout, err = strconv.Atoi(r.FormValue("picturall-timeout"))
	errors += ValidateNumber(err, "Picturall timeout")

	eo.VmixPort, err = strconv.Atoi(r.FormValue("vmix-port"))
	errors += ValidateNumber(err, "vMix port")
	eo.VmixTimer, err = strconv.Atoi(r.FormValue("vmix-timer"))
	errors += ValidateNumber(err, "vMix timer")
	eo.VmixInterval, err = strconv.Atoi(r.FormValue("vmix-interval"))
	errors += ValidateNumber(err, "vMix interval")
	eo.VmixTimeout, err = strconv.Atoi(r.FormValue("vmix-timeout"))
	errors += ValidateNumber(err, "vMix timeout")

	// Picturall ignore layers
	picturallIgnoreLayers := r.FormValue("picturall-ignore-layers")
	if ok, err := regexp.MatchString(`^(?:\d+,?)+$`, picturallIgnoreLayers); len(picturallIgnoreLayers) == 0 || (ok && err == nil) {
		eo.PicturallIgnoreLayers = picturallIgnoreLayers
	} else {
		errors += "<li>Picturall layer ignore list is not valid</li>"
	}

	// vMix ignore overlays
	vmixIgnoreOverlays := r.FormValue("vmix-ignore-overlays")
	if ok, err := regexp.MatchString(`^(?:\d+,?)+$`, vmixIgnoreOverlays); len(vmixIgnoreOverlays) == 0 || (ok && err == nil) {
		eo.VmixIgnoreOverlays = vmixIgnoreOverlays
	} else {
		errors += "<li>vMix overlay ignore list is not valid</li>"
	}

	eo.SignalColorStart = r.FormValue("signal-color-start")
	errors += ValidateColor(eo.SignalColorStart, "Signal color: start")
	eo.SignalColorWarning = r.FormValue("signal-color-warning")
	errors += ValidateColor(eo.SignalColorStart, "Signal color: warning")
	eo.SignalColorEnd = r.FormValue("signal-color-end")
	errors += ValidateColor(eo.SignalColorStart, "Signal color: end")

	eo.Source1.OvertimeColor = r.FormValue("source1-overtime-color")
	errors += ValidateColor(eo.Source1.OvertimeColor, "Source1 overtime color")
	eo.Source2.OvertimeColor = r.FormValue("source2-overtime-color")
	errors += ValidateColor(eo.Source2.OvertimeColor, "Source2 overtime color")
	eo.Source3.OvertimeColor = r.FormValue("source3-overtime-color")
	errors += ValidateColor(eo.Source3.OvertimeColor, "Source3 overtime color")
	eo.Source4.OvertimeColor = r.FormValue("source4-overtime-color")
	errors += ValidateColor(eo.Source4.OvertimeColor, "Source4 overtime color")

	eo.PicturallMediaColor = r.FormValue("picturall-media-color")
	errors += ValidateColor(eo.PicturallMediaColor, "Picturall media text color")
	eo.PicturallMediaBG = r.FormValue("picturall-media-bg")
	errors += ValidateColor(eo.PicturallMediaBG, "Picturall media background color")

	eo.VmixMediaColor = r.FormValue("vmix-media-color")
	errors += ValidateColor(eo.VmixMediaColor, "vMix media text color")
	eo.VmixMediaBG = r.FormValue("vmix-media-bg")
	errors += ValidateColor(eo.VmixMediaBG, "vMix media background color")

	// Tricaster
	eo.TricasterEnabled = r.FormValue("tricaster-enabled") != ""
	eo.TricasterAddress = r.FormValue("tricaster-address")
	eo.TricasterUser = r.FormValue("tricaster-username")
	eo.TricasterPassword = r.FormValue("tricaster-password")
	eo.TricasterTimer, err = strconv.Atoi(r.FormValue("tricaster-timer"))
	errors += ValidateNumber(err, "Tricaster destination timer")
	errors += ValidateTimer(eo.TricasterTimer, "Tricaster destination timer")
	eo.TricasterEventTimer, err = strconv.Atoi(r.FormValue("tricaster-event-timer"))
	errors += ValidateNumber(err, "Tricaster event destination timer")
	errors += ValidateTimer(eo.TricasterEventTimer, "Tricaster event destination timer")
	eo.TricasterMediaName = r.FormValue("tricaster-media-name") != ""
	eo.TricasterPVM = r.FormValue("tricaster-show-pvm") != ""
	eo.TricasterMediaColor = r.FormValue("tricaster-media-color")
	errors += ValidateColor(eo.TricasterMediaColor, "Tricaster media text color")
	eo.TricasterMediaBG = r.FormValue("tricaster-media-bg")
	errors += ValidateColor(eo.TricasterMediaBG, "Tricaster media background color")
	eo.TricasterInterval, err = strconv.Atoi(r.FormValue("tricaster-interval"))
	errors += ValidateNumber(err, "Tricaster polling interval")
	eo.TricasterTimeout, err = strconv.Atoi(r.FormValue("tricaster-timeout"))
	errors += ValidateNumber(err, "Tricaster timeout")
	eo.TricasterEvent = r.FormValue("tricaster-event-enabled") != ""

	return eo, errors
}
