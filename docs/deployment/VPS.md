# MORIS Production VPS Deployment

This document is the short operational runbook for deploying MORIS on a real Linux VPS.

## Supported shape

```text
Internet
   │
   ├── SSH
   │
   └── HTTPS (optional reverse proxy)
          │
          ▼
      MORIS API ───── Redis
          │              │
          └──── Workers ─┘
                 │
              yt-dlp
              FFmpeg
                 │
              /data/moris
```

Telegram uses long polling by default, so a public Telegram webhook is not required.

## Server sizing

Small installation:

- 2 vCPU
- 4 GB RAM
- 40+ GB SSD

For several simultaneous video conversions, use 4+ vCPU, 8+ GB RAM, and substantially more SSD space.

## Install

```bash
sudo apt update
sudo apt install -y git curl ca-certificates openssl
```

Install Docker Engine and Compose v2 using the official Docker instructions for your distribution.

Verify:

```bash
docker --version
docker compose version
```

## Deploy

```bash
git clone https://github.com/moris-ir/Moris-downloader.git ~/moris
cd ~/moris
cp .env.example .env
chmod 600 .env
openssl rand -hex 32
nano .env
```

Use production values, especially:

```dotenv
MORIS_ENV=production
MORIS_TELEGRAM_TOKEN=...
MORIS_ADMIN_TOKEN=...
MORIS_QUEUE_MODE=redis
MORIS_REDIS_ADDR=redis:6379
MORIS_STORAGE_DIR=/data/moris
```

Validate and launch:

```bash
docker compose config
docker compose build --pull
docker compose up -d
docker compose up -d --scale worker=2
```

## Verify

```bash
docker compose ps
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/ready
docker compose logs --tail=200 bot worker api
```

Then send `/start` to the Telegram bot.

## Firewall

Keep Redis, PostgreSQL, Prometheus and Grafana private.

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

If the API is not meant to be public, do not open port 8080 at the firewall.

## Reverse proxy

Put the admin/API behind a TLS reverse proxy if they must be reachable from the public internet. Restrict `/admin` and administrative API routes with an additional network access layer where possible.

## Update

```bash
cd ~/moris
git pull --ff-only
docker compose config
docker compose build --pull
docker compose up -d
docker compose ps
curl http://127.0.0.1:8080/health
```

## Rollback

For reliable production rollback, deploy immutable Git tags/releases and retain the previous image/tag. Avoid relying on `latest`.

## Logs

```bash
docker compose logs -f --tail=200 bot
docker compose logs -f --tail=200 worker
docker compose logs -f --tail=200 api
```

## Disk monitoring

```bash
df -h
docker system df
```

Never run `docker compose down -v` casually; named volumes contain persistent data.
