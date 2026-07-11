package domain

import "testing"

func mediaAsset(name string, dur float64) MediaAsset {
	d := dur
	return MediaAsset{ID: NewID(), Name: name, Path: "/tmp/" + name + ".mp4", Kind: KindVideo, Duration: &d, FileExtension: "MP4"}
}

// Ported from playlistClockStartAppliesToRowsAndExport.
func TestRecalculateAppliesClockStart(t *testing.T) {
	pl := []PlaylistItem{
		NewPlaylistItem("", mediaAsset("first", 10), 0, 0, 10),
		NewPlaylistItem("", mediaAsset("second", 20), 0, 0, 20),
	}
	out := Recalculate(pl, 6*60*60, nil)
	if len(out) != 2 {
		t.Fatalf("len=%d, want 2", len(out))
	}
	got := []string{
		BroadcastClockTimecode(out[0].StartOffset, FPS25),
		BroadcastClockTimecode(out[1].StartOffset, FPS25),
	}
	want := []string{"06:00:00:00", "06:00:10:00"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d start = %q, want %q", i, got[i], want[i])
		}
	}
}

// Ported from playlistClockStartWrapsAfterTwentyFourHours.
func TestRecalculateWrapsAfter24h(t *testing.T) {
	pl := []PlaylistItem{
		NewPlaylistItem("", mediaAsset("first", 24*60*60+5), 0, 0, 24*60*60+5),
		NewPlaylistItem("", mediaAsset("second", 10), 0, 0, 10),
	}
	out := Recalculate(pl, 0, nil)
	// second row starts at 24:00:05 sequentially, wraps to 00:00:05 on the clock.
	second := out[1]
	if got := Timecode(second.StartOffset, FPS25); got != "24:00:05:00" {
		t.Errorf("sequential = %q, want 24:00:05:00", got)
	}
	if got := BroadcastClockTimecode(second.StartOffset, FPS25); got != "00:00:05:00" {
		t.Errorf("broadcast = %q, want 00:00:05:00", got)
	}
}

// With an air date set, a leading date comment is injected and a new one appears
// each time the clock crosses a 24h boundary.
func TestRecalculateInjectsAirDateBoundaries(t *testing.T) {
	airDate := BroadcastAirDate{Day: 1, Month: 6, Year: 2026}
	pl := []PlaylistItem{
		NewPlaylistItem("", mediaAsset("a", 12*60*60), 0, 0, 12*60*60),
		NewPlaylistItem("", mediaAsset("b", 18*60*60), 0, 0, 18*60*60), // crosses into day 2
		NewPlaylistItem("", mediaAsset("c", 60), 0, 0, 60),
	}
	out := Recalculate(pl, 0, &airDate)

	var comments []string
	for _, r := range out {
		if r.IsCommentRow() && IsAirDateComment(r.CommentText) {
			comments = append(comments, r.CommentText)
		}
	}
	want := []string{"01 06 2026", "02 06 2026"}
	if len(comments) != len(want) {
		t.Fatalf("air-date comments = %v, want %v", comments, want)
	}
	for i := range want {
		if comments[i] != want[i] {
			t.Errorf("comment %d = %q, want %q", i, comments[i], want[i])
		}
	}
}

func TestAirDateHelpers(t *testing.T) {
	d := NormalizedAirDate(31, 13, 2026) // month clamps to 12, day clamps to 31
	if d.DisplayText() != "31 12 2026" {
		t.Errorf("normalized = %q, want 31 12 2026", d.DisplayText())
	}
	if got := (BroadcastAirDate{Day: 30, Month: 6, Year: 2026}).AddingDays(1); got.DisplayText() != "01 07 2026" {
		t.Errorf("addingDays = %q, want 01 07 2026", got.DisplayText())
	}
	if !IsAirDateComment("01 06 2026") || IsAirDateComment("Комментарий") {
		t.Error("IsAirDateComment mismatch")
	}
}
