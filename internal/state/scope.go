// Package state applies per-role visibility and write rules to the single
// application state document.
//
// Students see only their own records; parents see only their linked child.
// Staff keep the full snapshot. Writes from students/parents are merged into
// the global document instead of replacing it, so a scoped client can never
// wipe or tamper with records that belong to someone else.
package state

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
)

type Access struct {
	Role      string // "student" | "parent"
	StudentID string
}

func IsScopedRole(role string) bool {
	return role == "student" || role == "parent"
}

// IsStaffRole reports whether the role is a staff/developer account rather than
// a student or parent account.
func IsStaffRole(role string) bool {
	switch role {
	case "treasurer", "teacher", "discipline", "piket", "admin_tu", "headmaster", "developer":
		return true
	}
	return false
}

// staffKeys is the read allowlist per staff role. A role only receives the
// keys its portal actually renders; everything else stays on the server.
var staffKeys = map[string][]string{
	"treasurer": {"students", "invoices", "dispensations", "ledger", "reversalRequests"},
	"teacher":   {"students", "classJournal", "attendances", "examQuestions", "proctorStudents", "pklAttendanceLogs", "pklJournals", "examResetLogs"},
	"discipline": {"students", "violations", "peerReports", "careerConsultations",
		"counselingSessions"},
	"piket":     {"piketClassrooms", "piketGateLogs", "teacherImpacts"},
	"admin_tu":  {"students", "majors", "studentDocuments", "officialLetters", "dapodikLogs", "pklPlacements", "bkkJobs", "bkkApplications", "teacherImpacts"},
	"developer": {"bugReports", "examResetLogs"},
}

// StaffView returns the subset of the state document a staff role may read.
// The headmaster keeps the full document; all other roles get an allowlist with
// sensitive subfields (answer keys, OTP, private complaints) removed.
func StaffView(doc map[string]any, role string) map[string]any {
	if doc == nil {
		return nil
	}
	if role == "headmaster" {
		out := make(map[string]any, len(doc))
		for key, value := range doc {
			out[key] = value
		}
		return out
	}
	keys, ok := staffKeys[role]
	if !ok {
		return map[string]any{}
	}
	out := map[string]any{}
	for _, key := range keys {
		if value, exists := doc[key]; exists {
			out[key] = value
		}
	}
	// Teachers grade essays and run the exam kiosk, so they keep answer keys.
	// Every other staff role gets them stripped.
	if role != "teacher" {
		if value, exists := out["examQuestions"]; exists {
			out["examQuestions"] = redactKeys(value, "correctKey", "essayRubric")
		}
		if value, exists := out["dailyQuiz"]; exists {
			out["dailyQuiz"] = redactKeys(value, "correctKey", "explanation")
		}
	}
	if value, exists := out["studentReflection"]; exists {
		out["studentReflection"] = redactKeys(value, "privateComplaintText")
	}
	if value, exists := out["reversalRequests"]; exists {
		out["reversalRequests"] = redactKeys(value, "otpCode")
	}
	delete(out, "activeHeadmasterOtp")
	delete(out, "activeHeadmasterOtpExpiresAt")
	return out
}

var validBMWChoices = map[string]bool{
	"BEKERJA":     true,
	"MELANJUTKAN": true,
	"WIRAUSAHA":   true,
}

var validParentAttendance = map[string]bool{
	"SAKIT_ORTU": true,
	"IZIN_ORTU":  true,
}

// ScopedView returns the subset of the state a student or parent may read.
func ScopedView(doc map[string]any, access Access) map[string]any {
	if doc == nil {
		return nil
	}

	out := map[string]any{}
	student := findStudent(doc, access.StudentID)
	if student != nil {
		out["students"] = []any{student}
	} else {
		out["students"] = []any{}
	}

	out["invoices"] = filterOwned(doc, "invoices", access.StudentID)
	out["studentExpenses"] = filterOwned(doc, "studentExpenses", access.StudentID)

	if access.Role == "parent" {
		// ParentView only needs the child's profile, billing, expenses,
		// consultation booking, attendance (for leave requests), and the radar
		// reflection.
		out["ledger"] = ownLedger(doc, access.StudentID)
		out["careerConsultations"] = filterOwned(doc, "careerConsultations", access.StudentID)
		out["attendances"] = filterOwned(doc, "attendances", access.StudentID)
		out["studentReflection"] = ownReflectionOrEmpty(doc, access.StudentID)
		return out
	}

	for _, key := range []string{
		"dispensations", "studentDocuments", "taskSubmissions",
		"pklAttendanceLogs", "pklJournals", "bkkApplications",
		"attendances", "proctorStudents",
	} {
		out[key] = filterOwned(doc, key, access.StudentID)
	}
	if reflection := ownReflection(doc, access.StudentID); reflection != nil {
		out["studentReflection"] = reflection
	}

	// Class-wide material the student portal renders: journal, quiz, lowongan,
	// exam kiosk, and the single PKL placement. These carry no other students'
	// personal records.
	for _, key := range []string{"classJournal", "bkkJobs", "pklPlacements", "examSchedules"} {
		if value, ok := doc[key]; ok {
			out[key] = value
		}
	}
	// Answer keys and rubrics stay on the server; the browser must never be able
	// to score itself or read tomorrow's correct answer.
	if value, ok := doc["dailyQuiz"]; ok {
		out["dailyQuiz"] = redactKeys(value, "correctKey", "explanation")
	}
	if value, ok := doc["examQuestions"]; ok {
		out["examQuestions"] = redactKeys(value, "correctKey", "essayRubric")
	}

	return out
}

