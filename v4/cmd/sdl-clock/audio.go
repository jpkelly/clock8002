package main

import (
	_ "embed"
	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
	"gitlab.com/clock-8001/clock-8001/v4/clock"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

//go:embed 1kHz_100ms.wav
var shortWav []byte

//go:embed 1kHz_500ms.wav
var longWav []byte

var shortBeep *mix.Chunk
var longBeep *mix.Chunk

var numAudioSources int
var lastBeep []int
var lastVoice []int

var clips map[int]*mix.Chunk

func initAudio() {
	if !options.AudioEnabled && !options.TODBeep && !options.VoiceEnabled {
		return
	}
	var err error

	if err = mix.OpenAudio(44100, mix.DEFAULT_FORMAT, 2, 4096); err != nil {
		panic(err)
	}

	shortBeep, err = mix.QuickLoadWAV(shortWav)
	if err != nil {
		panic(err)
	}

	longBeep, err = mix.QuickLoadWAV(longWav)
	if err != nil {
		panic(err)
	}
	lastBeep = make([]int, 4)

	if options.VoiceEnabled {
		log.Printf("Loading voice files from dir: %v", options.VoiceDir)
		loadClips(options.VoiceDir)
		lastVoice = make([]int, 4)
	}
}

func loadClips(dir string) {
	filter := regexp.MustCompile(`^\d+\.wav$`)
	clips = make(map[int]*mix.Chunk)

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if !file.IsDir() && filter.MatchString(file.Name()) {
			secs, err := strconv.Atoi(strings.TrimSuffix(file.Name(), ".wav"))
			if err != nil {
				log.Printf("Error parsing file name: %v %v", file.Name(), err)
				continue
			}
			f := filepath.Join(dir, file.Name())
			chunk, err := mix.LoadWAV(f)
			if err != nil {
				log.Printf("Error loading wav: %v %v", f, err)
				continue
			}
			clips[secs] = chunk
			log.Printf("Added sample for %d seconds", secs)
		}
	}
}

func checkBeep(s *clock.State, i int) {
	if !options.AudioEnabled {
		return
	}
	clk := s.Clocks[i]
	if clk.Mode == clock.Countdown {
		if clk.Hours == 0 && clk.Minutes == 0 {
			if clk.Seconds <= 5 && lastBeep[i] > clk.Seconds {
				if clk.Seconds == 0 {
					longBeep.Play(-1, 0)
				} else {
					shortBeep.Play(-1, 0)
				}
			}
			lastBeep[i] = clk.Seconds
		}
	}
}

func checkVoice(s *clock.State, i int) {
	if !options.VoiceEnabled {
		return
	}
	clk := s.Clocks[i]
	if clk.Mode == clock.Countdown || clk.Mode == clock.Media {
		secs := (clk.Hours * 3600) + (clk.Minutes * 60) + clk.Seconds
		if clk.Expired && secs != 0 {
			return
		}
		if lastVoice[i] > secs {
			// Seconds has been lowered
			if chunk, ok := clips[secs]; ok {
				chunk.Play(-1, 0)
			}
		}
		lastVoice[i] = secs
	}
}

func todBeep(s *clock.State, i int) {
	if !options.TODBeep {
		return
	}
	clk := s.Clocks[i]

	if clk.Mode == clock.Normal {
		if clk.Minutes == 59 {
			if clk.Seconds >= 55 && lastBeep[i] < clk.Seconds {
				shortBeep.Play(-1, 0)
			}
		} else if clk.Minutes == 0 && clk.Seconds == 00 && lastBeep[i] == 59 {
			longBeep.Play(-1, 0)
		}
		lastBeep[i] = clk.Seconds
	}
}

func play() {

	defer mix.CloseAudio()

	// Play 4 times
	shortBeep.Play(1, 0)
	sdl.Delay(900)
	shortBeep.Play(1, 0)
	sdl.Delay(900)
	shortBeep.Play(1, 0)
	sdl.Delay(900)
	shortBeep.Play(1, 0)
	sdl.Delay(900)
	longBeep.Play(1, 0)
	sdl.Delay(500)

	// Wait until it finishes playing
	for mix.Playing(-1) == 1 {
		sdl.Delay(16)
	}
}
