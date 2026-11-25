# Master Plan WESMK — Semua Peran Berfungsi Penuh

**Tanggal:** 23 November 2026
**Basis kode:** `c311c2a` (master)
**Basis audit:** [`production-readiness.md`](production-readiness.md), [`arsip/requirements-traceability.md`](arsip/requirements-traceability.md)

> Dokumen ini ditulis 23 November 2026 dan sengaja dibiarkan apa adanya sebagai rencana. Rujukan
> ke "demo" di dalamnya adalah kondisi saat itu — mode demo **sudah dihapus total** pada
> 25 November 2026. Status terkini: [`PETA-PROYEK.md`](PETA-PROYEK.md).
**Tujuan:** mengubah aplikasi dari **demo dengan satu blob JSON** menjadi **ERP sekolah yang benar-benar
berjalan**: setiap alur punya sumber data nyata, otorisasi server per peran, audit, integrasi, dan
bukti uji — tanpa merombak UI/UX yang sudah ada.

---

## 0. Definisi "Berfungsi"

Satu alur dinyatakan **SELESAI** hanya jika memenuhi 7 syarat:

1. **Data nyata** — dibaca/ditulis dari tabel domain (bukan blob `app_state`), dengan constraint & FK.
2. **Otorisasi server** — peran, kepemilikan, dan status diverifikasi di server; UI hanya lapisan tampilan.
3. **Audit** — perubahan penting masuk `state_audit_log`/domain audit dengan aktor, waktu, dan alasan.
4. **Idempoten & aman** — retry tidak menggandakan transaksi; aksi destruktif punya konfirmasi + batas.
5. **Error & empty state** — kegagalan tidak pernah dilaporkan sebagai sukses.
6. **Uji** — unit + integrasi (Postgres/Redis) + smoke E2E per peran.
7. **Tanpa data demo di produksi** — seed demo mati, tidak ada PIN/OTP/tautan/label demo yang bocor.

Definisi ini dipakai sebagai *Definition of Done* di setiap fase.

---

## 1. Kondisi Awal (fakta terverifikasi)

| Aspek | Kondisi |
|---|---|
| Stack | Go 1.26, templ, HTMX, Alpine, Tailwind, pgx, Redis. `go vet` + `go test` hijau. |
| Data | 68 tabel dibuat di migrasi; aplikasi hanya memakai `users`, `app_state`, `document_blobs`, `state_audit_log`. Sisanya skema mati. |
| Sumber kebenaran | Satu dokumen `app_state` (JSON) + shadow sync ke `students`, `tuition_invoices`, `financial_audit_ledgers`. |
| Auth | Argon2id, sesi Redis single-device, rate limit login, CSRF token, RBAC baca staf per peran. |
| Nilai & uang | Kuis/CBT, kas tunai, PIN bendahara, dispensa, reversal, surat HMAC, callback gateway — di server. |
| Alur per peran | Semua 9 peran punya portal & route; mayoritas aksi menulis ke blob dan sudah berjalan untuk satu pengguna demo. |
| Yang belum ada | SSE proctoring, timer ujian di Redis, worker/queue, object storage (R2), gateway VA nyata, WA nyata, ekspor audit ke SIEM, backup/monitoring, E2E di CI, normalisasi data. |
| Keamanan terbuka | 2 P0 (bayar web di produksi, PIN DUDI publik), 4 P1 (OTP reuse, `/verify` bocor, rate-limit bypass, lost update). |
| Kualitas UI | CSP mematikan Alpine & `onsubmit` confirm; beberapa handler menelan error. |

**Kesimpulan:** fondasi UI, auth, dan sebagian logika domain sudah layak. Pekerjaan utama bukan
menulis ulang, melainkan **mengganti sumber data, menutup celah otorisasi, dan menyambungkan
integrasi**.

---

## 2. Arsitektur Target

### 2.1 Lapisan

```
cmd/wesmk              entrypoint, wiring, worker startup
internal/httpapi       router, middleware, handler JSON + web, SSE
internal/ui            templ, komponen, viewmodel per peran
internal/domain/*      logika bisnis per agregat (tanpa SQL/HTTP)
internal/store         implementasi Postgres per agregat + Redis + object storage
internal/jobs          worker Redis (WA, reminder, rekonsiliasi, export, backup)
internal/platform      config, observability, clock, id, feature flags
internal/db/migrations migrasi SQL versioned
```

Aturan:
- Handler **tidak** menyentuh JSON blob; handler memanggil service domain.
- Service domain menerima interface repository; transaksi dimiliki store.
- Setiap endpoint mutasi punya policy check terpusat (`authz.Can(role, action, resource)`).

### 2.2 Sumber kebenaran data

