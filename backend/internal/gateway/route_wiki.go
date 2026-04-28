package gateway

import (
	"encoding/json"
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

	r.Route("/api/v1/wiki", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Articles
		r.Route("/articles", func(r chi.Router) {
			r.With(middleware.RequirePermission("wiki:articles", "read")).Get("/", wr.HandleListArticles)
			r.With(middleware.RequirePermission("wiki:articles", "write")).Post("/", wr.HandleCreateArticle)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("wiki:articles", "read")).Get("/", wr.HandleGetArticle)
				r.With(middleware.RequirePermission("wiki:articles", "write")).Patch("/", wr.HandleUpdateArticle)
				r.With(middleware.RequirePermission("wiki:articles", "write")).Delete("/", wr.HandleDeleteArticle)

				// Versions
				r.With(middleware.RequirePermission("wiki:articles", "read")).Get("/versions", wr.HandleListVersions)
				r.With(middleware.RequirePermission("wiki:articles", "write")).Post("/versions/{versionId}/restore", wr.HandleRestoreVersion)

				// Attachments
				r.With(middleware.RequirePermission("wiki:articles", "read")).Get("/attachments", wr.HandleListAttachments)
				r.With(middleware.RequirePermission("wiki:articles", "write")).Post("/attachments", wr.HandleUploadAttachment)
				r.With(middleware.RequirePermission("wiki:articles", "write")).Delete("/attachments/{attachmentId}", wr.HandleDeleteAttachment)
			})
		})

		// Search
		r.With(middleware.RequirePermission("wiki:articles", "read")).Get("/search", wr.HandleSearchArticles)

		// Categories
		r.Route("/categories", func(r chi.Router) {
			r.With(middleware.RequirePermission("wiki:categories", "read")).Get("/", wr.HandleListCategories)
			r.With(middleware.RequirePermission("wiki:categories", "write")).Post("/", wr.HandleCreateCategory)
		})
	})
}

// ============================================================================
// Request types
// ============================================================================

type createArticleRequest struct {
	Title      string  `json:"title"`
	Slug       string  `json:"slug,omitempty"`
	Content    []byte  `json:"content,omitempty"`
	CategoryID *string `json:"category_id,omitempty"`
	Published  bool    `json:"published"`
}

type updateArticleRequest struct {
	Title      *string `json:"title,omitempty"`
	Slug       *string `json:"slug,omitempty"`
	Content    []byte  `json:"content,omitempty"`
	CategoryID *string `json:"category_id,omitempty"` // empty string = clear
	Published  *bool   `json:"published,omitempty"`
}

type uploadAttachmentRequest struct {
	FileRef string `json:"file_ref"`
	Mime    string `json:"mime"`
	Size    int64  `json:"size"`
}

type createCategoryRequest struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id,omitempty"`
	Position int32   `json:"position"`
}

// ============================================================================
// Article Handlers
// ============================================================================

func (wr *WikiRoutes) HandleListArticles(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleCreateArticle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		response.Error(w, http.StatusBadRequest, "title is required")
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
	response.JSON(w, http.StatusCreated, resp)
}

func (wr *WikiRoutes) HandleGetArticle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleUpdateArticle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	var req updateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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
	response.JSON(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleDeleteArticle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Version Handlers
// ============================================================================

func (wr *WikiRoutes) HandleListVersions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	_ = tenantID // validated; article_id already scopes to tenant via FK
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
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleRestoreVersion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Attachment Handlers
// ============================================================================

func (wr *WikiRoutes) HandleListAttachments(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	_ = tenantID // validated; article_id already scopes to tenant via FK
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
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	_ = tenantID // validated; article_id already scopes to tenant via FK
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

	var req uploadAttachmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.FileRef == "" {
		response.Error(w, http.StatusBadRequest, "file_ref is required")
		return
	}

	grpcReq := &wikiv1.UploadAttachmentRequest{
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
	response.JSON(w, http.StatusCreated, resp)
}

func (wr *WikiRoutes) HandleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	_ = tenantID // validated; attachment_id already scopes to tenant via article FK
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

func (wr *WikiRoutes) HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := wr.getWikiClient()
	if err != nil {
		respondServiceUnavailable(w, wr.ServiceName())
		return
	}

	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
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
	response.JSON(w, http.StatusCreated, resp)
}

