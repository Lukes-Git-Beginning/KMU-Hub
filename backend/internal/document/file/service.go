package file

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// Service handles document file business logic.
type Service struct {
	repo    Repository
	store   chatfile.FileStore
	maxSize int64
	emitter EventEmitter
}

// NewService creates a new document file service.
func NewService(repo Repository, store chatfile.FileStore, maxSizeBytes int64) *Service {
	return &Service{
		repo:    repo,
		store:   store,
		maxSize: maxSizeBytes,
	}
}

// SetEventEmitter sets the optional event emitter for notification events.
func (s *Service) SetEventEmitter(emitter EventEmitter) {
	s.emitter = emitter
}

// emitEvent emits a notification event if the emitter is configured.
// Nil-safe: does nothing if no emitter is set.
func (s *Service) emitEvent(ctx context.Context, eventType, actorID, resourceID string, targetUserIDs []string, title, body, deepLink string) {
	if s.emitter == nil {
		return
	}
	payload := models.EventPayload{
		Type:          eventType,
		Priority:      "normal",
		ActorID:       actorID,
		ModuleID:      event.ModuleDocument,
		ResourceID:    resourceID,
		TargetUserIDs: targetUserIDs,
		Title:         title,
		Body:          body,
		DeepLink:      deepLink,
		Timestamp:     time.Now(),
	}
	if err := s.emitter.EmitDocumentEvent(ctx, payload); err != nil {
		slog.Error("failed to emit document event",
			"type", eventType,
			"resource_id", resourceID,
			"error", err,
		)
	}
}

