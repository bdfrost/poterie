package db

import (
	"database/sql"
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

// Seed inserts flower data on first run, then upgrades the catalog in place
// when a newer catalog_version ships in a later release. Flowers the user
// customized via the admin panel (customized=1) are never overwritten;
// user-created flowers (names not in the embedded catalog) are untouched.
func Seed(d *DB) error {
	// Validate catalog before doing anything.
	cat, err := LoadCatalog()
	if err != nil {
		return fmt.Errorf("load catalog: %w", err)
	}
	if len(cat.Flowers) == 0 {
		return fmt.Errorf("seed: catalog is empty")
	}
	if cat.Version < 1 {
		return fmt.Errorf("seed: unsupported catalog version %d", cat.Version)
	}

	// Safety cap: refuse to seed more than 1000 flowers.
	if len(cat.Flowers) > 1000 {
		return fmt.Errorf("seed: catalog contains %d flowers (max 1000)", len(cat.Flowers))
	}

	count, err := d.Count()
	if err != nil {
		return fmt.Errorf("count: %w", err)
	}

	if count == 0 {
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		for i := range cat.Flowers {
			if err := insertFlowerTx(tx, &cat.Flowers[i]); err != nil {
				return fmt.Errorf("seed %s: %w", cat.Flowers[i].Name, err)
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return d.SetMeta("catalog_version", fmt.Sprint(cat.Version))
	}

	// Existing DB: apply catalog upgrade if the embedded catalog is newer.
	applied := 0
	if v := d.GetMeta("catalog_version"); v != "" {
		fmt.Sscanf(v, "%d", &applied)
	}
	if cat.Version <= applied {
		return nil
	}

	updated, inserted, skipped := 0, 0, 0
	for i := range cat.Flowers {
		f := &cat.Flowers[i]
		var existingID, customized int
		err := d.QueryRow("SELECT id, customized FROM flowers WHERE name = ?", f.Name).Scan(&existingID, &customized)
		switch {
		case err == sql.ErrNoRows:
			if err := d.insertFlower(f); err != nil {
				return fmt.Errorf("upgrade insert %s: %w", f.Name, err)
			}
			inserted++
		case err != nil:
			return fmt.Errorf("upgrade lookup %s: %w", f.Name, err)
		case customized != 0:
			skipped++
		default:
			f.ID = existingID
			if err := d.updateFlower(f); err != nil {
				return fmt.Errorf("upgrade update %s: %w", f.Name, err)
			}
			updated++
		}
	}
	if err := d.SetMeta("catalog_version", fmt.Sprint(cat.Version)); err != nil {
		return err
	}
	if updated+inserted+skipped > 0 {
		fmt.Printf("poterie: catalog upgraded to v%d (%d updated, %d new, %d customized skipped)\n",
			cat.Version, updated, inserted, skipped)
	}
	return nil
}

// insertFlower inserts a flower outside an explicit transaction.
func (d *DB) insertFlower(f *models.Flower) error {
	_, err := d.Exec(insertFlowerSQL,
		f.Name, f.BotanicalName, f.Role, f.ZoneMin, f.ZoneMax, f.Sun, f.SoilsCSV(),
		f.Color, f.BloomSeason, f.Height, f.Spacing, f.Description, f.ImageURL,
		f.ImageCredit, f.ImageLicense, f.ImageCreditURL, f.WikipediaURL)
	return err
}

// updateFlower overwrites all catalog-managed fields by ID. customized is
// deliberately NOT touched — this path only runs for customized=0 rows.
func (d *DB) updateFlower(f *models.Flower) error {
	_, err := d.Exec(`UPDATE flowers SET name=?, botanical_name=?, role=?, zone_min=?, zone_max=?,
		sun=?, soils=?, color=?, bloom_season=?, height=?, spacing=?, description=?, image_url=?,
		image_credit=?, image_license=?, image_credit_url=?, wikipedia_url=?
		WHERE id=?`,
		f.Name, f.BotanicalName, f.Role, f.ZoneMin, f.ZoneMax, f.Sun, f.SoilsCSV(),
		f.Color, f.BloomSeason, f.Height, f.Spacing, f.Description, f.ImageURL,
		f.ImageCredit, f.ImageLicense, f.ImageCreditURL, f.WikipediaURL, f.ID)
	return err
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

	for i := range parsed {
		if err := insertFlowerTx(tx, &parsed[i]); err != nil {
			// Skip duplicates (by name) during import — allows partial re-imports
			if strings.Contains(err.Error(), "UNIQUE") {
				continue
			}
			return fmt.Errorf("seed %s: %w", parsed[i].Name, err)
		}
	}
	return tx.Commit()
}

// insertFlowerTx inserts a flower inside an existing transaction.
func insertFlowerTx(tx *sql.Tx, f *models.Flower) error {
	_, err := tx.Exec(insertFlowerSQL,
		f.Name, f.BotanicalName, f.Role, f.ZoneMin, f.ZoneMax, f.Sun, f.SoilsCSV(),
		f.Color, f.BloomSeason, f.Height, f.Spacing, f.Description, f.ImageURL,
		f.ImageCredit, f.ImageLicense, f.ImageCreditURL, f.WikipediaURL)
	return err
}
