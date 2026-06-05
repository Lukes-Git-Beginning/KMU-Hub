package main

import (
	"context"
	"fmt"
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
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/email/attachment"
	emailcontact "github.com/kmuhub/kmuhub/internal/email/contact"
	"github.com/kmuhub/kmuhub/internal/email/contactlink"
	"github.com/kmuhub/kmuhub/internal/email/message"
	"github.com/kmuhub/kmuhub/internal/email/send"
	"github.com/kmuhub/kmuhub/internal/email/signature"
	emailsync "github.com/kmuhub/kmuhub/internal/email/sync"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/security/vault"
	"github.com/kmuhub/kmuhub/internal/server"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

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

	// MinIO client for attachment storage
	minioClient, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		slog.Error("failed to connect to minio", "error", err)
		os.Exit(1)
	}

	// Vault encryption for email credentials
	var vaultEncryptor account.VaultEncryptor
	if cfg.VaultMasterSecret != "" {
		emailKey, keyErr := vault.DeriveKey([]byte(cfg.VaultMasterSecret), "email-credentials")
		if keyErr != nil {
			slog.Error("failed to derive email encryption key", "error", keyErr)
			os.Exit(1)
		}
		vaultEncryptor = &emailVaultAdapter{key: emailKey}
		slog.Info("vault key derived for email credential encryption")
	} else {
		slog.Warn("VAULT_MASTER_SECRET not set, email credential encryption disabled (dev mode)")
		vaultEncryptor = &noopEncryptor{}
	}

	// Initialize repositories
	accountRepo := account.NewPostgresRepository(pool)
	messageRepo := message.NewPostgresRepository(pool)
	folderRepo := message.NewPostgresFolderRepository(pool)
	signatureRepo := signature.NewPostgresRepository(pool)
	linkRepo := contactlink.NewPostgresRepository(pool)

	// Initialize services
	accountService := account.NewService(accountRepo, vaultEncryptor)
	messageService := message.NewService(messageRepo, folderRepo)
	signatureService := signature.NewService(signatureRepo)

	// Send service (needs account for SMTP creds, message for local storage, signature for appending)
	sendAccountAdapter := &sendAccountAdapter{accountService: accountService}
	sendService := send.NewService(sendAccountAdapter, messageService, signatureService)

	// Attachment service (MinIO storage)
	attachStore := attachment.NewStore(minioClient, cfg.MinIOBucket)
	attachmentService := attachment.NewService(attachment.NewPostgresRepository(pool), attachStore)

	// Sync engine (background IMAP sync)
	folderSyncAdapter := &folderSyncAdapter{repo: folderRepo}
	attachSyncAdapter := &attachSyncAdapter{svc: attachmentService}
	syncEngine := emailsync.NewEngine(accountService, messageService, folderSyncAdapter, attachSyncAdapter)
	go func() {
		if syncErr := syncEngine.Start(ctx); syncErr != nil {
			slog.Error("sync engine failed to start", "error", syncErr)
		}
	}()

	// Contact import/export (shared with CRM service via proto)
	importService := emailcontact.NewImportService(nil, slog.Default())
	exportService := emailcontact.NewExportService(nil, slog.Default())

	// Metrics
	metricsRegistry := metrics.NewRegistry()

	// gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metricsRegistry.GRPCUnaryInterceptor(),
			middleware.TenantInboundUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)

	emailGRPC := server.NewEmailGRPCServer(
		accountService,
		messageService,
		sendService,
		signatureService,
		attachmentService,
		syncEngine,
		linkRepo,
		importService,
		exportService,
	)
	emailv1.RegisterEmailServiceServer(grpcServer, emailGRPC)

	// Initialize gRPC metrics after service registration
	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.EmailGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.EmailGRPCPort, "error", err)
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
		Addr:         cfg.EmailHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.EmailHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("email service starting", "port", cfg.EmailGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down email service")
	case <-ctx.Done():
	}

	// Graceful shutdown
	syncEngine.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("email service stopped")
}

