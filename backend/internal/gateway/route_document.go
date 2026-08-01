package gateway

import (
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

// DocumentRoutes handles HTTP routes for the Document backend service.
//
// lean: handlers that wrap a proto value in an ad-hoc envelope
// (e.g. map[string]any{"folder": resp.Folder}) stay on response.JSON rather
// than being migrated to protojson like the other modules. Two reasons hold
// today: (1) the desktop document-client.ts already runs normalizeWireTimestamps
// over every response, so the {seconds,nanos} timestamps are corrected
// client-side; (2) DocumentFile.file_size is int64 and the FE reads it as a
// number — protojson would emit it as a string and break the size display.
// Upgrade path: when the FE wire-time normalizer is removed module-wide, switch
// these to a proto-to-json.RawMessage envelope helper and coerce file_size on
// the FE (as inventar/work already do).
type DocumentRoutes struct {
	registry *ServiceRegistry
}

// NewDocumentRoutes creates a new DocumentRoutes with the given service registry.
func NewDocumentRoutes(registry *ServiceRegistry) *DocumentRoutes {
	return &DocumentRoutes{registry: registry}
}

// ServiceName returns the backend service name.
func (d *DocumentRoutes) ServiceName() string { return "document" }

// getDocumentClient lazily obtains a gRPC client for the Document service.
func (d *DocumentRoutes) getDocumentClient() (documentv1.DocumentServiceClient, error) {
	conn, err := d.registry.GetConnection("document")
	if err != nil {
		return nil, err
	}
	return documentv1.NewDocumentServiceClient(conn), nil
}

// RegisterRoutes registers all Document HTTP routes.
func (d *DocumentRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Additive guards: every route keeps its legacy coarse key AND accepts the
	// matching capability-catalogue key. Extend, never swap (see
	// middleware.RequirePermissionAny doc comment).
	//
	// Mapping verified against FileContextMenu.tsx/DokumentePage.tsx, not
	// assumed: canEdit (documents:file:edit) gates rename+move on files AND
	// new-subfolder+rename on folders — "new folder" in DokumentePage.tsx:144
	// is explicitly `canEditAllowed`, not upload. canUpload
	// (documents:file:upload) only gates registering an uploaded file.
	// Copy is commented "read-action — always visible" in the FE with no
	// capability check at all, so it keeps its coarse key only. Version
	// history is only reachable behind the version:restore menu item
	// (FileContextMenu.tsx:245), so the LIST route gets that alt key too, not
	// file:read. documents:template:manage and documents:share_link:create
	// have zero backend wiring (TemplateGalleryDialog and ShareDialog's
	// copyLink are FE-only mocks, verified no API call in either) — both stay
	// unwired, no route invented. Tags, entity links, search, virtual files,
	// WOPI and space initialization have no catalogue subject at all.
	var (
		docRead           = middleware.RequirePermissionAny([2]string{"documents", "read"}, [2]string{"documents:file", "read"})
		docDownload       = middleware.RequirePermissionAny([2]string{"documents", "read"}, [2]string{"documents:file", "download"})
		docUpload         = middleware.RequirePermissionAny([2]string{"documents", "write"}, [2]string{"documents:file", "upload"})
		docEdit           = middleware.RequirePermissionAny([2]string{"documents", "write"}, [2]string{"documents:file", "edit"})
		docDelete         = middleware.RequirePermissionAny([2]string{"documents", "delete"}, [2]string{"documents:file", "delete"})
		docShareManage    = middleware.RequirePermissionAny([2]string{"documents", "write"}, [2]string{"documents:share", "manage"})
		docVersionRestore = middleware.RequirePermissionAny([2]string{"documents", "write"}, [2]string{"documents:version", "restore"})
	)

	// Folders
	r.Route("/api/v1/documents/folders", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(docRead).Get("/", d.HandleListFolders)
		r.With(docRead).Get("/{id}", d.HandleGetFolder)
		r.With(docEdit).Post("/", d.HandleCreateFolder)
		r.With(docEdit).Put("/{id}", d.HandleUpdateFolder)
		r.With(docDelete).Delete("/{id}", d.HandleDeleteFolder)
		r.With(docRead).Get("/{id}/path", d.HandleGetFolderPath)
		r.With(middleware.RequirePermission("documents", "write")).Post("/initialize-user", d.HandleInitializeUserSpace)
		r.With(middleware.RequirePermission("documents", "write")).Post("/initialize-team", d.HandleInitializeTeamSpace)
	})

	// Files
	r.Route("/api/v1/documents/files", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(docRead).Get("/", d.HandleListFiles)
		// Register metadata for a file already uploaded to MinIO via a presigned
		// PUT URL (POST /api/v1/files/presign-upload → PUT to MinIO → register here).
		r.With(docUpload).Post("/", d.HandleRegisterUploadedFile)
		r.With(docRead).Get("/{id}", d.HandleGetFile)
		r.With(docEdit).Put("/{id}", d.HandleUpdateFile)
		r.With(docDelete).Delete("/{id}", d.HandleDeleteFile)
		r.With(middleware.RequirePermission("documents", "write")).Post("/{id}/copy", d.HandleCopyFile)
		r.With(docEdit).Post("/{id}/move", d.HandleMoveFile)
		r.With(docDownload).Get("/{id}/download-url", d.HandleGetFileDownloadURL)
		// Versions
		r.With(docVersionRestore).Get("/{id}/versions", d.HandleListFileVersions)
		r.With(docVersionRestore).Post("/{id}/versions/revert", d.HandleRevertFileVersion)
		// Entity links
		r.With(middleware.RequirePermission("documents", "read")).Get("/{id}/links", d.HandleListFileEntityLinks)
		r.With(middleware.RequirePermission("documents", "write")).Post("/{id}/links", d.HandleLinkFileToEntity)
		r.With(middleware.RequirePermission("documents", "write")).Delete("/{id}/links", d.HandleUnlinkFileFromEntity)
	})

	// Shares
	r.Route("/api/v1/documents/shares", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(docShareManage).Post("/", d.HandleShareEntity)
		r.With(docShareManage).Delete("/", d.HandleUnshareEntity)
		r.With(docRead).Get("/entity", d.HandleListShares)
		r.With(docRead).Get("/shared-with-me", d.HandleListSharedWithMe)
	})

	// Tags
	r.Route("/api/v1/documents/tags", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("documents", "read")).Get("/", d.HandleListTags)
		r.With(middleware.RequirePermission("documents", "write")).Post("/", d.HandleCreateTag)
		r.With(middleware.RequirePermission("documents", "delete")).Delete("/{id}", d.HandleDeleteTag)
		r.With(middleware.RequirePermission("documents", "write")).Post("/file", d.HandleTagFile)
		r.With(middleware.RequirePermission("documents", "write")).Delete("/file", d.HandleUntagFile)
	})

	// Search
	r.Route("/api/v1/documents/search", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("documents", "read")).Get("/", d.HandleSearchFiles)
	})

	// Virtual Files
	r.Route("/api/v1/documents/virtual", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("documents", "read")).Get("/", d.HandleListVirtualFiles)
	})

	// WOPI Token Generation (authenticated, not WOPI protocol)
	r.Route("/api/v1/documents/wopi", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("documents", "write")).Post("/token", d.HandleGenerateWOPIToken)
		r.With(middleware.RequirePermission("documents", "read")).Get("/discovery", d.HandleGetWOPIDiscovery)
	})
}

