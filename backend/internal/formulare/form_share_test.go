package formulare

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// seedPublicSchema puts an active, public schema into the stub repository.
func seedPublicSchema(t *testing.T, repo *stubRepo, fields string) *FormSchema {
	t.Helper()
	s := &FormSchema{
		ID:        uuid.New(),
		TenantID:  testTenant,
		Title:     "Stoerungsmeldung",
		Fields:    []byte(fields),
		Status:    FormSchemaStatusActive,
		IsPublic:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	repo.schemas[s.ID] = s
	return s
}

func mintLink(t *testing.T, svc *Service, schemaID uuid.UUID) *FormShareLink {
	t.Helper()
	link, err := svc.CreateShareLink(context.Background(), CreateShareLinkInput{
		TenantID:     testTenant,
		FormSchemaID: schemaID,
	})
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}
	return link
}

func TestCreateShareLink_MintsAUsableToken(t *testing.T) {
	svc, repo := newSvc()
	schema := seedPublicSchema(t, repo, `[]`)

	link := mintLink(t, svc, schema.ID)

	// 32 bytes base64url without padding is 43 characters -- the same shape
	// the report share links and CSAT survey links use.
	if len(link.Token) != 43 {
		t.Fatalf("token length = %d, want 43 (32 bytes base64url)", len(link.Token))
	}
	if !link.Usable(time.Now().UTC()) {
		t.Fatal("a freshly minted link must be usable")
	}
	if got := link.Status(time.Now().UTC()); got != ShareLinkStatusActive {
		t.Fatalf("status = %q, want %q", got, ShareLinkStatusActive)
	}
}

func TestCreateShareLink_RefusesANonPublicSchema(t *testing.T) {
	svc, repo := newSvc()
	schema := seedPublicSchema(t, repo, `[]`)
	schema.IsPublic = false

	_, err := svc.CreateShareLink(context.Background(), CreateShareLinkInput{
		TenantID:     testTenant,
		FormSchemaID: schema.ID,
	})
	if !errors.Is(err, ErrSchemaNotPublic) {
		t.Fatalf("err = %v, want ErrSchemaNotPublic", err)
	}
}

func TestSubmitByShareToken_StoresTheSubmissionAndResolvesTheTenant(t *testing.T) {
	svc, repo := newSvc()
	schema := seedPublicSchema(t, repo, `[]`)
	link := mintLink(t, svc, schema.ID)

	got, err := svc.SubmitByShareToken(context.Background(), link.Token, []byte(`{"f1":"kaputt"}`), "")
	if err != nil {
		t.Fatalf("SubmitByShareToken: %v", err)
	}
	if got.TenantID != testTenant {
		t.Fatalf("tenant = %s, want %s (must come from the token, not the caller)", got.TenantID, testTenant)
	}
	if got.FormSchemaID != schema.ID {
		t.Fatalf("form_schema_id = %s, want %s", got.FormSchemaID, schema.ID)
	}
	if _, ok := repo.submissions[got.SubmissionID]; !ok {
		t.Fatal("submission was not stored")
	}
	if repo.shareLinks[link.ID].SubmissionCount != 1 {
		t.Fatalf("submission_count = %d, want 1", repo.shareLinks[link.ID].SubmissionCount)
	}
}

// The whole point of the single error: unknown, malformed, revoked, expired,
// used up and schema-closed must be indistinguishable from outside.
func TestSubmitByShareToken_EveryTokenVerdictIsTheSameNotFound(t *testing.T) {
	answers := []byte(`{"f1":"x"}`)
	past := time.Now().UTC().Add(-time.Hour)
	one := 1

	cases := []struct {
		name  string
		setup func(t *testing.T, svc *Service, repo *stubRepo) string
	}{
		{
			name: "unknown token",
			setup: func(_ *testing.T, _ *Service, _ *stubRepo) string {
				return "Zm9vYmFyZm9vYmFyZm9vYmFyZm9vYmFyZm9vYmFyMDA"
			},
		},
		{
			name:  "empty token",
			setup: func(_ *testing.T, _ *Service, _ *stubRepo) string { return "" },
		},
		{
			name: "over-long token",
			setup: func(_ *testing.T, _ *Service, _ *stubRepo) string {
				return string(make([]byte, 200))
			},
		},
		{
			name: "revoked",
			setup: func(t *testing.T, svc *Service, repo *stubRepo) string {
				schema := seedPublicSchema(t, repo, `[]`)
				link := mintLink(t, svc, schema.ID)
				if err := svc.RevokeShareLink(context.Background(), link.ID, testTenant); err != nil {
					t.Fatalf("RevokeShareLink: %v", err)
				}
				return link.Token
			},
		},
		{
			name: "expired",
			setup: func(t *testing.T, svc *Service, repo *stubRepo) string {
				schema := seedPublicSchema(t, repo, `[]`)
				link := mintLink(t, svc, schema.ID)
				repo.shareLinks[link.ID].ExpiresAt = &past
				return link.Token
			},
		},
		{
			name: "quota used up",
			setup: func(t *testing.T, svc *Service, repo *stubRepo) string {
				schema := seedPublicSchema(t, repo, `[]`)
				link := mintLink(t, svc, schema.ID)
				repo.shareLinks[link.ID].MaxSubmissions = &one
				repo.shareLinks[link.ID].SubmissionCount = 1
				return link.Token
			},
		},
		{
			name: "schema switched back to private",
			setup: func(t *testing.T, svc *Service, repo *stubRepo) string {
				schema := seedPublicSchema(t, repo, `[]`)
				link := mintLink(t, svc, schema.ID)
				schema.IsPublic = false
				return link.Token
			},
		},
		{
			name: "schema archived",
			setup: func(t *testing.T, svc *Service, repo *stubRepo) string {
				schema := seedPublicSchema(t, repo, `[]`)
				link := mintLink(t, svc, schema.ID)
				schema.Status = FormSchemaStatusArchived
				return link.Token
			},
		},
		{
			name: "schema deleted",
			setup: func(t *testing.T, svc *Service, repo *stubRepo) string {
				schema := seedPublicSchema(t, repo, `[]`)
				link := mintLink(t, svc, schema.ID)
				now := time.Now().UTC()
				schema.DeletedAt = &now
				return link.Token
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newSvc()
			token := tc.setup(t, svc, repo)

			_, err := svc.SubmitByShareToken(context.Background(), token, answers, "")
			if !errors.Is(err, ErrShareLinkNotFound) {
				t.Fatalf("err = %v, want ErrShareLinkNotFound", err)
			}
		})
	}
}

