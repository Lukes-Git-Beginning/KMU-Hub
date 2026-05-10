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
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
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

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := auth.NewPostgresRepository(pool)
	tokenMaker := auth.NewTokenMaker(cfg.JWTSecret, cfg.AccessTokenExpiry, cfg.RefreshTokenExpiry)
	authService := auth.NewService(repo, tokenMaker)

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

	gdprRepo := gdpr.NewPostgresRepository(pool)
	gdprService := gdpr.NewService(gdprRepo)

	// Register default GDPR export and erasure handlers
	gdprService.RegisterExportHandler(&gdpr.AuthExportHandler{})
	gdprService.RegisterExportHandler(&gdpr.CRMExportHandler{})
	gdprService.RegisterExportHandler(&gdpr.ChatExportHandler{})
	gdprService.RegisterExportHandler(&gdpr.WorkExportHandler{})
	gdprService.RegisterExportHandler(&gdpr.CalendarExportHandler{})
	gdprService.RegisterExportHandler(&gdpr.SessionExportHandler{})
	gdprService.RegisterExportHandler(&gdpr.NotificationExportHandler{})
	gdprService.RegisterErasureHandler(&gdpr.AuthErasureHandler{})
	gdprService.RegisterErasureHandler(&gdpr.CRMErasureHandler{})
	gdprService.RegisterErasureHandler(&gdpr.ChatErasureHandler{})
	gdprService.RegisterErasureHandler(&gdpr.WorkErasureHandler{})
	gdprService.RegisterErasureHandler(&gdpr.CalendarErasureHandler{})
	gdprService.RegisterErasureHandler(&gdpr.NotificationErasureHandler{})
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
