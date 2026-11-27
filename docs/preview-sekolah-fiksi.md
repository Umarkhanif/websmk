# Preview sekolah fiksi (sales demo)

Jalankan instance WESMK terpisah dengan **sekolah fiksi "SMK Cendekia Nusantara"**
dan data master fiktif, untuk ditunjukkan ke calon customer web.

```bash
docker compose up -d          # Postgres + Redis (atau sudah jalan)
./scripts/preview.sh
# buka http://localhost:3002/
```

Yang dipisahkan dari instance produksi:

- database `wesmk_preview` (migrasi otomatis saat boot, seed idempotent)
- Redis db `1`, port `PREVIEW_PORT` (default 3002)
- nama sekolah `WESMK_SCHOOL_NAME` (default sekolah fiksi di atas)
- akun `WESMK_USERS` fiktif; password `WESMK_PREVIEW_PASSWORD`
  (default `GantiSaya#2026`, **ganti kalau dibagikan**; jangan expose preview publik)

Landing publik `/` tidak butuh login. Akun preview fiktif:
`admin.tu`, `kepala.sekolah`, `guru.demo`.

## Sekolah fiksi & data

Identitas: **SMK Cendekia Nusantara**, 4 program keahlian (TKJ, RPL, TBSM, AKL),
12 rombel, 24 siswa fiktif/rombel, kepala sekolah fiktif, 8 mitra industri fiktif.
Semua orang/siswa/perusahaan di `scripts/preview/preview_seed.sql` **bukan data nyata**.
Sumber angka landing tetap data DB instance itu sendiri (`renderHomePage`).

## Nama sekolah konfigurasi

`WESMK_SCHOOL_NAME` mengganti nama sekolah di seluruh UI (landing, login,
sidebar, verifikasi dokumen, TOTP issuer). Kosong → `SMK Negeri 1 Rekayasa`.
