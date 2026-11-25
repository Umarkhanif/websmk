# Font yang dibundel

Dulu halaman memuat font dari Google Fonts. Sekarang semua file ada di folder ini
dan di-embed ke binary, jadi tidak ada permintaan ke pihak ketiga dan tampilan
pertama sudah memakai typeface yang benar.

Diambil dari Google Fonts API (subset **latin** dan **latin-ext** saja):

| File | Keluarga | Berat | Ukuran |
|---|---|---|---|
| `plus-jakarta-sans-latin.woff2` | Plus Jakarta Sans | 200–800 (variable) | ±27 KB |
| `plus-jakarta-sans-latin-ext.woff2` | Plus Jakarta Sans | 200–800 (variable) | ±22 KB |
| `ibm-plex-mono-400-latin.woff2` | IBM Plex Mono | 400 | ±15 KB |
| `ibm-plex-mono-400-latin-ext.woff2` | IBM Plex Mono | 400 | ±14 KB |
| `ibm-plex-mono-500-latin.woff2` | IBM Plex Mono | 500 | ±15 KB |
| `ibm-plex-mono-500-latin-ext.woff2` | IBM Plex Mono | 500 | ±14 KB |
| `ibm-plex-mono-600-latin.woff2` | IBM Plex Mono | 600 | ±16 KB |
| `ibm-plex-mono-600-latin-ext.woff2` | IBM Plex Mono | 600 | ±14 KB |

`@font-face` untuk file-file ini ada di bagian atas `../app.css` (lengkap dengan
`unicode-range`). Keduanya memakai `font-display: swap`.

## Lisensi

Keduanya **SIL Open Font License 1.1**, boleh dipakai komersial dan dibundel:

- `OFL-PlusJakartaSans.txt`
- `OFL-IBMPlexMono.txt`

Jangan hapus file lisensi ini saat memperbarui font.

## Mengganti font

1. Unduh `woff2` subset latin + latin-ext.
2. Taruh di folder ini dan perbarui blok `@font-face` di `../app.css`.
3. `go build ./...` atau restart server agar aset embedded ikut berubah.
4. Naikkan query cache `app.css?v=...` di `../layout/base.templ` dan halaman publik.
