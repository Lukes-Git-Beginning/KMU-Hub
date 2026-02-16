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

	"github.com/redis/go-redis/v9"

	"github.com/kmuhub/kmuhub/internal/config"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/work/calendar"
	"github.com/kmuhub/kmuhub/internal/work/comment"
	"github.com/kmuhub/kmuhub/internal/work/event"
	"github.com/kmuhub/kmuhub/internal/work/holiday"
	"github.com/kmuhub/kmuhub/internal/work/livekit"
	"github.com/kmuhub/kmuhub/internal/work/meeting"
	"github.com/kmuhub/kmuhub/internal/work/presence"
	"github.com/kmuhub/kmuhub/internal/work/project"
	"github.com/kmuhub/kmuhub/internal/work/reaction"
	"github.com/kmuhub/kmuhub/internal/work/recording"
	"github.com/kmuhub/kmuhub/internal/work/resource"
	wstatus "github.com/kmuhub/kmuhub/internal/work/status"
	"github.com/kmuhub/kmuhub/internal/work/task"
	"github.com/kmuhub/kmuhub/internal/work/timeentry"
	"github.com/kmuhub/kmuhub/internal/work/video"
	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
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

	// Initialize repositories
	projectRepo := project.NewPostgresRepository(pool)
	statusRepo := wstatus.NewPostgresRepository(pool)
	taskRepo := task.NewPostgresRepository(pool)
	commentRepo := comment.NewPostgresRepository(pool)
	timeEntryRepo := timeentry.NewPostgresRepository(pool)

	// Initialize services
	projectService := project.NewService(projectRepo)
	statusService := wstatus.NewService(statusRepo)
	taskService := task.NewService(taskRepo, projectRepo)
	commentService := comment.NewService(commentRepo, taskRepo)
	timeEntryService := timeentry.NewService(timeEntryRepo)

	// Set event emitters for notification integration
	taskService.SetEventEmitter(task.NewPGEventEmitter(pool))
	commentService.SetEventEmitter(task.NewPGEventEmitter(pool))

	// Calendar domain repositories
	calendarRepo := calendar.NewPostgresRepository(pool)
	eventRepo := event.NewPostgresRepository(pool)
	resourceRepo := resource.NewPostgresRepository(pool)
	holidayRepo := holiday.NewPostgresRepository(pool)

	// Calendar domain services
	calendarService := calendar.NewService(calendarRepo)
	livekitService := livekit.NewService(cfg.LiveKitAPIKey, cfg.LiveKitAPISecret, cfg.LiveKitWSURL)
	nagerClient := holiday.NewNagerClient()
	holidayService := holiday.NewService(holidayRepo, nagerClient)
	resourceService := resource.NewService(resourceRepo)
	eventService := event.NewService(eventRepo, calendarRepo)
	eventService.SetEventEmitter(event.NewPGEventEmitter(pool))

	// Redis client (for presence store)
	redisOpts, redisErr := redis.ParseURL(cfg.RedisURL)
	if redisErr != nil {
		slog.Error("failed to parse redis url", "error", redisErr)
		os.Exit(1)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()
	if pingErr := redisClient.Ping(ctx).Err(); pingErr != nil {
		slog.Warn("redis not available, presence will be degraded", "error", pingErr)
	}

	// Video domain repositories
	videoRepo := video.NewPostgresRepository(pool)
	meetingRepo := meeting.NewPostgresRepository(pool)
	recordingRepo := recording.NewPostgresRepository(pool)
	reactionRepo := reaction.NewPostgresRepository(pool)
	presenceConfigRepo := presence.NewPostgresConfigRepository(pool)

	// Video domain services
	var roomMgr video.RoomManager
	if cfg.LiveKitAPIKey != "" && cfg.LiveKitAPISecret != "" {
		roomMgr = livekit.NewRoomManager(cfg.LiveKitAPIKey, cfg.LiveKitAPISecret, cfg.LiveKitWSURL)
	}
	videoService := video.NewService(videoRepo, roomMgr)

	var egressMgr recording.EgressManager
	if cfg.LiveKitAPIKey != "" && cfg.LiveKitAPISecret != "" && cfg.LiveKitEgressTemplateURL != "" {
		egressMgr = livekit.NewEgressManager(cfg.LiveKitAPIKey, cfg.LiveKitAPISecret, cfg.LiveKitWSURL)
	}
	recordingService := recording.NewService(recordingRepo, egressMgr, cfg.LiveKitEgressTemplateURL, recording.S3Config{
		Endpoint:  cfg.MinIOEndpoint,
		AccessKey: cfg.MinIOAccessKey,
		Secret:    cfg.MinIOSecretKey,
		Bucket:    cfg.MinIOBucket,
	})

	meetingService := meeting.NewService(meetingRepo)
	presenceStore := presence.NewRedisStore(redisClient)
	presenceService := presence.NewService(presenceStore, presenceConfigRepo)
	reactionService := reaction.NewService(reactionRepo)

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
	workGRPC := server.NewWorkGRPCServer(projectService, statusService, taskService, taskRepo, commentService, timeEntryService)
	workv1.RegisterWorkServiceServer(grpcServer, workGRPC)

	// Register CalendarService gRPC server (same binary, same port as WorkService)
	calendarGRPC := server.NewCalendarGRPCServer(calendarService, eventService, resourceService, holidayService, livekitService)
	calv1.RegisterCalendarServiceServer(grpcServer, calendarGRPC)

	// Register VideoService gRPC server (same binary, same port as WorkService + CalendarService)
	videoGRPC := server.NewVideoGRPCServer(videoService, meetingService, recordingService, presenceService, reactionService, taskService)
	videov1.RegisterVideoServiceServer(grpcServer, videoGRPC)

	// Initialize gRPC metrics after service registration
	metricsRegistry.InitializeGRPCMetrics(grpcServer)

	lis, err := net.Listen("tcp", cfg.WorkGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.WorkGRPCPort, "error", err)
		os.Exit(1)
	}

	// Health + metrics HTTP server
	healthCheckers := []health.Checker{
		health.NewPostgresChecker(pool),
		health.NewRedisChecker(redisClient),
	}

	healthRouter := chi.NewRouter()
	healthRouter.Get("/health", server.HealthHandler(healthCheckers))
	healthRouter.Handle("/metrics", metricsRegistry.Handler())

	healthSrv := &http.Server{
		Addr:         cfg.WorkHealthPort,
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("health/metrics server starting", "port", cfg.WorkHealthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health/metrics server failed", "error", err)
		}
	}()

	go func() {
		slog.Info("work service starting", "port", cfg.WorkGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down work service")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
	slog.Info("work service stopped")
}
