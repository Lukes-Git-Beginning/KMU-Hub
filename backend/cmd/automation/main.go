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

	"github.com/kmuhub/kmuhub/internal/automation/action"
	"github.com/kmuhub/kmuhub/internal/automation/condition"
	"github.com/kmuhub/kmuhub/internal/automation/engine"
	automationtemplate "github.com/kmuhub/kmuhub/internal/automation/template"
	"github.com/kmuhub/kmuhub/internal/automation/trigger"
	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/server"
	automationv1 "github.com/kmuhub/kmuhub/proto/automation/v1"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	calendarv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
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

	// =========================================================================
	// Repositories
	// =========================================================================
	repo := workflow.NewPostgresRepository(pool)

	// =========================================================================
	// Condition evaluator (from Plan 01)
	// =========================================================================
	condEvaluator := condition.NewEvaluator()

	// =========================================================================
	// Trigger registry
	// =========================================================================
	triggerReg := trigger.NewTriggerRegistry()
	slog.Info("trigger registry initialized", "count", triggerReg.Count())

	// =========================================================================
	// gRPC clients to other services
	// =========================================================================
	crmConn := dialService("crm", cfg.CRMGRPCAddress)
	workConn := dialService("work", cfg.WorkGRPCAddress)
	emailConn := dialService("email", cfg.EmailGRPCAddress)
	bizConn := dialService("biz", cfg.BizGRPCAddress)
	// Calendar is co-hosted on the work service
	calendarConn := workConn

	crmClient := crmv1.NewCRMServiceClient(crmConn)
	workClient := workv1.NewWorkServiceClient(workConn)
	emailClient := emailv1.NewEmailServiceClient(emailConn)
	calendarClient := calendarv1.NewCalendarServiceClient(calendarConn)
	bizClient := bizv1.NewFinanceServiceClient(bizConn)

	// =========================================================================
	// Action registry
	// =========================================================================
	actionReg := action.NewActionRegistry()

	// CRM actions
	actionReg.Register(
		action.NewUpdateDealFieldAction(crmClient),
		action.UpdateDealFieldDefinition(),
	)
	actionReg.Register(
		action.NewCreateContactAction(crmClient),
		action.CreateContactDefinition(),
	)

	// Work action
	actionReg.Register(
		action.NewCreateTaskAction(workClient),
		action.CreateTaskDefinition(),
	)

	// Email action
	actionReg.Register(
		action.NewSendEmailAction(emailClient),
		action.SendEmailDefinition(),
	)

	// Notification action (standalone, no gRPC client needed)
	actionReg.Register(
		action.NewSendNotificationAction(),
		action.SendNotificationDefinition(),
	)

	// Calendar action
	actionReg.Register(
		action.NewCreateCalendarEventAction(calendarClient),
		action.CreateCalendarEventDefinition(),
	)

	// Biz actions
	actionReg.Register(
		action.NewCreateInvoiceDraftAction(bizClient),
		action.CreateInvoiceDraftDefinition(),
	)
	actionReg.Register(
		action.NewCreateDunningAction(bizClient),
		action.CreateDunningDefinition(),
	)

	slog.Info("action registry initialized", "count", actionReg.Count())

	// =========================================================================
	// Execution logger
	// =========================================================================
	execLogger := engine.NewExecutionLogger(repo)

	// =========================================================================
	// Workflow engine
	// =========================================================================
	workflowEngine := engine.NewWorkflowEngine(condEvaluator, actionReg, execLogger, repo, repo)

	// =========================================================================
	// Trigger matcher + consumer
	// =========================================================================
	triggerMatcher := trigger.NewTriggerMatcher(repo, triggerReg)
	automationConsumer := engine.NewAutomationConsumer(triggerMatcher, workflowEngine)

	// =========================================================================
	// EventBus: register automation consumer for all events
	// =========================================================================
	eventBus := event.NewEventBus(cfg.DatabaseURL)
	eventBus.RegisterHandler("*", automationConsumer.HandleEvent)

	// =========================================================================
	// Time-based trigger poller
	// =========================================================================
	timeTriggerPoller := trigger.NewTimeTriggerPoller(repo, repo, workflowEngine, pool)
	go timeTriggerPoller.Start(ctx)

	// =========================================================================
	// Workflow service
	// =========================================================================
	workflowService := workflow.NewService(
		repo, repo, repo,
		func(triggerType string) (any, bool) {
			return triggerReg.Get(triggerType)
		},
		func() []any {
			defs := triggerReg.All()
			result := make([]any, len(defs))
			for i, d := range defs {
				result[i] = d
			}
			return result
		},
		func(actionType string) (any, bool) {
			return actionReg.Get(actionType)
		},
		func() []any {
			defs := actionReg.AllDefinitions()
			result := make([]any, len(defs))
			for i, d := range defs {
				result[i] = d
			}
			return result
		},
		condEvaluator,
	)
	// workflowService is used below by AutomationGRPCServer

	// =========================================================================
	// Template seeding
	// =========================================================================
	templateReg := automationtemplate.NewTemplateRegistry()
	if seedErr := templateReg.SeedToDatabase(ctx, repo); seedErr != nil {
		slog.Warn("failed to seed automation templates", "error", seedErr)
	}

	// =========================================================================
	// Metrics
	// =========================================================================
	metricsRegistry := metrics.NewRegistry()

	// =========================================================================
	// gRPC server
	// =========================================================================
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metricsRegistry.GRPCUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)

	automationGRPC := server.NewAutomationGRPCServer(workflowService)
	automationv1.RegisterAutomationServiceServer(grpcServer, automationGRPC)

	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.AutomationGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.AutomationGRPCPort, "error", err)
		os.Exit(1)
	}

	// =========================================================================
	// Health + metrics HTTP server
	// =========================================================================
	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	healthRouter.Get("/health", server.HealthHandler(healthCheckers))
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         cfg.AutomationHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.AutomationHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	// Start gRPC server
	go func() {
		slog.Info("automation service starting", "port", cfg.AutomationGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	// Start EventBus listener
	go func() {
		slog.Info("automation EventBus listener starting")
		if busErr := eventBus.Listen(ctx); busErr != nil && ctx.Err() == nil {
			slog.Error("event bus listener failed", "error", busErr)
		}
	}()

	// =========================================================================
	// Graceful shutdown
	// =========================================================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down automation service")
	case <-ctx.Done():
	}

	cancel() // Stop EventBus, poller, and any in-flight executions

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()

	// Close gRPC client connections
	for _, conn := range []*grpc.ClientConn{crmConn, workConn, emailConn, bizConn} {
		if conn != nil {
			conn.Close()
		}
	}

	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("automation service stopped")
}

// dialService creates a gRPC client connection to a service.
// Returns nil connection on failure (actions will return "service unavailable").
func dialService(name, address string) *grpc.ClientConn {
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Warn("failed to dial service, actions will be unavailable",
			"service", name,
			"address", address,
			"error", err,
		)
		return nil
	}
	slog.Info("gRPC client connected", "service", name, "address", address)
	return conn
}
