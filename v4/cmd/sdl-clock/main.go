package main

import (
	"errors"
	"fmt"
	"github.com/desertbit/timer"
	"github.com/jessevdk/go-flags"
	"github.com/veandco/go-sdl2/sdl"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	"gitlab.com/clock-8001/clock-8001/v4/util"
	"image/color"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
	_ "time/tzdata"
)

var parser = flags.NewParser(&options, flags.Default)
var showBackground bool
var backgroundNumber int

// Channel to notify for configuration changes
var confChan chan bool

const updateTime = time.Second / 30

func main() {
	var err error
	var info string

	log.SetFlags(log.Lmicroseconds)

	parseOptions()

	// Channel to notify for config changes
	confChan = make(chan bool, 1)

	if !options.DisableHTTP {
		go runHTTP()
	}

	if hw, ok := signalHardwareList[options.SignalType]; ok {
		hw.Init()
	}

	for _, m := range outputModules {
		m.Options()
	}

	log.Printf("Staring output modules")
	for _, m := range outputModules {
		go m.Run()
	}

	if runtime.GOOS == "windows" {
		// Windows is silly and allows too large windows...
		winWidth = 1280
		winHeight = 720
	}

	// Initialize SDL
	initSDL()
	defer sdl.Quit()
	defer window.Destroy()
	defer renderer.Destroy()

	isFullScreen := false
	if options.FullScreen {
		window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
		isFullScreen = true
	}

	setupScaling()
	initColors()
	initTextures()
	initAudio()

	if options.textClock {
		initTextClock()
	} else if options.Face == "288x144" {
		initSmallTextClock()
	} else if options.countdown {
		initCountdown()
	} else {
		initRoundClock()
	}

	// Trap SIGINT aka Ctrl-C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Intialize polling timers
	updateTicker := time.NewTicker(updateTime)
	eventTicker := time.NewTicker(time.Millisecond * 5)

	// Create main clock engine
	engine, err := clock.MakeEngine(options.EngineOptions)
	check(err)

	for i := 0; i < 3; i++ {
		engine.SetSourceColors(i, toRGBA(colors.row[i]), toRGBA(colors.rowBG[i]))
	}
	engine.SetTitleColors(toRGBA(colors.label), toRGBA(colors.labelBG))

	loadBackground(options.Background)

	log.Printf("Entering main loop")

	configStarted := false

	configTimer := timer.NewTimer(time.Second * 5)

	for {
		select {
		case <-sigChan:
			// SIGINT received, shutdown gracefully
			engine.Close()
			os.Exit(1)
		case <-confChan:
			log.Printf("Reloading config!")

			engine.Close()
			log.Printf("Clock engine closed")
			newOptions.configFile = options.configFile
			options = newOptions
			computeDerivedOptions()

			if hw, ok := signalHardwareList[options.SignalType]; ok {
				hw.Init()
			}
			log.Printf("HW init done")
			setupScaling()
			log.Printf("Scaling init done")
			initColors()
			log.Printf("Color init done")
			initTextures()
			log.Printf("texture init done")
			initAudio()

			log.Printf("->Initializing clock face")
			if options.textClock {
				initTextClock()
			} else if options.Face == "288x144" {
				initSmallTextClock()
			} else if options.countdown {
				initCountdown()
			} else {
				initRoundClock()
			}
			// Create main clock engine
			engine, err = clock.MakeEngine(options.EngineOptions)
			check(err)
			for i := 0; i < 3; i++ {
				engine.SetSourceColors(i, toRGBA(colors.row[i]), toRGBA(colors.rowBG[i]))
			}
			engine.SetTitleColors(toRGBA(colors.label), toRGBA(colors.labelBG))

			loadBackground(options.Background)
			log.Printf("Config reloaded!")
		case <-eventTicker.C:
			// SDL event polling
			e := sdl.PollEvent()
			switch t := e.(type) {
			case *sdl.QuitEvent:
				os.Exit(0)
			case *sdl.KeyboardEvent:
				key := t.Keysym.Sym
				if key == sdl.K_f {
					if !isFullScreen {
						window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
						setupScaling()
						initTextures()
						isFullScreen = true
					}
				} else if key == sdl.K_ESCAPE {
					if isFullScreen {
						window.SetFullscreen(0)
						setupScaling()
						initTextures()
						isFullScreen = false
					}
				} else if key == sdl.K_c {
					if !configStarted {
						configStarted = true
						url := fmt.Sprintf("http://localhost%s", options.HTTPPort)
						var err error

						switch runtime.GOOS {
						case "linux":
							err = exec.Command("xdg-open", url).Start()
						case "windows":
							err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
						case "darwin":
							err = exec.Command("open", url).Start()
						default:
							err = fmt.Errorf("unsupported platform")
						}
						if err != nil {
							log.Fatal(err)
						}
						configTimer.Reset(time.Second * 5)
					}
				}
			}
		case <-configTimer.C:
			configStarted = false
		case <-updateTicker.C:
			// Display update

			// startTime := time.Now()

			// Get the clock state snapshot
			state := engine.State()

			if state.ScreenFlash {
				drawWhiteScreen()
			} else {
				checkBackgroundUpdate(state)
				if options.textClock {
					drawTextClock(state)
				} else if options.Face == "288x144" {
					drawSmallTextClock(state)
				} else if options.countdown {
					drawCountdown()
				} else {
					drawRoundClocks(state)
				}
				for i := 0; i < numAudioSources; i++ {
					checkBeep(state, i)
					checkVoice(state, i)
					todBeep(state, i)
				}
			}

			if state.Info != "" {
				newInfo := fmt.Sprintf("%s\nKeyboard:\nF - Full screen\nEscape - Exit full screen\nC - Open web config", state.Info)
				if newInfo != info {
					info = newInfo

					updateInfoScreen(info)
				}
				drawInfoScreen()
			}

			updateCue(state)
			if state.CueShow {
				drawCue()
			}

			// Update the canvas
			renderer.Present()
			// debug.Printf("Frame time: %d ms\n", time.Now().Sub(startTime).Milliseconds())

			// Hardware signal color handling
			c := state.HardwareSignalColor
			if options.SignalFollow {
				// HW follows source 1
				c = state.Clocks[0].SignalColor
			}

			if signalHardware != nil {
				signalHardware.Fill(signalBrightness(c))
				signalHardware.Update()
			}

			if options.SignalToBG {
				colors.background = toSDLColor(c)
			}

			for _, m := range outputModules {
				m.Update(state)
			}
		}
	}
}

