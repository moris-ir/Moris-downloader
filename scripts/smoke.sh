#!/usr/bin/env sh
set -eu
curl -fsS http://localhost:8080/health >/dev/null
curl -fsS http://localhost:8080/ready >/dev/null
curl -fsS http://localhost:8080/metrics >/dev/null
echo 'MORIS smoke test: OK'
