package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/metrics"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

const (
	// maxConnsPerUser limits how many concurrent WebSocket connections a single user can have.
	maxConnsPerUser = 5
	// maxMsgsPerSecond is the per-user message rate limit (sustained).
	maxMsgsPerSecond = 10
	// msgBurstSize is the per-user burst allowance above the sustained rate.
	msgBurstSize = 20
	// wsAuthProtocol is the subprotocol marker used for Sec-WebSocket-Protocol auth.
	wsAuthProtocol = "access_token"
	// wsTokenRevalidateInterval is how often in-session tokens are re-checked.
	// A revoked or expired token detected during this tick closes the connection.
	wsTokenRevalidateInterval = 5 * time.Minute
)

// GuestSessionValidator validates guest tokens and enforces rate limits.
// Uses an interface to avoid circular imports between server and guest packages.
type GuestSessionValidator interface {
	ValidateToken(ctx context.Context, token string) (*GuestSessionInfo, error)
	CheckRateLimit(sessionID string) error
	UpdateLastActivity(ctx context.Context, sessionID uuid.UUID) error
}

// GuestSessionInfo contains guest session data needed by the WebSocket hub.
type GuestSessionInfo struct {
	ID          uuid.UUID
	ChannelID   uuid.UUID
	DisplayName string
	IsActive    bool
}

// UserInfoFunc is a callback to resolve user IDs to names
type UserInfoFunc func(ctx context.Context, userID string) (firstName, lastName string, err error)

// WSPresenceService is the interface for presence operations via WebSocket.
// Uses an interface to avoid circular imports between server and presence packages.
type WSPresenceService interface {
	// Heartbeat updates the user's last-seen time and returns their current status.
	Heartbeat(ctx context.Context, userID string) (status string, err error)
	// SetManualStatus sets the user's presence status manually (online, away, dnd, offline).
	SetManualStatus(ctx context.Context, userID, status string) error
}

// WSVideoService is the interface for call signaling via WebSocket.
// Uses an interface to avoid circular imports between server and video packages.
type WSVideoService interface {
	// NotifyCallAccepted is called when a user accepts an incoming call via WebSocket.
	NotifyCallAccepted(ctx context.Context, callID, userID string) error
	// NotifyCallDeclined is called when a user declines an incoming call via WebSocket.
	NotifyCallDeclined(ctx context.Context, callID, userID string) error
}

// msgRateLimiter tracks per-user message rates using a token bucket.
type msgRateLimiter struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newMsgRateLimiter() *msgRateLimiter {
	return &msgRateLimiter{
		tokens:     msgBurstSize,
		maxTokens:  msgBurstSize,
		refillRate: maxMsgsPerSecond,
		lastRefill: time.Now(),
	}
}

func (rl *msgRateLimiter) allow() bool {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.lastRefill = now
	rl.tokens += elapsed * rl.refillRate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	if rl.tokens < 1 {
		return false
	}
	rl.tokens--
	return true
}

