package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/wiki"
	wikiv1 "github.com/kmuhub/kmuhub/proto/wiki/v1"
)

// WikiGRPCServer implements the WikiService gRPC server.
type WikiGRPCServer struct {
	wikiv1.UnimplementedWikiServiceServer
	svc *wiki.Service
}

// NewWikiGRPCServer creates a new Wiki gRPC server.
func NewWikiGRPCServer(svc *wiki.Service) *WikiGRPCServer {
	return &WikiGRPCServer{svc: svc}
}

// ============================================================================
// Article RPCs
// ============================================================================

func (s *WikiGRPCServer) CreateArticle(ctx context.Context, req *wikiv1.CreateArticleRequest) (*wikiv1.ArticleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	authorID, err := uuid.Parse(req.GetAuthorId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid author_id: %v", err)
	}

	input := wiki.CreateArticleInput{
		TenantID:  tenantID,
		Title:     req.GetTitle(),
		Slug:      req.GetSlug(),
		Content:   req.GetContent(),
		AuthorID:  authorID,
		Published: req.GetPublished(),
	}

	if req.CategoryId != nil {
		id, err := uuid.Parse(*req.CategoryId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid category_id: %v", err)
		}
		input.CategoryID = &id
	}

	article, err := s.svc.CreateArticle(ctx, input)
	if err != nil {
		return nil, mapWikiError(err)
	}
	return &wikiv1.ArticleResponse{Article: articleToProto(article)}, nil
}

func (s *WikiGRPCServer) GetArticle(ctx context.Context, req *wikiv1.GetArticleRequest) (*wikiv1.ArticleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	articleID, err := uuid.Parse(req.GetArticleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid article_id: %v", err)
	}

	article, err := s.svc.GetArticle(ctx, tenantID, articleID)
	if err != nil {
		return nil, mapWikiError(err)
	}
	return &wikiv1.ArticleResponse{Article: articleToProto(article)}, nil
}

func (s *WikiGRPCServer) UpdateArticle(ctx context.Context, req *wikiv1.UpdateArticleRequest) (*wikiv1.ArticleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	articleID, err := uuid.Parse(req.GetArticleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid article_id: %v", err)
	}
	editorID, err := uuid.Parse(req.GetEditorId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid editor_id: %v", err)
	}

	input := wiki.UpdateArticleInput{
		TenantID:  tenantID,
		ArticleID: articleID,
		Content:   req.GetContent(),
		EditorID:  editorID,
	}

	if req.Title != nil {
		input.Title = req.Title
	}
	if req.Slug != nil {
		input.Slug = req.Slug
	}
	if req.Published != nil {
		input.Published = req.Published
	}

	if req.CategoryId != nil {
		if *req.CategoryId == "" {
			empty := uuid.Nil
			input.CategoryID = &empty
		} else {
			id, parseErr := uuid.Parse(*req.CategoryId)
			if parseErr != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid category_id: %v", parseErr)
			}
			input.CategoryID = &id
		}
	}

	article, err := s.svc.UpdateArticle(ctx, input)
	if err != nil {
		return nil, mapWikiError(err)
	}
	return &wikiv1.ArticleResponse{Article: articleToProto(article)}, nil
}

func (s *WikiGRPCServer) DeleteArticle(ctx context.Context, req *wikiv1.DeleteArticleRequest) (*wikiv1.DeleteArticleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	articleID, err := uuid.Parse(req.GetArticleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid article_id: %v", err)
	}

	if err := s.svc.DeleteArticle(ctx, tenantID, articleID); err != nil {
		return nil, mapWikiError(err)
	}
	return &wikiv1.DeleteArticleResponse{}, nil
}

