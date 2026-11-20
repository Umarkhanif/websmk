package state

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
)

// UpdateLeaderValidation records the class leader's validation of today's class
// session and triggers an emergency Piket alert if the class is empty.
func UpdateLeaderValidation(doc map[string]any, studentID, status string, now time.Time) error {
	valid := map[string]bool{"VALID_FULL": true, "TASK_ONLY": true, "EMPTY_CLASS": true}
	if !valid[status] {
		return fmt.Errorf("%w: status validasi tidak valid", ErrInvalid)
	}
	student := findStudent(doc, studentID)
	studentName := "Ketua Kelas"
	if student != nil {
		studentName = text(student["fullName"])
	}
	journal := asMap(doc["classJournal"])
	if journal == nil {
		journal = map[string]any{}
		doc["classJournal"] = journal
	}
	journal["leaderValidationStatus"] = status
	journal["validatedByStudentId"] = studentID
	journal["validatedAt"] = now.UTC().Format(time.RFC3339)

	if status == "EMPTY_CLASS" {
		for _, item := range asList(doc["piketClassrooms"]) {
			cls := asMap(item)
			if cls != nil && text(cls["id"]) == "cls-1" {
				cls["status"] = "RED_EMPTY"
				cls["note"] = fmt.Sprintf("Sinyal Darurat (Laporan %s): Kelas XII TKJ 1 kosong & butuh Guru Inval segera!", studentName)
			}
		}
	}
	return nil
}

// SubmitReflection stores a student's learning radar feedback.
func SubmitReflection(doc map[string]any, studentID, level, complaint string, now time.Time) error {
	student := findStudent(doc, studentID)
	if student == nil {
		return ErrNotFound
	}
	if level != "UNDERSTOOD" && level != "CONFUSED" {
		level = "UNDERSTOOD"
	}
	doc["studentReflection"] = map[string]any{
		"id":                   fmt.Sprintf("ref-%d", now.UnixNano()%100000),
		"journalId":            "jrn-tkj-01",
		"studentId":            studentID,
		"studentName":          text(student["fullName"]),
		"understandingLevel":   level,
		"privateComplaintText": strings.TrimSpace(complaint),
		"createdAt":            now.UTC().Format(time.RFC3339),
	}
	return nil
}

// SubmitPeerReport appends an anonymous peer bullying/facility report into escrow.
func SubmitPeerReport(doc map[string]any, category, location, targetName, description string, now time.Time) error {
	if strings.TrimSpace(targetName) == "" || strings.TrimSpace(description) == "" {
		return ErrInvalid
	}
	entry := map[string]any{
		"id":                   fmt.Sprintf("pr-%04d", now.UnixNano()%10000),
		"targetStudentName":    strings.TrimSpace(targetName),
		"targetStudentClass":   "Lintas Kelas SMK",
		"articleCategory":      strings.TrimSpace(category),
		"location":             strings.TrimSpace(location),
		"description":          strings.TrimSpace(description),
		"reporterClusterHash":  "cluster_hash_" + strings.ToLower(randomHex(4)),
		"distinctClusterCount": 12,
		"status":               "PENDING_ESCROW",
		"createdAt":            now.UTC().Format(time.RFC3339),
	}
	doc["peerReports"] = append([]any{entry}, asList(doc["peerReports"])...)
	return nil
}

// UploadStudentDocument marks a student document as uploaded and awaiting TU verification.
func UploadStudentDocument(doc map[string]any, studentID, docID, fileName string, fileSizeKB int, now time.Time) error {
	for _, item := range asList(doc["studentDocuments"]) {
		rec := asMap(item)
		if rec != nil && text(rec["id"]) == docID && text(rec["studentId"]) == studentID {
			if strings.TrimSpace(fileName) == "" {
				fileName = "berkas_resmi.pdf"
			}
			if fileSizeKB <= 0 {
				fileSizeKB = 240
			}
			rec["fileName"] = strings.TrimSpace(fileName)
			rec["fileSizeKb"] = fileSizeKB
			rec["mimeHeaderHex"] = "25504446"
			rec["uploadedAt"] = now.Format("2006-01-02")
			rec["status"] = "MENUNGGU_CEK"
			delete(rec, "rejectReason")
			return nil
		}
	}
	return ErrNotFound
}

// SubmitStudentTask records a workshop task submission.
func SubmitStudentTask(doc map[string]any, studentID, taskTitle, fileName string, fileSizeKB int, now time.Time) error {
	student := findStudent(doc, studentID)
	if student == nil || strings.TrimSpace(taskTitle) == "" {
		return ErrInvalid
	}
	journal := asMap(doc["classJournal"])
	subject := "Administrasi Infrastruktur Jaringan"
	if journal != nil && text(journal["subjectName"]) != "" {
		subject = text(journal["subjectName"])
	}
	if strings.TrimSpace(fileName) == "" {
		fileName = "laporan_praktik.pdf"
	}
	if fileSizeKB <= 0 {
		fileSizeKB = 310
	}
	entry := map[string]any{
		"id":          fmt.Sprintf("tsk-%04d", now.UnixNano()%10000),
		"studentId":   studentID,
		"studentName": text(student["fullName"]),
		"subjectName": subject,
		"taskTitle":   strings.TrimSpace(taskTitle),
		"fileName":    strings.TrimSpace(fileName),
		"fileSizeKb":  fileSizeKB,
		"submittedAt": now.In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02 15:04 WIB"),
		"status":      "TERKIRIM",
	}
	doc["taskSubmissions"] = append([]any{entry}, asList(doc["taskSubmissions"])...)
	return nil
}

// AddPklCheckin validates GPS distance with Haversine and records a PKL attendance log.
func AddPklCheckin(doc map[string]any, studentID string, lat, lng float64, now time.Time) error {
	student := findStudent(doc, studentID)
	if student == nil {
		return ErrNotFound
	}
	placements := asList(doc["pklPlacements"])
	var placement map[string]any
	if len(placements) > 0 {
		placement = asMap(placements[0])
	}
	targetLat, _ := number(placement["gpsLat"])
	targetLng, _ := number(placement["gpsLng"])
	radius, _ := number(placement["radiusMeters"])
	if radius <= 0 {
		radius = 100
	}
	if lat == 0 && lng == 0 {
		lat = targetLat + 0.00015
		lng = targetLng + 0.00012
	}
	dist := CalculateHaversineMeters(lat, lng, targetLat, targetLng)
	status := "VALID_IN_RADIUS"
	if float64(dist) > radius {
		status = "OUT_OF_GEOFENCE"
	}
	entry := map[string]any{
		"id":               fmt.Sprintf("pkl-att-%04d", now.UnixNano()%10000),
		"studentId":        studentID,
		"studentName":      text(student["fullName"]),
		"companyName":      text(placement["companyName"]),
		"timestamp":        now.In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02 15:04 WIB"),
		"studentLat":       lat,
		"studentLng":       lng,
		"distanceMeters":   dist,
		"geofenceStatus":   status,
		"exifVerified":     true,
		"selfiePreviewUrl": "https://images.unsplash.com/photo-1581092921461-eab62e97a780?w=300",
	}
	doc["pklAttendanceLogs"] = append([]any{entry}, asList(doc["pklAttendanceLogs"])...)
	return nil
}

