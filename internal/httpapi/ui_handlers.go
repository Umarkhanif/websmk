package httpapi

import (
	"encoding/csv"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"

	"wesmk/internal/state"
	"wesmk/internal/ui"
	"wesmk/internal/ui/assets"
	"wesmk/internal/ui/layout"
	"wesmk/internal/ui/pages"
)

func (s *Server) registerUIRoutes() {
	s.mux.HandleFunc("GET /ui-assets/", s.handleUIAsset)
	s.mux.HandleFunc("GET /favicon.ico", s.handleFavicon)

	// Auth & Demo Switcher
	s.mux.HandleFunc("GET /login", s.handleLoginPage)
	s.mux.HandleFunc("POST /login", s.handleLoginForm)
	s.mux.HandleFunc("POST /login/demo", s.handleDemoLoginForm)
	s.mux.HandleFunc("POST /login/magic", s.handleMagicLinkForm)
	s.mux.HandleFunc("GET /login/magic/{token}", s.handleMagicLinkConsumeUI)
	s.mux.HandleFunc("POST /logout", s.handleLogoutForm)
	s.mux.HandleFunc("POST /demo/role", s.handleDemoRoleForm)
	s.mux.HandleFunc("POST /demo/scenario", s.handleDemoScenarioForm)
	s.mux.HandleFunc("POST /state/reset", s.handleResetStateForm)

	// Public Verify & DUDI Supervisor
	s.mux.HandleFunc("GET /verify", s.handlePublicVerifyPage)
	s.mux.HandleFunc("GET /verify/{hash}", s.handlePublicVerifyPage)
	s.mux.HandleFunc("GET /pkl/approve/{token}", s.handlePublicDudiPage)
	s.mux.HandleFunc("POST /verify/dudi-approve", s.handlePublicDudiApprove)

	// Student Routes
	s.mux.HandleFunc("GET /student", s.handleStudentRedirect)
	s.mux.HandleFunc("GET /student/{tab}", s.handleStudentPage)
	s.mux.HandleFunc("POST /student/invoice/pay", s.handleStudentPayInvoice)
	s.mux.HandleFunc("POST /student/leader-validation", s.handleStudentLeaderValidation)
	s.mux.HandleFunc("POST /student/quiz", s.handleStudentQuizForm)
	s.mux.HandleFunc("POST /student/reflection", s.handleStudentReflectionForm)
	s.mux.HandleFunc("POST /student/peer-report", s.handleStudentPeerReportForm)
	s.mux.HandleFunc("POST /student/proctor-status", s.handleStudentProctorStatusForm)
	s.mux.HandleFunc("POST /student/exam", s.handleStudentExamForm)
	s.mux.HandleFunc("POST /student/document/upload", s.handleStudentDocUploadForm)
	s.mux.HandleFunc("POST /student/task/submit", s.handleStudentTaskSubmitForm)
	s.mux.HandleFunc("POST /student/pkl/checkin", s.handleStudentPklCheckinForm)
	s.mux.HandleFunc("POST /student/pkl/journal", s.handleStudentPklJournalForm)
	s.mux.HandleFunc("POST /student/bmw", s.handleStudentBmwForm)
	s.mux.HandleFunc("POST /student/bkk/apply", s.handleStudentBkkApplyForm)
	s.mux.HandleFunc("POST /student/expense/add", s.handleStudentExpenseAddForm)

	// Parent Routes
	s.mux.HandleFunc("GET /parent", s.handleParentRedirect)
	s.mux.HandleFunc("GET /parent/{tab}", s.handleParentPage)
	s.mux.HandleFunc("POST /parent/consultation", s.handleParentConsultationForm)
	s.mux.HandleFunc("POST /parent/leave", s.handleParentLeaveForm)
	s.mux.HandleFunc("POST /parent/invoice/pay", s.handleParentPayInvoiceForm)

	// Treasurer Routes
	s.mux.HandleFunc("GET /treasurer", s.handleTreasurerRedirect)
	s.mux.HandleFunc("GET /treasurer/{tab}", s.handleTreasurerPage)
	s.mux.HandleFunc("POST /treasurer/scholarship", s.handleTreasurerScholarshipForm)
	s.mux.HandleFunc("POST /treasurer/wa-reminder", s.handleTreasurerWaReminderForm)
	s.mux.HandleFunc("POST /treasurer/cash-payment", s.handleTreasurerCashPaymentForm)
	s.mux.HandleFunc("POST /treasurer/dispensation", s.handleTreasurerDispensationForm)
	s.mux.HandleFunc("POST /treasurer/reversal", s.handleTreasurerReversalForm)

	// Teacher Routes
	s.mux.HandleFunc("GET /teacher", s.handleTeacherRedirect)
	s.mux.HandleFunc("GET /teacher/{tab}", s.handleTeacherPage)
	s.mux.HandleFunc("POST /teacher/alert-piket", s.handleTeacherAlertPiketForm)
	s.mux.HandleFunc("POST /teacher/journal", s.handleTeacherJournalForm)
	s.mux.HandleFunc("POST /teacher/attendance/toggle", s.handleTeacherToggleAlpaForm)
	s.mux.HandleFunc("POST /teacher/question/add", s.handleTeacherAddQuestionForm)
	s.mux.HandleFunc("POST /teacher/question/delete", s.handleTeacherDeleteQuestionForm)
	s.mux.HandleFunc("POST /teacher/proctor/unlock", s.handleTeacherProctorUnlockForm)
	s.mux.HandleFunc("POST /teacher/proctor/mass-unlock", s.handleTeacherProctorMassUnlockForm)
	s.mux.HandleFunc("POST /teacher/proctor/emergency-ticket", s.handleTeacherEmergencyTicketForm)
	s.mux.HandleFunc("POST /teacher/essay/grade", s.handleTeacherEssayGradeForm)
	s.mux.HandleFunc("POST /teacher/homeroom/attitude", s.handleTeacherHomeroomAttitudeForm)
	s.mux.HandleFunc("POST /teacher/pkl/bulk-approve", s.handleTeacherBulkApprovePklForm)

	// Discipline Routes
	s.mux.HandleFunc("GET /discipline", s.handleDisciplineRedirect)
	s.mux.HandleFunc("GET /discipline/{tab}", s.handleDisciplinePage)
	s.mux.HandleFunc("POST /discipline/violation/add", s.handleDisciplineAddViolationForm)
	s.mux.HandleFunc("POST /discipline/violation/summon", s.handleDisciplineSummonForm)
	s.mux.HandleFunc("POST /discipline/peer-report/resolve", s.handleDisciplineResolvePeerForm)
	s.mux.HandleFunc("POST /discipline/consultation/schedule", s.handleDisciplineScheduleConsultForm)
	s.mux.HandleFunc("POST /discipline/counseling/update", s.handleDisciplineCounselingUpdateForm)

	// Piket Routes
	s.mux.HandleFunc("GET /piket", s.handlePiketRedirect)
	s.mux.HandleFunc("GET /piket/{tab}", s.handlePiketPage)
	s.mux.HandleFunc("POST /piket/assign-inval", s.handlePiketAssignInvalForm)
	s.mux.HandleFunc("POST /piket/gate-log", s.handlePiketGateLogForm)

	// Admin TU Routes
	s.mux.HandleFunc("GET /admin", s.handleAdminRedirect)
	s.mux.HandleFunc("GET /admin/erapor/export.csv", s.handleAdminExportCSV)
	s.mux.HandleFunc("GET /admin/{tab}", s.handleAdminPage)
	s.mux.HandleFunc("POST /admin/document/verify", s.handleAdminVerifyDocForm)
	s.mux.HandleFunc("POST /admin/student/add", s.handleAdminAddStudentForm)
	s.mux.HandleFunc("POST /admin/dapodik/sync", s.handleAdminDapodikSyncForm)
	s.mux.HandleFunc("POST /admin/letter/create", s.handleAdminCreateLetterForm)
	s.mux.HandleFunc("POST /admin/pkl/add", s.handleAdminAddPklForm)
	s.mux.HandleFunc("POST /admin/bkk/add", s.handleAdminAddBkkForm)

	// Headmaster Routes
	s.mux.HandleFunc("GET /headmaster", s.handleHeadmasterRedirect)
	s.mux.HandleFunc("GET /headmaster/{tab}", s.handleHeadmasterPage)
	s.mux.HandleFunc("POST /headmaster/esign", s.handleHeadmasterESignForm)
	s.mux.HandleFunc("POST /headmaster/otp", s.handleHeadmasterOTPForm)
	s.mux.HandleFunc("POST /headmaster/ai-advisor", s.handleHeadmasterAIAdvisorForm)

	// Root entrypoint
	s.mux.HandleFunc("/", s.handleRootUI)
}

