package rule

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines persistence for email rules. Every method is tenant
// scoped on both the read and the write side -- RLS is the second line of
// defence, not the first.
type Repository interface {
	Create(ctx context.Context, r *models.EmailRule) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailRule, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailRule, error)
	Update(ctx context.Context, r *models.EmailRule) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error

	// FolderBelongsToTenant reports whether a folder id is reachable from the
	// tenant's own accounts. Used to validate a move action's target.
	FolderBelongsToTenant(ctx context.Context, folderID, tenantID uuid.UUID) (bool, error)

	// ListApplyCandidates returns up to limit messages of the tenant, newest
	// first, excluding trashed ones.
	ListApplyCandidates(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EmailRuleCandidate, error)

	// ApplyToMessage writes the folder and label assignment computed for one
	// message.
	ApplyToMessage(ctx context.Context, tenantID, messageID, folderID uuid.UUID, labelIDs []uuid.UUID) error
}
