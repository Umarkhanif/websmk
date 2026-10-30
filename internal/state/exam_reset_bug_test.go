package state

import (
	"testing"
	"time"
)

func TestResetStudentExam(t *testing.T) {
	doc := map[string]any{
		"students": []any{
			map[string]any{"id": "std-test-1", "fullName": "Test Student", "cbtScore": 85.0},
		},
		"proctorStudents": []any{
			map[string]any{
				"studentId": "std-test-1",
				"name":      "Test Student",
				"status":    "SUBMITTED",
				"progress":  "20/20",
				"lockCount": 2,
				"warnCount": 3,
			},
		},
		"examSubmissions": []any{
			map[string]any{
				"studentId": "std-test-1",
				"examId":    "exam-cbt-tkj-01",
				"mcqScore":  85,
			},
		},
	}

	now := time.Now()
	err := ResetStudentExam(doc, "std-test-1", "exam-cbt-tkj-01", "usr-teacher-1", "Ir. Hendro Wijaya", "Restart perangkat lab", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify submission removed
	submissions := asList(doc["examSubmissions"])
	if len(submissions) != 0 {
		t.Fatalf("expected 0 submissions, got %d", len(submissions))
	}

	// Verify proctor student status active & counters reset
	ps := asMap(asList(doc["proctorStudents"])[0])
	if text(ps["status"]) != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %s", text(ps["status"]))
	}
	if lockCount, _ := number(ps["lockCount"]); lockCount != 0 {
		t.Fatalf("expected lockCount 0, got %v", lockCount)
	}

	// Verify audit log added
	logs := asList(doc["examResetLogs"])
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	log0 := asMap(logs[0])
	if text(log0["studentId"]) != "std-test-1" {
		t.Fatalf("expected studentId std-test-1, got %s", text(log0["studentId"]))
	}
	if text(log0["proctorName"]) != "Ir. Hendro Wijaya" {
		t.Fatalf("expected proctor Ir. Hendro Wijaya, got %s", text(log0["proctorName"]))
	}
}

func TestBulkResetStudentExams(t *testing.T) {
	doc := map[string]any{
		"students": []any{
			map[string]any{"id": "std-1", "fullName": "Student 1"},
			map[string]any{"id": "std-2", "fullName": "Student 2"},
		},
		"proctorStudents": []any{
			map[string]any{"studentId": "std-1", "status": "SUBMITTED"},
			map[string]any{"studentId": "std-2", "status": "LOCKED"},
		},
	}

	now := time.Now()
	count, err := BulkResetStudentExams(doc, []string{"ALL"}, "exam-cbt-tkj-01", "usr-teacher-1", "Ir. Hendro Wijaya", "Reset massal lab", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 students reset, got %d", count)
	}

	for _, item := range asList(doc["proctorStudents"]) {
		ps := asMap(item)
		if text(ps["status"]) != "ACTIVE" {
			t.Fatalf("expected status ACTIVE, got %s", text(ps["status"]))
		}
	}
}

func TestBugReportsLifecycle(t *testing.T) {
	doc := map[string]any{}
	now := time.Now()

	repID := AddBugReport(doc, map[string]any{
		"reporterUserId": "usr-student-1",
		"reporterName":   "Budi Santoso",
		"reporterRole":   "student",
		"category":       "CBT_EXAM",
		"title":          "Layar freeze saat submit",
		"description":    "Layar membeku saat menekan tombol kirim",
		"pageUrl":        "/student/exam-kiosk",
		"severity":       "critical",
	}, now)

	if repID == "" {
		t.Fatal("expected non-empty report ID")
	}

	reports := asList(doc["bugReports"])
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	r0 := asMap(reports[0])
	if text(r0["status"]) != "OPEN" {
		t.Fatalf("expected status OPEN, got %s", text(r0["status"]))
	}

	// Update status
	err := UpdateBugReportStatus(doc, repID, "INVESTIGATING", "Sedang dicek log Redis", now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("unexpected error updating bug status: %v", err)
	}

	r0Updated := asMap(asList(doc["bugReports"])[0])
	if text(r0Updated["status"]) != "INVESTIGATING" {
		t.Fatalf("expected status INVESTIGATING, got %s", text(r0Updated["status"]))
	}
	if text(r0Updated["developerNotes"]) != "Sedang dicek log Redis" {
		t.Fatalf("unexpected developer notes: %s", text(r0Updated["developerNotes"]))
	}
}
