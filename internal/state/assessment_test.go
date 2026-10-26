package state

import (
	"errors"
	"testing"
	"time"
)

const assessmentState = `{
  "students": [{"id": "std-1", "fullName": "Budi", "nisn": "0058471290"}],
  "dailyQuiz": {
    "id": "quiz-1",
    "questions": [
      {"id": "q1", "correctKey": "A"},
      {"id": "q2", "correctKey": "B"},
      {"id": "q3", "correctKey": "C"}
    ]
  },
  "quizAttempts": [],
  "examQuestions": [
    {"id": "e1", "examId": "exam-1", "questionNumber": 1, "questionType": "MCQ", "correctKey": "A"},
    {"id": "e2", "examId": "exam-1", "questionNumber": 2, "questionType": "MCQ", "correctKey": "B"},
    {"id": "e3", "examId": "exam-1", "questionNumber": 3, "questionType": "ESSAY"}
  ],
  "proctorStudents": [{"id": "ps-1", "studentId": "std-1", "status": "ACTIVE", "progress": "0/40"}],
  "examSubmissions": []
}`

func TestSubmitQuizScoresOnServerAndReplaysFirstAttempt(t *testing.T) {
	doc := mustDoc(t, assessmentState)
	now := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)

	result, err := SubmitQuiz(doc, "std-1", []QuizAnswer{
		{QuestionID: "q1", OptionKey: "A"},
		{QuestionID: "q2", OptionKey: "A"},
		{QuestionID: "q3", OptionKey: "C"},
	}, now)
	if err != nil {
		t.Fatalf("submit quiz: %v", err)
	}
	if result.Score != 67 || result.CorrectCount != 2 || result.QuestionCount != 3 {
		t.Fatalf("result = %+v, want 2/3 = 67", result)
	}
	if got, _ := number(findStudent(doc, "std-1")["dailyQuizAvg"]); got != 67 {
		t.Fatalf("stored dailyQuizAvg = %v, want 67", got)
	}

	replay, err := SubmitQuiz(doc, "std-1", []QuizAnswer{
		{QuestionID: "q1", OptionKey: "A"},
		{QuestionID: "q2", OptionKey: "B"},
		{QuestionID: "q3", OptionKey: "C"},
	}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("replay quiz: %v", err)
	}
	if !replay.AlreadySubmitted || replay.Score != 67 {
		t.Fatalf("replay = %+v, want the first 67 score", replay)
	}
	if got := len(asList(doc["quizAttempts"])); got != 1 {
		t.Fatalf("attempts = %d, want 1", got)
	}
}

func TestSubmitQuizRequiresAnswersAndBank(t *testing.T) {
	doc := mustDoc(t, assessmentState)
	if _, err := SubmitQuiz(doc, "std-1", nil, time.Now()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty answers error = %v, want ErrInvalid", err)
	}
	if _, err := SubmitQuiz(doc, "std-missing", []QuizAnswer{{QuestionID: "q1", OptionKey: "A"}}, time.Now()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown student error = %v, want ErrNotFound", err)
	}

	empty := mustDoc(t, `{"students": [{"id": "std-1"}]}`)
	if _, err := SubmitQuiz(empty, "std-1", []QuizAnswer{{QuestionID: "q1", OptionKey: "A"}}, time.Now()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing quiz error = %v, want ErrUnavailable", err)
	}
}

func TestSubmitExamGradesMcqAndQueuesEssay(t *testing.T) {
	doc := mustDoc(t, assessmentState)
	now := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)

	result, err := SubmitExam(doc, "std-1", []ExamAnswer{
		{QuestionNumber: 1, Answer: "a"},
		{QuestionNumber: 2, Answer: "A"},
		{QuestionNumber: 3, Answer: "Cek kabel, lalu DHCP pool, lalu firewall UDP 67."},
	}, now)
	if err != nil {
		t.Fatalf("submit exam: %v", err)
	}
	if result.McqCorrect != 1 || result.McqTotal != 2 || result.McqScore != 50 || result.EssayCount != 1 {
		t.Fatalf("result = %+v, want 1/2 MCQ = 50 and 1 essay queued", result)
	}

	if got, _ := number(findStudent(doc, "std-1")["cbtScore"]); got != 50 {
		t.Fatalf("stored cbtScore = %v, want the graded 50", got)
	}

	proctor := findOwnedRecord(asList(doc["proctorStudents"]), "ps-1", "std-1")
	if status := text(proctor["status"]); status != "SUBMITTED" {
		t.Fatalf("proctor status = %q, want SUBMITTED", status)
	}

	replay, err := SubmitExam(doc, "std-1", []ExamAnswer{
		{QuestionNumber: 1, Answer: "A"},
		{QuestionNumber: 2, Answer: "B"},
	}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("replay exam: %v", err)
	}
	if !replay.AlreadySubmitted || replay.McqScore != 50 {
		t.Fatalf("replay = %+v, want the first 50 score", replay)
	}
}

func TestSubmitExamIgnoresDuplicateAnswers(t *testing.T) {
	doc := mustDoc(t, assessmentState)

	result, err := SubmitExam(doc, "std-1", []ExamAnswer{
		{QuestionNumber: 1, Answer: "A"},
		{QuestionNumber: 1, Answer: "A"},
		{QuestionNumber: 2, Answer: "A"},
	}, time.Now())
	if err != nil {
		t.Fatalf("submit exam: %v", err)
	}
	if result.McqTotal != 2 || result.McqCorrect != 1 || result.McqScore != 50 {
		t.Fatalf("result = %+v, want duplicates collapsed to 1/2 = 50", result)
	}
}

func TestSubmitExamBlockedWhenProctorLocked(t *testing.T) {
	doc := mustDoc(t, `{
		"students": [{"id": "std-1", "fullName": "Budi", "nisn": "0058471290"}],
		"examQuestions": [
			{"id": "e1", "examId": "exam-1", "questionNumber": 1, "questionType": "MCQ", "correctKey": "A"}
		],
		"proctorStudents": [{"id": "ps-1", "studentId": "std-1", "status": "LOCKED", "warnCount": 2}],
		"examSubmissions": []
	}`)

	_, err := SubmitExam(doc, "std-1", []ExamAnswer{
		{QuestionNumber: 1, Answer: "A"},
	}, time.Now())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("locked student error = %v, want ErrForbidden", err)
	}
}