// AddPklJournal appends a daily PKL activity journal awaiting DUDI and teacher approval.
func AddPklJournal(doc map[string]any, studentID, activityTitle, competency string, now time.Time) error {
	student := findStudent(doc, studentID)
	if student == nil || strings.TrimSpace(activityTitle) == "" {
		return ErrInvalid
	}
	placements := asList(doc["pklPlacements"])
	placementID := "pkl-01"
	companyName := "PT Telkom Akses Regional Jawa Barat"
	if len(placements) > 0 {
		if p := asMap(placements[0]); p != nil {
			placementID = text(p["id"])
			companyName = text(p["companyName"])
		}
	}
	entry := map[string]any{
		"id":                 fmt.Sprintf("pkl-jrn-%04d", now.UnixNano()%10000),
		"studentId":          studentID,
		"studentName":        text(student["fullName"]),
		"placementId":        placementID,
		"companyName":        companyName,
		"date":               now.Format("2006-01-02"),
		"activityTitle":      strings.TrimSpace(activityTitle),
		"competencyLearned":  strings.TrimSpace(competency),
		"workPhotoUrl":       "https://images.unsplash.com/photo-1544717305-2782549b5136?w=400",
		"supervisorApproved": false,
		"teacherApproved":    false,
	}
	doc["pklJournals"] = append([]any{entry}, asList(doc["pklJournals"])...)
	return nil
}

// ApplyBkkJob submits a 1-click job application to a BKK partner opening.
func ApplyBkkJob(doc map[string]any, studentID, jobID string, now time.Time) error {
	student := findStudent(doc, studentID)
	if student == nil {
		return ErrNotFound
	}
	for _, item := range asList(doc["bkkApplications"]) {
		app := asMap(item)
		if app != nil && text(app["studentId"]) == studentID && text(app["jobId"]) == jobID {
			return nil
		}
	}
	var targetJob map[string]any
	for _, item := range asList(doc["bkkJobs"]) {
		job := asMap(item)
		if job != nil && text(job["id"]) == jobID {
			targetJob = job
			break
		}
	}
	if targetJob == nil {
		return ErrNotFound
	}
	cnt, _ := number(targetJob["applicantCount"])
	targetJob["applicantCount"] = cnt + 1
	entry := map[string]any{
		"id":          fmt.Sprintf("app-%04d", now.UnixNano()%10000),
		"jobId":       jobID,
		"jobTitle":    text(targetJob["jobTitle"]),
		"companyName": text(targetJob["companyName"]),
		"studentId":   studentID,
		"studentName": text(student["fullName"]),
		"appliedDate": now.Format("2006-01-02"),
		"status":      "TERKIRIM",
	}
	doc["bkkApplications"] = append([]any{entry}, asList(doc["bkkApplications"])...)
	return nil
}

// UpdateBmwChoice updates a student's post-graduation career path (BEKERJA / MELANJUTKAN / WIRAUSAHA).
func UpdateBmwChoice(doc map[string]any, studentID, choice string) error {
	if !validBMWChoices[choice] {
		return ErrInvalid
	}
	student := findStudent(doc, studentID)
	if student == nil {
		return ErrNotFound
	}
	student["bmwChoice"] = choice
	return nil
}

// AddStudentExpense adds a study/workshop expense record.
func AddStudentExpense(doc map[string]any, studentID, category, title string, amount int, note string, now time.Time) error {
	if strings.TrimSpace(title) == "" || amount <= 0 {
		return ErrInvalid
	}
	entry := map[string]any{
		"id":          fmt.Sprintf("exp-%04d", now.UnixNano()%10000),
		"studentId":   studentID,
		"category":    category,
		"itemTitle":   strings.TrimSpace(title),
		"amount":      amount,
		"date":        now.Format("2006-01-02"),
		"receiptNote": strings.TrimSpace(note),
	}
	doc["studentExpenses"] = append([]any{entry}, asList(doc["studentExpenses"])...)
	return nil
}

// UpdateProctorStatus records anti-cheat tab switch warnings or locks in CBT.
// The browser may only report WARN and LOCKED signals; counters, tokens, and the
// transition back to ACTIVE are owned by the server (teacher unlock endpoints).
func UpdateProctorStatus(doc map[string]any, studentID, status, reason string, now time.Time) error {
	student := findStudent(doc, studentID)
	if student == nil {
		return ErrNotFound
	}
	if status != "WARN" && status != "LOCKED" {
		return fmt.Errorf("%w: klien hanya boleh melaporkan WARN atau LOCKED", ErrInvalid)
	}
	nisn := text(student["nisn"])
	for _, item := range asList(doc["proctorStudents"]) {
		ps := asMap(item)
		if ps != nil && text(ps["studentId"]) == studentID {
			current := text(ps["status"])
			// A lock (and a submitted session) is permanent until an authorized
			// teacher/QR unlock. Later browser reports must not change it.
			if current == "LOCKED" || current == "SUBMITTED" {
				return nil
			}
			warnCnt, _ := number(ps["warnCount"])
			lockCnt, _ := number(ps["lockCount"])
			if status == "WARN" {
				warnCnt++
				ps["warnCount"] = warnCnt
				if warnCnt >= 3 {
					status = "LOCKED"
				}
			}
			if status == "LOCKED" {
				lockCnt++
				ps["lockCount"] = lockCnt
				sfx := nisn
				if len(sfx) > 4 {
					sfx = sfx[len(sfx)-4:]
				}
				ps["unlockToken"] = fmt.Sprintf("UNK-%s-%04d", sfx, now.UnixNano()%10000)
			}
			ps["status"] = status
			if reason != "" {
				ps["lastViolation"] = reason
			}
			return nil
		}
	}
	return nil
}