// recordActivity appends an entry to the file's audit trail. Best-effort: a
// failure here (e.g. transient DB error) is logged but never fails the
// primary operation, mirroring how version-record failures are handled above.
func (s *Service) recordActivity(ctx context.Context, tenantID, fileID, actorID uuid.UUID, action, detail string) {
	activity := &models.DocumentFileActivity{
		ID:        uuid.New(),
		TenantID:  tenantID,
		FileID:    fileID,
		Action:    action,
		ActorID:   actorID,
		Detail:    detail,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateActivity(ctx, activity); err != nil {
		slog.Error("failed to record document activity",
			"action", action,
			"file_id", fileID,
			"error", err,
		)
	}
}

// LogDownload records a "downloaded" activity entry for a file. Kept separate
// from GetDownloadURL (whose signature is shared with the WOPI adapter, which
// has no actor to attribute) so only the explicit gateway download-URL flow
// logs it.
func (s *Service) LogDownload(ctx context.Context, fileID, tenantID, actorID uuid.UUID) {
	s.recordActivity(ctx, tenantID, fileID, actorID, models.DocumentActivityDownloaded, "")
}

// ListActivity returns the audit trail for a file, newest first, scoped to the tenant.
func (s *Service) ListActivity(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID) ([]*models.DocumentFileActivity, error) {
	return s.repo.ListActivity(ctx, fileID, tenantID)
}

// maxCommentLength bounds a single comment body, matching the limit already
// enforced for task comments (internal/work/comment).
const maxCommentLength = 10000

// CreateComment adds a comment to a file, authored by the given user.
//
// lean: content is trimmed and length-checked but not HTML-sanitized here —
// the desktop client already runs DOMPurify before render (same split as
// internal/work/comment). Add server-side sanitization once a non-React
// consumer (email digest, export) renders comment content as HTML.
func (s *Service) CreateComment(ctx context.Context, tenantID, fileID, authorID uuid.UUID, content string) (*models.DocumentFileComment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrCommentContentRequired
	}
	if len(content) > maxCommentLength {
		return nil, ErrCommentContentTooLong
	}

	// Verify the file exists and belongs to the tenant before attaching a comment.
	if _, err := s.repo.GetByID(ctx, fileID, tenantID); err != nil {
		return nil, err
	}

	now := time.Now()
	comment := &models.DocumentFileComment{
		ID:        uuid.New(),
		TenantID:  tenantID,
		FileID:    fileID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	slog.Info("document comment created", "comment_id", comment.ID, "file_id", fileID)
	return s.repo.GetCommentByID(ctx, comment.ID, tenantID)
}

// UpdateComment edits a comment's content. Only the author may edit.
func (s *Service) UpdateComment(ctx context.Context, tenantID, commentID, actorID uuid.UUID, content string) (*models.DocumentFileComment, error) {
	existing, err := s.repo.GetCommentByID(ctx, commentID, tenantID)
	if err != nil {
		return nil, err
	}
	if existing.AuthorID != actorID {
		return nil, ErrCannotEditOthersComment
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrCommentContentRequired
	}
	if len(content) > maxCommentLength {
		return nil, ErrCommentContentTooLong
	}

	if err := s.repo.UpdateComment(ctx, commentID, tenantID, content); err != nil {
		return nil, err
	}
	return s.repo.GetCommentByID(ctx, commentID, tenantID)
}

// DeleteComment removes a comment. Only the author or an admin may delete.
func (s *Service) DeleteComment(ctx context.Context, tenantID, commentID, actorID uuid.UUID, isAdmin bool) error {
	existing, err := s.repo.GetCommentByID(ctx, commentID, tenantID)
	if err != nil {
		return err
	}
	if existing.AuthorID != actorID && !isAdmin {
		return ErrCannotDeleteOthersComment
	}
	return s.repo.DeleteComment(ctx, commentID, tenantID)
}

// ListComments returns the comments on a file, oldest first, scoped to the tenant.
func (s *Service) ListComments(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID) ([]*models.DocumentFileComment, error) {
	return s.repo.ListComments(ctx, fileID, tenantID)
}

// Share links (external, unauthenticated read/download links)

const (
	// shareLinkTokenBytes is the entropy behind a share link. 32 bytes (256
	// bits) is not a round number picked for looks: the public route answers
	// on the token alone, so the token is the entire access control.
	shareLinkTokenBytes = 32
	// maxSharePasswordLen caps what is fed to bcrypt. bcrypt silently ignores
	// input past 72 bytes, so an uncapped field would accept two different
	// passwords as the same one.
	maxSharePasswordLen = 72
	// maxShareLinkExpiryDays bounds a link's lifetime. An external
	// read/download path that outlives anyone's memory of creating it is the
	// failure mode here.
	maxShareLinkExpiryDays = 365
	// shareLinkBcryptCost matches internal/auth and internal/berichte.
	shareLinkBcryptCost = 12
)

// CreateShareLinkInput is the request to hand out an external read/download
// link for a file.
type CreateShareLinkInput struct {
	TenantID      uuid.UUID
	FileID        uuid.UUID
	ExpiresInDays *int32 // nil or 0 = never expires
	Password      string // empty = no password
	CreatedBy     *uuid.UUID
}

// CreateShareLink issues an external read/download link for one file.
//
// The file is read first, tenant-scoped: a link may only ever be minted for a
// file the caller's own tenant owns, and that read is what proves it. Without
// it a caller could name any UUID and get back a working public link to
// another tenant's file.
func (s *Service) CreateShareLink(ctx context.Context, in CreateShareLinkInput) (*models.DocumentShareLink, error) {
	if _, err := s.repo.GetByID(ctx, in.FileID, in.TenantID); err != nil {
		return nil, err
	}

	now := time.Now()
	link := &models.DocumentShareLink{
		ID:        uuid.New(),
		TenantID:  in.TenantID,
		FileID:    in.FileID,
		CreatedBy: in.CreatedBy,
		CreatedAt: now,
	}

	if in.ExpiresInDays != nil && *in.ExpiresInDays != 0 {
		days := *in.ExpiresInDays
		if days < 0 || days > maxShareLinkExpiryDays {
			return nil, ErrShareLinkExpiryInvalid
		}
		expiry := now.AddDate(0, 0, int(days))
		link.ExpiresAt = &expiry
	}

	if in.Password != "" {
		if len(in.Password) > maxSharePasswordLen {
			return nil, ErrSharePasswordTooLong
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), shareLinkBcryptCost)
		if err != nil {
			return nil, fmt.Errorf("hash share link password: %w", err)
		}
		hashStr := string(hash)
		link.PasswordHash = &hashStr
	}

	secret, err := newShareLinkSecret()
	if err != nil {
		return nil, err
	}
	link.Token = secret

	if err := s.repo.CreateShareLink(ctx, link); err != nil {
		return nil, fmt.Errorf("create share link: %w", err)
	}

	// The secret itself is never logged: this line would otherwise be a
	// working public link sitting in the log pipeline.
	slog.Info("document share link created",
		"share_id", link.ID,
		"file_id", link.FileID,
		"tenant_id", link.TenantID,
		"has_password", link.PasswordHash != nil,
		"expires_at", link.ExpiresAt,
	)
	return link, nil
}

