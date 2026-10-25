# Audit WESMK — UI/UX & Kesiapan Rilis

**Tanggal:** 25 Oktober 2026  
**Scope:** frontend React/Vite + API Go + PostgreSQL/Redis, 8 peran, desktop 1440×900 & mobile 390×844.  
**Status jujur:** sudah layak sebagai **demo/pilot**, belum layak menyimpan data siswa/keuangan nyata.

---

## 0. TL;DR

**Yang sudah bagus:** build, test, lint hijau; auth dan session di server; scoping data siswa/ortu di source sudah benar; error boundary, banner gagal sync, dan layout mobile sudah ada. Ini bukan proyek yang perlu dibongkar dari nol.

**Yang bikin terlihat "jelek":**
- Hampir semuanya abu-abu: **923** class `text-slate-*`, **494** `bg-slate-*`, **511** `border-slate-*`; warna non-abu cuma merah dan amber. Tidak ada identitas sekolah.
- Teks terlalu kecil: **204** pemakaian `text-[10px]`/`text-[11px]`. Di halaman Admin TU versi HP, **49 dari 51** elemen teks berukuran ≤12px.
- Dashboard banyak kosong: Guru cuma 3 kartu dengan ±70% kanvas kosong; Kepala Sekolah punya sidebar 256–288px untuk 1 menu.
- Brand masih "WESMK" (nama produk), bukan nama sekolah. Logo masih kotak "SMK", favicon tidak ada.
- Kontrol demo (ganti peran, skenario demo, PIN/OTP) selalu terlihat. Kalau ini di-deploy dengan demo mode menyala, siapa pun bisa jadi Kepala Sekolah.

**Dua temuan paling serius:**
1. **API yang sedang jalan adalah build lama.** Proses API start 04:14:50, sedangkan `state_handlers.go` dan `scope.go` terakhir diubah 05:18–05:21. Akibatnya, sesi **student** bisa `GET /api/state` dan menerima **seluruh data**: 4 siswa, 3 ledger keuangan, OTP kepala sekolah, dan kunci jawaban ujian. Source terbaru sudah benar — server lama ini yang bocor.
2. **localStorage menyimpan snapshot penuh dan tidak dibersihkan saat logout/ganti peran.** Gw tanam data sentinel, login sebagai siswa, dan data staf tetap ada di localStorage meskipun API sudah scoped. Di komputer lab/sekolah yang dipakai bergantian, ini kebocoran privasi.

**Estimasi realistis:**

| Target | Waktu | Yang harus selesai |
|---|---:|---|
| Demo presentasi yang rapi | **4–6 hari kerja** | Design token, shell, login, 4 layar utama, mobile, a11y dasar |
| Pilot internal sekolah | **3–4 minggu** | P0 keamanan, data domain, backup, akun nyata, monitoring |
| Produksi data siswa/keuangan | **6–10 minggu** | Normalisasi data, audit trail, endpoint per domain, uji keamanan |

---

## 1. Skor Audit

| Area | Skor | Catatan |
|---|---:|---|
| Visual design | **4/10** | Bersih, tapi generik, monokrom, dan terasa seperti template admin |
| UX / information architecture | **5/10** | Alur 8 peran lengkap; hierarki dan kepadatan belum konsisten |
| Mobile | **6/10** | Shell responsif, tapi tabel terpotong dan target sentuh kecil |
| Accessibility | **3/10** | Teks mini, kontras rendah, label form tidak terhubung, modal tanpa dialog semantics |
| Kualitas kode frontend | **5/10** | Jalan, tapi monolit 1.400–2.400 baris dan tanpa routing |
| Keamanan source | **5/10** | Auth server bagus; masih banyak keputusan keuangan/OTP di browser |
| Keamanan runtime saat ini | **2/10** | API lama membocorkan seluruh state ke siswa |
| Kesiapan produksi | **4/10** | Dokumen readiness sudah jujur; blocker domain belum selesai |

---

## 2. Bukti Audit

### 2.1 Build & test

| Pemeriksaan | Hasil |
|---|---|
| `npm run build` | Lulus. JS **561,95 KB** (gzip 148 KB), CSS 39,28 KB. Ada warning chunk >500 KB. |
| `npm test` | Lulus. 8 test frontend + seluruh paket Go. |
| `npm run lint` | Lulus (`tsc --noEmit` + `go vet`). |
| `npm audit --omit=dev` | 0 vulnerability. |
| Console browser | 2 error nyata: `/favicon.ico` 404 di dev (file favicon memang tidak ada; di production SPA fallback malah mengembalikan HTML) dan `/api/auth/me` 401 saat belum login. |
| Git | Folder ini **bukan git repo**. Tidak ada riwayat/rollback. |

