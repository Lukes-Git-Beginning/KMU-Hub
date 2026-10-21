package lexware

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// failingContactService fails every write so the sync result carries the error text
// the settings page shows to the user.
type failingContactService struct {
	*mockContactService
}

func (failingContactService) CreateForSync(context.Context, *ContactSyncData, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, errors.New("boom")
}

func (failingContactService) UpdateForSync(context.Context, uuid.UUID, *ContactSyncData) error {
	return errors.New("boom")
}

func TestUpsertInboundContact_FailureTextsNameCosmi(t *testing.T) {
	lc := &LexwareContact{
		ID:     "lx-1",
		Person: &LexwareContactPerson{FirstName: "Erika", LastName: "Musterfrau"},
	}

	cases := map[string]struct {
		mapping *models.LexwareEntityMapping
		want    string
	}{
		"create": {nil, "create Cosmi contact from lexware lx-1: boom"},
		"update": {&models.LexwareEntityMapping{EntityType: "contact", KmuhubID: uuid.New(), LexwareID: "lx-1"}, "update Cosmi contact "},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &mockRepository{
				getEntityMappingByLexwareIDFn: func(context.Context, uuid.UUID, string, string) (*models.LexwareEntityMapping, error) {
					if tc.mapping == nil {
						return nil, ErrMappingNotFound
					}
					return tc.mapping, nil
				},
			}
			cs := NewContactSyncer(nil, repo, NewFieldMapper(), failingContactService{newMockContactService()})
			result := &SyncResult{}

			cs.upsertInboundContact(context.Background(), uuid.New(), uuid.New(), lc, nil, result)

			require.Equal(t, 1, result.ItemsFailed)
			require.Len(t, result.Errors, 1)
			assert.Contains(t, result.Errors[0], tc.want)
			assert.NotContains(t, result.Errors[0], "KMU Hub")
		})
	}
}
