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

	"github.com/kmuhub/kmuhub/internal/cache"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/crm/activity"
	"github.com/kmuhub/kmuhub/internal/crm/company"
	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/crm/customfield"
	"github.com/kmuhub/kmuhub/internal/crm/deal"
	"github.com/kmuhub/kmuhub/internal/crm/pipelinestage"
	"github.com/kmuhub/kmuhub/internal/crm/report"
	"github.com/kmuhub/kmuhub/internal/crm/savedfilter"
	"github.com/kmuhub/kmuhub/internal/crm/search"
	"github.com/kmuhub/kmuhub/internal/crm/tag"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/server"
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

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Redis for cache-aside (best-effort)
	redisClient, redisErr := database.NewRedisClient(ctx, cfg.RedisURL)
	if redisErr != nil {
		slog.Warn("redis unavailable for crm cache, using direct db", "error", redisErr)
	}
	cacheClient := cache.NewClient(redisClient)

	// Initialize repositories
	customFieldRepo := customfield.NewPostgresRepository(pool)
	tagRepo := tag.NewPostgresRepository(pool)
	contactRepo := contact.NewPostgresRepository(pool)
	companyRepo := company.NewPostgresRepository(pool)
	pipelineStageRepo := pipelinestage.NewCachedRepository(
		pipelinestage.NewPostgresRepository(pool), cacheClient,
	)
	dealRepo := deal.NewPostgresRepository(pool)
	activityRepo := activity.NewPostgresRepository(pool)
	searchRepo := search.NewPostgresRepository(pool)
	savedFilterRepo := savedfilter.NewPostgresRepository(pool)
	reportRepo := report.NewPostgresRepository(pool)
	consentRepo := consent.NewPostgresRepository(pool)

	// Initialize services
	customFieldService := customfield.NewService(customFieldRepo)
	tagService := tag.NewService(tagRepo)
	contactService := contact.NewService(contactRepo)
	companyService := company.NewService(companyRepo)
	pipelineStageService := pipelinestage.NewService(pipelineStageRepo)
	dealService := deal.NewService(dealRepo)
	dealService.SetEventEmitter(deal.NewPGEventEmitter(pool))
	activityService := activity.NewService(activityRepo)
	consentService := consent.NewService(consentRepo)
	searchService := search.NewService(searchRepo)
	savedFilterService := savedfilter.NewService(savedFilterRepo)
	reportService := report.NewService(reportRepo)

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
	crmGRPC := server.NewCRMGRPCServer(customFieldService, tagService, contactService, companyService, pipelineStageService, dealService, activityService, consentService, searchService, savedFilterService, reportService)
	crmv1.RegisterCRMServiceServer(grpcServer, crmGRPC)

	// Initialize gRPC metrics after service registration
	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.CRMGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.CRMGRPCPort, "error", err)
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
		Addr:         cfg.CRMHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.CRMHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("crm service starting", "port", cfg.CRMGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down crm service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("crm service stopped")
}