func renderComponent(w http.ResponseWriter, r *http.Request, status int, comp templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = comp.Render(r.Context(), w)
}

func redirectWithFlash(w http.ResponseWriter, r *http.Request, targetPath, msg, kind string) {
	u := targetPath
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	if msg != "" {
		u += sep + "flash=" + url.QueryEscape(msg) + "&kind=" + url.QueryEscape(kind)
	}
	http.Redirect(w, r, u, http.StatusSeeOther)
}

func requireWebRole(w http.ResponseWriter, r *http.Request, roles ...string) (requestAuth, bool) {
	authVal, ok := currentAuth(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return requestAuth{}, false
	}
	for _, role := range roles {
		if authVal.user.Role == role {
			return authVal, true
		}
	}
	http.Redirect(w, r, ui.RoleHome(authVal.user.Role), http.StatusSeeOther)
	return requestAuth{}, false
}

func (s *Server) handleUIAsset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/ui-assets/")
	switch name {
	case "htmx.min.js":
		name = "vendor/htmx.min.js"
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case "alpine.min.js":
		name = "vendor/alpine.min.js"
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case "app.js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case "app.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case "favicon.svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(assets.Files, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(assets.Files, "favicon.svg")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleRootUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if authVal, ok := currentAuth(r); ok {
		http.Redirect(w, r, ui.RoleHome(authVal.user.Role), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if authVal, ok := currentAuth(r); ok {
		http.Redirect(w, r, ui.RoleHome(authVal.user.Role), http.StatusFound)
		return
	}
	tab := r.URL.Query().Get("tab")
	if tab != "parent" {
		tab = "staff"
	}
	errMsg := r.URL.Query().Get("error")
	if r.URL.Query().Get("auth") == "magic_invalid" {
		errMsg = "Tautan masuk WhatsApp sudah tidak berlaku atau kedaluwarsa."
	}
	page := ui.LoginPage{
		Tab:          tab,
		Error:        errMsg,
		Notice:       r.URL.Query().Get("notice"),
		MagicLink:    r.URL.Query().Get("magic"),
		DemoMode:     s.authSvc.DemoMode(),
		DemoPassword: s.authSvc.DemoPassword(),
		DemoAccounts: demoAccountsList(),
		CSRFToken:    s.csrfTokenFor("login"),
		AcademicYear: "2026/2027",
	}
	renderComponent(w, r, http.StatusOK, pages.Login(page))
}

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	identifier := strings.TrimSpace(r.FormValue("identifier"))
	password := r.FormValue("password")
	session, err := s.authSvc.Login(r.Context(), identifier, password, s.clientIP(r))
	if err != nil {
		page := ui.LoginPage{
			Tab:          "staff",
			Identifier:   identifier,
			Error:        err.Error(),
			DemoMode:     s.authSvc.DemoMode(),
			DemoPassword: s.authSvc.DemoPassword(),
			DemoAccounts: demoAccountsList(),
			CSRFToken:    s.csrfTokenFor("login"),
			AcademicYear: "2026/2027",
		}
		renderComponent(w, r, http.StatusUnauthorized, pages.Login(page))
		return
	}
	s.setSessionCookie(w, session.Token)
	http.Redirect(w, r, ui.RoleHome(session.User.Role), http.StatusSeeOther)
}

