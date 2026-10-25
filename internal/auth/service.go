package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"wesmk/internal/model"
)

const (
	CodeInvalidCredentials = "invalid_credentials"
	CodeLocked             = "locked"
	CodeDemoDisabled       = "demo_disabled"
	CodeUnauthenticated    = "unauthenticated"
	CodeRateLimited        = "rate_limited"
)

type Error struct {
	Code       string
	Message    string
	RetryAfter time.Duration
}

func (e *Error) Error() string { return e.Message }

type Repository interface {
	UserByIdentifier(ctx context.Context, identifier string) (*model.User, error)
	UserByID(ctx context.Context, id string) (*model.User, error)
	ParentByPhone(ctx context.Context, normalizedPhone string) (*model.User, error)
	DemoUserByRole(ctx context.Context, role string) (*model.User, error)
}

type Cache interface {
	CreateSession(ctx context.Context, token, userID string, ttl time.Duration) error
	SessionUser(ctx context.Context, token string) (string, error)
	DeleteSession(ctx context.Context, token string) error
	LoginFailureCount(ctx context.Context, key string) (int, error)
	RecordLoginFailure(ctx context.Context, key string, window time.Duration) (int, error)
	ClearLoginFailures(ctx context.Context, key string) error
	StoreMagicToken(ctx context.Context, tokenHash, userID string, ttl time.Duration) error
	ConsumeMagicToken(ctx context.Context, tokenHash string) (string, error)
	IncrementCounter(ctx context.Context, key string, window time.Duration) (int, error)
}

type Options struct {
	DemoMode         bool
	DemoPassword     string
	SessionTTL       time.Duration
	MaxLoginAttempts int
	LoginWindow      time.Duration
	MagicTokenTTL    time.Duration
}

type Service struct {
	repo  Repository
	cache Cache
	opts  Options
	dummy string
}

type Session struct {
	Token     string
	User      *model.User
	ExpiresAt time.Time
}

func New(repo Repository, cache Cache, options Options) *Service {
	if options.SessionTTL == 0 {
		options.SessionTTL = 8 * time.Hour
	}
	if options.MaxLoginAttempts == 0 {
		options.MaxLoginAttempts = 5
	}
	if options.LoginWindow == 0 {
		options.LoginWindow = 10 * time.Minute
	}
	if options.MagicTokenTTL == 0 {
		options.MagicTokenTTL = 5 * time.Minute
	}
	if options.DemoPassword == "" {
		options.DemoPassword = "Wesmk#2026"
	}
	return &Service{repo: repo, cache: cache, opts: options}
}

// Init hashes a throwaway password so unknown identifiers cost the same as
// known ones.
func (s *Service) Init() error {
	dummy, err := HashPassword(randomString(16))
	if err != nil {
		return err
	}
	s.dummy = dummy
	return nil
}

func (s *Service) DemoMode() bool            { return s.opts.DemoMode }
func (s *Service) DemoPassword() string      { return s.opts.DemoPassword }
func (s *Service) SessionTTL() time.Duration { return s.opts.SessionTTL }

func (s *Service) Login(ctx context.Context, identifier, password, ip string) (*Session, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || password == "" {
		return nil, &Error{Code: CodeInvalidCredentials, Message: "NISN/NIP atau kata sandi salah."}
	}

	key := "ratelimit:login:" + ip + "|" + strings.ToLower(identifier)
	failures, err := s.cache.LoginFailureCount(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read login failures: %w", err)
	}
	if failures >= s.opts.MaxLoginAttempts {
		return nil, &Error{
			Code:       CodeLocked,
			Message:    fmt.Sprintf("Terlalu banyak percobaan gagal. Coba lagi dalam %d menit.", int(s.opts.LoginWindow.Minutes())),
			RetryAfter: s.opts.LoginWindow,
		}
	}

	user, err := s.repo.UserByIdentifier(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}

	hash := s.dummy
	if user != nil {
		hash = user.PasswordHash
	}
	valid := VerifyPassword(password, hash)

	if user == nil || !valid {
		if _, err := s.cache.RecordLoginFailure(ctx, key, s.opts.LoginWindow); err != nil {
			return nil, fmt.Errorf("record login failure: %w", err)
		}
		return nil, &Error{Code: CodeInvalidCredentials, Message: "NISN/NIP atau kata sandi salah."}
	}

	if err := s.cache.ClearLoginFailures(ctx, key); err != nil {
		return nil, fmt.Errorf("clear login failures: %w", err)
	}
	return s.issueSession(ctx, user)
}

