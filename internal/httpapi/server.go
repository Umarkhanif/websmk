package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"wesmk/internal/ai"
	"wesmk/internal/auth"
	"wesmk/internal/config"
	"wesmk/internal/domain/attendance"
	"wesmk/internal/domain/bkk"
	"wesmk/internal/domain/exam"
	"wesmk/internal/domain/finance"
	"wesmk/internal/domain/identity"
	"wesmk/internal/domain/pkl"
	"wesmk/internal/model"
	"wesmk/internal/store"
)

const sessionCookieName = "wesmk_session"

type requestAuth struct {
	user  *model.User
	token string
}

type authContextKey struct{}

type Server struct {
	cfg      config.Config
	authSvc  *auth.Service
	storeDB  *store.Postgres
	cache    *store.Cache
	aiClient *ai.Client
	// Domain slices that read normalized tables instead of the blob.
	studentProfiles *identity.Service
	financeSvc      *finance.Service
	attendanceSvc   *attendance.Service
	examSvc         *exam.Service
	pklSvc          *pkl.Service
	bkkSvc          *bkk.Service
	mux             *http.ServeMux
}

func New(
	cfg config.Config,
	authSvc *auth.Service,
	storeDB *store.Postgres,
	cache *store.Cache,
	aiClient *ai.Client,
) *Server {
	server := &Server{
		cfg:             cfg,
		authSvc:         authSvc,
		storeDB:         storeDB,
		cache:           cache,
		aiClient:        aiClient,
		studentProfiles: identity.NewService(storeDB),
		financeSvc:      finance.NewService(storeDB),
		attendanceSvc:   attendance.NewService(storeDB),
		examSvc:         exam.NewService(storeDB),
		pklSvc:          pkl.NewService(storeDB),
		bkkSvc:          bkk.NewService(storeDB),
		mux:             http.NewServeMux(),
	}
	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	var handler http.Handler = s.mux
	handler = s.csrfGuard(handler)
	handler = s.enforcePrivilegedMFA(handler)
	handler = s.attachAuth(handler)
	handler = s.mutationGuards(handler)
	handler = s.securityHeaders(handler)
	handler = s.recoverPanic(handler)
	handler = s.accessLog(handler)
	return handler
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	s.mux.HandleFunc("GET /api/auth/config", s.handleAuthConfig)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/auth/mfa", s.handleMFA)
	s.mux.HandleFunc("GET /api/auth/me", s.handleMe)
	s.mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("POST /api/auth/magic-link", s.handleMagicLinkRequest)
	s.mux.HandleFunc("POST /api/auth/magic/{token}", s.handleMagicLinkConsume)

	s.mux.HandleFunc("POST /api/learning/quiz/submit", s.handleSubmitQuiz)
	s.mux.HandleFunc("POST /api/learning/exam/submit", s.handleSubmitExam)
	s.mux.HandleFunc("POST /api/learning/exam/start", s.handleStartExam)

	s.mux.HandleFunc("POST /api/payments/intent", s.handleCreatePaymentIntent)
	s.mux.HandleFunc("POST /api/payments/callback", s.handlePaymentCallback)

	s.mux.HandleFunc("GET /api/state", s.handleGetState)
	s.mux.HandleFunc("PUT /api/state", s.handlePutState)
	s.mux.HandleFunc("POST /api/state/reset", s.handleResetState)

	s.mux.HandleFunc("POST /api/ai/analyze", s.handleAnalyze)

	s.mux.HandleFunc("/api", s.handleAPINotFound)
	s.mux.HandleFunc("/api/", s.handleAPINotFound)
	s.registerUIRoutes()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]string{"postgres": "ok", "redis": "ok"}
	databaseUp := true
	if err := s.storeDB.Ping(ctx); err != nil {
		databaseUp = false
		checks["postgres"] = "down"
	}
	redisUp := true
	if err := s.cache.Ping(ctx); err != nil {
		redisUp = false
		checks["redis"] = "down"
	}

	status := http.StatusOK
	state := "ok"
	if !databaseUp || !redisUp {
		status = http.StatusServiceUnavailable
		state = "degraded"
	}
	// Keep the public probe minimal: model names and persistence details are
	// useful reconnaissance for an unauthenticated caller.
	writeJSON(w, status, map[string]any{
		"status": state,
		"checks": checks,
	})
}

