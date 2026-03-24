package main

import (
	// "fmt"
	"crypto/subtle"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	oscClient "gitlab.com/Depili/go-osc/osc"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
	"gitlab.com/clock-8001/clock-8001/v4/util"
)

func runHTTP() {
	if options.configFile == "" {
		// No config file specified, can't save the config
		log.Printf("No config specified, http config interface disabled")
		return
	}

	http.HandleFunc("/save", basicAuth(saveHandler))
	http.HandleFunc("/", basicAuth(indexHandler))
	http.HandleFunc("/export", func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Content-Disposition", "attachment;filename=clock.ini")
		http.ServeFile(res, req, options.configFile)
	})
	http.HandleFunc("/import", basicAuth(importHandler))
	http.HandleFunc("/api/settime", basicAuth(setTimeHandler))
	http.HandleFunc("/api/cue", basicAuth(cueTestHandler))

	log.Printf("HTTP config: listening on %v", options.HTTPPort)
	if err := http.ListenAndServe(options.HTTPPort, nil); err != nil {
		log.Printf("HTTP config server failed: %v", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t := htmlTemplate.New("config.html")

	t.Funcs(
		htmlTemplate.FuncMap{
			"checkbox": func(id string, label string, value bool) htmlTemplate.HTML {
				ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"checkbox\" id=\"%s\" name=\"%s\" ", id, label, id, id)
				if value {
					ret += " checked "
				}
				ret += "/></label>"
				return htmlTemplate.HTML(ret)
			},
			"number": func(id string, label string, value int) htmlTemplate.HTML {
				ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"number\" id=\"%s\" name=\"%s\" value=\"%d\" /></label>", id, label, id, id, value)
				return htmlTemplate.HTML(ret)
			},
			"text": func(id string, label string, value string) htmlTemplate.HTML {
				ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"text\" id=\"%s\" name=\"%s\" value=\"%s\" /></label>", id, label, id, id, value)
				return htmlTemplate.HTML(ret)
			},
			"color": func(id string, label string, value string) htmlTemplate.HTML {
				ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"color\" id=\"%s\" name=\"%s\" value=\"%s\" /></label>", id, label, id, id, value)
				return htmlTemplate.HTML(ret)
			},
			"counter": func(id string, label string, value int) htmlTemplate.HTML {
				ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"number\" min=\"0\" max=\"9\" id=\"%s\" name=\"%s\" value=\"%d\" /></label>", id, label, id, id, value)
				return htmlTemplate.HTML(ret)
			},
			"byte": func(id string, label string, value int) htmlTemplate.HTML {
				ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"number\" min=\"0\" max=\"255\" id=\"%s\" name=\"%s\" value=\"%d\" /></label>", id, label, id, id, value)
				return htmlTemplate.HTML(ret)
			},
			"uint8": func(id string, label string, value uint8) htmlTemplate.HTML {
				ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"number\" min=\"0\" max=\"255\" id=\"%s\" name=\"%s\" value=\"%d\" /></label>", id, label, id, id, value)
				return htmlTemplate.HTML(ret)
			},
			"add": func(a int, b int) int {
				return a + b
			},
			"version": func() string {
				return clock.VersionInfo()
			},
			"log": func() string {
				logFile.Seek(0, io.SeekStart)
				l, _ := io.ReadAll(logFile)
				return string(l)
			},
		})
	tmpl, err := t.Parse(configHTML)

	if err != nil {
		panic(err)
	}

	currentUser, err := user.Current()
	if err == nil && currentUser.Name == "root" {
		options.Raspberry = util.FileExists("/boot/config.txt")
	}

	if options.Raspberry {
		// Read out various config files for editing
		bytes, err := os.ReadFile("/boot/config.txt")
		if err != nil {
			panic(err)
		}
		options.ConfigTxt = string(bytes)
	}

	options.Fonts = make([]string, 0, 200)
	err = filepath.Walk(options.FontPath, func(path string, info os.FileInfo, err error) error {
		if matched, err := filepath.Match("*.ttf", filepath.Base(path)); err != nil {
			return err
		} else if matched {
			options.Fonts = append(options.Fonts, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error walking fontpath: %v", err)
	}

	if runtime.GOOS == "linux" {
		options.Serials = make([]string, 0, 20)
		err = filepath.Walk("/dev/", func(path string, info os.FileInfo, err error) error {
			if matched, err := filepath.Match("tty[AU]*", filepath.Base(path)); err != nil {
				return err
			} else if matched {
				options.Serials = append(options.Serials, path)
			}
			return nil
		})
		if err != nil {
			log.Fatalf("Error walking fontpath: %v", err)
		}
	}

	options.Timezones = tzList

	log.Printf("fonts: %v", options.Fonts)
	err = tmpl.Execute(w, options)
	if err != nil {
		panic(err)
	}
}

func importHandler(w http.ResponseWriter, r *http.Request) {

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("import")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	dst, err := os.OpenFile(options.configFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	log.Printf("Writing new config ini file")
	// Copy the uploaded file to the created file on the filesystem
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Signal the main loop to reload config
	select {
	case confChan <- true:
	default:
	}

	tmpl, err := htmlTemplate.New("confirm.html").Parse(confirmHTML)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		panic(err)
	}
}

// TODO: validation
func saveHandler(w http.ResponseWriter, r *http.Request) {
	var errors string
	var err error

	newOptions.EngineOptions, errors = util.ParseEngineOptions(r)

	// Booleans, no validation on them
	newOptions.Debug = r.FormValue("Debug") != ""
	newOptions.DisableHTTP = r.FormValue("DisableHTTP") != ""
	newOptions.NoARCorrection = r.FormValue("NoARCorrection") != ""
	newOptions.Countup = r.FormValue("Countup") != ""
	newOptions.DrawBoxes = r.FormValue("DrawBoxes") != ""
	newOptions.SignalFollow = r.FormValue("signal-hw-follow") != ""
	newOptions.SignalToBG = r.FormValue("SignalToBG") != ""
	newOptions.AudioEnabled = r.FormValue("AudioEnabled") != ""
	newOptions.TODBeep = r.FormValue("TODBeep") != ""
	newOptions.VoiceEnabled = r.FormValue("VoiceEnabled") != ""
	newOptions.FullScreen = r.FormValue("FullScreen") != ""
	newOptions.HideHours = r.FormValue("hide-hours") != ""
	newOptions.IconsDisable = r.FormValue("disable-icons") != ""
	newOptions.CueFullScreen = r.FormValue("cue-fullscreen") != ""
	newOptions.DynamicBG = r.FormValue("DynamicBG") != ""
	newOptions.VFD = r.FormValue("VFD") != ""

	newOptions.CuePosX, err = strconv.Atoi(r.FormValue("cue-pos-x"))
	errors += util.ValidateNumber(err, "Perfect Cue X position")
	if err == nil && newOptions.CuePosX < 0 {
		errors += "<li>Perfect Cue X position must be 0 or greater</li>"
	}

	newOptions.CuePosY, err = strconv.Atoi(r.FormValue("cue-pos-y"))
	errors += util.ValidateNumber(err, "Perfect Cue Y position")
	if err == nil && newOptions.CuePosY < 0 {
		errors += "<li>Perfect Cue Y position must be 0 or greater</li>"
	}

	newOptions.CueSize, err = strconv.Atoi(r.FormValue("cue-size"))
	errors += util.ValidateNumber(err, "PerfectCue overlay size")
	if err == nil && newOptions.CueSize < 1 {
		errors += "<li>PerfectCue overlay size must be at least 1 pixel</li>"
	}
	newOptions.CueWidth = newOptions.CueSize
	newOptions.CueHeight = newOptions.CueSize

	// Strings, will not be validated
	newOptions.HTTPUser = r.FormValue("HTTPUser")
	newOptions.HTTPPassword = r.FormValue("HTTPPassword")

	newOptions.VoiceDir = r.FormValue("VoiceDir")

	// Countdown face target
	t := r.FormValue("CountdownTarget")
	match, err := regexp.MatchString(`^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d`, t)
	if err != nil || !match {
		errors += fmt.Sprintf("<li>Countdown face target time is invalid (%s): %s", t, err)
	}
	newOptions.CountdownTarget = t

	// Clock face type
	newOptions.Face = r.FormValue("Face")
	if f := newOptions.Face; (f != "countdown") && (f != "round") && (f != "dual-round") && (f != "text") && (f != "quad") && (f != "dual") && (f != "max") && (f != "small") && (f != "single") && (f != "144") && (f != "192") && (f != "288x144") {
		errors += fmt.Sprintf("<li>Clock face selection is invalid (%s)</li>", newOptions.Face)
	}

	// Signal hardware type
	newOptions.SignalType = r.FormValue("signal-hw-type")
	if t := newOptions.SignalType; (t != "unicorn-hd") && (t != "none") {
		errors += fmt.Sprintf("<li>Signal hardware type selection is invalid (%s)</li>", newOptions.SignalType)
	}

	// Filenames
	newOptions.NumberFont = r.FormValue("NumberFont")
	errors += util.ValidateFile(newOptions.NumberFont, "Number font")
	newOptions.LabelFont = r.FormValue("LabelFont")
	errors += util.ValidateFile(newOptions.LabelFont, "Label font")
	newOptions.IconFont = r.FormValue("IconFont")
	errors += util.ValidateFile(newOptions.IconFont, "Icon font")
	newOptions.Background = r.FormValue("Background")
	// Missing BG is totally OK
	newOptions.BackgroundPath = r.FormValue("BackgroundPath")
	// Missing BG path is totally OK
	newOptions.Font = r.FormValue("Font")
	errors += util.ValidateFile(newOptions.Font, "Font for round clocks")

	// Integers
	newOptions.NumberFontSize, err = strconv.Atoi(r.FormValue("NumberFontSize"))
	errors += util.ValidateNumber(err, "Number font size")

	alpha, err := strconv.Atoi(r.FormValue("row1-alpha"))
	errors += util.ValidateNumber(err, "Row1 alpha")
	newOptions.Row1Alpha = uint8(alpha)

	alpha, err = strconv.Atoi(r.FormValue("row2-alpha"))
	errors += util.ValidateNumber(err, "Row2 alpha")
	newOptions.Row2Alpha = uint8(alpha)

	alpha, err = strconv.Atoi(r.FormValue("row3-alpha"))
	errors += util.ValidateNumber(err, "Row3 alpha")
	newOptions.Row3Alpha = uint8(alpha)

	alpha, err = strconv.Atoi(r.FormValue("row4-alpha"))
	errors += util.ValidateNumber(err, "Row4 alpha")
	newOptions.Row4Alpha = uint8(alpha)

	alpha, err = strconv.Atoi(r.FormValue("label-alpha"))
	errors += util.ValidateNumber(err, "Label alpha")
	newOptions.LabelAlpha = uint8(alpha)

	alpha, err = strconv.Atoi(r.FormValue("label-bg-alpha"))
	errors += util.ValidateNumber(err, "Label background alpha")
	newOptions.LabelBGAlpha = uint8(alpha)

	alpha, err = strconv.Atoi(r.FormValue("timer-bg-alpha"))
	errors += util.ValidateNumber(err, "Timer bg alpha")
	newOptions.TimerBGAlpha = uint8(alpha)

	newOptions.SignalBrightness, err = strconv.Atoi(r.FormValue("signal-hw-brightness"))
	errors += util.ValidateNumber(err, "Signal hardware brightness")

	// Colors
	newOptions.TextColor = r.FormValue("TextColor")
	errors += util.ValidateColor(newOptions.TextColor, "Round clock text color")
	newOptions.SecondColor = r.FormValue("SecColor")
	errors += util.ValidateColor(newOptions.SecondColor, "Round clock second ring color")
	newOptions.StaticColor = r.FormValue("StaticColor")
	errors += util.ValidateColor(newOptions.StaticColor, "Round clock static ring color")
	newOptions.CountdownColor = r.FormValue("CountdownColor")
	errors += util.ValidateColor(newOptions.CountdownColor, "Round clock countdown color")

	newOptions.Row1Color = r.FormValue("Row1Color")
	errors += util.ValidateColor(newOptions.Row1Color, "Text clock row 1 color")
	newOptions.Row2Color = r.FormValue("Row2Color")
	errors += util.ValidateColor(newOptions.Row2Color, "Text clock row 2 color")
	newOptions.Row3Color = r.FormValue("Row3Color")
	errors += util.ValidateColor(newOptions.Row3Color, "Text clock row 3 color")
	newOptions.Row4Color = r.FormValue("Row4Color")
	errors += util.ValidateColor(newOptions.Row4Color, "Text clock row 4 color")

	newOptions.LabelColor = r.FormValue("LabelColor")
	errors += util.ValidateColor(newOptions.LabelColor, "Text clock label color")
	newOptions.LabelBG = r.FormValue("LabelBG")
	errors += util.ValidateColor(newOptions.LabelBG, "Text clock label background color")
	newOptions.TimerBG = r.FormValue("TimerBG")
	errors += util.ValidateColor(newOptions.TimerBG, "Text clock timer background color")

	newOptions.BackgroundColor = r.FormValue("BackgroundColor")
	errors += util.ValidateColor(newOptions.BackgroundColor, "Background color")

	newOptions.HTTPPort = r.FormValue("HTTPPort")
	errors += util.ValidateAddr(newOptions.HTTPPort, "HTTP config interface address")

	// GPIO pulser
	newOptions.GpioEnabled = r.FormValue("gpio-enabled") != ""
	newOptions.SecA = r.FormValue("gpio-seconds-a-pin")
	newOptions.SecPulse = r.FormValue("gpio-seconds-pulse-pin")
	newOptions.SecTrigger = r.FormValue("gpio-seconds-trigger")
	newOptions.MinA = r.FormValue("gpio-minutes-a-pin")
	newOptions.MinPulse = r.FormValue("gpio-minutes-pulse-pin")
	newOptions.MinTrigger = r.FormValue("gpio-minutes-trigger")
	newOptions.HourA = r.FormValue("gpio-hours-a-pin")
	newOptions.HourPulse = r.FormValue("gpio-hours-pulse-pin")
	newOptions.HourTrigger = r.FormValue("gpio-hours-trigger")
	newOptions.InvertPolarity = r.FormValue("gpio-invert-polarity") != ""

	pulseDuration, err := strconv.Atoi(r.FormValue("gpio-pulse-duration"))
	errors += util.ValidateNumber(err, "GPIO pulse duration")
	newOptions.PulseDuration = pulseDuration

	if errors != "" {
		t := htmlTemplate.New("config.html")

		t.Funcs(
			htmlTemplate.FuncMap{
				"checkbox": func(id string, label string, value bool) htmlTemplate.HTML {
					ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"checkbox\" id=\"%s\" name=\"%s\" ", id, label, id, id)
					if value {
						ret += " checked "
					}
					ret += "/></label>"
					return htmlTemplate.HTML(ret)
				},
				"number": func(id string, label string, value int) htmlTemplate.HTML {
					ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"number\" id=\"%s\" name=\"%s\" value=\"%d\" /></label>", id, label, id, id, value)
					return htmlTemplate.HTML(ret)
				},
				"text": func(id string, label string, value string) htmlTemplate.HTML {
					ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"text\" id=\"%s\" name=\"%s\" value=\"%s\" /></label>", id, label, id, id, value)
					return htmlTemplate.HTML(ret)
				},
				"color": func(id string, label string, value string) htmlTemplate.HTML {
					ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"color\" id=\"%s\" name=\"%s\" value=\"%s\" /></label>", id, label, id, id, value)
					return htmlTemplate.HTML(ret)
				},
				"counter": func(id string, label string, value int) htmlTemplate.HTML {
					ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"number\" min=\"0\" max=\"9\" id=\"%s\" name=\"%s\" value=\"%d\" /></label>", id, label, id, id, value)
					return htmlTemplate.HTML(ret)
				},
				"byte": func(id string, label string, value int) htmlTemplate.HTML {
					ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"number\" min=\"0\" max=\"255\" id=\"%s\" name=\"%s\" value=\"%d\" /></label>", id, label, id, id, value)
					return htmlTemplate.HTML(ret)
				},
				"uint8": func(id string, label string, value uint8) htmlTemplate.HTML {
					ret := fmt.Sprintf("<label for=\"%s\"><span>%s</span><input type=\"number\" min=\"0\" max=\"255\" id=\"%s\" name=\"%s\" value=\"%d\" /></label>", id, label, id, id, value)
					return htmlTemplate.HTML(ret)
				},
				"add": func(a int, b int) int {
					return a + b
				},
			})
		tmpl, err := t.Parse(configHTML)
		if err != nil {
			panic(err)
		}
		newOptions.Errors = htmlTemplate.HTML(fmt.Sprintf("<ul>%s</ul>", errors))
		err = tmpl.Execute(w, newOptions)
		if err != nil {
			panic(err)
		}
	} else {
		log.Printf("Writing new config ini file")
		newOptions.writeConfig(options.configFile)

		// TODO render success page

		if r.FormValue("configtxt") != "" {
			bytes, err := os.ReadFile("/boot/config.txt")
			check(err)
			currentConfig := string(bytes)

			if r.FormValue("configtxt") != currentConfig {
				log.Printf("Writing /boot/config.txt")
				f, err := os.Create("/boot/config.txt")
				check(err)
				_, err = f.WriteString(r.FormValue("configtxt"))
				check(err)
				f.Sync()
				f.Close()
				// reboot the rpi
				go delayedReboot()
			}

		}

		// Signal the main loop to reload config
		select {
		case confChan <- true:
		default:
		}

		// Render success page

		tmpl, err := htmlTemplate.New("confirm.html").Parse(confirmHTML)
		if err != nil {
			panic(err)
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			panic(err)
		}
	}
}

