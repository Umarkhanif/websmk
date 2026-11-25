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
	CodeUnauthenticated    = "unauthenticated"
	CodeRateLimited        = "rate_limited"
	CodeMFARequired        = "mfa_required"
)

type Error struct {
	Code       string
	Message    string
	RetryAfter time.Duration
	Challenge  string
	UserID     string
}

func (e *Error) Error() string { return e.Message }

type Repository interface {
	UserByIdentifier(ctx context.Context, identifier string) (*model.User, error)
	UserByID(ctx context.Context, id string) (*model.User, error)
	ParentByPhone(ctx context.Context, normalizedPhone string) (*model.User, error)
	ConsumeRecoveryCode(ctx context.Context, userID, codeHash string) (bool, error)
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
	StoreMFAChallenge(ctx context.Context, tokenHash, userID string, ttl time.Duration) error
	MFAChallengeUser(ctx context.Context, tokenHash string) (string, error)
	DeleteMFAChallenge(ctx context.Context, tokenHash string) error
	IncrementCounter(ctx context.Context, key string, window time.Duration) (int, error)
}

type Options struct {
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
	// Method records how the session was authenticated for the login audit.
	Method string
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

	if user == nil || !valid || !user.IsActive {
		if _, err := s.cache.RecordLoginFailure(ctx, key, s.opts.LoginWindow); err != nil {
			return nil, fmt.Errorf("record login failure: %w", err)
		}
		return nil, &Error{Code: CodeInvalidCredentials, Message: "NISN/NIP atau kata sandi salah.", UserID: userIDOf(user)}
	}

	if err := s.cache.ClearLoginFailures(ctx, key); err != nil {
		return nil, fmt.Errorf("clear login failures: %w", err)
	}
	if user.TOTPEnabled {
		if user.TOTPSecret == "" {
			return nil, &Error{Code: CodeUnauthenticated, Message: "Konfigurasi MFA akun tidak valid. Hubungi Admin TU untuk reset MFA.", UserID: user.ID}
		}
		challenge := randomString(24)
		if err := s.cache.StoreMFAChallenge(ctx, hashToken(challenge), user.ID, 5*time.Minute); err != nil {
			return nil, fmt.Errorf("store mfa challenge: %w", err)
		}
		return nil, &Error{
			Code:      CodeMFARequired,
			Message:   "Masukkan kode 6 digit dari aplikasi autentikator.",
			Challenge: challenge,
		}
	}
	return s.issueSession(ctx, user, "password")
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
	if user == nil || !user.IsActive {
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
	return s.issueSession(ctx, user, "magic")
}

// VerifyMFA finishes a password login that required a TOTP code. The challenge
// is single-use and the attempt counter is bounded so a stolen challenge cannot
// be brute-forced.
func (s *Service) VerifyMFA(ctx context.Context, challenge, code string) (*Session, error) {
	if strings.TrimSpace(challenge) == "" || strings.TrimSpace(code) == "" {
		return nil, &Error{Code: CodeInvalidCredentials, Message: "Kode MFA tidak valid."}
	}
	hash := hashToken(strings.TrimSpace(challenge))
	attemptKey := "ratelimit:mfa:" + hash
	attempts, err := s.cache.IncrementCounter(ctx, attemptKey, 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("record mfa attempt: %w", err)
	}
	if attempts > 5 {
		_ = s.cache.DeleteMFAChallenge(ctx, hash)
		return nil, &Error{Code: CodeRateLimited, Message: "Terlalu banyak percobaan kode MFA. Login ulang."}
	}
	userID, err := s.cache.MFAChallengeUser(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("read mfa challenge: %w", err)
	}
	if userID == "" {
		return nil, &Error{Code: CodeInvalidCredentials, Message: "Sesi MFA kedaluwarsa. Silakan login ulang."}
	}
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load mfa user: %w", err)
	}
	if user == nil || !user.IsActive || !user.TOTPEnabled {
		return nil, &Error{Code: CodeInvalidCredentials, Message: "Kode MFA salah.", UserID: userID}
	}
	method := "totp"
	valid := false
	if len(strings.TrimSpace(code)) == 6 {
		valid = ValidateTOTP(user.TOTPSecret, code, time.Now())
	} else {
		consumed, err := s.repo.ConsumeRecoveryCode(ctx, user.ID, RecoveryCodeHash(user.ID, code))
		if err != nil {
			return nil, fmt.Errorf("consume recovery code: %w", err)
		}
		valid = consumed
		method = "recovery_code"
	}
	if !valid {
		return nil, &Error{Code: CodeInvalidCredentials, Message: "Kode MFA atau recovery code salah.", UserID: user.ID}
	}
	if err := s.cache.DeleteMFAChallenge(ctx, hash); err != nil {
		return nil, fmt.Errorf("delete mfa challenge: %w", err)
	}
	_ = s.cache.ClearLoginFailures(ctx, attemptKey)
	return s.issueSession(ctx, user, method)
}

func (s *Service) issueSession(ctx context.Context, user *model.User, method string) (*Session, error) {
	token := randomString(32)
	if err := s.cache.CreateSession(ctx, token, user.ID, s.opts.SessionTTL); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &Session{Token: token, User: user, ExpiresAt: time.Now().Add(s.opts.SessionTTL), Method: method}, nil
}

func userIDOf(user *model.User) string {
	if user == nil {
		return ""
	}
	return user.ID
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
