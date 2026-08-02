package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenMaker struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

type Claims struct {
	jwt.RegisteredClaims
	UserID      string   `json:"uid"`
	TenantID    string   `json:"tid"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"perms"`

	// Scopes carries only the capability keys that reach less far than the
	// whole tenant, as key -> "own" | "team" (see Service.NarrowedScopes).
	// A key that is absent — and every key in a token minted before this claim
	// existed — reaches "all".
	Scopes map[string]string `json:"scopes,omitempty"`
}

func NewTokenMaker(secret string, accessExpiry, refreshExpiry time.Duration) *TokenMaker {
	return &TokenMaker{
		secret:        []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// CreateAccessToken creates a signed JWT containing the user identity, tenant, roles and permissions.
// tenantID must be a non-empty UUID string; an empty string results in a legacy token that will be
// rejected by GetTenantID in the middleware (fail-closed).
// scopes may be nil, which means every permission in the token reaches "all".
func (tm *TokenMaker) CreateAccessToken(userID uuid.UUID, tenantID string, roles, permissions []string, scopes map[string]string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "kmuhub",
		},
		UserID:      userID.String(),
		TenantID:    tenantID,
		Roles:       roles,
		Permissions: permissions,
		Scopes:      scopes,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.secret)
}

func (tm *TokenMaker) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (tm *TokenMaker) CreateRefreshToken() (plain string, hash string, expiresAt time.Time) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	plain = hex.EncodeToString(b)
	hash = HashToken(plain)
	expiresAt = time.Now().Add(tm.refreshExpiry)
	return
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

const pendingTokenExpiry = 5 * time.Minute

// PendingClaims represents a short-lived token issued when 2FA verification is required.
type PendingClaims struct {
	jwt.RegisteredClaims
	UserID    string `json:"uid"`
	TokenType string `json:"type"`
}

// CreatePendingToken creates a 5-minute JWT for the 2FA verification step.
func (tm *TokenMaker) CreatePendingToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := PendingClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(pendingTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "kmuhub",
		},
		UserID:    userID.String(),
		TokenType: "2fa_pending",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.secret)
}

// ValidatePendingToken validates a 2FA pending token and returns the user ID.
func (tm *TokenMaker) ValidatePendingToken(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &PendingClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return uuid.Nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.secret, nil
	})
	if err != nil {
		return uuid.Nil, ErrInvalidPendingToken
	}

	claims, ok := token.Claims.(*PendingClaims)
	if !ok || !token.Valid {
		return uuid.Nil, ErrInvalidPendingToken
	}

	if claims.TokenType != "2fa_pending" {
		return uuid.Nil, ErrInvalidPendingToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, ErrInvalidPendingToken
	}

	return userID, nil
}
