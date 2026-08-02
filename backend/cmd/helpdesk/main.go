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
	"github.com/kmuhub/kmuhub/internal/helpdesk"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	helpdeskv1 "github.com/kmuhub/kmuhub/proto/helpdesk/v1"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
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

	// Repository + Service
	repo := helpdesk.NewPostgresRepository(pool)
	helpdeskService := helpdesk.NewService(repo, logger)

	// =========================================================================
	// Inbox gRPC Client (for CreateTicketFromMessage). InboxService is
	// co-hosted in the notification binary. Optional: nil disables the RPC
	// gracefully rather than failing helpdesk startup.
	// =========================================================================
	var inboxServiceClient inboxv1.InboxServiceClient
	if cfg.NotificationGRPCAddress != "" {
		inboxConn, inboxErr := grpc.NewClient(
			cfg.NotificationGRPCAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			// Propagate tenant/user from the inbound helpdesk context to the
			// inbox service -- GetMessage reads the tenant from metadata.
			grpc.WithUnaryInterceptor(middleware.TenantOutboundUnaryInterceptor()),
		)
		if inboxErr != nil {
			slog.Warn("failed to connect to inbox service, CreateTicketFromMessage disabled",
				"address", cfg.NotificationGRPCAddress,
				"error", inboxErr,
			)
		} else {
			defer inboxConn.Close()
			inboxServiceClient = inboxv1.NewInboxServiceClient(inboxConn)
			slog.Info("inbox client enabled (create-ticket-from-message)", "address", cfg.NotificationGRPCAddress)
		}
	} else {
		slog.Info("notification gRPC address not configured, CreateTicketFromMessage disabled")
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

	helpdeskGRPC := server.NewHelpdeskGRPCServer(helpdeskService, inboxServiceClient)
	helpdeskv1.RegisterHelpdeskServiceServer(grpcServer, helpdeskGRPC)

	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.HelpdeskGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.HelpdeskGRPCPort, "error", err)
		os.Exit(1)
	}

	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         cfg.HelpdeskHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.HelpdeskHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("helpdesk service starting", "port", cfg.HelpdeskGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down helpdesk service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("helpdesk service stopped")
}