// ListShareLinks returns the share links for a file, newest first, scoped to
// the tenant. Proves the file is the caller's before listing, same reasoning
// as CreateShareLink.
func (s *Service) ListShareLinks(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID) ([]*models.DocumentShareLink, error) {
	if _, err := s.repo.GetByID(ctx, fileID, tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListShareLinks(ctx, fileID, tenantID)
}

// RevokeShareLink cuts a link immediately; the next redemption attempt gets
// the same generic "invalid" answer an unknown token would.
func (s *Service) RevokeShareLink(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	if err := s.repo.RevokeShareLink(ctx, id, tenantID, time.Now()); err != nil {
		return err
	}
	slog.Info("document share link revoked", "share_id", id, "tenant_id", tenantID)
	return nil
}

// SharedFileDownload is what a redeemed share link resolves to.
type SharedFileDownload struct {
	DownloadURL string
	Filename    string
	ContentType string
	FileSize    int64
}

// RedeemShareLink serves the unauthenticated public read/download.
//
// This is the only path in the service that starts without a tenant, so the
// order of the checks is the security property, not a style choice:
//
//  1. resolve the link by its secret alone — the one read that must escape
//     RLS, because which tenant may be seen is exactly what it answers;
//  2. refuse revoked and expired links before anything else;
//  3. verify the password against the stored bcrypt hash, if any;
//  4. only then re-enter tenant scope, read the one file the link names, and
//     presign its download.
//
// Unknown token, revoked link, expired link, missing password and wrong
// password all return the identical ErrShareLinkInvalid: a caller who is
// guessing has nothing here to learn from — not whether the token exists, not
// whether it is merely password-protected. That is stricter than berichte's
// ShareToken (which distinguishes "not found" from "wrong password" by HTTP
// status), a deliberate choice: a file link is handed out with no
// accompanying authenticated party who could confirm out of band that a
// password prompt, rather than a dead link, is what the visitor is looking
// at.
func (s *Service) RedeemShareLink(ctx context.Context, token, password string) (*SharedFileDownload, error) {
	link, err := s.repo.GetShareLinkByToken(ctx, token)
	if err != nil {
		return nil, ErrShareLinkInvalid
	}

	now := time.Now()
	if !link.Usable(now) {
		slog.Info("document share link refused",
			"share_id", link.ID,
			"revoked", link.RevokedAt != nil,
			"expired", link.ExpiresAt != nil && !now.Before(*link.ExpiresAt),
		)
		return nil, ErrShareLinkInvalid
	}

	if link.PasswordHash != nil {
		if password == "" {
			return nil, ErrShareLinkInvalid
		}
		// bcrypt's own comparison is the constant-time one; a byte-wise
		// comparison of the hashes here would be the timing leak.
		if bcrypt.CompareHashAndPassword([]byte(*link.PasswordHash), []byte(password)) != nil {
			slog.Warn("document share link password rejected", "share_id", link.ID)
			return nil, ErrShareLinkInvalid
		}
	}

	tenantCtx := withShareLinkTenant(ctx, link.TenantID)
	f, err := s.repo.GetByID(tenantCtx, link.FileID, link.TenantID)
	if err != nil {
		// A link whose file is gone or soft-deleted is a dead link, not a
		// server fault.
		return nil, ErrShareLinkInvalid
	}

	url, err := s.store.GetPresignedURL(tenantCtx, f.StorageKey, 1*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("presign shared file: %w", err)
	}

	// Counting the view must not fail the read the visitor already earned.
	if err := s.repo.IncrementShareLinkView(tenantCtx, link.ID, link.TenantID); err != nil {
		slog.Warn("document share link view not counted", "share_id", link.ID, "error", err)
	}

	return &SharedFileDownload{
		DownloadURL: url,
		Filename:    f.Filename,
		ContentType: f.MimeType,
		FileSize:    f.FileSize,
	}, nil
}

// withShareLinkTenant attaches a tenant resolved from a share link to the
// context, so the repository and the RLS session GUCs see it. This is the
// single place a public path in this service may set a tenant, and it is
// only ever reached with the tenant the link itself resolved to.
func withShareLinkTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
}