func updateInfoScreen(info string) {
	var err error
	if infoTexture != nil {
		infoTexture.Destroy()
	}
	lines := strings.Split(info, "\n")

	height := infoFont.LineSkip() * (len(lines) + 1)
	infoTexture, err = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, 1024, int32(height))
	infoTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	check(err)

	err = renderer.SetRenderTarget(infoTexture)
	check(err)
	renderer.SetDrawColor(0, 0, 0, 128)
	renderer.Clear()

	var rowTexture *sdl.Texture
	h := int32(0)
	for _, l := range lines {
		rowTexture = renderText(l, infoFont, sdl.Color{R: 255, G: 255, B: 255, A: 128})
		rowTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
		_, _, texW, texH, _ := rowTexture.Query()

		err := renderer.Copy(rowTexture, nil, &sdl.Rect{X: 20, Y: h, H: texH, W: texW})
		check(err)

		h += texH
		rowTexture.Destroy()
	}

	log.Printf("Updated info texture:\n%s", info)
}

func drawInfoScreen() {
	_, _, texW, texH, _ := infoTexture.Query()
	renderer.SetRenderTarget(nil)
	err := renderer.Copy(infoTexture, nil, &sdl.Rect{X: 0, Y: 0, H: texH, W: texW})
	check(err)
}

