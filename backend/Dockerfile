# ── Build stage ───────────────────────────────────────────────────────
FROM golang:1.24-alpine AS build
WORKDIR /src

# CGO is needed by the SQLite driver; enable it and pull build deps.
RUN apk add --no-cache gcc musl-dev
ENV CGO_ENABLED=1

COPY go.mod go.sum* ./
RUN go mod download || true

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
