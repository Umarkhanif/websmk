package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ErrWANotConfigured means a magic link was issued but cannot be delivered
// because no WhatsApp webhook is configured.
var ErrWANotConfigured = errors.New("pengiriman WhatsApp belum dikonfigurasi")

// sendMagicLink delivers a one-time parent login link to a configured webhook.
// In development without a webhook the link is logged for local testing; in
// production delivery is mandatory so a magic link never silently disappears.
func (s *Server) sendMagicLink(ctx context.Context, phone, token string) error {
	if s.cfg.WANotifyURL == "" {
		if s.cfg.Env == "production" {
			return ErrWANotConfigured
		}
		log.Printf("[auth] mode demo: tautan magic untuk %s: /login/magic/%s", maskPhone(phone), token)
		return nil
	}

	appURL := strings.TrimSpace(os.Getenv("APP_URL"))
	if appURL == "" {
		appURL = "http://localhost:3000"
	}
	link := strings.TrimRight(appURL, "/") + "/login/magic/" + token
	payload, err := json.Marshal(map[string]string{
		"to":      phone,
		"message": "Tautan masuk sekali pakai WESMK (berlaku 5 menit): " + link,
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.WANotifyURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if s.cfg.WANotifyToken != "" {
		request.Header.Set("Authorization", "Bearer "+s.cfg.WANotifyToken)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("webhook WhatsApp: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("webhook WhatsApp mengembalikan status %d", response.StatusCode)
	}
	return nil
}

func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return "****" + phone[len(phone)-4:]
}
