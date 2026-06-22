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
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/work/calendar"
	"github.com/kmuhub/kmuhub/internal/work/comment"
	"github.com/kmuhub/kmuhub/internal/work/customfield"
	"github.com/kmuhub/kmuhub/internal/work/event"
	"github.com/kmuhub/kmuhub/internal/work/holiday"
	"github.com/kmuhub/kmuhub/internal/work/label"
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

	cfg, err := config.Load(ctx, config.RequireMinIO)
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
	bookingRepo := calendar.NewPostgresBookingRepository(pool)

	// Calendar domain services
	calendarService := calendar.NewService(calendarRepo)
	bookingService := calendar.NewBookingService(bookingRepo)
	// NewServiceWithTURN: pass TURN_SECRET + COTURN_HOST from config.
	// Both default to "" when not set, so TURN is transparently disabled
	// until the coturn CPX11 is provisioned and the env vars are populated.
	livekitService := livekit.NewServiceWithTURN(
		cfg.LiveKitAPIKey, cfg.LiveKitAPISecret, cfg.LiveKitWSURL,
		cfg.TURNSecret, cfg.COTURNHost,
	)
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
		// NewRoomManagerWithTURN: pass TURN_SECRET + COTURN_HOST from config.
		// Both default to "" when not set, transparently disabling TURN until
		// the coturn CAX11 env vars are populated.
		// Room management talks to the LiveKit server API — use the internal
		// docker-network URL, NOT the public WSS URL (unroutable from here).
		roomMgr = livekit.NewRoomManagerWithTURN(
			cfg.LiveKitAPIKey, cfg.LiveKitAPISecret, cfg.LiveKitServerAPIURL(),
			cfg.TURNSecret, cfg.COTURNHost,
		)
	}
	videoService := video.NewService(videoRepo, roomMgr)

	var egressMgr recording.EgressManager
	if cfg.LiveKitAPIKey != "" && cfg.LiveKitAPISecret != "" && cfg.LiveKitEgressTemplateURL != "" {
		egressMgr = livekit.NewEgressManager(cfg.LiveKitAPIKey, cfg.LiveKitAPISecret, cfg.LiveKitServerAPIURL())
	}
	recordingService := recording.NewService(recordingRepo, egressMgr, cfg.LiveKitEgressTemplateURL, recording.S3Config{
		Endpoint:  cfg.MinIOEndpoint,
		AccessKey: cfg.MinIOAccessKey,
		Secret:    cfg.MinIOSecretKey,
		Bucket:    cfg.MinIOBucket,
	})

	meetingService := meeting.NewServiceWithRoomManager(meetingRepo, roomMgr)
	presenceStore := presence.NewRedisStore(redisClient)
	presenceService := presence.NewService(presenceStore, presenceConfigRepo)
	reactionService := reaction.NewService(reactionRepo)

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
	// Label and custom field repositories + services
	labelRepo := label.NewPostgresRepository(pool)
	labelService := label.NewService(labelRepo)
	customFieldRepo := customfield.NewPostgresRepository(pool)
	customFieldService := customfield.NewService(customFieldRepo)

	workGRPC := server.NewWorkGRPCServer(projectService, statusService, taskService, taskRepo, commentService, timeEntryService, labelService, customFieldService)
	workv1.RegisterWorkServiceServer(grpcServer, workGRPC)

	// Register CalendarService gRPC server (same binary, same port as WorkService)
	calendarGRPC := server.NewCalendarGRPCServer(calendarService, eventService, resourceService, holidayService, livekitService)
	calendarGRPC.SetBookingService(bookingService)
	calv1.RegisterCalendarServiceServer(grpcServer, calendarGRPC)

	// Register VideoService gRPC server (same binary, same port as WorkService + CalendarService)
	// roomMgr doubles as the meeting-join token generator; cfg.LiveKitWSURL is the
	// PUBLIC signaling URL embedded in join responses for clients.
	videoGRPC := server.NewVideoGRPCServer(videoService, meetingService, recordingService, presenceService, reactionService, taskService, roomMgr, cfg.LiveKitWSURL)
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
	server.RegisterHealth(healthRouter, "/health", healthCheckers)
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

	// Start recording cleanup goroutine.
	// Runs once after a 1-minute startup delay, then every 24 hours.
	// Uses a system context so the cross-tenant query bypasses RLS row filtering
	// (recordings table has no app-visible tenant_id enforced at this path).
	// Errors are logged but never propagate — a transient failure must not crash
	// or degrade any other service in this binary.
	go func() {
		cleanupCtx := database.WithSystemContext(ctx)
		select {
		case <-time.After(1 * time.Minute):
		case <-ctx.Done():
			return
		}

		cleanupTicker := time.NewTicker(24 * time.Hour)
		defer cleanupTicker.Stop()

		runCleanup := func() {
			n, err := recordingService.CleanupExpiredRecordings(cleanupCtx)
			if err != nil {
				slog.Error("recording cleanup failed", "error", err)
				return
			}
			slog.Info("recording cleanup completed", "deleted", n)
		}

		runCleanup() // initial run immediately after startup delay

		for {
			select {
			case <-cleanupTicker.C:
				runCleanup()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start meeting auto-close sweeper goroutine.
	// Closes stale in_progress meetings (scheduled_end + grace < now) that have
	// no active LiveKit participants. Only active when LiveKit is configured.
	// Runs every 5 min after a 1-min startup delay.
	// Uses a system context so the cross-tenant query bypasses RLS row filtering.
	// Errors are logged but never propagate — a transient failure must not crash
	// or degrade any other service in this binary.
	startMeetingSweeper(ctx, meetingService, roomMgr, time.Duration(cfg.MeetingAutocloseGraceMinutes)*time.Minute)

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

// meetingSweepRoomManager is the subset of video.RoomManager needed by the sweeper.
type meetingSweepRoomManager interface {
	ListParticipants(ctx context.Context, roomName string) ([]string, error)
}

// meetingSweepService is the subset of meeting.Service needed by the sweeper.
type meetingSweepService interface {
	ListStaleMeetings(ctx context.Context, cutoff time.Time) ([]meeting.Meeting, error)
	CompleteMeetingByRoom(ctx context.Context, roomName string) error
}

// startMeetingSweeper launches the backstop auto-close goroutine.
// Noop when mgr is nil (LiveKit not configured).
func startMeetingSweeper(ctx context.Context, svc meetingSweepService, mgr meetingSweepRoomManager, grace time.Duration) {
	if mgr == nil {
		slog.Info("meeting sweeper disabled: LiveKit not configured")
		return
	}

	go func() {
		sweeperCtx := database.WithSystemContext(ctx)

		select {
		case <-time.After(1 * time.Minute):
		case <-ctx.Done():
			return
		}

		sweeperTicker := time.NewTicker(5 * time.Minute)
		defer sweeperTicker.Stop()

		runSweeper := func() {
			cutoff := time.Now().UTC().Add(-grace)
			stale, err := svc.ListStaleMeetings(sweeperCtx, cutoff)
			if err != nil {
				slog.Error("meeting sweeper: list stale meetings failed", "error", err)
				return
			}
			for _, m := range stale {
				if m.RoomName == nil || *m.RoomName == "" {
					slog.Warn("meeting sweeper: stale meeting has no room_name — skipping",
						"meeting_id", m.ID,
					)
					continue
				}
				participants, listErr := mgr.ListParticipants(sweeperCtx, *m.RoomName)
				if listErr != nil {
					slog.Warn("meeting sweeper: list participants failed — skipping",
						"meeting_id", m.ID,
						"room_name", *m.RoomName,
						"error", listErr,
					)
					continue
				}
				if len(participants) > 0 {
					slog.Debug("meeting sweeper: active participants present — not closing",
						"meeting_id", m.ID,
						"room_name", *m.RoomName,
						"participant_count", len(participants),
					)
					continue
				}
				// Room empty → close meeting. CompleteMeetingByRoom is idempotent.
				if endErr := svc.CompleteMeetingByRoom(sweeperCtx, *m.RoomName); endErr != nil {
					slog.Error("meeting sweeper: complete meeting failed",
						"meeting_id", m.ID,
						"room_name", *m.RoomName,
						"error", endErr,
					)
					continue
				}
				slog.Info("meeting sweeper: auto-closed stale meeting",
					"meeting_id", m.ID,
					"room_name", *m.RoomName,
					"scheduled_end", m.ScheduledEnd,
				)
			}
		}

		runSweeper() // initial run immediately after startup delay

		for {
			select {
			case <-sweeperTicker.C:
				runSweeper()
			case <-ctx.Done():
				return
			}
		}
	}()
}