// WebSocketHub manages WebSocket connections and message broadcasting
type WebSocketHub struct {
	chatClient chatv1.ChatServiceClient
	tokenMaker *auth.TokenMaker

	// allowedOrigins controls which origins may open WebSocket connections.
	allowedOrigins []string

	// connections maps user ID to their connections
	connections map[string]map[*websocket.Conn]struct{}
	// channelMembers maps channel ID to user IDs
	channelMembers map[string]map[string]struct{}
	// presenceSubscribers maps target user ID to subscriber user IDs
	// (who wants to receive presence updates for which users)
	presenceSubscribers map[string]map[string]struct{}
	// userNames caches user display names for typing indicators
	userNames    map[string]struct{ firstName, lastName string }
	userInfoFunc UserInfoFunc

	// rateLimiters tracks per-user message rate limits
	rateLimiters map[string]*msgRateLimiter

	// Guest connections: maps guest session ID to their connections
	guestConnections map[string]map[*websocket.Conn]struct{}
	// Guest session channel mapping: guest session ID -> channel ID (for isolation)
	guestChannels map[string]string

	// Optional services for presence and call signaling (set after construction)
	presenceService WSPresenceService
	videoService    WSVideoService
	guestService    GuestSessionValidator

	// Optional metrics registry for Prometheus instrumentation
	metrics *metrics.Registry

	mu sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub.
// allowedOrigins controls which browser origins may connect (e.g. ["https://app.zentria.tech"]).
// An empty slice means no connections will be accepted.
func NewWebSocketHub(chatClient chatv1.ChatServiceClient, tokenMaker *auth.TokenMaker, userInfoFunc UserInfoFunc, allowedOrigins []string) *WebSocketHub {
	return &WebSocketHub{
		chatClient:          chatClient,
		tokenMaker:          tokenMaker,
		allowedOrigins:      allowedOrigins,
		connections:         make(map[string]map[*websocket.Conn]struct{}),
		channelMembers:      make(map[string]map[string]struct{}),
		presenceSubscribers: make(map[string]map[string]struct{}),
		userNames:           make(map[string]struct{ firstName, lastName string }),
		userInfoFunc:        userInfoFunc,
		rateLimiters:        make(map[string]*msgRateLimiter),
		guestConnections:    make(map[string]map[*websocket.Conn]struct{}),
		guestChannels:       make(map[string]string),
	}
}

// WebSocket message types
const (
	// Client -> Server (Chat)
	WSMessageSend        = "message.send"
	WSTypingStart        = "typing.start"
	WSTypingStop         = "typing.stop"
	WSSubscribeChannel   = "channel.subscribe"
	WSUnsubscribeChannel = "channel.unsubscribe"
	WSMarkRead           = "channel.mark_read"

	// Server -> Client (Chat)
	WSMessageNew      = "message.new"
	WSMessageUpdated  = "message.updated"
	WSMessageDeleted  = "message.deleted"
	WSThreadReplyNew  = "thread.reply.new"
	WSTypingIndicator = "typing"
	WSReadReceipt     = "read_receipt"
	WSMentionNew      = "mention.new"
	WSFileUploaded    = "file.uploaded"
	WSError           = "error"

	// Notification events (Server -> Client)
	WSNotificationNew         = "notification.new"
	WSNotificationRead        = "notification.read"
	WSNotificationReadAll     = "notification.read_all"
	WSNotificationUnreadCount = "notification.unread_count"

	// Presence events
	WSPresenceUpdate    = "presence.update"     // Server -> Client
	WSPresenceHeartbeat = "presence.heartbeat"   // Client -> Server
	WSPresenceSubscribe = "presence.subscribe"   // Client -> Server
	WSPresenceSetStatus = "presence.set_status"  // Client -> Server (manual override)

	// Call events
	WSCallIncoming = "call.incoming" // Server -> Client
	WSCallAccepted = "call.accepted" // Client -> Server
	WSCallDeclined = "call.declined" // Client -> Server
	WSCallEnded    = "call.ended"    // Server -> Client

	// Reaction events
	WSReactionToggled = "reaction.toggled" // Server -> Client
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type            string          `json:"type"`
	ChannelID       string          `json:"channel_id,omitempty"`
	Content         string          `json:"content,omitempty"`
	MessageID       string          `json:"message_id,omitempty"`
	ParentMessageID string          `json:"parent_message_id,omitempty"`
	UserID          string          `json:"user_id,omitempty"`
	FirstName       string          `json:"first_name,omitempty"`
	LastName        string          `json:"last_name,omitempty"`
	Message         json.RawMessage `json:"message,omitempty"`
	Error           string          `json:"error,omitempty"`
}

// SetPresenceService injects an optional presence service for WebSocket-based heartbeats
// and status changes. Must be called before the hub starts handling connections.
func (h *WebSocketHub) SetPresenceService(svc WSPresenceService) {
	h.presenceService = svc
}

// SetVideoService injects an optional video/call service for WebSocket-based call signaling.
// Must be called before the hub starts handling connections.
func (h *WebSocketHub) SetVideoService(svc WSVideoService) {
	h.videoService = svc
}

// SetGuestService injects an optional guest session validator for WebSocket-based guest chat.
// Must be called before the hub starts handling connections.
func (h *WebSocketHub) SetGuestService(svc GuestSessionValidator) {
	h.guestService = svc
}

// SetMetrics injects an optional metrics registry for Prometheus instrumentation.
func (h *WebSocketHub) SetMetrics(reg *metrics.Registry) {
	h.metrics = reg
}

// extractToken reads the auth token from Sec-WebSocket-Protocol header first,
// falling back to the query parameter for backwards compatibility.
// Sec-WebSocket-Protocol format: "access_token, <jwt>"
func extractToken(r *http.Request) string {
	if proto := r.Header.Get("Sec-WebSocket-Protocol"); proto != "" {
		// Parse comma-separated subprotocols; the JWT is the non-marker value
		for _, p := range strings.Split(proto, ",") {
			p = strings.TrimSpace(p)
			if p != "" && p != wsAuthProtocol {
				return p
			}
		}
	}
	return r.URL.Query().Get("token")
}

// HandleWebSocket handles WebSocket connections.
// Authentication is done via Sec-WebSocket-Protocol header (preferred)
// or query parameter (fallback): /api/v1/ws?token=<JWT> or ?guest_token=<opaque>
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check for guest token first
	guestToken := r.URL.Query().Get("guest_token")
	if guestToken != "" {
		h.handleGuestWebSocket(w, r, guestToken)
		return
	}

	// Extract token from Sec-WebSocket-Protocol or query parameter
	token := extractToken(r)
	if token == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := h.tokenMaker.ValidateAccessToken(token)
	if err != nil {
		slog.Warn("websocket auth failed", "error", err, "remote_addr", r.RemoteAddr)
		if h.metrics != nil {
			h.metrics.WSErrors.WithLabelValues("auth_failed").Inc()
		}
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	userID := claims.UserID

	// Enforce per-user connection limit
	h.mu.RLock()
	connCount := len(h.connections[userID])
	h.mu.RUnlock()
	if connCount >= maxConnsPerUser {
		slog.Warn("websocket connection limit exceeded",
			"user_id", userID,
			"current", connCount,
			"max", maxConnsPerUser,
		)
		if h.metrics != nil {
			h.metrics.WSErrors.WithLabelValues("conn_limit").Inc()
		}
		http.Error(w, "too many connections", http.StatusTooManyRequests)
		return
	}

	// Build accept options with explicit origin allowlist
	acceptOpts := &websocket.AcceptOptions{
		OriginPatterns: h.allowedOrigins,
		Subprotocols:   []string{wsAuthProtocol},
	}

	conn, err := websocket.Accept(w, r, acceptOpts)
	if err != nil {
		slog.Error("websocket accept failed", "error", err, "remote_addr", r.RemoteAddr)
		return
	}

	// Register connection and cache user name
	h.registerConnection(userID, conn)
	h.cacheUserName(r.Context(), userID)
	defer h.unregisterConnection(userID, conn)

	slog.Info("websocket connected", "user_id", userID)

	// Handle messages
	ctx := r.Context()
	h.handleConnection(ctx, conn, userID, token)
}

// handleGuestWebSocket handles WebSocket connections for guest users.
func (h *WebSocketHub) handleGuestWebSocket(w http.ResponseWriter, r *http.Request, token string) {
	if h.guestService == nil {
		http.Error(w, "guest chat not available", http.StatusServiceUnavailable)
		return
	}

	session, err := h.guestService.ValidateToken(r.Context(), token)
	if err != nil {
		slog.Warn("guest websocket auth failed", "error", err)
		http.Error(w, "invalid guest token", http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.allowedOrigins,
	})
	if err != nil {
		slog.Error("guest websocket accept failed", "error", err, "remote_addr", r.RemoteAddr)
		return
	}

	sessionID := session.ID.String()
	channelID := session.ChannelID.String()

	h.registerGuestConnection(sessionID, channelID, conn)
	defer h.unregisterGuestConnection(sessionID, conn)

	slog.Info("guest websocket connected",
		"session_id", sessionID,
		"channel_id", channelID,
		"display_name", session.DisplayName,
	)

	ctx := r.Context()
	h.handleGuestConnectionLoop(ctx, conn, session)
}

func (h *WebSocketHub) registerConnection(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[userID] == nil {
		h.connections[userID] = make(map[*websocket.Conn]struct{})
	}
	h.connections[userID][conn] = struct{}{}

	if h.metrics != nil {
		h.metrics.WSConnectionsActive.WithLabelValues("user").Inc()
	}
}

func (h *WebSocketHub) unregisterConnection(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.connections[userID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.connections, userID)
		}
	}

	// Remove user from all channel subscriptions
	for channelID, members := range h.channelMembers {
		delete(members, userID)
		if len(members) == 0 {
			delete(h.channelMembers, channelID)
		}
	}

	// Clean up presence subscriptions for disconnected user
	h.cleanupPresenceSubscriptions(userID)

	// Remove rate limiter when user has no more connections
	if _, hasConns := h.connections[userID]; !hasConns {
		delete(h.rateLimiters, userID)
	}

	if h.metrics != nil {
		h.metrics.WSConnectionsActive.WithLabelValues("user").Dec()
	}

	_ = conn.Close(websocket.StatusNormalClosure, "connection closed")
}

