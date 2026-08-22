package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/document/file"
	"github.com/kmuhub/kmuhub/internal/document/folder"
	"github.com/kmuhub/kmuhub/internal/document/search"
	"github.com/kmuhub/kmuhub/internal/document/share"
	"github.com/kmuhub/kmuhub/internal/document/tag"
	"github.com/kmuhub/kmuhub/internal/models"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

// newTestDocumentServer builds a DocumentGRPCServer with every service left
// nil, mirroring the newTestHRServer/newTestFormulareServer trick: it is only
// safe for the request-validation branches that return before touching a
// service field. Several handlers in this file (GetSharedFile,
// GetPresignedUploadURL, GetPresignedDownloadURL, ListVirtualFiles) call
// straight into a service with zero prior validation - those are documented
// as untestable-without-a-stub in the journal rather than exercised here.
func newTestDocumentServer() *DocumentGRPCServer {
	return NewDocumentGRPCServer(nil, nil, nil, nil, nil, nil, nil, nil)
}

// ---------------------------------------------------------------------------
// mapDocumentError - one gRPC code per sentinel
// ---------------------------------------------------------------------------

func TestMapDocumentError_AllSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"nil", nil, codes.OK},

		// folder
		{"FolderNotFound", folder.ErrFolderNotFound, codes.NotFound},
		{"FolderNameRequired", folder.ErrFolderNameRequired, codes.InvalidArgument},
		{"FolderNameTooLong", folder.ErrFolderNameTooLong, codes.InvalidArgument},
		{"InvalidSpaceType", folder.ErrInvalidSpaceType, codes.InvalidArgument},
		{"CannotMoveToSelf", folder.ErrCannotMoveToSelf, codes.InvalidArgument},
		{"CannotDeleteSystemFolder", folder.ErrCannotDeleteSystemFolder, codes.FailedPrecondition},
		{"CannotRenameSystemFolder", folder.ErrCannotRenameSystemFolder, codes.FailedPrecondition},
		{"FolderNameConflict", folder.ErrFolderNameConflict, codes.AlreadyExists},
		{"CircularParent", folder.ErrCircularParent, codes.FailedPrecondition},
		{"FolderNotEmpty", folder.ErrFolderNotEmpty, codes.FailedPrecondition},

		// file
		{"FileNotFound", file.ErrFileNotFound, codes.NotFound},
		{"VersionNotFound", file.ErrVersionNotFound, codes.NotFound},
		{"FilenameRequired", file.ErrFilenameRequired, codes.InvalidArgument},
		{"FilenameTooLong", file.ErrFilenameTooLong, codes.InvalidArgument},
		{"FileSizeZero", file.ErrFileSizeZero, codes.InvalidArgument},
		{"FileTooLarge", file.ErrFileTooLarge, codes.InvalidArgument},
		{"InvalidEntityType_File", file.ErrInvalidEntityType, codes.InvalidArgument},
		{"StorageKeyMissing", file.ErrStorageKeyMissing, codes.InvalidArgument},
		{"VersionConflict", file.ErrVersionConflict, codes.AlreadyExists},
		{"FileDeleted", file.ErrFileDeleted, codes.FailedPrecondition},
		{"NoWritePermission", file.ErrNoWritePermission, codes.PermissionDenied},

		// comments
		{"CommentNotFound", file.ErrCommentNotFound, codes.NotFound},
		{"CommentContentRequired", file.ErrCommentContentRequired, codes.InvalidArgument},
		{"CommentContentTooLong", file.ErrCommentContentTooLong, codes.InvalidArgument},
		{"CannotEditOthersComment", file.ErrCannotEditOthersComment, codes.PermissionDenied},
		{"CannotDeleteOthersComment", file.ErrCannotDeleteOthersComment, codes.PermissionDenied},

		// share
		{"ShareNotFound", share.ErrShareNotFound, codes.NotFound},
		{"InvalidEntityType_Share", share.ErrInvalidEntityType, codes.InvalidArgument},
		{"InvalidPermission", share.ErrInvalidPermission, codes.InvalidArgument},
		{"CannotShareWithSelf", share.ErrCannotShareWithSelf, codes.InvalidArgument},
		{"AlreadyShared", share.ErrAlreadyShared, codes.AlreadyExists},

		// share links
		{"ShareLinkNotFound", file.ErrShareLinkNotFound, codes.NotFound},
		{"ShareLinkInvalid", file.ErrShareLinkInvalid, codes.NotFound},
		{"ShareLinkExpiryInvalid", file.ErrShareLinkExpiryInvalid, codes.InvalidArgument},
		{"SharePasswordTooLong", file.ErrSharePasswordTooLong, codes.InvalidArgument},

		// tags
		{"TagNotFound", tag.ErrTagNotFound, codes.NotFound},
		{"TagNameRequired", tag.ErrTagNameRequired, codes.InvalidArgument},
		{"TagNameTooLong", tag.ErrTagNameTooLong, codes.InvalidArgument},
		{"InvalidColor", tag.ErrInvalidColor, codes.InvalidArgument},
		{"TagNameConflict", tag.ErrTagNameConflict, codes.AlreadyExists},

		// search
		{"SearchQueryRequired", search.ErrSearchQueryRequired, codes.InvalidArgument},

		{"UnknownError", errUnmapped, codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapDocumentError(tc.err)
			if tc.err == nil {
				require.NoError(t, got)
				return
			}
			requireGRPCCode(t, got, tc.code)
		})
	}
}

