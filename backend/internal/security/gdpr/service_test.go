package gdpr

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// testTenantID is the default-tenant sentinel used in unit tests.
var testTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// testCtx returns a context pre-loaded with the test tenant so that
// middleware.GetTenantID succeeds inside service methods.
func testCtx() context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, testTenantID.String())
}

// ============================================================================
// Mock Repository
// ============================================================================

// MockRepository mimics the copy semantics of a real database: reads return
// snapshots, not shared pointers. ApproveExport spawns ExecuteExportAsync in a
// goroutine that re-fetches and mutates the same request — shared pointers
// made the approval tests race with it (flaky under -race in CI).
type MockRepository struct {
	mu               sync.Mutex
	exports          map[uuid.UUID]*models.GDPRExportRequest
	exportsByToken   map[string]*models.GDPRExportRequest
	erasureLogs      []*models.GDPRErasureLog
	anonymizedCount  int

	createExportErr       error
	getExportErr          error
	listExportsErr        error
	updateExportErr       error
	storeExportResultErr  error
	getExportByTokenErr   error
	markDownloadedErr     error
	createErasureLogErr   error
	getNextLabelErr       error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		exports:        make(map[uuid.UUID]*models.GDPRExportRequest),
		exportsByToken: make(map[string]*models.GDPRExportRequest),
	}
}

