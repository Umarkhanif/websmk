package state

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

var (
	ErrNotFound    = errors.New("data tidak ditemukan")
	ErrConflict    = errors.New("konflik data")
	ErrUnavailable = errors.New("data belum tersedia")
	ErrInvalid     = errors.New("data tidak valid")
	ErrForbidden   = errors.New("akses ditolak")
)

type QuizAnswer struct {
	QuestionID string `json:"questionId"`
	OptionKey  string `json:"optionKey"`
}

type QuizResult struct {
	QuizID           string `json:"quizId"`
	Score            int    `json:"score"`
	CorrectCount     int    `json:"correctCount"`
	QuestionCount    int    `json:"questionCount"`
	SubmittedAt      string `json:"submittedAt"`
	AlreadySubmitted bool   `json:"alreadySubmitted,omitempty"`
}

// SubmitQuiz scores answers against the server-held quiz and records one
// attempt per student per quiz. Replays return the first score.
func SubmitQuiz(doc map[string]any, studentID string, answers []QuizAnswer, now time.Time) (QuizResult, error) {
	student := findStudent(doc, studentID)
	if student == nil {
		return QuizResult{}, fmt.Errorf("%w: profil siswa", ErrNotFound)
	}
	quiz := asMap(doc["dailyQuiz"])
	questions := asList(quiz["questions"])
	if quiz == nil || len(questions) == 0 {
		return QuizResult{}, fmt.Errorf("%w: kuis harian", ErrUnavailable)
	}
	if len(answers) == 0 {
		return QuizResult{}, fmt.Errorf("%w: jawaban kosong", ErrInvalid)
	}

	quizID := text(quiz["id"])
	for _, item := range asList(doc["quizAttempts"]) {
		record := asMap(item)
		if record != nil && text(record["studentId"]) == studentID && text(record["quizId"]) == quizID {
			return quizResultFrom(record, true), nil
		}
	}

	answerByQuestion := map[string]string{}
	auditAnswers := []any{}
	for _, answer := range answers {
		questionID := strings.TrimSpace(answer.QuestionID)
		option := strings.TrimSpace(answer.OptionKey)
		if questionID == "" || option == "" {
			continue
		}
		answerByQuestion[questionID] = option
		auditAnswers = append(auditAnswers, map[string]any{
			"questionId": questionID,
			"optionKey":  option,
		})
	}
	if len(answerByQuestion) == 0 {
		return QuizResult{}, fmt.Errorf("%w: jawaban kosong", ErrInvalid)
	}

	correct := 0
	for _, item := range questions {
		question := asMap(item)
		if question == nil {
			continue
		}
		if answerByQuestion[text(question["id"])] == text(question["correctKey"]) {
			correct++
		}
	}

	total := len(questions)
	score := int(math.Round(float64(correct) / float64(total) * 100))
	submittedAt := now.UTC().Format(time.RFC3339)
	student["dailyQuizAvg"] = float64(score)

	record := map[string]any{
		"id":            fmt.Sprintf("quiz-%d", now.UnixNano()),
		"quizId":        quizID,
		"studentId":     studentID,
		"studentName":   text(student["fullName"]),
		"score":         score,
		"correctCount":  correct,
		"questionCount": total,
		"answers":       auditAnswers,
		"submittedAt":   submittedAt,
	}
	doc["quizAttempts"] = append(asList(doc["quizAttempts"]), record)

	return QuizResult{
		QuizID:        quizID,
		Score:         score,
		CorrectCount:  correct,
		QuestionCount: total,
		SubmittedAt:   submittedAt,
	}, nil
}

func quizResultFrom(record map[string]any, already bool) QuizResult {
	score, _ := number(record["score"])
	correct, _ := number(record["correctCount"])
	total, _ := number(record["questionCount"])
	return QuizResult{
		QuizID:           text(record["quizId"]),
		Score:            int(score),
		CorrectCount:     int(correct),
		QuestionCount:    int(total),
		SubmittedAt:      text(record["submittedAt"]),
		AlreadySubmitted: already,
	}
}

type ExamAnswer struct {
	QuestionNumber int    `json:"questionNumber"`
	Answer         string `json:"answer"`
}

