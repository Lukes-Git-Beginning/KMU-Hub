package wopi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/models"
)

// WOPIServerVersion is included in all WOPI responses.
const WOPIServerVersion = "kmuhub-wopi/1.0"

// FileServiceInterface defines the minimal methods the WOPI handler needs from the file service.
// Keeps this package decoupled from the file package to avoid circular imports.
type FileServiceInterface interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.DocumentFile, error)
	GetDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error)
	CreateVersion(ctx context.Context, fileID uuid.UUID, input VersionInput) (*models.DocumentFileVersion, error)
}

// VersionInput mirrors file.VersionInput for WOPI auto-versioning.
type VersionInput struct {
	Reader   io.Reader
	FileSize int64
	MimeType string
	Label    *string
	UserID   uuid.UUID
}

// Handler implements WOPI REST endpoints for OnlyOffice integration.
type Handler struct {
	tokenService *TokenService
	lockService  *LockService
	fileService  FileServiceInterface
	fileStore    chatfile.FileStore
}

// NewHandler creates a new WOPI handler.
func NewHandler(tokenService *TokenService, lockService *LockService, fileService FileServiceInterface, fileStore chatfile.FileStore) *Handler {
	return &Handler{
		tokenService: tokenService,
		lockService:  lockService,
		fileService:  fileService,
		fileStore:    fileStore,
	}
}

