package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env          string
	Port         string
	DatabaseURL  string
	RedisURL     string
	GeminiAPIKey string
	GeminiModel  string

	// ReceiptSecret signs ledger receipts server-side.
	ReceiptSecret string
	// PaymentWebhookSecret authenticates gateway callbacks. Empty disables the
	// webhook instead of accepting unsigned settlements.
	PaymentWebhookSecret string
	TreasurerPin         string

	DemoMode     bool
	DemoPassword string
	SecureCookie bool
	TrustProxy   bool

	SessionTTL time.Duration
}

func Load() Config {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("WESMK_ENV")))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("NODE_ENV")))
	}
	if env == "prod" {
		env = "production"
	}
	if env == "" {
		env = "development"
	}

	demoMode := env != "production"
	if value, ok := os.LookupEnv("WESMK_DEMO_MODE"); ok {
		demoMode = parseBool(value, demoMode)
	}

	// Production always gets Secure cookies; TLS terminates at the reverse proxy,
	// so the app cannot infer the scheme from APP_URL or the request itself.
	secureCookie := env == "production"
	if value, ok := os.LookupEnv("WESMK_SECURE_COOKIE"); ok {
		secureCookie = parseBool(value, secureCookie)
	}

	trustProxy := env == "production"
	if value, ok := os.LookupEnv("WESMK_TRUST_PROXY"); ok {
		trustProxy = parseBool(value, trustProxy)
	}

	demoPassword := os.Getenv("WESMK_DEMO_PASSWORD")
	if demoPassword == "" {
		demoPassword = "Wesmk#2026"
	}

	receiptSecret := os.Getenv("WESMK_RECEIPT_SECRET")
	if receiptSecret == "" {
		receiptSecret = "dev-receipt-secret-bukan-untuk-produksi"
	}

	return Config{
		Env:                  env,
		Port:                 envOrDefault("PORT", "3000"),
		DatabaseURL:          envOrDefault("DATABASE_URL", "postgres://wesmk:wesmk@localhost:5432/wesmk?sslmode=disable"),
		RedisURL:             envOrDefault("REDIS_URL", "redis://localhost:6379/0"),
		GeminiAPIKey:         os.Getenv("GEMINI_API_KEY"),
		GeminiModel:          envOrDefault("GEMINI_MODEL", "gemini-2.5-flash"),
		ReceiptSecret:        receiptSecret,
		PaymentWebhookSecret: os.Getenv("WESMK_PAYMENT_WEBHOOK_SECRET"),
		TreasurerPin:         envOrDefault("WESMK_TREASURER_PIN", "123456"),
		DemoMode:             demoMode,
		DemoPassword:         demoPassword,
		SecureCookie:         secureCookie,
		TrustProxy:           trustProxy,
		SessionTTL:           8 * time.Hour,
	}
}

// Validate refuses production boots that would silently fall back to the local
// development database or Redis.
func (c Config) Validate() error {
	if c.Env != "production" {
		return nil
	}
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) == "" {
		return errors.New("DATABASE_URL wajib diisi saat WESMK_ENV=production")
	}
	if strings.TrimSpace(os.Getenv("REDIS_URL")) == "" {
		return errors.New("REDIS_URL wajib diisi saat WESMK_ENV=production")
	}
	// A default secret would let anyone with the source forge a receipt.
	if len(strings.TrimSpace(os.Getenv("WESMK_RECEIPT_SECRET"))) < 16 {
		return errors.New("WESMK_RECEIPT_SECRET (minimal 16 karakter) wajib diisi saat WESMK_ENV=production")
	}
	// Demo accounts are a development aid; a public deployment must not expose
	// the role switcher or the shared demo password.
	if c.DemoMode {
		return errors.New("WESMK_DEMO_MODE harus false saat WESMK_ENV=production")
	}
	// The fallback PIN is public knowledge, so production has to set its own.
	pin := strings.TrimSpace(os.Getenv("WESMK_TREASURER_PIN"))
	if pin == "" || pin == "123456" {
		return errors.New("WESMK_TREASURER_PIN (6 digit, bukan default) wajib diisi saat WESMK_ENV=production")
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseBool(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fallback
		}
		return b
	}
}
