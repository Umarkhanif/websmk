# AI.md — WESMK (WAJIB DIBACA AI)

Style visual wajib project ini mengikuti referensi: **https://sfschool.org/About-Us**
("SFS skin"). Semua UI baru (landing & portal) harus konsisten dengan token dan
komponen yang sudah ada. Jangan ganti palet/font/radius tanpa persetujuan user.

## Token wajib (lihat `internal/ui/assets/app.css` `:root` & `landing.css` `.sf-body`)

- Primary/crimson: `#C60C46` (hover `#9E0A38`, soft `#FBE9EF`)
- Secondary/amber: `#FAA21B` (soft `#FDF1E0`, text `#7A4A05`)
- Third/teal: `#00AFA9`
- Slate/navy: `#373E43` (soft `#2E353A`), muted `#5C6770`/`#6C757C`
- Teks: `#181818`, muted `#5C6770`
- Surface: `#FFFFFF`, paper `#F8F9FA`, border `#E1E3E5`
- Font sans: `"Poppins"` (heading weight 800/900)
- Font serif: `"PT Serif"` (quote/intro)
- Radius: satu-sisi membulat (`0 76px 0 0`, `0 200px 0 0`, `0 12px 0 0`); tombol utama pill (`100px`); panel `6px`/`4px`

## Aturan

1. Pakai kelas `.sf-*` yang sudah ada; jangan buat komponen baru tanpa alasan.
2. Jangan ganti warna primary/secondary/third atau font.
3. Radius "satu sisi" untuk kartu/panel/hero/footer; pill untuk tombol utama.
4. Responsif: breakpoint 1080 / 900 / 560 (lihat media query di landing.css).
5. Aksesibilitas: `focus-visible` outline `#00AFA9`, skip-link, semantic HTML.
6. Aset gambar dari `internal/ui/assets/images`; embed pola `images/*.jpg` & `*.png`.
7. Jangan reintroduce React/Vite/Tailwind custom; stack terkunci templ/HTMX/Alpine.
8. Jangan tambah dependency frontend baru.

## Referensi cepat

- Visual: https://sfschool.org/About-Us
- Token portal: `internal/ui/assets/app.css` (blok "SFS skin (2026)")
- Token landing: `internal/ui/assets/landing.css` (`.sf-body`)
- Template: `internal/ui/pages/home.templ`

Aturan kerja umum (snapshot, commit, verifikasi) tetap di **AGENTS.md**.
