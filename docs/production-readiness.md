# WESMK — Audit Kesiapan Produksi

Terakhir diperbarui: 25 November 2026

> Peta proyek, status per peran, dan arah kerja ada di [`PETA-PROYEK.md`](PETA-PROYEK.md).
> Dokumen ini fokus pada blocker teknis dan checklist deploy.

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
| Config production | `DATABASE_URL`/`REDIS_URL`/`WESMK_RECEIPT_SECRET`/`WESMK_MFA_KEY`/`WESMK_TREASURER_PIN` wajib eksplisit (fail-closed); cookie `Secure` default aktif |
| AI | API key lewat header (bukan URL), error internal tidak lagi dikirim ke browser |
| Tulis state | Rate limit 120 tulisan/menit per akun; simpan dari klien diserialkan agar respons lama tidak menimpa snapshot baru |
| Frontend | Flash message, layar "data siswa belum tersedia", banner gagal sinkronisasi, `lang="id"`, timeout AI 45 detik |
| Magic link | Nomor ortu dinormalisasi saat upsert dan lookup langsung dengan index |
| Deploy | `Dockerfile` multi-stage (Go statis → distroless non-root), aset UI di-embed, sudah diuji read-only |
| Test | Unit test config, scoping state, auth, penilaian, pembayaran, tanda tangan webhook; `go vet` + `go test` hijau |
| Nilai kuis & CBT | `POST /api/learning/quiz/submit` dan `/exam/submit`: server memegang kunci jawaban, menghitung skor, menolak duplikat/jawaban ganda, satu attempt per siswa/ujian, menandai proctor `SUBMITTED`; kunci jawaban & rubrik dihapus dari respons siswa |
| Pembayaran SPP | `POST /api/payments/intent` (VA pending), `/callback` (satu-satunya jalur settle, HMAC + timestamp, idempoten). `paidAmount`, status tagihan, dan ledger hanya ditulis lewat transaksi terkunci; kwitansi ditandatangani `WESMK_RECEIPT_SECRET`; VA kedaluwarsa tidak bisa disettle |
| Scoping tulis | `PUT /api/state` dari siswa/ortu tidak lagi menerima `dailyQuizAvg`, `cbtScore`, `paidAmount`, atau entri ledger; nilai dan saldo hanya berubah lewat endpoint domain |

## Blocker produksi (harus selesai sebelum data nyata)

