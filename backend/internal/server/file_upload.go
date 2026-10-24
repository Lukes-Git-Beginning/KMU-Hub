package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// FileUploadHandler handles multipart file uploads directly (not via gRPC)
type FileUploadHandler struct {
	fileService *file.Service
	wsHub       *WebSocketHub
	idempotency idempotency.Repository
}

// NewFileUploadHandler creates a new file upload handler
func NewFileUploadHandler(fileService *file.Service, wsHub *WebSocketHub) *FileUploadHandler {
	return &FileUploadHandler{
		fileService: fileService,
		wsHub:       wsHub,
	}
}

// SetIdempotency enables retry deduplication via the optional Idempotency-Key
// header. The generic Idempotency middleware cannot guard this route: it reads
// the whole body into memory to hash it, and uploads go up to maxUploadSize.
func (h *FileUploadHandler) SetIdempotency(repo idempotency.Repository) {
	h.idempotency = repo
}

// maxUploadSize is the maximum request body size for file uploads (55 MB to allow overhead)
const maxUploadSize = 55 << 20

// HandleUploadFile handles multipart file upload
// POST /api/v1/files/upload
// Form fields: file (required), channel_id (optional), message_id (optional)
// Without channel_id the file is stored unattached under the caller's tenant
// and the response carries its storage_key — the task attachment flow records
// that key via POST /tasks/{id}/files.
// Header: Idempotency-Key (optional). A retry with the same key and the same
// channel/message/filename/size replays the first response instead of storing
// the file twice; the same key with a different upload is rejected with 422.
func (h *FileUploadHandler) HandleUploadFile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		response.Error(w, http.StatusBadRequest, "request body too large or invalid multipart form")
		return
	}

	// channel_id is optional; when present it must be valid.
	channelIDStr := r.FormValue("channel_id")
	var channelID uuid.UUID
	if channelIDStr != "" {
		parsed, parseErr := uuid.Parse(channelIDStr)
		if parseErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid channel_id")
			return
		}
		channelID = parsed
	}

	// Get optional message_id
	var messageID *uuid.UUID
	if msgIDStr := r.FormValue("message_id"); msgIDStr != "" {
		msgID, parseErr := uuid.Parse(msgIDStr)
		if parseErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid message_id")
			return
		}
		messageID = &msgID
	}

	// Get file from multipart form
	formFile, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer func() { _ = formFile.Close() }()

	// Detect MIME type from header
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	uploaderID, parseErr := uuid.Parse(userID)
	if parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	// The body has been parsed once already, so the fingerprint comes from the
	// parsed parts and the content is never read a second time.
	w, done, ok := middleware.BeginIdempotent(h.idempotency, w, r,
		fmt.Sprintf("%s|%s|%s|%d", channelID, uuidPtrToString(messageID), header.Filename, header.Size))
	if !ok {
		return
	}
	defer done()

	input := file.UploadInput{
		ChannelID: channelID,
		MessageID: messageID,
		Filename:  header.Filename,
		MimeType:  mimeType,
		FileSize:  header.Size,
		Reader:    formFile,
		UserID:    uploaderID,
	}

	if channelIDStr == "" {
		if messageID != nil {
			response.Error(w, http.StatusBadRequest, "message_id requires channel_id")
			return
		}
		upload, uploadErr := h.fileService.UploadUnattached(r.Context(), input)
		if uploadErr != nil {
			response.Error(w, uploadErrorStatus(uploadErr), uploadErr.Error())
			return
		}
		response.JSON(w, http.StatusCreated, map[string]interface{}{
			"storage_key": upload.StorageKey,
			"filename":    upload.Filename,
			"mime_type":   upload.MimeType,
			"file_size":   upload.FileSize,
		})
		return
	}

	result, uploadErr := h.fileService.Upload(r.Context(), input)
	if uploadErr != nil {
		response.Error(w, uploadErrorStatus(uploadErr), uploadErr.Error())
		return
	}

	// Broadcast file.uploaded via WebSocket
	if h.wsHub != nil {
		h.broadcastFileUploaded(r.Context(), result)
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"id":                  result.ID.String(),
		"message_id":          uuidPtrToString(result.MessageID),
		"channel_id":          result.ChannelID.String(),
		"filename":            result.Filename,
		"mime_type":           result.MimeType,
		"file_size":           result.FileSize,
		"uploaded_by":         result.UploadedBy.String(),
		"created_at":          result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"uploader_first_name": result.UploaderFirstName,
		"uploader_last_name":  result.UploaderLastName,
		"has_thumbnail":       result.ThumbnailKey != nil,
	})
}

func uploadErrorStatus(err error) int {
	switch err {
	case file.ErrNotChannelMember:
		return http.StatusForbidden
	case file.ErrChannelArchived:
		return http.StatusConflict
	case file.ErrFileTooLarge:
		return http.StatusRequestEntityTooLarge
	case file.ErrInvalidMimeType:
		return http.StatusUnsupportedMediaType
	case file.ErrQuotaExceeded:
		return http.StatusInsufficientStorage
	case file.ErrScanFailed:
		return http.StatusUnprocessableEntity
	}
	return http.StatusInternalServerError
}

func (h *FileUploadHandler) broadcastFileUploaded(ctx context.Context, f *models.ChatFileWithUploader) {
	fileData := map[string]interface{}{
		"id":                  f.ID.String(),
		"channel_id":          f.ChannelID.String(),
		"filename":            f.Filename,
		"mime_type":           f.MimeType,
		"file_size":           f.FileSize,
		"uploaded_by":         f.UploadedBy.String(),
		"uploader_first_name": f.UploaderFirstName,
		"uploader_last_name":  f.UploaderLastName,
	}

	h.wsHub.broadcastToChannel(ctx, f.ChannelID.String(), WSMessage{
		Type:      WSFileUploaded,
		ChannelID: f.ChannelID.String(),
		Message:   mustMarshal(fileData),
	})
}

func uuidPtrToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
