package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
	"github.com/kmuhub/kmuhub/internal/inbox/message"
	"github.com/kmuhub/kmuhub/internal/inbox/routing"
	"github.com/kmuhub/kmuhub/internal/inbox/team"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/delivery"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/notification/notification"
	"github.com/kmuhub/kmuhub/internal/notification/preference"
	"github.com/kmuhub/kmuhub/internal/server"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
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

	// =========================================================================
	// Inbox Service (co-hosted on the same gRPC server)
	// =========================================================================

	// Initialize inbox repositories
	inboxMessageRepo := message.NewPostgresRepository(pool)
	inboxTeamRepo := team.NewPostgresRepository(pool)
	inboxRoutingRepo := routing.NewPostgresRepository(pool)

	// Initialize channel adapters (nil clients for now -- they'll be connected
	// when concrete gRPC clients are available via gateway connections)
	adapterRegistry := adapter.NewAdapterRegistry()
	adapterRegistry.Register(adapter.NewEmailAdapter(nil))
	adapterRegistry.Register(adapter.NewChatAdapter(nil))
	adapterRegistry.Register(adapter.NewNotificationAdapter(nil))

	// Initialize inbox services
	inboxMessageService := message.NewService(inboxMessageRepo, adapterRegistry)
	inboxTeamService := team.NewService(inboxTeamRepo, inboxMessageRepo)
	inboxRoutingService := routing.NewService(inboxRoutingRepo, inboxMessageRepo, nil)

	// Create and register InboxConsumer on EventBus
	inboxConsumer := &InboxConsumer{
		messageService: inboxMessageService,
		messageRepo:    inboxMessageRepo,
		routingService: inboxRoutingService,
		adapters:       adapterRegistry,
	}
	eventBus.RegisterHandler("*", inboxConsumer.HandleEvent)

	// Start snooze worker as background goroutine
	inboxMessageService.StartSnoozeWorker(ctx, 60*time.Second)

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

	// Register InboxService on the same gRPC server
	inboxGRPC := server.NewInboxGRPCServer(inboxMessageService, inboxTeamService, inboxRoutingService)
	inboxv1.RegisterInboxServiceServer(grpcServer, inboxGRPC)
	slog.Info("inbox service co-hosted on notification gRPC server")

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
		slog.Info("notification+inbox service starting", "port", cfg.NotificationGRPCPort)
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

// ============================================================================
// Inbox Consumer (processes events into inbox messages)
// ============================================================================

// InboxConsumer sits between the EventBus and inbox services. It processes
// incoming events, creates InboxMessage entries for each target user,
// and applies routing rules.
type InboxConsumer struct {
	messageService *message.Service
	messageRepo    message.Repository
	routingService *routing.Service
	adapters       *adapter.AdapterRegistry
}

// HandleEvent processes an event from the EventBus into inbox messages.
// Each target user gets their own inbox_message entry.
func (ic *InboxConsumer) HandleEvent(ctx context.Context, evt models.EventPayload) error {
	// Skip events from inbox module to prevent circular loops (research pitfall #4)
	if evt.ModuleID == event.ModuleInbox {
		return nil
	}

	// Determine channel based on source module
	channel := ic.determineChannel(evt.ModuleID)

	// Create an inbox message for each target user
	for _, targetUserIDStr := range evt.TargetUserIDs {
		userID, err := uuid.Parse(targetUserIDStr)
		if err != nil {
			slog.Warn("inbox consumer: invalid target user_id, skipping",
				"user_id", targetUserIDStr,
				"event_type", evt.Type,
			)
			continue
		}

		msg := &models.InboxMessage{
			ID:         uuid.New(),
			UserID:     userID,
			Channel:    channel,
			SourceID:   evt.ResourceID,
			SenderName: ic.extractSenderName(evt),
			Subject:    evt.Title,
			Preview:    evt.Body,
			DeepLink:   evt.DeepLink,
			ReceivedAt: evt.Timestamp,
			Tags:       []string{},
		}

		// Set sender_id if actor is available
		if evt.ActorID != "" {
			if actorID, parseErr := uuid.Parse(evt.ActorID); parseErr == nil {
				msg.SenderID = &actorID
			}
		}

		// Create message (handles dedup internally via source_id)
		if err := ic.messageService.Create(ctx, msg); err != nil {
			if errors.Is(err, message.ErrDuplicateMessage) {
				// Duplicate is expected for already-processed events
				continue
			}
			slog.Error("inbox consumer: failed to create inbox message",
				"user_id", userID,
				"event_type", evt.Type,
				"error", err,
			)
			continue
		}

		// Apply routing rules to the new message
		if err := ic.routingService.EvaluateAndApply(ctx, msg); err != nil {
			slog.Warn("inbox consumer: routing evaluation failed",
				"message_id", msg.ID,
				"error", err,
			)
			// Non-fatal: message is created, routing is best-effort
		}

		// Emit pg_notify for gateway WebSocket push
		payload := struct {
			Type      string `json:"type"`
			UserID    string `json:"user_id"`
			MessageID string `json:"message_id"`
			Channel   string `json:"channel"`
			Subject   string `json:"subject"`
		}{
			Type:      "inbox.message.new",
			UserID:    userID.String(),
			MessageID: msg.ID.String(),
			Channel:   channel,
			Subject:   msg.Subject,
		}
		data, jsonErr := json.Marshal(payload)
		if jsonErr == nil {
			if notifyErr := ic.messageRepo.NotifyDelivery(ctx, string(data)); notifyErr != nil {
				slog.Warn("inbox consumer: failed to send delivery notification",
					"error", notifyErr,
				)
			}
		}
	}

	return nil
}

// determineChannel maps an event's module_id to an inbox channel.
func (ic *InboxConsumer) determineChannel(moduleID string) string {
	switch moduleID {
	case event.ModuleEmail:
		return "email"
	case event.ModuleChat:
		return "chat"
	default:
		return "notification"
	}
}

// extractSenderName extracts a display name for the sender from the event.
func (ic *InboxConsumer) extractSenderName(evt models.EventPayload) string {
	// Try to extract from payload
	if evt.Payload != nil {
		var p struct {
			SenderName string `json:"sender_name"`
			ActorName  string `json:"actor_name"`
		}
		if err := json.Unmarshal(evt.Payload, &p); err == nil {
			if p.SenderName != "" {
				return p.SenderName
			}
			if p.ActorName != "" {
				return p.ActorName
			}
		}
	}

	// Fall back to module name
	switch evt.ModuleID {
	case event.ModuleEmail:
		return "E-Mail"
	case event.ModuleChat:
		return "Chat"
	case event.ModuleCRM:
		return "CRM"
	case event.ModuleWork:
		return "Aufgaben"
	case event.ModuleBiz:
		return "Finanzen"
	case event.ModuleHR:
		return "Personal"
	case event.ModuleDocument:
		return "Dokumente"
	default:
		return "System"
	}
}

