# ── Stage 1 : Build React/Vite ────────────────────────────────────────────────
FROM node:22-alpine AS build

WORKDIR /app

# Copier uniquement les manifestes pour exploiter le cache Docker
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts

# Copier le code source et compiler
COPY . .
RUN npm run build

# ── Stage 2 : Serveur Nginx (image finale légère) ────────────────────────────
FROM nginx:1.27-alpine

RUN rm -rf /etc/nginx/conf.d/*
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf

# Récupérer le bundle compilé depuis le stage de build
COPY --from=build /app/dist/ /usr/share/nginx/html/

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost/health-frontend || exit 1
CMD ["nginx", "-g", "daemon off;"]
