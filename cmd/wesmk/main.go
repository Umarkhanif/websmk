package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"wesmk/internal/ai"
	"wesmk/internal/auth"
	"wesmk/internal/config"
	"wesmk/internal/db"
	"wesmk/internal/httpapi"
	"wesmk/internal/secure"
	"wesmk/internal/seed"
	"wesmk/internal/store"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config invalid: %v", err)
	}
	if cfg.Env == "production" && !cfg.SecureCookie {
		log.Printf("[WARN] cookie sesi tidak memakai flag Secure di production")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres init failed: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	cache, err := store.OpenCache(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis init failed: %v", err)
	}
	defer cache.Close()

	mfaCipher, err := secure.NewCipher(cfg.MFAKey)
	if err != nil {
		log.Fatalf("MFA encryption key invalid: %v", err)
	}
	storeDB := store.NewPostgres(pool, store.WithMFACipher(mfaCipher))
	if reencrypted, err := storeDB.EncryptLegacyTOTPSecrets(ctx); err != nil {
		log.Fatalf("encrypt legacy MFA secrets failed: %v", err)
	} else if reencrypted > 0 {
		log.Printf("[security] %d secret MFA lama dienkripsi ulang", reencrypted)
	}
	if err := seed.Run(ctx, storeDB, seed.Options{
		EnvUsersJSON: os.Getenv("WESMK_USERS"),
	}); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	// Shadow-copy core entities from the JSON blob into normalized tables.
	// A failure must not prevent the app from booting on the blob alone.
	if err := storeDB.BackfillDomainData(ctx); err != nil {
		log.Printf("[WARN] domain backfill skipped: %v", err)
	}

	authSvc := auth.New(storeDB, cache, auth.Options{
		SessionTTL: cfg.SessionTTL,
	})
	if err := authSvc.Init(); err != nil {
		log.Fatalf("auth init failed: %v", err)
	}

	api := httpapi.New(cfg, authSvc, storeDB, cache, ai.New(cfg.GeminiAPIKey, cfg.GeminiModel))
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}

	go func() {
		log.Printf("WESMK server listening on http://0.0.0.0:%s", cfg.Port)
		log.Printf("[auth] session: %s | secure cookie: %s", cfg.SessionTTL, onOff(cfg.SecureCookie))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server stopped unexpectedly: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("server stopped")
}

func onOff(value bool) string {
	if value {
		return "AKTIF"
	}
	return "NONAKTIF"
}
