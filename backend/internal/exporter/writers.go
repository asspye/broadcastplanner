package exporter

import (
	"strconv"
	"strings"

	"github.com/broadcastplanner/backend/internal/domain"
)

func itoa(i int) string { return strconv.Itoa(i) }

func escapeCSV(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

func escapeXML(v string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(v)
}

// CSV renders the universal/segments/items CSV (UTF-8, comma-separated).
// The header row is unquoted; data rows are quoted — matching the Swift output.
func CSV(playlist []domain.PlaylistItem, markersByPath map[string][]domain.AdMarker, fr domain.ProjectFrameRate, profile Profile) []byte {
	lines := []string{strings.Join(exportHeaders, ",")}
	for _, row := range buildRows(playlist, markersByPath, fr, profile) {
		vals := row.values()
		quoted := make([]string, len(vals))
		for i, v := range vals {
			quoted[i] = escapeCSV(v)
		}
		lines = append(lines, strings.Join(quoted, ","))
	}
	return []byte(strings.Join(lines, "\n"))
}

// PlaylistCSV dispatches on preset: TELE presets produce the ';'-separated CRLF
// format (CP1251 or UTF-8), everything else the universal CSV.
func PlaylistCSV(playlist []domain.PlaylistItem, markersByPath map[string][]domain.AdMarker, fr domain.ProjectFrameRate, preset Preset) []byte {
	if preset.IsTele() {
		return TeleCSV(playlist, markersByPath, fr, preset)
	}
	return CSV(playlist, markersByPath, fr, preset.Profile())
}

// teleHeaders is the fixed TELE column set consumed by the broadcast automation.
var teleHeaders = []string{"START TIME", "NAME", "TC_IN", "DURATION", "TC_OUT", "STORAGE", "LOGO", "SCTE", "REKLAMA"}

// teleRow derives the LOGO/SCTE/REKLAMA columns from an exportRow's graphics.
func teleRow(r exportRow) []string {
	set := map[string]bool{}
	for _, g := range strings.Split(r.graphics, ";") {
		g = strings.TrimSpace(g)
		if g != "" {
			set[g] = true
		}
	}

	var logoParts []string
	if set[string(domain.TagLogo)] {
		logoParts = append(logoParts, string(domain.TagLogo))
	}
	for _, t := range domain.AllGraphicTags {
		if t.IsAgeTag() && (set[t.DisplayName()] || set[string(t)]) {
			logoParts = append(logoParts, t.DisplayName())
			break
		}
	}
	if set[string(domain.TagSmoke)] {
		logoParts = append(logoParts, string(domain.TagSmoke))
	}

	scte := "SCTE_0"
	if set[string(domain.TagSCTE)] {
		scte = "SCTE_3"
	}
	reklama := ""
	if set[string(domain.TagReklama)] {
		reklama = string(domain.TagReklama)
	}

	return []string{r.start, r.title, r.tcIn, r.duration, r.tcOut, r.storage, strings.Join(logoParts, " "), scte, reklama}
}

// TeleCSV renders the TELE format: ';'-separated, all fields quoted, CRLF line
// endings, CP1251 (lossy) or UTF-8 depending on the preset.
func TeleCSV(playlist []domain.PlaylistItem, markersByPath map[string][]domain.AdMarker, fr domain.ProjectFrameRate, preset Preset) []byte {
	quoteJoin := func(vals []string) string {
		q := make([]string, len(vals))
		for i, v := range vals {
			q[i] = escapeCSV(v)
		}
		return strings.Join(q, ";")
	}

	lines := []string{quoteJoin(teleHeaders)}
	for _, row := range buildRows(playlist, markersByPath, fr, ProfileUniversal) {
		lines = append(lines, quoteJoin(teleRow(row)))
	}
	csv := strings.Join(lines, "\r\n")

	if preset == PresetTeleCP1251 {
		return EncodeCP1251(csv)
	}
	return []byte(csv)
}