// errUnmapped stands in for any error the switch in mapDocumentError does not
// recognize - it must fall through to the default Internal branch without
// leaking the underlying error text (mapDocumentError logs it via slog
// instead of returning it to the caller).
var errUnmapped = &unmappedErr{}

type unmappedErr struct{}

func (e *unmappedErr) Error() string { return "unmapped test error" }

// ---------------------------------------------------------------------------
// toProto* converters - fully populated + the one nil-slice regression
// ---------------------------------------------------------------------------

// None of the ten toProto* converters in document_grpc.go guard against a nil
// input pointer (unlike hr_grpc.go's toProto* family) - every real caller
// ranges over a service-returned slice or checks err == nil first, so a nil
// argument never reaches them on any real path. Documented here, not
// exercised, to avoid a spurious nil-pointer panic in the test itself.

func TestToProtoFolder_Populated(t *testing.T) {
	parentID := uuid.New()
	f := &models.DocumentFolder{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Name:      "Contracts",
		ParentID:  &parentID,
		SpaceType: models.FolderSpaceTeam,
		SpaceID:   uuid.New(),
		IsSystem:  true,
		Icon:      "folder",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		FileCount: 4,
	}
	p := toProtoFolder(f)
	require.Equal(t, f.ID.String(), p.Id)
	require.Equal(t, parentID.String(), p.ParentId)
	require.Equal(t, documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_TEAM, p.SpaceType)
	require.True(t, p.IsSystem)
	require.Equal(t, int32(4), p.FileCount)
}

func TestToProtoFolder_NoParent(t *testing.T) {
	f := &models.DocumentFolder{
		ID:        uuid.New(),
		SpaceType: models.FolderSpacePersonal,
		SpaceID:   uuid.New(),
		CreatedBy: uuid.New(),
	}
	p := toProtoFolder(f)
	require.Empty(t, p.ParentId)
	require.Equal(t, documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_PERSONAL, p.SpaceType)
}

func TestToProtoFile_Populated(t *testing.T) {
	thumb := "thumb-key"
	color := "#00ff00"
	f := &models.DocumentFile{
		ID:             uuid.New(),
		FolderID:       uuid.New(),
		Filename:       "report.pdf",
		MimeType:       "application/pdf",
		FileSize:       1024,
		StorageKey:     "key-1",
		ThumbnailKey:   &thumb,
		CurrentVersion: 3,
		OwnerID:        uuid.New(),
		IsFavorite:     true,
		IsDeleted:      false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Tags: []models.DocumentTag{
			{ID: uuid.New(), Name: "wichtig", Color: &color, CreatedBy: uuid.New(), CreatedAt: time.Now()},
		},
	}
	p := toProtoFile(f)
	require.Equal(t, "thumb-key", p.ThumbnailKey)
	require.Equal(t, int32(3), p.CurrentVersion)
	require.True(t, p.IsFavorite)
	require.Len(t, p.Tags, 1)
	require.Equal(t, "wichtig", p.Tags[0].Name)
}

// TestToProtoFile_EmptyTagsIsNotNil is the wire-shape regression this unit's
// scope calls out by name: a DocumentFile with zero tags used to leave
// pf.Tags as a nil slice (append onto an unset field with zero iterations),
// which protojson serializes as `null` instead of `[]` on the wire - exactly
// the class of drift the backlog notes for this module ("leere Listen als
// null"). Fixed in this commit by pre-allocating with make(...).
func TestToProtoFile_EmptyTagsIsNotNil(t *testing.T) {
	f := &models.DocumentFile{
		ID:       uuid.New(),
		FolderID: uuid.New(),
		OwnerID:  uuid.New(),
	}
	p := toProtoFile(f)
	require.NotNil(t, p.Tags)
	require.Len(t, p.Tags, 0)
}

