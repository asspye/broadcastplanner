package domain

import "strings"

// GraphicTag mirrors the Swift GraphicTag enum. RawValue is the on-air tag code
// as stored in project files and exports; keep these strings stable for
// compatibility with existing .tvassembly / TELE data.
type GraphicTag string

const (
	TagLogo    GraphicTag = "LOGO"
	TagSmoke   GraphicTag = "SMOKE"
	TagPlus0   GraphicTag = "+0"
	TagPlus6   GraphicTag = "+6"
	TagPlus12  GraphicTag = "+12"
	TagPlus16  GraphicTag = "+16"
	TagPlus18  GraphicTag = "+18"
	TagSCTE    GraphicTag = "SCTE"
	TagReklama GraphicTag = "REKLAMA"
)

// AllGraphicTags in declaration order (matches Swift allCases).
var AllGraphicTags = []GraphicTag{
	TagLogo, TagSmoke, TagPlus0, TagPlus6, TagPlus12, TagPlus16, TagPlus18, TagSCTE, TagReklama,
}

// AgeTags is the set of mutually-exclusive age ratings.
var ageTagSet = map[GraphicTag]bool{
	TagPlus0: true, TagPlus6: true, TagPlus12: true, TagPlus16: true, TagPlus18: true,
}

// DisplayName renders age tags as "0+".."18+"; other tags return the raw value.
func (t GraphicTag) DisplayName() string {
	switch t {
	case TagPlus0:
		return "0+"
	case TagPlus6:
		return "6+"
	case TagPlus12:
		return "12+"
	case TagPlus16:
		return "16+"
	case TagPlus18:
		return "18+"
	default:
		return string(t)
	}
}

// IsAgeTag reports whether the tag is an age rating.
func (t GraphicTag) IsAgeTag() bool { return ageTagSet[t] }

// SortOrder is the string sort key used to order tags in labels/exports.
func (t GraphicTag) SortOrder() string {
	switch t {
	case TagPlus0:
		return "00"
	case TagPlus6:
		return "06"
	case TagPlus12:
		return "12"
	case TagPlus16:
		return "16"
	case TagPlus18:
		return "18"
	default:
		return "50-" + string(t)
	}
}

// MatchGraphicTag resolves a label (raw value or display name, case-insensitive)
// to a GraphicTag, returning ok=false when nothing matches.
func MatchGraphicTag(label string) (GraphicTag, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(label))
	for _, t := range AllGraphicTags {
		if strings.ToUpper(string(t)) == normalized || strings.ToUpper(t.DisplayName()) == normalized {
			return t, true
		}
	}
	return "", false
}
