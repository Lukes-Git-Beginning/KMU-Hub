package berichte

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The share link is the only unauthenticated read into tenant data in this
// service, so these tests pin the refusals, not the happy path: what a link
// must NOT reach, and which of the ways it can be dead are distinguishable
// from outside. A green "it returns the document" alone would pass just as
// well with every guard removed.

func shareTestService(t *testing.T, now time.Time) (*Service, *mockRepository) {
	t.Helper()
	repo := newMockRepository()
	svc := NewService(repo, nil, Options{Clock: &fixedClock{now}})
	return svc, repo
}

func seedShareableDocument(repo *mockRepository, tenantID uuid.UUID) *Document {
	doc := &Document{
		ID:       uuid.New(),
		TenantID: tenantID,
		Title:    "Quartalsbericht",
		Module:   "cross",
		Status:   "released",
		Rows:     []byte(`[{"id":"r1"}]`),
		Settings: []byte(`{}`),
	}
	repo.documents[doc.ID] = doc
	return doc
}

func TestCreateShareToken_RefusesForeignDocument(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)

	owner := uuid.New()
	attacker := uuid.New()
	doc := seedShareableDocument(repo, owner)

	// The whole attack this guard exists for: name another tenant's document
	// and get back a working public link to it.
	_, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID:   attacker,
		DocumentID: doc.ID,
	})
	if err == nil {
		t.Fatal("minting a link for another tenant's document succeeded; the ownership read is missing")
	}
	if len(repo.shares) != 0 {
		t.Fatalf("a share row was written anyway: %d", len(repo.shares))
	}
}

func TestCreateShareToken_SecretIsFullEntropyAndUnique(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		token, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
			TenantID:   tenant,
			DocumentID: doc.ID,
		})
		if err != nil {
			t.Fatalf("create share token: %v", err)
		}
		raw, decErr := base64.RawURLEncoding.DecodeString(token.Token)
		if decErr != nil {
			t.Fatalf("token is not base64url: %v", decErr)
		}
		if len(raw) != shareTokenBytes {
			t.Fatalf("token carries %d bytes of entropy, want %d", len(raw), shareTokenBytes)
		}
		if seen[token.Token] {
			t.Fatal("two links got the same secret; the generator is not random")
		}
		seen[token.Token] = true

		// A link without a password must not silently get one, and an unset
		// expiry must mean "never", not "now".
		if token.PasswordHash != nil {
			t.Fatal("no password was requested but a hash was stored")
		}
		if token.ExpiresAt != nil {
			t.Fatalf("no expiry was requested but got %v", token.ExpiresAt)
		}
	}
}

func TestCreateShareToken_PasswordIsHashedNotStored(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	const password = "korrekt-pferd-batterie"
	token, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID:   tenant,
		DocumentID: doc.ID,
		Password:   password,
	})
	if err != nil {
		t.Fatalf("create share token: %v", err)
	}
	if token.PasswordHash == nil {
		t.Fatal("password was requested but no hash was stored")
	}
	if strings.Contains(*token.PasswordHash, password) {
		t.Fatal("the password itself is recoverable from the stored value")
	}
	if !strings.HasPrefix(*token.PasswordHash, "$2a$") {
		t.Fatalf("stored value is not a bcrypt hash: %q", *token.PasswordHash)
	}
}

func TestCreateShareToken_RejectsOversizedPasswordRatherThanTruncating(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	// bcrypt ignores input past 72 bytes. Accepting a longer password would
	// make two different passwords open the same link.
	_, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID:   tenant,
		DocumentID: doc.ID,
		Password:   strings.Repeat("a", maxSharePasswordLen+1),
	})
	if err != ErrSharePasswordTooLong {
		t.Fatalf("got %v, want ErrSharePasswordTooLong", err)
	}
}

func TestCreateShareToken_RejectsExpiryBeyondCap(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	for _, days := range []int32{-1, maxShareExpiryDays + 1} {
		d := days
		_, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
			TenantID:      tenant,
			DocumentID:    doc.ID,
			ExpiresInDays: &d,
		})
		if err != ErrInvalidShareExpiry {
			t.Fatalf("expires_in_days=%d: got %v, want ErrInvalidShareExpiry", days, err)
		}
	}
}

func TestGetSharedDocument_UnknownExpiredAndRevokedAreIndistinguishable(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	oneDay := int32(1)
	expired, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID: tenant, DocumentID: doc.ID, ExpiresInDays: &oneDay,
	})
	if err != nil {
		t.Fatalf("create expiring link: %v", err)
	}
	revoked, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID: tenant, DocumentID: doc.ID,
	})
	if err != nil {
		t.Fatalf("create link to revoke: %v", err)
	}
	if err := svc.RevokeShareToken(context.Background(), tenant, revoked.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	// Move past the expiry so the first link is dead too.
	svc.clock = &fixedClock{now.Add(48 * time.Hour)}

	cases := map[string]string{
		"never existed": "definitely-not-a-real-token",
		"empty":         "",
		"expired":       expired.Token,
		"revoked":       revoked.Token,
	}
	for name, secret := range cases {
		_, err := svc.GetSharedDocument(context.Background(), secret, "")
		if err != ErrShareNotFound {
			t.Fatalf("%s: got %v, want ErrShareNotFound — a distinct error tells an anonymous caller which link once existed", name, err)
		}
	}
}

