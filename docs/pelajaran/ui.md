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

## 2026-11-28 — kartu-logo-pakai-lambang-png
Masalah: Kartu logo landing memakai emblem SVG kecil + tiga baris wordmark, padahal situs acuan hanya menampilkan lambang tanpa teks; PNG logo baru juga 404 walau filenya ada di `assets/images`.
Penyebab: Aset gambar di-embed lewat pola `images/*.jpg` saja, dan `handleUIAsset` hanya mengizinkan ekstensi `.jpg` di bawah `images/`.
Perbaikan: Tambah `images/*.png` ke `//go:embed`, izinkan `.png` dengan `Content-Type: image/png`, lalu ganti isi `.sf-brandcard` dengan satu `<img>` lambang (600×542, `object-fit: contain`) tanpa wordmark.
Verifikasi: `templ generate`, `go vet ./...`, `go test ./...`; `/ui-assets/images/logo-crest.png` 200 `image/png`, kartu 270×240 dengan gambar 234×208, mobile lambang 46px.

## 2026-11-28 — slide-hero-keluar-border
Masalah: Saat slider hero berganti gambar, gambar lama masih terlihat menonjol di area sudut kiri-bawah (seperti keluar dari border melengkung).
Penyebab: `border-radius: 0 0 0 300px` dipasang di tiap `.sf-slide`, bukan di container; saat track bergeser, slide yang sedang tampil tidak lagi sejajar dengan sudut sehingga potongannya ikut bergeser.
Perbaikan: Radius dipindah ke `.sf-hero` (yang sudah `overflow: hidden`), radius di `.sf-slide` dan `.sf-hero-overlay` dihapus, dan breakpoint 900px memakai radius 150px di `.sf-hero` langsung.
Verifikasi: `templ generate`, `go vet ./...`; animasi dibekukan di tengah transisi (delay -3.75s, paused) → track `matrix(... -1360, 0)` dan sudut hero tetap terpotong rapi di desktop & mobile.

## 2026-11-28 — kartu-logo-teks-terpotong
Masalah: Setelah kartu logo hanya memakai lambang, teks "SMK / ENGINEERING FOR A BRIGHTER FUTURE" di bawah lambang hilang dari halaman.
Penyebab: PNG `logo-crest.png` dipotong hanya pada area lambang (y 97–839), bukan seluruh konten file asal (y 97–1090).
Perbaikan: PNG dibuat ulang dari file asal dengan bbox seluruh konten + margin 8px (720×799), templ memakai `width="720" height="799"`, dan `object-fit: contain` mengecilkannya agar lambang + teks muat dalam kartu (tinggi mobile 52px).
Verifikasi: `templ generate`, `go vet ./...`, `go test ./...`; `/ui-assets/images/logo-crest.png` 200 `image/png` 458087 byte; kartu desktop 270×240 berisi gambar 234×208, mobile gambar 46.8×52.

## 2026-11-28 — header-scroll-diblokir-csp
Masalah: Header landing tidak menempel atau berbayang saat digulir, tanpa pesan galat apa pun di antarmuka.
Penyebab: Skrip inline di `home.templ` diblokir CSP `script-src 'self' 'unsafe-eval'`, sehingga listener scroll tidak pernah terpasang.
Perbaikan: Logika dipindah ke `app.js` (toggle `is-scrolled` pada `[data-header]`); templ hanya menyisakan markup.
Verifikasi: Playwright 1440/1024/390 → `#header-area` tetap di top 0, tinggi konstan 101px, kelas `is-scrolled` muncul setelah scroll.

## 2026-11-28 — selektor-footer-menimpa-logo
Masalah: Merek footer menumpuk vertikal dan pil sosial kehilangan padding setelah tautan footer dirapikan.
Penyebab: `.sf-footer a { display:block; padding:4px 0 }` menimpa `.sf-logo` dan `.sf-social a` karena selektornya terlalu luas.
Perbaikan: Aturan dipersempit menjadi warna saja; tata letak dipindah ke `.sf-footer .sf-logo` dan `.sf-footer-cols a`.
Verifikasi: `display:inline-flex` pada logo, mark 46px dan teks sejajar mendatar; pil sosial kembali `padding: 0 10px`.

## 2026-11-28 — kartu-program-hanya-hover
Masalah: Detail kartu program tidak pernah muncul di ponsel karena interaksi hanya tersedia pada `:hover`.
Penyebab: Gaya reveal hanya didefinisikan untuk `:hover`, sementara layar sentuh tidak memiliki keadaan hover yang stabil.
Perbaikan: `@media (hover: none)` menampilkan detail secara permanen dan menumpuk kartu tanpa memotong teks.
Verifikasi: Playwright 390px `hasTouch` → teks kartu terbaca penuh dan tidak ada luapan horizontal.

## 2026-11-28 — kartu-program-terpotong-saat-hover
Masalah: Tombol "Lihat layanan" terpotong di tepi bawah kartu program saat overlay hover muncul.
Penyebab: Rasio tetap `650/488` + `overflow: hidden` mengunci tinggi kartu 175–205px, sedangkan isi overlay butuh hingga ~320px pada 4 kolom.
Perbaikan: `@media (min-width: 1081px)` memakai grid + `aspect-ratio: auto` agar isi overlay menentukan tinggi kartu; blok `hover: none` tetap menang di layar sentuh.
Verifikasi: Playwright 1440/1280/1200/1120/1081 → sela tombol 22–36px tanpa luapan; 1024/390 tetap memakai tata letak lama.
