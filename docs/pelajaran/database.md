## 2026-10-25 — database
Masalah: migrasi schema besar rawan gagal parsial.
Penyebab: urutan FK & trigger append-only.
Perbaikan: migration 003_schema.sql idempotent + DROP TRIGGER IF EXISTS.
Verifikasi: `go test ./...`, server boot, `\dt` 38 tabel.

## 2026-11-23 — cutover profil siswa
Masalah: profil siswa hanya hidup di blob `app_state`, tabel `students` tidak punya `app_id`/nama/kelas.
Penyebab: skema 003 terlalu minimal dan backfill lama tidak mengisi kolom baru.
Perbaikan: migrasi 007 (app_id + kolom profil), backfill sekali saat `app_id` kosong (termasuk numerik/flags), repository `identity` + authz terpusat, fallback blob.
Verifikasi: test unit authz/identity; test integrasi Postgres (`WESMK_TEST_DATABASE_URL`) untuk migrasi ganda, backfill baris lama, dan shadow sync.
Pelajaran: `COALESCE` saja tidak cukup untuk kolom numerik yang punya DEFAULT; harus dikaitkan ke penanda upgrade (`app_id` kosong), dan test integrasi menemukan bug ini.

## 2026-11-23 — cutover tagihan & ledger
Masalah: invoice/ledger hanya di blob; tabel domain tidak punya ID aplikasi, dan ledger append-only tidak bisa diisi metadata dengan UPDATE.
Penyebab: skema awal tidak menyimpan `app_id` invoice/receipt/student/actor.
Perbaikan: migrasi 008 (`app_id` invoice + tabel `financial_ledger_details`), backfill sekali via penanda `app_id`, shadow sync upsert metadata, repository `finance` + `authz`.
Verifikasi: test integrasi Postgres untuk invoice, ledger, sync pembayaran, restore metadata, dan trigger append-only.
Pelajaran: jangan melonggarkan trigger append-only demi backfill; simpan metadata tampilan di tabel terpisah.

## 2026-11-23 — cutover presensi
Masalah: presensi hanya ada di blob; tidak ada tabel domain dan status bisa hilang saat sync.
Penyebab: skema 003 tidak punya tabel attendance.
Perbaikan: migrasi 009 (`attendances` + `app_id` unik), backfill/shadow sync, repository `attendance` + authz, dashboard siswa & daftar guru.
Verifikasi: test unit policy/mapping; test integrasi Postgres untuk backfill, update status, penambahan hari, dan idempotensi re-backfill.
Pelajaran: kunci natural `app_id` + `ON CONFLICT DO UPDATE` membuat backfill aman diulang tanpa menggandakan presensi.

## 2026-11-23 — cutover bank soal & sesi ujian
Masalah: bank soal dan status proctoring hanya di blob; tabel ujian tidak punya ID aplikasi/progres.
Penyebab: skema awal untuk CBT belum dipakai dan `exam_sessions.exam_id` wajib.
Perbaikan: migrasi 010 (`app_id`, metadata soal, progress/lock/unlock/emergency), backfill/shadow sync, repository `exam` + authz, redaksi kunci jawaban untuk siswa.
Verifikasi: test unit redaksi/mapping; test integrasi Postgres untuk soal, opsi, variabel dinamis, sesi, sync status, dan idempotensi re-backfill.
Pelajaran: jangan mengirim struct domain apa adanya ke siswa; service harus membuat salinan tanpa kunci jawaban sebelum builder menerimanya.

## 2026-11-23 — lifecycle akun & e-sign
Masalah: tidak ada cara menonaktifkan akun/reset kata sandi, dan tombol e-sign kepala sekolah hanya menampilkan flash.
Penyebab: status akun tidak disimpan dan belum ada tabel tanda tangan.
Perbaikan: migrasi 012 (`users.is_active`), migrasi 013 (`executive_signatures`), service akun Admin TU, revoke sesi Redis, cek akun nonaktif saat login, hash+HMAC laporan kepsek, audit tiap aksi.
Verifikasi: test unit login nonaktif, test integrasi Postgres lifecycle akun + signature, `go vet`/`go test` hijau.
Pelajaran: jangan menonaktifkan akun tanpa mencabut token sesi; sesi lama tetap valid sampai key Redis dihapus.