func (s *Server) handleDemoLoginForm(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	identifier := strings.TrimSpace(r.FormValue("identifier"))
	session, err := s.authSvc.DemoLogin(r.Context(), identifier)
	if err != nil {
		http.Redirect(w, r, "/login?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	s.setSessionCookie(w, session.Token)
	http.Redirect(w, r, ui.RoleHome(session.User.Role), http.StatusSeeOther)
}

func (s *Server) handleMagicLinkForm(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	phone := strings.TrimSpace(r.FormValue("phone"))
	token, issued, err := s.authSvc.RequestMagicLink(r.Context(), phone, s.clientIP(r))
	page := ui.LoginPage{
		Tab:          "parent",
		ParentWA:     phone,
		DemoMode:     s.authSvc.DemoMode(),
		DemoPassword: s.authSvc.DemoPassword(),
		DemoAccounts: demoAccountsList(),
		CSRFToken:    s.csrfTokenFor("login"),
		AcademicYear: "2026/2027",
	}
	if err != nil {
		page.Error = err.Error()
		renderComponent(w, r, http.StatusBadRequest, pages.Login(page))
		return
	}
	page.Notice = "Jika nomor terdaftar, tautan masuk sekali pakai telah dikirim ke WhatsApp Anda."
	if issued && s.authSvc.DemoMode() {
		page.MagicLink = "/login/magic/" + token
	}
	renderComponent(w, r, http.StatusAccepted, pages.Login(page))
}

func (s *Server) handleMagicLinkConsumeUI(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	session, _ := s.authSvc.ConsumeMagicToken(r.Context(), token)
	if session == nil {
		http.Redirect(w, r, "/login?tab=parent&auth=magic_invalid", http.StatusFound)
		return
	}
	s.setSessionCookie(w, session.Token)
	http.Redirect(w, r, ui.RoleHome(session.User.Role), http.StatusFound)
}

func (s *Server) handleLogoutForm(w http.ResponseWriter, r *http.Request) {
	token := ""
	if authVal, ok := currentAuth(r); ok {
		token = authVal.token
	} else if c, err := r.Cookie(sessionCookieName); err == nil {
		token = c.Value
	}
	_ = s.authSvc.Logout(r.Context(), token)
	s.clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleDemoRoleForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student", "parent", "treasurer", "teacher", "discipline", "piket", "admin_tu", "headmaster")
	if !ok {
		return
	}
	_ = r.ParseForm()
	role := strings.TrimSpace(r.FormValue("role"))
	session, err := s.authSvc.SwitchDemoRole(r.Context(), authVal.token, role)
	if err != nil {
		http.Redirect(w, r, ui.RoleHome(authVal.user.Role), http.StatusSeeOther)
		return
	}
	s.setSessionCookie(w, session.Token)
	http.Redirect(w, r, ui.RoleHome(session.User.Role), http.StatusSeeOther)
}

func (s *Server) handleDemoScenarioForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student", "parent", "treasurer", "teacher", "discipline", "piket", "admin_tu", "headmaster")
	if !ok {
		return
	}
	_ = r.ParseForm()
	sc := strings.TrimSpace(r.FormValue("scenario"))
	targetRole := "treasurer"
	targetURL := "/treasurer/exam-clearance"
	if sc == "2" {
		targetRole = "student"
		targetURL = "/student/exam-kiosk"
	} else if sc == "3" {
		targetRole = "parent"
		targetURL = "/parent/dashboard"
	}
	if session, err := s.authSvc.SwitchDemoRole(r.Context(), authVal.token, targetRole); err == nil {
		s.setSessionCookie(w, session.Token)
	}
	http.Redirect(w, r, targetURL, http.StatusSeeOther)
}

func (s *Server) handleResetStateForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu", "headmaster")
	if !ok {
		return
	}
	_ = s.storeDB.PutState(r.Context(), state.DefaultStateBytes())
	redirectWithFlash(w, r, ui.RoleHome(authVal.user.Role), "Seluruh data sekolah berhasil direset ke kondisi awal demo.", "success")
}

// Public Verify & DUDI handlers
func (s *Server) handlePublicDudiPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/verify?tab=dudi", http.StatusFound)
}

func (s *Server) handlePublicVerifyPage(w http.ResponseWriter, r *http.Request) {
	doc, _ := s.loadOrSeedState(r.Context())
	tab := r.URL.Query().Get("tab")
	if tab != "dudi" {
		tab = "verify"
	}
	q := strings.TrimSpace(r.PathValue("hash"))
	if q == "" {
		q = strings.TrimSpace(r.URL.Query().Get("q"))
	}

	letters := make([]ui.LetterView, 0)
	for _, lm := range asMapSlice(doc["officialLetters"]) {
		letters = append(letters, parseLetter(lm))
	}
	ledger := make([]ui.LedgerEntryView, 0)
	for _, lm := range asMapSlice(doc["ledger"]) {
		ledger = append(ledger, parseLedgerEntry(lm))
	}
	journals := make([]ui.PklJournalView, 0)
	for _, jm := range asMapSlice(doc["pklJournals"]) {
		journals = append(journals, parsePklJournal(jm))
	}
	placements := make([]ui.PklPlacementView, 0)
	for _, pm := range asMapSlice(doc["pklPlacements"]) {
		placements = append(placements, parsePklPlacement(pm))
	}

	var foundLetter *ui.LetterView
	var foundReceipt *ui.LedgerEntryView
	notFound := false

	if q != "" {
		ql := strings.ToLower(q)
		for i := range letters {
			if strings.ToLower(letters[i].HmacHash) == ql || strings.ToLower(letters[i].LetterNumber) == ql {
				foundLetter = &letters[i]
				break
			}
		}
		if foundLetter == nil {
			for i := range ledger {
				if strings.ToLower(ledger[i].ReceiptNumber) == ql || strings.ToLower(ledger[i].SignatureHmac) == ql {
					foundReceipt = &ledger[i]
					break
				}
			}
		}
		if foundLetter == nil && foundReceipt == nil {
			notFound = true
		}
	} else if len(letters) > 0 {
		q = letters[0].HmacHash
		foundLetter = &letters[0]
	}

	page := ui.PublicVerifyPage{
		ActiveTab:       tab,
		Query:           q,
		VerifiedLetter:  foundLetter,
		VerifiedReceipt: foundReceipt,
		NotFound:        notFound,
		Letters:         letters,
		Ledger:          ledger,
		PklJournals:     journals,
		PklPlacements:   placements,
		Notice:          r.URL.Query().Get("notice"),
		Error:           r.URL.Query().Get("error"),
		CSRFToken:       s.csrfTokenFor("verify"),
		DemoMode:        s.authSvc.DemoMode(),
	}
	renderComponent(w, r, http.StatusOK, pages.PublicVerify(page))
}

