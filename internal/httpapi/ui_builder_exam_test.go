package httpapi

import (
	"testing"

	"wesmk/internal/domain/exam"
)

func TestExamQuestionViewFromDomain(t *testing.T) {
	question := exam.Question{
		AppID: "q-1", QuestionNumber: 3, QuestionType: "MCQ",
		SubjectName: "Jaringan", KDCode: "KD-3.1",
		TemplateContent: "VLAN {{vlan}} di router",
		DynamicVars:     map[string]any{"vlan": "10"},
		Options:         []exam.Option{{Key: "A", Text: "Trunk"}},
		CorrectKey:      "A",
	}
	view := examQuestionViewFromDomain(question)
	if view.QuestionText != "VLAN 10 di router" {
		t.Fatalf("template not substituted: %q", view.QuestionText)
	}
	if view.ID != "q-1" || view.KdCode != "KD-3.1" || len(view.Options) != 1 || view.CorrectKey != "A" {
		t.Fatalf("unexpected question view: %+v", view)
	}

	safe := examQuestionViewFromDomain(question.WithoutAnswers())
	if safe.CorrectKey != "" {
		t.Fatalf("student view leaked the answer: %+v", safe)
	}
}

func TestProctorViewFromDomain(t *testing.T) {
	view := proctorViewFromDomain(exam.Session{
		AppID: "ps-1", StudentAppID: "std-1", StudentName: "Budi", NISN: "005",
		Status: "LOCKED", Progress: "3/40", WarnCount: 2, LockCount: 1,
		LastViolation: "blur", UnlockToken: "tok", Emergency2h: true,
	})
	if view.ID != "ps-1" || view.Status != "LOCKED" || view.WarnCount != 2 ||
		view.LockCount != 1 || !view.Emergency2h || view.UnlockToken != "tok" {
		t.Fatalf("unexpected proctor view: %+v", view)
	}
}
