## 2026-10-25 — database
Masalah: migrasi schema besar rawan gagal parsial.
Penyebab: urutan FK & trigger append-only.
Perbaikan: migration 003_schema.sql idempotent + DROP TRIGGER IF EXISTS.
Verifikasi: `go test ./...`, server boot, `\dt` 38 tabel.
