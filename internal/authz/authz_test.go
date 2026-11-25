package authz

import "testing"

func TestCanRoleMatrix(t *testing.T) {
	owned := Ownership{Own: true}
	foreign := Ownership{Own: false}

	tests := []struct {
		name     string
		role     string
		action   Action
		resource Resource
		own      Ownership
		want     bool
	}{
		// Profil siswa
		{"siswa baca profil sendiri", "student", ActionRead, ResourceStudentProfile, owned, true},
		{"siswa baca profil lain", "student", ActionRead, ResourceStudentProfile, foreign, false},
		{"orang tua baca profil anak", "parent", ActionRead, ResourceStudentProfile, owned, true},
		{"orang tua baca profil lain", "parent", ActionRead, ResourceStudentProfile, foreign, false},
		{"guru baca profil", "teacher", ActionRead, ResourceStudentProfile, foreign, true},
		{"bendahara baca profil", "treasurer", ActionRead, ResourceStudentProfile, foreign, true},
		{"developer baca profil", "developer", ActionRead, ResourceStudentProfile, foreign, false},
		{"siswa ubah profil sendiri", "student", ActionUpdate, ResourceStudentProfile, owned, true},
		{"siswa ubah profil lain", "student", ActionUpdate, ResourceStudentProfile, foreign, false},
		{"admin TU buat profil", "admin_tu", ActionCreate, ResourceStudentProfile, foreign, true},
		{"kepsek hapus profil", "headmaster", ActionDelete, ResourceStudentProfile, foreign, false},

		// Keuangan
		{"bendahara baca keuangan", "treasurer", ActionRead, ResourceFinance, foreign, true},
		{"bendahara ubah keuangan", "treasurer", ActionUpdate, ResourceFinance, foreign, true},
		{"admin TU baca keuangan", "admin_tu", ActionRead, ResourceFinance, foreign, true},
		{"admin TU ubah keuangan", "admin_tu", ActionUpdate, ResourceFinance, foreign, false},
		{"kepsek approve reversal", "headmaster", ActionApprove, ResourceFinance, foreign, true},
		{"siswa baca tagihan sendiri", "student", ActionRead, ResourceFinance, owned, true},
		{"siswa baca tagihan lain", "student", ActionRead, ResourceFinance, foreign, false},
		{"guru baca keuangan", "teacher", ActionRead, ResourceFinance, foreign, false},

		// Presensi
		{"guru ubah presensi", "teacher", ActionUpdate, ResourceAttendance, foreign, true},
		{"piket ubah presensi", "piket", ActionUpdate, ResourceAttendance, foreign, true},
		{"orang tua ajukan izin anak", "parent", ActionSubmit, ResourceAttendance, owned, true},
		{"orang tua ajukan izin lain", "parent", ActionSubmit, ResourceAttendance, foreign, false},
		{"bendahara ubah presensi", "treasurer", ActionUpdate, ResourceAttendance, foreign, false},

		// Ujian
		{"siswa submit ujian sendiri", "student", ActionSubmit, ResourceExam, owned, true},
		{"siswa submit ujian lain", "student", ActionSubmit, ResourceExam, foreign, false},
		{"guru buat ujian", "teacher", ActionCreate, ResourceExam, foreign, true},
		{"guru baca kunci jawaban", "teacher", ActionRead, ResourceExam, foreign, true},
		{"admin TU baca ujian", "admin_tu", ActionRead, ResourceExam, foreign, false},
		{"siswa baca ujian", "student", ActionRead, ResourceExam, owned, true},

		// Kedisiplinan
		{"BK ubah pelanggaran", "discipline", ActionUpdate, ResourceDiscipline, foreign, true},
		{"guru laporkan pelanggaran", "teacher", ActionCreate, ResourceDiscipline, foreign, true},
		{"piket catat gerbang", "piket", ActionCreate, ResourceDiscipline, foreign, true},
		{"siswa lapor anonim", "student", ActionCreate, ResourceDiscipline, foreign, true},
		{"siswa ubah pelanggaran", "student", ActionUpdate, ResourceDiscipline, owned, false},
		{"guru baca catatan konseling", "teacher", ActionRead, ResourceDiscipline, foreign, true},

		// PKL
		{"siswa buat jurnal sendiri", "student", ActionCreate, ResourcePkl, owned, true},
		{"siswa buat jurnal lain", "student", ActionCreate, ResourcePkl, foreign, false},
		{"guru ACC jurnal", "teacher", ActionApprove, ResourcePkl, foreign, true},
		{"admin TU kelola mitra", "admin_tu", ActionCreate, ResourcePkl, foreign, true},
		{"developer baca PKL", "developer", ActionRead, ResourcePkl, foreign, false},

		// BKK
		{"siswa baca lowongan", "student", ActionRead, ResourceBkk, owned, true},
		{"siswa baca BKK tanpa kepemilikan", "student", ActionRead, ResourceBkk, foreign, false},
		{"siswa lamar", "student", ActionCreate, ResourceBkk, owned, true},
		{"admin TU kelola lowongan", "admin_tu", ActionCreate, ResourceBkk, foreign, true},
		{"guru ubah lowongan", "teacher", ActionUpdate, ResourceBkk, foreign, false},
		{"developer baca BKK", "developer", ActionRead, ResourceBkk, foreign, false},

		// Surat & pengaturan
		{"BK terbitkan surat", "discipline", ActionCreate, ResourceLetter, foreign, true},
		{"kepsek tanda tangan surat", "headmaster", ActionApprove, ResourceLetter, foreign, true},
		{"siswa baca surat sendiri", "student", ActionRead, ResourceLetter, owned, true},
		{"admin TU ubah pengaturan", "admin_tu", ActionUpdate, ResourceSchoolSettings, foreign, true},
		{"bendahara ubah pengaturan", "treasurer", ActionUpdate, ResourceSchoolSettings, foreign, false},

		// Bug report
		{"pengunjung tanpa peran tidak boleh", "", ActionCreate, ResourceBugReport, foreign, false},
		{"siswa kirim bug", "student", ActionCreate, ResourceBugReport, foreign, true},
		{"developer ubah bug", "developer", ActionUpdate, ResourceBugReport, foreign, true},
		{"guru ubah bug", "teacher", ActionUpdate, ResourceBugReport, foreign, false},

		// Default deny
		{"peran tak dikenal", "wizard", ActionRead, ResourceStudentProfile, owned, false},
		{"aksi tak dikenal", "admin_tu", Action("teleport"), ResourceStudentProfile, foreign, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Can(test.role, test.action, test.resource, test.own); got != test.want {
				t.Fatalf("Can(%q, %q, %q, own=%v) = %v, want %v",
					test.role, test.action, test.resource, test.own.Own, got, test.want)
			}
		})
	}
}
