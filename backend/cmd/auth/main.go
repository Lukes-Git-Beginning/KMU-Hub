package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/server"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
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

	repo := auth.NewPostgresRepository(pool)
	tokenMaker := auth.NewTokenMaker(cfg.JWTSecret, cfg.AccessTokenExpiry, cfg.RefreshTokenExpiry)
	authService := auth.NewService(repo, tokenMaker)

	grpcServer := grpc.NewServer()
	authGRPC := server.NewAuthGRPCServer(authService)
	authv1.RegisterAuthServiceServer(grpcServer, authGRPC)

	lis, err := net.Listen("tcp", cfg.AuthGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.AuthGRPCPort, "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("auth service starting", "port", cfg.AuthGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down auth service")
	case <-ctx.Done():
	}

	grpcServer.GracefulStop()
	slog.Info("auth service stopped")
}
