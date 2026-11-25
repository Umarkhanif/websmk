package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"wesmk/internal/auth"
	"wesmk/internal/model"
)

// recordLoginEvent writes to the append-only audit table. Authentication must
// continue even when the audit insert fails, so the error is logged only.
func (s *Server) recordLoginEvent(r *http.Request, userID, identifier, role, method string, success bool, reason string) {
	if s.storeDB == nil {
		return
	}
	event := model.LoginEvent{
		UserID:     userID,
		Identifier: strings.TrimSpace(identifier),
		Role:       role,
		Method:     method,
		Success:    success,
		Reason:     reason,
		IP:         s.clientIP(r),
		UserAgent:  r.UserAgent(),
	}
	if err := s.storeDB.RecordLoginEvent(r.Context(), event); err != nil {
		log.Printf("login audit write failed: %v", err)
	}
}

// authErrorDetails extracts the user and a user-safe reason from a login error.
func authErrorDetails(err error) (userID, reason string) {
	var authErr *auth.Error
	if errors.As(err, &authErr) {
		return authErr.UserID, authErr.Message
	}
	return "", "Terjadi kesalahan pada server."
}