// SubmitParentLeave updates a child's attendance status from an official parent form.
func SubmitParentLeave(doc map[string]any, studentID, leaveType, leaveDate, leaveNote string, now time.Time) error {
	status := "IZIN_ORTU"
	prefix := "Form Izin Resmi Akun Ortu"
	if strings.ToUpper(leaveType) == "SAKIT" {
		status = "SAKIT_ORTU"
		prefix = "Form Sakit Resmi Akun Ortu"
	}
	if leaveDate == "" {
		leaveDate = now.Format("2006-01-02")
	}
	source := fmt.Sprintf("%s (%s): %s", prefix, leaveDate, strings.TrimSpace(leaveNote))
	for _, item := range asList(doc["attendances"]) {
		att := asMap(item)
		if att != nil && text(att["studentId"]) == studentID {
			att["status"] = status
			att["verifiedSource"] = source
			att["updatedAt"] = now.UTC().Format(time.RFC3339)
			return nil
		}
	}
	return nil
}

// BookCareerConsultation creates a BMW career consultation ticket from a parent.
func BookCareerConsultation(doc map[string]any, studentID, chosenPath, topicNote string, now time.Time) error {
	student := findStudent(doc, studentID)
	if student == nil || strings.TrimSpace(topicNote) == "" {
		return ErrInvalid
	}
	if !validBMWChoices[chosenPath] {
		chosenPath = "BEKERJA"
	}
	entry := map[string]any{
		"id":          fmt.Sprintf("bmw-%04d", now.UnixNano()%10000),
		"studentId":   studentID,
		"studentName": text(student["fullName"]),
		"parentName":  text(student["parentName"]),
		"chosenPath":  chosenPath,
		"topicNote":   strings.TrimSpace(topicNote),
		"status":      "MENUNGGU_JADWAL",
		"createdAt":   now.Format("2006-01-02"),
	}
	doc["careerConsultations"] = append([]any{entry}, asList(doc["careerConsultations"])...)
	return nil
}

// UpdateScholarship updates a student's scholarship tier and recalculates unpaid invoices.
func UpdateScholarship(doc map[string]any, studentID, scholarshipType string, discountPct int) error {
	student := findStudent(doc, studentID)
	if student == nil {
		return ErrNotFound
	}
	if discountPct < 0 {
		discountPct = 0
	}
	if discountPct > 100 {
		discountPct = 100
	}
	student["scholarshipType"] = scholarshipType
	student["discountPercent"] = discountPct

	for _, item := range asList(doc["invoices"]) {
		inv := asMap(item)
		if inv != nil && text(inv["studentId"]) == studentID && text(inv["status"]) != "PAID" {
			base, _ := number(inv["baseAmount"])
			paid, _ := number(inv["paidAmount"])
			deduction := math.Round(base * float64(discountPct) / 100.0)
			finalAmt := math.Max(0, base-deduction)
			status := "UNPAID"
			if paid >= finalAmt {
				status = "PAID"
			} else if paid > 0 {
				status = "PARTIAL"
			}
			inv["scholarshipDeduction"] = deduction
			inv["finalAmount"] = finalAmt
			inv["status"] = status
		}
	}
	return nil
}

// SendWaBillingReminder records the polite WhatsApp reminder timestamp on an invoice.
func SendWaBillingReminder(doc map[string]any, invoiceID string, now time.Time) error {
	for _, item := range asList(doc["invoices"]) {
		inv := asMap(item)
		if inv != nil && recordID(inv["id"]) == invoiceID {
			inv["lastReminderSentAt"] = now.In(time.FixedZone("WIB", 7*3600)).Format("15:04 WIB")
			return nil
		}
	}
	return ErrNotFound
}

// GrantDispensation issues a 7-day exam clearance dispensation without altering cash balances.
func GrantDispensation(doc map[string]any, studentID, reason, grantedBy string, now time.Time) error {
	student := findStudent(doc, studentID)
	if student == nil {
		return ErrNotFound
	}
	if strings.TrimSpace(reason) == "" {
		reason = "Komitmen Cicilan Orang Tua Wali (Dispensasi 7 Hari)"
	}
	entry := map[string]any{
		"id":              fmt.Sprintf("disp-%04d", now.UnixNano()%10000),
		"studentId":       studentID,
		"studentName":     text(student["fullName"]),
		"examPeriod":      "Penilaian Akhir Semester Ganjil 2026",
		"clearanceType":   "DISPENSATION_7D",
		"reasonCategory":  strings.TrimSpace(reason),
		"agreementDocUrl": "https://r2.wesmk.sch.id/dispensations/surat_perjanjian.pdf",
		"validUntil":      now.Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339),
		"grantedBy":       grantedBy,
	}
	doc["dispensations"] = append([]any{entry}, asList(doc["dispensations"])...)
	return nil
}

// RecordCashPayment verifies the 6-digit treasurer PIN on the server and appends
// an HMAC-signed PAYMENT_CASH entry to the immutable ledger.
func RecordCashPayment(doc map[string]any, invoiceID string, amount int64, pinInput, expectedPIN, actorID, actorName, receiptSecret string, now time.Time) error {
	if strings.TrimSpace(expectedPIN) == "" {
		return fmt.Errorf("%w: PIN Kasir 6-digit belum dikonfigurasi di server", ErrInvalid)
	}
	if !hmac.Equal([]byte(expectedPIN), []byte(strings.TrimSpace(pinInput))) {
		return fmt.Errorf("%w: PIN Kasir 6-digit tidak sesuai", ErrInvalid)
	}
	if amount <= 0 {
		return fmt.Errorf("%w: nominal pembayaran harus lebih dari 0", ErrInvalid)
	}
	var targetInv map[string]any
	for _, item := range asList(doc["invoices"]) {
		inv := asMap(item)
		if inv != nil && recordID(inv["id"]) == invoiceID {
			targetInv = inv
			break
		}
	}
	if targetInv == nil {
		return ErrNotFound
	}
	finalAmt, _ := number(targetInv["finalAmount"])
	paidAmt, _ := number(targetInv["paidAmount"])
	nextPaid := math.Min(finalAmt, paidAmt+float64(amount))
	nextStatus := "PARTIAL"
	if nextPaid >= finalAmt {
		nextStatus = "PAID"
	}
	targetInv["paidAmount"] = nextPaid
	targetInv["status"] = nextStatus

	receiptNumber := fmt.Sprintf("KWT-CASH-%s-%s", now.UTC().Format("20060102"), randomHex(2))
	createdAt := now.UTC().Format(time.RFC3339)
	sig := ReceiptSignature(receiptSecret, invoiceID, receiptNumber, amount, createdAt)

	entry := map[string]any{
		"id":                   nextLedgerID(doc),
		"invoiceId":            invoiceID,
		"receiptNumber":        receiptNumber,
		"txType":               "PAYMENT_CASH",
		"amountDelta":          float64(amount),
		"studentName":          text(targetInv["studentName"]),
		"actorUserId":          actorID,
		"actorName":            actorName,
		"treasurerPinVerified": true,
		"signatureHmac":        sig,
		"createdAt":            createdAt,
	}
	doc["ledger"] = append([]any{entry}, asList(doc["ledger"])...)
	return nil
}