// CheckFileInfo handles GET /wopi/files/{file_id}
// Returns file metadata as required by the WOPI spec.
func (h *Handler) CheckFileInfo(w http.ResponseWriter, r *http.Request, fileIDStr string) {
	claims, ok := h.validateAccessToken(w, r)
	if !ok {
		return
	}

	if claims.FileID != fileIDStr {
		http.Error(w, "token file_id mismatch", http.StatusForbidden)
		return
	}

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		http.Error(w, "invalid file_id", http.StatusBadRequest)
		return
	}

	file, err := h.fileService.GetByID(r.Context(), fileID)
	if err != nil {
		slog.Warn("WOPI CheckFileInfo: file not found", "file_id", fileIDStr, "error", err)
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	info := map[string]interface{}{
		"BaseFileName":     file.Filename,
		"Size":             file.FileSize,
		"Version":          file.CurrentVersion,
		"OwnerId":          file.OwnerID.String(),
		"UserId":           claims.UserID,
		"UserFriendlyName": claims.UserName,
		"UserCanWrite":     claims.CanWrite,
		"UserCanRename":    claims.CanWrite,
		"SupportsLocks":    true,
		"SupportsUpdate":   true,
		"SupportsRename":   true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-WOPI-ServerVersion", WOPIServerVersion)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(info)
}

// GetFile handles GET /wopi/files/{file_id}/contents
// Streams the file content to the response.
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request, fileIDStr string) {
	claims, ok := h.validateAccessToken(w, r)
	if !ok {
		return
	}

	if claims.FileID != fileIDStr {
		http.Error(w, "token file_id mismatch", http.StatusForbidden)
		return
	}

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		http.Error(w, "invalid file_id", http.StatusBadRequest)
		return
	}

	file, err := h.fileService.GetByID(r.Context(), fileID)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	reader, err := h.fileStore.Download(r.Context(), file.StorageKey)
	if err != nil {
		slog.Error("WOPI GetFile: failed to download", "file_id", fileIDStr, "error", err)
		http.Error(w, "failed to download file", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("X-WOPI-ItemVersion", string(rune(file.CurrentVersion+'0')))
	w.Header().Set("X-WOPI-ServerVersion", WOPIServerVersion)
	w.WriteHeader(http.StatusOK)
	_ = io.Copy(w, reader)
}

// PutFile handles POST /wopi/files/{file_id}/contents
// Saves new file content from OnlyOffice (auto-versioning on save).
func (h *Handler) PutFile(w http.ResponseWriter, r *http.Request, fileIDStr string) {
	claims, ok := h.validateAccessToken(w, r)
	if !ok {
		return
	}

	if claims.FileID != fileIDStr {
		http.Error(w, "token file_id mismatch", http.StatusForbidden)
		return
	}

	if !claims.CanWrite {
		http.Error(w, "write permission required", http.StatusForbidden)
		return
	}

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		http.Error(w, "invalid file_id", http.StatusBadRequest)
		return
	}

	// Verify lock if file is locked
	wopiLock := r.Header.Get("X-WOPI-Lock")
	currentLock, lockErr := h.lockService.GetLock(r.Context(), fileIDStr)
	if lockErr == nil && currentLock != "" && currentLock != wopiLock {
		// File is locked by a different lock
		w.Header().Set("X-WOPI-Lock", currentLock)
		w.Header().Set("X-WOPI-ServerVersion", WOPIServerVersion)
		http.Error(w, "lock mismatch", http.StatusConflict)
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		http.Error(w, "invalid user in token", http.StatusBadRequest)
		return
	}

	file, err := h.fileService.GetByID(r.Context(), fileID)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	label := "Auto-saved via WOPI"
	versionInput := VersionInput{
		Reader:   r.Body,
		FileSize: r.ContentLength,
		MimeType: file.MimeType,
		Label:    &label,
		UserID:   userID,
	}

	_, err = h.fileService.CreateVersion(r.Context(), fileID, versionInput)
	if err != nil {
		slog.Error("WOPI PutFile: failed to create version", "file_id", fileIDStr, "error", err)
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-WOPI-ServerVersion", WOPIServerVersion)
	w.WriteHeader(http.StatusOK)
}

// HandleLockOperation handles POST /wopi/files/{file_id}
// Dispatches to Lock, Unlock, or RefreshLock based on X-WOPI-Override header.
func (h *Handler) HandleLockOperation(w http.ResponseWriter, r *http.Request, fileIDStr string) {
	claims, ok := h.validateAccessToken(w, r)
	if !ok {
		return
	}

	if claims.FileID != fileIDStr {
		http.Error(w, "token file_id mismatch", http.StatusForbidden)
		return
	}

	override := r.Header.Get("X-WOPI-Override")
	lockID := r.Header.Get("X-WOPI-Lock")

	w.Header().Set("X-WOPI-ServerVersion", WOPIServerVersion)

	switch override {
	case "LOCK":
		err := h.lockService.Lock(r.Context(), fileIDStr, lockID, claims.UserID)
		if err != nil {
			var conflictErr *LockConflictError
			if errors.As(err, &conflictErr) {
				w.Header().Set("X-WOPI-Lock", conflictErr.CurrentLockID)
				http.Error(w, "lock conflict", http.StatusConflict)
				return
			}
			slog.Error("WOPI Lock failed", "file_id", fileIDStr, "error", err)
			http.Error(w, "lock failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "UNLOCK":
		err := h.lockService.Unlock(r.Context(), fileIDStr, lockID)
		if err != nil {
			if errors.Is(err, ErrLockNotFound) {
				http.Error(w, "lock not found", http.StatusConflict)
				return
			}
			slog.Error("WOPI Unlock failed", "file_id", fileIDStr, "error", err)
			http.Error(w, "unlock failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "REFRESH_LOCK":
		err := h.lockService.RefreshLock(r.Context(), fileIDStr, lockID)
		if err != nil {
			if errors.Is(err, ErrLockNotFound) {
				http.Error(w, "lock not found", http.StatusConflict)
				return
			}
			slog.Error("WOPI RefreshLock failed", "file_id", fileIDStr, "error", err)
			http.Error(w, "refresh lock failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "GET_LOCK":
		currentLock, err := h.lockService.GetLock(r.Context(), fileIDStr)
		if err != nil {
			w.Header().Set("X-WOPI-Lock", "")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("X-WOPI-Lock", currentLock)
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "unsupported WOPI operation: "+override, http.StatusNotImplemented)
	}
}

// validateAccessToken validates the WOPI access_token query parameter.
func (h *Handler) validateAccessToken(w http.ResponseWriter, r *http.Request) (*WOPITokenClaims, bool) {
	token := r.URL.Query().Get("access_token")
	if token == "" {
		http.Error(w, "access_token required", http.StatusUnauthorized)
		return nil, false
	}

	claims, err := h.tokenService.Validate(token)
	if err != nil {
		http.Error(w, "invalid access_token", http.StatusUnauthorized)
		return nil, false
	}

	return claims, true
}
