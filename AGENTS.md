# Aturan Kerja Repository — WESMK (WAJIB DIBACA AI)

Project: **WESMK** — ERP SMK (Go + templ/HTMX + PostgreSQL 16 + Redis 7).
Stack terkunci: Go 1.23+, Go templ, HTMX 2.x, Alpine.js, Tailwind, pgx/v5, sqlc-style manual. **React/Vite sudah dihapus; jangan reintroduce.**

## 1. Snapshot sebelum edit massal
Sebelum mengubah >5 file atau menjalankan script apa pun:
```bash
git add -A && git commit -m "snapshot: sebelum <apa yang akan diubah>"
```
Belum ada git repo? Inisialisasi dulu: `git init && git add -A && git commit -m "baseline"`.

## 2. Commit di akhir setiap task
Commit hanya file yang disentuh via pathspec:
```bash
git add <file1> <file2> && git commit -m "<deskripsi>" -- <file1> <file2>
```
Dilarang `git add -A` / `git commit` polos di repo multi-sesi.

## 3. Dilarang perintah destruktif tanpa izin
Tidak pernah: `git reset --hard`, `git checkout --`, `git restore`, `git clean`, `rm -rf` pada folder project, branch switch yang membuang perubahan.

## 4. Dilarang menghapus .bak/.orig/.rej
File cadangan = bukti konten hilang.

## 5. Batch rewrite hanya dry-run + verifikasi
- Dry-run dulu, tampilkan hitungan file.
- Setelah tulis, verifikasi: `go vet ./...` + `go test ./...` harus hijau.
- Jika jumlah baris Go turun drastis (>10%), red flag — kembalikan.

## 6. Gaya kode: LAZY / ponytail (WAJIB)
- YAGNI: jangan buat file/class/helper/config/flag tanpa permintaan.
- Perbaiki 1 file, jangan refactor 200.
- No helper untuk 1 call site.
- Pakai yang sudah ada dulu (internal/*, Go stdlib).
- Tanpa dependency baru untuk hal yang bisa 5 baris.
- No batch script untuk masalah sekali pakai.
- Kalau ragu: jangan tulis, tanya user.

## 7. Aturan data & keamanan (WESMK)
- Uang, PIN, OTP, HMAC, kunci jawaban, rubrik **wajib server-side**.
- `financial_audit_ledgers` append-only (trigger sudah ada) — jangan UPDATE/DELETE.
- Jangan simpan data sensitif di browser/state client.
- `WESMK_DEMO_MODE=false` untuk deploy nyata; sembunyikan role switcher, PIN, OTP, akun demo.
- Jangan commit secret/API key.

## 8. Database & migrasi
- Semua perubahan schema via `internal/db/migrations/NNN_*.sql`.
- Migrasi harus idempotent (`CREATE TABLE IF NOT EXISTS`).
- Jangan edit migrasi yang sudah applied; buat migrasi baru.
- Jalankan `go test ./...` setelah migrasi.

## 9. Verifikasi mutlak
Sebelum selesai task:
```bash
go vet ./... && go test ./...
```
Server harus boot: `PORT=3001 go run ./cmd/wesmk`.

## 10. Struktur penting
```
cmd/wesmk/        entrypoint
internal/auth/    Argon2id, sesi, magic link, rate limit
internal/httpapi/ router, middleware, handler, UI templ
internal/store/   Postgres + Redis
internal/db/      pool + migrasi SQL embedded
internal/state/   scoping baca/merge tulis per peran
internal/ui/      templ + HTMX + assets
```

## 11. Catat pelajaran
Setelah fix bug, tambahkan 1 entri singkat di `docs/pelajaran/<area>.md` (jika ada) dalam commit yang sama. Maksimal 5 baris per entri, hanya bug nyata.