func checkBackgroundUpdate(state *clock.State) {
	// Check for background changes
	if backgroundNumber != state.Background {
		backgroundNumber = state.Background
		p := make([]string, 3)
		filemask := fmt.Sprintf("%d.*", state.Background)
		path := options.BackgroundPath
		p[0] = filepath.Join(path, filemask)
		p[1] = filepath.Join(path, "0"+filemask)
		p[2] = filepath.Join(path, "00"+filemask)
		for _, pattern := range p {
			files, _ := filepath.Glob(pattern)
			if files != nil {
				log.Printf("Loading background: %s", files[0])
				loadBackground(files[0])
				return
			}
		}
		log.Printf("Couldn't find background for number: %d", backgroundNumber)
		showBackground = false
	}
}

// parseOptions parses the command line options and provided ini file
func parseOptions() {
	options.Config = func(s string) error {
		ini := flags.NewIniParser(parser)
		options.configFile = s
		return ini.ParseFile(s)
	}

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			panic(err)
		}
	}

	// Load the platform specific default config files unless one is
	// given on command line
	if runtime.GOOS == "linux" && options.configFile == "" && !options.DumpConfig {
		linuxLoadConfig()
	}

	if runtime.GOOS == "windows" && options.configFile == "" && !options.DumpConfig {
		// Try to load a hardcoded default config file
		_, err := os.Stat("clock.ini")
		if err == nil {
			loadDefaultConfig("clock.ini")
		}
	}

	if runtime.GOOS == "darwin" && options.configFile == "" && !options.DumpConfig {
		// Running on osx and no config file provided
		// Assume we are running as an app

		// Set working directory to app resources
		darwinSetWD()

		// Try to load a hardcoded default config file
		darwinLoadConfig()

		env := os.Environ()
		log.Printf("Environment: \n%s", env)
	}

	computeDerivedOptions()
}

func computeDerivedOptions() {
	switch options.Face {
	case "max":
		options.textClock = true
	case "round":
	case "dual-round":
		options.dualClock = true
	case "small":
		// Legacy option
		options.small = true
		options.Face = "192"
	case "144":
		options.small = true
	case "192":
		options.small = true
	case "text":
		options.textClock = true
	case "single":
		options.textClock = true
		options.singleLine = true
	case "countdown":
		options.countdown = true
	}

	if options.Defaults {
		defaultSourceConfig()
	}

	if options.Debug {
		debug.Enabled = true
	}

	o := options.EngineOptions
	if len(o.SourceOptions) != 4 {
		o.SourceOptions = make([]*clock.SourceOptions, 4)
		o.SourceOptions[0] = o.Source1
		o.SourceOptions[1] = o.Source2
		o.SourceOptions[2] = o.Source3
		o.SourceOptions[3] = o.Source4
	}

	if len(o.TimerOptions) != 10 {
		o.TimerOptions = make([]*clock.TimerOptions, 10)
		o.TimerOptions[0] = o.Timer0
		o.TimerOptions[1] = o.Timer1
		o.TimerOptions[2] = o.Timer2
		o.TimerOptions[3] = o.Timer3
		o.TimerOptions[4] = o.Timer4
		o.TimerOptions[5] = o.Timer5
		o.TimerOptions[6] = o.Timer6
		o.TimerOptions[7] = o.Timer7
		o.TimerOptions[8] = o.Timer8
		o.TimerOptions[9] = o.Timer9
	}

	// Dump the current config to stdout
	if options.DumpConfig {
		dumpConfig()
	}
}

// dumpConfig dumps the clock configuration ini file to stdout
func dumpConfig() {
	tmpl, err := template.New("config.ini").Parse(configTemplate)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, options)
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}

// createRings creates the coordinate rings for the static and second rings
func createRings() {
	if options.Face == "144" {
		secCircles = util.Points(center144, secondRadius144, 60)
		staticCircles = util.Points(center144, staticRadius144, 12)
	} else if options.Face == "small" || options.Face == "192" {
		secCircles = util.Points(center192, secondRadius192, 60)
		staticCircles = util.Points(center192, staticRadius192, 12)
	} else {
		secCircles = util.Points(center1080, secondRadius1080, 60)
		staticCircles = util.Points(center1080, staticRadius1080, 12)
	}
}

