#!/usr/bin/env bash
set -euo pipefail

export PORT="${PORT:-3000}"

# Requires Postgres + Redis: `docker compose up -d`
exec go run ./cmd/wesmk
