# WESMK — Ekosistem ERP SMK Terpadu

UI server-rendered dengan **Go + templ + HTMX + Alpine.js + Tailwind** (tanpa React), backend **Go + PostgreSQL 16 + Redis 7**.
Kontrak JSON (`/api/auth/*`, `/api/state`, `/api/ai/analyze`) dipertahankan untuk integrasi lama.

## Prasyarat

- Go 1.26+ (lihat `go.mod`)
- Docker + Docker Compose (untuk Postgres & Redis lokal)

## Menjalankan (development)

```bash
docker compose up -d      # Postgres 16 + Redis 7
go run ./cmd/wesmk     # http://localhost:3000
```

Server Go menyajikan UI (templ/HTMX) dan seluruh endpoint API dari satu port (`PORT`, default `3000`).

Akun demo (mode demo aktif di luar production), kata sandi `Wesmk#2026`:

| Peran | NISN / NIP |
|---|---|
| Murid (Budi) | `0058471290` |
| Murid (Siti) | `0058471291` |
| Orang Tua | `081234567800` |
| Guru | `197804152005011003` |
| Bendahara | `198509122010012005` |
| Kesiswaan & BK | `197503112000031002` |
| Guru Piket | `198811202014021004` |
| Admin TU | `199201052019032011` |
| Kepala Sekolah | `196805121994121001` |

## Menjalankan (production)

```bash
go build -o wesmk-server ./cmd/wesmk
WESMK_ENV=production \
APP_URL=https://wesmk.example.sch.id \
DATABASE_URL='postgres://user:pass@host:5432/wesmk?sslmode=require' \
REDIS_URL='redis://host:6379/0' \
GEMINI_API_KEY='...' \
WESMK_RECEIPT_SECRET='...' \
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
  -e WESMK_DEMO_MODE=false \
  -e APP_URL=https://wesmk.example.sch.id \
  -e DATABASE_URL='postgres://user:pass@db-host:5432/wesmk?sslmode=require' \
  -e REDIS_URL='redis://redis-host:6379/0' \
  -e GEMINI_API_KEY='...' \
  -e WESMK_RECEIPT_SECRET='...' \
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
- Magic link orang tua: token sekali pakai, 5 menit, disimpan sebagai hash di Redis. Pengiriman
  WhatsApp lewat webhook `WESMK_WA_WEBHOOK_URL` (JSON `{"to","message"}`, Bearer token opsional).
  Di production tanpa webhook, permintaan ditolak 503; di mode demo tautan dikembalikan ke browser.
- `GET/PUT /api/state` dan `POST /api/ai/analyze` wajib sesi. Reset state hanya untuk `admin_tu` dan
  `headmaster`.
- Endpoint AI dibatasi 30 request/menit per akun dan hanya menerima tipe yang dikenal.
- Request yang mengubah data ditolak bila `Origin` tidak cocok (CSRF defense-in-depth).
- Mode demo hanya menyala bila `WESMK_DEMO_MODE=true` (default di luar production). Di production,
  akun demo dihapus saat startup dan akun nyata diisi lewat `WESMK_USERS`.

Semua variabel lingkungan terdokumentasi di `.env.example`.

## Nilai & pembayaran (server-authoritative)

Browser tidak lagi menghitung nilai atau menulis saldo tagihan. Endpoint yang dipakai portal:

| Endpoint | Siapa | Perilaku |
|---|---|---|
| `POST /api/learning/quiz/submit` | siswa | Menilai kuis dengan kunci jawaban di server, satu attempt per siswa (replay mengembalikan skor pertama), memperbarui `dailyQuizAvg` |
| `POST /api/learning/exam/submit` | siswa | Mengarsipkan jawaban CBT, menilai MCQ (esai menunggu rubrik guru), duplicate question diabaikan, memperbarui `cbtScore` dan proctor `SUBMITTED` |
| `POST /api/payments/intent` | siswa/ortu | Membuat/mengembalikan VA pending untuk tagihan sendiri; tidak mengubah status bayar |
| `POST /api/payments/pay` | siswa/ortu | **Hanya mode demo**: settle langsung + kwitansi HMAC |
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
internal/seed/        akun demo & akun dari WESMK_USERS
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
di-deploy untuk demo/pilot. Nilai kuis/CBT dan pembayaran SPP sudah otoritatif di server, tetapi
presensi, status proctor, OTP, surat, dan dokumen belum, dan data masih satu dokumen JSON — jadi belum
untuk data siswa/keuangan nyata sampai domain itu dipindahkan ke server dan data dinormalisasi.

## Catatan data

`/api/state` disimpan sebagai satu dokumen JSON global di Postgres (`app_state`), tetapi aksesnya
dibatasi per peran:

- **Siswa** hanya menerima profilnya sendiri, tagihan, pengeluaran, dispensasi, dokumen, tugas, PKL,
  lamaran BKK, presensi, sesi ujian, dan refleksi miliknya. Materi kelas (jurnal, kuis, lowongan, soal
  ujian, penempatan PKL) tetap dikirim karena dipakai portal siswa.
- **Orang tua** hanya menerima profil, tagihan, ledger, pengeluaran, presensi, konsultasi karier, dan
  refleksi anak yang terhubung (`studentProfileId`).
- **Staf** (guru, bendahara, kesiswaan, piket, TU, kepala sekolah) tetap menerima snapshot lengkap.
- `PUT /api/state` dari siswa/ortu tidak mengganti dokumen global. Hanya record miliknya yang digabung;
  unggahan dokumen masuk antrean review, dan field administratif (violations, ledger, OTP, pengaturan
  kelas, dsb.) diabaikan. Nilai (`dailyQuizAvg`, `cbtScore`) dan saldo tagihan tidak lagi diterima dari
  jalur ini — hanya dari endpoint nilai/pembayaran.
- Saat `app_state` masih kosong, snapshot klien pertama dipakai untuk mengisi data awal demo; setelah
  itu aturan di atas berlaku.