Urutan cutover per domain (bukan big-bang):

1. **Shadow write** (sudah untuk 3 domain): tulis blob + tabel.
2. **Backfill + rekonsiliasi** dari blob ke tabel; bandingkan jumlah & checksum.
3. **Dual read**: baca tabel, fallback blob, log perbedaan.
4. **Cutover**: endpoint baca hanya tabel; tulis hanya tabel.
5. **Bekukan** key blob domain itu (tolak/abaikan); setelah semua domain pindah, hapus `app_state`.

### 2.3 Matriks kepemilikan tulis (wajib dijaga policy)

| Domain | Siswa | Ortu | Guru | BK | Piket | Bendahara | Admin TU | Kepsek | Developer |
|---|---|---|---|---|---|---|---|---|---|
| Profil siswa | ubah terbatas (BMW) | – | – | – | – | – | CRUD | – | – |
| Tagihan & ledger | – | – | – | – | – | CRUD (PIN/OTP) | baca | baca | – |
| Nilai & ujian | submit | – | bank soal, nilai esai | – | – | – | – | – | – |
| Presensi | – | izin/sakit | toggle | – | – | – | – | – | – |
| Pelanggaran | lapor anonim | – | – | CRUD | catat gerbang | – | – | – | – |
| PKL | check-in, jurnal | – | ACC jurnal | – | – | – | CRUD mitra | – | – |
| BKK | lamar | – | – | – | – | – | CRUD lowongan | – | – |
| Surat resmi | – | – | – | terbitkan panggilan | – | – | terbitkan | esign | – |
| OTP reversal | – | – | – | – | – | minta | – | terbitkan/approve | – |
| Bug report | buat | buat | buat | buat | buat | buat | buat | buat | ubah status |
| Pengaturan sekolah | – | – | – | – | – | – | terbatas | penuh | – |

Matriks ini menjadi dasar test akses silang (IDOR) di fase G.

---

## 3. Roadmap Fase

Estimasi untuk **2 developer + AI assist**, 1 sprint = 1 minggu. Bisa diparalelkan sebagian.

```
Fase A  P0 & keamanan dasar             1 minggu   ── gate: produksi tidak bisa dicurangi
Fase B  Platform & data                 3 minggu   ── gate: domain layer + authz + worker + storage
Fase C  Keuangan & gatekeeper           3 minggu   ── gate: uang nyata, rekonsiliasi, audit
Fase D  CBT & proctoring                3 minggu   ── gate: 1.000 siswa, timer server, SSE
Fase E  Akademik & kelas                2 minggu   ── gate: presensi, jurnal, rapor dari data nyata
Fase F  PKL / BKK / karier              2 minggu   ── gate: geofence nyata, DUDI aman
Fase G  Peran lain & admin              3 minggu   ── gate: semua peran end-to-end + IDOR test
Fase H  Integrasi & otomasi             2 minggu   ── gate: WA, VA, storage, reminder, backup
Fase I  Kualitas & go-live               2 minggu   ── gate: checklist produksi hijau
```

**MVP pilot terkontrol (satu sekolah, data nyata terbatas):** A + B + C + D + E parsial ≈ **9–10 minggu**.
**Semua peran penuh:** **17–21 minggu**. Angka ini estimasi; dievaluasi ulang di akhir setiap fase.

---

### Fase A — Tutup P0 & keamanan dasar (1 minggu) — **SELESAI 23 Nov 2026**

**Tujuan:** produksi tidak bisa dirugikan lewat celah yang sudah ditemukan.

Pekerjaan:
1. Guard demo pada `POST /student/invoice/pay` dan `POST /parent/invoice/pay`; test regresi.
2. Hapus hint PIN DUDI dari template; hapus PIN dari pesan error; hash PIN mitra; rate limit
   approval; wajibkan token per-jurnal atau sesi supervisor.
3. OTP reversal: one-time, hash, TTL, batas percobaan, tidak disimpan di ledger, tidak lewat URL.
4. Minimalkan `/verify`: tampilkan detail hanya bila kode cocok; hapus daftar global surat/kwitansi/jurnal.
5. `PUT /api/state`: merge di dalam `UpdateStateAudited` (row lock); larang overwrite penuh kecuali
   endpoint admin khusus.
6. `TrustProxy=false` default; hanya aktif lewat env di belakang proxy tepercaya.
7. Perbaiki CSP: konfirmasi destruktif lewat `app.js`, Alpine diberi nonce/`unsafe-eval` atau diganti.
8. Perbaiki handler yang menelan error (`_ = mutateStateDoc`) → error ke flash + status code.
9. Magic link: halaman konfirmasi POST sebelum sesi dibuat.
10. Tambah test keamanan: akses silang per peran, replay pembayaran, OTP reuse, spoof XFF.

