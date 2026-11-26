# PETA PROYEK WESMK — Mulai dari Sini

**Terakhir diperbarui:** 25 November 2026
**Untuk siapa:** lo, 3 bulan lagi, setelah lupa semuanya.

Dokumen ini satu-satunya tempat untuk menjawab "project ini sebenarnya apa, isinya apa,
sudah sampai mana, dan besok ngapain". Audit dan rencana lama sudah dipindahkan ke `docs/arsip/` sebagai **arsip sejarah** —
jangan dibaca dulu, lihat bagian 14.

---

## 0. Lima kalimat kalau lo cuma punya 30 detik

1. WESMK adalah **ERP sekolah** (sistem operasional), bukan website profil sekolah.
   Halaman publiknya cuma login, verifikasi dokumen, dan approval PKL.
2. Satu **binary Go** menyajikan UI server-rendered (templ + HTMX + Alpine) **dan** API JSON
   di port yang sama.
3. Data utama masih **satu dokumen JSON raksasa** (`app_state`) di Postgres. Tabel domain
   sudah ada dan sebagian sudah jadi sumber baca — ini fase transisi, bukan dua sistem permanen.
4. Login: Argon2id → sesi acak di Redis (satu perangkat per akun) → MFA TOTP **wajib** untuk
   Admin TU, Bendahara, Kepala Sekolah, Developer.
5. Status jujur: **layak demo/pilot dengan data contoh**, belum layak menyimpan data
   siswa/keuangan sekolah yang nyata sampai migrasi data selesai.

---

## 1. Apa yang sebenarnya dibangun

Sekolah menengah kejuruan punya banyak pekerjaan administratif yang saling terkait:
kehadiran, nilai, uang SPP, PKL, bimbingan konseling, surat-menyurat, ujian. WESMK
menggabungkan semuanya ke satu aplikasi web dengan **9 peran** yang masing-masing hanya
melihat menu dan data yang relevan.

Falsafah produk (dari blueprint asli, tetap dipakai):

- Tidak ada ranking 1–40. Apresiasi pakai "Si Paling ..." berbasis perilaku nyata.
- Orang tua adalah partner, bukan pengawas: dashboard menekankan perkembangan, bukan menghakimi.
- Uang dan nilai **tidak boleh** dihitung di browser. Server yang memegang kunci jawaban,
  saldo, dan kwitansi.
- Satu layar satu tujuan. Tanpa grafik dekoratif atau metrik palsu.

**Skala kode saat ini:** ±21.000 baris Go + ±5.500 baris templ, 17 migrasi SQL,
49 file test Go.

| Lapisan | Teknologi | Catatan |
|---|---|---|
| Bahasa | Go 1.26 | satu binary, tanpa Node di produksi |
| UI | templ + HTMX 2 + Alpine.js + Tailwind | aset di-embed ke binary, tanpa build step |
| Database | PostgreSQL 16 | pgx v5, migrasi SQL tertanam |
| Cache/sesi | Redis 7 | sesi, rate limit, token MFA/magic link |
| AI | Gemini (`internal/ai`) | opsional; mati kalau `GEMINI_API_KEY` kosong |
| Deploy | Docker distroless, non-root | baca-saja, health check |

---

## 2. Siapa penggunanya, dan di mana mereka masuk

Semua portal punya pola URL sama: `/<peran>/<tab>`. Refresh dan bookmark aman.

