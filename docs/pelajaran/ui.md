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

## 2026-10-26 — exam-lobby-schedule
Masalah: Ruang persiapan ujian kosong melompong di HP & hanya punya 1 tombol statis tanpa jadwal multi-sesi.
Penyebab: Ketiadaan struktur jadwal multi-sesi harian & pembeda visual status ujian per mapel.
Perbaikan: Buat kartu jadwal mapel responsive dengan 4 status visual (Bisa Dikerjakan, Selesai, Menunggu Sesi, Belum Dibuka).
Verifikasi: `templ generate`, `go test ./...`, HTTP 200 OK.

## 2026-10-26 — mystery-orb-reveal
Masalah: Siswa membutuhkan visualisasi gamifikasi nilai rata-rata ujian bertahap yang interaktif.
Penyebab: Nilai ujian sebelumnya langsung muncul tanpa suspense atau umpan balik emosional.
Perbaikan: Buat mystery orb lingkaran 2-tahap klik dengan 4 tingkatan aura cahaya (Stellar Jade, Emas, Biru, Hijau).
Verifikasi: `templ generate`, `go test ./...`, HTTP 200 OK.

## 2026-10-26 — mystery-orb-lock-breakdown
Masalah: Lingkaran rata-rata bisa dibuka saat belum semua mapel selesai & tester mengotori tampilan.
Penyebab: `AllExamsGraded` hanya mengecek mapel 1 & JS tidak mengecek `data-unlocked`.
Perbaikan: Hapus tester box, kunci ketat orb jika belum selesai dinilai, dan buat rincian nilai per mapel dengan opsi sembunyikan nilai setelah orb terbuka.
Verifikasi: `templ generate`, `go test ./...`, HTTP 200 OK.

## 2026-10-27 — mystery-orb-2stage-spin
Masalah: Orb masih bisa dibuka karena cache browser aset lama & ikon gembok tidak diinginkan user.
Penyebab: Header cache 1 jam tanpa query versioning pada app.js/css serta icon gembok menggantikan tanda tanya.
Perbaikan: Kembalikan simbol permanen ?, pasang cache-busting v=20261027b, nolkan nilai rata-rata jika belum selesai dinilai, dan terapkan alur klik 1 berputar sedikit warna & klik 2 baru berwarna penuh.
Verifikasi: `templ generate`, `go test ./...`, HTTP 200 OK.

## 2026-10-28 — cbt-multisubject-rapor-anticheat
Masalah: Ruang ujian siswa terkunci setelah 1 mapel dikumpulkan meski masih ada mapel belum selesai, dan kartu nilai rata-rata berada di ruang persiapan.
Penyebab: Flag `examSubmitted` bersifat global per siswa tanpa melacak progres per ID mapel (`exam-sch-01/02/04`), serta strike anti-curang belum tersimpan per peringatan di server.
Perbaikan: Pindahkan kartu rata-rata ujian ke menu Rapor, buka otomatis mapel berikutnya yang belum selesai, sederhanakan teks UI, dan simpan setiap strike anti-curang ke server + blur soal sebelum layar penuh.
Verifikasi: `templ generate`, `go vet ./...`, `go test ./...`, HTTP 200 OK.

## 2026-10-28 — ui-design-simplification-all-portals
Masalah: Teks pada portal bendahara, kesiswaan, piket, admin TU, dan kepala sekolah terlalu padat dan bertele-tele.
Penyebab: Frasa deskripsi teknis panjang di judul halaman, tombol, dan kartu stat.
Perbaikan: Sederhanakan teks dan rapikan tata letak responsif (.grid-2, .grid-4, .dashboard-grid) di seluruh berkas portal.
Verifikasi: `templ generate`, `go vet ./...`, `go test ./...`, HTTP 200 OK di semua portal.

