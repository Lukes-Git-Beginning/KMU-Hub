package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/document/wopi"
)

// WOPIRoutes handles the WOPI protocol HTTP routes for OnlyOffice integration.
// These routes use WOPI token authentication (access_token query param),
// NOT the standard application auth middleware.
type WOPIRoutes struct {
	handler *wopi.Handler
}

// NewWOPIRoutes creates new WOPI protocol routes.
func NewWOPIRoutes(handler *wopi.Handler) *WOPIRoutes {
	return &WOPIRoutes{handler: handler}
}

// ServiceName returns the service name.
func (w *WOPIRoutes) ServiceName() string { return "wopi" }

// RegisterRoutes registers WOPI protocol endpoints.
// These are root-level routes (not under /api/v1) per WOPI spec.
// Authentication is handled by the WOPI handler itself via access_token.
func (w *WOPIRoutes) RegisterRoutes(r chi.Router, _ func(http.Handler) http.Handler) {
	r.Route("/wopi/files/{fileID}", func(r chi.Router) {
		// CheckFileInfo: GET /wopi/files/{file_id}
		r.Get("/", func(rw http.ResponseWriter, req *http.Request) {
			fileID := chi.URLParam(req, "fileID")
			w.handler.CheckFileInfo(rw, req, fileID)
		})

		// GetFile: GET /wopi/files/{file_id}/contents
		r.Get("/contents", func(rw http.ResponseWriter, req *http.Request) {
			fileID := chi.URLParam(req, "fileID")
			w.handler.GetFile(rw, req, fileID)
		})

		// PutFile: POST /wopi/files/{file_id}/contents
		r.Post("/contents", func(rw http.ResponseWriter, req *http.Request) {
			fileID := chi.URLParam(req, "fileID")
			w.handler.PutFile(rw, req, fileID)
		})

		// Lock operations: POST /wopi/files/{file_id}
		// X-WOPI-Override header determines the operation (LOCK, UNLOCK, REFRESH_LOCK, GET_LOCK)
		r.Post("/", func(rw http.ResponseWriter, req *http.Request) {
			fileID := chi.URLParam(req, "fileID")
			w.handler.HandleLockOperation(rw, req, fileID)
		})
	})
}