| Peran | Kode | Cara login | Portal & tab |
|---|---|---|---|
| Siswa | `student` | NISN + kata sandi | `/student` — dashboard, classroom, exam-lobby, exam-kiosk, documents, academic, rewind |
| Orang tua | `parent` | **Magic link WhatsApp** (tanpa kata sandi) | `/parent` — dashboard, roadmap, future-plan, services |
| Guru | `teacher` | NIP/username + kata sandi | `/teacher` — dashboard, classroom, exam-control, question-bank, homeroom-pkl |
| Kesiswaan/BK | `discipline` | kata sandi | `/discipline` — violations, transformation-index |
| Piket | `piket` | kata sandi | `/piket` — dashboard (radar kelas, log gerbang, inval) |
| Bendahara | `treasurer` | kata sandi + **MFA wajib** | `/treasurer` — dashboard, billing, exam-clearance, reports |
| Admin TU | `admin_tu` | kata sandi + **MFA wajib** | `/admin` — master-data, dapodik-erapor, pkl-bkk |
| Kepala Sekolah | `headmaster` | kata sandi + **MFA wajib** | `/headmaster` — executive |
| Developer | `developer` | kata sandi + **MFA wajib** | `/developer` — dashboard, resets, system |

Halaman non-portal:

| Route | Siapa | Fungsi |
|---|---|---|
| `/` | publik | beranda sekolah; akun yang sudah login langsung diarahkan ke portalnya |
| `/login` | publik | login + form magic link orang tua |
| `/login/mfa` | publik | langkah kedua MFA |
| `/security` | semua login | enrollment TOTP, recovery code, sesi aktif, riwayat login |
| `/verify/{hash}` | publik | verifikasi kwitansi/surat lewat HMAC server |
| `/pkl/approve/{token}` | supervisor DUDI | approve jurnal PKL dengan PIN |
| `/contact` | publik | kirim bug report |

**Akun dibuat bagaimana?** Admin TU mendaftarkan siswa/guru → satu transaksi membuat
profil `students`, akun login siswa, akun wali, dan relasi `users.student_profile_id`.
Akun staf juga dibuat dari portal Admin TU. Untuk bootstrap pertama, akun datang dari env
`WESMK_USERS`.

---

## 3. Arsitektur

### 3.1 Alur satu request

```
browser (HP/laptop)
   │  HTTPS
   ▼
http.Server (net/http stdlib)
   │  middleware, urut dari luar ke dalam:
   │  accessLog → recoverPanic → securityHeaders → mutationGuards (Origin)
   │  → attachAuth (cookie → Redis) → csrfGuard → enforcePrivilegedMFA
   ▼
ServeMux  ──┬── /api/*    handler JSON   → internal/state, internal/domain/*, internal/store
            ├── /<role>/* handler HTML   → internal/httpapi/ui_builder*.go → internal/ui/pages
            └── /assets, /ui-assets      aset yang di-embed
   ▼
Postgres (app_state + tabel domain)   Redis (sesi, limit, token)
```

Yang perlu diingat: **UI dan API berbagi logika yang sama**. Handler HTML memanggil builder
viewmodel yang membaca state/domain, bukan memanggil HTTP API sendiri.

### 3.2 Peta folder

```
cmd/wesmk/            entrypoint: load env → migrate → redis → seed → auth → http
internal/config/      baca & validasi env (production wajib 5 secret)
internal/auth/        Argon2id, login, sesi, magic link, TOTP, recovery, policy MFA
internal/authz/       satu matriks izin: Can(role, action, resource, ownership)
internal/db/          pool pgx + runner migrasi (schema_migrations + advisory lock)
internal/store/       Postgres per agregat + Redis + TOTP + audit
internal/domain/      logika bisnis per agregat, tanpa SQL/HTTP:
                      identity, finance, attendance, exam, pkl, bkk
internal/state/       dokumen app_state: scoping baca, merge tulis, mutasi, kalkulasi
internal/httpapi/     router, middleware, handler JSON, web handler, UI builder
internal/ui/          templ pages, komponen, layout, viewmodel, aset
internal/seed/        akun dari WESMK_USERS, bersihkan akun demo lama
internal/ai/          klien Gemini strict JSON
```

Aturan main yang dijaga: handler **tidak** menulis SQL; service domain tidak tahu HTTP;
`app_state` hanya disentuh oleh paket `state` dan `store`.

---

## 4. Model data — ini bagian yang paling bikin bingung

