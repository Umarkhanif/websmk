package auth

import (
	"context"
	"sync"
	"testing"
	"time"

	"wesmk/internal/model"
)

type fakeRepo struct {
	byIdentifier map[string]*model.User
	byID         map[string]*model.User
	byRole       map[string]*model.User
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		byIdentifier: map[string]*model.User{},
		byID:         map[string]*model.User{},
		byRole:       map[string]*model.User{},
	}
}

func (r *fakeRepo) add(user *model.User) {
	r.byIdentifier[user.Username] = user
	r.byID[user.ID] = user
	if _, exists := r.byRole[user.Role]; !exists {
		r.byRole[user.Role] = user
	}
}

func (r *fakeRepo) UserByIdentifier(_ context.Context, identifier string) (*model.User, error) {
	return r.byIdentifier[identifier], nil
}

func (r *fakeRepo) UserByID(_ context.Context, id string) (*model.User, error) {
	return r.byID[id], nil
}

func (r *fakeRepo) ParentByPhone(_ context.Context, normalized string) (*model.User, error) {
	for _, user := range r.byID {
		if user.Role == "parent" && model.NormalizePhone(user.PhoneWA) == normalized {
			return user, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) DemoUserByRole(_ context.Context, role string) (*model.User, error) {
	user, ok := r.byRole[role]
	if !ok || !user.IsDemo {
		return nil, nil
	}
	return user, nil
}

type fakeCache struct {
	mu         sync.Mutex
	sessions   map[string]string
	userTokens map[string]string
	failures   map[string]int
	magic      map[string]string
	counters   map[string]int
}

func newFakeCache() *fakeCache {
	return &fakeCache{
		sessions:   map[string]string{},
		userTokens: map[string]string{},
		failures:   map[string]int{},
		magic:      map[string]string{},
		counters:   map[string]int{},
	}
}

func (c *fakeCache) CreateSession(_ context.Context, token, userID string, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if previous, ok := c.userTokens[userID]; ok {
		delete(c.sessions, previous)
	}
	c.sessions[token] = userID
	c.userTokens[userID] = token
	return nil
}

func (c *fakeCache) SessionUser(_ context.Context, token string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessions[token], nil
}

func (c *fakeCache) DeleteSession(_ context.Context, token string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	userID := c.sessions[token]
	delete(c.sessions, token)
	if userID != "" && c.userTokens[userID] == token {
		delete(c.userTokens, userID)
	}
	return nil
}

func (c *fakeCache) LoginFailureCount(_ context.Context, key string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failures[key], nil
}

func (c *fakeCache) RecordLoginFailure(_ context.Context, key string, _ time.Duration) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures[key]++
	return c.failures[key], nil
}

func (c *fakeCache) ClearLoginFailures(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.failures, key)
	return nil
}

func (c *fakeCache) StoreMagicToken(_ context.Context, tokenHash, userID string, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.magic[tokenHash] = userID
	return nil
}

func (c *fakeCache) ConsumeMagicToken(_ context.Context, tokenHash string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	userID := c.magic[tokenHash]
	delete(c.magic, tokenHash)
	return userID, nil
}

func (c *fakeCache) IncrementCounter(_ context.Context, key string, _ time.Duration) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters[key]++
	return c.counters[key], nil
}

const testDemoPassword = "Test#12345"

func newTestService(t *testing.T, options Options) (*Service, *fakeRepo, *fakeCache) {
	t.Helper()

	hash, err := HashPassword(testDemoPassword)
	if err != nil {
		t.Fatalf("hash fixture: %v", err)
	}

	repo := newFakeRepo()
	repo.add(&model.User{
		ID: "usr-student", Username: "0058471290", FullName: "Murid Uji", Role: "student",
		PasswordHash: hash, IsDemo: true, PhoneWA: "081234567801",
	})
	repo.add(&model.User{
		ID: "usr-parent", Username: "081234567800", FullName: "Wali Uji", Role: "parent",
		PasswordHash: hash, IsDemo: true, PhoneWA: "081234567800",
	})
	repo.add(&model.User{
		ID: "usr-headmaster", Username: "196805121994121001", FullName: "Kepsek Uji", Role: "headmaster",
		PasswordHash: hash, IsDemo: true, PhoneWA: "081398765406",
	})

	if options.DemoMode {
		options.DemoPassword = testDemoPassword
	}
	if options.SessionTTL == 0 {
		options.SessionTTL = time.Hour
	}
	if options.MaxLoginAttempts == 0 {
		options.MaxLoginAttempts = 3
	}
	if options.LoginWindow == 0 {
		options.LoginWindow = 10 * time.Minute
	}
	if options.MagicTokenTTL == 0 {
		options.MagicTokenTTL = 5 * time.Minute
	}

	cache := newFakeCache()
	service := New(repo, cache, options)
	if err := service.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	return service, repo, cache
}

