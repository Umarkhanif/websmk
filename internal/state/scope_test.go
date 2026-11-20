package state

import (
	"encoding/json"
	"testing"
)

const sampleState = `{
  "students": [
    {"id": "std-1", "fullName": "Budi", "dailyQuizAvg": 80, "bmwChoice": "BEKERJA"},
    {"id": "std-2", "fullName": "Rahma", "dailyQuizAvg": 90, "bmwChoice": "MELANJUTKAN"}
  ],
  "invoices": [
    {"id": "inv-1", "studentId": "std-1", "studentName": "Budi", "finalAmount": 350000, "paidAmount": 100000, "status": "PARTIAL"},
    {"id": "inv-2", "studentId": "std-2", "studentName": "Rahma", "finalAmount": 350000, "paidAmount": 0, "status": "UNPAID"}
  ],
  "ledger": [
    {"id": 1, "invoiceId": "inv-1", "txType": "PAYMENT_VA", "amountDelta": 100000},
    {"id": 2, "invoiceId": "inv-2", "txType": "PAYMENT_VA", "amountDelta": 350000}
  ],
  "studentExpenses": [
    {"id": "exp-1", "studentId": "std-1", "amount": 50000},
    {"id": "exp-2", "studentId": "std-2", "amount": 75000}
  ],
  "dispensations": [{"id": "disp-1", "studentId": "std-1"}],
  "studentDocuments": [{"id": "doc-1", "studentId": "std-1", "status": "MENUNGGU_CEK"}],
  "taskSubmissions": [{"id": "task-1", "studentId": "std-1", "status": "TERKIRIM"}],
  "pklAttendanceLogs": [{"id": "pkl-att-1", "studentId": "std-1"}],
  "pklJournals": [{"id": "pkl-j-1", "studentId": "std-1"}],
  "bkkApplications": [{"id": "app-1", "studentId": "std-1"}],
  "attendances": [
    {"id": "att-1", "studentId": "std-1", "status": "HADIR_GERBANG"},
    {"id": "att-2", "studentId": "std-2", "status": "HADIR_GERBANG"}
  ],
  "proctorStudents": [
    {"id": "proc-1", "studentId": "std-1", "status": "ACTIVE"},
    {"id": "proc-2", "studentId": "std-2", "status": "ACTIVE"}
  ],
  "studentReflection": {"id": "ref-1", "studentId": "std-1", "privateComplaintText": "Bingung subnetting"},
  "careerConsultations": [
    {"id": "cons-1", "studentId": "std-1"},
    {"id": "cons-2", "studentId": "std-2"}
  ],
  "peerReports": [{"id": "pr-1", "targetStudentName": "Seseorang", "status": "VERIFIED_VALID"}],
  "classJournal": {"id": "journal-1", "subjectName": "Jaringan"},
  "dailyQuiz": {"id": "quiz-1", "questions": [{"id": "q1", "correctKey": "A"}]},
  "examQuestions": [{"id": "eq-1", "correctKey": "B"}],
  "bkkJobs": [{"id": "job-1", "title": "Teknisi Fiber"}],
  "pklPlacements": [{"id": "place-1", "supervisorPin": "123456"}],
  "violations": [{"id": "v-1", "studentId": "std-2", "pointsDeducted": 10}],
  "reversalRequests": [{"id": "rev-1"}],
  "activeHeadmasterOtp": "981240"
}`

func mustDoc(t *testing.T, raw string) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return doc
}

func records(t *testing.T, doc map[string]any, key string) []map[string]any {
	t.Helper()
	raw, ok := doc[key].([]any)
	if !ok {
		t.Fatalf("%s is not a list: %T", key, doc[key])
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		record, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("%s contains %T", key, item)
		}
		out = append(out, record)
	}
	return out
}

func ids(t *testing.T, doc map[string]any, key string) []string {
	t.Helper()
	out := []string{}
	for _, record := range records(t, doc, key) {
		out = append(out, recordID(record["id"]))
	}
	return out
}

