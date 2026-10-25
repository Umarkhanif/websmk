# WESMK — Audit Kesiapan Produksi

Terakhir diperbarui: 25 Oktober 2026

## Putusan

UI dan alur 8 peran sudah lengkap untuk **demo/pilot dengan data contoh**. Backend Go kini aman
untuk di-deploy sebagai aplikasi web: auth, sesi, scoping baca siswa/ortu, container non-root,
health check, dan migrasi versioned.

Nilai kuis/CBT dan pembayaran SPP kini **dihitung dan dicatat di server** (kunci jawaban tidak lagi
dikirim ke browser, kwitansi ditandatangani HMAC). Yang belum: presensi, status proctor, OTP, tanda
tangan surat, unggahan dokumen, dan seluruh data sekolah masih satu dokumen JSON. Karena itu sistem
**belum layak menyimpan data siswa, nilai, atau keuangan sekolah yang nyata**.

## Yang diperbaiki pada audit ini

| Area | Perubahan |
|---|---|
| Migrasi DB | `schema_migrations` + advisory lock; setiap file hanya dijalankan sekali, aman untuk deploy ulang |
| Health check | `/api/health` mengecek Postgres **dan** Redis, balas 503 saat degraded |
| Header keamanan | CSP, HSTS (production), COOP, nosniff, frame-ancestors, permissions policy |
| Cache aset | `assets/*` immutable 1 tahun, `index.html` no-cache |
| Config production | `DATABASE_URL`/`REDIS_URL` wajib eksplisit; cookie `Secure` default aktif; peringatan demo mode |
| AI | API key lewat header (bukan URL), error internal tidak lagi dikirim ke browser |
| Tulis state | Rate limit 120 tulisan/menit per akun; simpan dari klien diserialkan agar respons lama tidak menimpa snapshot baru |
| Frontend | Error boundary, layar "data siswa belum tersedia", banner gagal sinkronisasi, `lang="id"`, timeout AI 45 detik |
| Magic link | Nomor ortu dinormalisasi saat upsert dan lookup langsung dengan index |
| Deploy | `Dockerfile` multi-stage (Node build → Go statis → distroless non-root), sudah diuji read-only |
| Test | Unit test config, scoping state, auth, penilaian, pembayaran, tanda tangan webhook; `npm audit` dan `govulncheck` bersih |
| Nilai kuis & CBT | `POST /api/learning/quiz/submit` dan `/exam/submit`: server memegang kunci jawaban, menghitung skor, menolak duplikat/jawaban ganda, satu attempt per siswa/ujian, menandai proctor `SUBMITTED`; kunci jawaban & rubrik dihapus dari respons siswa |
| Pembayaran SPP | `POST /api/payments/intent` (VA pending), `/pay` (hanya mode demo), `/callback` (HMAC + timestamp, idempoten). `paidAmount`, status tagihan, dan ledger hanya ditulis lewat transaksi terkunci; kwitansi ditandatangani `WESMK_RECEIPT_SECRET`; VA kedaluwarsa tidak bisa disettle |
| Scoping tulis | `PUT /api/state` dari siswa/ortu tidak lagi menerima `dailyQuizAvg`, `cbtScore`, `paidAmount`, atau entri ledger; nilai dan saldo hanya berubah lewat endpoint domain |

## Blocker produksi (harus selesai sebelum data nyata)

| Prioritas | Masalah | Bukti | Arah perbaikan |
|---|---|---|---|
| Kritis | Presensi, status proctor (warn/lock/unlock), OTP, dan hash surat masih dihitung/ditulis dari browser lalu disimpan server | `src/App.tsx`, `StudentView.tsx`, `MergeOwned` menerima `proctorStudents`, `activeHeadmasterOtp` | Pindahkan ke endpoint per domain seperti nilai/pembayaran; server memegang timer, status ujian, dan OTP ber-TTL |
| Kritis | Pembayaran tunai bendahara masih membuat entri ledger + HMAC di browser | `App.tsx` `handleRecordCashPayment` | Endpoint kas dengan PIN bendahara, idempotensi, dan nomor kwitansi dari server |
| Kritis | Semua data sekolah satu blob `app_state`; tidak ada audit trail, foreign key, atau pengaman lost update | `internal/store/postgres.go`, `state_handlers.go` | Normalisasi tabel per entitas, API per entitas, transaksi + optimistic lock, audit log append-only |
| Kritis | Staf mana pun bisa membaca seluruh state (keuangan, BK, catatan wali) dan menulis field di luar perannya | GET staff mengembalikan snapshot penuh | Scoping baca/tulis per peran staf, bukan hanya siswa/ortu |
| Kritis | Saat `app_state` kosong, snapshot klien (termasuk data mock) menjadi isi database | `MergeOwned` mengembalikan `incoming` untuk state kosong | Setup/import data awal di server; tolak seed dari klien di production |
| Kritis | Unggah dokumen tidak menyimpan file; validasi magic bytes hanya di klien | `StudentView.tsx` mengirim metadata ke state | Endpoint multipart, validasi ukuran/MIME di server, object storage + presigned URL |
| Kritis | Magic link ortu belum punya kanal pengiriman; tautan GET bisa habis dipakai bot preview | `auth_handlers.go` hanya log / `devLink` | Integrasi WhatsApp gateway + halaman konfirmasi POST sebelum sesi dibuat |
| Kritis | Verifikasi publik kwitansi/surat memakai state browser, bukan HMAC server | `PublicVerifyView.tsx`, `App.tsx` `Math.random()` | Tanda tangan HMAC di server, endpoint `/verify/:hash`, QR berisi URL server |
| Kritis | OTP kepala sekolah tersimpan di state dan terbaca semua staf | `activeHeadmasterOtp` | OTP per permintaan reversal, di-hash, TTL pendek, rate limit, hanya kepala sekolah |
| Tinggi | Belum ada backup/restore, monitoring, dan alert | — | `pg_dump`/WAL terjadwal, uptime check `/api/health`, log retention |
| Tinggi | Belum ada reset password, lifecycle akun, dan MFA untuk bendahara/kepala sekolah | — | Alur undangan/reset, MFA untuk aksi keuangan |
| Tinggi | Adapter gateway VA nyata, rekonsiliasi harian, dan tombol tarik status belum ada | Endpoint intent/callback sudah siap, tetapi belum ada kredensial gateway | Sambungkan provider VA, jadwalkan rekonsiliasi, dan uji callback sandbox |

