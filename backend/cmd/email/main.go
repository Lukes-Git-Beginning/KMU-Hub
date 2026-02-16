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
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
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

	// TODO(10-05): Initialize email repositories
	// accountRepo := account.NewPostgresRepository(pool)
	// messageRepo := message.NewPostgresRepository(pool)
	// folderRepo  := ...
	// signatureRepo := ...

	// TODO(10-05): Initialize email services
	// accountService := account.NewService(accountRepo, vaultEncryptor)
	// messageService := message.NewService(messageRepo)
	// sendService := send.NewService(accountService, messageRepo)
	// signatureService := signature.NewService(signatureRepo)
	// attachmentService := attachment.NewService(minioClient)

	// TODO(10-05): Start sync engine
	// syncEngine := sync.NewEngine(accountService, messageService, attachmentService)
	// go syncEngine.Start(ctx)

	// Metrics
	metricsRegistry := metrics.NewRegistry()

	// gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metricsRegistry.GRPCUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)

	// Register stub EmailServiceServer (returns Unimplemented for all RPCs)
	emailv1.RegisterEmailServiceServer(grpcServer, &emailv1.UnimplementedEmailServiceServer{})

	// Initialize gRPC metrics after service registration
	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.EmailGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.EmailGRPCPort, "error", err)
		os.Exit(1)
	}

	// Health + metrics HTTP server
	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	healthRouter.Get("/health", server.HealthHandler(healthCheckers))
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         cfg.EmailHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.EmailHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("email service starting", "port", cfg.EmailGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down email service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("email service stopped")
}
