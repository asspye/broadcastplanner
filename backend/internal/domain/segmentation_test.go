package domain

import "testing"

// Ported from segmentationRespectsSourceRange in TimecodeTests.swift.
func TestSegmentationRespectsSourceRange(t *testing.T) {
	dur := 100.0
	asset := MediaAsset{ID: "a1", Name: "clip", Path: "/tmp/clip.mp4", Kind: KindVideo, Duration: &dur, FileExtension: "MP4"}
	item := NewPlaylistItem("i1", asset, 30, 20, 60) // startOffset 30, sourceIn 20, sourceOut 60
	markers := []AdMarker{
		{Kind: MarkerAdBreak, Time: 10},
		{Kind: MarkerAdBreak, Time: 35},
		{Kind: MarkerAdBreak, Time: 80},
	}

	segs := Segments(item, markers)

	if len(segs) != 2 {
		t.Fatalf("segment count = %d, want 2", len(segs))
	}
	checks := []struct {
		got, want float64
		name      string
	}{
		{segs[0].StartOffset, 30, "seg0.start"},
		{segs[0].SourceIn, 20, "seg0.in"},
		{segs[0].SourceOut, 35, "seg0.out"},
		{segs[1].StartOffset, 45, "seg1.start"},
		{segs[1].SourceIn, 35, "seg1.in"},
		{segs[1].SourceOut, 60, "seg1.out"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}
