package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	wikiv1 "github.com/kmuhub/kmuhub/proto/wiki/v1"
)

// WikiRoutes handles HTTP routes for the Wiki backend service.
type WikiRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewWikiRoutes creates a new WikiRoutes with the given service registry and feature flags.
func NewWikiRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *WikiRoutes {
	return &WikiRoutes{registry: registry, flags: flags}
}

// ServiceName returns the backend service name.
func (wr *WikiRoutes) ServiceName() string { return "wiki" }

// getWikiClient lazily obtains a gRPC client for the Wiki service.
func (wr *WikiRoutes) getWikiClient() (wikiv1.WikiServiceClient, error) {
	conn, err := wr.registry.GetConnection("wiki")
	if err != nil {
		return nil, err
	}
	return wikiv1.NewWikiServiceClient(conn), nil
}

// RegisterRoutes mounts all Wiki HTTP routes behind the feature flag modules.wiki.
// Routes are only registered if the flag is enabled.
func (wr *WikiRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !wr.flags.IsEnabled("modules.wiki") {
		return
	}

	// Additive guards: every route keeps its legacy coarse key AND accepts the
	// matching capability-catalogue key. Permissions are baked into the access
	// token at login and never re-read per request, so swapping a key outright
	// would 403 every user holding a still-valid token. Extend, never swap.
	//
	// `wiki:categories:read` has no catalogue counterpart (the catalogue gates
	// category administration only), so that guard stays as it is.
	var (
		wikiRead      = middleware.RequirePermissionAny([2]string{"wiki:articles", "read"}, [2]string{"wiki:article", "read"})
		wikiCreate    = middleware.RequirePermissionAny([2]string{"wiki:articles", "write"}, [2]string{"wiki:article", "create"})
		wikiEdit      = middleware.RequirePermissionAny([2]string{"wiki:articles", "write"}, [2]string{"wiki:article", "edit"})
		wikiDelete    = middleware.RequirePermissionAny([2]string{"wiki:articles", "write"}, [2]string{"wiki:article", "delete"})
		wikiShare     = middleware.RequirePermissionAny([2]string{"wiki:articles", "write"}, [2]string{"wiki:share_token", "create"})
		wikiCatManage = middleware.RequirePermissionAny([2]string{"wiki:categories", "write"}, [2]string{"wiki:category", "manage"})
	)

	r.Route("/api/v1/wiki", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Articles
		r.Route("/articles", func(r chi.Router) {
			r.With(wikiRead).Get("/", wr.HandleListArticles)
			r.With(wikiCreate).Post("/", wr.HandleCreateArticle)

			r.Route("/{id}", func(r chi.Router) {
				r.With(wikiRead).Get("/", wr.HandleGetArticle)
				r.With(wikiEdit).Patch("/", wr.HandleUpdateArticle)
				r.With(wikiDelete).Delete("/", wr.HandleDeleteArticle)

				// Versions
				r.With(wikiRead).Get("/versions", wr.HandleListVersions)
				r.With(wikiEdit).Post("/versions/{versionId}/restore", wr.HandleRestoreVersion)

				// Attachments
				r.With(wikiRead).Get("/attachments", wr.HandleListAttachments)
				r.With(wikiEdit).Post("/attachments", wr.HandleUploadAttachment)
				r.With(wikiDelete).Delete("/attachments/{attachmentId}", wr.HandleDeleteAttachment)

				// Share tokens — revoking is part of managing them, and the
				// catalogue has no separate revoke key.
				r.With(wikiRead).Get("/share", wr.HandleListShareTokens)
				r.With(wikiShare).Post("/share", wr.HandleCreateShareToken)
			})
		})

		// Search
		r.With(wikiRead).Get("/search", wr.HandleSearchArticles)

		// Versions (standalone get by ID)
		r.Route("/versions", func(r chi.Router) {
			r.With(wikiRead).Get("/{id}", wr.HandleGetVersion)
		})

		// Categories
		r.Route("/categories", func(r chi.Router) {
			r.With(middleware.RequirePermission("wiki:categories", "read")).Get("/", wr.HandleListCategories)
			r.With(wikiCatManage).Post("/", wr.HandleCreateCategory)
			r.Route("/{id}", func(r chi.Router) {
				r.With(wikiCatManage).Delete("/", wr.HandleDeleteCategory)
				r.With(wikiCatManage).Patch("/", wr.HandleUpdateCategory)
			})
		})

		// Share tokens
		r.Route("/share", func(r chi.Router) {
			r.With(wikiShare).Delete("/{tokenId}", wr.HandleRevokeShareToken)
		})
	})
}

