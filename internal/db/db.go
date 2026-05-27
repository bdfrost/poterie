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
	return nil
}

func (d *DB) Count() (int, error) {
	var n int
	err := d.QueryRow("SELECT COUNT(*) FROM flowers").Scan(&n)
	return n, err
}

func (d *DB) FindByCriteria(zone int, sun models.SunType, role models.FSTRole) ([]models.Flower, error) {
	rows, err := d.Query(`SELECT id, name, botanical_name, role, zone_min, zone_max, sun, soils,
		color, bloom_season, height, spacing, description, image_url
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
	rows, err := d.Query(`SELECT id, name, botanical_name, role, zone_min, zone_max, sun, soils,
		color, bloom_season, height, spacing, description, image_url
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
	rows, err := d.Query(`SELECT id, name, botanical_name, role, zone_min, zone_max, sun, soils,
		color, bloom_season, height, spacing, description, image_url
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
	row := d.QueryRow(`SELECT id, name, botanical_name, role, zone_min, zone_max, sun, soils,
		color, bloom_season, height, spacing, description, image_url
		FROM flowers WHERE id = ?`, id)
	return scanFlowerRowPtr(row)
}

func (d *DB) Upsert(f *models.Flower) error {
	soils := soilsCSV(f.Soils)
	if f.ID == 0 {
		result, err := d.Exec(`INSERT INTO flowers (name, botanical_name, role, zone_min, zone_max, sun, soils,
			color, bloom_season, height, spacing, description, image_url)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			f.Name, f.BotanicalName, f.Role, f.ZoneMin, f.ZoneMax, f.Sun, soils,
			f.Color, f.BloomSeason, f.Height, f.Spacing, f.Description, f.ImageURL)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err == nil {
			f.ID = int(id)
		}
		return nil
	}
	_, err := d.Exec(`UPDATE flowers SET name=?, botanical_name=?, role=?, zone_min=?, zone_max=?,
		sun=?, soils=?, color=?, bloom_season=?, height=?, spacing=?, description=?, image_url=?
		WHERE id=?`,
		f.Name, f.BotanicalName, f.Role, f.ZoneMin, f.ZoneMax, f.Sun, soils,
		f.Color, f.BloomSeason, f.Height, f.Spacing, f.Description, f.ImageURL, f.ID)
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
	err := r.Scan(&f.ID, &f.Name, &f.BotanicalName, &f.Role, &f.ZoneMin, &f.ZoneMax,
		&f.Sun, &soilsStr, &f.Color, &f.BloomSeason, &f.Height, &f.Spacing,
		&f.Description, &f.ImageURL)
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