func (s *Server) handleAPINotFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "Endpoint API tidak ditemukan.")
}

// Alpine evaluates x-data expressions with new Function, so it needs
// 'unsafe-eval'. Inline event handlers stay blocked; destructive actions use
// data-confirm and are wired in app.js instead.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self' 'unsafe-eval'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"font-src 'self'; " +
	"img-src 'self' data: blob: https://images.unsplash.com https://r2.wesmk.sch.id; " +
	"media-src 'self' blob: https://r2.wesmk.sch.id; " +
	"connect-src 'self'; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		if s.cfg.Env == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusRecorder) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/assets/") || r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		if recorder.status == 0 {
			recorder.status = http.StatusOK
		}
		log.Printf("http %s %s %d %s %s", r.Method, r.URL.Path, recorder.status,
			time.Since(start).Round(time.Millisecond), s.clientIP(r))
	})
}

// mutationGuards rejects cross-origin state-changing requests. The Origin check
// applies to every unsafe method, not only /api, so browser form POSTs are covered.
func (s *Server) mutationGuards(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if !s.originAllowed(r) {
				writeError(w, http.StatusForbidden, "Asal permintaan tidak diizinkan.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// csrfGuard validates the per-session CSRF token on state-changing web requests.
// API requests are excluded: they are JSON-only and rely on the Origin check plus
// the X-CSRF-Token header sent by the base layout.
func (s *Server) csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ui-assets/") || path == "/favicon.ico" {
			next.ServeHTTP(w, r)
			return
		}

		seed := ""
		if authValue, ok := currentAuth(r); ok {
			seed = authValue.token
		} else {
			switch {
			case strings.HasPrefix(path, "/login"):
				seed = "login"
			case strings.HasPrefix(path, "/verify"):
				seed = "verify"
			case strings.HasPrefix(path, "/contact"):
				seed = "contact"
			}
		}
		if seed == "" {
			// Unknown public endpoint: let the handler decide (usually a redirect).
			next.ServeHTTP(w, r)
			return
		}

		submitted := r.Header.Get("X-CSRF-Token")
		if submitted == "" {
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				_ = r.ParseMultipartForm(1 << 20)
			} else {
				_ = r.ParseForm()
			}
			submitted = r.FormValue("_csrf")
		}
		if !s.validCSRF(seed, submitted) {
			writeError(w, http.StatusForbidden, "Token keamanan tidak valid. Muat ulang halaman lalu coba lagi.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) attachAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/ui-assets/") || r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Demo mode: no login required; attach the synthetic demo user.
		if s.cfg.DemoNoAuth {
			user, err := s.storeDB.UserByID(r.Context(), "usr-demo")
			if err == nil && user != nil {
				ctx := context.WithValue(r.Context(), authContextKey{}, requestAuth{user: user, token: "demo"})
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err == nil && cookie.Value != "" {
			user, err := s.authSvc.UserForToken(r.Context(), cookie.Value)
			if err != nil {
				log.Printf("session lookup failed: %v", err)
			} else if user != nil {
				ctx := context.WithValue(r.Context(), authContextKey{}, requestAuth{user: user, token: cookie.Value})
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// enforcePrivilegedMFA keeps an authenticated privileged account inside the
// enrollment flow until TOTP is active. The session still exists so the user can
// open /security, but every other portal and API route is blocked.
func (s *Server) enforcePrivilegedMFA(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Demo mode skips the MFA gate entirely.
		if s.cfg.DemoNoAuth {
			next.ServeHTTP(w, r)
			return
		}

		authVal, ok := currentAuth(r)
		if !ok || !auth.RequiresMFA(authVal.user) || authVal.user.TOTPEnabled {
			next.ServeHTTP(w, r)
			return
		}

		path := r.URL.Path
		switch {
		case path == "/favicon.ico", strings.HasPrefix(path, "/ui-assets/"), strings.HasPrefix(path, "/assets/"):
			next.ServeHTTP(w, r)
			return
		case path == "/security", strings.HasPrefix(path, "/security/"):
			next.ServeHTTP(w, r)
			return
		case path == "/logout", path == "/api/auth/logout", path == "/api/auth/me":
			next.ServeHTTP(w, r)
			return
		case path == "/login", path == "/login/mfa", path == "/login/mfa/cancel", strings.HasPrefix(path, "/login/magic/"):
			next.ServeHTTP(w, r)
			return
		case path == "/api/auth/login", path == "/api/auth/mfa", path == "/api/auth/magic-link", strings.HasPrefix(path, "/api/auth/magic/"):
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(path, "/api/") {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"error": "MFA wajib diaktifkan sebelum menggunakan API.",
				"code":  "mfa_setup_required",
			})
			return
		}
		http.Redirect(w, r, "/security?required=1", http.StatusSeeOther)
	})
}

func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic on %s %s: %v\n%s", r.Method, r.URL.Path, recovered, debug.Stack())
				writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	allowed := map[string]bool{
		"http://" + r.Host:  true,
		"https://" + r.Host: true,
	}
	if appURL := os.Getenv("APP_URL"); appURL != "" {
		if parsed, err := url.Parse(appURL); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			allowed[parsed.Scheme+"://"+parsed.Host] = true
		}
	}
	return allowed[origin]
}

func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		// Only the last hop was appended by the trusted proxy directly in front
		// of this app. Use it instead of the leftmost value so a client cannot
		// choose its own rate-limit key.
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			candidate := strings.TrimSpace(parts[len(parts)-1])
			if net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rememberSession records where the token was created. Failure to store this
// metadata only degrades the security page; it never blocks login.
func (s *Server) rememberSession(r *http.Request, token, userID string) {
	if s.cache == nil || token == "" {
		return
	}
	info := model.SessionInfo{
		UserID:    userID,
		IP:        s.clientIP(r),
		UserAgent: r.UserAgent(),
		CreatedAt: time.Now(),
	}
	if err := s.cache.SetSessionInfo(r.Context(), token, info, s.authSvc.SessionTTL()); err != nil {
		log.Printf("session info write failed: %v", err)
	}
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		// Lax keeps the session usable right after a WhatsApp magic-link
		// redirect; CSRF is covered by the Origin check on unsafe methods.
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookie,
		MaxAge:   int(s.authSvc.SessionTTL().Seconds()),
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookie,
		MaxAge:   -1,
	})
}

