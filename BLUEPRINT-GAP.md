# Blueprint V3 vs Repo — Gap Analysis & Arah Kerja

**Sumber:** `~/Downloads/rencana sementara.pdf` — *WESMK MEGA SUPER BLUEPRINT (V3 ULTIMATE) — GO 1.23 + HTMX 2.X*, 51 halaman.  
**Tujuan dokumen ini:** memisahkan mana yang sudah sesuai blueprint, mana yang belum, dan menentukan jalur tercepat tanpa merusak UI/UX yang sudah ada.

---

## 1. Tujuan Proyek yang Gw Pahami dari PDF

WESMK adalah **ERP sekolah**, bukan website profil sekolah. Halaman publiknya hanya:

- `/login` — login NISN/NIP/No. WA, magic link orang tua.
- `/verify/:doc_hash` — verifikasi kwitansi/surat lewat HMAC.
- `/pkl/approve/:token` — supervisor DUDI menyetujui jurnal PKL dengan PIN.

Inti produknya: 8 role, 30+ route, 38 tabel PostgreSQL, CBT anti-curang, keuangan append-only, PKL/BKK, rapor holistik 360°, dan dashboard empatik orang tua. Filosofinya: memanusiakan murid, menghapus ranking 1–40, apresiasi "Si Paling...", anti-guru penjilat, kesiswaan "pisau bermata dua", dan orang tua sebagai partner.

### Stack yang dikunci blueprint

| Lapisan | Teknologi |
|---|---|
| Backend | Go 1.23+ (Fiber v2 / Chi Router), sqlc, pgx/v5 |
| UI | Go templ + HTMX 2.x + Alpine.js + Tailwind CSS |
| Database | PostgreSQL 16, 38 tabel, trigger append-only |
| Cache/queue | Redis 7 + Asynq |
| Real-time | Server-Sent Events (SSE) native Go |
| Storage | Cloudflare R2 via presigned URL |
| AI | Gemini 1.5 Flash, strict JSON, XML sandbox |
| Payment | Midtrans/Xendit VA, HMAC callback, active pull |
| WA | Fonnte/WAHA, handshake, spintax, micro-delay |
| Deploy | single binary Go, distroless read-only |

**RULE 1 blueprint:** dilarang membuat file `.php` atau menginstal **React/Vue/Next.js/Angular**. Seluruh UI wajib Go templ + HTMX + Alpine.js.

### Design yang dikunci blueprint

| Token | Nilai |
|---|---|
| Background | `#F2F4F8` |
| Surface | `#FFFFFF`, border `#E2E8F0`, radius 16px |
| Deep Navy | `#0B1730` |
| Vocational Teal | `#0A6C74` (hover `#08565D`) |
| Warm Gold | `#E8B84A` (soft `#FEFCE8`) |
| Status aman | text `#065F46`, bg `#D1FAE5`, dot `#10B981` |
| Status atensi | text `#92400E`, bg `#FEF3C7`, dot `#F59E0B` |
| Status kritis | text `#991B1B`, bg `#FEE2E2`, dot `#EF4444` |
| Status alpa | text `#1E293B`, bg `#E2E8F0`, dot `#334155` |
| Font | Plus Jakarta Sans + IBM Plex Mono |

Responsif: mobile 360–430px pakai bottom nav 4–5 tab, target sentuh **48×48px**, font dasar **14–16px**; desktop pakai sidebar kiri **260px** `#0B1730`, top context bar, grid 12 kolom.

---

## 2. Compliance Matrix

Status: ✅ sesuai · ⚠️ sebagian · ❌ belum / berbeda.

