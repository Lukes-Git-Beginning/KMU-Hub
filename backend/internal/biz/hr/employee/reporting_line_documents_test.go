package employee

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// rolesSeenDocRepo records the roles each document read runs under: they are
// what the database layer stamps into app.user_roles for hr_document_access.
type rolesSeenDocRepo struct {
	*mockDocRepo
	seen [][]string
}

func (r *rolesSeenDocRepo) record(ctx context.Context) {
	r.seen = append(r.seen, slices.Clone(middleware.GetUserRoles(ctx)))
}

func (r *rolesSeenDocRepo) ListByEmployee(ctx context.Context, employeeID uuid.UUID) ([]*models.EmployeeDocument, error) {
	r.record(ctx)
	return r.mockDocRepo.ListByEmployee(ctx, employeeID)
}

func (r *rolesSeenDocRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.EmployeeDocument, error) {
	r.record(ctx)
	return r.mockDocRepo.ListByTenant(ctx, tenantID)
}

func (r *rolesSeenDocRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.EmployeeDocument, error) {
	r.record(ctx)
	return r.mockDocRepo.GetByID(ctx, tenantID, id)
}

func (r *rolesSeenDocRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	r.record(ctx)
	return r.mockDocRepo.Delete(ctx, tenantID, id)
}

func TestListReportingLineDocuments(t *testing.T) {
	tenantID := uuid.New()
	managerID, reportID, strangerID, noProfileID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	empRepo := newMockEmployeeRepo()
	for user, boss := range map[uuid.UUID]uuid.UUID{reportID: managerID, strangerID: uuid.New()} {
		require.NoError(t, empRepo.Create(context.Background(), &models.EmployeeProfile{
			ID: uuid.New(), TenantID: tenantID, UserID: user, ManagerUserID: &boss,
		}))
	}
	foreignTenantReport := uuid.New()
	require.NoError(t, empRepo.Create(context.Background(), &models.EmployeeProfile{
		ID: uuid.New(), TenantID: uuid.New(), UserID: foreignTenantReport, ManagerUserID: &managerID,
	}))

	docs := &rolesSeenDocRepo{mockDocRepo: newMockDocRepo()}
	doc := func(employeeID uuid.UUID, key string) *models.EmployeeDocument {
		d := &models.EmployeeDocument{ID: uuid.New(), TenantID: tenantID, EmployeeID: employeeID, CategoryKey: key}
		docs.documents = append(docs.documents, d)
		return d
	}
	reportZeugnis := doc(reportID, "zeugnis")
	reportSalary := doc(reportID, SalaryDocumentCategoryKey)
	ownSalary := doc(managerID, SalaryDocumentCategoryKey)

	svc := NewService(empRepo, newMockDocCategoryRepo(), docs)
	ctx := context.WithValue(context.Background(), middleware.UserRolesKey, []string{"member", "manager"})
	ids := func(list []*models.EmployeeDocument) []uuid.UUID {
		out := make([]uuid.UUID, 0, len(list))
		for _, d := range list {
			out = append(out, d.ID)
		}
		return out
	}
	lastRoles := func() []string { return docs.seen[len(docs.seen)-1] }

	t.Run("a report's file is read with the manager role, salary hidden on request", func(t *testing.T) {
		got, err := svc.ListReportingLineDocuments(ctx, tenantID, reportID, managerID, true)
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{reportZeugnis.ID}, ids(got))
		assert.Equal(t, []string{"member", "manager"}, lastRoles())

		got, err = svc.ListReportingLineDocuments(ctx, tenantID, reportID, managerID, false)
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{reportZeugnis.ID, reportSalary.ID}, ids(got))
	})

	t.Run("the caller's own file is read without the manager role and keeps the salary", func(t *testing.T) {
		got, err := svc.ListReportingLineDocuments(ctx, tenantID, managerID, managerID, true)
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{ownSalary.ID}, ids(got))
		assert.Equal(t, []string{"member"}, lastRoles())
	})

	t.Run("anyone outside the reporting line is refused before any document read", func(t *testing.T) {
		before := len(docs.seen)
		for _, id := range []uuid.UUID{strangerID, noProfileID, foreignTenantReport} {
			_, err := svc.ListReportingLineDocuments(ctx, tenantID, id, managerID, false)
			assert.ErrorIs(t, err, ErrInsufficientDocumentAccess, "employee %s", id)
		}
		assert.Len(t, docs.seen, before)
	})

	t.Run("every other document access drops the manager role", func(t *testing.T) {
		_, err := svc.ListEmployeeDocuments(ctx, reportID)
		require.NoError(t, err)
		assert.Equal(t, []string{"member"}, lastRoles())

		_, err = svc.ListPersonnelDocuments(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, []string{"member"}, lastRoles())

		_, err = svc.GetEmployeeDocument(ctx, tenantID, reportZeugnis.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"member"}, lastRoles())

		require.NoError(t, svc.DeleteEmployeeDocument(ctx, tenantID, reportZeugnis.ID))
		assert.Equal(t, []string{"member"}, lastRoles())
	})
}
