//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"testing"
)

func TestDocumentFolderFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, _ := registerAndLoginAdmin(t, base)

	var folderID string

	// Create folder. The gateway wraps the folder object in a {"folder": ...}
	// envelope (canonical wire shape).
	t.Run("create_folder", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/documents/folders", map[string]interface{}{
			"name":       "E2E Test Folder",
			"space_type": "personal",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var wrapper map[string]interface{}
		decodeBody(t, body, &wrapper)
		folder, _ := wrapper["folder"].(map[string]interface{})

		id, ok := folder["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected folder id")
		}
		folderID = id
	})

	// Get folder
	t.Run("get_folder", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/documents/folders/"+folderID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var wrapper map[string]interface{}
		decodeBody(t, body, &wrapper)
		folder, _ := wrapper["folder"].(map[string]interface{})

		if folder["name"] != "E2E Test Folder" {
			t.Fatalf("expected folder name, got %v", folder["name"])
		}
	})

	// List folders
	t.Run("list_folders", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/documents/folders", token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// Create subfolder
	var subfolderID string
	t.Run("create_subfolder", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/documents/folders", map[string]interface{}{
			"name":       "E2E Subfolder",
			"parent_id":  folderID,
			"space_type": "personal",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var wrapper map[string]interface{}
		decodeBody(t, body, &wrapper)
		folder, _ := wrapper["folder"].(map[string]interface{})

		id, ok := folder["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected subfolder id")
		}
		subfolderID = id
	})

	// Get folder path
	t.Run("get_folder_path", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/documents/folders/"+subfolderID+"/path", token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// Update folder
	t.Run("update_folder", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/documents/folders/"+folderID, map[string]interface{}{
			"name": "E2E Test Folder Updated",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// Delete subfolder first, then parent
	t.Run("delete_subfolder", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/documents/folders/"+subfolderID, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("delete_folder", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/documents/folders/"+folderID, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// No auth
	t.Run("no_auth", func(t *testing.T) {
		resp, _ := getJSON(t, base+"/api/v1/documents/folders", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}

func TestDocumentFileMetadata(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, _ := registerAndLoginAdmin(t, base)

	// List files (should be empty initially)
	t.Run("list_files_empty", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/documents/files", token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// Note: File upload requires multipart/form-data which is tested separately
	// via the /api/v1/files/upload endpoint. These E2E tests focus on metadata CRUD.
}

// uploadChatFile posts a small text file to a chat channel through the
// multipart upload route the desktop uses and returns the stored file id.
func uploadChatFile(t *testing.T, base, token, channelID, filename, content string) string {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("channel_id", channelID); err != nil {
		t.Fatalf("failed to write channel_id field: %v", err)
	}
	// CreateFormFile would label the part application/octet-stream, which the
	// chat service rejects; the part needs the real content type.
	part, err := mw.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="file"; filename="` + filename + `"`},
		"Content-Type":        {"text/plain"},
	})
	if err != nil {
		t.Fatalf("failed to create file part: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write file part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("failed to close multipart body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, base+"/api/v1/files/upload", &buf)
	if err != nil {
		t.Fatalf("failed to create upload request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}
	defer resp.Body.Close()
	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	requireStatus(t, resp, body.Bytes(), http.StatusCreated)

	var uploaded map[string]interface{}
	decodeBody(t, body.Bytes(), &uploaded)
	id, ok := uploaded["id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected file id in upload response, got %s", body.String())
	}
	return id
}

// virtualFilesList reads GET /documents/virtual and returns the files by id.
// It also asserts the envelope the desktop reads (ListVirtualFilesResponse):
// `files` is always an array, never null, and `total` is a number.
func virtualFilesList(t *testing.T, base, token, query string) map[string]map[string]interface{} {
	t.Helper()
	resp, body := getJSON(t, base+"/api/v1/documents/virtual"+query, token)
	requireStatus(t, resp, body, http.StatusOK)

	var envelope map[string]json.RawMessage
	decodeBody(t, body, &envelope)
	rawFiles := bytes.TrimSpace(envelope["files"])
	if len(rawFiles) == 0 || rawFiles[0] != '[' {
		t.Fatalf("expected files to be a JSON array, got %s", string(envelope["files"]))
	}
	var total float64
	if err := json.Unmarshal(envelope["total"], &total); err != nil {
		t.Fatalf("expected numeric total, got %s", string(envelope["total"]))
	}

	var files []map[string]interface{}
	if err := json.Unmarshal(rawFiles, &files); err != nil {
		t.Fatalf("failed to decode files: %v", err)
	}
	byID := make(map[string]map[string]interface{}, len(files))
	for _, f := range files {
		if id, ok := f["id"].(string); ok {
			byID[id] = f
		}
	}
	return byID
}

// TestDocumentVirtualFilesFlow covers the document module's "Chat attachments"
// view: a file uploaded into a private channel shows up for the members of that
// channel and for nobody else. Both users are admins of the same tenant, so the
// only thing that hides the file from the second one is channel membership.
//
// lean: source_type, created_at and file_size are not asserted. The handler
// writes the raw proto through encoding/json, so those three leave the gateway
// as enum number, {seconds,nanos} and number rather than the FE's 'chat', ISO
// string and number; pinning that here would freeze the wrong shape. The
// open unit fe-virtual-files-view-wire-shape (BACKLOG-NEXT) owns them.
func TestDocumentVirtualFilesFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	ownerToken, ownerID := registerAndLoginAdmin(t, base)
	outsiderToken, _ := registerAndLoginAdmin(t, base)

	var channelID, fileID string

	t.Run("create_private_channel", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/channels", map[string]interface{}{
			"name":        "e2e-virtual-files",
			"description": "E2E virtual files channel",
			"is_private":  true,
		}, ownerToken)
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

	t.Run("upload_file", func(t *testing.T) {
		fileID = uploadChatFile(t, base, ownerToken, channelID, "e2e-virtual.txt", "virtual file e2e content")
	})

	t.Run("member_sees_file", func(t *testing.T) {
		f, ok := virtualFilesList(t, base, ownerToken, "")[fileID]
		if !ok {
			t.Fatal("uploaded chat file not found in the member's virtual files")
		}
		if f["filename"] != "e2e-virtual.txt" {
			t.Fatalf("expected filename e2e-virtual.txt, got %v", f["filename"])
		}
		if f["source_id"] != channelID {
			t.Fatalf("expected source_id to be the channel %s, got %v", channelID, f["source_id"])
		}
		if f["source_name"] != "e2e-virtual-files" {
			t.Fatalf("expected source_name e2e-virtual-files, got %v", f["source_name"])
		}
		if f["uploaded_by"] != ownerID {
			t.Fatalf("expected uploaded_by %s, got %v", ownerID, f["uploaded_by"])
		}
		if f["mime_type"] != "text/plain" {
			t.Fatalf("expected mime_type text/plain, got %v", f["mime_type"])
		}
	})

	t.Run("source_filter", func(t *testing.T) {
		if _, ok := virtualFilesList(t, base, ownerToken, "?source=chat")[fileID]; !ok {
			t.Fatal("chat file missing when filtering on source=chat")
		}
		if _, ok := virtualFilesList(t, base, ownerToken, "?source=email")[fileID]; ok {
			t.Fatal("chat file must not appear when filtering on source=email")
		}
	})

	t.Run("non_member_does_not_see_file", func(t *testing.T) {
		if _, ok := virtualFilesList(t, base, outsiderToken, "")[fileID]; ok {
			t.Fatal("chat file of a private channel leaked to a non-member")
		}
		if _, ok := virtualFilesList(t, base, outsiderToken, "?source=chat")[fileID]; ok {
			t.Fatal("chat file of a private channel leaked to a non-member on source=chat")
		}
	})

	t.Run("deleted_file_leaves_the_view", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/files/"+fileID, ownerToken)
		requireStatus(t, resp, body, http.StatusNoContent)

		if _, ok := virtualFilesList(t, base, ownerToken, "")[fileID]; ok {
			t.Fatal("deleted chat file still listed in the virtual files")
		}
	})

	t.Run("delete_channel", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/channels/"+channelID, ownerToken)
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("no_auth", func(t *testing.T) {
		resp, _ := getJSON(t, base+"/api/v1/documents/virtual", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}