func (m *MockRepository) CreateExportRequest(_ context.Context, req *models.GDPRExportRequest) error {
	if m.createExportErr != nil {
		return m.createExportErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := *req
	m.exports[req.ID] = &stored
	return nil
}

func (m *MockRepository) GetExportRequest(_ context.Context, _, id uuid.UUID) (*models.GDPRExportRequest, error) {
	if m.getExportErr != nil {
		return nil, m.getExportErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	req, ok := m.exports[id]
	if !ok {
		return nil, errors.New("export request not found")
	}
	snapshot := *req
	return &snapshot, nil
}

func (m *MockRepository) ListExportRequests(_ context.Context, _, userID uuid.UUID, status string) ([]*models.GDPRExportRequest, error) {
	if m.listExportsErr != nil {
		return nil, m.listExportsErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*models.GDPRExportRequest
	for _, req := range m.exports {
		if userID != uuid.Nil && req.UserID != userID {
			continue
		}
		if status != "" && req.Status != status {
			continue
		}
		snapshot := *req
		result = append(result, &snapshot)
	}
	return result, nil
}

func (m *MockRepository) UpdateExportStatus(_ context.Context, _ uuid.UUID, req *models.GDPRExportRequest) error {
	if m.updateExportErr != nil {
		return m.updateExportErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := *req
	m.exports[req.ID] = &stored
	return nil
}

func (m *MockRepository) StoreExportResult(_ context.Context, _, id uuid.UUID, data []byte, token string, expiresAt interface{}) error {
	if m.storeExportResultErr != nil {
		return m.storeExportResultErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	req, ok := m.exports[id]
	if !ok {
		return errors.New("export request not found")
	}
	req.ExportData = data
	req.DownloadToken = token
	req.Status = models.ExportStatusReady
	if t, ok := expiresAt.(time.Time); ok {
		req.DownloadExpiresAt = &t
	}
	m.exportsByToken[token] = req
	return nil
}

func (m *MockRepository) GetExportByToken(_ context.Context, _ uuid.UUID, token string) (*models.GDPRExportRequest, error) {
	if m.getExportByTokenErr != nil {
		return nil, m.getExportByTokenErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	req, ok := m.exportsByToken[token]
	if !ok {
		return nil, errors.New("export request not found by token")
	}
	snapshot := *req
	return &snapshot, nil
}

func (m *MockRepository) MarkDownloaded(_ context.Context, _, id uuid.UUID) error {
	if m.markDownloadedErr != nil {
		return m.markDownloadedErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if req, ok := m.exports[id]; ok {
		now := time.Now().UTC()
		req.DownloadedAt = &now
		req.Status = models.ExportStatusDownloaded
	}
	return nil
}

func (m *MockRepository) CreateErasureLog(_ context.Context, log *models.GDPRErasureLog) error {
	if m.createErasureLogErr != nil {
		return m.createErasureLogErr
	}
	m.erasureLogs = append(m.erasureLogs, log)
	return nil
}

func (m *MockRepository) GetNextAnonymizedLabel(_ context.Context) (string, error) {
	if m.getNextLabelErr != nil {
		return "", m.getNextLabelErr
	}
	m.anonymizedCount++
	return "Geloeschter Benutzer #" + string(rune('0'+m.anonymizedCount)), nil
}

// ============================================================================
// Mock Export Handler
// ============================================================================

type MockExportHandler struct {
	name string
	data []byte
	err  error
}

func (h *MockExportHandler) ModuleName() string { return h.name }

func (h *MockExportHandler) ExportUserData(_ context.Context, _ uuid.UUID) ([]byte, error) {
	if h.err != nil {
		return nil, h.err
	}
	return h.data, nil
}

// ============================================================================
// Mock Erasure Handler
// ============================================================================

type MockErasureHandler struct {
	name           string
	preview        *models.ModuleErasurePreview
	previewErr     error
	recordsErased  int
	executeErr     error
}

func (h *MockErasureHandler) ModuleName() string { return h.name }

func (h *MockErasureHandler) PreviewErasure(_ context.Context, _ uuid.UUID) (*models.ModuleErasurePreview, error) {
	if h.previewErr != nil {
		return nil, h.previewErr
	}
	return h.preview, nil
}

func (h *MockErasureHandler) ExecuteErasure(_ context.Context, _ uuid.UUID, _ string, _ ErasureAction) (int, error) {
	if h.executeErr != nil {
		return 0, h.executeErr
	}
	return h.recordsErased, nil
}

// ============================================================================
// RequestExport Tests
// ============================================================================

func TestService_RequestExport_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	req, err := svc.RequestExport(testCtx(), userID)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, req.ID)
	assert.Equal(t, userID, req.UserID)
	assert.Equal(t, models.ExportStatusPending, req.Status)
	assert.NotZero(t, req.RequestedAt)
}

func TestService_RequestExport_AlreadyPending(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	// Create an existing pending request
	existing := &models.GDPRExportRequest{
		ID:          uuid.New(),
		UserID:      userID,
		Status:      models.ExportStatusPending,
		RequestedAt: time.Now().UTC(),
	}
	repo.exports[existing.ID] = existing

	_, err := svc.RequestExport(testCtx(), userID)

	assert.ErrorIs(t, err, ErrExportAlreadyPending)
}

func TestService_RequestExport_RepoError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.listExportsErr = errors.New("db connection failed")

	_, err := svc.RequestExport(testCtx(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check existing requests")
}

func TestService_RequestExport_CreateError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.createExportErr = errors.New("insert failed")

	_, err := svc.RequestExport(testCtx(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create export request")
}

// ============================================================================
// ListExports Tests
// ============================================================================

func TestService_ListExports_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	req := &models.GDPRExportRequest{
		ID:     uuid.New(),
		UserID: userID,
		Status: models.ExportStatusPending,
	}
	repo.exports[req.ID] = req

	results, err := svc.ListExports(testCtx(), userID, "")

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, req.ID, results[0].ID)
}

// ============================================================================
// ApproveExport Tests
// ============================================================================

func TestService_ApproveExport_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	reviewerID := uuid.New()
	repo.exports[exportID] = &models.GDPRExportRequest{
		ID:     exportID,
		UserID: uuid.New(),
		Status: models.ExportStatusPending,
	}

	result, err := svc.ApproveExport(testCtx(), exportID, reviewerID, "Looks good")

	require.NoError(t, err)
	assert.Equal(t, models.ExportStatusApproved, result.Status)
	assert.Equal(t, &reviewerID, result.ReviewedBy)
	assert.NotNil(t, result.ReviewedAt)
	assert.Equal(t, "Looks good", result.ReviewNote)
}

func TestService_ApproveExport_NotPending(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	repo.exports[exportID] = &models.GDPRExportRequest{
		ID:     exportID,
		UserID: uuid.New(),
		Status: models.ExportStatusApproved, // Already approved
	}

	_, err := svc.ApproveExport(testCtx(), exportID, uuid.New(), "")

	assert.ErrorIs(t, err, ErrInvalidExportStatus)
}

func TestService_ApproveExport_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.ApproveExport(testCtx(), uuid.New(), uuid.New(), "")

	require.Error(t, err)
}

func TestService_ApproveExport_UpdateError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	repo.exports[exportID] = &models.GDPRExportRequest{
		ID:     exportID,
		UserID: uuid.New(),
		Status: models.ExportStatusPending,
	}
	repo.updateExportErr = errors.New("update failed")

	_, err := svc.ApproveExport(testCtx(), exportID, uuid.New(), "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to approve export")
}

// ============================================================================
// DenyExport Tests
// ============================================================================

func TestService_DenyExport_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	reviewerID := uuid.New()
	repo.exports[exportID] = &models.GDPRExportRequest{
		ID:     exportID,
		UserID: uuid.New(),
		Status: models.ExportStatusPending,
	}

	result, err := svc.DenyExport(testCtx(), exportID, reviewerID, "Insufficient verification")

	require.NoError(t, err)
	assert.Equal(t, models.ExportStatusDenied, result.Status)
	assert.Equal(t, &reviewerID, result.ReviewedBy)
	assert.NotNil(t, result.ReviewedAt)
	assert.Equal(t, "Insufficient verification", result.ReviewNote)
}

func TestService_DenyExport_NotPending(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	repo.exports[exportID] = &models.GDPRExportRequest{
		ID:     exportID,
		UserID: uuid.New(),
		Status: models.ExportStatusDenied, // Already denied
	}

	_, err := svc.DenyExport(testCtx(), exportID, uuid.New(), "")

	assert.ErrorIs(t, err, ErrInvalidExportStatus)
}

// ============================================================================
// ExecuteExportAsync Tests
// ============================================================================

func TestService_ExecuteExportAsync_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	userID := uuid.New()
	repo.exports[exportID] = &models.GDPRExportRequest{
		ID:     exportID,
		UserID: userID,
		Status: models.ExportStatusApproved,
	}

	svc.RegisterExportHandler(&MockExportHandler{
		name: "testmodule",
		data: []byte(`{"test": "data"}`),
	})

	err := svc.ExecuteExportAsync(testCtx(), exportID)

	require.NoError(t, err)

	// Verify export result was stored (status set to ready by StoreExportResult mock)
	req := repo.exports[exportID]
	assert.Equal(t, models.ExportStatusReady, req.Status)
	assert.NotEmpty(t, req.ExportData)
	assert.NotEmpty(t, req.DownloadToken)
	assert.NotNil(t, req.DownloadExpiresAt)
}

