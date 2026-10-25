package gateway

import (
	"context"
	"io"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/document/wopi"
	"github.com/kmuhub/kmuhub/internal/models"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

// WOPIFileAdapter bridges the document gRPC client to wopi.FileServiceInterface.
// This allows the gateway to serve WOPI endpoints using the document gRPC backend.
type WOPIFileAdapter struct {
	registry *ServiceRegistry
}

// NewWOPIFileAdapter creates a new adapter for WOPI file operations.
func NewWOPIFileAdapter(registry *ServiceRegistry) *WOPIFileAdapter {
	return &WOPIFileAdapter{registry: registry}
}

func (a *WOPIFileAdapter) getClient() (documentv1.DocumentServiceClient, error) {
	conn, err := a.registry.GetConnection("document")
	if err != nil {
		return nil, err
	}
	return documentv1.NewDocumentServiceClient(conn), nil
}

// GetByID retrieves a document file by ID via gRPC and converts to the domain model.
// tenantID is embedded in the gRPC JWT metadata by the gateway middleware.
func (a *WOPIFileAdapter) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.DocumentFile, error) {
	client, err := a.getClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetFile(ctx, &documentv1.GetFileRequest{Id: id.String()})
	if err != nil {
		return nil, err
	}

	f := resp.File
	ownerID, _ := uuid.Parse(f.OwnerId)
	folderID, _ := uuid.Parse(f.FolderId)

	return &models.DocumentFile{
		ID:             id,
		FolderID:       folderID,
		Filename:       f.Filename,
		MimeType:       f.MimeType,
		FileSize:       f.FileSize,
		StorageKey:     f.StorageKey,
		CurrentVersion: int(f.CurrentVersion),
		OwnerID:        ownerID,
		IsFavorite:     f.IsFavorite,
		IsDeleted:      f.IsDeleted,
	}, nil
}

// GetDownloadURL returns a presigned download URL via gRPC.
// tenantID is embedded in the gRPC JWT metadata by the gateway middleware.
func (a *WOPIFileAdapter) GetDownloadURL(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID) (string, error) {
	client, err := a.getClient()
	if err != nil {
		return "", err
	}

	resp, err := client.GetFileDownloadURL(ctx, &documentv1.GetFileDownloadURLRequest{Id: fileID.String()})
	if err != nil {
		return "", err
	}

	return resp.DownloadUrl, nil
}

// CreateVersion stores the editor's saved content as the file's new current
// version. The body travels in the request (the document service's receive
// limit covers the 50 MiB cap); an unchanged save creates no version there.
// tenantID is embedded in the gRPC metadata from the WOPI token claims.
func (a *WOPIFileAdapter) CreateVersion(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID, input wopi.VersionInput) (*models.DocumentFileVersion, error) {
	client, err := a.getClient()
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(io.LimitReader(input.Reader, maxDocumentUploadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxDocumentUploadBytes {
		return nil, wopi.ErrContentTooLarge
	}

	label := "Auto-saved via WOPI"
	if input.Label != nil {
		label = *input.Label
	}

	resp, err := client.CreateFileVersion(ctx, &documentv1.CreateFileVersionRequest{
		FileId:       fileID.String(),
		VersionLabel: label,
		Content:      body,
	}, grpc.MaxCallSendMsgSize(60<<20))
	if err != nil {
		return nil, err
	}

	v := resp.Version
	vID, _ := uuid.Parse(v.Id)
	vFileID, _ := uuid.Parse(v.FileId)
	vCreatedBy, _ := uuid.Parse(v.CreatedBy)

	return &models.DocumentFileVersion{
		ID:            vID,
		FileID:        vFileID,
		VersionNumber: int(v.VersionNumber),
		StorageKey:    v.StorageKey,
		FileSize:      v.FileSize,
		CreatedBy:     vCreatedBy,
	}, nil
}

// Ensure WOPIFileAdapter satisfies the interface at compile time.
var _ wopi.FileServiceInterface = (*WOPIFileAdapter)(nil)
