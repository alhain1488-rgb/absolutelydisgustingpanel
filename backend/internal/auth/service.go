package auth

import (
	"context"
	"errors"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/crypto"
)

// ErrInvalidCredentials is returned for any login/2FA failure. It is
// intentionally generic so callers cannot distinguish "bad user" from "bad
// password".
var ErrInvalidCredentials = errors.New("invalid credentials")

// Service implements the authentication use-cases.
type Service struct {
	repo   *Repo
	tokens *TokenManager
	cipher *crypto.Cipher
	issuer string
}

// NewService builds the auth service. issuer is used as the TOTP issuer label.
func NewService(repo *Repo, tokens *TokenManager, cipher *crypto.Cipher, issuer string) *Service {
	return &Service{repo: repo, tokens: tokens, cipher: cipher, issuer: issuer}
}

// LoginResult is returned from Login.
type LoginResult struct {
	Token   string `json:"token,omitempty"`
	Need2FA bool   `json:"need_2fa"`
	// Pending is a short-lived token the client must present to /2fa/verify.
	Pending string `json:"pending_token,omitempty"`
}

// Bootstrap creates the initial admin from configuration if none exists.
// It is safe to call on every start (idempotent).
func (s *Service) Bootstrap(ctx context.Context, username, password string) error {
	n, err := s.repo.Count(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.repo.Create(ctx, username, hash)
	return err
}

// Login verifies username/password. If TOTP is enabled it returns Need2FA with
// a pending token; otherwise it returns a full session token.
func (s *Service) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	admin, err := s.repo.GetByUsername(ctx, username)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !crypto.VerifyPassword(admin.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	if admin.TOTPEnabled {
		pending, err := s.tokens.IssuePending(admin.ID)
		if err != nil {
			return nil, err
		}
		return &LoginResult{Need2FA: true, Pending: pending}, nil
	}
	token, err := s.tokens.IssueSession(admin.ID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token}, nil
}

// CompleteTOTP verifies a TOTP code for a pending admin and returns a full
// session token.
func (s *Service) CompleteTOTP(ctx context.Context, adminID int64, code string) (string, error) {
	admin, err := s.repo.GetByID(ctx, adminID)
	if err != nil {
		return "", ErrInvalidCredentials
	}
	if !admin.TOTPEnabled || admin.TOTPSecretEnc == "" {
		return "", ErrInvalidCredentials
	}
	secret, err := s.cipher.DecryptString(admin.TOTPSecretEnc)
	if err != nil {
		return "", err
	}
	if !ValidateTOTP(secret, code) {
		return "", ErrInvalidCredentials
	}
	return s.tokens.IssueSession(admin.ID)
}

// SetupTOTP generates a new secret for the admin, stores it encrypted (not yet
// enabled) and returns the provisioning data.
func (s *Service) SetupTOTP(ctx context.Context, adminID int64) (*TOTPSetup, error) {
	admin, err := s.repo.GetByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	setup, err := GenerateTOTP(s.issuer, admin.Username)
	if err != nil {
		return nil, err
	}
	enc, err := s.cipher.EncryptString(setup.Secret)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetTOTPSecret(ctx, adminID, enc); err != nil {
		return nil, err
	}
	return setup, nil
}

// EnableTOTP confirms enrollment: it validates a code against the stored secret
// and, on success, marks TOTP enabled.
func (s *Service) EnableTOTP(ctx context.Context, adminID int64, code string) error {
	admin, err := s.repo.GetByID(ctx, adminID)
	if err != nil {
		return err
	}
	if admin.TOTPSecretEnc == "" {
		return errors.New("no TOTP secret set up; call setup first")
	}
	secret, err := s.cipher.DecryptString(admin.TOTPSecretEnc)
	if err != nil {
		return err
	}
	if !ValidateTOTP(secret, code) {
		return ErrInvalidCredentials
	}
	return s.repo.SetTOTPEnabled(ctx, adminID, true)
}

// Tokens exposes the token manager (for middleware).
func (s *Service) Tokens() *TokenManager { return s.tokens }

// Repo exposes the admin repo (for handlers needing the current admin).
func (s *Service) AdminRepo() *Repo { return s.repo }
