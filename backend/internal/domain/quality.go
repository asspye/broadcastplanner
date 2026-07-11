package domain

import (
	"fmt"
	"math"
	"sort"
)

// CheckSeverity mirrors Swift PlaylistCheckSeverity.
type CheckSeverity string

const (
	SeverityError   CheckSeverity = "ERROR"
	SeverityWarning CheckSeverity = "WARN"
	SeverityInfo    CheckSeverity = "INFO"
)

func (s CheckSeverity) rank() int {
	switch s {
	case SeverityError:
		return 3
	case SeverityWarning:
		return 2
	default:
		return 1
	}
}

// IssueCode mirrors Swift PlaylistIssueCode.
type IssueCode string

const (
	CodeAd        IssueCode = "AD"
	CodeFPS       IssueCode = "FPS"
	CodeMedia     IssueCode = "MEDIA"
	CodeDuration  IssueCode = "DUR"
	CodeRange     IssueCode = "RANGE"
	CodeShort     IssueCode = "SHORT"
	CodeDuplicate IssueCode = "DUP"
	CodeReadiness IssueCode = "READY"
)

// CheckIssue is one quality finding.
type CheckIssue struct {
	ID        string        `json:"id"`
	Severity  CheckSeverity `json:"severity"`
	Code      IssueCode     `json:"code"`
	ItemID    string        `json:"itemID,omitempty"`
	SegmentID string        `json:"segmentID,omitempty"`
	Title     string        `json:"title"`
	Detail    string        `json:"detail"`
}

// QualityReport mirrors Swift PlaylistQualityReport.
type QualityReport struct {
	TotalDuration        float64               `json:"totalDuration"`
	ItemCount            int                   `json:"itemCount"`
	SegmentCount         int                   `json:"segmentCount"`
	IssueCountBySeverity map[CheckSeverity]int `json:"issueCountBySeverity"`
	Issues               []CheckIssue          `json:"issues"`
}

// StatusTitle: ERROR > WARN > (EMPTY when no items) > OK.
func (r QualityReport) StatusTitle() string {
	if r.IssueCountBySeverity[SeverityError] > 0 {
		return "ERROR"
	}
	if r.IssueCountBySeverity[SeverityWarning] > 0 {
		return "WARN"
	}
	if r.ItemCount == 0 {
		return "EMPTY"
	}
	return "OK"
}

// IssuesFor returns issues for a given item/segment, sorted by severity desc —
// the row-badge query (Swift qualityIssues(for:segment:)).
func (r QualityReport) IssuesFor(itemID, segmentID string) []CheckIssue {
	var out []CheckIssue
	for _, iss := range r.Issues {
		if iss.SegmentID == segmentID || (iss.SegmentID == "" && iss.ItemID == itemID) {
			out = append(out, iss)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Severity.rank() > out[j].Severity.rank()
	})
	return out
}

// FileChecker reports whether a media path is currently available (real file /
// present S3 object). Virtual assets bypass this.
type FileChecker func(path string) bool

func sortedMarkers(byPath map[string][]AdMarker, path string) []AdMarker {
	ms := append([]AdMarker(nil), byPath[path]...)
	sort.SliceStable(ms, func(i, j int) bool { return ms[i].Time < ms[j].Time })
	return ms
}

func shouldRequireAdMarkers(item PlaylistItem) bool {
	const tenMinutes = 600.0
	if item.Asset.Duration == nil {
		return true
	}
	dur := *item.Asset.Duration
	isWholeFile := item.SourceIn <= 0.05 && math.Abs(item.SourceOut-dur) <= 0.05
	return isWholeFile && dur > tenMinutes
}

func duplicateRangeKey(item PlaylistItem, fr ProjectFrameRate) string {
	fps := fr.FramesPerSecond()
	in := int64(math.Round(item.SourceIn * fps))
	out := int64(math.Round(item.SourceOut * fps))
	return fmt.Sprintf("%s|%d|%d", item.Asset.Path, in, out)
}

