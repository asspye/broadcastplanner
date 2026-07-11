package domain

import "testing"

// Golden vectors ported from BroadcastPlanner Tests/TVAssemblyTests/TimecodeTests.swift.

func TestClock(t *testing.T) {
	cases := map[float64]string{
		0:    "00:00:00",
		65:   "00:01:05",
		3661: "01:01:01",
	}
	for in, want := range cases {
		if got := Clock(in); got != want {
			t.Errorf("Clock(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestTimecode(t *testing.T) {
	type tc struct {
		in   float64
		fr   ProjectFrameRate
		want string
	}
	cases := []tc{
		{0, FPS25, "00:00:00:00"},
		{1.04, FPS25, "00:00:01:01"},
		{1.5, FPS30, "00:00:01:15"},
	}
	for _, c := range cases {
		if got := Timecode(c.in, c.fr); got != c.want {
			t.Errorf("Timecode(%v,%s) = %q, want %q", c.in, c.fr, got, c.want)
		}
	}
}

// From playlistClockStartWrapsAfterTwentyFourHours: 24h+5s renders unwrapped in
// Timecode but wraps in BroadcastClockTimecode.
func TestTimecodeDoesNotWrapButBroadcastDoes(t *testing.T) {
	offset := float64(24*60*60 + 5)
	if got := Timecode(offset, FPS25); got != "24:00:05:00" {
		t.Errorf("Timecode wrap = %q, want 24:00:05:00", got)
	}
	if got := BroadcastClockTimecode(offset, FPS25); got != "00:00:05:00" {
		t.Errorf("BroadcastClockTimecode wrap = %q, want 00:00:05:00", got)
	}
}

func TestBroadcastClockStart(t *testing.T) {
	// 6h start renders as 06:00:00:00
	if got := BroadcastClockTimecode(6*60*60, FPS25); got != "06:00:00:00" {
		t.Errorf("BroadcastClockTimecode(6h) = %q, want 06:00:00:00", got)
	}
}
