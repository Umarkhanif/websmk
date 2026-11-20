# Audit WESMK — Kondisi Terkini

**Tanggal:** 20 November 2026  
**Commit acuan:** `5ca69a3` (branch `master`)  
**Scope:** backend Go 1.26 + UI `templ`/HTMX/Alpine, PostgreSQL 16, Redis 7, 9 peran (8 role blueprint + `developer`).  
**Metode:** verifikasi langsung dari source + `go vet`/`go test` + simulasi fresh clone. Bukan salinan audit versi React sebelumnya.

---

## 0. TL;DR

- Stack sudah **sesuai blueprint**: Go + templ + HTMX + Alpine. React/Vite sudah tidak ada.
- `go vet ./...` dan `go test ./...` **hijau**; `*_templ.go` **sinkron** dengan `.templ`; `git archive` → `go build` **lulus**.
- Tiga blocker integritas repo sudah diperbaiki di `5ca69a3` (vendor JS masuk git, CI Go 1.26).
- Sisa prioritas: **keamanan fail-closed** (PIN/OTP/secret default masih dipakai di produksi), **dokumen basi**, dan **schema mati** (66 tabel dibuat, hanya 2 dipakai).

---

## 1. Skor

| Area | Skor | Catatan |
|---|---:|---|
| Build & test | **9/10** | vet + test hijau; CI sempat salah versi Go (sudah diperbaiki) |
| Integritas repo | **7/10** | Vendor JS & CI sudah diperbaiki; masih ada binary sampah & dokumen basi |
| Arsitektur | **6/10** | UI server-rendered rapi; state masih satu blob JSON |
| Keamanan source | **5/10** | Auth & scoping siswa/ortu bagus; secret fallback & OTP belum fail-closed |
| Data & domain | **4/10** | 66 tabel ada di migrasi, tetapi hanya `users` + `app_state` yang dipakai |
| Kesiapan produksi | **4/10** | Layak demo/pilot terkontrol; belum untuk data siswa/keuangan nyata |

---

## 2. Bukti Audit

| Pemeriksaan | Hasil |
|---|---|
| `go vet ./...` | Lulus (nol output) |
| `go test ./...` | Lulus (auth, config, httpapi, model, state) |
| `templ generate` (di copy terpisah) | Nol diff — `_templ.go` sinkron |
| Fresh clone (`git archive` → `go build ./...`) | Lulus setelah vendor JS dimasukkan git |
| Ukuran kode | Go 23.301 baris (termasuk `_templ.go`), templ 4.431, `app.js` 972, `app.css` 3.170, `default_state.json` 1.153 |
| Git | 31 commit, working tree bersih, branch `master` |
| Migrasi | 4 file, 66 tabel; yang dipakai kode hanya `users` + `app_state` |

---

## 3. Temuan Keamanan

### P0 — fail-closed di produksi

| # | Temuan | Bukti |
|---|---|---|
| S1 | **Secret fallback masih dipakai.** Kalau env kosong, produksi memakai PIN kasir `123456`, OTP reversal `981240`, receipt secret dev, dan password demo default. `Config.Validate()` hanya memaksa `DATABASE_URL`/`REDIS_URL`/`RECEIPT_SECRET`. | `internal/config/config.go:70,82,104`; `internal/state/mutations.go:449,504`; `internal/httpapi/ui_builder.go:1248` |
| S2 | **OTP reversal tidak aman.** `activeHeadmasterOtp` disimpan di `app_state` (persist), tanpa TTL, ditampilkan di flash message dan kartu kepsek, lalu dipakai `ExecuteReversal`. | `internal/httpapi/ui_handlers.go:1485`; `internal/state/mutations.go:501-507,1064-1075` |
| S3 | **Seed dari browser.** `MergeOwned` menerima snapshot klien saat `app_state` masih kosong; produksi bisa terisi data mock dari klien pertama. | `internal/state/scope.go:124-128` |

### P1 — hardening

