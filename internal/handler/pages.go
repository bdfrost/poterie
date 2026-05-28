package handler

import (
	"embed"
	"fmt"
	"html/template"
	"io"
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
	// preloaded templates: base + partials (no page-specific content)
	baseTemplates *template.Template
	tmplMu        sync.Mutex
}

func NewRouter(database *db.DB, cfg *config.Config, assets embed.FS) *chi.Mux {
	h := &Handler{
		db:      database,
		cfg:     cfg,
		service: service.NewRecommendationService(database),
		assets:  assets,
	}

	// Initialize templates eagerly — crash at startup if broken, not on first request.
	// This avoids the sync.Once trap where a transient ParseFS failure leaves the
	// handler permanently broken (Once won't retry, nil template causes 500s forever).
	h.loadTemplates()

	r := chi.NewRouter()
	r.Use(chilog.Logger)
	r.Use(chilog.Recoverer)
	r.Use(chilog.RequestID)

	// Public pages
	r.Get("/", h.home)
	r.Get("/planner", h.planner)
	r.Post("/planner/recommend", h.recommend)
	r.Post("/planner/print", h.printPlan)
	r.Get("/planner/print", h.printPlan)
	r.Get("/guide", h.guide)

	// Admin pages (basic auth)
	r.Group(func(r chi.Router) {
		r.Use(adminAuth(cfg.AdminUser, cfg.AdminPass))
		r.Get("/admin", h.admin)
		r.Get("/admin/flowers", h.adminFlowers)
		r.Get("/admin/flowers/{id}/edit", h.adminEditFlower)
		r.Post("/admin/flowers/{id}/update", h.adminUpdateFlower)
		r.Post("/admin/flowers/new", h.adminNewFlower)
		r.Post("/admin/flowers/{id}/delete", h.adminDeleteFlower)
		// Catalog management
		r.Get("/admin/catalog/export", h.adminCatalogExport)
		r.Post("/admin/catalog/import", h.adminCatalogImport)
	})

	// API endpoints for HTMX
	r.Post("/api/fst", h.apiFST)
	r.Post("/api/fst/swap", h.apiFSTSwap)
	r.Get("/api/alternatives", h.apiAlternatives)
	r.Get("/api/health", h.health)

	return r
}

func (h *Handler) loadTemplates() {
	h.tmplMu.Lock()
	defer h.tmplMu.Unlock()
	if h.baseTemplates != nil {
		return
	}
	h.baseTemplates = template.Must(template.New("").Funcs(template.FuncMap{
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"merge": func(maps ...map[string]int) map[string]int {
			result := make(map[string]int)
			for _, m := range maps {
				for k, v := range m {
					result[k] += v
				}
			}
			return result
		},
		"list": func(items ...interface{}) []interface{} {
			return items
		},
		"intSlice": func(items ...int) []int {
			return items
		},
	}).ParseFS(h.assets,
		"templates/base.html",
		"templates/partials/*.html",
	))
}