// handleConnection drives the read-loop for an authenticated WebSocket connection.
// rawToken is the original JWT string used during the handshake; it is re-validated
// periodically so that revoked or expired tokens are caught in-session.
func (h *WebSocketHub) handleConnection(ctx context.Context, conn *websocket.Conn, userID, rawToken string) {
	// Start the periodic token-revalidation goroutine.
	// It runs independently of the read-loop (which blocks on wsjson.Read) and
	// terminates either when ctx is cancelled (normal disconnect) or when it
	// closes the connection itself (token no longer valid).
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.revalidateTokenLoop(ctx, conn, userID, rawToken)
	}()

	for {
		var msg WSMessage
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				slog.Info("websocket closed", "user_id", userID, "status", websocket.CloseStatus(err))
			} else {
				slog.Error("websocket read error", "user_id", userID, "error", err)
			}
			// Wait for the revalidation goroutine to finish before returning so
			// that the deferred unregisterConnection in HandleWebSocket runs last.
			<-done
			return
		}

		h.handleMessage(ctx, conn, userID, &msg)
	}
}

// revalidateTokenLoop runs in a separate goroutine and re-validates rawToken every
// wsTokenRevalidateInterval. If the token has expired or been revoked the connection
// is closed with StatusPolicyViolation, which wakes the blocked wsjson.Read in
// handleConnection and causes the normal disconnect path (defer unregisterConnection)
// to execute. The goroutine exits when ctx is done or after it closes the connection.
func (h *WebSocketHub) revalidateTokenLoop(ctx context.Context, conn *websocket.Conn, userID, rawToken string) {
	h.revalidateTokenLoopWithInterval(ctx, conn, userID, rawToken, wsTokenRevalidateInterval)
}

