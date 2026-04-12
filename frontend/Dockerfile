# Frontend déjà compilé (bundle Vite auto-contenu) — pas de build Node nécessaire
FROM nginx:1.27-alpine

RUN rm -rf /etc/nginx/conf.d/*
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY dist/ /usr/share/nginx/html/

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost/health-frontend || exit 1
CMD ["nginx", "-g", "daemon off;"]
