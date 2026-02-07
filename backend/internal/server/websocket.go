package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/kmuhub/kmuhub/internal/auth"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

// UserInfoFunc is a callback to resolve user IDs to names
type UserInfoFunc func(ctx context.Context, userID string) (firstName, lastName string, err error)

// WebSocketHub manages WebSocket connections and message broadcasting
type WebSocketHub struct {
	chatClient chatv1.ChatServiceClient
	tokenMaker *auth.TokenMaker

	// connections maps user ID to their connections
	connections map[string]map[*websocket.Conn]struct{}
	// channelMembers maps channel ID to user IDs
	channelMembers map[string]map[string]struct{}
	// userNames caches user display names for typing indicators
	userNames    map[string]struct{ firstName, lastName string }
	userInfoFunc UserInfoFunc
	mu           sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(chatClient chatv1.ChatServiceClient, tokenMaker *auth.TokenMaker, userInfoFunc UserInfoFunc) *WebSocketHub {
	return &WebSocketHub{
		chatClient:     chatClient,
		tokenMaker:     tokenMaker,
		connections:    make(map[string]map[*websocket.Conn]struct{}),
		channelMembers: make(map[string]map[string]struct{}),
		userNames:      make(map[string]struct{ firstName, lastName string }),
		userInfoFunc:   userInfoFunc,
	}
}

// WebSocket message types
const (
	// Client -> Server
	WSMessageSend        = "message.send"
	WSTypingStart        = "typing.start"
	WSTypingStop         = "typing.stop"
	WSSubscribeChannel   = "channel.subscribe"
	WSUnsubscribeChannel = "channel.unsubscribe"
	WSMarkRead           = "channel.mark_read"

	// Server -> Client
	WSMessageNew      = "message.new"
	WSMessageUpdated  = "message.updated"
	WSMessageDeleted  = "message.deleted"
	WSThreadReplyNew  = "thread.reply.new"
	WSTypingIndicator = "typing"
	WSReadReceipt     = "read_receipt"
	WSMentionNew      = "mention.new"
	WSError           = "error"
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

// HandleWebSocket handles WebSocket connections
// Authentication is done via query parameter: /api/v1/ws?token=<JWT>
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token query parameter is required", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := h.tokenMaker.ValidateAccessToken(token)
	if err != nil {
		slog.Warn("websocket auth failed", "error", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	userID := claims.UserID

	// Accept WebSocket connection
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Configure properly in production
	})
	if err != nil {
		slog.Error("websocket accept failed", "error", err)
		return
	}

	// Register connection and cache user name
	h.registerConnection(userID, conn)
	h.cacheUserName(r.Context(), userID)
	defer h.unregisterConnection(userID, conn)

	slog.Info("websocket connected", "user_id", userID)

	// Handle messages
	ctx := r.Context()
	h.handleConnection(ctx, conn, userID)
}

func (h *WebSocketHub) registerConnection(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[userID] == nil {
		h.connections[userID] = make(map[*websocket.Conn]struct{})
	}
	h.connections[userID][conn] = struct{}{}
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

	_ = conn.Close(websocket.StatusNormalClosure, "connection closed")
}

func (h *WebSocketHub) handleConnection(ctx context.Context, conn *websocket.Conn, userID string) {
	for {
		var msg WSMessage
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				slog.Info("websocket closed", "user_id", userID, "status", websocket.CloseStatus(err))
			} else {
				slog.Error("websocket read error", "user_id", userID, "error", err)
			}
			return
		}

		h.handleMessage(ctx, conn, userID, &msg)
	}
}

func (h *WebSocketHub) handleMessage(ctx context.Context, conn *websocket.Conn, userID string, msg *WSMessage) {
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

func (h *WebSocketHub) broadcastToChannel(ctx context.Context, channelID string, msg WSMessage) {
	h.mu.RLock()
	members := h.channelMembers[channelID]
	memberIDs := make([]string, 0, len(members))
	for userID := range members {
		memberIDs = append(memberIDs, userID)
	}
	h.mu.RUnlock()

	for _, userID := range memberIDs {
		h.sendToUser(ctx, userID, msg)
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

// ValidateChannelID validates a channel ID
func ValidateChannelID(channelID string) bool {
	_, err := uuid.Parse(channelID)
	return err == nil
}
