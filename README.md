# 🌿 Poterie

**Poterie** is a minimalist web app for gardeners — design beautiful flower arrangements using the **Filler-Spiller-Thriller (FST)** methodology.

## Features

- **FST Planner** — Select your USDA zone, sun exposure, soil type, color theme, and layout type to get a personalized flower arrangement
- **Color Themes** — Five coordinated palettes (Warm Sunset, Cool & Calm, Romantic Pinks, Monochrome Green, Jewel Tones) filter plants to harmonize your design
- **Visual SVG Mockup** — Top-down rendering of your container or bed arrangement with labeled flowers
  - **Pot layout:** Thriller in center, fillers surrounding on sides and back, spillers cascading over the front edge
  - **Bed layout:** Thriller back-center, fillers flanking, spillers cascading at the front
- **Zone Reference** — Interactive USDA zone modal with temperature bands and compatible flowers
- **Botanical Theme** — Redouté-inspired design with growing vine scrollwork and watercolor parchment background
- **Shopping List** — Clean A4 print layout with 3-column breakdown, deduplicated plants, and planting instructions
- **Plant Swap** — Swap individual flowers via dropdown while preserving positional layout and color theme
- **Shuffle** — Re-roll the whole arrangement with the same criteria in one click
- **Shareable Plans** — Copy a `/plan?...` link that reproduces your exact arrangement (survives swaps)
- **Bloom-Season Awareness** — Refined per-flower bloom windows (spring/summer/fall, incl. cool-season plants); the engine prefers picks that flower together and the notes call out succession bloomers
- **Public Catalog** — Browse all 58+ flowers at `/catalog` (filter by role), with a detail page per flower at `/flowers/{id}` (zones, sun, soil, bloom window, height, spacing, Wikipedia link)
- **Flower Catalog Data** — Versioned JSON catalog (externalized from code), Wikipedia links, and full plant data
- **Admin Panel** — Full CRUD for the flower database + catalog import/export (basic auth + CSRF protected at `/admin`)
- **Zero Dependencies** — Single Go binary with embedded SQLite and templates

## Tech Stack

| Layer | Technology |
|-------|------------|
| **Backend** | Go 1.22 + chi router |
| **Database** | SQLite (embedded, WAL mode) |
| **Frontend** | Go `html/template` + HTMX + Alpine.js |
| **Styling** | Tailwind CSS (CDN) + custom botanical CSS |
| **Visualization** | Inline SVG with Redouté aesthetics |
| **Container** | Multi-stage Docker → Alpine |
| **Security** | CSRF tokens, CSP headers, rate limiting |
| **CI/CD** | GitHub Actions → GHCR → ArgoCD |
| **DNS/Ingress** | cloudflared tunnel (HTTPS via Cloudflare) |
| **Kubernetes** | Helm chart, PVC via Synology CSI |

## Quick Start

```bash
# From source
go run .

# Or with Docker
docker run -p 8080:8080 ghcr.io/bdfrost/poterie:latest
```

Open `http://localhost:8080`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `/data/poterie.db` | SQLite database path (must be a writable location) |
| `VERSION` | `dev` | Application version |
| `ADMIN_USER` | `admin` | Admin panel username |
| `ADMIN_PASS` | **Required** | Admin panel password — **must not be the default**. The app will refuse to start if `changeme` is used. |
| `CSRF_SECRET` | auto-generated | 32-byte key for CSRF HMAC (set to a random string in production) |

## Security

- **CSRF Protection**: All admin POST endpoints require a CSRF token via hidden form field.
- **Rate Limiting**: 120 requests per minute per client IP across all endpoints.
- **CSP Headers**: Content-Security-Policy restricts script/style sources to known CDNs and the app's own static files.
- **No-clickjacking**: `X-Frame-Options: DENY` and CSP `frame-ancestors 'none'` prevent embedding in iframes.
- **Strong credentials enforced**: The app refuses to start with the default `changeme` password.

## Docker

```bash
docker build -t ghcr.io/bdfrost/poterie:latest .
docker run -p 8080:8080 \
  -v poterie-data:/data \
  -e ADMIN_USER=admin -e ADMIN_PASS=your-strong-password \
  -e CSRF_SECRET=random-32-byte-string-here \
  ghcr.io/bdfrost/poterie:latest
```

## Kubernetes

An official Helm chart is published to GHCR as an OCI artifact on every release:

```bash
# Minimal install (ephemeral storage, no ingress — good for evaluation)
helm install poterie oci://ghcr.io/bdfrost/charts/poterie -n poterie --create-namespace

# Production-style install
helm install poterie oci://ghcr.io/bdfrost/charts/poterie -n poterie --create-namespace \
  --set persistence.enabled=true \
  --set persistence.size=2Gi \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set 'ingress.hosts[0].host'=poterie.example.com \
  --set 'ingress.tls[0].secretName'=poterie-cert \
  --set 'ingress.tls[0].hosts[0]'=poterie.example.com \
  --set admin.existingSecret=poterie-admin
```

The chart never creates or contains secrets. For the `/admin` panel, create a
Secret yourself (SealedSecret, ExternalSecrets, SOPS, plain kubectl — your
choice) with `ADMIN_USER` and `ADMIN_PASS` keys, then point
`admin.existingSecret` at it:

```bash
kubectl -n poterie create secret generic poterie-admin \
  --from-literal=ADMIN_USER=admin --from-literal=ADMIN_PASS='pick-a-good-one'
```

The chart source lives in `charts/poterie/` in this repo; `values.yaml`
documents every knob (persistence, ingress, probes, resources, env).

