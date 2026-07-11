package domain

// Recalculate assigns StartOffset to every row from clockStart, advancing the
// clock only on timing rows, and (when airDate is set) injects air-date comment
// rows at each 24h boundary. Ported 1:1 from Swift recalculatePlaylist.
//
// It is pure: returns a new slice, does not mutate the input.
func Recalculate(playlist []PlaylistItem, clockStart float64, airDate *BroadcastAirDate) []PlaylistItem {
	cursor := clockStart

	// When an air date is set, drop pre-existing air-date comments; they are
	// regenerated deterministically below.
	source := playlist
	if airDate != nil {
		source = make([]PlaylistItem, 0, len(playlist))
		for _, item := range playlist {
			if item.IsCommentRow() && IsAirDateComment(item.CommentText) {
				continue
			}
			source = append(source, item)
		}
	}

	nextDateBoundary := clockStart + secondsPerDay
	airDateOffset := 0
	out := make([]PlaylistItem, 0, len(source)+1)

	if airDate != nil {
		row := NewCommentRow(airDate.DisplayText())
		row.StartOffset = cursor
		out = append(out, row)
	}

	for _, item := range source {
		copy := item
		copy.StartOffset = cursor
		out = append(out, copy)

		if !copy.IsNonTimingRow() {
			cursor += copy.Duration
			for airDate != nil && cursor >= nextDateBoundary-0.001 {
				airDateOffset++
				row := NewCommentRow(airDate.AddingDays(airDateOffset).DisplayText())
				row.StartOffset = cursor
				out = append(out, row)
				nextDateBoundary += secondsPerDay
			}
		}
	}

	return out
}
