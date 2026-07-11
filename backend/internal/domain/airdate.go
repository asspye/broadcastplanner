package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BroadcastAirDate mirrors the Swift BroadcastAirDate. DisplayText is "DD MM YYYY"
// (e.g. "01 06 2026") — the exact format written into air-date comment rows.
type BroadcastAirDate struct {
	Day   int `json:"day"`
	Month int `json:"month"`
	Year  int `json:"year"`
}

// DisplayText renders "DD MM YYYY".
func (d BroadcastAirDate) DisplayText() string {
	return fmt.Sprintf("%02d %02d %04d", d.Day, d.Month, d.Year)
}

var airDateCommentRe = regexp.MustCompile(`^\d{2}\s\d{2}\s\d{4}$`)

// IsAirDateComment reports whether a comment string is an air-date marker.
func IsAirDateComment(text string) bool {
	return airDateCommentRe.MatchString(text)
}

// DaysInMonth returns the number of days in a month, clamping invalid input.
func DaysInMonth(month, year int) int {
	if month < 1 || month > 12 {
		return 31
	}
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return first.AddDate(0, 1, -1).Day()
}

// NormalizedAirDate clamps day/month/year into valid ranges (mirrors Swift).
func NormalizedAirDate(day, month, year int) BroadcastAirDate {
	y := clamp(year, 1900, 2999)
	m := clamp(month, 1, 12)
	d := clamp(day, 1, DaysInMonth(m, y))
	return BroadcastAirDate{Day: d, Month: m, Year: y}
}

// AddingDays returns the air date shifted by n days (Gregorian).
func (d BroadcastAirDate) AddingDays(n int) BroadcastAirDate {
	t := time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, n)
	return BroadcastAirDate{Day: t.Day(), Month: int(t.Month()), Year: t.Year()}
}

// AirDateFromComment parses "DD MM YYYY" back into a BroadcastAirDate.
func AirDateFromComment(text string) (BroadcastAirDate, bool) {
	if !IsAirDateComment(text) {
		return BroadcastAirDate{}, false
	}
	parts := strings.Fields(text)
	if len(parts) != 3 {
		return BroadcastAirDate{}, false
	}
	d, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	y, _ := strconv.Atoi(parts[2])
	return NormalizedAirDate(d, m, y), true
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
