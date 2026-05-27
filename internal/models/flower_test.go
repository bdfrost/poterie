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
