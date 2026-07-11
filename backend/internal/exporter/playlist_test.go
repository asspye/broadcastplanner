package exporter

import (
	"strings"
	"testing"

	"github.com/broadcastplanner/backend/internal/domain"
)

func dur(v float64) *float64 { return &v }

// whole-file media row: sourceIn 0, sourceOut = duration.
func wholeRow(a domain.MediaAsset, startOffset, duration float64) domain.PlaylistItem {
	return domain.NewPlaylistItem("", a, startOffset, 0, duration)
}

// Ported from playlistExporterWritesSegmentProfileRows.
func TestCSVSegmentProfileRows(t *testing.T) {
	d := dim("")
	asset := domain.MediaAsset{
		ID: "a", Name: "clip", Path: "/tmp/clip.mp4", Kind: domain.KindVideo,
		Duration: dur(30), FileExtension: "MP4", Dimensions: d,
		GraphicTags: []domain.GraphicTag{domain.TagLogo, domain.TagPlus6},
	}
	playlist := []domain.PlaylistItem{wholeRow(asset, 0, 30)}
	markers := map[string][]domain.AdMarker{
		asset.Path: {{Kind: domain.MarkerAdBreak, Time: 10}, {Kind: domain.MarkerAdBreak, Time: 20}},
	}

	csv := string(CSV(playlist, markers, domain.FPS25, ProfileSegments))
	lines := strings.Split(csv, "\n")

	if len(lines) != 4 {
		t.Fatalf("line count = %d, want 4\n%s", len(lines), csv)
	}
	for _, want := range []string{`"Segments"`, `"6+; LOGO"`, `"00:00:10:00"`, `"00:00:20:00"`} {
		if !strings.Contains(csv, want) {
			t.Errorf("csv missing %q\n%s", want, csv)
		}
	}
}

// Ported from playlistExporterWritesTeleCSVPresets.
func TestTeleCSVPresets(t *testing.T) {
	asset := domain.MediaAsset{
		ID: "a", Name: "новости – тест 🎬", Path: "/tmp/новости.mp4", Kind: domain.KindVideo,
		Duration: dur(30), FrameRate: fr(25), Dimensions: dim("1920x1080"), FileExtension: "MP4",
		GraphicTags: []domain.GraphicTag{domain.TagLogo, domain.TagPlus16, domain.TagSmoke, domain.TagReklama},
		ExternalID:  "CL031345",
	}
	playlist := []domain.PlaylistItem{wholeRow(asset, 0, 30)}
	markers := map[string][]domain.AdMarker{asset.Path: {{Kind: domain.MarkerAdBreak, Time: 10}}}

	utf8 := string(TeleCSV(playlist, markers, domain.FPS25, PresetTeleUTF8))
	for _, want := range []string{
		`"START TIME";"NAME";"TC_IN";"DURATION";"TC_OUT";"STORAGE";"LOGO";"SCTE";"REKLAMA"`,
		`"новости – тест 🎬"`,
		`"CL031345"`,
		`"00:00:00:00";"новости – тест 🎬";"00:00:00:00";"00:00:10:00";"00:00:10:00";"CL031345"`,
		`"LOGO 16+ SMOKE";"SCTE_0";"REKLAMA"`,
	} {
		if !strings.Contains(utf8, want) {
			t.Errorf("TELE UTF-8 missing %q", want)
		}
	}
	if strings.Contains(utf8, "AD BREAK") {
		t.Error("TELE must not contain AD BREAK")
	}

	cp := TeleCSV(playlist, markers, domain.FPS25, PresetTeleCP1251)
	// "новости" round-trips back through the CP1251 decoder.
	if got := decodeCP1251(cp); !strings.Contains(got, `"новости`) {
		t.Errorf("CP1251 output missing Cyrillic: %q", got)
	}
}

// Ported from playlistExporterTeleCSVWritesSegmentsAsFullRows.
func TestTeleCSVSegmentsAsFullRows(t *testing.T) {
	asset := domain.MediaAsset{
		ID: "a", Name: "clip", Path: "/tmp/clip.mp4", Kind: domain.KindVideo,
		Duration: dur(30), FrameRate: fr(25), Dimensions: dim("1920x1080"), FileExtension: "MP4",
		GraphicTags: []domain.GraphicTag{domain.TagSCTE},
	}
	playlist := []domain.PlaylistItem{wholeRow(asset, 0, 30)}
	markers := map[string][]domain.AdMarker{
		asset.Path: {{Kind: domain.MarkerAdBreak, Time: 10}, {Kind: domain.MarkerAdBreak, Time: 20}},
	}

	csv := string(TeleCSV(playlist, markers, domain.FPS25, PresetTeleUTF8))
	lines := strings.Split(csv, "\r\n")
	if len(lines) != 4 {
		t.Fatalf("line count = %d, want 4\n%s", len(lines), csv)
	}
	for _, want := range []string{
		`"00:00:00:00";"clip";"00:00:00:00";"00:00:10:00";"00:00:10:00";"clip";"";"SCTE_3";""`,
		`"00:00:10:00";"clip";"00:00:10:00";"00:00:10:00";"00:00:20:00";"clip";"";"SCTE_3";""`,
		`"00:00:20:00";"clip";"00:00:20:00";"00:00:10:00";"00:00:30:00";"clip";"";"SCTE_3";""`,
	} {
		if !strings.Contains(csv, want) {
			t.Errorf("TELE segments missing:\n%s\ngot:\n%s", want, csv)
		}
	}
}

func fr(v float64) *float64 { return &v }
func dim(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// decodeCP1251 is a test-only reverse of EncodeCP1251 for round-trip checks.
func decodeCP1251(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		if c < 0x80 {
			sb.WriteByte(c)
			continue
		}
		if r, ok := cp1251High[c]; ok {
			sb.WriteRune(r)
		} else if c >= 0xC0 {
			sb.WriteRune(rune(0x0410 + int(c-0xC0)))
		} else {
			sb.WriteRune('?')
		}
	}
	return sb.String()
}