### 2.2 Screenshot

Semua ada di folder [`audit/`](audit/):

| Layar | File |
|---|---|
| Login desktop | [`audit/login-desktop.png`](audit/login-desktop.png) |
| Dashboard siswa desktop | [`audit/student-dashboard-desktop.png`](audit/student-dashboard-desktop.png) |
| Dashboard siswa mobile | [`audit/student-dashboard-mobile.png`](audit/student-dashboard-mobile.png) |
| Dashboard guru | [`audit/teacher-desktop.png`](audit/teacher-desktop.png) |
| Dashboard kepala sekolah | [`audit/headmaster-desktop.png`](audit/headmaster-desktop.png) |
| Loket SPP bendahara | [`audit/treasurer-billing-desktop.png`](audit/treasurer-billing-desktop.png) |
| Admin TU desktop | [`audit/admin-tu-desktop.png`](audit/admin-tu-desktop.png) |
| Admin TU mobile (tabel terpotong) | [`audit/admin-tu-mobile.png`](audit/admin-tu-mobile.png) |
| Ujian CBT mobile | [`audit/student-exam-mobile.png`](audit/student-exam-mobile.png) |
| Siswa setelah ganti dari staf | [`audit/student-after-staff-switch.png`](audit/student-after-staff-switch.png) |

### 2.3 Bukti kebocoran API lama

```
Proses API start        : 2026-10-25 04:14:50
state_handlers.go ubah  : 2026-10-25 05:18:49
scope.go ubah           : 2026-10-25 05:21:14

GET /api/auth/me        : role = student
GET /api/state (lama)   : 29 key, 4 students, 3 ledger, OTP 981240, correctKey "A"
GET /api/state (source) : 17 key, 1 student, 0 ledger, OTP null, correctKey null
```

Server source terbaru gw jalankan di port 3002 untuk pembanding; hasilnya sudah scoped dengan benar. Jadi perbaikannya bukan rewrite: **restart API** setelah perubahan Go.

### 2.4 Bukti kebocoran localStorage

Gw isi localStorage dengan penanda `LEAK-OTP-123456`, `LEAK-LEDGER`, `LEAK-COUNSELING`, dan `LEAK-EXAM`, lalu login sebagai siswa ke API yang sudah scoped:

```
API keys        : 17
API students    : 1
API OTP         : tidak ada
API ledger      : tidak ada

localStorage    : LEAK-OTP-123456      ✅ masih ada
                  LEAK-LEDGER          ✅ masih ada
                  LEAK-COUNSELING      ✅ masih ada
                  LEAK-EXAM            ✅ masih ada
```

Penyebabnya ada di `src/App.tsx`:
- `loadSaved()` (baris 114–123) memuat semua slice dari localStorage.
- `hydrateFromState()` (baris 177–208) hanya menimpa key yang dikirim server; key yang tidak dikirim (ledger, violations, counseling, OTP, dll.) tetap memakai nilai lama.
- Effect penyimpanan (baris 272–353) menulis lagi seluruh snapshot ke localStorage.
- `handleLogout()` (baris 426–431) **tidak** menghapus localStorage.

---

## 3. Temuan UI/UX

Prioritas: **P0 = sebelum ditunjukkan ke sekolah/stakeholder**, **P1 = minggu ini**, **P2 = polish**.

### P0 — yang paling menentukan "jelek"

