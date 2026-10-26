package models

import "strings"

// Flower represents a single flower/entry in the database
type Flower struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	BotanicalName string     `json:"botanical_name"`
	Role          FSTRole    `json:"role"`  // filler, spiller, thriller
	ZoneMin       int        `json:"zone_min"`
	ZoneMax       int        `json:"zone_max"`
	Sun           SunType    `json:"sun"`   // full_sun, part_shade, full_shade
	Soils         []SoilType `json:"soils"` // clay, loam, sandy, well_drained
	Color         string     `json:"color"`
	BloomSeason   string     `json:"bloom_season"` // spring, summer, fall, all_season
	Height        string     `json:"height"`    // e.g. "12-18in", "6-12in"
	Spacing       string     `json:"spacing"`   // e.g. "8-12in"
	Description   string     `json:"description"`
	ImageURL      string     `json:"image_url"` // emoji or URL
	WikipediaURL  string     `json:"wikipedia_url"` // link to Wikipedia
}

// FSTRole is the Filler/Spiller/Thriller role
type FSTRole string

const (
	RoleThriller FSTRole = "thriller"
	RoleFiller   FSTRole = "filler"
	RoleSpiller  FSTRole = "spiller"
)

// SunType is the sun requirement
type SunType string

const (
	SunFullShade SunType = "full_shade"
	SunPartShade SunType = "part_shade"
	SunFullSun   SunType = "full_sun"
)

// SoilType is a soil type
type SoilType string

const (
	SoilClay        SoilType = "clay"
	SoilLoam        SoilType = "loam"
	SoilSandy       SoilType = "sandy"
	SoilWellDrained SoilType = "well_drained"
)

// FSTRecommendation is a recommended set of flowers in FST layout
type FSTRecommendation struct {
	LayoutType string     `json:"layout_type"` // "container" or "bed"
	Zone       int        `json:"zone"`
	Sun        SunType    `json:"sun"`
	Soils      []SoilType `json:"soils"`
	Thriller   *Flower    `json:"thriller"`
	Fillers    []Flower   `json:"fillers"`
	Spillers   []Flower   `json:"spillers"`
	Notes      string     `json:"notes"`
}

// IsValidZone checks if a zone number is valid (3-10)
func IsValidZone(zone int) bool {
	return zone >= 3 && zone <= 10
}

// ColorPalette defines a named set of color keywords for coordinated layouts.
type ColorPalette string

const (
	PaletteNone       ColorPalette = ""
	PaletteWarm       ColorPalette = "warm"       // Red, Orange, Yellow, Gold
	PaletteCool       ColorPalette = "cool"       // Blue, Purple, Lavender, White, Silver
	PalettePinks      ColorPalette = "pinks"      // Pink, Rose, Magenta
	PaletteMonochrome ColorPalette = "monochrome" // Green, Foliage, Silver, Gray
	PaletteJewel      ColorPalette = "jewel"      // Deep Purple, Burgundy, Navy, Gold, Bronze
)

// PaletteColors maps each palette to its matching keywords.
var PaletteColors = map[ColorPalette][]string{
	PaletteWarm:       {"red", "orange", "yellow", "gold"},
	PaletteCool:       {"blue", "purple", "lavender", "white", "silver"},
	PalettePinks:      {"pink", "rose", "magenta"},
	PaletteMonochrome: {"green", "foliage", "silver", "gray", "bronze"},
	PaletteJewel:      {"purple", "burgundy", "navy", "gold", "bronze", "deep"},
}

// colorHex maps color keywords to representative hex values (Redouté palette:
// soft, slightly muted watercolor tones rather than saturated primaries).
var colorHex = map[string]string{
	"red":      "#C05B5B",
	"orange":   "#D98A4E",
	"yellow":   "#E3C766",
	"gold":     "#C9A24B",
	"pink":     "#E3A7BF",
	"rose":     "#D4899B",
	"magenta":  "#C46AA0",
	"blue":     "#7B9CC4",
	"purple":   "#9A7BB8",
	"lavender": "#B3A3CE",
	"violet":   "#9A7BB8",
	"white":    "#F5F2EA",
	"silver":   "#C4C9C0",
	"gray":     "#B8BDB4",
	"green":    "#8AAA78",
	"foliage":  "#7A9A6A",
	"bronze":   "#B08D57",
	"burgundy": "#8E4A5B",
	"navy":     "#5B6B8C",
	"deep":     "#7A5B8E",
	"coral":    "#E08A7A",
	"peach":    "#EDB894",
	"cream":    "#F0E8D0",
	"chartreuse": "#B8C46A",
	"apricot":  "#E8A870",
	"crimson":  "#B04850",
	"scarlet":  "#C05050",
}

// ColorHex returns a representative CSS hex color derived from the flower's
// color description (e.g. "Pink" -> soft pink hex). Falls back to sage green.
// Multi-color descriptions use the first recognized keyword.
func (f *Flower) ColorHex() string {
	lower := strings.ToLower(f.Color)
	best, bestIdx := "", len(lower)+1
	for kw, hex := range colorHex {
		if i := strings.Index(lower, kw); i >= 0 && i < bestIdx {
			best, bestIdx = hex, i
		}
	}
	if best == "" {
		return "#8AAA78" // sage green default
	}
	return best
}

// MatchesPalette returns true if the flower's color field contains any keyword
// from the given palette.
func (f *Flower) MatchesPalette(p ColorPalette) bool {
	if p == PaletteNone {
		return true // no filter
	}
	keywords, ok := PaletteColors[p]
	if !ok {
		return true // unknown palette = no filter
	}
	color := strings.ToLower(f.Color)
	for _, kw := range keywords {
		if strings.Contains(color, kw) {
			return true
		}
	}
	return false
}

// SoilsCSV returns comma-separated soil types
func (f *Flower) SoilsCSV() string {
	if len(f.Soils) == 0 {
		return ""
	}
	parts := make([]string, len(f.Soils))
	for i, s := range f.Soils {
		parts[i] = string(s)
	}
	return strings.Join(parts, ",")
}
