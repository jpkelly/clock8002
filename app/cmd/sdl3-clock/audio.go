package main

import (
	_ "embed"
	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/jpkelly/clock8002/app/clock"
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

var shortBeep []byte
var longBeep []byte

var numAudioSources int
var lastBeep []int
var lastVoice []int

var beepStream *sdl.AudioStream
var voiceStream *sdl.AudioStream

var clips map[int][]byte

func initAudio() {
	if !options.AudioEnabled && !options.TODBeep && !options.VoiceEnabled {
		return
	}
	var err error
	var spec sdl.AudioSpec

	// if err = mixer.OpenAudio(44100, mixer.DEFAULT_FORMAT, 2, 4096); err != nil {
	//	panic(err)
	// }

	shortIO, err := sdl.IOFromBytes(shortWav)
	if err != nil {
		panic(err)
	}
	longIO, err := sdl.IOFromBytes(longWav)
	if err != nil {
		panic(err)
	}

	shortBeep, err = sdl.LoadWAV_IO(shortIO, true, &spec)
	if err != nil {
		panic(err)
	}

	longBeep, err = sdl.LoadWAV_IO(longIO, true, &spec)
	if err != nil {
		panic(err)
	}

	if beepStream != nil {
		beepStream.Destroy()
	}
	beepStream = sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK.OpenAudioDeviceStream(&spec, 0)

	beepStream.ResumeDevice()

	lastBeep = make([]int, 4)

	if options.VoiceEnabled {
		log.Printf("Loading voice files from dir: %v", options.VoiceDir)
		loadClips(options.VoiceDir)
		lastVoice = make([]int, 4)
	}
}

func loadClips(dir string) {
	var spec sdl.AudioSpec
	filter := regexp.MustCompile(`^\d+\.wav$`)
	clips = make(map[int][]byte)

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
			chunk, err := sdl.LoadWAV(f, &spec)
			if err != nil {
				log.Printf("Error loading wav: %v %v", f, err)
				continue
			}
			clips[secs] = chunk
			log.Printf("Added sample for %d seconds", secs)
		}
	}

	if voiceStream != nil {
		voiceStream.Destroy()
	}

	voiceStream = sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK.OpenAudioDeviceStream(&spec, 0)
	voiceStream.ResumeDevice()
}

func checkBeep(s *clock.State, i int) {
	if !options.AudioEnabled {
		return
	}
	clk := s.Clocks[i]
	if clk.Mute {
		return
	}
	if clk.Mode == clock.Countdown {
		if clk.Hours == 0 && clk.Minutes == 0 {
			if clk.Seconds <= 5 && lastBeep[i] > clk.Seconds {
				if clk.Seconds == 0 {
					beepStream.PutData(longBeep)
				} else {
					beepStream.PutData(shortBeep)
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
	if clk.Mute {
		return
	}
	if clk.Mode == clock.Countdown || clk.Mode == clock.Media {
		secs := (clk.Hours * 3600) + (clk.Minutes * 60) + clk.Seconds
		if clk.Expired && secs != 0 {
			return
		}
		if lastVoice[i] > secs || lastVoice[i] < secs {
			// Seconds has been lowered
			if chunk, ok := clips[secs]; ok {
				voiceStream.PutData(chunk)
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
				beepStream.PutData(shortBeep)
			}
		} else if clk.Minutes == 0 && clk.Seconds == 00 && lastBeep[i] == 59 {
			beepStream.PutData(longBeep)
		}
		lastBeep[i] = clk.Seconds
	}
}
