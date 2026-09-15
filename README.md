# MORIS — Universal Media Intelligence Bot

**Created & Designed by Moris**  
Telegram: `https://telegram.me/moris3245`

MORIS is a production-oriented Telegram media intelligence and processing platform written in Go. It accepts public or otherwise authorized media URLs, analyzes available formats, queues work, downloads with `yt-dlp`, converts with FFmpeg, stores results, and delivers media back to Telegram.

> **Important:** MORIS is designed for lawful/public/authorized content only. Users are responsible for copyright, privacy, provider terms, and access permissions. Support for any third-party platform can change when that platform changes its rules, API, authentication, or anti-bot mechanisms.


**GitHub Repository:** https://github.com/moris-ir/Moris-downloader
## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Repository Layout](#repository-layout)
- [Requirements](#requirements)
- [Quick Start](#quick-start)
- [Real VPS / Server Deployment](#real-vps--server-deployment)
- [Production Environment](#production-environment)
- [Docker Operations](#docker-operations)
- [Telegram Bot Setup](#telegram-bot-setup)
- [Admin Panel](#admin-panel)
- [Monitoring](#monitoring)
- [API](#api)
- [Storage](#storage)
- [Scaling Workers](#scaling-workers)
- [Backups](#backups)
- [Updates](#updates)
- [Troubleshooting](#troubleshooting)
- [Security](#security)
- [Testing](#testing)
- [Development](#development)
- [Roadmap](#roadmap)
- [License](#license)

## Features

### Telegram

- Long polling bot
- `/start` and `/help`
- URL detection from normal messages
- Inline keyboards
- Video / Audio workflow selection
- Quality selection
- Job status and delivery
- Callback query handling

### Media Intelligence

- URL validation and platform classification
- YouTube
- Instagram
- TikTok
- Twitch
- Kick
- X / Twitter
- Facebook
- Reddit
- Pinterest
- SoundCloud
- Vimeo
- Generic `yt-dlp` compatible sources
- Metadata extraction
- Source format inspection
- Best / 1080p / 720p / 480p / 360p video selection
- MP4 / WebM / MKV video targets
- M4A / MP3 / WAV / FLAC audio targets
- FFmpeg conversion

### Queue & Workers

- Redis-backed queue
- In-memory queue for local tests
- Independent API and worker processes
- Worker concurrency
- Retry with backoff
- Timeout and cancellation-aware processing
- Per-user Redis rate limiting
- Cleanup of old job workspaces
- Horizontal worker scaling

### Users & Jobs

- Persistent local repository
- User registration/activity tracking
- Download and failure counters
- Plan/status fields
- User settings
- Job history
- Job cancellation
- Admin retry
- Output size tracking

### Music Intelligence

- Optional AudD recognition integration
- LRCLIB lyrics integration
- Provider interfaces designed for additional recognition/lyrics providers

### Operations

- `/health`
- `/ready`
- `/metrics`
- Prometheus configuration
- Grafana service in Compose
- Admin dashboard
- Docker Compose production layout
- CI workflow
- Race tests
- Go vet/build validation

## Architecture

```text
                         ┌────────────────────┐
                         │      Telegram      │
                         └─────────┬──────────┘
                                   │
                                   ▼
                         ┌────────────────────┐
                         │     MORIS Bot      │
                         │  Long Polling UI   │
                         └─────────┬──────────┘
                                   │ HTTP
                                   ▼
┌───────────────┐        ┌────────────────────┐
│ Admin / API   │───────►│    MORIS API       │
└───────────────┘        └─────────┬──────────┘
                                   │
                                   ▼
                         ┌────────────────────┐
                         │       Redis        │
                         │ Queue + RateLimit  │
                         └─────────┬──────────┘
                                   │
                         ┌─────────┴─────────┐
                         ▼                   ▼
                 ┌──────────────┐    ┌──────────────┐
                 │   Worker 1   │    │   Worker N   │
                 └──────┬───────┘    └──────┬───────┘
                        │                   │
                        └─────────┬─────────┘
                                  ▼
                       ┌─────────────────────┐
                       │      yt-dlp         │
                       │       FFmpeg        │
                       └─────────┬───────────┘
                                 ▼
                       ┌─────────────────────┐
                       │ Persistent Storage  │
                       └─────────┬───────────┘
                                 ▼
                       ┌─────────────────────┐
                       │ Telegram Delivery   │
                       └─────────────────────┘

PostgreSQL is included in the deployment stack and migrations are provided for the SQL persistence layer. The current default application repository remains the tested local file repository; do not assume PostgreSQL is being used merely because the container exists.
```

The media core is separated from Telegram so future HTTP clients, web interfaces, or other adapters can be added without rewriting the downloader.

## Repository Layout

```text
MORIS/
├── .github/
│   ├── workflows/ci.yml
│   ├── ISSUE_TEMPLATE/
│   └── PULL_REQUEST_TEMPLATE/
├── cmd/
│   ├── api/
│   ├── bot/
│   └── worker/
├── deployments/
│   └── prometheus/
├── docs/
│   ├── ARCHITECTURE.md
│   ├── FINAL.md
│   └── deployment/
├── internal/
│   ├── app/
│   ├── config/
│   ├── domain/
│   ├── httpapi/
│   ├── media/
│   ├── music/
│   ├── queue/
│   ├── redisx/
│   ├── repository/
│   ├── storage/
│   └── telegram/
├── migrations/
├── scripts/
├── data/
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── Makefile
└── README.md
```

## Requirements

### Local development

- Go 1.23+
- FFmpeg
- Python 3 + `yt-dlp`
- Git
- Redis only when using `MORIS_QUEUE_MODE=redis`

### Production server

Recommended:

- Ubuntu Server 24.04 LTS or another supported Linux distribution
- 2+ CPU cores
- 4 GB RAM minimum for a small installation
- SSD storage
- Docker Engine
- Docker Compose v2
- A Telegram bot token
- A strong random admin token

For heavy video processing, use more CPU/RAM and SSD capacity. Worker concurrency should be sized to the machine rather than blindly set to a large number.

## Quick Start

Clone the repository:

```bash
git clone https://github.com/moris-ir/Moris-downloader.git moris
cd moris
```

Create the environment file:

```bash
cp .env.example .env
nano .env
```

At minimum set:

```dotenv
MORIS_ENV=production
MORIS_TELEGRAM_TOKEN=YOUR_TELEGRAM_BOT_TOKEN
MORIS_ADMIN_TOKEN=GENERATE_A_LONG_RANDOM_SECRET
MORIS_QUEUE_MODE=redis
MORIS_REDIS_ADDR=redis:6379
MORIS_STORAGE_DIR=/data/moris
```

Build and start:

```bash
docker compose config
docker compose up --build -d
```

Check services:

```bash
docker compose ps
docker compose logs --tail=100 api
docker compose logs --tail=100 worker
docker compose logs --tail=100 bot
```

Health check from the server:

```bash
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/ready
```

Expected responses include `status: ok` and `status: ready`.

## Real VPS / Server Deployment

This is the recommended path for a real Linux server.

### 1. Connect to the server

```bash
ssh root@YOUR_SERVER_IP
```

Create a dedicated deployment user if you do not already have one:

```bash
adduser --disabled-password --gecos "" moris
usermod -aG sudo,docker moris
```

Log in as that user:

```bash
su - moris
```

### 2. Install Docker

Install Docker Engine and the Compose plugin using the official Docker installation instructions for your distribution.

Verify:

```bash
docker --version
docker compose version
```

### 3. Install Git

Ubuntu/Debian:

```bash
sudo apt update
sudo apt install -y git curl ca-certificates
```

### 4. Clone MORIS

```bash
cd ~
git clone https://github.com/moris-ir/Moris-downloader.git moris
cd ~/moris
```

### 5. Create production secrets

```bash
cp .env.example .env
chmod 600 .env
nano .env
```

Generate a strong admin token instead of using `change-me`:

```bash
openssl rand -hex 32
```

Put the generated value into `MORIS_ADMIN_TOKEN`.

Set:

```dotenv
MORIS_ENV=production
MORIS_TELEGRAM_TOKEN=...
MORIS_ADMIN_TOKEN=...
MORIS_QUEUE_MODE=redis
MORIS_REDIS_ADDR=redis:6379
MORIS_STORAGE_DIR=/data/moris
MORIS_WORKER_CONCURRENCY=2
MORIS_MAX_FILE_SIZE_MB=2048
MORIS_JOB_TIMEOUT=30m
MORIS_RATE_LIMIT_PER_MINUTE=20
MORIS_MAX_ATTEMPTS=3
MORIS_CLEANUP_AGE=24h
MORIS_TELEGRAM_MAX_UPLOAD_MB=49
```

Do not commit `.env`.

### 6. Validate Compose before starting

```bash
docker compose config
```

If this command reports an error, **do not start the stack**. Fix the configuration first.

### 7. Build the production images

```bash
docker compose build --pull
```

### 8. Start the core stack

```bash
docker compose up -d
```

### 9. Scale workers

For example, with 4 worker replicas:

```bash
docker compose up -d --scale worker=4
```

Do not use more workers than your CPU, RAM, bandwidth, and disk can sustain.

### 10. Verify the deployment

```bash
docker compose ps
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/ready
```

Then inspect logs:

```bash
docker compose logs -f --tail=200 bot worker api
```

### 11. Test the Telegram bot

Open the bot in Telegram and send:

```text
/start
```

Then send a public/authorized media URL. The bot should analyze it, enqueue the job, process it, and attempt Telegram delivery.

### 12. Keep the server alive

Docker containers use restart policies in the production Compose configuration. Verify after a reboot:

```bash
sudo reboot
```

Reconnect and run:

```bash
cd ~/moris
docker compose ps
curl http://127.0.0.1:8080/health
```

### 13. Firewall

Only expose ports you actually need. If a reverse proxy is used, normally only HTTP/HTTPS should be public.

Example with UFW:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

Do **not** expose Redis or PostgreSQL publicly.

### 14. Reverse proxy / HTTPS

For internet-facing API/admin endpoints, put Nginx, Caddy, Traefik, or another trusted reverse proxy in front of MORIS and terminate TLS there.

The application itself can remain bound to an internal/private interface. If you only need the Telegram bot, there is no requirement to expose the API publicly to Telegram because this project uses Telegram long polling.

### 15. Production checklist

Before calling the server production-ready:

- [ ] `.env` exists and is mode `600`
- [ ] Telegram token is valid
- [ ] `MORIS_ADMIN_TOKEN` is random and private
- [ ] Redis is internal-only
- [ ] PostgreSQL is internal-only
- [ ] Firewall is enabled
- [ ] Disk has sufficient free space
- [ ] Backups are configured
- [ ] `docker compose ps` is healthy
- [ ] `/health` works
- [ ] `/ready` works
- [ ] Telegram `/start` works
- [ ] A small authorized media URL succeeds
- [ ] Logs contain no secrets
- [ ] Worker count matches server resources

## Production Environment

`.env.example` is the source of truth for supported environment variables. Never commit a real `.env`.

Important settings:

| Variable | Purpose | Example |
|---|---|---|
| `MORIS_ENV` | Runtime environment | `production` |
| `MORIS_HTTP_ADDR` | API bind address | `:8080` |
| `MORIS_TELEGRAM_TOKEN` | Telegram bot token | secret |
| `MORIS_ADMIN_TOKEN` | Admin/API secret | secret |
| `MORIS_REDIS_ADDR` | Redis endpoint | `redis:6379` |
| `MORIS_QUEUE_MODE` | Queue backend | `redis` |
| `MORIS_STORAGE_DIR` | Persistent media state | `/data/moris` |
| `MORIS_WORKER_CONCURRENCY` | Jobs per worker process | `2` |
| `MORIS_MAX_FILE_SIZE_MB` | Download/output guard | `2048` |
| `MORIS_JOB_TIMEOUT` | Per-job timeout | `30m` |
| `MORIS_RATE_LIMIT_PER_MINUTE` | Per-user rate limit | `20` |
| `MORIS_MAX_ATTEMPTS` | Retry attempts | `3` |
| `MORIS_CLEANUP_AGE` | Old workspace cleanup | `24h` |
| `MORIS_RETRY_BASE` | Retry delay base | `5s` |
| `MORIS_TELEGRAM_MAX_UPLOAD_MB` | Telegram upload guard | `49` |
| `MORIS_AUDD_TOKEN` | Optional music recognition | secret |

## Docker Operations

Start:

```bash
docker compose up -d
```

Rebuild after source changes:

```bash
docker compose build --pull
docker compose up -d
```

Scale workers:

```bash
docker compose up -d --scale worker=8
```

Stop:

```bash
docker compose down
```

Stop without deleting named volumes:

```bash
docker compose down
```

**Never use `docker compose down -v` on production unless you intentionally want to delete persistent volumes.**

Logs:

```bash
docker compose logs -f --tail=200
```

Container shell:

```bash
docker compose exec api sh
docker compose exec worker sh
```

Resource usage:

```bash
docker stats
```

## Telegram Bot Setup

1. Open Telegram and talk to `@BotFather`.
2. Create a bot with `/newbot`.
3. Copy the token.
4. Put it in `MORIS_TELEGRAM_TOKEN`.
5. Start MORIS.
6. Send `/start` to the bot.

Because MORIS uses long polling, no Telegram webhook URL is required for the default deployment.

## Admin Panel

The admin UI is served by the API at:

```text
http://SERVER_IP:8080/admin
```

For a public production installation, put it behind HTTPS and a reverse proxy rather than exposing an administrative interface directly to the internet.

Admin API requests use:

```http
X-MORIS-ADMIN-TOKEN: YOUR_SECRET
```

Bot-to-API requests use:

```http
X-MORIS-BOT-TOKEN: YOUR_SECRET
```

Use long, random secrets and never paste them into GitHub issues or logs.

## Monitoring

MORIS exposes Prometheus metrics:

```text
GET /metrics
```

Prometheus and Grafana are included in `docker-compose.yml`.

Default local endpoints after Compose starts:

```text
Prometheus: http://SERVER_IP:9090
Grafana:    http://SERVER_IP:3000
```

For a public server, do not expose these ports directly without authentication/access control. Prefer an internal network plus a protected reverse proxy or VPN.

Useful operational checks:

```bash
docker compose ps
docker compose logs --tail=200 worker
curl -s http://127.0.0.1:8080/health
curl -s http://127.0.0.1:8080/ready
curl -s http://127.0.0.1:8080/metrics | head
```

## API

### Analyze media

```http
POST /api/v1/media/analyze
```

### Create job

```http
POST /api/v1/jobs
```

### Read job

```http
GET /api/v1/jobs/{id}
```

### Cancel job

```http
POST /api/v1/jobs/{id}/cancel?user_id=...
```

### Retry job

```http
POST /api/v1/jobs/{id}/retry
```

### User

```http
GET /api/v1/users/{id}
PUT /api/v1/users/{id}
```

### Music

```http
POST /api/v1/music/recognize
GET  /api/v1/music/lyrics?artist=...&title=...
```

### Admin

```http
GET /api/v1/admin/stats
GET /api/v1/admin/jobs
GET /api/v1/admin/users
```

### Operations

```http
GET /health
GET /ready
GET /metrics
GET /admin
```

## Storage

The default tested repository stores application state and job data under the configured storage directory. Docker maps this to a named persistent volume:

```text
moris_data:/data/moris
```

Back up this data regularly. If you change storage to another backend in a future adapter, keep the same repository/storage interfaces and document the migration procedure.

## Scaling Workers

MORIS is intentionally designed so API, bot, and workers are separate processes.

Small VPS:

```bash
docker compose up -d --scale worker=1
```

Medium VPS:

```bash
docker compose up -d --scale worker=4
```

Larger server:

```bash
docker compose up -d --scale worker=8
```

Benchmark before increasing concurrency. Video conversion is CPU- and I/O-heavy. Too many workers can make the server slower, exhaust memory, or fill the disk.

## Backups

Back up at minimum:

1. `.env` using a secure secret-management procedure.
2. Persistent MORIS data volume.
3. PostgreSQL volume if/when PostgreSQL is enabled as the active persistence adapter.
4. Any external object-storage configuration.

Example volume inspection:

```bash
docker volume ls | grep moris
```

Example archive of the local data volume from the project directory can be done through a temporary container; adapt the exact volume name to your Compose project:

```bash
docker run --rm \
  -v moris_moris_data:/data:ro \
  -v "$PWD":/backup \
  alpine:3.21 \
  tar czf /backup/moris-data-backup.tar.gz -C /data .
```

Test restores periodically. A backup that has never been restored is not a verified backup.

## Updates

Recommended update flow:

```bash
cd ~/moris
git pull --ff-only
cp .env .env.backup

docker compose config
docker compose build --pull
docker compose up -d

docker compose ps
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/ready
```

If the update is problematic, inspect:

```bash
docker compose logs --tail=300 api
docker compose logs --tail=300 worker
docker compose logs --tail=300 bot
```

For production, pin reviewed Git commits/tags instead of blindly deploying arbitrary branch changes.

## Troubleshooting

### Bot does not answer

```bash
docker compose logs --tail=200 bot
```

Check:

- Telegram token is correct.
- Only one bot deployment is polling the same bot token.
- Network access from the container is available.
- The bot container is running.

### Jobs remain queued

```bash
docker compose logs --tail=200 worker
```

Check Redis:

```bash
docker compose ps redis
docker compose logs --tail=100 redis
```

Also verify:

```dotenv
MORIS_QUEUE_MODE=redis
MORIS_REDIS_ADDR=redis:6379
```

### Download fails

Inspect worker logs and verify that `yt-dlp` exists inside the image:

```bash
docker compose exec worker yt-dlp --version
docker compose exec worker ffmpeg -version
```

A third-party platform can change its media delivery or authentication at any time. A failure does not necessarily mean MORIS itself is broken.

### Disk fills up

Check:

```bash
df -h
docker system df
```

MORIS performs cleanup of old job workspaces, but operational disk monitoring is still required.

### Permission problems

Check the storage mount:

```bash
docker compose exec worker sh -c 'ls -la /data/moris'
```

Do not solve permission issues by making the entire host filesystem world-writable.

## Security

- Never commit `.env` or real tokens.
- Use strong random admin secrets.
- Keep Redis and PostgreSQL private.
- Use HTTPS for internet-facing administration.
- Do not expose Prometheus/Grafana without access control.
- Validate URLs before invoking external programs.
- Never shell-interpolate user-controlled URLs.
- Enforce file-size and timeout limits.
- Keep Docker and host packages updated.
- Restrict SSH access and use key authentication where possible.
- Review provider terms and applicable law before enabling a source.

Read [SECURITY.md](SECURITY.md) before deploying publicly.

## Testing

Run the complete local validation suite:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/api ./cmd/worker ./cmd/bot
```

Smoke test:

```bash
./scripts/smoke.sh
```

Validate Compose:

```bash
docker compose config
```

The CI workflow performs the Go validation on every push/pull request.

## Development

Clone and enter the project:

```bash
git clone https://github.com/moris-ir/Moris-downloader.git moris
cd moris
```

Run tests:

```bash
go test ./...
```

Run API locally:

```bash
MORIS_QUEUE_MODE=memory \
MORIS_STORAGE_DIR=./data/moris \
go run ./cmd/api
```

Run worker separately when testing queue behavior:

```bash
MORIS_QUEUE_MODE=memory \
go run ./cmd/worker
```

Build all binaries:

```bash
make build
```

See `docs/ARCHITECTURE.md` for the internal design and extension points.

## GitHub Release Checklist

Before publishing a release:

- [ ] Update version/changelog.
- [ ] Run `go test ./...`.
- [ ] Run `go test -race ./...`.
- [ ] Run `go vet ./...`.
- [ ] Run all builds.
- [ ] Run `docker compose config`.
- [ ] Review `.env.example`.
- [ ] Confirm no secrets are tracked.
- [ ] Build the Docker image in CI or a machine with Docker.
- [ ] Test a real authorized Telegram workflow on staging.
- [ ] Tag the release.
- [ ] Publish release notes.

## Roadmap

The architecture leaves room for:

- PostgreSQL as the active repository backend
- S3 / Cloudflare R2 / MinIO storage adapters
- Redis Streams and stronger distributed job leasing
- Advanced VIP/subscription/payment providers
- More music recognition providers
- Rich lyrics metadata and caching
- Web dashboard expansion
- CDN/object-storage delivery
- Multi-node deployments
- API gateway and external developer API

These should be integrated behind existing interfaces rather than coupling them directly to Telegram handlers.

## License

See [LICENSE](LICENSE).

## Creator

**MORIS**  
Created & Designed by Moris  
Telegram: `https://telegram.me/moris3245`
