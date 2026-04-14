# 🍳 Home Cooking

> Self-hosted kitchen management — recipes, pantry, and shopping list in one place.

A personal web app to manage your recipe library and track your pantry inventory. Built with Go and React, fully containerized, zero cloud dependency.

---

<!-- 📸 SCREENSHOT — Hero image -->
<!-- Recommended: full-page screenshot of the dashboard on desktop -->
<!-- Save to: docs/screenshots/dashboard.png -->
<!-- Then replace this comment with: ![Dashboard](docs/screenshots/dashboard.png) -->

---

## Features

- **Recipe library** — Full CRUD with full-text search (SQLite FTS5), ingredients, step-by-step instructions, difficulty, tags, prep and cook times
- **Pantry inventory** — Track quantities with ±1 adjustments, expiry dates, per-category filters, and automatic low-stock alerts
- **Shopping list** — Auto-generated from low-stock items
- **Secure authentication** — Short-lived JWT access token (15 min) + httpOnly refresh token (7 days) with server-side revocation
- **Responsive** — Collapsible sidebar on desktop, bottom navigation on mobile
- **Self-hosted** — Single command to start, SQLite for storage, no external service required

---

## Stack

| | Technology |
|---|---|
| **Frontend** | React 18, TypeScript, Vite, Tailwind CSS |
| **Backend** | Go 1.22, Gin, zerolog |
| **Auth** | JWT HS256 · bcrypt cost 12 · double-token |
| **Database** | SQLite — WAL mode, FTS5, embedded migrations |
| **Proxy** | nginx 1.27 — reverse proxy + SPA serving |
| **Runtime** | Docker / Podman (rootless compatible) |

The Go binary uses [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) — a pure Go SQLite driver, no CGO. The final image is built on `scratch` (zero OS).

---

## Screenshots

<!-- 📸 Add screenshots here once the app is running -->
<!-- Suggested captures (save to docs/screenshots/): -->

### Dashboard
<!-- ![Dashboard desktop](docs/screenshots/dashboard-desktop.png) -->
<!-- ![Dashboard mobile](docs/screenshots/dashboard-mobile.png) -->

### Recipes
<!-- ![Recipe list](docs/screenshots/recipes-list.png) -->
<!-- ![Recipe detail](docs/screenshots/recipes-detail.png) -->
<!-- ![Create recipe](docs/screenshots/recipes-create.png) -->

### Pantry
<!-- ![Pantry desktop](docs/screenshots/pantry-desktop.png) -->
<!-- ![Pantry mobile](docs/screenshots/pantry-mobile.png) -->

