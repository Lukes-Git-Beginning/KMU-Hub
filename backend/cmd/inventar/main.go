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

	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/inventar"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
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

	// Repository and service
	repo := inventar.NewPostgresRepository(pool)
	inventarService := inventar.NewService(repo)

	metricsRegistry := metrics.NewRegistry()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RecoveryUnaryInterceptor(),
			metricsRegistry.GRPCUnaryInterceptor(),
			middleware.TenantInboundUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			middleware.RecoveryStreamInterceptor(),
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)

	inventarGRPC := server.NewInventarGRPCServer(inventarService)
	inventarv1.RegisterInventarServiceServer(grpcServer, inventarGRPC)

	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.InventarGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.InventarGRPCPort, "error", err)
		os.Exit(1)
	}

	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         cfg.InventarHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.InventarHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("inventar service starting", "port", cfg.InventarGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down inventar service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("inventar service stopped")
}