type ExamResult struct {
	ExamID           string `json:"examId"`
	McqCorrect       int    `json:"mcqCorrect"`
	McqTotal         int    `json:"mcqTotal"`
	McqScore         int    `json:"mcqScore"`
	EssayCount       int    `json:"essayCount"`
	SubmittedAt      string `json:"submittedAt"`
	AlreadySubmitted bool   `json:"alreadySubmitted,omitempty"`
}

// SubmitExam archives the raw answers and scores multiple-choice questions on
// the server. Essay answers are stored for the teacher's rubric review.
func SubmitExam(doc map[string]any, studentID string, answers []ExamAnswer, now time.Time) (ExamResult, error) {
	return SubmitExamForSubject(doc, studentID, "", answers, now)
}

// SubmitExamForSubject archives answers for a specific subject/exam schedule ID.
func SubmitExamForSubject(doc map[string]any, studentID, subjectID string, answers []ExamAnswer, now time.Time) (ExamResult, error) {
	student := findStudent(doc, studentID)
	if student == nil {
		return ExamResult{}, fmt.Errorf("%w: profil siswa", ErrNotFound)
	}
	if deadline, ok := ExamDeadline(doc, studentID); ok && now.After(deadline) {
		return ExamResult{}, fmt.Errorf("%w: waktu ujian habis", ErrInvalid)
	}
	for _, item := range asList(doc["proctorStudents"]) {
		ps := asMap(item)
		if ps != nil && text(ps["studentId"]) == studentID {
			if text(ps["status"]) == "LOCKED" {
				return ExamResult{}, fmt.Errorf("%w: sesi ujian sedang terkunci oleh sistem anti-curang", ErrForbidden)
			}
			break
		}
	}
	questions := asList(doc["examQuestions"])
	if len(questions) == 0 {
		return ExamResult{}, fmt.Errorf("%w: bank soal ujian", ErrUnavailable)
	}
	if len(answers) == 0 {
		return ExamResult{}, fmt.Errorf("%w: jawaban kosong", ErrInvalid)
	}

	first := asMap(questions[0])
	examID := text(first["examId"])
	if examID == "" {
		examID = "exam-active"
	}
	subjectID = strings.TrimSpace(subjectID)
	if subjectID != "" && subjectID != "exam-sch-01" && subjectID != "exam-cbt-tkj-01" {
		examID = subjectID
	}
	for _, item := range asList(doc["examSubmissions"]) {
		record := asMap(item)
		if record != nil && text(record["studentId"]) == studentID {
			recExamID := text(record["examId"])
			if recExamID == examID || (subjectID == "exam-sch-01" && (recExamID == "exam-cbt-tkj-01" || recExamID == "exam-sch-01")) {
				return examResultFrom(record, true), nil
			}
		}
	}

	sort.Slice(answers, func(i, j int) bool { return answers[i].QuestionNumber < answers[j].QuestionNumber })

	mcqCorrect := 0
	mcqTotal := 0
	essayCount := 0
	audit := []any{}
	seenNumbers := map[int]bool{}
	for _, answer := range answers {
		textAnswer := strings.TrimSpace(answer.Answer)
		if answer.QuestionNumber < 1 || textAnswer == "" || seenNumbers[answer.QuestionNumber] {
			continue
		}
		question := examQuestionFor(questions, answer.QuestionNumber)
		if question == nil {
			continue
		}
		// One answer per number: duplicates must not inflate the score.
		seenNumbers[answer.QuestionNumber] = true
		questionType := text(question["questionType"])
		correct := false
		if questionType == "ESSAY" {
			essayCount++
		} else {
			mcqTotal++
			correct = strings.EqualFold(textAnswer, text(question["correctKey"]))
			if correct {
				mcqCorrect++
			}
		}
		audit = append(audit, map[string]any{
			"questionNumber": answer.QuestionNumber,
			"questionId":     text(question["id"]),
			"questionType":   questionType,
			"answer":         textAnswer,
			"correct":        correct,
		})
	}
	if len(audit) == 0 {
		return ExamResult{}, fmt.Errorf("%w: jawaban kosong", ErrInvalid)
	}

	mcqScore := 0
	if mcqTotal > 0 {
		mcqScore = int(math.Round(float64(mcqCorrect) / float64(mcqTotal) * 100))
		// The CBT component of the holistic report is a server result, not a
		// number the kiosk can post about itself.
		student["cbtScore"] = float64(mcqScore)
	}
	submittedAt := now.UTC().Format(time.RFC3339)

	totalQuestions := len(questions)
	if totalQuestions < 40 {
		totalQuestions = 40
	}
	progress := fmt.Sprintf("%d/%d", len(audit), totalQuestions)
	markProctorSubmitted(doc, student, progress, now)

	doc["examSubmissions"] = append(asList(doc["examSubmissions"]), map[string]any{
		"id":          fmt.Sprintf("exam-%d", now.UnixNano()),
		"examId":      examID,
		"studentId":   studentID,
		"studentName": text(student["fullName"]),
		"answers":     audit,
		"mcqCorrect":  mcqCorrect,
		"mcqTotal":    mcqTotal,
		"mcqScore":    mcqScore,
		"essayCount":  essayCount,
		"submittedAt": submittedAt,
	})

	return ExamResult{
		ExamID:      examID,
		McqCorrect:  mcqCorrect,
		McqTotal:    mcqTotal,
		McqScore:    mcqScore,
		EssayCount:  essayCount,
		SubmittedAt: submittedAt,
	}, nil
}