Dulu aplikasi menyimpan **semua** data sekolah dalam satu dokumen JSON (`app_state`).
Itu cepat dibangun tapi mustahil dijaga: tidak ada foreign key, tidak ada transaksi,
dua orang menyimpan bersamaan saling menimpa.

Sekarang ada **dua lapis, dan itu disengaja**:

```
                    ┌───────────────────────────────┐
   tulis ──────────▶│  app_state (JSONB, 1 baris)   │  ← masih sumber utama
                    └──────────────┬────────────────┘
                                   │ shadow sync / backfill
                                   ▼
                    ┌───────────────────────────────┐
                    │  tabel domain (FK, transaksi) │  ← sumber baca untuk 5 domain
                    └───────────────────────────────┘
```

Status per domain:

| Domain | Tabel | Dibaca dari tabel? | Ditulis ke tabel? |
|---|---|---|---|
| Profil siswa | `students` | ✅ dengan fallback blob | ✅ shadow |
| Tagihan & ledger | `tuition_invoices`, `financial_audit_ledgers` | ✅ | ✅ shadow |
| Presensi | `attendances` | ✅ | ✅ shadow |
| Bank soal & sesi ujian | `exams`, `exam_questions`, `exam_sessions` | ✅ | ✅ shadow |
| PKL & BKK | `pkl_*`, `bkk_*` | ✅ | ✅ shadow |
| Kelas, jadwal, tahun ajaran | `016`–`017` | ✅ | ✅ langsung tabel |
| Presensi harian, izin, jurnal, gerbang, inval | `017` | ✅ | ✅ langsung tabel |
| Akun, MFA, audit login | `012`, `014`, `015` | ✅ | ✅ langsung tabel |
| Tugas, pelanggaran, surat, OTP, refleksi, bug | – | ❌ masih blob | ❌ masih blob |

**Aturan transisi (jangan dilanggar):**

1. Satu agregat per PR. Tidak ada migrasi besar sekaligus.
2. Urutan cutover: shadow write → backfill → dual read (fallback blob) → cutover baca →
   bekukan key blob → hapus key.
3. Jangan pernah menambah **sumber kebenaran ketiga**. Kalau butuh data baru, pilih: blob
   atau tabel, lalu ikuti polanya.
4. Snapshot dari browser **tidak boleh** mengisi data sekolah yang kosong. Sudah ditutup:
   `PUT /api/state` mengembalikan 409 kalau server belum punya data.

Siapa boleh baca apa (sudah berlaku di `internal/state/scope.go`):

- **Siswa/ortu** — hanya record miliknya. Tulis dari mereka **di-merge**, tidak menimpa global,
  dan field administratif (ledger, nilai, saldo) diabaikan.
- **Staf** — allowlist per peran. Bendahara tidak menerima bank soal; piket tidak menerima
  keuangan; guru tetap menerima kunci jawaban karena menilai esai. Kunci jawaban, OTP,
  dan PIN supervisor dihapus untuk peran lain.
- **Kepala Sekolah** — snapshot penuh (dia pemilik laporan).
- **Tulis dokumen global** — hanya `admin_tu` dan `headmaster`.

---

## 5. Status nyata: sudah jalan vs belum

### Sudah jalan dan terverifikasi

