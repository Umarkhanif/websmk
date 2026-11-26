package ui

// RoleGuide explains, in plain language, what a role is responsible for and how
// its work hands off to other roles. It is content, not permission: the actual
// access rules still live in the handlers and the state scope.
type RoleGuide struct {
	Role    string
	Tagline string
	Mission string
	Duties  []RoleDuty
	Steps   []RoleStep
	Handoff string
}

type RoleDuty struct {
	Title  string
	Detail string
}

type RoleStep struct {
	Title  string
	Detail string
	Href   string
}

// RoleGuideFor returns the working brief shown on a role's home page.
func RoleGuideFor(role string) RoleGuide {
	switch role {
	case "student":
		return RoleGuide{
			Role:    role,
			Tagline: "Buku kerja murid",
			Mission: "Datang, belajar, dan selesaikan tanggung jawab kelas. Semua catatanmu terkumpul di satu tempat.",
			Duties: []RoleDuty{
				{Title: "Cek presensi & SPP", Detail: "Presensi gerbang dan status tagihan muncul di beranda."},
				{Title: "Kerjakan kuis & tugas", Detail: "Kuis harian, tugas praktik, dan berkas diunggah dari menunya masing-masing."},
				{Title: "Siapkan ujian CBT", Detail: "Ujian terbuka hanya jika administrasi sudah beres atau ada dispensasi."},
			},
			Steps: []RoleStep{
				{Title: "Kelas & Kuis", Detail: "Agenda guru, kuis, umpan balik", Href: "/student/classroom"},
				{Title: "Berkas & Tugas", Detail: "Dokumen, unggahan praktik", Href: "/student/documents"},
				{Title: "Ruang Ujian", Detail: "CBT dengan pengawasan", Href: "/student/exam-lobby"},
				{Title: "Rapor & PKL", Detail: "Nilai, jurnal magang, BKK", Href: "/student/academic"},
			},
			Handoff: "Nilai dan berkasmu diperiksa Guru dan Admin TU. Kalau SPP menahan ujian, Bendahara yang memberi dispensasi.",
		}
	case "parent":
		return RoleGuide{
			Role:    role,
			Tagline: "Buku kerja orang tua",
			Mission: "Pantau perkembangan anak dan urus izin serta biaya pendidikan tanpa harus datang ke sekolah.",
			Duties: []RoleDuty{
				{Title: "Pantau perkembangan", Detail: "Kehadiran, nilai, dan catatan wali kelas tampil di beranda."},
				{Title: "Urus biaya pendidikan", Detail: "Lihat tagihan SPP dan rencana biaya sampai lulus."},
				{Title: "Ajukan izin & konsultasi", Detail: "Izin tidak masuk dan konsultasi jalur karir anak."},
			},
			Steps: []RoleStep{
				{Title: "Perkembangan", Detail: "Presensi & rapor berjalan", Href: "/parent/dashboard"},
				{Title: "Biaya SPP", Detail: "Tagihan & rencana biaya", Href: "/parent/roadmap"},
				{Title: "Izin & Layanan", Detail: "Ajukan izin anak", Href: "/parent/services"},
				{Title: "Konsultasi Karir", Detail: "Bekerja, kuliah, wirausaha", Href: "/parent/future-plan"},
			},
			Handoff: "Izin yang diajukan masuk ke wali kelas. Potongan atau penundaan SPP diputuskan Bendahara.",
		}
	case "treasurer":
		return RoleGuide{
			Role:    role,
			Tagline: "Buku kerja bendahara",
			Mission: "Jaga uang sekolah tetap rapi: terbitkan tagihan, catat pembayaran, dan seimbangkan buku kas.",
			Duties: []RoleDuty{
				{Title: "Tagihan & pembayaran", Detail: "Loket menerima VA maupun tunai dan menerbitkan kwitansi."},
				{Title: "Beasiswa & dispensasi", Detail: "Atur potongan siswa dan izin ujian bagi yang menunggak."},
				{Title: "Pembukuan", Detail: "Setiap transaksi masuk buku kas; koreksi butuh otorisasi Kepala Sekolah."},
			},
			Steps: []RoleStep{
				{Title: "Ringkasan Kas", Detail: "Tagihan, terkumpul, tunggakan", Href: "/treasurer/dashboard"},
				{Title: "Loket SPP", Detail: "Terima pembayaran & kwitansi", Href: "/treasurer/billing"},
				{Title: "Izin Ujian", Detail: "Dispensasi siswa menunggak", Href: "/treasurer/exam-clearance"},
				{Title: "Buku Kas", Detail: "Rekap & pembalik transaksi", Href: "/treasurer/reports"},
			},
			Handoff: "Data siswa dan kelas datang dari Admin TU. Pembalik transaksi harus disetujui Kepala Sekolah lewat OTP.",
		}
	case "teacher":
		return RoleGuide{
			Role:    role,
			Tagline: "Buku kerja guru",
			Mission: "Mengajar dengan catatan yang jelas: agenda, presensi, nilai, dan pengawasan ujian.",
			Duties: []RoleDuty{
				{Title: "Agenda & presensi", Detail: "Isi topik dan catatan kelas setiap slot mengajar."},
				{Title: "Tugas & penilaian", Detail: "Buat tugas, nilai unggahan siswa, dan kelola bank soal."},
				{Title: "Pengawasan ujian", Detail: "Pantau sesi CBT, buka kunci, dan tangani insiden."},
			},
			Steps: []RoleStep{
				{Title: "Kelas & Presensi", Detail: "Agenda, presensi, nilai tugas", Href: "/teacher/classroom"},
				{Title: "Bank Soal", Detail: "Soal ujian & kuis", Href: "/teacher/question-bank"},
				{Title: "Pengawasan Ujian", Detail: "Sesi siswa & reset", Href: "/teacher/exam-control"},
				{Title: "Wali & PKL", Detail: "Siswa wali & jurnal magang", Href: "/teacher/homeroom-pkl"},
			},
			Handoff: "Jadwal datang dari Admin TU. Agenda dan nilaimu terlihat oleh murid serta orang tua; pelanggaran ujian diteruskan ke Piket dan Kesiswaan.",
		}
	case "discipline":
		return RoleGuide{
			Role:    role,
			Tagline: "Buku kerja kesiswaan & BK",
			Mission: "Tegakkan tata tertib, lindungi pelapor, dan dampingi siswa sampai pulih.",
			Duties: []RoleDuty{
				{Title: "Catat pelanggaran", Detail: "Poin pengurang masuk ke rapor karakter siswa."},
				{Title: "Tindak lanjuti laporan", Detail: "Laporan sebaya diverifikasi sebelum diproses."},
				{Title: "Pendampingan", Detail: "Jadwalkan konsultasi dan pantau sesi konseling."},
			},
			Steps: []RoleStep{
				{Title: "Tata Tertib", Detail: "Pelanggaran, laporan, konseling", Href: "/discipline/violations"},
				{Title: "Perkembangan Sikap", Detail: "Indeks pemulihan siswa", Href: "/discipline/transformation-index"},
			},
			Handoff: "Surat panggilan disahkan Kepala Sekolah. Poin sikap memengaruhi rapor 360° yang dilihat murid dan orang tua.",
		}
	case "piket":
		return RoleGuide{
			Role:    role,
			Tagline: "Buku kerja guru piket",
			Mission: "Pastikan tidak ada kelas kosong dan setiap keluar-masuk gerbang tercatat.",
			Duties: []RoleDuty{
				{Title: "Pantau kelas", Detail: "Cocokkan jadwal hari ini dengan kondisi nyata."},
				{Title: "Tunjuk pengganti", Detail: "Catat guru pengganti saat pengampu berhalangan."},
				{Title: "Jaga gerbang", Detail: "Catat izin keluar-masuk dan teruskan ke wali kelas."},
			},
			Steps: []RoleStep{
				{Title: "Sinyal Guru", Detail: "Izin mengajar masuk ke piket", Href: "/teacher/dashboard"},
				{Title: "Papan Piket", Detail: "Kelas, pengganti, log gerbang", Href: "/piket/dashboard"},
				{Title: "Laporan", Detail: "Diteruskan ke TU & Kepsek", Href: ""},
			},
			Handoff: "Kelas kosong diteruskan ke Kesiswaan. Catatan gerbang dipakai untuk presensi siswa dan rekap kehadiran.",
		}
	case "admin_tu":
		return RoleGuide{
			Role:    role,
			Tagline: "Buku kerja tata usaha",
			Mission: "Jaga data induk sekolah tetap benar, karena semua portal lain membaca dari sini.",
			Duties: []RoleDuty{
				{Title: "Data pokok", Detail: "Siswa, kelas, jadwal, tahun ajaran, dan akun pengguna."},
				{Title: "Berkas & surat", Detail: "Verifikasi dokumen siswa serta terbitkan surat dan e-rapor."},
				{Title: "Mitra industri", Detail: "Kelola tempat PKL dan lowongan bursa kerja khusus."},
			},
			Steps: []RoleStep{
				{Title: "Data Pokok", Detail: "Siswa, kelas, jadwal, akun", Href: "/admin/master-data"},
				{Title: "Surat & e-Rapor", Detail: "Dokumen & legalitas", Href: "/admin/dapodik-erapor"},
				{Title: "PKL & BKK", Detail: "Mitra industri & lamaran", Href: "/admin/pkl-bkk"},
			},
			Handoff: "Jadwal yang kamu isi menentukan agenda Guru dan papan Piket. Akun yang kamu buat membuka portal tiap peran.",
		}
	case "headmaster":
		return RoleGuide{
			Role:    role,
			Tagline: "Buku kerja kepala sekolah",
			Mission: "Lihat kondisi sekolah secara utuh dan ambil keputusan yang butuh tanda tangan.",
			Duties: []RoleDuty{
				{Title: "Baca kinerja", Detail: "Indeks karakter dan dampak mengajar guru."},
				{Title: "Otorisasi kas", Detail: "Terbitkan OTP untuk koreksi transaksi bendahara."},
				{Title: "Sahkan dokumen", Detail: "Tanda tangan elektronik surat dan rapor."},
			},
			Steps: []RoleStep{
				{Title: "Pusat Kendali", Detail: "Kinerja, kas, karakter", Href: "/headmaster/executive"},
				{Title: "Otorisasi", Detail: "OTP pembalik transaksi", Href: "/headmaster/executive"},
				{Title: "E-Sign", Detail: "Sahkan laporan & surat", Href: "/headmaster/executive"},
			},
			Handoff: "Data berasal dari TU, Bendahara, dan Kesiswaan. OTP yang kamu terbitkan dipakai Bendahara untuk membalik transaksi.",
		}
	case "developer":
		return RoleGuide{
			Role:    role,
			Tagline: "Buku kerja developer",
			Mission: "Jaga aplikasi tetap sehat: tangani laporan bug dan audit tindakan sensitif.",
			Duties: []RoleDuty{
				{Title: "Triase bug", Detail: "Ubah status laporan dari antrean sampai selesai."},
				{Title: "Audit ujian", Detail: "Periksa reset sesi CBT yang dilakukan pengawas."},
				{Title: "Cek sistem", Detail: "Pantau kesiapan layanan dan basis data."},
			},
			Steps: []RoleStep{
				{Title: "Bug Reports", Detail: "Antrean & status perbaikan", Href: "/developer/dashboard"},
				{Title: "Audit Ujian", Detail: "Riwayat reset sesi", Href: "/developer/resets"},
				{Title: "Status Sistem", Detail: "Diagnostik layanan", Href: "/developer/system"},
			},
			Handoff: "Semua peran bisa melapor lewat menu Kontak. Status perbaikan terlihat oleh pelapor di halaman yang sama.",
		}
	default:
		return RoleGuide{}
	}
}

// RoleHomeTab reports which nav item acts as the role's landing page, so pages
// can show the working brief only once instead of on every tab.
func RoleHomeTab(role string) string {
	switch role {
	case "student":
		return "dashboard"
	case "parent":
		return "dashboard"
	case "treasurer":
		return "dashboard"
	case "teacher":
		return "dashboard"
	case "discipline":
		return "violations"
	case "piket":
		return "dashboard"
	case "admin_tu":
		return "master-data"
	case "headmaster":
		return "executive"
	case "developer":
		return "dashboard"
	default:
		return ""
	}
}
