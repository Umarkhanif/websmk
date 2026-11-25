package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wesmk/internal/config"
)

func TestSecurityHeadersAllowAlpineButBlockInlineScript(t *testing.T) {
	server := &Server{cfg: config.Config{Env: "test"}}
	handler := server.securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/", nil))

	csp := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self' 'unsafe-eval'") {
		t.Fatalf("CSP must allow Alpine's evaluator, got %q", csp)
	}
	if strings.Contains(csp, "script-src 'self' 'unsafe-inline'") {
		t.Fatalf("CSP must not allow inline scripts, got %q", csp)
	}
}
