package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"wesmk/internal/model"
	"wesmk/internal/state"
	"wesmk/internal/store"
	"wesmk/internal/ui"
)

func (s *Server) csrfTokenFor(token string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.ReceiptSecret))
	mac.Write([]byte("csrf:" + token))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

func (s *Server) validCSRF(rToken, submitted string) bool {
	if rToken == "" || submitted == "" {
		return false
	}
	expected := s.csrfTokenFor(rToken)
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(submitted)))
}

// loadStateForRole reads the state document and applies the role's read scope
// before it reaches a page builder. Student/parent builders keep the full doc
// and filter by their own profile ID; staff builders only receive their keys.
func (s *Server) loadStateForRole(ctx context.Context, role string) (map[string]any, error) {
	doc, err := s.loadOrSeedState(ctx)
	if err != nil {
		return nil, err
	}
	if state.IsStaffRole(role) && role != "headmaster" {
		return state.StaffView(doc, role), nil
	}
	return doc, nil
}

// loadOrSeedState reads the global state document from Postgres, seeding it
// with the embedded default state if the database is empty.
func (s *Server) loadOrSeedState(ctx context.Context) (map[string]any, error) {
	raw, err := s.storeDB.GetState(ctx)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		defBytes := state.DefaultStateBytes()
		if putErr := s.storeDB.PutState(ctx, defBytes); putErr != nil {
			return state.DefaultStateMap()
		}
		raw = defBytes
	}
	doc, err := decodeState(raw)
	if err != nil {
		return state.DefaultStateMap()
	}
	return doc, nil
}

func (s *Server) mutateStateDoc(ctx context.Context, fn func(map[string]any) error) error {
	// Ensure state exists first
	if _, err := s.loadOrSeedState(ctx); err != nil {
		return err
	}
	actorID, role := "", ""
	if auth, ok := requestAuthFromContext(ctx); ok {
		actorID, role = auth.user.ID, auth.user.Role
	}
	var fnErr error
	updated, err := s.storeDB.UpdateStateAudited(ctx, store.StateAudit{
		ActorID: actorID,
		Role:    role,
		Action:  "web.form",
	}, func(raw json.RawMessage) (json.RawMessage, error) {
		doc, err := decodeState(raw)
		if err != nil {
			doc, _ = state.DefaultStateMap()
		}
		if err := fn(doc); err != nil {
			fnErr = err
			return nil, err
		}
		return json.Marshal(doc)
	})
	if fnErr != nil {
		return fnErr
	}
	if err == nil {
		s.syncDomainShadow(ctx, updated)
	}
	return err
}

func (s *Server) buildRoleShell(authVal requestAuth, activeTab, flash, flashKind string) ui.RoleShell {
	nav := ui.NavFor(authVal.user.Role)
	return ui.RoleShell{
		Role:         authVal.user.Role,
		RoleLabel:    ui.RoleLabel(authVal.user.Role),
		FullName:     authVal.user.FullName,
		Initials:     ui.Initials(authVal.user.FullName),
		Username:     authVal.user.Username,
		SchoolName:   "SMK Negeri 1 Rekayasa",
		AcademicYear: "2026/2027",
		ActiveTab:    activeTab,
		PageTitle:    ui.ActiveLabel(nav, activeTab),
		Nav:          nav,
		CSRFToken:    s.csrfTokenFor(authVal.token),
		DemoMode:     s.authSvc.DemoMode(),
		Flash:        flash,
		FlashKind:    flashKind,
	}
}

func demoAccountsList() []ui.DemoAccount {
	return []ui.DemoAccount{
		{Label: "Muhammad Budi Santoso", Identifier: "0058471290", Group: "Murid · Ketua Kelas XII TKJ 1"},
		{Label: "Siti Rahmawati", Identifier: "0058471291", Group: "Murid · Beasiswa KIP"},
		{Label: "Bapak Ahmad Santoso", Identifier: "081234567800", Group: "Orang Tua / Wali Budi"},
		{Label: "Ir. Hendro Wijaya", Identifier: "197804152005011003", Group: "Guru Produktif & Wali Kelas"},
		{Label: "Hj. Ratna Sari, S.E.", Identifier: "198509122010012005", Group: "Bendahara Sekolah"},
		{Label: "Drs. Bambang Sudarmono", Identifier: "197503112000031002", Group: "Kesiswaan & BK"},
		{Label: "Rian Pratama, S.Pd.", Identifier: "198811202014021004", Group: "Guru Piket Harian"},
		{Label: "Nurul Hidayati, A.Md.", Identifier: "199201052019032011", Group: "Admin TU & Dapodik"},
		{Label: "Dr. H. Sulaiman Effendi", Identifier: "196805121994121001", Group: "Kepala Sekolah"},
		{Label: "Tim Developer WESMK", Identifier: "developer.admin", Group: "Developer & Bug Tracker"},
	}
}

func asMapSlice(v any) []map[string]any {
	raw, _ := v.([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok && m != nil {
			out = append(out, m)
		}
	}
	return out
}

func strVal(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	switch v := m[k].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return ""
	}
}

