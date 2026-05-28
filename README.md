# 🌿 Poterie

**Poterie** is a modern, minimalist web app for gardeners — design beautiful flower arrangements using the **Filler-Spiller-Thriller (FST)** methodology.

Live site: [poterie.frost.haus](https://poterie.frost.haus)

## Features

- **FST Planner** — Select your USDA zone, sun exposure, soil type, and layout type to get a personalized flower arrangement
- **Visual SVG Mockup** — Top-down rendering of your container or bed arrangement with labeled flowers
  - **Pot layout:** Thriller in center, fillers surrounding on sides and back, spillers cascading over the front edge
  - **Bed layout:** Thriller back-center, fillers flanking, spillers cascading at the front
- **Zone Reference** — Interactive USDA zone modal with temperature bands and compatible flowers
- **Botanical Theme** — Redouté-inspired design with growing vine scrollwork and watercolor parchment background
- **Shopping List** — Dedicated print page (`/planner/print`) with clean 3-column layout and planting instructions
- **Plant Swap** — Swap individual flowers via dropdown while preserving positional layout
- **Admin Panel** — Full CRUD for the flower database (basic auth protected at `/admin`)
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
| **CI/CD** | GitHub Actions → GHCR → ArgoCD |
| **DNS/Ingress** | cloudflared tunnel (HTTPS via Cloudflare) |
| **Kubernetes** | Helm chart, PVC via Synology CSI |

## Quick Start

```bash
# From source
go run ./cmd/server

# Or with Docker
docker run -p 8080:8080 ghcr.io/bdfrost/poterie:latest
```

Open `http://localhost:8080`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `/data/poterie.db` | SQLite database path |
| `ADMIN_USER` | `admin` | Admin panel username |
| `ADMIN_PASS` | `changeme` | Admin panel password |

## Docker

```bash
docker build -t ghcr.io/bdfrost/poterie:latest .
docker run -p 8080:8080 \
  -v poterie-data:/data \
  -e ADMIN_USER=admin -e ADMIN_PASS=changeme \
  ghcr.io/bdfrost/poterie:latest
```

## Kubernetes

Deployed via Helm chart in the ArgoCD GitOps repo (`github.com/bdfrost/argocd`).

```bash
helm install poterie charts/poterie/ -n poterie --create-namespace
```

Accessed through the cloudflared tunnel at `poterie.frost.haus`.

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

### Release Cycle (tag → prod)
```
git tag v0.5.x → CI builds/test/push → gitops bump → ArgoCD sync → verify
```

1. **Pre-tag:** `go test ./...` and `go build ./...` pass locally
2. **Tag push:** `git tag v0.5.x && git push origin main v0.5.x`
3. **CI gates:** test + staticcheck + smoke test pass (Trivy scan reports but doesn't block)
4. **GitOps bump:** Update `image` tag in `charts/poterie/values.yaml` → push to argocd repo
5. **Post-deploy:** `curl -sf https://poterie.frost.haus/api/health`
6. **Rollback:** Revert GitOps commit → ArgoCD auto-syncs to previous version

## Project Structure

```
├── cmd/server/main.go          # Entry point, router init
├── internal/
│   ├── config/config.go        # Env vars, defaults
│   ├── db/                     # SQLite schema + seed data
│   ├── handler/pages.go        # HTTP handlers (pages + API)
│   ├── models/flower.go        # Flower structs, enums
│   └── service/
│       ├── recommendation.go   # FST matching engine
│       └── ...
├── templates/                  # Go html/template + HTMX
│   ├── base.html
│   ├── planner.html            # Input form + zone map modal
│   ├── print.html              # Print-optimized shopping list
│   └── partials/
│       ├── fst.html            # Result card with SVG + swap
│       └── svg-layouts.html    # Pot & bed SVG layouts
├── static/css/style.css        # Botanical theme, print styles
├── static/js/app.js            # Theme toggle, vine animations
└── .github/workflows/ci.yml    # CI/CD pipeline
```

## Dev → Prod Release Process

To cut a release from `main`:

```bash
# 1. Ensure main is clean and tests pass
cd ~/poterie
go test ./... && go build ./...

# 2. Tag and push
git tag v0.5.19 && git push origin main v0.5.19

# 3. Wait for CI (CI → GHCR)
gh run watch --repo bdfrost/poterie

# 4. Bump GitOps repo
cd ~/argocd
# Edit charts/poterie/values.yaml → image: ghcr.io/bdfrost/poterie:0.5.19
git add . && git commit -m "bump: poterie → v0.5.19" && git push

# 5. Verify
sleep 30
curl -sf -o /dev/null -w "%{http_code}" https://poterie.frost.haus/api/health
# should return 200
```

## Flower Database

Seed data includes **50+ flowers** spanning USDA zones 3–11, with full/sun/shade and loam/clay/sandy/well-drained soil types. Each entry has bloom season, height, spacing, description, and compatibility info.

## Screenshots

*Coming soon*