// Reboot the pi after a short delay
func delayedReboot() {
	time.Sleep(time.Millisecond * 500)
	cmd := exec.Command("reboot")
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func (options *clockOptions) writeConfig(path string) {
	tmpl, err := template.New("config.ini").Parse(configTemplate)
	if err != nil {
		panic(err)
	}
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(f, options)
	if err != nil {
		panic(err)
	}
	f.Sync()
	f.Close()
}

func findCommandPath(name string, fallbacks []string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	for _, candidate := range fallbacks {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func syncHardwareClock() error {
	hwclockPath := findCommandPath("hwclock", []string{"/usr/sbin/hwclock", "/sbin/hwclock", "/usr/bin/hwclock", "/bin/hwclock"})
	if hwclockPath == "" {
		return fmt.Errorf("hwclock command not found")
	}

	args := []string{"--systohc", "--utc"}
	hwCmd := exec.Command(hwclockPath, args...)
	if err := hwCmd.Run(); err == nil {
		return nil
	} else {
		sudoPath := findCommandPath("sudo", []string{"/usr/bin/sudo", "/bin/sudo"})
		if sudoPath == "" {
			return fmt.Errorf("direct hwclock failed and sudo command not found: %w", err)
		}
		fullArgs := append([]string{"-n", hwclockPath}, args...)
		sudoCmd := exec.Command(sudoPath, fullArgs...)
		if sudoErr := sudoCmd.Run(); sudoErr != nil {
			return fmt.Errorf("direct hwclock failed: %w; sudo hwclock failed: %v", err, sudoErr)
		}
		log.Printf("Web UI settime: RTC updated via sudo hwclock fallback")
		return nil
	}
}

func setTimeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	timeStr := r.FormValue("time")
	if timeStr == "" {
		http.Error(w, "Missing time parameter", http.StatusBadRequest)
		return
	}

	_, lookErr := exec.LookPath("date")
	if lookErr != nil {
		http.Error(w, "date command not found", http.StatusInternalServerError)
		return
	}

	// Parse ISO 8601 format (RFC3339Nano) to support sub-second precision.
	// Accepts: 2026-03-20T17:04:50.123Z (from toISOString) or 2026-03-20T10:04:50 (legacy).
	var parsedTime time.Time
	if t, parseErr := time.Parse(time.RFC3339Nano, timeStr); parseErr == nil {
		parsedTime = t
	} else {
		match, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T([01]\d|2[0-3]):[0-5]\d:[0-5]\d$`, timeStr)
		if !match {
			http.Error(w, "Invalid time format", http.StatusBadRequest)
			return
		}
		legacyTime, legacyErr := time.ParseInLocation("2006-01-02T15:04:05", timeStr, time.Local)
		if legacyErr != nil {
			http.Error(w, "Invalid time format", http.StatusBadRequest)
			return
		}
		parsedTime = legacyTime
	}

	// Use @seconds.nanoseconds format for nanosecond precision.
	dateString := fmt.Sprintf("@%d.%09d", parsedTime.Unix(), parsedTime.Nanosecond())
	log.Printf("Web UI settime: setting system date to %s (raw input: %s)", parsedTime.Format(time.RFC3339Nano), timeStr)
	cmd := exec.Command("date", "--set", dateString) // #nosec strict validation above
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to set system time: %v", err)
		http.Error(w, "Failed to set time", http.StatusInternalServerError)
		return
	}

	// Best-effort RTC sync so time survives reboot even when NTP is disabled.
	if err := syncHardwareClock(); err != nil {
		log.Printf("Warning: failed to update hardware clock: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func localOSCClient() (*oscClient.Client, error) {
	if options.EngineOptions == nil {
		return nil, errors.New("engine options not initialized")
	}
	if options.EngineOptions.DisableOSC {
		return nil, errors.New("osc control is disabled")
	}

	listenAddr := strings.TrimSpace(options.EngineOptions.ListenAddr)
	if listenAddr == "" {
		return nil, errors.New("osc listen address is empty")
	}

	portStr := ""
	if strings.HasPrefix(listenAddr, ":") {
		portStr = strings.TrimPrefix(listenAddr, ":")
	} else {
		_, p, err := net.SplitHostPort(listenAddr)
		if err != nil {
			return nil, fmt.Errorf("invalid osc listen address %q: %w", listenAddr, err)
		}
		portStr = p
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid osc port %q: %w", portStr, err)
	}

	return oscClient.NewClient("127.0.0.1", port), nil
}

func cueTestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	action := r.FormValue("action")
	if action == "" {
		http.Error(w, "Missing action parameter", http.StatusBadRequest)
		return
	}

	client, err := localOSCClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var msg *oscClient.Message
	switch action {
	case "right":
		msg = oscClient.NewMessage("/clock/cue/right")
		msg.Append("")
	case "left":
		msg = oscClient.NewMessage("/clock/cue/left")
		msg.Append("")
	case "blank_on":
		msg = oscClient.NewMessage("/clock/cue/blank")
		msg.Append("")
		msg.Append(true)
	case "blank_off":
		msg = oscClient.NewMessage("/clock/cue/blank")
		msg.Append("")
		msg.Append(false)
	default:
		http.Error(w, "Invalid action parameter", http.StatusBadRequest)
		return
	}

	if err := client.Send(msg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to send cue OSC message: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func basicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()

		if !ok || subtle.ConstantTimeCompare([]byte(options.HTTPUser), []byte(user)) != 1 || subtle.ConstantTimeCompare([]byte(options.HTTPPassword), []byte(pass)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Clock-8001 config"`)
			w.WriteHeader(401)
			w.Write([]byte("Unauthorised.\n"))
			return
		}

		handler(w, r)
	}
}
