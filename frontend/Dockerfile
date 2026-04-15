# ── Stage 1 : Build ──────────────────────────────────────────────────────────
# Compile les sources TypeScript/React avec Vite
FROM node:22-alpine AS build

WORKDIR /app

# Copier d'abord les fichiers de dépendances pour profiter du cache Docker
COPY package.json package-lock.json ./
RUN npm ci

# Copier les sources et compiler
COPY . .
RUN npm run build

# ── Stage 2 : Runtime ────────────────────────────────────────────────────────
# Sert le bundle compilé via nginx (HTTP + HTTPS auto-signé)
FROM nginx:1.27-alpine

# Génération du certificat auto-signé au build (valable 10 ans)
# Le CN utilise un wildcard pour accepter localhost et toute IP locale.
RUN apk add --no-cache openssl \
    && mkdir -p /etc/nginx/ssl \
    && openssl req -x509 -nodes -days 3650 \
         -newkey rsa:2048 \
         -keyout /etc/nginx/ssl/selfsigned.key \
         -out    /etc/nginx/ssl/selfsigned.crt \
         -subj   "/C=FR/ST=Local/L=Local/O=HomeCooking/CN=localhost" \
         -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:192.168.1.18" \
    && apk del openssl

RUN rm -rf /etc/nginx/conf.d/*
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html/

EXPOSE 80 443
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- --no-check-certificate https://localhost/health-frontend || exit 1
CMD ["nginx", "-g", "daemon off;"]