## 2026-10-29 — form-ux-declutter-info-bubble
Masalah: Desain form kaku seperti boilerplate generik dan teks petunjuk statis membuat tampilan berantakan.
Penyebab: Input flat dengan focus ring OS, dropdown mentah, serta paragraf penjelasan permanen di bawah label.
Perbaikan: Buat komponen InfoBubble (`!`) popover interaktif, rapikan `.field-head` & label, perhalus fokus kontrol & tombol berkedalaman di `app.css`.
Verifikasi: `templ generate`, `go vet ./...`, `go test ./...`, HTTP 200 OK.

## 2026-10-30 — info-bubble-viewport-clipping
Masalah: Popover petunjuk (!) terpotong di tepi kanan layar atau kartu.
Penyebab: .info-bubble-content memakai `left: 0`, membuat popover melebar ke kanan keluar dari viewport.
Perbaikan: Alihkan ke `right: 0; left: auto;`, beri panah penunjuk atas, dan auto-clamp di `app.js`.
Verifikasi: `templ generate`, `go vet ./...`, `go test ./...`, HTTP 200 OK.

## 2026-10-30 — proctor-exam-reset-and-bug-tracker
Masalah: Pengawas CBT belum memiliki kontrol reset ujian siswa/lab dan belum ada portal pelaporan bug untuk pengembang.
Penyebab: Ketiadaan endpoint reset & audit trail reset CBT di state serta tidak tersedianya peran developer internal.
Perbaikan: Tambahkan reset ujian individu & bulk di live proctoring beserta audit log, buat form `/contact` terintegrasi, dan hadirkan portal peran `developer` (`developer.admin`).
Verifikasi: `templ generate`, `go vet ./...`, `go test ./...`, HTTP 200 OK di `/contact`, `/teacher/exam-control`, dan `/developer/dashboard`.

## 2026-11-27 — templ-raw-dalam-script-tag
Masalah: `@templ.Raw(...)` di dalam `<script>` dirender literal, bukan JSON.
Penyebab: templ memperlakukan isi `<script>` sebagai teks mentah, tidak mengevaluasi sintaks `@`.
Perbaikan: Bungkus seluruh tag: `@templ.Raw("<script ...>" + json + "</script>")`.
Verifikasi: `templ generate`, `go vet ./...`, `go test ./...`, output render diperiksa.

## 2026-11-28 — landing-terlewati-saat-demo-noauth
Masalah: Dengan `WESMK_DEMO_NOAUTH=true`, `/` dan `/login` langsung 302 ke dashboard peran sehingga landing page tidak pernah tampil.
Penyebab: `attachAuth` menempelkan user demo ke semua path, termasuk halaman publik.
Perbaikan: Halaman tamu (`/`, `/contact`, `/verify*`) tidak lagi memakai user demo, sesi nyata selalu diutamakan, dan tautan portal di landing tetap membuka portal demo.
Verifikasi: `go vet ./...`, `go test ./...`; mode demo `/` 200 landing, `/login?tab=staff` 302 ke `/teacher/dashboard` 200.

## 2026-11-28 — wordmark-landing-hardcoded
Masalah: Kartu brand landing menampilkan "SMK Cendekia Nusantara" (sisa sekolah fiksi) padahal judul dan isi halaman memakai nama sekolah asli.
Penyebab: Wordmark 3 baris ditulis hardcoded di `home.templ`, tidak mengikuti `WESMK_SCHOOL_NAME`.
Perbaikan: `ui.SchoolLines()` memecah `SchoolName()` jadi 3 baris; templ merender hasilnya.
Verifikasi: `templ generate`, `go vet ./...`, `go test ./...`; render `/` menunjukkan "SMK / Negeri 1 / Rekayasa".

## 2026-11-28 — logout-kembali-ke-login
Masalah: Setelah "Keluar dari portal", pengunjung mendarat di `/login` (dan di mode demo `/login` langsung memantul ke portal lagi), jadi landing page tidak pernah terlihat.
Penyebab: `handleLogoutForm` mengarahkan ke `/login`, bukan halaman publik.
Perbaikan: Logout sekarang redirect ke `/` (landing page).
Verifikasi: `go vet ./...`, `go test ./internal/httpapi/...`; POST `/logout` → 303 `/`, GET `/` → 200 landing.
