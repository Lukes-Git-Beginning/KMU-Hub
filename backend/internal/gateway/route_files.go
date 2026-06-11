package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

// FileRoutes handles generic presigned upload/download URL endpoints.
// These routes allow browser clients to upload files directly to MinIO
// without streaming through the gateway, using short-lived presigned URLs.
type FileRoutes struct {
	registry *ServiceRegistry
}

// NewFileRoutes creates a new FileRoutes with the given service registry.
func NewFileRoutes(registry *ServiceRegistry) *FileRoutes {
	return &FileRoutes{registry: registry}
}

// ServiceName returns the logical service name for logging.
func (f *FileRoutes) ServiceName() string { return "files" }

// getDocumentClient lazily obtains a gRPC client for the Document service,
// which hosts the presigned URL generation logic.
func (f *FileRoutes) getDocumentClient() (documentv1.DocumentServiceClient, error) {
	conn, err := f.registry.GetConnection("document")
	if err != nil {
		return nil, err
	}
	return documentv1.NewDocumentServiceClient(conn), nil
}

// RegisterRoutes registers presigned URL routes.
// Both routes require authentication — the tenant ID is read from the JWT
// by the Auth middleware and propagated to the document service via gRPC metadata.
func (f *FileRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/files", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("files", "write")).
			Post("/presign-upload", f.HandlePresignUpload)
		r.With(middleware.RequirePermission("files", "read")).
			Get("/presign-download", f.HandlePresignDownload)
	})
}

// presignUploadRequest is the JSON body for POST /api/v1/files/presign-upload.
type presignUploadRequest struct {
	Scope       string `json:"scope"        validate:"required"`
	FileName    string `json:"file_name"    validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
	SizeBytes   int64  `json:"size_bytes"   validate:"gt=0"`
}

// HandlePresignUpload handles POST /api/v1/files/presign-upload.
// Returns a short-lived (15 min) presigned PUT URL for browser-direct upload.
func (f *FileRoutes) HandlePresignUpload(w http.ResponseWriter, r *http.Request) {
	client, err := f.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, f.ServiceName())
		return
	}

	req, ok := decodeAndValidate[presignUploadRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.GetPresignedUploadURL(r.Context(), &documentv1.GetPresignedUploadURLRequest{
		Scope:       req.Scope,
		FileName:    req.FileName,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"upload_url": resp.UploadUrl,
		"object_key": resp.ObjectKey,
		"expires_at": resp.ExpiresAt.AsTime(),
	})
}

// HandlePresignDownload handles GET /api/v1/files/presign-download?object_key=...
// Returns a short-lived (1 hour) presigned GET URL.
// The document service enforces tenant ownership of the object key.
func (f *FileRoutes) HandlePresignDownload(w http.ResponseWriter, r *http.Request) {
	client, err := f.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, f.ServiceName())
		return
	}

	objectKey := r.URL.Query().Get("object_key")
	if objectKey == "" {
		response.Error(w, http.StatusBadRequest, "query parameter 'object_key' is required")
		return
	}

	resp, err := client.GetPresignedDownloadURL(r.Context(), &documentv1.GetPresignedDownloadURLRequest{
		ObjectKey: objectKey,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"download_url": resp.DownloadUrl,
		"object_key":   objectKey,
		"expires_at":   resp.ExpiresAt.AsTime(),
	})
}
