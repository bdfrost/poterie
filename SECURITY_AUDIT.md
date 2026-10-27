# Poterie Security Audit

Date: 2026-05-29
Scope: Full codebase of poterie (Go web app, Chi router, SQLite, embedded templates)
Reviewer: Hermes Security Analysis

---

## Severity Summary

| Severity | Count | IDs |
|----------|-------|-----|
| CRITICAL | 3 | C1, C2, C3 |
| HIGH | 4 | H1, H2, H3, H4 |
| MEDIUM | 4 | M1, M2, M3, M4 |
| LOW | 2 | L1, L2 |

---

## CRITICAL

### C1 — Path Traversal via Static File Server

**File:** `main.go:45`
**Risk:** Read arbitrary files (OS-level path traversal to /etc/passwd, DB, env vars)

```go
r.Handle("/static/*", http.FileServer(http.FS(assets)))
```

Chi's `Handle("/static/*", ...)` includes the full remainder after `/static/` in `r.URL.Path`. A request to:

```
GET /static/../../main.go
```

or on some Go/embed versions:

```
GET /static/../../../etc/passwd
```

may bypass the embed sandbox depending on Go version. The Go `embed.FS` documentation notes that `Open()` validates paths containing `..` but the behavior varies across Go versions and the `http.FileServer` wrapper may not apply the validation consistently.

**Fix:** Replace with a custom handler that explicitly strips the prefix and rejects `..`:

```go
r.Handle("/static/*", http.StripPrefix("/static/",
    http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        path := strings.TrimPrefix(r.URL.Path, "/static/")
        if path == "" || strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
            http.NotFound(w, r)
            return
        }
        http.FileServer(http.FS(assets)).ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "path", path)))
    })))
```

Or best: implement a custom handler that opens from `embed.FS` directly with extension allowlisting.

---

### C2 — No CSRF Protection on Admin POST Endpoints

**File:** `internal/handler/pages.go:57-68`

```go
r.Group(func(r chi.Router) {
    r.Use(adminAuth(cfg.AdminUser, cfg.AdminPass))
    r.Get("/admin", h.admin)
    r.Post("/admin/flowers/{id}/update", h.adminUpdateFlower)   // No CSRF!
    r.Post("/admin/flowers/new", h.adminNewFlower)              // No CSRF!
    r.Post("/admin/flowers/{id}/delete", h.adminDeleteFlower)   // No CSRF!
    r.Post("/admin/catalog/import", h.adminCatalogImport)       // No CSRF!
})
```

`gorilla/csrf` is in `go.mod` but **never used**. The admin group has basic auth but **zero CSRF protection**. An attacker can craft a malicious page that submits forms to:

- **Delete any flower** (`POST /admin/flowers/{id}/delete`)
- **Add arbitrary flowers** (`POST /admin/flowers/new`)
- **Import a malicious catalog** (`POST /admin/catalog/import`) — this is especially dangerous as it loads JSON into the database

Since the app runs behind a TLS-terminating proxy (nginx), the attacker must trick an authenticated admin into visiting their page (social engineering / XSS). But if the admin is logged in and visits a malicious page, all admin state changes are trivially triggerable.

**Fix:** Add `csrf.Protect` middleware to the admin group:

```go
r.Group(func(r chi.Router) {
    r.Use(adminAuth(cfg.AdminUser, cfg.AdminPass))
    r.Use(csrf.Protect([]byte(csrfSecret()), csrf.Secure(false)))
    // ... routes
})
```

Add `{{csrfField}}` to all admin forms.

---

### C3 — Stored XSS via ImageURL / Database Fields

**File:** `templates/partials/svg-layouts.html` (and others), `internal/handler/pages.go`

The `ImageURL` field accepts arbitrary user input (stored in SQLite, rendered in HTML). The `safeCSS` function is a **no-op**:

```go
// pages.go:86-88
"safeCSS": func(s string) template.CSS {
    return template.CSS(s)  // NO VALIDATION AT ALL
},
```