func TestToProtoFileVersion_Populated(t *testing.T) {
	label := "v2-final"
	v := &models.DocumentFileVersion{
		ID:            uuid.New(),
		FileID:        uuid.New(),
		VersionNumber: 2,
		VersionLabel:  &label,
		StorageKey:    "key",
		FileSize:      512,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now(),
	}
	p := toProtoFileVersion(v)
	require.Equal(t, int32(2), p.VersionNumber)
	require.Equal(t, "v2-final", p.VersionLabel)
}

func TestToProtoFileVersion_NoLabel(t *testing.T) {
	v := &models.DocumentFileVersion{ID: uuid.New(), FileID: uuid.New(), CreatedBy: uuid.New()}
	p := toProtoFileVersion(v)
	require.Empty(t, p.VersionLabel)
}

func TestToProtoShare_Populated(t *testing.T) {
	sh := &models.DocumentShare{
		ID:                 uuid.New(),
		EntityType:         "file",
		EntityID:           uuid.New(),
		SharedWithUserID:   uuid.New(),
		SharedWithUserName: "Jane",
		Permission:         models.SharePermissionWrite,
		SharedBy:           uuid.New(),
		SharedByName:       "Boss",
		CreatedAt:          time.Now(),
	}
	p := toProtoShare(sh)
	require.Equal(t, documentv1.SharePermission_SHARE_PERMISSION_WRITE, p.Permission)
	require.Equal(t, "Jane", p.SharedWithUserName)
}

func TestToProtoTag_Populated(t *testing.T) {
	color := "#123456"
	tg := &models.DocumentTag{
		ID:        uuid.New(),
		Name:      "urgent",
		Color:     &color,
		FileCount: 7,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
	}
	p := toProtoTag(tg)
	require.Equal(t, "#123456", p.Color)
	require.Equal(t, int32(7), p.FileCount)
}

func TestToProtoTag_NoColor(t *testing.T) {
	tg := &models.DocumentTag{ID: uuid.New(), Name: "plain", CreatedBy: uuid.New()}
	p := toProtoTag(tg)
	require.Empty(t, p.Color)
}

func TestToProtoEntityLink_Populated(t *testing.T) {
	link := &models.DocumentEntityLink{
		ID:         uuid.New(),
		FileID:     uuid.New(),
		EntityType: "contact",
		EntityID:   uuid.New(),
		EntityName: "Acme GmbH",
		LinkedBy:   uuid.New(),
		CreatedAt:  time.Now(),
	}
	p := toProtoEntityLink(link)
	require.Equal(t, "contact", p.EntityType)
	require.Equal(t, "Acme GmbH", p.EntityName)
}

func TestToProtoFileActivity_Populated(t *testing.T) {
	a := &models.DocumentFileActivity{
		ID:        uuid.New(),
		FileID:    uuid.New(),
		Action:    models.DocumentActivityRenamed,
		ActorID:   uuid.New(),
		ActorName: "Jane",
		Detail:    "old -> new",
		CreatedAt: time.Now(),
	}
	p := toProtoFileActivity(a)
	require.Equal(t, "renamed", p.Action)
	require.Equal(t, "old -> new", p.Detail)
}

