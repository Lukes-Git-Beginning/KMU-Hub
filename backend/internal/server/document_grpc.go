package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/document/file"
	"github.com/kmuhub/kmuhub/internal/document/folder"
	"github.com/kmuhub/kmuhub/internal/document/search"
	"github.com/kmuhub/kmuhub/internal/document/share"
	"github.com/kmuhub/kmuhub/internal/document/tag"
	"github.com/kmuhub/kmuhub/internal/document/virtual"
	"github.com/kmuhub/kmuhub/internal/document/wopi"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

// DocumentGRPCServer implements the Document gRPC service.
type DocumentGRPCServer struct {
	documentv1.UnimplementedDocumentServiceServer
	folderService  *folder.Service
	fileService    *file.Service
	shareService   *share.Service
	tagService     *tag.Service
	searchService  *search.Service
	virtualService *virtual.Service
	tokenService   *wopi.TokenService
	fileRepo       search.FileRepo // for async text extraction bridging
}

// NewDocumentGRPCServer creates a new Document gRPC server.
func NewDocumentGRPCServer(
	folderService *folder.Service,
	fileService *file.Service,
	shareService *share.Service,
	tagService *tag.Service,
	searchService *search.Service,
	virtualService *virtual.Service,
	tokenService *wopi.TokenService,
	fileRepo search.FileRepo,
) *DocumentGRPCServer {
	return &DocumentGRPCServer{
		folderService:  folderService,
		fileService:    fileService,
		shareService:   shareService,
		tagService:     tagService,
		searchService:  searchService,
		virtualService: virtualService,
		tokenService:   tokenService,
		fileRepo:       fileRepo,
	}
}

// ============================================================================
// Folder Operations
// ============================================================================

func (s *DocumentGRPCServer) CreateFolder(ctx context.Context, req *documentv1.CreateFolderRequest) (*documentv1.CreateFolderResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	input := folder.CreateInput{
		TenantID:  tenantID,
		Name:      req.Name,
		SpaceType: spaceTypeFromProto(req.SpaceType),
		SpaceID:   parseUUIDOrNil(req.SpaceId),
		Icon:      req.Icon,
		CreatedBy: createdBy,
	}

	if req.ParentId != "" {
		pid, pidErr := uuid.Parse(req.ParentId)
		if pidErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_id")
		}
		input.ParentID = &pid
	}

	f, err := s.folderService.Create(ctx, input)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.CreateFolderResponse{Folder: toProtoFolder(f)}, nil
}

func (s *DocumentGRPCServer) GetFolder(ctx context.Context, req *documentv1.GetFolderRequest) (*documentv1.GetFolderResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	f, err := s.folderService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.GetFolderResponse{Folder: toProtoFolder(f)}, nil
}

func (s *DocumentGRPCServer) ListFolders(ctx context.Context, req *documentv1.ListFoldersRequest) (*documentv1.ListFoldersResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	filter := folder.ListFilter{TenantID: tenantID}

	if req.ParentId != "" {
		pid, err := uuid.Parse(req.ParentId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_id")
		}
		filter.ParentID = &pid
	}

	if req.SpaceType != documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_UNSPECIFIED {
		st := spaceTypeFromProto(req.SpaceType)
		filter.SpaceType = &st
	}

	if req.SpaceId != "" {
		sid, err := uuid.Parse(req.SpaceId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid space_id")
		}
		filter.SpaceID = &sid
	}

	folders, _, err := s.folderService.List(ctx, filter)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoFolders := make([]*documentv1.DocumentFolder, 0, len(folders))
	for _, f := range folders {
		protoFolders = append(protoFolders, toProtoFolder(f))
	}

	return &documentv1.ListFoldersResponse{Folders: protoFolders}, nil
}

func (s *DocumentGRPCServer) UpdateFolder(ctx context.Context, req *documentv1.UpdateFolderRequest) (*documentv1.UpdateFolderResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	input := folder.UpdateInput{}
	if req.Name != nil {
		input.Name = req.Name
	}
	if req.ParentId != nil {
		pid, pidErr := uuid.Parse(*req.ParentId)
		if pidErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_id")
		}
		input.ParentID = &pid
	}
	if req.Icon != nil {
		input.Icon = req.Icon
	}

	if err := s.folderService.Update(ctx, tenantID, id, input); err != nil {
		return nil, mapDocumentError(err)
	}

	f, err := s.folderService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.UpdateFolderResponse{Folder: toProtoFolder(f)}, nil
}

func (s *DocumentGRPCServer) DeleteFolder(ctx context.Context, req *documentv1.DeleteFolderRequest) (*documentv1.DeleteFolderResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	if err := s.folderService.Delete(ctx, tenantID, id); err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.DeleteFolderResponse{}, nil
}

func (s *DocumentGRPCServer) GetFolderPath(ctx context.Context, req *documentv1.GetFolderPathRequest) (*documentv1.GetFolderPathResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	segments, err := s.folderService.GetPath(ctx, id)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoSegments := make([]*documentv1.FolderPathSegment, 0, len(segments))
	for _, seg := range segments {
		protoSegments = append(protoSegments, &documentv1.FolderPathSegment{
			Id:   seg.ID.String(),
			Name: seg.Name,
		})
	}

	return &documentv1.GetFolderPathResponse{Segments: protoSegments}, nil
}

func (s *DocumentGRPCServer) InitializeUserSpace(ctx context.Context, req *documentv1.InitializeUserSpaceRequest) (*documentv1.InitializeUserSpaceResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.folderService.InitializeUserSpace(ctx, tenantID, userID); err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.InitializeUserSpaceResponse{}, nil
}

func (s *DocumentGRPCServer) InitializeTeamSpace(ctx context.Context, req *documentv1.InitializeTeamSpaceRequest) (*documentv1.InitializeTeamSpaceResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	teamID, err := uuid.Parse(req.TeamId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid team_id")
	}

	if err := s.folderService.InitializeTeamSpace(ctx, tenantID, teamID, "Team"); err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.InitializeTeamSpaceResponse{}, nil
}