Attackers can store malicious URLs in `image_url`:
- `javascript:alert(document.cookie)` — executes JS if rendered in `src` or `href`
- `<img src=x onerror=alert(1)>` — XSS via HTML attribute
- CSS injection: `url(javascript:alert(1))` if used in CSS context

**Fixes needed:**
1. Make `safeCSS` actually validate/escape input
2. URL-escape `ImageURL` when rendered in HTML contexts
3. Sanitize `ImageURL` on write: reject dangerous schemes (`javascript:`, `data:`, empty)

---

## HIGH

### H1 — Timing Attack on Admin Credentials

**File:** `internal/handler/pages.go:154-166`

```go
func adminAuth(user, pass string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            u, p, ok := r.BasicAuth()
            if !ok || u != user || p != pass {  // LEAKS username THEN password length
                w.Header().Set("WWW-Authenticate", `Basic realm="Admin"`)
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

Simple `!=` comparison returns early on first differing character. An attacker can measure request timing to determine password length character-by-character. With ~90 bits of entropy (recommended admin password), this is feasible: 90 chars × ~20 chars/attempt = ~1800 requests to fully recover the password.

**Fix:** Use `crypto/subtle.ConstantTimeCompare`:

```go
import "crypto/subtle"
// ...
if !ok || subtle.ConstantTimeCompare([]byte(u), []byte(user)) != 1 ||
   subtle.ConstantTimeCompare([]byte(p), []byte(pass)) != 1 {
```

---

### H2 — No Input Validation on Admin Flower CRUD

**File:** `internal/handler/pages.go:405-441` (adminNewFlower), `pages.go:477-514` (adminUpdateFlower)

All form values are read directly without length limits:

```go
f := &models.Flower{
    Name:          r.FormValue("name"),          // unlimited length
    Description:   r.FormValue("description"),    // unlimited length
    BotanicalName: r.FormValue("botanical_name"), // unlimited length
    Color:         r.FormValue("color"),          // unlimited length
    Height:        r.FormValue("height"),         // unlimited length
    Spacing:       r.FormValue("spacing"),        // unlimited length
    ImageURL:      r.FormValue("image_url"),      // no URL validation
    WikipediaURL:  r.FormValue("wikipedia_url"),  // no URL validation
}
```

No validation on `Role`, `Sun`, or other enum fields. The `description` field could be arbitrarily long (DoS / DB bloat). URLs are not validated at all in the admin form — only in the catalog import JSON validation.

**Fix:** Add length limits and validation:
- `Name`: max 100 chars
- `BotanicalName`: max 200 chars
- `Description`: max 2000 chars
- `Color`, `Height`, `Spacing`: max 100 chars each
- `ImageURL`, `WikipediaURL`: validate URL scheme
- `Role`, `Sun`, `BloomSeason`: validate against known enums

---

### H3 — No Rate Limiting on Admin Endpoints

**File:** `internal/handler/pages.go`

The rate limiter exists but is applied globally at 120 req/min. There's no additional protection on auth endpoints (no brute-force lockout), admin CRUD operations (no write-rate limiting), or the catalog import endpoint.

**Fix:** Add stricter rate limiting for admin endpoints (e.g., 10 req/min for auth, 20 req/min for writes). Consider brute-force protection with lockout after N failed attempts.

---

### H4 — No Validation on Catalog Import JSON Structure

**File:** `internal/handler/pages.go:374-403`

The catalog import limits file size (512KB) but doesn't validate the JSON structure. The `SeedFromJSON` function in `db/seed.go:82-108` unmarshals JSON directly:

```go
func SeedFromJSON(d *DB, data []byte) error {
    var parsed []models.Flower
    if err := json.Unmarshal(data, &parsed); err != nil { ... }
    // No field validation, no type checking, no schema enforcement
}
```

A malicious JSON could:
- Store `image_url` as `javascript:alert(1)` → stored XSS (C3)
- Store arbitrary values in `role`, `sun`, `zone_min`, etc.
- Create thousands of entries within the 512KB limit

**Fix:** Add JSON schema validation before import. Validate each flower entry's fields against expected types, enums, and lengths.

---

## MEDIUM

### M1 — Referrer-Policy: no-referrer Too Aggressive

**File:** `templates/base.html`

```html
<meta name="referrer" content="no-referrer">
```

This prevents ALL referrer information from being sent, which:
- Breaks CSRF protection (if implemented via referrer validation)
- Prevents debugging/monitoring of external link clicks
- Hides analytics data

**Fix:** Use `strict-origin-when-cross-origin` instead:

```html
<meta name="referrer" content="strict-origin-when-cross-origin">
```

---

### M2 — Version String Exposed in Template

**File:** `templates/base.html`, `internal/handler/pages.go:126`

```go
data["Version"] = h.cfg.Version
```

The version is passed to the template and rendered in the footer. This gives attackers intelligence about the specific deployment version for targeted exploit lookups.

**Fix:** Remove version from public templates. Only include in health check (authenticated) or omit entirely.

---

### M3 — Hardcoded Email in Templates (PII Leakage)

**File:** `templates/base.html`

```html
<!-- Brian Frost — bfrost@brainboy.com -->
```

This exposes Brian's personal email in every page's HTML source.

**Fix:** Remove or replace with a generic contact link.

---

### M4 — Health Endpoint Exposes No Auth

**File:** `internal/handler/pages.go:168-171`

```go
func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"status":"ok"}`))
}
```