| Prioritas | Masalah | Bukti | Arah perbaikan |
|---|---|---|---|
| Kritis | Status proctor (warn/lock/unlock) dan OTP reversal belum sepenuhnya server-authoritative: status masih dikirim browser, OTP disimpan di `app_state` tanpa TTL dan ditampilkan di UI | `internal/httpapi/ui_handlers.go` `handleStudentProctorStatusForm`; `internal/state/mutations.go` `ExecuteReversal`, `GenerateReversalOTP` | Server memegang timer/status ujian; OTP per-request, ber-TTL, tidak ditampilkan |
| Sebagian | Profil siswa, tagihan, ledger, presensi, bank soal, sesi ujian, PKL, dan BKK sudah dibaca dari tabel domain (fallback blob) | `internal/domain/{identity,finance,attendance,exam,pkl,bkk}`, migrasi `007`–`011` | Bekukan key blob per agregat, lalu hapus fallback setelah masa dual-read |
| Selesai | Scoping baca staf per peran: allowlist field di `state.StaffView`; kunci jawaban/OTP/PIN dihapus untuk peran yang tidak perlu; tulis dokumen global hanya `admin_tu`/`headmaster` | `internal/state/scope.go` `staffKeys`, `state_handlers.go` | Scoping tulis per peran staf masih belum selengkap baca (sebagian aksi form masih menulis key blob di luar domainnya) |
| Selesai | Snapshot klien tidak bisa lagi menjadi data awal sekolah: `PUT /api/state` mengembalikan 409 saat `app_state` kosong | `internal/httpapi/state_handlers.go` `errStateNotInitialized` | — |
| Selesai | Unggah dokumen: multipart, batas 500 KB, validasi MIME/magic bytes di server, blob di `document_blobs`, metadata server-owned | `internal/httpapi/ui_handlers.go` `handleStudentDocUploadForm`, migrasi `005_document_blobs.sql` | Object storage R2 belum dipakai — blob masih di Postgres |
| Selesai | Magic link ortu: webhook WA via `WESMK_WA_WEBHOOK_URL` (production fail-closed 503 bila kosong); GET hanya halaman konfirmasi, token dikonsumsi lewat POST | `internal/httpapi/notify.go`, `ui_handlers.go`, `auth_handlers.go` | — |
| Selesai | Verifikasi publik kwitansi/surat already HMAC server-side (`/verify/{hash}` mencocokkan `signatureHmac`/`hmacHash` yang dibuat server) | `internal/httpapi/ui_handlers.go` `handlePublicVerifyPage` | — |
| Kritis | OTP kepala sekolah tersimpan di state dan terbaca semua staf | `activeHeadmasterOtp` | OTP per permintaan reversal, di-hash, TTL pendek, rate limit, hanya kepala sekolah |
| Tinggi | Belum ada backup/restore, monitoring, dan alert | — | `pg_dump`/WAL terjadwal, uptime check `/api/health`, log retention |
| Selesai | Lifecycle akun, MFA wajib untuk Admin TU/Bendahara/Kepala Sekolah/Developer, recovery code sekali pakai, secret TOTP terenkripsi AES-256-GCM, sesi aktif, dan audit login append-only | `internal/auth/{totp,recovery,policy}.go`, `internal/secure/mfa.go`, `internal/store/{security,totp,redis}.go`, migrasi `012`, `014`, `015` | Rotasi `WESMK_MFA_KEY` harus disertai reset MFA semua akun; halaman sesi saat ini menampilkan satu sesi karena desain single-device |
| Selesai | Halaman sesi aktif dan audit login tersedia di `/security`; Admin TU/Developer juga melihat audit login seluruh akun | `internal/httpapi/ui_handlers.go`, `internal/store/security.go`, `internal/ui/pages/security.templ` | Ekspor audit ke SIEM/retensi eksternal belum ada |
| Selesai | Onboarding siswa dari Admin TU membuat profil `students`, akun login siswa, akun wali, dan relasi `users.student_profile_id` secara transaksional; form mengisi nama/WA wali dan kata sandi awal | `internal/store/onboarding.go`, `internal/httpapi/ui_handlers.go`, `internal/state/mutations.go` | Satu akun wali masih terhubung ke satu anak; keluarga dengan saudara kandung butuh tabel relasi multi-anak |
| Selesai | Master data tahun ajaran (satu aktif), kelas per jurusan/wali kelas, dan jadwal per kelas tersimpan di Postgres; kelas/jadwal bisa diubah dan dihapus dengan proteksi relasi | `internal/store/masterdata.go`, `internal/store/portal_ops.go`, migrasi `016`–`017` | Tarif SPP per tahun dan profil sekolah belum ada |
| Selesai | Rollover tahun ajaran eksplisit: setiap kelas dipetakan ke kelas tujuan sejurusan, tingkat 12 diluluskan, siswa aktif dipindahkan beserta sinkronisasi nama/jurusan, lalu tahun tujuan diaktifkan dalam satu transaksi dan audit | `internal/store/masterdata.go` `RolloverStudents`, `internal/httpapi/ui_handlers.go`, `internal/ui/pages/admin_portal.templ` | Belum ada rollover otomatis dari kalender atau pembatalan rollover |
| Selesai | Presensi per kelas/tanggal, izin orang tua per tanggal, jurnal mengajar per slot jadwal, log gerbang tervalidasi NISN, dan penugasan inval tersimpan di tabel domain | `internal/db/migrations/017_portal_operations.sql`, `internal/store/portal_ops.go` | Timer/status proctor dan rekap presensi lintas semester belum |
| Selesai | Tugas guru difilter berdasarkan guru dan kelas yang diampu; penilaian ditolak bila tugas bukan milik kelas tersebut | `internal/httpapi/ui_builder.go`, `internal/httpapi/ui_handlers.go` | Tugas masih disimpan di `app_state`, belum menjadi tabel domain |
| Selesai | Halaman Admin TU dipecah per bagian (Siswa, Kelas & Jadwal, Akun, Dokumen), dengan direktori siswa, pencarian, pagination, form ubah/hapus kelas dan jadwal, serta panel kenaikan kelas/kelulusan | `internal/ui/pages/admin_portal.templ`, `internal/store/portal_ops.go`, `internal/store/masterdata.go` | Impor CSV belum |
| Sebagian | E-sign laporan kepala sekolah kini menyimpan hash + HMAC dan audit, tetapi belum menghasilkan PDF bertanda tangan | `internal/store/executive.go`, migrasi `013` | Buat PDF + QR verifikasi dari hash tersimpan |
| Tinggi | Adapter gateway VA nyata, rekonsiliasi harian, dan tombol tarik status belum ada | Endpoint intent/callback sudah siap, tetapi belum ada kredensial gateway | Sambungkan provider VA, jadwalkan rekonsiliasi, dan uji callback sandbox |