// newShareLinkSecret draws a link secret from crypto/rand. base64url keeps it
// usable in a URL path segment without escaping.
func newShareLinkSecret() (string, error) {
	buf := make([]byte, shareLinkTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate share link token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// UploadInput contains the data needed to upload a document file.
type UploadInput struct {
	TenantID  uuid.UUID
	FolderID  uuid.UUID
	Filename  string
	MimeType  string
	FileSize  int64
	Reader    io.Reader
	SpaceType string
	SpaceID   uuid.UUID
	OwnerID   uuid.UUID
}

// Upload uploads a file to MinIO and creates a DB record with an initial version.
func (s *Service) Upload(ctx context.Context, input UploadInput) (*models.DocumentFile, error) {
	// Validate filename
	filename := strings.TrimSpace(input.Filename)
	if filename == "" {
		return nil, ErrFilenameRequired
	}
	if len(filename) > 255 {
		return nil, ErrFilenameTooLong
	}

	// Validate file size
	if input.FileSize <= 0 {
		return nil, ErrFileSizeZero
	}
	if input.FileSize > s.maxSize {
		return nil, ErrFileTooLarge
	}

	fileID := uuid.New()
	now := time.Now()

	// Generate storage key
	storageKey := fmt.Sprintf("documents/%s/%s/%s/%s/%s",
		input.SpaceType, input.SpaceID, input.FolderID, fileID, filename)

	// Upload to object storage
	if err := s.store.Upload(ctx, storageKey, input.Reader, input.FileSize, input.MimeType); err != nil {
		return nil, err
	}

	// Create DB record
	file := &models.DocumentFile{
		ID:             fileID,
		TenantID:       input.TenantID,
		FolderID:       input.FolderID,
		Filename:       filename,
		MimeType:       input.MimeType,
		FileSize:       input.FileSize,
		StorageKey:     storageKey,
		CurrentVersion: 1,
		OwnerID:        input.OwnerID,
		IsFavorite:     false,
		IsDeleted:      false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, file); err != nil {
		// Best-effort cleanup of uploaded object
		_ = s.store.Delete(ctx, storageKey)
		return nil, err
	}

	// Create initial version record
	version := &models.DocumentFileVersion{
		ID:            uuid.New(),
		TenantID:      input.TenantID,
		FileID:        fileID,
		VersionNumber: 1,
		StorageKey:    storageKey,
		FileSize:      input.FileSize,
		CreatedBy:     input.OwnerID,
		CreatedAt:     now,
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		slog.Error("failed to create initial version record",
			"file_id", fileID,
			"error", err,
		)
	}

	slog.Info("file uploaded",
		"file_id", fileID,
		"filename", filename,
		"size", input.FileSize,
		"folder_id", input.FolderID,
		"owner_id", input.OwnerID,
	)

	s.emitEvent(ctx, event.EventDocumentUploaded, input.OwnerID.String(), fileID.String(), nil,
		"Dokument hochgeladen", filename, "/documents/"+fileID.String())
	s.recordActivity(ctx, input.TenantID, fileID, input.OwnerID, models.DocumentActivityUploaded, "")

	return file, nil
}

// RegisterInput contains metadata for a file already uploaded to object storage
// via a presigned PUT URL (browser-direct upload).
type RegisterInput struct {
	TenantID   uuid.UUID
	FolderID   uuid.UUID
	Filename   string
	MimeType   string
	FileSize   int64
	StorageKey string
	OwnerID    uuid.UUID
}

// Register records metadata for a file that the client already uploaded to
// object storage through a presigned URL. Unlike Upload it does NOT touch the
// object store — the bytes are already in MinIO at input.StorageKey. It creates
// the DB record plus the initial version pointing at that key.
func (s *Service) Register(ctx context.Context, input RegisterInput) (*models.DocumentFile, error) {
	filename := strings.TrimSpace(input.Filename)
	if filename == "" {
		return nil, ErrFilenameRequired
	}
	if len(filename) > 255 {
		return nil, ErrFilenameTooLong
	}
	if input.FileSize <= 0 {
		return nil, ErrFileSizeZero
	}
	if input.FileSize > s.maxSize {
		return nil, ErrFileTooLarge
	}
	if strings.TrimSpace(input.StorageKey) == "" {
		return nil, ErrStorageKeyMissing
	}

	fileID := uuid.New()
	now := time.Now()

	file := &models.DocumentFile{
		ID:             fileID,
		TenantID:       input.TenantID,
		FolderID:       input.FolderID,
		Filename:       filename,
		MimeType:       input.MimeType,
		FileSize:       input.FileSize,
		StorageKey:     input.StorageKey,
		CurrentVersion: 1,
		OwnerID:        input.OwnerID,
		IsFavorite:     false,
		IsDeleted:      false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, file); err != nil {
		return nil, err
	}

	version := &models.DocumentFileVersion{
		ID:            uuid.New(),
		TenantID:      input.TenantID,
		FileID:        fileID,
		VersionNumber: 1,
		StorageKey:    input.StorageKey,
		FileSize:      input.FileSize,
		CreatedBy:     input.OwnerID,
		CreatedAt:     now,
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		slog.Error("failed to create initial version record",
			"file_id", fileID,
			"error", err,
		)
	}

	slog.Info("file registered (presigned upload)",
		"file_id", fileID,
		"filename", filename,
		"size", input.FileSize,
		"folder_id", input.FolderID,
		"owner_id", input.OwnerID,
	)

	s.emitEvent(ctx, event.EventDocumentUploaded, input.OwnerID.String(), fileID.String(), nil,
		"Dokument hochgeladen", filename, "/documents/"+fileID.String())
	s.recordActivity(ctx, input.TenantID, fileID, input.OwnerID, models.DocumentActivityUploaded, "")

	return file, nil
}

// GetByID retrieves a file by ID, scoped to the tenant.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.DocumentFile, error) {
	file, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if file.IsDeleted {
		return nil, ErrFileDeleted
	}
	return file, nil
}

// List retrieves files matching the given filter.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*models.DocumentFile, int, error) {
	return s.repo.List(ctx, filter)
}