| # | Temuan | Bukti | Dampak | Fix cepat |
|---|---|---|---|---|
| V1 | **Tidak ada website publik sekolah.** Yang ada hanya halaman login + portal ERP. Tidak ada beranda, profil, berita, prestasi, PPDB, galeri, atau kontak. | `src/App.tsx:813–844` menampilkan `LoginView` kalau belum ada sesi; tidak ada router. | Kalau target lo "website sekolah", ini bukan sekadar kurang cantik — halaman utamanya belum ada. | Tambah landing page publik + route `/portal` untuk login. |
| V2 | **Tipografi mini.** 204 pemakaian 10–11px; di Admin TU mobile 49/51 teks ≤12px. | `grep -ro "text-\[1[01]px\]" src` = 204; screenshot mobile. | Susah dibaca, terutama guru/orang tua di HP; terlihat seperti prototipe. | Label minimal 12px, body 14px, input 16px, heading 20–32px. |
| V3 | **Kontras label rendah.** 55 pemakaian `text-slate-400`; di atas putih kontrasnya ±2,56:1 (WCAG AA butuh 4,5:1). | `#94A3B8` di atas `#FFFFFF`. | Label status/tanggal susah dibaca, gagal a11y. | Ganti `text-slate-400` → `text-slate-500`/`600`; sisakan 400 untuk disabled. |
| V4 | **Palet 100% abu-abu.** 923 `text-slate`, 494 `bg-slate`, 511 `border-slate`; warna non-abu hanya merah (105) dan amber (21). Tidak ada hijau untuk "lunas/berhasil". | Hitungan grep seluruh `src/`. | Tidak ada brand, status sulit dibedakan, terasa dingin/generik. | Satu warna brand (navy/teal) + hijau/amber/merah untuk status. |
| V5 | **Brand salah fokus.** "WESMK" mendominasi; nama sekolah cuma badge kecil; logo masih kotak "SMK"; file favicon tidak ada. | `Header.tsx:80–95`, `LoginView.tsx:107–114`, `index.html`. | Pengunjung tidak langsung tahu ini sekolah mana. | Pakai logo + "SMK Negeri 1 Rekayasa" sebagai identitas utama; WESMK jadi nama produk kecil. |
| V6 | **Kontrol demo selalu tampil.** Ganti peran, "Skenario Demo", "Mode Demo", PIN bendahara, OTP kepsek terlihat di UI. Label peran di header malah bahasa Inggris: "Student", "Teacher", "Headmaster". | `Header.tsx:62–71`, `:132–273`; `TreasurerView.tsx:39`, `:101`; screenshot. | Menurunkan kepercayaan; kalau demo mode menyala di production, siapa pun bisa naik jadi kepsek. | Sembunyikan semua kontrol demo di production; label peran pakai Bahasa Indonesia; PIN/OTP jangan pernah ditampilkan. |

### P1 — bikin rapi dan konsisten

| # | Temuan | Bukti | Fix |
|---|---|---|---|
| V7 | **Dashboard kosong / tidak seimbang.** Guru: 3 kartu, sisanya kosong. Kepsek: sidebar 1 menu, konten utama hanya 4 KPI + 1 tabel. | Screenshot guru & kepsek. | Isi dengan konten hari ini: jadwal, pengumuman, tugas, aktivitas terbaru, quick actions. Sidebar 1 menu sebaiknya jadi top-nav. |
| V8 | **Header halaman tidak konsisten.** Semua view punya eyebrow "MODUL • ROLE (KODE)" + headline marketing, ukuran 1–3 baris, aksi di kanan/kiri berbeda. | `AdminTuView.tsx:182`, `TeacherView.tsx:247`, `HeadmasterView.tsx:104`, dll. | Satu pola: judul 24–28px, subtitle 1 baris, maksimal 2 aksi utama. Hapus copy "100% Objektif". |
| V9 | **Card overload.** 268 `rounded-xl/2xl`, 511 border; hampir semua konten dibungkus kartu, kadang kartu di dalam kartu. | Hitungan grep + screenshot. | Kurangi border; pakai spacing/shadow; satu kartu = satu ide. |
| V10 | **Tabel mobile terpotong.** Di Admin TU 390px, tombol "Periksa" terpotong; wrapper scroll 363px vs 309px. | Screenshot `admin-tu-mobile.png`; ukur `scrollWidth`. | Ubah tabel jadi card list di mobile, atau beri scroll affordance + kolom aksi sticky. |
| V11 | **Target sentuh kecil.** Tombol "Periksa" 80×24px; tombol header 38px; avatar 32×32px. Rekomendasi 44×44px. | Ukur bounding box di mobile. | Naikkan padding/min-height; icon button 40–44px. |
| V12 | **Feedback pakai `alert()`.** 15 pemakaian; tidak ada toast/inline error. | `grep alert(`. | Toast + error inline; `alert` hanya untuk konfirmasi destruktif. |
| V13 | **Animasi mati.** `animate-in`, `slide-in-from-*`, `zoom-in-*` dipakai 4×, tapi `tailwindcss-animate` tidak terpasang. | `package.json`, `Header.tsx:153`, `BottomNav.tsx:84`. | Pasang plugin atau hapus class; jangan biarkan menu "snap". |
| V14 | **Login paling bagus, tapi masih generik.** Tidak ada foto/identitas sekolah; kolom demo 9 tombol mendominasi. | `login-desktop.png`. | Foto sekolah/ilustrasi, logo asli, demo accounts dilipat di accordion "Coba demo". |

