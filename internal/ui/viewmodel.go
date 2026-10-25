package ui

import "strings"

// RoleShell is the shared chrome for every authenticated page. It deliberately
// contains only presentation data: handlers decide what the user may see.
type RoleShell struct {
	Role         string
	RoleLabel    string
	FullName     string
	Initials     string
	Username     string
	SchoolName   string
	AcademicYear string
	ActiveTab    string
	PageTitle    string
	Nav          []NavItem
	CSRFToken    string
	DemoMode     bool
	Flash        string
	FlashKind    string
}

type NavItem struct {
	ID         string
	Label      string
	ShortLabel string
	Badge      string
	Icon       string
	Href       string
}

type StudentDashboard struct {
	Shell           RoleShell
	Student         StudentView
	Invoice         InvoiceView
	HasInvoice      bool
	IsCleared       bool
	HasDispensation bool
	Attendance      AttendanceView
	Quiz            QuizView
	QuizQuestions   []QuizQuestionView
	Journal         JournalView
	ShowLeaderPanel bool
	LeaderStatus    string
	LeaderFeedback  string
}

type QuizQuestionView struct {
	ID      string
	Text    string
	Options []QuizOptionView
}

type QuizOptionView struct {
	Key  string
	Text string
}

type QuizResultView struct {
	Score   int
	Correct int
	Total   int
	Already bool
	Message string
}

type StudentView struct {
	ID                string
	FullName          string
	NISN              string
	NIK               string
	ClassroomName     string
	ParentName        string
	IsKetuaKelas      bool
	IsSekretaris      bool
	ScholarshipType   string
	DiscountPercent   int
	AttitudeScore     int
	DailyQuizAvg      int
	ActiveBadge       string
	ActiveBadgeReason string
	HomeroomNote      string
	BMWChoice         string
}

type InvoiceView struct {
	ID          string
	Number      string
	Period      string
	Status      string
	StatusLabel string
	BaseAmount  int
	FinalAmount int
	PaidAmount  int
	Remaining   int
	VANumber    string
	Provider    string
	IsPaid      bool
}

type AttendanceView struct {
	Status      string
	StatusLabel string
	Time        string
	Source      string
}

type QuizView struct {
	ID            string
	Subject       string
	Topic         string
	QuestionCount int
	Deadline      string
	Average       int
	Attempted     bool
	Score         int
}

type JournalView struct {
	Subject      string
	Topic        string
	TeacherName  string
	Summary      string
	LeaderStatus string
}

type PlaceholderPage struct {
	Shell       RoleShell
	Title       string
	Description string
	Module      string
}

type LoginPage struct {
	Tab          string
	Identifier   string
	ParentWA     string
	Error        string
	Notice       string
	MagicLink    string
	DemoMode     bool
	DemoPassword string
	DemoAccounts []DemoAccount
	CSRFToken    string
	AcademicYear string
}

type DemoAccount struct {
	Label      string
	Identifier string
	Group      string
}

