//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func TestChatChannelFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, _ := registerAndLoginAdmin(t, base)

	var channelID string

	// Create channel
	t.Run("create_channel", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/channels", map[string]interface{}{
			"name":        "e2e-test-channel",
			"description": "E2E test channel",
			"is_private":  false,
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Channel map[string]interface{} `json:"channel"`
		}
		decodeBody(t, body, &result)

		id, ok := result.Channel["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected channel id")
		}
		channelID = id
	})

	// Get channel
	t.Run("get_channel", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/channels/"+channelID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Channel map[string]interface{} `json:"channel"`
		}
		decodeBody(t, body, &result)

		if result.Channel["name"] != "e2e-test-channel" {
			t.Fatalf("expected channel name e2e-test-channel, got %v", result.Channel["name"])
		}
	})

	// List channels
	t.Run("list_channels", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/channels", token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Channels []map[string]interface{} `json:"channels"`
		}
		decodeBody(t, body, &result)

		found := false
		for _, ch := range result.Channels {
			if ch["id"] == channelID {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("created channel not found in list")
		}
	})

	// Send message
	var messageID string
	t.Run("send_message", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/channels/"+channelID+"/messages", map[string]interface{}{
			"content": "Hello from E2E test!",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Message map[string]interface{} `json:"message"`
		}
		decodeBody(t, body, &result)

		id, ok := result.Message["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected message id")
		}
		messageID = id
	})

	// List messages
	t.Run("list_messages", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/channels/"+channelID+"/messages", token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Messages []map[string]interface{} `json:"messages"`
		}
		decodeBody(t, body, &result)

		found := false
		for _, msg := range result.Messages {
			if msg["id"] == messageID {
				found = true
				if msg["content"] != "Hello from E2E test!" {
					t.Fatalf("expected message content, got %v", msg["content"])
				}
				break
			}
		}
		if !found {
			t.Fatal("sent message not found in list")
		}
	})

	// Update message
	t.Run("update_message", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/messages/"+messageID, map[string]interface{}{
			"content": "Updated E2E message",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// Delete message
	t.Run("delete_message", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/messages/"+messageID, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// Update channel
	t.Run("update_channel", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/channels/"+channelID, map[string]interface{}{
			"name":        "e2e-test-channel-updated",
			"description": "Updated E2E test channel",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// Delete channel
	t.Run("delete_channel", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/channels/"+channelID, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// No auth
	t.Run("no_auth", func(t *testing.T) {
		resp, _ := getJSON(t, base+"/api/v1/channels", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}

func TestChatDMFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token1, _ := registerAndLoginAdmin(t, base)
	token2, userID2 := registerAndLoginAdmin(t, base)

	var dmChannelID string

	// Create DM
	t.Run("create_dm", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/channels/dm", map[string]interface{}{
			"other_user_id": userID2,
		}, token1)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Channel map[string]interface{} `json:"channel"`
		}
		decodeBody(t, body, &result)

		id, ok := result.Channel["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected DM channel id")
		}
		dmChannelID = id
	})

	// Send DM
	t.Run("send_dm", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/channels/"+dmChannelID+"/messages", map[string]interface{}{
			"content": "Hello DM!",
		}, token1)
		requireStatus(t, resp, body, http.StatusCreated)
	})

	// Other user can read DM
	t.Run("other_user_reads_dm", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/channels/"+dmChannelID+"/messages", token2)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Messages []map[string]interface{} `json:"messages"`
		}
		decodeBody(t, body, &result)

		if len(result.Messages) == 0 {
			t.Fatal("expected at least one DM message")
		}
	})

	// List DMs
	t.Run("list_dms", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/channels/dm", token1)
		requireStatus(t, resp, body, http.StatusOK)
	})
}