| Area | Blueprint V3 | Repo saat ini | Status |
|---|---|---|---|
| Framework UI | Go templ + HTMX + Alpine | React 19 + Vite | ❌ |
| Bundle JS | ~21 KB | 562 KB (148 KB gzip) | ❌ |
| Router backend | Fiber/Chi | `net/http` stdlib | ⚠️ |
| Database | 38 tabel + FK + trigger | 2 tabel (`users`, `app_state` JSONB) | ❌ |
| Query layer | sqlc | pgx manual | ❌ |
| Queue | Asynq | tidak ada | ❌ |
| SSE proctoring | native Go SSE | tidak ada | ❌ |
| Storage | R2 presigned URL | metadata saja | ❌ |
| AI | Gemini strict JSON + XML sandbox | client Gemini + prompt XML (sebagian) | ⚠️ |
| Payment | Midtrans/Xendit + HMAC + active pull | intent/callback HMAC, gateway demo | ⚠️ |
| WhatsApp | Fonnte/WAHA + handshake | hanya log/devLink | ❌ |
| Auth | Argon2id, Redis session, rate limit | sudah ada | ✅ |
| RBAC | 8 role terisolasi | UI 8 role; server scoping hanya student/parent | ⚠️ |
| CBT | zero-key, Redis timer, IndexedDB, lock permanen | UI demo; timer/lock di client | ⚠️ |
| Ledger | append-only trigger + OTP reversal | ledger dibuat client; belum immutable | ❌ |
| Public verify | HMAC server | state browser | ❌ |
| Design token | navy/teal/gold, radius 16 | slate semua, token tidak dipakai | ❌ |
| Halaman/route | 30+ route URL | 27 tab, tanpa URL router | ⚠️ |
| Komponen | 10 komponen templ reusable | komponen React | ❌ |
| Deploy | distroless non-root, health | sudah ada | ✅ |
| Test | unit + load test | unit saja, E2E manual | ⚠️ |
| Dokumen | blueprint + readiness | sudah ada | ✅ |

**Kesimpulan:** repo ini adalah **prototipe React yang meniru fitur UI blueprint**, bukan implementasi blueprint. Secara tampilan dan alur sudah banyak tercakup; secara arsitektur, data, dan keamanan masih jauh.

---

## 3. Yang Bisa Dipakai Ulang

Jangan buang semuanya. Yang masih berguna:

- Auth Go: Argon2id, Redis session, rate limit, cookie, CSP, health check, Docker distroless.
- Scoping baca/tulis siswa/ortu di `internal/state/scope.go` — logikanya bisa diporting.
- 8 role UI dan 27 tab sebagai **wireframe/reference** untuk versi templ+HTMX.
- Mock data 1.473 baris untuk demo/pitching.
- Rumus matematika: Haversine, transformasi, holistic score, outlier.
- 3 skenario demo yang sudah ada di header.
- Dokumen readiness yang sudah jujur soal blocker.

---

## 4. Dua Jalur

### Jalur A — Tetap React, selesaikan demo (cepat)

**Cocok kalau:** targetnya presentasi ke sekolah/yayasan dalam hitungan hari, bukan audit arsitektur.

- UI/UX existing dipertahankan; hanya visual cleanup + fitur yang kurang.
- Auth/scoping/bug P0 diperbaiki.
- Blueprint dipakai sebagai referensi produk dan token warna.
- **Tidak bisa** diklaim 100% sesuai blueprint karena Rule 1 (React dilarang).

**Estimasi:** 4–6 hari kerja untuk demo rapi; 3–4 minggu untuk pilot dengan data nyata (masih React, masih belum 38 tabel).

**Risiko:** kalau dosen/juri/pemberi kerja mengecek stack, ini gagal di Rule 1. Kalau tujuan cuma demo fungsional, ini jalur tercepat.

### Jalur B — Migrasi ke Go + templ + HTMX (sesuai blueprint)

**Cocok kalau:** blueprint adalah kontrak yang tidak bisa dilanggar, atau targetnya produksi jangka panjang.

- Repo React dibekukan sebagai referensi visual.
- Bangun skeleton Go+templ+HTMX+Alpine+Tailwind.
- Port layout, token, 10 komponen, login, dashboard siswa, CBT, finance, lalu role lain.
- Bangun 38 tabel + sqlc + trigger + Redis + Asynq + SSE + R2 + WA/VA.
- React tidak dipakai lagi di produksi.

**Estimasi:** blueprint sendiri merencanakan **12 minggu untuk 2 developer**. Dengan AI dan scope MVP, realistis **6–10 minggu**; full 38 tabel + integrasi bisa 10–12 minggu.

**Risiko:** jauh lebih lama; tapi ini satu-satunya jalur yang benar-benar sesuai blueprint.

### Jalur C — Hybrid (tidak disarankan)

Poles React untuk demo sambil membangun HTMX paralel. Ini menggandakan pekerjaan dan bikin dua source of truth. Kalau mau pindah, pindah setelah demo; jangan paralel.

---

