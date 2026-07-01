package clients

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// Service holds client use-cases.
type Service struct {
	repo *Repo
}

// NewService builds the client service.
func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// Repo exposes the repository.
func (s *Service) Repo() *Repo { return s.repo }

// Create generates identity (uuid, password, subscription token) and stores a
// new, enabled client.
func (s *Service) Create(ctx context.Context, name, remark string) (*Client, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("name is required")
	}
	password, err := randomToken(16)
	if err != nil {
		return nil, err
	}
	token, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, &Client{
		Name:              name,
		UUID:              uuid.NewString(),
		Password:          password,
		SubscriptionToken: token,
		Enabled:           true,
		Remark:            remark,
	})
}

// Update changes name/remark.
func (s *Service) Update(ctx context.Context, id int64, name, remark string) (*Client, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("name is required")
	}
	if err := s.repo.UpdateInfo(ctx, id, name, remark); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

// SetEnabled enables/disables a client.
func (s *Service) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.SetEnabled(ctx, id, enabled)
}

// SetInbounds replaces the client's access grants.
func (s *Service) SetInbounds(ctx context.Context, id int64, inboundIDs []int64) (*Client, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, err
	}
	if err := s.repo.SetInbounds(ctx, id, inboundIDs); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

// RotateToken issues a fresh subscription token.
func (s *Service) RotateToken(ctx context.Context, id int64) (*Client, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, err
	}
	token, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	if err := s.repo.RotateToken(ctx, id, token); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

// randomToken returns a URL-safe base64 token of n random bytes.
func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
