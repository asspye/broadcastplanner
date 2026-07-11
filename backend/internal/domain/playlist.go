package domain

// RowKind mirrors Swift PlaylistItem.RowKind.
type RowKind string

const (
	RowMedia       RowKind = "media"
	RowComment     RowKind = "comment"
	RowBreakHeader RowKind = "breakHeader"
	RowLiveBreak   RowKind = "liveBreak"
)

// MarkerKind mirrors Swift MarkerKind (RawValues are the on-air labels).
type MarkerKind string

const (
	MarkerIn      MarkerKind = "IN"
	MarkerOut     MarkerKind = "OUT"
	MarkerAdBreak MarkerKind = "AD BREAK"
)

// AdMarker is a timed marker attached to an asset (keyed by asset path).
type AdMarker struct {
	ID   string     `json:"id"`
	Kind MarkerKind `json:"kind"`
	Time float64    `json:"time"`
	Note string     `json:"note"`
}

// PlaylistItem is one row of the playlist. Duration is derived as
// max(0, SourceOut-SourceIn); use NewPlaylistItem to keep it consistent.
type PlaylistItem struct {
	ID          string     `json:"id"`
	Asset       MediaAsset `json:"asset"`
	StartOffset float64    `json:"startOffset"`
	Duration    float64    `json:"duration"`
	SourceIn    float64    `json:"sourceIn"`
	SourceOut   float64    `json:"sourceOut"`
	Note        string     `json:"note"`
	RowKind     RowKind    `json:"rowKind"`
	CommentText string     `json:"commentText"`
}

// NewPlaylistItem builds a media row, deriving Duration = max(0, sourceOut-sourceIn),
// mirroring the Swift PlaylistItem initializer. ID may be empty to auto-assign.
func NewPlaylistItem(id string, asset MediaAsset, startOffset, sourceIn, sourceOut float64) PlaylistItem {
	if id == "" {
		id = NewID()
	}
	dur := sourceOut - sourceIn
	if dur < 0 {
		dur = 0
	}
	return PlaylistItem{
		ID:          id,
		Asset:       asset,
		StartOffset: startOffset,
		Duration:    dur,
		SourceIn:    sourceIn,
		SourceOut:   sourceOut,
		RowKind:     RowMedia,
	}
}

// NewCommentRow builds a comment row backed by a virtual placeholder asset,
// mirroring the Swift PlaylistItem(commentText:) initializer.
func NewCommentRow(text string) PlaylistItem {
	return PlaylistItem{
		ID: NewID(),
		Asset: MediaAsset{
			Name:   "Комментарий",
			Path:   "/TVAssembly/CommentRow",
			Kind:   KindUnknown,
			Status: "Комментарий",
		},
		RowKind:     RowComment,
		CommentText: text,
	}
}

// IsCommentRow / IsBreakHeaderRow / IsLiveBreakPlaceholder / IsNonTimingRow
// mirror the Swift computed flags. Non-timing rows do not advance the clock.
func (p PlaylistItem) IsCommentRow() bool           { return p.RowKind == RowComment }
func (p PlaylistItem) IsBreakHeaderRow() bool       { return p.RowKind == RowBreakHeader }
func (p PlaylistItem) IsLiveBreakPlaceholder() bool { return p.RowKind == RowLiveBreak }
func (p PlaylistItem) IsNonTimingRow() bool         { return p.IsCommentRow() || p.IsBreakHeaderRow() }

// PlaylistSegment is a slice of a PlaylistItem between AD BREAK cut points.
type PlaylistSegment struct {
	ID           string     `json:"id"`
	ParentItemID string     `json:"parentItemID"`
	SegmentIndex int        `json:"segmentIndex"`
	Asset        MediaAsset `json:"asset"`
	StartOffset  float64    `json:"startOffset"`
	EndOffset    float64    `json:"endOffset"`
	SourceIn     float64    `json:"sourceIn"`
	SourceOut    float64    `json:"sourceOut"`
}

// Duration of a segment.
func (s PlaylistSegment) Duration() float64 {
	d := s.SourceOut - s.SourceIn
	if d < 0 {
		return 0
	}
	return d
}