### P2 — polish

- Semua tombol primary memakai hitam; pertimbangkan brand color untuk aksi utama dan outline untuk sekunder.
- Status `PAID` masih abu-abu; `UNPAID` merah outline, `PARTIAL` amber outline. Pakai badge solid/soft dengan ikon.
- Tidak ada empty state yang menarik; beberapa tabel hanya kosong.
- Tidak ada skeleton loading; beberapa layar hanya teks "Memeriksa sesi aman…".
- Bahasa campur: "Student", "Teacher", "Headmaster", "Dashboard", "CBT", "PAID" vs label Indonesia.
- Avatar stok Unsplash (12 URL) dan font Google eksternal; untuk sekolah lebih baik foto asli/logo sendiri.

---

## 4. Temuan Teknis Frontend

### P0 — privasi & integritas

| # | Temuan | Bukti | Fix |
|---|---|---|---|
| T1 | **API lama bocor seluruh state ke siswa.** | Proses start 04:14 vs source 05:18; `GET /api/state` student = 29 key/4 siswa/ledger/OTP/answer key. | Restart API. Tambahkan watcher Go (`air`/`reflex`) supaya source dan server tidak pernah beda versi. |
| T2 | **localStorage menyimpan seluruh snapshot, tidak dibersihkan saat logout/ganti peran.** | Test sentinel di atas; `App.tsx:94`, `114–123`, `177–208`, `272–353`, `426–431`. | Hapus `localStorage` saat logout; reset semua slice ke fallback saat ganti peran; jangan simpan data sensitif di browser. Idealnya server state tidak di-persist ke localStorage sama sekali. |
| T3 | **Uang, PIN, OTP, HMAC, dan QR dibuat di browser.** | `App.tsx:490–512` (`handleRecordCashPayment`, `Math.random` HMAC), `:774–786` (hash surat), `TreasurerView.tsx:39` (PIN `123456` hardcoded), `:100` (validasi PIN di client), `HeadmasterView.tsx:57` (OTP `Math.random`). | Pindahkan ke endpoint server: kas + PIN terverifikasi, OTP ber-TTL, HMAC/QR ditandatangani server. |
| T4 | **Verifikasi publik memakai state browser, bukan tanda tangan server.** | `App.tsx:907–921` mengirim `officialLetters`/`ledger` lokal ke `PublicVerifyView`; docs readiness juga menyebut ini. | Endpoint `/verify/:hash` yang memverifikasi HMAC di server; QR berisi URL server. |
| T5 | **Demo mode = eskalasi peran.** Saat demo menyala, user bisa switch ke `headmaster` lewat UI. | `Header.tsx:132–205`, `/api/auth/demo-role`. | Production: `WESMK_DEMO_MODE=false`; jangan pernah deploy demo mode ke domain publik. |

### P1 — arsitektur & performa

