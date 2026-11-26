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
	// MFAKey encrypts TOTP secrets at rest. Production must set a dedicated
	// value; development falls back to ReceiptSecret so local data stays readable.
	MFAKey string
	// PaymentWebhookSecret authenticates gateway callbacks. Empty disables the
	// webhook instead of accepting unsigned settlements.
	PaymentWebhookSecret string
	TreasurerPin         string

	// WANotifyURL is an HTTP webhook that delivers magic links to parents.
	// The webhook receives {"to","message"}; WANotifyToken is sent as a Bearer token.
	WANotifyURL   string
	WANotifyToken string

	SecureCookie bool
	TrustProxy   bool
	// DomainStudentProfile reads student profile fields from the normalized
	// students table, falling back to app_state when the row is not there yet.
	DomainStudentProfile bool
	// DomainFinance reads invoices and ledger rows from the normalized tables,
	// falling back to app_state when the domain slice is empty.
	DomainFinance bool
	// DomainAttendance reads daily attendance from the normalized table.
	DomainAttendance bool
	// DomainExam reads the question bank and proctor sessions from tables.
	DomainExam bool
	// DomainPklBkk reads PKL placements/journals/logs and BKK listings from tables.
	DomainPklBkk bool

	SessionTTL time.Duration

	// DemoNoAuth disables authentication entirely for local demos. It is always
	// forced off in production regardless of the env value.
	DemoNoAuth bool
	// DemoRole is the role attached to the synthetic demo user (default teacher).
	DemoRole string
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

	// Production always gets Secure cookies; TLS terminates at the reverse proxy,
	// so the app cannot infer the scheme from APP_URL or the request itself.
	secureCookie := env == "production"
	if value, ok := os.LookupEnv("WESMK_SECURE_COOKIE"); ok {
		secureCookie = parseBool(value, secureCookie)
	}

	// Never trust a client-supplied forwarding header unless the deployment
	// explicitly opts in behind a known reverse proxy.
	trustProxy := false
	if value, ok := os.LookupEnv("WESMK_TRUST_PROXY"); ok {
		trustProxy = parseBool(value, trustProxy)
	}

	domainStudentProfile := true
	if value, ok := os.LookupEnv("WESMK_DOMAIN_STUDENT_PROFILE"); ok {
		domainStudentProfile = parseBool(value, domainStudentProfile)
	}

	domainFinance := true
	if value, ok := os.LookupEnv("WESMK_DOMAIN_FINANCE"); ok {
		domainFinance = parseBool(value, domainFinance)
	}

	domainAttendance := true
	if value, ok := os.LookupEnv("WESMK_DOMAIN_ATTENDANCE"); ok {
		domainAttendance = parseBool(value, domainAttendance)
	}

	domainExam := true
	if value, ok := os.LookupEnv("WESMK_DOMAIN_EXAM"); ok {
		domainExam = parseBool(value, domainExam)
	}

	domainPklBkk := true
	if value, ok := os.LookupEnv("WESMK_DOMAIN_PKL_BKK"); ok {
		domainPklBkk = parseBool(value, domainPklBkk)
	}

	receiptSecret := os.Getenv("WESMK_RECEIPT_SECRET")
	if receiptSecret == "" {
		receiptSecret = "dev-receipt-secret-bukan-untuk-produksi"
	}
	mfaKey := os.Getenv("WESMK_MFA_KEY")
	if mfaKey == "" {
		mfaKey = receiptSecret
	}

	// Demo mode is never allowed in production.
	demoNoAuth := false
	if env != "production" {
		demoNoAuth = parseBool(os.Getenv("WESMK_DEMO_NOAUTH"), false)
	}
	demoRole := strings.TrimSpace(os.Getenv("WESMK_DEMO_ROLE"))
	if demoRole == "" {
		demoRole = "teacher"
	}

	return Config{
		Env:                  env,
		Port:                 envOrDefault("PORT", "3000"),
		DatabaseURL:          envOrDefault("DATABASE_URL", "postgres://wesmk:wesmk@localhost:5432/wesmk?sslmode=disable"),
		RedisURL:             envOrDefault("REDIS_URL", "redis://localhost:6379/0"),
		GeminiAPIKey:         os.Getenv("GEMINI_API_KEY"),
		GeminiModel:          envOrDefault("GEMINI_MODEL", "gemini-2.5-flash"),
		ReceiptSecret:        receiptSecret,
		MFAKey:               mfaKey,
		PaymentWebhookSecret: os.Getenv("WESMK_PAYMENT_WEBHOOK_SECRET"),
		TreasurerPin:         envOrDefault("WESMK_TREASURER_PIN", "123456"),
		WANotifyURL:          os.Getenv("WESMK_WA_WEBHOOK_URL"),
		WANotifyToken:        os.Getenv("WESMK_WA_WEBHOOK_TOKEN"),
		SecureCookie:         secureCookie,
		TrustProxy:           trustProxy,
		DomainStudentProfile: domainStudentProfile,
		DomainFinance:        domainFinance,
		DomainAttendance:     domainAttendance,
		DomainExam:           domainExam,
		DomainPklBkk:         domainPklBkk,
		SessionTTL:           8 * time.Hour,
		DemoNoAuth:           demoNoAuth,
		DemoRole:             demoRole,
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
	// Losing or sharing this key makes every stored TOTP secret unreadable.
	if len(strings.TrimSpace(os.Getenv("WESMK_MFA_KEY"))) < 32 {
		return errors.New("WESMK_MFA_KEY (minimal 32 karakter) wajib diisi saat WESMK_ENV=production")
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