**Chart versioning is automatic** — `Chart.yaml` carries `0.0.0-dev`
placeholders; the `publish-chart` CI job stamps `version` and `appVersion`
from the git tag at release time. Never hand-edit them for a release.

## CI/CD Pipeline

### Development Cycle
```
write → go build → go test → go run (manual test) → git commit → push
```

### Pipeline Stages (on push/PR)
| Stage | Purpose |
|-------|---------|
| **test** | `go test -cover` + coverage check → **staticcheck** |
| **smoke-test** | Docker build → start container → verify `/api/health`, `/api/fst`, `/planner` |
| **build-and-push** | Multi-stage Docker → push to GHCR → **Trivy CVE scan** (reported) |
| **release** | Auto-create GitHub release |
| **publish-chart** | `helm lint` + `helm package` → push chart to `oci://ghcr.io/bdfrost/charts` |

### Release Cycle (tag → prod)
```
git tag v*.X → CI builds/test/push → gitops bump → ArgoCD sync → verify
```

1. **Pre-tag:** `go test ./...` and `go build ./...` pass locally. **If the
   catalog data changed** (new flowers, photos, zone/bloom fixes), bump
   `catalog_version` in both `data/catalog-v1.json` and
   `internal/db/catalog-v1.json` — existing installs auto-upgrade their
   flower database to the new catalog on startup (user-customized flowers
   are never overwritten).
2. **Tag push:** `git tag vX.Y.Z && git push origin main vX.Y.Z`
3. **CI gates:** test + staticcheck + smoke test pass (Trivy scan reports but doesn't block)
4. **GitOps bump:** Update `image` tag in `charts/poterie/values.yaml` → push to argocd repo
5. **Post-deploy:** `curl -sf https://<your-domain>/api/health`
6. **Rollback:** Revert GitOps commit → ArgoCD auto-syncs to previous version

## Project Structure

```
├── main.go                       # Entry point, router init
├── data/catalog-v1.json          # Versioned flower catalog (externalized data)
├── internal/
│   ├── config/config.go          # Env vars, defaults, validation
│   ├── db/                       # SQLite schema, catalog loader, seed
│   ├── handler/pages.go          # HTTP handlers (pages + API + admin)
│   ├── models/flower.go          # Flower structs, enums, color palettes
│   └── service/
│       ├── recommendation.go     # FST matching engine with color filtering
│       └── ...
├── templates/                    # Go html/template + HTMX
│   ├── base.html
│   ├── planner.html              # Input form + zone map modal
│   ├── print.html                # Print-optimized shopping list
│   └── partials/
│       ├── fst.html              # Result card with SVG + swap
│       └── svg-layouts.html      # Pot & bed SVG layouts
├── static/
│   ├── css/style.css             # Botanical theme, print styles
│   └── js/app.js                 # Theme toggle, vine animations
├── .github/workflows/ci.yml      # CI/CD pipeline
└── charts/poterie/               # Public Helm chart (published to GHCR OCI on release)
```

## Dev → Prod Release Process

To cut a release from `main`:

```bash
# 1. Ensure main is clean and tests pass
cd ~/poterie
go test ./... && go build ./...

# 2. Tag and push
git tag vX.Y.Z && git push origin main vX.Y.Z

# 3. Wait for CI (CI → GHCR)
gh run watch --repo bdfrost/poterie

# 4. Bump GitOps repo
cd ~/argocd
# Edit charts/poterie/values.yaml → image: ghcr.io/bdfrost/poterie:X.Y.Z
git add . && git commit -m "bump: poterie → vX.Y.Z" && git push

# 5. Verify
sleep 30
curl -sf -o /dev/null -w "%{http_code}" https://<your-domain>/api/health
# should return 200
```

## Flower Catalog

Poterie ships with a **versioned JSON catalog** (`data/catalog-v1.json`) containing 58+ flowers pre-seeded into the database. Each entry includes:

- Common name, botanical name, FST role (thriller/filler/spiller)
- USDA zone range, sun requirement, compatible soil types
- Color description (used by the color theme filter)
- Bloom season, height, spacing, description
- Wikipedia URL for further reading

### For Self-Hosters: Custom Catalogs

Poterie separates application logic from flower data, making it easy to customize:

1. **Export** your catalog from the admin panel (`/admin/catalog/export`)
2. **Edit** the JSON — add your own regional flowers, remove ones you don't grow, update colors
3. **Import** it via the admin panel (`/admin/catalog/import`) or rebuild with your own `data/catalog-v1.json`

The catalog format:
```json
{
  "catalog_version": 1,
  "flowers": [
    {
      "name": "Sunflower",
      "botanical_name": "Helianthus annuus",
      "role": "thriller",
      "zone_min": 3,
      "zone_max": 10,
      "sun": "full_sun",
      "soils": ["loam", "sandy", "well_drained"],
      "color": "Yellow",
      "bloom_season": "summer",
      "height": "3-10ft",
      "spacing": "12-24in",
      "description": "Iconic tall blooms.",
      "image_url": "🌻",
      "wikipedia_url": "https://en.wikipedia.org/wiki/Helianthus_annuus"
    }
  ]
}
```

## Color Themes

The planner includes a color theme filter that narrows recommendations to harmonious palettes:

| Theme | Colors |
|-------|--------|
| **Warm Sunset** | Reds, oranges, yellows, golds |
| **Cool & Calm** | Blues, purples, lavenders, whites, silvers |
| **Romantic Pinks** | Pinks, roses, magentas |
| **Monochrome Green** | Greens, foliage, silvers, bronzes |
| **Jewel Tones** | Deep purples, burgundies, golds, bronzes |

If a palette is too restrictive for your zone/sun combo, Poterie falls back to showing all compatible plants so you always get recommendations.

## Screenshots

*Coming soon*
