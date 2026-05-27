package handler

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"

	"github.com/bdfrost/poterie/internal/config"
	"github.com/bdfrost/poterie/internal/db"
	"github.com/bdfrost/poterie/internal/models"
	"github.com/bdfrost/poterie/internal/service"
	"github.com/go-chi/chi/v5"
	chilog "github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	db      *db.DB
	cfg     *config.Config
	service *service.RecommendationService
	assets  embed.FS
	templates *template.Template
	templatesOnce sync.Once
}

func NewRouter(database *db.DB, cfg *config.Config, assets embed.FS) *chi.Mux {
	h := &Handler{
		db:      database,
		cfg:     cfg,
		service: service.NewRecommendationService(database),
		assets:  assets,
	}

	r := chi.NewRouter()
	r.Use(chilog.Logger)
	r.Use(chilog.Recoverer)
	r.Use(chilog.RequestID)

	// Public pages
	r.Get("/", h.home)
	r.Get("/planner", h.planner)
	r.Post("/planner/recommend", h.recommend)
	r.Get("/guide", h.guide)

	// Admin pages (basic auth)
	r.Group(func(r chi.Router) {
		r.Use(adminAuth(cfg.AdminUser, cfg.AdminPass))
		r.Get("/admin", h.admin)
		r.Get("/admin/flowers", h.adminFlowers)
		r.Post("/admin/flowers/new", h.adminNewFlower)
		r.Post("/admin/flowers/{id}/delete", h.adminDeleteFlower)
	})

	// API endpoints for HTMX
	r.Post("/api/fst", h.apiFST)
	r.Get("/api/health", h.health)

	return r
}

func (h *Handler) initTemplates() {
	h.templatesOnce.Do(func() {
		h.templates = template.Must(template.New("").Funcs(template.FuncMap{
			"safeCSS": func(s string) template.CSS {
				return template.CSS(s)
			},
		}).ParseFS(h.assets,
			"templates/base.html",
			"templates/home.html",
			"templates/planner.html",
			"templates/result.html",
			"templates/guide.html",
			"templates/admin.html",
			"templates/partials/*.html",
		))
	})
}