func findByID(t *testing.T, doc map[string]any, key, id string) map[string]any {
	t.Helper()
	for _, record := range records(t, doc, key) {
		if recordID(record["id"]) == id {
			return record
		}
	}
	return nil
}

func TestIsScopedRole(t *testing.T) {
	for _, role := range []string{"student", "parent"} {
		if !IsScopedRole(role) {
			t.Fatalf("%s should be scoped", role)
		}
	}
	for _, role := range []string{"teacher", "treasurer", "discipline", "piket", "admin_tu", "headmaster"} {
		if IsScopedRole(role) {
			t.Fatalf("%s should not be scoped", role)
		}
	}
}

func TestScopedViewStudentOnlyOwn(t *testing.T) {
	doc := mustDoc(t, sampleState)

	view := ScopedView(doc, Access{Role: "student", StudentID: "std-1"})

	if got := ids(t, view, "students"); len(got) != 1 || got[0] != "std-1" {
		t.Fatalf("students = %v, want only std-1", got)
	}
	want := map[string]string{
		"invoices":          "inv-1",
		"studentExpenses":   "exp-1",
		"dispensations":     "disp-1",
		"studentDocuments":  "doc-1",
		"taskSubmissions":   "task-1",
		"pklAttendanceLogs": "pkl-att-1",
		"pklJournals":       "pkl-j-1",
		"bkkApplications":   "app-1",
		"attendances":       "att-1",
		"proctorStudents":   "proc-1",
	}
	for key, id := range want {
		got := ids(t, view, key)
		if len(got) != 1 || got[0] != id {
			t.Fatalf("%s = %v, want only %s", key, got, id)
		}
	}
	if _, ok := view["violations"]; ok {
		t.Fatal("student must not receive the discipline registry")
	}
	if _, ok := view["reversalRequests"]; ok {
		t.Fatal("student must not receive finance reversal requests")
	}
	if _, ok := view["activeHeadmasterOtp"]; ok {
		t.Fatal("student must not receive the headmaster OTP")
	}
	if _, ok := view["ledger"]; ok {
		t.Fatal("student must not receive the audit ledger")
	}
	if _, ok := view["peerReports"]; ok {
		t.Fatal("student must not receive the anonymous peer report registry")
	}

	for _, key := range []string{"classJournal", "dailyQuiz", "bkkJobs", "examQuestions", "pklPlacements"} {
		if _, ok := view[key]; !ok {
			t.Fatalf("student view is missing class-wide %s", key)
		}
	}

	// The browser must not receive the answer key: scoring happens on the
	// server, so a student cannot read tomorrow's correct option.
	quiz := asMap(view["dailyQuiz"])
	firstQuizQuestion := asMap(asList(quiz["questions"])[0])
	if _, ok := firstQuizQuestion["correctKey"]; ok {
		t.Fatal("quiz answer key leaked to the student view")
	}
	firstExamQuestion := asMap(asList(view["examQuestions"])[0])
	if _, ok := firstExamQuestion["correctKey"]; ok {
		t.Fatal("exam answer key leaked to the student view")
	}
	if _, ok := firstQuizQuestion["id"]; !ok {
		t.Fatal("redaction must keep the question id and other fields")
	}
	if _, ok := asMap(asList(asMap(doc["dailyQuiz"])["questions"])[0])["correctKey"]; !ok {
		t.Fatal("redaction must not mutate the stored answer key")
	}
}

