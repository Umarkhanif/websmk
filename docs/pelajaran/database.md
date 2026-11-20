## 2026-10-25 — database
Masalah: migrasi schema besar rawan gagal parsial.
Penyebab: urutan FK & trigger append-only.
Perbaikan: migration 003_schema.sql idempotent + DROP TRIGGER IF EXISTS.
Verifikasi: `go test ./...`, server boot, `\dt` 38 tabel.

## 2026-11-20 — state audit & domain shadow
Masalah: mutasi `app_state` tidak meninggalkan jejak, tabel domain cepat basi.
Penyebab: write path hanya mengubah blob JSON.
Perbaikan: `UpdateStateAudited` + trigger append-only `state_audit_log`, backfill idempotent, shadow sync pasca-mutasi.
Verifikasi: `go vet ./...` + `go test ./...` hijau; fresh clone `go build ./...` lulus.
