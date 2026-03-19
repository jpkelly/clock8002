package main

import (
	"fmt"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var backgrounds []map[int]*sdl.Texture
var lastBG []int

var defaultBackground *sdl.Texture
var timerBG []*sdl.Texture
var lastTimer int

func checkDynamicBG(s *clock.State) {
	if !options.DynamicBG {
		return
	}

	for _, clk := range s.Clocks {

		if clk.ActiveTimer == -1 {
			lastTimer = -1
			setBackground(defaultBackground)
			continue
		}

		if clk.Mode == clock.Countdown || clk.Mode == clock.Media {

			// Set the default BG for timer on timer change
			if lastTimer != clk.ActiveTimer {
				if timerBG[clk.ActiveTimer] != nil {
					log.Printf("Set default timer BG")
					setBackground(timerBG[clk.ActiveTimer])
				}
			}
			lastTimer = clk.ActiveTimer

			// Check for time threshold BG
			secs := (clk.Hours * 3600) + (clk.Minutes * 60) + clk.Seconds
			if clk.Expired && secs != 0 {
				return
			}
			if lastBG[clk.ActiveTimer] > secs {
				// Seconds has been lowered, check for BG change
				if t, ok := backgrounds[clk.ActiveTimer][secs]; ok {
					log.Printf("Set timed BG")
					setBackground(t)
				}
			}
			lastBG[clk.ActiveTimer] = secs
			return
		}
	}
	setBackground(defaultBackground)
}

func destroyBackgrounds() {
	if defaultBackground != nil {
		defaultBackground.Destroy()
		defaultBackground = nil
	}
	backgroundTexture = nil
	for i, t := range timerBG {
		if t != nil {
			t.Destroy()
			timerBG[i] = nil
		}
	}
	for _, m := range backgrounds {
		for _, t := range m {
			if t != nil {
				t.Destroy()
			}
		}
	}
}

func initBackgrounds() {
	destroyBackgrounds()
	lastBG = make([]int, clock.NumCounters)
	backgrounds = make([]map[int]*sdl.Texture, clock.NumCounters)
	timerBG = make([]*sdl.Texture, clock.NumCounters)
	lastTimer = -1

	log.Printf("Loading dynamic timer backgrounds...")
	for i := 0; i < clock.NumCounters; i++ {
		log.Printf("-> Timer %d", i)
		timer := fmt.Sprintf("timer%d", i)
		loadBackgrounds(filepath.Join(options.BackgroundPath, timer), i)
	}
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

// loadBackground loads and processes the background image into sdl.Texture
func loadBackground(file string) {
	t, err := loadImage(file)

	if err != nil {
		// Failed to load background image, continue without it
		log.Printf("Error loading background image: %v %v\n", options.Background, err)
		log.Printf("Disabling background image.")
		if defaultBackground != nil {
			defaultBackground.Destroy()
			defaultBackground = nil
		}
		showBackground = false
		return
	}
	if defaultBackground != nil {
		defaultBackground.Destroy()
	}
	defaultBackground = t
	setBackground(defaultBackground)
}

func setBackground(t *sdl.Texture) {
	if t != nil {
		backgroundTexture = t
		showBackground = true
	} else {
		showBackground = false
	}
}

func loadImage(file string) (*sdl.Texture, error) {
	image, err := img.Load(file)
	if err != nil {
		return nil, err
	}

	texture, err := renderer.CreateTextureFromSurface(image)
	image.Free()
	if err != nil {
		texture.Destroy()
		return nil, err
	}

	err = texture.SetBlendMode(sdl.BLENDMODE_NONE)
	if err != nil {
		texture.Destroy()
		return nil, err
	}

	return texture, nil
}

func loadBackgrounds(dir string, timer int) {
	filter := regexp.MustCompile(`^\d+\..+$`)
	def := regexp.MustCompile(`^default\..+$`)
	backgrounds[timer] = make(map[int]*sdl.Texture)

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("ERROR: Timer %d backgrounds: %v", timer, err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if def.MatchString(file.Name()) {
			f := filepath.Join(dir, file.Name())
			t, err := loadImage(f)
			if err != nil {
				log.Printf("Error loading BG: %v %v", f, err)
				continue
			}
			timerBG[timer] = t
			log.Printf("Added Default BG")
		} else if filter.MatchString(file.Name()) {
			secs, err := strconv.Atoi(strings.Split(file.Name(), ".")[0])
			if err != nil {
				log.Printf("Error parsing file name: %v %v", file.Name(), err)
				continue
			}
			f := filepath.Join(dir, file.Name())
			t, err := loadImage(f)
			if err != nil {
				log.Printf("Error loading BG: %v %v", f, err)
				continue
			}
			backgrounds[timer][secs] = t
			log.Printf("Added BG for %d seconds", secs)
		}
	}
}