func TestScopedViewParentOnlyLinkedChild(t *testing.T) {
	doc := mustDoc(t, sampleState)

	view := ScopedView(doc, Access{Role: "parent", StudentID: "std-1"})

	if got := ids(t, view, "students"); len(got) != 1 || got[0] != "std-1" {
		t.Fatalf("students = %v, want only std-1", got)
	}
	if got := ids(t, view, "invoices"); len(got) != 1 || got[0] != "inv-1" {
		t.Fatalf("invoices = %v, want only inv-1", got)
	}
	if got := ids(t, view, "ledger"); len(got) != 1 || got[0] != "1" {
		t.Fatalf("ledger = %v, want only the invoice-1 entry", got)
	}
	if got := ids(t, view, "attendances"); len(got) != 1 || got[0] != "att-1" {
		t.Fatalf("attendances = %v, want only att-1", got)
	}
	if got := ids(t, view, "studentExpenses"); len(got) != 1 || got[0] != "exp-1" {
		t.Fatalf("studentExpenses = %v, want only exp-1", got)
	}
	if got := ids(t, view, "careerConsultations"); len(got) != 1 || got[0] != "cons-1" {
		t.Fatalf("careerConsultations = %v, want only cons-1", got)
	}
	reflection, ok := view["studentReflection"].(map[string]any)
	if !ok || text(reflection["studentId"]) != "std-1" {
		t.Fatalf("studentReflection = %#v, want the child's reflection", view["studentReflection"])
	}
	for _, key := range []string{"violations", "examQuestions", "peerReports", "activeHeadmasterOtp", "reversalRequests"} {
		if _, ok := view[key]; ok {
			t.Fatalf("parent view must not expose %s", key)
		}
	}
}

func TestScopedViewParentWithoutReflectionGetsEmptyRecord(t *testing.T) {
	doc := mustDoc(t, sampleState)

	view := ScopedView(doc, Access{Role: "parent", StudentID: "std-2"})

	reflection, ok := view["studentReflection"].(map[string]any)
	if !ok {
		t.Fatal("studentReflection must always be present for parents")
	}
	if text(reflection["studentId"]) != "std-2" || text(reflection["privateComplaintText"]) != "" {
		t.Fatalf("reflection = %#v, want a blank record for std-2", reflection)
	}
}

