package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emersion/go-webdav/caldav"
	"github.com/emersion/go-webdav/carddav"

	"github.com/kmuhub/kmuhub/internal/auth"
	caldavpkg "github.com/kmuhub/kmuhub/internal/caldav"
	"github.com/kmuhub/kmuhub/internal/gateway"
	"github.com/kmuhub/kmuhub/internal/server"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

// notificationDeliveryPayload is the JSON payload sent via pg_notify('notification_delivery', ...).
type notificationDeliveryPayload struct {
	UserID       string          `json:"user_id"`
	Notification json.RawMessage `json:"notification"`
	DesktopPush  bool            `json:"desktop_push"`
	Sound        string          `json:"sound"`
}

// setupWebSocketHub creates the WebSocket hub using lazy connections from the registry.
// Returns nil if either chat or auth service connections cannot be obtained.
func setupWebSocketHub(registry *gateway.ServiceRegistry, tokenMaker *auth.TokenMaker, allowedOrigins []string) *server.WebSocketHub {
	chatConn, err := registry.GetConnection("chat")
	if err != nil {
		slog.Warn("chat connection unavailable for websocket hub", "error", err)
		return nil
	}
	chatClient := chatv1.NewChatServiceClient(chatConn)

	authConn, err := registry.GetConnection("auth")
	if err != nil {
		slog.Warn("auth connection unavailable for websocket user lookup", "error", err)
		return nil
	}
	authClient := authv1.NewAuthServiceClient(authConn)

	return server.NewWebSocketHub(chatClient, tokenMaker, func(ctx context.Context, userID string) (string, string, error) {
		resp, err := authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
		if err != nil {
			return "", "", err
		}
		return resp.User.FirstName, resp.User.LastName, nil
	}, allowedOrigins)
}

// startNotificationDeliveryListener listens on the PostgreSQL notification_delivery channel
// and pushes real-time notifications to connected WebSocket clients.
func startNotificationDeliveryListener(ctx context.Context, databaseURL string, wsHub *server.WebSocketHub) {
	for {
		if ctx.Err() != nil {
			return
		}

		if err := listenNotificationDelivery(ctx, databaseURL, wsHub); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("notification delivery listener failed, reconnecting", "error", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func listenNotificationDelivery(ctx context.Context, databaseURL string, wsHub *server.WebSocketHub) error {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "LISTEN notification_delivery"); err != nil {
		return err
	}

	slog.Info("notification delivery listener started")

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}

		var payload notificationDeliveryPayload
		if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
			slog.Error("failed to parse notification delivery payload",
				"error", err,
				"payload", notification.Payload,
			)
			continue
		}

		wsHub.SendNotificationToUser(ctx, payload.UserID, payload.Notification, payload.DesktopPush, payload.Sound)
	}
}

// setupCalDAV constructs all CalDAV/CardDAV services, backends, handlers, and adapters,
// and returns a CalDAVRoutes ready to be registered on the router.
func setupCalDAV(
	pool *pgxpool.Pool,
	registry *gateway.ServiceRegistry,
	authMiddleware func(http.Handler) http.Handler,
) *gateway.CalDAVRoutes {
	pushSubService := caldavpkg.NewPushSubscriptionService(pool)
	pushNotifier := caldavpkg.NewPushNotifier(pushSubService)
	caldavSyncService := caldavpkg.NewSyncTokenService(pool, pushNotifier)
	caldavAppPwService := caldavpkg.NewAppPasswordService(
		caldavpkg.NewPostgresAppPasswordRepository(pool), pool,
	)
	caldavBackend := caldavpkg.NewCalDAVBackend(registry, caldavSyncService, pool)
	carddavBackend := caldavpkg.NewCardDAVBackend(registry, caldavSyncService, pool)

	caldavHandler := &caldav.Handler{Backend: caldavBackend, Prefix: "/caldav"}
	carddavHandler := &carddav.Handler{Backend: carddavBackend, Prefix: "/carddav"}

	caldavPwAdapter := &caldavPasswordAdapter{svc: caldavAppPwService}
	caldavUserPrefRepo := caldavpkg.NewPostgresUserPreferenceRepository(pool)
	caldavUserPrefAdapter := &caldavUserPrefAdapter{repo: caldavUserPrefRepo}

	return gateway.NewCalDAVRoutes(
		caldavHandler, carddavHandler,
		caldavPwAdapter, caldavUserPrefAdapter,
		caldavpkg.CtxWithUser, authMiddleware,
	)
}
