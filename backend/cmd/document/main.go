package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/document/file"
	"github.com/kmuhub/kmuhub/internal/document/folder"
	"github.com/kmuhub/kmuhub/internal/document/search"
	"github.com/kmuhub/kmuhub/internal/document/share"
	"github.com/kmuhub/kmuhub/internal/document/tag"
	"github.com/kmuhub/kmuhub/internal/document/virtual"
	"github.com/kmuhub/kmuhub/internal/document/wopi"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/server"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
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

	// MinIO file store for document storage (reuse chat/file MinIO adapter)
	fileStore, err := chatfile.NewMinIOStore(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		slog.Error("failed to connect to minio", "error", err)
		os.Exit(1)
	}

	// Initialize repositories
	folderRepo := folder.NewPostgresRepository(pool)
	fileRepo := file.NewPostgresRepository(pool)
	shareRepo := share.NewPostgresRepository(pool)
	tagRepo := tag.NewPostgresRepository(pool)
	searchRepo := search.NewPostgresRepository(pool)
	virtualRepo := virtual.NewPostgresRepository(pool)

	// Initialize services
	maxFileSize := int64(cfg.FileSizeLimitMB) * 1024 * 1024
	folderService := folder.NewService(folderRepo)
	fileService := file.NewService(fileRepo, fileStore, maxFileSize)
	shareService := share.NewService(shareRepo)
	tagService := tag.NewService(tagRepo)

	// Search service with text extractor (1 MB max index)
	extractor := search.NewExtractor(search.DefaultMaxIndexBytes)
	searchService := search.NewService(searchRepo, extractor, fileStore)

	virtualService := virtual.NewService(virtualRepo)

	// WOPI services for OnlyOffice collaborative editing
	tokenService := wopi.NewTokenService(cfg.WOPIJWTSecret)
	lockService := wopi.NewLockService(pool)

	slog.Info("document services initialized",
		"minio_endpoint", cfg.MinIOEndpoint,
		"minio_bucket", cfg.MinIOBucket,
		"max_file_size_mb", cfg.FileSizeLimitMB,
	)

	// Start WOPI lock cleanup goroutine (every 5 minutes)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cleaned, cleanErr := lockService.CleanExpired(ctx)
				if cleanErr != nil {
					slog.Error("WOPI lock cleanup failed", "error", cleanErr)
				} else if cleaned > 0 {
					slog.Info("WOPI expired locks cleaned", "count", cleaned)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

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

	// The file.PostgresRepository implements search.FileRepo (UpdateSearchContent)
	documentGRPC := server.NewDocumentGRPCServer(
		folderService,
		fileService,
		shareService,
		tagService,
		searchService,
		virtualService,
		tokenService,
		fileRepo,
	)
	documentv1.RegisterDocumentServiceServer(grpcServer, documentGRPC)

	// Initialize gRPC metrics after service registration
	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.DocumentGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.DocumentGRPCPort, "error", err)
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
		Addr:         cfg.DocumentHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.DocumentHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("document service starting", "port", cfg.DocumentGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down document service")
	case <-ctx.Done():
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("document service stopped")
}

// wopiFileAdapter bridges file.Service to wopi.FileServiceInterface.
// The WOPI package defines its own VersionInput to avoid circular imports,
// so this adapter converts between the two structurally identical types.
type wopiFileAdapter struct {
	svc *file.Service
}

func (a *wopiFileAdapter) GetByID(ctx context.Context, id uuid.UUID) (*models.DocumentFile, error) {
	return a.svc.GetByID(ctx, id)
}

func (a *wopiFileAdapter) GetDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error) {
	return a.svc.GetDownloadURL(ctx, fileID)
}

func (a *wopiFileAdapter) CreateVersion(ctx context.Context, fileID uuid.UUID, input wopi.VersionInput) (*models.DocumentFileVersion, error) {
	return a.svc.CreateVersion(ctx, fileID, file.VersionInput{
		Reader:   input.Reader,
		FileSize: input.FileSize,
		MimeType: input.MimeType,
		Label:    input.Label,
		UserID:   input.UserID,
	})
}

// Ensure wopiFileAdapter satisfies the interface at compile time.
var _ wopi.FileServiceInterface = (*wopiFileAdapter)(nil)

// Suppress unused import warnings for packages used by adapter types.
var _ io.Reader
