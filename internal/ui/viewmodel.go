package ui

import (
	"strings"
	"time"
)

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
	Progress        StudentProgressView
}

// StudentProgressView is the ongoing report card: the numbers a student (and
// their parent) can act on, rather than raw tables.
type StudentProgressView struct {
	HolisticScore  string
	ExamAverage    int
	CompletedExams int
	TotalExams     int
	QuizAverage    int
	AttitudeScore  int
	Attendance     int
	SppRemaining   int
	PendingTasks   int
	PklJournals    int
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
	Label       string
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

// HomePage is the public school landing page. Guests see it before login;
// signed-in accounts are redirected straight to their own portal. Every number
// on it comes from the database, never from a hardcoded fixture.
type HomePage struct {
	AcademicYear string
	Stats        []HomeStat
	Majors       []HomeMajor
}

type HomeStat struct {
	Label string
	Value string
}

type HomeMajor struct {
	Code     string
	Name     string
	Detail   string
	Classes  int
	Students int
}

type LoginPage struct {
	Tab          string
	Identifier   string
	ParentWA     string
	Error        string
	Notice       string
	CSRFToken    string
	AcademicYear string
}

type MagicLinkPage struct {
	Token     string
	CSRFToken string
	Error     string
}

type MFAPage struct {
	CSRFToken string
	Error     string
}

type SecurityPage struct {
	Shell         RoleShell
	Enabled       bool
	Required      bool
	Secret        string
	OTPAuthURI    string
	RecoveryCodes []string
	RecoveryCount int
	Session       SecuritySession
	LoginEvents   []SecurityLoginEvent
	AdminEvents   []SecurityLoginEvent
	ShowAdmin     bool
	Error         string
}

type SecuritySession struct {
	IP        string
	UserAgent string
	StartedAt time.Time
	ExpiresAt time.Time
}

type SecurityLoginEvent struct {
	CreatedAt  time.Time
	Identifier string
	Role       string
	Method     string
	Success    bool
	Reason     string
	IP         string
	UserAgent  string
}

// NavFor returns the same navigation labels and order as the React prototype,
// so the migration does not change the information architecture.
// Order matters: when a role has more than four items, the first three become
// the mobile bottom bar (see MobileItems).
func NavFor(role string) []NavItem {
	switch role {
	case "student":
		return []NavItem{
			{ID: "dashboard", Label: "Beranda Murid", ShortLabel: "Beranda", Icon: "dashboard", Href: "/student/dashboard"},
			{ID: "classroom", Label: "Kelas & Kuis", ShortLabel: "Kelas", Badge: "Kuis", Icon: "book", Href: "/student/classroom"},
			{ID: "exam-lobby", Label: "Persiapan Ujian", ShortLabel: "Persiapan", Icon: "file-check", Href: "/student/exam-lobby"},
			{ID: "exam-kiosk", Label: "Ujian CBT", ShortLabel: "Ujian", Badge: "Terjadwal", Icon: "monitor", Href: "/student/exam-kiosk"},
			{ID: "documents", Label: "Berkas & Tugas", ShortLabel: "Berkas", Icon: "file-text", Href: "/student/documents"},
			{ID: "academic", Label: "Rapor, PKL & Karir", ShortLabel: "Rapor & PKL", Badge: "Prestasi", Icon: "award", Href: "/student/academic"},
			{ID: "rewind", Label: "Kilas Balik", ShortLabel: "Kilas", Badge: "Ringkasan", Icon: "sparkles", Href: "/student/rewind"},
		}
	case "parent":
		return []NavItem{
			{ID: "dashboard", Label: "Perkembangan Anak", ShortLabel: "Perkembangan", Badge: "Info Anak", Icon: "heart", Href: "/parent/dashboard"},
			{ID: "roadmap", Label: "Biaya SPP", ShortLabel: "Biaya SPP", Icon: "trending", Href: "/parent/roadmap"},
			{ID: "future-plan", Label: "Konsultasi Karir", ShortLabel: "Karir", Icon: "compass", Href: "/parent/future-plan"},
			{ID: "services", Label: "Izin & Layanan", ShortLabel: "Layanan", Icon: "wallet", Href: "/parent/services"},
		}
	case "treasurer":
		return []NavItem{
			{ID: "dashboard", Label: "Ringkasan Kas", ShortLabel: "Ringkasan", Badge: "Arus Kas", Icon: "dashboard", Href: "/treasurer/dashboard"},
			{ID: "billing", Label: "Loket SPP", ShortLabel: "Loket SPP", Badge: "Kwitansi", Icon: "receipt", Href: "/treasurer/billing"},
			{ID: "exam-clearance", Label: "Izin Ujian", ShortLabel: "Izin Ujian", Badge: "Dispensasi", Icon: "shield-alert", Href: "/treasurer/exam-clearance"},
			{ID: "reports", Label: "Buku Kas", ShortLabel: "Buku Kas", Badge: "Pembukuan", Icon: "file-text", Href: "/treasurer/reports"},
		}
	case "teacher":
		return []NavItem{
			{ID: "dashboard", Label: "Beranda Guru", ShortLabel: "Beranda", Icon: "dashboard", Href: "/teacher/dashboard"},
			{ID: "classroom", Label: "Kelas & Presensi", ShortLabel: "Kelas", Badge: "Presensi", Icon: "calendar", Href: "/teacher/classroom"},
			{ID: "exam-control", Label: "Pengawasan Ujian", ShortLabel: "Pengawasan", Badge: "Langsung", Icon: "monitor", Href: "/teacher/exam-control"},
			{ID: "question-bank", Label: "Bank Soal", ShortLabel: "Bank Soal", Icon: "help", Href: "/teacher/question-bank"},
			{ID: "homeroom-pkl", Label: "Wali Kelas & PKL", ShortLabel: "Wali & PKL", Badge: "Pantau", Icon: "users", Href: "/teacher/homeroom-pkl"},
		}
	case "discipline":
		return []NavItem{
			{ID: "violations", Label: "Tata Tertib & BK", ShortLabel: "Tata Tertib", Badge: "Konseling", Icon: "alert", Href: "/discipline/violations"},
			{ID: "transformation-index", Label: "Perkembangan Sikap", ShortLabel: "Perkembangan", Badge: "Evaluasi", Icon: "flame", Href: "/discipline/transformation-index"},
		}
	case "piket":
		return []NavItem{
			{ID: "dashboard", Label: "Pemantauan Kelas Harian", ShortLabel: "Piket Harian", Badge: "Piket", Icon: "radio", Href: "/piket/dashboard"},
		}
	case "admin_tu":
		return []NavItem{
			{ID: "master-data", Label: "Data Pokok", ShortLabel: "Data Pokok", Icon: "sliders", Href: "/admin/master-data"},
			{ID: "dapodik-erapor", Label: "Surat & e-Rapor", ShortLabel: "e-Rapor", Badge: "Resmi", Icon: "file-text", Href: "/admin/dapodik-erapor"},
			{ID: "pkl-bkk", Label: "PKL & BKK", ShortLabel: "PKL & BKK", Icon: "briefcase", Href: "/admin/pkl-bkk"},
		}
	case "headmaster":
		return []NavItem{
			{ID: "executive", Label: "Pusat Kendali", ShortLabel: "Pusat Kendali", Badge: "Utama", Icon: "graduation", Href: "/headmaster/executive"},
		}
	case "developer":
		return []NavItem{
			{ID: "dashboard", Label: "Laporan Bug", ShortLabel: "Bug", Badge: "Aktif", Icon: "bug", Href: "/developer/dashboard"},
			{ID: "resets", Label: "Audit Reset Ujian", ShortLabel: "Audit Ujian", Icon: "rotate-ccw", Href: "/developer/resets"},
			{ID: "system", Label: "Status Sistem", ShortLabel: "Sistem", Icon: "sliders", Href: "/developer/system"},
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
	case "developer":
		return "/developer/dashboard"
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
	case "developer":
		return "Developer"
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
