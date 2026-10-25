package linksecret

import (
	"encoding/base64"
	"testing"
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
