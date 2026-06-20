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

	"github.com/kmuhub/kmuhub/internal/berichte"
	"github.com/kmuhub/kmuhub/internal/berichte/executor"
	"github.com/kmuhub/kmuhub/internal/berichte/export"
	"github.com/kmuhub/kmuhub/internal/berichte/scheduler"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	berichtev1 "github.com/kmuhub/kmuhub/proto/berichte/v1"
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

	// Repository, executor, and service
	repo := berichte.NewPostgresRepository(pool)
	exec := executor.New(executor.Deps{}) // downstream repos nil in Sprint 1; added per module in Sprint 2+
	svc := berichte.NewService(repo, exec, berichte.Options{})

	exporterFactory := server.BerichteExporterFactory(func(format string) (server.BerichteExporter, error) {
		return export.NewExporter(format)
	})

	// Scheduler (exporter=nil, mailer=nil → skipped until WP-3/SMTP wired)
	sched := scheduler.New(repo, svc, nil, nil, scheduler.Config{})

	schedCtx, schedCancel := context.WithCancel(ctx)
	go func() {
		if err := sched.Run(schedCtx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("berichte scheduler stopped unexpectedly", "error", err)
		}
	}()
	defer schedCancel()

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

	berichteGRPC := server.NewBerichteGRPCServer(svc, exporterFactory)
	berichtev1.RegisterBerichteServiceServer(grpcServer, berichteGRPC)

	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.BerichteGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.BerichteGRPCPort, "error", err)
		os.Exit(1)
	}

	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         cfg.BerichteHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.BerichteHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("berichte service starting", "port", cfg.BerichteGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down berichte service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("berichte service stopped")
}