// ExecuteReversal verifies the Headmaster's 6-digit OTP and appends a REVERSAL_VOID
// entry to the append-only ledger without deleting historical rows.
func ExecuteReversal(doc map[string]any, ledgerID int64, reason, otpInput, actorID, actorName, receiptSecret string, now time.Time) error {
	expectedOTP := strings.TrimSpace(text(doc["activeHeadmasterOtp"]))
	if expectedOTP == "" {
		return fmt.Errorf("%w: kode OTP Kepala Sekolah belum diterbitkan", ErrInvalid)
	}
	if expiry := strings.TrimSpace(text(doc["activeHeadmasterOtpExpiresAt"])); expiry != "" {
		expiresAt, err := time.Parse(time.RFC3339, expiry)
		if err != nil || now.After(expiresAt) {
			return fmt.Errorf("%w: kode OTP Kepala Sekolah sudah kedaluwarsa", ErrInvalid)
		}
	}
	if !hmac.Equal([]byte(expectedOTP), []byte(strings.TrimSpace(otpInput))) {
		return fmt.Errorf("%w: kode OTP Kepala Sekolah tidak valid", ErrInvalid)
	}
	var original map[string]any
	for _, item := range asList(doc["ledger"]) {
		rec := asMap(item)
		if rec != nil {
			idVal, _ := number(rec["id"])
			if int64(idVal) == ledgerID {
				original = rec
				break
			}
		}
	}
	if original == nil {
		return ErrNotFound
	}
	origDelta, _ := number(original["amountDelta"])
	revDelta := -int64(math.Round(origDelta))
	invoiceID := text(original["invoiceId"])
	receiptNumber := fmt.Sprintf("REV-%s-%s", now.UTC().Format("20060102"), randomHex(2))
	createdAt := now.UTC().Format(time.RFC3339)
	sig := ReceiptSignature(receiptSecret, invoiceID, receiptNumber, revDelta, createdAt)

	entry := map[string]any{
		"id":                   nextLedgerID(doc),
		"invoiceId":            invoiceID,
		"receiptNumber":        receiptNumber,
		"txType":               "REVERSAL_VOID",
		"amountDelta":          float64(revDelta),
		"studentName":          text(original["studentName"]),
		"actorUserId":          actorID,
		"actorName":            actorName,
		"treasurerPinVerified": true,
		"reversalApprovalId":   "appr-otp-" + expectedOTP,
		"signatureHmac":        sig,
		"createdAt":            createdAt,
	}
	doc["ledger"] = append([]any{entry}, asList(doc["ledger"])...)

	for _, item := range asList(doc["reversalRequests"]) {
		req := asMap(item)
		if req != nil {
			reqLedgerID, _ := number(req["ledgerId"])
			if int64(reqLedgerID) == ledgerID {
				req["status"] = "SELESAI_DIBUKUKAN"
			}
		}
	}
	return nil
}

// AlertPiketAbsent sends a 1-click teacher absence signal to the Piket radar.
func AlertPiketAbsent(doc map[string]any, teacherName string) {
	for _, item := range asList(doc["piketClassrooms"]) {
		cls := asMap(item)
		if cls != nil && text(cls["id"]) == "cls-1" {
			cls["status"] = "RED_EMPTY"
			cls["note"] = fmt.Sprintf("Sinyal Darurat (1-Klik Izin Guru %s): Kelas XII TKJ 1 membutuhkan Guru Inval segera!", teacherName)
		}
	}
}

// UpdateClassJournal updates the active syllabus topic, teacher notes, and daily quiz topic.
func UpdateClassJournal(doc map[string]any, topic, notes, aiSmartNotes string) error {
	if strings.TrimSpace(topic) == "" {
		return ErrInvalid
	}
	journal := asMap(doc["classJournal"])
	if journal == nil {
		journal = map[string]any{}
		doc["classJournal"] = journal
	}
	journal["syllabusTopic"] = strings.TrimSpace(topic)
	if strings.TrimSpace(notes) != "" {
		journal["teacherNotes"] = strings.TrimSpace(notes)
	}
	if strings.TrimSpace(aiSmartNotes) != "" {
		journal["aiSmartNotes"] = strings.TrimSpace(aiSmartNotes)
	}
	if quiz := asMap(doc["dailyQuiz"]); quiz != nil {
		quiz["topic"] = strings.TrimSpace(topic)
	}
	return nil
}

// ToggleStudentAlpa toggles a student's class attendance between ALPA_BOLOS and HADIR_GERBANG.
func ToggleStudentAlpa(doc map[string]any, attendanceID string, now time.Time) error {
	for _, item := range asList(doc["attendances"]) {
		att := asMap(item)
		if att != nil && text(att["id"]) == attendanceID {
			curr := text(att["status"])
			next := "ALPA_BOLOS"
			source := "Ditandai Alpa / Bolos oleh Guru di Kelas"
			if curr == "ALPA_BOLOS" {
				next = "HADIR_GERBANG"
				source = "Tap Kartu RFID Gerbang"
			}
			att["status"] = next
			att["verifiedSource"] = source
			if next == "ALPA_BOLOS" {
				v := map[string]any{
					"id":             fmt.Sprintf("v-%04d", now.UnixNano()%10000),
					"studentId":      text(att["studentId"]),
					"studentName":    text(att["studentName"]) + " (XII TKJ 1)",
					"articleCode":    "PASAL-14",
					"articleTitle":   "Membolos Jam Pelajaran Kejuruan (Ditandai Guru Mapel)",
					"sourceType":     "CLASS_ALPA",
					"pointsDeducted": 15,
					"status":         "TERCATAT",
					"createdAt":      now.UTC().Format(time.RFC3339),
				}
				doc["violations"] = append([]any{v}, asList(doc["violations"])...)
			}
			return nil
		}
	}
	return ErrNotFound
}

