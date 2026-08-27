package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Covers the reaction/bookmark/search/mention handler group in route_chat.go
// that route_chat_test.go and route_chat_membership_test.go leave untested:
// HandleGetMessages, HandleGetThreadReplies, HandleGetUnreadCounts,
// HandleGetUserMentions, HandleListDMs, HandleListReactions,
// HandleToggleReaction, HandleGetReactionSummary, HandleListBookmarks,
// HandleToggleBookmark, HandleListChannelFiles, HandleGetFileThumbnailURL,
// HandleSearchChat, HandleUpdateChannel.

const (
	testChatMessageID2 = "770e8400-e29b-41d4-a716-446655440002"
	testChatFileID     = "880e8400-e29b-41d4-a716-446655440003"
)

// --- HandleGetMessages ---

func TestHandleGetMessages_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetMessages)
}

func TestHandleGetMessages_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/"+testChatChannelID+"/messages", nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	withAuthRequired(routes.HandleGetMessages)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetMessages_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/bad/messages", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleGetMessages(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetMessages_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/"+testChatChannelID+"/messages?before=x&after=y", nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withUserID(req, "user-123")
	routes.HandleGetMessages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetThreadReplies ---

func TestHandleGetThreadReplies_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetThreadReplies)
}

func TestHandleGetThreadReplies_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/messages/"+testChatMessageID2+"/thread", nil)
	req = withChiURLParam(req, "id", testChatMessageID2)
	withAuthRequired(routes.HandleGetThreadReplies)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetThreadReplies_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/messages/bad/thread", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleGetThreadReplies(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetThreadReplies_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/messages/"+testChatMessageID2+"/thread", nil)
	req = withChiURLParam(req, "id", testChatMessageID2)
	req = withUserID(req, "user-123")
	routes.HandleGetThreadReplies(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetUnreadCounts ---

func TestHandleGetUnreadCounts_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetUnreadCounts)
}

func TestHandleGetUnreadCounts_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/unread", nil)
	withAuthRequired(routes.HandleGetUnreadCounts)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetUnreadCounts_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/unread", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetUnreadCounts(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetUserMentions ---

func TestHandleGetUserMentions_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetUserMentions)
}

func TestHandleGetUserMentions_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/messages/mentions", nil)
	withAuthRequired(routes.HandleGetUserMentions)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetUserMentions_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/messages/mentions", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetUserMentions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListDMs ---

func TestHandleListDMs_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListDMs)
}

func TestHandleListDMs_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/dm", nil)
	withAuthRequired(routes.HandleListDMs)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListDMs_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/dm", nil)
	req = withUserID(req, "user-123")
	routes.HandleListDMs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListReactions ---

func TestHandleListReactions_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListReactions)
}

func TestHandleListReactions_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/messages/"+testChatMessageID2+"/reactions", nil)
	req = withChiURLParam(req, "id", testChatMessageID2)
	withAuthRequired(routes.HandleListReactions)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListReactions_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/messages/bad/reactions", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleListReactions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListReactions_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/messages/"+testChatMessageID2+"/reactions", nil)
	req = withChiURLParam(req, "id", testChatMessageID2)
	req = withUserID(req, "user-123")
	routes.HandleListReactions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleToggleReaction ---

func TestHandleToggleReaction_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleToggleReaction)
}

func TestHandleToggleReaction_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/"+testChatMessageID2+"/reactions",
		jsonBody(t, map[string]interface{}{"emoji": "thumbsup"}))
	req = withChiURLParam(req, "id", testChatMessageID2)
	withAuthRequired(routes.HandleToggleReaction)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleToggleReaction_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/bad/reactions",
		jsonBody(t, map[string]interface{}{"emoji": "thumbsup"}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleToggleReaction(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleToggleReaction_InvalidJSON(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/"+testChatMessageID2+"/reactions", invalidJSON())
	req = withChiURLParam(req, "id", testChatMessageID2)
	req = withUserID(req, "user-123")
	routes.HandleToggleReaction(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleToggleReaction_MissingEmoji(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/"+testChatMessageID2+"/reactions",
		jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", testChatMessageID2)
	req = withUserID(req, "user-123")
	routes.HandleToggleReaction(rec, req)
	assertValidationError(t, rec, "emoji")
}

func TestHandleToggleReaction_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/"+testChatMessageID2+"/reactions",
		jsonBody(t, map[string]interface{}{"emoji": "thumbsup"}))
	req = withChiURLParam(req, "id", testChatMessageID2)
	req = withUserID(req, "user-123")
	routes.HandleToggleReaction(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetReactionSummary ---

func TestHandleGetReactionSummary_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetReactionSummary)
}

func TestHandleGetReactionSummary_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/reactions/summary",
		jsonBody(t, map[string]interface{}{"message_ids": []string{testChatMessageID2}}))
	withAuthRequired(routes.HandleGetReactionSummary)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetReactionSummary_MissingMessageIDs(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/reactions/summary", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleGetReactionSummary(rec, req)
	assertValidationError(t, rec, "message_ids")
}

func TestHandleGetReactionSummary_InvalidMessageIDFormat(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/reactions/summary",
		jsonBody(t, map[string]interface{}{"message_ids": []string{"not-a-uuid"}}))
	req = withUserID(req, "user-123")
	routes.HandleGetReactionSummary(rec, req)
	assertValidationError(t, rec, "message_ids[0]")
}

