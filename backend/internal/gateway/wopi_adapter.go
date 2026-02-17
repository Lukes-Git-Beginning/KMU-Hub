package gateway

import (
	"bytes"
	"context"
	"io"

	"github.com/google/uuid"

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
func (a *WOPIFileAdapter) GetByID(ctx context.Context, id uuid.UUID) (*models.DocumentFile, error) {
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
func (a *WOPIFileAdapter) GetDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error) {
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

// CreateVersion creates a new file version via gRPC.
// Note: For WOPI PutFile, the body is streamed. Since gRPC doesn't support
// streaming the body in a unary RPC, we pass metadata only. The actual file
// content save is handled by the gateway reading the body and uploading to
// MinIO directly, then calling the gRPC to record the version metadata.
// For simplicity in MVP, we return a stub version.
func (a *WOPIFileAdapter) CreateVersion(ctx context.Context, fileID uuid.UUID, input wopi.VersionInput) (*models.DocumentFileVersion, error) {
	client, err := a.getClient()
	if err != nil {
		return nil, err
	}

	// Read the body into a buffer to get the storage key
	// In production, this would upload to MinIO and get a storage key.
	// For now, we read the body and create a version record.
	body, err := io.ReadAll(input.Reader)
	if err != nil {
		return nil, err
	}
	_ = body // Content is handled by the WOPI handler's fileStore

	label := "Auto-saved via WOPI"
	if input.Label != nil {
		label = *input.Label
	}

	resp, err := client.CreateFileVersion(ctx, &documentv1.CreateFileVersionRequest{
		FileId:       fileID.String(),
		StorageKey:   "", // Will be set by service
		FileSize:     int64(len(body)),
		CreatedBy:    input.UserID.String(),
		VersionLabel: label,
	})
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

// Suppress unused import.
var _ = bytes.NewReader
