package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidZone(t *testing.T) {
	assert.True(t, IsValidZone(3))
	assert.True(t, IsValidZone(7))
	assert.True(t, IsValidZone(10))
	assert.False(t, IsValidZone(2))
	assert.False(t, IsValidZone(11))
	assert.False(t, IsValidZone(0))
	assert.False(t, IsValidZone(-1))
}

func TestSoilsCSV(t *testing.T) {
	f := &Flower{Soils: []SoilType{SoilLoam, SoilSandy}}
	assert.Equal(t, "loam,sandy", f.SoilsCSV())

	f2 := &Flower{Soils: []SoilType{}}
	assert.Equal(t, "", f2.SoilsCSV())
}

func TestFSTRoleConstants(t *testing.T) {
	assert.Equal(t, FSTRole("thriller"), RoleThriller)
	assert.Equal(t, FSTRole("filler"), RoleFiller)
	assert.Equal(t, FSTRole("spiller"), RoleSpiller)
}

func TestSunTypeConstants(t *testing.T) {
	assert.Equal(t, SunType("full_sun"), SunFullSun)
	assert.Equal(t, SunType("part_shade"), SunPartShade)
	assert.Equal(t, SunType("full_shade"), SunFullShade)
}

func TestBloomSeasons(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"summer", []string{"summer"}},
		{"summer_fall", []string{"summer", "fall"}},
		{"spring_summer_fall", []string{"spring", "summer", "fall"}},
		{"all_season", []string{"spring", "summer", "fall"}},
		{"", []string{"spring", "summer", "fall"}},
		{"spring_fall", []string{"spring", "fall"}},
	}
	for _, c := range cases {
		f := &Flower{BloomSeason: c.in}
		assert.Equal(t, c.want, f.BloomSeasons(), "input %q", c.in)
	}
}

func TestBloomSeasonLabel(t *testing.T) {
	assert.Equal(t, "Year-round", (&Flower{BloomSeason: "all_season"}).BloomSeasonLabel())
	assert.Equal(t, "Summer–Fall", (&Flower{BloomSeason: "summer_fall"}).BloomSeasonLabel())
	assert.Equal(t, "Spring", (&Flower{BloomSeason: "spring"}).BloomSeasonLabel())
	assert.Equal(t, "Year-round", (&Flower{BloomSeason: "spring_summer_fall"}).BloomSeasonLabel())
}

func TestOverlapsSeason(t *testing.T) {
	thriller := &Flower{BloomSeason: "summer"}
	assert.True(t, thriller.OverlapsSeason(&Flower{BloomSeason: "summer_fall"}))
	assert.True(t, thriller.OverlapsSeason(&Flower{BloomSeason: "all_season"}))
	assert.False(t, thriller.OverlapsSeason(&Flower{BloomSeason: "spring"}))
	assert.False(t, thriller.OverlapsSeason(&Flower{BloomSeason: "spring_fall"}))
}
