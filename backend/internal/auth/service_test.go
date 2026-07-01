package auth

import (
	"context"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/crypto"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
)

func newService(t *testing.T) (*Service, *Repo) {
	t.Helper()
	database, err := db.InMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	cipher, err := crypto.NewCipher(make([]byte, 32))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	repo := NewRepo(database)
	svc := NewService(repo, NewTokenManager([]byte("test-secret")), cipher, "Test")
	return svc, repo
}

func TestBootstrap_Idempotent(t *testing.T) {
	svc, repo := newService(t)
	ctx := context.Background()

	if err := svc.Bootstrap(ctx, "admin", "pw"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := svc.Bootstrap(ctx, "admin", "pw"); err != nil {
		t.Fatalf("bootstrap again: %v", err)
	}
	n, _ := repo.Count(ctx)
	if n != 1 {
		t.Fatalf("expected 1 admin, got %d", n)
	}
}

func TestLogin_SuccessNo2FA(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	_ = svc.Bootstrap(ctx, "admin", "pw")

	res, err := svc.Login(ctx, "admin", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.Need2FA || res.Token == "" {
		t.Fatalf("expected direct session token, got %+v", res)
	}
	claims, err := svc.Tokens().Parse(res.Token)
	if err != nil || claims.Stage != StageAuth {
		t.Fatalf("expected auth-stage token, got %v err=%v", claims, err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	_ = svc.Bootstrap(ctx, "admin", "pw")

	if _, err := svc.Login(ctx, "admin", "nope"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if _, err := svc.Login(ctx, "ghost", "pw"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for unknown user, got %v", err)
	}
}

func TestTOTP_FullCycle(t *testing.T) {
	svc, repo := newService(t)
	ctx := context.Background()
	_ = svc.Bootstrap(ctx, "admin", "pw")
	admin, _ := repo.GetByUsername(ctx, "admin")

	// setup
	setup, err := svc.SetupTOTP(ctx, admin.ID)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if setup.OtpauthURL == "" || setup.QR == "" {
		t.Fatal("setup should return otpauth url and qr")
	}

	code, _ := totp.GenerateCode(setup.Secret, time.Now())

	// enable with valid code
	if err := svc.EnableTOTP(ctx, admin.ID, code); err != nil {
		t.Fatalf("enable: %v", err)
	}

	// now login requires 2FA
	res, err := svc.Login(ctx, "admin", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !res.Need2FA || res.Pending == "" {
		t.Fatalf("expected need_2fa with pending token, got %+v", res)
	}
	pendClaims, _ := svc.Tokens().Parse(res.Pending)
	if pendClaims.Stage != StagePending {
		t.Fatalf("expected pending stage, got %q", pendClaims.Stage)
	}

	// complete 2fa
	code2, _ := totp.GenerateCode(setup.Secret, time.Now())
	token, err := svc.CompleteTOTP(ctx, admin.ID, code2)
	if err != nil {
		t.Fatalf("complete totp: %v", err)
	}
	if token == "" {
		t.Fatal("expected session token after 2fa")
	}

	// wrong code rejected
	if _, err := svc.CompleteTOTP(ctx, admin.ID, "000000"); err == nil {
		t.Fatal("expected failure for wrong totp code")
	}
}

func TestEnableTOTP_WrongCode(t *testing.T) {
	svc, repo := newService(t)
	ctx := context.Background()
	_ = svc.Bootstrap(ctx, "admin", "pw")
	admin, _ := repo.GetByUsername(ctx, "admin")
	if _, err := svc.SetupTOTP(ctx, admin.ID); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := svc.EnableTOTP(ctx, admin.ID, "123456"); err == nil {
		t.Fatal("expected enable to fail with wrong code")
	}
}
