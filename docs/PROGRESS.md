# PROGRESS — журнал выполнения

Короткие записи по завершении каждой фазы: что сделано, что проверено.

<!-- Новые записи добавляются сверху. -->

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
