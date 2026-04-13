# 🍳 Cooking Home

A self-hosted kitchen management web app — track your recipes, manage your pantry, and never run out of ingredients again.

Built with a Go backend, React frontend, and SQLite for zero-dependency storage. Runs entirely in Docker/Podman with no external services required.

---

<!-- Screenshot: app overview (dashboard on desktop) -->
![alt text](https://github.com/Kitslap/HomeCooking/blob/main/image.jpg?raw=true)
<!-- Suggested tool: browser at http://localhost:3000, full-page screenshot -->

---

## Features

- **Recipe library** — Create, edit, and search recipes with full-text search (SQLite FTS5). Attach ingredients, step-by-step instructions, tags, difficulty level, and prep/cook times.
- **Pantry inventory** — Track stock quantities with +/− adjustments, expiry dates, low-stock alerts, and category filters.
- **Secure auth** — Double-token authentication: short-lived JWT access token (15 min) + httpOnly refresh token (7 days) with DB-side revocation.
- **Fully self-hosted** — Single `docker compose up` command, no cloud dependency, no registry needed.
- **Responsive UI** — Works on desktop and mobile. Bottom navigation on small screens, collapsible sidebar on desktop.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | React 18, TypeScript, Vite, Tailwind CSS |
| Backend | Go 1.22, Gin, zerolog |
| Auth | JWT HS256 (golang-jwt/jwt v5), bcrypt cost 12 |
| Database | SQLite with WAL mode, FTS5, embedded migrations |
| Proxy | nginx 1.27 (reverse proxy + SPA serving) |
| Container | Docker / Podman (rootless compatible) |

The Go backend uses [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) — a pure Go SQLite driver with **no CGO dependency**, enabling a fully static binary on a `scratch` Docker image.

---

## Screenshots

<!-- 📸 SCREENSHOT BLOCK — Replace placeholders with actual images -->
<!-- Format: ![alt text](docs/screenshots/filename.png) -->

### Dashboard

<!-- 📸 Desktop: stats cards + recent recipes + low-stock alerts -->
<!-- 📸 Mobile: stacked cards, bottom nav visible -->

### Recipes

<!-- 📸 Desktop: split panel — recipe list on left, detail on right -->
<!-- 📸 Mobile: recipe list view -->
<!-- 📸 Mobile: recipe detail with ← Retour button -->
<!-- 📸 Create/edit modal with ingredients and steps -->

### Pantry inventory

<!-- 📸 Desktop: table view with +/− quantity buttons, level badges -->
<!-- 📸 Mobile: card view per item -->
<!-- 📸 Add item modal (bottom sheet on mobile) -->

### Authentication

<!-- 📸 Login / register screen (mobile-friendly card) -->

---

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) ≥ 24 **or** [Podman](https://podman.io/) ≥ 4 with `podman-compose`
- No other dependencies required

### 1. Clone the repository

```bash
git clone https://github.com/pierreburbaud/cooking-home.git
cd cooking-home
```

### 2. Configure environment

```bash
cp .env.example .env
```

Open `.env` and set a strong `JWT_SECRET` (minimum 32 characters):

```env
JWT_SECRET=your-very-long-random-secret-key-here
FRONTEND_PORT=3000
```

> ⚠️ The app will refuse to start if `JWT_SECRET` is absent or too short.

### 3. Start the app

**Docker:**
```bash
docker compose up --build -d
```

**Podman:**
```bash
podman compose -f docker-compose.yml up --build -d
```

The app is available at **http://localhost:3000** (or whichever port you set in `.env`).

### 4. Create your account

Open the app, click **S'inscrire**, and register your account. All data is stored locally in a SQLite volume.

---

## Project Structure

```
cooking-home/
├── backend/                   # Go API
│   ├── cmd/server/main.go     # Entry point, router setup, graceful shutdown
│   ├── internal/
│   │   ├── auth/              # JWT generation/validation, login/register handlers
│   │   ├── config/            # Typed config loaded from environment
│   │   ├── db/                # SQLite connection, embedded migrations
│   │   │   └── migrations/    # SQL migration files (001_init.sql, …)
│   │   ├── middleware/        # CORS, rate limiter, JWT auth, security headers, logger
│   │   ├── recipe/            # Recipe CRUD — handler + repository
│   │   └── storage/           # Pantry CRUD — handler + repository
│   └── Dockerfile             # Multi-stage: golang:alpine → scratch (zero OS)
│
├── frontend/                  # React SPA
│   ├── dist/                  # Pre-built bundle served by nginx
│   ├── docker/nginx.conf      # nginx config: proxy /api/*, SPA fallback, gzip
│   └── Dockerfile             # nginx:alpine serving the pre-built bundle
│
├── docker-compose.yml         # Production stack
├── docker-compose.dev.yml     # Development overrides
├── .env.example               # Environment variable template
└── README.md
```

---

## Security

The app is designed for personal/self-hosted use with security-conscious defaults:

- **bcrypt cost 12** for password hashing
- **Timing-safe** register endpoint (prevents user enumeration)
- **Token revocation** — refresh tokens are stored and invalidated server-side on logout
- **Per-IP rate limiting** via token bucket (`golang.org/x/time/rate`)
- **Strict CORS** — whitelist-only origin validation
- **Security headers** — `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`, CSP
- **HSTS** enabled in production mode
- **Zero OS attack surface** — backend runs on a `scratch` Docker image (no shell, no utilities)
- **SQLite WAL mode** with `foreign_keys=on` and `busy_timeout=5000`

---

## API Overview

All routes are prefixed with `/api/v1`. Protected routes require `Authorization: Bearer <token>`.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | — | Create account |
| POST | `/auth/login` | — | Login, returns access + refresh token |
| POST | `/auth/refresh` | — | Rotate refresh token |
| POST | `/auth/logout` | ✓ | Revoke refresh token |
| GET | `/auth/me` | ✓ | Current user info |
| GET | `/recipes` | ✓ | List recipes (search, pagination) |
| POST | `/recipes` | ✓ | Create recipe |
| GET | `/recipes/:id` | ✓ | Get recipe detail |
| PATCH | `/recipes/:id` | ✓ | Update recipe |
| DELETE | `/recipes/:id` | ✓ | Delete recipe |
| GET | `/storage` | ✓ | List pantry items (filter, search) |
| POST | `/storage` | ✓ | Add item |
| GET | `/storage/stats` | ✓ | Stock statistics |
| GET | `/storage/alerts` | ✓ | Low-stock and expiring items |
| GET | `/storage/shopping-list` | ✓ | Auto-generated shopping list |
| GET | `/storage/:id` | ✓ | Get item detail |
| PATCH | `/storage/:id` | ✓ | Update item |
| DELETE | `/storage/:id` | ✓ | Delete item |
| PATCH | `/storage/:id/quantity` | ✓ | Adjust quantity (±delta) |

---

## Configuration Reference

All configuration is done via environment variables (`.env` file or shell):

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | **required** | HMAC secret, minimum 32 chars |
| `FRONTEND_PORT` | `3000` | Host port for the web UI |
| `PORT` | `8080` | Internal backend port |
| `ENV` | `production` | `production` or `development` |
| `DB_PATH` | `/data/cooking-home.db` | SQLite file path |
| `CORS_ORIGINS` | `http://localhost:3000` | Allowed origins (comma-separated) |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `7d` | Refresh token lifetime |
| `RATE_LIMIT_RPS` | `20` | Requests per second per IP |
| `RATE_LIMIT_BURST` | `40` | Burst allowance per IP |

---

## Development

To rebuild the frontend bundle after making changes to the React source:

```bash
cd cooking-home-src    # your React source directory
npm install
npm run build
cp -r dist/* ../frontend/dist/
```

Then rebuild the Docker image:

```bash
docker compose up --build -d frontend
```

---

## Contributing

Contributions are welcome. Please open an issue before submitting a large PR so we can discuss the approach.

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Commit your changes: `git commit -m "feat: add my feature"`
4. Push and open a Pull Request

---

## License

MIT — see [LICENSE](https://github.com/Kitslap/HomeCooking/blob/main/LICENSE) for details.

---

<div align="center">
  Made with Claude & ☕ for home cooks who like clean code.
</div>
