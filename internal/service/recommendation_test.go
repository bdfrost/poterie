package service

import (
	"os"
	"testing"

	"github.com/bdfrost/poterie/internal/db"
	"github.com/bdfrost/poterie/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *RecommendationService {
	t.Helper()
	f, err := os.CreateTemp("", "poterie-test-*.db")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	f.Close()

	d, err := db.New(f.Name())
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })

	require.NoError(t, d.Init()) // create tables
	require.NoError(t, db.Seed(d))
	return NewRecommendationService(d)
}

func TestGenerate_ContainerFullSun(t *testing.T) {
	svc := newTestService(t)

	rec, err := svc.Generate(7, models.SunFullSun, "container", models.SoilLoam, models.PaletteNone)
	require.NoError(t, err)

	assert.Equal(t, "container", rec.LayoutType)
	assert.Equal(t, 7, rec.Zone)
	assert.NotNil(t, rec.Thriller)
	assert.Equal(t, models.SunFullSun, rec.Thriller.Sun)
	assert.True(t, models.IsValidZone(rec.Zone))
	assert.NotNil(t, rec.Notes)
}

func TestGenerate_BedPartShade(t *testing.T) {
	svc := newTestService(t)

	rec, err := svc.Generate(5, models.SunPartShade, "bed", models.SoilClay, models.PaletteNone)
	require.NoError(t, err)

	assert.Equal(t, "bed", rec.LayoutType)
	assert.NotNil(t, rec.Thriller)
	// Thriller should support part_shade
	assert.Equal(t, models.SunPartShade, rec.Thriller.Sun)
}

func TestGenerate_FullShadeZone4(t *testing.T) {
	svc := newTestService(t)

	rec, err := svc.Generate(4, models.SunFullShade, "container", models.SoilLoam, models.PaletteNone)
	require.NoError(t, err)

	assert.NotNil(t, rec.Thriller)
	// Should return a shade-tolerant thriller (Hosta or Japanese Painted Fern)
	assert.Equal(t, models.SunFullShade, rec.Thriller.Sun)
}

func TestGenerate_InvalidZone(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Generate(1, models.SunFullSun, "container", models.SoilLoam, models.PaletteNone)
	assert.Error(t, err)

	_, err = svc.Generate(11, models.SunFullSun, "container", models.SoilLoam, models.PaletteNone)
	assert.Error(t, err)
}

func TestGenerate_NoCompatibleFlowers(t *testing.T) {
	svc := newTestService(t)

	// Zone 10 with full_shade should have few options
	rec, err := svc.Generate(10, models.SunFullShade, "container", models.SoilLoam, models.PaletteNone)
	require.NoError(t, err)
	// May or may not find flowers — ensure no crash
	assert.NotNil(t, rec)
}

func TestGenerateContainer_GenerateBed(t *testing.T) {
	svc := newTestService(t)

	c, err := svc.GenerateContainer(7, models.SunFullSun, models.SoilLoam, models.PaletteNone)
	require.NoError(t, err)
	assert.Equal(t, "container", c.LayoutType)

	b, err := svc.GenerateBed(7, models.SunFullSun, models.SoilLoam, models.PaletteNone)
	require.NoError(t, err)
	assert.Equal(t, "bed", b.LayoutType)
}

func TestGetAllFlowers(t *testing.T) {
	svc := newTestService(t)

	flowers, err := svc.GetAllFlowers()
	require.NoError(t, err)
	assert.True(t, len(flowers) > 50, "expected >50 flowers, got %d", len(flowers))

	// Verify roles
	var thrillers, fillers, spillers int
	for _, f := range flowers {
		switch f.Role {
		case models.RoleThriller:
			thrillers++
		case models.RoleFiller:
			fillers++
		case models.RoleSpiller:
			spillers++
		}
	}
	assert.True(t, thrillers > 0)
	assert.True(t, fillers > 0)
	assert.True(t, spillers > 0)
}

func TestUpsertAndDelete(t *testing.T) {
	svc := newTestService(t)

	// Create new flower
	f := &models.Flower{
		Name:          "Test Flower",
		BotanicalName: "Testus florus",
		Role:          models.RoleFiller,
		ZoneMin:       5, ZoneMax: 8,
		Sun:           models.SunFullSun,
		Soils:         []models.SoilType{models.SoilLoam},
		Color:         "Blue",
		BloomSeason:   "summer",
		Height:        "12in",
		Spacing:       "8in",
		Description:   "Test description",
	}
	require.NoError(t, svc.UpsertFlower(f))
	assert.True(t, f.ID > 0)

	// Retrieve
	got, err := svc.GetFlowerByID(f.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Flower", got.Name)

	// Update
	got.Name = "Updated Flower"
	require.NoError(t, svc.UpsertFlower(got))
	got2, err := svc.GetFlowerByID(f.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Flower", got2.Name)

	// Delete
	require.NoError(t, svc.DeleteFlower(f.ID))
	_, err = svc.GetFlowerByID(f.ID)
	assert.Error(t, err) // Should fail after deletion
}

func TestFilterBySoil(t *testing.T) {
	// Test that soil filtering works — pick a zone/sun combo with known sandy supporters
	svc := newTestService(t)
	rec, err := svc.Generate(7, models.SunFullSun, "container", models.SoilLoam, models.PaletteNone)
	require.NoError(t, err)

	// All recommended flowers should support loam soil (our most common)
	if rec.Thriller != nil {
		hasLoam := false
		for _, s := range rec.Thriller.Soils {
			if s == models.SoilLoam {
				hasLoam = true
				break
			}
		}
		assert.True(t, hasLoam, "Thriller should support loam soil, got %+v", rec.Thriller.Soils)
	}
}

func TestFilterByColor(t *testing.T) {
	svc := newTestService(t)

	// Warm palette should only return flowers with red/orange/yellow/gold
	rec, err := svc.Generate(7, models.SunFullSun, "container", models.SoilLoam, models.PaletteWarm)
	require.NoError(t, err)

	if rec.Thriller != nil {
		assert.True(t, rec.Thriller.MatchesPalette(models.PaletteWarm),
			"Thriller %s should match warm palette, color=%s", rec.Thriller.Name, rec.Thriller.Color)
	}

	// Verify filler colors
	for _, f := range rec.Fillers {
		assert.True(t, f.MatchesPalette(models.PaletteWarm),
			"Filler %s should match warm palette, color=%s", f.Name, f.Color)
	}
}

func TestColorFallback(t *testing.T) {
	svc := newTestService(t)

	// PaletteJewel is restrictive — if no matches, should fallback to all
	rec, err := svc.Generate(7, models.SunFullSun, "container", models.SoilLoam, models.PaletteJewel)
	require.NoError(t, err)
	// Should not crash even if palette is empty
	assert.NotNil(t, rec)
}
