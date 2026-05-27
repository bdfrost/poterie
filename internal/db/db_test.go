package db

import (
	"os"
	"testing"

	"github.com/bdfrost/poterie/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	f, err := os.CreateTemp("", "poterie-db-*.db")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	f.Close()

	d, err := New(f.Name())
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	require.NoError(t, d.Init())
	return d
}

func TestNewAndInit(t *testing.T) {
	d := newTestDB(t)
	count, err := d.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSeed(t *testing.T) {
	d := newTestDB(t)

	err := Seed(d)
	require.NoError(t, err)

	count, err := d.Count()
	require.NoError(t, err)
	assert.True(t, count > 50, "expected >50 seeded flowers, got %d", count)

	// Seed is idempotent
	err = Seed(d)
	require.NoError(t, err)

	count2, err := d.Count()
	require.NoError(t, err)
	assert.Equal(t, count, count2, "second seed should be no-op")
}

func TestFindByCriteria(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, Seed(d))

	flowers, err := d.FindByCriteria(7, models.SunFullSun, models.RoleThriller)
	require.NoError(t, err)
	assert.True(t, len(flowers) > 0)

	// All returned thrillers should be compatible with zone 7, full sun
	for _, f := range flowers {
		assert.Equal(t, models.RoleThriller, f.Role)
		assert.LessOrEqual(t, f.ZoneMin, 7)
		assert.GreaterOrEqual(t, f.ZoneMax, 7)
		assert.Equal(t, models.SunFullSun, f.Sun)
	}
}

func TestFindByRole(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, Seed(d))

	fillers, err := d.FindByRole(models.RoleFiller)
	require.NoError(t, err)
	assert.True(t, len(fillers) > 0)
	for _, f := range fillers {
		assert.Equal(t, models.RoleFiller, f.Role)
	}
}

func TestFindAll(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, Seed(d))

	all, err := d.FindAll()
	require.NoError(t, err)
	count, _ := d.Count()
	assert.Equal(t, count, len(all))
}

func TestFindByID(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, Seed(d))

	// Find a real flower
	all, _ := d.FindAll()
	require.True(t, len(all) > 0)

	f, err := d.FindByID(all[0].ID)
	require.NoError(t, err)
	assert.Equal(t, all[0].Name, f.Name)

	// Non-existent ID
	_, err = d.FindByID(999999)
	assert.Error(t, err)
}

func TestUpsertAndDelete(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, Seed(d))

	f := &models.Flower{
		Name: "Test", BotanicalName: "Testus", Role: models.RoleFiller,
		ZoneMin: 5, ZoneMax: 8, Sun: models.SunFullSun,
		Soils: []models.SoilType{models.SoilLoam},
	}
	require.NoError(t, d.Upsert(f))
	assert.True(t, f.ID > 0)

	// Verify
	got, err := d.FindByID(f.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test", got.Name)

	// Update
	got.Name = "Updated"
	require.NoError(t, d.Upsert(got))
	got2, err := d.FindByID(f.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got2.Name)

	// Delete
	require.NoError(t, d.Delete(f.ID))
	_, err = d.FindByID(f.ID)
	assert.Error(t, err)
}
