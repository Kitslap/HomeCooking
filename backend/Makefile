# ── Cooking Home — Makefile ───────────────────────────────────────────────
BINARY    := cooking-home-server
BUILD_DIR := ./bin
CMD       := ./cmd/server

.PHONY: all build run dev test tidy lint clean migrate

all: build

## build — compile le binaire de production
build:
	@echo "→ Build $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY) $(CMD)
	@echo "✓ $(BUILD_DIR)/$(BINARY)"

## run — lance le serveur (nécessite .env)
run: build
	@$(BUILD_DIR)/$(BINARY)

## dev — hot-reload avec Air (go install github.com/air-verse/air@latest)
dev:
	@which air > /dev/null 2>&1 || (echo "Installez Air: go install github.com/air-verse/air@latest" && exit 1)
	air -c .air.toml

## test — lance tous les tests
test:
	go test ./... -v -race -cover

## tidy — nettoie et met à jour go.sum
tidy:
	go mod tidy

## lint — analyse statique (nécessite golangci-lint)
lint:
	golangci-lint run ./...

## clean — supprime les artefacts de build
clean:
	@rm -rf $(BUILD_DIR)
	@echo "✓ Clean"

## migrate — applique les migrations (via goose si installé, sinon embedded)
migrate:
	@echo "→ Migrations via le serveur au démarrage (embedded)"

## help — affiche cette aide
help:
	@echo ""
	@echo "Usage: make <target>"
	@grep -E '^## ' Makefile | sed 's/## /  /'
	@echo ""
