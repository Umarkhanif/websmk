package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebRedirectPlainFormUsesSeeOther(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/student/invoice/pay", nil)

	webRedirect(recorder, request, "/student/dashboard?flash=ok")

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if got := recorder.Header().Get("Location"); got != "/student/dashboard?flash=ok" {
		t.Fatalf("Location = %q", got)
	}
	if got := recorder.Header().Get("HX-Redirect"); got != "" {
		t.Fatalf("unexpected HX-Redirect for a plain form: %q", got)
	}
}

func TestWebRedirectHtmxAsksForFullNavigation(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/student/invoice/pay", nil)
	request.Header.Set("HX-Request", "true")

	webRedirect(recorder, request, "/student/dashboard?flash=ok")

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("HX-Redirect"); got != "/student/dashboard?flash=ok" {
		t.Fatalf("HX-Redirect = %q", got)
	}
	if got := recorder.Header().Get("Location"); got != "" {
		t.Fatalf("unexpected Location for an htmx request: %q", got)
	}
}