| Area | Bukti |
|---|---|
| Login, sesi, logout, rate limit | `internal/auth/*`, test unit |
| MFA TOTP wajib + recovery code + audit login | `internal/auth/{totp,recovery,policy}.go`, migrasi `014`–`015` |
| Scoping baca siswa/ortu/staf + merge tulis | `internal/state/scope.go`, `authz` |
| Penilaian kuis & CBT di server (kunci jawaban tidak ke browser) | `POST /api/learning/quiz/submit`, `/exam/submit` |
| Pembayaran: intent VA, callback HMAC, kwitansi bertanda tangan, idempoten | `internal/httpapi/payment_handlers.go`, `internal/state/payment.go` |
| Kas tunai + PIN bendahara + dispensasi + reversal | `POST /treasurer/cash-payment`, `/dispensation`, `/reversal` |
| Verifikasi publik HMAC | `/verify/{hash}` |
| Magic link ortu (POST konfirmasi, token sekali pakai) | `ui_handlers.go`, `notify.go` |
| Upload dokumen & tugas (validasi magic bytes; dokumen 500 KB, tugas 10 MB; disimpan di `document_blobs`) | migrasi `005` |
| Master data: tahun ajaran, kelas, jadwal, rollover | `internal/store/masterdata.go`, migrasi `016`–`017` |
| Onboarding siswa + akun siswa + akun wali (satu transaksi) | `internal/store/onboarding.go` |
| E-sign kepala sekolah (hash + HMAC) | `executive_signatures`, migrasi `013` |
| Container distroless non-root, health check DB+Redis, migrasi idempoten | `Dockerfile`, `/api/health` |

### Belum ada / masih lemah

| Prioritas | Hal | Kenapa penting |
|---|---|---|
| Tinggi | Gateway VA nyata + rekonsiliasi harian | Endpoint siap, tapi belum ada provider. Tanpa ini SPP online belum nyata. |
| Tinggi | Timer ujian & status proctor di server | Sekarang masih dikirim/dikendalikan dari browser → bisa dicurangi. |
| Tinggi | OTP reversal kepala sekolah: TTL + sekali pakai | Sekarang disimpan di state dan terbaca staf (sudah di-redact, tapi tetap lemah). |
| Tinggi | Backup/restore + monitoring + alert | Belum ada sama sekali. Ini wajib sebelum data nyata. |
| Sedang | Worker/queue (reminder WA, rekonsiliasi, ekspor) | Belum ada `Asynq`/worker. Pengingat masih dicatat sebagai timestamp. |
| Sedang | Object storage (R2/S3) | Dokumen masih blob di Postgres; tumbuh terus. |
| Sedang | SSE proctoring | Pengawas belum lihat status real-time. |
| Sedang | Tugas, pelanggaran, surat, OTP → tabel | Masih blob. |
| Sedang | E2E browser di CI | Masih manual. |
| Rendah | Impor CSV, PDF rapor/kwitansi, PWA offline, Dapodik otomatis | Bisa menyusul. |

Dokumen `docs/production-readiness.md` memuat tabel blocker yang lebih rinci per file.
Kalau ada konflik dengan dokumen ini, **dokumen ini yang lebih baru untuk status produk**;
`production-readiness.md` yang lebih baru untuk detail teknis.

---

## 6. Keamanan: alur login & yang sudah dijaga

```
POST /login (identifier + password)
   ├─ cek rate limit: 5 gagal / 10 menit per IP+identifier
   ├─ Argon2id verify
   ├─ kalau akun nonaktif → tolak
   ├─ kalau peran wajib MFA & belum enroll → sesi dibuat, tapi hanya /security yang bisa dibuka
   └─ kalau MFA aktif → challenge di Redis → POST /login/mfa (TOTP atau recovery code)
        ▼
   sesi: token acak 32 byte di Redis (auth:session:*), cookie HttpOnly SameSite=Lax,
         Secure otomatis di production, TTL 8 jam, SATU sesi aktif per akun
```

Yang sudah dijaga:

- Password tidak pernah dikirim balik; hash tidak keluar dari server.
- CSRF: semua POST web wajib `X-CSRF-Token`; API mengandalkan cek `Origin` + JSON-only.
- Header: CSP, HSTS (production), nosniff, frame-ancestors none, permissions policy.
- Secret di production **fail-closed**: app menolak boot kalau `DATABASE_URL`, `REDIS_URL`,
  `WESMK_RECEIPT_SECRET` (<16), `WESMK_MFA_KEY` (<32), atau `WESMK_TREASURER_PIN` default.