// ============================================================================
// Folder Handlers
// ============================================================================

type createFolderRequest struct {
	Name      string  `json:"name"       validate:"required"`
	ParentID  *string `json:"parent_id"  validate:"omitempty,uuid"`
	SpaceType string  `json:"space_type" validate:"omitempty,oneof=personal team project"`
	SpaceID   *string `json:"space_id"   validate:"omitempty,uuid"`
	Icon      string  `json:"icon"`
}

func (d *DocumentRoutes) HandleCreateFolder(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createFolderRequest](w, r)
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

	var parentID, spaceID string
	if req.ParentID != nil {
		parentID = *req.ParentID
	}
	if req.SpaceID != nil {
		spaceID = *req.SpaceID
	}
	resp, err := client.CreateFolder(r.Context(), &documentv1.CreateFolderRequest{
		Name:      req.Name,
		ParentId:  parentID,
		SpaceType: spaceTypeToProto(req.SpaceType),
		SpaceId:   spaceID,
		Icon:      req.Icon,
		CreatedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{"folder": resp.Folder})
}

func (d *DocumentRoutes) HandleGetFolder(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	resp, err := client.GetFolder(r.Context(), &documentv1.GetFolderRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{"folder": resp.Folder})
}

func (d *DocumentRoutes) HandleListFolders(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	resp, err := client.ListFolders(r.Context(), &documentv1.ListFoldersRequest{
		ParentId:  r.URL.Query().Get("parent_id"),
		SpaceType: spaceTypeToProto(r.URL.Query().Get("space_type")),
		SpaceId:   r.URL.Query().Get("space_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoListWrapped(w, http.StatusOK, "folders", resp.Folders, map[string]any{"total": len(resp.Folders)})
}

type updateFolderRequest struct {
	Name     *string `json:"name"`
	ParentID *string `json:"parent_id" validate:"omitempty,uuid"`
	Icon     *string `json:"icon"`
}

func (d *DocumentRoutes) HandleUpdateFolder(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[updateFolderRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateFolder(r.Context(), &documentv1.UpdateFolderRequest{
		Id:       id,
		Name:     req.Name,
		ParentId: req.ParentID,
		Icon:     req.Icon,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{"folder": resp.Folder})
}

func (d *DocumentRoutes) HandleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	_, err = client.DeleteFolder(r.Context(), &documentv1.DeleteFolderRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (d *DocumentRoutes) HandleGetFolderPath(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	resp, err := client.GetFolderPath(r.Context(), &documentv1.GetFolderPathRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoListWrapped(w, http.StatusOK, "segments", resp.Segments, nil)
}

type initializeUserSpaceRequest struct {
	UserID *string `json:"user_id" validate:"omitempty,uuid"`
}

func (d *DocumentRoutes) HandleInitializeUserSpace(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	// lean: tolerate empty body — user_id is optional; fall back to JWT identity
	var req initializeUserSpaceRequest
	if r.ContentLength != 0 {
		var ok bool
		req, ok = decodeAndValidate[initializeUserSpaceRequest](w, r)
		if !ok {
			return
		}
	}

	var userID string
	if req.UserID != nil {
		userID = *req.UserID
	}
	if userID == "" {
		userID = middleware.GetUserID(r.Context())
	}

	_, err = client.InitializeUserSpace(r.Context(), &documentv1.InitializeUserSpaceRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"initialized": true})
}

type initializeTeamSpaceRequest struct {
	TeamID *string `json:"team_id" validate:"omitempty,uuid"`
}

func (d *DocumentRoutes) HandleInitializeTeamSpace(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	req, ok := decodeAndValidate[initializeTeamSpaceRequest](w, r)
	if !ok {
		return
	}

	var teamID string
	if req.TeamID != nil {
		teamID = *req.TeamID
	}
	_, err = client.InitializeTeamSpace(r.Context(), &documentv1.InitializeTeamSpaceRequest{
		TeamId: teamID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"initialized": true})
}

// ============================================================================
// File Handlers
// ============================================================================

func (d *DocumentRoutes) HandleGetFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	resp, err := client.GetFile(r.Context(), &documentv1.GetFileRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{"file": resp.File})
}

type registerUploadedFileRequest struct {
	FolderID   string `json:"folder_id"   validate:"required,uuid"`
	Filename   string `json:"filename"    validate:"required"`
	MimeType   string `json:"mime_type"`
	FileSize   int64  `json:"file_size"   validate:"gt=0"`
	StorageKey string `json:"storage_key" validate:"required"`
}

// HandleRegisterUploadedFile records metadata for a file the browser already
// uploaded to MinIO through a presigned PUT URL. Owner is taken from the JWT.
func (d *DocumentRoutes) HandleRegisterUploadedFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[registerUploadedFileRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.RegisterUploadedFile(r.Context(), &documentv1.RegisterUploadedFileRequest{
		FolderId:   req.FolderID,
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

	response.JSON(w, http.StatusCreated, map[string]interface{}{"file": resp.File})
}

func (d *DocumentRoutes) HandleListFiles(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)

	req := &documentv1.ListFilesRequest{
		FolderId: r.URL.Query().Get("folder_id"),
		OwnerId:  r.URL.Query().Get("owner_id"),
		Page:     int32(page),
		PerPage:  int32(pageSize),
	}

	if sortField := r.URL.Query().Get("sort"); sortField != "" {
		req.SortField = fileSortFieldToProto(sortField)
	}
	if sortDir := r.URL.Query().Get("sort_dir"); sortDir != "" {
		req.SortDirection = sortDirToProto(sortDir)
	}
	if r.URL.Query().Get("favorites") == "true" {
		req.FilterFavorites = true
		req.IsFavorite = true
	}

	resp, err := client.ListFiles(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"files": emptyIfNil(resp.Files),
		"total": resp.Total,
	})
}

type updateFileRequest struct {
	Filename   *string `json:"filename"`
	FolderID   *string `json:"folder_id"   validate:"omitempty,uuid"`
	IsFavorite *bool   `json:"is_favorite"`
}

func (d *DocumentRoutes) HandleUpdateFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[updateFileRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateFile(r.Context(), &documentv1.UpdateFileRequest{
		Id:         id,
		Filename:   req.Filename,
		FolderId:   req.FolderID,
		IsFavorite: req.IsFavorite,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{"file": resp.File})
}

func (d *DocumentRoutes) HandleDeleteFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	_, err = client.DeleteFile(r.Context(), &documentv1.DeleteFileRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

type copyFileRequest struct {
	TargetFolderID string `json:"target_folder_id" validate:"required,uuid"`
}

func (d *DocumentRoutes) HandleCopyFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[copyFileRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CopyFile(r.Context(), &documentv1.CopyFileRequest{
		Id:             id,
		TargetFolderId: req.TargetFolderID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{"file": resp.File})
}

type moveFileRequest struct {
	TargetFolderID string `json:"target_folder_id" validate:"required,uuid"`
}

func (d *DocumentRoutes) HandleMoveFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[moveFileRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.MoveFile(r.Context(), &documentv1.MoveFileRequest{
		Id:             id,
		TargetFolderId: req.TargetFolderID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{"file": resp.File})
}

func (d *DocumentRoutes) HandleGetFileDownloadURL(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	resp, err := client.GetFileDownloadURL(r.Context(), &documentv1.GetFileDownloadURLRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"download_url": resp.DownloadUrl,
		"filename":     resp.Filename,
		"content_type": resp.ContentType,
		"file_size":    resp.FileSize,
	})
}

// ============================================================================
// Version Handlers
// ============================================================================

func (d *DocumentRoutes) HandleListFileVersions(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")
	resp, err := client.ListFileVersions(r.Context(), &documentv1.ListFileVersionsRequest{
		FileId: fileID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoListWrapped(w, http.StatusOK, "versions", resp.Versions, nil)
}

type revertVersionRequest struct {
	VersionNumber int32 `json:"version_number" validate:"gt=0"`
}

func (d *DocumentRoutes) HandleRevertFileVersion(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[revertVersionRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.RevertFileVersion(r.Context(), &documentv1.RevertFileVersionRequest{
		FileId:        fileID,
		VersionNumber: req.VersionNumber,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// FE expects {version}; resp.File is the now-current file after revert. lean: rename key only;
	// if FE needs version metadata instead of the file object, add a version field to the proto.
	response.JSON(w, http.StatusOK, map[string]interface{}{"version": resp.File})
}

// ============================================================================
// Entity Link Handlers
// ============================================================================

type linkFileToEntityRequest struct {
	EntityType string `json:"entity_type" validate:"required"`
	EntityID   string `json:"entity_id"   validate:"required,uuid"`
}

func (d *DocumentRoutes) HandleLinkFileToEntity(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[linkFileToEntityRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.LinkFileToEntity(r.Context(), &documentv1.LinkFileToEntityRequest{
		FileId:     fileID,
		EntityType: req.EntityType,
		EntityId:   req.EntityID,
		LinkedBy:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{"link": resp.Link})
}

type unlinkFileFromEntityRequest struct {
	EntityType string `json:"entity_type" validate:"required"`
	EntityID   string `json:"entity_id"   validate:"required,uuid"`
}

func (d *DocumentRoutes) HandleUnlinkFileFromEntity(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[unlinkFileFromEntityRequest](w, r)
	if !ok {
		return
	}

	_, err = client.UnlinkFileFromEntity(r.Context(), &documentv1.UnlinkFileFromEntityRequest{
		FileId:     fileID,
		EntityType: req.EntityType,
		EntityId:   req.EntityID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"unlinked": true})
}

func (d *DocumentRoutes) HandleListFileEntityLinks(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")
	resp, err := client.ListFileEntityLinks(r.Context(), &documentv1.ListFileEntityLinksRequest{
		FileId: fileID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoListWrapped(w, http.StatusOK, "links", resp.Links, nil)
}

// ============================================================================
// Share Handlers
// ============================================================================

type shareEntityRequest struct {
	EntityType       string `json:"entity_type"         validate:"required"`
	EntityID         string `json:"entity_id"           validate:"required,uuid"`
	SharedWithUserID string `json:"shared_with_user_id" validate:"required,uuid"`
	Permission       string `json:"permission"          validate:"required,oneof=read write"`
}

func (d *DocumentRoutes) HandleShareEntity(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[shareEntityRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ShareEntity(r.Context(), &documentv1.ShareEntityRequest{
		EntityType:       req.EntityType,
		EntityId:         req.EntityID,
		SharedWithUserId: req.SharedWithUserID,
		Permission:       sharePermissionToProto(req.Permission),
		SharedBy:         userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{"share": resp.Share})
}

type unshareEntityRequest struct {
	EntityType       string `json:"entity_type"         validate:"required"`
	EntityID         string `json:"entity_id"           validate:"required,uuid"`
	SharedWithUserID string `json:"shared_with_user_id" validate:"required,uuid"`
}

func (d *DocumentRoutes) HandleUnshareEntity(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	req, ok := decodeAndValidate[unshareEntityRequest](w, r)
	if !ok {
		return
	}

	_, err = client.UnshareEntity(r.Context(), &documentv1.UnshareEntityRequest{
		EntityType:       req.EntityType,
		EntityId:         req.EntityID,
		SharedWithUserId: req.SharedWithUserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"unshared": true})
}

func (d *DocumentRoutes) HandleListShares(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	entityType := r.URL.Query().Get("entity_type")
	entityID := r.URL.Query().Get("entity_id")

	if entityType == "" || entityID == "" {
		response.Error(w, http.StatusBadRequest, "entity_type and entity_id are required")
		return
	}

	resp, err := client.ListShares(r.Context(), &documentv1.ListSharesRequest{
		EntityType: entityType,
		EntityId:   entityID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoListWrapped(w, http.StatusOK, "shares", resp.Shares, nil)
}

func (d *DocumentRoutes) HandleListSharedWithMe(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.ListSharedWithMe(r.Context(), &documentv1.ListSharedWithMeRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"files":   emptyIfNil(resp.Files),
		"folders": emptyIfNil(resp.Folders),
		"total":   resp.Total,
	})
}

// ============================================================================
// Tag Handlers
// ============================================================================

func (d *DocumentRoutes) HandleListTags(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	resp, err := client.ListTags(r.Context(), &documentv1.ListTagsRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoListWrapped(w, http.StatusOK, "tags", resp.Tags, nil)
}

type createDocumentTagRequest struct {
	Name  string `json:"name"  validate:"required"`
	Color string `json:"color"`
}

func (d *DocumentRoutes) HandleCreateTag(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createDocumentTagRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateTag(r.Context(), &documentv1.CreateTagRequest{
		Name:      req.Name,
		Color:     req.Color,
		CreatedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{"tag": resp.Tag})
}

func (d *DocumentRoutes) HandleDeleteTag(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	_, err = client.DeleteTag(r.Context(), &documentv1.DeleteTagRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

type tagFileRequest struct {
	FileID string `json:"file_id" validate:"required,uuid"`
	TagID  string `json:"tag_id"  validate:"required,uuid"`
}

func (d *DocumentRoutes) HandleTagFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	req, ok := decodeAndValidate[tagFileRequest](w, r)
	if !ok {
		return
	}

	_, err = client.TagFile(r.Context(), &documentv1.TagFileRequest{
		FileId: req.FileID,
		TagId:  req.TagID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"tagged": true})
}

type untagFileRequest struct {
	FileID string `json:"file_id" validate:"required,uuid"`
	TagID  string `json:"tag_id"  validate:"required,uuid"`
}

func (d *DocumentRoutes) HandleUntagFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	req, ok := decodeAndValidate[untagFileRequest](w, r)
	if !ok {
		return
	}

	_, err = client.UntagFile(r.Context(), &documentv1.UntagFileRequest{
		FileId: req.FileID,
		TagId:  req.TagID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"untagged": true})
}

// ============================================================================
// Search Handler
// ============================================================================

func (d *DocumentRoutes) HandleSearchFiles(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		response.Error(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.SearchFiles(r.Context(), &documentv1.SearchFilesRequest{
		Query:    query,
		FolderId: r.URL.Query().Get("folder_id"),
		TagIds:   r.URL.Query()["tag_ids"],
		Page:     int32(page),
		PerPage:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"results": emptyIfNil(resp.Results),
		"total":   resp.Total,
	})
}

// ============================================================================
// Virtual Files Handler
// ============================================================================

func (d *DocumentRoutes) HandleListVirtualFiles(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)

	sourceType := virtualFileSourceToProto(r.URL.Query().Get("source"))

	resp, err := client.ListVirtualFiles(r.Context(), &documentv1.ListVirtualFilesRequest{
		SourceType: sourceType,
		Page:       int32(page),
		PerPage:    int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"files": emptyIfNil(resp.Files),
		"total": resp.Total,
	})
}

// ============================================================================
// WOPI Token Handler
// ============================================================================

type generateWOPITokenRequest struct {
	FileID string `json:"file_id" validate:"required,uuid"`
}

func (d *DocumentRoutes) HandleGenerateWOPIToken(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[generateWOPITokenRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.GenerateWOPIToken(r.Context(), &documentv1.GenerateWOPITokenRequest{
		FileId: req.FileID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"access_token":     resp.AccessToken,
		"access_token_ttl": resp.TtlSeconds,
	})
}

func (d *DocumentRoutes) HandleGetWOPIDiscovery(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	resp, err := client.GetWOPIDiscovery(r.Context(), &documentv1.GetWOPIDiscoveryRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoList(w, http.StatusOK, resp.Actions)
}

// emptyIfNil coalesces a nil slice to a non-nil empty one before it reaches
// encoding/json, which serializes nil as `null` and a non-nil empty slice as
// `[]`. Handlers here mostly go through response.ProtoList(Wrapped), which
// already guarantees `[]`; this covers the few that still hand a raw proto
// repeated field to response.JSON.
func emptyIfNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// ============================================================================
// Proto Enum Helpers
// ============================================================================

func spaceTypeToProto(st string) documentv1.FolderSpaceType {
	switch st {
	case "personal":
		return documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_PERSONAL
	case "team":
		return documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_TEAM
	case "project":
		return documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_PROJECT
	default:
		return documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_UNSPECIFIED
	}
}

func sharePermissionToProto(p string) documentv1.SharePermission {
	switch p {
	case "read":
		return documentv1.SharePermission_SHARE_PERMISSION_READ
	case "write":
		return documentv1.SharePermission_SHARE_PERMISSION_WRITE
	default:
		return documentv1.SharePermission_SHARE_PERMISSION_READ
	}
}

func fileSortFieldToProto(f string) documentv1.FileSortField {
	switch f {
	case "name":
		return documentv1.FileSortField_FILE_SORT_NAME
	case "size":
		return documentv1.FileSortField_FILE_SORT_SIZE
	case "date":
		return documentv1.FileSortField_FILE_SORT_DATE
	case "type":
		return documentv1.FileSortField_FILE_SORT_TYPE
	default:
		return documentv1.FileSortField_FILE_SORT_UNSPECIFIED
	}
}

func sortDirToProto(d string) documentv1.SortDirection {
	switch d {
	case "asc":
		return documentv1.SortDirection_SORT_ASC
	case "desc":
		return documentv1.SortDirection_SORT_DESC
	default:
		return documentv1.SortDirection_SORT_DIRECTION_UNSPECIFIED
	}
}

func virtualFileSourceToProto(s string) documentv1.VirtualFileSource {
	switch s {
	case "chat":
		return documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_CHAT
	case "email":
		return documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_EMAIL
	case "task":
		return documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_TASK
	default:
		return documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_UNSPECIFIED
	}
}

// Suppress unused import warnings.
var _ = io.EOF
var _ = strconv.Itoa