func TestHandleGetReactionSummary_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/reactions/summary",
		jsonBody(t, map[string]interface{}{"message_ids": []string{testChatMessageID2}}))
	req = withUserID(req, "user-123")
	routes.HandleGetReactionSummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListBookmarks ---

func TestHandleListBookmarks_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListBookmarks)
}

func TestHandleListBookmarks_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/messages/bookmarks", nil)
	withAuthRequired(routes.HandleListBookmarks)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListBookmarks_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/messages/bookmarks", nil)
	req = withUserID(req, "user-123")
	routes.HandleListBookmarks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleToggleBookmark ---

func TestHandleToggleBookmark_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleToggleBookmark)
}

func TestHandleToggleBookmark_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/"+testChatMessageID2+"/bookmark", nil)
	req = withChiURLParam(req, "id", testChatMessageID2)
	withAuthRequired(routes.HandleToggleBookmark)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleToggleBookmark_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/bad/bookmark", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleToggleBookmark(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleToggleBookmark_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/messages/"+testChatMessageID2+"/bookmark", nil)
	req = withChiURLParam(req, "id", testChatMessageID2)
	req = withUserID(req, "user-123")
	routes.HandleToggleBookmark(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListChannelFiles ---

func TestHandleListChannelFiles_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListChannelFiles)
}

func TestHandleListChannelFiles_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/"+testChatChannelID+"/files", nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	withAuthRequired(routes.HandleListChannelFiles)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListChannelFiles_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/bad/files", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleListChannelFiles(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListChannelFiles_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/"+testChatChannelID+"/files", nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withUserID(req, "user-123")
	routes.HandleListChannelFiles(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetFileThumbnailURL ---

func TestHandleGetFileThumbnailURL_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetFileThumbnailURL)
}

func TestHandleGetFileThumbnailURL_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/files/"+testChatFileID+"/thumbnail", nil)
	req = withChiURLParam(req, "id", testChatFileID)
	withAuthRequired(routes.HandleGetFileThumbnailURL)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetFileThumbnailURL_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/files/bad/thumbnail", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleGetFileThumbnailURL(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetFileThumbnailURL_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/files/"+testChatFileID+"/thumbnail", nil)
	req = withChiURLParam(req, "id", testChatFileID)
	req = withUserID(req, "user-123")
	routes.HandleGetFileThumbnailURL(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateChannel ---

func TestHandleUpdateChannel_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateChannel)
}

func TestHandleUpdateChannel_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/"+testChatChannelID, jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", testChatChannelID)
	withAuthRequired(routes.HandleUpdateChannel)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateChannel_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleUpdateChannel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateChannel_InvalidJSON(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/"+testChatChannelID, invalidJSON())
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withUserID(req, "user-123")
	routes.HandleUpdateChannel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateChannel_EmptyName(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/"+testChatChannelID,
		jsonBody(t, map[string]interface{}{"name": ""}))
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withUserID(req, "user-123")
	routes.HandleUpdateChannel(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdateChannel_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/"+testChatChannelID,
		jsonBody(t, map[string]interface{}{"name": "renamed"}))
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withUserID(req, "user-123")
	routes.HandleUpdateChannel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSearchChat ---

func TestHandleSearchChat_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/chat/search?q=hello", nil)
	req = withUserID(req, "user-123")
	routes.HandleSearchChat(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSearchChat_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/chat/search?q=hello", nil)
	withAuthRequired(routes.HandleSearchChat)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSearchChat_MissingQuery(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/chat/search", nil)
	req = withUserID(req, "user-123")
	routes.HandleSearchChat(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "q")
}

func TestHandleSearchChat_InvalidChannelID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/chat/search?q=hello&channel_id=not-a-uuid", nil)
	req = withUserID(req, "user-123")
	routes.HandleSearchChat(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "channel_id")
}

func TestHandleSearchChat_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/chat/search?q=hello&channel_id="+testChatChannelID+"&page=2&page_size=10", nil)
	req = withUserID(req, "user-123")
	routes.HandleSearchChat(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
