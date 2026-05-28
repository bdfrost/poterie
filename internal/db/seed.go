package db

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bdfrost/poterie/internal/models"
)

//go:embed catalog-v1.json
var catalogJSON []byte

// ExportCatalogJSON returns the raw embedded catalog JSON bytes.
func ExportCatalogJSON() []byte {
	return catalogJSON
}

// Catalog holds the versioned flower catalog loaded from JSON.
type Catalog struct {
	Version int             `json:"catalog_version"`
	Flowers []models.Flower `json:"flowers"`
}

// LoadCatalog parses the embedded catalog JSON.
func LoadCatalog() (*Catalog, error) {
	var cat Catalog
	if err := json.Unmarshal(catalogJSON, &cat); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	return &cat, nil
}

// Seed inserts flower data if the database is empty.
func Seed(d *DB) error {
	count, err := d.Count()
	if err != nil {
		return fmt.Errorf("count: %w", err)
	}
	if count > 0 {
		return nil // already seeded
	}

	cat, err := LoadCatalog()
	if err != nil {
		return err
	}

	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, f := range cat.Flowers {
		if _, err := tx.Exec(`INSERT INTO flowers (name, botanical_name, role, zone_min, zone_max, sun, soils,
			color, bloom_season, height, spacing, description, image_url, wikipedia_url)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			f.Name, f.BotanicalName, f.Role, f.ZoneMin, f.ZoneMax, f.Sun, f.SoilsCSV(),
			f.Color, f.BloomSeason, f.Height, f.Spacing, f.Description, f.ImageURL, f.WikipediaURL); err != nil {
			return fmt.Errorf("seed %s: %w", f.Name, err)
		}
	}
	return tx.Commit()
}

// SeedFromJSON allows seeding from an arbitrary JSON catalog
// (used by admin import or custom catalogs).
func SeedFromJSON(d *DB, data []byte) error {
	var parsed []models.Flower
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("parse catalog JSON: %w", err)
	}

	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, f := range parsed {
		if _, err := tx.Exec(`INSERT INTO flowers (name, botanical_name, role, zone_min, zone_max, sun, soils,
			color, bloom_season, height, spacing, description, image_url, wikipedia_url)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			f.Name, f.BotanicalName, f.Role, f.ZoneMin, f.ZoneMax, f.Sun, f.SoilsCSV(),
			f.Color, f.BloomSeason, f.Height, f.Spacing, f.Description, f.ImageURL, f.WikipediaURL); err != nil {
			// Skip duplicates (by name) during import — allows partial re-imports
			if strings.Contains(err.Error(), "UNIQUE") {
				continue
			}
			return fmt.Errorf("seed %s: %w", f.Name, err)
		}
	}
	return tx.Commit()
}
