# Foto halaman masuk & beranda

Empat foto ini dibundel ke dalam binary supaya template tidak bergantung pada
URL stok. Tiga foto login gratis untuk penggunaan komersial di bawah
[Pexels License](https://www.pexels.com/license/).

| File | Sumber | Fotografer |
|---|---|---|
| `login-hero.jpg` | [Pexels 35865718](https://www.pexels.com/photo/indonesian-high-school-students-in-classroom-35865718/) | Airlangga Jati |
| `login-campus.jpg` | [Pexels 35161340](https://www.pexels.com/photo/ceremonial-flag-exchange-in-school-yard-35161340/) | Tommy Kurniawan |
| `login-lab.jpg` | [Pexels 7742816](https://www.pexels.com/photo/students-busy-using-laptops-in-the-classroom-7742816/) | Alena Darmel |
| `home-hero.jpg` | Gambar placeholder hasil generate, **bukan foto sekolah sungguhan** | — |
| `home-workshop.jpg` | Gambar placeholder hasil generate, **bukan foto sekolah sungguhan** | — |

## Mengganti dengan foto sekolah sendiri

1. Timpa file dengan nama yang sama.
2. Gunakan orientasi landscape:
   - `home-hero.jpg`: sekitar 1600×1000 px; dipotong `4 / 3` di desktop dan
     `16 / 10` di layar kecil, jadi letakkan objek utama di tengah.
   - `login-hero.jpg`: sekitar 1800×1200 px, objek utama di tengah.
   - `login-campus.jpg` dan `login-lab.jpg`: sekitar 1100×730 px, objek utama
     di tengah karena gambar dipotong `object-fit: cover`.
   - `home-workshop.jpg`: sekitar 1400×1050 px, dipotong `4 / 3` di beranda.
3. Jalankan `go build ./...` atau restart server agar aset embedded ikut berubah.
4. Jika memakai reverse proxy/CDN, naikkan query cache pada
   `internal/ui/pages/login.templ` dan `internal/ui/layout/base.templ`
   (misalnya `app.css?v=20261126b`) atau bersihkan cache aset.

Jangan memakai foto siswa tanpa izin sekolah/wali. Foto suasana kelas, lab,
gedung, atau kegiatan tanpa wajah yang jelas lebih aman untuk template yang
dijual ke banyak sekolah. `home-hero.jpg` dan `home-workshop.jpg` wajib diganti
sebelum dipakai publik: keduanya hanya pengisi tata letak, bukan dokumentasi
sekolah.