| # | Temuan | Bukti | Fix |
|---|---|---|---|
| T6 | **Monolit.** `StudentView.tsx` 2.433 baris; `App.tsx` 1.448; `TeacherView.tsx` 1.140; 28 slice state di App. | `wc -l`. | Pecah per fitur; pindahkan state server ke query/context per domain; kurangi props drilling. |
| T7 | **Tidak ada routing URL.** Navigasi hanya `currentTab`; refresh kembali ke tab default; docs malah mencantumkan route `/student/dashboard`, `/finance/billing`, dll. | `App.tsx:110`, `roleNavigationMap`; `package.json` tidak punya router. | React Router/TanStack Router; URL per tab; deep link; tombol back browser. |
| T8 | **Bundle 562 KB, semua role view di-import eager.** | `App.tsx:69–77`; warning Vite >500 KB. | `React.lazy` per role, `Suspense`, manualChunks; target initial <250 KB. |
| T9 | **Type safety longgar.** `strict` tidak aktif; 9 `any`; build tidak menjalankan `tsc`. | `tsconfig.json`; `package.json` script build hanya `vite build`. | `strict: true`, `noUncheckedIndexedAccess`, `tsc --noEmit && vite build`, ESLint/Prettier. |
| T10 | **A11y dasar belum ada.** 62 `<label>` tanpa `htmlFor`; 153 tombol, hanya 1 focus ring; 10 modal tanpa `role="dialog"`/`aria-modal`/focus trap/Escape; hanya 1 `aria-label`. | Grep + inspeksi. | Label `htmlFor`/`id`; global `:focus-visible`; modal dialog + Escape + focus trap; aria-label untuk icon button. |
| T11 | **Tidak ada CI / git.** | Tidak ada `.github`, bukan git repo. | `git init`, commit baseline, CI: build + test + lint; preview deploy. |
| T12 | **Aset & SEO publik minim.** Tidak ada file favicon, `og:image`, canonical, robots.txt, sitemap; font Google render-blocking; avatar Unsplash. | `index.html`; `find public` kosong. | Tambah favicon/OG image/robots/sitemap; self-host font; foto sekolah sendiri. |

### P2 — kualitas hidup

- `vite.config.ts:11` masih pakai `__dirname`; Vite memberi warning dan akan jadi masalah di versi berikutnya.
- Tidak ada error tracking (Sentry) atau analytics.
- `alert()` 15×, tidak ada toast system.
- Belum ada test komponen/E2E; test HTTP masih manual.
- `npm run dev` tidak hot-reload Go; perubahan backend tidak kepakai sampai restart (ini akar T1).

---

## 5. Temuan Backend & Keamanan

### Yang sudah benar

- Password Argon2id; hash tidak dikirim ke browser.
- Sesi Redis, cookie `HttpOnly`, `SameSite=Lax`, TTL 8 jam, single-session.
- Rate limit login 5×/10 menit per IP+identifier.
- Origin check untuk request mutasi; CSP, HSTS, COOP, nosniff, frame-ancestors.
- Scoping baca siswa/ortu dan `MergeOwned` untuk tulis sudah benar **di source**.
- Kuis/CBT dan pembayaran VA punya endpoint server-authoritative; callback HMAC.
- `go test ./...`, `go vet`, `npm audit` bersih.

### Yang masih blocker (ringkas; detail ada di [`docs/production-readiness.md`](docs/production-readiness.md))

| Prioritas | Blocker | Bukti |
|---|---|---|
| Kritis | Presensi, status proctor, OTP, hash surat masih dihitung/ditulis client | `App.tsx`, `StudentView.tsx`, `HeadmasterView.tsx` |
| Kritis | Pembayaran tunai bendahara membuat ledger + HMAC di browser; PIN hardcoded | `App.tsx:490`, `TreasurerView.tsx:39` |
| Kritis | Semua data sekolah satu blob `app_state`; tidak ada audit trail/FK/optimistic lock | `internal/store/postgres.go`, `state_handlers.go` |
| Kritis | Staf mana pun bisa baca seluruh state dan tulis field di luar perannya | `scopeFor()` hanya scoped untuk student/parent |
| Kritis | Saat `app_state` kosong, snapshot client (mock) jadi isi database | `MergeOwned()` mengembalikan `incoming` |
| Kritis | Upload dokumen hanya metadata; validasi magic bytes di client | `StudentView.tsx` |
| Kritis | Magic link ortu belum ada kanal kirim; GET bisa dikonsumsi bot preview | `auth_handlers.go` |
| Kritis | Verifikasi publik pakai state browser | `PublicVerifyView.tsx`, `App.tsx` |
| Tinggi | Belum ada backup/restore, monitoring, alert | — |
| Tinggi | Belum ada reset password, lifecycle akun, MFA untuk keuangan | — |
| Tinggi | Gateway VA nyata + rekonsiliasi belum tersambung | — |

**Tambahan dari audit ini:**
- **Stale server** (T1) dan **localStorage leak** (T2) belum tercatat di dokumen readiness.
- **Demo role switch** adalah eskalasi sengaja; pastikan `WESMK_DEMO_MODE=false` di production.
- `activeHeadmasterOtp` ikut tersimpan di state dan localStorage; ini harus dihapus dari client state.
- Docker compose membuka Postgres/Redis ke `0.0.0.0`; aman untuk lokal, bahaya kalau dijalankan di server publik.

---

