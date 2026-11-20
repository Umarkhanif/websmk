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
	if err := Load().Validate(); err != nil {
		t.Fatalf("production with explicit datastores, receipt secret, and PIN: %v", err)
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
	if cfg.DemoMode {
		t.Fatal("demo mode must default off in production")
	}
}

func TestDevelopmentDefaults(t *testing.T) {
	t.Setenv("WESMK_ENV", "development")
	t.Setenv("WESMK_SECURE_COOKIE", "")
	t.Setenv("WESMK_DEMO_MODE", "")
	t.Setenv("APP_URL", "")

	cfg := Load()
	if cfg.SecureCookie {
		t.Fatal("development should not force Secure cookies")
	}
	if !cfg.DemoMode {
		t.Fatal("demo mode should default on outside production")
	}
}
