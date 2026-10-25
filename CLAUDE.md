# CLAUDE.md

Aturan kerja repository ini ada di **[AGENTS.md](AGENTS.md)** — WAJIB dibaca sebelum menyentuh file apa pun.

Ringkasan yang paling sering dilanggar:
1. Snapshot (`git add -A && git commit`) sebelum edit >5 file.
2. Commit per task via pathspec, bukan `git add -A`.
3. Tidak pernah `git reset --hard` / `git checkout --` / `git clean` / `rm -rf` tanpa izin.
4. Tidak hapus `*.bak`/`*.orig`/`*.rej`.
5. Kode lazy/ponytail: YAGNI, fix 1 file, no helper 1 call site, pakai yang ada.
6. Verifikasi: `go vet ./... && go test ./...` harus hijau.
7. Data sensitif (uang, PIN, OTP, HMAC, kunci) wajib server-side.