// Update updates file metadata, scoped to the tenant. actorID attributes any
// resulting "renamed"/"moved" activity entries; toggling IsFavorite alone
// records no activity (a preference, not an auditable file change).
func (s *Service) Update(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, actorID uuid.UUID, input UpdateInput) error {
	// Validate filename if changing
	if input.Filename != nil {
		name := strings.TrimSpace(*input.Filename)
		if name == "" {
			return ErrFilenameRequired
		}
		if len(name) > 255 {
			return ErrFilenameTooLong
		}
		input.Filename = &name
	}

	if err := s.repo.Update(ctx, id, tenantID, input); err != nil {
		return err
	}

	slog.Info("file updated",
		"file_id", id,
	)

	if input.Filename != nil {
		s.recordActivity(ctx, tenantID, id, actorID, models.DocumentActivityRenamed, *input.Filename)
	}
	if input.FolderID != nil {
		s.recordActivity(ctx, tenantID, id, actorID, models.DocumentActivityMoved, "")
	}

	return nil
}

// Delete soft-deletes a file, scoped to the tenant. Does NOT remove from MinIO (enables recovery).
func (s *Service) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	file, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if file.IsDeleted {
		return nil // idempotent
	}

	if err := s.repo.SoftDelete(ctx, id, tenantID); err != nil {
		return err
	}

	slog.Info("file soft-deleted",
		"file_id", id,
		"filename", file.Filename,
	)
	return nil
}

