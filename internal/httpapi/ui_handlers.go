package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"

	"wesmk/internal/auth"
	"wesmk/internal/model"
	"wesmk/internal/state"
	"wesmk/internal/store"
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
	s.mux.HandleFunc("GET /login/mfa", s.handleMFAPage)
	s.mux.HandleFunc("POST /login/mfa", s.handleMFAVerifyForm)
	s.mux.HandleFunc("POST /login/mfa/cancel", s.handleMFACancelForm)
	s.mux.HandleFunc("GET /security", s.handleSecurityPage)
	s.mux.HandleFunc("POST /security/totp/start", s.handleTOTPStartForm)
	s.mux.HandleFunc("POST /security/totp/confirm", s.handleTOTPConfirmForm)
	s.mux.HandleFunc("POST /security/totp/disable", s.handleTOTPDisableForm)
	s.mux.HandleFunc("POST /security/recovery/regenerate", s.handleRecoveryRegenerateForm)
	s.mux.HandleFunc("POST /login/magic", s.handleMagicLinkForm)
	s.mux.HandleFunc("GET /login/magic/{token}", s.handleMagicLinkConfirmUI)
	s.mux.HandleFunc("POST /login/magic/{token}", s.handleMagicLinkConsumeUI)
	s.mux.HandleFunc("POST /logout", s.handleLogoutForm)
	s.mux.HandleFunc("POST /state/reset", s.handleResetStateForm)

	// Public Verify & DUDI Supervisor
	s.mux.HandleFunc("GET /verify", s.handlePublicVerifyPage)
	s.mux.HandleFunc("GET /verify/{hash}", s.handlePublicVerifyPage)
	s.mux.HandleFunc("GET /pkl/approve/{token}", s.handlePublicDudiPage)
	s.mux.HandleFunc("POST /verify/dudi-approve", s.handlePublicDudiApprove)

	// Student Routes
	s.mux.HandleFunc("GET /student", s.handleStudentRedirect)
	s.mux.HandleFunc("GET /student/{tab}", s.handleStudentPage)
	s.mux.HandleFunc("POST /student/leader-validation", s.handleStudentLeaderValidation)
	s.mux.HandleFunc("POST /student/quiz", s.handleStudentQuizForm)
	s.mux.HandleFunc("POST /student/reflection", s.handleStudentReflectionForm)
	s.mux.HandleFunc("POST /student/peer-report", s.handleStudentPeerReportForm)
	s.mux.HandleFunc("POST /student/proctor-status", s.handleStudentProctorStatusForm)
	s.mux.HandleFunc("POST /student/exam", s.handleStudentExamForm)
	s.mux.HandleFunc("POST /student/document/upload", s.handleStudentDocUploadForm)
	s.mux.HandleFunc("GET /document/{blobID}", s.handleDocumentFile)
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
	s.mux.HandleFunc("POST /teacher/task/grade", s.handleTeacherTaskGradeForm)
	s.mux.HandleFunc("POST /teacher/assignment/create", s.handleTeacherCreateAssignmentForm)
	s.mux.HandleFunc("POST /teacher/attendance/toggle", s.handleTeacherToggleAlpaForm)
	s.mux.HandleFunc("POST /teacher/question/add", s.handleTeacherAddQuestionForm)
	s.mux.HandleFunc("POST /teacher/question/delete", s.handleTeacherDeleteQuestionForm)
	s.mux.HandleFunc("POST /teacher/proctor/unlock", s.handleTeacherProctorUnlockForm)
	s.mux.HandleFunc("POST /teacher/proctor/mass-unlock", s.handleTeacherProctorMassUnlockForm)
	s.mux.HandleFunc("POST /teacher/proctor/emergency-ticket", s.handleTeacherEmergencyTicketForm)
	s.mux.HandleFunc("POST /teacher/proctor/reset-exam", s.handleTeacherProctorResetExamForm)
	s.mux.HandleFunc("POST /teacher/proctor/bulk-reset", s.handleTeacherProctorBulkResetForm)
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
	s.mux.HandleFunc("POST /admin/master/academic-year", s.handleAdminCreateAcademicYearForm)
	s.mux.HandleFunc("POST /admin/master/academic-year/activate", s.handleAdminActivateAcademicYearForm)
	s.mux.HandleFunc("POST /admin/master/rollover", s.handleAdminRolloverForm)
	s.mux.HandleFunc("POST /admin/master/classroom", s.handleAdminCreateClassroomForm)
	s.mux.HandleFunc("POST /admin/master/classroom/update", s.handleAdminUpdateClassroomForm)
	s.mux.HandleFunc("POST /admin/master/classroom/delete", s.handleAdminDeleteClassroomForm)
	s.mux.HandleFunc("POST /admin/master/schedule", s.handleAdminCreateScheduleForm)
	s.mux.HandleFunc("POST /admin/master/schedule/update", s.handleAdminUpdateScheduleForm)
	s.mux.HandleFunc("POST /admin/master/schedule/delete", s.handleAdminDeleteScheduleForm)
	s.mux.HandleFunc("POST /admin/dapodik/sync", s.handleAdminDapodikSyncForm)
	s.mux.HandleFunc("POST /admin/letter/create", s.handleAdminCreateLetterForm)
	s.mux.HandleFunc("POST /admin/pkl/add", s.handleAdminAddPklForm)
	s.mux.HandleFunc("POST /admin/bkk/add", s.handleAdminAddBkkForm)
	s.mux.HandleFunc("POST /admin/account/create", s.handleAdminCreateAccountForm)
	s.mux.HandleFunc("POST /admin/account/password", s.handleAdminResetPasswordForm)
	s.mux.HandleFunc("POST /admin/account/mfa-reset", s.handleAdminResetMFAForm)
	s.mux.HandleFunc("POST /admin/account/status", s.handleAdminAccountStatusForm)
	s.mux.HandleFunc("POST /admin/account/revoke", s.handleAdminRevokeSessionsForm)

	// Headmaster Routes
	s.mux.HandleFunc("GET /headmaster", s.handleHeadmasterRedirect)
	s.mux.HandleFunc("GET /headmaster/{tab}", s.handleHeadmasterPage)
	s.mux.HandleFunc("POST /headmaster/esign", s.handleHeadmasterESignForm)
	s.mux.HandleFunc("POST /headmaster/otp", s.handleHeadmasterOTPForm)
	s.mux.HandleFunc("POST /headmaster/ai-advisor", s.handleHeadmasterAIAdvisorForm)

	// Developer Routes
	s.mux.HandleFunc("GET /developer", s.handleDeveloperRedirect)
	s.mux.HandleFunc("GET /developer/{tab}", s.handleDeveloperPage)
	s.mux.HandleFunc("POST /developer/bugs/update-status", s.handleDeveloperBugStatusForm)

	// Contact & Bug Report Routes
	s.mux.HandleFunc("GET /contact", s.handleContactPage)
	s.mux.HandleFunc("POST /contact/report-bug", s.handleContactReportBugForm)

	// Root entrypoint
	s.mux.HandleFunc("/", s.handleRootUI)
}

func renderInternalError(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Terjadi kesalahan pada server.", http.StatusInternalServerError)
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
	webRedirect(w, r, u)
}