// redactKeys deep-copies value without the given object keys, leaving the
// stored document untouched.
func redactKeys(value any, keys ...string) any {
	switch typed := value.(type) {
	case map[string]any:
		copyRecord := make(map[string]any, len(typed))
		for key, item := range typed {
			if slices.Contains(keys, key) {
				continue
			}
			copyRecord[key] = redactKeys(item, keys...)
		}
		return copyRecord
	case []any:
		copyList := make([]any, len(typed))
		for index, item := range typed {
			copyList[index] = redactKeys(item, keys...)
		}
		return copyList
	default:
		return value
	}
}

// MergeOwned applies the writable subset of a student/parent snapshot onto the
// existing global document. Records owned by someone else are ignored, as are
// global/administrative slices.
func MergeOwned(existing, incoming map[string]any, access Access) (map[string]any, error) {
	if len(existing) == 0 {
		// First write on an empty database: keep the client bootstrap snapshot
		// so the demo has data to show. Subsequent writes are merged below.
		return incoming, nil
	}
	if access.StudentID == "" {
		return existing, errors.New("missing student profile")
	}

	student := findStudent(existing, access.StudentID)
	if student == nil {
		return existing, nil
	}

	mergeStudentProfile(student, findStudent(incoming, access.StudentID), access.Role)
	// Grades and invoice balances are server-owned: scoped clients cannot set
	// dailyQuizAvg, paidAmount, or append receipts. They call the dedicated
	// endpoints instead.

	switch access.Role {
	case "student":
		mergeReflection(existing, incoming, access.StudentID)
		// Proctor status and uploaded document metadata are server-owned:
		// violations and uploads go through dedicated endpoints only.
		appendRecords(existing, incoming, "taskSubmissions", access.StudentID, func(rec map[string]any) {
			rec["status"] = "TERKIRIM"
			delete(rec, "score")
		})
		appendRecords(existing, incoming, "pklAttendanceLogs", access.StudentID, nil)
		appendRecords(existing, incoming, "pklJournals", access.StudentID, func(rec map[string]any) {
			rec["supervisorApproved"] = false
			rec["teacherApproved"] = false
			delete(rec, "approvedAt")
		})
		appendRecords(existing, incoming, "bkkApplications", access.StudentID, func(rec map[string]any) {
			rec["status"] = "TERKIRIM"
		})
		appendRecords(existing, incoming, "studentExpenses", access.StudentID, nil)
		appendPeerReports(existing, incoming)
	case "parent":
		mergeAttendances(existing, incoming, access.StudentID)
		appendRecords(existing, incoming, "careerConsultations", access.StudentID, func(rec map[string]any) {
			rec["status"] = "MENUNGGU_JADWAL"
		})
	}

	return existing, nil
}

func findStudent(doc map[string]any, studentID string) map[string]any {
	if studentID == "" {
		return nil
	}
	for _, item := range asList(doc["students"]) {
		record := asMap(item)
		if record != nil && recordID(record["id"]) == studentID {
			return record
		}
	}
	return nil
}

func filterOwned(doc map[string]any, key, studentID string) []any {
	out := []any{}
	for _, item := range asList(doc[key]) {
		record := asMap(item)
		if record != nil && text(record["studentId"]) == studentID {
			out = append(out, record)
		}
	}
	return out
}

