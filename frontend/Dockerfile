# ── Build stage ───────────────────────────────────────────────────────
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci || npm install
COPY . .
RUN npm run build

# ── Runtime: static files served by Caddy ─────────────────────────────
# The built SPA is copied out by the caddy service via a shared volume,
# but we also expose it as its own tiny static image for flexibility.
FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