// ============================================================================
// File Operations
// ============================================================================

func (s *DocumentGRPCServer) GetFile(ctx context.Context, req *documentv1.GetFileRequest) (*documentv1.GetFileResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file id")
	}

	f, err := s.fileService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.GetFileResponse{File: toProtoFile(f)}, nil
}

// RegisterUploadedFile records metadata for a file already uploaded to object
// storage via a presigned PUT URL. The tenant is taken from the gRPC context.
func (s *DocumentGRPCServer) RegisterUploadedFile(ctx context.Context, req *documentv1.RegisterUploadedFileRequest) (*documentv1.RegisterUploadedFileResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	folderID, err := uuid.Parse(req.FolderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner id")
	}

	f, err := s.fileService.Register(ctx, file.RegisterInput{
		TenantID:   tenantID,
		FolderID:   folderID,
		Filename:   req.Filename,
		MimeType:   req.MimeType,
		FileSize:   req.FileSize,
		StorageKey: req.StorageKey,
		OwnerID:    ownerID,
	})
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.RegisterUploadedFileResponse{File: toProtoFile(f)}, nil
}

// UploadFile stores file bytes the gateway received over multipart directly
// (unlike RegisterUploadedFile, whose bytes already sit in MinIO). folder_id
// is resolved through the tenant-scoped folder service BEFORE anything is
// written — folderService.GetByID returns folder.ErrFolderNotFound for a
// folder outside the caller's tenant, which is exactly the check that keeps
// this endpoint from writing into another tenant's folder. The resolved
// folder's own space also seeds the storage key, matching how fileService.Upload
// is used everywhere else in this package.
func (s *DocumentGRPCServer) UploadFile(ctx context.Context, req *documentv1.UploadFileRequest) (*documentv1.UploadFileResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	folderID, err := uuid.Parse(req.FolderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}
	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner id")
	}

	targetFolder, err := s.folderService.GetByID(ctx, tenantID, folderID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	f, err := s.fileService.Upload(ctx, file.UploadInput{
		TenantID:  tenantID,
		FolderID:  folderID,
		Filename:  req.Filename,
		MimeType:  req.MimeType,
		FileSize:  req.FileSize,
		Reader:    bytes.NewReader(req.Content),
		SpaceType: targetFolder.SpaceType,
		SpaceID:   targetFolder.SpaceID,
		OwnerID:   ownerID,
	})
	if err != nil {
		return nil, mapDocumentError(err)
	}

	// tag_id is optional. TagFile itself verifies the tag belongs to the
	// caller's tenant (tag.PostgresRepository.TagFile) — a tag from another
	// tenant is rejected the same way a foreign folder would be, not silently
	// attached.
	if req.TagId != "" {
		tagID, err := uuid.Parse(req.TagId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tag id")
		}
		if err := s.tagService.TagFile(ctx, tenantID, f.ID, tagID); err != nil {
			return nil, mapDocumentError(err)
		}
	}

	return &documentv1.UploadFileResponse{File: toProtoFile(f)}, nil
}

func (s *DocumentGRPCServer) ListFiles(ctx context.Context, req *documentv1.ListFilesRequest) (*documentv1.ListFilesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	filter := file.ListFilter{
		TenantID: tenantID,
		Limit:    int(req.PerPage),
		Offset:   int((req.Page - 1) * req.PerPage),
	}

	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	if req.FolderId != "" {
		fid, err := uuid.Parse(req.FolderId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid folder_id")
		}
		filter.FolderID = &fid
	}

	if req.OwnerId != "" {
		oid, err := uuid.Parse(req.OwnerId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
		}
		filter.OwnerID = &oid
	}

	if req.FilterFavorites {
		filter.IsFavorite = &req.IsFavorite
	}

	filter.SortField = sortFieldFromProto(req.SortField)
	filter.SortDir = sortDirFromProto(req.SortDirection)

	files, total, err := s.fileService.List(ctx, filter)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoFiles := make([]*documentv1.DocumentFile, 0, len(files))
	for _, f := range files {
		protoFiles = append(protoFiles, toProtoFile(f))
	}

	return &documentv1.ListFilesResponse{Files: protoFiles, Total: int32(total)}, nil
}

func (s *DocumentGRPCServer) UpdateFile(ctx context.Context, req *documentv1.UpdateFileRequest) (*documentv1.UpdateFileResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file id")
	}

	input := file.UpdateInput{}
	if req.Filename != nil {
		input.Filename = req.Filename
	}
	if req.FolderId != nil {
		fid, fidErr := uuid.Parse(*req.FolderId)
		if fidErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid folder_id")
		}
		input.FolderID = &fid
	}
	if req.IsFavorite != nil {
		input.IsFavorite = req.IsFavorite
	}

	actorID, actorErr := actorIDFromContext(ctx)
	if actorErr != nil {
		return nil, actorErr
	}

	if err := s.fileService.Update(ctx, id, tenantID, actorID, input); err != nil {
		return nil, mapDocumentError(err)
	}

	f, err := s.fileService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.UpdateFileResponse{File: toProtoFile(f)}, nil
}

func (s *DocumentGRPCServer) DeleteFile(ctx context.Context, req *documentv1.DeleteFileRequest) (*documentv1.DeleteFileResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file id")
	}

	if err := s.fileService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.DeleteFileResponse{}, nil
}

func (s *DocumentGRPCServer) CopyFile(ctx context.Context, req *documentv1.CopyFileRequest) (*documentv1.CopyFileResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file id")
	}

	targetFolderID, err := uuid.Parse(req.TargetFolderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target_folder_id")
	}

	actorID, actorErr := actorIDFromContext(ctx)
	if actorErr != nil {
		return nil, actorErr
	}

	f, err := s.fileService.Copy(ctx, id, targetFolderID, actorID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.CopyFileResponse{File: toProtoFile(f)}, nil
}