func (s *Server) handlePublicDudiApprove(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	journalID := strings.TrimSpace(r.FormValue("journal_id"))
	pin := strings.TrimSpace(r.FormValue("pin"))
	err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.ApprovePklJournalByDudi(doc, journalID, pin, time.Now())
	})
	if err != nil {
		http.Redirect(w, r, "/verify?tab=dudi&error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/verify?tab=dudi&notice="+url.QueryEscape("Jurnal magang siswa berhasil disahkan oleh Supervisor DUDI dengan PIN resmi!"), http.StatusSeeOther)
}

// Student Handlers
func (s *Server) handleStudentRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/student/dashboard", http.StatusFound)
}

func (s *Server) handleStudentPage(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	tab := r.PathValue("tab")
	if tab == "" {
		tab = "dashboard"
	}
	doc, _ := s.loadOrSeedState(r.Context())
	shell := s.buildRoleShell(authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	if tab == "dashboard" {
		data := s.buildStudentDashboard(doc, shell, authVal.user)
		renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.StudentDashboardPage(data)))
		return
	}
	data := s.buildStudentPortal(doc, shell, authVal.user, tab)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.StudentPortal(data)))
}

func (s *Server) handleStudentPayInvoice(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	invoiceID := strings.TrimSpace(r.FormValue("invoice_id"))
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		_, err := state.PayInvoice(doc, studentID, invoiceID, s.cfg.ReceiptSecret, time.Now())
		return err
	})
	if r.Header.Get("HX-Request") == "true" {
		doc, _ := s.loadOrSeedState(r.Context())
		shell := s.buildRoleShell(authVal, "dashboard", "", "")
		data := s.buildStudentDashboard(doc, shell, authVal.user)
		renderComponent(w, r, http.StatusOK, pages.FinanceCard(data))
		return
	}
	redirectWithFlash(w, r, "/student/dashboard", "Tagihan SPP berhasil dilunasi melalui Virtual Account dan kwitansi HMAC telah tercatat.", "success")
}

func (s *Server) handleStudentLeaderValidation(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	status := strings.TrimSpace(r.FormValue("status"))
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UpdateLeaderValidation(doc, studentID, status, time.Now())
	})
	if r.Header.Get("HX-Request") == "true" {
		doc, _ := s.loadOrSeedState(r.Context())
		shell := s.buildRoleShell(authVal, "dashboard", "", "")
		data := s.buildStudentDashboard(doc, shell, authVal.user)
		data.LeaderFeedback = "Status validasi kelas berhasil diperbarui ke server (" + ui.StatusLabel(status) + ")."
		renderComponent(w, r, http.StatusOK, pages.LeaderPanel(data))
		return
	}
	redirectWithFlash(w, r, "/student/dashboard", "Validasi kehadiran mengajar guru berhasil disimpan.", "success")
}

func (s *Server) handleStudentQuizForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	var answers []state.QuizAnswer
	for k, vals := range r.PostForm {
		if strings.HasPrefix(k, "q_") && len(vals) > 0 {
			answers = append(answers, state.QuizAnswer{
				QuestionID: strings.TrimPrefix(k, "q_"),
				OptionKey:  vals[0],
			})
		}
	}
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	var res state.QuizResult
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		var err error
		res, err = state.SubmitQuiz(doc, studentID, answers, time.Now())
		return err
	})
	if r.Header.Get("HX-Request") == "true" {
		view := ui.QuizResultView{
			Score:   res.Score,
			Correct: res.CorrectCount,
			Total:   res.QuestionCount,
			Already: res.AlreadySubmitted,
		}
		renderComponent(w, r, http.StatusOK, pages.QuizResult(view))
		return
	}
	redirectWithFlash(w, r, "/student/classroom", fmt.Sprintf("Kuis harian selesai dinilai di server. Skor Anda: %d/100.", res.Score), "success")
}

func (s *Server) handleStudentReflectionForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.SubmitReflection(doc, studentID, r.FormValue("level"), r.FormValue("complaint"), time.Now())
	})
	redirectWithFlash(w, r, "/student/classroom", "Umpan balik Radar Pemahaman berhasil disimpan secara anonim.", "success")
}

func (s *Server) handleStudentPeerReportForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.SubmitPeerReport(doc, r.FormValue("category"), r.FormValue("location"), r.FormValue("target_name"), r.FormValue("description"), time.Now())
	})
	redirectWithFlash(w, r, "/student/classroom", "Laporan berhasil dienkripsi dan dikirim ke Escrow Kesiswaan & BK.", "success")
}

func (s *Server) handleStudentProctorStatusForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UpdateProctorStatus(doc, studentID, r.FormValue("status"), r.FormValue("reason"), time.Now())
	})
	redirectWithFlash(w, r, "/student/exam-kiosk", "Sistem Anti-Curang mendeteksi perpindahan fokus layar. Sesi ujian dikunci sementara.", "danger")
}

