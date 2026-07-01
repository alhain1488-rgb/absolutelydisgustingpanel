# PROGRESS — журнал выполнения

Короткие записи по завершении каждой фазы: что сделано, что проверено.

<!-- Новые записи добавляются сверху. -->

## Фаза 7 — Сборка, документация, доставка

**Сделано.**
- Swagger: `/swagger` отдаёт интерактивный UI (swagger-ui-dist с CDN),
  `/swagger/openapi.yaml` — встроенный (`go:embed`) контракт. Копия
  `docs/openapi.yaml` лежит в `backend/internal/httpapi/openapi.yaml` для
  эмбеда (держать в синхроне при изменении контракта).
- `README.md` — полный: назначение, возможности, требования, быстрый старт,
  таблица переменных окружения, подключение реального VPS, добавление протокола
  через реестр, Swagger, troubleshooting, что делает человек.
- `scripts/smoke.sh` — смоук из `SPEC.md §12`: healthz → login → сервер (мок) →
  inbound → клиент → выдача → `GET /sub/{token}` возвращает непустую валидную
  подписку.
- Инфраструктура доставки: `docker-compose.yml` (backend + frontend-статика +
  Caddy TLS/reverse-proxy), `deploy/Caddyfile` (маршрутизация `/api` `/sub`
  `/swagger` на backend, SPA-fallback на статику).

**Проверено.**
- `go build/vet/test`, `gofmt -l .`, `golangci-lint run` — чисто; `npm run
  build/test/lint` — чисто; `docker compose config` — валиден.
- Ручной прогон против запущенного backend: `/swagger/` → 200,
  `/swagger/openapi.yaml` → 200 (application/yaml); `scripts/smoke.sh` → SMOKE OK
  (валидный `vless://…`).

**Definition of Done.** Полный проходной путь (`cp .env.example .env` → запуск →
вход c опц. 2FA → сервер → inbound-ы → клиент → рабочая subscription-ссылка + QR
→ дашборд, Swagger на `/swagger`) реализован и покрыт тестами. Живой
`docker compose up` на проде и наведение на реальные VPS — за человеком
(Docker-демон в среде разработки недоступен; см. `docs/QUESTIONS.md`).

## Фаза 6 — Frontend

**Сделано.**
- Каркас: Vite + React + TS + Tailwind + React Router + TanStack Query;
  `AuthProvider` (login → опц. TOTP-шаг → JWT в localStorage, `/me`),
  `ThemeProvider` (dark/light через класс на `html`, сохраняется),
  типизированный API-клиент, UI-примитивы (Button/Input/Select/Card/Modal/…).
- Страницы: **Login** (+ шаг 2FA), **Dashboard** (счётчики + карточки серверов),
  **Servers** (список + добавление, «Check»), **ServerDetail** (метрики CPU/RAM/
  диск, рестарт Xray, CRUD inbound-ов с формой протокола/транспорта/Reality),
  **Clients** (список + создание), **ClientDetail** (выдача доступа чекбоксами по
  серверам, subscription-ссылка, QR, скачивание, rotate-token, enable/disable),
  **Settings** (домен/базовый URL, тема, настройка и включение 2FA), **Logs**
  (таблица аудита).
- Типы API синхронизированы с `docs/openapi.yaml`; dev-прокси Vite на backend.

**Проверено.**
- `npm run build` (tsc + vite) — OK; `npm run test` (vitest) — 1 passed;
  `npm run lint` (eslint) — чисто (0 warnings/errors).

## Фаза 5 — Клиенты, доступы и подписка

**Сделано.**
- `internal/clients` — модель + репозиторий (CRUD, grants `client_inbounds`
  транзакцией, токен) + сервис: генерация UUID/пароля/subscription-token,
  enable/disable, выдача/отзыв доступа, rotate-token.
- `internal/subscription` — сборка base64-подписки из всех разрешённых и
  включённых inbound-ов клиента на всех серверах через реестр протоколов; список
  raw-URI для админки; QR (PNG). Отключённый клиент → пустая подписка.
- `internal/sync` — движок: сборка полного `config.json` сервера из включённых
  inbound-ов и включённых выданных клиентов; пуш по SSH: mkdir → backup →
  upload → `xray -test` (с восстановлением бэкапа при провале) → restart;
  идемпотентность по SHA-256 (пустые пуши пропускаются); запись `last_sync_*`.
- `internal/httpapi` — `/api/clients/*` (CRUD, enable/disable, inbounds,
  rotate-token, links, qrcode, config), публичный rate-limited `GET /sub/{token}`,
  `/api/dashboard/summary`, `/api/settings` (GET/PUT); best-effort авто-sync в
  фоне при изменениях inbound-ов/доступов/клиентов.

**Проверено.**
- `go test ./...` — зелёно: клиенты (identity, grants replace, rotate,
  enable/disable), subscription e2e (2 inbound-а на разных серверах → 2 URI;
  disable → пусто; revoke → 1), QR PNG; sync (push-шаги, идемпотентность,
  восстановление бэкапа при провале `xray -test`, запись ошибки).
- `go vet`, `gofmt -l .`, `golangci-lint run` — чисто (0 issues).
- Ручной e2e против запущенного backend: login → сервер → inbound (ключи Reality
  сгенерированы) → клиент → выдача → `/sub` отдаёт валидный `vless://…` +
  QR (image/png) + dashboard.

## Фаза 4 — Inbound-ы и реестр протоколов

**Сделано.**
- `internal/protocols` — интерфейс `Protocol` (BuildInbound/BuildLink) + реестр;
  адаптеры **VLESS** (Reality и TLS), **VMess**, **Trojan**, **Shadowsocks**;
  общая схема параметров транспорта/безопасности (`Stream`) и протокол-настроек
  (`Settings`) в JSON-полях (без ALTER TABLE); генерация X25519-ключей Reality
  (base64 raw-URL, как `xray x25519`), shortId, SS-пароля. `hysteria2` —
  зарегистрированная-как-документация заглушка (`ErrNotImplemented`, в реестр
  MVP не добавлена).
- `internal/inbounds` — модель + репозиторий (CRUD в рамках сервера) + сервис:
  валидация протокола/порта, автогенерация ключей Reality/пароля SS на бэкенде,
  дефолты; параметры хранятся в `settings_json`/`stream_settings_json`.
- `internal/xrayconfig` — сборка полного `config.json` из inbound-ов и выданных
  клиентов через реестр (log + inbounds + freedom/blackhole outbounds).
- `internal/httpapi` — `/api/servers/{id}/inbounds` (list/create) и
  `/api/inbounds/{id}` (get/update/delete) под JWT + аудит.
- `test/xray-node` — Docker-образ (Go + xray-core) и интеграционный тест под
  тегом `integration` (`TestGeneratedConfig_AcceptedByXray`): конфиг с
  VLESS Reality + VMess + Trojan проходит `xray -test`.

**Проверено.**
- `go test ./...` — зелёно: реестр (каждый адаптер строит валидный inbound и
  корректный URI; publicKey не утекает в конфиг Reality), генерация ключей,
  сервис inbound-ов (Reality/SS автогенерация, валидация, update сохраняет ключи),
  сборка `config.json` (3 протокола).
- `go vet`, `gofmt -l .`, `golangci-lint run` — чисто (0 issues).
- Живой `xray -test` — вне автономной среды (нет Docker-демона); инфраструктура
  и тест готовы, прогон человеком/CI (см. `docs/QUESTIONS.md`).

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