func TestLoginRejectsWrongPasswordAndAcceptsCorrectOne(t *testing.T) {
	service, _, _ := newTestService(t, Options{DemoMode: true})
	ctx := context.Background()

	_, err := service.Login(ctx, "0058471290", "salah", "10.0.0.1")
	if !IsErrorCode(err, CodeInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	session, err := service.Login(ctx, "0058471290", testDemoPassword, "10.0.0.1")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if session.User.ID != "usr-student" {
		t.Fatalf("unexpected user %s", session.User.ID)
	}
	user, err := service.UserForToken(ctx, session.Token)
	if err != nil || user == nil || user.ID != "usr-student" {
		t.Fatalf("session not valid: %v %v", user, err)
	}
}

func TestLoginIsRateLimited(t *testing.T) {
	service, _, _ := newTestService(t, Options{DemoMode: true})
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := service.Login(ctx, "0058471290", "salah", "10.0.0.2")
		if !IsErrorCode(err, CodeInvalidCredentials) {
			t.Fatalf("attempt %d: expected invalid credentials, got %v", i+1, err)
		}
	}

	_, err := service.Login(ctx, "0058471290", testDemoPassword, "10.0.0.2")
	if !IsErrorCode(err, CodeLocked) {
		t.Fatalf("expected locked, got %v", err)
	}

	// Another IP is unaffected.
	if _, err := service.Login(ctx, "0058471290", testDemoPassword, "10.0.0.3"); err != nil {
		t.Fatalf("other ip should still log in: %v", err)
	}
}

func TestSingleDeviceSession(t *testing.T) {
	service, _, _ := newTestService(t, Options{DemoMode: true})
	ctx := context.Background()

	first, err := service.Login(ctx, "0058471290", testDemoPassword, "10.0.0.4")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Login(ctx, "0058471290", testDemoPassword, "10.0.0.4")
	if err != nil {
		t.Fatal(err)
	}

	if user, _ := service.UserForToken(ctx, first.Token); user != nil {
		t.Fatal("first session should have been revoked")
	}
	if user, _ := service.UserForToken(ctx, second.Token); user == nil {
		t.Fatal("second session should be valid")
	}
}

func TestDemoShortcutsDisabledOutsideDemoMode(t *testing.T) {
	service, _, _ := newTestService(t, Options{DemoMode: false})
	ctx := context.Background()

	session, err := service.Login(ctx, "0058471290", testDemoPassword, "10.0.0.5")
	if err != nil {
		t.Fatalf("real login should work: %v", err)
	}

	if _, err := service.DemoLogin(ctx, "0058471290"); !IsErrorCode(err, CodeDemoDisabled) {
		t.Fatalf("expected demo disabled, got %v", err)
	}
	if _, err := service.SwitchDemoRole(ctx, session.Token, "headmaster"); !IsErrorCode(err, CodeDemoDisabled) {
		t.Fatalf("expected demo disabled, got %v", err)
	}
}

func TestSwitchDemoRoleReplacesSession(t *testing.T) {
	service, _, _ := newTestService(t, Options{DemoMode: true})
	ctx := context.Background()

	session, err := service.Login(ctx, "0058471290", testDemoPassword, "10.0.0.6")
	if err != nil {
		t.Fatal(err)
	}

	switched, err := service.SwitchDemoRole(ctx, session.Token, "headmaster")
	if err != nil {
		t.Fatalf("switch failed: %v", err)
	}
	if switched.User.Role != "headmaster" {
		t.Fatalf("unexpected role %s", switched.User.Role)
	}
	if user, _ := service.UserForToken(ctx, session.Token); user != nil {
		t.Fatal("old session should be revoked after role switch")
	}
	if user, _ := service.UserForToken(ctx, switched.Token); user == nil {
		t.Fatal("new session should be valid")
	}
}

func TestMagicLinkIsSingleUse(t *testing.T) {
	service, _, _ := newTestService(t, Options{DemoMode: true})
	ctx := context.Background()

	if token, issued, err := service.RequestMagicLink(ctx, "081200000000", "10.0.0.7"); err != nil || issued || token != "" {
		t.Fatalf("unknown number should not issue a token: %q %v %v", token, issued, err)
	}

	token, issued, err := service.RequestMagicLink(ctx, "081234567800", "10.0.0.7")
	if err != nil || !issued || token == "" {
		t.Fatalf("expected token, got %q %v %v", token, issued, err)
	}

	session, err := service.ConsumeMagicToken(ctx, token)
	if err != nil || session == nil || session.User.ID != "usr-parent" {
		t.Fatalf("consume failed: %v %v", session, err)
	}
	if second, _ := service.ConsumeMagicToken(ctx, token); second != nil {
		t.Fatal("magic link must be single use")
	}
}
