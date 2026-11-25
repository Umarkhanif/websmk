# Foto halaman masuk

Tiga foto ini dibundel ke dalam binary supaya template tidak bergantung pada URL
stok. Semuanya gratis untuk penggunaan komersial di bawah
[Pexels License](https://www.pexels.com/license/).

| File | Sumber | Fotografer |
|---|---|---|
| `login-hero.jpg` | [Pexels 35865718](https://www.pexels.com/photo/indonesian-high-school-students-in-classroom-35865718/) | Airlangga Jati |
| `login-campus.jpg` | [Pexels 35161340](https://www.pexels.com/photo/ceremonial-flag-exchange-in-school-yard-35161340/) | Tommy Kurniawan |
| `login-lab.jpg` | [Pexels 7742816](https://www.pexels.com/photo/students-busy-using-laptops-in-the-classroom-7742816/) | Alena Darmel |

## Mengganti dengan foto sekolah sendiri

1. Timpa ketiga file dengan nama yang sama.
2. Gunakan orientasi landscape:
   - `login-hero.jpg`: sekitar 1800×1200 px, objek utama di tengah.
   - `login-campus.jpg` dan `login-lab.jpg`: sekitar 1100×730 px, objek utama
     di tengah karena gambar dipotong `object-fit: cover`.
3. Jalankan `go build ./...` atau restart server agar aset embedded ikut berubah.
4. Jika memakai reverse proxy/CDN, naikkan query cache pada
   `internal/ui/pages/login.templ` (misalnya `app.css?v=20261124p`) atau
   bersihkan cache aset.

Jangan memakai foto siswa tanpa izin sekolah/wali. Foto suasana kelas, lab,
gedung, atau kegiatan tanpa wajah yang jelas lebih aman untuk template yang
dijual ke banyak sekolah.