func defaultSourceConfig() {
	options.EngineOptions.Source1 = &clock.SourceOptions{
		Text:          "",
		LTC:           true,
		Timer:         true,
		Counter:       1,
		Tod:           true,
		TimeZone:      "Europe/Helsinki",
		OvertimeColor: "#FF0000",
	}
	options.EngineOptions.Source2 = &clock.SourceOptions{
		Text:          "",
		LTC:           true,
		Timer:         true,
		Counter:       2,
		Tod:           true,
		TimeZone:      "Europe/Helsinki",
		OvertimeColor: "#FF0000",
	}
	options.EngineOptions.Source3 = &clock.SourceOptions{
		Text:          "",
		LTC:           true,
		Timer:         true,
		Counter:       3,
		Tod:           true,
		TimeZone:      "Europe/Helsinki",
		OvertimeColor: "#FF0000",
	}
	options.EngineOptions.Source4 = &clock.SourceOptions{
		Text:          "",
		LTC:           true,
		Timer:         true,
		Counter:       4,
		Tod:           true,
		TimeZone:      "Europe/Helsinki",
		OvertimeColor: "#FF0000",
	}
}

func signalBrightness(c color.RGBA) color.RGBA {
	return color.RGBA{
		R: uint8(int(c.R) * options.SignalBrightness / 255),
		G: uint8(int(c.G) * options.SignalBrightness / 255),
		B: uint8(int(c.B) * options.SignalBrightness / 255),
		A: 255,
	}
}

func copyAndLoadConfig(path string) {
	// User config location
	config := filepath.Join(path, "clock.ini")

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating config path: %v", err)
		}
	}

	// Create a log file
	f, err := os.OpenFile(filepath.Join(path, "clock.log"), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}

	log.SetOutput(f)

	_, err = os.Stat(config)
	if err != nil {
		log.Printf("User config not found, loading defaults: %v", err)
		src := "clock.ini"

		_, err = os.Stat(src)
		if err == nil {
			copyConfig(src, config)
		} else {
			log.Fatalf("Default configuration file clock.ini is missing: %v", err)
		}
	}
	loadDefaultConfig(config)

}

func linuxLoadConfig() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory: %v", err)
	}

	path := filepath.Join(homeDir, ".config", "clock-8001")

	copyAndLoadConfig(path)
}

func darwinSetWD() {
	app, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting application directory: %v", err)
	}

	appDir := filepath.Dir(app)
	appDir = filepath.Join(appDir, "..", "Resources")

	// If not running inside a bundle (and the ../Resources does not exist)
	// just silently give up
	if _, err := os.Stat(appDir); errors.Is(err, os.ErrNotExist) {
		return
	}

	err = os.Chdir(appDir)
	if err != nil {
		log.Printf("Error changing to the application directory: %v", err)
	}
}

func darwinLoadConfig() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory: %v", err)
	}

	path := filepath.Join(homeDir, "Library", "Application Support", "clock-8001")
	copyAndLoadConfig(path)
}

func loadDefaultConfig(config string) {
	log.Printf("Using default user config: %v", config)
	ini := flags.NewIniParser(parser)
	options.configFile = config
	err := ini.ParseFile(config)
	if err != nil {
		panic(err)
	}
}

// copyConfig copies a (default) config file to a destination
func copyConfig(src string, dst string) {
	in, err := os.Open(src)
	if err != nil {
		log.Fatalf("Error opening default config: %v", err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		log.Fatalf("Error creating user config file %s: %v", dst, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		log.Fatalf("Error copying default config to user config: %v", err)
	}
	err = out.Sync()
	if err != nil {
		log.Fatalf("Error syncing new config file: %v", err)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
