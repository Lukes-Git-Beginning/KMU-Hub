package wopi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// fakeFileService is a hand-rolled stub of FileServiceInterface — the
// interface exists precisely to keep this package decoupled from a real
// file.Service and its own DB dependency (see the type's doc comment).
type fakeFileService struct {
	getByIDFn        func(ctx context.Context, id, tenantID uuid.UUID) (*models.DocumentFile, error)
	createVersionFn  func(ctx context.Context, fileID, tenantID uuid.UUID, input VersionInput) (*models.DocumentFileVersion, error)
	createVersionArg *VersionInput
}

func (f *fakeFileService) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.DocumentFile, error) {
	return f.getByIDFn(ctx, id, tenantID)
}

func (f *fakeFileService) GetDownloadURL(ctx context.Context, fileID, tenantID uuid.UUID) (string, error) {
	return "", nil
}

func (f *fakeFileService) CreateVersion(ctx context.Context, fileID, tenantID uuid.UUID, input VersionInput) (*models.DocumentFileVersion, error) {
	f.createVersionArg = &input
	return f.createVersionFn(ctx, fileID, tenantID, input)
}

// fakeFileStore implements chatfile.FileStore for GetFile's download path.
type fakeFileStore struct {
	downloadFn func(ctx context.Context, key string) (io.ReadCloser, error)
}

func (f *fakeFileStore) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	return nil
}
func (f *fakeFileStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return f.downloadFn(ctx, key)
}
func (f *fakeFileStore) Delete(ctx context.Context, key string) error { return nil }
func (f *fakeFileStore) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "", nil
}
func (f *fakeFileStore) GetPresignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "", nil
}

var _ chatfile.FileStore = (*fakeFileStore)(nil)

func mustToken(t *testing.T, svc *TokenService, fileID, userID, tenantID string, canWrite bool) string {
	t.Helper()
	tok, _, err := svc.Generate(fileID, userID, "Tester", tenantID, canWrite)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return tok
}

func reqWithToken(fileIDStr, token string) *http.Request {
	url := "/wopi/files/" + fileIDStr
	if token != "" {
		url += "?access_token=" + token
	}
	return httptest.NewRequest(http.MethodGet, url, nil)
}

// --- CheckFileInfo ---

