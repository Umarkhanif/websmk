package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wesmk/internal/state"
)

type invoicePayload struct {
	InvoiceID string `json:"invoiceId"`
}

func decodeInvoiceID(w http.ResponseWriter, r *http.Request) (string, bool) {
	var payload invoicePayload
	if !decodeJSON(w, r, &payload) {
		return "", false
	}
	invoiceID := strings.TrimSpace(payload.InvoiceID)
	if invoiceID == "" || len(invoiceID) > 64 {
		writeError(w, http.StatusBadRequest, "ID tagihan tidak valid.")
		return "", false
	}
	return invoiceID, true
}

// handleCreatePaymentIntent issues (or returns) a pending virtual account for
// the caller's own unpaid invoice. It never marks the invoice paid.
func (s *Server) handleCreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	studentID, ok := studentScopedID(w, r, "student", "parent")
	if !ok {
		return
	}
	invoiceID, ok := decodeInvoiceID(w, r)
	if !ok {
		return
	}
	if !s.allowDomainWrite(w, r, "payment-intent") {
		return
	}

	now := time.Now()
	s.updateState(w, r, func(doc map[string]any) (any, error) {
		return state.CreatePaymentIntent(doc, studentID, invoiceID, now)
	})
}

// handlePaymentCallback is the only production settlement path. Requests must
// carry a fresh timestamp and an HMAC of the raw body, so a leaked intent id is
// not enough to mark an invoice paid.
func (s *Server) handlePaymentCallback(w http.ResponseWriter, r *http.Request) {
	if s.cfg.PaymentWebhookSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "Gateway pembayaran belum dikonfigurasi.")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8<<10))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Body callback tidak valid.")
		return
	}
	timestamp := r.Header.Get("X-Wesmk-Timestamp")
	if !validWebhookSignature(s.cfg.PaymentWebhookSecret, timestamp, body, r.Header.Get("X-Wesmk-Signature")) {
		writeError(w, http.StatusUnauthorized, "Tanda tangan callback tidak valid.")
		return
	}

	var payload struct {
		IntentID    string `json:"intentId"`
		ProviderRef string `json:"providerRef"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Body callback tidak valid.")
		return
	}
	intentID := strings.TrimSpace(payload.IntentID)
	if intentID == "" || len(intentID) > 64 || len(payload.ProviderRef) > 128 {
		writeError(w, http.StatusBadRequest, "Data callback tidak valid.")
		return
	}

	now := time.Now()
	s.updateState(w, r, func(doc map[string]any) (any, error) {
		return state.SettlePaymentIntent(doc, intentID, strings.TrimSpace(payload.ProviderRef), s.cfg.ReceiptSecret, now)
	})
}

func validWebhookSignature(secret, timestamp string, body []byte, signature string) bool {
	if timestamp == "" || signature == "" {
		return false
	}
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || time.Since(time.Unix(seconds, 0)).Abs() > 5*time.Minute {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(signature))) == 1
}