func (s *Service) DemoLogin(ctx context.Context, identifier string) (*Session, error) {
	if !s.opts.DemoMode {
		return nil, &Error{Code: CodeDemoDisabled, Message: "Mode demo dinonaktifkan pada server ini."}
	}

	user, err := s.repo.UserByIdentifier(ctx, strings.TrimSpace(identifier))
	if err != nil {
		return nil, fmt.Errorf("lookup demo user: %w", err)
	}
	if user == nil || !user.IsDemo {
		return nil, &Error{Code: CodeInvalidCredentials, Message: "Akun demo tidak ditemukan."}
	}
	return s.issueSession(ctx, user)
}

func (s *Service) SwitchDemoRole(ctx context.Context, token, role string) (*Session, error) {
	if !s.opts.DemoMode {
		return nil, &Error{Code: CodeDemoDisabled, Message: "Mode demo dinonaktifkan pada server ini."}
	}

	current, err := s.UserForToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, &Error{Code: CodeUnauthenticated, Message: "Sesi tidak valid. Silakan login kembali."}
	}

	target, err := s.repo.DemoUserByRole(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("lookup demo role: %w", err)
	}
	if target == nil {
		return nil, &Error{Code: CodeInvalidCredentials, Message: "Peran demo tidak tersedia."}
	}

	// Changing roles replaces the current session; the old token must stop working.
	if err := s.cache.DeleteSession(ctx, token); err != nil {
		return nil, fmt.Errorf("delete previous session: %w", err)
	}
	return s.issueSession(ctx, target)
}

func (s *Service) UserForToken(ctx context.Context, token string) (*model.User, error) {
	if token == "" {
		return nil, nil
	}
	userID, err := s.cache.SessionUser(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}
	if userID == "" {
		return nil, nil
	}

	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load session user: %w", err)
	}
	if user == nil {
		_ = s.cache.DeleteSession(ctx, token)
		return nil, nil
	}
	return user, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.cache.DeleteSession(ctx, token)
}

// RequestMagicLink returns a raw token only when a matching parent exists. The
// caller must return the same response for unknown numbers to avoid enumeration.
func (s *Service) RequestMagicLink(ctx context.Context, phone, ip string) (string, bool, error) {
	normalized := model.NormalizePhone(phone)
	if len(normalized) < 8 {
		return "", false, &Error{Code: CodeInvalidCredentials, Message: "Nomor WhatsApp tidak valid."}
	}

	key := "ratelimit:magic:" + ip + "|" + normalized
	count, err := s.cache.IncrementCounter(ctx, key, s.opts.LoginWindow)
	if err != nil {
		return "", false, fmt.Errorf("record magic link request: %w", err)
	}
	if count > 5 {
		return "", false, &Error{
			Code:       CodeRateLimited,
			Message:    "Terlalu banyak permintaan tautan. Coba lagi nanti.",
			RetryAfter: s.opts.LoginWindow,
		}
	}

	parent, err := s.repo.ParentByPhone(ctx, normalized)
	if err != nil {
		return "", false, fmt.Errorf("lookup parent: %w", err)
	}
	if parent == nil {
		return "", false, nil
	}

	token := randomString(24)
	if err := s.cache.StoreMagicToken(ctx, hashToken(token), parent.ID, s.opts.MagicTokenTTL); err != nil {
		return "", false, fmt.Errorf("store magic token: %w", err)
	}
	return token, true, nil
}

func (s *Service) ConsumeMagicToken(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, nil
	}
	userID, err := s.cache.ConsumeMagicToken(ctx, hashToken(token))
	if err != nil {
		return nil, fmt.Errorf("consume magic token: %w", err)
	}
	if userID == "" {
		return nil, nil
	}

	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load magic link user: %w", err)
	}
	if user == nil {
		return nil, nil
	}
	return s.issueSession(ctx, user)
}

func (s *Service) issueSession(ctx context.Context, user *model.User) (*Session, error) {
	token := randomString(32)
	if err := s.cache.CreateSession(ctx, token, user.ID, s.opts.SessionTTL); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &Session{Token: token, User: user, ExpiresAt: time.Now().Add(s.opts.SessionTTL)}, nil
}

func randomString(bytesCount int) string {
	buf := make([]byte, bytesCount)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func IsErrorCode(err error, code string) bool {
	var authErr *Error
	return errors.As(err, &authErr) && authErr.Code == code
}
