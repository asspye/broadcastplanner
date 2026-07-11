package domain

import (
	"fmt"
	"math"
	"sort"
)

// SegmentLabel renders "N of M" ("N из M") for a segment (or the item as a whole
// when seg is nil), computed from the asset's AD BREAK markers across the whole
// file — so split rows keep their original position label. Ported 1:1 from Swift
// segmentLabel(for:segment:).
func SegmentLabel(item PlaylistItem, markers []AdMarker, seg *PlaylistSegment) string {
	if item.IsNonTimingRow() {
		return "1 из 1"
	}

	fallbackIndex := 1
	if seg != nil {
		fallbackIndex = seg.SegmentIndex
	}
	fallbackTotal := len(Segments(item, markers))
	if fallbackTotal < 1 {
		fallbackTotal = 1
	}

	if item.Asset.Duration == nil || *item.Asset.Duration <= 0 {
		return fmt.Sprintf("%d из %d", fallbackIndex, fallbackTotal)
	}
	assetDuration := *item.Asset.Duration

	var cuts []float64
	for _, m := range markers {
		if m.Kind != MarkerAdBreak || m.Time <= 0.01 || m.Time >= assetDuration-0.01 {
			continue
		}
		dup := false
		for _, c := range cuts {
			if math.Abs(c-m.Time) < 0.01 {
				dup = true
				break
			}
		}
		if !dup {
			cuts = append(cuts, m.Time)
		}
	}
	sort.Float64s(cuts)

	boundaries := append([]float64{0}, cuts...)
	boundaries = append(boundaries, assetDuration)
	sort.Float64s(boundaries)

	total := len(boundaries) - 1
	if total < 1 {
		total = 1
	}

	sourceIn := item.SourceIn
	sourceOut := item.SourceOut
	if seg != nil {
		sourceIn = seg.SourceIn
		sourceOut = seg.SourceOut
	}
	const tol = 0.05

	for i := 0; i < total; i++ {
		if math.Abs(sourceIn-boundaries[i]) <= tol && math.Abs(sourceOut-boundaries[i+1]) <= tol {
			return fmt.Sprintf("%d из %d", i+1, total)
		}
	}

	mid := (sourceIn + sourceOut) / 2
	for i := 0; i < total; i++ {
		if mid >= boundaries[i]-tol && mid <= boundaries[i+1]+tol {
			return fmt.Sprintf("%d из %d", i+1, total)
		}
	}

	return fmt.Sprintf("%d из %d", fallbackIndex, fallbackTotal)
}
