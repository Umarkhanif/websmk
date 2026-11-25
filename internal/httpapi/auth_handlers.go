package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"wesmk/internal/auth"
	"wesmk/internal/model"
)

func (s *Server) handleAuthConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Identifier) == "" || request.Password == "" {
		writeError(w, http.StatusBadRequest, "NISN/NIP dan kata sandi wajib diisi.")
		return
	}

	session, err := s.authSvc.Login(r.Context(), request.Identifier, request.Password, s.clientIP(r))
	if err != nil {
		var authErr *auth.Error
		if errors.As(err, &authErr) && authErr.Code == auth.CodeMFARequired {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error":     authErr.Message,
				"code":      auth.CodeMFARequired,
				"challenge": authErr.Challenge,
			})
			return
		}
		userID, reason := authErrorDetails(err)
		s.recordLoginEvent(r, userID, request.Identifier, "", "password", false, reason)
		writeAuthError(w, err)
		return
	}

	s.setSessionCookie(w, session.Token)
	s.rememberSession(r, session.Token, session.User.ID)
	s.recordLoginEvent(r, session.User.ID, request.Identifier, session.User.Role, session.Method, true, "")
	writeJSON(w, http.StatusOK, map[string]any{
		"user":             session.User,
		"mfaSetupRequired": auth.RequiresMFA(session.User) && !session.User.TOTPEnabled,
	})
}

func (s *Server) handleMFA(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Challenge string `json:"challenge"`
		Code      string `json:"code"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	session, err := s.authSvc.VerifyMFA(r.Context(), request.Challenge, request.Code)
	if err != nil {
		userID, reason := authErrorDetails(err)
		s.recordLoginEvent(r, userID, "", "", "mfa", false, reason)
		writeAuthError(w, err)
		return
	}
	s.setSessionCookie(w, session.Token)
	s.rememberSession(r, session.Token, session.User.ID)
	s.recordLoginEvent(r, session.User.ID, session.User.Username, session.User.Role, session.Method, true, "")
	writeJSON(w, http.StatusOK, map[string]any{"user": session.User})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	authValue, ok := requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":             authValue.user,
		"mfaSetupRequired": auth.RequiresMFA(authValue.user) && !authValue.user.TOTPEnabled,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := ""
	if authValue, ok := currentAuth(r); ok {
		token = authValue.token
	}
	if token == "" {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			token = cookie.Value
		}
	}

	if err := s.authSvc.Logout(r.Context(), token); err != nil {
		log.Printf("logout failed: %v", err)
	}
	s.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (s *Server) handleMagicLinkRequest(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Phone string `json:"phone"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	normalized := model.NormalizePhone(request.Phone)
	if len(normalized) < 8 {
		writeError(w, http.StatusBadRequest, "Nomor WhatsApp tidak valid.")
		return
	}

	token, issued, err := s.authSvc.RequestMagicLink(r.Context(), request.Phone, s.clientIP(r))
	if err != nil {
		var authErr *auth.Error
		if errors.As(err, &authErr) {
			writeAuthError(w, err)
			return
		}
		log.Printf("magic link failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}

	if issued {
		suffix := normalized
		if len(suffix) > 4 {
			suffix = suffix[len(suffix)-4:]
		}
		log.Printf("[auth] Tautan magic dibuat untuk nomor berakhiran %s.", suffix)
		if err := s.sendMagicLink(r.Context(), normalized, token); err != nil {
			log.Printf("[auth] kirim tautan magic gagal: %v", err)
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		log.Printf("[auth] Pengiriman WhatsApp belum dikonfigurasi; tautan hanya tersedia di log server.")
	}

	// Same response whether or not the number is registered (no enumeration).
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
}

func (s *Server) handleMagicLinkConsume(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	session, err := s.authSvc.ConsumeMagicToken(r.Context(), token)
	if err != nil {
		log.Printf("consume magic link failed: %v", err)
	}
	if session == nil {
		http.Redirect(w, r, "/?auth=magic_invalid", http.StatusFound)
		return
	}

	s.setSessionCookie(w, session.Token)
	http.Redirect(w, r, "/", http.StatusFound)
}
