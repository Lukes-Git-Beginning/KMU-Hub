package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

// DocumentRoutes handles HTTP routes for the Document backend service.
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
	// Folders
	r.Route("/api/v1/documents/folders", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("documents", "read")).Get("/", d.HandleListFolders)
		r.With(middleware.RequirePermission("documents", "read")).Get("/{id}", d.HandleGetFolder)
		r.With(middleware.RequirePermission("documents", "write")).Post("/", d.HandleCreateFolder)
		r.With(middleware.RequirePermission("documents", "write")).Put("/{id}", d.HandleUpdateFolder)
		r.With(middleware.RequirePermission("documents", "delete")).Delete("/{id}", d.HandleDeleteFolder)
		r.With(middleware.RequirePermission("documents", "read")).Get("/{id}/path", d.HandleGetFolderPath)
		r.With(middleware.RequirePermission("documents", "write")).Post("/initialize-user", d.HandleInitializeUserSpace)
		r.With(middleware.RequirePermission("documents", "write")).Post("/initialize-team", d.HandleInitializeTeamSpace)
	})

	// Files
	r.Route("/api/v1/documents/files", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("documents", "read")).Get("/", d.HandleListFiles)
		r.With(middleware.RequirePermission("documents", "read")).Get("/{id}", d.HandleGetFile)
		r.With(middleware.RequirePermission("documents", "write")).Put("/{id}", d.HandleUpdateFile)
		r.With(middleware.RequirePermission("documents", "delete")).Delete("/{id}", d.HandleDeleteFile)
		r.With(middleware.RequirePermission("documents", "write")).Post("/{id}/copy", d.HandleCopyFile)
		r.With(middleware.RequirePermission("documents", "write")).Post("/{id}/move", d.HandleMoveFile)
		r.With(middleware.RequirePermission("documents", "read")).Get("/{id}/download-url", d.HandleGetFileDownloadURL)
		// Versions
		r.With(middleware.RequirePermission("documents", "read")).Get("/{id}/versions", d.HandleListFileVersions)
		r.With(middleware.RequirePermission("documents", "write")).Post("/{id}/versions/revert", d.HandleRevertFileVersion)
		// Entity links
		r.With(middleware.RequirePermission("documents", "read")).Get("/{id}/links", d.HandleListFileEntityLinks)
		r.With(middleware.RequirePermission("documents", "write")).Post("/{id}/links", d.HandleLinkFileToEntity)
		r.With(middleware.RequirePermission("documents", "write")).Delete("/{id}/links", d.HandleUnlinkFileFromEntity)
	})

	// Shares
	r.Route("/api/v1/documents/shares", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("documents", "write")).Post("/", d.HandleShareEntity)
		r.With(middleware.RequirePermission("documents", "write")).Delete("/", d.HandleUnshareEntity)
		r.With(middleware.RequirePermission("documents", "read")).Get("/entity", d.HandleListShares)
		r.With(middleware.RequirePermission("documents", "read")).Get("/shared-with-me", d.HandleListSharedWithMe)
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
	Name      string `json:"name"`
	ParentID  string `json:"parent_id"`
	SpaceType string `json:"space_type"`
	SpaceID   string `json:"space_id"`
	Icon      string `json:"icon"`
}

