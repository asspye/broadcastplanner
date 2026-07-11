package domain

// ProjectFrameRate mirrors the Swift ProjectFrameRate enum. RawValue is the
// display/serialised string (e.g. "23.98"); FramesPerSecond is the real fps used
// to convert seconds<->frames; NominalFrameCount is the integer frame count used
// when rendering a timecode (24/25/30/60). This split matters for drop-frame
// rates like 23.98/29.97/59.94.
type ProjectFrameRate string

const (
	FPS2398 ProjectFrameRate = "23.98"
	FPS24   ProjectFrameRate = "24"
	FPS25   ProjectFrameRate = "25"
	FPS2997 ProjectFrameRate = "29.97"
	FPS30   ProjectFrameRate = "30"
	FPS5994 ProjectFrameRate = "59.94"
	FPS60   ProjectFrameRate = "60"
)

// AllFrameRates lists every supported rate in UI order.
var AllFrameRates = []ProjectFrameRate{FPS2398, FPS24, FPS25, FPS2997, FPS30, FPS5994, FPS60}

// FramesPerSecond returns the real (fractional) frame rate.
func (r ProjectFrameRate) FramesPerSecond() float64 {
	switch r {
	case FPS2398:
		return 23.976
	case FPS24:
		return 24
	case FPS25:
		return 25
	case FPS2997:
		return 29.97
	case FPS30:
		return 30
	case FPS5994:
		return 59.94
	case FPS60:
		return 60
	default:
		return 25
	}
}

// NominalFrameCount returns the integer number of frames per second used for TC.
func (r ProjectFrameRate) NominalFrameCount() int64 {
	switch r {
	case FPS2398, FPS24:
		return 24
	case FPS25:
		return 25
	case FPS2997, FPS30:
		return 30
	case FPS5994, FPS60:
		return 60
	default:
		return 25
	}
}

// FrameRateFromRaw resolves a raw string to a ProjectFrameRate, defaulting to 25.
func FrameRateFromRaw(raw string) ProjectFrameRate {
	for _, r := range AllFrameRates {
		if string(r) == raw {
			return r
		}
	}
	return FPS25
}
