package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bdfrost/poterie/internal/config"
	"github.com/bdfrost/poterie/internal/db"
	"github.com/bdfrost/poterie/internal/handler"
)

//go:embed templates/* templates/partials/* static/css/* static/js/* static/favicon.svg
var assets embed.FS

func main() {
	cfg := config.New()

	// Initialize database
	database, err := db.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	// Validate config before starting — rejects default ADMIN_PASS.
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config error: %v", err)
	}

	if err := database.Init(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	// Seed if empty
	if err := db.Seed(database); err != nil {
		log.Fatalf("failed to seed database: %v", err)
	}

	r := handler.NewRouter(database, cfg, assets)

	// Serve static files from embedded assets with path traversal protection
	r.Handle("/static/*", staticFileHandler(assets))

	// Favicon fallback — browsers always request /favicon.ico
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/favicon.svg", http.StatusMovedPermanently)
	})

	// Security headers applied to all responses. Wrap the mux at serve time —
	// chi panics on Use() after routes are registered.
	log.Printf("Poterie starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, securityHeaders(r)); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

// securityHeaders sets safe default response headers on every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

// allowedStaticExt holds the set of file extensions permitted under /static/.
var allowedStaticExt = map[string]bool{
	".css":   true,
	".js":    true,
	".svg":   true,
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".webp":  true,
	".ico":   true,
	".woff":  true,
	".woff2": true,
	".ttf":   true,
	".eot":   true,
}

// staticFileHandler serves files from an embed.FS with path traversal protection
// and extension allowlisting.
func staticFileHandler(assets embed.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Chi passes the full remainder after "/static/" as r.URL.Path.
		path := strings.TrimPrefix(r.URL.Path, "/static/")

		// Reject empty paths
		if path == "" {
			http.NotFound(w, r)
			return
		}

		// Reject path traversal
		if strings.Contains(path, "..") {
			http.NotFound(w, r)
			return
		}

		// Reject absolute paths
		if strings.HasPrefix(path, "/") {
			http.NotFound(w, r)
			return
		}

		// Reject backslashes (Windows-style traversal)
		if strings.Contains(path, "\\") {
			http.NotFound(w, r)
			return
		}

		// Extension allowlist
		ext := strings.ToLower(filepath.Ext(path))
		if !allowedStaticExt[ext] {
			http.NotFound(w, r)
			return
		}

		// Open from embed.FS — this validates the path against the embed sandbox.
		// Embed patterns include the static/ prefix, so re-add it here.
		f, err := assets.Open("static/" + path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}

		// Set content type
		w.Header().Set("Content-Type", mime.TypeByExtension(ext))
		w.Header().Set("Cache-Control", "public, max-age=3600")

		// Read file body
		data, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
}
