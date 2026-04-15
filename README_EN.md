<div align="center">
  <img src="home_cooking_logo_transparent.png" alt="Home Cooking" width="360" />
  <br /><br />
  <em>Self-hosted kitchen management - recipes, pantry, and shopping list in one place.</em>
  <br /><br />
  A personal web app to manage your recipe library and track your pantry inventory.<br />
  Built with Go and React, fully containerized, zero cloud dependency.
  <br /><br />
  <a href="README.md">🇫🇷 Version française</a>
</div>

<div align="center">
  <img src="screens/auth_page.png" alt="Login" width="360" />
  <br /><br />
  <img src="screens/dashboard.png" alt="Dashboard" width="270" />
  &nbsp;
  <img src="screens/recettes.png" alt="Recipes" width="270" />
  &nbsp;
  <img src="screens/inventaire.png" alt="Pantry" width="270" />
</div>

---

## Features

- **Recipe library**: full CRUD with full-text search (SQLite FTS5), ingredients, step-by-step instructions, difficulty, tags, prep and cook times
- **Pantry inventory**: track quantities with ±1 adjustments, expiry dates, per-category filters, and automatic low-stock alerts
- **Shopping list**: auto-generated from low-stock items
- **Secure authentication**: short-lived JWT access token (15 min) + httpOnly refresh token (7 days) with server-side revocation
- **Responsive**: collapsible sidebar on desktop, bottom navigation on mobile
- **Self-hosted**: single command to start, SQLite for storage, no external service required

---

## Stack

| | Technology |
|---|---|
| **Frontend** | React 18, TypeScript, Vite, Tailwind CSS |
| **Backend** | Go 1.22, Gin, zerolog |
| **Auth** | JWT HS256 · bcrypt cost 12 · double-token |
| **Database** | SQLite: WAL mode, FTS5, embedded migrations |
| **Proxy** | nginx 1.27: reverse proxy + SPA serving |
| **Runtime** | Docker / Podman (rootless compatible) |

The Go binary uses [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite); a pure Go SQLite driver, no CGO. The final image is built on `scratch` (zero OS).

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
LOCAL_IP=192.168.x.x                           # your local IP (see HTTPS section)
```

Or generate and inject the secret in one command:
```bash
echo "JWT_SECRET=$(openssl rand -hex 64)" >> .env
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

Open **https://localhost:3443** to access the application (accept the self-signed certificate on first visit).

### 4. Initial Setup

On the first launch, the **setup wizard** will guide you through creating the administrator account. This wizard is only available once; as soon as the first user is created, the `/setup` endpoint is permanently locked.

<div align="center">
  <img src="screens/setup.png" alt="Setup wizard" width="600" />
</div>

The setup creates an admin account with full privileges. Additional users can only be created by an admin via the protected `/auth/register` endpoint.

### 5. Start & Stop

```bash
# Stop the stack
docker compose down          # Docker
podman compose down          # Podman

# Restart (no rebuild)
docker compose up -d         # Docker
podman compose up -d         # Podman
```

> Data is persisted in a Docker volume (`home-cooking-sqlite`). Running `down` does not delete the database. To fully reset: `docker compose down -v`.

---

## HTTPS (self-signed certificate)

The application runs on **HTTPS by default** in production using a self-signed SSL certificate generated automatically during the Docker build. No manual certificate management is required.

### How it works

When running `docker compose up --build`, the frontend Dockerfile generates a 2048-bit RSA certificate valid for 10 years. Nginx listens on port 443 (HTTPS) and automatically redirects port 80 (HTTP) to HTTPS.

### Accessing from another device on your local network

By default, the certificate is only valid for `localhost` and `127.0.0.1`. To access the app from another device (phone, tablet…), declare your local IP in `.env` **before building**:

```env
LOCAL_IP=192.168.1.42
```

This IP will be included in the SSL certificate (SAN field) and in the backend CORS configuration. To find your local IP:

```bash
# Linux / macOS
ip route get 1 | awk '{print $7}'
# or
hostname -I | awk '{print $1}'
```

> ⚠️ If your local IP changes, you need to **rebuild** the image to regenerate the certificate: `docker compose up --build -d`