- Secret TOTP dienkripsi AES-256-GCM; recovery code disimpan sebagai hash, sekali pakai.
- Rate limit: tulis state 120/menit per akun, AI 30/menit per akun.
- `X-Forwarded-For` hanya dipercaya kalau `WESMK_TRUST_PROXY=true` (default mati).

Yang **tidak** boleh dilakukan: menaruh PIN, OTP, kunci jawaban, atau saldo di browser,
di URL, atau di field baru dalam `app_state`.

---

## 7. Peta route

**API JSON** (semua butuh sesi kecuali health & auth):

| Method | Path | Guna |
|---|---|---|
| GET | `/api/health` | status Postgres + Redis (503 kalau degraded) |
| GET | `/api/auth/config` | peran mana yang wajib MFA |
| POST | `/api/auth/login`, `/mfa`, `/logout` | autentikasi |
| GET | `/api/auth/me` | identitas sesi |
| POST | `/api/auth/magic-link`, `/api/auth/magic/{token}` | orang tua |
| POST | `/api/learning/quiz/submit` | nilai kuis di server |
| POST | `/api/learning/exam/start`, `/exam/submit` | CBT |
| POST | `/api/payments/intent`, `/callback` | VA + webhook gateway |
| GET/PUT | `/api/state` | snapshot global (scoped per peran) |
| POST | `/api/state/reset` | admin_tu / headmaster saja |
| POST | `/api/ai/analyze` | analisis Gemini |

**Web form** (server-rendered, POST + redirect + flash): lihat `registerUIRoutes()` di
`internal/httpapi/ui_handlers.go` — itu daftar lengkap dan selalu akurat. Polanya:
`POST /<peran>/<aksi>`, misal `/teacher/journal`, `/admin/master/rollover`,
`/treasurer/cash-payment`.

---

## 8. Konfigurasi

### Wajib saat `WESMK_ENV=production`

| Env | Isi | Catatan |
|---|---|---|
| `WESMK_ENV` | `production` | mengaktifkan cookie Secure + HSTS + validasi |
| `DATABASE_URL` | `postgres://...` | wajib eksplisit |
| `REDIS_URL` | `redis://` atau `rediss://` | wajib eksplisit |
| `WESMK_RECEIPT_SECRET` | acak ≥16 char | menandatangani kwitansi |
| `WESMK_MFA_KEY` | acak ≥32 char | mengenkripsi secret TOTP. **Kalau hilang, semua akun harus reset MFA.** |
| `WESMK_TREASURER_PIN` | 6 digit, bukan `123456` | PIN kas bendahara |

### Opsional / sesuai kebutuhan

| Env | Efek kalau kosong |
|---|---|
| `APP_URL` | dipakai untuk cek Origin; sebaiknya diisi |
| `WESMK_USERS` | JSON array akun bootstrap; kosong = tidak ada akun baru |
| `GEMINI_API_KEY` | fitur AI mati |
| `WESMK_WA_WEBHOOK_URL` + `_TOKEN` | magic link ortu ditolak 503 di production |
| `WESMK_PAYMENT_WEBHOOK_SECRET` | callback pembayaran ditolak (fail-closed) |
| `WESMK_SECURE_COOKIE` | default `true` di production |
| `WESMK_TRUST_PROXY` | default `false`; isi `true` hanya di belakang proxy tepercaya |
| `PORT` | default `3000` |

### Flag cutover data (default semua `true`)

`WESMK_DOMAIN_STUDENT_PROFILE`, `WESMK_DOMAIN_FINANCE`, `WESMK_DOMAIN_ATTENDANCE`,
`WESMK_DOMAIN_EXAM`, `WESMK_DOMAIN_PKL_BKK` — set `false` untuk memaksa baca dari blob
kalau ada masalah setelah deploy. Ini tombol rollback per domain.

---

## 9. Jalanin di lokal (5 menit)

