# ── Build stage ───────────────────────────────────────────────────────
FROM golang:1.25-alpine AS build
WORKDIR /src

# Pure-Go SQLite driver (modernc.org/sqlite) — no CGO required.
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/panel ./cmd/panel

# ── Runtime stage ─────────────────────────────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache ca-certificates openssh-client && \
    adduser -D -u 10001 panel
WORKDIR /app
COPY --from=build /out/panel /app/panel
USER panel
EXPOSE 8080
ENTRYPOINT ["/app/panel"]
