package service

import (
	"fmt"
	"math/rand"

	"github.com/bdfrost/poterie/internal/db"
	"github.com/bdfrost/poterie/internal/models"
)

// RecommendationService generates FST recommendations
type RecommendationService struct {
	db *db.DB
}

func NewRecommendationService(database *db.DB) *RecommendationService {
	return &RecommendationService{db: database}
}

// Generate creates an FST recommendation based on user criteria
func (r *RecommendationService) Generate(zone int, sun models.SunType, layoutType string, soil models.SoilType) (*models.FSTRecommendation, error) {
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
	if len(thrillers) == 0 {
		rec.Notes = "No thrillers found for your zone/sun combo. Try adjusting your selections."
		return rec, nil
	}
	rec.Thriller = &thrillers[rand.Intn(len(thrillers))]

	// Find compatible fillers
	fillers, err := r.db.FindByCriteria(zone, sun, models.RoleFiller)
	if err != nil {
		return nil, err
	}
	fillers = filterBySoil(fillers, soil)
	rec.Fillers = pickMultiple(fillers, 3)

	// Find compatible spillers
	spillers, err := r.db.FindByCriteria(zone, sun, models.RoleSpiller)
	if err != nil {
		return nil, err
	}
	spillers = filterBySoil(spillers, soil)
	rec.Spillers = pickMultiple(spillers, 2)

	rec.Notes = generateNotes(rec)
	return rec, nil
}

// GenerateBed creates a bed-style FST recommendation (more fillers)
func (r *RecommendationService) GenerateBed(zone int, sun models.SunType, soil models.SoilType) (*models.FSTRecommendation, error) {
	return r.Generate(zone, sun, "bed", soil)
}

// GenerateContainer creates a container-style FST recommendation
func (r *RecommendationService) GenerateContainer(zone int, sun models.SunType, soil models.SoilType) (*models.FSTRecommendation, error) {
	return r.Generate(zone, sun, "container", soil)
}

// filterBySoil returns flowers that support the given soil type
func filterBySoil(flowers []models.Flower, soil models.SoilType) []models.Flower {
	var result []models.Flower
	for _, f := range flowers {
		for _, s := range f.Soils {
			if s == soil {
				result = append(result, f)
				break
			}
		}
	}
	return result
}

// pickMultiple randomly selects n items (or all if fewer available)
func pickMultiple(flowers []models.Flower, n int) []models.Flower {
	if len(flowers) == 0 {
		return nil
	}
	if len(flowers) <= n {
		return flowers
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
		return "Try adjusting your zone or sun selection for more options."
	}
	return fmt.Sprintf("This %s combination features %s as your dramatic centerpiece, with %d filler(s) to add body and color, and %d spiller(s) to cascade over the edges.",
		rec.LayoutType, rec.Thriller.Name, len(rec.Fillers), len(rec.Spillers))
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
