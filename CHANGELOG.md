# Changelog

## 2.1.0 - 2026-09-15

### Added
- Redis RESP client with Redis-backed job queue (`RPUSH`/`BLPOP`).
- File-backed shared repository with atomic writes and OS file locking for multi-process/local persistence.
- Telegram long-polling update loop with commands, inline keyboard callbacks and URL extraction.
- Bot-to-API authenticated job creation using `X-MORIS-BOT-TOKEN`.
- Worker-side Telegram upload for video/audio/document results.
- Persistent job output metadata and Telegram chat ID.
- Race-enabled test coverage for repository, queue, HTTP auth and Redis RESP parsing.

### Changed
- API no longer starts worker goroutines; dedicated worker processes consume the queue.
- Docker deployments can use Redis as the shared queue while API/worker share persistent state volume.

### Known limitations
- PostgreSQL repository adapter is still pending; current shared persistence adapter is filesystem-based.
- FFmpeg conversion/format normalization, VIP/billing, music recognition/lyrics providers and admin web UI remain next milestones.

## 2.2.0 - Final runnable core
- Added FFmpeg conversion/normalization and dynamic quality policy.
- Added retry, cleanup, output-size and Telegram-upload guards.
- Added Redis rate limiting and richer admin endpoints.
- Added Telegram metadata/quality selection UI.
- Expanded platform detection and audio formats.
- Updated documentation to clearly separate implemented core from future adapters.
