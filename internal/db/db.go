package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/bdfrost/poterie/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// flowerColumns is the single column list used by all SELECT/INSERT/UPDATE
// statements — keep in sync with scanFlowerRaw and schema.sql.
const flowerColumns = `id, name, botanical_name, role, zone_min, zone_max, sun, soils,
	color, bloom_season, height, spacing, description, image_url,
	image_credit, image_license, image_credit_url, wikipedia_url, customized`

// insertFlowerSQL inserts all catalog-managed fields (not id, not customized).
const insertFlowerSQL = `INSERT INTO flowers (name, botanical_name, role, zone_min, zone_max, sun, soils,
	color, bloom_season, height, spacing, description, image_url,
	image_credit, image_license, image_credit_url, wikipedia_url)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

type DB struct {
	*sql.DB
}

func New(path string) (*DB, error) {
	d, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := d.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("pragma: %w", err)
	}
	return &DB{d}, nil
}

func (d *DB) Init() error {
	if _, err := d.Exec(schemaSQL); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	// Back-compat migrations for older databases (no-op when columns exist).
	d.Exec("ALTER TABLE flowers ADD COLUMN wikipedia_url TEXT NOT NULL DEFAULT ''")
	d.Exec("ALTER TABLE flowers ADD COLUMN image_credit TEXT NOT NULL DEFAULT ''")
	d.Exec("ALTER TABLE flowers ADD COLUMN image_license TEXT NOT NULL DEFAULT ''")
	d.Exec("ALTER TABLE flowers ADD COLUMN image_credit_url TEXT NOT NULL DEFAULT ''")
	d.Exec("ALTER TABLE flowers ADD COLUMN customized INTEGER NOT NULL DEFAULT 0")
	return nil
}

// GetMeta returns a meta value ("" when unset).
func (d *DB) GetMeta(key string) string {
	var v string
	d.QueryRow("SELECT value FROM meta WHERE key = ?", key).Scan(&v)
	return v
}

// SetMeta upserts a meta value.
func (d *DB) SetMeta(key, value string) error {
	_, err := d.Exec("INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}

func (d *DB) Count() (int, error) {
	var n int
	err := d.QueryRow("SELECT COUNT(*) FROM flowers").Scan(&n)
	return n, err
}

func (d *DB) FindByCriteria(zone int, sun models.SunType, role models.FSTRole) ([]models.Flower, error) {
	rows, err := d.Query(`SELECT `+flowerColumns+`
		FROM flowers WHERE ? BETWEEN zone_min AND zone_max AND sun = ? AND role = ?
		ORDER BY name`, zone, sun, role)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var flowers []models.Flower
	for rows.Next() {
		f, err := scanFlowerRow(rows)
		if err != nil {
			return nil, err
		}
		flowers = append(flowers, f)
	}
	return flowers, rows.Err()
}

func (d *DB) FindAll() ([]models.Flower, error) {
	rows, err := d.Query(`SELECT `+flowerColumns+`
		FROM flowers ORDER BY role, name`)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var flowers []models.Flower
	for rows.Next() {
		f, err := scanFlowerRow(rows)
		if err != nil {
			return nil, err
		}
		flowers = append(flowers, f)
	}
	return flowers, rows.Err()
}

func (d *DB) FindByRole(role models.FSTRole) ([]models.Flower, error) {
	rows, err := d.Query(`SELECT `+flowerColumns+`
		FROM flowers WHERE role = ? ORDER BY name`, role)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var flowers []models.Flower
	for rows.Next() {
		f, err := scanFlowerRow(rows)
		if err != nil {
			return nil, err
		}
		flowers = append(flowers, f)
	}
	return flowers, rows.Err()
}

func (d *DB) FindByID(id int) (*models.Flower, error) {
	row := d.QueryRow(`SELECT `+flowerColumns+`
		FROM flowers WHERE id = ?`, id)
	return scanFlowerRowPtr(row)
}

func (d *DB) Upsert(f *models.Flower) error {
	if f.ID == 0 {
		result, err := d.Exec(insertFlowerSQL,
			f.Name, f.BotanicalName, f.Role, f.ZoneMin, f.ZoneMax, f.Sun, soilsCSV(f.Soils),
			f.Color, f.BloomSeason, f.Height, f.Spacing, f.Description, f.ImageURL,
			f.ImageCredit, f.ImageLicense, f.ImageCreditURL, f.WikipediaURL)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err == nil {
			f.ID = int(id)
		}
		return nil
	}
	if err := d.updateFlower(f); err != nil {
		return err
	}
	// Rows written through the admin panel are user data — flag them so
	// catalog auto-upgrades never overwrite them.
	_, err := d.Exec("UPDATE flowers SET customized = 1 WHERE id = ?", f.ID)
	return err
}

func (d *DB) Delete(id int) error {
	_, err := d.Exec("DELETE FROM flowers WHERE id = ?", id)
	return err
}

// scanFlowerRow scans a row into a Flower
func scanFlowerRow(r interface{ Scan(...interface{}) error }) (models.Flower, error) {
	f, err := scanFlowerRaw(r)
	return f, err
}

func scanFlowerRowPtr(r interface{ Scan(...interface{}) error }) (*models.Flower, error) {
	f, err := scanFlowerRaw(r)
	return &f, err
}

func scanFlowerRaw(r interface{ Scan(...interface{}) error }) (models.Flower, error) {
	var f models.Flower
	var soilsStr string
	var customized int
	err := r.Scan(&f.ID, &f.Name, &f.BotanicalName, &f.Role, &f.ZoneMin, &f.ZoneMax,
		&f.Sun, &soilsStr, &f.Color, &f.BloomSeason, &f.Height, &f.Spacing,
		&f.Description, &f.ImageURL, &f.ImageCredit, &f.ImageLicense,
		&f.ImageCreditURL, &f.WikipediaURL, &customized)
	f.Customized = customized != 0
	if err != nil {
		return f, err
	}
	if soilsStr != "" {
		for _, s := range strings.Split(soilsStr, ",") {
			f.Soils = append(f.Soils, models.SoilType(strings.TrimSpace(s)))
		}
	}
	return f, nil
}

// soilsCSV converts a slice of SoilType to a comma-separated string
func soilsCSV(soils []models.SoilType) string {
	if len(soils) == 0 {
		return ""
	}
	parts := make([]string, len(soils))
	for i, s := range soils {
		parts[i] = string(s)
	}
	return strings.Join(parts, ",")
}