## 2026-11-23 — cutover PKL & BKK
Masalah: portal PKL/BKK membaca blob, sementara tabel `pkl_*`/`bkk_*` hanya punya kolom minimal.
Penyebab: skema awal tidak menyimpan ID aplikasi, geofence, supervisor, jurnal, atau status lamaran.
Perbaikan: migrasi 011 (`app_id` + field portal), backfill/shadow sync berurutan (placement → log/jurnal, lowongan → lamaran), repository `pkl`/`bkk`, policy `ResourceBkk`, dan redaksi PIN supervisor di service.
Verifikasi: test unit policy/redaksi; test integrasi Postgres untuk placement, log geofence, jurnal, lowongan, lamaran, dan idempotensi sync ulang.
Pelajaran: saat sync dua tingkat, simpan UUID hasil upsert di map agar anak (log/jurnal/lamaran) tidak kehilangan induknya; jangan pernah mengandalkan filter builder untuk menyembunyikan PIN.

## 2026-11-20 — state audit & domain shadow
Masalah: mutasi `app_state` tidak meninggalkan jejak, tabel domain cepat basi.
Penyebab: write path hanya mengubah blob JSON.
Perbaikan: `UpdateStateAudited` + trigger append-only `state_audit_log`, backfill idempotent, shadow sync pasca-mutasi.
Verifikasi: `go vet ./...` + `go test ./...` hijau; fresh clone `go build ./...` lulus.

## 2026-11-24 — operasional portal (presensi, izin, piket, jurnal)
Masalah: piket menampilkan kelas contoh, izin orang tua mengabaikan tanggal, presensi guru tidak difilter kelas, dan jurnal mengajar global saling menimpa.
Penyebab: tabel domain (`attendances`, `parent_leave_requests`, `substitute_assignments`, `class_journals`) belum dipakai dan belum punya kolom tanggal/kelas.
Perbaikan: migrasi 017 (tanggal/kelas presensi, jurnal per slot, log gerbang, penugasan inval), store `portal_ops`, handler memvalidasi NISN/kelas/guru, UI Admin dipecah per bagian.
Verifikasi: test integrasi Postgres `TestPortalOperationsUseNormalizedData`, E2E HTTP admin→guru→piket→orang tua, `go vet`/`go test` hijau.
Pelajaran: filter kelas harus di query dan otorisasi server, bukan hanya di template; upsert presensi butuh kunci per siswa+tanggal agar izin tidak menggandakan baris.

## 2026-11-24 — ubah/hapus master kelas & jadwal
Masalah: Admin TU hanya bisa menambah kelas/jadwal; salah input tidak bisa diperbaiki dan penghapusan tidak punya proteksi.
Penyebab: belum ada operasi update/delete dan halaman hanya menampilkan tabel read-only.
Perbaikan: `UpdateClassroom`, `DeleteClassroom`, `UpdateSchedule`, `DeleteSchedule`; validasi ulang jadwal; kelas yang masih punya siswa/jadwal ditolak dengan `ErrMasterDataInUse`; sinkronisasi `students.classroom_name/major_code` dalam transaksi; audit aksi.
Verifikasi: test integrasi Postgres untuk update, sinkronisasi profil siswa, penolakan hapus kelas terpakai, hapus jadwal/kelas kosong; E2E Admin TU create→update→delete.
Pelajaran: rename kelas harus ikut memperbarui kolom denormalisasi di profil siswa; cek pemakaian relasi sebelum DELETE agar pesannya jelas, bukan error foreign key mentah.

## 2026-11-24 — rollover tahun ajaran
Masalah: pergantian tahun ajaran masih manual dan rawan; naik kelas, lulus, dan aktivasi semester bisa terjadi setengah jalan.
Penyebab: belum ada operasi rollover dan kelas dipakai sebagai slot lintas tahun.
Perbaikan: `RolloverStudents` memetakan kelas asal ke kelas tujuan sejurusan/tingkat berikutnya, meluluskan tingkat 12, memindahkan siswa aktif beserta `classroom_name`/`major_code`, lalu mengaktifkan tahun tujuan dalam satu transaksi. Kelas tujuan harus kosong; proses dari tingkat tertinggi lebih dulu agar slot XII turun ke XI dan XI ke X. Audit mencatat jumlah naik/lulus dan pemetaannya.
Verifikasi: test integrasi Postgres untuk penolakan mapping tidak valid, rantai X→XI→XII + kelulusan XII, sinkronisasi profil, dan aktivasi tahun; E2E Admin TU.
Pelajaran: rollover harus memproses tingkat tertinggi lebih dulu. Tanpa urutan itu, siswa XI dan X bisa bercampur di kelas tujuan yang belum dikosongkan.
