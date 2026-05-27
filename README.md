# 🌿 Poterie

**A modern, minimalist web app for gardeners** — design beautiful flower arrangements using the **Filler-Spiller-Thriller (FST)** methodology.

## Features

- **FST Planner** — Select your USDA zone, sun exposure, soil type, and layout type to get a personalized flower arrangement
- **Visual SVG Mockup** — Top-down rendering of your container or bed arrangement with labeled flowers
- **Botanical Theme** — Redouté-inspired design with two selectable themes (Bright/Sage)
- **Scroll Animations** — Growing vine effects as you scroll
- **Shopping List** — Export-ready shopping list with print/PDF support
- **Admin Panel** — Browse and manage the flower database (basic auth protected)
- **Zero Dependencies** — Single Go binary with embedded SQLite

## Tech Stack

- **Go** — idiomatic backend with chi router
- **SQLite** — embedded database (WAL mode)
- **HTMX + Alpine.js** — lightweight SPA without a build step
- **Tailwind CSS** — utility-first styling via CDN
- **SVG** — inline FST arrangement visualization

## Quick Start

```bash
go run .
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
docker run -p 8080:8080 ghcr.io/bdfrost/poterie:latest
```

## Kubernetes

A Helm chart is included in `helm/fst-planner/`. Configure your values and install:

```bash
helm install poterie helm/fst-planner/ -n poterie --create-namespace
```

## CI/CD

GitHub Actions workflow (`.github/workflows/ci.yml`):
1. **Test** — Runs all tests on PR and push
2. **Build & Push** — On version tags (`v*`), builds Docker image and pushes to GHCR
3. **Release** — Creates a GitHub release with auto-generated notes

## Test Coverage

```
config:    100.0%
db:         85.4%
handler:    14.0%
models:    100.0%
service:    91.7%
```

Core packages all exceed 80% coverage.

## Project Structure

```
├── main.go              # Entry point
├── internal/
│   ├── config/          # App configuration
│   ├── db/              # SQLite database + schema + seed
│   ├── models/          # Data structures
│   ├── handler/         # HTTP handlers + HTMX endpoints
│   └── service/         # FST recommendation engine
├── templates/           # Go HTML templates
│   ├── partials/        # SVG layouts, result cards, decorative vines
├── static/css/          # Custom botanical theme CSS
├── static/js/           # Client-side JS (vines, animations)
├── helm/                # Helm chart for Kubernetes
└── .github/workflows/   # CI/CD pipeline
```

## License

MIT — Built by [Brian Frost](mailto:bfrost@brainboy.com)