// AddExamQuestion appends a new question to the vocational question bank.
func AddExamQuestion(doc map[string]any, qType, questionText, optA, optB, optC, optD, correctKey, rubric string, now time.Time) error {
	if strings.TrimSpace(questionText) == "" {
		return ErrInvalid
	}
	if qType != "ESSAY" {
		qType = "MULTIPLE_CHOICE"
	}
	questions := asList(doc["examQuestions"])
	entry := map[string]any{
		"id":             fmt.Sprintf("q-%04d", now.UnixNano()%10000),
		"examId":         "exam-cbt-tkj-01",
		"questionNumber": len(questions) + 1,
		"questionType":   qType,
		"subjectName":    "Administrasi Infrastruktur Jaringan",
		"kdCode":         "KD-3.4",
		"questionText":   strings.TrimSpace(questionText),
		"options": []any{
			map[string]any{"key": "A", "text": optA},
			map[string]any{"key": "B", "text": optB},
			map[string]any{"key": "C", "text": optC},
			map[string]any{"key": "D", "text": optD},
		},
		"correctKey":  strings.ToUpper(strings.TrimSpace(correctKey)),
		"essayRubric": strings.TrimSpace(rubric),
	}
	doc["examQuestions"] = append(questions, entry)
	return nil
}

// DeleteExamQuestion removes a question from the bank and renumbers the remaining items.
func DeleteExamQuestion(doc map[string]any, questionID string) {
	var out []any
	for _, item := range asList(doc["examQuestions"]) {
		q := asMap(item)
		if q == nil || text(q["id"]) == questionID {
			continue
		}
		q["questionNumber"] = len(out) + 1
		out = append(out, q)
	}
	doc["examQuestions"] = out
}

// UnlockProctorStudent unlocks a locked CBT student session.
func UnlockProctorStudent(doc map[string]any, studentID string) {
	UnlockProctorStudentWithToken(doc, studentID, "")
}

// UnlockProctorStudentWithToken verifies the student's QR unlock token before
// restoring the session.
func UnlockProctorStudentWithToken(doc map[string]any, studentID, token string) bool {
	for _, item := range asList(doc["proctorStudents"]) {
		ps := asMap(item)
		if ps == nil || text(ps["studentId"]) != studentID {
			continue
		}
		if token != "" && text(ps["unlockToken"]) != strings.TrimSpace(token) {
			return false
		}
		ps["status"] = "ACTIVE"
		ps["lastViolation"] = "Telah di-unlock oleh Pengawas"
		return true
	}
	return false
}

// MassUnlockProctor unlocks all student sessions in the lab at once.
func MassUnlockProctor(doc map[string]any) {
	for _, item := range asList(doc["proctorStudents"]) {
		if ps := asMap(item); ps != nil {
			ps["status"] = "ACTIVE"
			ps["lastViolation"] = "Mass Unlock Ruangan: Kendala WiFi Lab Teratasi"
		}
	}
}

// GrantEmergency2hTicket issues a 2-hour proctor emergency exam clearance ticket.
func GrantEmergency2hTicket(doc map[string]any, studentID, grantedBy string, now time.Time) {
	studentName := studentID
	if s := findStudent(doc, studentID); s != nil {
		studentName = text(s["fullName"])
	}
	for _, item := range asList(doc["proctorStudents"]) {
		if ps := asMap(item); ps != nil && text(ps["studentId"]) == studentID {
			ps["status"] = "ACTIVE"
			ps["hasEmergency2hTicket"] = true
			if text(ps["name"]) != "" {
				studentName = text(ps["name"])
			}
		}
	}
	disp := map[string]any{
		"id":              fmt.Sprintf("disp-2h-%04d", now.UnixNano()%10000),
		"studentId":       studentID,
		"studentName":     studentName,
		"examPeriod":      "Sesi Darurat 2 Jam Ujian CBT",
		"clearanceType":   "EMERGENCY_SESSION_2H",
		"reasonCategory":  "Tiket Darurat 2 Jam Pengawas Ruangan",
		"agreementDocUrl": "https://r2.wesmk.sch.id/dispensations/tiket_darurat_2h.pdf",
		"validUntil":      now.Add(2 * time.Hour).UTC().Format(time.RFC3339),
		"grantedBy":       grantedBy,
	}
	doc["dispensations"] = append([]any{disp}, asList(doc["dispensations"])...)
}