func TestToProtoFileComment_Populated(t *testing.T) {
	c := &models.DocumentFileComment{
		ID:         uuid.New(),
		FileID:     uuid.New(),
		AuthorID:   uuid.New(),
		AuthorName: "Jane",
		Content:    "looks good",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	p := toProtoFileComment(c)
	require.Equal(t, "looks good", p.Content)
	require.Equal(t, "Jane", p.AuthorName)
}

func TestToProtoShareLink_Populated(t *testing.T) {
	hash := "hashed"
	expiresAt := time.Now().Add(24 * time.Hour)
	l := &models.DocumentShareLink{
		ID:           uuid.New(),
		FileID:       uuid.New(),
		Token:        "tok",
		PasswordHash: &hash,
		ExpiresAt:    &expiresAt,
		ViewCount:    5,
		CreatedAt:    time.Now(),
	}
	p := toProtoShareLink(l)
	require.True(t, p.HasPassword)
	require.NotNil(t, p.ExpiresAt)
	require.Equal(t, int32(5), p.ViewCount)
}

func TestToProtoShareLink_NoPasswordNoExpiry(t *testing.T) {
	l := &models.DocumentShareLink{ID: uuid.New(), FileID: uuid.New(), Token: "tok"}
	p := toProtoShareLink(l)
	require.False(t, p.HasPassword)
	require.Nil(t, p.ExpiresAt)
}

func TestToProtoVirtualFile_Populated(t *testing.T) {
	f := &models.VirtualFile{
		ID:             uuid.New(),
		Filename:       "attachment.png",
		MimeType:       "image/png",
		FileSize:       2048,
		StorageKey:     "vf-key",
		SourceType:     models.VirtualFileSourceChat,
		SourceID:       uuid.New(),
		SourceName:     "#general",
		UploadedBy:     uuid.New(),
		UploadedByName: "Jane",
		CreatedAt:      time.Now(),
	}
	p := toProtoVirtualFile(f)
	require.Equal(t, documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_CHAT, p.SourceType)
	require.Equal(t, "#general", p.SourceName)
}

// ---------------------------------------------------------------------------
// Enum converters - round trip over every real value plus the Unspecified
// default in both directions
// ---------------------------------------------------------------------------

func TestSpaceTypeRoundTrip(t *testing.T) {
	cases := []struct {
		domain string
		proto  documentv1.FolderSpaceType
	}{
		{models.FolderSpacePersonal, documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_PERSONAL},
		{models.FolderSpaceTeam, documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_TEAM},
		{models.FolderSpaceProject, documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_PROJECT},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, spaceTypeToProto(tc.domain))
		require.Equal(t, tc.domain, spaceTypeFromProto(tc.proto))
	}
}

func TestSpaceTypeFromProto_UnspecifiedDefaultsToPersonal(t *testing.T) {
	require.Equal(t, models.FolderSpacePersonal, spaceTypeFromProto(documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_UNSPECIFIED))
}

func TestSpaceTypeToProto_UnknownDomainValue(t *testing.T) {
	require.Equal(t, documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_UNSPECIFIED, spaceTypeToProto("bogus"))
}

func TestPermissionRoundTrip(t *testing.T) {
	cases := []struct {
		domain string
		proto  documentv1.SharePermission
	}{
		{models.SharePermissionRead, documentv1.SharePermission_SHARE_PERMISSION_READ},
		{models.SharePermissionWrite, documentv1.SharePermission_SHARE_PERMISSION_WRITE},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, permissionToProto(tc.domain))
		require.Equal(t, tc.domain, permissionFromProto(tc.proto))
	}
}

func TestPermissionFromProto_UnspecifiedDefaultsToRead(t *testing.T) {
	require.Equal(t, models.SharePermissionRead, permissionFromProto(documentv1.SharePermission_SHARE_PERMISSION_UNSPECIFIED))
}

func TestPermissionToProto_UnknownDomainValue(t *testing.T) {
	require.Equal(t, documentv1.SharePermission_SHARE_PERMISSION_UNSPECIFIED, permissionToProto("bogus"))
}

func TestSortFieldFromProto_AllValues(t *testing.T) {
	cases := []struct {
		proto  documentv1.FileSortField
		domain string
	}{
		{documentv1.FileSortField_FILE_SORT_NAME, "name"},
		{documentv1.FileSortField_FILE_SORT_SIZE, "size"},
		{documentv1.FileSortField_FILE_SORT_DATE, "date"},
		{documentv1.FileSortField_FILE_SORT_TYPE, "type"},
		{documentv1.FileSortField_FILE_SORT_UNSPECIFIED, "date"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.domain, sortFieldFromProto(tc.proto))
	}
}

func TestSortDirFromProto_AllValues(t *testing.T) {
	cases := []struct {
		proto  documentv1.SortDirection
		domain string
	}{
		{documentv1.SortDirection_SORT_ASC, "asc"},
		{documentv1.SortDirection_SORT_DESC, "desc"},
		{documentv1.SortDirection_SORT_DIRECTION_UNSPECIFIED, "desc"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.domain, sortDirFromProto(tc.proto))
	}
}

func TestVirtualSourceRoundTrip(t *testing.T) {
	cases := []struct {
		domain string
		proto  documentv1.VirtualFileSource
	}{
		{models.VirtualFileSourceChat, documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_CHAT},
		{models.VirtualFileSourceEmail, documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_EMAIL},
		{models.VirtualFileSourceTask, documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_TASK},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, virtualSourceToProto(tc.domain))
		require.Equal(t, tc.domain, virtualSourceFromProto(tc.proto))
	}
}

func TestVirtualSourceFromProto_UnspecifiedDefaultsToEmpty(t *testing.T) {
	require.Empty(t, virtualSourceFromProto(documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_UNSPECIFIED))
}

