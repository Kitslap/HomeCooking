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
# Sert le bundle compilé via nginx
FROM nginx:1.27-alpine

RUN rm -rf /etc/nginx/conf.d/*
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html/

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost/health-frontend || exit 1
CMD ["nginx", "-g", "daemon off;"]
