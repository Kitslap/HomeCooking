# Cooking Home — Backend Gateway (Go)

API REST sécurisée alimentant le frontend React.  
Stack : **Go 1.22 · Gin · JWT · bcrypt · SQLite (CGO-free)**

## Démarrage rapide

```bash
# 1. Prérequis — Go 1.22+
go version   # doit afficher go1.22+

# 2. Cloner et entrer dans le dossier backend
cd backend/

# 3. Configurer l'environnement
cp .env.example .env
# ⚠️  Obligatoire : générer un vrai secret JWT
openssl rand -hex 64   # copier la sortie dans JWT_SECRET du .env

# 4. Installer les dépendances
go mod tidy

# 5. Lancer en développement
make run
# ou directement :
go run ./cmd/server

# Le serveur écoute sur http://localhost:8080
```

## Structure du projet

```
backend/
├── cmd/server/main.go              — point d'entrée, router Gin, graceful shutdown
├── internal/
│   ├── config/config.go            — chargement et validation des env vars
│   ├── middleware/
│   │   ├── logger.go               — logs JSON structurés (zerolog)
│   │   ├── cors.go                 — CORS strict par whitelist d'origines
│   │   ├── ratelimiter.go          — token bucket par IP (golang.org/x/time/rate)
│   │   ├── jwt.go                  — validation JWT sur les routes protégées
│   │   └── security.go             — en-têtes HTTP sécurité (CSP, HSTS…)
│   ├── auth/
│   │   ├── jwt.go                  — génération/validation access + refresh tokens
│   │   └── handler.go              — /auth/register, /login, /refresh, /logout
│   └── db/
│       ├── db.go                   — init SQLite, migrations embedded au démarrage
│       └── migrations/
│           └── 001_init.sql        — schéma complet (users, recipes, storage, FTS5)
├── .env.example                    — variables d'environnement documentées
├── Makefile                        — build, run, test, lint, clean
└── go.mod
```

## Routes disponibles

| Méthode | Path                    | Auth  | Description              |
|---------|-------------------------|-------|--------------------------|
| GET     | `/health`               | —     | Healthcheck              |
| POST    | `/api/v1/auth/register` | —     | Inscription              |
| POST    | `/api/v1/auth/login`    | —     | Connexion                |
| POST    | `/api/v1/auth/refresh`  | Cookie| Renouvellement token     |
| POST    | `/api/v1/auth/logout`   | Cookie| Déconnexion              |
| GET     | `/api/v1/me`            | JWT   | Profil utilisateur       |
| …       | `/api/v1/recipes/*`     | JWT   | *(prochaine brique)*     |
| …       | `/api/v1/storage/*`     | JWT   | *(prochaine brique)*     |

## Sécurité

- **JWT double-token** : access (15 min) + refresh httpOnly cookie (7 jours), rotation à chaque refresh
- **bcrypt coût 12** : ~250ms par hash, résistant au brute-force
- **Rate limiting** : 20 req/s par IP avec rafale de 40 (configurable)
- **CORS whitelist** : aucun wildcard, origines explicites
- **En-têtes HTTP** : CSP, HSTS (prod), X-Frame-Options, Referrer-Policy…
- **SQLite WAL** : transactions sûres, foreign keys activées
- **Graceful shutdown** : les requêtes en cours se terminent avant l'arrêt

## Prochaines briques

```bash
# Depuis le schéma d'architecture dans le frontend, cliquer sur :
# ▸ "Recipes API"  → génère internal/recipe/handler.go + repository.go
# ▸ "Storage API"  → génère internal/storage/handler.go + repository.go
```
