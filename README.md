# Xray Panel

Веб-панель для личного централизованного управления несколькими VPS с Xray:
серверы, inbound-ы разных протоколов, клиенты и их подписки. Управление нодами —
по SSH через генерацию `config.json`. Без биллинга, лимитов трафика и
мультиарендности.

Стек: **Go** · **React + TypeScript** · **SQLite** · **Docker Compose**.

> Проект в активной разработке. Порядок реализации — `docs/ROADMAP.md`,
> спецификация — `docs/SPEC.md`, архитектура — `docs/ARCHITECTURE.md`.
> Полный README (быстрый старт, переменные окружения, troubleshooting)
> оформляется в Фазе 7.

## Быстрый старт (dev)

```bash
# Backend
cd backend && go test ./... && go run ./cmd/panel

# Frontend
cd frontend && npm ci && npm run dev

# Всё вместе
cp .env.example .env   # заполнить секреты
docker compose up --build
```

## Документация

- `docs/SPEC.md` — что строим и почему.
- `docs/ROADMAP.md` — план по фазам.
- `docs/ARCHITECTURE.md` — архитектурные решения.
- `docs/ERD.md` — модель данных.
- `docs/openapi.yaml` — контракт REST API.
- `docs/PROGRESS.md` — журнал выполнения.
- `docs/QUESTIONS.md` — открытые вопросы к человеку.