func (s *WikiGRPCServer) ListArticles(ctx context.Context, req *wikiv1.ListArticlesRequest) (*wikiv1.ListArticlesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := wiki.ListArticlesInput{
		TenantID: tenantID,
		Search:   req.GetSearch(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
		SortBy:   req.GetSortBy(),
		SortDesc: req.GetSortDesc(),
	}

	if req.CategoryId != nil {
		id, err := uuid.Parse(*req.CategoryId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid category_id: %v", err)
		}
		input.CategoryID = &id
	}
	if req.AuthorId != nil {
		id, err := uuid.Parse(*req.AuthorId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid author_id: %v", err)
		}
		input.AuthorID = &id
	}
	if req.Published != nil {
		input.Published = req.Published
	}

	articles, total, err := s.svc.ListArticles(ctx, input)
	if err != nil {
		return nil, mapWikiError(err)
	}

	protoArticles := make([]*wikiv1.Article, len(articles))
	for i, a := range articles {
		protoArticles[i] = articleToProto(a)
	}
	return &wikiv1.ListArticlesResponse{
		Articles: protoArticles,
		Total:    int32(total),
	}, nil
}

func (s *WikiGRPCServer) SearchArticles(ctx context.Context, req *wikiv1.SearchArticlesRequest) (*wikiv1.SearchArticlesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	articles, err := s.svc.SearchArticles(ctx, tenantID, req.GetQuery(), int(req.GetLimit()))
	if err != nil {
		return nil, mapWikiError(err)
	}

	protoArticles := make([]*wikiv1.Article, len(articles))
	for i, a := range articles {
		protoArticles[i] = articleToProto(a)
	}
	return &wikiv1.SearchArticlesResponse{Articles: protoArticles}, nil
}

// ============================================================================
// Version RPCs
// ============================================================================

func (s *WikiGRPCServer) ListVersions(ctx context.Context, req *wikiv1.ListVersionsRequest) (*wikiv1.ListVersionsResponse, error) {
	articleID, err := uuid.Parse(req.GetArticleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid article_id: %v", err)
	}

	versions, err := s.svc.ListVersions(ctx, articleID)
	if err != nil {
		return nil, mapWikiError(err)
	}

	protoVersions := make([]*wikiv1.Version, len(versions))
	for i, v := range versions {
		protoVersions[i] = versionToProto(v)
	}
	return &wikiv1.ListVersionsResponse{Versions: protoVersions}, nil
}

func (s *WikiGRPCServer) GetVersion(ctx context.Context, req *wikiv1.GetVersionRequest) (*wikiv1.VersionResponse, error) {
	versionID, err := uuid.Parse(req.GetVersionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version_id: %v", err)
	}

	v, err := s.svc.GetVersion(ctx, versionID)
	if err != nil {
		return nil, mapWikiError(err)
	}
	return &wikiv1.VersionResponse{Version: versionToProto(v)}, nil
}

func (s *WikiGRPCServer) RestoreVersion(ctx context.Context, req *wikiv1.RestoreVersionRequest) (*wikiv1.ArticleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	articleID, err := uuid.Parse(req.GetArticleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid article_id: %v", err)
	}
	versionID, err := uuid.Parse(req.GetVersionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version_id: %v", err)
	}
	editorID, err := uuid.Parse(req.GetEditorId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid editor_id: %v", err)
	}

	article, err := s.svc.RestoreVersion(ctx, tenantID, articleID, versionID, editorID)
	if err != nil {
		return nil, mapWikiError(err)
	}
	return &wikiv1.ArticleResponse{Article: articleToProto(article)}, nil
}

// ============================================================================
// Attachment RPCs
// ============================================================================

func (s *WikiGRPCServer) UploadAttachment(ctx context.Context, req *wikiv1.UploadAttachmentRequest) (*wikiv1.AttachmentResponse, error) {
	articleID, err := uuid.Parse(req.GetArticleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid article_id: %v", err)
	}

	input := wiki.UploadInput{
		ArticleID: articleID,
		FileRef:   req.GetFileRef(),
		Mime:      req.GetMime(),
		Size:      req.GetSize(),
	}

	if req.UploadedBy != nil {
		id, err := uuid.Parse(*req.UploadedBy)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid uploaded_by: %v", err)
		}
		input.UploadedBy = &id
	}

	attachment, err := s.svc.UploadAttachment(ctx, input)
	if err != nil {
		return nil, mapWikiError(err)
	}
	return &wikiv1.AttachmentResponse{Attachment: attachmentToProto(attachment)}, nil
}

func (s *WikiGRPCServer) ListAttachments(ctx context.Context, req *wikiv1.ListAttachmentsRequest) (*wikiv1.ListAttachmentsResponse, error) {
	articleID, err := uuid.Parse(req.GetArticleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid article_id: %v", err)
	}

	attachments, err := s.svc.ListAttachments(ctx, articleID)
	if err != nil {
		return nil, mapWikiError(err)
	}

	protoAttachments := make([]*wikiv1.Attachment, len(attachments))
	for i, a := range attachments {
		protoAttachments[i] = attachmentToProto(a)
	}
	return &wikiv1.ListAttachmentsResponse{Attachments: protoAttachments}, nil
}