// Copy copies a file to a target folder by downloading and re-uploading the content.
// tenantID is used for the initial GetByID isolation check; the copy inherits the same tenantID.
func (s *Service) Copy(ctx context.Context, fileID, targetFolderID, userID uuid.UUID, tenantID uuid.UUID) (*models.DocumentFile, error) {
	file, err := s.repo.GetByID(ctx, fileID, tenantID)
	if err != nil {
		return nil, err
	}
	if file.IsDeleted {
		return nil, ErrFileDeleted
	}

	// Download original from MinIO
	reader, err := s.store.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	newFileID := uuid.New()
	now := time.Now()

	// Generate new storage key
	newStorageKey := fmt.Sprintf("documents/copy/%s/%s/%s",
		targetFolderID, newFileID, file.Filename)

	// Upload copy to MinIO
	if err := s.store.Upload(ctx, newStorageKey, reader, file.FileSize, file.MimeType); err != nil {
		return nil, err
	}

	// Create new DB record
	newFile := &models.DocumentFile{
		ID:             newFileID,
		TenantID:       tenantID,
		FolderID:       targetFolderID,
		Filename:       file.Filename,
		MimeType:       file.MimeType,
		FileSize:       file.FileSize,
		StorageKey:     newStorageKey,
		CurrentVersion: 1,
		OwnerID:        userID,
		IsFavorite:     false,
		IsDeleted:      false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, newFile); err != nil {
		_ = s.store.Delete(ctx, newStorageKey)
		return nil, err
	}

	// Create initial version for copy
	version := &models.DocumentFileVersion{
		ID:            uuid.New(),
		TenantID:      tenantID,
		FileID:        newFileID,
		VersionNumber: 1,
		StorageKey:    newStorageKey,
		FileSize:      file.FileSize,
		CreatedBy:     userID,
		CreatedAt:     now,
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		slog.Error("failed to create version for copied file",
			"file_id", newFileID,
			"error", err,
		)
	}

	slog.Info("file copied",
		"source_file_id", fileID,
		"new_file_id", newFileID,
		"target_folder_id", targetFolderID,
		"copied_by", userID,
	)

	s.recordActivity(ctx, tenantID, newFileID, userID, models.DocumentActivityCopied, file.Filename)

	return newFile, nil
}

// Move moves a file to a different folder (just updates folder_id, MinIO key stays the same).
// tenantID is used for isolation checks; actorID attributes the resulting "moved" activity entry.
func (s *Service) Move(ctx context.Context, fileID, targetFolderID, actorID uuid.UUID, tenantID uuid.UUID) error {
	file, err := s.repo.GetByID(ctx, fileID, tenantID)
	if err != nil {
		return err
	}
	if file.IsDeleted {
		return ErrFileDeleted
	}

	if err := s.repo.Update(ctx, fileID, tenantID, UpdateInput{FolderID: &targetFolderID}); err != nil {
		return err
	}

	slog.Info("file moved",
		"file_id", fileID,
		"from_folder_id", file.FolderID,
		"to_folder_id", targetFolderID,
	)

	s.recordActivity(ctx, tenantID, fileID, actorID, models.DocumentActivityMoved, "")

	return nil
}

