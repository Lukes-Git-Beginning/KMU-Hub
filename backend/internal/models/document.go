package models

import (
	"time"

	"github.com/google/uuid"
)

// Folder space type constants
const (
	FolderSpacePersonal = "personal"
	FolderSpaceTeam     = "team"
	FolderSpaceProject  = "project"
)

// Share permission constants
const (
	SharePermissionRead  = "read"
	SharePermissionWrite = "write"
)

// Share entity type constants
const (
	ShareEntityFile   = "file"
	ShareEntityFolder = "folder"
)

// Virtual file source constants
const (
	VirtualFileSourceChat  = "chat"
	VirtualFileSourceEmail = "email"
	VirtualFileSourceTask  = "task"
)

// DocumentFolder represents a folder in the document hierarchy.
type DocumentFolder struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	Name      string     `json:"name"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	SpaceType string     `json:"space_type"`
	SpaceID   uuid.UUID  `json:"space_id"`
	IsSystem  bool       `json:"is_system"`
	Icon      string     `json:"icon"`
	CreatedBy uuid.UUID  `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	FileCount int        `json:"file_count"` // Computed, not stored
}

// DocumentFile represents a file stored in the document system.
type DocumentFile struct {
	ID             uuid.UUID     `json:"id"`
	TenantID       uuid.UUID     `json:"tenant_id"`
	FolderID       uuid.UUID     `json:"folder_id"`
	Filename       string        `json:"filename"`
	MimeType       string        `json:"mime_type"`
	FileSize       int64         `json:"file_size"`
	StorageKey     string        `json:"storage_key"`
	ThumbnailKey   *string       `json:"thumbnail_key,omitempty"`
	CurrentVersion int           `json:"current_version"`
	OwnerID        uuid.UUID     `json:"owner_id"`
	IsFavorite     bool          `json:"is_favorite"`
	IsDeleted      bool          `json:"is_deleted"`
	ContentText    *string       `json:"content_text,omitempty"`
	Tags           []DocumentTag `json:"tags"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	DeletedAt      *time.Time    `json:"deleted_at,omitempty"`
}

// DocumentFileVersion represents a specific version of a file.
type DocumentFileVersion struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	FileID        uuid.UUID `json:"file_id"`
	VersionNumber int       `json:"version_number"`
	VersionLabel  *string   `json:"version_label,omitempty"`
	StorageKey    string    `json:"storage_key"`
	FileSize      int64     `json:"file_size"`
	CreatedBy     uuid.UUID `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

// DocumentShare represents a sharing relationship between a file/folder and a user.
type DocumentShare struct {
	ID                 uuid.UUID `json:"id"`
	TenantID           uuid.UUID `json:"tenant_id"`
	EntityType         string    `json:"entity_type"` // "file" or "folder"
	EntityID           uuid.UUID `json:"entity_id"`
	SharedWithUserID   uuid.UUID `json:"shared_with_user_id"`
	SharedWithUserName string    `json:"shared_with_user_name"` // Denormalized from JOIN
	Permission         string    `json:"permission"`            // "read" or "write"
	SharedBy           uuid.UUID `json:"shared_by"`
	SharedByName       string    `json:"shared_by_name"` // Denormalized from JOIN
	CreatedAt          time.Time `json:"created_at"`
}

// DocumentTag represents a label that can be applied to files.
type DocumentTag struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color,omitempty"`
	FileCount int       `json:"file_count"` // Computed, not stored
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// Document file activity action constants (FE contract: DocumentActivityAction).
const (
	DocumentActivityUploaded       = "uploaded"
	DocumentActivityRenamed        = "renamed"
	DocumentActivityMoved          = "moved"
	DocumentActivityCopied         = "copied"
	DocumentActivityDownloaded     = "downloaded"
	DocumentActivityShared         = "shared"
	DocumentActivityVersionCreated = "version_created"
	DocumentActivityReverted       = "reverted"
)

// DocumentFileActivity represents one append-only entry in a file's audit trail.
type DocumentFileActivity struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	FileID    uuid.UUID `json:"file_id"`
	Action    string    `json:"action"`
	ActorID   uuid.UUID `json:"actor_id"`
	ActorName string    `json:"actor_name"` // Denormalized from JOIN
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// DocumentFileComment represents a comment on a document file.
type DocumentFileComment struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	FileID     uuid.UUID `json:"file_id"`
	AuthorID   uuid.UUID `json:"author_id"`
	AuthorName string    `json:"author_name"` // Denormalized from JOIN
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// DocumentEntityLink represents a link between a file and a CRM entity.
type DocumentEntityLink struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	FileID     uuid.UUID `json:"file_id"`
	EntityType string    `json:"entity_type"` // contact, company, deal, project, task
	EntityID   uuid.UUID `json:"entity_id"`
	EntityName string    `json:"entity_name"` // Denormalized display name
	LinkedBy   uuid.UUID `json:"linked_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// DocumentShareLink is an external, unauthenticated read/download link for a
// single document file, optionally password- and expiry-protected.
type DocumentShareLink struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	FileID       uuid.UUID  `json:"file_id"`
	Token        string     `json:"token"`
	PasswordHash *string    `json:"-"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RevokedAt    *time.Time `json:"-"`
	ViewCount    int        `json:"view_count"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Usable reports whether the link still grants access at the given instant.
// Both conditions answer the same generic "invalid link" upstream; kept apart
// here only so the caller can log which one fired.
func (l *DocumentShareLink) Usable(now time.Time) bool {
	if l.RevokedAt != nil {
		return false
	}
	if l.ExpiresAt != nil && !now.Before(*l.ExpiresAt) {
		return false
	}
	return true
}

// VirtualFile represents a file from another subsystem (chat, email, task)
// surfaced in the document browser without physical copy.
type VirtualFile struct {
	ID             uuid.UUID `json:"id"`
	Filename       string    `json:"filename"`
	MimeType       string    `json:"mime_type"`
	FileSize       int64     `json:"file_size"`
	StorageKey     string    `json:"storage_key"`
	SourceType     string    `json:"source_type"` // "chat", "email", "task"
	SourceID       uuid.UUID `json:"source_id"`
	SourceName     string    `json:"source_name"`
	UploadedBy     uuid.UUID `json:"uploaded_by"`
	UploadedByName string    `json:"uploaded_by_name"`
	CreatedAt      time.Time `json:"created_at"`
}

// WOPILock represents an active WOPI lock for collaborative editing.
type WOPILock struct {
	FileID    uuid.UUID `json:"file_id"`
	LockID    string    `json:"lock_id"`
	LockedBy  uuid.UUID `json:"locked_by"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// FolderPathSegment represents a single segment in a folder breadcrumb path.
type FolderPathSegment struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