// GradeEssayToRapor updates a student's CBT score after teacher essay grading.
func GradeEssayToRapor(doc map[string]any, studentID string, score int) error {
	student := findStudent(doc, studentID)
	if student == nil {
		return ErrNotFound
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	student["cbtScore"] = float64(score)
	return nil
}

// UpdateHomeroomAttitude filters 5 teacher attitude ratings with MAD and updates the student profile.
func UpdateHomeroomAttitude(doc map[string]any, studentID string, scores []float64, homeroomNote string) (float64, error) {
	student := findStudent(doc, studentID)
	if student == nil {
		return 0, ErrNotFound
	}
	hasViol := make([]bool, len(scores))
	filtered, _, _, _, _ := FilterOutlierAttitudeRatings(scores, hasViol)
	student["attitudeScore"] = math.Round(filtered)
	if strings.TrimSpace(homeroomNote) != "" {
		student["homeroomNote"] = strings.TrimSpace(homeroomNote)
	}
	return filtered, nil
}

// BulkApprovePklJournals approves all pending student PKL journals as homeroom/advisor teacher.
func BulkApprovePklJournals(doc map[string]any) {
	for _, item := range asList(doc["pklJournals"]) {
		if j := asMap(item); j != nil {
			j["teacherApproved"] = true
		}
	}
}

// AddManualViolation records a discipline violation.
func AddManualViolation(doc map[string]any, studentName, articleCode, articleTitle string, points int, now time.Time) error {
	if strings.TrimSpace(studentName) == "" || strings.TrimSpace(articleTitle) == "" {
		return ErrInvalid
	}
	if points <= 0 {
		points = 10
	}
	entry := map[string]any{
		"id":             fmt.Sprintf("v-%04d", now.UnixNano()%10000),
		"studentId":      fmt.Sprintf("std-%03d", now.UnixNano()%1000),
		"studentName":    strings.TrimSpace(studentName),
		"articleCode":    strings.TrimSpace(articleCode),
		"articleTitle":   strings.TrimSpace(articleTitle),
		"sourceType":     "MANUAL",
		"pointsDeducted": points,
		"status":         "TERCATAT",
		"createdAt":      now.UTC().Format(time.RFC3339),
	}
	doc["violations"] = append([]any{entry}, asList(doc["violations"])...)
	return nil
}

// IssueSummonLetter issues an HMAC-signed parent summon letter for a violation.
func IssueSummonLetter(doc map[string]any, violationID, secret string, now time.Time) (string, error) {
	for _, item := range asList(doc["violations"]) {
		v := asMap(item)
		if v != nil && text(v["id"]) == violationID {
			studentName := text(v["studentName"])
			letter := CreateOfficialLetter(doc, "SURAT_PANGGILAN_ORTU", studentName, "Orang Tua / Wali dari "+studentName, secret, now)
			hash := text(letter["hmacHash"])
			v["status"] = "SURAT_PANGGILAN_1"
			v["summonLetterHash"] = hash
			return hash, nil
		}
	}
	return "", ErrNotFound
}

// ResolvePeerReport updates the status of an escrow peer bullying report.
func ResolvePeerReport(doc map[string]any, reportID, status string) {
	for _, item := range asList(doc["peerReports"]) {
		if pr := asMap(item); pr != nil && text(pr["id"]) == reportID {
			pr["status"] = status
		}
	}
}

// ScheduleConsultation marks a parent career consultation ticket as scheduled by BK.
func ScheduleConsultation(doc map[string]any, consultID string) {
	for _, item := range asList(doc["careerConsultations"]) {
		if c := asMap(item); c != nil && text(c["id"]) == consultID {
			c["status"] = "TERJADWAL_BK"
		}
	}
}

// UpdateCounselingSession updates a student's counseling recovery status.
func UpdateCounselingSession(doc map[string]any, sessionID, status, notes string) {
	for _, item := range asList(doc["counselingSessions"]) {
		if cs := asMap(item); cs != nil && text(cs["id"]) == sessionID {
			if status != "" {
				cs["recoveryStatus"] = status
			}
			if strings.TrimSpace(notes) != "" {
				cs["notes"] = strings.TrimSpace(notes)
			}
			checkins, _ := number(cs["dailyCheckinsCompleted"])
			cs["dailyCheckinsCompleted"] = checkins + 1
		}
	}
}

// AssignInvalTeacher assigns a substitute teacher to an empty classroom and awards +5 Teacher Impact points.
func AssignInvalTeacher(doc map[string]any, classroomID, teacherName, taskNote string) {
	for _, item := range asList(doc["piketClassrooms"]) {
		if cls := asMap(item); cls != nil && text(cls["id"]) == classroomID {
			cls["status"] = "GREEN"
			cls["note"] = fmt.Sprintf("Guru Inval Ditugaskan: %s (+5 Poin Teacher Impact). Tugas: %s.", teacherName, taskNote)
		}
	}
	for _, item := range asList(doc["teacherImpacts"]) {
		if t := asMap(item); t != nil && text(t["teacherName"]) == teacherName {
			bonus, _ := number(t["substituteBonusPoints"])
			growth, _ := number(t["calibratedScoreGrowth"])
			lowConf, _ := number(t["lowConfusionRate"])
			leaderVal, _ := number(t["leaderValidationRate"])
			nextBonus := bonus + 5
			t["substituteBonusPoints"] = nextBonus
			t["finalImpactScore"] = CalculateTeacherImpactScore(growth, lowConf, leaderVal, 88, nextBonus)
		}
	}
}

// AddGateLog records a student gate entry/exit and auto-triggers PASAL-02 if late.
func AddGateLog(doc map[string]any, nisn, studentName, logType, reason string, now time.Time) {
	if strings.TrimSpace(nisn) == "" {
		nisn = "0058471292"
	}
	if strings.TrimSpace(studentName) == "" {
		studentName = "Deni Kurniawan (XII TKJ 1)"
	}
	entry := map[string]any{
		"id":          fmt.Sprintf("gt-%04d", now.UnixNano()%10000),
		"nisn":        strings.TrimSpace(nisn),
		"studentName": strings.TrimSpace(studentName),
		"logType":     logType,
		"reason":      strings.TrimSpace(reason),
		"recordedAt":  now.In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02 15:04 WIB"),
		"qrPassHash":  "PASS-GATE-" + randomHex(3),
	}
	doc["piketGateLogs"] = append([]any{entry}, asList(doc["piketGateLogs"])...)

	if logType == "TERLAMBAT_MASUK" {
		v := map[string]any{
			"id":             fmt.Sprintf("v-%04d", now.UnixNano()%10000),
			"studentId":      "std-" + nisn,
			"studentName":    strings.TrimSpace(studentName),
			"articleCode":    "PASAL-02",
			"articleTitle":   fmt.Sprintf("Keterlambatan Masuk Gerbang >15m (%s)", strings.TrimSpace(reason)),
			"sourceType":     "GATE_TAP",
			"pointsDeducted": 5,
			"status":         "TERCATAT",
			"createdAt":      now.UTC().Format(time.RFC3339),
		}
		doc["violations"] = append([]any{v}, asList(doc["violations"])...)
	}
}

// VerifyStudentDoc approves or rejects a student document in the Admin TU queue.
func VerifyStudentDoc(doc map[string]any, docID string, approved bool, rejectReason string) {
	for _, item := range asList(doc["studentDocuments"]) {
		if d := asMap(item); d != nil && text(d["id"]) == docID {
			if approved {
				d["status"] = "VALID_DISETUJUI"
				delete(d, "rejectReason")
			} else {
				d["status"] = "DITOLAK"
				d["rejectReason"] = strings.TrimSpace(rejectReason)
			}
		}
	}
}

// AddNewStudent adds a student profile and initial tuition invoice in Admin TU.
func AddNewStudent(doc map[string]any, fullName, nisn, classroomName, majorCode string, now time.Time) error {
	if strings.TrimSpace(fullName) == "" || strings.TrimSpace(nisn) == "" {
		return ErrInvalid
	}
	newID := fmt.Sprintf("std-%04d", now.UnixNano()%10000)
	nikSuffix := nisn
	if len(nikSuffix) > 10 {
		nikSuffix = nikSuffix[:10]
	}
	std := map[string]any{
		"id":                     newID,
		"userId":                 "usr-" + newID,
		"nisn":                   strings.TrimSpace(nisn),
		"nik":                    "320128" + nikSuffix,
		"fullName":               strings.TrimSpace(fullName),
		"classroomId":            "cls-" + strings.ToLower(strings.ReplaceAll(classroomName, " ", "-")),
		"classroomName":          classroomName,
		"majorCode":              majorCode,
		"parentId":               "prt-" + newID,
		"parentName":             "Wali dari " + strings.TrimSpace(fullName),
		"parentPhone":            "081234567899",
		"isKetuaKelas":           false,
		"isSekretaris":           false,
		"isAlumni":               false,
		"scholarshipType":        "NONE",
		"discountPercent":        0,
		"attitudeScore":          90,
		"cbtScore":               85,
		"dailyQuizAvg":           88,
		"attendanceConductScore": 95,
		"activeBadge":            "Siswa Vokasi Aktif",
		"activeBadgeReason":      "Disiplin mengikuti kegiatan praktikum bengkel.",
		"homeroomNote":           "Terus pertahankan semangat belajar kejuruan.",
		"bmwChoice":              "BEKERJA",
	}
	doc["students"] = append(asList(doc["students"]), std)

	inv := map[string]any{
		"id":                   fmt.Sprintf("inv-%04d", now.UnixNano()%10000),
		"invoiceNumber":        fmt.Sprintf("INV/2026/11/%04d", 1000+(now.UnixNano()%9000)),
		"studentId":            newID,
		"studentName":          strings.TrimSpace(fullName),
		"periodMonthYear":      "November 2026",
		"baseAmount":           350000,
		"scholarshipDeduction": 0,
		"finalAmount":          350000,
		"paidAmount":           0,
		"status":               "UNPAID",
		"vaNumber":             "88019" + strings.TrimSpace(nisn),
		"gatewayProvider":      "Bank Mandiri (Midtrans)",
		"createdAt":            now.UTC().Format(time.RFC3339),
	}
	doc["invoices"] = append(asList(doc["invoices"]), inv)
	return nil
}

// CreateOfficialLetter issues an HMAC-SHA256 signed official school letter.
func CreateOfficialLetter(doc map[string]any, letterType, studentName, recipient, secret string, now time.Time) map[string]any {
	letterNumber := fmt.Sprintf("421.5/%d/SMK-TI/X/2026", 100+(now.UnixNano()%899))
	issueDate := now.Format("2006-01-02")
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%s|%s|%s|%s", letterNumber, letterType, studentName, issueDate)
	hash := hex.EncodeToString(mac.Sum(nil))[:32]

	letter := map[string]any{
		"id":           fmt.Sprintf("ltr-%04d", now.UnixNano()%10000),
		"letterNumber": letterNumber,
		"letterType":   letterType,
		"studentName":  strings.TrimSpace(studentName),
		"recipient":    strings.TrimSpace(recipient),
		"issueDate":    issueDate,
		"signerName":   "Dr. H. Sulaiman Effendi, M.Pd.",
		"signerRole":   "Kepala Sekolah",
		"hmacHash":     hash,
		"isVerified":   true,
	}
	doc["officialLetters"] = append([]any{letter}, asList(doc["officialLetters"])...)
	return letter
}

// TriggerDapodikSync appends a full Dapodik & e-Rapor synchronization log.
func TriggerDapodikSync(doc map[string]any, now time.Time) {
	studentsCount := len(asList(doc["students"]))
	entry := map[string]any{
		"id":            fmt.Sprintf("sync-%04d", now.UnixNano()%10000),
		"syncTimestamp": now.In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02 15:04 WIB"),
		"entitySynced":  fmt.Sprintf("Sinkronisasi Penuh (%d Profil Siswa & Nilai e-Rapor 360°)", studentsCount),
		"recordsCount":  1340 + studentsCount,
		"status":        "SUKSES_100%",
	}
	doc["dapodikLogs"] = append([]any{entry}, asList(doc["dapodikLogs"])...)
}

// AddPklPlacement adds an industrial partner (DUDI) with geofence and supervisor PIN.
func AddPklPlacement(doc map[string]any, companyName, sector, address string, lat, lng float64, radius int, pin string, now time.Time) error {
	if strings.TrimSpace(companyName) == "" {
		return ErrInvalid
	}
	if radius <= 0 {
		radius = 100
	}
	if strings.TrimSpace(pin) == "" {
		pin = "112233"
	}
	entry := map[string]any{
		"id":              fmt.Sprintf("pkl-%04d", now.UnixNano()%10000),
		"companyName":     strings.TrimSpace(companyName),
		"industrySector":  strings.TrimSpace(sector),
		"address":         strings.TrimSpace(address),
		"gpsLat":          lat,
		"gpsLng":          lng,
		"radiusMeters":    radius,
		"supervisorName":  "Supervisor Mitra DUDI",
		"supervisorPhone": "081199887766",
		"supervisorPin":   strings.TrimSpace(pin),
	}
	doc["pklPlacements"] = append(asList(doc["pklPlacements"]), entry)
	return nil
}

// AddBkkJob adds a new BKK job vacancy.
func AddBkkJob(doc map[string]any, companyName, jobTitle, salaryRange string, majors []string, now time.Time) error {
	if strings.TrimSpace(companyName) == "" || strings.TrimSpace(jobTitle) == "" {
		return ErrInvalid
	}
	if len(majors) == 0 {
		majors = []string{"TKJ", "RPL"}
	}
	mAny := make([]any, len(majors))
	for i, m := range majors {
		mAny[i] = m
	}
	entry := map[string]any{
		"id":             fmt.Sprintf("job-%04d", now.UnixNano()%10000),
		"companyName":    strings.TrimSpace(companyName),
		"jobTitle":       strings.TrimSpace(jobTitle),
		"targetedMajors": mAny,
		"salaryRange":    strings.TrimSpace(salaryRange),
		"deadlineDate":   "2026-12-31",
		"requirements":   []any{"Lulusan SMK", "Portofolio Rapor Holistik 360°"},
		"applicantCount": 1,
	}
	doc["bkkJobs"] = append(asList(doc["bkkJobs"]), entry)
	return nil
}

// GenerateReversalOTP generates a fresh 6-digit Headmaster OTP for ledger reversal.
func GenerateReversalOTP(doc map[string]any, reqID string, now time.Time) string {
	otp := fmt.Sprintf("%06d", 100000+(now.UnixNano()%900000))
	doc["activeHeadmasterOtp"] = otp
	doc["activeHeadmasterOtpExpiresAt"] = now.Add(5 * time.Minute).UTC().Format(time.RFC3339)
	if reqID != "" {
		for _, item := range asList(doc["reversalRequests"]) {
			if r := asMap(item); r != nil && text(r["id"]) == reqID {
				r["status"] = "OTORISASI_OTP_TERBIT"
				r["otpCode"] = otp
			}
		}
	}
	return otp
}

// ApprovePklJournalByDudi verifies the DUDI supervisor's 6-digit PIN and approves the journal.
func ApprovePklJournalByDudi(doc map[string]any, journalID, pinInput string, now time.Time) error {
	var targetJournal map[string]any
	for _, item := range asList(doc["pklJournals"]) {
		if j := asMap(item); j != nil && text(j["id"]) == journalID {
			targetJournal = j
			break
		}
	}
	if targetJournal == nil {
		return ErrNotFound
	}
	expectedPin := "445566"
	placementID := text(targetJournal["placementId"])
	for _, item := range asList(doc["pklPlacements"]) {
		if p := asMap(item); p != nil && (text(p["id"]) == placementID || placementID == "") {
			if text(p["supervisorPin"]) != "" {
				expectedPin = text(p["supervisorPin"])
			}
			break
		}
	}
	if strings.TrimSpace(pinInput) != expectedPin {
		return fmt.Errorf("%w: PIN Supervisor DUDI salah (Gunakan PIN resmi mitra, demo: %s)", ErrInvalid, expectedPin)
	}
	targetJournal["supervisorApproved"] = true
	targetJournal["approvedAt"] = now.UTC().Format(time.RFC3339)
	return nil
}

// ResetStudentExam clears a student's submission and restores their active CBT session, logging the action for audit.
func ResetStudentExam(doc map[string]any, studentID, subjectID, proctorUserID, proctorName, reason string, now time.Time) error {
	student := findStudent(doc, studentID)
	studentName := studentID
	if student != nil {
		studentName = text(student["fullName"])
		student["cbtScore"] = 0
	}

	// 1. Remove exam submission so student can restart the exam
	var keptSubmissions []any
	for _, item := range asList(doc["examSubmissions"]) {
		rec := asMap(item)
		if rec == nil {
			continue
		}
		if text(rec["studentId"]) == studentID {
			if subjectID == "" || text(rec["examId"]) == subjectID || subjectID == "exam-cbt-tkj-01" || subjectID == "exam-sch-01" {
				continue
			}
		}
		keptSubmissions = append(keptSubmissions, rec)
	}
	doc["examSubmissions"] = keptSubmissions

	// 2. Reset proctor student status
	for _, item := range asList(doc["proctorStudents"]) {
		ps := asMap(item)
		if ps == nil || text(ps["studentId"]) != studentID {
			continue
		}
		if text(ps["name"]) != "" {
			studentName = text(ps["name"])
		}
		ps["status"] = "ACTIVE"
		ps["progress"] = "0/40"
		ps["lockCount"] = 0
		ps["warnCount"] = 0
		ps["unlockToken"] = ""
		ps["hasEmergency2hTicket"] = true
		cleanReason := strings.TrimSpace(reason)
		if cleanReason == "" {
			cleanReason = "Restart Ujian oleh Pengawas"
		}
		ps["lastViolation"] = fmt.Sprintf("Reset Ujian oleh Pengawas (%s): %s", proctorName, cleanReason)
	}

	if strings.TrimSpace(reason) == "" {
		reason = "Kendala teknis / izin pengawas ruangan"
	}
	if strings.TrimSpace(subjectID) == "" {
		subjectID = "exam-cbt-tkj-01"
	}

	// 3. Append to audit log
	logEntry := map[string]any{
		"id":            fmt.Sprintf("rst-%04d", now.UnixNano()%10000),
		"studentId":     studentID,
		"studentName":   studentName,
		"subjectId":     subjectID,
		"proctorUserId": proctorUserID,
		"proctorName":   proctorName,
		"reason":        reason,
		"isBulk":        false,
		"createdAt":     now.UTC().Format(time.RFC3339),
	}
	doc["examResetLogs"] = append([]any{logEntry}, asList(doc["examResetLogs"])...)
	return nil
}

// BulkResetStudentExams resets exams for multiple students or all students in the room.
func BulkResetStudentExams(doc map[string]any, studentIDs []string, subjectID, proctorUserID, proctorName, reason string, now time.Time) (int, error) {
	if strings.TrimSpace(reason) == "" {
		reason = "Reset Massal Pengawas Ruangan"
	}

	targetIDs := make(map[string]bool)
	all := false
	for _, id := range studentIDs {
		if id == "ALL" || id == "*" {
			all = true
			break
		}
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			targetIDs[trimmed] = true
		}
	}

	if all || len(targetIDs) == 0 {
		for _, item := range asList(doc["proctorStudents"]) {
			if ps := asMap(item); ps != nil {
				targetIDs[text(ps["studentId"])] = true
			}
		}
	}

	count := 0
	for sID := range targetIDs {
		_ = ResetStudentExam(doc, sID, subjectID, proctorUserID, proctorName, reason, now)
		count++
	}

	if count > 1 {
		bulkLog := map[string]any{
			"id":            fmt.Sprintf("rst-bulk-%04d", now.UnixNano()%10000),
			"studentId":     fmt.Sprintf("%d Siswa", count),
			"studentName":   fmt.Sprintf("Reset Massal (%d Peserta Ruangan)", count),
			"subjectId":     subjectID,
			"proctorUserId": proctorUserID,
			"proctorName":   proctorName,
			"reason":        reason,
			"isBulk":        true,
			"createdAt":     now.UTC().Format(time.RFC3339),
		}
		doc["examResetLogs"] = append([]any{bulkLog}, asList(doc["examResetLogs"])...)
	}

	return count, nil
}

