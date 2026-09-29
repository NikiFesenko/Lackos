# ── Stage 1: Build ─────────────────────────────────────────────────────────
# Phase 7: Full Vite + React 18 + TypeScript build.
FROM node:20-alpine AS builder

WORKDIR /app

# Install deps separately so the layer is cached unless package.json changes.
COPY package.json package-lock.json* ./
RUN npm ci --prefer-offline

COPY . .
RUN npm run build

# ── Stage 2: Serve ──────────────────────────────────────────────────────────
FROM nginx:1.27-alpine AS runtime

# Remove default nginx content.
RUN rm -rf /usr/share/nginx/html/*

# Copy Vite build output.
COPY --from=builder /app/dist /usr/share/nginx/html

# Nginx config: serve SPA (try_files), proxy /api to adminapi service.
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
