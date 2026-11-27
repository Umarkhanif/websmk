#!/usr/bin/env bash
# Preview sekolah fiksi untuk sales/demo customer.
#
# Menjalankan WESMK instance TERPISAH:
#   - database `wesmk_preview` (migrasi otomatis saat app boot)
#   - Redis db 1 (namespace cache terpisah dari instance utama)
#   - port ${PREVIEW_PORT:-3002}
#   - nama sekolah ${WESMK_SCHOOL_NAME:-SMK Cendekia Nusantara} (fiksi)
#   - data master fiktif dari scripts/preview/preview_seed.sql
#
# Produksi tidak tersentuh. Infrastruktur Postgres+Redis diambil dari
# docker-compose.yml repo ini (`docker compose up -d`).
#
# Usage: ./scripts/preview.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${PREVIEW_PORT:-3002}"
SCHOOL="${WESMK_SCHOOL_NAME:-SMK Cendekia Nusantara}"
PREVIEW_PASSWORD="${WESMK_PREVIEW_PASSWORD:-GantiSaya#2026}"
DB="wesmk_preview"
PG_CONTAINER="${WESMK_PG_CONTAINER:-wesmk-postgres}"
BIN="${TMPDIR:-/tmp/opencode}/wesmk-preview"

mkdir -p "$(dirname "$BIN")"

pg() { docker exec -i "$PG_CONTAINER" psql -U wesmk "$@"; }

echo "==> Infrastruktur Postgres + Redis"
(cd "$ROOT" && docker compose up -d --wait postgres redis)

echo "==> Database preview: $DB"
if ! pg -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '$DB'" | grep -q 1; then
    pg -d postgres -c "CREATE DATABASE $DB"
fi

echo "==> Build WESMK"
(cd "$ROOT" && go build -o "$BIN" ./cmd/wesmk)

WESMK_USERS=$(cat <<EOF
[
  {"identifier":"admin.tu","password":"$PREVIEW_PASSWORD","fullName":"Admin TU","role":"admin_tu"},
  {"identifier":"kepala.sekolah","password":"$PREVIEW_PASSWORD","fullName":"Drs. H. Rudi Santoso","role":"headmaster"},
  {"identifier":"guru.demo","password":"$PREVIEW_PASSWORD","fullName":"Siti Rahmawati, S.Pd.","role":"teacher"}
]
EOF
)

echo "==> Jalankan instance preview di http://localhost:$PORT"
PORT="$PORT" \
WESMK_ENV=development \
DATABASE_URL="postgres://wesmk:wesmk@localhost:5432/$DB?sslmode=disable" \
REDIS_URL="redis://localhost:6379/1" \
WESMK_SCHOOL_NAME="$SCHOOL" \
WESMK_USERS="$WESMK_USERS" \
"$BIN" &
APP_PID=$!
trap 'kill "$APP_PID" 2>/dev/null || true' EXIT

echo "==> Tunggu server siap"
for _ in $(seq 1 60); do
    if curl -sf "http://localhost:$PORT/" >/dev/null 2>&1; then break; fi
    sleep 1
done

echo "==> Seed data sekolah fiksi"
pg -d "$DB" -v ON_ERROR_STOP=1 < "$ROOT/scripts/preview/preview_seed.sql"

cat <<EOF

==================================================================
  PREVIEW SIAP — sekolah fiksi, data fiktif
  Landing : http://localhost:$PORT/
  Sekolah : $SCHOOL

  Akun preview (fiktif, jangan dipakai produksi):
    admin.tu        / $PREVIEW_PASSWORD   (admin TU)
    kepala.sekolah  / $PREVIEW_PASSWORD   (kepala sekolah)
    guru.demo       / $PREVIEW_PASSWORD   (guru)

  Landing publik tidak butuh login. Ctrl+C untuk berhenti.
  Override password: WESMK_PREVIEW_PASSWORD=... ./scripts/preview.sh
==================================================================
EOF

wait "$APP_PID"
