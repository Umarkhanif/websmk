package config

import "testing"

func TestProductionRequiresExplicitDatastores(t *testing.T) {
	t.Setenv("WESMK_ENV", "production")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")

	cfg := Load()
	if err := cfg.Validate(); err == nil {
		t.Fatal("production without DATABASE_URL/REDIS_URL must fail validation")
	}

	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/wesmk")
	t.Setenv("REDIS_URL", "redis://cache:6379/0")
	t.Setenv("WESMK_RECEIPT_SECRET", "")
	if err := Load().Validate(); err == nil {
		t.Fatal("production without WESMK_RECEIPT_SECRET must fail validation")
	}

	t.Setenv("WESMK_RECEIPT_SECRET", "rahasia-kwitansi-sekolah-2026")
	t.Setenv("WESMK_TREASURER_PIN", "")
	if err := Load().Validate(); err == nil {
		t.Fatal("production without WESMK_TREASURER_PIN must fail validation")
	}

	t.Setenv("WESMK_TREASURER_PIN", "482913")
	t.Setenv("WESMK_MFA_KEY", "")
	if err := Load().Validate(); err == nil {
		t.Fatal("production without WESMK_MFA_KEY must fail validation")
	}

	t.Setenv("WESMK_MFA_KEY", "kunci-enkripsi-mfa-produksi-32-karakter")
	if err := Load().Validate(); err != nil {
		t.Fatalf("production with explicit datastores, receipt secret, MFA key, and PIN: %v", err)
	}
}

func TestProductionDefaultsToSecureCookie(t *testing.T) {
	t.Setenv("WESMK_ENV", "prod")
	t.Setenv("WESMK_SECURE_COOKIE", "")
	t.Setenv("APP_URL", "http://localhost:3000")

	cfg := Load()
	if cfg.Env != "production" {
		t.Fatalf("env = %q, want production", cfg.Env)
	}
	if !cfg.SecureCookie {
		t.Fatal("production cookies must be Secure even without an https APP_URL")
	}
}

func TestTrustProxyOffUnlessExplicitlyEnabled(t *testing.T) {
	t.Setenv("WESMK_ENV", "production")
	t.Setenv("WESMK_TRUST_PROXY", "")
	if Load().TrustProxy {
		t.Fatal("production must not trust X-Forwarded-For by default")
	}

	t.Setenv("WESMK_TRUST_PROXY", "true")
	if !Load().TrustProxy {
		t.Fatal("WESMK_TRUST_PROXY=true should enable proxy trust")
	}
}

func TestDomainStudentProfileDefaultsOn(t *testing.T) {
	t.Setenv("WESMK_DOMAIN_STUDENT_PROFILE", "")
	if !Load().DomainStudentProfile {
		t.Fatal("domain student profile should be on by default")
	}

	t.Setenv("WESMK_DOMAIN_STUDENT_PROFILE", "false")
	if Load().DomainStudentProfile {
		t.Fatal("WESMK_DOMAIN_STUDENT_PROFILE=false must disable the table read")
	}
}

func TestDomainFinanceDefaultsOn(t *testing.T) {
	t.Setenv("WESMK_DOMAIN_FINANCE", "")
	if !Load().DomainFinance {
		t.Fatal("domain finance should be on by default")
	}

	t.Setenv("WESMK_DOMAIN_FINANCE", "false")
	if Load().DomainFinance {
		t.Fatal("WESMK_DOMAIN_FINANCE=false must disable the table read")
	}
}

func TestDomainAttendanceDefaultsOn(t *testing.T) {
	t.Setenv("WESMK_DOMAIN_ATTENDANCE", "")
	if !Load().DomainAttendance {
		t.Fatal("domain attendance should be on by default")
	}

	t.Setenv("WESMK_DOMAIN_ATTENDANCE", "false")
	if Load().DomainAttendance {
		t.Fatal("WESMK_DOMAIN_ATTENDANCE=false must disable the table read")
	}
}

func TestDomainExamDefaultsOn(t *testing.T) {
	t.Setenv("WESMK_DOMAIN_EXAM", "")
	if !Load().DomainExam {
		t.Fatal("domain exam should be on by default")
	}

	t.Setenv("WESMK_DOMAIN_EXAM", "false")
	if Load().DomainExam {
		t.Fatal("WESMK_DOMAIN_EXAM=false must disable the table read")
	}
}

func TestDevelopmentDefaults(t *testing.T) {
	t.Setenv("WESMK_ENV", "development")
	t.Setenv("WESMK_SECURE_COOKIE", "")
	t.Setenv("APP_URL", "")

	cfg := Load()
	if cfg.SecureCookie {
		t.Fatal("development should not force Secure cookies")
	}
}
