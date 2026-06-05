//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func TestDocumentFolderFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, _ := registerAndLoginAdmin(t, base)

	var folderID string

	// Create folder. The gateway writes the folder object at the top level
	// of the response (no {"folder": ...} envelope).
	t.Run("create_folder", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/documents/folders", map[string]interface{}{
			"name":       "E2E Test Folder",
			"space_type": "personal",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var folder map[string]interface{}
		decodeBody(t, body, &folder)

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

		var folder map[string]interface{}
		decodeBody(t, body, &folder)

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

		var folder map[string]interface{}
		decodeBody(t, body, &folder)

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
