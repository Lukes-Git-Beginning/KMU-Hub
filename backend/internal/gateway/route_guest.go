package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/chat/guest"
	"github.com/kmuhub/kmuhub/internal/server/response"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

// guestSessionContextKey is the context key for the validated guest session.
type guestSessionContextKey struct{}

// GuestRoutes handles public HTTP routes for guest chat operations.
type GuestRoutes struct {
	guestService *guest.Service
	registry     *ServiceRegistry
}

// NewGuestRoutes creates a new GuestRoutes.
func NewGuestRoutes(guestService *guest.Service, registry *ServiceRegistry) *GuestRoutes {
	return &GuestRoutes{
		guestService: guestService,
		registry:     registry,
	}
}

func (g *GuestRoutes) getChatClient() (chatv1.ChatServiceClient, error) {
	conn, err := g.registry.GetConnection("chat")
	if err != nil {
		return nil, err
	}
	return chatv1.NewChatServiceClient(conn), nil
}

// RegisterPublicRoutes registers guest chat routes on the router (no JWT auth middleware).
func (g *GuestRoutes) RegisterPublicRoutes(r chi.Router) {
	r.Route("/api/v1/guest", func(r chi.Router) {
		// Public session management
		r.Post("/sessions", g.HandleCreateSession)
		r.Post("/sessions/validate", g.HandleValidateToken)

		// Guest-authenticated routes (X-Guest-Token header)
		r.Group(func(r chi.Router) {
			r.Use(g.guestAuthMiddleware)
			r.Get("/channels/{channelID}/messages", g.HandleGetMessages)
			r.Post("/channels/{channelID}/messages", g.HandleSendMessage)
			r.Get("/config", g.HandleGetConfig)
		})
	})
}

// guestAuthMiddleware validates the X-Guest-Token header and stores the session in context.
func (g *GuestRoutes) guestAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Guest-Token")
		if token == "" {
			response.Error(w, http.StatusUnauthorized, "X-Guest-Token header required")
			return
		}

		session, err := g.guestService.ValidateToken(r.Context(), token)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "invalid or expired guest token")
			return
		}

		ctx := context.WithValue(r.Context(), guestSessionContextKey{}, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getGuestSession(ctx context.Context) *guest.GuestSession {
	session, _ := ctx.Value(guestSessionContextKey{}).(*guest.GuestSession)
	return session
}

// ============================================================================
// Handlers
// ============================================================================

type createSessionRequest struct {
	ChannelID   string  `json:"channel_id"`
	DisplayName string  `json:"display_name"`
	Email       *string `json:"email,omitempty"`
}

type createSessionResponse struct {
	Token   string              `json:"token"`
	Session *guest.GuestSession `json:"session"`
}

func (g *GuestRoutes) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	channelID, err := uuid.Parse(req.ChannelID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel_id")
		return
	}

	if req.DisplayName == "" {
		response.Error(w, http.StatusBadRequest, "display_name is required")
		return
	}

	// Extract IP and user agent from request
	ip := r.RemoteAddr
	ua := r.UserAgent()

	result, err := g.guestService.CreateSession(r.Context(), guest.CreateSessionInput{
		ChannelID:   channelID,
		DisplayName: req.DisplayName,
		Email:       req.Email,
		IPAddress:   &ip,
		UserAgent:   &ua,
	})
	if err != nil {
		slog.Error("failed to create guest session", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	response.JSON(w, http.StatusCreated, createSessionResponse{
		Token:   result.Token,
		Session: result.Session,
	})
}

type validateTokenRequest struct {
	Token string `json:"token"`
}

func (g *GuestRoutes) HandleValidateToken(w http.ResponseWriter, r *http.Request) {
	var req validateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		response.Error(w, http.StatusBadRequest, "token is required")
		return
	}

	session, err := g.guestService.ValidateToken(r.Context(), req.Token)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid or expired guest token")
		return
	}

	// Return session info + config
	config, _ := g.guestService.GetConfig(r.Context(), session.ChannelID)

	type resp struct {
		Session *guest.GuestSession      `json:"session"`
		Config  *guest.GuestChannelConfig `json:"config,omitempty"`
	}

	response.JSON(w, http.StatusOK, resp{
		Session: session,
		Config:  config,
	})
}

func (g *GuestRoutes) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	client, err := g.getChatClient()
	if err != nil {
		response.Error(w, http.StatusServiceUnavailable, "chat service unavailable")
		return
	}

	session := getGuestSession(r.Context())
	if session == nil {
		response.Error(w, http.StatusUnauthorized, "guest session required")
		return
	}

	channelID := chi.URLParam(r, "channelID")
	if channelID != session.ChannelID.String() {
		response.Error(w, http.StatusForbidden, "cannot access other channels")
		return
	}

	limit := parseLimit(r, 50, 100)

	grpcReq := &chatv1.GetMessagesRequest{
		ChannelId: channelID,
		UserId:    uuid.Nil.String(), // Guest -- service will bypass membership for guest-enabled channels
		Limit:     int32(limit),
	}

	if before := r.URL.Query().Get("before"); before != "" {
		grpcReq.Before = &before
	}

	resp, err := client.GetMessages(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type guestSendMessageRequest struct {
	Content string `json:"content"`
}

func (g *GuestRoutes) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	client, err := g.getChatClient()
	if err != nil {
		response.Error(w, http.StatusServiceUnavailable, "chat service unavailable")
		return
	}

	session := getGuestSession(r.Context())
	if session == nil {
		response.Error(w, http.StatusUnauthorized, "guest session required")
		return
	}

	channelID := chi.URLParam(r, "channelID")
	if channelID != session.ChannelID.String() {
		response.Error(w, http.StatusForbidden, "cannot send to other channels")
		return
	}

	var req guestSendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		response.Error(w, http.StatusBadRequest, "content is required")
		return
	}

	// Check rate limit
	if err := g.guestService.CheckRateLimit(session.ID); err != nil {
		response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	sessionID := session.ID.String()
	grpcReq := &chatv1.SendMessageRequest{
		ChannelId:      channelID,
		Content:        req.Content,
		GuestSessionId: &sessionID,
	}

	resp, err := client.SendMessage(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (g *GuestRoutes) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	session := getGuestSession(r.Context())
	if session == nil {
		response.Error(w, http.StatusUnauthorized, "guest session required")
		return
	}

	config, err := g.guestService.GetConfig(r.Context(), session.ChannelID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "config not found")
		return
	}

	// Return only public config fields
	type publicConfig struct {
		WelcomeMessage string  `json:"welcome_message"`
		LogoURL        *string `json:"logo_url,omitempty"`
		PrimaryColor   string  `json:"primary_color"`
		MaxFileSizeMB  int     `json:"max_file_size_mb"`
		AllowedTypes   string  `json:"allowed_file_types"`
	}

	response.JSON(w, http.StatusOK, publicConfig{
		WelcomeMessage: config.WelcomeMessage,
		LogoURL:        config.LogoURL,
		PrimaryColor:   config.PrimaryColor,
		MaxFileSizeMB:  config.MaxFileSizeMB,
		AllowedTypes:   config.AllowedFileTypes,
	})
}
