package util

import (
	"fmt"
	"os"
	"regexp"
	"time"
)

// ValidateFile checks that a given file exists
func ValidateFile(filename, title string) (msg string) {
	if !FileExists(filename) {
		msg = fmt.Sprintf("<li>%s: file does not exist (%s)</li>", title, filename)
	}
	return
}

// ValidateNumber checks for valid numer
func ValidateNumber(err error, title string) (msg string) {
	if err != nil {
		msg = fmt.Sprintf("<li>%s: error parsing number</li>", title)
	}
	return
}

// ValidateTimer check for valid timer number
func ValidateTimer(timer int, title string) (msg string) {
	if timer < 0 || timer > 9 {
		msg = fmt.Sprintf("<li>%s: timer number not in range 0-9 (%d)</li>", title, timer)
	}
	return
}

// ValidateColor checks for valid CSS color
func ValidateColor(color string, title string) (msg string) {
	match, err := regexp.MatchString(`^#([0-9a-fA-F]{3}){1,2}$`, color)

	if !match {
		msg = fmt.Sprintf("<li>%s: incorrect format for a color (%s)</li>", title, color)
	} else if err != nil {
		msg = fmt.Sprintf("<li>%s: %v</li>", title, err)
	}
	return
}

// ValidateAddr checks for valid address and port combination
func ValidateAddr(addr, title string) (msg string) {
	match, err := regexp.MatchString(`^.*:\d*$`, addr)

	if !match {
		msg = fmt.Sprintf("<li>%s: address not formatted correctly (%s)</li>", title, addr)
	} else if err != nil {
		msg = fmt.Sprintf("<li>%s: %v</li>", title, err)
	}
	return
}

// ValidateTZ checks for valid timezone
func ValidateTZ(zone, title string) (msg string) {
	_, err := time.LoadLocation(zone)
	if err != nil {
		msg = fmt.Sprintf("<li>%s: %v</li>", title, err)
	}
	return
}

// FileExists checks if a file exists and is not a directory
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
