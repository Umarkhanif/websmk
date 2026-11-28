#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
export PATH="$PATH:$(go env GOPATH)/bin"

# Port & kredensial dari .env (PORT=3001 di demo). Butuh Postgres + Redis:
# `docker compose up -d`
if [ -f .env ]; then set -a; . ./.env; set +a; fi
export PORT="${PORT:-3000}"

# Matikan instance lama yang masih memegang port ini.
pid=$(ss -ltnp 2>/dev/null | grep ":${PORT} " | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2 || true)
if [ -n "${pid}" ]; then
  echo "stop instance lama (pid ${pid}) di port ${PORT}"
  kill "${pid}" 2>/dev/null || true
  sleep 1
fi

# .templ -> _templ.go. Wajib, karena aset & template ikut di-embed ke binary.
if command -v templ >/dev/null; then templ generate ./... >/dev/null; fi

echo "WESMK jalan di http://localhost:${PORT}"
exec go run ./cmd/wesmk
