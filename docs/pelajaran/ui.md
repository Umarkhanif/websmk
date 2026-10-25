## 2026-10-25 — ui
Masalah: repo punya React prototype + Go templ HTMX berdampingan.
Penyebab: frontend lama tidak dipakai server.
Perbaikan: hapus src/Vite/tsconfig/package.json, server pakai templ/HTMX.
Verifikasi: `go build ./...`, `go test ./...`, login 200.
