package service

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/bdfrost/poterie/internal/db"
	"github.com/bdfrost/poterie/internal/models"
)

func init() {
	// Go <1.20 does not auto-seed; harmless no-op on 1.20+.
	rand.Seed(time.Now().UnixNano())
}

// RecommendationService generates FST recommendations
type RecommendationService struct {
	db *db.DB
}

func NewRecommendationService(database *db.DB) *RecommendationService {
	return &RecommendationService{db: database}
}

// Generate creates an FST recommendation based on user criteria
func (r *RecommendationService) Generate(zone int, sun models.SunType, layoutType string, soil models.SoilType, palette models.ColorPalette) (*models.FSTRecommendation, error) {
	if !models.IsValidZone(zone) {
		return nil, fmt.Errorf("zone %d is out of range (3-10)", zone)
	}

	rec := &models.FSTRecommendation{
		LayoutType: layoutType,
		Zone:       zone,
		Sun:        sun,
		Soils:      []models.SoilType{soil},
	}

	// Find compatible thrillers
	thrillers, err := r.db.FindByCriteria(zone, sun, models.RoleThriller)
	if err != nil {
		return nil, err
	}
	thrillers = filterByColor(thrillers, palette)
	if len(thrillers) == 0 {
		rec.Notes = fmt.Sprintf("No thrillers found for your zone/sun/color combo. %sTry adjusting your selections.", noteForPalette(palette))
		return rec, nil
	}
	rec.Thriller = &thrillers[rand.Intn(len(thrillers))]

	// Find compatible fillers
	fillers, err := r.db.FindByCriteria(zone, sun, models.RoleFiller)
	if err != nil {
		return nil, err
	}
	fillers = filterBySoil(fillers, soil)
	fillers = filterByColor(fillers, palette)
	// Prefer fillers whose bloom season overlaps the thriller's, so the
	// arrangement actually flowers together. Falls back to the full pool
	// when the overlap is too small to fill the slots.
	fillers = preferSeasonOverlap(fillers, rec.Thriller, 3)
	rec.Fillers = pickMultiple(fillers, 3)

	// Find compatible spillers
	spillers, err := r.db.FindByCriteria(zone, sun, models.RoleSpiller)
	if err != nil {
		return nil, err
	}
	spillers = filterBySoil(spillers, soil)
	spillers = filterByColor(spillers, palette)
	spillers = preferSeasonOverlap(spillers, rec.Thriller, 2)
	rec.Spillers = pickMultiple(spillers, 2)

	rec.Notes = generateNotes(rec)
	return rec, nil
}

// GenerateBed creates a bed-style FST recommendation (more fillers)
func (r *RecommendationService) GenerateBed(zone int, sun models.SunType, soil models.SoilType, palette models.ColorPalette) (*models.FSTRecommendation, error) {
	return r.Generate(zone, sun, "bed", soil, palette)
}

// GenerateContainer creates a container-style FST recommendation
func (r *RecommendationService) GenerateContainer(zone int, sun models.SunType, soil models.SoilType, palette models.ColorPalette) (*models.FSTRecommendation, error) {
	return r.Generate(zone, sun, "container", soil, palette)
}

// filterByColor returns flowers whose color matches the given palette.
func filterByColor(flowers []models.Flower, palette models.ColorPalette) []models.Flower {
	if palette == models.PaletteNone {
		return flowers
	}
	var filtered []models.Flower
	for _, f := range flowers {
		if f.MatchesPalette(palette) {
			filtered = append(filtered, f)
		}
	}
	if len(filtered) == 0 {
		return flowers // fallback: return all if palette is too restrictive
	}
	return filtered
}

// noteForPalette returns a helpful suggestion when no matches found.
func noteForPalette(p models.ColorPalette) string {
	if p != models.PaletteNone {
		return "Try a different color theme, or "
	}
	return "Try adjusting your zone or sun selection, or "
}

// filterBySoil returns flowers that support the given soil type.
// Loam is treated as universally compatible (most garden plants tolerate it).
// Uses a two-pass approach to avoid duplicate entries:
//   1. First pass: find exact soil matches
//   2. Second pass (only if exact is empty): include loam-tolerant plants
//   3. Fallback: return all candidates if still no matches
func filterBySoil(flowers []models.Flower, soil models.SoilType) []models.Flower {
	// Pass 1: exact soil match only
	var exact []models.Flower
	for _, f := range flowers {
		for _, s := range f.Soils {
			if s == soil {
				exact = append(exact, f)
				break
			}
		}
	}
	if len(exact) > 0 {
		return exact
	}

	// Pass 2: loam-tolerant plants (only if no exact matches)
	if soil != models.SoilLoam {
		var loamTolerant []models.Flower
		for _, f := range flowers {
			for _, s := range f.Soils {
				if s == models.SoilLoam {
					loamTolerant = append(loamTolerant, f)
					break
				}
			}
		}
		if len(loamTolerant) > 0 {
			return loamTolerant
		}
	}

	// Fallback: return all candidates if nothing matched
	return flowers
}