func TestHandler_CheckFileInfo(t *testing.T) {
	tokenSvc := NewTokenService("test-secret")
	fileID := uuid.New()
	tenantID := uuid.New()
	ownerID := uuid.New()

	t.Run("success", func(t *testing.T) {
		fs := &fakeFileService{
			getByIDFn: func(ctx context.Context, id, tid uuid.UUID) (*models.DocumentFile, error) {
				if id != fileID || tid != tenantID {
					t.Errorf("GetByID called with id=%s tenantID=%s, want %s/%s", id, tid, fileID, tenantID)
				}
				return &models.DocumentFile{
					ID: fileID, Filename: "report.docx", FileSize: 1234,
					CurrentVersion: 3, OwnerID: ownerID,
				}, nil
			},
		}
		h := NewHandler(tokenSvc, nil, fs, nil)
		token := mustToken(t, tokenSvc, fileID.String(), "user-1", tenantID.String(), true)

		rec := httptest.NewRecorder()
		h.CheckFileInfo(rec, reqWithToken(fileID.String(), token), fileID.String())

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		var info map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if info["BaseFileName"] != "report.docx" {
			t.Errorf("BaseFileName = %v, want report.docx", info["BaseFileName"])
		}
		if info["UserCanWrite"] != true {
			t.Errorf("UserCanWrite = %v, want true", info["UserCanWrite"])
		}
		if rec.Header().Get("X-WOPI-ServerVersion") != WOPIServerVersion {
			t.Errorf("X-WOPI-ServerVersion = %q, want %q", rec.Header().Get("X-WOPI-ServerVersion"), WOPIServerVersion)
		}
	})

	t.Run("missing access_token", func(t *testing.T) {
		h := NewHandler(tokenSvc, nil, &fakeFileService{}, nil)
		rec := httptest.NewRecorder()
		h.CheckFileInfo(rec, reqWithToken(fileID.String(), ""), fileID.String())
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("invalid access_token", func(t *testing.T) {
		h := NewHandler(tokenSvc, nil, &fakeFileService{}, nil)
		rec := httptest.NewRecorder()
		h.CheckFileInfo(rec, reqWithToken(fileID.String(), "garbage"), fileID.String())
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("token file_id mismatch", func(t *testing.T) {
		h := NewHandler(tokenSvc, nil, &fakeFileService{}, nil)
		token := mustToken(t, tokenSvc, fileID.String(), "user-1", tenantID.String(), true)
		other := uuid.New().String()
		rec := httptest.NewRecorder()
		h.CheckFileInfo(rec, reqWithToken(other, token), other)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("non-uuid file_id", func(t *testing.T) {
		h := NewHandler(tokenSvc, nil, &fakeFileService{}, nil)
		token := mustToken(t, tokenSvc, "not-a-uuid", "user-1", tenantID.String(), true)
		rec := httptest.NewRecorder()
		h.CheckFileInfo(rec, reqWithToken("not-a-uuid", token), "not-a-uuid")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		fs := &fakeFileService{
			getByIDFn: func(ctx context.Context, id, tid uuid.UUID) (*models.DocumentFile, error) {
				return nil, errors.New("not found")
			},
		}
		h := NewHandler(tokenSvc, nil, fs, nil)
		token := mustToken(t, tokenSvc, fileID.String(), "user-1", tenantID.String(), true)
		rec := httptest.NewRecorder()
		h.CheckFileInfo(rec, reqWithToken(fileID.String(), token), fileID.String())
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})
}

// --- GetFile ---

func TestHandler_GetFile(t *testing.T) {
	tokenSvc := NewTokenService("test-secret")
	fileID := uuid.New()
	tenantID := uuid.New()

	t.Run("success streams content", func(t *testing.T) {
		fs := &fakeFileService{
			getByIDFn: func(ctx context.Context, id, tid uuid.UUID) (*models.DocumentFile, error) {
				return &models.DocumentFile{
					ID: fileID, MimeType: "application/pdf", CurrentVersion: 2,
					StorageKey: "documents/report.pdf",
				}, nil
			},
		}
		store := &fakeFileStore{
			downloadFn: func(ctx context.Context, key string) (io.ReadCloser, error) {
				if key != "documents/report.pdf" {
					t.Errorf("Download key = %q, want documents/report.pdf", key)
				}
				return io.NopCloser(strings.NewReader("pdf-bytes")), nil
			},
		}
		h := NewHandler(tokenSvc, nil, fs, store)
		token := mustToken(t, tokenSvc, fileID.String(), "user-1", tenantID.String(), true)

		rec := httptest.NewRecorder()
		h.GetFile(rec, reqWithToken(fileID.String(), token), fileID.String())

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if rec.Body.String() != "pdf-bytes" {
			t.Errorf("body = %q, want pdf-bytes", rec.Body.String())
		}
		if rec.Header().Get("Content-Type") != "application/pdf" {
			t.Errorf("Content-Type = %q, want application/pdf", rec.Header().Get("Content-Type"))
		}
		if rec.Header().Get("X-WOPI-ItemVersion") != "2" {
			t.Errorf("X-WOPI-ItemVersion = %q, want 2", rec.Header().Get("X-WOPI-ItemVersion"))
		}
	})

	t.Run("download failure maps to 500", func(t *testing.T) {
		fs := &fakeFileService{
			getByIDFn: func(ctx context.Context, id, tid uuid.UUID) (*models.DocumentFile, error) {
				return &models.DocumentFile{ID: fileID, StorageKey: "x"}, nil
			},
		}
		store := &fakeFileStore{
			downloadFn: func(ctx context.Context, key string) (io.ReadCloser, error) {
				return nil, errors.New("minio unreachable")
			},
		}
		h := NewHandler(tokenSvc, nil, fs, store)
		token := mustToken(t, tokenSvc, fileID.String(), "user-1", tenantID.String(), true)

		rec := httptest.NewRecorder()
		h.GetFile(rec, reqWithToken(fileID.String(), token), fileID.String())
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})

	t.Run("token file_id mismatch", func(t *testing.T) {
		h := NewHandler(tokenSvc, nil, &fakeFileService{}, &fakeFileStore{})
		token := mustToken(t, tokenSvc, fileID.String(), "user-1", tenantID.String(), true)
		other := uuid.New().String()
		rec := httptest.NewRecorder()
		h.GetFile(rec, reqWithToken(other, token), other)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})
}

// --- PutFile (requires a real *LockService — GetLock is called unconditionally) ---

func TestHandler_PutFile(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })
	lockSvc := NewLockService(pool)

	tokenSvc := NewTokenService("test-secret")
	fileID := uuid.New()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("success creates version", func(t *testing.T) {
		fs := &fakeFileService{
			getByIDFn: func(ctx context.Context, id, tid uuid.UUID) (*models.DocumentFile, error) {
				return &models.DocumentFile{ID: fileID, MimeType: "application/pdf"}, nil
			},
			createVersionFn: func(ctx context.Context, fID, tID uuid.UUID, input VersionInput) (*models.DocumentFileVersion, error) {
				return &models.DocumentFileVersion{ID: uuid.New()}, nil
			},
		}
		h := NewHandler(tokenSvc, lockSvc, fs, nil)
		token := mustToken(t, tokenSvc, fileID.String(), userID.String(), tenantID.String(), true)

		req := httptest.NewRequest(http.MethodPost, "/wopi/files/"+fileID.String()+"/contents?access_token="+token, strings.NewReader("new-bytes"))
		rec := httptest.NewRecorder()
		h.PutFile(rec, req, fileID.String())

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		if fs.createVersionArg == nil {
			t.Fatal("CreateVersion was not called")
		}
		if fs.createVersionArg.MimeType != "application/pdf" {
			t.Errorf("CreateVersion MimeType = %q, want application/pdf", fs.createVersionArg.MimeType)
		}
		if fs.createVersionArg.UserID != userID {
			t.Errorf("CreateVersion UserID = %s, want %s", fs.createVersionArg.UserID, userID)
		}
		if fs.createVersionArg.Label == nil || *fs.createVersionArg.Label != "Auto-saved via WOPI" {
			t.Errorf("CreateVersion Label = %v, want \"Auto-saved via WOPI\"", fs.createVersionArg.Label)
		}
	})

	t.Run("write permission required", func(t *testing.T) {
		h := NewHandler(tokenSvc, lockSvc, &fakeFileService{}, nil)
		token := mustToken(t, tokenSvc, fileID.String(), userID.String(), tenantID.String(), false)

		req := httptest.NewRequest(http.MethodPost, "/wopi/files/"+fileID.String()+"/contents?access_token="+token, strings.NewReader("x"))
		rec := httptest.NewRecorder()
		h.PutFile(rec, req, fileID.String())
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("invalid user id in token", func(t *testing.T) {
		h := NewHandler(tokenSvc, lockSvc, &fakeFileService{}, nil)
		token := mustToken(t, tokenSvc, fileID.String(), "not-a-uuid", tenantID.String(), true)

		req := httptest.NewRequest(http.MethodPost, "/wopi/files/"+fileID.String()+"/contents?access_token="+token, strings.NewReader("x"))
		rec := httptest.NewRecorder()
		h.PutFile(rec, req, fileID.String())
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("file not found", func(t *testing.T) {
		fs := &fakeFileService{
			getByIDFn: func(ctx context.Context, id, tid uuid.UUID) (*models.DocumentFile, error) {
				return nil, errors.New("not found")
			},
		}
		h := NewHandler(tokenSvc, lockSvc, fs, nil)
		token := mustToken(t, tokenSvc, fileID.String(), userID.String(), tenantID.String(), true)

		req := httptest.NewRequest(http.MethodPost, "/wopi/files/"+fileID.String()+"/contents?access_token="+token, strings.NewReader("x"))
		rec := httptest.NewRecorder()
		h.PutFile(rec, req, fileID.String())
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("CreateVersion failure maps to 500", func(t *testing.T) {
		fs := &fakeFileService{
			getByIDFn: func(ctx context.Context, id, tid uuid.UUID) (*models.DocumentFile, error) {
				return &models.DocumentFile{ID: fileID}, nil
			},
			createVersionFn: func(ctx context.Context, fID, tID uuid.UUID, input VersionInput) (*models.DocumentFileVersion, error) {
				return nil, errors.New("disk full")
			},
		}
		h := NewHandler(tokenSvc, lockSvc, fs, nil)
		token := mustToken(t, tokenSvc, fileID.String(), userID.String(), tenantID.String(), true)

		req := httptest.NewRequest(http.MethodPost, "/wopi/files/"+fileID.String()+"/contents?access_token="+token, strings.NewReader("x"))
		rec := httptest.NewRecorder()
		h.PutFile(rec, req, fileID.String())
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})

	// Real lock-conflict scenarios need a genuine wopi_locks row, which in
	// turn needs a real document_files row for the FK — hence a seeded
	// fixture instead of the bare fakeFileService-only fileID used above.
	t.Run("lock-conflict guard rejects a mismatched X-WOPI-Lock", func(t *testing.T) {
		fx := seedWopiLockFixture(t, pool, "putfile-lock-conflict")
		seedRealWopiLock(t, pool, fx, "held-by-someone-else", time.Now().Add(30*time.Minute))

		fs := &fakeFileService{
			getByIDFn: func(ctx context.Context, id, tid uuid.UUID) (*models.DocumentFile, error) {
				return &models.DocumentFile{ID: fx.file}, nil
			},
			createVersionFn: func(ctx context.Context, fID, tID uuid.UUID, input VersionInput) (*models.DocumentFileVersion, error) {
				t.Fatal("CreateVersion must not be called on a lock conflict")
				return nil, nil
			},
		}
		h := NewHandler(tokenSvc, lockSvc, fs, nil)
		token := mustToken(t, tokenSvc, fx.file.String(), fx.user.String(), fx.tenant.String(), true)

		req := httptest.NewRequest(http.MethodPost, "/wopi/files/"+fx.file.String()+"/contents?access_token="+token, strings.NewReader("x"))
		req.Header.Set("X-WOPI-Lock", "some-other-lock")
		rec := httptest.NewRecorder()
		h.PutFile(rec, req, fx.file.String())
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409, body=%s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("X-WOPI-Lock"); got != "held-by-someone-else" {
			t.Errorf("X-WOPI-Lock = %q, want held-by-someone-else", got)
		}
	})

	t.Run("save succeeds when X-WOPI-Lock matches the current lock", func(t *testing.T) {
		fx := seedWopiLockFixture(t, pool, "putfile-lock-match")
		seedRealWopiLock(t, pool, fx, "my-lock", time.Now().Add(30*time.Minute))

		fs := &fakeFileService{
			getByIDFn: func(ctx context.Context, id, tid uuid.UUID) (*models.DocumentFile, error) {
				return &models.DocumentFile{ID: fx.file}, nil
			},
			createVersionFn: func(ctx context.Context, fID, tID uuid.UUID, input VersionInput) (*models.DocumentFileVersion, error) {
				return &models.DocumentFileVersion{ID: uuid.New()}, nil
			},
		}
		h := NewHandler(tokenSvc, lockSvc, fs, nil)
		token := mustToken(t, tokenSvc, fx.file.String(), fx.user.String(), fx.tenant.String(), true)

		req := httptest.NewRequest(http.MethodPost, "/wopi/files/"+fx.file.String()+"/contents?access_token="+token, strings.NewReader("x"))
		req.Header.Set("X-WOPI-Lock", "my-lock")
		rec := httptest.NewRecorder()
		h.PutFile(rec, req, fx.file.String())
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
	})
}

// --- HandleLockOperation (requires a real *LockService) ---

func TestHandler_HandleLockOperation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })
	lockSvc := NewLockService(pool)

	tokenSvc := NewTokenService("test-secret")
	fx := seedWopiLockFixture(t, pool, "handle-lock-op")

	newReq := func(override, lockHeader, token string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/wopi/files/"+fx.file.String()+"?access_token="+token, nil)
		if override != "" {
			req.Header.Set("X-WOPI-Override", override)
		}
		if lockHeader != "" {
			req.Header.Set("X-WOPI-Lock", lockHeader)
		}
		return req
	}

	h := NewHandler(tokenSvc, lockSvc, &fakeFileService{}, nil)
	token := mustToken(t, tokenSvc, fx.file.String(), fx.user.String(), fx.tenant.String(), true)

	t.Run("token file_id mismatch", func(t *testing.T) {
		other := uuid.New().String()
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("LOCK", "l1", token), other)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("unsupported override", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("SOMETHING_ELSE", "l1", token), fx.file.String())
		if rec.Code != http.StatusNotImplemented {
			t.Fatalf("status = %d, want 501", rec.Code)
		}
	})

	// The subtests below are a deliberate stateful sequence against the
	// SAME file, proving a full LOCK -> conflict -> GET_LOCK -> REFRESH_LOCK
	// -> UNLOCK round trip against the real wopi_locks table.

	t.Run("LOCK acquires a new lock", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("LOCK", "lock-a", token), fx.file.String())
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("LOCK with same lockID refreshes (idempotent)", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("LOCK", "lock-a", token), fx.file.String())
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("LOCK with different lockID conflicts", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("LOCK", "lock-b", token), fx.file.String())
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409, body=%s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("X-WOPI-Lock"); got != "lock-a" {
			t.Errorf("X-WOPI-Lock = %q, want lock-a", got)
		}
	})

	t.Run("GET_LOCK reports the current lock", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("GET_LOCK", "", token), fx.file.String())
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("X-WOPI-Lock"); got != "lock-a" {
			t.Errorf("X-WOPI-Lock = %q, want lock-a", got)
		}
	})

	t.Run("REFRESH_LOCK with wrong lockID fails", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("REFRESH_LOCK", "lock-b", token), fx.file.String())
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409, body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("REFRESH_LOCK with correct lockID succeeds", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("REFRESH_LOCK", "lock-a", token), fx.file.String())
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("UNLOCK with wrong lockID fails", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("UNLOCK", "lock-b", token), fx.file.String())
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409, body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("UNLOCK with correct lockID succeeds", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("UNLOCK", "lock-a", token), fx.file.String())
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("GET_LOCK after unlock reports unlocked", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleLockOperation(rec, newReq("GET_LOCK", "", token), fx.file.String())
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("X-WOPI-Lock"); got != "" {
			t.Errorf("X-WOPI-Lock = %q, want empty", got)
		}
	})
}

func TestParseTenantIDFromClaims(t *testing.T) {
	valid := uuid.New()
	cases := []struct {
		name string
		in   string
		want uuid.UUID
	}{
		{"empty", "", uuid.Nil},
		{"invalid", "not-a-uuid", uuid.Nil},
		{"valid", valid.String(), valid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTenantIDFromClaims(&WOPITokenClaims{TenantID: tc.in})
			if got != tc.want {
				t.Errorf("parseTenantIDFromClaims(%q) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}
