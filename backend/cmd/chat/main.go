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

	"github.com/kmuhub/kmuhub/internal/chat/channel"
	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/chat/langdetect"
	"github.com/kmuhub/kmuhub/internal/chat/message"
	"github.com/kmuhub/kmuhub/internal/chat/search"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/work/reaction"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load(ctx, config.RequireMinIO)
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

	// Initialize repositories
	channelRepo := channel.NewPostgresRepository(pool)
	messageRepo := message.NewPostgresRepository(pool)
	fileRepo := file.NewPostgresRepository(pool)
	searchRepo := search.NewPostgresRepository(pool)
	reactionRepo := reaction.NewPostgresRepository(pool)

	// Initialize MinIO file store.
	// When MINIO_PUBLIC_ENDPOINT is set, presigned URLs are signed with the
	// public-facing endpoint so they are reachable from the browser / Electron.
	var fileStore file.FileStore
	if cfg.MinIOPublicEndpoint != "" {
		fileStore, err = file.NewMinIOStoreWithPublicEndpoint(
			cfg.MinIOEndpoint,
			cfg.MinIOAccessKey,
			cfg.MinIOSecretKey,
			cfg.MinIOBucket,
			cfg.MinIOUseSSL,
			cfg.MinIOPublicEndpoint,
			cfg.MinIOPublicUseSSL,
		)
	} else {
		fileStore, err = file.NewMinIOStore(
			cfg.MinIOEndpoint,
			cfg.MinIOAccessKey,
			cfg.MinIOSecretKey,
			cfg.MinIOBucket,
			cfg.MinIOUseSSL,
		)
	}
	if err != nil {
		slog.Error("failed to connect to minio", "error", err)
		os.Exit(1)
	}

	// Initialize file service components
	fileScanner := &file.NoOpScanner{}
	thumbnailGen := file.NewImagingGenerator()

	// Initialize language detector
	langDetector := langdetect.NewLinguaDetector()

	// Initialize services
	channelService := channel.NewService(channelRepo)
	messageService := message.NewService(messageRepo)
	messageService.SetLangDetector(langDetector)
	messageService.SetEventEmitter(message.NewPGEventEmitter(pool))
	fileService := file.NewService(fileRepo, fileStore, fileScanner, thumbnailGen, cfg.FileSizeLimitMB)
	searchService := search.NewService(searchRepo, langDetector)
	reactionService := reaction.NewService(reactionRepo)

	// Metrics
	metricsRegistry := metrics.NewRegistry()

	// gRPC server
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
	chatGRPC := server.NewChatGRPCServer(channelService, messageService, fileService, searchService, reactionService)
	chatv1.RegisterChatServiceServer(grpcServer, chatGRPC)

	// Initialize gRPC metrics after service registration
	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.ChatGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.ChatGRPCPort, "error", err)
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
		Addr:         cfg.ChatHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.ChatHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("chat service starting", "port", cfg.ChatGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down chat service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("chat service stopped")
}
