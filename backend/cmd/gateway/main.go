package main

import (
	"context"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/cache"
	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/chat/guest"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/document/wopi"
	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/gateway"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/security"
	"github.com/kmuhub/kmuhub/internal/server"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load(ctx, config.RequireMinIO, config.RequireWOPI)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Feature-flag registry — resolved once at startup via env vars
	flagRegistry := featureflag.NewRegistry().Load(os.Getenv)
	slog.Info("feature flags loaded", "count", len(flagRegistry.All()))

	// Redis for rate limiting (best-effort)
	redisClient, err := database.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		slog.Warn("redis unavailable, using in-memory rate limiting", "error", err)
	}

	// =========================================================================
	// Service Registry: register backend services with lazy gRPC connections
	// =========================================================================
	tlsCfg, err := gateway.BuildClientTLSConfig(cfg.GRPCTLSCertFile, cfg.GRPCTLSKeyFile, cfg.GRPCTLSCAFile)
	if err != nil {
		slog.Error("failed to build gRPC TLS config", "error", err)
		os.Exit(1)
	}
	if tlsCfg != nil {
		slog.Info("gRPC mTLS enabled for service-to-service communication")
	}
	registry := gateway.NewServiceRegistry(tlsCfg)
	registry.SetGRPCTimeout(cfg.GRPCTimeout)
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
	registry.Register("wiki", cfg.WikiGRPCAddress)
	registry.Register("helpdesk", cfg.HelpdeskGRPCAddress)
	registry.Register("berichte", cfg.BerichteGRPCAddress)
	registry.Register("formulare", cfg.FormulareGRPCAddress)
	registry.Register("inventar", cfg.InventarGRPCAddress)
	registry.Register("einkauf", cfg.EinkaufGRPCAddress)
	registry.Register("produktion", cfg.ProduktionGRPCAddress)
	registry.Register("vertraege", cfg.VertraegeGRPCAddress)
	registry.Register("rapporte", cfg.RapporteGRPCAddress)
	registry.Register("schichten", cfg.SchichtenGRPCAddress)
	registry.Register("vermietung", cfg.VermietungGRPCAddress)
	registry.Register("fuhrpark", cfg.FuhrparkGRPCAddress)
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

	// Idempotency repository + cleanup worker (WarnMode: warns but never blocks)
	idempotencyRepo := idempotency.NewPostgresRepository(pool)
	go middleware.IdempotencyCleanupWorker(ctx, idempotencyRepo)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Metrics(metricsRegistry))
	r.Use(middleware.RequestID)
	r.Use(middleware.SecurityHeaders(cfg.BehindProxy))
	r.Use(middleware.Logging)
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	// Records the caller's IP/user agent in the context so the outbound gRPC
	// interceptor can forward them. Must sit above the route groups, not
	// inside the authenticated one: login is the call that creates the
	// session row those values describe.
	r.Use(middleware.ClientInfo(cfg.BehindProxy))

	// IP filter middleware: reject blocked IPs before rate limiting
	ipFilter := gateway.NewIPFilterMiddleware(registry, cfg.BehindProxy)
	r.Use(ipFilter.Middleware)

	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimitRPS, cfg.BehindProxy)
	r.Use(rateLimiter.Middleware)

	// Scoped rate limiter for unauthenticated public endpoints (/api/v1/public/*).
	// Uses a separate Redis key prefix ("ratelimit:public:") so public and global
	// counters are independent — a burst on booking pages does not exhaust global tokens.
	publicRateLimiter := middleware.NewRateLimiterWithPrefix(redisClient, cfg.PublicRateLimitRPS, "ratelimit:public", cfg.BehindProxy)

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

	// Idempotency deduplication middleware.
	// Default: WarnMode (logs missing header, does not block) — safe for production
	// until all clients reliably send the Idempotency-Key header.
	// Set IDEMPOTENCY_MODE=hard to enable HardMode (400 on missing key).
	// Dev compose sets this to "hard" by default for early regression detection.
	// Combined with authMiddleware so the chain is: Auth → Idempotency → Handler.
	// This guarantees tenant+user context is populated before idempotency reads it.
	idempotencyMode := middleware.WarnMode
	if os.Getenv("IDEMPOTENCY_MODE") == "hard" {
		idempotencyMode = middleware.HardMode
		slog.Info("idempotency running in HardMode — missing Idempotency-Key returns 400")
	}
	idempotencyMW := middleware.Idempotency(idempotencyRepo, idempotencyMode)
	authWithIdempotency := func(next http.Handler) http.Handler {
		return authMiddleware(idempotencyMW(next))
	}

	// Dashboard layout service (gateway-local, not gRPC)
	cacheClient := cache.NewClient(redisClient)
	dashboardService := gateway.NewDashboardStack(pool, cacheClient)

	// CRM extension routes (gRPC proxy: duplicates, timeline, consent)
	crmExt := gateway.NewCRMExtRoutes(registry)

	// Biz extension services (gRPC proxy: time-to-invoice)
	bizExt := gateway.NewBizExtRoutes(registry)

	// WOPI handler for OnlyOffice collaborative editing
	wopiTokenService := wopi.NewTokenService(cfg.WOPIJWTSecret)
	wopiLockService := wopi.NewLockService(pool)
	wopiFileAdapter := gateway.NewWOPIFileAdapter(registry)
	wopiHandler := wopi.NewHandler(wopiTokenService, wopiLockService, wopiFileAdapter, fileStore)

	isProd := strings.EqualFold(cfg.Env, "production")

	// videoRoutes is kept as a named variable so the WS hub can be injected
	// after hub construction (hub depends on chat gRPC client, built below).
	videoRoutes := gateway.NewVideoRoutes(registry, cfg.LiveKitAPIKey, cfg.LiveKitAPISecret)

	// Captcha verifier (provider-agnostic siteverify — Cloudflare Turnstile by default).
	// Disabled when CAPTCHA_SECRET is empty (default). No prod assertion: the hook
	// is opt-in; absence is acceptable until frontend widget is wired up.
	captchaVerifier := security.NewCaptchaVerifier(cfg.CaptchaSecret, cfg.CaptchaVerifyURL)
	if captchaVerifier.Enabled() {
		slog.Info("captcha verification enabled", "verify_url", cfg.CaptchaVerifyURL)
	}

	// Booking routes — admin (via registrars) + public (outside loop)
	bookingRoutes := gateway.NewBookingRoutes(registry, captchaVerifier)

	// Berichte routes — authenticated (via registrars) + the public read of a
	// shared report (outside loop, behind the strict public rate limiter)
	berichteRoutes := gateway.NewBerichteRoutes(registry, flagRegistry)

	// CRM routes — standard (via registrars) + advisory protocols (outside loop)
	crmRoutes := gateway.NewCRMRoutes(registry, crmExt)

	// Document routes — standard (via registrars) + the public redemption of a
	// share link (outside loop, behind the strict public rate limiter)
	documentRoutes := gateway.NewDocumentRoutes(registry)
	// Automation: authenticated CRUD via the registrar loop, the public
	// inbound webhook trigger outside it — same split as document/berichte.
	automationRoutes := gateway.NewAutomationRoutes(registry)
	// Helpdesk: authenticated routes via the registrar loop, the public
	// redemption of a CSAT survey link outside it — same split as berichte.
	helpdeskRoutes := gateway.NewHelpdeskRoutes(registry, flagRegistry)
	// Formulare: authenticated CRUD via the registrar loop, the public
	// share-link fill-out outside it -- same split as berichte/helpdesk.
	formulareRoutes := gateway.NewFormulareRoutes(registry, flagRegistry)
	// Wiki: authenticated CRUD via the registrar loop, the public read of one
	// shared article outside it -- same split as berichte/helpdesk/formulare.
	wikiRoutes := gateway.NewWikiRoutes(registry, flagRegistry)

	registrars := []gateway.RouteRegistrar{
		gateway.NewAuthRoutes(registry),
		crmRoutes,
		gateway.NewChatRoutes(registry),
		gateway.NewNotificationRoutes(registry),
		gateway.NewWorkRoutes(registry),
		gateway.NewCalendarRoutes(registry),
		videoRoutes,
		gateway.NewSecurityRoutes(registry),
		gateway.NewEmailRoutes(registry),
		documentRoutes,
		gateway.NewBizRoutes(registry),
		gateway.NewBexioRoutes(registry, cfg.BexioStateSecret),
		// DATEV upload shares the Bexio OAuth state secret: the signed state only
		// binds tenant + expiry, both flows are admin-only, and a second secret
		// would be a new required env var in production for no security gain.
		gateway.NewDatevUploadRoutes(registry, cfg.BexioStateSecret),
		gateway.NewLexwareRoutes(registry, cfg.LexwareWebhookSecret, isProd),
		// Teams/Slack integration admin config, channel mappings, account
		// linking and the inbound webhooks. The webhook routes tunnel the raw
		// request to the notification service (HandlePlatformWebhook): the
		// signature check needs the untouched bytes, the signing secret belongs
		// where the platform tokens already are, and the tenant has to be
		// resolved from the platform identity — none of which a gateway holding
		// a direct DB repo could do without bypassing the gRPC layer.
		gateway.NewIntegrationRoutes(registry),
		gateway.NewHRRoutes(registry, bizExt),
		gateway.NewInboxRoutes(registry),
		automationRoutes,
		gateway.NewDialerRoutes(registry),
		wikiRoutes,
		helpdeskRoutes,
		berichteRoutes,
		formulareRoutes,
		gateway.NewInventarRoutes(registry, flagRegistry),
		gateway.NewEinkaufRoutes(registry, flagRegistry),
		gateway.NewProduktionRoutes(registry, flagRegistry),
		gateway.NewVertraegeRoutes(registry, flagRegistry),
		gateway.NewRapporteRoutes(registry, flagRegistry),
		gateway.NewSchichtenRoutes(registry, flagRegistry),
		gateway.NewVermietungRoutes(registry, flagRegistry),
		gateway.NewFuhrparkRoutes(registry, flagRegistry),
		gateway.NewFileRoutes(registry),
		gateway.NewGlobalSearchRoutes(registry),
		gateway.NewDashboardRoutes(dashboardService),
		gateway.NewFeatureFlagRoutes(flagRegistry),
		gateway.NewHealthRoutes(healthCheckers, registry),
		bookingRoutes,
		gateway.NewSettingsRoutes(registry, flagRegistry),
		gateway.NewCustomizationRoutes(registry),
	}

	for _, reg := range registrars {
		reg.RegisterRoutes(r, authWithIdempotency)
		slog.Info("routes registered", "service", reg.ServiceName())
	}

	// Advisory-protocol routes (CRM, separate registration — ZFA Beratungsprotokoll)
	crmRoutes.RegisterAdvisoryRoutes(r, authWithIdempotency)
	slog.Info("routes registered", "service", "crm-advisory")

	// Plugin API routes — gated behind plugins.api flag (Phase D, off by default).
	// To enable in dev: COSMI_PLUGIN_API_ENABLED=true
	if flagRegistry.IsEnabled("plugins.api") {
		pluginRoutes := gateway.NewPluginRoutes(registry, flagRegistry)
		pluginRoutes.RegisterRoutes(r, authWithIdempotency)
		slog.Info("routes registered", "service", pluginRoutes.ServiceName())
	} else {
		slog.Info("plugin API routes disabled (plugins.api=false)")
	}

	// WOPI protocol routes (root-level, no standard auth -- uses WOPI access_token)
	wopiRoutes := gateway.NewWOPIRoutes(wopiHandler)
	wopiRoutes.RegisterRoutes(r, nil)
	slog.Info("routes registered", "service", "wopi")

	// CalDAV/CardDAV protocol routes (Basic Auth, not JWT)
	caldavRoutes := setupCalDAV(pool, registry, authMiddleware, "http://127.0.0.1"+cfg.GatewayHTTPPort)
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

	// Public booking routes (no auth middleware).
	// publicRateLimiter applies a strict per-IP limit (PUBLIC_RATE_LIMIT_RPS, default 10)
	// independent of the global limiter — prevents booking-spam and page scraping.
	bookingRoutes.RegisterPublicRoutes(r, publicRateLimiter.Middleware)
	slog.Info("routes registered", "service", "booking-public")

	// Public read of a shared report (no auth middleware). Same strict per-IP
	// limiter: the share token is the whole credential, so this route must not
	// sit behind the generous authenticated limit.
	berichteRoutes.RegisterPublicRoutes(r, publicRateLimiter.Middleware)
	slog.Info("routes registered", "service", "berichte-public")

	// Public redemption of a document share link (no auth middleware). Same
	// strict per-IP limiter, same reasoning as berichte-public.
	documentRoutes.RegisterPublicRoutes(r, publicRateLimiter.Middleware)
	slog.Info("routes registered", "service", "document-public")

	// Public inbound webhook trigger for automations (no auth middleware).
	// Same strict per-IP limiter: the per-automation HMAC secret is the real
	// gate, but per-IP throttling still bounds signature-guessing attempts.
	automationRoutes.RegisterPublicRoutes(r, publicRateLimiter.Middleware)
	slog.Info("routes registered", "service", "automation-public")

	// Public redemption of a CSAT survey link (no auth middleware). Same strict
	// per-IP limiter: the survey token is the whole credential, and this one
	// writes, so an unthrottled guessing run would also be a rating-spam run.
	helpdeskRoutes.RegisterPublicRoutes(r, publicRateLimiter.Middleware)
	slog.Info("routes registered", "service", "helpdesk-public")

	// Public fill-out of a shared form (no auth middleware). Same strict
	// per-IP limiter: the share token is the whole credential, and this one
	// writes, so an unthrottled guessing run would also be a submission-spam
	// run.
	formulareRoutes.RegisterPublicRoutes(r, publicRateLimiter.Middleware)
	slog.Info("routes registered", "service", "formulare-public")

	// Public read of one shared wiki article (no auth middleware). Same strict
	// per-IP limiter: the share token is the whole credential, and it grants
	// exactly one page — the limiter is what bounds a guessing run.
	wikiRoutes.RegisterPublicRoutes(r, publicRateLimiter.Middleware)
	slog.Info("routes registered", "service", "wiki-public")

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
	wsHub := setupWebSocketHub(registry, tokenMaker, cfg.CORSAllowedOrigins, redisClient)
	if wsHub != nil {
		wsHub.SetMetrics(metricsRegistry)
		// Wire guest service to WebSocket hub for guest token validation
		wsHub.SetGuestService(&guestServiceAdapter{svc: guestService})
		// Wire hub into video routes so recording.started events can be pushed
		// in real-time to meeting participants.
		videoRoutes.SetWSHub(wsHub)
		// Wire the Video service into the hub so call.accepted/call.declined
		// WebSocket signals are forwarded to the Video gRPC service.
		wsHub.SetVideoService(&videoWSAdapter{registry: registry, hub: wsHub})

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