## 6. Gap Produk: "Website Sekolah" vs "Portal ERP"

Ini penting diputuskan sebelum desain. Saat ini produk adalah **portal ERP login**, bukan website sekolah publik.

| Kebutuhan website sekolah | Status |
|---|---|
| Beranda publik (sambutan, keunggulan, statistik) | ❌ belum ada |
| Profil sekolah, visi-misi, struktur, guru | ❌ belum ada |
| Berita, pengumuman, agenda, prestasi | ❌ belum ada |
| Galeri / ekstrakurikuler | ❌ belum ada |
| PPDB / informasi penerimaan | ❌ belum ada |
| Kontak, lokasi, media sosial | ❌ belum ada |
| Login portal siswa/guru/orang tua | ✅ ada |
| Dashboard 8 peran | ✅ ada (demo) |
| Verifikasi kwitansi/surat publik | ⚠️ ada, tapi belum server-authoritative |

**Rekomendasi:** kalau targetnya benar-benar "website sekolah", buat **landing page publik** 5 halaman dulu (Beranda, Profil, Berita/Prestasi, PPDB, Kontak) dengan CTA "Masuk Portal". Portal ERP tetap di `/portal`. Ini bisa memakai design system yang sama dan menambah 1–2 hari kerja, tapi langsung mengubah persepsi "ini website sekolah".

---

## 7. Rencana Tercepat (biar bisa selesai)

### Hari 0 — 2–3 jam, lakukan sekarang

1. **Restart API** (`npm run dev` atau `npm run dev:api`). Ini menutup kebocoran state.
2. **Bersihkan localStorage** saat logout dan saat ganti peran; reset semua slice ke fallback.
3. **Pastikan `WESMK_DEMO_MODE=false`** untuk deployment nyata; sembunyikan role switcher, PIN, OTP, dan akun demo.
4. **Tambah favicon** dan hapus error `/favicon.ico` 404.
5. **Global focus ring** + ganti `text-slate-400` → `text-slate-500`.
6. Commit baseline (`git init` + commit) sebelum refactor.

### Sprint 1 — 3 hari: bikin "tidak jelek"

| Hari | Pekerjaan | Output |
|---|---|---|
| 1 | Design token: warna brand, type scale, radius, shadow; shell header/sidebar; login | Login + shell terasa satu identitas |
| 2 | Dashboard siswa + guru: isi konten hari ini, jadwal, pengumuman, tugas, quick actions | 2 layar utama tidak kosong |
| 3 | Bendahara (loket + status warna) + kepsek (KPI + tabel); mobile 390/768; a11y dasar | 4 layar utama rapi di HP/desktop |

### Sprint 2 — 2 hari: konsistensi & rilis

- Semua role mewarisi token; hapus copy marketing, ganti label Inggris → Indonesia.
- Tabel mobile jadi card list; target sentuh ≥44px; modal `role="dialog"` + Escape.
- `React.lazy` per role + router; build ulang; cek bundle.
- Smoke test 8 role, login/logout, refresh, deep link, console bersih.

### Sebelum pilot — 2–4 minggu

- Pindahkan kas/PIN/OTP/HMAC/verifikasi publik ke server.
- Pindahkan presensi/proctor/upload ke endpoint domain.
- Normalisasi `app_state` atau minimal audit log + optimistic lock.
- Backup/restore, monitoring, akun nyata, matikan demo.
- Uji akses per peran staf (bukan cuma siswa/ortu).

---

## 8. Arah Desain Konkret

### Palet yang disarankan

Sekolah terasa formal, hangat, dan terpercaya. Jangan semua slate.

| Token | Hex | Pakai untuk |
|---|---|---|
| `brand-900` | `#0B2A4A` | Judul, sidebar, teks kuat |
| `brand-700` | `#14507A` | Tombol primary, active nav |
| `brand-500` | `#1D6FA5` | Link, chart, aksen |
| `accent` | `#F2A93B` | Highlight, badge prestasi |
| `bg` | `#F6F7F9` | Background aplikasi |
| `surface` | `#FFFFFF` | Kartu |
| `ink` | `#0F1F2E` | Teks utama |
| `muted` | `#5B6B7C` | Teks sekunder |
| `success` | `#15803D` | Lunas, hadir, berhasil |
| `warning` | `#B45309` | Partial, menunggu |
| `danger` | `#B91C1C` | Tunggakan, gagal, tolak |

