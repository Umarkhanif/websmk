# PLAN BESAR PENGEMBANGAN WESMK (SESUAI BLUEPRINT V3)

> **Rencana terkini (23 November 2026):** gunakan
> [`master-plan-2026-11-23.md`](../master-plan-2026-11-23.md). Dokumen ini dipertahankan
> sebagai rencana awal (basis Blueprint V3) dan status di bawah sudah tidak akurat.

Basis: `rencana sementara.pdf` (WESMK MEGA SUPER BLUEPRINT V3) + audit repo saat ini.
Target: Go 1.26+ + templ/HTMX, PostgreSQL 16, Redis 7, 9 role, 38+ tabel, server-authoritative.

## Status saat ini
- UI Go+templ/HTMX sudah berjalan; auth, 8 role, navigasi, state scoping siswa/ortu aktif.
- Data masih satu blob `app_state`; banyak fitur masih demo/client-side.
- Schema 66 tabel sudah ada di migrasi, tetapi yang dipakai aplikasi baru `users` + `app_state` (blob JSON); normalisasi belum.
- CBT timer/lock, upload R2, WA/VA, audit trail, backup/monitoring belum; kas/PIN/OTP sudah server-side tapi belum fail-closed.

## Prinsip eksekusi
- UI freeze: layout/navigasi/alur tidak diubah; hanya token, a11y, status, copy.
- Semua data sensitif (uang, PIN, OTP, HMAC, kunci jawaban) wajib server-side.
- Setiap fase harus lulus: `go test ./...`, `go vet ./...`, smoke test 8 role, console bersih.

## Fase 0 — Fondasi data (minggu 1)
- Buat migrasi 38 tabel (users, students, classes, invoices, ledger, exams, questions, sessions, documents, PKL, BKK, dll).
- Tambah FK, index, constraint, trigger append-only `financial_audit_ledgers`.
- Migrasi state blob → tabel domain (script one-time, idempotent).
- Tambah optimistic lock/versioning untuk mutasi sensitif.
- Acceptance: DB schema lengkap, migrasi lulus di DB bersih, `go test ./...` hijau.

## Fase 1 — RBAC staf + keamanan data (minggu 1-2)
- Scoping baca/tulis per role staf (guru, wali kelas, BK, bendahara, admin TU, piket, kepsek).
- Hapus data sensitif dari localStorage & client state.
- Hapus `activeHeadmasterOtp`, PIN/OTP dari response client.
- Tambah audit log untuk mutasi staf.
- Acceptance: `GET /api/state` per role hanya data miliknya; test akses silang lulus.

## Fase 2 — CBT server-authoritative (minggu 2-4)
- Soal tanpa kunci jawaban di payload; timer di Redis; auto-hydrate <1s.
- Lock permanen 1,5 detik; QR unlock siswa; mass unlock satu ruangan.
- SSE proctoring: status siswa, warning, lock.
- Layar `Bekal Ilmu` pasca-ujian.
- Acak variabel soal akuntansi/IP per siswa.
- Honeypot AI trap.
- Acceptance: 1.000 siswa serentak, zero lag, timer tetap jalan saat refresh/ganti device.

## Fase 3 — Keuangan & Gatekeeper SPP (minggu 3-5)
- Ledger append-only + OTP reversal kepsek.
- PIN 6 digit diverifikasi server.
- Kas tunai server-side + kwitansi HMAC server.
- Dispensasi 7 hari + surat perjanjian upload.
- Tiket darurat 1 sesi (2 jam) pengawas.
- Gatekeeper ujian: blokir CBT jika SPP tunggak; dispensasi/tiket darurat.
- VA webhook HMAC + active pull + rekonsiliasi.
- Bot WA H-3/H-0/H+3/H+7.
- Export PDF/Excel native Go.
- Acceptance: dua pembayaran bersamaan tidak race; reversal butuh OTP; verify server-side.

## Fase 4 — PKL, BKK, tracer study (minggu 5-7)
- PKL geotagging + EXIF + geofence radius.
- PIN supervisor DUDI untuk ACC jurnal.
- Jurnal mingguan, bulk approve.
- BKK: lamaran, lowongan, tracer study BMW (Bekerja/Melanjutkan/Wirausaha).
- Konsultasi karir BMW orang tua-BK.
- Acceptance: check-in di luar radius ditolak; EXIF palsu ditolak.

## Fase 5 — AI & pedagogi (minggu 6-8)
- Bank soal AI daily quiz 30-50 variasi per topik; timer 45s forward-only.
- Radar keluh kesah (`Paham`/`Bingung`/`Nggak Ngerti`).
- Essay grading AI dengan XML sandbox + strict JSON.
- AI draft narasi wali kelas.
- Kalibrasi kesulitan soal lintas kelas.
- Voice agenda 10 detik.
- Acceptance: hemat token, anti-contekan grup WA, output JSON valid.

## Fase 6 — Upload & dokumen (minggu 7-8)
- R2 presigned URL; validasi magic bytes server; kompresi <500KB; UUID rename.
- Verifikasi publik HMAC server (`/verify/:hash`).
- Upload berkas TU, surat dispensasi, dokumen siswa.
- Acceptance: file >500KB ditolak; magic bytes mismatch ditolak; verify server-side.

## Fase 7 — Admin/Headmaster/Dapodik & reporting (minggu 8-10)
- Admin TU: tambah siswa, Dapodik sync, surat, PKL/BKK.
- Headmaster: OTP, esign, AI advisor, dashboard KPI.
- Guru: question bank, exam control, homeroom/PKL, attitude.
- Kesiswaan: pelanggaran, transformasi index, peer report, counseling.
- Piket: gate log, inval.
- Export CSV/PDF/Excel.
- Acceptance: semua modul 8 role berfungsi end-to-end.

## Fase 8 — Operasional & kualitas (minggu 10-12)
- Backup/restore diuji; monitoring + alert; health check.
- CI: build + test + lint + migration check.
- E2E smoke 8 role, deep link, refresh, mobile.
- Performance: aset UI di-embed (HTMX + Alpine + `app.js`), load test CBT.
- Error tracking (Sentry) + analytics.
- Acceptance: checklist production-readiness.md semua hijau.

## Urutan eksekusi yang disarankan
1. Fase 0-1 (data + keamanan) — blocker utama.
2. Fase 2-3 (CBT + keuangan) — nilai inti sekolah.
3. Fase 4-6 (PKL/BKK + AI + upload).
4. Fase 7-8 (admin/ops).
