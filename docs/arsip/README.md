# Arsip Dokumen

Dokumen di folder ini **sudah tidak menggambarkan kondisi kode saat ini**. Disimpan hanya
sebagai catatan sejarah keputusan dan temuan, bukan untuk dijadikan acuan.

**Acuan yang berlaku sekarang:**

- [`../PETA-PROYEK.md`](../PETA-PROYEK.md) — apa isi proyek, status nyata, arah kerja.
- [`../production-readiness.md`](../production-readiness.md) — blocker teknis & checklist deploy.
- [`../master-plan-2026-11-23.md`](../master-plan-2026-11-23.md) — rencana fase A–I.
- [`../../AGENTS.md`](../../AGENTS.md) — aturan kerja repo.

| File | Tanggal | Kenapa diarsipkan | Pengganti |
|---|---|---|---|
| `AUDIT-2026-11-20.md` | 20 Nov 2026 | Audit lama. Sebagian besar temuan P0/P1-nya sudah diperbaiki; angka skor dan daftar "blocker" sudah tidak akurat. | `../production-readiness.md` |
| `AUDIT-2026-11-23.md` | 23 Nov 2026 | Audit terakhir sebelum Fase A ditutup. Masih berguna untuk melacak asal-usul sebuah perbaikan, tapi semua isinya sudah ditindaklanjuti. | `../production-readiness.md` |
| `requirements-traceability.md` | 20 Nov 2026 | Matriks blueprint dengan route yang **tidak ada** di kode (`/finance/*`, `/student/exam-kiosk/:exam_id`) dan status "READY" yang berarti "demo jalan", bukan "siap produksi". Menyesatkan kalau dibaca sebagai status. | `../PETA-PROYEK.md` §2 & §7 |
| `BLUEPRINT-GAP-react.md` | 25 Okt–20 Nov 2026 | Analisis gap saat repo masih React, termasuk perbandingan Jalur A/B/C yang sudah diputuskan (Jalur B, selesai). Masih berguna sebagai rujukan token desain & falsafah produk. | `../PETA-PROYEK.md`, `../master-plan-2026-11-23.md` |
| `PLAN-awal.md` | awal proyek | Rencana 12 minggu berbasis Blueprint V3 sebelum migrasi. Sudah digantikan master plan yang mencerminkan kondisi nyata. | `../master-plan-2026-11-23.md` |

Catatan: dokumen arsip ini masih menyebut "mode demo", `WESMK_DEMO_MODE`, dan React.
Semua itu **sudah tidak ada di kode** — demo dihapus total pada 25 November 2026.

Kalau folder ini dirasa tidak perlu lagi, hapus saja; riwayat lengkapnya tetap ada di Git.
