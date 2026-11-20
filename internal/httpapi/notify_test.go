package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wesmk/internal/config"
)

func TestSendMagicLinkPostsToWebhook(t *testing.T) {
	var received map[string]string
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer rahasia" {
			t.Errorf("Authorization = %q, want Bearer rahasia", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer webhook.Close()

	server := &Server{cfg: config.Config{
		WANotifyURL:   webhook.URL,
		WANotifyToken: "rahasia",
	}}
	t.Setenv("APP_URL", "https://sekolah.example")
	if err := server.sendMagicLink(context.Background(), "081234567800", "token-123"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if received["to"] != "081234567800" {
		t.Fatalf("to = %q", received["to"])
	}
	if !strings.Contains(received["message"], "https://sekolah.example/login/magic/token-123") {
		t.Fatalf("message = %q", received["message"])
	}
}

func TestSendMagicLinkProductionRequiresWebhook(t *testing.T) {
	server := &Server{cfg: config.Config{Env: "production"}}
	err := server.sendMagicLink(context.Background(), "081234567800", "token-123")
	if !errors.Is(err, ErrWANotConfigured) {
		t.Fatalf("error = %v, want ErrWANotConfigured", err)
	}
}