// revalidateTokenLoopWithInterval is the internal implementation of revalidateTokenLoop
// with a configurable tick interval. It exists to allow unit tests to inject a short
// interval without waiting for the 5-minute production constant.
func (h *WebSocketHub) revalidateTokenLoopWithInterval(ctx context.Context, conn *websocket.Conn, userID, rawToken string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := h.tokenMaker.ValidateAccessToken(rawToken); err != nil {
				slog.Info("ws connection closed: token no longer valid",
					"user_id", userID,
					"reason", err,
				)
				// Close with PolicyViolation so the client knows the session ended
				// due to an auth failure (distinct from a normal closure).
				_ = conn.Close(websocket.StatusPolicyViolation, "token expired or revoked")
				return
			}
		}
	}
}

func (h *WebSocketHub) checkMsgRate(userID string) bool {
	h.mu.Lock()
	rl, ok := h.rateLimiters[userID]
	if !ok {
		rl = newMsgRateLimiter()
		h.rateLimiters[userID] = rl
	}
	allowed := rl.allow()
	h.mu.Unlock()
	return allowed
}

func (h *WebSocketHub) handleMessage(ctx context.Context, conn *websocket.Conn, userID string, msg *WSMessage) {
	if !h.checkMsgRate(userID) {
		slog.Warn("websocket message rate limit exceeded", "user_id", userID, "type", msg.Type)
		if h.metrics != nil {
			h.metrics.WSErrors.WithLabelValues("rate_limited").Inc()
		}
		h.sendError(ctx, conn, "rate limit exceeded")
		return
	}

	if h.metrics != nil {
		h.metrics.WSMessagesTotal.WithLabelValues("inbound", msg.Type).Inc()
	}

	switch msg.Type {
	case WSMessageSend:
		h.handleSendMessage(ctx, conn, userID, msg)
	case WSTypingStart, WSTypingStop:
		h.handleTypingIndicator(ctx, userID, msg)
	case WSSubscribeChannel:
		h.handleSubscribeChannel(ctx, conn, userID, msg)
	case WSUnsubscribeChannel:
		h.handleUnsubscribeChannel(userID, msg)
	case WSMarkRead:
		h.handleMarkRead(ctx, conn, userID, msg)
	case WSPresenceHeartbeat:
		h.handlePresenceHeartbeat(ctx, conn, userID)
	case WSPresenceSubscribe:
		h.handlePresenceSubscribe(ctx, conn, userID, msg)
	case WSPresenceSetStatus:
		h.handlePresenceSetStatus(ctx, conn, userID, msg)
	case WSCallAccepted:
		h.handleCallAccepted(ctx, conn, userID, msg)
	case WSCallDeclined:
		h.handleCallDeclined(ctx, conn, userID, msg)
	default:
		h.sendError(ctx, conn, "unknown message type: "+msg.Type)
	}
}

func (h *WebSocketHub) handleSendMessage(ctx context.Context, conn *websocket.Conn, userID string, msg *WSMessage) {
	if msg.ChannelID == "" || msg.Content == "" {
		h.sendError(ctx, conn, "channel_id and content are required")
		return
	}

	grpcReq := &chatv1.SendMessageRequest{
		ChannelId: msg.ChannelID,
		Content:   msg.Content,
		CreatedBy: userID,
	}

	if msg.ParentMessageID != "" {
		grpcReq.ParentMessageId = &msg.ParentMessageID
	}

	// Send message via gRPC
	resp, err := h.chatClient.SendMessage(ctx, grpcReq)
	if err != nil {
		slog.Error("failed to send message", "error", err)
		h.sendError(ctx, conn, "failed to send message")
		return
	}

	// Determine broadcast type: thread reply or regular message
	msgType := WSMessageNew
	if msg.ParentMessageID != "" {
		msgType = WSThreadReplyNew
	}

	// Broadcast to channel subscribers
	h.broadcastToChannel(ctx, msg.ChannelID, WSMessage{
		Type:            msgType,
		ChannelID:       msg.ChannelID,
		ParentMessageID: msg.ParentMessageID,
		Message:         mustMarshal(resp.Message),
	})

	// Send mention.new to mentioned users not currently subscribed to the channel
	if resp.Message != nil && len(resp.Message.Mentions) > 0 {
		h.mu.RLock()
		subscribedUsers := h.channelMembers[msg.ChannelID]
		h.mu.RUnlock()

		for _, mention := range resp.Message.Mentions {
			mentionedUserID := mention.UserId
			// Skip if user is already subscribed to the channel
			if _, subscribed := subscribedUsers[mentionedUserID]; subscribed {
				continue
			}
			h.sendToUser(ctx, mentionedUserID, WSMessage{
				Type:      WSMentionNew,
				ChannelID: msg.ChannelID,
				Message:   mustMarshal(resp.Message),
			})
		}
	}
}

func (h *WebSocketHub) handleTypingIndicator(ctx context.Context, userID string, msg *WSMessage) {
	if msg.ChannelID == "" {
		return
	}

	firstName, lastName := h.getUserName(userID)

	// Broadcast typing indicator to channel (except sender)
	h.broadcastToChannelExcept(ctx, msg.ChannelID, userID, WSMessage{
		Type:      WSTypingIndicator,
		ChannelID: msg.ChannelID,
		UserID:    userID,
		FirstName: firstName,
		LastName:  lastName,
		Content:   msg.Type, // "typing.start" or "typing.stop"
	})
}

