package absence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Mock repositories
// ============================================================================

type mockAbsenceRepo struct {
	entries []*AbsenceEntry
}

func (m *mockAbsenceRepo) GetAbsenceCalendar(_ context.Context, _ AbsenceFilter) ([]*AbsenceEntry, error) {
	return m.entries, nil
}

type mockSettingsRepo struct {
	settings *models.HRCompanySettings
	err      error
}

func (m *mockSettingsRepo) GetByTenant(_ context.Context, _ uuid.UUID) (*models.HRCompanySettings, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.settings, nil
}

func (m *mockSettingsRepo) Upsert(_ context.Context, _ *models.HRCompanySettings) error {
	return nil
}

// ============================================================================
// Test fixtures
// ============================================================================

var (
	testTenantID   = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	testEmployeeID = uuid.MustParse("20000000-0000-0000-0000-000000000001")
)

func testEntries() []*AbsenceEntry {
	return []*AbsenceEntry{
		{
			EmployeeID:    testEmployeeID,
			EmployeeName:  "Max Mustermann",
			Department:    "Engineering",
			LeaveTypeName: "Urlaub",
			LeaveTypeKey:  "urlaub",
			Color:         "#3d8abf",
			StartDate:     time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			EndDate:       time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
			Status:        models.LeaveStatusApproved,
		},
		{
			EmployeeID:    uuid.MustParse("20000000-0000-0000-0000-000000000002"),
			EmployeeName:  "Erika Musterfrau",
			Department:    "Marketing",
			LeaveTypeName: "Krankheit",
			LeaveTypeKey:  "krank",
			Color:         "#bf3d3d",
			StartDate:     time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
			EndDate:       time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
			Status:        models.LeaveStatusApproved,
		},
	}
}

// ============================================================================
// Tests
// ============================================================================

func TestGetAbsenceCalendar_ShowReasonTrue(t *testing.T) {
	absenceRepo := &mockAbsenceRepo{entries: testEntries()}
	settingsRepo := &mockSettingsRepo{
		settings: &models.HRCompanySettings{
			ShowAbsenceReason: true,
		},
	}

	svc := NewService(absenceRepo, settingsRepo)
	ctx := context.Background()

	entries, err := svc.GetAbsenceCalendar(ctx, testTenantID, AbsenceFilter{
		StartDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Len(t, entries, 2)

	// Verify actual leave type names and colors are preserved
	assert.Equal(t, "Urlaub", entries[0].LeaveTypeName)
	assert.Equal(t, "#3d8abf", entries[0].Color)
	assert.Equal(t, "Krankheit", entries[1].LeaveTypeName)
	assert.Equal(t, "#bf3d3d", entries[1].Color)
}

func TestGetAbsenceCalendar_ShowReasonFalse(t *testing.T) {
	absenceRepo := &mockAbsenceRepo{entries: testEntries()}
	settingsRepo := &mockSettingsRepo{
		settings: &models.HRCompanySettings{
			ShowAbsenceReason: false,
		},
	}

	svc := NewService(absenceRepo, settingsRepo)
	ctx := context.Background()

	entries, err := svc.GetAbsenceCalendar(ctx, testTenantID, AbsenceFilter{
		StartDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Len(t, entries, 2)

	// All names should be masked to "Abwesend" and colors to neutral gray
	for _, entry := range entries {
		assert.Equal(t, "Abwesend", entry.LeaveTypeName)
		assert.Equal(t, "absent", entry.LeaveTypeKey)
		assert.Equal(t, "#9ca3af", entry.Color)
	}
}

func TestGetAbsenceCalendar_InvalidDateRange(t *testing.T) {
	absenceRepo := &mockAbsenceRepo{entries: testEntries()}
	settingsRepo := &mockSettingsRepo{
		settings: &models.HRCompanySettings{ShowAbsenceReason: true},
	}

	svc := NewService(absenceRepo, settingsRepo)
	ctx := context.Background()

	_, err := svc.GetAbsenceCalendar(ctx, testTenantID, AbsenceFilter{
		StartDate: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	assert.ErrorIs(t, err, ErrInvalidDateRange)
}

func TestGetAbsenceCalendar_SettingsErrorFallsBackToShowReasons(t *testing.T) {
	absenceRepo := &mockAbsenceRepo{entries: testEntries()}
	settingsRepo := &mockSettingsRepo{
		err: assert.AnError,
	}

	svc := NewService(absenceRepo, settingsRepo)
	ctx := context.Background()

	entries, err := svc.GetAbsenceCalendar(ctx, testTenantID, AbsenceFilter{
		StartDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Len(t, entries, 2)

	// Reasons should be visible (fallback when settings fail)
	assert.Equal(t, "Urlaub", entries[0].LeaveTypeName)
	assert.Equal(t, "#3d8abf", entries[0].Color)
}
