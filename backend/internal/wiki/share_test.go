package wiki

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// These tests cover the public redemption path. What they are really guarding
// is that every refusal — expired, no read permission, unknown, malformed,
// article gone, article unpublished — comes back as the SAME error. A route
// that answers differently per cause is an oracle for which tokens exist, and
// that is the defect worth a test, not the happy path.

func addShareToken(repo *mockRepository, article *Article, expiresAt *time.Time, permissions ...string) *ShareToken {
	if len(permissions) == 0 {
		permissions = []string{sharePermissionRead}
	}
	t := &ShareToken{
		ID:          uuid.New(),
		TenantID:    article.TenantID,
		ArticleID:   article.ID,
		Token:       uuid.New().String(),
		ExpiresAt:   expiresAt,
		Permissions: permissions,
		CreatedAt:   time.Now().UTC(),
	}
	repo.tokens[t.Token] = t
	return t
}

func TestService_RedeemShareToken_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	article := addArticle(repo, uuid.New(), "Handbuch", "handbuch", true)
	article.Content = []byte(`{"type":"doc","content":[]}`)
	share := addShareToken(repo, article, nil)

	got, err := svc.RedeemShareToken(context.Background(), share.Token)

	require.NoError(t, err)
	assert.Equal(t, "Handbuch", got.Title)
	assert.JSONEq(t, `{"type":"doc","content":[]}`, string(got.Content))
	assert.Equal(t, article.UpdatedAt, got.UpdatedAt)
}

// The anonymous view must not carry what identifies where the page lives. This
// is a type-level guarantee (SharedArticle has three fields), and the test
// exists so that widening it back to *Article is a deliberate act.
func TestService_RedeemShareToken_LeaksNoTenantOrAuthor(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	article := addArticle(repo, uuid.New(), "Handbuch", "handbuch", true)
	share := addShareToken(repo, article, nil)

	got, err := svc.RedeemShareToken(context.Background(), share.Token)

	require.NoError(t, err)
	assert.NotContains(t, string(got.Content), article.TenantID.String())
	assert.NotContains(t, string(got.Content), article.AuthorID.String())
}

// The token names one article, and nothing about the request can widen that:
// a second article of the same tenant stays out of reach.
func TestService_RedeemShareToken_ReachesExactlyOneArticle(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	shared := addArticle(repo, tenantID, "Geteilt", "geteilt", true)
	sibling := addArticle(repo, tenantID, "Nicht geteilt", "nicht-geteilt", true)
	share := addShareToken(repo, shared, nil)

	got, err := svc.RedeemShareToken(context.Background(), share.Token)

	require.NoError(t, err)
	assert.Equal(t, shared.Title, got.Title)
	assert.NotEqual(t, sibling.Title, got.Title)
}

// The tenant comes from the token, never from the caller: a context carrying
// somebody else's tenant must not redirect the read.
func TestService_RedeemShareToken_TenantComesFromToken(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	article := addArticle(repo, uuid.New(), "Handbuch", "handbuch", true)
	share := addShareToken(repo, article, nil)

	foreign := context.WithValue(context.Background(), middleware.TenantIDKey, uuid.New().String())
	got, err := svc.RedeemShareToken(foreign, share.Token)

	require.NoError(t, err)
	assert.Equal(t, "Handbuch", got.Title)
}

func TestService_RedeemShareToken_RefusalsAreIndistinguishable(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour)

	tests := []struct {
		name  string
		token func(repo *mockRepository) string
	}{
		{
			name:  "unknown token",
			token: func(*mockRepository) string { return uuid.New().String() },
		},
		{
			name:  "empty token",
			token: func(*mockRepository) string { return "" },
		},
		{
			name: "malformed, over-long token",
			token: func(*mockRepository) string {
				return string(make([]byte, shareTokenMaxLength+1))
			},
		},
		{
			name: "expired token",
			token: func(repo *mockRepository) string {
				a := addArticle(repo, uuid.New(), "Handbuch", "handbuch", true)
				return addShareToken(repo, a, &past).Token
			},
		},
		{
			name: "revoked token (row is gone)",
			token: func(repo *mockRepository) string {
				a := addArticle(repo, uuid.New(), "Handbuch", "handbuch", true)
				share := addShareToken(repo, a, nil)
				delete(repo.tokens, share.Token)
				return share.Token
			},
		},
		{
			name: "token granting no read",
			token: func(repo *mockRepository) string {
				a := addArticle(repo, uuid.New(), "Handbuch", "handbuch", true)
				return addShareToken(repo, a, nil, "comment").Token
			},
		},
		{
			name: "article unpublished",
			token: func(repo *mockRepository) string {
				a := addArticle(repo, uuid.New(), "Entwurf", "entwurf", false)
				return addShareToken(repo, a, nil).Token
			},
		},
		{
			name: "article deleted",
			token: func(repo *mockRepository) string {
				a := addArticle(repo, uuid.New(), "Handbuch", "handbuch", true)
				share := addShareToken(repo, a, nil)
				delete(repo.articles, a.ID)
				return share.Token
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo)

			_, err := svc.RedeemShareToken(context.Background(), tc.token(repo))

			require.ErrorIs(t, err, ErrShareTokenNotFound)
		})
	}
}

// Minted tokens are 32 bytes of crypto/rand, not a UUID. Tokens created before
// that change stay redeemable — the lookup matches the column, not a format —
// which is why RedeemShareToken above is exercised with uuid-shaped tokens.
func TestService_CreateShareToken_TokenEntropy(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	article := addArticle(repo, uuid.New(), "Handbuch", "handbuch", true)

	first, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID:  article.TenantID,
		ArticleID: article.ID,
	})
	require.NoError(t, err)
	second, err := svc.CreateShareToken(context.Background(), CreateShareTokenInput{
		TenantID:  article.TenantID,
		ArticleID: article.ID,
	})
	require.NoError(t, err)

	raw, decodeErr := base64.RawURLEncoding.DecodeString(first.Token)
	require.NoError(t, decodeErr, "token must be base64url without padding")
	assert.Len(t, raw, shareTokenBytes)
	assert.NotEqual(t, first.Token, second.Token)
	assert.Equal(t, []string{sharePermissionRead}, first.Permissions)
}
