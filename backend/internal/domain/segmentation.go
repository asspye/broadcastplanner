package domain

import (
	"fmt"
	"math"
	"sort"
)

// Segments slices a PlaylistItem into segments at AD BREAK markers that fall
// strictly inside (SourceIn, SourceOut). Ported 1:1 from Swift
// PlaylistSegmentation.segments. Cut points are de-duplicated with a 0.01s
// tolerance and sorted ascending.
func Segments(item PlaylistItem, markers []AdMarker) []PlaylistSegment {
	if item.IsNonTimingRow() {
		return nil
	}

	var cutPoints []float64
	for _, m := range markers {
		if m.Kind != MarkerAdBreak {
			continue
		}
		if m.Time <= item.SourceIn || m.Time >= item.SourceOut {
			continue
		}
		dup := false
		for _, existing := range cutPoints {
			if math.Abs(existing-m.Time) < 0.01 {
				dup = true
				break
			}
		}
		if !dup {
			cutPoints = append(cutPoints, m.Time)
		}
	}
	sort.Float64s(cutPoints)

	boundaries := append(cutPoints, item.SourceOut)
	segments := make([]PlaylistSegment, 0, len(boundaries))
	sourceIn := item.SourceIn
	for _, sourceOut := range boundaries {
		index := len(segments) + 1
		lead := sourceIn - item.SourceIn
		if lead < 0 {
			lead = 0
		}
		segStart := item.StartOffset + lead
		segLen := sourceOut - sourceIn
		if segLen < 0 {
			segLen = 0
		}
		segments = append(segments, PlaylistSegment{
			ID:           fmt.Sprintf("%s-%d", item.ID, index),
			ParentItemID: item.ID,
			SegmentIndex: index,
			Asset:        item.Asset,
			StartOffset:  segStart,
			EndOffset:    segStart + segLen,
			SourceIn:     sourceIn,
			SourceOut:    sourceOut,
		})
		sourceIn = sourceOut
	}
	return segments
}