func currentAuth(r *http.Request) (requestAuth, bool) {
	return requestAuthFromContext(r.Context())
}

func requestAuthFromContext(ctx context.Context) (requestAuth, bool) {
	value, ok := ctx.Value(authContextKey{}).(requestAuth)
	return value, ok && value.user != nil
}

func requireUser(w http.ResponseWriter, r *http.Request) (requestAuth, bool) {
	authValue, ok := currentAuth(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Autentikasi diperlukan. Silakan login kembali.")
		return requestAuth{}, false
	}
	return authValue, true
}

func requireRole(w http.ResponseWriter, r *http.Request, roles ...string) (requestAuth, bool) {
	authValue, ok := requireUser(w, r)
	if !ok {
		return requestAuth{}, false
	}
	for _, role := range roles {
		if authValue.user.Role == role {
			return authValue, true
		}
	}
	writeError(w, http.StatusForbidden, "Peran akun Anda tidak diizinkan untuk aksi ini.")
	return requestAuth{}, false
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeAuthError(w http.ResponseWriter, err error) {
	var authErr *auth.Error
	if errors.As(err, &authErr) {
		switch authErr.Code {
		case auth.CodeLocked, auth.CodeRateLimited:
			if authErr.RetryAfter > 0 {
				w.Header().Set("Retry-After", strconv.Itoa(int(authErr.RetryAfter.Seconds())))
			}
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error":        authErr.Message,
				"retryAfterMs": authErr.RetryAfter.Milliseconds(),
			})
		case auth.CodeUnauthenticated:
			writeError(w, http.StatusUnauthorized, authErr.Message)
		default:
			writeError(w, http.StatusUnauthorized, authErr.Message)
		}
		return
	}

	log.Printf("auth error: %v", err)
	writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "Ukuran body melebihi batas 2 MB.")
			return false
		}
		writeError(w, http.StatusBadRequest, "Body JSON tidak valid.")
		return false
	}
	return true
}

func timestampNow() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}