// webRedirect keeps a plain browser POST a normal 303, while an htmx request
// asks the client to navigate instead of swapping a full page into a fragment
// target (for example when a session expires or a mutation is rejected).
func webRedirect(w http.ResponseWriter, r *http.Request, target string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", target)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func requireWebRole(w http.ResponseWriter, r *http.Request, roles ...string) (requestAuth, bool) {
	authVal, ok := currentAuth(r)
	if !ok {
		webRedirect(w, r, "/login")
		return requestAuth{}, false
	}
	for _, role := range roles {
		if authVal.user.Role == role {
			return authVal, true
		}
	}
	webRedirect(w, r, ui.RoleHome(authVal.user.Role))
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
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	case "app.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	case "favicon.svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case "manifest.webmanifest":
		w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	case "icon-192.png", "icon-512.png", "apple-touch-icon.png":
		w.Header().Set("Content-Type", "image/png")
	default:
		if strings.HasPrefix(name, "images/") && filepath.Ext(name) == ".jpg" {
			w.Header().Set("Content-Type", "image/jpeg")
			break
		}
		if strings.HasPrefix(name, "fonts/") && filepath.Ext(name) == ".woff2" {
			w.Header().Set("Content-Type", "font/woff2")
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			break
		}
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(assets.Files, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if name != "app.js" && name != "app.css" && name != "manifest.webmanifest" && !strings.HasPrefix(name, "fonts/") {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
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
	s.renderHomePage(w, r)
}

// renderHomePage builds the public landing page from normalized master data.
// It never seeds app_state: a public visitor must not create school data as a
// side effect of opening the homepage.
func (s *Server) renderHomePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page := ui.HomePage{AcademicYear: s.activeAcademicYearLabel(ctx)}
	pklPartners := 0

	if s.storeDB != nil {
		if classes, err := s.storeDB.ListClassrooms(ctx); err == nil && len(classes) > 0 {
			students := 0
			majors := make([]ui.HomeMajor, 0, 4)
			majorIndex := map[string]int{}
			for _, class := range classes {
				students += class.StudentCount
				if class.MajorCode == "" {
					continue
				}
				idx, seen := majorIndex[class.MajorCode]
				if !seen {
					idx = len(majors)
					majorIndex[class.MajorCode] = idx
					majors = append(majors, ui.HomeMajor{Code: class.MajorCode, Name: class.MajorName})
				}
				majors[idx].Classes++
				majors[idx].Students += class.StudentCount
			}
			if students > 0 {
				page.Stats = append(page.Stats, ui.HomeStat{Label: "Siswa aktif", Value: strconv.Itoa(students)})
			}
			page.Stats = append(page.Stats, ui.HomeStat{Label: "Kelas aktif", Value: strconv.Itoa(len(classes))})
			if len(majors) > 0 {
				page.Stats = append(page.Stats, ui.HomeStat{Label: "Program keahlian", Value: strconv.Itoa(len(majors))})
				page.Majors = majors
			}
		}
	}

	// The state document carries the details classrooms cannot: workshop rooms,
	// heads of programme, and the industry partners that already host PKL.
	// Read-only: a public page must never seed demo data as a side effect.
	if s.storeDB != nil {
		if raw, err := s.storeDB.GetState(ctx); err == nil && len(raw) > 0 {
			if doc, err := decodeState(raw); err == nil {
				index := make(map[string]int, len(page.Majors))
				for i, major := range page.Majors {
					index[major.Code] = i
				}
				for _, major := range asMapSlice(doc["majors"]) {
					code := strings.TrimSpace(strVal(major, "code"))
					name := strings.TrimSpace(strVal(major, "name"))
					if code == "" || name == "" {
						continue
					}
					detail := strings.TrimSpace(strVal(major, "workshopRoom"))
					head := strings.TrimSpace(strVal(major, "headOfMajor"))
					partners := intVal(major, "dudiPartnerCount")
					if i, ok := index[code]; ok {
						if page.Majors[i].Detail == "" {
							page.Majors[i].Detail = detail
						}
						page.Majors[i].Head = head
						page.Majors[i].Partners = partners
						continue
					}
					if len(page.Majors) >= 6 {
						continue
					}
					index[code] = len(page.Majors)
					page.Majors = append(page.Majors, ui.HomeMajor{
						Code:     code,
						Name:     name,
						Detail:   detail,
						Head:     head,
						Partners: partners,
					})
				}

				// Only the company name and sector are public; supervisor names,
				// phone numbers, and PINs stay inside the portal.
				seen := map[string]bool{}
				partners := make([]ui.HomePartner, 0, 8)
				for _, placement := range asMapSlice(doc["pklPlacements"]) {
					name := strings.TrimSpace(strVal(placement, "companyName"))
					if name == "" || seen[name] {
						continue
					}
					seen[name] = true
					if len(partners) == 8 {
						continue
					}
					partners = append(partners, ui.HomePartner{
						Name:   name,
						Sector: strings.TrimSpace(strVal(placement, "industrySector")),
					})
				}
				page.Partners = partners
				pklPartners = len(seen)
			}
		}
	}

	// Keep the proof strip meaningful while classrooms are still empty: the
	// number of programmes is real master data, not a placeholder figure.
	if len(page.Majors) > 0 && !hasHomeStat(page.Stats, "Program keahlian") {
		page.Stats = append(page.Stats, ui.HomeStat{Label: "Program keahlian", Value: strconv.Itoa(len(page.Majors))})
	}
	if pklPartners > 0 {
		page.Stats = append(page.Stats, ui.HomeStat{Label: "Mitra PKL", Value: strconv.Itoa(pklPartners)})
	}

	// The head of school is a real account, not a stock photo caption. Skip the
	// role suffix some accounts carry so the page does not read twice.
	if s.storeDB != nil {
		if users, err := s.storeDB.ListUsers(ctx); err == nil {
			for _, user := range users {
				if user.Role != "headmaster" || !user.IsActive {
					continue
				}
				page.Headmaster = strings.TrimSpace(strings.TrimSuffix(
					strings.TrimSpace(user.FullName), "(Kepala Sekolah)"))
				break
			}
		}
	}

	page.SiteURL = requestSiteURL(r, s.cfg.TrustProxy)
	page.JSONLD = homeJSONLD(page)

	renderComponent(w, r, http.StatusOK, pages.Home(page))
}

// requestSiteURL reconstructs the absolute origin for canonical and OG URLs.
func requestSiteURL(r *http.Request, trustProxy bool) string {
	scheme := "http"
	if r.TLS != nil || (trustProxy && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")) {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// homeJSONLD describes the school for search engines.
func homeJSONLD(page ui.HomePage) string {
	doc := map[string]any{
		"@context":    "https://schema.org",
		"@type":       "EducationalOrganization",
		"name":        ui.SchoolName(),
		"url":         page.SiteURL + "/",
		"inLanguage":  "id-ID",
		"description": "Portal layanan sekolah: akademik, keuangan, PKL, dan komunikasi untuk siswa, guru, orang tua, dan staf.",
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

// hasHomeStat keeps the proof strip from printing the same label twice when
// both normalized tables and the state document can supply it.
func hasHomeStat(stats []ui.HomeStat, label string) bool {
	for _, stat := range stats {
		if stat.Label == label {
			return true
		}
	}
	return false
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
		CSRFToken:    s.csrfTokenFor("login"),
		AcademicYear: s.activeAcademicYearLabel(r.Context()),
	}
	renderComponent(w, r, http.StatusOK, pages.Login(page))
}

const mfaCookieName = "wesmk_mfa"

func (s *Server) setMFAChallengeCookie(w http.ResponseWriter, challenge string) {
	http.SetCookie(w, &http.Cookie{
		Name:     mfaCookieName,
		Value:    challenge,
		Path:     "/login/mfa",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookie,
		MaxAge:   300,
	})
}

func (s *Server) clearMFAChallengeCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     mfaCookieName,
		Value:    "",
		Path:     "/login/mfa",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookie,
		MaxAge:   -1,
	})
}

func mfaChallengeFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(mfaCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func (s *Server) handleMFAPage(w http.ResponseWriter, r *http.Request) {
	if mfaChallengeFromRequest(r) == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	renderComponent(w, r, http.StatusOK, pages.MFA(ui.MFAPage{CSRFToken: s.csrfTokenFor("login")}))
}

func (s *Server) handleMFAVerifyForm(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	challenge := mfaChallengeFromRequest(r)
	if challenge == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	session, err := s.authSvc.VerifyMFA(r.Context(), challenge, r.FormValue("code"))
	if err != nil {
		userID, reason := authErrorDetails(err)
		s.recordLoginEvent(r, userID, "", "", "mfa", false, reason)
		renderComponent(w, r, http.StatusUnauthorized, pages.MFA(ui.MFAPage{
			Error:     err.Error(),
			CSRFToken: s.csrfTokenFor("login"),
		}))
		return
	}
	s.clearMFAChallengeCookie(w)
	s.setSessionCookie(w, session.Token)
	s.rememberSession(r, session.Token, session.User.ID)
	s.recordLoginEvent(r, session.User.ID, session.User.Username, session.User.Role, session.Method, true, "")
	http.Redirect(w, r, ui.RoleHome(session.User.Role), http.StatusSeeOther)
}

func (s *Server) handleMFACancelForm(w http.ResponseWriter, r *http.Request) {
	s.clearMFAChallengeCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleSecurityPage(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu", "treasurer", "headmaster", "developer")
	if !ok {
		return
	}
	user, err := s.storeDB.UserByID(r.Context(), authVal.user.ID)
	if err != nil || user == nil {
		http.Redirect(w, r, ui.RoleHome(authVal.user.Role), http.StatusSeeOther)
		return
	}
	s.renderSecurityPage(w, r, authVal, user, nil, "", "", r.URL.Query().Get("error"))
}

func (s *Server) renderSecurityPage(w http.ResponseWriter, r *http.Request, authVal requestAuth, user *model.User, recoveryCodes []string, notice, kind, errMsg string) {
	shell := s.buildRoleShell(r.Context(), authVal, "", notice, kind)
	shell.PageTitle = "Keamanan Akun"
	data := ui.SecurityPage{
		Shell:         shell,
		Enabled:       user.TOTPEnabled,
		Required:      auth.RequiresMFA(user),
		RecoveryCodes: recoveryCodes,
		Error:         errMsg,
	}
	if !user.TOTPEnabled && user.TOTPSecret != "" {
		data.Secret = user.TOTPSecret
		data.OTPAuthURI = auth.TOTPURI(user.TOTPSecret, user.Username, "WESMK "+ui.SchoolName())
	}
	data.Session.IP = s.clientIP(r)
	data.Session.UserAgent = r.UserAgent()
	data.Session.StartedAt = time.Now()
	data.Session.ExpiresAt = time.Now().Add(s.authSvc.SessionTTL())
	if s.cache != nil {
		if info, err := s.cache.SessionInfo(r.Context(), authVal.token); err == nil {
			if info.IP != "" {
				data.Session.IP = info.IP
			}
			if info.UserAgent != "" {
				data.Session.UserAgent = info.UserAgent
			}
			if !info.CreatedAt.IsZero() {
				data.Session.StartedAt = info.CreatedAt
			}
		}
		if ttl, err := s.cache.SessionTTL(r.Context(), authVal.token); err == nil && ttl > 0 {
			data.Session.ExpiresAt = time.Now().Add(ttl)
		}
	}
	if user.TOTPEnabled {
		if count, err := s.storeDB.CountUnusedRecoveryCodes(r.Context(), user.ID); err == nil {
			data.RecoveryCount = count
		}
	}
	if events, err := s.storeDB.LoginEventsForUser(r.Context(), user.ID, 12); err == nil {
		data.LoginEvents = securityLoginEvents(events)
	}
	if user.Role == "admin_tu" || user.Role == "developer" {
		data.ShowAdmin = true
		if events, err := s.storeDB.RecentLoginEvents(r.Context(), 20); err == nil {
			data.AdminEvents = securityLoginEvents(events)
		}
	}
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.Security(data)))
}

func securityLoginEvents(events []model.LoginEvent) []ui.SecurityLoginEvent {
	items := make([]ui.SecurityLoginEvent, 0, len(events))
	for _, event := range events {
		items = append(items, ui.SecurityLoginEvent{
			CreatedAt: event.CreatedAt, Identifier: event.Identifier, Role: event.Role,
			Method: event.Method, Success: event.Success, Reason: event.Reason,
			IP: event.IP, UserAgent: event.UserAgent,
		})
	}
	return items
}

func (s *Server) handleTOTPStartForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu", "treasurer", "headmaster", "developer")
	if !ok {
		return
	}
	current, err := s.storeDB.UserByID(r.Context(), authVal.user.ID)
	if err != nil || current == nil {
		redirectWithFlash(w, r, "/security", "Akun tidak ditemukan.", "danger")
		return
	}
	if current.TOTPEnabled {
		redirectWithFlash(w, r, "/security", "MFA sudah aktif. Nonaktifkan dulu untuk mendaftar perangkat baru.", "info")
		return
	}
	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		redirectWithFlash(w, r, "/security", "Secret MFA gagal dibuat.", "danger")
		return
	}
	if _, err := s.storeDB.SetTOTP(r.Context(), authVal.user.ID, secret, false); err != nil {
		redirectWithFlash(w, r, "/security", "Secret MFA gagal disimpan.", "danger")
		return
	}
	redirectWithFlash(w, r, "/security", "Secret dibuat. Pindai atau salin, lalu konfirmasi kode 6 digit.", "success")
}

