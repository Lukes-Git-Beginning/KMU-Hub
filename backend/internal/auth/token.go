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