func TestVirtualSourceToProto_UnknownDomainValue(t *testing.T) {
	require.Equal(t, documentv1.VirtualFileSource_VIRTUAL_FILE_SOURCE_UNSPECIFIED, virtualSourceToProto("bogus"))
}

func TestParseUUIDOrNil(t *testing.T) {
	id := uuid.New()
	require.Equal(t, id, parseUUIDOrNil(id.String()))
	require.Equal(t, uuid.Nil, parseUUIDOrNil("not-a-uuid"))
	require.Equal(t, uuid.Nil, parseUUIDOrNil(""))
}

func TestActorIDFromContext(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		_, err := actorIDFromContext(context.Background())
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("invalid", func(t *testing.T) {
		ctx := ctxWithTenant(uuid.New())
		_, err := actorIDFromContext(ctx)
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("present", func(t *testing.T) {
		userID := uuid.New()
		ctx := ctxWithActorAndTenant(userID, uuid.New())
		got, err := actorIDFromContext(ctx)
		require.NoError(t, err)
		require.Equal(t, userID, got)
	})
}

func TestGetWOPIDiscovery_ReturnsStaticActions(t *testing.T) {
	ts := newTestDocumentServer()
	resp, err := ts.GetWOPIDiscovery(context.Background(), &documentv1.GetWOPIDiscoveryRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Actions, 18)
}

// CreateFileVersion builds its response directly from the request without
// touching any service field (see the comment in document_grpc.go: "For
// gRPC version creation, we don't have a reader"), so unlike every other
// handler in this file it is safe to exercise end to end against the
// nil-service server, not just its validation branches.
func TestCreateFileVersion_BuildsResponseWithoutService(t *testing.T) {
	ts := newTestDocumentServer()
	fileID := uuid.New()
	createdBy := uuid.New()
	resp, err := ts.CreateFileVersion(context.Background(), &documentv1.CreateFileVersionRequest{
		FileId:       fileID.String(),
		CreatedBy:    createdBy.String(),
		StorageKey:   "key",
		FileSize:     100,
		VersionLabel: "v1",
	})
	require.NoError(t, err)
	require.Equal(t, fileID.String(), resp.Version.FileId)
	require.Equal(t, createdBy.String(), resp.Version.CreatedBy)
	require.Equal(t, "v1", resp.Version.VersionLabel)
}

// ---------------------------------------------------------------------------
// Handler validation paths - every branch that returns before touching a nil
// service field. Methods with zero such branches (GetSharedFile,
// GetPresignedUploadURL, GetPresignedDownloadURL, ListVirtualFiles) call
// straight into a service and cannot be exercised without a stub repository;
// documented in the journal instead of tested here.
// ---------------------------------------------------------------------------

func TestDocumentHandlers_Validation(t *testing.T) {
	bg := context.Background()
	tenant := uuid.New()
	tenantCtx := ctxWithTenant(tenant)
	someID := uuid.New().String()

	// -- Folder --

	t.Run("CreateFolder_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateFolder(bg, &documentv1.CreateFolderRequest{})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateFolder_InvalidCreatedBy", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateFolder(tenantCtx, &documentv1.CreateFolderRequest{CreatedBy: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateFolder_InvalidParentID", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateFolder(tenantCtx, &documentv1.CreateFolderRequest{
			CreatedBy: someID, ParentId: "bogus",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetFolder_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().GetFolder(bg, &documentv1.GetFolderRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetFolder_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().GetFolder(tenantCtx, &documentv1.GetFolderRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListFolders_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFolders(bg, &documentv1.ListFoldersRequest{})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListFolders_InvalidParentID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFolders(tenantCtx, &documentv1.ListFoldersRequest{ParentId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListFolders_InvalidSpaceID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFolders(tenantCtx, &documentv1.ListFoldersRequest{SpaceId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateFolder_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().UpdateFolder(bg, &documentv1.UpdateFolderRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateFolder_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().UpdateFolder(tenantCtx, &documentv1.UpdateFolderRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateFolder_InvalidParentID", func(t *testing.T) {
		bogus := "bogus"
		_, err := newTestDocumentServer().UpdateFolder(tenantCtx, &documentv1.UpdateFolderRequest{Id: someID, ParentId: &bogus})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteFolder_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteFolder(bg, &documentv1.DeleteFolderRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteFolder_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteFolder(tenantCtx, &documentv1.DeleteFolderRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetFolderPath_InvalidID", func(t *testing.T) {
		// No tenant check at all in this handler - see journal.
		_, err := newTestDocumentServer().GetFolderPath(bg, &documentv1.GetFolderPathRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("InitializeUserSpace_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().InitializeUserSpace(bg, &documentv1.InitializeUserSpaceRequest{UserId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("InitializeUserSpace_InvalidUserID", func(t *testing.T) {
		_, err := newTestDocumentServer().InitializeUserSpace(tenantCtx, &documentv1.InitializeUserSpaceRequest{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("InitializeTeamSpace_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().InitializeTeamSpace(bg, &documentv1.InitializeTeamSpaceRequest{TeamId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("InitializeTeamSpace_InvalidTeamID", func(t *testing.T) {
		_, err := newTestDocumentServer().InitializeTeamSpace(tenantCtx, &documentv1.InitializeTeamSpaceRequest{TeamId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// -- File --

	t.Run("GetFile_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().GetFile(bg, &documentv1.GetFileRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetFile_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().GetFile(tenantCtx, &documentv1.GetFileRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RegisterUploadedFile_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().RegisterUploadedFile(bg, &documentv1.RegisterUploadedFileRequest{})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RegisterUploadedFile_InvalidFolderID", func(t *testing.T) {
		_, err := newTestDocumentServer().RegisterUploadedFile(tenantCtx, &documentv1.RegisterUploadedFileRequest{FolderId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RegisterUploadedFile_InvalidOwnerID", func(t *testing.T) {
		_, err := newTestDocumentServer().RegisterUploadedFile(tenantCtx, &documentv1.RegisterUploadedFileRequest{
			FolderId: someID, OwnerId: "bogus",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UploadFile_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().UploadFile(bg, &documentv1.UploadFileRequest{})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UploadFile_InvalidFolderID", func(t *testing.T) {
		_, err := newTestDocumentServer().UploadFile(tenantCtx, &documentv1.UploadFileRequest{FolderId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UploadFile_InvalidOwnerID", func(t *testing.T) {
		_, err := newTestDocumentServer().UploadFile(tenantCtx, &documentv1.UploadFileRequest{
			FolderId: someID, OwnerId: "bogus",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListFiles_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFiles(bg, &documentv1.ListFilesRequest{})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListFiles_InvalidFolderID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFiles(tenantCtx, &documentv1.ListFilesRequest{FolderId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListFiles_InvalidOwnerID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFiles(tenantCtx, &documentv1.ListFilesRequest{OwnerId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateFile_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().UpdateFile(bg, &documentv1.UpdateFileRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateFile_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().UpdateFile(tenantCtx, &documentv1.UpdateFileRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateFile_InvalidFolderID", func(t *testing.T) {
		bogus := "bogus"
		_, err := newTestDocumentServer().UpdateFile(tenantCtx, &documentv1.UpdateFileRequest{Id: someID, FolderId: &bogus})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateFile_MissingActor", func(t *testing.T) {
		_, err := newTestDocumentServer().UpdateFile(tenantCtx, &documentv1.UpdateFileRequest{Id: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("DeleteFile_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteFile(bg, &documentv1.DeleteFileRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteFile_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteFile(tenantCtx, &documentv1.DeleteFileRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CopyFile_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().CopyFile(bg, &documentv1.CopyFileRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CopyFile_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().CopyFile(tenantCtx, &documentv1.CopyFileRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CopyFile_InvalidTargetFolderID", func(t *testing.T) {
		_, err := newTestDocumentServer().CopyFile(tenantCtx, &documentv1.CopyFileRequest{Id: someID, TargetFolderId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CopyFile_MissingActor", func(t *testing.T) {
		_, err := newTestDocumentServer().CopyFile(tenantCtx, &documentv1.CopyFileRequest{Id: someID, TargetFolderId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("MoveFile_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().MoveFile(bg, &documentv1.MoveFileRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("MoveFile_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().MoveFile(tenantCtx, &documentv1.MoveFileRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("MoveFile_InvalidTargetFolderID", func(t *testing.T) {
		_, err := newTestDocumentServer().MoveFile(tenantCtx, &documentv1.MoveFileRequest{Id: someID, TargetFolderId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("MoveFile_MissingActor", func(t *testing.T) {
		_, err := newTestDocumentServer().MoveFile(tenantCtx, &documentv1.MoveFileRequest{Id: someID, TargetFolderId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("GetFileDownloadURL_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().GetFileDownloadURL(bg, &documentv1.GetFileDownloadURLRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetFileDownloadURL_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().GetFileDownloadURL(tenantCtx, &documentv1.GetFileDownloadURLRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetFileVersionDownloadURL_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().GetFileVersionDownloadURL(bg, &documentv1.GetFileVersionDownloadURLRequest{
			FileId: someID, VersionId: someID,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetFileVersionDownloadURL_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().GetFileVersionDownloadURL(tenantCtx, &documentv1.GetFileVersionDownloadURLRequest{
			FileId: "bogus", VersionId: someID,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetFileVersionDownloadURL_InvalidVersionID", func(t *testing.T) {
		_, err := newTestDocumentServer().GetFileVersionDownloadURL(tenantCtx, &documentv1.GetFileVersionDownloadURLRequest{
			FileId: someID, VersionId: "bogus",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateFileVersion_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateFileVersion(bg, &documentv1.CreateFileVersionRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateFileVersion_InvalidCreatedBy", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateFileVersion(bg, &documentv1.CreateFileVersionRequest{FileId: someID, CreatedBy: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListFileVersions_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFileVersions(bg, &documentv1.ListFileVersionsRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RevertFileVersion_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().RevertFileVersion(bg, &documentv1.RevertFileVersionRequest{FileId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RevertFileVersion_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().RevertFileVersion(tenantCtx, &documentv1.RevertFileVersionRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RevertFileVersion_MissingActor", func(t *testing.T) {
		_, err := newTestDocumentServer().RevertFileVersion(tenantCtx, &documentv1.RevertFileVersionRequest{FileId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("ListFileActivity_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFileActivity(bg, &documentv1.ListFileActivityRequest{FileId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListFileActivity_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFileActivity(tenantCtx, &documentv1.ListFileActivityRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// -- Comments --

	t.Run("ListFileComments_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFileComments(bg, &documentv1.ListFileCommentsRequest{FileId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListFileComments_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFileComments(tenantCtx, &documentv1.ListFileCommentsRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateFileComment_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateFileComment(bg, &documentv1.CreateFileCommentRequest{FileId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateFileComment_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateFileComment(tenantCtx, &documentv1.CreateFileCommentRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateFileComment_MissingActor", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateFileComment(tenantCtx, &documentv1.CreateFileCommentRequest{FileId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("UpdateFileComment_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().UpdateFileComment(bg, &documentv1.UpdateFileCommentRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateFileComment_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().UpdateFileComment(tenantCtx, &documentv1.UpdateFileCommentRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateFileComment_MissingActor", func(t *testing.T) {
		_, err := newTestDocumentServer().UpdateFileComment(tenantCtx, &documentv1.UpdateFileCommentRequest{Id: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("DeleteFileComment_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteFileComment(bg, &documentv1.DeleteFileCommentRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteFileComment_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteFileComment(tenantCtx, &documentv1.DeleteFileCommentRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteFileComment_MissingActor", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteFileComment(tenantCtx, &documentv1.DeleteFileCommentRequest{Id: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	// -- Share links --

	t.Run("CreateShareLink_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateShareLink(bg, &documentv1.CreateShareLinkRequest{FileId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateShareLink_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateShareLink(tenantCtx, &documentv1.CreateShareLinkRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListShareLinks_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().ListShareLinks(bg, &documentv1.ListShareLinksRequest{FileId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListShareLinks_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListShareLinks(tenantCtx, &documentv1.ListShareLinksRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RevokeShareLink_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().RevokeShareLink(bg, &documentv1.RevokeShareLinkRequest{Id: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RevokeShareLink_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().RevokeShareLink(tenantCtx, &documentv1.RevokeShareLinkRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// -- Share --

	t.Run("ShareEntity_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().ShareEntity(bg, &documentv1.ShareEntityRequest{EntityId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("ShareEntity_InvalidEntityID", func(t *testing.T) {
		_, err := newTestDocumentServer().ShareEntity(tenantCtx, &documentv1.ShareEntityRequest{EntityId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ShareEntity_InvalidSharedWithUserID", func(t *testing.T) {
		_, err := newTestDocumentServer().ShareEntity(tenantCtx, &documentv1.ShareEntityRequest{
			EntityId: someID, SharedWithUserId: "bogus",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ShareEntity_InvalidSharedBy", func(t *testing.T) {
		_, err := newTestDocumentServer().ShareEntity(tenantCtx, &documentv1.ShareEntityRequest{
			EntityId: someID, SharedWithUserId: someID, SharedBy: "bogus",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UnshareEntity_InvalidEntityID", func(t *testing.T) {
		_, err := newTestDocumentServer().UnshareEntity(bg, &documentv1.UnshareEntityRequest{EntityId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListShares_InvalidEntityID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListShares(bg, &documentv1.ListSharesRequest{EntityId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListSharedWithMe_InvalidUserID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListSharedWithMe(bg, &documentv1.ListSharedWithMeRequest{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// -- Tags --

	t.Run("CreateTag_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateTag(bg, &documentv1.CreateTagRequest{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("CreateTag_InvalidCreatedBy", func(t *testing.T) {
		_, err := newTestDocumentServer().CreateTag(tenantCtx, &documentv1.CreateTagRequest{CreatedBy: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListTags_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().ListTags(bg, &documentv1.ListTagsRequest{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("DeleteTag_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteTag(bg, &documentv1.DeleteTagRequest{Id: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("DeleteTag_InvalidID", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteTag(tenantCtx, &documentv1.DeleteTagRequest{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("TagFile_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().TagFile(bg, &documentv1.TagFileRequest{FileId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("TagFile_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().TagFile(tenantCtx, &documentv1.TagFileRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("TagFile_InvalidTagID", func(t *testing.T) {
		_, err := newTestDocumentServer().TagFile(tenantCtx, &documentv1.TagFileRequest{FileId: someID, TagId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UntagFile_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().UntagFile(bg, &documentv1.UntagFileRequest{FileId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("UntagFile_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().UntagFile(tenantCtx, &documentv1.UntagFileRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UntagFile_InvalidTagID", func(t *testing.T) {
		_, err := newTestDocumentServer().UntagFile(tenantCtx, &documentv1.UntagFileRequest{FileId: someID, TagId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// -- Entity links --

	t.Run("LinkFileToEntity_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().LinkFileToEntity(bg, &documentv1.LinkFileToEntityRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("LinkFileToEntity_InvalidEntityID", func(t *testing.T) {
		_, err := newTestDocumentServer().LinkFileToEntity(bg, &documentv1.LinkFileToEntityRequest{
			FileId: someID, EntityId: "bogus",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("LinkFileToEntity_InvalidLinkedBy", func(t *testing.T) {
		_, err := newTestDocumentServer().LinkFileToEntity(bg, &documentv1.LinkFileToEntityRequest{
			FileId: someID, EntityId: someID, LinkedBy: "bogus",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("LinkFileToEntity_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().LinkFileToEntity(bg, &documentv1.LinkFileToEntityRequest{
			FileId: someID, EntityId: someID, LinkedBy: someID,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UnlinkFileFromEntity_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().UnlinkFileFromEntity(bg, &documentv1.UnlinkFileFromEntityRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UnlinkFileFromEntity_InvalidEntityID", func(t *testing.T) {
		_, err := newTestDocumentServer().UnlinkFileFromEntity(bg, &documentv1.UnlinkFileFromEntityRequest{
			FileId: someID, EntityId: "bogus",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UnlinkFileFromEntity_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().UnlinkFileFromEntity(bg, &documentv1.UnlinkFileFromEntityRequest{
			FileId: someID, EntityId: someID,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteEntityLink_InvalidLinkID", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteEntityLink(bg, &documentv1.DeleteEntityLinkRequest{LinkId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteEntityLink_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().DeleteEntityLink(bg, &documentv1.DeleteEntityLinkRequest{LinkId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListFileEntityLinks_InvalidFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFileEntityLinks(bg, &documentv1.ListFileEntityLinksRequest{FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListFileEntityLinks_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFileEntityLinks(bg, &documentv1.ListFileEntityLinksRequest{FileId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListFilesByEntity_InvalidEntityID", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFilesByEntity(bg, &documentv1.ListFilesByEntityRequest{EntityId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListFilesByEntity_MissingTenant", func(t *testing.T) {
		_, err := newTestDocumentServer().ListFilesByEntity(bg, &documentv1.ListFilesByEntityRequest{EntityId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// -- Search --

	t.Run("SearchFiles_InvalidFolderID", func(t *testing.T) {
		_, err := newTestDocumentServer().SearchFiles(bg, &documentv1.SearchFilesRequest{FolderId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("SearchFiles_InvalidTagID", func(t *testing.T) {
		_, err := newTestDocumentServer().SearchFiles(bg, &documentv1.SearchFilesRequest{TagIds: []string{"bogus"}})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// -- WOPI --

	t.Run("GenerateWOPIToken_MissingFileID", func(t *testing.T) {
		_, err := newTestDocumentServer().GenerateWOPIToken(bg, &documentv1.GenerateWOPITokenRequest{UserId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GenerateWOPIToken_MissingUserID", func(t *testing.T) {
		_, err := newTestDocumentServer().GenerateWOPIToken(bg, &documentv1.GenerateWOPITokenRequest{FileId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}