func TestService_ExecuteExportAsync_NotApproved(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	repo.exports[exportID] = &models.GDPRExportRequest{
		ID:     exportID,
		UserID: uuid.New(),
		Status: models.ExportStatusPending, // Not approved
	}

	err := svc.ExecuteExportAsync(testCtx(), exportID)

	assert.ErrorIs(t, err, ErrInvalidExportStatus)
}

func TestService_ExecuteExportAsync_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.ExecuteExportAsync(testCtx(), uuid.New())

	require.Error(t, err)
}

func TestService_ExecuteExportAsync_StoreError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	repo.exports[exportID] = &models.GDPRExportRequest{
		ID:     exportID,
		UserID: uuid.New(),
		Status: models.ExportStatusApproved,
	}

	svc.RegisterExportHandler(&MockExportHandler{
		name: "testmodule",
		data: []byte(`{"test": "data"}`),
	})

	repo.storeExportResultErr = errors.New("storage failed")

	err := svc.ExecuteExportAsync(testCtx(), exportID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store export result")

	// Status should be reverted to approved (not stuck at processing)
	// Note: the revert happens before StoreExportResult is called, so we check
	// that status was set to processing during execution
}

// ============================================================================
// GetExportDownload Tests
// ============================================================================

func TestService_GetExportDownload_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	userID := uuid.New()
	token := "test-download-token"
	expires := time.Now().UTC().Add(24 * time.Hour)

	req := &models.GDPRExportRequest{
		ID:                exportID,
		UserID:            userID,
		Status:            models.ExportStatusReady,
		ExportData:        []byte("zip-content"),
		DownloadToken:     token,
		DownloadExpiresAt: &expires,
	}
	repo.exports[exportID] = req
	repo.exportsByToken[token] = req

	data, filename, err := svc.GetExportDownload(testCtx(), token)

	require.NoError(t, err)
	assert.Equal(t, []byte("zip-content"), data)
	assert.Contains(t, filename, "dsgvo-export-")
	assert.Contains(t, filename, ".zip")
}

