# WESMK — Ekosistem ERP SMK Terpadu

UI server-rendered dengan **Go + templ + HTMX + Alpine.js + Tailwind** (tanpa React), backend **Go + PostgreSQL 16 + Redis 7**.
Kontrak JSON (`/api/auth/*`, `/api/state`, `/api/ai/analyze`) dipertahankan untuk integrasi lama.

> **Baru mulai atau sudah lama tidak memegang proyek ini?** Baca
> [`docs/PETA-PROYEK.md`](docs/PETA-PROYEK.md) dulu — peta peran, arsitektur, status nyata,
> dan arah kerja ada di satu tempat.

## Prasyarat

- Go 1.26+ (lihat `go.mod`)
- Docker + Docker Compose (untuk Postgres & Redis lokal)

## Menjalankan (development)

```bash
docker compose up -d      # Postgres 16 + Redis 7
go run ./cmd/wesmk     # http://localhost:3000
```

Server Go menyajikan UI (templ/HTMX) dan seluruh endpoint API dari satu port (`PORT`, default `3000`).

Akun dibuat lewat `WESMK_USERS` (JSON array) saat server boot. Untuk localhost,
`.env` sudah berisi satu akun bootstrap **Admin TU** (`admin.tu` / `GantiSaya#2026`) —
segera ganti kata sandinya. Setelah login, Admin TU mendaftarkan siswa/guru/wali
lewat menu onboarding sehingga setiap orang punya akun nyata sendiri.

### Memasang portal di layar utama

Portal sudah menyertakan manifest dan ikon, jadi bisa dipasang seperti aplikasi.
Di Chrome Android pilih menu **Instal aplikasi** / **Tambahkan ke layar utama**;
di Safari iOS pilih **Bagikan → Tambahkan ke Layar Utama**. Portal terbuka tanpa
address bar, tetapi masih memerlukan koneksi karena belum ada mode offline.

### Mengganti foto halaman masuk

Foto login dibundel di `internal/ui/assets/images/` dan tidak mengambil gambar
dari internet saat runtime. Timpa `login-hero.jpg`, `login-campus.jpg`, dan
`login-lab.jpg` dengan foto sekolah masing-masing, lalu build ulang. Petunjuk
ukuran, lisensi, dan kredit ada di [`internal/ui/assets/images/README.md`](internal/ui/assets/images/README.md).

## Menjalankan (production)

```bash
go build -o wesmk-server ./cmd/wesmk
WESMK_ENV=production \
APP_URL=https://wesmk.example.sch.id \
DATABASE_URL='postgres://user:pass@host:5432/wesmk?sslmode=require' \
REDIS_URL='redis://host:6379/0' \
GEMINI_API_KEY='...' \
WESMK_RECEIPT_SECRET='...' \
WESMK_MFA_KEY='...' \
WESMK_PAYMENT_WEBHOOK_SECRET='...' \
WESMK_TREASURER_PIN='...' \
WESMK_WA_WEBHOOK_URL='https://wa-relay.example.sch.id/send' \
WESMK_WA_WEBHOOK_TOKEN='...' \
./wesmk-server
```


### Container (produksi)

```bash
docker build -t wesmk:latest .
docker run --rm -p 3000:3000 --read-only \
  -e WESMK_ENV=production \
  -e APP_URL=https://wesmk.example.sch.id \
  -e DATABASE_URL='postgres://user:pass@db-host:5432/wesmk?sslmode=require' \
  -e REDIS_URL='redis://redis-host:6379/0' \
  -e GEMINI_API_KEY='...' \
  -e WESMK_RECEIPT_SECRET='...' \
  -e WESMK_MFA_KEY='...' \
  -e WESMK_PAYMENT_WEBHOOK_SECRET='...' \
  -e WESMK_USERS='[{"identifier":"198509122010012005","password":"GantiSaya#2026","fullName":"Bendahara","role":"treasurer"}]' \
  wesmk:latest
```

Image memakai distroless, berjalan sebagai non-root, dan tidak menulis ke disk. Di production,
cookie sesi otomatis `Secure` dan `DATABASE_URL`/`REDIS_URL` wajib diisi eksplisit.

## Autentikasi & keamanan

- Password di-hash dengan **Argon2id**; hash tidak pernah dikirim ke browser.
- Sesi berupa token acak di **Redis** (`auth:session:*`) dengan cookie `HttpOnly`, `SameSite=Lax`,
  TTL 8 jam, dan **satu sesi aktif per akun** (single-device guard).
- Login dibatasi 5 percobaan gagal / 10 menit per IP + identifier (Redis).
- MFA TOTP **wajib** untuk `admin_tu`, `treasurer`, `headmaster`, dan `developer`. Sesi baru
  hanya dapat membuka `/security` sampai enrollment selesai. Secret dienkripsi AES-256-GCM
  dengan `WESMK_MFA_KEY`; 10 recovery code sekali pakai disimpan sebagai hash.
- `/security` menampilkan sesi aktif, riwayat login, dan (Admin TU/Developer) audit login global.
- Magic link orang tua: token sekali pakai, 5 menit, disimpan sebagai hash di Redis. Pengiriman
  WhatsApp lewat webhook `WESMK_WA_WEBHOOK_URL` (JSON `{"to","message"}`, Bearer token opsional).
  Di production tanpa webhook, permintaan ditolak 503.