// BuildQualityReport ports Swift buildPlaylistQualityReport. fileExists may be nil
// (treated as "not available" for non-virtual assets).
func BuildQualityReport(playlist []PlaylistItem, markersByPath map[string][]AdMarker, fr ProjectFrameRate, fileExists FileChecker) QualityReport {
	if fileExists == nil {
		fileExists = func(string) bool { return false }
	}
	var issues []CheckIssue
	segmentCount := 0
	totalDuration := 0.0
	projectFps := fr.FramesPerSecond()

	// Group non-live media items by source range for duplicate detection.
	dupGroups := map[string][]PlaylistItem{}
	for _, item := range playlist {
		if item.IsNonTimingRow() || item.IsLiveBreakPlaceholder() {
			continue
		}
		dupGroups[duplicateRangeKey(item, fr)] = append(dupGroups[duplicateRangeKey(item, fr)], item)
	}

	for i, item := range playlist {
		if !item.IsNonTimingRow() {
			totalDuration += item.Duration
		}
		if item.IsNonTimingRow() {
			continue
		}
		n := i + 1
		markers := sortedMarkers(markersByPath, item.Asset.Path)
		segs := Segments(item, markers)
		segmentCount += len(segs)
		available := item.Asset.IsVirtual() || fileExists(item.Asset.Path)

		if item.Asset.IsMissingPlaceholder() || (!item.Asset.IsVirtual() && !fileExists(item.Asset.Path)) {
			issues = append(issues, CheckIssue{
				ID: fmt.Sprintf("missing-media-%s", item.ID), Severity: SeverityError, Code: CodeMedia,
				ItemID: item.ID, Title: fmt.Sprintf("#%d файл не найден", n), Detail: item.Asset.Path,
			})
		}
		if item.Asset.Duration == nil {
			issues = append(issues, CheckIssue{
				ID: fmt.Sprintf("missing-duration-%s", item.ID), Severity: SeverityError, Code: CodeDuration,
				ItemID: item.ID, Title: fmt.Sprintf("#%d нет длительности", n), Detail: item.Asset.Name,
			})
		}
		if !item.IsLiveBreakPlaceholder() && item.Asset.Status != "Готово" {
			issues = append(issues, CheckIssue{
				ID: fmt.Sprintf("not-ready-%s", item.ID), Severity: SeverityWarning, Code: CodeReadiness,
				ItemID: item.ID, Title: fmt.Sprintf("#%d анализ не завершен", n),
				Detail: fmt.Sprintf("%s · %s", item.Asset.Name, item.Asset.Status),
			})
		}
		if !item.IsLiveBreakPlaceholder() && item.Asset.FrameRate != nil && math.Abs(*item.Asset.FrameRate-projectFps) > 0.2 {
			fr := *item.Asset.FrameRate
			issues = append(issues, CheckIssue{
				ID: fmt.Sprintf("fps-%s", item.ID), Severity: SeverityWarning, Code: CodeFPS,
				ItemID: item.ID, Title: fmt.Sprintf("#%d FPS отличается", n),
				Detail: fmt.Sprintf("%s fps вместо %s", Decimal(&fr), string(projectFrameRateRaw(projectFps))),
			})
		}
		if !item.IsLiveBreakPlaceholder() && item.Asset.Kind == KindVideo && shouldRequireAdMarkers(item) && !hasAdBreak(markers) {
			issues = append(issues, CheckIssue{
				ID: fmt.Sprintf("no-markers-%s", item.ID), Severity: SeverityWarning, Code: CodeAd,
				ItemID: item.ID, Title: fmt.Sprintf("#%d нет AD-точек", n), Detail: item.Asset.Name,
			})
		}
		if available && item.Asset.Duration != nil {
			dur := *item.Asset.Duration
			for _, seg := range segs {
				if seg.SourceIn < -0.01 || seg.SourceOut > dur+0.01 {
					issues = append(issues, CheckIssue{
						ID: fmt.Sprintf("range-%s", seg.ID), Severity: SeverityError, Code: CodeRange,
						ItemID: item.ID, SegmentID: seg.ID,
						Title:  fmt.Sprintf("#%d.%d диапазон вне файла", n, seg.SegmentIndex),
						Detail: fmt.Sprintf("%s · файл %s, TC %s-%s", item.Asset.Name, Timecode(dur, fr), Timecode(seg.SourceIn, fr), Timecode(seg.SourceOut, fr)),
					})
				}
			}
		}
		for _, seg := range segs {
			if seg.Duration() < 5 {
				issues = append(issues, CheckIssue{
					ID: fmt.Sprintf("short-%s", seg.ID), Severity: SeverityWarning, Code: CodeShort,
					ItemID: item.ID, SegmentID: seg.ID,
					Title:  fmt.Sprintf("#%d.%d короткий сегмент", n, seg.SegmentIndex),
					Detail: fmt.Sprintf("%s · %s", item.Asset.Name, Timecode(seg.Duration(), fr)),
				})
			}
		}
		if item.Duration <= 0 {
			issues = append(issues, CheckIssue{
				ID: fmt.Sprintf("zero-duration-%s", item.ID), Severity: SeverityError, Code: CodeDuration,
				ItemID: item.ID, Title: fmt.Sprintf("#%d нулевая длительность", n), Detail: item.Asset.Name,
			})
		}
	}

	// Duplicate ranges (info) — stable order by key then item.
	dupKeys := make([]string, 0, len(dupGroups))
	for k := range dupGroups {
		dupKeys = append(dupKeys, k)
	}
	sort.Strings(dupKeys)
	for _, k := range dupKeys {
		items := dupGroups[k]
		if len(items) <= 1 {
			continue
		}
		for _, item := range items {
			issues = append(issues, CheckIssue{
				ID: fmt.Sprintf("duplicate-%s-%s", k, item.ID), Severity: SeverityInfo, Code: CodeDuplicate,
				ItemID: item.ID, Title: "Повтор диапазона",
				Detail: fmt.Sprintf("%s · %s-%s · %d раз", item.Asset.Name, Timecode(item.SourceIn, fr), Timecode(item.SourceOut, fr), len(items)),
			})
		}
	}

	counts := map[CheckSeverity]int{}
	for _, iss := range issues {
		counts[iss.Severity]++
	}
	return QualityReport{
		TotalDuration: totalDuration, ItemCount: len(playlist), SegmentCount: segmentCount,
		IssueCountBySeverity: counts, Issues: issues,
	}
}

func hasAdBreak(markers []AdMarker) bool {
	for _, m := range markers {
		if m.Kind == MarkerAdBreak {
			return true
		}
	}
	return false
}

// projectFrameRateRaw finds the raw label for a real fps value (for FPS detail
// text). Best-effort; falls back to a formatted number.
func projectFrameRateRaw(fps float64) ProjectFrameRate {
	for _, r := range AllFrameRates {
		if math.Abs(r.FramesPerSecond()-fps) < 0.001 {
			return r
		}
	}
	return ProjectFrameRate(fmt.Sprintf("%.2f", fps))
}
