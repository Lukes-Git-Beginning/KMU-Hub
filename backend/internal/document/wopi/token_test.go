package wopi

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenService_GenerateAndValidate_Roundtrip(t *testing.T) {
	svc := NewTokenService("test-secret-abc123")

	tokenString, ttlMs, err := svc.Generate("file-1", "user-1", "Alice", "tenant-1", true)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if ttlMs != int64(wopiTokenTTL/time.Millisecond) {
		t.Errorf("ttlMs = %d, want %d", ttlMs, int64(wopiTokenTTL/time.Millisecond))
	}

	claims, err := svc.Validate(tokenString)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if claims.FileID != "file-1" || claims.UserID != "user-1" || claims.UserName != "Alice" ||
		claims.TenantID != "tenant-1" || !claims.CanWrite {
		t.Errorf("claims = %+v, want file-1/user-1/Alice/tenant-1/true", claims)
	}
}

func TestTokenService_Validate_RejectsMalformedToken(t *testing.T) {
	svc := NewTokenService("test-secret-abc123")

	if _, err := svc.Validate("not-a-jwt"); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestTokenService_Validate_RejectsTamperedSignature(t *testing.T) {
	generator := NewTokenService("secret-a")
	validator := NewTokenService("secret-b")

	tokenString, _, err := generator.Generate("file-1", "user-1", "Alice", "tenant-1", true)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if _, err := validator.Validate(tokenString); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken (different secret must be rejected)", err)
	}
}

func TestTokenService_Validate_RejectsExpiredToken(t *testing.T) {
	svc := NewTokenService("test-secret-abc123")

	now := time.Now()
	claims := WOPITokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * wopiTokenTTL)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Minute)),
			Issuer:    "kmuhub-wopi",
		},
		FileID:   "file-1",
		UserID:   "user-1",
		UserName: "Alice",
		CanWrite: true,
		TenantID: "tenant-1",
	}
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(svc.secret)
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	if _, err := svc.Validate(tokenString); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken (expired token must be rejected)", err)
	}
}

// TestTokenService_Validate_RejectsNoneAlgorithm guards against the classic
// "alg=none" JWT forgery: a token whose header claims no signature is
// required at all. validateAccessToken hands out full write access to a
// document on a valid token, so accepting an unsigned token here would let
// anyone forge access to any file_id/tenant_id/can_write combination.
func TestTokenService_Validate_RejectsNoneAlgorithm(t *testing.T) {
	svc := NewTokenService("test-secret-abc123")

	claims := WOPITokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(wopiTokenTTL)),
			Issuer:    "kmuhub-wopi",
		},
		FileID:   "file-1",
		UserID:   "attacker",
		UserName: "Attacker",
		CanWrite: true,
		TenantID: "tenant-1",
	}
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none-alg token: %v", err)
	}

	if _, err := svc.Validate(tokenString); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken (alg=none forgery must be rejected)", err)
	}
}