func (h *WebSocketHub) handleSubscribeChannel(ctx context.Context, conn *websocket.Conn, userID string, msg *WSMessage) {
	if msg.ChannelID == "" {
		h.sendError(ctx, conn, "channel_id is required")
		return
	}

	// Verify user is a member of the channel
	_, err := h.chatClient.GetChannel(ctx, &chatv1.GetChannelRequest{
		Id:     msg.ChannelID,
		UserId: userID,
	})
	if err != nil {
		h.sendError(ctx, conn, "cannot subscribe to channel")
		return
	}

	h.mu.Lock()
	if h.channelMembers[msg.ChannelID] == nil {
		h.channelMembers[msg.ChannelID] = make(map[string]struct{})
	}
	h.channelMembers[msg.ChannelID][userID] = struct{}{}
	h.mu.Unlock()

	slog.Info("user subscribed to channel", "user_id", userID, "channel_id", msg.ChannelID)
}

func (h *WebSocketHub) handleUnsubscribeChannel(userID string, msg *WSMessage) {
	if msg.ChannelID == "" {
		return
	}

	h.mu.Lock()
	if members, ok := h.channelMembers[msg.ChannelID]; ok {
		delete(members, userID)
		if len(members) == 0 {
			delete(h.channelMembers, msg.ChannelID)
		}
	}
	h.mu.Unlock()

	slog.Info("user unsubscribed from channel", "user_id", userID, "channel_id", msg.ChannelID)
}

func (h *WebSocketHub) sendError(ctx context.Context, conn *websocket.Conn, errMsg string) {
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_ = wsjson.Write(writeCtx, conn, WSMessage{
		Type:  WSError,
		Error: errMsg,
	})
}

// ============================================================================
// Guest Connection Management
// ============================================================================

func (h *WebSocketHub) registerGuestConnection(sessionID, channelID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.guestConnections[sessionID] == nil {
		h.guestConnections[sessionID] = make(map[*websocket.Conn]struct{})
	}
	h.guestConnections[sessionID][conn] = struct{}{}
	h.guestChannels[sessionID] = channelID

	if h.metrics != nil {
		h.metrics.WSConnectionsActive.WithLabelValues("guest").Inc()
	}
}

func (h *WebSocketHub) unregisterGuestConnection(sessionID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.guestConnections[sessionID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.guestConnections, sessionID)
			delete(h.guestChannels, sessionID)
		}
	}

	if h.metrics != nil {
		h.metrics.WSConnectionsActive.WithLabelValues("guest").Dec()
	}

	_ = conn.Close(websocket.StatusNormalClosure, "connection closed")
}

func (h *WebSocketHub) handleGuestConnectionLoop(ctx context.Context, conn *websocket.Conn, session *GuestSessionInfo) {
	sessionID := session.ID.String()
	for {
		var msg WSMessage
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				slog.Info("guest websocket closed", "session_id", sessionID, "status", websocket.CloseStatus(err))
			} else {
				slog.Error("guest websocket read error", "session_id", sessionID, "error", err)
			}
			return
		}

		h.handleGuestMessage(ctx, conn, session, &msg)
	}
}

func (h *WebSocketHub) handleGuestMessage(ctx context.Context, conn *websocket.Conn, session *GuestSessionInfo, msg *WSMessage) {
	switch msg.Type {
	case WSMessageSend:
		h.handleGuestSendMessage(ctx, conn, session, msg)
	case WSTypingStart, WSTypingStop:
		h.handleGuestTypingIndicator(ctx, session, msg)
	default:
		h.sendError(ctx, conn, "not allowed for guests")
	}
}

func (h *WebSocketHub) handleGuestSendMessage(ctx context.Context, conn *websocket.Conn, session *GuestSessionInfo, msg *WSMessage) {
	channelID := session.ChannelID.String()

	if msg.Content == "" {
		h.sendError(ctx, conn, "content is required")
		return
	}

	// Enforce channel isolation
	if msg.ChannelID != "" && msg.ChannelID != channelID {
		h.sendError(ctx, conn, "cannot send to other channels")
		return
	}

	// Check rate limit
	if err := h.guestService.CheckRateLimit(session.ID.String()); err != nil {
		h.sendError(ctx, conn, "rate limit exceeded")
		return
	}

	sessionID := session.ID.String()
	grpcReq := &chatv1.SendMessageRequest{
		ChannelId:      channelID,
		Content:        msg.Content,
		GuestSessionId: &sessionID,
	}

	resp, err := h.chatClient.SendMessage(ctx, grpcReq)
	if err != nil {
		slog.Error("failed to send guest message", "error", err, "session_id", sessionID)
		h.sendError(ctx, conn, "failed to send message")
		return
	}

	// Broadcast to channel subscribers (users + guests)
	h.broadcastToChannel(ctx, channelID, WSMessage{
		Type:      WSMessageNew,
		ChannelID: channelID,
		Message:   mustMarshal(resp.Message),
	})
}