## Catatan penting (bukan blocker demo)

- CSRF mengandalkan `Origin` + `SameSite=Lax`; belum ada token CSRF eksplisit.
- Login dibatasi per IP+identifier (5/10 menit); belum ada batas global per IP untuk credential spraying.
- Banner peringatan muncul saat muat/simpan state gagal, tetapi belum ada retry otomatis atau antrean offline.
- Tidak ada routing URL/deep link; refresh kembali ke tab default.
- Aset UI (HTMX, Alpine, `app.js`, `app.css`) di-embed ke binary; tidak ada build step Node.
- Font Google dan avatar Unsplash masih eksternal; pertimbangkan self-host untuk privasi/uptime.
- Unit test + integration Postgres sudah berjalan di CI; E2E browser masih manual dan belum menjadi gate CI.
- Matriks blueprint lama (`docs/arsip/requirements-traceability.md`) menandai banyak alur "READY"; itu berarti alur UI demo berjalan, bukan verifikasi produksi.
- Tugas siswa sudah memakai unggahan berkas nyata (PDF/JPG/PNG, maks 10 MB), pembuatan tugas per kelas dengan tenggat, penolakan pengumpulan ganda/lintas kelas/lewat tenggat, dan penilaian guru. Belum ada object storage, versioning tugas, atau notifikasi tenggat.

## Checklist deploy (pilot/demo terkontrol)

1. `docker build -t wesmk:latest .`
2. Siapkan Postgres 16 + Redis 7 privat (jangan buka ke publik), TLS ke DB.
3. Jalankan container sebagai non-root (sudah default) dengan env:
   `WESMK_ENV=production`, `APP_URL=https://...`,
   `DATABASE_URL=...`, `REDIS_URL=...`, `GEMINI_API_KEY=...`, `WESMK_USERS='[...]'`,
   `WESMK_RECEIPT_SECRET=...` (wajib, >= 16 karakter), `WESMK_MFA_KEY=...` (wajib, >= 32 karakter acak),
   `WESMK_PAYMENT_WEBHOOK_SECRET=...`,
   `WESMK_TREASURER_PIN=...` (bukan `123456`), `WESMK_WA_WEBHOOK_URL=...`
   (wajib bila menerima pembayaran VA; kosong = callback 503).
4. `WESMK_SECURE_COOKIE` biarkan default (`true` di production).
5. Pasang reverse proxy TLS; set `WESMK_TRUST_PROXY=true` hanya jika proxy tepercaya.
6. Health check container ke `/api/health`; alert bila 503.
7. Backup harian Postgres + uji restore; simpan log minimal 30 hari.
8. Ganti kata sandi akun bootstrap `WESMK_USERS` setelah login pertama.
9. Batasi akses jaringan ke port aplikasi; jangan ekspos Postgres/Redis.

## Bukti verifikasi audit

- `go test ./...`, `go vet ./...` — lulus; `templ generate` — nol diff.
- `govulncheck ./...` — 0 reachable vulnerability.
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