**Acceptance:** semua P0/P1 di audit tertutup; `go vet`/`go test` hijau; uji manual membuktikan
siswa/ortu tidak bisa melunasi tagihan di mode produksi.

**Status pelaksanaan:** 10/10 item selesai. P0-1/P0-2 dan P1-1..P1-4 ditutup dengan test
regresi; CSP + konfirmasi destruktif diperbaiki; handler web tidak lagi menelan error mutasi;
magic link memakai POST; `/api/health` diminimalkan; handler upload membersihkan blob yatim dan
submit ujian kosong ditolak. Seluruh P2 audit juga ditutup. Bukti: `go vet ./...`, `go build ./...`,
`go test -count=1 ./...` hijau.

Catatan residual: OTP reversal belum di-hash karena kode harus dapat dibaca Kepala Sekolah di
portal; menyimpan hash saja membutuhkan alur reveal satu kali yang mengubah UI (di luar freeze).
Yang sudah ada: TTL 5 menit, sekali pakai, batas 10 percobaan/10 menit per bendahara, tidak masuk
ledger/URL. Fase B–I tetap pekerjaan multi-minggu berikutnya dan membutuhkan keputusan provider
(VA, WA, storage) serta migrasi skema per agregat.

---

### Fase B — Platform & data (3 minggu) — **B.1 DIMULAI 23 Nov 2026**

**Tujuan:** fondasi agar semua domain bisa pindah dari blob tanpa mengulang pekerjaan.

B.1 Struktur domain
- Paket `internal/domain/{identity,academic,finance,learning,pkl,discipline,letters,notify}`.
- Interface repository per agregat; implementasi `store` per file.
- `authz` terpusat: `Can(role, action, resource, ownership)` + test tabel kebenaran.

B.2 Lifecycle akun
- Undang user, reset password, ubah peran, nonaktifkan akun, cabut semua sesi.
- MFA TOTP wajib untuk bendahara, kepala sekolah, admin TU, developer, dengan recovery code.
- Halaman "sesi aktif" + audit login (selesai).
- Akun dibuat bersamaan saat Admin TU menambah siswa/guru.

B.3 Pengaturan sekolah
- Profil sekolah, tahun ajaran & semester aktif, jurusan, kelas, jadwal, tarif SPP, template surat.
- Rollover tahun ajaran (naik kelas/lulus) sebagai operasi eksplisit + audit.

**Status B.3 (23 Nov 2026, sebagian):** Admin TU sudah bisa menambah tahun ajaran, mengaktifkan
tepat satu semester, membuat kelas per jurusan/wali kelas, dan menyusun jadwal per kelas. Tahun
ajaran aktif tampil di sidebar, halaman login, dan portal. Onboarding siswa memakai kelas master
dan menyimpan `students.classroom_id`. Jadwal hari ini muncul di portal guru dan papan piket;
radar piket, penugasan inval, log gerbang, izin orang tua, presensi per tanggal, dan jurnal per slot
jadwal sudah memakai tabel domain. Halaman Admin TU dipecah menjadi Siswa, Kelas & Jadwal, Akun,
dan Dokumen; kelas dan jadwal bisa diubah/dihapus dengan proteksi relasi. Profil sekolah, tarif SPP
per tahun, template surat, dan impor CSV belum ada; rollover naik kelas/lulus sudah menjadi operasi
eksplisit dengan pemetaan kelas, kelulusan tingkat 12, transaksi tunggal, dan audit. Tugas masih
disimpan di `app_state`.

B.4 Platform
- Worker Redis: job queue dengan retry, dead-letter, jadwal (reminder, rekonsiliasi, export, backup).
- Object storage R2/S3: presigned upload, validasi magic bytes, retention, ganti `document_blobs`.
- Observability: request ID, structured log, `/metrics` (Prometheus), error tracking.
- Feature flag per domain untuk cutover bertahap.

B.5 Migrasi data
- Backfill lengkap per agregat + laporan rekonsiliasi (jumlah, total, checksum).
- Dual-read dengan log diff; hapus fallback setelah 2 minggu nol diff.

**Acceptance:** domain layer dipakai minimal satu alur (contoh: profil siswa); authz punya test
matrix; worker & storage berjalan di staging; migrasi bisa diulang idempoten.

**Status B.1 (23 Nov 2026):** `internal/authz` menjadi satu-satunya policy matrix
(`Can(role, action, resource, ownership)`) dengan test tabel peran. Empat agregat sudah dibaca dari
tabel domain dengan fallback blob:

