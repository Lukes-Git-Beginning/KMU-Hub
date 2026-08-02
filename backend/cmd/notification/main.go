package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
	"github.com/kmuhub/kmuhub/internal/inbox/message"
	"github.com/kmuhub/kmuhub/internal/inbox/routing"
	"github.com/kmuhub/kmuhub/internal/inbox/team"
	"github.com/kmuhub/kmuhub/internal/inbox/thread"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/delivery"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
	slackadapter "github.com/kmuhub/kmuhub/internal/notification/integration/slack"
	teamsadapter "github.com/kmuhub/kmuhub/internal/notification/integration/teams"
	"github.com/kmuhub/kmuhub/internal/notification/notification"
	"github.com/kmuhub/kmuhub/internal/notification/preference"
	"github.com/kmuhub/kmuhub/internal/server"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
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

	// =========================================================================
	// Integration Forwarder (Teams & Slack notification forwarding)
	// =========================================================================

	integrationRepo := integration.NewPostgresRepository(pool)
	accountLinkService := integration.NewAccountLinkService(integrationRepo)
	rateLimiter := integration.NewRateLimiter()

	// Platform clients double as connection probers for the admin-facing
	// connection test; a platform missing here cannot be tested and the RPC
	// reports that instead of a green check.
	connectionProbers := make(map[string]integration.ConnectionProber)

	// Inbound webhook processors. A platform only appears here when it can
	// actually verify a request; without its credentials the RPC answers
	// "not configured" rather than processing an unverified payload.
	webhookProcessors := make(map[string]integration.WebhookProcessor)

	// Initialize Teams client (nil-safe: disabled when env vars not set)
	var teamsClient integration.PlatformPoster
	teamsAppID := os.Getenv("TEAMS_APP_ID")
	teamsAppPassword := os.Getenv("TEAMS_APP_PASSWORD")
	if teamsAppID != "" && teamsAppPassword != "" {
		tc, err := teamsadapter.NewClient(teamsAppID, teamsAppPassword)
		if err != nil {
			slog.Error("failed to create teams client", "error", err)
		} else {
			teamsClient = tc
			connectionProbers[integration.PlatformTeams] = tc
			webhookProcessors[integration.PlatformTeams] = teamsadapter.NewWebhookHandler(
				tc, integrationRepo, accountLinkService, integrationRepo, notifService)
		}
	}

	// Initialize Slack client (nil-safe: disabled when env vars not set)
	var slackClient integration.PlatformPoster
	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackBotToken != "" {
		sc := slackadapter.NewClient(slackBotToken)
		slackClient = sc
		connectionProbers[integration.PlatformSlack] = sc

		// Deliberately os.Getenv, never config.RequireX: an absent signing
		// secret must leave the inbound routes off, not crash the service.
		if slackSigningSecret := os.Getenv("SLACK_SIGNING_SECRET"); slackSigningSecret != "" {
			webhookProcessors[integration.PlatformSlack] = slackadapter.NewWebhookHandler(
				sc, slackSigningSecret, integrationRepo, accountLinkService, integrationRepo, notifService)
		} else {
			slog.Warn("slack bot token set without SLACK_SIGNING_SECRET -- inbound slack webhooks stay disabled")
		}
	}

	// Initialize forwarder and register as delivery callback
	forwarder := integration.NewForwarder(integrationRepo, teamsClient, slackClient, rateLimiter)
	dispatcher.OnDeliver(forwarder.HandleNotification)

	slog.Info("integration forwarder initialized",
		"teams_enabled", teamsClient != nil,
		"slack_enabled", slackClient != nil,
		"inbound_webhooks", len(webhookProcessors),
	)

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
	inboxThreadRepo := thread.NewPostgresRepository(pool)

	// Initialize channel adapters. Email is wired to the real email service
	// (below); chat and notification stay nil-safe stubs until their own
	// cross-service clients are wired (they degrade gracefully -- empty
	// fetch results, no-op reply/forward).
	var emailClient adapter.EmailClient
	if cfg.EmailGRPCAddress != "" {
		emailConn, emailErr := grpc.NewClient(
			cfg.EmailGRPCAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			// Propagate tenant from the inbound notification/inbox context to
			// email -- the email gRPC handlers read tenant from metadata and
			// answer InvalidArgument without it.
			grpc.WithUnaryInterceptor(middleware.TenantOutboundUnaryInterceptor()),
		)
		if emailErr != nil {
			slog.Warn("failed to connect to email service, inbox email reply/forward disabled",
				"address", cfg.EmailGRPCAddress,
				"error", emailErr,
			)
		} else {
			defer emailConn.Close()
			emailClient = &emailGRPCClient{client: emailv1.NewEmailServiceClient(emailConn)}
			slog.Info("email client enabled (inbox reply/forward)", "address", cfg.EmailGRPCAddress)
		}
	}

	adapterRegistry := adapter.NewAdapterRegistry()
	adapterRegistry.Register(adapter.NewEmailAdapter(emailClient))
	adapterRegistry.Register(adapter.NewChatAdapter(nil))
	adapterRegistry.Register(adapter.NewNotificationAdapter(nil))

	// Initialize inbox services
	inboxMessageService := message.NewService(inboxMessageRepo, adapterRegistry)
	inboxTeamService := team.NewService(inboxTeamRepo, inboxMessageRepo)
	inboxRoutingService := routing.NewService(inboxRoutingRepo, inboxMessageRepo, nil)
	inboxThreadService := thread.NewService(inboxThreadRepo)

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
			middleware.RecoveryUnaryInterceptor(),
			metricsRegistry.GRPCUnaryInterceptor(),
			middleware.TenantInboundUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			middleware.RecoveryStreamInterceptor(),
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)
	notifGRPC := server.NewNotificationGRPCServer(notifService, prefService, registry,
		server.WithIntegration(integrationRepo, accountLinkService),
		server.WithConnectionProbers(connectionProbers),
		server.WithPlatformWebhooks(webhookProcessors),
	)
	notificationv1.RegisterNotificationServiceServer(grpcServer, notifGRPC)

	// Register InboxService on the same gRPC server
	inboxGRPC := server.NewInboxGRPCServer(inboxMessageService, inboxTeamService, inboxRoutingService, inboxThreadService)
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
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
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
			TenantID:   evt.TenantID,
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

	// Top-level actor name (set by emitters that resolve it before emit)
	if evt.ActorName != "" {
		return evt.ActorName
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

// ============================================================================
// Email Adapter Client (bridges the inbox adapter.EmailClient interface to
// the real email gRPC service)
// ============================================================================

// emailGRPCClient implements adapter.EmailClient against the real email
// service. sourceID/threadID parameters carry the *email message* id -- the
// same id inbox.HandleEvent stores as InboxMessage.SourceID when it consumes
// an email.received event (see internal/email/message/service.go emitEvent).
type emailGRPCClient struct {
	client emailv1.EmailServiceClient
}

// resolveAccountID looks up the user's email account. There is at most one
// account per (user, tenant) -- see email/account.Service.GetByUserIDAndTenant.
func (c *emailGRPCClient) resolveAccountID(ctx context.Context, userID uuid.UUID) (string, error) {
	resp, err := c.client.GetEmailAccount(ctx, &emailv1.GetEmailAccountRequest{UserId: userID.String()})
	if err != nil {
		return "", fmt.Errorf("resolve email account: %w", err)
	}
	return resp.GetAccount().GetId(), nil
}

// ListMessages fetches the most recent inbox-folder messages for a user's
// email account, filtered client-side to those received after since.
//
// lean: single page (50 messages), no multi-folder/pagination sweep -- there
// is currently no caller of FetchNewMessages (no polling worker exists, the
// live inbox path is event-driven via InboxConsumer.HandleEvent). Upgrade to
// full pagination once a poller starts calling this path.
func (c *emailGRPCClient) ListMessages(ctx context.Context, userID uuid.UUID, since time.Time) ([]adapter.EmailMessageData, error) {
	accountID, err := c.resolveAccountID(ctx, userID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}

	folders, err := c.client.ListFolders(ctx, &emailv1.ListFoldersRequest{AccountId: accountID})
	if err != nil {
		return nil, fmt.Errorf("list email folders: %w", err)
	}

	var inboxFolderID string
	for _, f := range folders.GetFolders() {
		if f.GetFolderType() == "inbox" {
			inboxFolderID = f.GetId()
			break
		}
	}
	if inboxFolderID == "" {
		return nil, nil
	}

	msgs, err := c.client.ListMessages(ctx, &emailv1.ListMessagesRequest{
		FolderId: inboxFolderID,
		Page:     1,
		PerPage:  50,
		SortBy:   "date",
		SortDesc: true,
	})
	if err != nil {
		return nil, fmt.Errorf("list email messages: %w", err)
	}

	result := make([]adapter.EmailMessageData, 0, len(msgs.GetMessages()))
	for _, m := range msgs.GetMessages() {
		receivedAt, parseErr := time.Parse(time.RFC3339, m.GetDate())
		if parseErr != nil {
			slog.Warn("email client: unparseable message date, skipping", "message_id", m.GetId(), "date", m.GetDate())
			continue
		}
		if !receivedAt.After(since) {
			continue
		}
		result = append(result, adapter.EmailMessageData{
			ThreadID:   m.GetThreadId(),
			FromName:   m.GetFrom().GetName(),
			FromEmail:  m.GetFrom().GetEmail(),
			Subject:    m.GetSubject(),
			Body:       m.GetBodyText(),
			ReceivedAt: receivedAt,
			MessageID:  m.GetId(),
		})
	}
	return result, nil
}

// SendReply sends a reply to the email message identified by sourceID.
func (c *emailGRPCClient) SendReply(ctx context.Context, sourceID string, userID uuid.UUID, body string) error {
	accountID, err := c.resolveAccountID(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := c.client.ReplyEmail(ctx, &emailv1.ReplyEmailRequest{
		AccountId:         accountID,
		OriginalMessageId: sourceID,
		BodyText:          body,
	}); err != nil {
		return fmt.Errorf("reply email: %w", err)
	}
	return nil
}

// ForwardEmail forwards the email message identified by sourceID to a new recipient.
func (c *emailGRPCClient) ForwardEmail(ctx context.Context, sourceID string, userID uuid.UUID, to string, note string) error {
	accountID, err := c.resolveAccountID(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := c.client.ForwardEmail(ctx, &emailv1.ForwardEmailRequest{
		AccountId:         accountID,
		OriginalMessageId: sourceID,
		To:                []*emailv1.EmailAddress{{Email: to}},
		BodyText:          note,
	}); err != nil {
		return fmt.Errorf("forward email: %w", err)
	}
	return nil
}

// MarkRead marks the email message identified by sourceID as read.
func (c *emailGRPCClient) MarkRead(ctx context.Context, sourceID string, _ uuid.UUID) error {
	if _, err := c.client.MarkRead(ctx, &emailv1.MarkReadRequest{Id: sourceID}); err != nil {
		return fmt.Errorf("mark email read: %w", err)
	}
	return nil
}
