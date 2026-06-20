package main

import (
	"context"
	"fmt"
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
	"github.com/kmuhub/kmuhub/internal/fuhrpark"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/server"
	fuhrparkv1 "github.com/kmuhub/kmuhub/proto/fuhrpark/v1"
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

	grpcPort := cfg.FuhrparkGRPCPort
	healthPort := cfg.FuhrparkHealthPort

	// Repository and service
	repo := fuhrpark.NewPostgresRepository(pool)
	svc := fuhrpark.NewService(repo)

	// Start TUEV reminder cron worker in background.
	// Wire PGEventEmitter so reminders emit pg_notify events that the notification
	// service picks up and fans out to users with the fuhrpark module enabled.
	tuevEmitter := fuhrpark.NewPGEventEmitter(pool)
	worker := fuhrpark.NewTuevWorker(repo, pool, logger).WithEventEmitter(tuevEmitter)
	go func() {
		if runErr := worker.Run(ctx, tuevLockAndRun(ctx, pool, repo, worker)); runErr != nil {
			slog.Error("fuhrpark TUEV worker exited with error", "error", runErr)
		}
	}()

	metricsRegistry := metrics.NewRegistry()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metricsRegistry.GRPCUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			metricsRegistry.GRPCStreamInterceptor(),
		),
	)

	fuhrparkGRPC := server.NewFuhrparkGRPCServer(svc)
	fuhrparkv1.RegisterFuhrparkServiceServer(grpcServer, fuhrparkGRPC)

	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		slog.Error("failed to listen", "port", grpcPort, "error", err)
		os.Exit(1)
	}

	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
	}

	healthRouter := chi.NewRouter()
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         healthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", healthPort)
		if serveErr := healthSrv.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", serveErr)
		}
	}()

	go func() {
		slog.Info("fuhrpark service starting", "port", grpcPort)
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			slog.Error("grpc server failed", "error", serveErr)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down fuhrpark service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if shutdownErr := healthSrv.Shutdown(shutdownCtx); shutdownErr != nil {
		slog.Error("health server shutdown failed", "error", shutdownErr)
	}
	slog.Info("fuhrpark service stopped")
}

// tuevLockAndRun returns a function that, on each invocation:
//  1. Begins a transaction.
//  2. Tries to acquire pg_try_advisory_xact_lock (fuhrpark_tuev_cron).
//  3. If lock acquired: runs ProcessTuevReminders and commits.
//  4. If lock not acquired: another replica is running — skip.
//
// The advisory lock is automatically released when the transaction ends.
func tuevLockAndRun(ctx context.Context, pool *pgxpool.Pool, repo fuhrpark.Repository, worker *fuhrpark.TuevWorker) func(ctx context.Context) error {
	lockKey := fuhrpark.LockKey()
	return func(ctx context.Context) error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for advisory lock: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		var acquired bool
		if scanErr := tx.QueryRow(ctx,
			`SELECT pg_try_advisory_xact_lock($1)`, lockKey,
		).Scan(&acquired); scanErr != nil {
			return fmt.Errorf("advisory lock query: %w", scanErr)
		}

		if !acquired {
			slog.Debug("fuhrpark TUEV cron: lock not acquired (another replica is running)")
			return nil
		}

		slog.Debug("fuhrpark TUEV cron: lock acquired, running scan")
		if runErr := worker.ProcessTuevReminders(ctx, time.Now()); runErr != nil {
			return fmt.Errorf("process TUEV reminders: %w", runErr)
		}

		return tx.Commit(ctx)
	}
}