### Exposed ports (production)

| Port | Protocol | Behavior |
|------|----------|----------|
| `3443` | HTTPS | Main entry point |
| `3080` | HTTP | Automatically redirects to HTTPS |

These ports are configurable via `FRONTEND_PORT` and `FRONTEND_HTTP_PORT` in `.env`.

### Browser warning

On first visit, your browser will show an "insecure connection" warning — this is expected for self-signed certificates. Click "Advanced" then "Accept the risk" (wording varies by browser). The warning will not reappear for that domain.

### Development mode

Dev mode (`docker-compose.dev.yml`) stays on **HTTP port 3000**, with no workflow changes.

---

## Project Structure

```
HomeCooking/
│
├── backend/                        # Go API
│   ├── cmd/server/main.go          # Entry point, router, graceful shutdown
│   ├── internal/
│   │   ├── auth/                   # JWT, bcrypt, login handlers, admin registration
│   │   ├── config/                 # Typed config from environment
│   │   ├── db/                     # SQLite connection + embedded migrations
│   │   │   └── migrations/         # SQL files (001_init.sql, …)
│   │   ├── middleware/             # CORS, rate limiter, JWT auth, security headers, logger
│   │   ├── recipe/                 # Recipe CRUD: handler + repository
│   │   ├── setup/                  # First-launch wizard: admin account creation
│   │   └── storage/               # Pantry CRUD: handler + repository
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile                  # Multi-stage: golang:alpine → scratch
│
├── frontend/                       # React SPA
│   ├── src/
│   │   ├── pages/                  # Dashboard, Recipes, Storage, Auth, Setup
│   │   ├── components/             # Layout, UI primitives
│   │   └── lib/api.ts              # Typed HTTP client
│   ├── docker/nginx.conf           # Reverse proxy config
│   ├── package.json
│   ├── vite.config.ts
│   └── Dockerfile                  # Multi-stage: node:alpine (build) → nginx:alpine
│
├── docker-compose.yml              # Production stack
├── docker-compose.dev.yml          # Development overrides
├── .env.example                    # Environment template
└── README.md
```

---

## API

All routes are prefixed `/api/v1`. Protected routes require `Authorization: Bearer <token>`.

**Setup** (first launch only)

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| GET | `/setup/status` | | Check if setup is needed |
| POST | `/setup` | | Create first admin account (locked after use) |

**Auth**

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| POST | `/auth/login` | | Login → access + refresh token |
| POST | `/auth/refresh` | | Rotate refresh token |
| POST | `/auth/logout` | ✓ | Revoke session |
| POST | `/auth/register` | ✓ admin | Create account (admin only) |
| GET | `/me` | ✓ | Current user |

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
| Docker image | `scratch` base; no shell, no OS utilities |
| SQLite | WAL mode, `foreign_keys=on`, `busy_timeout=5000` |

---

## Configuration

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | **required** | HMAC secret, min 32 chars |
| `LOCAL_IP` | `127.0.0.1` | Local IP for SSL certificate and CORS |
| `FRONTEND_PORT` | `3443` | HTTPS host port for the UI |
| `FRONTEND_HTTP_PORT` | `3080` | HTTP host port (redirects to HTTPS) |
| `PORT` | `8080` | Internal backend port |
| `ENV` | `production` | `production` or `development` |
| `DB_PATH` | `/data/home-cooking.db` | SQLite file path |
| `CORS_ORIGINS` | auto | Computed from `LOCAL_IP` and `FRONTEND_PORT` |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `7d` | Refresh token lifetime |
| `RATE_LIMIT_RPS` | `20` | Requests/second per IP |
| `RATE_LIMIT_BURST` | `40` | Burst allowance per IP |

---

## Contributing

1. Fork the repo
2. Create a branch: `git checkout -b feat/my-feature`
3. Commit: `git commit -m "feat: describe your change"`
4. Open a Pull Request

Please open an issue first for any significant change so we can align on approach.

---

## License

MIT - see [LICENSE](https://github.com/Kitslap/HomeCooking?tab=MIT-1-ov-file) for details.

---

<div align="center">
Made for home cooks who like clean code.
</div>