```bash
docker compose up -d          # Postgres 16 + Redis 7 (port dari .env, default 5432/6379)
go run ./cmd/wesmk            # http://localhost:3001 sesuai .env
```

- `templ generate` hanya perlu kalau lo mengubah file `.templ`. Binary templ ada di `~/go/bin/templ`.
- Akun bootstrap lokal dari `WESMK_USERS` di `.env` (file ini gitignored — jangan pernah di-commit).
- Server otomatis menjalankan migrasi, backfill domain, lalu seed akun.
- Cek cepat: `curl -s localhost:3001/api/health`.

Verifikasi wajib sebelum commit (aturan `AGENTS.md`):

```bash
go vet ./... && go test ./...
```

---

## 10. Deploy gratis untuk uji coba di HP

Tiga akun gratis, tidak perlu VPS/domain berbayar:

| Komponen | Layanan | Kenapa |
|---|---|---|
| App | **Render** web service (Docker) | build dari repo, HTTPS otomatis, subdomain gratis |
| Postgres | **Neon** free | 1 GB, tidak kedaluwarsa (Postgres gratis Render expire 30 hari) |
| Redis | **Upstash** free | 250 MB, beri `rediss://` |

Langkah: push repo ke GitHub → buat service di Render → isi env wajib (bagian 8) →
deploy → buka URL di HP → login Admin TU → scan QR MFA.

Batasan yang harus diterima: service gratis Render **tidur setelah 15 menit** (bangun
30–60 detik), dan kuota Neon kecil. Cukup untuk tes sendiri, bukan untuk jam sekolah.

---

## 11. Blueprint ke depan

Sumber: `docs/master-plan-2026-11-23.md`. Definisi "berfungsi" = data dari tabel +
otorisasi server + audit + idempoten + error jelas + ada test + tanpa data demo.

| Fase | Isi | Status |
|---|---|---|
| **A** | Tutup P0: demo payment, PIN DUDI, OTP, `/verify` bocor, CSP, error ditelan | ✅ selesai 23 Nov |
| **B** | Platform & data: paket domain, authz, lifecycle akun, MFA, master data, worker, storage, backfill | 🔶 sebagian (B.1–B.3 jalan; worker/storage/backfill penuh belum) |
| **C** | Keuangan & gatekeeper: tabel jadi sumber utama, gateway VA nyata, rekonsiliasi, PDF | ⬜ belum |
| **D** | CBT & proctoring: timer di Redis, lock server, SSE, antrean esai, uji 1.000 sesi | ⬜ belum |
| **E** | Akademik & kelas: presensi, jurnal, rapor 360 reproducible, badge "Si Paling" | 🔶 presensi & jurnal sudah; rapor belum |
| **F** | PKL/BKK/karier: geofence, PIN DUDI aman, tracer study | 🔶 data & portal jalan; geofence belum |
| **G** | Peran lain & admin: pelanggaran, peer report, surat PDF, KPI kepsek, audit viewer | 🔶 tersebar, belum tuntas |
| **H** | Integrasi: WA nyata, VA, object storage, notifikasi, job terjadwal | ⬜ belum |
| **I** | Kualitas: E2E CI, load test, backup drill, monitoring, a11y | ⬜ belum |

**Kalau waktu terbatas, urutan yang jangan ditawar untuk pilot satu sekolah:**
A ✅ → B (tuntas) → C → D → E dasar → G.3 (akun & master data).
PKL/BKK, AI, PWA, dan Dapodik otomatis boleh menyusul.

**Langkah berikutnya yang paling masuk akal, berurutan:**

1. **Deploy gratis** dan pakai sendiri seminggu. Ini mengungkap bug yang tidak terlihat di lokal.
2. **Backup + monitoring** (`pg_dump` harian + uptime check `/api/health`). Murah, menyelamatkan.
3. **Tutup dua celah uang/ujian**: timer ujian di server, lalu OTP reversal ber-TTL.
4. **Pilih gateway VA** dan sambungkan sandbox → baru berani terima SPP online.
5. **Lanjutkan cutover blob → tabel** satu agregat per PR: tugas, pelanggaran, surat.