func (h *WebSocketHub) handleGuestTypingIndicator(ctx context.Context, session *GuestSessionInfo, msg *WSMessage) {
	channelID := session.ChannelID.String()

	h.broadcastToChannelExcept(ctx, channelID, "", WSMessage{
		Type:      WSTypingIndicator,
		ChannelID: channelID,
		UserID:    session.ID.String(),
		FirstName: session.DisplayName,
		Content:   msg.Type,
	})
}

// sendToGuest sends a message to all connections of a specific guest session.
func (h *WebSocketHub) sendToGuest(ctx context.Context, sessionID string, msg WSMessage) {
	h.mu.RLock()
	conns := h.guestConnections[sessionID]
	connList := make([]*websocket.Conn, 0, len(conns))
	for conn := range conns {
		connList = append(connList, conn)
	}
	h.mu.RUnlock()

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for _, conn := range connList {
		if err := wsjson.Write(writeCtx, conn, msg); err != nil {
			slog.Warn("failed to send message to guest", "session_id", sessionID, "error", err)
		}
	}
}

func (h *WebSocketHub) broadcastToChannel(ctx context.Context, channelID string, msg WSMessage) {
	h.mu.RLock()
	members := h.channelMembers[channelID]
	memberIDs := make([]string, 0, len(members))
	for userID := range members {
		memberIDs = append(memberIDs, userID)
	}

	// Collect guest sessions connected to this channel
	var guestSessionIDs []string
	for sessionID, chID := range h.guestChannels {
		if chID == channelID {
			guestSessionIDs = append(guestSessionIDs, sessionID)
		}
	}
	h.mu.RUnlock()

	for _, userID := range memberIDs {
		h.sendToUser(ctx, userID, msg)
	}

	for _, sessionID := range guestSessionIDs {
		h.sendToGuest(ctx, sessionID, msg)
	}
}

func (h *WebSocketHub) broadcastToChannelExcept(ctx context.Context, channelID, excludeUserID string, msg WSMessage) {
	h.mu.RLock()
	members := h.channelMembers[channelID]
	memberIDs := make([]string, 0, len(members))
	for userID := range members {
		if userID != excludeUserID {
			memberIDs = append(memberIDs, userID)
		}
	}
	h.mu.RUnlock()

	for _, userID := range memberIDs {
		h.sendToUser(ctx, userID, msg)
	}
}

func (h *WebSocketHub) sendToUser(ctx context.Context, userID string, msg WSMessage) {
	h.mu.RLock()
	conns := h.connections[userID]
	connList := make([]*websocket.Conn, 0, len(conns))
	for conn := range conns {
		connList = append(connList, conn)
	}
	h.mu.RUnlock()

	if h.metrics != nil {
		h.metrics.WSMessagesTotal.WithLabelValues("outbound", msg.Type).Inc()
	}

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for _, conn := range connList {
		if err := wsjson.Write(writeCtx, conn, msg); err != nil {
			slog.Warn("failed to send message to user", "user_id", userID, "error", err)
		}
	}
}

// BroadcastMessageCreated broadcasts a new message to channel subscribers
func (h *WebSocketHub) BroadcastMessageCreated(ctx context.Context, channelID string, message *chatv1.MessageInfo) {
	h.broadcastToChannel(ctx, channelID, WSMessage{
		Type:      WSMessageNew,
		ChannelID: channelID,
		Message:   mustMarshal(message),
	})
}

// BroadcastMessageUpdated broadcasts an updated message to channel subscribers
func (h *WebSocketHub) BroadcastMessageUpdated(ctx context.Context, channelID string, message *chatv1.MessageInfo) {
	h.broadcastToChannel(ctx, channelID, WSMessage{
		Type:      WSMessageUpdated,
		ChannelID: channelID,
		Message:   mustMarshal(message),
	})
}

// BroadcastMessageDeleted broadcasts a deleted message to channel subscribers
func (h *WebSocketHub) BroadcastMessageDeleted(ctx context.Context, channelID, messageID string) {
	h.broadcastToChannel(ctx, channelID, WSMessage{
		Type:      WSMessageDeleted,
		ChannelID: channelID,
		MessageID: messageID,
	})
}

func (h *WebSocketHub) handleMarkRead(ctx context.Context, conn *websocket.Conn, userID string, msg *WSMessage) {
	if msg.ChannelID == "" || msg.MessageID == "" {
		h.sendError(ctx, conn, "channel_id and message_id are required")
		return
	}

	_, err := h.chatClient.MarkChannelRead(ctx, &chatv1.MarkChannelReadRequest{
		ChannelId: msg.ChannelID,
		UserId:    userID,
		MessageId: msg.MessageID,
	})
	if err != nil {
		slog.Error("failed to mark channel as read", "error", err)
		h.sendError(ctx, conn, "failed to mark channel as read")
		return
	}

	// Broadcast read receipt to user's other connections (multi-device sync)
	h.mu.RLock()
	conns := h.connections[userID]
	connList := make([]*websocket.Conn, 0, len(conns))
	for c := range conns {
		if c != conn { // Exclude the connection that sent the mark_read
			connList = append(connList, c)
		}
	}
	h.mu.RUnlock()

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	readMsg := WSMessage{
		Type:      WSReadReceipt,
		ChannelID: msg.ChannelID,
		MessageID: msg.MessageID,
		UserID:    userID,
	}

	for _, c := range connList {
		_ = wsjson.Write(writeCtx, c, readMsg)
	}
}

