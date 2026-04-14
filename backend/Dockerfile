# ─────────────────────────────────────────────────────────────────────────────
# Dockerfile — Home Cooking Backend (Go)
# Build multi-stage :
#   Stage 1 "builder" : compile le binaire statique (CGO_ENABLED=0)
#   Stage 2 "runtime" : image scratch (~0 Mo d'OS) — surface d'attaque minimale
# ─────────────────────────────────────────────────────────────────────────────

# ── Stage 1 : Build ──────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

# Métadonnées de build
LABEL stage="builder"

# Dépendances système minimales pour la compilation
# git : nécessaire pour go mod download sur certains modules
# ca-certificates : pour les connexions HTTPS lors du téléchargement des modules
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copie de go.mod uniquement — go.sum sera généré automatiquement par go mod download
# (go.sum n'existe pas encore si go mod tidy n'a pas été lancé en local)
COPY go.mod ./
RUN go mod download

# Copie du code source
COPY . .

# Génère go.sum et nettoie les dépendances inutiles
RUN go mod tidy

# Compilation statique — CGO_ENABLED=0 car modernc.org/sqlite est pur Go
# -ldflags "-s -w" : supprime les infos de debug pour réduire la taille (~30%)
# -trimpath         : supprime les chemins locaux du binaire (reproductibilité)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -o /app/bin/server \
    ./cmd/server

# Vérification que le binaire est bien statique (outil "file" disponible dans alpine)
RUN apk add --no-cache file && \
    file /app/bin/server | grep -q "statically linked" && \
    echo "✓ Binaire statique OK" || \
    (echo "⚠️  Le binaire n'est pas statique !" && exit 1)


# ── Stage 2 : Runtime ─────────────────────────────────────────────────────────
# "scratch" = image vide, zéro OS, zéro shell, zéro vulnérabilité système
FROM scratch AS runtime

# Copie des certificats CA (pour les appels HTTPS éventuels depuis le backend)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copie des timezones (pour les calculs de date/expiration)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copie du binaire compilé uniquement
COPY --from=builder /app/bin/server /server

# Exposition du port applicatif (documentaire — override via docker-compose)
EXPOSE 8080

# Utilisateur non-root : impossible dans scratch (pas de /etc/passwd)
# La sécurité est assurée par l'absence totale d'OS et de shell
# → aucun exploit ne peut escalader vers un shell inexistant

# Volume pour la base SQLite persistante
VOLUME ["/data"]

# Variables d'environnement par défaut (surchargées par docker-compose)
ENV PORT=8080 \
    ENV=production \
    DB_PATH=/data/home-cooking.db

# Point d'entrée unique — pas de shell intermédiaire (forme exec)
ENTRYPOINT ["/server"]
