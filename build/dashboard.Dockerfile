# ── Build stage ──────────────────────────────────────────────────────────────
# Phase 1: serves a static placeholder page.
# Phase 7: replaced with a full Vite + React build.
FROM node:20-alpine AS builder

WORKDIR /app

# Phase 7 will add: COPY web/package*.json ./ && RUN npm ci
# Phase 7 will add: COPY web/ . && RUN npm run build

# For Phase 1, copy the static placeholder directly.
COPY web/ ./dist/

# ── Final stage ───────────────────────────────────────────────────────────────
FROM nginx:1.27-alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY web/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 3000