- **Profil siswa** (`internal/domain/identity`): migrasi `007`, `app_id` + kolom profil, backfill
  sekali untuk baris lama, shadow sync; dashboard siswa/ortu memakai `authz` sebelum menerima data.
- **Tagihan & ledger** (`internal/domain/finance`): migrasi `008` menambah `app_id` invoice dan
  `financial_ledger_details` (karena ledger append-only tidak boleh di-UPDATE). Repository membaca
  invoice per siswa/semua dan ledger per siswa/semua; dashboard siswa, portal ortu, dan laporan
  bendahara memakai hasil tabel. Total tagihan/terkumpul dihitung ulang dari data domain.
- **Presensi** (`internal/domain/attendance`): migrasi `009` membuat tabel `attendances` dengan
  `app_id`; backfill/shadow sync mengisi status, sumber verifikasi, dan waktu. Dashboard siswa
  memakai satu rekaman terbaru; daftar presensi guru memakai seluruh tabel. Guard jumlah mencegah
  data yang belum terpetakan menutupi baris blob.
- **Bank soal & sesi ujian** (`internal/domain/exam`): migrasi `010` menambah `app_id` pada
  `exams`/`exam_questions`/`exam_sessions`, metadata soal, dan kolom progres proctoring. Kiosk
  siswa menerima soal tanpa `correctKey`/`essayRubric`; bank soal guru tetap lengkap. Grid
  pengawas membaca sesi dari tabel, termasuk status, progres, warning, lock, dan tiket darurat.
- **PKL & BKK** (`internal/domain/pkl`, `internal/domain/bkk`): migrasi `011` menambah `app_id`
  serta field portal (mitra, geofence, supervisor, check-in, jurnal, deadline/requirements
  lowongan, status lamaran). Backfill/shadow sync mengisi placements, log presensi PKL, jurnal,
  lowongan BKK, dan lamaran. Portal siswa/ortu membaca placement, log, jurnal, lowongan, dan
  lamaran milik siswa; guru membaca semua log/jurnal; admin TU membaca semua placement, lowongan,
  dan lamaran. PIN supervisor selalu dihapus di service sebelum data mencapai portal. Guard
  jumlah mencegah slice domain yang belum lengkap menutupi blob.

**Status B.2 (23 Nov 2026, sebagian):** Admin TU kini punya lifecycle akun dasar: daftar akun,
pembuatan akun staf, reset kata sandi (Argon2id), aktif/nonaktif, dan cabut sesi. Akun nonaktif
ditolak saat login dan sesinya dihapus. Setiap tindakan masuk `state_audit_log`. MFA TOTP untuk
Admin TU, Bendahara, Kepala Sekolah, dan Developer sudah tersedia: enrollment lewat halaman
Keamanan Akun, verifikasi dua langkah saat login (web + API), rate limit percobaan, reset MFA
oleh Admin TU, dan audit saat MFA diaktifkan/dinonaktifkan/direset. Per 23 Nov 2026 MFA wajib
untuk keempat peran tersebut: sesi hanya boleh membuka `/security` sampai enrollment selesai.
Secret dienkripsi AES-256-GCM dengan `WESMK_MFA_KEY`, recovery code disimpan sebagai hash dan
sekali pakai, serta `/security` menampilkan sesi aktif, riwayat login akun, dan (untuk Admin TU/
Developer) audit login seluruh akun. Admin TU yang menambah siswa kini juga membuat profil
`students`, akun login siswa (NISN + kata sandi awal), dan akun wali (magic link WhatsApp) dalam
satu transaksi; NISN/WA duplikat ditolak. Guru juga sudah bisa menilai tugas siswa dari portal, dan
e-sign kepala sekolah menyimpan hash + HMAC di `executive_signatures` (migrasi `013`).
Tugas siswa kini diunggah sebagai berkas nyata (PDF/JPG/PNG, maks 10 MB) ke `document_blobs`,
bukan nama berkas palsu; guru dapat membuka berkas dan menyimpan nilai yang langsung tampil di
portal siswa. Guru juga dapat membuat tugas per kelas dengan tenggat; server menolak tugas masa
lalu, pengumpulan lintas kelas, pengumpulan setelah tenggat, dan pengumpulan ganda untuk satu
tugas. Design system global juga dirapikan (palet netral–teal, sidebar, topbar, tabel,
form, dan fokus aksesibel) tanpa mengubah informasi arsitektur navigasi.

