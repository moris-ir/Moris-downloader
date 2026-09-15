# Contributing

1. Create a focused branch.
2. Run `gofmt -w .` on changed Go files.
3. Run `go test ./...`.
4. Keep domain logic independent from Telegram and HTTP transports.
5. Do not commit secrets or generated media.
