package state

import "testing"

const staffDoc = `{
  "students": [{"id": "std-1", "nisn": "0058471290"}],
  "invoices": [{"id": "inv-1", "studentId": "std-1", "paidAmount": 350000}],
  "ledger": [{"id": 1, "invoiceId": "inv-1", "amountDelta": 350000}],
  "examQuestions": [{"id": "q1", "correctKey": "A", "essayRubric": "RAHASIA"}],
  "studentReflection": {"studentId": "std-1", "privateComplaintText": "Rahasia BK"},
  "reversalRequests": [{"id": "rev-1", "otpCode": "981240", "amountOriginal": 350000}],
  "counselingSessions": [{"id": "c-1", "notes": "Rahasia konseling"}],
  "activeHeadmasterOtp": "981240",
  "activeHeadmasterOtpExpiresAt": "2026-11-20T12:00:00Z",
  "bugReports": [{"id": "bug-1", "description": "laporan"}],
  "examResetLogs": [{"id": "reset-1"}],
  "officialLetters": [{"id": "ltr-1"}],
  "teacherImpacts": [{"id": "t-1", "finalImpactScore": 90}]
}`

func TestStaffViewTreasurer(t *testing.T) {
	view := StaffView(mustDoc(t, staffDoc), "treasurer")
	for _, key := range []string{"students", "invoices", "ledger", "reversalRequests", "dispensations"} {
		if _, ok := view[key]; !ok && key != "dispensations" {
			t.Fatalf("treasurer view missing %q", key)
		}
	}
	for _, key := range []string{"examQuestions", "studentReflection", "counselingSessions", "bugReports", "activeHeadmasterOtp", "activeHeadmasterOtpExpiresAt"} {
		if _, ok := view[key]; ok {
			t.Fatalf("treasurer view must not contain %q", key)
		}
	}
	requests := asList(view["reversalRequests"])
	if len(requests) != 1 {
		t.Fatalf("reversalRequests = %v", requests)
	}
	if _, ok := asMap(requests[0])["otpCode"]; ok {
		t.Fatal("treasurer view must not expose reversal OTP")
	}
}

func TestStaffViewTeacherKeepsKeysButNoFinance(t *testing.T) {
	view := StaffView(mustDoc(t, staffDoc), "teacher")
	if _, ok := view["examQuestions"]; !ok {
		t.Fatal("teacher needs exam questions")
	}
	questions := asList(view["examQuestions"])
	if text(asMap(questions[0])["correctKey"]) != "A" {
		t.Fatal("teacher needs the answer key to grade")
	}
	for _, key := range []string{"invoices", "ledger", "activeHeadmasterOtp", "counselingSessions"} {
		if _, ok := view[key]; ok {
			t.Fatalf("teacher view must not contain %q", key)
		}
	}
}

func TestStaffViewDisciplineAndPiket(t *testing.T) {
	discipline := StaffView(mustDoc(t, staffDoc), "discipline")
	if _, ok := discipline["counselingSessions"]; !ok {
		t.Fatal("discipline needs counseling sessions")
	}
	if _, ok := discipline["invoices"]; ok {
		t.Fatal("discipline must not see invoices")
	}

	piket := StaffView(mustDoc(t, staffDoc), "piket")
	if _, ok := piket["teacherImpacts"]; !ok {
		t.Fatal("piket needs teacher impacts")
	}
	if _, ok := piket["students"]; ok {
		t.Fatal("piket view must not contain students")
	}
}

func TestStaffViewHeadmasterAndDeveloper(t *testing.T) {
	head := StaffView(mustDoc(t, staffDoc), "headmaster")
	if text(head["activeHeadmasterOtp"]) != "981240" {
		t.Fatal("headmaster keeps own OTP")
	}

	dev := StaffView(mustDoc(t, staffDoc), "developer")
	if _, ok := dev["bugReports"]; !ok {
		t.Fatal("developer needs bug reports")
	}
	for _, key := range []string{"students", "invoices", "activeHeadmasterOtp", "examQuestions"} {
		if _, ok := dev[key]; ok {
			t.Fatalf("developer view must not contain %q", key)
		}
	}
}

func TestStaffViewUnknownRoleSeesNothing(t *testing.T) {
	view := StaffView(mustDoc(t, staffDoc), "unknown")
	if len(view) != 0 {
		t.Fatalf("unknown role view = %v, want empty", view)
	}
}