## Catatan penting (bukan blocker demo)

- CSRF mengandalkan `Origin` + `SameSite=Lax`; belum ada token CSRF eksplisit.
- Login dibatasi per IP+identifier (5/10 menit); belum ada batas global per IP untuk credential spraying.
- Banner peringatan muncul saat muat/simpan state gagal, tetapi belum ada retry otomatis atau antrean offline.
- Tidak ada routing URL/deep link; refresh kembali ke tab default.
- Bundle JS ~560 KB; pertimbangkan code splitting.
- Font Google dan avatar Unsplash masih eksternal; pertimbangkan self-host untuk privasi/uptime.
- Test HTTP end-to-end masih manual (curl + headless Chrome), belum masuk CI.
- `docs/requirements-traceability.md` menandai banyak alur "READY"; artinya alur UI demo berjalan, bukan verifikasi produksi.

## Checklist deploy (pilot/demo terkontrol)

1. `docker build -t wesmk:latest .`
2. Siapkan Postgres 16 + Redis 7 privat (jangan buka ke publik), TLS ke DB.
3. Jalankan container sebagai non-root (sudah default) dengan env:
   `WESMK_ENV=production`, `WESMK_DEMO_MODE=false`, `APP_URL=https://...`,
   `DATABASE_URL=...`, `REDIS_URL=...`, `GEMINI_API_KEY=...`, `WESMK_USERS='[...]'`,
   `WESMK_RECEIPT_SECRET=...` (wajib, >= 16 karakter), `WESMK_PAYMENT_WEBHOOK_SECRET=...`
   (wajib bila menerima pembayaran VA; kosong = callback 503).
4. `WESMK_SECURE_COOKIE` biarkan default (`true` di production).
5. Pasang reverse proxy TLS; set `WESMK_TRUST_PROXY=true` hanya jika proxy tepercaya.
6. Health check container ke `/api/health`; alert bila 503.
7. Backup harian Postgres + uji restore; simpan log minimal 30 hari.
8. Rotasi password akun demo, atau matikan demo mode sepenuhnya.
9. Batasi akses jaringan ke port aplikasi; jangan ekspos Postgres/Redis.

## Bukti verifikasi audit

- `go test ./...`, `go vet ./...`, `tsc --noEmit`, `npm test`, `npm run build` — lulus.
- `npm audit --omit=dev` — 0 vulnerability; `govulncheck ./...` — 0 reachable vulnerability.
- `/api/health` mengembalikan status Postgres/Redis; migrasi tercatat di `schema_migrations`.
- Rate limit tulis state: 120×200 lalu 429 pada request ke-121.
- CSP diuji di Chrome headless: portal kepala sekolah render, font termuat, 0 error security/CSP.
- Image `wesmk:audit` dijalankan `--read-only --network host`: health 200, login 200, SPA 200.
- Scoping siswa/ortu diuji ulang: data sendiri, tamper lintas siswa ditolak, dan klien tidak bisa menaikkan
  `dailyQuizAvg`, `cbtScore`, atau `paidAmount` lewat `PUT /api/state`.
- E2E curl: kuis 3/3 → `score 100` dan replay tetap 100; CBT 1/2 MCQ → `50`, esai masuk antrean, proctor
  `SUBMITTED`; SPP 350.000 → `KWT-VA-*` + HMAC 64 hex; bayar ulang → 409; callback tanpa tanda tangan → 401;
  callback valid → PAID, replay → `alreadyPaid`; `/pay` di production → 404.
- E2E Chrome headless: kuis siswa menampilkan `Selesai • Skor 100/100` dari server, tombol pembayaran
  membuka gatekeeper CBT, tanpa error konsol/CSP.
