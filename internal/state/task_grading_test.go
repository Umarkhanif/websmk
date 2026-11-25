package state

import (
	"testing"
	"time"
)

func TestSubmitStudentTaskStoresBlobMetadata(t *testing.T) {
	doc := map[string]any{"students": []any{map[string]any{"id": "siswa-1", "fullName": "Siswa Uji"}}}
	if err := SubmitStudentTask(doc, "siswa-1", "", "Laporan VLAN", "vlan.pdf", "blob-1", "application/pdf", 120, time.Now()); err != nil {
		t.Fatalf("submit: %v", err)
	}
	record := asMap(asList(doc["taskSubmissions"])[0])
	if record["blobId"] != "blob-1" || record["mimeType"] != "application/pdf" || record["fileName"] != "vlan.pdf" {
		t.Fatalf("unexpected submission: %+v", record)
	}
	if err := SubmitStudentTask(doc, "siswa-1", "", "Tanpa berkas", "x.pdf", "", "application/pdf", 12, time.Now()); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid without blob, got %v", err)
	}
}

func TestAssignmentClassAndDueDateAreEnforced(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	doc := map[string]any{
		"students": []any{map[string]any{
			"id": "siswa-1", "fullName": "Siswa Uji", "classroomName": "XII TKJ 1",
		}},
	}
	if err := CreateAssignment(doc, "Guru Uji", "XII TKJ 1", "Jaringan", "Tugas VLAN", "Instruksi", "2026-12-01", now); err != nil {
		t.Fatalf("create: %v", err)
	}
	assignmentID := asMap(asList(doc["assignments"])[0])["id"].(string)
	if err := SubmitStudentTask(doc, "siswa-1", assignmentID, "", "vlan.pdf", "blob-1", "application/pdf", 12, now); err != nil {
		t.Fatalf("linked submit: %v", err)
	}
	record := asMap(asList(doc["taskSubmissions"])[0])
	if record["assignmentId"] != assignmentID || record["taskTitle"] != "Tugas VLAN" {
		t.Fatalf("unexpected linked submission: %+v", record)
	}

	if err := CreateAssignment(doc, "Guru Uji", "XII TKJ 1", "Jaringan", "Tugas Lewat", "", "2026-11-20", now); err != ErrInvalid {
		t.Fatalf("past due date must be rejected, got %v", err)
	}
	if err := SubmitStudentTask(doc, "siswa-1", assignmentID, "Tugas", "x.pdf", "blob-2", "application/pdf", 12, now); err != ErrInvalid {
		t.Fatalf("duplicate submission must be rejected, got %v", err)
	}
	if err := SubmitStudentTask(doc, "siswa-1", "asg-missing", "Tugas", "x.pdf", "blob-3", "application/pdf", 12, now); err != ErrNotFound {
		t.Fatalf("missing assignment must be rejected, got %v", err)
	}
}

func TestGradeTaskSubmission(t *testing.T) {
	doc := map[string]any{
		"taskSubmissions": []any{
			map[string]any{"id": "tsk-1", "studentId": "siswa-1", "status": "TERKIRIM", "score": 0},
		},
	}
	if err := GradeTaskSubmission(doc, "tsk-1", 88, time.Now()); err != nil {
		t.Fatalf("grade: %v", err)
	}
	record := asMap(asList(doc["taskSubmissions"])[0])
	if record["status"] != "DINILAI" || record["score"] != 88 {
		t.Fatalf("unexpected record: %+v", record)
	}
	if record["gradedAt"] == "" {
		t.Fatal("gradedAt should be filled")
	}
}

func TestGradeTaskSubmissionRejectsBadInput(t *testing.T) {
	doc := map[string]any{"taskSubmissions": []any{map[string]any{"id": "tsk-1"}}}
	if err := GradeTaskSubmission(doc, "tsk-1", 101, time.Now()); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	if err := GradeTaskSubmission(doc, "missing", 80, time.Now()); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