func (h *WebSocketHub) cacheUserName(ctx context.Context, userID string) {
	if h.userInfoFunc == nil {
		return
	}

	h.mu.RLock()
	_, cached := h.userNames[userID]
	h.mu.RUnlock()
	if cached {
		return
	}

	firstName, lastName, err := h.userInfoFunc(ctx, userID)
	if err != nil {
		slog.Warn("failed to cache user name", "user_id", userID, "error", err)
		return
	}

	h.mu.Lock()
	h.userNames[userID] = struct{ firstName, lastName string }{firstName, lastName}
	h.mu.Unlock()
}

func (h *WebSocketHub) getUserName(userID string) (string, string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	name, ok := h.userNames[userID]
	if !ok {
		return "", ""
	}
	return name.firstName, name.lastName
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		slog.Error("failed to marshal message", "error", err)
		return nil
	}
	return data
}

// SendNotificationToUser sends a real-time notification to all connections of a user.
// This is called by the gateway's notification delivery listener when a new
// notification is stored and ready for delivery.
func (h *WebSocketHub) SendNotificationToUser(ctx context.Context, userID string, notification interface{}, desktopPush bool, sound string) {
	payload := map[string]interface{}{
		"notification": notification,
		"desktop_push": desktopPush,
		"sound":        sound,
	}

	h.sendToUser(ctx, userID, WSMessage{
		Type:    WSNotificationNew,
		Message: mustMarshal(payload),
	})
}

// SendNotificationRead sends a notification-read sync message to all connections
// of a user (for multi-device sync when a notification is marked as read).
func (h *WebSocketHub) SendNotificationRead(ctx context.Context, userID string, notificationID string) {
	payload := map[string]interface{}{
		"notification_id": notificationID,
	}

	h.sendToUser(ctx, userID, WSMessage{
		Type:    WSNotificationRead,
		Message: mustMarshal(payload),
	})
}

// SendNotificationReadAll sends a read-all sync message to all connections
// of a user (for multi-device sync when all notifications are marked as read).
func (h *WebSocketHub) SendNotificationReadAll(ctx context.Context, userID string, markedCount int) {
	payload := map[string]interface{}{
		"marked_count": markedCount,
	}

	h.sendToUser(ctx, userID, WSMessage{
		Type:    WSNotificationReadAll,
		Message: mustMarshal(payload),
	})
}

// SendNotificationUnreadCount sends an updated unread count to all connections of a user.
func (h *WebSocketHub) SendNotificationUnreadCount(ctx context.Context, userID string, count int) {
	payload := map[string]interface{}{
		"unread_count": count,
	}

	h.sendToUser(ctx, userID, WSMessage{
		Type:    WSNotificationUnreadCount,
		Message: mustMarshal(payload),
	})
}

// ============================================================================
// Presence Handlers
// ============================================================================

// handlePresenceHeartbeat processes a client heartbeat and broadcasts the updated
// presence status to all users subscribed to this user's presence.
func (h *WebSocketHub) handlePresenceHeartbeat(ctx context.Context, conn *websocket.Conn, userID string) {
	if h.presenceService == nil {
		return // Presence service not configured, silently ignore
	}

	status, err := h.presenceService.Heartbeat(ctx, userID)
	if err != nil {
		slog.Warn("presence heartbeat failed", "user_id", userID, "error", err)
		return
	}

	h.BroadcastPresenceUpdate(ctx, userID, status)
}

// handlePresenceSubscribe registers the user as a subscriber to presence updates
// for the user IDs listed in the message payload.
// Message format: { "type": "presence.subscribe", "message": {"user_ids": ["id1", "id2"]} }
func (h *WebSocketHub) handlePresenceSubscribe(ctx context.Context, conn *websocket.Conn, userID string, msg *WSMessage) {
	var payload struct {
		UserIDs []string `json:"user_ids"`
	}
	if msg.Message != nil {
		if err := json.Unmarshal(msg.Message, &payload); err != nil {
			h.sendError(ctx, conn, "invalid presence.subscribe payload")
			return
		}
	}

	if len(payload.UserIDs) == 0 {
		h.sendError(ctx, conn, "user_ids required for presence.subscribe")
		return
	}

	h.mu.Lock()
	for _, targetUserID := range payload.UserIDs {
		if h.presenceSubscribers[targetUserID] == nil {
			h.presenceSubscribers[targetUserID] = make(map[string]struct{})
		}
		h.presenceSubscribers[targetUserID][userID] = struct{}{}
	}
	h.mu.Unlock()

	slog.Debug("presence subscription added",
		"subscriber", userID,
		"targets", payload.UserIDs,
	)
}

