package gateway

import (
	"context"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

// ============================================================================
// Contact Files Handlers
//
// Files attached to a contact go through the Document service's generic
// file+entity-link mechanism (entity_type="contact") rather than a second,
// CRM-specific file store.
// ============================================================================

// contactFilesFolderName is the shared, tenant-wide system folder that holds
// every file attached to a CRM contact.
//
// lean: one shared folder for all contacts rather than a folder-per-contact
// tree; revisit if contacts ever need their own attachment folder/browser.
const contactFilesFolderName = "CRM-Kontaktanhänge"

// getDocumentClient lazily obtains a gRPC client for the Document service,
// used to store and link the files attached to a contact.
func (c *CRMRoutes) getDocumentClient() (documentv1.DocumentServiceClient, error) {
	conn, err := c.registry.GetConnection("document")
	if err != nil {
		return nil, err
	}
	return documentv1.NewDocumentServiceClient(conn), nil
}

// contactFileResponse mirrors the desktop client's ContactFile type
// (api/hooks/useContacts.ts) -- a flat shape, not the nested DocumentFile
// proto message.
type contactFileResponse struct {
	ID         string `json:"id"`
	ContactID  string `json:"contact_id"`
	Filename   string `json:"filename"`
	MimeType   string `json:"mime_type"`
	StorageKey string `json:"storage_key"`
	CreatedAt  string `json:"created_at"`
}

func toContactFileResponse(contactID string, f *documentv1.DocumentFile) contactFileResponse {
	return contactFileResponse{
		ID:         f.Id,
		ContactID:  contactID,
		Filename:   f.Filename,
		MimeType:   f.MimeType,
		StorageKey: f.StorageKey,
		CreatedAt:  f.CreatedAt.AsTime().Format(time.RFC3339),
	}
}

// verifyContactOwnership confirms the contact exists for the caller's tenant
// before touching the Document service -- RLS alone would let a guessed
// contact ID attach a file link, it just wouldn't be visible again. Returns
// the CRM client's connection error separately so the caller can surface it
// as 503 rather than folding it into the RPC error path.
func (c *CRMRoutes) verifyContactOwnership(ctx context.Context, contactID string) (connErr, rpcErr error) {
	crmClient, err := c.getCRMClient()
	if err != nil {
		return err, nil
	}
	_, err = crmClient.GetContact(ctx, &crmv1.GetContactRequest{Id: contactID})
	return nil, err
}

// HandleListContactFiles handles GET /api/v1/contacts/{id}/files.
func (c *CRMRoutes) HandleListContactFiles(w http.ResponseWriter, r *http.Request) {
	contactID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if connErr, rpcErr := c.verifyContactOwnership(r.Context(), contactID); connErr != nil || rpcErr != nil {
		if connErr != nil {
			respondServiceUnavailable(w, c.ServiceName())
		} else {
			respondGRPCError(w, rpcErr)
		}
		return
	}

	docClient, err := c.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, "document")
		return
	}

	resp, err := docClient.ListFilesByEntity(r.Context(), &documentv1.ListFilesByEntityRequest{
		EntityType: "contact",
		EntityId:   contactID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	files := make([]contactFileResponse, 0, len(resp.Files))
	for _, f := range resp.Files {
		files = append(files, toContactFileResponse(contactID, f))
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{"files": files})
}

// createContactFileRequest is the JSON body for POST /api/v1/contacts/{id}/files.
// storage_key comes from a prior POST /api/v1/files/presign-upload (scope
// "kontakte") call -- the browser uploads the bytes to MinIO directly, then
// this endpoint only records the metadata and links it to the contact.
type createContactFileRequest struct {
	Filename   string `json:"filename"    validate:"required"`
	MimeType   string `json:"mime_type"`
	StorageKey string `json:"storage_key" validate:"required"`
	FileSize   int64  `json:"file_size"   validate:"gt=0"`
}

// HandleCreateContactFile handles POST /api/v1/contacts/{id}/files.
func (c *CRMRoutes) HandleCreateContactFile(w http.ResponseWriter, r *http.Request) {
	contactID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if connErr, rpcErr := c.verifyContactOwnership(r.Context(), contactID); connErr != nil || rpcErr != nil {
		if connErr != nil {
			respondServiceUnavailable(w, c.ServiceName())
		} else {
			respondGRPCError(w, rpcErr)
		}
		return
	}

	req, ok := decodeAndValidate[createContactFileRequest](w, r)
	if !ok {
		return
	}

	docClient, err := c.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, "document")
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID := middleware.GetUserID(r.Context())

	folderID, err := c.resolveContactFilesFolder(r.Context(), docClient, tenantID.String(), userID)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	registerResp, err := docClient.RegisterUploadedFile(r.Context(), &documentv1.RegisterUploadedFileRequest{
		FolderId:   folderID,
		Filename:   req.Filename,
		MimeType:   req.MimeType,
		FileSize:   req.FileSize,
		StorageKey: req.StorageKey,
		OwnerId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	if _, err := docClient.LinkFileToEntity(r.Context(), &documentv1.LinkFileToEntityRequest{
		FileId:     registerResp.File.Id,
		EntityType: "contact",
		EntityId:   contactID,
		LinkedBy:   userID,
	}); err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"file": toContactFileResponse(contactID, registerResp.File),
	})
}

// resolveContactFilesFolder returns the ID of the shared system folder that
// holds contact attachments, creating it on first use. Concurrent first
// callers race on CreateFolder; the folder Service rejects the loser with a
// name-conflict error (AlreadyExists), so the loser just re-fetches the
// winner's folder instead of failing the request.
func (c *CRMRoutes) resolveContactFilesFolder(ctx context.Context, client documentv1.DocumentServiceClient, tenantSpaceID, userID string) (string, error) {
	if id, err := c.findContactFilesFolder(ctx, client, tenantSpaceID); err != nil || id != "" {
		return id, err
	}

	createResp, err := client.CreateFolder(ctx, &documentv1.CreateFolderRequest{
		Name:      contactFilesFolderName,
		SpaceType: documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_TEAM,
		SpaceId:   tenantSpaceID,
		Icon:      "paperclip",
		CreatedBy: userID,
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return c.findContactFilesFolder(ctx, client, tenantSpaceID)
		}
		return "", err
	}
	return createResp.Folder.Id, nil
}

func (c *CRMRoutes) findContactFilesFolder(ctx context.Context, client documentv1.DocumentServiceClient, tenantSpaceID string) (string, error) {
	listResp, err := client.ListFolders(ctx, &documentv1.ListFoldersRequest{
		SpaceType: documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_TEAM,
		SpaceId:   tenantSpaceID,
	})
	if err != nil {
		return "", err
	}
	for _, f := range listResp.Folders {
		if f.Name == contactFilesFolderName {
			return f.Id, nil
		}
	}
	return "", nil
}