func (d *DocumentRoutes) HandleCreateFolder(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	var req createFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.CreateFolder(r.Context(), &documentv1.CreateFolderRequest{
		Name:      req.Name,
		ParentId:  req.ParentID,
		SpaceType: spaceTypeToProto(req.SpaceType),
		SpaceId:   req.SpaceID,
		Icon:      req.Icon,
		CreatedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.Folder)
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

	response.JSON(w, http.StatusOK, resp.Folder)
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

	response.JSON(w, http.StatusOK, resp.Folders)
}

type updateFolderRequest struct {
	Name     *string `json:"name"`
	ParentID *string `json:"parent_id"`
	Icon     *string `json:"icon"`
}

func (d *DocumentRoutes) HandleUpdateFolder(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	var req updateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	response.JSON(w, http.StatusOK, resp.Folder)
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

	response.JSON(w, http.StatusOK, resp.Segments)
}

type initializeUserSpaceRequest struct {
	UserID string `json:"user_id"`
}

func (d *DocumentRoutes) HandleInitializeUserSpace(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	var req initializeUserSpaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID := req.UserID
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
	TeamID string `json:"team_id"`
}

func (d *DocumentRoutes) HandleInitializeTeamSpace(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	var req initializeTeamSpaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.InitializeTeamSpace(r.Context(), &documentv1.InitializeTeamSpaceRequest{
		TeamId: req.TeamID,
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

	response.JSON(w, http.StatusOK, resp.File)
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
		"files": resp.Files,
		"total": resp.Total,
	})
}

type updateFileRequest struct {
	Filename   *string `json:"filename"`
	FolderID   *string `json:"folder_id"`
	IsFavorite *bool   `json:"is_favorite"`
}

func (d *DocumentRoutes) HandleUpdateFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	var req updateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	response.JSON(w, http.StatusOK, resp.File)
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
	TargetFolderID string `json:"target_folder_id"`
}

func (d *DocumentRoutes) HandleCopyFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	var req copyFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	response.JSON(w, http.StatusCreated, resp.File)
}

type moveFileRequest struct {
	TargetFolderID string `json:"target_folder_id"`
}

func (d *DocumentRoutes) HandleMoveFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	var req moveFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	response.JSON(w, http.StatusOK, resp.File)
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

	response.JSON(w, http.StatusOK, resp.Versions)
}

type revertVersionRequest struct {
	VersionNumber int32 `json:"version_number"`
}

func (d *DocumentRoutes) HandleRevertFileVersion(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")

	var req revertVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	response.JSON(w, http.StatusOK, resp.File)
}

// ============================================================================
// Entity Link Handlers
// ============================================================================

type linkFileToEntityRequest struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

func (d *DocumentRoutes) HandleLinkFileToEntity(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	var req linkFileToEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	response.JSON(w, http.StatusCreated, resp.Link)
}

type unlinkFileFromEntityRequest struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

func (d *DocumentRoutes) HandleUnlinkFileFromEntity(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")

	var req unlinkFileFromEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	response.JSON(w, http.StatusOK, resp.Links)
}

// ============================================================================
// Share Handlers
// ============================================================================

type shareEntityRequest struct {
	EntityType       string `json:"entity_type"`
	EntityID         string `json:"entity_id"`
	SharedWithUserID string `json:"shared_with_user_id"`
	Permission       string `json:"permission"`
}

func (d *DocumentRoutes) HandleShareEntity(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req shareEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	response.JSON(w, http.StatusCreated, resp.Share)
}

type unshareEntityRequest struct {
	EntityType       string `json:"entity_type"`
	EntityID         string `json:"entity_id"`
	SharedWithUserID string `json:"shared_with_user_id"`
}

func (d *DocumentRoutes) HandleUnshareEntity(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	var req unshareEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	response.JSON(w, http.StatusOK, resp.Shares)
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
		"files":   resp.Files,
		"folders": resp.Folders,
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

	response.JSON(w, http.StatusOK, resp.Tags)
}

type createDocumentTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (d *DocumentRoutes) HandleCreateTag(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createDocumentTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	response.JSON(w, http.StatusCreated, resp.Tag)
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
	FileID string `json:"file_id"`
	TagID  string `json:"tag_id"`
}

func (d *DocumentRoutes) HandleTagFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	var req tagFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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
	FileID string `json:"file_id"`
	TagID  string `json:"tag_id"`
}

func (d *DocumentRoutes) HandleUntagFile(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	var req untagFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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
		"results": resp.Results,
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
		"files": resp.Files,
		"total": resp.Total,
	})
}

// ============================================================================
// WOPI Token Handler
// ============================================================================

type generateWOPITokenRequest struct {
	FileID string `json:"file_id"`
}

func (d *DocumentRoutes) HandleGenerateWOPIToken(w http.ResponseWriter, r *http.Request) {
	client, err := d.getDocumentClient()
	if err != nil {
		respondServiceUnavailable(w, d.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req generateWOPITokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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
		"access_token":    resp.AccessToken,
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

	response.JSON(w, http.StatusOK, resp.Actions)
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