func TestGetSharedDocument_ExpiryIsEnforcedAtTheBoundary(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	oneDay := int32(1)
	link, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID: tenant, DocumentID: doc.ID, ExpiresInDays: &oneDay,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// One second before the expiry the link still works ...
	svc.clock = &fixedClock{link.ExpiresAt.Add(-time.Second)}
	if _, err := svc.GetSharedDocument(context.Background(), link.Token, ""); err != nil {
		t.Fatalf("link died early: %v", err)
	}
	// ... and at the instant itself it does not. An off-by-one here is an
	// extra day of access nobody asked for.
	svc.clock = &fixedClock{*link.ExpiresAt}
	if _, err := svc.GetSharedDocument(context.Background(), link.Token, ""); err != ErrShareNotFound {
		t.Fatalf("link outlived its expiry: %v", err)
	}
}

func TestGetSharedDocument_PasswordGates(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	link, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID: tenant, DocumentID: doc.ID, Password: "geheim",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := svc.GetSharedDocument(context.Background(), link.Token, ""); err != ErrSharePasswordRequired {
		t.Fatalf("no password: got %v, want ErrSharePasswordRequired", err)
	}
	if _, err := svc.GetSharedDocument(context.Background(), link.Token, "falsch"); err != ErrSharePasswordInvalid {
		t.Fatalf("wrong password: got %v, want ErrSharePasswordInvalid", err)
	}
	if _, err := svc.GetSharedDocument(context.Background(), link.Token, "geheim"); err != nil {
		t.Fatalf("correct password rejected: %v", err)
	}
}

func TestGetSharedDocument_FailedAttemptsDoNotCountAsViews(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	link, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID: tenant, DocumentID: doc.ID, Password: "geheim",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, _ = svc.GetSharedDocument(context.Background(), link.Token, "")
	_, _ = svc.GetSharedDocument(context.Background(), link.Token, "falsch")
	if got := repo.shares[link.ID].ViewCount; got != 0 {
		t.Fatalf("refused attempts counted as %d views; the counter is what an admin reads to judge the link", got)
	}

	if _, err := svc.GetSharedDocument(context.Background(), link.Token, "geheim"); err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := repo.shares[link.ID].ViewCount; got != 1 {
		t.Fatalf("view_count = %d after one successful read, want 1", got)
	}
}

func TestGetSharedDocument_ReachesOnlyTheSharedDocument(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)

	tenant := uuid.New()
	shared := seedShareableDocument(repo, tenant)
	// A second document of the SAME tenant, never shared: resolving the tenant
	// must not turn into tenant-wide read access.
	sibling := seedShareableDocument(repo, tenant)
	// And one of another tenant entirely.
	foreign := seedShareableDocument(repo, uuid.New())

	link, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID: tenant, DocumentID: shared.ID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := svc.GetSharedDocument(context.Background(), link.Token, "")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.ID != shared.ID {
		t.Fatalf("got document %s, want the shared one %s", got.ID, shared.ID)
	}
	if got.ID == sibling.ID || got.ID == foreign.ID {
		t.Fatal("the public path returned a document the link does not name")
	}
}

func TestGetSharedDocument_DeadWhenTheDocumentIsGone(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	link, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID: tenant, DocumentID: doc.ID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	delete(repo.documents, doc.ID)

	// A link whose document was deleted is a dead link, not a 500.
	if _, err := svc.GetSharedDocument(context.Background(), link.Token, ""); err != ErrShareNotFound {
		t.Fatalf("got %v, want ErrShareNotFound", err)
	}
}

func TestListShareTokens_HidesRevokedAndForeignLinks(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	live, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{TenantID: tenant, DocumentID: doc.ID})
	if err != nil {
		t.Fatalf("create live: %v", err)
	}
	dead, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{TenantID: tenant, DocumentID: doc.ID})
	if err != nil {
		t.Fatalf("create dead: %v", err)
	}
	if err := svc.RevokeShareToken(context.Background(), tenant, dead.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	tokens, err := svc.ListShareTokens(context.Background(), tenant, doc.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tokens) != 1 || tokens[0].ID != live.ID {
		t.Fatalf("listed %d links, want only the live one", len(tokens))
	}

	// Another tenant asking about the same document id gets the document
	// refusal, not an empty list that would confirm the id exists.
	if _, err := svc.ListShareTokens(context.Background(), uuid.New(), doc.ID); err == nil {
		t.Fatal("listing another tenant's document shares succeeded")
	}
}

func TestRevokeShareToken_IsIdempotentlyRefusedAndTenantScoped(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	svc, repo := shareTestService(t, now)
	tenant := uuid.New()
	doc := seedShareableDocument(repo, tenant)

	link, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{TenantID: tenant, DocumentID: doc.ID})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Another tenant must not be able to cut a link it does not own — and the
	// link must still work afterwards.
	if err := svc.RevokeShareToken(context.Background(), uuid.New(), link.ID); err != ErrShareNotFound {
		t.Fatalf("foreign revoke: got %v, want ErrShareNotFound", err)
	}
	if _, err := svc.GetSharedDocument(context.Background(), link.Token, ""); err != nil {
		t.Fatalf("link died from a foreign revoke: %v", err)
	}

	if err := svc.RevokeShareToken(context.Background(), tenant, link.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if err := svc.RevokeShareToken(context.Background(), tenant, link.ID); err != ErrShareNotFound {
		t.Fatalf("second revoke: got %v, want ErrShareNotFound", err)
	}
}
