package wiki

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

const (
	// shareTokenBytes matches report_share_tokens (000252), form_share_tokens
	// (000293) and the CSAT survey links: 32 bytes of crypto/rand, base64url-
	// encoded to 43 URL-safe characters. The token IS the credential, so it is
	// drawn the same way everywhere.
	shareTokenBytes = 32

	// shareTokenMaxLength is the longest token string worth a database round
	// trip. Real tokens are 43 characters; anything past this is a probe, and
	// answering it should not cost a query.
	shareTokenMaxLength = 128

	// sharePermissionRead is the only permission a redeemed link ever grants.
	// The column exists since 000076 and defaults to {"read"}; a link minted
	// without it reads as "not a reading link" and is refused.
	sharePermissionRead = "read"
)

// newShareToken draws a link secret from crypto/rand. base64url keeps it usable
// as a URL path segment without escaping.
//
// Tokens minted before this existed were `uuid.New().String()`. Those stay
// redeemable — the lookup is an exact match on the column, not on a format —
// but every new link is drawn at full width.
func newShareToken() (string, error) {
	buf := make([]byte, shareTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate wiki share token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// withTenant attaches a tenant resolved from a share token to the context, so
// the repository and the RLS session GUCs see it. This is the single place a
// public path in this service may set a tenant, and it is only ever reached
// with the tenant the token itself resolved to.
func withTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
}

// Usable reports whether a share token may still be redeemed. Revocation is
// soft since 000297: the row survives with revoked_at set, so the check has to
// happen here rather than being implied by the row's absence. All three
// reasons answer the caller with the same 404 an invented token gets.
func (t *ShareToken) Usable(now time.Time) bool {
	if t.RevokedAt != nil {
		return false
	}
	if t.ExpiresAt != nil && !now.Before(*t.ExpiresAt) {
		return false
	}
	return t.grants(sharePermissionRead)
}

func (t *ShareToken) grants(permission string) bool {
	return slices.Contains(t.Permissions, permission)
}

// SharedArticle is the anonymous view of one shared page. Deliberately narrower
// than Article: a visitor holding a link has no business learning the tenant,
// the author, or which category the page sits in.
type SharedArticle struct {
	Title     string
	Content   []byte
	UpdatedAt time.Time
}

// RedeemShareToken serves the unauthenticated public read of one shared page.
//
// This is the one path in the service that begins without a tenant, so the
// order of the steps is the security property, not a matter of taste (same
// shape as formulare.Service.SubmitByShareToken and berichte GetSharedDocument):
//
//  1. resolve the token in the system context — the single read that must
//     escape RLS, because which tenant the caller may see is precisely what it
//     answers;
//  2. refuse expired tokens and tokens that grant no read as the same "not
//     found" an invented token gets;
//  3. re-enter tenant scope with the tenant the token itself named, and only
//     then read the one article it points at. The system context is never
//     carried past the lookup.
//
// The article is re-checked on every redemption: unpublishing a page must stop
// the links already handed out, the same way formulare re-checks `is_public`.
// A link resolves to exactly one page — never a tree of children, never a
// listing — so the reach of a leaked token is one document and stays one.
func (s *Service) RedeemShareToken(ctx context.Context, token string) (*SharedArticle, error) {
	// A shape check before the lookup: it is the caller's own error, says
	// nothing about which tokens exist, and must not buy a query.
	if token == "" || len(token) > shareTokenMaxLength {
		return nil, ErrShareTokenNotFound
	}

	share, err := s.repo.GetShareTokenByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if !share.Usable(now) {
		slog.Info("wiki share token refused",
			"token_id", share.ID,
			"revoked", share.RevokedAt != nil,
			"expired", share.ExpiresAt != nil && !now.Before(*share.ExpiresAt),
			"grants_read", share.grants(sharePermissionRead),
		)
		return nil, ErrShareTokenNotFound
	}

	ctx = withTenant(ctx, share.TenantID)

	article, err := s.repo.GetArticleByID(ctx, share.TenantID, share.ArticleID)
	if err != nil {
		// A token whose article is gone is a dead link, not a server fault.
		if errors.Is(err, ErrArticleNotFound) {
			return nil, ErrShareTokenNotFound
		}
		return nil, err
	}
	if !article.Published {
		// Same 404: a draft that was shared and then unpublished — or shared
		// while still a draft — must not be readable from outside, and the
		// route must not report which of the two happened.
		slog.Info("wiki share token refused by article state",
			"token_id", share.ID,
			"article_id", article.ID,
		)
		return nil, ErrShareTokenNotFound
	}

	slog.Info("wiki share token redeemed",
		"token_id", share.ID,
		"article_id", article.ID,
		"tenant_id", share.TenantID,
	)
	return &SharedArticle{
		Title:     article.Title,
		Content:   article.Content,
		UpdatedAt: article.UpdatedAt,
	}, nil
}
