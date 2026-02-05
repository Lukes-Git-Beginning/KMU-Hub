package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
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

	// gRPC connection to auth service
	authConn, err := grpc.NewClient(
		cfg.AuthGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("failed to connect to auth service", "error", err)
		os.Exit(1)
	}
	defer func() { _ = authConn.Close() }()

	authClient := authv1.NewAuthServiceClient(authConn)

	// gRPC connection to CRM service
	crmConn, err := grpc.NewClient(
		cfg.CRMGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("failed to connect to crm service", "error", err)
		os.Exit(1)
	}
	defer func() { _ = crmConn.Close() }()

	crmClient := crmv1.NewCRMServiceClient(crmConn)

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

	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimitRPS)
	r.Use(rateLimiter.Middleware)

	// Register routes
	handler := server.NewGatewayHandler(authClient, crmClient, healthCheckers)
	handler.RegisterRoutes(r, middleware.Auth(localAuthService))

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
