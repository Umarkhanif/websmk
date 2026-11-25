package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"wesmk/internal/db"
)

func TestExamDomainCutover(t *testing.T) {
	dsn := os.Getenv("WESMK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("WESMK_TEST_DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	suffix := time.Now().UnixNano()
	userID := fmt.Sprintf("itest-exam-user-%d", suffix)
	studentAppID := fmt.Sprintf("itest-exam-std-%d", suffix)
	examAppID := fmt.Sprintf("itest-exam-%d", suffix)
	nisn := fmt.Sprintf("%010d", suffix%10000000000)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM exam_questions WHERE app_id LIKE $1`, "itest-q-"+fmt.Sprint(suffix)+"%")
		_, _ = pool.Exec(context.Background(), `DELETE FROM exams WHERE app_id = $1`, examAppID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, identifier, full_name, role, password_hash)
		VALUES ($1, $1, 'Siswa Ujian', 'student', 'x')
	`, userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	doc := map[string]any{
		"students": []any{map[string]any{
			"id": studentAppID, "userId": userID, "nisn": nisn, "fullName": "Siswa Ujian",
		}},
		"examQuestions": []any{
			map[string]any{
				"id": "itest-q-1-" + fmt.Sprint(suffix), "examId": examAppID,
				"questionNumber": float64(1), "questionType": "MCQ",
				"subjectName": "Jaringan", "kdCode": "KD-3.1",
				"questionText": "Apa itu VLAN?", "correctKey": "A",
				"options": []any{
					map[string]any{"key": "A", "text": "Virtual LAN"},
					map[string]any{"key": "B", "text": "Very Large Array"},
				},
				"dynamicVars": map[string]any{"x": "1"},
			},
			map[string]any{
				"id": "itest-q-2-" + fmt.Sprint(suffix), "examId": examAppID,
				"questionNumber": float64(2), "questionType": "ESSAY",
				"questionText": "Jelaskan trunking.", "essayRubric": "Sebutkan 802.1Q",
			},
		},
		"proctorStudents": []any{map[string]any{
			"id": "itest-ps-" + fmt.Sprint(suffix), "studentId": studentAppID,
			"name": "Siswa Ujian", "nisn": nisn, "progress": "1/2",
			"status": "ACTIVE", "warnCount": float64(1), "lockCount": float64(0),
			"lastViolation": "", "unlockToken": "tok", "hasEmergency2hTicket": true,
		}},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	service := NewPostgres(pool)
	if err := service.PutState(ctx, raw); err != nil {
		t.Fatalf("put state: %v", err)
	}
	if err := service.BackfillDomainData(ctx); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	questions, err := service.ExamQuestions(ctx)
	if err != nil {
		t.Fatalf("questions: %v", err)
	}
	if len(questions) != 2 {
		t.Fatalf("questions = %d, want 2", len(questions))
	}
	if questions[0].CorrectKey != "A" || len(questions[0].Options) != 2 ||
		questions[0].ExamAppID != examAppID || questions[0].DynamicVars["x"] != "1" {
		t.Fatalf("unexpected question: %+v", questions[0])
	}
	if questions[1].QuestionType != "ESSAY" || questions[1].EssayRubric == "" {
		t.Fatalf("unexpected essay question: %+v", questions[1])
	}

	sessions, err := service.ExamSessionForStudent(ctx, studentAppID)
	if err != nil {
		t.Fatalf("sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Status != "ACTIVE" || sessions[0].WarnCount != 1 ||
		sessions[0].Progress != "1/2" || !sessions[0].Emergency2h {
		t.Fatalf("unexpected session: %+v", sessions)
	}

	// Shadow sync updates the session and the bank.
	doc["proctorStudents"].([]any)[0].(map[string]any)["status"] = "SUBMITTED"
	doc["proctorStudents"].([]any)[0].(map[string]any)["progress"] = "2/2"
	doc["examQuestions"].([]any)[0].(map[string]any)["questionText"] = "Apa itu VLAN trunk?"
	if err := service.SyncDomainFromState(ctx, doc); err != nil {
		t.Fatalf("sync: %v", err)
	}
	sessions, err = service.ExamSessionForStudent(ctx, studentAppID)
	if err != nil || len(sessions) != 1 || sessions[0].Status != "SUBMITTED" || sessions[0].Progress != "2/2" {
		t.Fatalf("session after sync = %+v, err %v", sessions, err)
	}
	questions, err = service.ExamQuestions(ctx)
	if err != nil {
		t.Fatalf("questions after sync: %v", err)
	}
	found := false
	for _, question := range questions {
		if question.AppID == "itest-q-1-"+fmt.Sprint(suffix) && question.QuestionText == "Apa itu VLAN trunk?" {
			found = true
		}
	}
	if !found {
		t.Fatal("question text was not updated")
	}

	// Boot backfill is idempotent.
	if err := service.BackfillDomainData(ctx); err != nil {
		t.Fatalf("re-backfill: %v", err)
	}
	sessions, err = service.ExamSessionForStudent(ctx, studentAppID)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("session after re-backfill = %d, err %v", len(sessions), err)
	}
}