// NavFor returns the same navigation labels and order as the React prototype,
// so the migration does not change the information architecture.
func NavFor(role string) []NavItem {
	switch role {
	case "student":
		return []NavItem{
			{ID: "dashboard", Label: "Beranda Belajar", ShortLabel: "Beranda", Icon: "dashboard", Href: "/student/dashboard"},
			{ID: "classroom", Label: "Kelas & Materi Harian", ShortLabel: "Kelas", Badge: "Kuis Singkat", Icon: "book", Href: "/student/classroom"},
			{ID: "exam-lobby", Label: "Ruang Persiapan Ujian", ShortLabel: "Persiapan", Icon: "file-check", Href: "/student/exam-lobby"},
			{ID: "exam-kiosk", Label: "Ruang Ujian Siswa", ShortLabel: "Ujian CBT", Badge: "Terjadwal", Icon: "monitor", Href: "/student/exam-kiosk"},
			{ID: "documents", Label: "Unggah Berkas Siswa", ShortLabel: "Berkas", Icon: "file-text", Href: "/student/documents"},
			{ID: "academic", Label: "Rapor, Nilai PKL & Karir", ShortLabel: "Rapor & PKL", Badge: "Prestasi", Icon: "award", Href: "/student/academic"},
			{ID: "rewind", Label: "Kilas Catatan Siswa", ShortLabel: "Kilas Balik", Badge: "Ringkasan", Icon: "sparkles", Href: "/student/rewind"},
		}
	case "parent":
		return []NavItem{
			{ID: "dashboard", Label: "Perkembangan Belajar Anak", ShortLabel: "Perkembangan", Badge: "Info Anak", Icon: "heart", Href: "/parent/dashboard"},
			{ID: "roadmap", Label: "Rencana Biaya Pendidikan", ShortLabel: "Biaya SPP", Icon: "trending", Href: "/parent/roadmap"},
			{ID: "future-plan", Label: "Konsultasi Masa Depan BMW", ShortLabel: "Karir BMW", Icon: "compass", Href: "/parent/future-plan"},
			{ID: "services", Label: "Layanan Izin / SPP Siswa", ShortLabel: "Layanan", Icon: "wallet", Href: "/parent/services"},
		}
	case "treasurer":
		return []NavItem{
			{ID: "dashboard", Label: "Ringkasan Keuangan Sekolah", ShortLabel: "Ringkasan", Badge: "Arus Kas", Icon: "dashboard", Href: "/treasurer/dashboard"},
			{ID: "billing", Label: "Loket Pembayaran SPP", ShortLabel: "Loket SPP", Badge: "Kwitansi", Icon: "receipt", Href: "/treasurer/billing"},
			{ID: "exam-clearance", Label: "Pemeriksaan Izin Ujian", ShortLabel: "Izin Ujian", Badge: "Dispensasi", Icon: "shield-alert", Href: "/treasurer/exam-clearance"},
			{ID: "reports", Label: "Buku Kas & Pembukuan", ShortLabel: "Buku Kas", Badge: "Catatan Baku", Icon: "file-text", Href: "/treasurer/reports"},
		}
	case "teacher":
		return []NavItem{
			{ID: "dashboard", Label: "Beranda Guru & Izin Mengajar", ShortLabel: "Beranda", Icon: "dashboard", Href: "/teacher/dashboard"},
			{ID: "classroom", Label: "Ruang Mengajar & Agenda", ShortLabel: "Kelas", Badge: "Presensi", Icon: "calendar", Href: "/teacher/classroom"},
			{ID: "question-bank", Label: "Bank Soal Kejuruan", ShortLabel: "Bank Soal", Icon: "help", Href: "/teacher/question-bank"},
			{ID: "exam-control", Label: "Pengawasan Ujian Siswa", ShortLabel: "Pengawasan", Badge: "Langsung", Icon: "monitor", Href: "/teacher/exam-control"},
			{ID: "homeroom-pkl", Label: "Wali Kelas & Magang Siswa", ShortLabel: "Wali & PKL", Badge: "Pantau", Icon: "users", Href: "/teacher/homeroom-pkl"},
		}
	case "discipline":
		return []NavItem{
			{ID: "violations", Label: "Bimbingan & Tata Tertib", ShortLabel: "Tata Tertib", Badge: "Konseling", Icon: "alert", Href: "/discipline/violations"},
			{ID: "transformation-index", Label: "Perkembangan Sikap Siswa", ShortLabel: "Perkembangan", Badge: "Evaluasi", Icon: "flame", Href: "/discipline/transformation-index"},
		}
	case "piket":
		return []NavItem{
			{ID: "dashboard", Label: "Pemantauan Kelas Harian", ShortLabel: "Piket Harian", Badge: "Piket", Icon: "radio", Href: "/piket/dashboard"},
		}
	case "admin_tu":
		return []NavItem{
			{ID: "master-data", Label: "Data Pokok & Arsip Berkas", ShortLabel: "Data Pokok", Icon: "sliders", Href: "/admin/master-data"},
			{ID: "dapodik-erapor", Label: "Layanan Surat & e-Rapor", ShortLabel: "e-Rapor", Badge: "Resmi", Icon: "file-text", Href: "/admin/dapodik-erapor"},
			{ID: "pkl-bkk", Label: "Mitra Industri & Bursa Kerja", ShortLabel: "Mitra & BKK", Icon: "briefcase", Href: "/admin/pkl-bkk"},
		}
	case "headmaster":
		return []NavItem{
			{ID: "executive", Label: "Pusat Kendali Kepala Sekolah", ShortLabel: "Pusat Kendali", Badge: "Utama", Icon: "graduation", Href: "/headmaster/executive"},
		}
	default:
		return nil
	}
}

func AriaCurrent(item NavItem, active string) string {
	if item.ID == active {
		return "page"
	}
	return "false"
}

func NavClass(item NavItem, active string) string {
	if item.ID == active {
		return "nav-link is-active"
	}
	return "nav-link"
}

func MobileItems(items []NavItem) []NavItem {
	if len(items) <= 4 {
		return items
	}
	return items[:3]
}

func MoreItems(items []NavItem) []NavItem {
	if len(items) <= 4 {
		return nil
	}
	return items[3:]
}

func HasMore(items []NavItem) bool { return len(items) > 4 }

func MoreActive(items []NavItem, active string) bool {
	for _, item := range items {
		if item.ID == active {
			return true
		}
	}
	return false
}

func ActiveLabel(items []NavItem, active string) string {
	for _, item := range items {
		if item.ID == active {
			return item.Label
		}
	}
	return "Portal Sekolah"
}

func RoleHome(role string) string {
	switch role {
	case "student":
		return "/student/dashboard"
	case "parent":
		return "/parent/dashboard"
	case "treasurer":
		return "/treasurer/dashboard"
	case "teacher":
		return "/teacher/dashboard"
	case "discipline":
		return "/discipline/violations"
	case "piket":
		return "/piket/dashboard"
	case "admin_tu":
		return "/admin/master-data"
	case "headmaster":
		return "/headmaster/executive"
	default:
		return "/login"
	}
}

func RoleLabel(role string) string {
	switch role {
	case "student":
		return "Murid"
	case "parent":
		return "Orang Tua"
	case "treasurer":
		return "Bendahara"
	case "teacher":
		return "Guru"
	case "discipline":
		return "Kesiswaan & BK"
	case "piket":
		return "Guru Piket"
	case "admin_tu":
		return "Admin TU"
	case "headmaster":
		return "Kepala Sekolah"
	default:
		return "Pengguna"
	}
}

func Initials(name string) string {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "W"
	}
	if len(parts) == 1 {
		runes := []rune(parts[0])
		if len(runes) > 1 {
			return strings.ToUpper(string(runes[:2]))
		}
		return strings.ToUpper(string(runes))
	}
	first := []rune(parts[0])
	last := []rune(parts[len(parts)-1])
	out := ""
	if len(first) > 0 {
		out += string(first[0])
	}
	if len(last) > 0 {
		out += string(last[0])
	}
	return strings.ToUpper(out)
}
