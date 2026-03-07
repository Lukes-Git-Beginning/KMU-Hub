package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/chat/file"
)

func TestFileUpload_HappyPath(t *testing.T) {
	handler, repo, _, _ := newTestFileUploadHandler()
	userID := uuid.New()
	channelID := uuid.New()

	repo.members[repo.memberKey(channelID, userID)] = true

	req := newMultipartUploadRequestWithMessageID(t,
		userID.String(), channelID.String(), "",
		"test.png", "image/png", []byte("fake-png-data"))
	rec := httptest.NewRecorder()

	handler.HandleUploadFile(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.NotEmpty(t, body["id"])
	require.Equal(t, channelID.String(), body["channel_id"])
	require.Equal(t, userID.String(), body["uploaded_by"])
	require.Equal(t, "test.png", body["filename"])
	require.Equal(t, false, body["has_thumbnail"])
}

func TestFileUpload_MissingAuth(t *testing.T) {
	handler, _, _, _ := newTestFileUploadHandler()
	channelID := uuid.New()

	// No userID in context
	req := newMultipartUploadRequest(t, "", channelID.String(), "test.png", "image/png", []byte("data"))
	rec := httptest.NewRecorder()

	handler.HandleUploadFile(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestFileUpload_InvalidChannelID(t *testing.T) {
	handler, _, _, _ := newTestFileUploadHandler()
	userID := uuid.New()

	tests := []struct {
		name      string
		channelID string
	}{
		{"missing channel_id", ""},
		{"malformed channel_id", "not-a-uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newMultipartUploadRequest(t, userID.String(), tt.channelID, "test.png", "image/png", []byte("data"))
			rec := httptest.NewRecorder()

			handler.HandleUploadFile(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestFileUpload_MissingFile(t *testing.T) {
	handler, _, _, _ := newTestFileUploadHandler()
	userID := uuid.New()
	channelID := uuid.New()

	// No file field — pass empty filename to skip file creation
	req := newMultipartUploadRequest(t, userID.String(), channelID.String(), "", "", nil)
	rec := httptest.NewRecorder()

	handler.HandleUploadFile(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestFileUpload_InvalidMessageID(t *testing.T) {
	handler, repo, _, _ := newTestFileUploadHandler()
	userID := uuid.New()
	channelID := uuid.New()

	repo.members[repo.memberKey(channelID, userID)] = true

	req := newMultipartUploadRequestWithMessageID(t,
		userID.String(), channelID.String(), "bad-uuid",
		"test.png", "image/png", []byte("data"))
	rec := httptest.NewRecorder()

	handler.HandleUploadFile(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestFileUpload_ServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(repo *fileMockRepo, scanner *fileMockScanner)
		mimeType   string
		wantStatus int
	}{
		{
			name: "not channel member",
			setup: func(repo *fileMockRepo, _ *fileMockScanner) {
				// members map empty → not a member
			},
			mimeType:   "image/png",
			wantStatus: http.StatusForbidden,
		},
		{
			name: "channel archived",
			setup: func(repo *fileMockRepo, _ *fileMockScanner) {
				// Will be set per-test below (need channelID)
			},
			mimeType:   "image/png",
			wantStatus: http.StatusConflict,
		},
		{
			name: "invalid mime type",
			setup: func(repo *fileMockRepo, _ *fileMockScanner) {
				// member + not archived, but bad mime
			},
			mimeType:   "application/x-evil",
			wantStatus: http.StatusUnsupportedMediaType,
		},
		{
			name: "quota exceeded",
			setup: func(repo *fileMockRepo, _ *fileMockScanner) {
				repo.quota.UsedBytes = repo.quota.MaxBytes // full
			},
			mimeType:   "image/png",
			wantStatus: http.StatusInsufficientStorage,
		},
		{
			name: "scan failed",
			setup: func(_ *fileMockRepo, scanner *fileMockScanner) {
				scanner.scanErr = file.ErrScanFailed
			},
			mimeType:   "image/png",
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo, _, scanner := newTestFileUploadHandler()
			userID := uuid.New()
			channelID := uuid.New()

			// Default: member + not archived (overridden by setup)
			if tt.name != "not channel member" {
				repo.members[repo.memberKey(channelID, userID)] = true
			}
			if tt.name == "channel archived" {
				repo.archived[channelID] = true
			}

			tt.setup(repo, scanner)

			req := newMultipartUploadRequestWithMessageID(t,
				userID.String(), channelID.String(), "",
				"test.png", tt.mimeType, []byte("data"))
			rec := httptest.NewRecorder()

			handler.HandleUploadFile(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestFileUpload_FileTooLarge(t *testing.T) {
	// Create handler with 0 MB max to trigger file too large
	repo := newFileMockRepo()
	store := &fileMockStore{}
	scanner := &fileMockScanner{}
	thumbGen := &fileMockThumbGen{}
	svc := file.NewService(repo, store, scanner, thumbGen, 0) // 0 MB max
	handler := NewFileUploadHandler(svc, nil)

	userID := uuid.New()
	channelID := uuid.New()
	repo.members[repo.memberKey(channelID, userID)] = true

	req := newMultipartUploadRequestWithMessageID(t,
		userID.String(), channelID.String(), "",
		"test.png", "image/png", []byte("data"))
	rec := httptest.NewRecorder()

	handler.HandleUploadFile(rec, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestFileUpload_OptionalMessageID(t *testing.T) {
	handler, repo, _, _ := newTestFileUploadHandler()
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	repo.members[repo.memberKey(channelID, userID)] = true

	req := newMultipartUploadRequestWithMessageID(t,
		userID.String(), channelID.String(), messageID.String(),
		"doc.pdf", "application/pdf", []byte("fake-pdf"))
	rec := httptest.NewRecorder()

	handler.HandleUploadFile(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Equal(t, messageID.String(), body["message_id"])
}

func TestUUIDPtrToString(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.Equal(t, "", uuidPtrToString(nil))
	})
	t.Run("valid", func(t *testing.T) {
		id := uuid.New()
		require.Equal(t, id.String(), uuidPtrToString(&id))
	})
}