Test unit + integrasi Postgres (`WESMK_TEST_DATABASE_URL`) lulus, termasuk migrasi ganda, baris
pra-007/pra-008, restore metadata ledger, trigger append-only, duplikasi presensi saat
re-backfill, bank soal/sesi ujian (termasuk opsi, variabel dinamis, dan redaksi kunci jawaban di
service siswa), serta PKL/BKK (placement, log geofence, jurnal, lowongan, lamaran, redaksi PIN,
idempoten saat sync ulang). Sisa B.2–B.5 (pengaturan sekolah, worker/storage, backfill domain
lain) belum dikerjakan, dan jalur tulis masih blob + shadow sync.

---

### Fase C — Keuangan & gatekeeper (3 minggu)

**Tujuan:** uang, tagihan, dan pemblokiran ujian berjalan nyata dan dapat direkonsiliasi.

Pekerjaan:
1. Tabel jadi sumber kebenaran: `tuition_invoices`, `financial_audit_ledgers`, `payment_intents`,
   `scholarships`, `dispensations`, `reversal_requests`.
2. Integrasi gateway VA (pilih satu provider), callback HMAC, active pull status, rekonsiliasi harian,
   penanganan expired/refund/ganda.
3. Kas tunai: PIN (hash + rate limit), kuitansi bernomor, PDF/QR, pengurangan parsial.
4. Reversal: request bendahara → OTP kepsek one-time → entri ledger baru (append-only), tidak pernah
   UPDATE/DELETE.
5. Dispensasi 7 hari + tiket darurat 2 jam: masa berlaku, audit, dipakai gatekeeper.
6. Gatekeeper CBT benar-benar dicek di server saat start exam (bukan hanya di UI).
7. Beasiswa/KIP: aturan diskon, perhitungan ulang tagihan, riwayat.
8. Reminder WA H-3/H-0/H+3/H+7 lewat worker (bukan sekadar timestamp).
9. Halaman laporan: arus kas, tunggakan, rekonsiliasi, export CSV/PDF.
10. Test: dua pembayaran bersamaan, callback ganda, OTP salah/reuse, nominal negatif, invoice orang lain.

**Acceptance:** tidak ada jalur pelunasan tanpa uang nyata di production; total ledger = total
pembayaran gateway + kas; rekonsiliasi otomatis dan bisa diaudit.

---

### Fase D — CBT & proctoring (3 minggu)

**Tujuan:** ujian berjalan untuk banyak siswa serentak, anti-curang, dan hasilnya bisa dipertanggungjawabkan.

Pekerjaan:
1. Tabel: `exams`, `exam_questions`, `exam_sessions`, `exam_answers`, `exam_violations`,
   `exam_resets`. Kunci jawaban tidak pernah masuk payload siswa.
2. Timer di Redis per sesi; mulai hanya lewat endpoint; refresh/ganti perangkat tidak menambah waktu.
3. Autosave jawaban (debounce) + submit idempoten; deadline server menolak submit terlambat
   kecuali dispensasi.
4. Lock dikendalikan server (blur event → WARN/LOCKED), unlock hanya via token guru/QR, mass unlock
   per ruangan, tiket darurat, reset dengan alasan + audit (single & bulk).
5. SSE proctoring: grid status real-time, warning, lock, progress; fallback polling.
6. Bank soal: tipe MCQ/kompleks/esai, variabel dinamis, rubrik, import/export, versioning.
7. Penilaian esai: antrean guru, rubrik server, AI draft (opsional) yang tidak pernah final otomatis.
8. Hasil: skor MCQ server, status esai, halaman refleksi "Bekal Ilmu", rapor.
9. Uji beban: 1.000 sesi serentak, latensi submit, reconnect SSE.

**Acceptance:** siswa tidak bisa membaca kunci jawaban, menambah waktu, atau unlock sendiri;
pengawas melihat status <2 detik; 1.000 siswa tanpa error 5xx.

---

### Fase E — Akademik & kelas (2 minggu)

Pekerjaan:
1. Master akademik: kelas, jurusan, jadwal, wali kelas, guru mapel.
2. Presensi: sumber RFID/gerbang, manual guru, izin orang tua; satu record per siswa/hari; audit.
3. Jurnal kelas + catatan guru; agenda suara (opsional) disimpan ke storage.
4. Rapor Holistik 360° dihitung dari data nyata (CBT, kuis, sikap, kehadiran) + versi/penyimpanan.
5. Badge "Si Paling..." berbasis aturan, bukan teks statis; Rewind memakai data semester nyata.
6. Rekap wali kelas: sikap (MAD), catatan, approval PKL, export.
7. Kuis harian: bank soal, attempt unik, timer forward-only, streak.
8. Test perhitungan nilai, presensi ganda, izin setelah jam pelajaran.

**Acceptance:** nilai rapor dapat direproduksi dari data sumber; tidak ada nilai yang hanya hidup di UI.

---

### Fase F — PKL, BKK, karier (2 minggu)

