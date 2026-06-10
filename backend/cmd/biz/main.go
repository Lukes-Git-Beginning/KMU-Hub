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

	"github.com/kmuhub/kmuhub/internal/biz/bexio"
	"github.com/kmuhub/kmuhub/internal/biz/creditnote"
	"github.com/kmuhub/kmuhub/internal/biz/dashboard"
	"github.com/kmuhub/kmuhub/internal/biz/datev"
	"github.com/kmuhub/kmuhub/internal/biz/dunning"
	"github.com/kmuhub/kmuhub/internal/biz/einvoice"
	"github.com/kmuhub/kmuhub/internal/biz/gobdarchive"
	"github.com/kmuhub/kmuhub/internal/biz/hr/absence"
	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/lexware"
	"github.com/kmuhub/kmuhub/internal/biz/payment"
	"github.com/kmuhub/kmuhub/internal/biz/pdf"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/cache"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/security/vault"
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

	cfg, err := config.Load(ctx, config.RequireVault, config.RequireMinIO)
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
		slog.Warn("redis unavailable for biz cache, using direct db", "error", redisErr)
	}
	cacheClient := cache.NewClient(redisClient)

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
	dashboardRepo := dashboard.NewCachedRepository(
		dashboard.NewPostgresRepository(pool), cacheClient,
	)

	// Shared repositories (used by multiple services)
	numberSeqRepo := quote.NewPostgresNumberSequenceRepo(pool)
	companySettingsRepo := quote.NewPostgresCompanySettingsRepo(pool)
	// workTimeRepo is shared: used by BizGRPCServer (CreateInvoiceFromTimeEntries)
	// and by the HR gRPC server (time tracking). Initialized here so it can be
	// passed to NewBizGRPCServer before the HR services block.
	workTimeRepo := timetracking.NewPostgresWorkTimeRepo(pool)

	// =========================================================================
	// CRM gRPC Client (for DealValueUpdater + CreateQuoteFromDeal)
	// =========================================================================
	var dealUpdater quote.DealValueUpdater
	var crmServiceClient crmv1.CRMServiceClient
	if cfg.CRMGRPCAddress != "" {
		crmConn, crmErr := grpc.NewClient(
			cfg.CRMGRPCAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			// Propagate tenant/user from the inbound biz context to CRM —
			// CRM handlers read the tenant from metadata and answer
			// Unauthenticated without it (CreateQuoteFromDeal orchestration).
			grpc.WithUnaryInterceptor(middleware.TenantOutboundUnaryInterceptor()),
		)
		if crmErr != nil {
			slog.Warn("failed to connect to CRM service, deal value sync and quote-from-deal disabled",
				"address", cfg.CRMGRPCAddress,
				"error", crmErr,
			)
		} else {
			defer crmConn.Close()
			crmServiceClient = crmv1.NewCRMServiceClient(crmConn)
			dealUpdater = &crmDealUpdater{crmClient: crmServiceClient}
			slog.Info("CRM client enabled (deal value sync + quote-from-deal)", "address", cfg.CRMGRPCAddress)
		}
	} else {
		slog.Info("CRM gRPC address not configured, deal value sync and quote-from-deal disabled")
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
			AccentColor:              "#1a73e8",
			DefaultPaymentTermsDays:  30,
			DefaultQuoteValidityDays: 30,
		}
	}
	pdfGen := pdf.NewGenerator(*settings)

	// DATEV exporter
	datevExp := datev.NewExporter()

	// =========================================================================
	// GoBD Belegarchiv — MinIO file store + service (§147 AO)
	// =========================================================================
	var gobdArchiveSvc *gobdarchive.Service
	gobdFileStore, minioErr := chatfile.NewMinIOStore(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
	)
	if minioErr != nil {
		slog.Error("failed to connect to minio for gobd archive", "error", minioErr)
		os.Exit(1)
	}
	gobdRepo := gobdarchive.NewPostgresRepository(pool)
	gobdArchiveSvc = gobdarchive.NewService(gobdRepo, gobdFileStore, invoiceSvc)
	slog.Info("gobd archive service initialized")

	// Incoming e-invoice processing (E-Rechnung Eingang)
	einvoiceRepo := einvoice.NewPostgresRepository(pool)
	einvoiceSvc := einvoice.NewService(einvoiceRepo)

	// =========================================================================
	// gRPC Server
	// =========================================================================
	metricsRegistry := metrics.NewRegistry()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metricsRegistry.GRPCUnaryInterceptor(),
			middleware.TenantInboundUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			metricsRegistry.GRPCStreamInterceptor(),
		),
		grpc.MaxRecvMsgSize(60<<20),
		grpc.MaxSendMsgSize(60<<20),
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
		workTimeRepo,
		crmServiceClient,
		gobdArchiveSvc,
		einvoiceSvc,
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
	// workTimeRepo is already initialized above (shared with BizGRPCServer)
	breakRepo := timetracking.NewPostgresBreakRepo(pool)

	leaveSvc := leave.NewService(leaveRequestRepo, leaveBalanceRepo, leaveTypeRepo, hrSettingsRepo, employeeRepo)
	employeeSvc := employee.NewService(employeeRepo, docCategoryRepo, employeeDocRepo)
	absenceSvc := absence.NewService(absenceRepo, hrSettingsRepo)
	timetrackingSvc := timetracking.NewService(workTimeRepo, breakRepo, employeeRepo, hrSettingsRepo, pool)

	hrGRPC := server.NewHRGRPCServer(leaveSvc, timetrackingSvc, employeeSvc, absenceSvc, hrSettingsRepo)
	hrv1.RegisterHRServiceServer(grpcServer, hrGRPC)

	slog.Info("HR services registered on biz gRPC server")

	// =========================================================================
	// Bexio Integration (optional, depends on BEXIO_CLIENT_ID being set)
	// =========================================================================
	var bexioSvc *bexio.Service
	if cfg.BexioClientID != "" {
		bexioConfig := bexio.DefaultClientConfig(cfg.BexioClientID, cfg.BexioClientSecret, cfg.BexioRedirectURL)
		bexioRepo := bexio.NewPostgresRepository(pool)
		bexioConfigRepo := bexio.NewPostgresIntegrationConfigRepo(pool)

		var vaultSvc bexio.VaultService
		if cfg.VaultMasterSecret != "" {
			vaultRepo := vault.NewPostgresRepository(pool)
			vs, vaultErr := vault.NewService(vaultRepo, cfg.VaultMasterSecret)
			if vaultErr != nil {
				slog.Error("failed to initialize vault for bexio", "error", vaultErr)
				os.Exit(1)
			}
			vaultSvc = vs
		} else {
			slog.Warn("VAULT_MASTER_SECRET not set, Bexio token storage unavailable")
		}

		bexioClient := bexio.NewClient(bexioConfig, vaultSvc)
		bexioSvc = bexio.NewService(
			bexioClient, bexioRepo, bexioConfigRepo, vaultSvc, bexioConfig,
			nil, // ContactService: wired via CRM gRPC in gateway (not available in biz binary)
			invoiceSvc, invoiceSvc, quoteRepo,
		)

		bexioGRPC := server.NewBexioGRPCServer(bexioSvc)
		bizv1.RegisterBexioIntegrationServiceServer(grpcServer, bexioGRPC)

		if err := bexioSvc.StartScheduler(ctx); err != nil {
			slog.Warn("bexio scheduler start failed", "error", err)
		} else {
			slog.Info("bexio integration enabled, scheduler started")
		}
	} else {
		slog.Info("Bexio integration disabled (BEXIO_CLIENT_ID not set)")
	}

	// =========================================================================
	// Lexware Office Integration (optional, depends on vault)
	// =========================================================================
	var lexwareSvc *lexware.Service
	if cfg.VaultMasterSecret != "" {
		lexwareConfig := lexware.DefaultClientConfig()
		if cfg.LexwareAPIBaseURL != "" {
			lexwareConfig.BaseURL = cfg.LexwareAPIBaseURL
		}
		lexwareRepo := lexware.NewPostgresRepository(pool)
		lexwareConfigRepo := lexware.NewPostgresIntegrationConfigRepo(pool)

		var lexwareVault lexware.VaultService
		vaultRepo := vault.NewPostgresRepository(pool)
		vs, vaultErr := vault.NewService(vaultRepo, cfg.VaultMasterSecret)
		if vaultErr != nil {
			slog.Error("failed to initialize vault for lexware", "error", vaultErr)
		} else {
			lexwareVault = vs
		}

		if lexwareVault != nil {
			lexwareSvc = lexware.NewService(
				lexwareRepo, lexwareConfigRepo, lexwareVault,
				nil, // ContactService
				invoiceSvc, quoteRepo,
			)

			lexwareGRPC := server.NewLexwareGRPCServer(lexwareSvc)
			bizv1.RegisterLexwareIntegrationServiceServer(grpcServer, lexwareGRPC)

			if err := lexwareSvc.StartScheduler(ctx); err != nil {
				slog.Warn("lexware scheduler start failed", "error", err)
			} else {
				slog.Info("lexware integration enabled, scheduler started")
			}
		}
	} else {
		slog.Info("Lexware integration disabled (VAULT_MASTER_SECRET not set)")
	}

	// =========================================================================
	// DATEV Upload Service (optional, depends on DATEV_CLIENT_ID)
	// =========================================================================
	var datevUploadSvc *datev.UploadService
	if cfg.DatevClientID != "" {
		var datevVault datev.VaultService
		if cfg.VaultMasterSecret != "" {
			vaultRepo := vault.NewPostgresRepository(pool)
			vs, vaultErr := vault.NewService(vaultRepo, cfg.VaultMasterSecret)
			if vaultErr != nil {
				slog.Error("failed to initialize vault for datev", "error", vaultErr)
			} else {
				datevVault = vs
			}
		}

		if datevVault != nil {
			datevOAuth := datev.NewOAuthManager(datevVault, cfg.DatevClientID, cfg.DatevClientSecret, cfg.DatevTokenURL)
			datevUploader := datev.NewUploader(datevOAuth, cfg.DatevAPIBaseURL)
			datevBelegUploader := datev.NewBelegbilderUploader(datevOAuth, cfg.DatevAPIBaseURL)
			datevUploadRepo := datev.NewPostgresUploadRepository(pool)
			datevConfigRepo := datev.NewPostgresIntegrationConfigRepo(pool)

			datevUploadSvc = datev.NewUploadService(datevExp, datevUploader, datevBelegUploader, datevUploadRepo, datevConfigRepo, datevOAuth)

			datevUploadGRPC := server.NewDatevUploadGRPCServer(datevUploadSvc)
			bizv1.RegisterDatevUploadServiceServer(grpcServer, datevUploadGRPC)

			slog.Info("DATEV upload service enabled")
		}
	} else {
		slog.Info("DATEV upload service disabled (DATEV_CLIENT_ID not set)")
	}

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
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
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
	if lexwareSvc != nil {
		lexwareSvc.StopScheduler()
		slog.Info("lexware scheduler stopped")
	}

	if bexioSvc != nil {
		bexioSvc.StopScheduler()
		slog.Info("bexio scheduler stopped")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("biz service stopped")
}
