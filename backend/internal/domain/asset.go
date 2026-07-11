package domain

import (
	"sort"
	"strings"
)

// MediaKind mirrors the Swift MediaKind enum. RawValues are Russian strings that
// are serialised into project files and the media catalog; keep them stable.
type MediaKind string

const (
	KindVideo   MediaKind = "Видео"
	KindAudio   MediaKind = "Аудио"
	KindImage   MediaKind = "Графика"
	KindUnknown MediaKind = "Файл"
)

// Virtual path prefixes used for placeholder assets that have no real file.
const (
	virtualPrefix      = "/TVAssembly/"
	missingMediaPrefix = "/TVAssembly/MissingMedia/"
)

// MediaAsset is a single item in the media library. Path replaces the Swift URL;
// all matching/marker keys use Path.
type MediaAsset struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Path              string       `json:"path"`
	Kind              MediaKind    `json:"kind"`
	Duration          *float64     `json:"duration,omitempty"`
	FrameRate         *float64     `json:"frameRate,omitempty"`
	Dimensions        *string      `json:"dimensions,omitempty"`
	FileExtension     string       `json:"fileExtension"`
	Status            string       `json:"status"`
	GraphicTags       []GraphicTag `json:"graphicTags"`
	ExternalID        string       `json:"externalID"`
	Comment           string       `json:"comment"`
	ProductionYear    string       `json:"productionYear"`
	Director          string       `json:"director"`
	Production        string       `json:"production"`
	Genre             string       `json:"genre"`
	Synopsis          string       `json:"synopsis"`
	Categories        []string     `json:"categories"`
	CustomGraphicTags []string     `json:"customGraphicTags"`
}

// IsVirtual reports whether the asset is a placeholder without a real file.
func (a MediaAsset) IsVirtual() bool { return strings.HasPrefix(a.Path, virtualPrefix) }

// IsMissingPlaceholder reports a "missing media" placeholder from playlist import.
func (a MediaAsset) IsMissingPlaceholder() bool {
	return strings.HasPrefix(a.Path, missingMediaPrefix)
}

// StorageName is the ExternalID when present, otherwise Name.
func (a MediaAsset) StorageName() string {
	trimmed := strings.TrimSpace(a.ExternalID)
	if trimmed == "" {
		return a.Name
	}
	return trimmed
}

// GraphicLabels returns built-in tag display names (sorted by SortOrder) followed
// by sorted custom tags — the exact ordering used by exports.
func (a MediaAsset) GraphicLabels() []string {
	builtIn := append([]GraphicTag(nil), a.GraphicTags...)
	sort.Slice(builtIn, func(i, j int) bool {
		return builtIn[i].SortOrder() < builtIn[j].SortOrder()
	})
	labels := make([]string, 0, len(builtIn)+len(a.CustomGraphicTags))
	for _, t := range builtIn {
		labels = append(labels, t.DisplayName())
	}
	custom := append([]string(nil), a.CustomGraphicTags...)
	sort.Strings(custom)
	labels = append(labels, custom...)
	return labels
}

// HasGraphicTag reports membership in the built-in tag set.
func (a MediaAsset) HasGraphicTag(tag GraphicTag) bool {
	for _, t := range a.GraphicTags {
		if t == tag {
			return true
		}
	}
	return false
}
