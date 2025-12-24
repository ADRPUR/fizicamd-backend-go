# FizicaMD Backend (Go)

Go implementation mirroring the Java/Python backend endpoints and database schema.

## Setup

```bash
go mod download
```

Create `.env` from `.env.example` (auto-loaded on startup), then run:

```bash
./scripts/run-dev.sh
```

## Docker

```bash
docker compose up --build
```

## Migrations

SQL migrations live in `migrations/` and are applied on startup.

## Notes
- WebSocket metrics endpoint: `/ws/metrics?token=...`
- API base: `/api`
- Default port: `8080`