| # | Temuan | Bukti |
|---|---|---|
| S4 | **CSRF token hiasan.** Semua form memuat `_csrf`, tetapi `validCSRF()` tidak pernah dipanggil; proteksi hanya Origin check. | `internal/httpapi/ui_builder.go:27-31`; `grep 'validCSRF('` nol pemanggil |
| S5 | **RBAC staf belum ada.** Hanya `student`/`parent` yang di-scope; semua staf membaca snapshot penuh dan menulis field lintas peran. | `internal/state/scope.go:22-24`; `internal/httpapi/state_handlers.go` |
| S6 | **Proctor status masih dari browser.** Siswa mengirim `status` lewat `POST /student/proctor-status`; server menerimanya. | `internal/httpapi/ui_handlers.go:621-646` |
| S7 | **Unggah dokumen metadata saja.** Tidak ada file yang disimpan; ukuran di-hardcode (`280`). | `internal/httpapi/ui_handlers.go:696-710` |
| S8 | **Magic link without channel.** Orang tua belum terintegrasi WhatsApp; tautan `GET` bisa dikonsumsi bot preview. | `internal/httpapi/auth_handlers.go` |

### Yang sudah benar

- Password Argon2id; hash tidak pernah ke browser.
- Sesi Redis, cookie `HttpOnly`, `SameSite=Lax`, TTL 8 jam, single-session.
- Rate limit login 5×/10 menit per IP+identifier; rate limit tulis state 120/menit.
- Origin check untuk method mutasi; CSP, HSTS, COOP, nosniff, frame-ancestors.
- Scoping baca/tulis siswa/ortu sudah benar; kunci jawaban & rubrik dihapus dari respons siswa.
- Nilai kuis/CBT, pembayaran SPP, kas tunai, reversal, surat resmi, dan verify publik already server-side (HMAC/ledger append-only).
- Demo mode default mati di production; `SwitchDemoRole` menolak saat demo off.

---

## 4. Temuan Repo & Dokumen

| # | Temuan | Status |
|---|---|---|
| R1 | Vendor JS (`htmx.min.js`, `alpine.min.js`) ter-ignore `.gitignore` → fresh clone gagal build | **Diperbaiki** `5ca69a3` |
| R2 | CI memakai Go 1.23 padahal `go.mod` butuh 1.26 | **Diperbaiki** `5ca69a3` |
| R3 | `wesmk.exe` (22 MB, PE32+ Windows) di root, untracked | Belum dibersihkan |
| R4 | `AUDIT.md` lama + `audit/*.png` (11 screenshot) milik versi React | Berubah di commit ini |
| R5 | `BLUEPRINT-GAP.md` & `docs/production-readiness.md` menyebut React/`src/App.tsx`/`npm`/bundle 560 KB | Diperbarui |
| R6 | Versi Go tidak konsisten: `AGENTS.md`/`GEMINI.md` 1.23+, `README.md`/`go.mod` 1.26 | Disamakan ke 1.26 |
| R7 | 66 tabel dibuat di migrasi, hanya `users` + `app_state` dipakai → ~64 tabel mati | Belum diputus |
| R8 | `internal/ui/pages/placeholder.templ` (+ `_templ.go`) tidak dipakai handler manapun | Belum dibersihkan |
| R9 | `docker-compose.yml` membuka Postgres/Redis ke `0.0.0.0` | Bahaya di server publik |

---

## 5. Rekomendasi (urutan)

1. **Keamanan fail-closed (P0):** produksi menolak PIN/OTP/secret default; OTP per-request + TTL + hash + tidak ditampilkan; `MergeOwned` menolak seed klien di produksi; validasi CSRF sungguhan.
2. **Schema:** putuskan 66 tabel — pakai sebagai target normalisasi (`PLAN.md` Fase 0) atau hapus yang tidak dipakai. Jangan biarkan schema mati.
3. **Dokumen:** samakan versi Go; tandai dokumen historis; buang/arsipkan aset React lama.
4. **Rumah tangga:** hapus `wesmk.exe` & `placeholder.templ`; batasi port DB di compose.

---

## 6. Catatan

Dokumen ini menggantikan audit React/Vite 25 Oktober 2026. Screenshot React lama di
`audit/` hanya bukti historis, bukan kondisi UI saat ini.
