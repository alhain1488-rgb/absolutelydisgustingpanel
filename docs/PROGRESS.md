# PROGRESS — журнал выполнения

Короткие записи по завершении каждой фазы: что сделано, что проверено.

<!-- Новые записи добавляются сверху. -->

## Фаза 3 — Управление серверами

**Сделано.**
- `internal/ssh` — интерфейс `Runner` (Run/Upload) + реальный клиент на
  `golang.org/x/crypto/ssh` (per-op dial, key/password auth, upload через
  `cat > file`); парсинг метрик (`/proc/meminfo`, `df`, дельта `/proc/stat`,
  uptime), детект статуса Xray (`systemctl is-active`), restart, `xray -test`,
  ping; пакет `sshtest` с scriptable `FakeRunner` для юнит-тестов.
- `internal/servers` — модель + репозиторий (CRUD, update check/sync-результата);
  сервис: create/update с шифрованием SSH-секрета (AES-GCM), `Check`
  (ping→status, DNS→IP, гео best-effort), `Stats`, `RestartXray`; фабрика
  раннеров и резолвер инъектируемы (тесты на моках). Гео — `Geolocator`
  (реализация ip-api, опциональна).
- `internal/httpapi` — маршруты `/api/servers` (list/create/get/update/delete/
  check/restart-xray/stats) под JWT + аудит; `/api/logs` (журнал аудита).

**Проверено.**
- `go test ./...` — зелёно: парсинг метрик (CPU 25%/MEM 50%/DISK), статус Xray,
  restart-ошибка, ping; сервис — шифрование секрета, check online/offline+geo,
  stats, update без перезаписи секрета; HTTP CRUD-флоу + 401/404.
- `go vet`, `gofmt -l .`, `golangci-lint run` — чисто (0 issues).

## Фаза 2 — Аутентификация и аудит

**Сделано.**
- `internal/auth` — репозиторий админа; логин по bcrypt; JWT-сессии
  (`TokenManager`: полный `auth`-токен 12h + короткий `2fa`-pending 5m);
  TOTP setup/enable/verify (`pquerna/otp`, QR через `skip2/go-qrcode`),
  секрет хранится зашифрованным (AES-GCM); middleware Bearer-авторизации;
  fixed-window rate-limiter; bootstrap первичного админа из env (идемпотентно).
- `internal/audit` — `Recorder`: запись админ-действий в `audit_logs` (кто,
  действие, объект, IP, UA, JSON-детали) и листинг (newest-first).
- `internal/httpapi` — маршруты `/api/auth/*`: login, 2fa/verify, 2fa/setup,
  2fa/enable, logout, me; аудит логина/2FA/logout; собственный trusted-proxy
  `realIP` middleware вместо deprecated chi RealIP.
- `cmd/panel` — сборка cipher/auth/audit, bootstrap админа, rate-limit логина.

**Проверено.**
- `go test ./...` — зелёно: login success/fail, полный цикл TOTP
  (setup→enable→login-need2fa→verify), bootstrap idempotent, rate-limit
  (window/reset), audit record+list, HTTP-флоу (401/200, `/me` с токеном и без).
- `go vet ./...`, `gofmt -l .`, `golangci-lint run` — чисто (0 issues).

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
