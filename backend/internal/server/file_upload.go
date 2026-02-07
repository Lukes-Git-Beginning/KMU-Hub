package server

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// FileUploadHandler handles multipart file uploads directly (not via gRPC)
type FileUploadHandler struct {
	fileService *file.Service
	wsHub       *WebSocketHub
}

// NewFileUploadHandler creates a new file upload handler
func NewFileUploadHandler(fileService *file.Service, wsHub *WebSocketHub) *FileUploadHandler {
	return &FileUploadHandler{
		fileService: fileService,
		wsHub:       wsHub,
	}
}

// maxUploadSize is the maximum request body size for file uploads (55 MB to allow overhead)
const maxUploadSize = 55 << 20

// HandleUploadFile handles multipart file upload
// POST /api/v1/files/upload
// Form fields: file (required), channel_id (required), message_id (optional)
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

	// Get channel_id
	channelIDStr := r.FormValue("channel_id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "valid channel_id is required")
		return
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

	input := file.UploadInput{
		ChannelID: channelID,
		MessageID: messageID,
		Filename:  header.Filename,
		MimeType:  mimeType,
		FileSize:  header.Size,
		Reader:    formFile,
		UserID:    uploaderID,
	}

	result, uploadErr := h.fileService.Upload(r.Context(), input)
	if uploadErr != nil {
		statusCode := http.StatusInternalServerError
		switch uploadErr {
		case file.ErrNotChannelMember:
			statusCode = http.StatusForbidden
		case file.ErrChannelArchived:
			statusCode = http.StatusConflict
		case file.ErrFileTooLarge:
			statusCode = http.StatusRequestEntityTooLarge
		case file.ErrInvalidMimeType:
			statusCode = http.StatusUnsupportedMediaType
		case file.ErrQuotaExceeded:
			statusCode = http.StatusInsufficientStorage
		case file.ErrScanFailed:
			statusCode = http.StatusUnprocessableEntity
		}
		response.Error(w, statusCode, uploadErr.Error())
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