// attachSyncAdapter bridges attachment.Service -> emailsync.AttachmentStorer interface.
// The interface uses interface{} for reader while the service uses io.Reader.
type attachSyncAdapter struct {
	svc *attachment.Service
}

func (a *attachSyncAdapter) CreateFromStream(ctx context.Context, messageID uuid.UUID, accountID uuid.UUID, messageUID uint32, filename, contentType string, size int64, reader any, contentID string, isInline bool, tenantID uuid.UUID) (*models.EmailAttachment, error) {
	r, ok := reader.(io.Reader)
	if !ok {
		return nil, fmt.Errorf("reader must implement io.Reader")
	}
	return a.svc.CreateFromStream(ctx, messageID, accountID, messageUID, filename, contentType, size, r, contentID, isInline, tenantID)
}

// folderSyncAdapter bridges message.FolderRepository -> emailsync.FolderSyncer interface.
// The FolderSyncer interface uses CreateFolder while FolderRepository uses Create.
type folderSyncAdapter struct {
	repo message.FolderRepository
}

func (a *folderSyncAdapter) GetByIMAPName(ctx context.Context, accountID uuid.UUID, imapName string) (*models.EmailFolder, error) {
	return a.repo.GetByIMAPName(ctx, accountID, imapName)
}

func (a *folderSyncAdapter) CreateFolder(ctx context.Context, folder *models.EmailFolder) error {
	return a.repo.Create(ctx, folder)
}

func (a *folderSyncAdapter) UpdateUIDValidity(ctx context.Context, folderID uuid.UUID, uidValidity int64) error {
	return a.repo.UpdateUIDValidity(ctx, folderID, uidValidity)
}

func (a *folderSyncAdapter) DeleteMessagesByFolder(ctx context.Context, folderID uuid.UUID) error {
	return a.repo.DeleteMessagesByFolder(ctx, folderID)
}

func (a *folderSyncAdapter) ListByAccount(ctx context.Context, accountID uuid.UUID) ([]*models.EmailFolder, error) {
	return a.repo.ListByAccount(ctx, accountID)
}

func (a *folderSyncAdapter) UpdateCounts(ctx context.Context, folderID uuid.UUID, messageCount, unreadCount int) error {
	return a.repo.UpdateCounts(ctx, folderID, messageCount, unreadCount)
}

// sendAccountAdapter bridges account.Service -> send.AccountProvider interface.
// Both Credentials structs are structurally identical but live in different packages.
type sendAccountAdapter struct {
	accountService *account.Service
}

func (a *sendAccountAdapter) GetDecryptedCredentials(ctx context.Context, accountID uuid.UUID, tenantID uuid.UUID) (*send.Credentials, error) {
	creds, err := a.accountService.GetDecryptedCredentials(ctx, accountID, tenantID)
	if err != nil {
		return nil, err
	}
	return &send.Credentials{
		IMAPHost: creds.IMAPHost,
		IMAPPort: creds.IMAPPort,
		SMTPHost: creds.SMTPHost,
		SMTPPort: creds.SMTPPort,
		Username: creds.Username,
		Password: creds.Password,
		UseSSL:   creds.UseSSL,
	}, nil
}

// emailVaultAdapter uses a derived AES-256 key to satisfy account.VaultEncryptor.
type emailVaultAdapter struct {
	key []byte
}

func (a *emailVaultAdapter) Encrypt(_ context.Context, plaintext []byte) (string, error) {
	return vault.Encrypt(plaintext, a.key)
}

func (a *emailVaultAdapter) Decrypt(_ context.Context, encrypted string) ([]byte, error) {
	return vault.Decrypt(encrypted, a.key)
}

// noopEncryptor stores credentials as plaintext (dev mode only).
type noopEncryptor struct{}

func (n *noopEncryptor) Encrypt(_ context.Context, plaintext []byte) (string, error) {
	return string(plaintext), nil
}

func (n *noopEncryptor) Decrypt(_ context.Context, encrypted string) ([]byte, error) {
	return []byte(encrypted), nil
}
