package task

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/project"
)

// memoryStore is an object store that keeps keys only and presigns by echoing the key.
type memoryStore struct{ keys []string }

func (m *memoryStore) Upload(_ context.Context, key string, _ io.Reader, _ int64, _ string) error {
	m.keys = append(m.keys, key)
	return nil
}
func (m *memoryStore) Download(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}
func (m *memoryStore) Delete(context.Context, string) error { return nil }
func (m *memoryStore) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://files.test/" + key, nil
}
func (m *memoryStore) GetPresignedUploadURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

// TestTaskFileAttachFlow_RealSQL walks the desktop's "attach a file to a task" flow against the
// real repositories as kmuhub_app: upload without a channel -> storage_key under the tenant
// prefix -> attach to a (standalone) task -> download URL by the task file id. Another tenant's
// key is refused before anything is written, and another tenant cannot download the file.
func TestTaskFileAttachFlow_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Task File Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Task File Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	me := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", me)
	other := seedWorkUser(t, pool, tenantOther)
	defer testutil.CleanupRow(t, pool, "users", other)

	store := &memoryStore{}
	uploads := chatfile.NewService(chatfile.NewPostgresRepository(pool), store, &chatfile.NoOpScanner{}, chatfile.NewImagingGenerator(), 50)
	repo := NewPostgresRepository(pool)
	svc := NewService(repo, project.NewPostgresRepository(pool))
	svc.SetFileSigner(store)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	created, err := svc.Create(ctxOwn, CreateInput{TenantID: tenantOwn, Title: "with file", Priority: "high", AssigneeID: &me, CreatedBy: me})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tasks", created.ID)

	content := []byte("%PDF-1.4 test")
	upload := func(ctx context.Context, user uuid.UUID) *chatfile.UnattachedUpload {
		t.Helper()
		up, upErr := uploads.UploadUnattached(ctx, chatfile.UploadInput{
			Filename: "angebot.pdf", MimeType: "application/pdf", FileSize: int64(len(content)),
			Reader: bytes.NewReader(content), UserID: user,
		})
		if upErr != nil {
			t.Fatalf("upload without channel: %v", upErr)
		}
		return up
	}

	own := upload(ctxOwn, me)
	if !strings.HasPrefix(own.StorageKey, "tenants/"+tenantOwn.String()+"/uploads/") {
		t.Fatalf("storage key must sit under the tenant prefix, got %q", own.StorageKey)
	}

	attached := &models.TaskFile{
		ID: uuid.New(), TaskID: created.ID, Filename: own.Filename, MimeType: own.MimeType,
		FileSize: own.FileSize, StorageKey: own.StorageKey, UploadedBy: me, CreatedAt: time.Now(),
	}
	if err := svc.AttachFile(ctxOwn, tenantOwn, me, attached); err != nil {
		t.Fatalf("attach own upload to a standalone task: %v", err)
	}
	files, err := repo.ListFiles(ctxOwn, created.ID)
	if err != nil || len(files) != 1 || files[0].StorageKey != own.StorageKey {
		t.Fatalf("want the attached file listed, got %v (err %v)", files, err)
	}

	url, err := svc.GetFileDownloadURL(ctxOwn, tenantOwn, attached.ID)
	if err != nil || url != "https://files.test/"+own.StorageKey {
		t.Fatalf("download by task file id: url=%q err=%v", url, err)
	}

	// Another tenant's upload key is refused and nothing is written.
	foreign := upload(ctxOther, other)
	err = svc.AttachFile(ctxOwn, tenantOwn, me, &models.TaskFile{
		ID: uuid.New(), TaskID: created.ID, Filename: "x.pdf", MimeType: "application/pdf",
		FileSize: 1, StorageKey: foreign.StorageKey, UploadedBy: me, CreatedAt: time.Now(),
	})
	if !errors.Is(err, ErrForeignStorageKey) {
		t.Fatalf("foreign tenant key: want ErrForeignStorageKey, got %v", err)
	}
	if files, _ = repo.ListFiles(ctxOwn, created.ID); len(files) != 1 {
		t.Fatalf("a refused key must not be recorded, task has %d files", len(files))
	}

	// Another tenant cannot download the file by its id.
	if _, err = svc.GetFileDownloadURL(ctxOther, tenantOther, attached.ID); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("download from another tenant: want ErrFileNotFound, got %v", err)
	}

	// Attaching to another tenant's task id finds no task.
	err = svc.AttachFile(ctxOther, tenantOther, other, &models.TaskFile{
		ID: uuid.New(), TaskID: created.ID, Filename: "y.pdf", MimeType: "application/pdf",
		FileSize: 1, StorageKey: foreign.StorageKey, UploadedBy: other, CreatedAt: time.Now(),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("attach to another tenant's task: want ErrNotFound, got %v", err)
	}
}
