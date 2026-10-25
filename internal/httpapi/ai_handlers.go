package httpapi

import (
	"log"
	"net/http"
	"strings"
	"time"
)

var aiTypes = map[string]bool{
	"essay_grading":        true,
	"empathy_parent":       true,
	"lesson_summary":       true,
	"strategic_headmaster": true,
}

const (
	aiRateLimit  = 30
	aiRateWindow = time.Minute
)

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	authValue, ok := requireUser(w, r)
	if !ok {
		return
	}

	var request struct {
		Prompt string `json:"prompt"`
		Type   string `json:"type"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	if request.Type != "" && !aiTypes[request.Type] {
		writeError(w, http.StatusBadRequest, "Tipe analisis AI tidak dikenal.")
		return
	}
	if strings.TrimSpace(request.Prompt) == "" {
		writeError(w, http.StatusBadRequest, "Prompt is required")
		return
	}
	if len(request.Prompt) > 20000 {
		writeError(w, http.StatusRequestEntityTooLarge, "Prompt terlalu panjang (maksimal 20.000 karakter).")
		return
	}

	count, err := s.cache.IncrementCounter(r.Context(), "ratelimit:ai:"+authValue.user.ID, aiRateWindow)
	if err != nil {
		log.Printf("ai rate limit failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}
	if count > aiRateLimit {
		writeError(w, http.StatusTooManyRequests, "Batas permintaan AI tercapai. Coba lagi sebentar lagi.")
		return
	}

	if !s.aiClient.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "GEMINI_API_KEY is not configured; using client fallback")
		return
	}

	result, err := s.aiClient.Analyze(r.Context(), request.Type, request.Prompt)
	if err != nil {
		log.Printf("gemini error (type=%q): %v", request.Type, err)
		writeError(w, http.StatusBadGateway, "Layanan AI sedang tidak tersedia. Coba lagi nanti.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": result})
}
