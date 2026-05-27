package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaults(t *testing.T) {
	// Clear any env vars
	os.Unsetenv("PORT")
	os.Unsetenv("DB_PATH")
	os.Unsetenv("ADMIN_USER")
	os.Unsetenv("ADMIN_PASS")
	os.Unsetenv("EXTERNAL_URL")

	cfg := New()
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "/data/poterie.db", cfg.DBPath)
	assert.Equal(t, "admin", cfg.AdminUser)
	assert.Equal(t, "changeme", cfg.AdminPass)
	assert.Equal(t, "", cfg.ExternalURL)
}

func TestNewFromEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("DB_PATH", "/tmp/test.db")
	os.Setenv("ADMIN_USER", "bob")
	os.Setenv("ADMIN_PASS", "secret")
	os.Setenv("EXTERNAL_URL", "https://poterie.example.com")

	cfg := New()
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "/tmp/test.db", cfg.DBPath)
	assert.Equal(t, "bob", cfg.AdminUser)
	assert.Equal(t, "secret", cfg.AdminPass)
	assert.Equal(t, "https://poterie.example.com", cfg.ExternalURL)

	// Cleanup
	os.Unsetenv("PORT")
	os.Unsetenv("DB_PATH")
	os.Unsetenv("ADMIN_USER")
	os.Unsetenv("ADMIN_PASS")
	os.Unsetenv("EXTERNAL_URL")
}