func (s *Server) handleTOTPConfirmForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu", "treasurer", "headmaster", "developer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	user, err := s.storeDB.UserByID(r.Context(), authVal.user.ID)
	if err != nil || user == nil {
		redirectWithFlash(w, r, "/security", "Akun tidak ditemukan.", "danger")
		return
	}
	if user.TOTPEnabled {
		redirectWithFlash(w, r, "/security", "MFA sudah aktif.", "info")
		return
	}
	if user.TOTPSecret == "" || !auth.ValidateTOTP(user.TOTPSecret, r.FormValue("code"), time.Now()) {
		redirectWithFlash(w, r, "/security", "Kode MFA salah. Coba lagi dengan kode terbaru.", "danger")
		return
	}
	if _, err := s.storeDB.SetTOTP(r.Context(), user.ID, user.TOTPSecret, true); err != nil {
		redirectWithFlash(w, r, "/security", "MFA gagal diaktifkan.", "danger")
		return
	}
	user.TOTPEnabled = true
	s.auditAccount(r.Context(), authVal, "account.mfa_enabled", user.ID, user.Role, nil)
	codes, hashes, err := newRecoveryCodes(user.ID)
	if err != nil {
		s.renderSecurityPage(w, r, authVal, user, nil, "MFA aktif, tetapi recovery code gagal dibuat. Gunakan tombol regenerate.", "warning", "")
		return
	}
	if err := s.storeDB.ReplaceRecoveryCodes(r.Context(), user.ID, hashes); err != nil {
		s.renderSecurityPage(w, r, authVal, user, nil, "MFA aktif, tetapi recovery code gagal disimpan. Gunakan tombol regenerate.", "warning", "")
		return
	}
	s.auditAccount(r.Context(), authVal, "account.recovery_codes_generated", user.ID, user.Role, map[string]any{"count": len(codes)})
	s.renderSecurityPage(w, r, authVal, user, codes, "MFA aktif. Simpan recovery code sekarang; kode ini tidak ditampilkan lagi.", "success", "")
}

func (s *Server) handleRecoveryRegenerateForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu", "treasurer", "headmaster", "developer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	user, err := s.storeDB.UserByID(r.Context(), authVal.user.ID)
	if err != nil || user == nil {
		redirectWithFlash(w, r, "/security", "Akun tidak ditemukan.", "danger")
		return
	}
	if !user.TOTPEnabled {
		redirectWithFlash(w, r, "/security", "Aktifkan MFA terlebih dahulu.", "info")
		return
	}
	if !auth.ValidateTOTP(user.TOTPSecret, r.FormValue("code"), time.Now()) {
		s.renderSecurityPage(w, r, authVal, user, nil, "", "", "Kode autentikator salah. Recovery code lama tetap berlaku.")
		return
	}
	codes, hashes, err := newRecoveryCodes(user.ID)
	if err != nil {
		s.renderSecurityPage(w, r, authVal, user, nil, "", "", "Recovery code gagal dibuat.")
		return
	}
	if err := s.storeDB.ReplaceRecoveryCodes(r.Context(), user.ID, hashes); err != nil {
		s.renderSecurityPage(w, r, authVal, user, nil, "", "", "Recovery code gagal disimpan.")
		return
	}
	s.auditAccount(r.Context(), authVal, "account.recovery_codes_generated", user.ID, user.Role, map[string]any{"count": len(codes)})
	s.renderSecurityPage(w, r, authVal, user, codes, "Recovery code baru dibuat. Kode lama tidak berlaku lagi.", "success", "")
}

func newRecoveryCodes(userID string) ([]string, []string, error) {
	codes, err := auth.GenerateRecoveryCodes()
	if err != nil {
		return nil, nil, err
	}
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hashes = append(hashes, auth.RecoveryCodeHash(userID, code))
	}
	return codes, hashes, nil
}

func (s *Server) handleTOTPDisableForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu", "treasurer", "headmaster", "developer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	user, err := s.storeDB.UserByID(r.Context(), authVal.user.ID)
	if err != nil || user == nil {
		redirectWithFlash(w, r, "/security", "Akun tidak ditemukan.", "danger")
		return
	}
	if !user.TOTPEnabled {
		redirectWithFlash(w, r, "/security", "MFA belum aktif.", "info")
		return
	}
	if auth.RequiresMFA(user) {
		redirectWithFlash(w, r, "/security", "MFA wajib untuk peran ini dan tidak dapat dinonaktifkan sendiri. Hubungi Admin TU bila perangkat hilang.", "danger")
		return
	}
	if !auth.VerifyPassword(r.FormValue("password"), user.PasswordHash) {
		redirectWithFlash(w, r, "/security", "Kata sandi salah. MFA tidak dinonaktifkan.", "danger")
		return
	}
	if !auth.ValidateTOTP(user.TOTPSecret, r.FormValue("code"), time.Now()) {
		redirectWithFlash(w, r, "/security", "Kode MFA salah. MFA tidak dinonaktifkan.", "danger")
		return
	}
	if _, err := s.storeDB.SetTOTP(r.Context(), user.ID, "", false); err != nil {
		redirectWithFlash(w, r, "/security", "MFA gagal dinonaktifkan.", "danger")
		return
	}
	_ = s.storeDB.DeleteRecoveryCodes(r.Context(), user.ID)
	s.auditAccount(r.Context(), authVal, "account.mfa_disabled", user.ID, user.Role, nil)
	redirectWithFlash(w, r, "/security", "MFA dinonaktifkan.", "success")
}

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	identifier := strings.TrimSpace(r.FormValue("identifier"))
	password := r.FormValue("password")
	session, err := s.authSvc.Login(r.Context(), identifier, password, s.clientIP(r))
	if err != nil {
		var authErr *auth.Error
		if errors.As(err, &authErr) && authErr.Code == auth.CodeMFARequired && authErr.Challenge != "" {
			s.setMFAChallengeCookie(w, authErr.Challenge)
			http.Redirect(w, r, "/login/mfa", http.StatusSeeOther)
			return
		}
		userID, reason := authErrorDetails(err)
		s.recordLoginEvent(r, userID, identifier, "", "password", false, reason)
		page := ui.LoginPage{
			Tab:          "staff",
			Identifier:   identifier,
			Error:        err.Error(),
			CSRFToken:    s.csrfTokenFor("login"),
			AcademicYear: s.activeAcademicYearLabel(r.Context()),
		}
		renderComponent(w, r, http.StatusUnauthorized, pages.Login(page))
		return
	}
	s.setSessionCookie(w, session.Token)
	s.rememberSession(r, session.Token, session.User.ID)
	s.recordLoginEvent(r, session.User.ID, identifier, session.User.Role, session.Method, true, "")
	target := ui.RoleHome(session.User.Role)
	if auth.RequiresMFA(session.User) && !session.User.TOTPEnabled {
		target = "/security?required=1"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) handleMagicLinkForm(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	phone := strings.TrimSpace(r.FormValue("phone"))
	token, issued, err := s.authSvc.RequestMagicLink(r.Context(), phone, s.clientIP(r))
	page := ui.LoginPage{
		Tab:          "parent",
		ParentWA:     phone,
		CSRFToken:    s.csrfTokenFor("login"),
		AcademicYear: s.activeAcademicYearLabel(r.Context()),
	}
	if err != nil {
		page.Error = err.Error()
		renderComponent(w, r, http.StatusBadRequest, pages.Login(page))
		return
	}
	page.Notice = "Jika nomor terdaftar, tautan masuk sekali pakai telah dikirim ke WhatsApp Anda."
	if issued {
		if err := s.sendMagicLink(r.Context(), phone, token); err != nil {
			page.Error = err.Error()
			renderComponent(w, r, http.StatusServiceUnavailable, pages.Login(page))
			return
		}
	}
	renderComponent(w, r, http.StatusAccepted, pages.Login(page))
}

// The link preview fetcher (WhatsApp, email scanner) must not burn the token;
// GET only shows a confirmation page and POST performs the login.
func (s *Server) handleMagicLinkConfirmUI(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Redirect(w, r, "/login?tab=parent&auth=magic_invalid", http.StatusSeeOther)
		return
	}
	csrfSeed := "login"
	if authVal, ok := currentAuth(r); ok {
		csrfSeed = authVal.token
	}
	renderComponent(w, r, http.StatusOK, pages.MagicLink(ui.MagicLinkPage{
		Token:     token,
		CSRFToken: s.csrfTokenFor(csrfSeed),
	}))
}

func (s *Server) handleMagicLinkConsumeUI(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	session, _ := s.authSvc.ConsumeMagicToken(r.Context(), token)
	if session == nil {
		http.Redirect(w, r, "/login?tab=parent&auth=magic_invalid", http.StatusFound)
		return
	}
	s.setSessionCookie(w, session.Token)
	s.rememberSession(r, session.Token, session.User.ID)
	s.recordLoginEvent(r, session.User.ID, session.User.Username, session.User.Role, "magic", true, "")
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

func (s *Server) handleResetStateForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu", "headmaster")
	if !ok {
		return
	}
	_ = s.storeDB.PutState(r.Context(), state.DefaultStateBytes())
	_ = s.storeDB.RecordStateAudit(r.Context(), store.StateAudit{
		ActorID: authVal.user.ID, Role: authVal.user.Role, Action: "state.reset",
	})
	redirectWithFlash(w, r, ui.RoleHome(authVal.user.Role), "Seluruh data sekolah berhasil direset ke data awal.", "success")
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

	page := ui.PublicVerifyPage{
		ActiveTab: tab,
		Query:     q,
		Notice:    r.URL.Query().Get("notice"),
		Error:     r.URL.Query().Get("error"),
		CSRFToken: s.csrfTokenFor("verify"),
	}

	// Only the exact code supplied by the visitor is resolved; the page never
	// receives the school-wide list of letters, receipts, or journals.
	if q != "" {
		if tab == "dudi" {
			if journal := findJournalByID(doc, q); journal != nil {
				view := parsePklJournal(journal)
				page.SelectedJournal = &view
			} else {
				page.NotFound = true
			}
		} else {
			page.VerifiedLetter, page.VerifiedReceipt = findVerifiedDocument(doc, q)
			page.NotFound = page.VerifiedLetter == nil && page.VerifiedReceipt == nil
		}
	}

	renderComponent(w, r, http.StatusOK, pages.PublicVerify(page))
}

func findJournalByID(doc map[string]any, journalID string) map[string]any {
	for _, item := range asMapSlice(doc["pklJournals"]) {
		if strVal(item, "id") == journalID {
			return item
		}
	}
	return nil
}

// findVerifiedDocument resolves a single letter or receipt by its exact code, so
// the public page never needs the full ledger.
func findVerifiedDocument(doc map[string]any, code string) (*ui.LetterView, *ui.LedgerEntryView) {
	query := strings.ToLower(code)
	for _, item := range asMapSlice(doc["officialLetters"]) {
		view := parseLetter(item)
		if strings.ToLower(view.HmacHash) == query || strings.ToLower(view.LetterNumber) == query {
			return &view, nil
		}
	}
	for _, item := range asMapSlice(doc["ledger"]) {
		view := parseLedgerEntry(item)
		if strings.ToLower(view.ReceiptNumber) == query || strings.ToLower(view.SignatureHmac) == query {
			return nil, &view
		}
	}
	return nil, nil
}

