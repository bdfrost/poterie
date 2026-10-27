package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

type Config struct {
	Port       string
	DBPath     string
	AdminUser  string
	AdminPass  string
	ExternalURL string
	Version    string
	CSRFSecret string
}

// These are set via -ldflags at build time
var (
	Version   = "dev"
	GitCommit = "unknown"
)

func New() *Config {
	c := &Config{
		Port:        envOr("PORT", "8080"),
		DBPath:      envOr("DB_PATH", "/data/poterie.db"),
		AdminUser:   envOr("ADMIN_USER", "admin"),
		AdminPass:   envOr("ADMIN_PASS", "changeme"),
		ExternalURL: envOr("EXTERNAL_URL", ""),
		Version:     Version,
		CSRFSecret:  envOr("CSRF_SECRET", ""),
	}
	// Generate a CSRF secret at startup if not configured.
	// In production, set CSRF_SECRET via environment variable for persistence.
	if c.CSRFSecret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		c.CSRFSecret = hex.EncodeToString(b)
	}
	return c
}

// Validate checks that non-default credentials are in use.
// Returns an error if ADMIN_PASS is the default (preventing accidental
// production deployments with "changeme").
func (c *Config) Validate() error {
	if c.AdminPass == "changeme" {
		return fmt.Errorf("security: ADMIN_PASS must not be the default value " +
			"(set the ADMIN_PASS environment variable to a strong password)")
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
