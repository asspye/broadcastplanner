// Package exporter renders playlists to the on-air file formats (universal
// CSV/XML/XLSX and the TELE CSV used by the broadcast automation), ported from
// the Swift PlaylistExporter.
package exporter

import (
	"path"
	"sort"
	"strings"

	"github.com/broadcastplanner/backend/internal/domain"
)

// Profile mirrors Swift PlaylistExportProfile.
type Profile string

const (
	ProfileUniversal Profile = "Universal"
	ProfileSegments  Profile = "Segments"
	ProfileItems     Profile = "Items"
)

// Preset mirrors Swift PlaylistCSVExportPreset.
type Preset string

const (
	PresetUniversal  Preset = "Universal"
	PresetSegments   Preset = "Segments"
	PresetItems      Preset = "Items"
	PresetTeleCP1251 Preset = "TELE CP1251"
	PresetTeleUTF8   Preset = "TELE UTF-8"
)

// IsTele reports whether the preset targets the TELE automation format.
func (p Preset) IsTele() bool { return p == PresetTeleCP1251 || p == PresetTeleUTF8 }

// Profile maps a preset to the row-generation profile (TELE uses Universal).
func (p Preset) Profile() Profile {
	switch p {
	case PresetSegments:
		return ProfileSegments
	case PresetItems:
		return ProfileItems
	default:
		return ProfileUniversal
	}
}

// exportRow is the intermediate representation shared by every output format.
type exportRow struct {
	itemIndex   int
	segmentIdx  int
	mode        string
	title       string
	start       string
	duration    string
	tcIn        string
	tcOut       string
	rowType     string
	storage     string
	graphics    string
	file        string
	pathStr     string
	markers     []domain.AdMarker
	markersText string
	note        string
}

var exportHeaders = []string{
	"#", "Segment", "Mode", "Start", "Duration", "TC_IN", "TC_OUT",
	"Type", "Storage", "Graphics", "File", "Path", "Markers", "Note",
}

func (r exportRow) values() []string {
	seg := ""
	if r.segmentIdx != 0 {
		seg = itoa(r.segmentIdx)
	}
	return []string{
		itoa(r.itemIndex), seg, r.mode, r.start, r.duration, r.tcIn, r.tcOut,
		r.rowType, r.storage, r.graphics, r.file, r.pathStr, r.markersText, r.note,
	}
}

// BuildRows generates export rows for a playlist under the given profile.
func buildRows(playlist []domain.PlaylistItem, markersByPath map[string][]domain.AdMarker, fr domain.ProjectFrameRate, profile Profile) []exportRow {
	var rows []exportRow
	for i, item := range playlist {
		if item.IsNonTimingRow() {
			continue
		}
		itemMarkers := markersForItem(item, markersByPath)
		switch profile {
		case ProfileUniversal, ProfileSegments:
			for _, seg := range domain.Segments(item, itemMarkers) {
				rowMarkers := itemMarkers
				if profile == ProfileSegments {
					rowMarkers = markersInRange(itemMarkers, seg.SourceIn, seg.SourceOut)
				}
				rows = append(rows, exportRow{
					itemIndex:   i + 1,
					segmentIdx:  seg.SegmentIndex,
					mode:        string(profile),
					title:       item.Asset.Name,
					start:       domain.BroadcastClockTimecode(seg.StartOffset, fr),
					duration:    domain.Timecode(seg.Duration(), fr),
					tcIn:        domain.Timecode(seg.SourceIn, fr),
					tcOut:       domain.Timecode(seg.SourceOut, fr),
					rowType:     string(item.Asset.Kind),
					storage:     item.Asset.StorageName(),
					graphics:    graphicsSummary(item.Asset),
					file:        path.Base(item.Asset.Path),
					pathStr:     item.Asset.Path,
					markers:     rowMarkers,
					markersText: markerSummary(rowMarkers, fr),
					note:        item.Note,
				})
			}
		case ProfileItems:
			m := markersInRange(itemMarkers, item.SourceIn, item.SourceOut)
			rows = append(rows, exportRow{
				itemIndex:   i + 1,
				segmentIdx:  0,
				mode:        string(profile),
				title:       item.Asset.Name,
				start:       domain.BroadcastClockTimecode(item.StartOffset, fr),
				duration:    domain.Timecode(item.Duration, fr),
				tcIn:        domain.Timecode(item.SourceIn, fr),
				tcOut:       domain.Timecode(item.SourceOut, fr),
				rowType:     string(item.Asset.Kind),
				storage:     item.Asset.StorageName(),
				graphics:    graphicsSummary(item.Asset),
				file:        path.Base(item.Asset.Path),
				pathStr:     item.Asset.Path,
				markers:     m,
				markersText: markerSummary(m, fr),
				note:        item.Note,
			})
		}
	}
	return rows
}

func markersForItem(item domain.PlaylistItem, byPath map[string][]domain.AdMarker) []domain.AdMarker {
	ms := append([]domain.AdMarker(nil), byPath[item.Asset.Path]...)
	sort.SliceStable(ms, func(i, j int) bool { return ms[i].Time < ms[j].Time })
	return ms
}

func markersInRange(ms []domain.AdMarker, lo, hi float64) []domain.AdMarker {
	var out []domain.AdMarker
	for _, m := range ms {
		if m.Time >= lo && m.Time <= hi {
			out = append(out, m)
		}
	}
	return out
}

func markerSummary(ms []domain.AdMarker, fr domain.ProjectFrameRate) string {
	parts := make([]string, 0, len(ms))
	for _, m := range ms {
		parts = append(parts, string(m.Kind)+" "+domain.Timecode(m.Time, fr))
	}
	return strings.Join(parts, "; ")
}

func graphicsSummary(a domain.MediaAsset) string {
	return strings.Join(a.GraphicLabels(), "; ")
}
