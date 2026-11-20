# GEMINI.md — WESMK

Project: **WESMK** — ERP SMK (Go + templ/HTMX + PostgreSQL 16 + Redis 7).
Aturan lengkap: lihat **AGENTS.md**.

Ringkas:
- Stack terkunci: Go 1.26+, templ, HTMX, Alpine, Tailwind, pgx. Jangan reintroduce React.
- Snapshot sebelum edit massal; commit per task via pathspec.
- Dilarang perintah destruktif tanpa izin; jangan hapus .bak/.orig/.rej.
- Verifikasi mutlak: `go vet ./... && go test ./...`.
- Data sensitif (uang, PIN, OTP, HMAC, kunci jawaban) wajib server-side.
- Migrasi schema via `internal/db/migrations/NNN_*.sql`, idempotent.
- Gaya kode lazy/ponytail: YAGNI, fix 1 file, no helper 1 call site.
