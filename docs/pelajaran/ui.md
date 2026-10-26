## 2026-10-25 — ui
Masalah: repo punya React prototype + Go templ HTMX berdampingan.
Penyebab: frontend lama tidak dipakai server.
Perbaikan: hapus src/Vite/tsconfig/package.json, server pakai templ/HTMX.
Verifikasi: `go build ./...`, `go test ./...`, login 200.

## 2026-10-26 — cbt-mobile-anticheat
Masalah: UI ujian CBT menduplikasi sidebar di bawah kartu soal pada HP & rawan dicurangi via beralih app / split-screen.
Penyebab: Desain awal desktop-only tanpa Bottom Sheet modal serta deteksi fokus hanya mengandalkan event window blur standar.
Perbaikan: Sembunyikan sidebar di layar <1024px, buat Bottom Sheet navigasi soal, dan pasang sensor visibility/pagehide/split-screen multi-tier.
Verifikasi: `templ generate`, `go vet ./...`, `go test ./...`, HTTP 200 OK.
