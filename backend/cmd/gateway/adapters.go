package main

// adapters.go bridges between internal packages and gateway interfaces,
// breaking import cycles that would arise if the gateway package imported
// the chat/guest or caldav packages directly.
//
// All three adapters live in package main (cmd/gateway) because that is the
// only package that is allowed to import both sides of each cycle:
//   - guestServiceAdapter:     server ↔ chat/guest
//   - caldavPasswordAdapter:   gateway ↔ caldav
//   - caldavUserPrefAdapter:   gateway ↔ caldav

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	caldavpkg "github.com/kmuhub/kmuhub/internal/caldav"
	"github.com/kmuhub/kmuhub/internal/chat/guest"
	"github.com/kmuhub/kmuhub/internal/gateway"
	"github.com/kmuhub/kmuhub/internal/server"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// =========================================================================
// guestServiceAdapter adapts guest.Service to server.GuestSessionValidator
// =========================================================================

type guestServiceAdapter struct {
	svc *guest.Service
}

func (a *guestServiceAdapter) ValidateToken(ctx context.Context, token string) (*server.GuestSessionInfo, error) {
	session, err := a.svc.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return &server.GuestSessionInfo{
		ID:          session.ID,
		ChannelID:   session.ChannelID,
		DisplayName: session.DisplayName,
		IsActive:    session.IsActive,
	}, nil
}

func (a *guestServiceAdapter) CheckRateLimit(sessionID string) error {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	return a.svc.CheckRateLimit(id)
}

func (a *guestServiceAdapter) UpdateLastActivity(ctx context.Context, sessionID uuid.UUID) error {
	// ValidateToken already updates last activity
	return nil
}

// =========================================================================
// caldavPasswordAdapter adapts caldavpkg.AppPasswordService to
// gateway.CalDAVPasswordService
// =========================================================================

type caldavPasswordAdapter struct {
	svc *caldavpkg.AppPasswordService
}

func (a *caldavPasswordAdapter) Validate(ctx context.Context, username, password string) (uuid.UUID, error) {
	return a.svc.Validate(ctx, username, password)
}

func (a *caldavPasswordAdapter) Create(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID, label string) (string, *gateway.CalDAVPasswordInfo, error) {
	plaintext, pw, err := a.svc.Create(ctx, userID, tenantID, label)
	if err != nil {
		return "", nil, err
	}
	return plaintext, &gateway.CalDAVPasswordInfo{
		ID:             pw.ID,
		Label:          pw.Label,
		PasswordPrefix: pw.PasswordPrefix,
		LastUsedAt:     pw.LastUsedAt,
		CreatedAt:      pw.CreatedAt,
		RevokedAt:      pw.RevokedAt,
	}, nil
}

func (a *caldavPasswordAdapter) List(ctx context.Context, userID uuid.UUID) ([]*gateway.CalDAVPasswordInfo, error) {
	passwords, err := a.svc.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*gateway.CalDAVPasswordInfo, len(passwords))
	for i, pw := range passwords {
		result[i] = &gateway.CalDAVPasswordInfo{
			ID:             pw.ID,
			Label:          pw.Label,
			PasswordPrefix: pw.PasswordPrefix,
			LastUsedAt:     pw.LastUsedAt,
			CreatedAt:      pw.CreatedAt,
			RevokedAt:      pw.RevokedAt,
		}
	}
	return result, nil
}

func (a *caldavPasswordAdapter) Revoke(ctx context.Context, id, userID uuid.UUID) error {
	return a.svc.Revoke(ctx, id, userID)
}

func (a *caldavPasswordAdapter) IsOrgEnabled(ctx context.Context) (bool, error) {
	return a.svc.IsOrgEnabled(ctx)
}

func (a *caldavPasswordAdapter) SetOrgEnabled(ctx context.Context, enabled bool) error {
	return a.svc.SetOrgEnabled(ctx, enabled)
}

// =========================================================================
// caldavUserPrefAdapter adapts caldavpkg.PostgresUserPreferenceRepository to
// gateway.CalDAVUserPreferenceService
// =========================================================================

type caldavUserPrefAdapter struct {
	repo *caldavpkg.PostgresUserPreferenceRepository
}

func (a *caldavUserPrefAdapter) GetCalDAVEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	return a.repo.GetCalDAVEnabled(ctx, userID)
}

func (a *caldavUserPrefAdapter) SetCalDAVEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	return a.repo.SetCalDAVEnabled(ctx, userID, enabled)
}

func (a *caldavUserPrefAdapter) ListCalDAVUsers(ctx context.Context) ([]gateway.CalDAVUserInfo, error) {
	users, err := a.repo.ListCalDAVUsers(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]gateway.CalDAVUserInfo, len(users))
	for i, u := range users {
		result[i] = gateway.CalDAVUserInfo{
			ID:            u.ID,
			Email:         u.Email,
			FirstName:     u.FirstName,
			LastName:      u.LastName,
			CalDAVEnabled: u.CalDAVEnabled,
			PasswordCount: u.PasswordCount,
			LastUsed:      u.LastUsed,
		}
	}
	return result, nil
}

func (a *caldavUserPrefAdapter) RevokeAllUserPasswords(ctx context.Context, userID uuid.UUID) (int64, error) {
	return a.repo.RevokeAllUserPasswords(ctx, userID)
}

// =========================================================================
// videoWSAdapter adapts the Video gRPC service (+ WebSocket hub) to
// server.WSVideoService
// =========================================================================

// videoWSAdapter forwards WebSocket call-signaling events (accept/decline)
// to the Video gRPC service and, for declines, notifies the call initiator
// via the WebSocket hub. Video runs behind the "work" gRPC connection, same
// as gateway.VideoRoutes.
type videoWSAdapter struct {
	registry *gateway.ServiceRegistry
	hub      *server.WebSocketHub
}

// getClient lazily obtains a gRPC client for the Video service.
func (a *videoWSAdapter) getClient() (videov1.VideoServiceClient, error) {
	conn, err := a.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return videov1.NewVideoServiceClient(conn), nil
}

// NotifyCallAccepted acknowledges an accept signal. The actual join (LiveKit
// token issuance) happens via the existing REST endpoint
// (POST /api/v1/video/calls/{id}/join), so no further gRPC action is needed
// here — this is a logging no-op that satisfies server.WSVideoService.
func (a *videoWSAdapter) NotifyCallAccepted(ctx context.Context, callID, userID string) error {
	slog.Info("call accept acknowledged via websocket adapter", "call_id", callID, "user_id", userID)
	return nil
}

// NotifyCallDeclined ends the call (there is no dedicated DECLINED CallStatus
// enum; ENDED is reused) and notifies the initiator via the WebSocket hub so
// their client can dismiss the "ringing" UI in real time.
func (a *videoWSAdapter) NotifyCallDeclined(ctx context.Context, callID, userID string) error {
	client, err := a.getClient()
	if err != nil {
		return err
	}

	session, err := client.GetCall(ctx, &videov1.GetCallRequest{CallId: callID})
	if err != nil {
		return err
	}

	if _, err := client.EndCall(ctx, &videov1.EndCallRequest{CallId: callID, UserId: userID}); err != nil {
		return err
	}

	if a.hub != nil {
		a.hub.BroadcastCallEnded(ctx, session.GetInitiatorId(), map[string]interface{}{
			"call_id":     callID,
			"reason":      "declined",
			"declined_by": userID,
		})
	}

	return nil
}