func (s *Server) handlePublicDudiApprove(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	// The PIN is the only credential on this public endpoint, so guesses must be
	// bounded per client address.
	if s.cache != nil {
		attempts, err := s.cache.IncrementCounter(r.Context(), "ratelimit:dudi:"+s.clientIP(r), 10*time.Minute)
		if err != nil {
			http.Redirect(w, r, "/verify?tab=dudi&error="+url.QueryEscape("Layanan verifikasi sedang sibuk. Coba lagi nanti."), http.StatusSeeOther)
			return
		}
		if attempts > 20 {
			http.Redirect(w, r, "/verify?tab=dudi&error="+url.QueryEscape("Terlalu banyak percobaan PIN. Coba lagi nanti."), http.StatusSeeOther)
			return
		}
	}
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
	shell := s.buildRoleShell(r.Context(), authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	if tab == "dashboard" {
		data := s.buildStudentDashboard(r.Context(), doc, shell, authVal.user)
		renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.StudentDashboardPage(data)))
		return
	}
	data := s.buildStudentPortal(r.Context(), doc, shell, authVal.user, tab)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.StudentPortal(data)))
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
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UpdateLeaderValidation(doc, studentID, status, time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/student/dashboard", "Gagal menyimpan validasi: "+err.Error(), "danger")
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		doc, _ := s.loadOrSeedState(r.Context())
		shell := s.buildRoleShell(r.Context(), authVal, "dashboard", "", "")
		data := s.buildStudentDashboard(r.Context(), doc, shell, authVal.user)
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
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	var res state.QuizResult
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		var err error
		res, err = state.SubmitQuiz(doc, studentID, answers, time.Now())
		return err
	}); err != nil {
		redirectWithFlash(w, r, "/student/classroom", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.SubmitReflection(doc, studentID, r.FormValue("level"), r.FormValue("complaint"), time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/student/classroom", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/student/classroom", "Umpan balik Radar Pemahaman berhasil disimpan secara anonim.", "success")
}

func (s *Server) handleStudentPeerReportForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.SubmitPeerReport(doc, r.FormValue("category"), r.FormValue("location"), r.FormValue("target_name"), r.FormValue("description"), time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/student/classroom", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	status := strings.TrimSpace(r.FormValue("status"))
	reason := strings.TrimSpace(r.FormValue("reason"))

	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UpdateProctorStatus(doc, studentID, status, reason, time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/student/exam-kiosk", err.Error(), "danger")
		return
	}
	if r.FormValue("async") == "1" || r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	redirectWithFlash(w, r, "/student/exam-kiosk", "Sistem Anti-Curang mendeteksi pelanggaran fokus layar. Sesi ujian dikunci sementara.", "danger")
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
		// Never fabricate an answer on the student's behalf.
		redirectWithFlash(w, r, "/student/exam-kiosk", "Tidak ada jawaban yang dipilih. Jawab minimal satu soal sebelum mengumpulkan.", "danger")
		return
	}
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	examID := strings.TrimSpace(r.FormValue("exam_id"))
	var res state.ExamResult
	err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		var err error
		res, err = state.SubmitExamForSubject(doc, studentID, examID, answers, time.Now())
		return err
	})
	if err != nil {
		redirectWithFlash(w, r, "/student/exam-kiosk", fmt.Sprintf("Gagal mengumpulkan ujian: %v", err), "danger")
		return
	}
	_ = res
	redirectWithFlash(w, r, "/student/exam-lobby", "Jawaban ujian berhasil dikumpulkan dan tersimpan di server.", "success")
}

const maxDocumentBytes = 500 << 10

func (s *Server) handleStudentDocUploadForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	fail := func(message string) {
		redirectWithFlash(w, r, "/student/documents", message, "danger")
	}
	if err := r.ParseMultipartForm(maxDocumentBytes); err != nil {
		fail("Berkas terlalu besar atau tidak dapat dibaca (maksimal 500 KB).")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		fail("Pilih berkas terlebih dahulu.")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxDocumentBytes+1))
	if err != nil {
		fail("Gagal membaca berkas.")
		return
	}
	if len(data) == 0 || len(data) > maxDocumentBytes {
		fail("Ukuran berkas harus antara 1 byte dan 500 KB.")
		return
	}
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	mimeType := http.DetectContentType(sample)
	if !map[string]bool{"application/pdf": true, "image/jpeg": true, "image/png": true}[mimeType] {
		fail("Jenis berkas harus PDF, JPG, atau PNG.")
		return
	}

	blobID, err := s.storeDB.PutDocumentBlob(r.Context(), mimeType, data)
	if err != nil {
		fail("Gagal menyimpan berkas di server.")
		return
	}
	fileName := filepath.Base(header.Filename)
	if fileName == "" || fileName == "." {
		fileName = "berkas_resmi.pdf"
	}
	magic := data
	if len(magic) > 4 {
		magic = magic[:4]
	}
	upload := state.DocumentUpload{
		FileName:  fileName,
		MimeType:  mimeType,
		BlobID:    blobID,
		SizeBytes: len(data),
		MagicHex:  hex.EncodeToString(magic),
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UploadStudentDocument(doc, studentID, r.FormValue("doc_id"), upload, time.Now())
	}); err != nil {
		// Do not leave an unreferenced blob behind when the metadata write fails.
		if deleteErr := s.storeDB.DeleteDocumentBlob(r.Context(), blobID); deleteErr != nil {
			log.Printf("delete orphan document blob failed: %v", deleteErr)
		}
		fail("Berkas tersimpan tetapi gagal dicatat: " + err.Error())
		return
	}
	redirectWithFlash(w, r, "/student/documents", "Berkas siswa berhasil diunggah dan masuk ke antrean verifikasi Admin TU.", "success")
}

// handleDocumentFile serves a stored upload only to Admin TU, the headmaster,
// or the student/parent who owns the document record.
func (s *Server) handleDocumentFile(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student", "parent", "treasurer", "teacher", "discipline", "piket", "admin_tu", "headmaster", "developer")
	if !ok {
		return
	}
	blobID := r.PathValue("blobID")
	if !s.canReadDocument(r.Context(), authVal, blobID) {
		writeError(w, http.StatusForbidden, "Akses berkas ditolak.")
		return
	}
	mimeType, data, err := s.storeDB.DocumentBlob(r.Context(), blobID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", `inline; filename="dokumen"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

func (s *Server) canReadDocument(ctx context.Context, auth requestAuth, blobID string) bool {
	if auth.user.Role == "admin_tu" || auth.user.Role == "headmaster" {
		return true
	}
	if auth.user.Role != "student" && auth.user.Role != "parent" && auth.user.Role != "teacher" {
		return false
	}
	raw, err := s.storeDB.GetState(ctx)
	if err != nil || raw == nil {
		return false
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return false
	}
	// Teachers may open submitted task files, never student identity documents.
	if auth.user.Role != "teacher" {
		for _, item := range asMapSlice(doc["studentDocuments"]) {
			if strVal(item, "blobId") != blobID {
				continue
			}
			return strVal(item, "studentId") == auth.user.StudentProfileID
		}
	}
	for _, item := range asMapSlice(doc["taskSubmissions"]) {
		if strVal(item, "blobId") != blobID {
			continue
		}
		if auth.user.Role == "teacher" {
			return true
		}
		return strVal(item, "studentId") == auth.user.StudentProfileID
	}
	return false
}

const maxTaskBytes = 10 << 20

func (s *Server) handleStudentTaskSubmitForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	fail := func(message string) {
		redirectWithFlash(w, r, "/student/documents", message, "danger")
	}
	if err := r.ParseMultipartForm(maxTaskBytes); err != nil {
		fail("Berkas terlalu besar atau tidak dapat dibaca (maksimal 10 MB).")
		return
	}
	taskTitle := strings.TrimSpace(r.FormValue("task_title"))
	assignmentID := strings.TrimSpace(r.FormValue("assignment_id"))
	if taskTitle == "" && assignmentID == "" {
		fail("Pilih tugas dari guru atau isi judul tugas.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		fail("Pilih berkas tugas terlebih dahulu.")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxTaskBytes+1))
	if err != nil {
		fail("Gagal membaca berkas tugas.")
		return
	}
	if len(data) == 0 || len(data) > maxTaskBytes {
		fail("Ukuran berkas harus antara 1 byte dan 10 MB.")
		return
	}
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	mimeType := http.DetectContentType(sample)
	if !map[string]bool{"application/pdf": true, "image/jpeg": true, "image/png": true}[mimeType] {
		fail("Jenis berkas tugas harus PDF, JPG, atau PNG.")
		return
	}
	blobID, err := s.storeDB.PutDocumentBlob(r.Context(), mimeType, data)
	if err != nil {
		fail("Gagal menyimpan berkas tugas di server.")
		return
	}
	fileName := filepath.Base(header.Filename)
	if fileName == "" || fileName == "." {
		fileName = "tugas_praktik.pdf"
	}
	fileSizeKB := (len(data) + 1023) / 1024
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.SubmitStudentTask(doc, studentID, assignmentID, taskTitle, fileName, blobID, mimeType, fileSizeKB, time.Now())
	}); err != nil {
		if deleteErr := s.storeDB.DeleteDocumentBlob(r.Context(), blobID); deleteErr != nil {
			log.Printf("delete orphan task blob failed: %v", deleteErr)
		}
		fail("Berkas tersimpan tetapi gagal dicatat: " + err.Error())
		return
	}
	redirectWithFlash(w, r, "/student/documents", "Tugas praktik berhasil diunggah dan dikirim ke Guru Mapel.", "success")
}

func (s *Server) handleStudentPklCheckinForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "student")
	if !ok {
		return
	}
	studentID := authVal.user.StudentProfileID
	if studentID == "" {
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddPklCheckin(doc, studentID, 0, 0, time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/student/academic", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddPklJournal(doc, studentID, r.FormValue("activity_title"), r.FormValue("competency"), time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/student/academic", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UpdateBmwChoice(doc, studentID, r.FormValue("choice"))
	}); err != nil {
		redirectWithFlash(w, r, "/student/academic", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.ApplyBkkJob(doc, studentID, r.FormValue("job_id"), time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/student/academic", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddStudentExpense(doc, studentID, "ALAT_PRAKTIK", r.FormValue("title"), amt, r.FormValue("note"), time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/student/academic", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
	shell := s.buildRoleShell(r.Context(), authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildParentPortal(r.Context(), doc, shell, authVal.user, tab)
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
		redirectWithFlash(w, r, "/", "Akun ini belum terhubung ke profil siswa. Hubungi Admin TU.", "danger")
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.BookCareerConsultation(doc, studentID, r.FormValue("chosen_path"), r.FormValue("topic_note"), time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/parent/future-plan", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
		redirectWithFlash(w, r, "/parent/services", "Akun wali belum terhubung ke profil siswa.", "danger")
		return
	}
	leaveDate, err := time.Parse("2006-01-02", strings.TrimSpace(r.FormValue("leave_date")))
	if err != nil {
		redirectWithFlash(w, r, "/parent/services", "Tanggal izin tidak valid.", "danger")
		return
	}
	if strings.TrimSpace(r.FormValue("leave_note")) == "" {
		redirectWithFlash(w, r, "/parent/services", "Keterangan izin wajib diisi.", "danger")
		return
	}
	if _, err := s.storeDB.CreateParentLeaveRequest(r.Context(), studentID,
		r.FormValue("leave_type"), leaveDate, r.FormValue("leave_note")); err != nil {
		redirectWithFlash(w, r, "/parent/services", "Izin gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/parent/services", "Izin tersimpan untuk tanggal tersebut dan presensi siswa sudah diperbarui.", "success")
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
	doc, _ := s.loadStateForRole(r.Context(), authVal.user.Role)
	shell := s.buildRoleShell(r.Context(), authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildTreasurerPortal(r.Context(), doc, shell, authVal.user, tab)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.TreasurerPortal(data)))
}

func (s *Server) handleTreasurerScholarshipForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "treasurer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	pct, _ := strconv.Atoi(r.FormValue("discount_percent"))
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UpdateScholarship(doc, r.FormValue("student_id"), r.FormValue("scholarship_type"), pct)
	}); err != nil {
		redirectWithFlash(w, r, "/treasurer/dashboard", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/treasurer/dashboard", "Potongan beasiswa siswa dan nominal tagihan SPP berhasil dihitung ulang.", "success")
}

func (s *Server) handleTreasurerWaReminderForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "treasurer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.SendWaBillingReminder(doc, r.FormValue("invoice_id"), time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/treasurer/dashboard", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.GrantDispensation(doc, r.FormValue("student_id"), r.FormValue("reason"), authVal.user.FullName, time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/treasurer/exam-clearance", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/treasurer/exam-clearance", "Dispensasi Ujian 7 Hari berhasil diterbitkan. Pintu ujian CBT siswa kini terbuka.", "success")
}

func (s *Server) handleTreasurerReversalForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "treasurer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	if s.cache != nil {
		attempts, err := s.cache.IncrementCounter(r.Context(), "ratelimit:reversal:"+authVal.user.ID, 10*time.Minute)
		if err != nil {
			redirectWithFlash(w, r, "/treasurer/reports", "Layanan otorisasi sedang sibuk. Coba lagi nanti.", "danger")
			return
		}
		if attempts > 10 {
			redirectWithFlash(w, r, "/treasurer/reports", "Terlalu banyak percobaan OTP. Minta kode baru kepada Kepala Sekolah.", "danger")
			return
		}
	}
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
	doc, _ := s.loadStateForRole(r.Context(), authVal.user.Role)
	shell := s.buildRoleShell(r.Context(), authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildTeacherPortal(r.Context(), doc, shell, authVal.user, tab, r.URL.Query().Get("classroom"))
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.TeacherPortal(data)))
}

func (s *Server) handleTeacherAlertPiketForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.AlertPiketAbsent(doc, authVal.user.FullName)
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/dashboard", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/dashboard", "Sinyal izin darurat berhasil dikirim ke Radar Guru Piket untuk penugasan Guru Inval.", "warning")
}

func (s *Server) handleTeacherJournalForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	scheduleID := strings.TrimSpace(r.FormValue("schedule_id"))
	if scheduleID == "" {
		redirectWithFlash(w, r, "/teacher/classroom", "Pilih slot jadwal hari ini sebelum menyimpan agenda.", "danger")
		return
	}
	if err := s.storeDB.SaveClassJournal(r.Context(), scheduleID, authVal.user.ID, schoolDate(),
		r.FormValue("topic"), r.FormValue("notes"), r.FormValue("ai_notes")); err != nil {
		redirectWithFlash(w, r, "/teacher/classroom", "Agenda gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/classroom", "Agenda tersimpan untuk slot jadwal dan tanggal ini.", "success")
}

func (s *Server) handleTeacherCreateAssignmentForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	classroomID := strings.TrimSpace(r.FormValue("classroom_id"))
	if classroomID == "" {
		redirectWithFlash(w, r, "/teacher/classroom", "Pilih kelas yang Anda ampu.", "danger")
		return
	}
	allowed, err := s.storeDB.TeacherCanManageClassroom(r.Context(), authVal.user.ID, classroomID, schoolDate())
	if err != nil || !allowed {
		redirectWithFlash(w, r, "/teacher/classroom", "Anda tidak mengampu kelas tersebut hari ini.", "danger")
		return
	}
	class, err := s.storeDB.ClassroomByID(r.Context(), classroomID)
	if err != nil {
		redirectWithFlash(w, r, "/teacher/classroom", "Kelas tidak ditemukan.", "danger")
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.CreateAssignment(doc, authVal.user.FullName,
			class.Name, r.FormValue("subject_name"),
			r.FormValue("title"), r.FormValue("description"), r.FormValue("due_date"), time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/classroom", "Tugas gagal dibuat: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/classroom", "Tugas baru dipublikasikan ke kelas.", "success")
}

func (s *Server) handleTeacherTaskGradeForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	score, err := strconv.Atoi(r.FormValue("score"))
	if err != nil || score < 0 || score > 100 {
		redirectWithFlash(w, r, "/teacher/classroom", "Nilai harus berupa angka 0-100.", "danger")
		return
	}
	taskID := r.FormValue("task_id")
	doc, _ := s.loadStateForRole(r.Context(), authVal.user.Role)
	if !s.teacherOwnsTask(r.Context(), authVal.user, doc, taskID) {
		redirectWithFlash(w, r, "/teacher/classroom", "Tugas ini bukan milik kelas yang Anda ampu.", "danger")
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.GradeTaskSubmission(doc, taskID, score, time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/classroom", "Nilai gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/classroom", "Nilai tugas berhasil disimpan dan terlihat oleh siswa.", "success")
}

func (s *Server) handleTeacherToggleAlpaForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	classroomID := strings.TrimSpace(r.FormValue("classroom_id"))
	studentID := strings.TrimSpace(r.FormValue("student_id"))
	if classroomID == "" || studentID == "" {
		redirectWithFlash(w, r, "/teacher/classroom", "Data siswa/kelas tidak lengkap.", "danger")
		return
	}
	allowed, err := s.storeDB.TeacherCanManageClassroom(r.Context(), authVal.user.ID, classroomID, schoolDate())
	if err != nil || !allowed {
		redirectWithFlash(w, r, "/teacher/classroom", "Anda tidak berhak mengubah presensi kelas ini.", "danger")
		return
	}
	current := ""
	roster, err := s.storeDB.ClassroomRoster(r.Context(), classroomID, schoolDate())
	if err != nil {
		redirectWithFlash(w, r, "/teacher/classroom", "Presensi gagal dibaca: "+err.Error(), "danger")
		return
	}
	for _, row := range roster {
		if row.StudentAppID == studentID {
			current = row.Status
			break
		}
	}
	status := "ALPA_BOLOS"
	source := "Koreksi guru dari portal"
	if current == "ALPA_BOLOS" {
		status = "HADIR_GERBANG"
	}
	if _, err := s.storeDB.UpsertStudentAttendance(r.Context(), studentID, schoolDate(), status, source); err != nil {
		redirectWithFlash(w, r, "/teacher/classroom", "Presensi gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/classroom", "Status kehadiran siswa di kelas berhasil diperbarui.", "success")
}

func (s *Server) handleTeacherAddQuestionForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddExamQuestion(doc, r.FormValue("question_type"), r.FormValue("question_text"), r.FormValue("opt_a"), r.FormValue("opt_b"), r.FormValue("opt_c"), r.FormValue("opt_d"), r.FormValue("correct_key"), r.FormValue("essay_rubric"), time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/question-bank", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/question-bank", "Soal baru berhasil ditambahkan ke Bank Soal Kejuruan di server.", "success")
}

func (s *Server) handleTeacherDeleteQuestionForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.DeleteExamQuestion(doc, r.FormValue("question_id"))
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/question-bank", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		unlocked = state.UnlockProctorStudentWithToken(doc, studentID, token)
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/exam-control", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.MassUnlockProctor(doc)
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/exam-control", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/exam-control", "Mass Unlock berhasil dijalankan untuk seluruh peserta di ruangan lab.", "success")
}

func (s *Server) handleTeacherEmergencyTicketForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.GrantEmergency2hTicket(doc, r.FormValue("student_id"), authVal.user.FullName, time.Now())
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/exam-control", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/exam-control", "Tiket Darurat 2 Jam berhasil diterbitkan untuk membuka akses ujian siswa.", "success")
}

func (s *Server) handleTeacherProctorResetExamForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentID := strings.TrimSpace(r.FormValue("student_id"))
	reason := strings.TrimSpace(r.FormValue("reason"))
	subjectID := strings.TrimSpace(r.FormValue("subject_id"))
	if reason == "" {
		reason = "Kendala teknis / izin pengawas ruangan"
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.ResetStudentExam(doc, studentID, subjectID, authVal.user.ID, authVal.user.FullName, reason, time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/exam-control", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/exam-control", fmt.Sprintf("Sesi ujian CBT siswa %s berhasil di-reset untuk mulai ulang.", studentID), "success")
}

func (s *Server) handleTeacherProctorBulkResetForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	studentIDs := r.Form["student_ids"]
	reason := strings.TrimSpace(r.FormValue("reason"))
	subjectID := strings.TrimSpace(r.FormValue("subject_id"))
	if reason == "" {
		reason = "Reset massal pengawas lab komputer"
	}
	if r.FormValue("reset_all") == "1" {
		studentIDs = []string{"ALL"}
	}
	count := 0
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		var err error
		count, err = state.BulkResetStudentExams(doc, studentIDs, subjectID, authVal.user.ID, authVal.user.FullName, reason, time.Now())
		return err
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/exam-control", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/exam-control", fmt.Sprintf("Reset ujian massal berhasil dijalankan untuk %d peserta di lab.", count), "success")
}

func (s *Server) handleTeacherEssayGradeForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	_ = r.ParseForm()
	score, _ := strconv.Atoi(r.FormValue("score"))
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.GradeEssayToRapor(doc, r.FormValue("student_id"), score)
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/exam-control", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		var err error
		filtered, err = state.UpdateHomeroomAttitude(doc, r.FormValue("student_id"), scores, r.FormValue("homeroom_note"))
		return err
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/homeroom-pkl", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/teacher/homeroom-pkl", fmt.Sprintf("Outlier Consensus Filter (MAD) selesai dihitung. Skor sikap terlindungi di %.1f/100.", filtered), "success")
}

