# PROGRESS — журнал выполнения

Короткие записи по завершении каждой фазы: что сделано, что проверено.

<!-- Новые записи добавляются сверху. -->

## Фаза 1 — Фундамент backend

**Сделано.**
- `internal/config` — чтение и валидация env; отсутствие обязательной переменной
  или неверная длина `PANEL_ENCRYPTION_KEY` → понятная фатальная ошибка старта.
- `internal/crypto` — AES-256-GCM (`nonce||ct||tag`, base64) + bcrypt (cost 12).
- `internal/logging` — структурный `slog` JSON-логгер с уровнями.
- `internal/migrations` — встроенный (`embed`) аддитивный раннер миграций,
  учёт в `schema_migrations`; `0001_init` — полная схема из `SPEC.md §4`.
- `internal/db` — открытие SQLite (pure-Go `modernc.org/sqlite`, без CGO),
  WAL + foreign_keys + busy_timeout, авто-миграции при старте, in-memory для тестов.
- `internal/httpapi` — `chi`-роутер, middleware (request id, real ip, recover,
  timeout, request log), `/healthz`, `/readyz` (пинг БД).
- `cmd/panel` — сборка зависимостей, graceful shutdown.

**Решение по слою БД.** Вместо sqlc выбраны рукописные типобезопасные репозитории
на `database/sql` (portable-SQL, без codegen-тулчейна в автономной среде) —
переносимость на Postgres сохраняется. Драйвер — pure-Go `modernc.org/sqlite`
(сборка и тесты без CGO).

**Проверено.**
- `go test ./...` — зелёно (миграции на чистой БД, идемпотентность, FK,
  AES round-trip/tamper/wrong-key, bcrypt verify, health/ready).
- `go vet ./...` и `gofmt -l .` — чисто.
- Локальный запуск: БД мигрирует, `curl /healthz` и `/readyz` → 200.

## Фаза 0 — Архитектура и каркас

**Сделано.**
- Реорганизация репозитория в целевую структуру (`SPEC.md §11`): `docs/`,
  `backend/`, `frontend/`, `deploy/`, `test/`, `.claude/`.
- `docs/ARCHITECTURE.md` — решения (SSH-конфиг-механизм, реестр протоколов,
  модель доступа через inbound-ы, шифрование секретов, слой БД).
- `docs/ERD.md` — mermaid ER-диаграмма из `SPEC.md §4` + инварианты.
- `docs/openapi.yaml` — design-first контракт всех эндпоинтов из `SPEC.md §10`.
- Пустые `docs/PROGRESS.md`, `docs/QUESTIONS.md` (с фиксацией решения по Hysteria2).
- Backend-скелет: `go.mod`, `cmd/panel/main.go`, дерево `internal/*`, `Dockerfile`.
- Frontend-скелет: Vite + React + TS + Tailwind, `App`, тест, ESLint, `Dockerfile`.
- Инфраструктура: `docker-compose.yml` (backend + frontend + Caddy),
  `deploy/Caddyfile` (TLS + reverse proxy), `.env.example`, `.gitignore`, `README`.

**Проверено.**
- `go build ./...` — OK.
- `npm run build` — OK; `npm run test` — 1 passed; `npm run lint` — чисто.
- `docker compose config` — валиден.
