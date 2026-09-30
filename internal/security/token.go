package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/config"
)

// TokenPair is the JWT pair returned on login/registration.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Claims carries the identity of an authenticated user.
type Claims struct {
	UserID  uuid.UUID `json:"uid"`
	Role    string    `json:"role"`
	OrgID   uuid.UUID `json:"org_id,omitempty"`
	StoreID uuid.UUID `json:"store_id,omitempty"`
	Type    string    `json:"typ"`
	MFA     bool      `json:"mfa,omitempty"`
	jwt.RegisteredClaims
}

// Manager issues and verifies JWT access and refresh tokens.
type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

// NewManager builds a JWT manager from config.
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		accessSecret:  []byte(cfg.JWTAccessSecret),
		refreshSecret: []byte(cfg.JWTRefreshSecret),
		accessTTL:     cfg.JWTAccessTTL,
		refreshTTL:    cfg.JWTRefreshTTL,
	}
}

// Issue creates a fresh access+refresh token pair.
func (m *Manager) Issue(userID uuid.UUID, role string, orgID, storeID uuid.UUID) (*TokenPair, error) {
	return m.issue(userID, role, orgID, storeID, false)
}

// IssueMFA creates a token pair indicating that the admin completed TOTP.
func (m *Manager) IssueMFA(userID uuid.UUID, role string, orgID, storeID uuid.UUID) (*TokenPair, error) {
	return m.issue(userID, role, orgID, storeID, true)
}

func (m *Manager) issue(userID uuid.UUID, role string, orgID, storeID uuid.UUID, mfa bool) (*TokenPair, error) {
	now := time.Now()

	access, err := m.sign(m.accessSecret, userID, role, orgID, storeID, "access", mfa, now, m.accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := m.sign(m.refreshSecret, userID, role, orgID, storeID, "refresh", mfa, now, m.refreshTTL)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(m.accessTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// IssueMFAChallenge issues a short-lived, narrowly scoped token that can only
// be exchanged for a login session after the authenticator code is verified.
func (m *Manager) IssueMFAChallenge(userID uuid.UUID) (string, error) {
	now := time.Now()
	return m.sign(m.accessSecret, userID, "admin", uuid.Nil, uuid.Nil, "mfa_challenge", false, now, 5*time.Minute)
}

// ParseMFAChallenge validates an authenticator login challenge token.
func (m *Manager) ParseMFAChallenge(tokenString string) (*Claims, error) {
	return m.parse(tokenString, m.accessSecret, "mfa_challenge")
}

func (m *Manager) sign(secret []byte, userID uuid.UUID, role string, orgID, storeID uuid.UUID, typ string, mfa bool, now time.Time, ttl time.Duration) (string, error) {
	claims := &Claims{
		UserID:  userID,
		Role:    role,
		OrgID:   orgID,
		StoreID: storeID,
		Type:    typ,
		MFA:     mfa,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseAccess validates an access token and returns its claims.
func (m *Manager) ParseAccess(tokenString string) (*Claims, error) {
	return m.parse(tokenString, m.accessSecret, "access")
}

// ParseRefresh validates a refresh token and returns its claims.
func (m *Manager) ParseRefresh(tokenString string) (*Claims, error) {
	return m.parse(tokenString, m.refreshSecret, "refresh")
}

func (m *Manager) parse(tokenString string, secret []byte, expectedType string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("security: unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("security: invalid token")
	}
	if claims.Type != expectedType {
		return nil, errors.New("security: wrong token type")
	}
	return claims, nil
}