// AddBugReport records an incident or bug report directed to the developer admin page.
func AddBugReport(doc map[string]any, report map[string]any, now time.Time) string {
	reportID := fmt.Sprintf("bug-%04d", now.UnixNano()%10000)
	title := strings.TrimSpace(text(report["title"]))
	if title == "" {
		title = "Laporan Masalah Sistem"
	}
	category := strings.TrimSpace(text(report["category"]))
	if category == "" {
		category = "UMUM"
	}
	severity := strings.TrimSpace(text(report["severity"]))
	if severity == "" {
		severity = "medium"
	}
	entry := map[string]any{
		"id":             reportID,
		"reporterUserId": text(report["reporterUserId"]),
		"reporterName":   text(report["reporterName"]),
		"reporterRole":   text(report["reporterRole"]),
		"category":       category,
		"title":          title,
		"description":    strings.TrimSpace(text(report["description"])),
		"pageUrl":        strings.TrimSpace(text(report["pageUrl"])),
		"severity":       severity,
		"status":         "OPEN",
		"developerNotes": "",
		"createdAt":      now.UTC().Format(time.RFC3339),
		"updatedAt":      now.UTC().Format(time.RFC3339),
	}
	doc["bugReports"] = append([]any{entry}, asList(doc["bugReports"])...)
	return reportID
}

// UpdateBugReportStatus modifies status and appends developer notes on a bug report.
func UpdateBugReportStatus(doc map[string]any, reportID, status, devNotes string, now time.Time) error {
	for _, item := range asList(doc["bugReports"]) {
		rec := asMap(item)
		if rec != nil && text(rec["id"]) == reportID {
			if strings.TrimSpace(status) != "" {
				rec["status"] = strings.ToUpper(strings.TrimSpace(status))
			}
			if strings.TrimSpace(devNotes) != "" {
				rec["developerNotes"] = strings.TrimSpace(devNotes)
			}
			rec["updatedAt"] = now.UTC().Format(time.RFC3339)
			return nil
		}
	}
	return ErrNotFound
}