func examResultFrom(record map[string]any, already bool) ExamResult {
	correct, _ := number(record["mcqCorrect"])
	total, _ := number(record["mcqTotal"])
	score, _ := number(record["mcqScore"])
	essay, _ := number(record["essayCount"])
	return ExamResult{
		ExamID:           text(record["examId"]),
		McqCorrect:       int(correct),
		McqTotal:         int(total),
		McqScore:         int(score),
		EssayCount:       int(essay),
		SubmittedAt:      text(record["submittedAt"]),
		AlreadySubmitted: already,
	}
}

// examQuestionFor mirrors the kiosk's question-number mapping: exact number
// first, then the bank repeats to fill the 40-question session.
func examQuestionFor(questions []any, number int) map[string]any {
	for _, item := range questions {
		question := asMap(item)
		if question != nil && intValue(question["questionNumber"]) == number {
			return question
		}
	}
	return asMap(questions[(number-1)%len(questions)])
}

func markProctorSubmitted(doc map[string]any, student map[string]any, progress string, now time.Time) {
	studentID := text(student["id"])
	for _, item := range asList(doc["proctorStudents"]) {
		session := asMap(item)
		if session != nil && text(session["studentId"]) == studentID {
			session["status"] = "SUBMITTED"
			session["progress"] = progress
			return
		}
	}
	doc["proctorStudents"] = append(asList(doc["proctorStudents"]), map[string]any{
		"id":            fmt.Sprintf("ps-%d", now.UnixNano()),
		"studentId":     studentID,
		"name":          text(student["fullName"]),
		"nisn":          text(student["nisn"]),
		"progress":      progress,
		"status":        "SUBMITTED",
		"warnCount":     0,
		"lockCount":     0,
		"lastViolation": "",
	})
}

func intValue(value any) int {
	if parsed, ok := number(value); ok {
		return int(parsed)
	}
	return 0
}

// StartExam records a server-side CBT deadline for a student. Repeated calls
// keep the first deadline so a refresh cannot extend the timer.
func StartExam(doc map[string]any, studentID string, now time.Time, duration time.Duration) (time.Time, error) {
	student := findStudent(doc, studentID)
	if student == nil {
		return time.Time{}, fmt.Errorf("%w: profil siswa", ErrNotFound)
	}
	if existing := text(student["examDeadlineAt"]); existing != "" {
		if parsed, err := time.Parse(time.RFC3339, existing); err == nil {
			return parsed, nil
		}
	}
	deadline := now.Add(duration).UTC()
	student["examStartedAt"] = now.UTC().Format(time.RFC3339)
	student["examDeadlineAt"] = deadline.Format(time.RFC3339)
	return deadline, nil
}

// ExamDeadline returns the server-side CBT deadline, if any.
func ExamDeadline(doc map[string]any, studentID string) (time.Time, bool) {
	student := findStudent(doc, studentID)
	if student == nil {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, text(student["examDeadlineAt"]))
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}