// The quota has to hold in the write, not only in the read: two visitors
// passing the advisory check at once must not both get through.
func TestSubmitByShareToken_QuotaIsEnforcedByTheWrite(t *testing.T) {
	svc, repo := newSvc()
	schema := seedPublicSchema(t, repo, `[]`)
	one := 1
	link, err := svc.CreateShareLink(context.Background(), CreateShareLinkInput{
		TenantID:       testTenant,
		FormSchemaID:   schema.ID,
		MaxSubmissions: &one,
	})
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	if _, err := svc.SubmitByShareToken(context.Background(), link.Token, []byte(`{"a":1}`), ""); err != nil {
		t.Fatalf("first submission: %v", err)
	}
	if _, err := svc.SubmitByShareToken(context.Background(), link.Token, []byte(`{"a":2}`), ""); !errors.Is(err, ErrShareLinkNotFound) {
		t.Fatalf("second submission err = %v, want ErrShareLinkNotFound", err)
	}
	if got := repo.shareLinks[link.ID].SubmissionCount; got != 1 {
		t.Fatalf("submission_count = %d, want 1", got)
	}
}

func TestSubmitByShareToken_RejectsFileFields(t *testing.T) {
	svc, repo := newSvc()
	schema := seedPublicSchema(t, repo, `[{"id":"f1","label":"Foto","type":"file"}]`)
	link := mintLink(t, svc, schema.ID)

	_, err := svc.SubmitByShareToken(context.Background(), link.Token, []byte(`{"f1":"x"}`), "")
	if !errors.Is(err, ErrShareLinkFileFieldUnsupported) {
		t.Fatalf("err = %v, want ErrShareLinkFileFieldUnsupported", err)
	}
	if len(repo.submissions) != 0 {
		t.Fatal("a rejected form must not store a submission")
	}
}

func TestSubmitByShareToken_RejectsAnOversizedBody(t *testing.T) {
	svc, repo := newSvc()
	schema := seedPublicSchema(t, repo, `[]`)
	link := mintLink(t, svc, schema.ID)

	_, err := svc.SubmitByShareToken(context.Background(), link.Token, make([]byte, maxPublicAnswersBytes+1), "")
	if !errors.Is(err, ErrAnswersTooLarge) {
		t.Fatalf("err = %v, want ErrAnswersTooLarge", err)
	}
}

// An oversized body must be refused before the token is ever looked up: the
// size is the caller's own error and must not buy a database round trip.
func TestSubmitByShareToken_SizeIsCheckedBeforeTheToken(t *testing.T) {
	svc, _ := newSvc()

	_, err := svc.SubmitByShareToken(context.Background(), "not-a-real-token", make([]byte, maxPublicAnswersBytes+1), "")
	if !errors.Is(err, ErrAnswersTooLarge) {
		t.Fatalf("err = %v, want ErrAnswersTooLarge (not the token verdict)", err)
	}
}

func TestRevokeShareLink_IsIdempotentAndKeepsTheFirstTimestamp(t *testing.T) {
	svc, repo := newSvc()
	schema := seedPublicSchema(t, repo, `[]`)
	link := mintLink(t, svc, schema.ID)

	if err := svc.RevokeShareLink(context.Background(), link.ID, testTenant); err != nil {
		t.Fatalf("first revoke: %v", err)
	}
	first := *repo.shareLinks[link.ID].RevokedAt

	if err := svc.RevokeShareLink(context.Background(), link.ID, testTenant); err != nil {
		t.Fatalf("second revoke: %v", err)
	}
	if !repo.shareLinks[link.ID].RevokedAt.Equal(first) {
		t.Fatal("re-revoking must not rewrite when the link was cut")
	}
	if got := repo.shareLinks[link.ID].Status(time.Now().UTC()); got != ShareLinkStatusDisabled {
		t.Fatalf("status = %q, want %q", got, ShareLinkStatusDisabled)
	}
}

func TestRevokeShareLink_RefusesAnotherTenant(t *testing.T) {
	svc, repo := newSvc()
	schema := seedPublicSchema(t, repo, `[]`)
	link := mintLink(t, svc, schema.ID)

	err := svc.RevokeShareLink(context.Background(), link.ID, uuid.New())
	if !errors.Is(err, ErrShareLinkNotFound) {
		t.Fatalf("err = %v, want ErrShareLinkNotFound", err)
	}
}