// GetDownloadURL returns a presigned URL for downloading the file (1-hour expiry).
func (s *Service) GetDownloadURL(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID) (string, error) {
	file, err := s.repo.GetByID(ctx, fileID, tenantID)
	if err != nil {
		return "", err
	}
	if file.IsDeleted {
		return "", ErrFileDeleted
	}

	return s.store.GetPresignedURL(ctx, file.StorageKey, 1*time.Hour)
}

// VersionInput contains the data needed to create a new file version.
type VersionInput struct {
	Reader   io.Reader
	FileSize int64
	MimeType string
	Label    *string
	UserID   uuid.UUID
}

// CreateVersion uploads new content and creates a new version record.
// tenantID is used to scope the file lookup.
func (s *Service) CreateVersion(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID, input VersionInput) (*models.DocumentFileVersion, error) {
	file, err := s.repo.GetByID(ctx, fileID, tenantID)
	if err != nil {
		return nil, err
	}
	if file.IsDeleted {
		return nil, ErrFileDeleted
	}

	newVersionNumber := file.CurrentVersion + 1
	now := time.Now()

	// Generate version-specific storage key
	storageKey := fmt.Sprintf("documents/versions/%s/%d/%s",
		fileID, newVersionNumber, file.Filename)

	// Upload new content
	if err := s.store.Upload(ctx, storageKey, input.Reader, input.FileSize, input.MimeType); err != nil {
		return nil, err
	}

	version := &models.DocumentFileVersion{
		ID:            uuid.New(),
		TenantID:      tenantID,
		FileID:        fileID,
		VersionNumber: newVersionNumber,
		VersionLabel:  input.Label,
		StorageKey:    storageKey,
		FileSize:      input.FileSize,
		CreatedBy:     input.UserID,
		CreatedAt:     now,
	}

	if err := s.repo.CreateVersion(ctx, version); err != nil {
		_ = s.store.Delete(ctx, storageKey)
		return nil, err
	}

	// Update current version on the file
	if err := s.repo.UpdateCurrentVersion(ctx, fileID, newVersionNumber); err != nil {
		slog.Error("failed to update current version",
			"file_id", fileID,
			"version", newVersionNumber,
			"error", err,
		)
	}

	slog.Info("file version created",
		"file_id", fileID,
		"version", newVersionNumber,
		"created_by", input.UserID,
	)

	s.emitEvent(ctx, event.EventDocumentVersioned, input.UserID.String(), fileID.String(), nil,
		"Neue Dokumentversion", fmt.Sprintf("Version %d", newVersionNumber), "/documents/"+fileID.String())
	s.recordActivity(ctx, tenantID, fileID, input.UserID, models.DocumentActivityVersionCreated,
		fmt.Sprintf("Version %d", newVersionNumber))

	return version, nil
}

// ListVersions returns all versions for a file, newest first.
func (s *Service) ListVersions(ctx context.Context, fileID uuid.UUID) ([]*models.DocumentFileVersion, error) {
	return s.repo.ListVersions(ctx, fileID)
}

