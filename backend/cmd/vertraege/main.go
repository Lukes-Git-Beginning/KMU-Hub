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
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/vertraege"
	vertraegev1 "github.com/kmuhub/kmuhub/proto/vertraege/v1"
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

	grpcPort := cfg.VertraegeGRPCPort
	healthPort := cfg.VertraegeHealthPort

	// Repository, service, and worker
	repo := vertraege.NewPostgresRepository(pool)
	svc := vertraege.NewService(repo)
	emitter := vertraege.NewPGEventEmitter(pool)
	worker := vertraege.NewReminderWorker(repo, emitter, logger)

	// Start reminder worker in background
	go func() {
		if runErr := worker.Run(ctx); runErr != nil {
			slog.Error("vertraege reminder worker exited with error", "error", runErr)
		}
	}()

	metricsRegistry := metrics.NewRegistry()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metricsRegistry.GRPCUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)

	vertraegeGRPC := server.NewVertraegeGRPCServer(svc)
	vertraegev1.RegisterVertraegeServiceServer(grpcServer, vertraegeGRPC)

	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		slog.Error("failed to listen", "port", grpcPort, "error", err)
		os.Exit(1)
	}

	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         healthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", healthPort)
		if serveErr := healthSrv.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", serveErr)
		}
	}()

	go func() {
		slog.Info("vertraege service starting", "port", grpcPort)
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			slog.Error("grpc server failed", "error", serveErr)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down vertraege service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if shutdownErr := healthSrv.Shutdown(shutdownCtx); shutdownErr != nil {
		slog.Error("health server shutdown failed", "error", shutdownErr)
	}
	slog.Info("vertraege service stopped")
}
