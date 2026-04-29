package wopi

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// WOPI token TTL per WOPI spec recommendation (10 hours).
const wopiTokenTTL = 10 * time.Hour

// Errors for WOPI token operations.
var (
	ErrInvalidToken = errors.New("invalid or expired WOPI token")
	ErrTokenMissing = errors.New("access_token query parameter is required")
)

// WOPITokenClaims contains the claims embedded in a WOPI JWT.
type WOPITokenClaims struct {
	jwt.RegisteredClaims
	FileID   string `json:"file_id"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	CanWrite bool   `json:"can_write"`
	TenantID string `json:"tenant_id"`
}

// TokenService generates and validates WOPI-specific JWTs.
// Uses a SEPARATE secret from the main application JWT.
type TokenService struct {
	secret []byte
}

// NewTokenService creates a new WOPI token service.
// The secret must be from the WOPI_JWT_SECRET env var, NOT the app JWT secret.
func NewTokenService(secret string) *TokenService {
	return &TokenService{
		secret: []byte(secret),
	}
}

// Generate creates a new WOPI JWT token for the given file and user.
// tenantID is encoded in the token so the WOPI handler can scope file lookups.
// Returns the token string and TTL in milliseconds.
func (s *TokenService) Generate(fileID, userID, userName, tenantID string, canWrite bool) (string, int64, error) {
	now := time.Now()
	claims := WOPITokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(wopiTokenTTL)),
			Issuer:    "kmuhub-wopi",
		},
		FileID:   fileID,
		UserID:   userID,
		UserName: userName,
		CanWrite: canWrite,
		TenantID: tenantID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", 0, err
	}

	ttlMs := int64(wopiTokenTTL / time.Millisecond)
	return tokenString, ttlMs, nil
}

// Validate parses and validates a WOPI JWT token.
func (s *TokenService) Validate(tokenString string) (*WOPITokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &WOPITokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*WOPITokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
