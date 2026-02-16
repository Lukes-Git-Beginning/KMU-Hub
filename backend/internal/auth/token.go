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
	Roles       []string `json:"roles"`
	Permissions []string `json:"perms"`
}

func NewTokenMaker(secret string, accessExpiry, refreshExpiry time.Duration) *TokenMaker {
	return &TokenMaker{
		secret:        []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (tm *TokenMaker) CreateAccessToken(userID uuid.UUID, roles, permissions []string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "kmuhub",
		},
		UserID:      userID.String(),
		Roles:       roles,
		Permissions: permissions,
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
