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
