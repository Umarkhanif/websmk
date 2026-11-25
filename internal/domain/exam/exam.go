// Package exam owns question-bank and proctor-session reads from the
// normalized tables. Answer keys are stripped before a student view is built.
package exam

import (
	"context"
	"errors"

	"wesmk/internal/authz"
)

var ErrNotFound = errors.New("data ujian tidak ditemukan")

type Option struct {
	Key  string
	Text string
}

type Question struct {
	AppID              string
	ExamAppID          string
	QuestionNumber     int
	QuestionType       string
	SubjectName        string
	KDCode             string
	CompetencyTag      string
	QuestionText       string
	TemplateContent    string
	Options            []Option
	CorrectKey         string
	EssayRubric        string
	DynamicVars        map[string]any
	RenderShadowCanvas bool
}

// WithoutAnswers returns a copy safe for a student kiosk payload.
func (q Question) WithoutAnswers() Question {
	q.CorrectKey = ""
	q.EssayRubric = ""
	return q
}

type Session struct {
	AppID         string
	StudentAppID  string
	StudentName   string
	NISN          string
	Status        string
	Progress      string
	WarnCount     int
	LockCount     int
	LastViolation string
	UnlockToken   string
	Emergency2h   bool
}

type Repository interface {
	ExamQuestions(ctx context.Context) ([]Question, error)
	ExamSessions(ctx context.Context) ([]Session, error)
	ExamSessionForStudent(ctx context.Context, studentAppID string) ([]Session, error)
}

type Viewer struct {
	UserID    string
	Role      string
	StudentID string
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// QuestionsForStudent enforces the student role and strips answers.
func (s *Service) QuestionsForStudent(ctx context.Context, viewer Viewer, studentAppID string) ([]Question, error) {
	if s == nil || s.repo == nil || studentAppID == "" {
		return nil, ErrNotFound
	}
	own := viewer.StudentID != "" && viewer.StudentID == studentAppID
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceExam, authz.Ownership{Own: own}) {
		return nil, authz.ErrDenied
	}
	questions, err := s.repo.ExamQuestions(ctx)
	if err != nil {
		return nil, err
	}
	safe := make([]Question, 0, len(questions))
	for _, question := range questions {
		safe = append(safe, question.WithoutAnswers())
	}
	return safe, nil
}

// QuestionsForTeacher returns the full bank, including answer keys.
func (s *Service) QuestionsForTeacher(ctx context.Context, viewer Viewer) ([]Question, error) {
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceExam, authz.Ownership{}) {
		return nil, authz.ErrDenied
	}
	return s.repo.ExamQuestions(ctx)
}

func (s *Service) SessionsForTeacher(ctx context.Context, viewer Viewer) ([]Session, error) {
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceExam, authz.Ownership{}) {
		return nil, authz.ErrDenied
	}
	return s.repo.ExamSessions(ctx)
}

func (s *Service) SessionForStudent(ctx context.Context, viewer Viewer, studentAppID string) ([]Session, error) {
	if s == nil || s.repo == nil || studentAppID == "" {
		return nil, ErrNotFound
	}
	own := viewer.StudentID != "" && viewer.StudentID == studentAppID
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceExam, authz.Ownership{Own: own}) {
		return nil, authz.ErrDenied
	}
	return s.repo.ExamSessionForStudent(ctx, studentAppID)
}