Pekerjaan:
1. Mitra DUDI: alamat, geofence (lat/lng/radius), supervisor, PIN hash, kuota, masa PKL.
2. Check-in: koordinat dikirim siswa, server validasi jarak (Haversine), deteksi mock location,
   validasi EXIF/foto, tolak di luar radius; jangan mengarang koordinat.
3. Jurnal harian/mingguan, foto, approval DUDI via token aman, ACC guru, rekap.
4. BKK: lowongan, lamaran, status seleksi, tracer study BMW.
5. Konsultasi karier orang tua–BK: jadwal, status, catatan.
6. Statistik penempatan & serapan lulusan.
7. Test: check-in luar radius, PIN salah, jurnal milik siswa lain, approval ganda.

**Acceptance:** tidak ada check-in palsu yang lolos; DUDI hanya bisa menyetujui jurnal mitranya.

---

### Fase G — Peran lain & admin (3 minggu)

G.1 Kesiswaan/BK
- Pelanggaran dari gerbang, guru, manual; poin & pasal; surat panggilan PDF+QR.
- Peer report escrow: ambang laporan, anonimisasi, tindak lanjut, audit akses.
- Sesi konseling, indeks transformasi, catatan privat (hanya BK).
- Scoping: guru tidak boleh membaca catatan konseling.

G.2 Piket
- Radar kelas kosong dari jadwal nyata, penugasan guru inval + poin dampak, log gerbang.
- Notifikasi ke guru inval (in-app/WA).

G.3 Admin TU
- CRUD master data (siswa, guru, kelas, jurusan, jadwal), impor CSV, validasi duplikat NISN/NIP.
- Verifikasi dokumen dengan preview aman + alasan penolakan.
- Surat resmi: nomor otomatis, template, PDF, QR verifikasi.
- Dapodik: export format resmi + impor dengan laporan error, bukan sekadar log.
- Undangan akun siswa/guru/orang tua (tautan sekali pakai).

G.4 Kepala Sekolah
- KPI dari agregasi tabel (kehadiran, kas, inval, serapan BMW), filter periode.
- Approval OTP/esign: dokumen PDF ditandatangani + hash, bukan no-op.
- AI advisor dengan guardrail (data agregat, tanpa PII), audit prompt.

G.5 Developer
- Daftar bug, status, prioritas, catatan; audit perubahan status.
- Halaman audit log (filter aktor/aksi/waktu) untuk admin & kepsek.

**Acceptance:** setiap peran lulus skenario E2E-nya; test IDOR lintas peran hijau; tidak ada
endpoint yang mengembalikan data di luar matriks kepemilikan.

---

### Fase H — Integrasi & otomasi (2 minggu)

1. WhatsApp: provider (Fonnte/WAHA), handshake, template, antrean, status kirim, opt-out.
2. Payment gateway: produksi/sandbox, rekonsiliasi harian, alert selisih.
3. Object storage produksi + lifecycle (retensi dokumen, hapus sesuai kebijakan).
4. Email transaksional untuk undangan/reset (bila dipakai).
5. Notifikasi in-app per peran: tagihan, ujian, jurnal, pelanggaran, approval.
6. Job terjadwal: reminder, rekonsiliasi, export malam, backup, pembersihan blob yatim.
7. Kuota & biaya AI: rate limit, cache, fallback tanpa AI.

**Acceptance:** tidak ada lagi data yang "hilang" karena hanya dicatat sebagai timestamp; setiap
job punya status, retry, dan alert kegagalan.

---

### Fase I — Kualitas & go-live (2 minggu)

1. Unit test domain ≥70% untuk logika uang/nilai/ujian; integration test Postgres+Redis di CI.
2. E2E Playwright: login tiap peran, alur utama, akses silang, upload, bayar, CBT, PKL.
3. Load test k6: 1.000 CBT, 200 pembayaran/menit, 5.000 request health.
4. Uji keamanan: `govulncheck`, IDOR, XSS, CSRF, rate limit, header, dependency audit.
5. Backup/restore drill: RPO/RTO terukur, runbook.
6. Monitoring & alert: error rate, latency p95, DB pool, Redis, disk, job gagal.
7. A11y & mobile: kontras, fokus, target sentuh 48px, tabel responsif.
8. Dokumentasi operator: deploy, rollback, rotasi secret, insiden, prosedur tahun ajaran.
9. Pilot satu sekolah dengan data terbatas + pendampingan; ukur keberhasilan dan perbaiki.

**Acceptance:** checklist produksi hijau, tidak ada P0/P1 terbuka, restore terbukti, pilot berjalan
tanpa intervensi manual.

---

## 4. Status per Peran & Target

