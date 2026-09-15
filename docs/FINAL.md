# MORIS Final Operations Guide

## Services

- `moris-api`: HTTP API, health/readiness, metrics, admin dashboard.
- `moris-worker`: queue consumers, yt-dlp/FFmpeg processing, Telegram delivery.
- `moris-bot`: Telegram long polling and inline UI.
- Redis: distributed queue and rate limiting in `MORIS_QUEUE_MODE=redis`.
- PostgreSQL: schema/migrations are supplied for deployments that replace the dependency-free file repository with a SQL adapter.

## Required production settings

Set `MORIS_TELEGRAM_TOKEN` and a strong `MORIS_ADMIN_TOKEN`. Use Redis mode for multiple workers. Keep `/data/moris` on persistent storage. Never commit `.env`.

## Verification

Run:

```bash
go test -race ./...
go vet ./...
go build ./cmd/api ./cmd/worker ./cmd/bot
```

Then, with Docker available:

```bash
docker compose config
docker compose up --build -d
docker compose up --scale worker=8 -d
```

## Media and music

MORIS uses `yt-dlp` for public/authorized source extraction and FFmpeg for conversion. Optional AudD recognition is enabled by `MORIS_AUDD_TOKEN`; lyrics use LRCLIB.

## API highlights

- `POST /api/v1/media/analyze`
- `POST /api/v1/jobs`
- `GET /api/v1/jobs/{id}`
- `POST /api/v1/jobs/{id}/cancel?user_id=...`
- `POST /api/v1/jobs/{id}/retry` (admin)
- `GET/PUT /api/v1/users/{id}`
- `GET /api/v1/users/{id}?history=1`
- `POST /api/v1/music/recognize`
- `GET /api/v1/music/lyrics?artist=...&title=...`
- `GET /admin`
- `GET /health`, `/ready`, `/metrics`

## Scope

Only download content the operator/user is authorized to access. Provider terms, copyright, privacy, authentication and regional restrictions remain applicable.