// ============================================================================
// Request types
// ============================================================================

type createArticleRequest struct {
	Title      string  `json:"title" validate:"required"`
	Slug       string  `json:"slug,omitempty"`
	Content    []byte  `json:"content,omitempty"`
	CategoryID *string `json:"category_id,omitempty" validate:"omitempty,uuid"`
	Published  bool    `json:"published"`
}

type updateArticleRequest struct {
	Title      *string `json:"title,omitempty" validate:"omitempty,min=1"`
	Slug       *string `json:"slug,omitempty"`
	Content    []byte  `json:"content,omitempty"`
	CategoryID *string `json:"category_id,omitempty"` // empty string = clear
	Published  *bool   `json:"published,omitempty"`
}

type uploadAttachmentRequest struct {
	FileRef string `json:"file_ref" validate:"required"`
	Mime    string `json:"mime"`
	Size    int64  `json:"size"`
}

type createCategoryRequest struct {
	Name     string  `json:"name" validate:"required"`
	ParentID *string `json:"parent_id,omitempty" validate:"omitempty,uuid"`
	Position int32   `json:"position"`
}

// ============================================================================
// Article Handlers
// ============================================================================

func (wr *WikiRoutes) HandleListArticles(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	q := r.URL.Query()

	grpcReq := &wikiv1.ListArticlesRequest{
		TenantId: tenantID.String(),
		Search:   q.Get("search"),
		Page:     int32(page),
		PageSize: int32(pageSize),
		SortBy:   q.Get("sort_by"),
	}

	if sd := q.Get("sort_desc"); sd == "true" || sd == "1" {
		grpcReq.SortDesc = true
	}
	if catID := q.Get("category_id"); catID != "" {
		grpcReq.CategoryId = &catID
	}
	if authorID := q.Get("author_id"); authorID != "" {
		grpcReq.AuthorId = &authorID
	}
	if pub := q.Get("published"); pub != "" {
		v := pub == "true" || pub == "1"
		grpcReq.Published = &v
	}

	resp, err := client.ListArticles(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleCreateArticle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createArticleRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &wikiv1.CreateArticleRequest{
		TenantId:   tenantID.String(),
		Title:      req.Title,
		Slug:       req.Slug,
		Content:    req.Content,
		AuthorId:   userID,
		Published:  req.Published,
		CategoryId: req.CategoryID,
	}

	resp, err := client.CreateArticle(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (wr *WikiRoutes) HandleGetArticle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetArticle(r.Context(), &wikiv1.GetArticleRequest{
		TenantId:  tenantID.String(),
		ArticleId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleUpdateArticle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[updateArticleRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &wikiv1.UpdateArticleRequest{
		TenantId:   tenantID.String(),
		ArticleId:  id,
		Title:      req.Title,
		Slug:       req.Slug,
		Content:    req.Content,
		Published:  req.Published,
		CategoryId: req.CategoryID,
		EditorId:   userID,
	}

	resp, err := client.UpdateArticle(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleDeleteArticle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteArticle(r.Context(), &wikiv1.DeleteArticleRequest{
		TenantId:  tenantID.String(),
		ArticleId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (wr *WikiRoutes) HandleSearchArticles(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		response.Error(w, http.StatusBadRequest, "q query parameter is required")
		return
	}

	limit := parseLimit(r, 20, 100)

	resp, err := client.SearchArticles(r.Context(), &wikiv1.SearchArticlesRequest{
		TenantId: tenantID.String(),
		Query:    q,
		Limit:    int32(limit),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Version Handlers
// ============================================================================

func (wr *WikiRoutes) HandleListVersions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListVersions(r.Context(), &wikiv1.ListVersionsRequest{
		ArticleId: id,
		TenantId:  tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleRestoreVersion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	articleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	versionID, ok := validateUUIDParam(w, r, "versionId")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.RestoreVersion(r.Context(), &wikiv1.RestoreVersionRequest{
		TenantId:  tenantID.String(),
		ArticleId: articleID,
		VersionId: versionID,
		EditorId:  userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Attachment Handlers
// ============================================================================

func (wr *WikiRoutes) HandleListAttachments(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListAttachments(r.Context(), &wikiv1.ListAttachmentsRequest{
		ArticleId: id,
		TenantId:  tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	articleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[uploadAttachmentRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &wikiv1.UploadAttachmentRequest{
		TenantId:   tenantID.String(),
		ArticleId:  articleID,
		FileRef:    req.FileRef,
		Mime:       req.Mime,
		Size:       req.Size,
		UploadedBy: &userID,
	}

	resp, err := client.UploadAttachment(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (wr *WikiRoutes) HandleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	attachmentID, ok := validateUUIDParam(w, r, "attachmentId")
	if !ok {
		return
	}

	_, err = client.DeleteAttachment(r.Context(), &wikiv1.DeleteAttachmentRequest{
		AttachmentId: attachmentID,
		TenantId:     tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Category Handlers
// ============================================================================

func (wr *WikiRoutes) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	resp, err := client.ListCategories(r.Context(), &wikiv1.ListCategoriesRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createCategoryRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &wikiv1.CreateCategoryRequest{
		TenantId: tenantID.String(),
		Name:     req.Name,
		Position: req.Position,
		ParentId: req.ParentID,
	}

	resp, err := client.CreateCategory(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (wr *WikiRoutes) HandleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteCategory(r.Context(), &wikiv1.DeleteCategoryRequest{
		TenantId:   tenantID.String(),
		CategoryId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type updateCategoryRequest struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=1"`
	ParentID *string `json:"parent_id,omitempty"`
	Position *int32  `json:"position,omitempty"`
}

func (wr *WikiRoutes) HandleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateCategoryRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &wikiv1.UpdateCategoryRequest{
		TenantId:   tenantID.String(),
		CategoryId: id,
		Name:       req.Name,
		ParentId:   req.ParentID,
		Position:   req.Position,
	}

	resp, err := client.UpdateCategory(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

type createShareTokenHTTPRequest struct {
	ExpiresAt   *string  `json:"expires_at,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

func (wr *WikiRoutes) HandleCreateShareToken(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	articleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[createShareTokenHTTPRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &wikiv1.CreateShareTokenRequest{
		TenantId:    tenantID.String(),
		ArticleId:   articleID,
		Permissions: req.Permissions,
		ExpiresAt:   req.ExpiresAt,
	}

	resp, err := client.CreateShareToken(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (wr *WikiRoutes) HandleListShareTokens(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	articleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListShareTokens(r.Context(), &wikiv1.ListShareTokensRequest{
		TenantId:  tenantID.String(),
		ArticleId: articleID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleRevokeShareToken(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	tokenID, ok := validateUUIDParam(w, r, "tokenId")
	if !ok {
		return
	}

	_, err = client.RevokeShareToken(r.Context(), &wikiv1.RevokeShareTokenRequest{
		TenantId: tenantID.String(),
		TokenId:  tokenID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (wr *WikiRoutes) HandleGetVersion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetVersion(r.Context(), &wikiv1.GetVersionRequest{
		VersionId: id,
		TenantId:  tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}
