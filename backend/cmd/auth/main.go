package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/security/audit"
	"github.com/kmuhub/kmuhub/internal/security/gdpr"
	"github.com/kmuhub/kmuhub/internal/security/password"
	"github.com/kmuhub/kmuhub/internal/security/vault"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/settings"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
	settingsv1 "github.com/kmuhub/kmuhub/proto/settings/v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load(ctx, config.RequireVault, config.RequireSystemSMTP)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := auth.NewPostgresRepository(pool)
	tokenMaker := auth.NewTokenMaker(cfg.JWTSecret, cfg.AccessTokenExpiry, cfg.RefreshTokenExpiry)
	authService := auth.NewService(repo, tokenMaker)

	// Wire password-reset mailer. In production SYSTEM_SMTP_* is required
	// (config.RequireSystemSMTP) so a missing relay fails fast at startup instead
	// of silently dropping reset mails; dev/CI falls back to logging.
	mailer := &systemMailer{
		host:     cfg.SystemSMTPHost,
		port:     cfg.SystemSMTPPort,
		username: cfg.SystemSMTPUser,
		password: cfg.SystemSMTPPassword,
		from:     cfg.SystemSMTPFrom,
	}
	authService.SetMailer(mailer)
	authService.SetResetBaseURL(cfg.PasswordResetBaseURL)

	// Security services
	var vaultService *vault.Service
	if cfg.VaultMasterSecret != "" {
		vaultRepo := vault.NewPostgresRepository(pool)
		var vaultErr error
		vaultService, vaultErr = vault.NewService(vaultRepo, cfg.VaultMasterSecret)
		if vaultErr != nil {
			slog.Error("failed to initialize vault service", "error", vaultErr)
			os.Exit(1)
		}
		// Wire vault into auth service for TOTP encryption
		authService.SetVaultService(&vaultAdapter{vaultService})
		slog.Info("vault service initialized, TOTP encryption enabled")
	} else {
		slog.Warn("VAULT_MASTER_SECRET not set, TOTP secrets stored unencrypted (dev mode)")
	}

	auditRepo := audit.NewPostgresRepository(pool)
	auditService := audit.NewService(auditRepo)

	passwordRepo := password.NewPostgresRepository(pool)
	passwordService := password.NewService(passwordRepo)
	// Wire password strength validator into auth service (used during reset).
	authService.SetPasswordValidator(passwordService)

	gdprRepo := gdpr.NewPostgresRepository(pool)
	gdprService := gdpr.NewService(gdprRepo)

	// Register default GDPR export and erasure handlers
	gdprService.RegisterExportHandler(gdpr.NewAuthExportHandler(pool))
	gdprService.RegisterExportHandler(gdpr.NewCRMExportHandler(pool))
	gdprService.RegisterExportHandler(gdpr.NewChatExportHandler(pool))
	gdprService.RegisterExportHandler(gdpr.NewWorkExportHandler(pool))
	gdprService.RegisterExportHandler(gdpr.NewCalendarExportHandler(pool))
	gdprService.RegisterExportHandler(gdpr.NewSessionExportHandler(pool))
	gdprService.RegisterExportHandler(gdpr.NewNotificationExportHandler(pool))
	gdprService.RegisterErasureHandler(gdpr.NewAuthErasureHandler(pool))
	gdprService.RegisterErasureHandler(gdpr.NewCRMErasureHandler(pool))
	gdprService.RegisterErasureHandler(gdpr.NewChatErasureHandler(pool))
	gdprService.RegisterErasureHandler(gdpr.NewWorkErasureHandler(pool))
	gdprService.RegisterErasureHandler(gdpr.NewCalendarErasureHandler(pool))
	gdprService.RegisterErasureHandler(gdpr.NewNotificationErasureHandler(pool))
	gdprService.RegisterErasureHandler(&gdpr.AuditErasureHandler{})

	// Metrics
	metricsRegistry := metrics.NewRegistry()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metricsRegistry.GRPCUnaryInterceptor(),
			middleware.TenantInboundUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)
	authGRPC := server.NewAuthGRPCServer(authService)
	authv1.RegisterAuthServiceServer(grpcServer, authGRPC)

	securityGRPC := server.NewSecurityGRPCServer(auditService, vaultService, gdprService, passwordService, pool)
	securityv1.RegisterSecurityServiceServer(grpcServer, securityGRPC)

	// Settings foundation (tenant/user settings + module leads) — co-located in auth process
	settingsRepo := settings.NewPostgresRepository(pool)
	settingsRoleChecker := settings.NewPostgresRoleChecker(pool)
	settingsSvc := settings.NewService(settingsRepo, settingsRoleChecker)
	settingsGRPC := server.NewSettingsGRPCServer(settingsSvc)
	settingsv1.RegisterSettingsServiceServer(grpcServer, settingsGRPC)

	// Initialize gRPC metrics after service registration
	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.AuthGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.AuthGRPCPort, "error", err)
		os.Exit(1)
	}

	// Health + metrics HTTP server
	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         cfg.HealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.HealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("auth service starting", "port", cfg.AuthGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down auth service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("auth service stopped")
}

// vaultAdapter wraps vault.Service to satisfy auth.VaultEncryptor interface
// which requires context parameters.
type vaultAdapter struct {
	svc *vault.Service
}

func (a *vaultAdapter) EncryptTOTPSecret(_ context.Context, plainSecret string) (string, error) {
	return a.svc.EncryptTOTPSecret(plainSecret)
}

func (a *vaultAdapter) DecryptTOTPSecret(_ context.Context, encryptedSecret string) (string, error) {
	return a.svc.DecryptTOTPSecret(encryptedSecret)
}