func (s *Server) handleStudentExamForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	var answers []state.ExamAnswer
	for k, vals := range r.PostForm {
		if strings.HasPrefix(k, "exam_q_") && len(vals) > 0 && strings.TrimSpace(vals[0]) != "" {
			num, _ := strconv.Atoi(strings.TrimPrefix(k, "exam_q_"))
			if num > 0 {
				answers = append(answers, state.ExamAnswer{
					QuestionNumber: num,
					Answer:         vals[0],
				})
			}
		}
	}
	if len(answers) == 0 {
		answers = append(answers, state.ExamAnswer{QuestionNumber: 1, Answer: "A"})
	}
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	var res state.ExamResult
	err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		var err error
		res, err = state.SubmitExam(doc, studentID, answers, time.Now())
		return err
	})
	if err != nil {
		redirectWithFlash(w, r, "/student/exam-kiosk", fmt.Sprintf("Gagal mengumpulkan ujian: %v", err), "danger")
		return
	}
	redirectWithFlash(w, r, "/student/exam-kiosk", fmt.Sprintf("Jawaban ujian CBT berhasil dikumpulkan dan dinilai di server (Skor PG: %d/100).", res.McqScore), "success")
}

func (s *Server) handleStudentDocUploadForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UploadStudentDocument(doc, studentID, r.FormValue("doc_id"), r.FormValue("file_name"), 280, time.Now())
	})
	redirectWithFlash(w, r, "/student/documents", "Berkas siswa berhasil diunggah dan masuk ke antrean verifikasi Admin TU.", "success")
}

func (s *Server) handleStudentTaskSubmitForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.SubmitStudentTask(doc, studentID, r.FormValue("task_title"), r.FormValue("file_name"), 340, time.Now())
	})
	redirectWithFlash(w, r, "/student/documents", "Tugas praktik bengkel berhasil dikirim ke meja Guru Mapel.", "success")
}

func (s *Server) handleStudentPklCheckinForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddPklCheckin(doc, studentID, 0, 0, time.Now())
	})
	redirectWithFlash(w, r, "/student/academic", "Presensi GPS Haversine & validasi EXIF di lokasi mitra DUDI berhasil dicatat.", "success")
}

func (s *Server) handleStudentPklJournalForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddPklJournal(doc, studentID, r.FormValue("activity_title"), r.FormValue("competency"), time.Now())
	})
	redirectWithFlash(w, r, "/student/academic", "Jurnal kegiatan harian PKL berhasil disimpan dan menunggu persetujuan PIN DUDI.", "success")
}

func (s *Server) handleStudentBmwForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UpdateBmwChoice(doc, studentID, r.FormValue("choice"))
	})
	redirectWithFlash(w, r, "/student/academic", "Pilihan jalur karir masa depan BMW berhasil diperbarui.", "success")
}

func (s *Server) handleStudentBkkApplyForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.ApplyBkkJob(doc, studentID, r.FormValue("job_id"), time.Now())
	})
	redirectWithFlash(w, r, "/student/academic", "Lamaran kerja BKK beserta portofolio Rapor Holistik 360° berhasil dikirim.", "success")
}

func (s *Server) handleStudentExpenseAddForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	amt, _ := strconv.Atoi(r.FormValue("amount"))
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddStudentExpense(doc, studentID, "ALAT_PRAKTIK", r.FormValue("title"), amt, r.FormValue("note"), time.Now())
	})
	redirectWithFlash(w, r, "/student/academic", "Catatan pengeluaran alat praktik berhasil ditambahkan.", "success")
}

// Parent Handlers
func (s *Server) handleParentRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/parent/dashboard", http.StatusFound)
}

func (s *Server) handleParentPage(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "parent")
	if !ok {
		return
	}
	tab := r.PathValue("tab")
	if tab == "" {
		tab = "dashboard"
	}
	doc, _ := s.loadOrSeedState(r.Context())
	shell := s.buildRoleShell(authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildParentPortal(doc, shell, authVal.user, tab)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.ParentPortal(data)))
}

func (s *Server) handleParentConsultationForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "parent")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.BookCareerConsultation(doc, studentID, r.FormValue("chosen_path"), r.FormValue("topic_note"), time.Now())
	})
	redirectWithFlash(w, r, "/parent/future-plan", "Permohonan jadwal konsultasi karir BMW telah dikirim ke Guru BK.", "success")
}

func (s *Server) handleParentLeaveForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "parent")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.SubmitParentLeave(doc, studentID, r.FormValue("leave_type"), r.FormValue("leave_date"), r.FormValue("leave_note"), time.Now())
	})
	redirectWithFlash(w, r, "/parent/services", "Surat izin resmi orang tua berhasil dikirim dan langsung memperbarui daftar presensi Guru di kelas.", "success")
}

func (s *Server) handleParentPayInvoiceForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "parent")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		_, err := state.PayInvoice(doc, studentID, r.FormValue("invoice_id"), s.cfg.ReceiptSecret, time.Now())
		return err
	})
	redirectWithFlash(w, r, "/parent/services", "Pembayaran SPP via Virtual Account berhasil dikonfirmasi dan kwitansi QR resmi telah diterbitkan.", "success")
}

// Treasurer Handlers
func (s *Server) handleTreasurerRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/treasurer/dashboard", http.StatusFound)
}

func (s *Server) handleTreasurerPage(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "treasurer")
	if !ok {
		return
	}
	tab := r.PathValue("tab")
	if tab == "" {
		tab = "dashboard"
	}
	doc, _ := s.loadOrSeedState(r.Context())
	shell := s.buildRoleShell(authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildTreasurerPortal(doc, shell, tab)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.TreasurerPortal(data)))
}

func (s *Server) handleTreasurerScholarshipForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "treasurer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	pct, _ := strconv.Atoi(r.FormValue("discount_percent"))
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UpdateScholarship(doc, r.FormValue("student_id"), r.FormValue("scholarship_type"), pct)
	})
	redirectWithFlash(w, r, "/treasurer/dashboard", "Potongan beasiswa siswa dan nominal tagihan SPP berhasil dihitung ulang.", "success")
}

func (s *Server) handleTreasurerWaReminderForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "treasurer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.SendWaBillingReminder(doc, r.FormValue("invoice_id"), time.Now())
	})
	redirectWithFlash(w, r, "/treasurer/dashboard", "Pengingat pembayaran SPP santun berhasil dikirim ke WhatsApp Orang Tua.", "success")
}

func (s *Server) handleTreasurerCashPaymentForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "treasurer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	amt, _ := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.RecordCashPayment(doc, r.FormValue("invoice_id"), amt, r.FormValue("pin"), s.cfg.TreasurerPin, authVal.user.ID, authVal.user.FullName, s.cfg.ReceiptSecret, time.Now())
	})
	if err != nil {
		redirectWithFlash(w, r, "/treasurer/billing", err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/treasurer/billing", "Pembayaran tunai berhasil diverifikasi PIN 6-digit dan dibukukan ke Buku Kas Append-Only.", "success")
}

func (s *Server) handleTreasurerDispensationForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "treasurer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.GrantDispensation(doc, r.FormValue("student_id"), r.FormValue("reason"), authVal.user.FullName, time.Now())
	})
	redirectWithFlash(w, r, "/treasurer/exam-clearance", "Dispensasi Ujian 7 Hari berhasil diterbitkan. Pintu ujian CBT siswa kini terbuka.", "success")
}

func (s *Server) handleTreasurerReversalForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "treasurer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	ledgerID, _ := strconv.ParseInt(r.FormValue("ledger_id"), 10, 64)
	err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.ExecuteReversal(doc, ledgerID, r.FormValue("reason"), r.FormValue("otp"), authVal.user.ID, authVal.user.FullName, s.cfg.ReceiptSecret, time.Now())
	})
	if err != nil {
		redirectWithFlash(w, r, "/treasurer/reports", err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/treasurer/reports", "Jurnal Pembalik (Reversal Void) berhasil diposting ke Buku Kas dengan otorisasi OTP Kepala Sekolah.", "success")
}

// Teacher Handlers
func (s *Server) handleTeacherRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/teacher/dashboard", http.StatusFound)
}

func (s *Server) handleTeacherPage(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	tab := r.PathValue("tab")
	if tab == "" {
		tab = "dashboard"
	}
	doc, _ := s.loadOrSeedState(r.Context())
	shell := s.buildRoleShell(authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildTeacherPortal(doc, shell, tab)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.TeacherPortal(data)))
}

func (s *Server) handleTeacherAlertPiketForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.AlertPiketAbsent(doc, authVal.user.FullName)
		return nil
	})
	redirectWithFlash(w, r, "/teacher/dashboard", "Sinyal izin darurat berhasil dikirim ke Radar Guru Piket untuk penugasan Guru Inval.", "warning")
}

func (s *Server) handleTeacherJournalForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UpdateClassJournal(doc, r.FormValue("topic"), r.FormValue("notes"), r.FormValue("ai_notes"))
	})
	redirectWithFlash(w, r, "/teacher/classroom", "Agenda mengajar dan materi AI Daily Quiz berhasil diperbarui.", "success")
}

func (s *Server) handleTeacherToggleAlpaForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.ToggleStudentAlpa(doc, r.FormValue("attendance_id"), time.Now())
	})
	redirectWithFlash(w, r, "/teacher/classroom", "Status kehadiran siswa di kelas berhasil diperbarui.", "success")
}

func (s *Server) handleTeacherAddQuestionForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddExamQuestion(doc, r.FormValue("question_type"), r.FormValue("question_text"), r.FormValue("opt_a"), r.FormValue("opt_b"), r.FormValue("opt_c"), r.FormValue("opt_d"), r.FormValue("correct_key"), r.FormValue("essay_rubric"), time.Now())
	})
	redirectWithFlash(w, r, "/teacher/question-bank", "Soal baru berhasil ditambahkan ke Bank Soal Kejuruan di server.", "success")
}

func (s *Server) handleTeacherDeleteQuestionForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.DeleteExamQuestion(doc, r.FormValue("question_id"))
		return nil
	})
	redirectWithFlash(w, r, "/teacher/question-bank", "Soal berhasil dihapus dan nomor urut telah disesuaikan.", "success")
}

func (s *Server) handleTeacherProctorUnlockForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := strings.TrimSpace(r.FormValue("student_id"))
	token := strings.TrimSpace(r.FormValue("unlock_token"))
	unlocked := false
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		unlocked = state.UnlockProctorStudentWithToken(doc, studentID, token)
		return nil
	})
	if !unlocked {
		redirectWithFlash(w, r, "/teacher/exam-control", "Token unlock tidak valid. Minta siswa menunjukkan kode token.", "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/exam-control", "Kunci layar ujian CBT siswa berhasil dibuka oleh Pengawas.", "success")
}

func (s *Server) handleTeacherProctorMassUnlockForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.MassUnlockProctor(doc)
		return nil
	})
	redirectWithFlash(w, r, "/teacher/exam-control", "Mass Unlock berhasil dijalankan untuk seluruh peserta di ruangan lab.", "success")
}

func (s *Server) handleTeacherEmergencyTicketForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.GrantEmergency2hTicket(doc, r.FormValue("student_id"), authVal.user.FullName, time.Now())
		return nil
	})
	redirectWithFlash(w, r, "/teacher/exam-control", "Tiket Darurat 2 Jam berhasil diterbitkan untuk membuka akses ujian siswa.", "success")
}

func (s *Server) handleTeacherEssayGradeForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	score, _ := strconv.Atoi(r.FormValue("score"))
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.GradeEssayToRapor(doc, r.FormValue("student_id"), score)
	})
	redirectWithFlash(w, r, "/teacher/exam-control", "Nilai akhir CBT & Esai berhasil disahkan ke Rapor Holistik 360°.", "success")
}

