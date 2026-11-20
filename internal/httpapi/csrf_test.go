package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"wesmk/internal/config"
	"wesmk/internal/model"
)

func newCSRFTester() (*Server, *bool) {
	server := &Server{cfg: config.Config{ReceiptSecret: "test-receipt-secret"}}
	called := false
	return server, &called
}

func TestCSRFGuardAllowsGetAndUnknownPublicPost(t *testing.T) {
	server, called := newCSRFTester()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { *called = true })
	handler := server.csrfGuard(next)

	get := httptest.NewRequest(http.MethodGet, "/student/dashboard", nil)
	handler.ServeHTTP(httptest.NewRecorder(), get)
	if !*called {
		t.Fatal("GET must pass without a token")
	}

	*called = false
	post := httptest.NewRequest(http.MethodPost, "/unknown-public", nil)
	handler.ServeHTTP(httptest.NewRecorder(), post)
	if !*called {
		t.Fatal("unknown public POST should fall through to the handler")
	}
}

func TestCSRFGuardEnforcesToken(t *testing.T) {
	server, called := newCSRFTester()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { *called = true })
	handler := server.csrfGuard(next)

	recorder := httptest.NewRecorder()
	wrong := httptest.NewRequest(http.MethodPost, "/login", nil)
	wrong.Header.Set("X-CSRF-Token", "salah")
	handler.ServeHTTP(recorder, wrong)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("wrong token status = %d, want 403", recorder.Code)
	}
	if *called {
		t.Fatal("handler must not run with a wrong token")
	}

	*called = false
	recorder = httptest.NewRecorder()
	right := httptest.NewRequest(http.MethodPost, "/login", nil)
	right.Header.Set("X-CSRF-Token", server.csrfTokenFor("login"))
	handler.ServeHTTP(recorder, right)
	if recorder.Code != http.StatusOK || !*called {
		t.Fatalf("correct token status = %d called = %v, want 200/true", recorder.Code, *called)
	}
}

func TestCSRFGuardUsesSessionTokenWhenAuthenticated(t *testing.T) {
	server, called := newCSRFTester()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { *called = true })
	handler := server.csrfGuard(next)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/student/quiz", nil)
	ctx := context.WithValue(request.Context(), authContextKey{}, requestAuth{
		user:  &model.User{ID: "usr-1", Role: "student"},
		token: "session-token",
	})
	request = request.WithContext(ctx)
	request.Header.Set("X-CSRF-Token", server.csrfTokenFor("session-token"))
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !*called {
		t.Fatalf("session token status = %d called = %v, want 200/true", recorder.Code, *called)
	}
}

func TestCSRFGuardSkipsAPI(t *testing.T) {
	server, called := newCSRFTester()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { *called = true })
	handler := server.csrfGuard(next)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/state", nil)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !*called {
		t.Fatalf("API POST status = %d called = %v, want 200/true", recorder.Code, *called)
	}
}
