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
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/produktion"
	"github.com/kmuhub/kmuhub/internal/server"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
	produktionv1 "github.com/kmuhub/kmuhub/proto/produktion/v1"
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
	repo := produktion.NewPostgresRepository(pool)
	produktionService := produktion.NewService(repo)

	// Wire inventar gRPC client for the material availability check. The
	// connection is established lazily; failure to connect is non-fatal
	// since the inventar lookup is best-effort (graceful degradation).
	inventarConn, inventarConnErr := grpc.NewClient(
		cfg.InventarGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if inventarConnErr != nil {
		slog.Warn("failed to create inventar gRPC connection; material availability check disabled",
			"address", cfg.InventarGRPCAddress, "error", inventarConnErr)
	} else {
		inventarClient := inventarv1.NewInventarServiceClient(inventarConn)
		produktionService.WithInventarLookup(produktion.NewGRPCInventarLookup(inventarClient))
		defer inventarConn.Close()
		slog.Info("inventar gRPC client connected", "address", cfg.InventarGRPCAddress)
	}

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

	produktionGRPC := server.NewProduktionGRPCServer(produktionService)
	produktionv1.RegisterProduktionServiceServer(grpcServer, produktionGRPC)

	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.ProduktionGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.ProduktionGRPCPort, "error", err)
		os.Exit(1)
	}

	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         cfg.ProduktionHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.ProduktionHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("produktion service starting", "port", cfg.ProduktionGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down produktion service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("produktion service stopped")
}