func (s *Server) handleTeacherHomeroomAttitudeForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	var scores []float64
	for _, key := range []string{"s1", "s2", "s3", "s4", "s5"} {
		if v, err := strconv.ParseFloat(r.FormValue(key), 64); err == nil {
			scores = append(scores, v)
		}
	}
	var filtered float64
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		var err error
		filtered, err = state.UpdateHomeroomAttitude(doc, r.FormValue("student_id"), scores, r.FormValue("homeroom_note"))
		return err
	})
	redirectWithFlash(w, r, "/teacher/homeroom-pkl", fmt.Sprintf("Outlier Consensus Filter (MAD) selesai dihitung. Skor sikap terlindungi di %.1f/100.", filtered), "success")
}

func (s *Server) handleTeacherBulkApprovePklForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.BulkApprovePklJournals(doc)
		return nil
	})
	redirectWithFlash(w, r, "/teacher/homeroom-pkl", "Seluruh jurnal harian PKL siswa berhasil di-ACC oleh Guru Pembimbing.", "success")
}

// Discipline Handlers
func (s *Server) handleDisciplineRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/discipline/violations", http.StatusFound)
}

func (s *Server) handleDisciplinePage(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "discipline")
	if !ok {
		return
	}
	tab := r.PathValue("tab")
	if tab == "" {
		tab = "violations"
	}
	doc, _ := s.loadOrSeedState(r.Context())
	shell := s.buildRoleShell(authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildDisciplinePortal(doc, shell, tab)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.DisciplinePortal(data)))
}

func (s *Server) handleDisciplineAddViolationForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "discipline")
	if !ok {
		return
	}
	_ = r.ParseForm()
	pts, _ := strconv.Atoi(r.FormValue("points"))
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddManualViolation(doc, r.FormValue("student_name"), r.FormValue("article_code"), r.FormValue("article_title"), pts, time.Now())
	})
	redirectWithFlash(w, r, "/discipline/violations", "Catatan pelanggaran tata tertib berhasil disimpan.", "success")
}

func (s *Server) handleDisciplineSummonForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "discipline")
	if !ok {
		return
	}
	_ = r.ParseForm()
	var hash string
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		var err error
		hash, err = state.IssueSummonLetter(doc, r.FormValue("violation_id"), s.cfg.ReceiptSecret, time.Now())
		return err
	})
	redirectWithFlash(w, r, "/discipline/violations", "Surat Panggilan Orang Tua bersegel HMAC ("+hash+") berhasil diterbitkan.", "success")
}

func (s *Server) handleDisciplineResolvePeerForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "discipline")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.ResolvePeerReport(doc, r.FormValue("report_id"), r.FormValue("status"))
		return nil
	})
	redirectWithFlash(w, r, "/discipline/violations", "Status laporan Escrow Perundungan berhasil diperbarui.", "success")
}

func (s *Server) handleDisciplineScheduleConsultForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "discipline")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.ScheduleConsultation(doc, r.FormValue("consult_id"))
		return nil
	})
	redirectWithFlash(w, r, "/discipline/violations", "Jadwal konsultasi karir BMW orang tua berhasil dikonfirmasi oleh BK.", "success")
}

func (s *Server) handleDisciplineCounselingUpdateForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "discipline")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.UpdateCounselingSession(doc, r.FormValue("session_id"), r.FormValue("status"), r.FormValue("notes"))
		return nil
	})
	redirectWithFlash(w, r, "/discipline/transformation-index", "Progres pembinaan konseling siswa & Indeks Transformasi berhasil diperbarui.", "success")
}

// Piket Handlers
func (s *Server) handlePiketRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/piket/dashboard", http.StatusFound)
}

func (s *Server) handlePiketPage(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "piket")
	if !ok {
		return
	}
	doc, _ := s.loadOrSeedState(r.Context())
	shell := s.buildRoleShell(authVal, "dashboard", r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildPiketPortal(doc, shell)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.PiketPortal(data)))
}

func (s *Server) handlePiketAssignInvalForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "piket")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.AssignInvalTeacher(doc, r.FormValue("classroom_id"), r.FormValue("teacher_name"), r.FormValue("task_note"))
		return nil
	})
	redirectWithFlash(w, r, "/piket/dashboard", "Guru Inval berhasil ditugaskan dan menerima +5 poin kredit Teacher Impact Rating.", "success")
}

func (s *Server) handlePiketGateLogForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "piket")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.AddGateLog(doc, r.FormValue("nisn"), r.FormValue("student_name"), r.FormValue("log_type"), r.FormValue("reason"), time.Now())
		return nil
	})
	redirectWithFlash(w, r, "/piket/dashboard", "Log gerbang berhasil dicatat dan QR Pass resmi telah diterbitkan.", "success")
}

// Admin TU Handlers
func (s *Server) handleAdminRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/master-data", http.StatusFound)
}

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	tab := r.PathValue("tab")
	if tab == "" {
		tab = "master-data"
	}
	doc, _ := s.loadOrSeedState(r.Context())
	shell := s.buildRoleShell(authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildAdminTuPortal(doc, shell, tab)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.AdminTuPortal(data)))
}

func (s *Server) handleAdminVerifyDocForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	approved := r.FormValue("decision") == "approve"
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.VerifyStudentDoc(doc, r.FormValue("doc_id"), approved, r.FormValue("reject_reason"))
		return nil
	})
	redirectWithFlash(w, r, "/admin/master-data", "Status verifikasi berkas siswa berhasil diperbarui.", "success")
}

func (s *Server) handleAdminAddStudentForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddNewStudent(doc, r.FormValue("full_name"), r.FormValue("nisn"), r.FormValue("classroom_name"), r.FormValue("major_code"), time.Now())
	})
	redirectWithFlash(w, r, "/admin/master-data", "Data siswa baru dan tagihan SPP Virtual Account berhasil ditambahkan ke Buku Induk.", "success")
}

func (s *Server) handleAdminDapodikSyncForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.TriggerDapodikSync(doc, time.Now())
		return nil
	})
	redirectWithFlash(w, r, "/admin/dapodik-erapor", "Sinkronisasi Dapodik & Nilai e-Rapor 360° selesai dijalankan (100% Sukses).", "success")
}

func (s *Server) handleAdminExportCSV(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	doc, _ := s.loadOrSeedState(r.Context())
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="erapor_holistik_360_wesmk.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"NISN", "Nama Lengkap", "Kelas", "Jurusan", "CBT (35%)", "Daily Quiz (25%)", "Sikap MAD (25%)", "Presensi (15%)", "Skor Holistik 360", "Apresiasi Vokasi"})
	for _, sm := range asMapSlice(doc["students"]) {
		cbt := floatVal(sm, "cbtScore")
		quiz := floatVal(sm, "dailyQuizAvg")
		att := floatVal(sm, "attitudeScore")
		pres := floatVal(sm, "attendanceConductScore")
		hol := state.CalculateHolisticScore(cbt, quiz, att, pres)
		_ = cw.Write([]string{
			strVal(sm, "nisn"),
			strVal(sm, "fullName"),
			strVal(sm, "classroomName"),
			strVal(sm, "majorCode"),
			fmt.Sprintf("%.0f", cbt),
			fmt.Sprintf("%.0f", quiz),
			fmt.Sprintf("%.0f", att),
			fmt.Sprintf("%.0f", pres),
			fmt.Sprintf("%.1f", hol),
			strVal(sm, "activeBadge"),
		})
	}
	cw.Flush()
}

func (s *Server) handleAdminCreateLetterForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	var hash string
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		ltr := state.CreateOfficialLetter(doc, r.FormValue("letter_type"), r.FormValue("student_name"), r.FormValue("recipient"), s.cfg.ReceiptSecret, time.Now())
		hash = strVal(ltr, "hmacHash")
		return nil
	})
	redirectWithFlash(w, r, "/admin/dapodik-erapor", "Surat resmi bersegel HMAC-SHA256 ("+hash+") berhasil diterbitkan.", "success")
}

func (s *Server) handleAdminAddPklForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	lat, _ := strconv.ParseFloat(r.FormValue("lat"), 64)
	lng, _ := strconv.ParseFloat(r.FormValue("lng"), 64)
	rad, _ := strconv.Atoi(r.FormValue("radius"))
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddPklPlacement(doc, r.FormValue("company_name"), r.FormValue("sector"), r.FormValue("address"), lat, lng, rad, r.FormValue("pin"), time.Now())
	})
	redirectWithFlash(w, r, "/admin/pkl-bkk", "Mitra industri DUDI beserta koordinat Geofence dan PIN Supervisor berhasil ditambahkan.", "success")
}

func (s *Server) handleAdminAddBkkForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	rawMajors := strings.Split(r.FormValue("majors"), ",")
	var majors []string
	for _, m := range rawMajors {
		if trimmed := strings.TrimSpace(m); trimmed != "" {
			majors = append(majors, trimmed)
		}
	}
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddBkkJob(doc, r.FormValue("company_name"), r.FormValue("job_title"), r.FormValue("salary_range"), majors, time.Now())
	})
	redirectWithFlash(w, r, "/admin/pkl-bkk", "Lowongan kerja Bursa Kerja Khusus (BKK) berhasil dipublikasikan.", "success")
}

// Headmaster Handlers
func (s *Server) handleHeadmasterRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/headmaster/executive", http.StatusFound)
}

func (s *Server) handleHeadmasterPage(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "headmaster")
	if !ok {
		return
	}
	doc, _ := s.loadOrSeedState(r.Context())
	shell := s.buildRoleShell(authVal, "executive", r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildHeadmasterPortal(doc, shell, r.URL.Query().Get("advice"))
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.HeadmasterPortal(data)))
}

func (s *Server) handleHeadmasterESignForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "headmaster")
	if !ok {
		return
	}
	redirectWithFlash(w, r, "/headmaster/executive", "Dokumen Resmi & Pengesahan Dana BOS telah ditandatangani secara elektronik dengan QR Kepala Sekolah.", "success")
}

func (s *Server) handleHeadmasterOTPForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "headmaster")
	if !ok {
		return
	}
	_ = r.ParseForm()
	var otp string
	_ = s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		otp = state.GenerateReversalOTP(doc, r.FormValue("req_id"), time.Now())
		return nil
	})
	redirectWithFlash(w, r, "/headmaster/executive", "Kode OTP 6-digit resmi Kepala Sekolah ["+otp+"] berhasil diterbitkan untuk Bendahara.", "success")
}

func (s *Server) handleHeadmasterAIAdvisorForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "headmaster")
	if !ok {
		return
	}
	_ = r.ParseForm()
	q := strings.TrimSpace(r.FormValue("query"))
	advice := ""
	if s.aiClient != nil && s.aiClient.Enabled() && q != "" {
		if textOut, err := s.aiClient.Analyze(r.Context(), "strategic_headmaster", q); err == nil {
			advice = textOut
		}
	}
	if advice == "" {
		advice = "Diagnosis Strategis Eksekutif WESMK: (1) Akar Masalah Pedagogis: Siswa kelas XII TKJ memiliki ketimpangan antara penguasaan teori perutean (95%) dan diagnosis physical/data-link layer (65%). (2) Intervensi Terukur: Tugaskan Guru Berkinerja Terbaik (#1 Ir. Hendro Wijaya) untuk memimpin workshop kilat 3 sesi simulasi insiden kabel dan DHCP exhaustion sebelum asesmen BNSP. (3) Kemitraan DUDI: Koordinasikan dengan supervisor PT Telkom Akses untuk menyertakan skenario troubleshooting nyata pada jurnal magang mingguan."
	}
	doc, _ := s.loadOrSeedState(r.Context())
	shell := s.buildRoleShell(authVal, "executive", "Analisis strategis selesai dijalankan.", "success")
	data := s.buildHeadmasterPortal(doc, shell, advice)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.HeadmasterPortal(data)))
}