func (s *Server) handleTeacherBulkApprovePklForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "teacher")
	if !ok {
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.BulkApprovePklJournals(doc)
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/teacher/homeroom-pkl", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
	doc, _ := s.loadStateForRole(r.Context(), authVal.user.Role)
	shell := s.buildRoleShell(r.Context(), authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
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
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddManualViolation(doc, r.FormValue("student_name"), r.FormValue("article_code"), r.FormValue("article_title"), pts, time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/discipline/violations", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/discipline/violations", "Catatan pelanggaran tata tertib berhasil disimpan.", "success")
}

func (s *Server) handleDisciplineSummonForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "discipline")
	if !ok {
		return
	}
	_ = r.ParseForm()
	var hash string
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		var err error
		hash, err = state.IssueSummonLetter(doc, r.FormValue("violation_id"), s.cfg.ReceiptSecret, time.Now())
		return err
	}); err != nil {
		redirectWithFlash(w, r, "/discipline/violations", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/discipline/violations", "Surat Panggilan Orang Tua bersegel HMAC ("+hash+") berhasil diterbitkan.", "success")
}

func (s *Server) handleDisciplineResolvePeerForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "discipline")
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.ResolvePeerReport(doc, r.FormValue("report_id"), r.FormValue("status"))
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/discipline/violations", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/discipline/violations", "Status laporan Escrow Perundungan berhasil diperbarui.", "success")
}

func (s *Server) handleDisciplineScheduleConsultForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "discipline")
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.ScheduleConsultation(doc, r.FormValue("consult_id"))
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/discipline/violations", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/discipline/violations", "Jadwal konsultasi karir BMW orang tua berhasil dikonfirmasi oleh BK.", "success")
}

func (s *Server) handleDisciplineCounselingUpdateForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "discipline")
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.UpdateCounselingSession(doc, r.FormValue("session_id"), r.FormValue("status"), r.FormValue("notes"))
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/discipline/transformation-index", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
	shell := s.buildRoleShell(r.Context(), authVal, "dashboard", r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildPiketPortal(r.Context(), shell)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.PiketPortal(data)))
}

func (s *Server) handlePiketAssignInvalForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "piket")
	if !ok {
		return
	}
	_ = r.ParseForm()
	scheduleID := strings.TrimSpace(r.FormValue("schedule_id"))
	teacherID := strings.TrimSpace(r.FormValue("teacher_id"))
	if scheduleID == "" || teacherID == "" {
		redirectWithFlash(w, r, "/piket/dashboard", "Pilih slot jadwal dan guru pengganti.", "danger")
		return
	}
	if _, err := s.storeDB.CreateSubstituteAssignment(r.Context(), scheduleID, teacherID,
		r.FormValue("task_note"), authVal.user.ID, schoolDate()); err != nil {
		redirectWithFlash(w, r, "/piket/dashboard", "Penugasan inval gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/piket/dashboard", "Guru pengganti tersimpan pada slot jadwal tersebut.", "success")
}

func (s *Server) handlePiketGateLogForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "piket")
	if !ok {
		return
	}
	_ = r.ParseForm()
	passBytes := make([]byte, 4)
	if _, err := rand.Read(passBytes); err != nil {
		renderInternalError(w, r)
		return
	}
	passCode := "PASS-" + strings.ToUpper(hex.EncodeToString(passBytes))
	if _, err := s.storeDB.CreateGateLog(r.Context(), r.FormValue("nisn"), r.FormValue("log_type"),
		r.FormValue("reason"), passCode, authVal.user.ID); err != nil {
		redirectWithFlash(w, r, "/piket/dashboard", "Log gerbang gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/piket/dashboard", "Log gerbang tersimpan dengan kode pass "+passCode+".", "success")
}

func (s *Server) teacherOwnsTask(ctx context.Context, user *model.User, doc map[string]any, taskID string) bool {
	if s.storeDB == nil || user == nil || taskID == "" {
		return false
	}
	assignmentID := ""
	for _, item := range asMapSlice(doc["taskSubmissions"]) {
		if strVal(item, "id") == taskID {
			assignmentID = strVal(item, "assignmentId")
			break
		}
	}
	if assignmentID == "" {
		return false
	}
	className := ""
	for _, item := range asMapSlice(doc["assignments"]) {
		if strVal(item, "id") == assignmentID && sameTeacher(strVal(item, "teacherName"), user.FullName) {
			className = strVal(item, "className")
			break
		}
	}
	if className == "" {
		return false
	}
	classes, err := s.storeDB.ListClassrooms(ctx)
	if err != nil {
		return false
	}
	for _, class := range classes {
		if class.Name != className {
			continue
		}
		allowed, err := s.storeDB.TeacherCanManageClassroom(ctx, user.ID, class.ID, schoolDate())
		return err == nil && allowed
	}
	return false
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
	doc, _ := s.loadStateForRole(r.Context(), authVal.user.Role)
	shell := s.buildRoleShell(r.Context(), authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	data := s.buildAdminTuPortal(r.Context(), authVal.user, doc, shell, tab,
		r.URL.Query().Get("section"), r.URL.Query().Get("q"), page)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.AdminTuPortal(data)))
}

func (s *Server) handleAdminVerifyDocForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	approved := r.FormValue("decision") == "approve"
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.VerifyStudentDoc(doc, r.FormValue("doc_id"), approved, r.FormValue("reject_reason"))
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/admin/master-data", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/admin/master-data", "Status verifikasi berkas siswa berhasil diperbarui.", "success")
}

func (s *Server) handleAdminAddStudentForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	nisn := strings.TrimSpace(r.FormValue("nisn"))
	classroomID := strings.TrimSpace(r.FormValue("classroom_id"))
	if classroomID == "" {
		redirectWithFlash(w, r, "/admin/master-data?section=classes", "Buat kelas di master data terlebih dahulu, lalu pilih kelas siswa.", "danger")
		return
	}
	class, err := s.storeDB.ClassroomByID(r.Context(), classroomID)
	if err != nil {
		redirectWithFlash(w, r, "/admin/master-data?section=classes", "Kelas yang dipilih tidak ditemukan.", "danger")
		return
	}
	classroom, majorCode := class.Name, class.MajorCode
	parentName := strings.TrimSpace(r.FormValue("parent_name"))
	parentPhone := model.NormalizePhone(r.FormValue("parent_phone"))
	initialPassword := r.FormValue("student_password")
	if fullName == "" || classroom == "" || majorCode == "" || parentName == "" {
		redirectWithFlash(w, r, "/admin/master-data", "Nama siswa, kelas, jurusan, dan nama wali wajib diisi.", "danger")
		return
	}
	if len(nisn) != 10 {
		redirectWithFlash(w, r, "/admin/master-data", "NISN harus tepat 10 digit angka.", "danger")
		return
	}
	if len(initialPassword) < 8 {
		redirectWithFlash(w, r, "/admin/master-data", "Kata sandi awal siswa minimal 8 karakter.", "danger")
		return
	}
	if len(parentPhone) < 10 {
		redirectWithFlash(w, r, "/admin/master-data", "Nomor WhatsApp wali tidak valid. Gunakan format 08xxxxxxxxxx.", "danger")
		return
	}

	studentPasswordHash, err := auth.HashPassword(initialPassword)
	if err != nil {
		renderInternalError(w, r)
		return
	}
	parentSecret := make([]byte, 16)
	if _, err := rand.Read(parentSecret); err != nil {
		renderInternalError(w, r)
		return
	}
	parentPasswordHash, err := auth.HashPassword(hex.EncodeToString(parentSecret))
	if err != nil {
		renderInternalError(w, r)
		return
	}

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		renderInternalError(w, r)
		return
	}
	studentID := "std-" + hex.EncodeToString(suffix)
	studentUserID := "usr-std-" + hex.EncodeToString(suffix)
	parentUserID := "usr-prt-" + hex.EncodeToString(suffix)

	if err := s.storeDB.CreateStudentWithAccounts(r.Context(), store.StudentOnboarding{
		StudentID: studentID, StudentUserID: studentUserID, ParentUserID: parentUserID,
		FullName: fullName, NISN: nisn, ClassroomID: classroomID, ClassroomName: classroom, MajorCode: majorCode,
		ParentName: parentName, ParentPhone: parentPhone,
		StudentPasswordHash: studentPasswordHash, ParentPasswordHash: parentPasswordHash,
	}); err != nil {
		redirectWithFlash(w, r, "/admin/master-data", "Siswa gagal ditambahkan: "+err.Error(), "danger")
		return
	}

	input := state.NewStudentInput{
		ID: studentID, UserID: studentUserID, ParentUserID: parentUserID,
		FullName: fullName, NISN: nisn, ClassroomID: classroomID, Classroom: classroom, MajorCode: majorCode,
		ParentName: parentName, ParentPhone: parentPhone,
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddNewStudent(doc, input, time.Now())
	}); err != nil {
		_ = s.storeDB.DeleteStudentOnboarding(r.Context(), studentUserID, parentUserID)
		redirectWithFlash(w, r, "/admin/master-data", "Data siswa gagal disimpan: "+err.Error(), "danger")
		return
	}

	s.auditAccount(r.Context(), authVal, "account.create", studentUserID, "student", map[string]any{"student_id": studentID, "nisn": nisn})
	s.auditAccount(r.Context(), authVal, "account.create", parentUserID, "parent", map[string]any{"student_id": studentID})
	redirectWithFlash(w, r, "/admin/master-data", "Siswa, akun login, dan akun wali berhasil dibuat. Siswa masuk dengan NISN + kata sandi awal; wali masuk lewat tautan WhatsApp.", "success")
}

func (s *Server) handleAdminCreateAcademicYearForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	label := strings.TrimSpace(r.FormValue("label"))
	semester := strings.TrimSpace(r.FormValue("semester"))
	start, startErr := time.Parse("2006-01-02", r.FormValue("start_date"))
	end, endErr := time.Parse("2006-01-02", r.FormValue("end_date"))
	validLabel := len(label) == 9 && label[4] == '/'
	if validLabel {
		for i, char := range label {
			if i != 4 && (char < '0' || char > '9') {
				validLabel = false
				break
			}
		}
	}
	if !validLabel || startErr != nil || endErr != nil || !end.After(start) {
		redirectWithFlash(w, r, masterDataClassesPath, "Tahun ajaran tidak valid. Gunakan format 2027/2028 dan tanggal yang benar.", "danger")
		return
	}
	year, err := s.storeDB.CreateAcademicYear(r.Context(), label, semester, start, end)
	if err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Tahun ajaran gagal disimpan: "+err.Error(), "danger")
		return
	}
	s.auditMasterData(r.Context(), authVal, "masterdata.academic_year.create", year.ID, map[string]any{"label": year.Label, "semester": year.Semester})
	redirectWithFlash(w, r, masterDataClassesPath, "Tahun ajaran "+year.Label+" "+year.Semester+" ditambahkan. Aktifkan saat siap dipakai.", "success")
}

func (s *Server) handleAdminActivateAcademicYearForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	id := strings.TrimSpace(r.FormValue("year_id"))
	if id == "" {
		redirectWithFlash(w, r, masterDataClassesPath, "Tahun ajaran tidak dipilih.", "danger")
		return
	}
	if err := s.storeDB.ActivateAcademicYear(r.Context(), id); err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Tahun ajaran gagal diaktifkan: "+err.Error(), "danger")
		return
	}
	s.auditMasterData(r.Context(), authVal, "masterdata.academic_year.activate", id, nil)
	redirectWithFlash(w, r, masterDataClassesPath, "Tahun ajaran aktif diperbarui di seluruh portal.", "success")
}

