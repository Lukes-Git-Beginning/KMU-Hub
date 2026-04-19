package main

import (
	"context"
	"errors"
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
	"github.com/kmuhub/kmuhub/internal/formulare"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/server"
	formularev1 "github.com/kmuhub/kmuhub/proto/formulare/v1"
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

	repo := formulare.NewPostgresRepository(pool)
	svc := formulare.NewService(repo, logger)

	// Webhook delivery worker — runs as a background goroutine.
	worker := formulare.NewWebhookWorker(repo, logger)
	workerCtx, workerCancel := context.WithCancel(ctx)
	go func() {
		if err := worker.Run(workerCtx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("formulare webhook worker stopped unexpectedly", "error", err)
		}
	}()
	defer workerCancel()

	metricsRegistry := metrics.NewRegistry()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metricsRegistry.GRPCUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)

	formulareGRPC := server.NewFormulareGRPCServer(svc)
	formularev1.RegisterFormulareServiceServer(grpcServer, formulareGRPC)

	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.FormulareGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.FormulareGRPCPort, "error", err)
		os.Exit(1)
	}

	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	healthRouter.Get("/healthz", server.HealthHandler(healthCheckers))
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         cfg.FormulareHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("formulare health/metrics server starting", "port", cfg.FormulareHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("formulare health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("formulare service starting", "port", cfg.FormulareGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("formulare grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down formulare service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("formulare health server shutdown failed", "error", err)
	}
	slog.Info("formulare service stopped")
}