func (h *Handler) render(w http.ResponseWriter, pagename string, data map[string]interface{}) {
	h.loadTemplates()
	if h.baseTemplates == nil {
		http.Error(w, "templates failed to initialize", http.StatusInternalServerError)
		return
	}
	data["Title"] = "Poterie"
	if title, ok := data["PageTitle"].(string); ok {
		data["Title"] = title
	}
	data["Version"] = h.cfg.Version

	// Clone the preloaded base+partials template set and add this page.
	// Parsing only one page at a time avoids the "content" namespace collision
	// that occurs when {{define "content"}} exists across multiple files loaded together.
	if h.baseTemplates == nil {
		http.Error(w, "templates not initialized", http.StatusInternalServerError)
		return
	}
	tmpl, err := h.baseTemplates.Clone()
	if err != nil {
		http.Error(w, "template clone error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Parse in just this page's content block
	pagePath := strings.TrimSuffix(pagename, ".html")
	_, err = tmpl.ParseFS(h.assets, fmt.Sprintf("templates/%s.html", pagePath))
	if err != nil {
		http.Error(w, "template parse error for "+pagename+": "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
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
		"Palettes": []struct{ Value, Label, Description string }{
			{"", "Any Color", "No color restriction"},
			{"warm", "Warm Sunset", "Reds, oranges, yellows, golds"},
			{"cool", "Cool & Calm", "Blues, purples, lavenders, whites"},
			{"pinks", "Romantic Pinks", "Pinks, roses, magentas"},
			{"monochrome", "Monochrome Green", "Foliage-focused, greens & silvers"},
			{"jewel", "Jewel Tones", "Deep purples, burgundies, golds, bronzes"},
		},
	})
}

// countPlants returns a map of plant ID -> quantity (handles duplicates from swap)
func countPlants(plants []models.Flower) map[int]int {
	counts := make(map[int]int)
	for _, p := range plants {
		counts[p.ID]++
	}
	return counts
}

// plantWithQty pairs a flower with its quantity for display
type plantWithQty struct {
	Flower models.Flower
	Qty    int
}

// deduplicatePlants returns unique plants with their quantities
func deduplicatePlants(plants []models.Flower) []plantWithQty {
	counts := countPlants(plants)
	if len(counts) == 0 {
		return nil
	}
	result := make([]plantWithQty, 0, len(counts))
	// Iterate in insertion order to keep consistent display
	seen := make(map[int]bool)
	for _, p := range plants {
		if !seen[p.ID] {
			seen[p.ID] = true
			result = append(result, plantWithQty{Flower: p, Qty: counts[p.ID]})
		}
	}
	return result
}

// POST /planner/recommend - full page redirect to results
func (h *Handler) recommend(w http.ResponseWriter, r *http.Request) {
	zone := parseInt(r.FormValue("zone"), 5)
	layoutType := r.FormValue("layout")
	sun := models.SunType(r.FormValue("sun"))
	soil := models.SoilType(r.FormValue("soil"))
	palette := models.ColorPalette(r.FormValue("palette"))

	if !models.IsValidZone(zone) {
		http.Error(w, "Invalid zone (3-10)", http.StatusBadRequest)
		return
	}

	rec, err := h.service.Generate(zone, sun, layoutType, soil, palette)
	if err != nil {
		http.Error(w, "Failed to generate recommendation", http.StatusInternalServerError)
		return
	}
	alters := map[string]interface{}{
		"Thrillers": fetchAlts(h.service, zone, sun, soil, models.RoleThriller, palette),
		"Fillers":   fetchAlts(h.service, zone, sun, soil, models.RoleFiller, palette),
		"Spillers":  fetchAlts(h.service, zone, sun, soil, models.RoleSpiller, palette),
	}

	h.render(w, "result.html", map[string]interface{}{
		"PageTitle":      "Your FST Arrangement",
		"Rec":            rec,
		"Zone":           zone,
		"Sun":            string(sun),
		"Soil":           string(soil),
		"Palette":        string(palette),
		"Layout":         layoutType,
		"Alternatives":   alters,
		"FillerCounts":   countPlants(rec.Fillers),
		"SpillerCounts":  countPlants(rec.Spillers),
		"FilledFillers":  deduplicatePlants(rec.Fillers),
		"FilledSpillers": deduplicatePlants(rec.Spillers),
	})
}

// POST /api/fst - HTMX endpoint returning just the result partial
func (h *Handler) apiFST(w http.ResponseWriter, r *http.Request) {
	zone := parseInt(r.FormValue("zone"), 5)
	layoutType := r.FormValue("layout")
	sun := models.SunType(r.FormValue("sun"))
	soil := models.SoilType(r.FormValue("soil"))
	palette := models.ColorPalette(r.FormValue("palette"))

	rec, err := h.service.Generate(zone, sun, layoutType, soil, palette)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.loadTemplates()
	// Parse the result-card partial into the base template clone for execution
	tmpl, err := h.baseTemplates.Clone()
	if err != nil {
		http.Error(w, "template clone error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "partials/result-card", map[string]interface{}{
		"Rec":            rec,
		"Zone":           zone,
		"Sun":            string(sun),
		"Soil":           string(soil),
		"Palette":        string(palette),
		"Layout":         layoutType,
		"FillerCounts":   countPlants(rec.Fillers),
		"SpillerCounts":  countPlants(rec.Spillers),
		"FilledFillers":  deduplicatePlants(rec.Fillers),
		"FilledSpillers": deduplicatePlants(rec.Spillers),
		"Alternatives": map[string]interface{}{
			"Thrillers": fetchAlts(h.service, zone, sun, soil, models.RoleThriller, palette),
			"Fillers":   fetchAlts(h.service, zone, sun, soil, models.RoleFiller, palette),
			"Spillers":  fetchAlts(h.service, zone, sun, soil, models.RoleSpiller, palette),
		},
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

// GET /admin/catalog/export - download the embedded catalog JSON
func (h *Handler) adminCatalogExport(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.ExportCatalogJSON()
	if err != nil {
		http.Error(w, "Failed to export catalog", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"catalog.json\"")
	w.Write(data)
}

// POST /admin/catalog/import - upload a custom catalog JSON
func (h *Handler) adminCatalogImport(w http.ResponseWriter, r *http.Request) {
	// Limit upload to 512KB
	r.Body = http.MaxBytesReader(w, r.Body, 512*1024)

	if err := r.ParseMultipartForm(512 * 1024); err != nil {
		http.Error(w, "Failed to parse upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("catalog")
	if err != nil {
		http.Error(w, "Failed to read file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 512*1024))
	if err != nil {
		http.Error(w, "Failed to read data: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.ImportCatalogJSON(data); err != nil {
		http.Error(w, "Import failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
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
		WikipediaURL:  r.FormValue("wikipedia_url"),
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

// GET /admin/flowers/{id}/edit - edit flower form
func (h *Handler) adminEditFlower(w http.ResponseWriter, r *http.Request) {
	id := parseInt(chi.URLParam(r, "id"), 0)
	if id == 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	f, err := h.service.GetFlowerByID(id)
	if err != nil {
		http.Error(w, "Flower not found", http.StatusNotFound)
		return
	}
	h.render(w, "admin_edit.html", map[string]interface{}{
		"PageTitle": "Edit Flower",
		"Flower":    f,
	})
}

// POST /admin/flowers/{id}/update - save flower changes
func (h *Handler) adminUpdateFlower(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	id := parseInt(chi.URLParam(r, "id"), 0)
	f := &models.Flower{
		ID:            id,
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
		WikipediaURL:  r.FormValue("wikipedia_url"),
	}

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

	http.Redirect(w, r, "/admin/flowers", http.StatusSeeOther)
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

// GET /api/alternatives - HTMX endpoint returning dropdown options for a plant role
func (h *Handler) apiAlternatives(w http.ResponseWriter, r *http.Request) {
	zone := parseInt(r.URL.Query().Get("zone"), 5)
	sun := models.SunType(r.URL.Query().Get("sun"))
	soil := models.SoilType(r.URL.Query().Get("soil"))
	role := models.FSTRole(r.URL.Query().Get("role"))
	currentID := parseInt(r.URL.Query().Get("currentId"), 0)
	palette := models.ColorPalette(r.URL.Query().Get("palette"))

	alternatives, err := h.service.GetAlternatives(zone, sun, soil, role, palette)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.loadTemplates()
	tmpl, err := h.baseTemplates.Clone()
	if err != nil {
		http.Error(w, "template clone error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "partials/alt-dropdown", map[string]interface{}{
		"Alternatives": alternatives,
		"CurrentID":    currentID,
		"Role":         string(role),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// POST /api/fst/swap - replace a plant in the current FST layout
func (h *Handler) apiFSTSwap(w http.ResponseWriter, r *http.Request) {
	zone := parseInt(r.FormValue("zone"), 5)
	sun := models.SunType(r.FormValue("sun"))
	soil := models.SoilType(r.FormValue("soil"))
	layout := r.FormValue("layout")
	role := models.FSTRole(r.FormValue("swap_role"))
	newID := parseInt(r.FormValue("swap_id"), 0)
	slotIdx := parseInt(r.FormValue("swap_slot"), 0)
	palette := models.ColorPalette(r.FormValue("palette"))

	if newID == 0 {
		http.Error(w, "missing plant selection", http.StatusBadRequest)
		return
	}

	// Fetch the newly selected plant
	newPlant, err := h.service.GetFlowerByID(newID)
	if err != nil {
		http.Error(w, "plant not found", http.StatusBadRequest)
		return
	}

	// Build the recommendation from the preserved plant IDs sent by the client.
	// This keeps all existing plants unchanged and only replaces the one the user picked.
	thrillerID := parseInt(r.FormValue("thriller_id"), 0)
	rec := &models.FSTRecommendation{
		LayoutType: layout,
		Zone:       zone,
		Sun:        sun,
		Soils:      []models.SoilType{soil},
		Notes:      "Custom arrangement",
	}
	// Load thriller
	if thrillerID > 0 {
		if f, err := h.service.GetFlowerByID(thrillerID); err == nil {
			rec.Thriller = f
		}
	}
	// Load fillers
	for i := 0; i < 3; i++ {
		fid := parseInt(r.FormValue(fmt.Sprintf("filler_%d", i)), 0)
		if fid > 0 {
			if f, err := h.service.GetFlowerByID(fid); err == nil {
				rec.Fillers = append(rec.Fillers, *f)
			}
		}
	}
	// Load spillers
	for i := 0; i < 2; i++ {
		fid := parseInt(r.FormValue(fmt.Sprintf("spiller_%d", i)), 0)
		if fid > 0 {
			if f, err := h.service.GetFlowerByID(fid); err == nil {
				rec.Spillers = append(rec.Spillers, *f)
			}
		}
	}
	// Generate a meaningful note based on actual plants
	if rec.Thriller != nil && (len(rec.Fillers) > 0 || len(rec.Spillers) > 0) {
		rec.Notes = fmt.Sprintf("Your custom %s features %s as the centerpiece, %d filler(s) for body, and %d spiller(s) for trailing edges.",
			layout, rec.Thriller.Name, len(rec.Fillers), len(rec.Spillers))
	}

	// Apply the swap — replace the selected plant at its slot
	switch role {
	case models.RoleThriller:
		rec.Thriller = newPlant
	case models.RoleFiller:
		if slotIdx < len(rec.Fillers) {
			rec.Fillers[slotIdx] = *newPlant
		}
	case models.RoleSpiller:
		if slotIdx < len(rec.Spillers) {
			rec.Spillers[slotIdx] = *newPlant
		}
	}

	h.loadTemplates()
	tmpl, err := h.baseTemplates.Clone()
	if err != nil {
		http.Error(w, "template clone error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "partials/result-card", map[string]interface{}{
		"Rec":            rec,
		"Zone":           zone,
		"Sun":            string(sun),
		"Soil":           string(soil),
		"Palette":        string(palette),
		"Layout":         layout,
		"FillerCounts":   countPlants(rec.Fillers),
		"SpillerCounts":  countPlants(rec.Spillers),
		"FilledFillers":  deduplicatePlants(rec.Fillers),
		"FilledSpillers": deduplicatePlants(rec.Spillers),
		"Alternatives": map[string]interface{}{
			"Thrillers": fetchAlts(h.service, zone, sun, soil, models.RoleThriller, palette),
			"Fillers":   fetchAlts(h.service, zone, sun, soil, models.RoleFiller, palette),
			"Spillers":  fetchAlts(h.service, zone, sun, soil, models.RoleSpiller, palette),
		},
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GET/POST /planner/print - printable shopping list page
// GET: regenerate from zone/sun/soil/layout params (bookmarkable)
// POST: build from current plant IDs in form fields (preserves swaps)
func (h *Handler) printPlan(w http.ResponseWriter, r *http.Request) {
	var rec *models.FSTRecommendation
	var zone int
	var sun models.SunType
	var soil models.SoilType
	var layout string
	var palette models.ColorPalette

	if r.Method == http.MethodPost {
		// Build from current plant IDs (preserves swaps)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		zone = parseInt(r.FormValue("zone"), 0)
		sun = models.SunType(r.FormValue("sun"))
		soil = models.SoilType(r.FormValue("soil"))
		layout = r.FormValue("layout")
		palette = models.ColorPalette(r.FormValue("palette"))

		if !models.IsValidZone(zone) || layout == "" {
			http.Redirect(w, r, "/planner", http.StatusSeeOther)
			return
		}

		thrillerID := parseInt(r.FormValue("thriller_id"), 0)
		rec = &models.FSTRecommendation{
			LayoutType: layout,
			Zone:       zone,
			Sun:        sun,
			Soils:      []models.SoilType{soil},
			Notes:      "Custom arrangement",
		}
		if thrillerID > 0 {
			if f, err := h.service.GetFlowerByID(thrillerID); err == nil {
				rec.Thriller = f
			}
		}
		for i := 0; i < 3; i++ {
			fid := parseInt(r.FormValue(fmt.Sprintf("filler_%d", i)), 0)
			if fid > 0 {
				if f, err := h.service.GetFlowerByID(fid); err == nil {
					rec.Fillers = append(rec.Fillers, *f)
				}
			}
		}
		for i := 0; i < 2; i++ {
			fid := parseInt(r.FormValue(fmt.Sprintf("spiller_%d", i)), 0)
			if fid > 0 {
				if f, err := h.service.GetFlowerByID(fid); err == nil {
					rec.Spillers = append(rec.Spillers, *f)
				}
			}
		}
		if rec.Thriller != nil && (len(rec.Fillers) > 0 || len(rec.Spillers) > 0) {
			rec.Notes = fmt.Sprintf("Your custom %s features %s as the centerpiece, %d filler(s) for body, and %d spiller(s) for trailing edges.",
				layout, rec.Thriller.Name, len(rec.Fillers), len(rec.Spillers))
		}
	} else {
		// GET: generate fresh from params
		zone = parseInt(r.URL.Query().Get("zone"), 5)
		layout = r.URL.Query().Get("layout")
		sun = models.SunType(r.URL.Query().Get("sun"))
		soil = models.SoilType(r.URL.Query().Get("soil"))
		palette = models.ColorPalette(r.URL.Query().Get("palette"))

		if !models.IsValidZone(zone) || layout == "" {
			http.Redirect(w, r, "/planner", http.StatusSeeOther)
			return
		}

		var err error
		rec, err = h.service.Generate(zone, sun, layout, soil, palette)
		if err != nil {
			http.Error(w, "Failed to generate recommendation", http.StatusInternalServerError)
			return
		}
	}

	h.render(w, "print.html", map[string]interface{}{
		"PageTitle":      "Shopping List",
		"Rec":            rec,
		"Zone":           zone,
		"Sun":            string(sun),
		"Soil":           string(soil),
		"Palette":        string(palette),
		"Layout":         layout,
		"FillerCounts":   countPlants(rec.Fillers),
		"SpillerCounts":  countPlants(rec.Spillers),
		"FilledFillers":  deduplicatePlants(rec.Fillers),
		"FilledSpillers": deduplicatePlants(rec.Spillers),
	})
}

// fetchAlts is a helper to safely get alternatives without error checking in handlers
func fetchAlts(svc *service.RecommendationService, zone int, sun models.SunType, soil models.SoilType, role models.FSTRole, palette models.ColorPalette) []models.Flower {
	alts, _ := svc.GetAlternatives(zone, sun, soil, role, palette)
	return alts
}
