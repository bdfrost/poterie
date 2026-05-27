package config

import "os"

type Config struct {
	Port        string
	DBPath      string
	AdminUser   string
	AdminPass   string
	ExternalURL string
}

func New() *Config {
	return &Config{
		Port:        envOr("PORT", "8080"),
		DBPath:      envOr("DB_PATH", "/data/poterie.db"),
		AdminUser:   envOr("ADMIN_USER", "admin"),
		AdminPass:   envOr("ADMIN_PASS", "changeme"),
		ExternalURL: envOr("EXTERNAL_URL", ""),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