## 5. Fitur yang Masih Kurang (Prioritas)

### P0 — keamanan & integritas data

1. **CBT server-authoritative:** timer di Redis, payload tanpa kunci jawaban, session per siswa, lock permanen, auto-hydrate.
2. **Live proctoring SSE:** status siswa, warning, lock, QR unlock, mass unlock.
3. **Kas server-side:** PIN 6 digit diverifikasi server, ledger append-only, reversal OTP kepsek.
4. **Public verify HMAC:** `/verify/:hash` benar-benar menghitung ulang tanda tangan server.
5. **Upload R2:** presigned URL, magic bytes, kompresi <500KB, UUID rename.
6. **Staff RBAC:** scoping baca/tulis per role staf, bukan snapshot penuh.
7. **Schema 38 tabel + sqlc + audit trail:** minimal users, students, invoices, ledger, exams, questions, sessions, documents, PKL.
8. **Hapus data sensitif dari localStorage** dan client state.

### P1 — integrasi yang disebut blueprint

9. WhatsApp gateway + handshake + spintax + micro-delay.
10. Midtrans/Xendit VA + webhook + active pull + rekonsiliasi.
11. Excel/PDF export native Go (excelize, maroto/wkhtmltopdf).
12. AI: PDF silabus, voice agenda 10 detik, essay grading XML sandbox, daily quiz pool.
13. Dapodik sync + e-Rapor.
14. PKL geotagging + EXIF + PIN DUDI.
15. BKK/tracer study BMW.

### P2 — fitur UI yang belum terlihat

16. Route `/student/exam-reflection/:session_id` sebagai halaman terpisah (sekarang "Bekal Ilmu" ada di dalam kiosk).
17. Route URL asli untuk semua tab (deep link, refresh, back).
18. Sertifikat "Si Paling..." dan Rewind share/download yang benar-benar menghasilkan file.
19. Empty state, skeleton, toast, error inline.
20. PWA/offline untuk CBT.

---

## 6. Working Agreement: UI/UX Freeze

Sesuai permintaan lo: **layout, navigasi, alur, dan struktur halaman tidak diubah.**

Yang **boleh**:

- Ganti token warna ke navy/teal/gold blueprint.
- Rapikan type scale: tidak ada teks <12px, body 14–16px, angka pakai IBM Plex Mono.
- Rapikan spacing, border, shadow, radius 16px.
- Perbaiki kontras, status badge, focus ring, target sentuh 48px.
- Hapus copy marketing/fake metrics, ganti label Inggris ke Indonesia.
- Sembunyikan kontrol demo di production.
- Perbaiki overflow tabel di mobile.
- Tambah toast/empty state tanpa mengubah alur.

Yang **tidak boleh**:

- Pindah posisi sidebar/header/kartu.
- Menambah/mengurangi tab navigasi.
- Mengubah urutan langkah ujian, pembayaran, atau form.
- Menambah halaman publik yang tidak ada di blueprint.
- Mengganti komponen dengan gaya baru yang mengubah cara pakai.

**Clean, bukan AI slop:**

- Tidak ada gradient, glassmorphism, neon, 3D blob, emoji dekoratif, atau shadow berlebihan.
- Tidak ada angka/metric palsu tanpa label "demo".
- Satu layar = satu tujuan utama; satu aksi primer.
- Palet terbatas: navy, teal, gold, status colors, abu-abu netral.
- Angka/timer/NISN/nominal pakai IBM Plex Mono; sisanya Plus Jakarta Sans.
- Konten nyata/empty state, bukan dummy yang menyamar sebagai data asli.

---

## 7. Keputusan yang Gw Butuh dari Lo

1. **Stack:** tetap React (cepat, demo) atau migrasi ke Go+templ+HTMX (sesuai blueprint, lama)?
2. **UI freeze:** layout/navigasi/alur saja yang dibekukan, atau warna/typography/spacing juga tidak boleh berubah?
3. **Prioritas fitur pertama:** keamanan/data (CBT, kas, verify, RBAC) atau integrasi (WA, VA, R2, Excel/PDF)?

Kalau lo pilih **React + visual cleanup**, gw bisa langsung eksekusi tanpa mengubah UI/UX. Kalau lo pilih **HTMX**, gw berhenti polish React dan mulai skeleton Go+templ+HTMX.
