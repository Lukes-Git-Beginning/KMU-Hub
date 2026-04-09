package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/emersion/go-webdav/caldav"
	"github.com/emersion/go-webdav/carddav"

	"github.com/kmuhub/kmuhub/internal/cache"
	"github.com/kmuhub/kmuhub/internal/auth"
	caldavpkg "github.com/kmuhub/kmuhub/internal/caldav"
	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/chat/guest"
	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
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
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
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
	registry.Register("automation", cfg.AutomationGRPCAddress)
	registry.Register("plugin", cfg.PluginGRPCAddress)
	registry.Register("dialer", cfg.DialerGRPCAddress)
	defer registry.Close()

	// Database for file upload handler (direct access, not via gRPC)
	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Initialize MinIO file store for upload handler (best-effort)
	var fileStore file.FileStore
	minioStore, err := file.NewMinIOStore(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		slog.Warn("minio unavailable, file upload and WOPI disabled", "error", err)
		fileStore = file.NewUnavailableStore()
	} else {
		fileStore = minioStore
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
	r.Use(middleware.SecurityHeaders(cfg.BehindProxy))
	r.Use(middleware.Logging)
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	// IP filter middleware: reject blocked IPs before rate limiting
	ipFilter := gateway.NewIPFilterMiddleware(registry)
	r.Use(ipFilter.Middleware)

	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimitRPS)
	r.Use(rateLimiter.Middleware)

	// Audit logger: logs security-relevant events via gRPC (worker pool)
	auditLogger := middleware.NewAuditLogger(func() (securityv1.SecurityServiceClient, error) {
		conn, err := registry.GetConnection("auth")
		if err != nil {
			return nil, err
		}
		return securityv1.NewSecurityServiceClient(conn), nil
	})
	auditLogger.Start(10)
	defer auditLogger.Close()
	r.Use(auditLogger.Middleware)

	// pprof profiling (only in non-production)
	if os.Getenv("ENABLE_PPROF") == "true" {
		r.Mount("/debug/pprof", http.DefaultServeMux)
		slog.Info("pprof profiling enabled on /debug/pprof")
	}

	// =========================================================================
	// Register per-service route handlers via RouteRegistrar interface
	// =========================================================================
	authMiddleware := middleware.Auth(localAuthService)

	// Dashboard layout service (direct DB access, not gRPC)
	cacheClient := cache.NewClient(redisClient)
	dashboardRepo := gateway.NewCachedDashboardRepository(
		gateway.NewPostgresDashboardRepository(pool), cacheClient,
	)
	dashboardService := gateway.NewDashboardService(dashboardRepo)

	// CRM extension services (direct DB access: duplicates, timeline, consent)
	crmExt := gateway.NewCRMExtRoutes(pool)

	// Biz extension services (direct DB access: time-to-invoice)
	bizExt := gateway.NewBizExtRoutes(pool)

	// WOPI handler for OnlyOffice collaborative editing
	wopiTokenService := wopi.NewTokenService(cfg.WOPIJWTSecret)
	wopiLockService := wopi.NewLockService(pool)
	wopiFileAdapter := gateway.NewWOPIFileAdapter(registry)
	wopiHandler := wopi.NewHandler(wopiTokenService, wopiLockService, wopiFileAdapter, fileStore)

	registrars := []gateway.RouteRegistrar{
		gateway.NewAuthRoutes(registry),
		gateway.NewCRMRoutes(registry, crmExt),
		gateway.NewChatRoutes(registry),
		gateway.NewNotificationRoutes(registry),
		gateway.NewWorkRoutes(registry),
		gateway.NewCalendarRoutes(registry),
		gateway.NewVideoRoutes(registry),
		gateway.NewSecurityRoutes(registry),
		gateway.NewEmailRoutes(registry),
		gateway.NewDocumentRoutes(registry),
		gateway.NewBizRoutes(registry),
		gateway.NewBexioRoutes(registry),
		gateway.NewHRRoutes(registry, bizExt),
		gateway.NewInboxRoutes(registry),
		gateway.NewAutomationRoutes(registry),
		gateway.NewPluginRoutes(registry),
		gateway.NewDialerRoutes(registry),
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

	// CalDAV/CardDAV protocol routes (Basic Auth, not JWT)
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
	caldavRoutes := gateway.NewCalDAVRoutes(
		caldavHandler, carddavHandler,
		caldavPwAdapter, caldavpkg.CtxWithUser,
		pool, authMiddleware,
	)
	caldavRoutes.RegisterRoutes(r)
	slog.Info("routes registered", "service", "caldav")

	// =========================================================================
	// Guest Chat Support
	// =========================================================================
	guestRepo := guest.NewPostgresRepository(pool)
	guestService := guest.NewService(guestRepo)

	// Guest routes (public, no auth middleware)
	guestRoutes := gateway.NewGuestRoutes(guestService, registry)
	guestRoutes.RegisterPublicRoutes(r)
	slog.Info("routes registered", "service", "guest")

	// Guest inbox adapter
	guestAdapter := adapter.NewGuestAdapter(pool)
	_ = guestAdapter // registered below if adapter registry exists

	// =========================================================================
	// Guest Chat SPA static files
	// =========================================================================
	guestChatDir := filepath.Join(".", "guest-chat", "dist")
	if _, err := os.Stat(guestChatDir); err == nil {
		fileServer := http.FileServer(http.Dir(guestChatDir))
		// Serve static assets (JS, CSS, images) directly
		r.Handle("/guest/assets/*", http.StripPrefix("/guest/", fileServer))
		// SPA fallback: any /guest/* path serves index.html
		r.Get("/guest/*", func(w http.ResponseWriter, req *http.Request) {
			http.ServeFile(w, req, filepath.Join(guestChatDir, "index.html"))
		})
		slog.Info("guest chat SPA enabled", "dir", guestChatDir)
	} else {
		slog.Info("guest chat SPA not found, /guest/* disabled", "dir", guestChatDir)
	}

	// =========================================================================
	// WebSocket hub (cross-cutting: needs chat + auth gRPC clients)
	// =========================================================================
	wsHub := setupWebSocketHub(registry, tokenMaker, cfg.CORSAllowedOrigins)
	if wsHub != nil {
		wsHub.SetMetrics(metricsRegistry)
		// Wire guest service to WebSocket hub for guest token validation
		wsHub.SetGuestService(&guestServiceAdapter{svc: guestService})

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

// guestServiceAdapter adapts guest.Service to server.GuestSessionValidator,
// breaking the import cycle between server and guest packages.
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

// caldavPasswordAdapter adapts caldavpkg.AppPasswordService to gateway.CalDAVPasswordService,
// breaking the import cycle between gateway and caldav packages.
type caldavPasswordAdapter struct {
	svc *caldavpkg.AppPasswordService
}

func (a *caldavPasswordAdapter) Validate(ctx context.Context, username, password string) (uuid.UUID, error) {
	return a.svc.Validate(ctx, username, password)
}

func (a *caldavPasswordAdapter) Create(ctx context.Context, userID uuid.UUID, label string) (string, *gateway.CalDAVPasswordInfo, error) {
	plaintext, pw, err := a.svc.Create(ctx, userID, label)
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