func ownLedger(doc map[string]any, studentID string) []any {
	invoiceIDs := ownInvoiceIDs(doc, studentID)
	out := []any{}
	for _, item := range asList(doc["ledger"]) {
		record := asMap(item)
		if record != nil && invoiceIDs[text(record["invoiceId"])] {
			out = append(out, record)
		}
	}
	return out
}

func ownInvoiceIDs(doc map[string]any, studentID string) map[string]bool {
	ids := map[string]bool{}
	for _, item := range asList(doc["invoices"]) {
		record := asMap(item)
		if record != nil && text(record["studentId"]) == studentID {
			ids[recordID(record["id"])] = true
		}
	}
	return ids
}

func ownReflection(doc map[string]any, studentID string) map[string]any {
	record := asMap(doc["studentReflection"])
	if record == nil || text(record["studentId"]) != studentID {
		return nil
	}
	return record
}

// ownReflectionOrEmpty keeps the key present for parents so a child without a
// reflection never leaves another student's reflection in the browser state.
func ownReflectionOrEmpty(doc map[string]any, studentID string) map[string]any {
	if record := ownReflection(doc, studentID); record != nil {
		return record
	}
	return map[string]any{
		"id":                   "",
		"journalId":            "",
		"studentId":            studentID,
		"studentName":          "",
		"understandingLevel":   "UNDERSTOOD",
		"privateComplaintText": "",
		"createdAt":            "",
	}
}

func mergeStudentProfile(existing, incoming map[string]any, role string) {
	if existing == nil || incoming == nil || role != "student" {
		return
	}
	if choice := text(incoming["bmwChoice"]); validBMWChoices[choice] {
		existing["bmwChoice"] = choice
	}
}

func mergeReflection(existing, incoming map[string]any, studentID string) {
	record := asMap(incoming["studentReflection"])
	if record == nil || text(record["studentId"]) != studentID {
		return
	}
	// Never overwrite a reflection that belongs to another student; the demo
	// model keeps a single reflection record.
	if current := asMap(existing["studentReflection"]); current != nil && text(current["studentId"]) != studentID {
		return
	}
	existing["studentReflection"] = record
}

func mergeAttendances(existing, incoming map[string]any, studentID string) {
	list := asList(existing["attendances"])
	for _, item := range asList(incoming["attendances"]) {
		record := asMap(item)
		if record == nil || text(record["studentId"]) != studentID {
			continue
		}
		status := text(record["status"])
		if !validParentAttendance[status] {
			continue
		}
		target := findOwnedRecord(list, recordID(record["id"]), studentID)
		if target == nil {
			continue
		}
		target["status"] = status
		if source, ok := record["verifiedSource"]; ok {
			target["verifiedSource"] = source
		}
		if updatedAt, ok := record["updatedAt"]; ok {
			target["updatedAt"] = updatedAt
		}
	}
}

func appendRecords(existing, incoming map[string]any, key, studentID string, sanitize func(map[string]any)) {
	current := asList(existing[key])
	seen := map[string]bool{}
	for _, item := range current {
		if record := asMap(item); record != nil {
			seen[recordID(record["id"])] = true
		}
	}

	for _, item := range asList(incoming[key]) {
		record := asMap(item)
		if record == nil || text(record["studentId"]) != studentID {
			continue
		}
		id := recordID(record["id"])
		if id == "" || seen[id] {
			continue
		}
		if sanitize != nil {
			sanitize(record)
		}
		current = append(current, record)
		seen[id] = true
	}
	existing[key] = current
}

func appendPeerReports(existing, incoming map[string]any) {
	current := asList(existing["peerReports"])
	seen := map[string]bool{}
	for _, item := range current {
		if record := asMap(item); record != nil {
			seen[recordID(record["id"])] = true
		}
	}

	for _, item := range asList(incoming["peerReports"]) {
		record := asMap(item)
		if record == nil {
			continue
		}
		id := recordID(record["id"])
		if id == "" || seen[id] {
			continue
		}
		// Anonymous reports are append-only and always re-enter escrow review.
		record["status"] = "PENDING_ESCROW"
		current = append(current, record)
		seen[id] = true
	}
	existing["peerReports"] = current
}

func findOwnedRecord(list []any, id, studentID string) map[string]any {
	if id == "" || studentID == "" {
		return nil
	}
	for _, item := range list {
		record := asMap(item)
		if record != nil && recordID(record["id"]) == id && text(record["studentId"]) == studentID {
			return record
		}
	}
	return nil
}

func asList(value any) []any {
	list, _ := value.([]any)
	return list
}

func asMap(value any) map[string]any {
	record, _ := value.(map[string]any)
	return record
}

func text(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func number(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func recordID(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}
