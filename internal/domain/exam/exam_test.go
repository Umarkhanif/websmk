package exam

import (
	"context"
	"errors"
	"testing"

	"wesmk/internal/authz"
)

type fakeRepo struct {
	questions []Question
	sessions  []Session
}

func (f fakeRepo) ExamQuestions(context.Context) ([]Question, error) { return f.questions, nil }
func (f fakeRepo) ExamSessions(context.Context) ([]Session, error)   { return f.sessions, nil }
func (f fakeRepo) ExamSessionForStudent(context.Context, string) ([]Session, error) {
	return f.sessions, nil
}

func TestStudentQuestionsStripAnswers(t *testing.T) {
	repo := fakeRepo{questions: []Question{{
		AppID: "q-1", QuestionText: "Soal", CorrectKey: "A", EssayRubric: "rahasia",
	}}}
	service := NewService(repo)

	questions, err := service.QuestionsForStudent(context.Background(),
		Viewer{Role: "student", StudentID: "std-1"}, "std-1")
	if err != nil {
		t.Fatalf("student questions: %v", err)
	}
	if questions[0].CorrectKey != "" || questions[0].EssayRubric != "" {
		t.Fatalf("student payload leaks answers: %+v", questions[0])
	}
	// The stored question must not be mutated for the teacher path.
	if repo.questions[0].CorrectKey != "A" {
		t.Fatal("WithoutAnswers mutated the source question")
	}

	if _, err := service.QuestionsForStudent(context.Background(),
		Viewer{Role: "student", StudentID: "std-2"}, "std-1"); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("foreign student err = %v, want denied", err)
	}
	if _, err := service.QuestionsForStudent(context.Background(),
		Viewer{Role: "parent", StudentID: "std-1"}, "std-1"); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("parent questions err = %v, want denied", err)
	}
}

func TestTeacherQuestionsKeepAnswers(t *testing.T) {
	repo := fakeRepo{questions: []Question{{AppID: "q-1", CorrectKey: "A", EssayRubric: "rubrik"}}}
	service := NewService(repo)

	questions, err := service.QuestionsForTeacher(context.Background(), Viewer{Role: "teacher"})
	if err != nil || questions[0].CorrectKey != "A" || questions[0].EssayRubric != "rubrik" {
		t.Fatalf("teacher questions = %+v, err %v", questions, err)
	}
	if _, err := service.QuestionsForTeacher(context.Background(), Viewer{Role: "student"}); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("student teacher-bank err = %v, want denied", err)
	}
}

func TestExamSessionPolicy(t *testing.T) {
	service := NewService(fakeRepo{sessions: []Session{{AppID: "ps-1"}}})

	if _, err := service.SessionsForTeacher(context.Background(), Viewer{Role: "teacher"}); err != nil {
		t.Fatalf("teacher sessions: %v", err)
	}
	if _, err := service.SessionForStudent(context.Background(),
		Viewer{Role: "student", StudentID: "std-1"}, "std-1"); err != nil {
		t.Fatalf("student own session: %v", err)
	}
	if _, err := service.SessionForStudent(context.Background(),
		Viewer{Role: "student", StudentID: "std-2"}, "std-1"); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("foreign session err = %v, want denied", err)
	}
}