---

## 12. Cara kerja harian (biar tidak bingung mulai dari mana)

| Mau ngapain | Sentuh file | Verifikasi |
|---|---|---|
| Ubah tampilan satu tab | `internal/ui/pages/<portal>.templ` + `ui_builder*.go` | `templ generate`, buka halamannya |
| Tambah aksi/form | `ui_handlers.go` + `internal/state/mutations.go` | `go test ./...` |
| Ubah logika bisnis | `internal/domain/<agregat>` | `go test ./internal/domain/...` |
| Ubah skema DB | **file migrasi baru** di `internal/db/migrations/` (jangan edit yang lama) | jalankan server, cek migrasi, `go test ./...` |
| Ubah izin peran | `internal/authz/authz.go` + `internal/state/scope.go` | `go test ./internal/authz ./internal/state` |
| Ubah env/konfigurasi | `internal/config/config.go` + `.env.example` + README | `go test ./internal/config` |

Aturan repo yang tetap berlaku:

- Jangan reintroduce React/Vue/Next. UI tetap templ + HTMX + Alpine.
- Jangan commit secret; `.env` gitignored.
- Commit hanya file yang disentuh (pathspec), jangan `git add -A`.
- Perubahan skema **hanya** lewat migrasi baru, harus idempoten untuk DB kosong dan DB berisi.
- Jangan menulis ulang UI atau mengganti stack di tengah jalan.

---

## 13. Glosarium

| Istilah | Arti di proyek ini |
|---|---|
| **app_state / blob** | satu dokumen JSON di Postgres berisi hampir semua data sekolah |
| **tabel domain** | tabel Postgres ber-FK per agregat (students, invoices, attendances, ...) |
| **shadow write** | menulis ke blob dan tabel sekaligus dalam satu transaksi |
| **dual read** | baca dari tabel, kalau kosong fallback ke blob |
| **cutover** | mematikan fallback blob untuk satu domain |
| **scoped** | peran siswa/ortu; hanya melihat record miliknya |
| **gatekeeper** | aturan server yang memblokir ujian kalau SPP menunggak |
| **MFA / TOTP** | kode 6 digit dari aplikasi authenticator |
| **magic link** | tautan sekali pakai 5 menit untuk login orang tua |
| **VA** | virtual account untuk pembayaran transfer |
| **HMAC** | tanda tangan server untuk kwitansi/surat agar tidak bisa dipalsukan |
| **single-device guard** | login baru membatalkan sesi lama akun yang sama |
| **DUDI** | dunia usaha/dunia industri, mitra PKL |

---

## 14. Dokumen lain: mana yang dipakai, mana yang arsip

| Dokumen | Status | Pakai untuk |
|---|---|---|
| `docs/PETA-PROYEK.md` (ini) | **aktif** | pintu masuk, status, arah |
| `docs/production-readiness.md` | **aktif** | detail blocker teknis + checklist deploy |
| `docs/master-plan-2026-11-23.md` | **aktif** | rincian fase A–I, matriks kepemilikan data |
| `docs/pelajaran/` | **aktif** | catatan bug nyata & pelajarannya (wajib diisi tiap fix) |
| `README.md` | **aktif** | cara jalan & ringkasan cepat |
| `AGENTS.md` (+ `CLAUDE.md`, `GEMINI.md`) | **aktif** | aturan kerja repo |
| `docs/arsip/` | arsip | audit lama, rencana awal, matriks blueprint React-era |

Dokumen arsip sudah dikeluarkan dari root supaya tidak terbaca sebagai status terkini; indeksnya
ada di [`docs/arsip/README.md`](arsip/README.md). Kalau dokumen lama menyebut "mode demo" atau
`WESMK_DEMO_MODE`, itu **sudah tidak ada** — demo dihapus total dari kode pada 25 November 2026.
