package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func signBody(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestValidWebhookSignature(t *testing.T) {
	const secret = "gateway-secret"
	body := []byte(`{"intentId":"pi-1","providerRef":"ref-1"}`)
	timestamp := fmt.Sprint(time.Now().Unix())
	signature := signBody(secret, timestamp, body)

	if !validWebhookSignature(secret, timestamp, body, signature) {
		t.Fatal("a fresh, correctly signed callback must be accepted")
	}
	if validWebhookSignature(secret, timestamp, []byte(`{"intentId":"pi-2"}`), signature) {
		t.Fatal("a tampered body must be rejected")
	}
	if validWebhookSignature("other-secret", timestamp, body, signature) {
		t.Fatal("a wrong secret must be rejected")
	}
	if validWebhookSignature(secret, timestamp, body, "") {
		t.Fatal("a missing signature must be rejected")
	}

	stale := fmt.Sprint(time.Now().Add(-10 * time.Minute).Unix())
	if validWebhookSignature(secret, stale, body, signBody(secret, stale, body)) {
		t.Fatal("a stale timestamp must be rejected")
	}
}
