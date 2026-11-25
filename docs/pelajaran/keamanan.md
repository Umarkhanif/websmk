## 2026-11-23 — pembayaran web
Masalah: `POST /student/invoice/pay` dan `/parent/invoice/pay` tetap melunasi tagihan di produksi.
Penyebab: handler web memanggil `state.PayInvoice` tanpa cek demo; tombol hanya disembunyikan di template.
Perbaikan: guard `!s.cfg.DemoMode` di kedua handler (404) sebelum otorisasi; regresi di `web_payment_guard_test.go`.
Verifikasi: test production 404 dan demo tetap redirect; `go vet` + `go test` hijau.

## 2026-11-23 — batch Fase A/P2
Masalah: CSP mematikan Alpine & konfirmasi inline; magic link dibakar bot lewat GET; error mutasi ditelan.
Penyebab: `script-src` tanpa `unsafe-eval`, konsumsi token di GET, `_ = s.mutateStateDoc` di 40+ handler.
Perbaikan: `unsafe-eval` khusus Alpine + `data-confirm`, magic link GET=konfirmasi/POST=login, semua mutasi memeriksa error, contact rate limit, blob yatim dihapus, submit ujian kosong ditolak, health diminimalkan.
Verifikasi: `TestSecurityHeadersAllowAlpineButBlockInlineScript`, `TestMagicLinkConfirmRequiresPost`, `go vet`/`go test` hijau.

## 2026-11-23 — OTP reversal
Masalah: satu OTP Kepala Sekolah bisa membatalkan banyak transaksi dan tersimpan di ledger/URL.
Penyebab: OTP tidak dihapus setelah dipakai; `reversalApprovalId` memuat OTP; flash memakai query string.
Perbaikan: OTP one-time, referensi approval acak, `otpCode` request dibersihkan, flash tanpa OTP.
Verifikasi: `TestReversalOTPIsSingleUseAndNotPersisted`; `go test ./...` hijau.

## 2026-11-23 — verifikasi publik
Masalah: `/verify` mengirim seluruh surat, kwitansi, dan jurnal PKL ke pengunjung anonim.
Penyebab: handler memuat semua list lalu template merendernya sebagai daftar/tabel.
Perbaikan: hanya kode yang dicari yang di-resolve; tab DUDI memakai kode jurnal, bukan daftar.
Verifikasi: `TestPublicVerifyDudiRendersOnlyRequestedJournal`; daftar global hilang dari template.

## 2026-11-23 — TrustProxy
Masalah: rate limit login bisa di-bypass dengan `X-Forwarded-For` palsu di production.
Penyebab: trust proxy default aktif dan memakai hop pertama header.
Perbaikan: default `false`; saat aktif pakai hop terakhir yang valid sebagai IP klien.
Verifikasi: `TestTrustProxyOffUnlessExplicitlyEnabled`, `TestClientIP*`.

## 2026-11-23 — PUT /api/state
Masalah: merge scoped memakai GetState + PutState sehingga mutasi bersamaan saling menimpa.
Penyebab: read-modify-write di luar transaksi/row lock.
Perbaikan: merge/overwrite dijalankan dalam `UpdateStateAudited` (`SELECT ... FOR UPDATE`) + audit satu transaksi.
Verifikasi: `go vet`/`go test` hijau; jalur DB perlu uji integrasi Postgres (belum ada di CI).

## 2026-11-23 — MFA TOTP akun istimewa
Masalah: kata sandi saja tidak cukup untuk akun yang bisa mengubah keuangan, akun, dan laporan kepsek.
Perbaikan: migrasi 014 (`users.totp_secret`, `users.totp_enabled`), TOTP SHA-1 6 digit/30 detik dengan toleransi ±1 langkah, challenge login sekali pakai di Redis (5 menit, maks 5 percobaan), cookie HttpOnly, enrollment di `/security`, reset MFA oleh Admin TU + cabut sesi, audit `account.mfa_*`.
Verifikasi: vektor RFC 6238, unit challenge single-use/akun nonaktif, E2E Chrome enrollment → login dua langkah → kode salah ditolak → kode benar masuk, reset MFA lewat UI, API `/api/auth/login` + `/api/auth/mfa`.
Catatan: MFA masih opsional; secret tersimpan di kolom `users.totp_secret` dan perlu enkripsi at-rest serta recovery code sebelum diwajibkan.

## 2026-11-23 — MFA wajib, enkripsi secret, recovery code, audit login
Masalah: MFA opsional membuat akun Admin TU/Bendahara/Kepala Sekolah tetap bisa dipakai hanya dengan kata sandi; secret TOTP plaintext; tidak ada pemulihan bila perangkat hilang; sesi dan riwayat login tidak terlihat.
Perbaikan: migrasi `015` (recovery hash + `auth_login_events` append-only); secret TOTP dienkripsi AES-256-GCM dengan `WESMK_MFA_KEY` dan data lama di-re-encrypt saat boot; 10 recovery code ber-hash sekali pakai; peran istimewa dikunci ke `/security` sampai enrollment selesai; `/security` menampilkan sesi aktif, riwayat login, dan audit global untuk Admin TU/Developer.
Verifikasi: round-trip enkripsi/wrong-key/legacy, unit recovery sekali pakai + policy peran, integrasi Postgres (kolom `v1:`, re-encrypt, consume, audit), E2E Chrome wajib MFA → enrollment → recovery code login → kode bekas ditolak → portal terbuka.
Catatan: rotasi `WESMK_MFA_KEY` saat ini membuat secret lama tidak terbaca; reset MFA semua akun atau tambahkan dukungan key-ring sebelum rotasi.

## 2026-11-23 — tugas per kelas
Masalah: siswa bisa mengumpulkan tugas ke kelas lain, setelah tenggat, atau berulang kali.
Penyebab: submission hanya menyimpan judul dan tidak terhubung ke entitas tugas.
Perbaikan: assignment (`id`, kelas, mapel, tenggat), validasi kelas + tenggat di mutation server, satu submission per siswa/tugas, pilihan tugas difilter di portal siswa.
Verifikasi: unit kelas/tenggat/duplikat, E2E Chrome guru buat tugas → siswa unggah → guru nilai 95 → siswa lihat nilai.

## 2026-11-23 — unggah tugas siswa
Masalah: form tugas hanya mengirim nama berkas palsu, sementara UI menjanjikan PDF maksimal 10 MB.
Penyebab: handler tugas belum memakai `document_blobs` dan hanya memanggil mutasi state.
Perbaikan: multipart 10 MB, validasi magic bytes PDF/JPG/PNG, blob server-side, metadata blob di submission, akses guru hanya ke berkas tugas (bukan dokumen identitas), hapus blob yatim saat mutasi gagal.
Verifikasi: unit metadata/penolakan blob kosong, E2E Chrome upload → guru melihat berkas → nilai 91 tersimpan, `go vet`/`go test` hijau.

## 2026-11-23 — PIN supervisor DUDI
Masalah: PIN DUDI tercetak di halaman publik, bocor di pesan error, dan endpoint approval bisa di-brute force.
Penyebab: hint literal di template, error menyertakan PIN benar, PIN plaintext di `app_state` dan payload siswa.
Perbaikan: hint dihapus, error generik, PIN di-hash (legacy di-upgrade), redaksi di `ScopedView`/`StaffView`, rate limit 20/10 menit.
Verifikasi: test hashing, error tanpa kebocoran, redaksi payload, dan render template tanpa PIN; `go test ./...` hijau.