func TestMergeOwnedRejectsForeignRecords(t *testing.T) {
	existing := mustDoc(t, sampleState)
	incoming := mustDoc(t, `{
      "students": [
        {"id": "std-1", "dailyQuizAvg": 95},
        {"id": "std-2", "dailyQuizAvg": 5}
      ],
      "invoices": [
        {"id": "inv-1", "studentId": "std-1", "paidAmount": 350000},
        {"id": "inv-2", "studentId": "std-1", "paidAmount": 0}
      ],
      "studentDocuments": [{"id": "doc-1", "studentId": "std-1", "fileName": "kk.pdf", "status": "VALID_DISETUJUI"}],
      "attendances": [{"id": "att-1", "studentId": "std-1", "status": "HADIR_GERBANG"}],
      "proctorStudents": [{"id": "proc-1", "studentId": "std-1", "status": "SUBMITTED"}]
    }`)

	merged, err := MergeOwned(existing, incoming, Access{Role: "student", StudentID: "std-1"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	student1 := findStudent(merged, "std-1")
	student2 := findStudent(merged, "std-2")
	// Grades are server-owned now, so even the student's own write is ignored.
	if got, _ := number(student1["dailyQuizAvg"]); got != 80 {
		t.Fatalf("own quiz average = %v, want the server value 80", got)
	}
	if got, _ := number(student2["dailyQuizAvg"]); got != 90 {
		t.Fatalf("foreign quiz average = %v, want it unchanged at 90", got)
	}

	ownInvoice := findOwnedRecord(asList(merged["invoices"]), "inv-1", "std-1")
	if paid, _ := number(ownInvoice["paidAmount"]); paid != 100000 {
		t.Fatalf("own invoice paid = %v, want the server value 100000", paid)
	}
	if status := text(ownInvoice["status"]); status != "PARTIAL" {
		t.Fatalf("own invoice status = %q, want PARTIAL", status)
	}

	invoice2 := findOwnedRecord(asList(merged["invoices"]), "inv-2", "std-2")
	if name := text(invoice2["studentName"]); name != "Rahma" {
		t.Fatalf("foreign invoice owner = %q, want Rahma", name)
	}
	if paid, _ := number(invoice2["paidAmount"]); paid != 0 {
		t.Fatalf("foreign invoice paid = %v, want 0", paid)
	}

	document := findOwnedRecord(asList(merged["studentDocuments"]), "doc-1", "std-1")
	if status := text(document["status"]); status != "MENUNGGU_CEK" {
		t.Fatalf("document status = %q, want the upload to re-enter review", status)
	}
	if file := text(document["fileName"]); file != "" {
		t.Fatalf("document file = %q, want server-owned metadata (client write ignored)", file)
	}

	attendance := findOwnedRecord(asList(merged["attendances"]), "att-1", "std-1")
	if status := text(attendance["status"]); status != "HADIR_GERBANG" {
		t.Fatalf("student attendance = %q, want it unchanged (parents own leave requests)", status)
	}

	proctor := findOwnedRecord(asList(merged["proctorStudents"]), "proc-1", "std-1")
	if status := text(proctor["status"]); status != "ACTIVE" {
		t.Fatalf("proctor status = %q, want server-owned ACTIVE (client write ignored)", status)
	}
}

func TestMergeOwnedAppendsOnlyOwnRecords(t *testing.T) {
	existing := mustDoc(t, sampleState)
	incoming := mustDoc(t, `{
      "taskSubmissions": [
        {"id": "task-new", "studentId": "std-1", "status": "DINILAI", "score": 100, "fileName": "tugas.pdf"},
        {"id": "task-foreign", "studentId": "std-2", "fileName": "bocor.pdf"}
      ],
      "pklJournals": [{"id": "pkl-new", "studentId": "std-1", "supervisorApproved": true, "teacherApproved": true}],
      "bkkApplications": [{"id": "app-new", "studentId": "std-1", "status": "DITERIMA"}],
      "studentExpenses": [{"id": "exp-new", "studentId": "std-1", "amount": 20000}],
      "peerReports": [{"id": "pr-new", "targetStudentName": "X", "status": "VERIFIED_VALID"}],
      "violations": [{"id": "v-new", "studentId": "std-2", "pointsDeducted": 50}]
    }`)

	merged, err := MergeOwned(existing, incoming, Access{Role: "student", StudentID: "std-1"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	task := findOwnedRecord(asList(merged["taskSubmissions"]), "task-new", "std-1")
	if status := text(task["status"]); status != "TERKIRIM" {
		t.Fatalf("new task status = %q, want TERKIRIM", status)
	}
	if _, ok := task["score"]; ok {
		t.Fatal("new task must not carry a self-assigned score")
	}
	if findOwnedRecord(asList(merged["taskSubmissions"]), "task-foreign", "std-2") != nil {
		t.Fatal("a student must not append submissions for another student")
	}

	journal := findOwnedRecord(asList(merged["pklJournals"]), "pkl-new", "std-1")
	if approved, _ := journal["supervisorApproved"].(bool); approved {
		t.Fatal("a student must not self-approve a PKL journal")
	}
	application := findOwnedRecord(asList(merged["bkkApplications"]), "app-new", "std-1")
	if status := text(application["status"]); status != "TERKIRIM" {
		t.Fatalf("new application status = %q, want TERKIRIM", status)
	}
	if findOwnedRecord(asList(merged["studentExpenses"]), "exp-new", "std-1") == nil {
		t.Fatal("own expense was not appended")
	}

	report := findByID(t, merged, "peerReports", "pr-new")
	if report == nil {
		t.Fatal("peer report was not appended")
	}
	if status := text(report["status"]); status != "PENDING_ESCROW" {
		t.Fatalf("peer report status = %q, want PENDING_ESCROW", status)
	}
	if findOwnedRecord(asList(merged["violations"]), "v-new", "std-2") != nil {
		t.Fatal("a student must not append discipline violations")
	}
}

func TestMergeOwnedIgnoresClientPaymentWrites(t *testing.T) {
	existing := mustDoc(t, sampleState)

	incoming := mustDoc(t, `{
      "invoices":[{"id":"inv-1","studentId":"std-1","paidAmount":350000,"status":"PAID"}],
      "ledger":[{"id":999,"invoiceId":"inv-1","txType":"PAYMENT_VA","amountDelta":250000,"signatureHmac":"palsu"}]
    }`)
	if _, err := MergeOwned(existing, incoming, Access{Role: "student", StudentID: "std-1"}); err != nil {
		t.Fatalf("merge payment: %v", err)
	}

	invoice := findOwnedRecord(asList(existing["invoices"]), "inv-1", "std-1")
	if paid, _ := number(invoice["paidAmount"]); paid != 100000 {
		t.Fatalf("paid after a client write = %v, want the server value 100000", paid)
	}
	if status := text(invoice["status"]); status != "PARTIAL" {
		t.Fatalf("status after a client write = %q, want PARTIAL", status)
	}
	if got := len(asList(existing["ledger"])); got != 2 {
		t.Fatalf("ledger entries = %d, want the client receipt rejected", got)
	}

	parent := mustDoc(t, `{"invoices":[{"id":"inv-1","studentId":"std-1","paidAmount":0}]}`)
	if _, err := MergeOwned(existing, parent, Access{Role: "parent", StudentID: "std-1"}); err != nil {
		t.Fatalf("merge parent payment: %v", err)
	}
	invoice = findOwnedRecord(asList(existing["invoices"]), "inv-1", "std-1")
	if paid, _ := number(invoice["paidAmount"]); paid != 100000 {
		t.Fatalf("paid after a parent write = %v, want 100000", paid)
	}
}

func TestMergeOwnedParentLeaveAndConsultation(t *testing.T) {
	existing := mustDoc(t, sampleState)
	incoming := mustDoc(t, `{
      "attendances": [
        {"id": "att-1", "studentId": "std-1", "status": "SAKIT_ORTU", "verifiedSource": "Surat dokter", "updatedAt": "2026-10-25T08:00:00Z"},
        {"id": "att-2", "studentId": "std-2", "status": "ALPA_BOLOS"}
      ],
      "careerConsultations": [{"id": "cons-new", "studentId": "std-1", "status": "TERJADWAL_BK"}]
    }`)

	merged, err := MergeOwned(existing, incoming, Access{Role: "parent", StudentID: "std-1"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	attendance := findOwnedRecord(asList(merged["attendances"]), "att-1", "std-1")
	if status := text(attendance["status"]); status != "SAKIT_ORTU" {
		t.Fatalf("own child attendance = %q, want SAKIT_ORTU", status)
	}
	if source := text(attendance["verifiedSource"]); source != "Surat dokter" {
		t.Fatalf("leave source = %q, want the parent's note", source)
	}

	foreign := findOwnedRecord(asList(merged["attendances"]), "att-2", "std-2")
	if status := text(foreign["status"]); status != "HADIR_GERBANG" {
		t.Fatalf("foreign attendance = %q, want it unchanged", status)
	}

	consultation := findOwnedRecord(asList(merged["careerConsultations"]), "cons-new", "std-1")
	if status := text(consultation["status"]); status != "MENUNGGU_JADWAL" {
		t.Fatalf("new consultation status = %q, want MENUNGGU_JADWAL", status)
	}

	// A parent cannot escalate an absence into a school-side status.
	escalation := mustDoc(t, `{"attendances":[{"id":"att-1","studentId":"std-1","status":"ALPA_BOLOS"}]}`)
	if _, err := MergeOwned(merged, escalation, Access{Role: "parent", StudentID: "std-1"}); err != nil {
		t.Fatalf("merge escalation: %v", err)
	}
	attendance = findOwnedRecord(asList(merged["attendances"]), "att-1", "std-1")
	if status := text(attendance["status"]); status != "SAKIT_ORTU" {
		t.Fatalf("attendance after escalation = %q, want SAKIT_ORTU", status)
	}
}

func TestMergeOwnedRequiresLinkedStudent(t *testing.T) {
	existing := mustDoc(t, sampleState)

	_, err := MergeOwned(existing, map[string]any{"students": []any{}}, Access{Role: "student"})
	if err == nil {
		t.Fatal("merge without a linked student must fail")
	}
}

func TestScopedViewEmptyDocument(t *testing.T) {
	if view := ScopedView(nil, Access{Role: "student", StudentID: "std-1"}); view != nil {
		t.Fatalf("ScopedView(nil) = %#v, want nil", view)
	}
}