func intVal(m map[string]any, k string) int {
	if m == nil {
		return 0
	}
	switch v := m[k].(type) {
	case float64:
		return int(math.Round(v))
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func floatVal(m map[string]any, k string) float64 {
	if m == nil {
		return 0
	}
	switch v := m[k].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func boolVal(m map[string]any, k string) bool {
	if m == nil {
		return false
	}
	b, _ := m[k].(bool)
	return b
}

func parseStudentView(m map[string]any) ui.StudentView {
	return ui.StudentView{
		ID:                strVal(m, "id"),
		FullName:          strVal(m, "fullName"),
		NISN:              strVal(m, "nisn"),
		NIK:               strVal(m, "nik"),
		ClassroomName:     strVal(m, "classroomName"),
		ParentName:        strVal(m, "parentName"),
		IsKetuaKelas:      boolVal(m, "isKetuaKelas"),
		IsSekretaris:      boolVal(m, "isSekretaris"),
		ScholarshipType:   strVal(m, "scholarshipType"),
		DiscountPercent:   intVal(m, "discountPercent"),
		AttitudeScore:     intVal(m, "attitudeScore"),
		DailyQuizAvg:      intVal(m, "dailyQuizAvg"),
		ActiveBadge:       strVal(m, "activeBadge"),
		ActiveBadgeReason: strVal(m, "activeBadgeReason"),
		HomeroomNote:      strVal(m, "homeroomNote"),
		BMWChoice:         strVal(m, "bmwChoice"),
	}
}

func parseInvoiceView(m map[string]any) ui.InvoiceView {
	finalAmt := intVal(m, "finalAmount")
	paidAmt := intVal(m, "paidAmount")
	rem := finalAmt - paidAmt
	if rem < 0 {
		rem = 0
	}
	status := strVal(m, "status")
	return ui.InvoiceView{
		ID:          strVal(m, "id"),
		Number:      strVal(m, "invoiceNumber"),
		Period:      strVal(m, "periodMonthYear"),
		Status:      status,
		StatusLabel: ui.StatusLabel(status),
		BaseAmount:  intVal(m, "baseAmount"),
		FinalAmount: finalAmt,
		PaidAmount:  paidAmt,
		Remaining:   rem,
		VANumber:    strVal(m, "vaNumber"),
		Provider:    strVal(m, "gatewayProvider"),
		IsPaid:      status == "PAID" || rem == 0,
	}
}

func findStudentMap(doc map[string]any, studentID string) map[string]any {
	students := asMapSlice(doc["students"])
	for _, s := range students {
		if strVal(s, "id") == studentID {
			return s
		}
	}
	if len(students) > 0 {
		return students[0]
	}
	return nil
}

func (s *Server) buildStudentDashboard(doc map[string]any, shell ui.RoleShell, user *model.User) ui.StudentDashboard {
	studentID := user.StudentProfileID
	if studentID == "" {
		studentID = "std-budi-01"
	}
	stdMap := findStudentMap(doc, studentID)
	student := parseStudentView(stdMap)

	var invoice ui.InvoiceView
	hasInvoice := false
	for _, invMap := range asMapSlice(doc["invoices"]) {
		if strVal(invMap, "studentId") == student.ID {
			invoice = parseInvoiceView(invMap)
			hasInvoice = true
			break
		}
	}

	hasDisp := false
	for _, dMap := range asMapSlice(doc["dispensations"]) {
		if strVal(dMap, "studentId") != student.ID {
			continue
		}
		if vu := strVal(dMap, "validUntil"); vu != "" {
			if t, err := time.Parse(time.RFC3339, vu); err == nil && time.Now().After(t) {
				continue
			}
		}
		hasDisp = true
		break
	}
	isCleared := !hasInvoice || invoice.IsPaid || hasDisp

	att := ui.AttendanceView{
		Status:      "HADIR_GERBANG",
		StatusLabel: ui.StatusLabel("HADIR_GERBANG"),
		Time:        "06:48 WIB",
		Source:      "Tap Kartu RFID Gerbang Utama",
	}
	for _, aMap := range asMapSlice(doc["attendances"]) {
		if strVal(aMap, "studentId") == student.ID {
			st := strVal(aMap, "status")
			att = ui.AttendanceView{
				Status:      st,
				StatusLabel: ui.StatusLabel(st),
				Time:        strVal(aMap, "gateTapTime"),
				Source:      strVal(aMap, "verifiedSource"),
			}
			break
		}
	}

	quizMap, _ := doc["dailyQuiz"].(map[string]any)
	qRaw := asMapSlice(quizMap["questions"])
	qViews := make([]ui.QuizQuestionView, 0, len(qRaw))
	for _, qm := range qRaw {
		opts := make([]ui.QuizOptionView, 0)
		for _, om := range asMapSlice(qm["options"]) {
			opts = append(opts, ui.QuizOptionView{
				Key:  strVal(om, "key"),
				Text: strVal(om, "text"),
			})
		}
		qViews = append(qViews, ui.QuizQuestionView{
			ID:      strVal(qm, "id"),
			Text:    strVal(qm, "questionText"),
			Options: opts,
		})
	}

	quizID := strVal(quizMap, "id")
	attempted := false
	quizScore := student.DailyQuizAvg
	for _, attMap := range asMapSlice(doc["quizAttempts"]) {
		if strVal(attMap, "studentId") == student.ID && strVal(attMap, "quizId") == quizID {
			attempted = true
			quizScore = intVal(attMap, "score")
			break
		}
	}

	quiz := ui.QuizView{
		ID:            quizID,
		Subject:       strVal(quizMap, "subjectName"),
		Topic:         strVal(quizMap, "topic"),
		QuestionCount: len(qViews),
		Deadline:      strVal(quizMap, "deadlineTime"),
		Average:       student.DailyQuizAvg,
		Attempted:     attempted,
		Score:         quizScore,
	}

	jMap, _ := doc["classJournal"].(map[string]any)
	leaderStatus := strVal(jMap, "leaderValidationStatus")
	if leaderStatus == "" {
		leaderStatus = "VALID_FULL"
	}
	journal := ui.JournalView{
		Subject:      strVal(jMap, "subjectName"),
		Topic:        strVal(jMap, "syllabusTopic"),
		TeacherName:  strVal(jMap, "teacherName"),
		Summary:      strVal(jMap, "teacherNotes"),
		LeaderStatus: leaderStatus,
	}

	return ui.StudentDashboard{
		Shell:           shell,
		Student:         student,
		Invoice:         invoice,
		HasInvoice:      hasInvoice,
		IsCleared:       isCleared,
		HasDispensation: hasDisp,
		Attendance:      att,
		Quiz:            quiz,
		QuizQuestions:   qViews,
		Journal:         journal,
		ShowLeaderPanel: student.IsKetuaKelas || student.IsSekretaris,
		LeaderStatus:    leaderStatus,
	}
}

func (s *Server) buildStudentPortal(doc map[string]any, shell ui.RoleShell, user *model.User, tab string, demoGraded ...bool) ui.StudentPortalPage {
	isDemoGraded := len(demoGraded) > 0 && demoGraded[0]
	dash := s.buildStudentDashboard(doc, shell, user)
	stdMap := findStudentMap(doc, dash.Student.ID)
	cbtScore := intVal(stdMap, "cbtScore")
	attScore := intVal(stdMap, "attendanceConductScore")
	holistic := state.CalculateHolisticScore(
		float64(cbtScore),
		float64(dash.Student.DailyQuizAvg),
		float64(dash.Student.AttitudeScore),
		float64(attScore),
	)

	jMap, _ := doc["classJournal"].(map[string]any)
	refMap, _ := doc["studentReflection"].(map[string]any)

	examQs := make([]ui.ExamQuestionView, 0)
	for _, qm := range asMapSlice(doc["examQuestions"]) {
		opts := make([]ui.QuizOptionView, 0)
		for _, om := range asMapSlice(qm["options"]) {
			opts = append(opts, ui.QuizOptionView{Key: strVal(om, "key"), Text: strVal(om, "text")})
		}
		qText := strVal(qm, "questionText")
		if qText == "" {
			qText = strVal(qm, "templateContent")
			if dynVars, ok := qm["dynamicVars"].(map[string]any); ok {
				for k, v := range dynVars {
					placeholder := "{{" + k + "}}"
					qText = strings.ReplaceAll(qText, placeholder, fmt.Sprintf("%v", v))
				}
			}
		}
		tag := strVal(qm, "kdCode")
		if tag == "" {
			tag = strVal(qm, "competencyTag")
		}
		examQs = append(examQs, ui.ExamQuestionView{
			ID:             strVal(qm, "id"),
			QuestionNumber: intVal(qm, "questionNumber"),
			QuestionType:   strVal(qm, "questionType"),
			SubjectName:    strVal(qm, "subjectName"),
			KdCode:         tag,
			QuestionText:   qText,
			Options:        opts,
		})
	}

	proctor := ui.ProctorSessionView{
		StudentID: dash.Student.ID,
		Name:      dash.Student.FullName,
		NISN:      dash.Student.NISN,
		Status:    "ACTIVE",
		Progress:  "0/40",
	}
	for _, pm := range asMapSlice(doc["proctorStudents"]) {
		if strVal(pm, "studentId") == dash.Student.ID {
			proctor = ui.ProctorSessionView{
				ID:            strVal(pm, "id"),
				StudentID:     strVal(pm, "studentId"),
				Name:          strVal(pm, "name"),
				NISN:          strVal(pm, "nisn"),
				Progress:      strVal(pm, "progress"),
				Status:        strVal(pm, "status"),
				WarnCount:     intVal(pm, "warnCount"),
				LockCount:     intVal(pm, "lockCount"),
				LastViolation: strVal(pm, "lastViolation"),
				UnlockToken:   strVal(pm, "unlockToken"),
				Emergency2h:   boolVal(pm, "hasEmergency2hTicket"),
			}
			break
		}
	}

	submittedScores := map[string]int{}
	examScore := cbtScore
	for _, sub := range asMapSlice(doc["examSubmissions"]) {
		if strVal(sub, "studentId") == dash.Student.ID {
			sc := intVal(sub, "mcqScore")
			examScore = sc
			eid := strVal(sub, "examId")
			if eid == "" || eid == "exam-cbt-tkj-01" || eid == "exam-active" || eid == "exam-sch-01" {
				submittedScores["exam-sch-01"] = sc
			} else {
				submittedScores[eid] = sc
			}
		}
	}

	sch1Done := false
	sch1Score := 0
	if sc, ok := submittedScores["exam-sch-01"]; ok {
		sch1Done = true
		sch1Score = sc
	}
	sch2Done := false
	sch2Score := 0
	if sc, ok := submittedScores["exam-sch-02"]; ok {
		sch2Done = true
		sch2Score = sc
	}
	sch4Done := false
	sch4Score := 0
	if sc, ok := submittedScores["exam-sch-04"]; ok {
		sch4Done = true
		sch4Score = sc
	}

	schedules := make([]ui.ExamScheduleView, 0, 4)
	if isDemoGraded {
		if sch1Score <= 0 {
			sch1Score = 96
		}
		if sch2Score <= 0 {
			sch2Score = 94
		}
		if sch4Score <= 0 {
			sch4Score = 90
		}
		schedules = []ui.ExamScheduleView{
			{
				ID:            "exam-sch-01",
				SubjectName:   "Administrasi Infrastruktur Jaringan",
				TeacherName:   "Ir. Hendro Wijaya",
				DateLabel:     "Hari Ini",
				TimeSlot:      "08.00 - 09.30 WIB",
				DurationMin:   90,
				QuestionCount: len(examQs),
				Room:          "Lab TKJ 1",
				SessionNumber: 1,
				Status:        "SUBMITTED",
				StatusBadge:   "Selesai",
				BadgeClass:    "badge-teal",
				Score:         sch1Score,
			},
			{
				ID:            "exam-sch-02",
				SubjectName:   "Keamanan Jaringan & Sistem Otomasi",
				TeacherName:   "Ir. Hendro Wijaya",
				DateLabel:     "Hari Ini",
				TimeSlot:      "10.00 - 11.30 WIB",
				DurationMin:   90,
				QuestionCount: len(examQs),
				Room:          "Lab TKJ 1",
				SessionNumber: 2,
				Status:        "SUBMITTED",
				StatusBadge:   "Selesai",
				BadgeClass:    "badge-teal",
				Score:         sch2Score,
			},
			{
				ID:            "exam-sch-03",
				SubjectName:   "Teknologi Layanan Jaringan",
				TeacherName:   "Rian Pratama, S.Pd.",
				DateLabel:     "Kemarin",
				TimeSlot:      "08.00 - 09.30 WIB",
				DurationMin:   90,
				QuestionCount: len(examQs),
				Room:          "Lab TKJ 2",
				SessionNumber: 1,
				Status:        "SUBMITTED",
				StatusBadge:   "Selesai",
				BadgeClass:    "badge-teal",
				Score:         92,
			},
			{
				ID:            "exam-sch-04",
				SubjectName:   "Produk Kreatif & Kewirausahaan",
				TeacherName:   "Hj. Ratna Sari, S.E.",
				DateLabel:     "Besok",
				TimeSlot:      "08.00 - 09.30 WIB",
				DurationMin:   90,
				QuestionCount: len(examQs),
				Room:          "R. Teori 12",
				SessionNumber: 1,
				Status:        "SUBMITTED",
				StatusBadge:   "Selesai",
				BadgeClass:    "badge-teal",
				Score:         sch4Score,
			},
		}
	} else {
		sch1Status := "READY"
		sch1Badge := "Bisa Dikerjakan"
		sch1Class := "badge-success"
		sch1URL := "/student/exam-kiosk?subject=exam-sch-01"
		if sch1Done {
			sch1Status = "SUBMITTED"
			sch1Badge = "Selesai"
			sch1Class = "badge-teal"
			sch1URL = ""
		}

		sch2Status := "WAITING"
		sch2Badge := "Menunggu Giliran"
		sch2Class := "badge-warning"
		sch2URL := ""
		if sch2Done {
			sch2Status = "SUBMITTED"
			sch2Badge = "Selesai"
			sch2Class = "badge-teal"
		} else if sch1Done {
			sch2Status = "READY"
			sch2Badge = "Bisa Dikerjakan"
			sch2Class = "badge-success"
			sch2URL = "/student/exam-kiosk?subject=exam-sch-02"
		}

		sch4Status := "UPCOMING"
		sch4Badge := "Belum Dibuka"
		sch4Class := "badge-neutral"
		sch4URL := ""
		if sch4Done {
			sch4Status = "SUBMITTED"
			sch4Badge = "Selesai"
			sch4Class = "badge-teal"
		} else if sch1Done && sch2Done {
			sch4Status = "READY"
			sch4Badge = "Bisa Dikerjakan"
			sch4Class = "badge-success"
			sch4URL = "/student/exam-kiosk?subject=exam-sch-04"
		}

		schedules = []ui.ExamScheduleView{
			{
				ID:            "exam-sch-01",
				SubjectName:   "Administrasi Infrastruktur Jaringan",
				TeacherName:   "Ir. Hendro Wijaya",
				DateLabel:     "Hari Ini · Sesi 1",
				TimeSlot:      "08.00 - 09.30 WIB",
				DurationMin:   90,
				QuestionCount: len(examQs),
				Room:          "Lab TKJ 1",
				SessionNumber: 1,
				Status:        sch1Status,
				StatusBadge:   sch1Badge,
				BadgeClass:    sch1Class,
				Score:         sch1Score,
				ActionURL:     sch1URL,
			},
			{
				ID:            "exam-sch-02",
				SubjectName:   "Keamanan Jaringan & Sistem Otomasi",
				TeacherName:   "Ir. Hendro Wijaya",
				DateLabel:     "Hari Ini · Sesi 2",
				TimeSlot:      "10.00 - 11.30 WIB",
				DurationMin:   90,
				QuestionCount: len(examQs),
				Room:          "Lab TKJ 1",
				SessionNumber: 2,
				Status:        sch2Status,
				StatusBadge:   sch2Badge,
				BadgeClass:    sch2Class,
				Score:         sch2Score,
				ActionURL:     sch2URL,
			},
			{
				ID:            "exam-sch-03",
				SubjectName:   "Teknologi Layanan Jaringan",
				TeacherName:   "Rian Pratama, S.Pd.",
				DateLabel:     "Kemarin · Sesi 1",
				TimeSlot:      "08.00 - 09.30 WIB",
				DurationMin:   90,
				QuestionCount: len(examQs),
				Room:          "Lab TKJ 2",
				SessionNumber: 1,
				Status:        "SUBMITTED",
				StatusBadge:   "Selesai",
				BadgeClass:    "badge-teal",
				Score:         92,
			},
			{
				ID:            "exam-sch-04",
				SubjectName:   "Produk Kreatif & Kewirausahaan",
				TeacherName:   "Hj. Ratna Sari, S.E.",
				DateLabel:     "Jadwal Lanjutan",
				TimeSlot:      "08.00 - 09.30 WIB",
				DurationMin:   90,
				QuestionCount: len(examQs),
				Room:          "R. Teori 12",
				SessionNumber: 1,
				Status:        sch4Status,
				StatusBadge:   sch4Badge,
				BadgeClass:    sch4Class,
				Score:         sch4Score,
				ActionURL:     sch4URL,
			},
		}
	}

	activeExamID := "exam-sch-01"
	activeSubjectName := "Administrasi Infrastruktur Jaringan"
	activeSubjectTeacher := "Ir. Hendro Wijaya"
	activeSubjectRoom := "Lab TKJ 1"
	activeSubjectTime := "08.00 - 09.30 WIB"
	hasReadyExam := false
	completedExamsCount := 0
	for _, sc := range schedules {
		if sc.Status == "SUBMITTED" {
			completedExamsCount++
		} else if sc.Status == "READY" && !hasReadyExam {
			hasReadyExam = true
			activeExamID = sc.ID
			activeSubjectName = sc.SubjectName
			activeSubjectTeacher = sc.TeacherName
			activeSubjectRoom = sc.Room
			activeSubjectTime = sc.TimeSlot
		}
	}

	examSubmitted := !hasReadyExam && completedExamsCount == len(schedules)
	if hasReadyExam && proctor.Status == "SUBMITTED" {
		proctor.Status = "ACTIVE"
	}
	for i := range examQs {
		examQs[i].SubjectName = activeSubjectName
	}

	docs := make([]ui.StudentDocView, 0)
	for _, dm := range asMapSlice(doc["studentDocuments"]) {
		if strVal(dm, "studentId") == dash.Student.ID {
			docs = append(docs, parseStudentDoc(dm))
		}
	}

	tasks := make([]ui.TaskSubmissionView, 0)
	for _, tm := range asMapSlice(doc["taskSubmissions"]) {
		if strVal(tm, "studentId") == dash.Student.ID {
			tasks = append(tasks, ui.TaskSubmissionView{
				ID:          strVal(tm, "id"),
				StudentID:   strVal(tm, "studentId"),
				StudentName: strVal(tm, "studentName"),
				SubjectName: strVal(tm, "subjectName"),
				TaskTitle:   strVal(tm, "taskTitle"),
				FileName:    strVal(tm, "fileName"),
				FileSizeKB:  intVal(tm, "fileSizeKb"),
				SubmittedAt: strVal(tm, "submittedAt"),
				Status:      strVal(tm, "status"),
				Score:       intVal(tm, "score"),
			})
		}
	}

	var pklPlacement ui.PklPlacementView
	if list := asMapSlice(doc["pklPlacements"]); len(list) > 0 {
		pklPlacement = parsePklPlacement(list[0])
	}

	pklLogs := make([]ui.PklLogView, 0)
	for _, lm := range asMapSlice(doc["pklAttendanceLogs"]) {
		if strVal(lm, "studentId") == dash.Student.ID {
			pklLogs = append(pklLogs, parsePklLog(lm))
		}
	}

	pklJournals := make([]ui.PklJournalView, 0)
	for _, jm := range asMapSlice(doc["pklJournals"]) {
		if strVal(jm, "studentId") == dash.Student.ID {
			pklJournals = append(pklJournals, parsePklJournal(jm))
		}
	}

	appliedJobs := map[string]bool{}
	bkkApps := make([]ui.BkkAppView, 0)
	for _, am := range asMapSlice(doc["bkkApplications"]) {
		app := parseBkkApp(am)
		if app.StudentID == dash.Student.ID {
			appliedJobs[app.JobID] = true
			bkkApps = append(bkkApps, app)
		}
	}

	bkkJobs := make([]ui.BkkJobView, 0)
	for _, jm := range asMapSlice(doc["bkkJobs"]) {
		job := parseBkkJob(jm)
		job.AlreadyApplied = appliedJobs[job.ID]
		bkkJobs = append(bkkJobs, job)
	}

	expenses := make([]ui.StudentExpenseView, 0)
	totalExp := 0
	for _, em := range asMapSlice(doc["studentExpenses"]) {
		if strVal(em, "studentId") == dash.Student.ID {
			e := ui.StudentExpenseView{
				ID:          strVal(em, "id"),
				StudentID:   strVal(em, "studentId"),
				Category:    strVal(em, "category"),
				ItemTitle:   strVal(em, "itemTitle"),
				Amount:      intVal(em, "amount"),
				Date:        strVal(em, "date"),
				ReceiptNote: strVal(em, "receiptNote"),
			}
			totalExp += e.Amount
			expenses = append(expenses, e)
		}
	}

	allGraded := len(schedules) > 0
	totalScore := 0
	gradedCount := 0
	for _, sc := range schedules {
		if sc.Status != "SUBMITTED" {
			allGraded = false
		} else {
			totalScore += sc.Score
			gradedCount++
		}
	}
	avgScore := 0
	if allGraded && len(schedules) > 0 {
		avgScore = totalScore / len(schedules)
	}
	tier := "stellar"
	if avgScore >= 91 {
		tier = "stellar"
	} else if avgScore >= 81 {
		tier = "gold"
	} else if avgScore >= 71 {
		tier = "blue"
	} else {
		tier = "green"
	}

	return ui.StudentPortalPage{
		Shell:                  shell,
		Tab:                    tab,
		Student:                dash.Student,
		CbtScore:               cbtScore,
		AttendanceConductScore: attScore,
		HolisticScore:          fmt.Sprintf("%.1f", holistic),
		Invoice:                dash.Invoice,
		HasInvoice:             dash.HasInvoice,
		IsCleared:              dash.IsCleared,
		HasDispensation:        dash.HasDispensation,
		Journal:                dash.Journal,
		TeacherNotes:           strVal(jMap, "teacherNotes"),
		AiSmartNotes:           strVal(jMap, "aiSmartNotes"),
		Quiz:                   dash.Quiz,
		QuizQuestions:          dash.QuizQuestions,
		ReflectionLevel:        strVal(refMap, "understandingLevel"),
		ReflectionComplaint:    strVal(refMap, "privateComplaintText"),
		ExamQuestions:          examQs,
		ExamSchedules:          schedules,
		ActiveExamID:           activeExamID,
		ActiveSubjectName:      activeSubjectName,
		ActiveSubjectTeacher:   activeSubjectTeacher,
		ActiveSubjectRoom:      activeSubjectRoom,
		ActiveSubjectTime:      activeSubjectTime,
		CompletedExamsCount:    completedExamsCount,
		TotalExamsCount:        len(schedules),
		ExamAverageScore:       avgScore,
		AllExamsGraded:         allGraded,
		ExamTier:               tier,
		IsDemoGraded:           isDemoGraded,
		Proctor:                proctor,
		ExamSubmitted:          examSubmitted,
		ExamScore:              examScore,
		Documents:              docs,
		TaskSubmissions:        tasks,
		PklPlacement:           pklPlacement,
		PklLogs:                pklLogs,
		PklJournals:            pklJournals,
		BkkJobs:                bkkJobs,
		BkkApps:                bkkApps,
		Expenses:               expenses,
		TotalExpense:           totalExp,
	}
}

func (s *Server) buildParentPortal(doc map[string]any, shell ui.RoleShell, user *model.User, tab string) ui.ParentPortalPage {
	sp := s.buildStudentPortal(doc, shell, user, "dashboard")
	dash := s.buildStudentDashboard(doc, shell, user)

	invs := make([]ui.InvoiceView, 0)
	ownInvIDs := map[string]bool{}
	for _, im := range asMapSlice(doc["invoices"]) {
		if strVal(im, "studentId") == sp.Student.ID {
			iv := parseInvoiceView(im)
			ownInvIDs[iv.ID] = true
			invs = append(invs, iv)
		}
	}

	ledger := make([]ui.LedgerEntryView, 0)
	for _, lm := range asMapSlice(doc["ledger"]) {
		entry := parseLedgerEntry(lm)
		if ownInvIDs[entry.InvoiceID] {
			ledger = append(ledger, entry)
		}
	}

	consults := make([]ui.ConsultationView, 0)
	for _, cm := range asMapSlice(doc["careerConsultations"]) {
		if strVal(cm, "studentId") == sp.Student.ID {
			consults = append(consults, parseConsultation(cm))
		}
	}

	refLevel := sp.ReflectionLevel
	if refLevel == "" {
		refLevel = "UNDERSTOOD"
	}

	return ui.ParentPortalPage{
		Shell:                  shell,
		Tab:                    tab,
		Student:                sp.Student,
		CbtScore:               sp.CbtScore,
		AttendanceConductScore: sp.AttendanceConductScore,
		HolisticScore:          sp.HolisticScore,
		Attendance:             dash.Attendance,
		ReflectionLevel:        refLevel,
		ReflectionComplaint:    sp.ReflectionComplaint,
		Invoices:               invs,
		Ledger:                 ledger,
		Expenses:               sp.Expenses,
		TotalExpense:           sp.TotalExpense,
		Consultations:          consults,
	}
}

func (s *Server) buildTreasurerPortal(doc map[string]any, shell ui.RoleShell, tab string) ui.TreasurerPortalPage {
	students := make([]ui.StudentView, 0)
	for _, sm := range asMapSlice(doc["students"]) {
		students = append(students, parseStudentView(sm))
	}

	invoices := make([]ui.InvoiceView, 0)
	totalBilled := 0
	totalCollected := 0
	for _, im := range asMapSlice(doc["invoices"]) {
		iv := parseInvoiceView(im)
		totalBilled += iv.FinalAmount
		totalCollected += iv.PaidAmount
		invoices = append(invoices, iv)
	}
	totalArrears := totalBilled - totalCollected
	if totalArrears < 0 {
		totalArrears = 0
	}

	disps := make([]ui.DispensationView, 0)
	for _, dm := range asMapSlice(doc["dispensations"]) {
		disps = append(disps, ui.DispensationView{
			ID:            strVal(dm, "id"),
			StudentID:     strVal(dm, "studentId"),
			StudentName:   strVal(dm, "studentName"),
			ExamPeriod:    strVal(dm, "examPeriod"),
			ClearanceType: strVal(dm, "clearanceType"),
			Reason:        strVal(dm, "reasonCategory"),
			ValidUntil:    strVal(dm, "validUntil"),
			GrantedBy:     strVal(dm, "grantedBy"),
		})
	}

	ledger := make([]ui.LedgerEntryView, 0)
	for _, lm := range asMapSlice(doc["ledger"]) {
		ledger = append(ledger, parseLedgerEntry(lm))
	}

	revs := make([]ui.ReversalRequestView, 0)
	for _, rm := range asMapSlice(doc["reversalRequests"]) {
		revs = append(revs, parseReversalReq(rm))
	}

	return ui.TreasurerPortalPage{
		Shell:            shell,
		Tab:              tab,
		TotalBilled:      totalBilled,
		TotalCollected:   totalCollected,
		TotalArrears:     totalArrears,
		CollectionRate:   ui.Pct(totalCollected, totalBilled),
		Students:         students,
		Invoices:         invoices,
		Dispensations:    disps,
		Ledger:           ledger,
		ReversalRequests: revs,
	}
}

func (s *Server) buildTeacherPortal(doc map[string]any, shell ui.RoleShell, tab string) ui.TeacherPortalPage {
	stdMap := findStudentMap(doc, "std-budi-01")
	activeStudent := parseStudentView(stdMap)
	cbtScore := intVal(stdMap, "cbtScore")

	jMap, _ := doc["classJournal"].(map[string]any)
	journal := ui.JournalView{
		Subject:      strVal(jMap, "subjectName"),
		Topic:        strVal(jMap, "syllabusTopic"),
		TeacherName:  strVal(jMap, "teacherName"),
		Summary:      strVal(jMap, "teacherNotes"),
		LeaderStatus: strVal(jMap, "leaderValidationStatus"),
	}

	atts := make([]ui.AttendanceRowView, 0)
	for _, am := range asMapSlice(doc["attendances"]) {
		st := strVal(am, "status")
		atts = append(atts, ui.AttendanceRowView{
			ID:             strVal(am, "id"),
			StudentID:      strVal(am, "studentId"),
			StudentName:    strVal(am, "studentName"),
			NISN:           strVal(am, "nisn"),
			Status:         st,
			StatusLabel:    ui.StatusLabel(st),
			GateTapTime:    strVal(am, "gateTapTime"),
			VerifiedSource: strVal(am, "verifiedSource"),
		})
	}

	examQs := make([]ui.ExamQuestionView, 0)
	for _, qm := range asMapSlice(doc["examQuestions"]) {
		qText := strVal(qm, "questionText")
		if qText == "" {
			qText = strVal(qm, "templateContent")
			if dynVars, ok := qm["dynamicVars"].(map[string]any); ok {
				for k, v := range dynVars {
					placeholder := "{{" + k + "}}"
					qText = strings.ReplaceAll(qText, placeholder, fmt.Sprintf("%v", v))
				}
			}
		}
		tag := strVal(qm, "kdCode")
		if tag == "" {
			tag = strVal(qm, "competencyTag")
		}
		examQs = append(examQs, ui.ExamQuestionView{
			ID:             strVal(qm, "id"),
			QuestionNumber: intVal(qm, "questionNumber"),
			QuestionType:   strVal(qm, "questionType"),
			SubjectName:    strVal(qm, "subjectName"),
			KdCode:         tag,
			QuestionText:   qText,
			CorrectKey:     strVal(qm, "correctKey"),
			EssayRubric:    strVal(qm, "essayRubric"),
		})
	}

	proctors := make([]ui.ProctorSessionView, 0)
	for _, pm := range asMapSlice(doc["proctorStudents"]) {
		proctors = append(proctors, ui.ProctorSessionView{
			ID:            strVal(pm, "id"),
			StudentID:     strVal(pm, "studentId"),
			Name:          strVal(pm, "name"),
			NISN:          strVal(pm, "nisn"),
			Progress:      strVal(pm, "progress"),
			Status:        strVal(pm, "status"),
			WarnCount:     intVal(pm, "warnCount"),
			LockCount:     intVal(pm, "lockCount"),
			LastViolation: strVal(pm, "lastViolation"),
			UnlockToken:   strVal(pm, "unlockToken"),
			Emergency2h:   boolVal(pm, "hasEmergency2hTicket"),
		})
	}

	pklLogs := make([]ui.PklLogView, 0)
	for _, lm := range asMapSlice(doc["pklAttendanceLogs"]) {
		pklLogs = append(pklLogs, parsePklLog(lm))
	}

	pklJournals := make([]ui.PklJournalView, 0)
	for _, jm := range asMapSlice(doc["pklJournals"]) {
		pklJournals = append(pklJournals, parsePklJournal(jm))
	}

	resetLogs := make([]ui.ExamResetLogView, 0)
	for _, rm := range asMapSlice(doc["examResetLogs"]) {
		resetLogs = append(resetLogs, ui.ExamResetLogView{
			ID:          strVal(rm, "id"),
			StudentID:   strVal(rm, "studentId"),
			StudentName: strVal(rm, "studentName"),
			SubjectID:   strVal(rm, "subjectId"),
			ProctorName: strVal(rm, "proctorName"),
			Reason:      strVal(rm, "reason"),
			IsBulk:      boolVal(rm, "isBulk"),
			CreatedAt:   strVal(rm, "createdAt"),
		})
	}

	return ui.TeacherPortalPage{
		Shell:           shell,
		Tab:             tab,
		ActiveStudent:   activeStudent,
		CbtScore:        cbtScore,
		Journal:         journal,
		TeacherNotes:    strVal(jMap, "teacherNotes"),
		AiSmartNotes:    strVal(jMap, "aiSmartNotes"),
		Attendances:     atts,
		ExamQuestions:   examQs,
		ProctorStudents: proctors,
		ExamResetLogs:   resetLogs,
		PklLogs:         pklLogs,
		PklJournals:     pklJournals,
	}
}

func (s *Server) buildDisciplinePortal(doc map[string]any, shell ui.RoleShell, tab string) ui.DisciplinePortalPage {
	viols := make([]ui.ViolationView, 0)
	for _, vm := range asMapSlice(doc["violations"]) {
		viols = append(viols, ui.ViolationView{
			ID:               strVal(vm, "id"),
			StudentID:        strVal(vm, "studentId"),
			StudentName:      strVal(vm, "studentName"),
			ArticleCode:      strVal(vm, "articleCode"),
			ArticleTitle:     strVal(vm, "articleTitle"),
			SourceType:       strVal(vm, "sourceType"),
			PointsDeducted:   intVal(vm, "pointsDeducted"),
			Status:           strVal(vm, "status"),
			SummonLetterHash: strVal(vm, "summonLetterHash"),
			CreatedAt:        strVal(vm, "createdAt"),
		})
	}

	prs := make([]ui.PeerReportView, 0)
	for _, pm := range asMapSlice(doc["peerReports"]) {
		prs = append(prs, ui.PeerReportView{
			ID:                   strVal(pm, "id"),
			TargetStudentName:    strVal(pm, "targetStudentName"),
			TargetStudentClass:   strVal(pm, "targetStudentClass"),
			ArticleCategory:      strVal(pm, "articleCategory"),
			Location:             strVal(pm, "location"),
			Description:          strVal(pm, "description"),
			ReporterClusterHash:  strVal(pm, "reporterClusterHash"),
			DistinctClusterCount: intVal(pm, "distinctClusterCount"),
			Status:               strVal(pm, "status"),
			CreatedAt:            strVal(pm, "createdAt"),
		})
	}

	consults := make([]ui.ConsultationView, 0)
	for _, cm := range asMapSlice(doc["careerConsultations"]) {
		consults = append(consults, parseConsultation(cm))
	}

	csList := make([]ui.CounselingView, 0)
	recovered := 8
	relapsed := 1
	active := 0
	for _, cm := range asMapSlice(doc["counselingSessions"]) {
		st := strVal(cm, "recoveryStatus")
		if st == "RECOVERED_SUCCESS" {
			recovered++
		} else if st == "RELAPSED_FAIL" {
			relapsed++
		} else {
			active++
		}
		csList = append(csList, ui.CounselingView{
			ID:                     strVal(cm, "id"),
			StudentID:              strVal(cm, "studentId"),
			StudentName:            strVal(cm, "studentName"),
			CounselorName:          strVal(cm, "counselorName"),
			IssueSummary:           strVal(cm, "issueSummary"),
			StartDate:              strVal(cm, "startMonitoringDate"),
			EndDate:                strVal(cm, "endMonitoringDate"),
			RecoveryStatus:         st,
			DailyCheckinsCompleted: intVal(cm, "dailyCheckinsCompleted"),
			Notes:                  strVal(cm, "notes"),
		})
	}

	tScore, tTrend := state.CalculateTransformationIndex(active, recovered, relapsed)

	return ui.DisciplinePortalPage{
		Shell:               shell,
		Tab:                 tab,
		Violations:          viols,
		PeerReports:         prs,
		Consultations:       consults,
		CounselingSessions:  csList,
		TransformationScore: fmt.Sprintf("%.1f", tScore),
		TransformationTrend: tTrend,
		RecoveredCount:      recovered,
		ActiveCount:         active,
	}
}

func (s *Server) buildPiketPortal(doc map[string]any, shell ui.RoleShell) ui.PiketPortalPage {
	clsList := make([]ui.PiketClassroomView, 0)
	for _, cm := range asMapSlice(doc["piketClassrooms"]) {
		clsList = append(clsList, ui.PiketClassroomView{
			ID:      strVal(cm, "id"),
			Name:    strVal(cm, "name"),
			Subject: strVal(cm, "subject"),
			Teacher: strVal(cm, "teacher"),
			Status:  strVal(cm, "status"),
			Note:    strVal(cm, "note"),
		})
	}

	gateLogs := make([]ui.GateLogView, 0)
	for _, gm := range asMapSlice(doc["piketGateLogs"]) {
		gateLogs = append(gateLogs, ui.GateLogView{
			ID:          strVal(gm, "id"),
			NISN:        strVal(gm, "nisn"),
			StudentName: strVal(gm, "studentName"),
			LogType:     strVal(gm, "logType"),
			Reason:      strVal(gm, "reason"),
			RecordedAt:  strVal(gm, "recordedAt"),
			QrPassHash:  strVal(gm, "qrPassHash"),
		})
	}

	return ui.PiketPortalPage{
		Shell:      shell,
		Classrooms: clsList,
		GateLogs:   gateLogs,
		Teachers:   parseTeachers(doc),
	}
}

func (s *Server) buildAdminTuPortal(doc map[string]any, shell ui.RoleShell, tab string) ui.AdminTuPortalPage {
	students := make([]ui.StudentView, 0)
	for _, sm := range asMapSlice(doc["students"]) {
		students = append(students, parseStudentView(sm))
	}

	majors := make([]ui.MajorView, 0)
	for _, mm := range asMapSlice(doc["majors"]) {
		majors = append(majors, ui.MajorView{
			Code:             strVal(mm, "code"),
			Name:             strVal(mm, "name"),
			HeadOfMajor:      strVal(mm, "headOfMajor"),
			WorkshopRoom:     strVal(mm, "workshopRoom"),
			StudentCount:     intVal(mm, "studentCount"),
			DudiPartnerCount: intVal(mm, "dudiPartnerCount"),
		})
	}

	docs := make([]ui.StudentDocView, 0)
	for _, dm := range asMapSlice(doc["studentDocuments"]) {
		docs = append(docs, parseStudentDoc(dm))
	}

	letters := make([]ui.LetterView, 0)
	for _, lm := range asMapSlice(doc["officialLetters"]) {
		letters = append(letters, parseLetter(lm))
	}

	dapodik := make([]ui.DapodikLogView, 0)
	for _, dm := range asMapSlice(doc["dapodikLogs"]) {
		dapodik = append(dapodik, ui.DapodikLogView{
			ID:            strVal(dm, "id"),
			SyncTimestamp: strVal(dm, "syncTimestamp"),
			EntitySynced:  strVal(dm, "entitySynced"),
			RecordsCount:  intVal(dm, "recordsCount"),
			Status:        strVal(dm, "status"),
		})
	}

	placements := make([]ui.PklPlacementView, 0)
	for _, pm := range asMapSlice(doc["pklPlacements"]) {
		placements = append(placements, parsePklPlacement(pm))
	}

	jobs := make([]ui.BkkJobView, 0)
	for _, jm := range asMapSlice(doc["bkkJobs"]) {
		jobs = append(jobs, parseBkkJob(jm))
	}

	apps := make([]ui.BkkAppView, 0)
	for _, am := range asMapSlice(doc["bkkApplications"]) {
		apps = append(apps, parseBkkApp(am))
	}

	return ui.AdminTuPortalPage{
		Shell:         shell,
		Tab:           tab,
		Students:      students,
		Majors:        majors,
		Teachers:      parseTeachers(doc),
		Documents:     docs,
		Letters:       letters,
		DapodikLogs:   dapodik,
		PklPlacements: placements,
		BkkJobs:       jobs,
		BkkApps:       apps,
	}
}

func (s *Server) buildHeadmasterPortal(doc map[string]any, shell ui.RoleShell, aiAdvice string) ui.HeadmasterPortalPage {
	disc := s.buildDisciplinePortal(doc, shell, "transformation-index")
	revs := make([]ui.ReversalRequestView, 0)
	for _, rm := range asMapSlice(doc["reversalRequests"]) {
		revs = append(revs, parseReversalReq(rm))
	}
	otp := strVal(doc, "activeHeadmasterOtp")
	teachers := parseTeachers(doc)
	sort.Slice(teachers, func(i, j int) bool {
		return teachers[i].FinalImpactScore > teachers[j].FinalImpactScore
	})

	return ui.HeadmasterPortalPage{
		Shell:               shell,
		Teachers:            teachers,
		ReversalRequests:    revs,
		CounselingSessions:  disc.CounselingSessions,
		TransformationScore: disc.TransformationScore,
		TransformationTrend: disc.TransformationTrend,
		ActiveOTP:           otp,
		AiAdvice:            aiAdvice,
	}
}

func parseTeachers(doc map[string]any) []ui.TeacherImpactView {
	out := make([]ui.TeacherImpactView, 0)
	for _, tm := range asMapSlice(doc["teacherImpacts"]) {
		out = append(out, ui.TeacherImpactView{
			ID:                    strVal(tm, "id"),
			TeacherName:           strVal(tm, "teacherName"),
			NIP:                   strVal(tm, "nip"),
			Subject:               strVal(tm, "subject"),
			CalibratedScoreGrowth: floatVal(tm, "calibratedScoreGrowth"),
			LowConfusionRate:      floatVal(tm, "lowConfusionRate"),
			LeaderValidationRate:  floatVal(tm, "leaderValidationRate"),
			SubstituteBonusPoints: floatVal(tm, "substituteBonusPoints"),
			FinalImpactScore:      floatVal(tm, "finalImpactScore"),
		})
	}
	return out
}

func parseStudentDoc(dm map[string]any) ui.StudentDocView {
	return ui.StudentDocView{
		ID:           strVal(dm, "id"),
		StudentID:    strVal(dm, "studentId"),
		StudentName:  strVal(dm, "studentName"),
		DocType:      strVal(dm, "docType"),
		DocName:      strVal(dm, "docName"),
		FileName:     strVal(dm, "fileName"),
		BlobID:       strVal(dm, "blobId"),
		FileSizeKB:   intVal(dm, "fileSizeKb"),
		UploadedAt:   strVal(dm, "uploadedAt"),
		Status:       strVal(dm, "status"),
		RejectReason: strVal(dm, "rejectReason"),
	}
}

func parsePklPlacement(pm map[string]any) ui.PklPlacementView {
	return ui.PklPlacementView{
		ID:              strVal(pm, "id"),
		CompanyName:     strVal(pm, "companyName"),
		IndustrySector:  strVal(pm, "industrySector"),
		Address:         strVal(pm, "address"),
		GpsLat:          floatVal(pm, "gpsLat"),
		GpsLng:          floatVal(pm, "gpsLng"),
		RadiusMeters:    intVal(pm, "radiusMeters"),
		SupervisorName:  strVal(pm, "supervisorName"),
		SupervisorPhone: strVal(pm, "supervisorPhone"),
		SupervisorPin:   strVal(pm, "supervisorPin"),
	}
}

func parsePklLog(lm map[string]any) ui.PklLogView {
	return ui.PklLogView{
		ID:             strVal(lm, "id"),
		StudentID:      strVal(lm, "studentId"),
		StudentName:    strVal(lm, "studentName"),
		CompanyName:    strVal(lm, "companyName"),
		Timestamp:      strVal(lm, "timestamp"),
		DistanceMeters: intVal(lm, "distanceMeters"),
		GeofenceStatus: strVal(lm, "geofenceStatus"),
		ExifVerified:   boolVal(lm, "exifVerified"),
	}
}

func parsePklJournal(jm map[string]any) ui.PklJournalView {
	return ui.PklJournalView{
		ID:                 strVal(jm, "id"),
		StudentID:          strVal(jm, "studentId"),
		StudentName:        strVal(jm, "studentName"),
		PlacementID:        strVal(jm, "placementId"),
		CompanyName:        strVal(jm, "companyName"),
		Date:               strVal(jm, "date"),
		ActivityTitle:      strVal(jm, "activityTitle"),
		CompetencyLearned:  strVal(jm, "competencyLearned"),
		SupervisorApproved: boolVal(jm, "supervisorApproved"),
		TeacherApproved:    boolVal(jm, "teacherApproved"),
	}
}

func parseBkkJob(jm map[string]any) ui.BkkJobView {
	rawMajors, _ := jm["targetedMajors"].([]any)
	mStrs := make([]string, 0, len(rawMajors))
	for _, m := range rawMajors {
		if s, ok := m.(string); ok {
			mStrs = append(mStrs, s)
		}
	}
	return ui.BkkJobView{
		ID:             strVal(jm, "id"),
		CompanyName:    strVal(jm, "companyName"),
		JobTitle:       strVal(jm, "jobTitle"),
		TargetedMajors: strings.Join(mStrs, ", "),
		SalaryRange:    strVal(jm, "salaryRange"),
		DeadlineDate:   strVal(jm, "deadlineDate"),
		ApplicantCount: intVal(jm, "applicantCount"),
	}
}

func parseBkkApp(am map[string]any) ui.BkkAppView {
	return ui.BkkAppView{
		ID:          strVal(am, "id"),
		JobID:       strVal(am, "jobId"),
		JobTitle:    strVal(am, "jobTitle"),
		CompanyName: strVal(am, "companyName"),
		StudentID:   strVal(am, "studentId"),
		StudentName: strVal(am, "studentName"),
		AppliedDate: strVal(am, "appliedDate"),
		Status:      strVal(am, "status"),
	}
}

func parseLedgerEntry(lm map[string]any) ui.LedgerEntryView {
	return ui.LedgerEntryView{
		ID:               int64(intVal(lm, "id")),
		InvoiceID:        strVal(lm, "invoiceId"),
		ReceiptNumber:    strVal(lm, "receiptNumber"),
		TxType:           strVal(lm, "txType"),
		AmountDelta:      intVal(lm, "amountDelta"),
		StudentName:      strVal(lm, "studentName"),
		ActorName:        strVal(lm, "actorName"),
		PinVerified:      boolVal(lm, "treasurerPinVerified"),
		ReversalApproval: strVal(lm, "reversalApprovalId"),
		SignatureHmac:    strVal(lm, "signatureHmac"),
		CreatedAt:        strVal(lm, "createdAt"),
	}
}

func parseLetter(lm map[string]any) ui.LetterView {
	return ui.LetterView{
		ID:           strVal(lm, "id"),
		LetterNumber: strVal(lm, "letterNumber"),
		LetterType:   strVal(lm, "letterType"),
		StudentName:  strVal(lm, "studentName"),
		Recipient:    strVal(lm, "recipient"),
		IssueDate:    strVal(lm, "issueDate"),
		SignerName:   strVal(lm, "signerName"),
		SignerRole:   strVal(lm, "signerRole"),
		HmacHash:     strVal(lm, "hmacHash"),
	}
}

func parseConsultation(cm map[string]any) ui.ConsultationView {
	return ui.ConsultationView{
		ID:          strVal(cm, "id"),
		StudentID:   strVal(cm, "studentId"),
		StudentName: strVal(cm, "studentName"),
		ParentName:  strVal(cm, "parentName"),
		ChosenPath:  strVal(cm, "chosenPath"),
		TopicNote:   strVal(cm, "topicNote"),
		Status:      strVal(cm, "status"),
		CreatedAt:   strVal(cm, "createdAt"),
	}
}

func parseReversalReq(rm map[string]any) ui.ReversalRequestView {
	return ui.ReversalRequestView{
		ID:              strVal(rm, "id"),
		LedgerID:        int64(intVal(rm, "ledgerId")),
		TxID:            strVal(rm, "txId"),
		CashierName:     strVal(rm, "cashierName"),
		AmountOriginal:  intVal(rm, "amountOriginal"),
		AmountCorrected: intVal(rm, "amountCorrected"),
		Reason:          strVal(rm, "reason"),
		Status:          strVal(rm, "status"),
		OtpCode:         strVal(rm, "otpCode"),
	}
}

func (s *Server) buildDeveloperDashboard(doc map[string]any, shell ui.RoleShell, tab string) ui.DeveloperPortalPage {
	bugs := make([]ui.BugReportView, 0)
	total := 0
	open := 0
	investigating := 0
	resolved := 0

	for _, bm := range asMapSlice(doc["bugReports"]) {
		status := strings.ToUpper(strVal(bm, "status"))
		if status == "" {
			status = "OPEN"
		}
		sev := strings.ToLower(strVal(bm, "severity"))
		if sev == "" {
			sev = "medium"
		}
		total++
		switch status {
		case "OPEN":
			open++
		case "INVESTIGATING":
			investigating++
		case "FIXED", "RESOLVED", "CLOSED":
			resolved++
		}

		sevClass := "badge-neutral"
		switch sev {
		case "critical":
			sevClass = "badge-danger"
		case "high", "major":
			sevClass = "badge-warning"
		case "medium":
			sevClass = "badge-teal"
		case "low", "enhancement":
			sevClass = "badge-neutral"
		}

		stClass := "badge-neutral"
		switch status {
		case "OPEN":
			stClass = "badge-danger"
		case "INVESTIGATING":
			stClass = "badge-warning"
		case "FIXED", "RESOLVED":
			stClass = "badge-success"
		case "CLOSED":
			stClass = "badge-teal"
		}

		cat := strVal(bm, "category")
		catLabel := cat
		switch cat {
		case "CBT_EXAM":
			catLabel = "Ujian & CBT"
		case "ATTENDANCE_GATE":
			catLabel = "Presensi & Gerbang"
		case "FINANCE_SPP":
			catLabel = "Keuangan & SPP"
		case "RAPOR_HOLISTIK":
			catLabel = "e-Rapor & Nilai"
		case "UI_UX":
			catLabel = "Tampilan / UI / Responsif"
		case "AUTH_ACCESS":
			catLabel = "Akun & Hak Akses"
		default:
			catLabel = "Umum / Sistem"
		}

		bugs = append(bugs, ui.BugReportView{
			ID:             strVal(bm, "id"),
			ReporterUserID: strVal(bm, "reporterUserId"),
			ReporterName:   strVal(bm, "reporterName"),
			ReporterRole:   strVal(bm, "reporterRole"),
			Category:       cat,
			CategoryLabel:  catLabel,
			Title:          strVal(bm, "title"),
			Description:    strVal(bm, "description"),
			PageURL:        strVal(bm, "pageUrl"),
			Severity:       strings.ToUpper(sev),
			SeverityClass:  sevClass,
			Status:         status,
			StatusClass:    stClass,
			DeveloperNotes: strVal(bm, "developerNotes"),
			CreatedAt:      strVal(bm, "createdAt"),
		})
	}

	resets := make([]ui.ExamResetLogView, 0)
	for _, rm := range asMapSlice(doc["examResetLogs"]) {
		resets = append(resets, ui.ExamResetLogView{
			ID:          strVal(rm, "id"),
			StudentID:   strVal(rm, "studentId"),
			StudentName: strVal(rm, "studentName"),
			SubjectID:   strVal(rm, "subjectId"),
			ProctorName: strVal(rm, "proctorName"),
			Reason:      strVal(rm, "reason"),
			IsBulk:      boolVal(rm, "isBulk"),
			CreatedAt:   strVal(rm, "createdAt"),
		})
	}

	return ui.DeveloperPortalPage{
		Shell:             shell,
		Tab:               tab,
		TotalBugs:         total,
		OpenBugs:          open,
		InvestigatingBugs: investigating,
		ResolvedBugs:      resolved,
		BugReports:        bugs,
		ExamResetLogs:     resets,
	}
}

func (s *Server) buildContactPage(shell ui.RoleShell, reporterName, reporterEmail, reporterRole, refererURL, successMsg, errorMsg string) ui.ContactPage {
	return ui.ContactPage{
		Shell:         shell,
		ReporterName:  reporterName,
		ReporterEmail: reporterEmail,
		ReporterRole:  reporterRole,
		RefererURL:    refererURL,
		SuccessMsg:    successMsg,
		ErrorMsg:      errorMsg,
	}
}