| Peran | Alur utama | Sekarang | Target setelah roadmap |
|---|---|---|---|
| **Siswa** | dashboard, tagihan, kuis, CBT, dokumen, PKL, BKK, rapor, rewind | UI + sebagian server; check-in PKL mengarang koordinat, ukuran tugas hardcode, rewind statis | semua dari tabel; gatekeeper, timer, geofence, rapor nyata |
| **Orang tua** | dashboard, roadmap biaya, konsultasi, izin, tagihan | magic link, izin & konsultasi jalan; bayar web tidak aman; AI empathy statis | magic link aman, tagihan hanya via gateway, notifikasi |
| **Guru** | jadwal, presensi, jurnal, bank soal, proctoring, esai, wali kelas | jalan di blob; proctoring tanpa SSE; timer klien | SSE, timer server, penilaian tervalidasi, rapor |
| **BK/Kesiswaan** | pelanggaran, peer report, konseling, surat panggilan | tulis blob; surat HMAC; catatan privat belum benar-benar terbatas | tabel + scoping ketat + PDF + audit akses |
| **Piket** | radar kelas, inval, log gerbang | simulasi kelas statis (`cls-1`) | dari jadwal nyata + notifikasi |
| **Bendahara** | kas, PIN, tagihan, beasiswa, dispensa, reversal | kas/PIN/ledger server; reminder WA belum kirim; reversal OTP lemah | gateway, rekonsiliasi, OTP one-time, PDF |
| **Admin TU** | master data, dokumen, surat, Dapodik, PKL/BKK | CRUD terbatas; tambah siswa tanpa akun; Dapodik hanya log | CRUD penuh, impor/ekspor, undangan akun |
| **Kepala Sekolah** | KPI, approval, esign, AI | KPI dari blob; esign no-op; OTP ditampilkan | KPI agregat, esign PDF, approval aman |
| **Developer** | bug report, audit | status bug jalan | audit viewer, observability |
| **Publik** | verify, approval DUDI | bocor daftar & PIN (P0) | minimal, aman, rate limited |

---

## 5. Kebutuhan Non-Fungsional

| Kebutuhan | Target |
|---|---|
| Keamanan | tidak ada P0/P1; 100% endpoint mutasi ber-policy; secret fail-closed; OTP/PIN hash; sesi bisa dicabut |
| Privasi | minimisasi data di `/verify`; PII tidak dikirim ke AI; retensi dokumen; audit akses; kepatuhan UU PDP |
| Performa | halaman portal p95 < 500 ms; submit ujian p95 < 300 ms; 1.000 sesi CBT serentak |
| Ketersediaan | 99,5% jam sekolah; degradasi terkontrol bila Redis/WA/gateway down |
| Data | backup harian + WAL; RPO ≤ 15 menit, RTO ≤ 2 jam (target pilot) |
| Observability | request ID, log terstruktur, metrik, alert error/latency/job/DB |
| Aksesibilitas | WCAG AA dasar, keyboard, kontras, target sentuh 48px |
| Maintainability | domain test, migrasi idempoten, feature flag, dokumentasi operator |

---

## 6. Strategi Migrasi Blob → Tabel

1. **Bekukan bentuk data**: definisikan skema tabel per agregat + mapping field JSON.
2. **Backfill idempoten** dari `app_state`; simpan laporan jumlah/checksum per tabel.
3. **Shadow write** untuk semua mutasi domain (bukan hanya 3 tabel saat ini), dalam transaksi yang sama.
4. **Dual read** dengan comparator; log diff ke tabel `migration_diffs`; target nol diff 7 hari.
5. **Cutover per agregat** di belakang feature flag; rollback = matikan flag (blob masih ditulis).
6. Setelah semua agregat pindah: hentikan tulis blob, arsipkan `app_state` (read-only), lalu hapus.
7. Uji migrasi di salinan produksi sebelum eksekusi; sediakan skrip verifikasi pasca-cutover.

Aturan: **satu agregat per PR**, tidak ada migrasi besar sekaligus; setiap migrasi harus bisa
dijalankan pada DB kosong dan DB berisi data.

---

## 7. Strategi Uji

| Lapis | Isi | Gate |
|---|---|---|
| Unit | kalkulasi uang, nilai, MAD, Haversine, policy, state machine ujian | setiap PR |
| Integrasi | Postgres+Redis nyata (container), transaksi, row lock, migrasi | setiap PR domain |
| Kontrak HTTP | semua route, status code, auth, CSRF, IDOR | setiap PR handler |
| E2E | Playwright per peran + alur lintas peran | nightly & sebelum rilis |
| Beban | k6 CBT/pembayaran/health | akhir Fase D & I |
| Keamanan | govulncheck, dependency audit, header, rate limit, fuzz input | setiap rilis |
| Chaos ringan | Redis/WA/gateway down | akhir Fase H |