While the health endpoint only returns `{"status":"ok"}`, it's public. In combination with other vulnerabilities, it helps attackers confirm the app is running.

**Fix:** Either remove the health endpoint, add basic auth, or place it on a separate port.

---

## LOW

### L1 — go-sqlite3 CGO Dependency

**File:** `go.mod`

Uses `github.com/mattn/go-sqlite3` which requires CGO and a C compiler. This:
- Makes container builds more complex
- Introduces C-level attack surface (buffer overflows in SQLite)
- Increases binary size

**Fix:** Consider `modernc.org/sqlite` for pure-Go SQLite (no CGO). Trade-off: slightly slower performance.

---

### L2 — rand.Intn() Not Cryptographically Secure

**File:** `internal/service/recommendation.go:43`

```go
rec.Thriller = &thrillers[rand.Intn(len(thrillers))]
```

Uses `math/rand` (default seed = 1) which is predictable. For a flower recommendation app this is fine — it's not used for security. But if any crypto-related changes are made, the seed must be changed.

**Fix:** Already acceptable for this use case. Document that `math/rand` is used intentionally for randomness (not security).

---

## Additional Observations

1. **Good:** All DB queries use parameterized queries (`?` placeholders) — no SQL injection via the database layer.
2. **Good:** `ADMIN_PASS` default value check in `config.Validate()` prevents accidental production deployment with default credentials.
3. **Good:** DB file is excluded from git via `.gitignore`.
4. **Good:** `http.MaxBytesReader` limits upload size on catalog import.
5. **Good:** `safeCSS` template function exists (even though it's a no-op — the intent is there).
6. **Missing:** No `Content-Security-Policy` header.
7. **Missing:** No `X-Content-Type-Options: nosniff` header.
8. **Missing:** No `X-Frame-Options` header.
9. **Missing:** No `X-XSS-Protection` header (legacy, but cheap insurance).
10. **Missing:** No audit logging for admin CRUD operations.

---

## Recommended Priority Order

1. **C3** — Fix stored XSS vectors (ImageURL validation, safeCSS)
2. **C2** — Add CSRF protection to admin group
3. **C1** — Fix path traversal in static file server
4. **H1** — Constant-time comparison for admin auth
5. **H2** — Add input validation to admin CRUD
6. **H4** — Validate catalog import JSON structure
7. **M1** — Fix Referrer-Policy
8. **M2/M3** — Remove PII/version from templates
9. **M4** — Protect health endpoint
