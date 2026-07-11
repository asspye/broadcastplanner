package domain

import (
	"fmt"
	"math"
)

// Timecode formatters ported 1:1 from Swift `BroadcastFormatters`.
// TimeInterval (seconds) is represented as float64 throughout.

const secondsPerDay = 24 * 60 * 60

// Clock renders whole seconds as "HH:MM:SS" (used for marker positions).
func Clock(interval float64) string {
	totalFrames := int64(math.Floor(interval))
	if totalFrames < 0 {
		totalFrames = 0
	}
	hours := totalFrames / 3600
	minutes := (totalFrames % 3600) / 60
	seconds := totalFrames % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// Decimal renders an optional float as "%.2f" or "-" when nil.
func Decimal(value *float64) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f", *value)
}

// Timecode renders "HH:MM:SS:FF" for a duration. Hours are NOT wrapped, so a
// value past 24h renders as "24:00:05:00" (matches the Swift semantics).
func Timecode(interval float64, fr ProjectFrameRate) string {
	fps := fr.FramesPerSecond()
	nominal := fr.NominalFrameCount()
	totalFrames := int64(math.Round(interval * fps))
	if totalFrames < 0 {
		totalFrames = 0
	}
	frames := totalFrames % nominal
	totalSeconds := totalFrames / nominal
	seconds := totalSeconds % 60
	minutes := (totalSeconds / 60) % 60
	hours := totalSeconds / 3600
	return fmt.Sprintf("%02d:%02d:%02d:%02d", hours, minutes, seconds, frames)
}

// BroadcastClockTimecode renders "HH:MM:SS:FF" wrapped into a 24h broadcast day.
func BroadcastClockTimecode(interval float64, fr ProjectFrameRate) string {
	fps := fr.FramesPerSecond()
	nominal := fr.NominalFrameCount()
	framesPerDay := int64(secondsPerDay) * nominal
	rawFrames := int64(math.Round(interval * fps))
	totalFrames := ((rawFrames % framesPerDay) + framesPerDay) % framesPerDay
	frames := totalFrames % nominal
	totalSeconds := totalFrames / nominal
	seconds := totalSeconds % 60
	minutes := (totalSeconds / 60) % 60
	hours := (totalSeconds / 3600) % 24
	return fmt.Sprintf("%02d:%02d:%02d:%02d", hours, minutes, seconds, frames)
}
