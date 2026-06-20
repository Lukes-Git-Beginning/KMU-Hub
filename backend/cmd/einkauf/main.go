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
	"github.com/kmuhub/kmuhub/internal/einkauf"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	einkaufv1 "github.com/kmuhub/kmuhub/proto/einkauf/v1"
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
	repo := einkauf.NewPostgresRepository(pool)
	// NewServiceExtended wires both the base repo and the extended repo
	// (catalog, supplier ratings, framework contracts) so all RPCs work.
	einkaufService := einkauf.NewServiceExtended(repo)

	// Wire inventar gRPC client for cross-module stock adjustment on goods receipt.
	// The connection is established lazily; failure to connect is non-fatal since
	// inventar integration is best-effort (graceful degradation).
	inventarConn, inventarConnErr := grpc.NewClient(
		cfg.InventarGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if inventarConnErr != nil {
		slog.Warn("failed to create inventar gRPC connection; stock sync on goods receipt disabled",
			"address", cfg.InventarGRPCAddress, "error", inventarConnErr)
	} else {
		inventarClient := inventarv1.NewInventarServiceClient(inventarConn)
		einkaufService.WithInventarAdjuster(einkauf.NewGRPCInventarAdjuster(inventarClient))
		defer inventarConn.Close()
		slog.Info("inventar gRPC client connected", "address", cfg.InventarGRPCAddress)
	}

	metricsRegistry := metrics.NewRegistry()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RecoveryUnaryInterceptor(),
			metricsRegistry.GRPCUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			middleware.RecoveryStreamInterceptor(),
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)

	einkaufGRPC := server.NewEinkaufGRPCServer(einkaufService)
	einkaufv1.RegisterEinkaufServiceServer(grpcServer, einkaufGRPC)

	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.EinkaufGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.EinkaufGRPCPort, "error", err)
		os.Exit(1)
	}

	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         cfg.EinkaufHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.EinkaufHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("einkauf service starting", "port", cfg.EinkaufGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down einkauf service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("einkauf service stopped")
}