func (s *DocumentGRPCServer) MoveFile(ctx context.Context, req *documentv1.MoveFileRequest) (*documentv1.MoveFileResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file id")
	}

	targetFolderID, err := uuid.Parse(req.TargetFolderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target_folder_id")
	}

	actorID, actorErr := actorIDFromContext(ctx)
	if actorErr != nil {
		return nil, actorErr
	}

	if err := s.fileService.Move(ctx, id, targetFolderID, actorID, tenantID); err != nil {
		return nil, mapDocumentError(err)
	}

	f, err := s.fileService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.MoveFileResponse{File: toProtoFile(f)}, nil
}

func (s *DocumentGRPCServer) GetFileDownloadURL(ctx context.Context, req *documentv1.GetFileDownloadURLRequest) (*documentv1.GetFileDownloadURLResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file id")
	}

	url, err := s.fileService.GetDownloadURL(ctx, id, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	f, err := s.fileService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	if actorID, actorErr := actorIDFromContext(ctx); actorErr == nil {
		s.fileService.LogDownload(ctx, id, tenantID, actorID)
	}

	return &documentv1.GetFileDownloadURLResponse{
		DownloadUrl: url,
		Filename:    f.Filename,
		ContentType: f.MimeType,
		FileSize:    f.FileSize,
	}, nil
}

func (s *DocumentGRPCServer) GetFileVersionDownloadURL(ctx context.Context, req *documentv1.GetFileVersionDownloadURLRequest) (*documentv1.GetFileVersionDownloadURLResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file id")
	}
	versionID, err := uuid.Parse(req.VersionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid version id")
	}

	url, filename, contentType, fileSize, err := s.fileService.GetVersionDownloadURL(ctx, fileID, versionID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	if actorID, actorErr := actorIDFromContext(ctx); actorErr == nil {
		s.fileService.LogDownload(ctx, fileID, tenantID, actorID)
	}

	return &documentv1.GetFileVersionDownloadURLResponse{
		DownloadUrl: url,
		Filename:    filename,
		ContentType: contentType,
		FileSize:    fileSize,
	}, nil
}

func (s *DocumentGRPCServer) CreateFileVersion(ctx context.Context, req *documentv1.CreateFileVersionRequest) (*documentv1.CreateFileVersionResponse, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	// Note: For gRPC version creation, we don't have a reader. This RPC is for
	// metadata-only version recording when the storage_key is already known.
	// WOPI PutFile uses the file service directly (not via gRPC).
	v := &documentv1.DocumentFileVersion{
		FileId:       req.FileId,
		StorageKey:   req.StorageKey,
		FileSize:     req.FileSize,
		CreatedBy:    createdBy.String(),
		VersionLabel: req.VersionLabel,
	}
	_ = fileID // Used for validation

	return &documentv1.CreateFileVersionResponse{Version: v}, nil
}