### Type scale

| Peran | Ukuran | Contoh |
|---|---:|---|
| Eyebrow/label | 11–12px | "TAHUN AJARAN 2026/2027" |
| Body/table | 13–14px | Isi tabel, paragraf |
| Input | 16px | Agar tidak auto-zoom di iOS |
| Judul halaman | 24–28px | "Loket Pembayaran SPP" |
| Judul section | 16–18px | "Antrean Berkas" |
| Angka KPI | 28–36px | "Rp 0", "92,8" |

Aturan: tidak ada teks <12px; `text-slate-400` hanya untuk disabled; line-height 1.5.

### Layout & komponen

- Max width 1280–1440, sidebar 240, gutter 24/32, spacing 8pt.
- Satu halaman = satu judul + satu subtitle + maksimal 2 aksi utama.
- Dashboard: sapaan + tanggal, 3–4 KPI, "Jadwal Hari Ini", "Pengumuman", "Tugas/Approval", "Aktivitas Terbaru".
- Tabel: header sticky, status badge, hover, empty state; mobile jadi card.
- Login: logo sekolah + foto/ilustrasi, CTA jelas, demo accounts dilipat.
- Header: identitas sekolah di kiri; search/notifikasi/bantuan; user menu. Role switcher hanya di mode demo.
- Modal: `role="dialog"`, `aria-modal`, focus trap, Escape, klik overlay.
- Toast untuk sukses/gagal; skeleton untuk loading.

### Copywriting

- Ganti "WESMK" di header jadi "SMK Negeri 1 Rekayasa" (WESMK kecil di bawah).
- Ganti "Student/Teacher/Headmaster" → "Siswa/Guru/Kepala Sekolah".
- Hapus klaim berlebihan: "100% Objektif", "Anti-Curang", "Pisau Bermata Dua" — kecuali memang fitur nyata dan bisa dibuktikan.
- Gunakan bahasa yang menenangkan untuk orang tua: "Perkembangan anak", "Tagihan", "Izin", bukan "Gatekeeper" atau "Escrow".

---

## 9. Checklist "Selesai"

### Visual
- [ ] Tidak ada teks <12px.
- [ ] Tidak ada halaman yang 70% kosong.
- [ ] Semua status punya warna semantik (hijau/amber/merah).
- [ ] Logo + nama sekolah tampil di header/login/favicon.
- [ ] 1440, 768, 390 tidak ada konten terpotong.
- [ ] Tidak ada kontrol demo di production.

### Fungsional
- [ ] Login/logout 8 peran.
- [ ] Refresh & deep link tidak kembali ke tab default.
- [ ] Semua tabel bisa dibaca/diaksi di HP.
- [ ] Console bersih (favicon, 401 noise).
- [ ] Build <250 KB initial JS.

### Keamanan
- [ ] API yang jalan = source terbaru.
- [ ] `GET /api/state` student = hanya data sendiri (uji 4 siswa/ledger/OTP).
- [ ] localStorage bersih setelah logout/ganti peran.
- [ ] Tidak ada PIN/OTP/HMAC/uang yang dibuat di browser.
- [ ] `WESMK_DEMO_MODE=false`, akun demo dihapus.
- [ ] Backup + restore diuji.

### Operasional
- [ ] Git repo + CI build/test/lint.
- [ ] Health check + monitoring + alert.
- [ ] Error tracking.
- [ ] Dokumentasi deploy & rollback.

---

## 10. Keputusan yang Perlu Lo Ambil

1. **Target produk:** website sekolah publik, portal ERP, atau keduanya? Rekomendasi: keduanya, dengan landing page publik + `/portal`.
2. **Target rilis:** demo presentasi (4–6 hari) atau pilot dengan data nyata (3–4 minggu)?
3. **Mode demo:** apakah akan di-deploy publik? Kalau ya, demo mode harus tetap menyala dan datanya harus 100% fiktif; kalau tidak, matikan sekarang.
4. **Arah visual:** formal navy (aman, cocok sekolah negeri) atau lebih hangat/friendly (cocok SMK dan orang tua)? Rekomendasi: navy + amber.

Kalau lo setuju, langkah paling cepat adalah gw eksekusi **Hari 0 + Sprint 1** dulu: restart API, bersihkan localStorage, bikin design token, rapikan login/shell, lalu 4 layar utama. Setelah itu baru sentuh backend production.
