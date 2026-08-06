package wiki

import (
	"time"

	"github.com/google/uuid"
)

// Article represents a wiki article.
type Article struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	Title      string     `json:"title"`
	Slug       string     `json:"slug"`
	Content    []byte     `json:"content"` // raw JSONB (TipTap JSON)
	AuthorID   uuid.UUID  `json:"author_id"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	Published  bool       `json:"published"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Version represents a historic snapshot of an article.
type Version struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	ArticleID     uuid.UUID  `json:"article_id"`
	VersionNumber int        `json:"version_number"`
	Content       []byte     `json:"content"`
	ChangedBy     *uuid.UUID `json:"changed_by,omitempty"`
	ChangedAt     time.Time  `json:"changed_at"`
}

// Attachment represents a file attached to an article.
type Attachment struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	ArticleID  uuid.UUID  `json:"article_id"`
	FileRef    string     `json:"file_ref"`
	Mime       string     `json:"mime"`
	Size       int64      `json:"size"`
	UploadedBy *uuid.UUID `json:"uploaded_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Category represents a hierarchical wiki category.
type Category struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	Name      string     `json:"name"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Position  int        `json:"position"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ShareToken represents a time-limited public share link for an article.
type ShareToken struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	ArticleID   uuid.UUID  `json:"article_id"`
	Token       string     `json:"token"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
	// RevokedAt is the moment the link was cut. Nil means live. Revoked
	// tokens stay in the listing on purpose (000297): a revocation the author
	// cannot see afterwards would be a hard delete with extra steps.
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	// CreatedBy is nil for every token minted before 000297, and for one whose
	// author has since been deleted.
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
}
