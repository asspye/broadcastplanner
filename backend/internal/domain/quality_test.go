package domain

import "testing"

func withFPS(a MediaAsset, fps float64) MediaAsset { f := fps; a.FrameRate = &f; return a }
func ready(a MediaAsset) MediaAsset                { a.Status = "Готово"; return a }

// Ported from playlistSegmentLabelUsesSourceMarkersAcrossSplitRows.
func TestSegmentLabelAcrossSplitRows(t *testing.T) {
	a := ready(mediaAsset("movie", 100))
	markers := []AdMarker{
		{Kind: MarkerAdBreak, Time: 20}, {Kind: MarkerAdBreak, Time: 40}, {Kind: MarkerAdBreak, Time: 70},
	}
	item := NewPlaylistItem("", a, 120, 40, 70)
	segs := Segments(item, markers)
	if got := SegmentLabel(item, markers, &segs[0]); got != "3 из 4" {
		t.Errorf("label(seg) = %q, want 3 из 4", got)
	}
	if got := SegmentLabel(item, markers, nil); got != "3 из 4" {
		t.Errorf("label(item) = %q, want 3 из 4", got)
	}
}

// Ported from playlistQualityReportFindsWarnings (FPS + short segment warnings).
func TestQualityFindsWarnings(t *testing.T) {
	a := withFPS(ready(mediaAsset("clip", 100)), 29.97)
	exists := func(string) bool { return true }
	pl := []PlaylistItem{
		NewPlaylistItem("", a, 0, 0, 4), // short (<5s)
		NewPlaylistItem("", a, 0, 4, 14),
	}
	rep := BuildQualityReport(pl, nil, FPS25, exists)

	if rep.ItemCount != 2 || rep.SegmentCount != 2 {
		t.Fatalf("counts item=%d seg=%d", rep.ItemCount, rep.SegmentCount)
	}
	if rep.IssueCountBySeverity[SeverityWarning] < 2 {
		t.Errorf("want >=2 warnings, got %d", rep.IssueCountBySeverity[SeverityWarning])
	}
	if !hasCode(rep, CodeFPS) || !hasCode(rep, CodeShort) {
		t.Error("missing FPS or SHORT")
	}
	if hasCode(rep, CodeDuplicate) {
		t.Error("unexpected duplicate (different source ranges)")
	}
}

// Ported from playlistQualityIssuesReturnRowBadges (missing file + fps + ad on a
// whole 700s video with no markers).
func TestQualityRowBadges(t *testing.T) {
	a := withFPS(ready(mediaAsset("badge", 700)), 29.97)
	a.Path = "/tmp/missing-badge.mp4"
	item := NewPlaylistItem("", a, 0, 0, 700)
	rep := BuildQualityReport([]PlaylistItem{item}, nil, FPS25, func(string) bool { return false })
	seg := Segments(item, nil)[0]
	codes := map[IssueCode]bool{}
	for _, iss := range rep.IssuesFor(item.ID, seg.ID) {
		codes[iss.Code] = true
	}
	for _, want := range []IssueCode{CodeMedia, CodeFPS, CodeAd} {
		if !codes[want] {
			t.Errorf("missing code %s", want)
		}
	}
}

// Ported from playlistQualityDoesNotRequireAdForWholeVideoUnderTenMinutes.
func TestQualityNoAdUnderTenMinutes(t *testing.T) {
	a := ready(mediaAsset("short", 300))
	item := NewPlaylistItem("", a, 0, 0, 300)
	rep := BuildQualityReport([]PlaylistItem{item}, nil, FPS25, func(string) bool { return true })
	if hasCode(rep, CodeAd) {
		t.Error("should not require AD for <10min whole video")
	}
}

// Ported from playlistQualityReportsRangeOutsideRelinkedFileDuration.
func TestQualityRangeOutsideFile(t *testing.T) {
	a := ready(mediaAsset("shortfile", 30))
	item := NewPlaylistItem("", a, 0, 0, 45) // out beyond 30s file
	rep := BuildQualityReport([]PlaylistItem{item}, nil, FPS25, func(string) bool { return true })
	if !hasSeverityCode(rep, SeverityError, CodeRange) {
		t.Error("expected RANGE error")
	}
}

// Ported from playlistQualityReportFindsDuplicateSourceRangesOnly.
func TestQualityDuplicateRanges(t *testing.T) {
	a := ready(mediaAsset("clip", 100))
	pl := []PlaylistItem{
		NewPlaylistItem("", a, 0, 20, 30),
		NewPlaylistItem("", a, 10, 20, 30),
	}
	rep := BuildQualityReport(pl, nil, FPS25, func(string) bool { return true })
	if !hasCode(rep, CodeDuplicate) {
		t.Error("expected duplicate range issue")
	}
}

func hasCode(r QualityReport, c IssueCode) bool {
	for _, i := range r.Issues {
		if i.Code == c {
			return true
		}
	}
	return false
}
func hasSeverityCode(r QualityReport, s CheckSeverity, c IssueCode) bool {
	for _, i := range r.Issues {
		if i.Code == c && i.Severity == s {
			return true
		}
	}
	return false
}
