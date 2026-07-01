package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Token stages. A "2fa" token is a short-lived pending token issued after a
// correct password when TOTP is required; an "auth" token is a full session.
const (
	StagePending = "2fa"
	StageAuth    = "auth"
)

const (
	pendingTTL = 5 * time.Minute
	sessionTTL = 12 * time.Hour
)

// Claims are the JWT claims carried in panel tokens.
type Claims struct {
	AdminID int64  `json:"admin_id"`
	Stage   string `json:"stage"`
	jwt.RegisteredClaims
}

// TokenManager issues and verifies JWTs.
type TokenManager struct {
	secret []byte
	now    func() time.Time
}

// NewTokenManager builds a TokenManager with the given signing secret.
func NewTokenManager(secret []byte) *TokenManager {
	return &TokenManager{secret: secret, now: time.Now}
}

// IssueSession issues a full session token for an admin.
func (m *TokenManager) IssueSession(adminID int64) (string, error) {
	return m.issue(adminID, StageAuth, sessionTTL)
}

// IssuePending issues a short-lived token that only permits completing 2FA.
func (m *TokenManager) IssuePending(adminID int64) (string, error) {
	return m.issue(adminID, StagePending, pendingTTL)
}

func (m *TokenManager) issue(adminID int64, stage string, ttl time.Duration) (string, error) {
	now := m.now()
	claims := Claims{
		AdminID: adminID,
		Stage:   stage,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(m.secret)
}

// Parse validates a token and returns its claims.
func (m *TokenManager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}