// preferSeasonOverlap narrows candidates to flowers whose bloom season
// overlaps the thriller's. Returns the narrowed pool only if it can fill n
// slots; otherwise returns the original pool (graceful degradation — an
// arrangement that blooms in sequence beats no arrangement).
func preferSeasonOverlap(flowers []models.Flower, thriller *models.Flower, n int) []models.Flower {
	if thriller == nil || len(flowers) <= n {
		return flowers
	}
	var overlap []models.Flower
	for i := range flowers {
		if flowers[i].OverlapsSeason(thriller) {
			overlap = append(overlap, flowers[i])
		}
	}
	if len(overlap) >= n {
		return overlap
	}
	return flowers
}

// pickMultiple randomly selects n items.
// When the candidate pool is too small (e.g., from strict color filtering),
// it duplicates randomly from the available set to always return n items.
func pickMultiple(flowers []models.Flower, n int) []models.Flower {
	if len(flowers) == 0 {
		return nil
	}
	if len(flowers) <= n {
		// Pool is smaller than needed — fill with duplicates
		result := make([]models.Flower, n)
		copy(result, flowers)
		for i := len(flowers); i < n; i++ {
			result[i] = flowers[rand.Intn(len(flowers))]
		}
		// Shuffle the result so duplicates aren't all at the end
		perm := rand.Perm(n)
		shuffled := make([]models.Flower, n)
		for i := 0; i < n; i++ {
			shuffled[i] = result[perm[i]]
		}
		return shuffled
	}
	// Shuffle and pick
	perm := rand.Perm(len(flowers))
	result := make([]models.Flower, n)
	for i := 0; i < n; i++ {
		result[i] = flowers[perm[i]]
	}
	return result
}

func generateNotes(rec *models.FSTRecommendation) string {
	if rec.Thriller == nil {
		return "Try adjusting your zone, sun, or color selection for more options."
	}
	notes := fmt.Sprintf("This %s combination features %s as your dramatic centerpiece, with %d filler(s) to add body and color, and %d spiller(s) to cascade over the edges.",
		rec.LayoutType, rec.Thriller.Name, len(rec.Fillers), len(rec.Spillers))

	// Bloom-season summary: the union of seasons across all picks.
	present := map[string]bool{}
	all := append(append([]models.Flower{*rec.Thriller}, rec.Fillers...), rec.Spillers...)
	for _, f := range all {
		for _, s := range f.BloomSeasons() {
			present[s] = true
		}
	}
	var seasons []string
	for _, s := range []string{"spring", "summer", "fall"} {
		if present[s] {
			seasons = append(seasons, s)
		}
	}
	if len(seasons) > 0 {
		notes += fmt.Sprintf(" Expect blooms %s.", bloomPhrase(seasons))
	}
	// Warn when a pick doesn't overlap the thriller (succession planting).
	for _, f := range all[1:] {
		if !f.OverlapsSeason(rec.Thriller) {
			notes += fmt.Sprintf(" Note: %s (%s) blooms outside the thriller's season — it'll carry the pot when %s rests.",
				f.Name, f.BloomSeasonLabel(), rec.Thriller.Name)
			break
		}
	}
	return notes
}

// bloomPhrase renders ["spring","summer"] as "from spring through summer".
func bloomPhrase(seasons []string) string {
	if len(seasons) == 3 {
		return "nearly year-round"
	}
	if len(seasons) == 1 {
		return "in " + seasons[0]
	}
	return "from " + seasons[0] + " through " + seasons[len(seasons)-1]
}

// GetAllFlowers returns all flowers in the database (for admin)
func (r *RecommendationService) GetAllFlowers() ([]models.Flower, error) {
	return r.db.FindAll()
}

// GetFlowerByID returns a single flower
func (r *RecommendationService) GetFlowerByID(id int) (*models.Flower, error) {
	return r.db.FindByID(id)
}

// UpsertFlower creates or updates a flower
func (r *RecommendationService) UpsertFlower(f *models.Flower) error {
	return r.db.Upsert(f)
}

// DeleteFlower removes a flower
func (r *RecommendationService) DeleteFlower(id int) error {
	return r.db.Delete(id)
}

// GetAlternatives returns compatible flowers for a given role/filter combo
// (for the dropdown swap feature in the visual layout).
func (r *RecommendationService) GetAlternatives(zone int, sun models.SunType, soil models.SoilType, role models.FSTRole, palette models.ColorPalette) ([]models.Flower, error) {
	flowers, err := r.db.FindByCriteria(zone, sun, role)
	if err != nil {
		return nil, err
	}
	if soil != "" {
		flowers = filterBySoil(flowers, soil)
	}
	flowers = filterByColor(flowers, palette)
	return flowers, nil
}

// ExportCatalogJSON returns the embedded catalog as raw JSON bytes.
func (r *RecommendationService) ExportCatalogJSON() ([]byte, error) {
	return db.ExportCatalogJSON(), nil
}

// ImportCatalogJSON seeds the database from a custom catalog.
func (r *RecommendationService) ImportCatalogJSON(data []byte) error {
	return db.SeedFromJSON(r.db, data)
}

// GetCatalogInfo returns metadata about the embedded catalog.
func (r *RecommendationService) GetCatalogInfo() (*db.Catalog, error) {
	return db.LoadCatalog()
}
