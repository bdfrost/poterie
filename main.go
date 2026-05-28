package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"

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

	if err := database.Init(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	// Seed if empty
	if err := db.Seed(database); err != nil {
		log.Fatalf("failed to seed database: %v", err)
	}

	r := handler.NewRouter(database, cfg, assets)

	// Serve static files from embedded assets
	r.Handle("/static/*", http.FileServer(http.FS(assets)))

	// Favicon fallback — browsers always request /favicon.ico
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/favicon.svg", http.StatusMovedPermanently)
	})

	log.Printf("Poterie starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