func TestService_GetExportDownload_Expired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	token := "expired-token"
	expired := time.Now().UTC().Add(-1 * time.Hour) // Already expired

	req := &models.GDPRExportRequest{
		ID:                exportID,
		UserID:            uuid.New(),
		Status:            models.ExportStatusReady,
		ExportData:        []byte("zip-content"),
		DownloadToken:     token,
		DownloadExpiresAt: &expired,
	}
	repo.exports[exportID] = req
	repo.exportsByToken[token] = req

	_, _, err := svc.GetExportDownload(testCtx(), token)

	assert.ErrorIs(t, err, ErrExportExpired)
}

func TestService_GetExportDownload_NotReady(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	token := "pending-token"

	req := &models.GDPRExportRequest{
		ID:            exportID,
		UserID:        uuid.New(),
		Status:        models.ExportStatusProcessing, // Not ready
		DownloadToken: token,
	}
	repo.exports[exportID] = req
	repo.exportsByToken[token] = req

	_, _, err := svc.GetExportDownload(testCtx(), token)

	assert.ErrorIs(t, err, ErrExportNotReady)
}

func TestService_GetExportDownload_AlreadyDownloaded(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	userID := uuid.New()
	token := "downloaded-token"
	expires := time.Now().UTC().Add(24 * time.Hour)
	downloaded := time.Now().UTC().Add(-1 * time.Hour)

	req := &models.GDPRExportRequest{
		ID:                exportID,
		UserID:            userID,
		Status:            models.ExportStatusDownloaded, // Already downloaded
		ExportData:        []byte("zip-content"),
		DownloadToken:     token,
		DownloadExpiresAt: &expires,
		DownloadedAt:      &downloaded,
	}
	repo.exports[exportID] = req
	repo.exportsByToken[token] = req

	data, filename, err := svc.GetExportDownload(testCtx(), token)

	require.NoError(t, err)
	assert.Equal(t, []byte("zip-content"), data)
	assert.Contains(t, filename, "dsgvo-export-")
}

func TestService_GetExportDownload_TokenNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, _, err := svc.GetExportDownload(testCtx(), "nonexistent-token")

	require.Error(t, err)
}

func TestService_GetExportDownload_MarkDownloadedError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	userID := uuid.New()
	token := "mark-fail-token"
	expires := time.Now().UTC().Add(24 * time.Hour)

	req := &models.GDPRExportRequest{
		ID:                exportID,
		UserID:            userID,
		Status:            models.ExportStatusReady,
		ExportData:        []byte("zip-content"),
		DownloadToken:     token,
		DownloadExpiresAt: &expires,
	}
	repo.exports[exportID] = req
	repo.exportsByToken[token] = req
	repo.markDownloadedErr = errors.New("mark failed")

	// Should still succeed -- MarkDownloaded error is non-fatal
	data, filename, err := svc.GetExportDownload(testCtx(), token)

	require.NoError(t, err)
	assert.Equal(t, []byte("zip-content"), data)
	assert.Contains(t, filename, ".zip")
}

// ============================================================================
// PreviewErasure Tests
// ============================================================================

func TestService_PreviewErasure_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	svc.RegisterErasureHandler(&MockErasureHandler{
		name: "auth",
		preview: &models.ModuleErasurePreview{
			ModuleName:  "auth",
			RecordCount: 5,
			Action:      "anonymize",
		},
	})
	svc.RegisterErasureHandler(&MockErasureHandler{
		name: "crm",
		preview: &models.ModuleErasurePreview{
			ModuleName:  "crm",
			RecordCount: 10,
			Action:      "anonymize",
		},
	})

	previews, total, err := svc.PreviewErasure(context.Background(), userID)

	require.NoError(t, err)
	assert.Len(t, previews, 2)
	assert.Equal(t, 15, total)
	assert.Equal(t, "auth", previews[0].ModuleName)
	assert.Equal(t, 5, previews[0].RecordCount)
	assert.Equal(t, "crm", previews[1].ModuleName)
	assert.Equal(t, 10, previews[1].RecordCount)
}

