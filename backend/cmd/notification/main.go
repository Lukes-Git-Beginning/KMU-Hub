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
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/delivery"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/notification/notification"
	"github.com/kmuhub/kmuhub/internal/notification/preference"
	"github.com/kmuhub/kmuhub/internal/server"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
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

	// Initialize repositories
	notifRepo := notification.NewPostgresRepository(pool)
	prefRepo := preference.NewPostgresRepository(pool)

	// Initialize event type registry and load from DB
	registry := event.NewEventTypeRegistry()
	eventTypeRepo := newEventTypeRepo(pool)
	if err := registry.LoadFromDB(ctx, eventTypeRepo); err != nil {
		slog.Warn("failed to load event types from database, using empty registry",
			"error", err,
		)
	} else {
		slog.Info("event types loaded", "count", len(registry.ListAll()))
	}

	// Initialize services
	prefService := preference.NewService(prefRepo)
	grouper := notification.NewGrouper(notifRepo, 30*time.Second)
	dispatcher := delivery.NewDispatcher()
	notifService := notification.NewService(notifRepo, prefService, registry, grouper, dispatcher)

	// Initialize event bus
	eventBus := event.NewEventBus(cfg.DatabaseURL, event.WithReconnectWait(5*time.Second))

	// Register the main event handler that processes all events
	eventBus.RegisterHandler("*", notifService.ProcessEvent)

	// Start event bus listener in background
	go func() {
		slog.Info("starting event bus listener")
		if err := eventBus.Listen(ctx); err != nil && ctx.Err() == nil {
			slog.Error("event bus listener failed", "error", err)
		}
	}()

	// Process backlog of unprocessed events
	go func() {
		time.Sleep(2 * time.Second)
		backlogRepo := newEventRepoAdapter(notifRepo)
		if err := eventBus.ProcessBacklog(ctx, backlogRepo); err != nil {
			slog.Error("failed to process event backlog", "error", err)
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
	notifGRPC := server.NewNotificationGRPCServer(notifService, prefService, registry)
	notificationv1.RegisterNotificationServiceServer(grpcServer, notifGRPC)

	// Initialize gRPC metrics after service registration
	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.NotificationGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.NotificationGRPCPort, "error", err)
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
		Addr:         cfg.NotificationHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.NotificationHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("notification service starting", "port", cfg.NotificationGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down notification service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("notification service stopped")
}

// ============================================================================
// Event Type Repository (for loading event types from DB into registry)
// ============================================================================

type eventTypeRepo struct {
	pool *pgxpool.Pool
}

func newEventTypeRepo(pool *pgxpool.Pool) *eventTypeRepo {
	return &eventTypeRepo{pool: pool}
}

func (r *eventTypeRepo) ListAll(ctx context.Context) ([]models.EventType, error) {
	query := `
		SELECT id, module_id, event_key, display_name, description,
			default_priority, default_in_app, default_desktop_push, default_sound,
			created_at, updated_at
		FROM event_types
		ORDER BY module_id, event_key`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []models.EventType
	for rows.Next() {
		var et models.EventType
		err := rows.Scan(
			&et.ID, &et.ModuleID, &et.EventKey, &et.DisplayName, &et.Description,
			&et.DefaultPriority, &et.DefaultInApp, &et.DefaultDesktopPush, &et.DefaultSound,
			&et.CreatedAt, &et.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		types = append(types, et)
	}

	return types, rows.Err()
}

func (r *eventTypeRepo) GetByKey(ctx context.Context, key string) (*models.EventType, error) {
	query := `
		SELECT id, module_id, event_key, display_name, description,
			default_priority, default_in_app, default_desktop_push, default_sound,
			created_at, updated_at
		FROM event_types WHERE event_key = $1`

	var et models.EventType
	err := r.pool.QueryRow(ctx, query, key).Scan(
		&et.ID, &et.ModuleID, &et.EventKey, &et.DisplayName, &et.Description,
		&et.DefaultPriority, &et.DefaultInApp, &et.DefaultDesktopPush, &et.DefaultSound,
		&et.CreatedAt, &et.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &et, nil
}

// ============================================================================
// Event Repository Adapter (bridges notification repo to event bus interface)
// ============================================================================

type eventRepoAdapter struct {
	notifRepo *notification.PostgresRepository
}

func newEventRepoAdapter(notifRepo *notification.PostgresRepository) *eventRepoAdapter {
	return &eventRepoAdapter{notifRepo: notifRepo}
}

func (a *eventRepoAdapter) CreateEvent(ctx context.Context, evt *models.Event) error {
	return a.notifRepo.CreateEvent(ctx, evt)
}

func (a *eventRepoAdapter) ListUnprocessed(ctx context.Context, limit int) ([]models.Event, error) {
	return a.notifRepo.ListUnprocessedEvents(ctx, limit)
}

func (a *eventRepoAdapter) MarkProcessed(ctx context.Context, eventID string) error {
	return a.notifRepo.MarkEventProcessed(ctx, eventID)
}

