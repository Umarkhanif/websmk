package httpapi

import (
	"net/http"
	"time"

	"wesmk/internal/state"
)

// Student-scoped writes (quiz, exam, payment) share one rate limit so a single
// account cannot hammer the state document.
const maxDomainWritesPerMinute = 30

func studentScopedID(w http.ResponseWriter, r *http.Request, roles ...string) (string, bool) {
	authValue, ok := requireRole(w, r, roles...)
	if !ok {
		return "", false
	}
	if authValue.user.StudentProfileID == "" {
		writeError(w, http.StatusForbidden, "Akun belum terhubung ke profil siswa.")
		return "", false
	}
	return authValue.user.StudentProfileID, true
}

func (s *Server) allowDomainWrite(w http.ResponseWriter, r *http.Request, scope string) bool {
	authValue, ok := requireUser(w, r)
	if !ok {
		return false
	}
	count, err := s.cache.IncrementCounter(r.Context(), "ratelimit:"+scope+":"+authValue.user.ID, time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return false
	}
	if count > maxDomainWritesPerMinute {
		writeError(w, http.StatusTooManyRequests, "Terlalu banyak permintaan. Tunggu sebentar lalu coba lagi.")
		return false
	}
	return true
}

type quizSubmitPayload struct {
	Answers []state.QuizAnswer `json:"answers"`
}

// handleSubmitQuiz scores the daily quiz from the server copy of the answer key
// and records one immutable attempt per student.
func (s *Server) handleSubmitQuiz(w http.ResponseWriter, r *http.Request) {
	studentID, ok := studentScopedID(w, r, "student")
	if !ok {
		return
	}

	var payload quizSubmitPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if len(payload.Answers) == 0 || len(payload.Answers) > 50 {
		writeError(w, http.StatusBadRequest, "Jawaban kuis tidak valid.")
		return
	}
	if !s.allowDomainWrite(w, r, "quiz") {
		return
	}

	now := time.Now()
	s.updateState(w, r, func(doc map[string]any) (any, error) {
		return state.SubmitQuiz(doc, studentID, payload.Answers, now)
	})
}

type examSubmitPayload struct {
	Answers []state.ExamAnswer `json:"answers"`
}

// handleSubmitExam archives the raw CBT answers and grades multiple choice on
// the server. Essay answers wait for the teacher's rubric review.
func (s *Server) handleSubmitExam(w http.ResponseWriter, r *http.Request) {
	studentID, ok := studentScopedID(w, r, "student")
	if !ok {
		return
	}

	var payload examSubmitPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if len(payload.Answers) == 0 || len(payload.Answers) > 200 {
		writeError(w, http.StatusBadRequest, "Jawaban ujian tidak valid.")
		return
	}
	for _, answer := range payload.Answers {
		if answer.QuestionNumber < 1 || answer.QuestionNumber > 500 {
			writeError(w, http.StatusBadRequest, "Nomor soal tidak valid.")
			return
		}
	}
	if !s.allowDomainWrite(w, r, "exam") {
		return
	}

	now := time.Now()
	s.updateState(w, r, func(doc map[string]any) (any, error) {
		return state.SubmitExam(doc, studentID, payload.Answers, now)
	})
}

// handleStartExam records a server-side CBT deadline (90 minutes) for the
// student. Repeated calls keep the first deadline.
func (s *Server) handleStartExam(w http.ResponseWriter, r *http.Request) {
	studentID, ok := studentScopedID(w, r, "student")
	if !ok {
		return
	}
	if !s.allowDomainWrite(w, r, "exam") {
		return
	}
	var deadline time.Time
	err := s.mutateStateDoc(r.Context(), func(doc map[string]any) error {
		value, err := state.StartExam(doc, studentID, time.Now(), 90*time.Minute)
		if err != nil {
			return err
		}
		deadline = value
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "Gagal memulai ujian.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deadline": deadline.UTC().Format(time.RFC3339)})
}