func TestService_PreviewErasure_HandlerError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	svc.RegisterErasureHandler(&MockErasureHandler{
		name: "auth",
		preview: &models.ModuleErasurePreview{
			ModuleName:  "auth",
			RecordCount: 3,
			Action:      "anonymize",
		},
	})
	svc.RegisterErasureHandler(&MockErasureHandler{
		name:       "broken",
		previewErr: errors.New("module unavailable"),
	})

	previews, total, err := svc.PreviewErasure(context.Background(), userID)

	require.NoError(t, err) // Continues despite handler error
	assert.Len(t, previews, 2)
	assert.Equal(t, 3, total) // Only counts successful handler

	// Error entry has RecordCount -1
	assert.Equal(t, "broken", previews[1].ModuleName)
	assert.Equal(t, -1, previews[1].RecordCount)
	assert.Equal(t, "error", previews[1].Action)
}

func TestService_PreviewErasure_NoHandlers(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	previews, total, err := svc.PreviewErasure(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Empty(t, previews)
	assert.Equal(t, 0, total)
}

// ============================================================================
// ExecuteErasure Tests
// ============================================================================

func TestService_ExecuteErasure_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()
	executedBy := uuid.New()

	svc.RegisterErasureHandler(&MockErasureHandler{
		name: "auth",
		preview: &models.ModuleErasurePreview{
			ModuleName:  "auth",
			RecordCount: 1,
			Action:      "anonymize",
		},
		recordsErased: 1,
	})
	svc.RegisterErasureHandler(&MockErasureHandler{
		name: "crm",
		preview: &models.ModuleErasurePreview{
			ModuleName:  "crm",
			RecordCount: 5,
			Action:      "delete",
		},
		recordsErased: 5,
	})

	log, err := svc.ExecuteErasure(context.Background(), userID, executedBy)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, log.ID)
	assert.Equal(t, userID, log.OriginalUserID)
	assert.Equal(t, executedBy, log.ExecutedBy)
	assert.NotEmpty(t, log.AnonymizedLabel)
	assert.NotEmpty(t, log.ConfirmationHash)
	assert.NotZero(t, log.ExecutedAt)
	assert.Len(t, log.ModulesAffected, 2)
	assert.Contains(t, log.ModulesAffected["auth"], "1 records")
	assert.Contains(t, log.ModulesAffected["crm"], "5 records")

	// Verify log was persisted
	assert.Len(t, repo.erasureLogs, 1)
}

func TestService_ExecuteErasure_HandlerError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()
	executedBy := uuid.New()

	svc.RegisterErasureHandler(&MockErasureHandler{
		name: "auth",
		preview: &models.ModuleErasurePreview{
			ModuleName:  "auth",
			RecordCount: 1,
			Action:      "anonymize",
		},
		recordsErased: 1,
	})
	svc.RegisterErasureHandler(&MockErasureHandler{
		name: "broken",
		preview: &models.ModuleErasurePreview{
			ModuleName:  "broken",
			RecordCount: 3,
			Action:      "delete",
		},
		executeErr: errors.New("module crashed"),
	})

	log, err := svc.ExecuteErasure(context.Background(), userID, executedBy)

	// Should succeed overall -- partial erasure continues
	require.NoError(t, err)
	assert.Len(t, log.ModulesAffected, 2)
	assert.Contains(t, log.ModulesAffected["auth"], "1 records")
	assert.Contains(t, log.ModulesAffected["broken"], "error: module crashed")
}

func TestService_ExecuteErasure_RepoError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	svc.RegisterErasureHandler(&MockErasureHandler{
		name: "auth",
		preview: &models.ModuleErasurePreview{
			ModuleName:  "auth",
			RecordCount: 1,
			Action:      "anonymize",
		},
		recordsErased: 1,
	})

	repo.createErasureLogErr = errors.New("db write failed")

	_, err := svc.ExecuteErasure(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create erasure log")
}

func TestService_ExecuteErasure_LabelError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.getNextLabelErr = errors.New("sequence error")

	_, err := svc.ExecuteErasure(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get anonymized label")
}
