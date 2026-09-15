# MORIS Architecture

MORIS is a modular Go application split into API, Telegram adapter, worker, queue, media engine, repository and storage boundaries.

## Runtime

`Telegram -> Bot -> authenticated API -> Repository + Queue -> Worker -> yt-dlp -> FFmpeg -> Telegram`

The API never performs the expensive media download itself. Workers consume jobs and use bounded contexts. Multiple workers can run against the same Redis queue.

## Current persistence

The default durable cross-process state is a locked JSON repository on the shared storage volume. Writes use a temporary file and atomic rename; reads/writes are protected with an OS file lock.

PostgreSQL migrations are included in `migrations/001_init.sql` as the schema contract for the next repository adapter. The current code deliberately avoids an external Go database dependency so the repository can be built and tested in an offline environment.

## Queue

Redis mode uses a small RESP client and list queue:

- `RPUSH` enqueue
- `BLPOP` worker consumption
- `LLEN` metrics
- `INCR`/`EXPIRE` rate limiting

Memory queue remains available for isolated unit tests and single-process development.

## Media engine

1. Validate URL.
2. Detect platform.
3. Ask yt-dlp for metadata/formats.
4. Select a bounded format based on job type and requested quality.
5. Download with yt-dlp.
6. Normalize/convert with FFmpeg when required.
7. Enforce maximum output size.
8. Deliver to Telegram when a chat ID is present.
9. Keep the workspace for the configured cleanup period.

No user URL is concatenated into a shell command; subprocesses receive argument arrays through `exec.CommandContext`.

## Reliability

- Worker job timeout
- Configurable retry count
- Backoff between retries
- Context cancellation
- File-size guard
- Telegram upload-size guard
- Cleanup loop for old workspaces
- `/health`, `/ready`, `/metrics`

## Security boundaries

Bot-to-API calls use `X-MORIS-BOT-TOKEN`. Admin endpoints use a separate header and require the configured admin token. Secrets are loaded from environment variables and are not part of the repository.
