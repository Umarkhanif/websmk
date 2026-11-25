package httpapi

import (
	"net/http/httptest"
	"testing"

	"wesmk/internal/config"
)

func TestClientIPIgnoresForwardedForUnlessTrustedProxy(t *testing.T) {
	server := &Server{cfg: config.Config{TrustProxy: false}}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "203.0.113.9:4567"
	request.Header.Set("X-Forwarded-For", "1.2.3.4")

	if got := server.clientIP(request); got != "203.0.113.9" {
		t.Fatalf("clientIP = %q, want the socket address", got)
	}
}

func TestClientIPUsesLastHopFromTrustedProxy(t *testing.T) {
	server := &Server{cfg: config.Config{TrustProxy: true}}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.0.0.2:443"
	request.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.7")

	if got := server.clientIP(request); got != "198.51.100.7" {
		t.Fatalf("clientIP = %q, want the proxy-appended hop", got)
	}
}

func TestClientIPFallsBackOnInvalidForwardedValue(t *testing.T) {
	server := &Server{cfg: config.Config{TrustProxy: true}}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.0.0.2:443"
	request.Header.Set("X-Forwarded-For", "1.2.3.4, not-an-ip")

	if got := server.clientIP(request); got != "10.0.0.2" {
		t.Fatalf("clientIP = %q, want the socket address when the header is invalid", got)
	}
}