---

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) ≥ 24 **or** [Podman](https://podman.io/) ≥ 4 with `podman-compose`

### 1. Clone

```bash
git clone https://github.com/Kitslap/HomeCooking.git
cd HomeCooking
```

### 2. Configure

```bash
cp .env.example .env
```

Edit `.env` and set a strong secret:

```env
JWT_SECRET=your-very-long-random-secret-here   # minimum 32 characters
FRONTEND_PORT=3000
```

Generate a secure value with:
```bash
openssl rand -hex 64
```

> ⚠️ The app will refuse to start if `JWT_SECRET` is missing or too short.

### 3. Run

**Docker:**
```bash
docker compose up --build -d
```

**Podman:**
```bash
podman compose -f docker-compose.yml up --build -d
```

Open **http://localhost:3000**, register your account, and start cooking.

---

## Project Structure

```
HomeCooking/
│
├── backend/                        # Go API
│   ├── cmd/server/main.go          # Entry point, router, graceful shutdown
│   ├── internal/
│   │   ├── auth/                   # JWT, bcrypt, login/register handlers
│   │   ├── config/                 # Typed config from environment
│   │   ├── db/                     # SQLite connection + embedded migrations
│   │   │   └── migrations/         # SQL files (001_init.sql, …)
│   │   ├── middleware/             # CORS, rate limiter, JWT auth, security headers, logger
│   │   ├── recipe/                 # Recipe CRUD — handler + repository
│   │   └── storage/               # Pantry CRUD — handler + repository
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile                  # Multi-stage: golang:alpine → scratch
│
├── frontend/                       # React SPA
│   ├── src/
│   │   ├── pages/                  # Dashboard, Recipes, Storage, Auth
│   │   ├── components/             # Layout, UI primitives
│   │   └── lib/api.ts              # Typed HTTP client
│   ├── dist/                       # Pre-built bundle (served by nginx)
│   ├── docker/nginx.conf           # Reverse proxy config
│   ├── package.json
│   ├── vite.config.ts
│   └── Dockerfile                  # nginx:alpine serving the bundle
│
├── docker-compose.yml              # Production stack
├── docker-compose.dev.yml          # Development overrides
├── .env.example                    # Environment template
└── README.md
```

---

## Building from Source

### Backend

```bash
cd backend
go mod download
go build -o home-cooking ./cmd/server
```

### Frontend

```bash
cd frontend
npm install
npm run build
# output: frontend/dist/
```

Then rebuild the Docker image to pick up your changes:

```bash
docker compose up --build -d frontend
```

---

## API

All routes are prefixed `/api/v1`. Protected routes require `Authorization: Bearer <token>`.

**Auth**

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| POST | `/auth/register` | | Create account |
| POST | `/auth/login` | | Login → access + refresh token |
| POST | `/auth/refresh` | | Rotate refresh token |
| POST | `/auth/logout` | ✓ | Revoke session |
| GET | `/auth/me` | ✓ | Current user |

**Recipes**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/recipes` | List (search, pagination) |
| POST | `/recipes` | Create |
| GET | `/recipes/:id` | Detail |
| PATCH | `/recipes/:id` | Update |
| DELETE | `/recipes/:id` | Delete |

**Pantry**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/storage` | List (filter, search) |
| POST | `/storage` | Add item |
| GET | `/storage/stats` | Stock statistics |
| GET | `/storage/alerts` | Low-stock + expiring items |
| GET | `/storage/shopping-list` | Auto shopping list |
| GET | `/storage/:id` | Item detail |
| PATCH | `/storage/:id` | Update item |
| PATCH | `/storage/:id/quantity` | Adjust quantity (±delta) |
| DELETE | `/storage/:id` | Delete item |

---

## Security

| Measure | Detail |
|---|---|
| Password hashing | bcrypt cost 12 |
| Token strategy | Access token 15 min + httpOnly refresh 7 days |
| Token revocation | Refresh tokens stored and invalidated server-side |
| Enumeration protection | Timing-safe register endpoint |
| Rate limiting | Per-IP token bucket (`golang.org/x/time/rate`) |
| CORS | Strict origin whitelist |
| HTTP headers | `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, CSP, HSTS (production) |
| Docker image | `scratch` base — no shell, no OS utilities |
| SQLite | WAL mode, `foreign_keys=on`, `busy_timeout=5000` |

---

## Configuration

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | **required** | HMAC secret, min 32 chars |
| `FRONTEND_PORT` | `3000` | Host port for the UI |
| `PORT` | `8080` | Internal backend port |
| `ENV` | `production` | `production` or `development` |
| `DB_PATH` | `/data/home-cooking.db` | SQLite file path |
| `CORS_ORIGINS` | `http://localhost:3000` | Allowed origins, comma-separated |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `7d` | Refresh token lifetime |
| `RATE_LIMIT_RPS` | `20` | Requests/second per IP |
| `RATE_LIMIT_BURST` | `40` | Burst allowance per IP |

---

## Roadmap

- [ ] Meal planning — weekly calendar
- [ ] Recipe → shopping list (compare ingredients vs. pantry)
- [ ] Recipe image upload
- [ ] PWA / offline support
- [ ] Barcode scanning for pantry items
- [ ] Import recipe from URL
- [ ] Multi-user / household sharing

---

## Contributing

1. Fork the repo
2. Create a branch: `git checkout -b feat/my-feature`
3. Commit: `git commit -m "feat: describe your change"`
4. Open a Pull Request

Please open an issue first for any significant change so we can align on approach.

---

## License

MIT — see [LICENSE](https://github.com/Kitslap/HomeCooking?tab=MIT-1-ov-file) for details.

---

<div align="center">
Made for home cooks who like clean code.
</div>