func (s *WikiGRPCServer) DeleteAttachment(ctx context.Context, req *wikiv1.DeleteAttachmentRequest) (*wikiv1.DeleteAttachmentResponse, error) {
	attachmentID, err := uuid.Parse(req.GetAttachmentId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid attachment_id: %v", err)
	}

	if err := s.svc.DeleteAttachment(ctx, attachmentID); err != nil {
		return nil, mapWikiError(err)
	}
	return &wikiv1.DeleteAttachmentResponse{}, nil
}

// ============================================================================
// Category RPCs
// ============================================================================

func (s *WikiGRPCServer) ListCategories(ctx context.Context, req *wikiv1.ListCategoriesRequest) (*wikiv1.ListCategoriesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	categories, err := s.svc.ListCategories(ctx, tenantID)
	if err != nil {
		return nil, mapWikiError(err)
	}

	protoCategories := make([]*wikiv1.Category, len(categories))
	for i, c := range categories {
		protoCategories[i] = wikiCategoryToProto(c)
	}
	return &wikiv1.ListCategoriesResponse{Categories: protoCategories}, nil
}

func (s *WikiGRPCServer) CreateCategory(ctx context.Context, req *wikiv1.CreateCategoryRequest) (*wikiv1.CategoryResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := wiki.CreateCategoryInput{
		TenantID: tenantID,
		Name:     req.GetName(),
		Position: int(req.GetPosition()),
	}

	if req.ParentId != nil {
		id, err := uuid.Parse(*req.ParentId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parent_id: %v", err)
		}
		input.ParentID = &id
	}

	category, err := s.svc.CreateCategory(ctx, input)
	if err != nil {
		return nil, mapWikiError(err)
	}
	return &wikiv1.CategoryResponse{Category: wikiCategoryToProto(category)}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func articleToProto(a *wiki.Article) *wikiv1.Article {
	if a == nil {
		return nil
	}
	proto := &wikiv1.Article{
		Id:        a.ID.String(),
		TenantId:  a.TenantID.String(),
		Title:     a.Title,
		Slug:      a.Slug,
		Content:   a.Content,
		AuthorId:  a.AuthorID.String(),
		Published: a.Published,
		CreatedAt: timestamppb.New(a.CreatedAt),
		UpdatedAt: timestamppb.New(a.UpdatedAt),
	}
	if a.CategoryID != nil {
		s := a.CategoryID.String()
		proto.CategoryId = &s
	}
	return proto
}

func versionToProto(v *wiki.Version) *wikiv1.Version {
	if v == nil {
		return nil
	}
	proto := &wikiv1.Version{
		Id:            v.ID.String(),
		ArticleId:     v.ArticleID.String(),
		VersionNumber: int32(v.VersionNumber),
		Content:       v.Content,
		ChangedAt:     timestamppb.New(v.ChangedAt),
	}
	if v.ChangedBy != nil {
		s := v.ChangedBy.String()
		proto.ChangedBy = &s
	}
	return proto
}

func attachmentToProto(a *wiki.Attachment) *wikiv1.Attachment {
	if a == nil {
		return nil
	}
	proto := &wikiv1.Attachment{
		Id:        a.ID.String(),
		ArticleId: a.ArticleID.String(),
		FileRef:   a.FileRef,
		Mime:      a.Mime,
		Size:      a.Size,
		CreatedAt: timestamppb.New(a.CreatedAt),
	}
	if a.UploadedBy != nil {
		s := a.UploadedBy.String()
		proto.UploadedBy = &s
	}
	return proto
}

func wikiCategoryToProto(c *wiki.Category) *wikiv1.Category {
	if c == nil {
		return nil
	}
	proto := &wikiv1.Category{
		Id:        c.ID.String(),
		TenantId:  c.TenantID.String(),
		Name:      c.Name,
		Position:  int32(c.Position),
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
	if c.ParentID != nil {
		s := c.ParentID.String()
		proto.ParentId = &s
	}
	return proto
}

// ============================================================================
// Error mapping
// ============================================================================

func mapWikiError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, wiki.ErrArticleNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, wiki.ErrVersionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, wiki.ErrCategoryNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, wiki.ErrSlugTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, wiki.ErrInvalidContent):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled wiki service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