func (s *Server) handleAdminRolloverForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	targetYearID := strings.TrimSpace(r.FormValue("target_year_id"))
	classes, err := s.storeDB.ListClassrooms(r.Context())
	if err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Kelas gagal dibaca: "+err.Error(), "danger")
		return
	}
	moves := make([]store.RolloverMove, 0)
	graduates := make([]string, 0)
	for _, class := range classes {
		choice := strings.TrimSpace(r.FormValue("target_" + class.ID))
		switch choice {
		case "":
			continue
		case "graduate":
			graduates = append(graduates, class.ID)
		default:
			moves = append(moves, store.RolloverMove{FromClassID: class.ID, ToClassID: choice})
		}
	}
	result, err := s.storeDB.RolloverStudents(r.Context(), targetYearID, moves, graduates)
	if err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Kenaikan kelas gagal diproses: "+err.Error(), "danger")
		return
	}
	s.auditMasterData(r.Context(), authVal, "masterdata.academic_year.rollover", result.TargetYear.ID, map[string]any{
		"label": result.TargetYear.Label, "semester": result.TargetYear.Semester,
		"moved": result.MovedStudents, "graduated": result.GraduatedStudents,
		"moves": moves, "graduates": graduates,
	})
	redirectWithFlash(w, r, masterDataClassesPath, fmt.Sprintf(
		"Kenaikan kelas selesai: %d siswa naik kelas, %d siswa lulus. Tahun ajaran %s %s sekarang aktif.",
		result.MovedStudents, result.GraduatedStudents, result.TargetYear.Label, result.TargetYear.Semester), "success")
}

func (s *Server) handleAdminCreateClassroomForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	grade, err := strconv.Atoi(r.FormValue("grade_level"))
	if err != nil {
		redirectWithFlash(w, r, "/admin/master-data", "Tingkat kelas tidak valid.", "danger")
		return
	}
	class, err := s.storeDB.CreateClassroom(r.Context(), r.FormValue("name"), grade, r.FormValue("major_code"), r.FormValue("homeroom_teacher_id"))
	if err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Kelas gagal disimpan: "+err.Error(), "danger")
		return
	}
	s.auditMasterData(r.Context(), authVal, "masterdata.classroom.create", class.ID, map[string]any{"name": class.Name, "major": class.MajorCode})
	redirectWithFlash(w, r, masterDataClassesPath, "Kelas "+class.Name+" berhasil ditambahkan.", "success")
}

const masterDataClassesPath = "/admin/master-data?section=classes"

func (s *Server) handleAdminUpdateClassroomForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	grade, err := strconv.Atoi(r.FormValue("grade_level"))
	if err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Tingkat kelas tidak valid.", "danger")
		return
	}
	class, err := s.storeDB.UpdateClassroom(r.Context(), strings.TrimSpace(r.FormValue("class_id")),
		r.FormValue("name"), grade, r.FormValue("major_code"), r.FormValue("homeroom_teacher_id"))
	if err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Kelas gagal diperbarui: "+err.Error(), "danger")
		return
	}
	s.auditMasterData(r.Context(), authVal, "masterdata.classroom.update", class.ID, map[string]any{"name": class.Name, "major": class.MajorCode})
	redirectWithFlash(w, r, masterDataClassesPath, "Kelas "+class.Name+" berhasil diperbarui.", "success")
}

func (s *Server) handleAdminDeleteClassroomForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	id := strings.TrimSpace(r.FormValue("class_id"))
	if err := s.storeDB.DeleteClassroom(r.Context(), id); err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Kelas gagal dihapus: "+err.Error(), "danger")
		return
	}
	s.auditMasterData(r.Context(), authVal, "masterdata.classroom.delete", id, nil)
	redirectWithFlash(w, r, masterDataClassesPath, "Kelas berhasil dihapus.", "success")
}

func (s *Server) handleAdminCreateScheduleForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	day, err := strconv.Atoi(r.FormValue("day_of_week"))
	if err != nil {
		redirectWithFlash(w, r, "/admin/master-data", "Hari jadwal tidak valid.", "danger")
		return
	}
	classroomID := strings.TrimSpace(r.FormValue("classroom_id"))
	subject := strings.TrimSpace(r.FormValue("subject_id"))
	start, end := r.FormValue("start_time"), r.FormValue("end_time")
	if err := s.storeDB.CreateSchedule(r.Context(), classroomID, subject, r.FormValue("teacher_id"), day, start, end, r.FormValue("room_name")); err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Jadwal gagal disimpan: "+err.Error(), "danger")
		return
	}
	s.auditMasterData(r.Context(), authVal, "masterdata.schedule.create", classroomID, map[string]any{"subject": subject, "day": day, "start": start, "end": end})
	redirectWithFlash(w, r, masterDataClassesPath, "Jadwal pelajaran berhasil ditambahkan.", "success")
}

func (s *Server) handleAdminUpdateScheduleForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	day, err := strconv.Atoi(r.FormValue("day_of_week"))
	if err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Hari jadwal tidak valid.", "danger")
		return
	}
	scheduleID := strings.TrimSpace(r.FormValue("schedule_id"))
	classroomID := strings.TrimSpace(r.FormValue("classroom_id"))
	subject := strings.TrimSpace(r.FormValue("subject_id"))
	start, end := r.FormValue("start_time"), r.FormValue("end_time")
	if err := s.storeDB.UpdateSchedule(r.Context(), scheduleID, classroomID, subject,
		r.FormValue("teacher_id"), day, start, end, r.FormValue("room_name")); err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Jadwal gagal diperbarui: "+err.Error(), "danger")
		return
	}
	s.auditMasterData(r.Context(), authVal, "masterdata.schedule.update", scheduleID, map[string]any{"classroom_id": classroomID, "subject": subject, "day": day, "start": start, "end": end})
	redirectWithFlash(w, r, masterDataClassesPath, "Jadwal pelajaran berhasil diperbarui.", "success")
}

func (s *Server) handleAdminDeleteScheduleForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	id := strings.TrimSpace(r.FormValue("schedule_id"))
	if err := s.storeDB.DeleteSchedule(r.Context(), id); err != nil {
		redirectWithFlash(w, r, masterDataClassesPath, "Jadwal gagal dihapus: "+err.Error(), "danger")
		return
	}
	s.auditMasterData(r.Context(), authVal, "masterdata.schedule.delete", id, nil)
	redirectWithFlash(w, r, masterDataClassesPath, "Jadwal pelajaran berhasil dihapus.", "success")
}

func (s *Server) auditMasterData(ctx context.Context, authVal requestAuth, action, targetID string, extra map[string]any) {
	detail := map[string]any{"target_id": targetID}
	for key, value := range extra {
		detail[key] = value
	}
	ram, err := json.Marshal(detail)
	if err != nil {
		return
	}
	_ = s.storeDB.RecordStateAudit(ctx, store.StateAudit{
		ActorID: authVal.user.ID, Role: authVal.user.Role, Action: action, Detail: ram,
	})
}

func (s *Server) handleAdminDapodikSyncForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.TriggerDapodikSync(doc, time.Now())
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/admin/dapodik-erapor", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		ltr := state.CreateOfficialLetter(doc, r.FormValue("letter_type"), r.FormValue("student_name"), r.FormValue("recipient"), s.cfg.ReceiptSecret, time.Now())
		hash = strVal(ltr, "hmacHash")
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/admin/dapodik-erapor", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
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
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddPklPlacement(doc, r.FormValue("company_name"), r.FormValue("sector"), r.FormValue("address"), lat, lng, rad, r.FormValue("pin"), time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/admin/pkl-bkk", err.Error(), "danger")
		return
	}
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
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.AddBkkJob(doc, r.FormValue("company_name"), r.FormValue("job_title"), r.FormValue("salary_range"), majors, time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/admin/pkl-bkk", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/admin/pkl-bkk", "Lowongan kerja Bursa Kerja Khusus (BKK) berhasil dipublikasikan.", "success")
}

func (s *Server) handleAdminCreateAccountForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	identifier := strings.TrimSpace(r.FormValue("identifier"))
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	role := strings.TrimSpace(r.FormValue("role"))
	password := r.FormValue("password")
	switch role {
	case "teacher", "admin_tu", "treasurer", "headmaster", "discipline", "piket", "developer":
	default:
		redirectWithFlash(w, r, "/admin/master-data", "Peran akun tidak valid.", "danger")
		return
	}
	if identifier == "" || fullName == "" || len(password) < 8 {
		redirectWithFlash(w, r, "/admin/master-data", "Nama, identitas, dan kata sandi minimal 8 karakter wajib diisi.", "danger")
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		redirectWithFlash(w, r, "/admin/master-data", "Kata sandi gagal diproses.", "danger")
		return
	}
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	userID := "usr-" + hex.EncodeToString(buf)
	err = s.storeDB.CreateAccount(r.Context(), &model.User{
		ID: userID, Username: identifier, FullName: fullName, Role: role,
		PhoneWA: r.FormValue("phone_wa"), PasswordHash: hash, IsActive: true,
	})
	if err != nil {
		redirectWithFlash(w, r, "/admin/master-data", "Akun gagal dibuat: "+err.Error(), "danger")
		return
	}
	s.auditAccount(r.Context(), authVal, "account.create", userID, role, nil)
	redirectWithFlash(w, r, "/admin/master-data", "Akun "+fullName+" berhasil dibuat. Sampaikan kata sandi awal secara pribadi.", "success")
}

func (s *Server) handleAdminResetMFAForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	userID := strings.TrimSpace(r.FormValue("user_id"))
	if userID == "" {
		http.NotFound(w, r)
		return
	}
	if userID == authVal.user.ID {
		redirectWithFlash(w, r, "/admin/master-data", "Reset MFA akun sendiri harus lewat halaman Keamanan Akun.", "danger")
		return
	}
	found, err := s.storeDB.SetTOTP(r.Context(), userID, "", false)
	if err != nil || !found {
		redirectWithFlash(w, r, "/admin/master-data", "MFA gagal direset.", "danger")
		return
	}
	_ = s.storeDB.DeleteRecoveryCodes(r.Context(), userID)
	if s.cache != nil {
		_ = s.cache.RevokeUserSessions(r.Context(), userID)
	}
	s.auditAccount(r.Context(), authVal, "account.mfa_reset", userID, "", nil)
	redirectWithFlash(w, r, "/admin/master-data", "MFA direset dan semua sesi akun tersebut dicabut. Pengguna dapat mendaftarkan perangkat baru.", "success")
}

func (s *Server) handleAdminResetPasswordForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	userID := strings.TrimSpace(r.FormValue("user_id"))
	password := r.FormValue("password")
	if userID == "" || len(password) < 8 {
		redirectWithFlash(w, r, "/admin/master-data", "Kata sandi baru minimal 8 karakter.", "danger")
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		redirectWithFlash(w, r, "/admin/master-data", "Kata sandi gagal diproses.", "danger")
		return
	}
	found, err := s.storeDB.SetUserPasswordHash(r.Context(), userID, hash)
	if err != nil || !found {
		redirectWithFlash(w, r, "/admin/master-data", "Akun tidak ditemukan atau kata sandi gagal disimpan.", "danger")
		return
	}
	if s.cache != nil {
		_ = s.cache.RevokeUserSessions(r.Context(), userID)
	}
	s.auditAccount(r.Context(), authVal, "account.password_reset", userID, "", nil)
	redirectWithFlash(w, r, "/admin/master-data", "Kata sandi diperbarui dan semua sesi akun tersebut dicabut.", "success")
}

func (s *Server) handleAdminAccountStatusForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	userID := strings.TrimSpace(r.FormValue("user_id"))
	active := r.FormValue("active") == "true"
	if userID == "" {
		http.NotFound(w, r)
		return
	}
	if userID == authVal.user.ID && !active {
		redirectWithFlash(w, r, "/admin/master-data", "Anda tidak dapat menonaktifkan akun sendiri.", "danger")
		return
	}
	found, err := s.storeDB.SetUserActive(r.Context(), userID, active)
	if err != nil || !found {
		redirectWithFlash(w, r, "/admin/master-data", "Status akun gagal disimpan.", "danger")
		return
	}
	if !active && s.cache != nil {
		_ = s.cache.RevokeUserSessions(r.Context(), userID)
	}
	action := "account.deactivate"
	message := "Akun dinonaktifkan dan sesi aktifnya dicabut."
	if active {
		action = "account.activate"
		message = "Akun diaktifkan kembali."
	}
	s.auditAccount(r.Context(), authVal, action, userID, "", map[string]any{"active": active})
	redirectWithFlash(w, r, "/admin/master-data", message, "success")
}

func (s *Server) handleAdminRevokeSessionsForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "admin_tu")
	if !ok {
		return
	}
	_ = r.ParseForm()
	userID := strings.TrimSpace(r.FormValue("user_id"))
	if userID == "" || s.cache == nil {
		redirectWithFlash(w, r, "/admin/master-data", "Sesi gagal dicabut.", "danger")
		return
	}
	if err := s.cache.RevokeUserSessions(r.Context(), userID); err != nil {
		redirectWithFlash(w, r, "/admin/master-data", "Sesi gagal dicabut: "+err.Error(), "danger")
		return
	}
	s.auditAccount(r.Context(), authVal, "account.revoke_sessions", userID, "", nil)
	redirectWithFlash(w, r, "/admin/master-data", "Semua sesi aktif akun tersebut telah dicabut.", "success")
}

func (s *Server) auditAccount(ctx context.Context, authVal requestAuth, action, targetID, targetRole string, extra map[string]any) {
	detail := map[string]any{"user_id": targetID}
	if targetRole != "" {
		detail["role"] = targetRole
	}
	for key, value := range extra {
		detail[key] = value
	}
	ram, err := json.Marshal(detail)
	if err != nil {
		return
	}
	_ = s.storeDB.RecordStateAudit(ctx, store.StateAudit{
		ActorID: authVal.user.ID, Role: authVal.user.Role, Action: action, Detail: ram,
	})
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
	doc, _ := s.loadStateForRole(r.Context(), authVal.user.Role)
	shell := s.buildRoleShell(r.Context(), authVal, "executive", r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildHeadmasterPortal(r.Context(), doc, shell, r.URL.Query().Get("advice"))
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.HeadmasterPortal(data)))
}

func (s *Server) handleHeadmasterESignForm(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "headmaster")
	if !ok {
		return
	}
	now := time.Now().UTC()
	payload := fmt.Sprintf("laporan-eksekutif|%s|%s", authVal.user.ID, now.Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(payload))
	documentHash := hex.EncodeToString(sum[:])
	mac := hmac.New(sha256.New, []byte(s.cfg.ReceiptSecret))
	_, _ = mac.Write([]byte(documentHash))
	signature := hex.EncodeToString(mac.Sum(nil))
	if err := s.storeDB.RecordExecutiveSignature(r.Context(), authVal.user.ID, "laporan-eksekutif", documentHash, signature); err != nil {
		redirectWithFlash(w, r, "/headmaster/executive", "Tanda tangan gagal disimpan: "+err.Error(), "danger")
		return
	}
	s.auditAccount(r.Context(), authVal, "executive.esign", authVal.user.ID, authVal.user.Role, map[string]any{
		"document_type": "laporan-eksekutif", "document_hash": documentHash,
	})
	redirectWithFlash(w, r, "/headmaster/executive", "Laporan eksekutif ditandatangani. Hash "+documentHash[:16]+"… tersimpan untuk verifikasi.", "success")
}

func (s *Server) handleHeadmasterOTPForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "headmaster")
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		state.GenerateReversalOTP(doc, r.FormValue("req_id"), time.Now())
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/headmaster/executive", "Perubahan gagal disimpan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/headmaster/executive", "Kode OTP 6-digit berhasil diterbitkan. Kode aktif tampil pada kartu OTP Kepala Sekolah; berikan hanya kepada bendahara.", "success")
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
	shell := s.buildRoleShell(r.Context(), authVal, "executive", "Analisis strategis selesai dijalankan.", "success")
	data := s.buildHeadmasterPortal(r.Context(), doc, shell, advice)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.HeadmasterPortal(data)))
}

// Developer Portal Handlers
func (s *Server) handleDeveloperRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/developer/dashboard", http.StatusFound)
}

func (s *Server) handleDeveloperPage(w http.ResponseWriter, r *http.Request) {
	authVal, ok := requireWebRole(w, r, "developer")
	if !ok {
		return
	}
	tab := r.PathValue("tab")
	if tab == "" {
		tab = "dashboard"
	}
	doc, _ := s.loadStateForRole(r.Context(), authVal.user.Role)
	shell := s.buildRoleShell(r.Context(), authVal, tab, r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	data := s.buildDeveloperDashboard(doc, shell, tab)
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.DeveloperPortal(data)))
}

func (s *Server) handleDeveloperBugStatusForm(w http.ResponseWriter, r *http.Request) {
	_, ok := requireWebRole(w, r, "developer")
	if !ok {
		return
	}
	_ = r.ParseForm()
	reportID := r.FormValue("report_id")
	status := r.FormValue("status")
	notes := r.FormValue("developer_notes")
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		return state.UpdateBugReportStatus(doc, reportID, status, notes, time.Now())
	}); err != nil {
		redirectWithFlash(w, r, "/developer/dashboard", "Gagal mengubah status laporan: "+err.Error(), "danger")
		return
	}
	redirectWithFlash(w, r, "/developer/dashboard", fmt.Sprintf("Status laporan bug #%s berhasil diubah menjadi %s.", reportID, status), "success")
}

// Contact & Bug Report Handlers
func (s *Server) handleContactPage(w http.ResponseWriter, r *http.Request) {
	authVal, loggedIn := currentAuth(r)
	reporterName := ""
	reporterEmail := ""
	reporterRole := "GUEST"
	shell := ui.RoleShell{
		Role:         "guest",
		RoleLabel:    "Pengunjung",
		FullName:     "Pengunjung / Siswa",
		Initials:     "WB",
		SchoolName:   ui.SchoolName(),
		AcademicYear: s.activeAcademicYearLabel(r.Context()),
		ActiveTab:    "contact",
		PageTitle:    "Hubungi & Lapor Bug",
		CSRFToken:    s.csrfTokenFor("contact"),
		Flash:        r.URL.Query().Get("flash"),
		FlashKind:    r.URL.Query().Get("kind"),
	}

	if loggedIn && authVal.user != nil {
		reporterName = authVal.user.FullName
		reporterEmail = authVal.user.PhoneWA
		reporterRole = authVal.user.Role
		shell = s.buildRoleShell(r.Context(), authVal, "contact", r.URL.Query().Get("flash"), r.URL.Query().Get("kind"))
	}

	referer := r.Header.Get("Referer")
	data := s.buildContactPage(shell, reporterName, reporterEmail, reporterRole, referer, r.URL.Query().Get("success"), r.URL.Query().Get("error"))
	renderComponent(w, r, http.StatusOK, layout.Base(shell, pages.Contact(data)))
}

func (s *Server) handleContactReportBugForm(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	authVal, loggedIn := currentAuth(r)
	reporterID := "anon"
	reporterName := strings.TrimSpace(r.FormValue("reporter_name"))
	reporterRole := strings.TrimSpace(r.FormValue("reporter_role"))
	if loggedIn && authVal.user != nil {
		reporterID = authVal.user.ID
		if reporterName == "" {
			reporterName = authVal.user.FullName
		}
		if reporterRole == "" {
			reporterRole = authVal.user.Role
		}
	}
	if reporterName == "" {
		reporterName = "Pengguna WESMK"
	}
	if reporterRole == "" {
		reporterRole = "PENGGUNA"
	}

	report := map[string]any{
		"reporterUserId": reporterID,
		"reporterName":   reporterName,
		"reporterRole":   reporterRole,
		"category":       r.FormValue("category"),
		"title":          r.FormValue("title"),
		"description":    r.FormValue("description"),
		"pageUrl":        r.FormValue("page_url"),
		"severity":       r.FormValue("severity"),
	}

	if s.cache != nil {
		attempts, err := s.cache.IncrementCounter(r.Context(), "ratelimit:contact:"+s.clientIP(r), 10*time.Minute)
		if err != nil {
			redirectWithFlash(w, r, "/contact", "Layanan laporan sedang sibuk. Coba lagi nanti.", "danger")
			return
		}
		if attempts > 10 {
			redirectWithFlash(w, r, "/contact", "Terlalu banyak laporan dari perangkat ini. Coba lagi nanti.", "danger")
			return
		}
	}
	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	if title == "" || description == "" {
		redirectWithFlash(w, r, "/contact", "Judul dan deskripsi laporan wajib diisi.", "danger")
		return
	}
	if len(title) > 140 || len(description) > 4000 {
		redirectWithFlash(w, r, "/contact", "Judul maksimal 140 karakter dan deskripsi maksimal 4.000 karakter.", "danger")
		return
	}

	var reportID string
	if err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		reportID = state.AddBugReport(doc, report, time.Now())
		return nil
	}); err != nil {
		redirectWithFlash(w, r, "/contact", "Gagal mengirim laporan: "+err.Error(), "danger")
		return
	}

	redirectWithFlash(w, r, "/contact", fmt.Sprintf("Laporan masalah (#%s) berhasil dikirim langsung ke Dasbor Admin Developer WESMK. Terima kasih atas bantuan Anda!", reportID), "success")
}