CI wajib: `go vet`, `go test`, `templ generate` tanpa diff, migration check, build image.
PR tidak boleh merge bila salah satu gate merah.

---

## 8. Organisasi Kerja

- **Ritme:** sprint 1 minggu, demo di akhir sprint dengan skenario per peran, retrospektif singkat.
- **Branch:** trunk-based, feature flag untuk cutover; tidak ada branch panjang.
- **Peran tim:** 1 backend/domain, 1 full-stack/UI+test, QA paruh waktu saat fase D–I.
- **Dokumentasi:** setiap fase memperbarui `PETA-PROYEK.md` dan `production-readiness.md`,
  dan `docs/pelajaran/<area>.md` untuk bug nyata.
- **Aturan repo:** tetap patuh `AGENTS.md` (snapshot sebelum perubahan besar, commit pathspec,
  dilarang perintah destruktif, verifikasi `go vet` + `go test`).

---

## 9. Risiko & Mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Normalisasi melebar tak terkendali | jadwal molor | satu agregat per PR, feature flag, batas fase |
| Gateway/WA terlambat disediakan | Fase C/H tertahan | adapter + fake provider untuk staging, kontrak jelas |
| Data lama kotor saat backfill | cutover gagal | comparator, laporan diff, uji di salinan produksi |
| Regresi UI saat CSP/JS diubah | pengawas gagal reset | E2E proctoring, feature flag, uji manual |
| Beban CBT melebihi kapasitas | ujian gagal | load test awal, rate limit, pooling, fallback polling |
| Insiden keamanan baru | kehilangan data/uang | audit tiap fase, penetration test sebelum go-live |
| Scope creep fitur baru | MVP tidak selesai | UI freeze, backlog terpisah, kriteria "berfungsi" |

---

## 10. Prioritas Bila Waktu Terbatas

**Wajib untuk pilot (jangan ditawar):**
A (P0), B (authz + data fondasi), C (keuangan + gatekeeper), D (CBT), E (presensi + rapor dasar),
G.3 (akun & master data). PKL/BKK dan AI bisa menyusul.

**Bisa ditunda:** SSE penuh (polling dulu), AI esai otomatis, PWA/offline, rewind audio,
integrasi Dapodik otomatis (impor/ekspor manual dulu), ekspor audit ke SIEM, dan rotasi kunci
enkripsi MFA tanpa reset massal.

---

## 11. Milestone & Bukti

| Milestone | Fase | Bukti yang harus ada |
|---|---|---|
| M1 — Aman | A | test P0, demo tagihan ditolak di production |
| M2 — Platform | B | authz matrix test, worker & storage jalan, backfill report |
| M3 — Uang nyata | C | rekonsiliasi gateway = ledger, kas + PDF |
| M4 — CBT siap | D | 1.000 sesi k6, SSE <2 detik, kunci jawaban tidak di payload |
| M5 — Akademik | E | rapor reproducible dari tabel |
| M6 — PKL aman | F | geofence & PIN test |
| M7 — Semua peran | G | E2E 9 peran + IDOR test |
| M8 — Integrasi | H | WA/VA/storage produksi, job monitor |
| M9 — Go-live | I | checklist hijau, restore drill, pilot report |

---

## 12. Yang Tidak Boleh Dilakukan

- Menulis ulang UI atau mengganti stack (templ/HTMX/Alpine tetap).
- Membuat sistem paralel (blob + tabel sebagai dua sumber kebenaran permanen).
- Menambah halaman publik baru di luar blueprint.
- Menyimpan PIN/OTP/kunci jawaban/saldo di browser atau blob baru.
- Melakukan migrasi besar sekaligus tanpa feature flag dan rollback.
- Menambah dependency baru tanpa kebutuhan yang jelas (patuh aturan lazy di `AGENTS.md`).

---

## 13. Langkah Pertama (minggu ini)

1. Kerjakan Fase A sampai semua P0/P1 audit tertutup + test.
2. Siapkan skeleton `internal/domain` + `authz` untuk satu agregat percontoh: **profil siswa**.
3. Definisikan mapping tabel untuk `students`, `invoices`, `ledger` (3 domain yang sudah punya shadow).
4. Pilih provider VA & WhatsApp, buat adapter + fake provider untuk staging.
5. Tambah CI: `templ generate` diff check, integration test Postgres/Redis, build image.
6. Jadwalkan demo internal akhir sprint: "produksi tidak bisa dicurangi" + satu alur siswa end-to-end.
