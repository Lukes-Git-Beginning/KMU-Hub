package gateway

// route_wopi_test.go covers route_wopi.go (unit
// cov-gateway-crm-contact-files-wopi), which had no test file. route_wopi.go
// itself is a thin chi-routing shim -- CheckFileInfo/GetFile/PutFile/
// HandleLockOperation live in internal/document/wopi and are already
// extensively unit-tested there (handler_test.go, lock_test.go, token_test.go),
// including the three token failure cases this unit's done_when calls out:
// expired (TestTokenService_Validate_RejectsExpiredToken), tampered
// (TestTokenService_Validate_RejectsTamperedSignature), and wrong-file
// (TestHandler_CheckFileInfo/"token file_id mismatch"). What is NOT tested
// anywhere else is whether route_wopi.go's RegisterRoutes wires the
// {fileID} chi param and the four WOPI sub-paths to the right handler
// method -- that is what this file proves, end to end through a real
// router, a real TokenService and a real DB-backed LockService (WOPI routes
// bypass the standard app-auth middleware entirely, so there is no gRPC
// client boundary here the way there is for every other route file: the
// whole stack below the HTTP layer is exercisable in-process).

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/document/wopi"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// wopiRouteFakeFileService is a minimal FileServiceInterface stub -- the
// interface exists precisely so wopi.Handler doesn't need a real
// file.Service (and its own DB dependency) to be exercised.
type wopiRouteFakeFileService struct {
	file *models.DocumentFile
}

func (f *wopiRouteFakeFileService) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.DocumentFile, error) {
	if f.file == nil {
		return nil, errors.New("not found")
	}
	return f.file, nil
}

func (f *wopiRouteFakeFileService) GetDownloadURL(_ context.Context, _ uuid.UUID, _ uuid.UUID) (string, error) {
	return "", nil
}

func (f *wopiRouteFakeFileService) CreateVersion(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ wopi.VersionInput) (*models.DocumentFileVersion, error) {
	return &models.DocumentFileVersion{}, nil
}

var _ wopi.FileServiceInterface = (*wopiRouteFakeFileService)(nil)

// wopiRouteFakeFileStore implements chatfile.FileStore for GetFile's download path.
type wopiRouteFakeFileStore struct{ content string }

func (s *wopiRouteFakeFileStore) Upload(context.Context, string, io.Reader, int64, string) error {
	return nil
}
func (s *wopiRouteFakeFileStore) Download(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(s.content)), nil
}
func (s *wopiRouteFakeFileStore) Delete(context.Context, string) error { return nil }
func (s *wopiRouteFakeFileStore) GetPresignedURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}
func (s *wopiRouteFakeFileStore) GetPresignedUploadURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

var _ chatfile.FileStore = (*wopiRouteFakeFileStore)(nil)

// seedWopiRouteFixture creates the tenant/user/folder/file rows wopi_locks'
// foreign keys require (file_id -> document_files, locked_by -> users), the
// same fixture shape as internal/document/wopi/lock_test.go's
// seedWopiLockFixture -- LOCK is the one dispatch path that needs a real row
// to write against.
func seedWopiRouteFixture(t *testing.T) (tenantID, userID, fileID uuid.UUID, pool *pgxpool.Pool) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool = testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID = uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "wopi route test")

	userID = testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"email":         "wopi-route-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Wopi", "last_name": "Route",
	})

	folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "WOPI Route Test Folder",
		"space_type": "personal", "space_id": uuid.New(), "created_by": userID,
	})

	fileID = testutil.SeedRow(t, pool, "document_files", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "folder_id": folderID,
		"filename": "route-test.docx", "mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"file_size": 9, "storage_key": "documents/wopi-route-test/" + uuid.NewString(), "owner_id": userID,
	})

	return tenantID, userID, fileID, pool
}

func TestWOPIRoutes_ServiceName(t *testing.T) {
	routes := NewWOPIRoutes(nil)
	if routes.ServiceName() != "wopi" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "wopi")
	}
}

// TestWOPIRoutes_Dispatch mounts the real router and proves each of the four
// sub-paths reaches its own handler method with the {fileID} param threaded
// through correctly -- CheckFileInfo (GET /), GetFile (GET /contents),
// PutFile (POST /contents) and HandleLockOperation (POST / with
// X-WOPI-Override) are otherwise indistinguishable from outside without this
// test, since all four share the same {fileID} route segment.
func TestWOPIRoutes_Dispatch(t *testing.T) {
	tenantID, userID, fileID, pool := seedWopiRouteFixture(t)

	tokenSvc := wopi.NewTokenService("test-wopi-route-secret")
	lockSvc := wopi.NewLockService(pool)
	fileSvc := &wopiRouteFakeFileService{file: &models.DocumentFile{
		ID: fileID, Filename: "route-test.docx", FileSize: 9,
		CurrentVersion: 1, OwnerID: userID, MimeType: "text/plain",
	}}
	fileStore := &wopiRouteFakeFileStore{content: "contents"}
	handler := wopi.NewHandler(tokenSvc, lockSvc, fileSvc, fileStore)

	router := chi.NewRouter()
	NewWOPIRoutes(handler).RegisterRoutes(router, nil)

	token, _, err := tokenSvc.Generate(fileID.String(), userID.String(), "Tester", tenantID.String(), true)
	if err != nil {
		t.Fatalf("Generate token: %v", err)
	}
	base := "/wopi/files/" + fileID.String()

	t.Run("GET / dispatches to CheckFileInfo", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, base+"?access_token="+token, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "route-test.docx") {
			t.Errorf("body = %s, want it to contain the seeded filename", rec.Body.String())
		}
	})

	t.Run("GET /contents dispatches to GetFile", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, base+"/contents?access_token="+token, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		if rec.Body.String() != "contents" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "contents")
		}
	})

	t.Run("POST /contents dispatches to PutFile", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, base+"/contents?access_token="+token, strings.NewReader("new contents"))
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST / with X-WOPI-Override LOCK dispatches to HandleLockOperation", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, base+"?access_token="+token, nil)
		req.Header.Set("X-WOPI-Override", "LOCK")
		req.Header.Set("X-WOPI-Lock", "route-dispatch-lock")
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		// Unlock again so a re-run of this test (or a parallel one against
		// the same fixture) doesn't collide with a stale lock.
		unlockRec := httptest.NewRecorder()
		unlockReq := httptest.NewRequest(http.MethodPost, base+"?access_token="+token, nil)
		unlockReq.Header.Set("X-WOPI-Override", "UNLOCK")
		unlockReq.Header.Set("X-WOPI-Lock", "route-dispatch-lock")
		router.ServeHTTP(unlockRec, unlockReq)
		if unlockRec.Code != http.StatusOK {
			t.Errorf("unlock cleanup status = %d, want 200, body=%s", unlockRec.Code, unlockRec.Body.String())
		}
	})

	t.Run("wrong file_id in URL vs token is rejected through the full router", func(t *testing.T) {
		other := uuid.New().String()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/wopi/files/"+other+"?access_token="+token, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403, body=%s", rec.Code, rec.Body.String())
		}
	})
}
