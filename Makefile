# ─────────────────────────────────────────────────────────────────────────────
# Makefile racine — Home Cooking
# Raccourcis Docker Compose pour le développement et la production.
# ─────────────────────────────────────────────────────────────────────────────

# Auto-détection Docker vs Podman
# Override manuel : make start COMPOSE="podman compose"
DOCKER := $(shell command -v podman 2>/dev/null && echo "podman" || echo "docker")
COMPOSE      := $(DOCKER) compose -f docker-compose.yml
COMPOSE_DEV  := $(DOCKER) compose -f docker-compose.yml -f docker-compose.dev.yml

.PHONY: all setup start stop restart build logs clean ps shell-backend shell-frontend

## setup — initialise l'environnement (copie .env si absent)
setup:
	@if [ ! -f .env ]; then \
	  cp .env.example .env; \
	  echo ""; \
	  echo "╔══════════════════════════════════════════════════════╗"; \
	  echo "║  ⚠️   .env créé depuis .env.example                  ║"; \
	  echo "║  👉  Éditez JWT_SECRET avec :                        ║"; \
	  echo "║       openssl rand -hex 64                           ║"; \
	  echo "╚══════════════════════════════════════════════════════╝"; \
	  echo ""; \
	fi

## build — compile les deux images Docker
build: setup
	$(COMPOSE) build --no-cache

## start — démarre en production (arrière-plan)
start: setup
	$(COMPOSE) up -d --build
	@echo ""
	@echo "✓ Home Cooking démarré → http://localhost:$$(grep FRONTEND_PORT .env | cut -d= -f2 || echo 3000)"
	@echo "  Backend API  → http://localhost:$$(grep FRONTEND_PORT .env | cut -d= -f2 || echo 3000)/api/v1"
	@echo ""

## dev — démarre en mode développement (logs colorés, backend sur :8080)
dev: setup
	$(COMPOSE_DEV) up --build

## stop — arrête les containers sans supprimer les données
stop:
	$(COMPOSE) down

## restart — redémarre les services
restart:
	$(COMPOSE) restart

## logs — affiche les logs en temps réel (Ctrl-C pour quitter)
logs:
	$(COMPOSE) logs -f

## logs-backend — logs du backend uniquement
logs-backend:
	$(COMPOSE) logs -f backend

## logs-frontend — logs du frontend (nginx) uniquement
logs-frontend:
	$(COMPOSE) logs -f frontend

## ps — état des containers
ps:
	$(COMPOSE) ps

## shell-backend — ouvre un shell dans le container backend
## ⚠️  Impossible avec l'image scratch (pas de shell) — utilisez le mode dev
shell-backend:
	@echo "⚠️  L'image de production (scratch) n'a pas de shell."
	@echo "   En mode dev : make dev puis dans un autre terminal :"
	@echo "   docker compose exec backend sh"

## shell-frontend — ouvre un shell dans le container nginx
shell-frontend:
	$(COMPOSE) exec frontend sh

## clean — supprime containers, images et volumes (⚠️  données SQLite perdues)
clean:
	$(COMPOSE) down -v --rmi all
	@echo "✓ Nettoyage complet"

## backup-db — sauvegarde la base SQLite dans ./backups/
backup-db:
	@mkdir -p backups
	@TS=$$(date +%Y%m%d_%H%M%S); \
	 docker run --rm \
	   -v home-cooking-sqlite:/data \
	   -v $$(pwd)/backups:/backups \
	   alpine sh -c "cp /data/home-cooking.db /backups/home-cooking_$$TS.db" && \
	 echo "✓ Sauvegarde : backups/home-cooking_$$TS.db"

## help
help:
	@echo ""
	@echo "Usage: make <target>"
	@grep -E '^## ' Makefile | sed 's/## /  /'
	@echo ""
