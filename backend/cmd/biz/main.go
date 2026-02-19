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
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kmuhub/kmuhub/internal/biz/creditnote"
	"github.com/kmuhub/kmuhub/internal/biz/dashboard"
	"github.com/kmuhub/kmuhub/internal/biz/datev"
	"github.com/kmuhub/kmuhub/internal/biz/dunning"
	"github.com/kmuhub/kmuhub/internal/biz/hr/absence"
	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/payment"
	"github.com/kmuhub/kmuhub/internal/biz/pdf"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/server"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// crmDealUpdater implements quote.DealValueUpdater by calling CRM UpdateDeal RPC.
// This fulfills the locked decision: "Deal value auto-syncs when quote is created or modified."
type crmDealUpdater struct {
	crmClient crmv1.CRMServiceClient
}

func (u *crmDealUpdater) UpdateDealValue(ctx context.Context, dealID uuid.UUID, grossTotal decimal.Decimal) error {
	// CRM proto UpdateDealRequest.Value is *float64 (not string)
	val := grossTotal.InexactFloat64()
	_, err := u.crmClient.UpdateDeal(ctx, &crmv1.UpdateDealRequest{
		Id:    dealID.String(),
		Value: &val,
	})
	return err
}

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

	slog.Info("biz service initialized",
		"grpc_port", cfg.BizGRPCPort,
		"health_port", cfg.BizHealthPort,
	)

	// =========================================================================
	// Initialize Repositories
	// =========================================================================
	quoteRepo := quote.NewPostgresRepository(pool)
	invoiceRepo := invoice.NewPostgresRepository(pool)
	creditNoteRepo := creditnote.NewPostgresRepository(pool)
	paymentRepo := payment.NewPostgresRepository(pool)
	dunningRepo := dunning.NewPostgresRepository(pool)
	dunningConfigRepo := dunning.NewPostgresConfigRepository(pool)
	dashboardRepo := dashboard.NewPostgresRepository(pool)

	// Shared repositories (used by multiple services)
	numberSeqRepo := quote.NewPostgresNumberSequenceRepo(pool)
	companySettingsRepo := quote.NewPostgresCompanySettingsRepo(pool)

	// =========================================================================
	// CRM gRPC Client (for DealValueUpdater)
	// =========================================================================
	var dealUpdater quote.DealValueUpdater
	if cfg.CRMGRPCAddress != "" {
		crmConn, crmErr := grpc.NewClient(
			cfg.CRMGRPCAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if crmErr != nil {
			slog.Warn("failed to connect to CRM service, deal value sync disabled",
				"address", cfg.CRMGRPCAddress,
				"error", crmErr,
			)
		} else {
			defer crmConn.Close()
			crmClient := crmv1.NewCRMServiceClient(crmConn)
			dealUpdater = &crmDealUpdater{crmClient: crmClient}
			slog.Info("CRM deal value sync enabled", "address", cfg.CRMGRPCAddress)
		}
	} else {
		slog.Info("CRM gRPC address not configured, deal value sync disabled")
	}

	// =========================================================================
	// Initialize Services
	// =========================================================================
	quoteSvc := quote.NewService(quoteRepo, numberSeqRepo, companySettingsRepo, dealUpdater)
	invoiceSvc := invoice.NewService(invoiceRepo, numberSeqRepo, companySettingsRepo, quoteRepo)
	creditNoteSvc := creditnote.NewService(creditNoteRepo, invoiceRepo, numberSeqRepo)
	paymentSvc := payment.NewService(paymentRepo, invoiceRepo, invoiceRepo)
	dunningSvc := dunning.NewService(dunningRepo, dunningConfigRepo, invoiceRepo)
	dashboardSvc := dashboard.NewService(dashboardRepo)

	// PDF generator with company settings defaults
	settings, settingsErr := companySettingsRepo.GetByTenantID(ctx, uuid.Nil)
	if settingsErr != nil || settings == nil {
		settings = &models.CompanySettings{
			AccentColor:             "#1a73e8",
			DefaultPaymentTermsDays: 30,
			DefaultQuoteValidityDays: 30,
		}
	}
	pdfGen := pdf.NewGenerator(*settings)

	// DATEV exporter
	datevExp := datev.NewExporter()

	// =========================================================================
	// gRPC Server
	// =========================================================================
	metricsRegistry := metrics.NewRegistry()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metricsRegistry.GRPCUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)

	bizGRPC := server.NewBizGRPCServer(
		quoteSvc,
		invoiceSvc,
		creditNoteSvc,
		paymentSvc,
		dunningSvc,
		dashboardSvc,
		pdfGen,
		datevExp,
		companySettingsRepo,
	)
	bizv1.RegisterFinanceServiceServer(grpcServer, bizGRPC)

	// =========================================================================
	// HR Service Initialization (shares same gRPC server as finance)
	// =========================================================================
	leaveRequestRepo := leave.NewPostgresLeaveRequestRepo(pool)
	leaveBalanceRepo := leave.NewPostgresLeaveBalanceRepo(pool)
	leaveTypeRepo := leave.NewPostgresLeaveTypeRepo(pool)
	hrSettingsRepo := leave.NewPostgresHRSettingsRepo(pool)
	employeeRepo := employee.NewPostgresEmployeeRepo(pool)
	docCategoryRepo := employee.NewPostgresDocCategoryRepo(pool)
	employeeDocRepo := employee.NewPostgresEmployeeDocRepo(pool)
	absenceRepo := absence.NewPostgresAbsenceRepo(pool)
	workTimeRepo := timetracking.NewPostgresWorkTimeRepo(pool)
	breakRepo := timetracking.NewPostgresBreakRepo(pool)

	leaveSvc := leave.NewService(leaveRequestRepo, leaveBalanceRepo, leaveTypeRepo, hrSettingsRepo, employeeRepo)
	employeeSvc := employee.NewService(employeeRepo, docCategoryRepo, employeeDocRepo)
	absenceSvc := absence.NewService(absenceRepo, hrSettingsRepo)
	timetrackingSvc := timetracking.NewService(workTimeRepo, breakRepo, employeeRepo, hrSettingsRepo, pool)

	hrGRPC := server.NewHRGRPCServer(leaveSvc, timetrackingSvc, employeeSvc, absenceSvc, hrSettingsRepo)
	hrv1.RegisterHRServiceServer(grpcServer, hrGRPC)

	slog.Info("HR services registered on biz gRPC server")

	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.BizGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.BizGRPCPort, "error", err)
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
		Addr:         cfg.BizHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.BizHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("biz service starting", "port", cfg.BizGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down biz service")
	case <-ctx.Done():
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("biz service stopped")
}