- `GET/PUT /api/state` dan `POST /api/ai/analyze` wajib sesi. Reset state hanya untuk `admin_tu` dan
  `headmaster`.
- Endpoint AI dibatasi 30 request/menit per akun dan hanya menerima tipe yang dikenal.
- Request yang mengubah data ditolak bila `Origin` tidak cocok (CSRF defense-in-depth).
- Mode demo sudah dihapus dari kode. Akun nyata dibuat/di-upsert lewat `WESMK_USERS` saat
  startup; akun demo lama otomatis dihapus dari database.

Semua variabel lingkungan terdokumentasi di `.env.example`.

## Nilai & pembayaran (server-authoritative)

Browser tidak lagi menghitung nilai atau menulis saldo tagihan. Endpoint yang dipakai portal:

| Endpoint | Siapa | Perilaku |
|---|---|---|
| `POST /api/learning/quiz/submit` | siswa | Menilai kuis dengan kunci jawaban di server, satu attempt per siswa (replay mengembalikan skor pertama), memperbarui `dailyQuizAvg` |
| `POST /api/learning/exam/submit` | siswa | Mengarsipkan jawaban CBT, menilai MCQ (esai menunggu rubrik guru), duplicate question diabaikan, memperbarui `cbtScore` dan proctor `SUBMITTED` |
| `POST /api/payments/intent` | siswa/ortu | Membuat/mengembalikan VA pending untuk tagihan sendiri; tidak mengubah status bayar |
| `POST /api/payments/callback` | gateway | Satu-satunya jalur settle produksi; wajib `X-Wesmk-Timestamp` + `X-Wesmk-Signature` (HMAC body), idempoten |

Kunci jawaban, rubrik esai, dan `paidAmount` tidak pernah diterima dari browser. Respons
`GET /api/state` untuk siswa/ortu juga menghapus kunci jawaban dan rubrik. Semua mutasi memakai
`UPDATE ... FOR UPDATE` sehingga dua pembayaran bersamaan tidak saling menimpa.

## Struktur

```
cmd/wesmk/            entrypoint server Go
internal/auth/        Argon2id, login, sesi, magic link, rate limit
internal/httpapi/     router, middleware, handler auth/state/AI, UI templ/HTMX
internal/store/       Postgres (users, app_state) + Redis (sesi, limit, token)
internal/db/          koneksi pool + migrasi SQL tertanam
internal/seed/        akun dari WESMK_USERS + pembersihan akun demo lama
internal/state/       scoping baca & merge tulis per peran (siswa/orang tua)
internal/ai/          klien Gemini generateContent (strict JSON schema)
```

## Test & lint

```bash
go test ./...   # unit test Go
go vet ./...    # static analysis Go
```

## Kesiapan produksi

Status lengkap, blocker yang tersisa, dan checklist deploy ada di
[`docs/production-readiness.md`](docs/production-readiness.md). Ringkasnya: aplikasi aman
di-deploy untuk demo/pilot. Nilai kuis/CBT dan pembayaran SPP sudah otoritatif di server; presensi
harian, izin orang tua, jurnal mengajar, jadwal, penugasan piket, serta rollover kelas/kelulusan
sudah memakai tabel domain. Status proctor, OTP, surat, dan dokumen belum sepenuhnya
server-authoritative, dan banyak domain lain
masih dibaca dari `app_state` — jadi belum untuk seluruh data siswa/keuangan nyata sampai cutover
selesai.

## Catatan data

`/api/state` disimpan sebagai satu dokumen JSON global di Postgres (`app_state`), tetapi aksesnya
dibatasi per peran:

- **Siswa** hanya menerima profilnya sendiri, tagihan, pengeluaran, dispensasi, dokumen, tugas, PKL,
  lamaran BKK, presensi, sesi ujian, dan refleksi miliknya. Materi kelas (jurnal, kuis, lowongan, soal
  ujian, penempatan PKL) tetap dikirim karena dipakai portal siswa.
- **Orang tua** hanya menerima profil, tagihan, ledger, pengeluaran, presensi, konsultasi karier, dan
  refleksi anak yang terhubung (`studentProfileId`).
- **Staf** (guru, bendahara, kesiswaan, piket, TU) menerima allowlist field sesuai portalnya; kunci
  jawaban, OTP, dan PIN supervisor dihapus untuk peran yang tidak membutuhkan. **Kepala Sekolah**
  menerima snapshot penuh karena memegang laporan.
- `PUT /api/state` dari siswa/ortu tidak mengganti dokumen global. Hanya record miliknya yang digabung;
  unggahan dokumen masuk antrean review, dan field administratif (violations, ledger, OTP, pengaturan
  kelas, dsb.) diabaikan. Nilai (`dailyQuizAvg`, `cbtScore`) dan saldo tagihan tidak lagi diterima dari
  jalur ini — hanya dari endpoint nilai/pembayaran.
- Saat `app_state` masih kosong, server menolak `PUT /api/state` dengan 409. Snapshot dari browser
  tidak pernah dipakai sebagai data awal sekolah; data awal diisi dari server.
