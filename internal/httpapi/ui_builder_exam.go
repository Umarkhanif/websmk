package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"wesmk/internal/authz"
	"wesmk/internal/domain/exam"
	"wesmk/internal/model"
	"wesmk/internal/ui"
)

func domainExamViewer(user *model.User) exam.Viewer {
	if user == nil {
		return exam.Viewer{}
	}
	return exam.Viewer{UserID: user.ID, Role: user.Role, StudentID: user.StudentProfileID}
}

func examQuestionViewFromDomain(question exam.Question) ui.ExamQuestionView {
	text := question.QuestionText
	if text == "" {
		text = question.TemplateContent
		for key, value := range question.DynamicVars {
			text = strings.ReplaceAll(text, "{{"+key+"}}", fmt.Sprintf("%v", value))
		}
	}
	tag := question.KDCode
	if tag == "" {
		tag = question.CompetencyTag
	}
	options := make([]ui.QuizOptionView, 0, len(question.Options))
	for _, option := range question.Options {
		options = append(options, ui.QuizOptionView{Key: option.Key, Text: option.Text})
	}
	return ui.ExamQuestionView{
		ID:             question.AppID,
		QuestionNumber: question.QuestionNumber,
		QuestionType:   question.QuestionType,
		SubjectName:    question.SubjectName,
		KdCode:         tag,
		QuestionText:   text,
		Options:        options,
		CorrectKey:     question.CorrectKey,
		EssayRubric:    question.EssayRubric,
	}
}

func examQuestionViewsFromDomain(questions []exam.Question) []ui.ExamQuestionView {
	views := make([]ui.ExamQuestionView, 0, len(questions))
	for _, question := range questions {
		views = append(views, examQuestionViewFromDomain(question))
	}
	return views
}

func proctorViewFromDomain(session exam.Session) ui.ProctorSessionView {
	return ui.ProctorSessionView{
		ID:            session.AppID,
		StudentID:     session.StudentAppID,
		Name:          session.StudentName,
		NISN:          session.NISN,
		Progress:      session.Progress,
		Status:        session.Status,
		WarnCount:     session.WarnCount,
		LockCount:     session.LockCount,
		LastViolation: session.LastViolation,
		UnlockToken:   session.UnlockToken,
		Emergency2h:   session.Emergency2h,
	}
}

func proctorViewsFromDomain(sessions []exam.Session) []ui.ProctorSessionView {
	views := make([]ui.ProctorSessionView, 0, len(sessions))
	for _, session := range sessions {
		views = append(views, proctorViewFromDomain(session))
	}
	return views
}

func (s *Server) domainStudentExamQuestions(ctx context.Context, user *model.User, studentID string) ([]exam.Question, bool) {
	if !s.cfg.DomainExam || s.examSvc == nil || user == nil || studentID == "" {
		return nil, false
	}
	questions, err := s.examSvc.QuestionsForStudent(ctx, domainExamViewer(user), studentID)
	if err != nil {
		logExamFallback(err)
		return nil, false
	}
	if len(questions) == 0 {
		return nil, false
	}
	return questions, true
}

func (s *Server) domainTeacherExamQuestions(ctx context.Context, user *model.User) ([]exam.Question, bool) {
	if !s.cfg.DomainExam || s.examSvc == nil || user == nil {
		return nil, false
	}
	questions, err := s.examSvc.QuestionsForTeacher(ctx, domainExamViewer(user))
	if err != nil {
		logExamFallback(err)
		return nil, false
	}
	if len(questions) == 0 {
		return nil, false
	}
	return questions, true
}

func (s *Server) domainStudentExamSession(ctx context.Context, user *model.User, studentID string) ([]exam.Session, bool) {
	if !s.cfg.DomainExam || s.examSvc == nil || user == nil || studentID == "" {
		return nil, false
	}
	sessions, err := s.examSvc.SessionForStudent(ctx, domainExamViewer(user), studentID)
	if err != nil {
		logExamFallback(err)
		return nil, false
	}
	if len(sessions) == 0 {
		return nil, false
	}
	return sessions, true
}

func (s *Server) domainTeacherExamSessions(ctx context.Context, user *model.User) ([]exam.Session, bool) {
	if !s.cfg.DomainExam || s.examSvc == nil || user == nil {
		return nil, false
	}
	sessions, err := s.examSvc.SessionsForTeacher(ctx, domainExamViewer(user))
	if err != nil {
		logExamFallback(err)
		return nil, false
	}
	if len(sessions) == 0 {
		return nil, false
	}
	return sessions, true
}

func logExamFallback(err error) {
	if err == nil || errors.Is(err, authz.ErrDenied) || errors.Is(err, exam.ErrNotFound) {
		return
	}
	log.Printf("domain exam read failed: %v", err)
}
