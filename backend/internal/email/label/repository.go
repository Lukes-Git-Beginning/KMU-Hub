package label

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines persistence for email labels. Every method is tenant
// scoped on both the read and the write side -- RLS is the second line of
// defence, not the first.
type Repository interface {
	Create(ctx context.Context, l *models.EmailLabel) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailLabel, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailLabel, error)
	Update(ctx context.Context, l *models.EmailLabel) error

	// Delete removes a label and, in the same transaction, strips its id from
	// every message of the tenant that still carries it. label_ids has no
	// foreign key (it is an array, not a join table), so this is the only
	// place that keeps it from going stale.
	Delete(ctx context.Context, id, tenantID uuid.UUID) error

	// LabelIDsBelongToTenant reports whether every given id is a label of the
	// tenant. Used to validate an assignment's target set before it is written
	// into a message.
	LabelIDsBelongToTenant(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (bool, error)

	// AssignToMessage replaces the label set of a message. Returns
	// ErrMessageNotFound if the message does not belong to the tenant.
	AssignToMessage(ctx context.Context, tenantID, messageID uuid.UUID, labelIDs []uuid.UUID) error
}