func (h *Handler) render(w http.ResponseWriter, name string, data map[string]interface{}) {
	h.initTemplates()
	data["Title"] = "FST Garden Planner"
	if title, ok := data["PageTitle"].(string); ok {
		data["Title"] = title
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func adminAuth(user, pass string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok || u != user || p != pass {
				w.Header().Set("WWW-Authenticate", `Basic realm="Admin"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// Page: Home
func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	h.render(w, "home.html", map[string]interface{}{
		"PageTitle": "Design Your Perfect Flower Arrangement",
	})
}

// Page: Planner
func (h *Handler) planner(w http.ResponseWriter, r *http.Request) {
	h.render(w, "planner.html", map[string]interface{}{
		"PageTitle": "FST Planner",
		"Zones":     []int{3, 4, 5, 6, 7, 8, 9, 10},
		"SunTypes": []struct{ Value, Label string }{
			{"full_sun", "Full Sun (6+ hours direct)"},
			{"part_shade", "Part Shade (3-6 hours)"},
			{"full_shade", "Full Shade (<3 hours)"},
		},
		"SoilTypes": []struct{ Value, Label string }{
			{"clay", "Clay"},
			{"loam", "Loam (ideal)"},
			{"sandy", "Sandy"},
			{"well_drained", "Well-Drained"},
		},
	})
}

// POST /planner/recommend - full page redirect to results
func (h *Handler) recommend(w http.ResponseWriter, r *http.Request) {
	zone := parseInt(r.FormValue("zone"), 5)
	layoutType := r.FormValue("layout")
	sun := models.SunType(r.FormValue("sun"))
	soil := models.SoilType(r.FormValue("soil"))

	if !models.IsValidZone(zone) {
		http.Error(w, "Invalid zone (3-10)", http.StatusBadRequest)
		return
	}

	rec, err := h.service.Generate(zone, sun, layoutType, soil)
	if err != nil {
		http.Error(w, "Failed to generate recommendation", http.StatusInternalServerError)
		return
	}

	h.render(w, "result.html", map[string]interface{}{
		"PageTitle": "Your FST Arrangement",
		"Rec":       rec,
		"Zone":      zone,
		"Sun":       string(sun),
		"Soil":      string(soil),
		"Layout":    layoutType,
	})
}

// POST /api/fst - HTMX endpoint returning just the result partial
func (h *Handler) apiFST(w http.ResponseWriter, r *http.Request) {
	zone := parseInt(r.FormValue("zone"), 5)
	layoutType := r.FormValue("layout")
	sun := models.SunType(r.FormValue("sun"))
	soil := models.SoilType(r.FormValue("soil"))

	rec, err := h.service.Generate(zone, sun, layoutType, soil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.initTemplates()
	if err := h.templates.ExecuteTemplate(w, "partials/result-card", map[string]interface{}{
		"Rec":    rec,
		"Zone":   zone,
		"Sun":    string(sun),
		"Soil":   string(soil),
		"Layout": layoutType,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Page: FST Guide
func (h *Handler) guide(w http.ResponseWriter, r *http.Request) {
	h.render(w, "guide.html", map[string]interface{}{
		"PageTitle": "FST Guide - Filler, Spiller, Thriller",
	})
}

// Page: Admin Dashboard
func (h *Handler) admin(w http.ResponseWriter, r *http.Request) {
	flowers, err := h.service.GetAllFlowers()
	if err != nil {
		http.Error(w, "Failed to load flowers", http.StatusInternalServerError)
		return
	}

	counts := map[string]int{}
	for _, f := range flowers {
		counts[string(f.Role)]++
	}

	h.render(w, "admin.html", map[string]interface{}{
		"PageTitle": "Admin Dashboard",
		"Flowers":   flowers,
		"Counts":    counts,
		"Total":     len(flowers),
	})
}

func (h *Handler) adminFlowers(w http.ResponseWriter, r *http.Request) {
	flowers, err := h.service.GetAllFlowers()
	if err != nil {
		http.Error(w, "Failed to load flowers", http.StatusInternalServerError)
		return
	}
	h.render(w, "admin.html", map[string]interface{}{
		"PageTitle": "Admin - All Flowers",
		"Flowers":   flowers,
	})
}

func (h *Handler) adminNewFlower(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	f := &models.Flower{
		Name:          r.FormValue("name"),
		BotanicalName: r.FormValue("botanical_name"),
		Role:          models.FSTRole(r.FormValue("role")),
		ZoneMin:       parseInt(r.FormValue("zone_min"), 3),
		ZoneMax:       parseInt(r.FormValue("zone_max"), 10),
		Sun:           models.SunType(r.FormValue("sun")),
		Color:         r.FormValue("color"),
		BloomSeason:   r.FormValue("bloom_season"),
		Height:        r.FormValue("height"),
		Spacing:       r.FormValue("spacing"),
		Description:   r.FormValue("description"),
		ImageURL:      r.FormValue("image_url"),
	}

	// Parse comma-separated soils
	soilsStr := r.FormValue("soils")
	if soilsStr != "" {
		for _, s := range strings.Split(soilsStr, ",") {
			f.Soils = append(f.Soils, models.SoilType(strings.TrimSpace(s)))
		}
	}

	if err := h.service.UpsertFlower(f); err != nil {
		http.Error(w, "Failed to save flower", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) adminDeleteFlower(w http.ResponseWriter, r *http.Request) {
	id := parseInt(chi.URLParam(r, "id"), 0)
	if id == 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteFlower(id); err != nil {
		http.Error(w, "Failed to delete flower", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func parseInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return fallback
	}
	return n
}