// RevertVersion reverts to a previous version by creating a new version with the old content.
// tenantID is used to scope the file lookup.
func (s *Service) RevertVersion(ctx context.Context, fileID uuid.UUID, versionNumber int, userID uuid.UUID, tenantID uuid.UUID) (*models.DocumentFileVersion, error) {
	file, err := s.repo.GetByID(ctx, fileID, tenantID)
	if err != nil {
		return nil, err
	}
	if file.IsDeleted {
		return nil, ErrFileDeleted
	}

	// Get the target version
	targetVersion, err := s.repo.GetVersion(ctx, fileID, versionNumber)
	if err != nil {
		return nil, err
	}

	// Download old version content from MinIO
	reader, err := s.store.Download(ctx, targetVersion.StorageKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	newVersionNumber := file.CurrentVersion + 1
	now := time.Now()

	// Upload as new version
	newStorageKey := fmt.Sprintf("documents/versions/%s/%d/%s",
		fileID, newVersionNumber, file.Filename)

	if err := s.store.Upload(ctx, newStorageKey, reader, targetVersion.FileSize, file.MimeType); err != nil {
		return nil, err
	}

	revertLabel := fmt.Sprintf("Reverted from v%d", versionNumber)
	version := &models.DocumentFileVersion{
		ID:            uuid.New(),
		TenantID:      tenantID,
		FileID:        fileID,
		VersionNumber: newVersionNumber,
		VersionLabel:  &revertLabel,
		StorageKey:    newStorageKey,
		FileSize:      targetVersion.FileSize,
		CreatedBy:     userID,
		CreatedAt:     now,
	}

	if err := s.repo.CreateVersion(ctx, version); err != nil {
		_ = s.store.Delete(ctx, newStorageKey)
		return nil, err
	}

	if err := s.repo.UpdateCurrentVersion(ctx, fileID, newVersionNumber); err != nil {
		slog.Error("failed to update current version after revert",
			"file_id", fileID,
			"version", newVersionNumber,
			"error", err,
		)
	}

	slog.Info("file version reverted",
		"file_id", fileID,
		"from_version", versionNumber,
		"new_version", newVersionNumber,
		"reverted_by", userID,
	)

	s.recordActivity(ctx, tenantID, fileID, userID, models.DocumentActivityReverted,
		fmt.Sprintf("Reverted from v%d", versionNumber))

	return version, nil
}

// LinkToEntity creates a link between a file and a CRM/PM entity.
// tenantID is used to scope the file lookup.
func (s *Service) LinkToEntity(ctx context.Context, fileID uuid.UUID, entityType string, entityID, userID uuid.UUID, tenantID uuid.UUID) error {
	if !AllowedEntityTypes[entityType] {
		return ErrInvalidEntityType
	}

	// Verify file exists and not deleted
	file, err := s.repo.GetByID(ctx, fileID, tenantID)
	if err != nil {
		return err
	}
	if file.IsDeleted {
		return ErrFileDeleted
	}

	link := &models.DocumentEntityLink{
		ID:         uuid.New(),
		TenantID:   tenantID,
		FileID:     fileID,
		EntityType: entityType,
		EntityID:   entityID,
		LinkedBy:   userID,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateEntityLink(ctx, link); err != nil {
		return err
	}

	slog.Info("file linked to entity",
		"file_id", fileID,
		"entity_type", entityType,
		"entity_id", entityID,
		"linked_by", userID,
	)

	s.emitEvent(ctx, event.EventDocumentShared, userID.String(), fileID.String(), nil,
		"Dokument geteilt", "", "/documents/"+fileID.String())
	s.recordActivity(ctx, tenantID, fileID, userID, models.DocumentActivityShared, entityType)

	return nil
}

// UnlinkFromEntity removes a link between a file and an entity, scoped to
// tenantID so a caller can never delete another tenant's link by guessing a
// link ID.
func (s *Service) UnlinkFromEntity(ctx context.Context, linkID uuid.UUID, tenantID uuid.UUID) error {
	return s.repo.DeleteEntityLink(ctx, linkID, tenantID)
}

// ListEntityLinks returns all entity links for a file, scoped to tenantID.
func (s *Service) ListEntityLinks(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID) ([]*models.DocumentEntityLink, error) {
	return s.repo.ListEntityLinks(ctx, fileID, tenantID)
}

// ListByEntity returns all files linked to the given entity (e.g. a CRM
// contact), scoped to tenantID.
func (s *Service) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID, tenantID uuid.UUID) ([]*models.DocumentFile, error) {
	if !AllowedEntityTypes[entityType] {
		return nil, ErrInvalidEntityType
	}
	return s.repo.ListFilesByEntity(ctx, entityType, entityID, tenantID)
}