// handlePresenceSetStatus handles a manual presence status change from the client.
// Message format: { "type": "presence.set_status", "content": "away" }
func (h *WebSocketHub) handlePresenceSetStatus(ctx context.Context, conn *websocket.Conn, userID string, msg *WSMessage) {
	if h.presenceService == nil {
		h.sendError(ctx, conn, "presence service unavailable")
		return
	}

	status := msg.Content
	if status == "" {
		h.sendError(ctx, conn, "status is required for presence.set_status")
		return
	}

	if err := h.presenceService.SetManualStatus(ctx, userID, status); err != nil {
		slog.Warn("set presence status failed", "user_id", userID, "status", status, "error", err)
		h.sendError(ctx, conn, "failed to set presence status")
		return
	}

	h.BroadcastPresenceUpdate(ctx, userID, status)
}

// ============================================================================
// Call Signaling Handlers
// ============================================================================

// handleCallAccepted forwards a call acceptance signal to the video service.
// Message format: { "type": "call.accepted", "message_id": "<call_id>" }
func (h *WebSocketHub) handleCallAccepted(ctx context.Context, conn *websocket.Conn, userID string, msg *WSMessage) {
	if h.videoService == nil {
		h.sendError(ctx, conn, "video service unavailable")
		return
	}

	callID := msg.MessageID
	if callID == "" {
		h.sendError(ctx, conn, "message_id (call_id) is required for call.accepted")
		return
	}

	if err := h.videoService.NotifyCallAccepted(ctx, callID, userID); err != nil {
		slog.Warn("call accepted notification failed", "call_id", callID, "user_id", userID, "error", err)
		h.sendError(ctx, conn, "failed to accept call")
		return
	}

	slog.Info("call accepted via websocket", "call_id", callID, "user_id", userID)
}

// handleCallDeclined forwards a call decline signal to the video service.
// Message format: { "type": "call.declined", "message_id": "<call_id>" }
func (h *WebSocketHub) handleCallDeclined(ctx context.Context, conn *websocket.Conn, userID string, msg *WSMessage) {
	if h.videoService == nil {
		h.sendError(ctx, conn, "video service unavailable")
		return
	}

	callID := msg.MessageID
	if callID == "" {
		h.sendError(ctx, conn, "message_id (call_id) is required for call.declined")
		return
	}

	if err := h.videoService.NotifyCallDeclined(ctx, callID, userID); err != nil {
		slog.Warn("call declined notification failed", "call_id", callID, "user_id", userID, "error", err)
		h.sendError(ctx, conn, "failed to decline call")
		return
	}

	slog.Info("call declined via websocket", "call_id", callID, "user_id", userID)
}

// ============================================================================
// Broadcast Methods (Presence, Call, Reaction)
// ============================================================================

// BroadcastPresenceUpdate sends a presence status update to all users who are subscribed
// to the given user's presence updates.
func (h *WebSocketHub) BroadcastPresenceUpdate(ctx context.Context, userID, status string) {
	h.mu.RLock()
	subscribers := h.presenceSubscribers[userID]
	subscriberIDs := make([]string, 0, len(subscribers))
	for subID := range subscribers {
		subscriberIDs = append(subscriberIDs, subID)
	}
	h.mu.RUnlock()

	if len(subscriberIDs) == 0 {
		return
	}

	payload := map[string]interface{}{
		"user_id": userID,
		"status":  status,
	}

	msg := WSMessage{
		Type:    WSPresenceUpdate,
		UserID:  userID,
		Message: mustMarshal(payload),
	}

	for _, subID := range subscriberIDs {
		h.sendToUser(ctx, subID, msg)
	}
}

// BroadcastCallIncoming sends an incoming call notification to a specific user.
// This triggers the call overlay UI on the target user's client.
func (h *WebSocketHub) BroadcastCallIncoming(ctx context.Context, targetUserID string, callData map[string]interface{}) {
	h.sendToUser(ctx, targetUserID, WSMessage{
		Type:    WSCallIncoming,
		Message: mustMarshal(callData),
	})
}

// BroadcastCallEnded notifies a user that a call has ended.
func (h *WebSocketHub) BroadcastCallEnded(ctx context.Context, targetUserID string, callData map[string]interface{}) {
	h.sendToUser(ctx, targetUserID, WSMessage{
		Type:    WSCallEnded,
		Message: mustMarshal(callData),
	})
}

// BroadcastReactionToggled broadcasts a reaction toggle event to all members
// of the given channel.
func (h *WebSocketHub) BroadcastReactionToggled(ctx context.Context, channelID string, reactionData map[string]interface{}) {
	h.broadcastToChannel(ctx, channelID, WSMessage{
		Type:      WSReactionToggled,
		ChannelID: channelID,
		Message:   mustMarshal(reactionData),
	})
}

// ============================================================================
// Connection Cleanup Helpers
// ============================================================================

// cleanupPresenceSubscriptions removes a user from all presence subscription maps.
// Called when a user disconnects.
func (h *WebSocketHub) cleanupPresenceSubscriptions(userID string) {
	// Remove user as a subscriber from all targets
	for targetID, subs := range h.presenceSubscribers {
		delete(subs, userID)
		if len(subs) == 0 {
			delete(h.presenceSubscribers, targetID)
		}
	}

	// Remove the user's own subscriber list (nobody should receive updates anymore)
	delete(h.presenceSubscribers, userID)
}

// ValidateChannelID validates a channel ID
func ValidateChannelID(channelID string) bool {
	_, err := uuid.Parse(channelID)
	return err == nil
}
