package integration

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Webhook kinds distinguish the inbound endpoints of one platform. The gateway
// sets them from the route it served; the processor switches on them instead of
// sniffing the payload.
const (
	WebhookKindSlackInteraction = "slack_interaction"
	WebhookKindSlackCommand     = "slack_command"
	WebhookKindTeamsActivity    = "teams_activity"
)

// WebhookRequest is an inbound platform request as it arrived at the gateway.
//
// Body stays raw bytes all the way from the socket to the signature check:
// Slack signs the exact request body, so parsing and re-encoding it anywhere in
// between — as the pre-tunnel handler did with url.Values.Encode() — reorders
// the fields and makes every multi-field payload fail verification.
type WebhookRequest struct {
	Kind    string
	Body    []byte
	Headers map[string]string
}

// WebhookResponse is written back to the platform verbatim by the gateway.
type WebhookResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

// WebhookProcessor handles one platform's inbound webhooks. It lives in the
// notification service, never in the gateway: it needs the signing secret, the
// integration repository and a tenant, none of which belong in a proxy.
type WebhookProcessor interface {
	Process(ctx context.Context, req WebhookRequest) WebhookResponse
}

// TenantResolver maps a platform workspace identity to a Cosmi tenant.
//
// Inbound webhooks are unauthenticated, so the tenant cannot come from a JWT.
// It is resolved from the identity the platform itself asserts (Slack team_id,
// Teams channelData.tenant.id) against the workspace id an admin stored in
// integration_configs.metadata. An unresolvable workspace is refused, never
// guessed — guessing would hand one customer's Slack workspace another
// customer's data.
type TenantResolver interface {
	ResolveTenant(ctx context.Context, platform, workspaceID string) (uuid.UUID, error)
}

// NotificationAcknowledger marks a notification read on behalf of the linked
// user. The signature matches notification.Service.MarkRead so the service
// satisfies it directly.
type NotificationAcknowledger interface {
	MarkRead(ctx context.Context, tenantID, notifID, userID uuid.UUID) (*models.Notification, error)
}

// WorkspaceMetadataKey returns the integration_configs.metadata key that holds
// the platform's workspace identifier. Each platform names it differently and
// an admin types it in as free JSON, so the two names are pinned here rather
// than repeated at every call site.
func WorkspaceMetadataKey(platform string) string {
	switch platform {
	case PlatformSlack:
		return "team_id" // Slack workspace id, e.g. T0123ABCD
	case PlatformTeams:
		return "tenant_id" // Azure AD tenant id of the Teams organisation
	default:
		return ""
	}
}

// WithTenant attaches a resolved tenant to the context so the repository and
// the RLS session GUCs see it. This is the single place a webhook path may set
// a tenant, and it is only ever reached with a value the platform's own
// identity resolved to.
func WithTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
}

// TextResponse builds a plain-text webhook reply.
func TextResponse(statusCode int, text string) WebhookResponse {
	return WebhookResponse{
		StatusCode:  statusCode,
		ContentType: "text/plain; charset=utf-8",
		Body:        []byte(text),
	}
}

// JSONResponse builds a JSON webhook reply. A payload that cannot be marshalled
// is a programming error, not a platform error, so it degrades to 500 rather
// than shipping a truncated body.
func JSONResponse(statusCode int, payload any) WebhookResponse {
	body, err := json.Marshal(payload)
	if err != nil {
		return TextResponse(http.StatusInternalServerError, "internal error")
	}
	return WebhookResponse{
		StatusCode:  statusCode,
		ContentType: "application/json",
		Body:        body,
	}
}

// EmptyResponse acknowledges a webhook without a body.
func EmptyResponse(statusCode int) WebhookResponse {
	return WebhookResponse{StatusCode: statusCode}
}
