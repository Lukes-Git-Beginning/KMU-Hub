package linksecret

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	b, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if a == b {
		t.Fatal("two secrets are equal")
	}
	raw, err := base64.RawURLEncoding.DecodeString(a)
	if err != nil {
		t.Fatalf("secret is not base64url: %v", err)
	}
	if len(raw) != tokenBytes || len(a) != 43 {
		t.Fatalf("secret has %d bytes / %d chars, want %d / 43", len(raw), len(a), tokenBytes)
	}
}

func TestHash(t *testing.T) {
	// Known SHA-256 of "abc" -- the value Postgres' sha256() gives as well,
	// which the backfill in the migrations relies on.
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := Hash("abc"); got != want {
		t.Fatalf("Hash(abc) = %s, want %s", got, want)
	}
	if Hash("abc") == Hash("abd") {
		t.Fatal("different secrets hash equal")
	}
}

func TestExpiry(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	limit := now.Add(MaxLifetime)

	got, err := Expiry(nil, now)
	if err != nil || !got.Equal(limit) {
		t.Fatalf("no request: got %v, %v; want %v", got, err, limit)
	}

	week := now.AddDate(0, 0, 7)
	if got, err := Expiry(&week, now); err != nil || !got.Equal(week) {
		t.Fatalf("a week: got %v, %v; want %v", got, err, week)
	}
	if got, err := Expiry(&limit, now); err != nil || !got.Equal(limit) {
		t.Fatalf("exactly the cap: got %v, %v; want %v", got, err, limit)
	}

	tooLong := now.AddDate(0, 0, 400)
	if _, err := Expiry(&tooLong, now); !errors.Is(err, ErrLifetimeTooLong) {
		t.Fatalf("400 days: got %v, want ErrLifetimeTooLong", err)
	}
}