func (s *DocumentGRPCServer) ListFileVersions(ctx context.Context, req *documentv1.ListFileVersionsRequest) (*documentv1.ListFileVersionsResponse, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	versions, err := s.fileService.ListVersions(ctx, fileID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoVersions := make([]*documentv1.DocumentFileVersion, 0, len(versions))
	for _, v := range versions {
		protoVersions = append(protoVersions, toProtoFileVersion(v))
	}

	return &documentv1.ListFileVersionsResponse{Versions: protoVersions}, nil
}

func (s *DocumentGRPCServer) RevertFileVersion(ctx context.Context, req *documentv1.RevertFileVersionRequest) (*documentv1.RevertFileVersionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	actorID, actorErr := actorIDFromContext(ctx)
	if actorErr != nil {
		return nil, actorErr
	}

	_, err = s.fileService.RevertVersion(ctx, fileID, int(req.VersionNumber), actorID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	f, err := s.fileService.GetByID(ctx, fileID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.RevertFileVersionResponse{File: toProtoFile(f)}, nil
}

func (s *DocumentGRPCServer) ListFileActivity(ctx context.Context, req *documentv1.ListFileActivityRequest) (*documentv1.ListFileActivityResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	activities, err := s.fileService.ListActivity(ctx, fileID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoActivities := make([]*documentv1.DocumentFileActivity, 0, len(activities))
	for _, a := range activities {
		protoActivities = append(protoActivities, toProtoFileActivity(a))
	}

	return &documentv1.ListFileActivityResponse{Activities: protoActivities}, nil
}

// ============================================================================
// Comment Operations
// ============================================================================

func (s *DocumentGRPCServer) ListFileComments(ctx context.Context, req *documentv1.ListFileCommentsRequest) (*documentv1.ListFileCommentsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	comments, err := s.fileService.ListComments(ctx, fileID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoComments := make([]*documentv1.DocumentFileComment, 0, len(comments))
	for _, c := range comments {
		protoComments = append(protoComments, toProtoFileComment(c))
	}

	return &documentv1.ListFileCommentsResponse{Comments: protoComments}, nil
}

func (s *DocumentGRPCServer) CreateFileComment(ctx context.Context, req *documentv1.CreateFileCommentRequest) (*documentv1.CreateFileCommentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	actorID, actorErr := actorIDFromContext(ctx)
	if actorErr != nil {
		return nil, actorErr
	}

	c, err := s.fileService.CreateComment(ctx, tenantID, fileID, actorID, req.Content)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.CreateFileCommentResponse{Comment: toProtoFileComment(c)}, nil
}

func (s *DocumentGRPCServer) UpdateFileComment(ctx context.Context, req *documentv1.UpdateFileCommentRequest) (*documentv1.UpdateFileCommentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	commentID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid comment id")
	}

	actorID, actorErr := actorIDFromContext(ctx)
	if actorErr != nil {
		return nil, actorErr
	}

	c, err := s.fileService.UpdateComment(ctx, tenantID, commentID, actorID, req.Content)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.UpdateFileCommentResponse{Comment: toProtoFileComment(c)}, nil
}

func (s *DocumentGRPCServer) DeleteFileComment(ctx context.Context, req *documentv1.DeleteFileCommentRequest) (*documentv1.DeleteFileCommentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	commentID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid comment id")
	}

	actorID, actorErr := actorIDFromContext(ctx)
	if actorErr != nil {
		return nil, actorErr
	}

	if err := s.fileService.DeleteComment(ctx, tenantID, commentID, actorID, req.IsAdmin); err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.DeleteFileCommentResponse{}, nil
}

// ============================================================================
// Share Link Operations (external, unauthenticated read/download links)
// ============================================================================

func (s *DocumentGRPCServer) CreateShareLink(ctx context.Context, req *documentv1.CreateShareLinkRequest) (*documentv1.CreateShareLinkResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	in := file.CreateShareLinkInput{
		TenantID:      tenantID,
		FileID:        fileID,
		ExpiresInDays: req.ExpiresInDays,
	}
	if req.Password != nil {
		in.Password = *req.Password
	}
	if req.CreatedBy != nil {
		if createdBy, parseErr := uuid.Parse(*req.CreatedBy); parseErr == nil {
			in.CreatedBy = &createdBy
		}
	}

	link, err := s.fileService.CreateShareLink(ctx, in)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.CreateShareLinkResponse{ShareLink: toProtoShareLink(link)}, nil
}

func (s *DocumentGRPCServer) ListShareLinks(ctx context.Context, req *documentv1.ListShareLinksRequest) (*documentv1.ListShareLinksResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	links, err := s.fileService.ListShareLinks(ctx, fileID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoLinks := make([]*documentv1.ShareLink, 0, len(links))
	for _, l := range links {
		protoLinks = append(protoLinks, toProtoShareLink(l))
	}

	return &documentv1.ListShareLinksResponse{ShareLinks: protoLinks}, nil
}

func (s *DocumentGRPCServer) RevokeShareLink(ctx context.Context, req *documentv1.RevokeShareLinkRequest) (*documentv1.RevokeShareLinkResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid share link id")
	}

	if err := s.fileService.RevokeShareLink(ctx, id, tenantID); err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.RevokeShareLinkResponse{}, nil
}

// GetSharedFile serves the unauthenticated public redemption of a share link.
// Unlike every other handler in this file, it takes no tenant from the
// context — the token itself resolves one, deep inside
// file.Service.RedeemShareLink.
func (s *DocumentGRPCServer) GetSharedFile(ctx context.Context, req *documentv1.GetSharedFileRequest) (*documentv1.GetSharedFileResponse, error) {
	password := ""
	if req.Password != nil {
		password = *req.Password
	}

	dl, err := s.fileService.RedeemShareLink(ctx, req.Token, password)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.GetSharedFileResponse{
		DownloadUrl: dl.DownloadURL,
		Filename:    dl.Filename,
		ContentType: dl.ContentType,
		FileSize:    dl.FileSize,
	}, nil
}

// ============================================================================
// Share Operations
// ============================================================================

func (s *DocumentGRPCServer) ShareEntity(ctx context.Context, req *documentv1.ShareEntityRequest) (*documentv1.ShareEntityResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}

	entityID, err := uuid.Parse(req.EntityId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entity_id")
	}

	sharedWith, err := uuid.Parse(req.SharedWithUserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid shared_with_user_id")
	}

	sharedBy, err := uuid.Parse(req.SharedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid shared_by")
	}

	input := share.ShareInput{
		TenantID:         tenantID,
		EntityType:       req.EntityType,
		EntityID:         entityID,
		SharedWithUserID: sharedWith,
		Permission:       permissionFromProto(req.Permission),
		SharedBy:         sharedBy,
	}

	sh, err := s.shareService.ShareEntity(ctx, input)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.ShareEntityResponse{Share: toProtoShare(sh)}, nil
}

func (s *DocumentGRPCServer) UnshareEntity(ctx context.Context, req *documentv1.UnshareEntityRequest) (*documentv1.UnshareEntityResponse, error) {
	entityID, err := uuid.Parse(req.EntityId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entity_id")
	}

	// UnshareEntity in share service uses shareID; here we look up by entity+user
	shares, err := s.shareService.ListByEntity(ctx, req.EntityType, entityID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	sharedWithUserID, err := uuid.Parse(req.SharedWithUserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid shared_with_user_id")
	}

	for _, sh := range shares {
		if sh.SharedWithUserID == sharedWithUserID {
			if err := s.shareService.UnshareEntity(ctx, sh.ID); err != nil {
				return nil, mapDocumentError(err)
			}
			return &documentv1.UnshareEntityResponse{}, nil
		}
	}

	return nil, status.Error(codes.NotFound, "share not found")
}

func (s *DocumentGRPCServer) ListShares(ctx context.Context, req *documentv1.ListSharesRequest) (*documentv1.ListSharesResponse, error) {
	entityID, err := uuid.Parse(req.EntityId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entity_id")
	}

	shares, err := s.shareService.ListByEntity(ctx, req.EntityType, entityID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoShares := make([]*documentv1.DocumentShare, 0, len(shares))
	for _, sh := range shares {
		protoShares = append(protoShares, toProtoShare(sh))
	}

	return &documentv1.ListSharesResponse{Shares: protoShares}, nil
}

func (s *DocumentGRPCServer) ListSharedWithMe(ctx context.Context, req *documentv1.ListSharedWithMeRequest) (*documentv1.ListSharedWithMeResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	fileShares, err := s.shareService.ListSharedWithMe(ctx, userID, "file")
	if err != nil {
		return nil, mapDocumentError(err)
	}

	folderShares, err := s.shareService.ListSharedWithMe(ctx, userID, "folder")
	if err != nil {
		return nil, mapDocumentError(err)
	}

	// For now return empty lists - the full implementation would fetch file/folder details
	return &documentv1.ListSharedWithMeResponse{
		Files:   make([]*documentv1.DocumentFile, 0),
		Folders: make([]*documentv1.DocumentFolder, 0),
		Total:   int32(len(fileShares) + len(folderShares)),
	}, nil
}

// ============================================================================
// Tag Operations
// ============================================================================

func (s *DocumentGRPCServer) CreateTag(ctx context.Context, req *documentv1.CreateTagRequest) (*documentv1.CreateTagResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	t, err := s.tagService.CreateTag(ctx, req.Name, req.Color, createdBy, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.CreateTagResponse{Tag: toProtoTag(t)}, nil
}

func (s *DocumentGRPCServer) ListTags(ctx context.Context, req *documentv1.ListTagsRequest) (*documentv1.ListTagsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}

	tags, err := s.tagService.ListTags(ctx, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoTags := make([]*documentv1.DocumentTag, 0, len(tags))
	for _, t := range tags {
		protoTags = append(protoTags, toProtoTag(t))
	}

	return &documentv1.ListTagsResponse{Tags: protoTags}, nil
}

func (s *DocumentGRPCServer) DeleteTag(ctx context.Context, req *documentv1.DeleteTagRequest) (*documentv1.DeleteTagResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tag id")
	}

	if err := s.tagService.DeleteTag(ctx, tenantID, id); err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.DeleteTagResponse{}, nil
}

func (s *DocumentGRPCServer) TagFile(ctx context.Context, req *documentv1.TagFileRequest) (*documentv1.TagFileResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	tagID, err := uuid.Parse(req.TagId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tag_id")
	}

	if err := s.tagService.TagFile(ctx, tenantID, fileID, tagID); err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.TagFileResponse{}, nil
}

func (s *DocumentGRPCServer) UntagFile(ctx context.Context, req *documentv1.UntagFileRequest) (*documentv1.UntagFileResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	tagID, err := uuid.Parse(req.TagId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tag_id")
	}

	if err := s.tagService.UntagFile(ctx, tenantID, fileID, tagID); err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.UntagFileResponse{}, nil
}

// ============================================================================
// Entity Link Operations
// ============================================================================

func (s *DocumentGRPCServer) LinkFileToEntity(ctx context.Context, req *documentv1.LinkFileToEntityRequest) (*documentv1.LinkFileToEntityResponse, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	entityID, err := uuid.Parse(req.EntityId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entity_id")
	}

	linkedBy, err := uuid.Parse(req.LinkedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid linked_by")
	}

	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	if err := s.fileService.LinkToEntity(ctx, fileID, req.EntityType, entityID, linkedBy, tenantID); err != nil {
		return nil, mapDocumentError(err)
	}

	// Return the created link by fetching links for this file
	links, err := s.fileService.ListEntityLinks(ctx, fileID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	// Find the most recently created link for this entity
	for _, link := range links {
		if link.EntityType == req.EntityType && link.EntityID == entityID {
			return &documentv1.LinkFileToEntityResponse{Link: toProtoEntityLink(link)}, nil
		}
	}

	return &documentv1.LinkFileToEntityResponse{}, nil
}

func (s *DocumentGRPCServer) UnlinkFileFromEntity(ctx context.Context, req *documentv1.UnlinkFileFromEntityRequest) (*documentv1.UnlinkFileFromEntityResponse, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	entityID, err := uuid.Parse(req.EntityId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entity_id")
	}

	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	// Find the link by file+entity and delete it
	links, err := s.fileService.ListEntityLinks(ctx, fileID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	for _, link := range links {
		if link.EntityType == req.EntityType && link.EntityID == entityID {
			if err := s.fileService.UnlinkFromEntity(ctx, link.ID, tenantID); err != nil {
				return nil, mapDocumentError(err)
			}
			return &documentv1.UnlinkFileFromEntityResponse{}, nil
		}
	}

	return nil, status.Error(codes.NotFound, "entity link not found")
}

// DeleteEntityLink removes an entity link by its own ID, without requiring
// the caller to know which file or entity it points at.
func (s *DocumentGRPCServer) DeleteEntityLink(ctx context.Context, req *documentv1.DeleteEntityLinkRequest) (*documentv1.DeleteEntityLinkResponse, error) {
	linkID, err := uuid.Parse(req.LinkId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid link_id")
	}

	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	if err := s.fileService.UnlinkFromEntity(ctx, linkID, tenantID); err != nil {
		return nil, mapDocumentError(err)
	}

	return &documentv1.DeleteEntityLinkResponse{}, nil
}

func (s *DocumentGRPCServer) ListFileEntityLinks(ctx context.Context, req *documentv1.ListFileEntityLinksRequest) (*documentv1.ListFileEntityLinksResponse, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	links, err := s.fileService.ListEntityLinks(ctx, fileID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoLinks := make([]*documentv1.DocumentEntityLink, 0, len(links))
	for _, link := range links {
		protoLinks = append(protoLinks, toProtoEntityLink(link))
	}

	return &documentv1.ListFileEntityLinksResponse{Links: protoLinks}, nil
}

// ListFilesByEntity returns the files linked to an arbitrary entity (e.g. a
// CRM contact), the reverse lookup of ListFileEntityLinks.
func (s *DocumentGRPCServer) ListFilesByEntity(ctx context.Context, req *documentv1.ListFilesByEntityRequest) (*documentv1.ListFilesByEntityResponse, error) {
	entityID, err := uuid.Parse(req.EntityId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entity_id")
	}

	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	files, err := s.fileService.ListByEntity(ctx, req.EntityType, entityID, tenantID)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoFiles := make([]*documentv1.DocumentFile, 0, len(files))
	for _, f := range files {
		protoFiles = append(protoFiles, toProtoFile(f))
	}

	return &documentv1.ListFilesByEntityResponse{Files: protoFiles}, nil
}

// ============================================================================
// Search Operations
// ============================================================================

func (s *DocumentGRPCServer) SearchFiles(ctx context.Context, req *documentv1.SearchFilesRequest) (*documentv1.SearchFilesResponse, error) {
	filter := search.SearchFilter{
		Limit:  int(req.PerPage),
		Offset: int((req.Page - 1) * req.PerPage),
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	if req.FolderId != "" {
		fid, err := uuid.Parse(req.FolderId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid folder_id")
		}
		filter.FolderID = &fid
	}

	for _, tagIDStr := range req.TagIds {
		tid, err := uuid.Parse(tagIDStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tag_id: "+tagIDStr)
		}
		filter.TagIDs = append(filter.TagIDs, tid)
	}

	results, total, err := s.searchService.Search(ctx, req.Query, filter)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoResults := make([]*documentv1.FileSearchResult, 0, len(results))
	for _, r := range results {
		protoResults = append(protoResults, &documentv1.FileSearchResult{
			File:    toProtoFile(&r.File),
			Rank:    float32(r.Rank),
			Snippet: r.Snippet,
		})
	}

	return &documentv1.SearchFilesResponse{Results: protoResults, Total: int32(total)}, nil
}

func (s *DocumentGRPCServer) ListVirtualFiles(ctx context.Context, req *documentv1.ListVirtualFilesRequest) (*documentv1.ListVirtualFilesResponse, error) {
	limit := int(req.PerPage)
	if limit <= 0 {
		limit = 50
	}
	offset := int((req.Page - 1) * req.PerPage)
	if offset < 0 {
		offset = 0
	}

	sourceType := virtualSourceFromProto(req.SourceType)

	// Use Nil for userID since gateway handles auth context
	files, total, err := s.virtualService.ListVirtualFiles(ctx, uuid.Nil, sourceType, limit, offset)
	if err != nil {
		return nil, mapDocumentError(err)
	}

	protoFiles := make([]*documentv1.VirtualFile, 0, len(files))
	for _, f := range files {
		protoFiles = append(protoFiles, toProtoVirtualFile(f))
	}

	return &documentv1.ListVirtualFilesResponse{Files: protoFiles, Total: int32(total)}, nil
}

// ============================================================================
// WOPI Operations
// ============================================================================

func (s *DocumentGRPCServer) GenerateWOPIToken(ctx context.Context, req *documentv1.GenerateWOPITokenRequest) (*documentv1.GenerateWOPITokenResponse, error) {
	if req.FileId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id and user_id are required")
	}

	// Extract tenantID from JWT context to embed in WOPI token for file scoping.
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	tenantIDStr := ""
	if tenantErr == nil {
		tenantIDStr = tenantID.String()
	}

	// Default user name (gateway can pass more info)
	userName := "User"

	token, ttlMs, err := s.tokenService.Generate(req.FileId, req.UserId, userName, tenantIDStr, true)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate WOPI token")
	}

	ttlSeconds := ttlMs / 1000

	return &documentv1.GenerateWOPITokenResponse{
		AccessToken: token,
		TtlSeconds:  ttlSeconds,
	}, nil
}

// ============================================================================
// Presigned URL Operations
// ============================================================================

func (s *DocumentGRPCServer) GetPresignedUploadURL(ctx context.Context, req *documentv1.GetPresignedUploadURLRequest) (*documentv1.GetPresignedUploadURLResponse, error) {
	result, err := s.fileService.GetPresignedUploadURL(ctx, req.Scope, req.FileName, req.ContentType, req.SizeBytes)
	if err != nil {
		// Service already returns gRPC status errors.
		return nil, err
	}

	return &documentv1.GetPresignedUploadURLResponse{
		UploadUrl: result.URL,
		ObjectKey: result.ObjectKey,
		ExpiresAt: timestamppb.New(result.ExpiresAt),
	}, nil
}

func (s *DocumentGRPCServer) GetPresignedDownloadURL(ctx context.Context, req *documentv1.GetPresignedDownloadURLRequest) (*documentv1.GetPresignedDownloadURLResponse, error) {
	result, err := s.fileService.GetPresignedDownloadURL(ctx, req.ObjectKey)
	if err != nil {
		// Service already returns gRPC status errors.
		return nil, err
	}

	return &documentv1.GetPresignedDownloadURLResponse{
		DownloadUrl: result.URL,
		ExpiresAt:   timestamppb.New(result.ExpiresAt),
	}, nil
}

func (s *DocumentGRPCServer) GetWOPIDiscovery(ctx context.Context, req *documentv1.GetWOPIDiscoveryRequest) (*documentv1.GetWOPIDiscoveryResponse, error) {
	// Return a static list of supported actions for OnlyOffice
	actions := []*documentv1.WOPIAction{
		{App: "Word", Name: "edit", Ext: "docx"},
		{App: "Word", Name: "view", Ext: "docx"},
		{App: "Word", Name: "edit", Ext: "doc"},
		{App: "Word", Name: "view", Ext: "doc"},
		{App: "Word", Name: "edit", Ext: "odt"},
		{App: "Word", Name: "view", Ext: "odt"},
		{App: "Excel", Name: "edit", Ext: "xlsx"},
		{App: "Excel", Name: "view", Ext: "xlsx"},
		{App: "Excel", Name: "edit", Ext: "xls"},
		{App: "Excel", Name: "view", Ext: "xls"},
		{App: "Excel", Name: "edit", Ext: "ods"},
		{App: "Excel", Name: "view", Ext: "ods"},
		{App: "PowerPoint", Name: "edit", Ext: "pptx"},
		{App: "PowerPoint", Name: "view", Ext: "pptx"},
		{App: "PowerPoint", Name: "edit", Ext: "ppt"},
		{App: "PowerPoint", Name: "view", Ext: "ppt"},
		{App: "PowerPoint", Name: "edit", Ext: "odp"},
		{App: "PowerPoint", Name: "view", Ext: "odp"},
	}

	return &documentv1.GetWOPIDiscoveryResponse{
		Actions: actions,
	}, nil
}

// ============================================================================
// Proto Conversion Helpers
// ============================================================================

func toProtoFolder(f *models.DocumentFolder) *documentv1.DocumentFolder {
	pf := &documentv1.DocumentFolder{
		Id:        f.ID.String(),
		Name:      f.Name,
		SpaceType: spaceTypeToProto(f.SpaceType),
		SpaceId:   f.SpaceID.String(),
		IsSystem:  f.IsSystem,
		Icon:      f.Icon,
		CreatedBy: f.CreatedBy.String(),
		CreatedAt: timestamppb.New(f.CreatedAt),
		UpdatedAt: timestamppb.New(f.UpdatedAt),
		FileCount: int32(f.FileCount),
	}
	if f.ParentID != nil {
		pf.ParentId = f.ParentID.String()
	}
	return pf
}

func toProtoFile(f *models.DocumentFile) *documentv1.DocumentFile {
	pf := &documentv1.DocumentFile{
		Id:             f.ID.String(),
		FolderId:       f.FolderID.String(),
		Filename:       f.Filename,
		MimeType:       f.MimeType,
		FileSize:       f.FileSize,
		StorageKey:     f.StorageKey,
		CurrentVersion: int32(f.CurrentVersion),
		OwnerId:        f.OwnerID.String(),
		IsFavorite:     f.IsFavorite,
		IsDeleted:      f.IsDeleted,
		CreatedAt:      timestamppb.New(f.CreatedAt),
		UpdatedAt:      timestamppb.New(f.UpdatedAt),
	}
	if f.ThumbnailKey != nil {
		pf.ThumbnailKey = *f.ThumbnailKey
	}
	pf.Tags = make([]*documentv1.DocumentTag, 0, len(f.Tags))
	for _, t := range f.Tags {
		pf.Tags = append(pf.Tags, toProtoTag(&t))
	}
	return pf
}

func toProtoFileVersion(v *models.DocumentFileVersion) *documentv1.DocumentFileVersion {
	pv := &documentv1.DocumentFileVersion{
		Id:            v.ID.String(),
		FileId:        v.FileID.String(),
		VersionNumber: int32(v.VersionNumber),
		StorageKey:    v.StorageKey,
		FileSize:      v.FileSize,
		CreatedBy:     v.CreatedBy.String(),
		CreatedAt:     timestamppb.New(v.CreatedAt),
	}
	if v.VersionLabel != nil {
		pv.VersionLabel = *v.VersionLabel
	}
	return pv
}

func toProtoShare(sh *models.DocumentShare) *documentv1.DocumentShare {
	return &documentv1.DocumentShare{
		Id:                 sh.ID.String(),
		EntityType:         sh.EntityType,
		EntityId:           sh.EntityID.String(),
		SharedWithUserId:   sh.SharedWithUserID.String(),
		SharedWithUserName: sh.SharedWithUserName,
		Permission:         permissionToProto(sh.Permission),
		SharedBy:           sh.SharedBy.String(),
		SharedByName:       sh.SharedByName,
		CreatedAt:          timestamppb.New(sh.CreatedAt),
	}
}

func toProtoTag(t *models.DocumentTag) *documentv1.DocumentTag {
	pt := &documentv1.DocumentTag{
		Id:        t.ID.String(),
		Name:      t.Name,
		FileCount: int32(t.FileCount),
		CreatedBy: t.CreatedBy.String(),
		CreatedAt: timestamppb.New(t.CreatedAt),
	}
	if t.Color != nil {
		pt.Color = *t.Color
	}
	return pt
}

func toProtoEntityLink(link *models.DocumentEntityLink) *documentv1.DocumentEntityLink {
	return &documentv1.DocumentEntityLink{
		Id:         link.ID.String(),
		FileId:     link.FileID.String(),
		EntityType: link.EntityType,
		EntityId:   link.EntityID.String(),
		EntityName: link.EntityName,
		LinkedBy:   link.LinkedBy.String(),
		CreatedAt:  timestamppb.New(link.CreatedAt),
	}
}

func toProtoFileActivity(a *models.DocumentFileActivity) *documentv1.DocumentFileActivity {
	return &documentv1.DocumentFileActivity{
		Id:        a.ID.String(),
		FileId:    a.FileID.String(),
		Action:    a.Action,
		ActorId:   a.ActorID.String(),
		ActorName: a.ActorName,
		Detail:    a.Detail,
		CreatedAt: timestamppb.New(a.CreatedAt),
	}
}

func toProtoFileComment(c *models.DocumentFileComment) *documentv1.DocumentFileComment {
	return &documentv1.DocumentFileComment{
		Id:         c.ID.String(),
		FileId:     c.FileID.String(),
		AuthorId:   c.AuthorID.String(),
		AuthorName: c.AuthorName,
		Content:    c.Content,
		CreatedAt:  timestamppb.New(c.CreatedAt),
		UpdatedAt:  timestamppb.New(c.UpdatedAt),
	}
}

func toProtoShareLink(l *models.DocumentShareLink) *documentv1.ShareLink {
	link := &documentv1.ShareLink{
		Id:          l.ID.String(),
		FileId:      l.FileID.String(),
		Token:       l.Token,
		HasPassword: l.PasswordHash != nil,
		ViewCount:   int32(l.ViewCount),
		CreatedAt:   timestamppb.New(l.CreatedAt),
	}
	if l.ExpiresAt != nil {
		link.ExpiresAt = timestamppb.New(*l.ExpiresAt)
	}
	return link
}

func toProtoVirtualFile(f *models.VirtualFile) *documentv1.VirtualFile {
	return &documentv1.VirtualFile{
		Id:             f.ID.String(),
		Filename:       f.Filename,
		MimeType:       f.MimeType,
		FileSize:       f.FileSize,
		StorageKey:     f.StorageKey,
		SourceType:     virtualSourceToProto(f.SourceType),
		SourceId:       f.SourceID.String(),
		SourceName:     f.SourceName,
		UploadedBy:     f.UploadedBy.String(),
		UploadedByName: f.UploadedByName,
		CreatedAt:      timestamppb.New(f.CreatedAt),
	}
}

// ============================================================================
// Enum Conversion Helpers
// ============================================================================

func spaceTypeFromProto(st documentv1.FolderSpaceType) string {
	switch st {
	case documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_PERSONAL:
		return "personal"
	case documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_TEAM:
		return "team"
	case documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_PROJECT:
		return "project"
	default:
		return "personal"
	}
}

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

func permissionFromProto(p documentv1.SharePermission) string {
	switch p {
	case documentv1.SharePermission_SHARE_PERMISSION_READ:
		return "read"
	case documentv1.SharePermission_SHARE_PERMISSION_WRITE:
		return "write"
	default:
		return "read"
	}
}

func permissionToProto(p string) documentv1.SharePermission {
	switch p {
	case "read":
		return documentv1.SharePermission_SHARE_PERMISSION_READ
	case "write":
		return documentv1.SharePermission_SHARE_PERMISSION_WRITE
	default:
		return documentv1.SharePermission_SHARE_PERMISSION_UNSPECIFIED
	}
}

func sortFieldFromProto(f documentv1.FileSortField) string {
	switch f {
	case documentv1.FileSortField_FILE_SORT_NAME:
		return "name"
	case documentv1.FileSortField_FILE_SORT_SIZE:
		return "size"
	case documentv1.FileSortField_FILE_SORT_DATE:
		return "date"
	case documentv1.FileSortField_FILE_SORT_TYPE:
		return "type"
	default:
		return "date"
	}
}

func sortDirFromProto(d documentv1.SortDirection) string {
	switch d {
	case documentv1.SortDirection_SORT_ASC:
		return "asc"
	case documentv1.SortDirection_SORT_DESC:
		return "desc"
	default:
		return "desc"
	}
}

func virtualSourceFromProto(s documentv1.VirtualFileSource) string {
	switch s {
	case documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_CHAT:
		return "chat"
	case documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_EMAIL:
		return "email"
	case documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_TASK:
		return "task"
	default:
		return ""
	}
}

func virtualSourceToProto(s string) documentv1.VirtualFileSource {
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

func parseUUIDOrNil(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// actorIDFromContext reads the acting user from the gRPC context, populated
// by middleware.TenantInboundUnaryInterceptor from the gateway-propagated
// x-user-id metadata (see registry.go's TenantOutboundUnaryInterceptor wiring).
// Used to attribute document activity entries to a real actor instead of
// silently defaulting to uuid.Nil, which would fail the activity row's
// NOT NULL actor_id FK.
func actorIDFromContext(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(middleware.GetUserID(ctx))
	if err != nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing user context")
	}
	return id, nil
}

// ============================================================================
// Error Mapping
// ============================================================================

func mapDocumentError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	// Folder errors
	case errors.Is(err, folder.ErrFolderNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, folder.ErrFolderNameRequired),
		errors.Is(err, folder.ErrFolderNameTooLong),
		errors.Is(err, folder.ErrInvalidSpaceType),
		errors.Is(err, folder.ErrCannotMoveToSelf):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, folder.ErrCannotDeleteSystemFolder),
		errors.Is(err, folder.ErrCannotRenameSystemFolder):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, folder.ErrFolderNameConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, folder.ErrCircularParent):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, folder.ErrFolderNotEmpty):
		return status.Error(codes.FailedPrecondition, err.Error())

	// File errors
	case errors.Is(err, file.ErrFileNotFound),
		errors.Is(err, file.ErrVersionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, file.ErrFilenameRequired),
		errors.Is(err, file.ErrFilenameTooLong),
		errors.Is(err, file.ErrFileSizeZero),
		errors.Is(err, file.ErrFileTooLarge),
		errors.Is(err, file.ErrInvalidEntityType),
		errors.Is(err, file.ErrStorageKeyMissing):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, file.ErrVersionConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, file.ErrFileDeleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, file.ErrNoWritePermission):
		return status.Error(codes.PermissionDenied, err.Error())

	// Comment errors
	case errors.Is(err, file.ErrCommentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, file.ErrCommentContentRequired),
		errors.Is(err, file.ErrCommentContentTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, file.ErrCannotEditOthersComment),
		errors.Is(err, file.ErrCannotDeleteOthersComment):
		return status.Error(codes.PermissionDenied, err.Error())

	// Share errors
	case errors.Is(err, share.ErrShareNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, share.ErrInvalidEntityType),
		errors.Is(err, share.ErrInvalidPermission),
		errors.Is(err, share.ErrCannotShareWithSelf):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, share.ErrAlreadyShared):
		return status.Error(codes.AlreadyExists, err.Error())

	// Share link errors. ErrShareLinkInvalid is deliberately the single
	// answer for every way the public redemption route can fail (unknown
	// token, revoked, expired, missing or wrong password) — see
	// file.Service.RedeemShareLink.
	case errors.Is(err, file.ErrShareLinkNotFound), errors.Is(err, file.ErrShareLinkInvalid):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, file.ErrShareLinkExpiryInvalid),
		errors.Is(err, file.ErrSharePasswordTooLong):
		return status.Error(codes.InvalidArgument, err.Error())

	// Tag errors
	case errors.Is(err, tag.ErrTagNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, tag.ErrTagNameRequired),
		errors.Is(err, tag.ErrTagNameTooLong),
		errors.Is(err, tag.ErrInvalidColor):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, tag.ErrTagNameConflict):
		return status.Error(codes.AlreadyExists, err.Error())

	// Search errors
	case errors.Is(err, search.ErrSearchQueryRequired):
		return status.Error(codes.InvalidArgument, err.Error())

	default:
		slog.Error("unhandled document service error", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
