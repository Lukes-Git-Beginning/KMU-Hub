package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/document/wopi"
	"github.com/kmuhub/kmuhub/internal/gateway"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load(ctx)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Redis for rate limiting (best-effort)
	redisClient, err := database.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		slog.Warn("redis unavailable, using in-memory rate limiting", "error", err)
	}

	// =========================================================================
	// Service Registry: register backend services with lazy gRPC connections
	// =========================================================================
	registry := gateway.NewServiceRegistry()
	registry.Register("auth", cfg.AuthGRPCAddress)
	registry.Register("crm", cfg.CRMGRPCAddress)
	registry.Register("chat", cfg.ChatGRPCAddress)
	registry.Register("notification", cfg.NotificationGRPCAddress)
	registry.Register("work", cfg.WorkGRPCAddress)
	registry.Register("email", cfg.EmailGRPCAddress)
	registry.Register("document", cfg.DocumentGRPCAddress)
	registry.Register("biz", cfg.BizGRPCAddress)
	defer registry.Close()

	// Database for file upload handler (direct access, not via gRPC)
	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Initialize MinIO file store for upload handler
	fileStore, err := file.NewMinIOStore(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		slog.Error("failed to connect to minio", "error", err)
		os.Exit(1)
	}

	fileRepo := file.NewPostgresRepository(pool)
	fileScanner := &file.NoOpScanner{}
	thumbnailGen := file.NewImagingGenerator()
	fileService := file.NewService(fileRepo, fileStore, fileScanner, thumbnailGen, cfg.FileSizeLimitMB)

	// Token maker for local token validation in middleware
	tokenMaker := auth.NewTokenMaker(cfg.JWTSecret, cfg.AccessTokenExpiry, cfg.RefreshTokenExpiry)
	// Minimal auth service for token validation only (no DB needed)
	localAuthService := auth.NewService(nil, tokenMaker)

	// Metrics
	metricsRegistry := metrics.NewRegistry()

	// Health checkers
	healthCheckers := []health.Checker{
		health.NewRedisChecker(redisClient),
	}

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Metrics(metricsRegistry))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	// IP filter middleware: reject blocked IPs before rate limiting
	ipFilter := gateway.NewIPFilterMiddleware(registry)
	r.Use(ipFilter.Middleware)

	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimitRPS)
	r.Use(rateLimiter.Middleware)

	// =========================================================================
	// Register per-service route handlers via RouteRegistrar interface
	// =========================================================================
	authMiddleware := middleware.Auth(localAuthService)

	// Dashboard layout service (direct DB access, not gRPC)
	dashboardRepo := gateway.NewPostgresDashboardRepository(pool)
	dashboardService := gateway.NewDashboardService(dashboardRepo)

	// WOPI handler for OnlyOffice collaborative editing
	wopiTokenService := wopi.NewTokenService(cfg.WOPIJWTSecret)
	wopiLockService := wopi.NewLockService(pool)
	wopiFileAdapter := gateway.NewWOPIFileAdapter(registry)
	wopiHandler := wopi.NewHandler(wopiTokenService, wopiLockService, wopiFileAdapter, fileStore)

	registrars := []gateway.RouteRegistrar{
		gateway.NewAuthRoutes(registry),
		gateway.NewCRMRoutes(registry),
		gateway.NewChatRoutes(registry),
		gateway.NewNotificationRoutes(registry),
		gateway.NewWorkRoutes(registry),
		gateway.NewCalendarRoutes(registry),
		gateway.NewVideoRoutes(registry),
		gateway.NewSecurityRoutes(registry),
		gateway.NewEmailRoutes(registry),
		gateway.NewDocumentRoutes(registry),
		gateway.NewBizRoutes(registry),
		gateway.NewHRRoutes(registry),
		gateway.NewGlobalSearchRoutes(registry),
		gateway.NewDashboardRoutes(dashboardService),
		gateway.NewHealthRoutes(healthCheckers, registry),
	}

	for _, reg := range registrars {
		reg.RegisterRoutes(r, authMiddleware)
		slog.Info("routes registered", "service", reg.ServiceName())
	}

	// WOPI protocol routes (root-level, no standard auth -- uses WOPI access_token)
	wopiRoutes := gateway.NewWOPIRoutes(wopiHandler)
	wopiRoutes.RegisterRoutes(r, nil)
	slog.Info("routes registered", "service", "wopi")

	// =========================================================================
	// WebSocket hub (cross-cutting: needs chat + auth gRPC clients)
	// =========================================================================
	wsHub := setupWebSocketHub(registry, tokenMaker)
	if wsHub != nil {
		r.Get("/api/v1/ws", wsHub.HandleWebSocket)

		// File upload handler (multipart/form-data, not gRPC)
		fileUploadHandler := server.NewFileUploadHandler(fileService, wsHub)
		r.With(authMiddleware).
			With(middleware.RequirePermission("files", "write")).
			Post("/api/v1/files/upload", fileUploadHandler.HandleUploadFile)
	} else {
		slog.Warn("websocket hub not available, /api/v1/ws and file upload disabled")
	}

	// =========================================================================
	// Notification delivery listener (PostgreSQL LISTEN on notification_delivery)
	// Receives signals from the notification service when a notification is stored,
	// and pushes real-time WebSocket messages to connected users.
	// =========================================================================
	if wsHub != nil {
		go startNotificationDeliveryListener(ctx, cfg.DatabaseURL, wsHub)
	}

	// HTTP server
	srv := &http.Server{
		Addr:         cfg.GatewayHTTPPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Metrics server (separate port)
	metricsRouter := chi.NewRouter()
	metricsRouter.Handle("/metrics", metricsRegistry.Handler())
	metricsSrv := &http.Server{
		Addr:         cfg.MetricsPort,
		Handler:      metricsRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("metrics server starting", "port", cfg.MetricsPort)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("gateway starting", "port", cfg.GatewayHTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down gateway")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("metrics server shutdown failed", "error", err)
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown failed", "error", err)
	}
	slog.Info("gateway stopped")
}

// notificationDeliveryPayload is the JSON payload sent via pg_notify('notification_delivery', ...).
type notificationDeliveryPayload struct {
	UserID       string          `json:"user_id"`
	Notification json.RawMessage `json:"notification"`
	DesktopPush  bool            `json:"desktop_push"`
	Sound        string          `json:"sound"`
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

// setupWebSocketHub creates the WebSocket hub using lazy connections from the registry.
// Returns nil if either chat or auth service connections cannot be obtained.
func setupWebSocketHub(registry *gateway.ServiceRegistry, tokenMaker *auth.TokenMaker) *server.WebSocketHub {
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
	})
}
