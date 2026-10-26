package file

// Service paths the happy-path tests in service_test.go leave out: repository
// failures at every step, thumbnail generation during Upload, the paging
// clamps, and the asynchronous storage cleanup after Delete.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

var errBoom = errors.New("boom")

// failingRepo wraps the in-memory mock and fails selected calls.
type failingRepo struct {
	*MockRepository
	memberErr   error
	archivedErr error
	quotaErr    error
	deleteErr   error
	roleErr     error
	listErr     error
	incErr      error
	decErr      error

	listLimit, listOffset int
}

func (r *failingRepo) IsChannelMember(ctx context.Context, c, u uuid.UUID) (bool, error) {
	if r.memberErr != nil {
		return false, r.memberErr
	}
	return r.MockRepository.IsChannelMember(ctx, c, u)
}

func (r *failingRepo) IsChannelArchived(ctx context.Context, c uuid.UUID) (bool, error) {
	if r.archivedErr != nil {
		return false, r.archivedErr
	}
	return r.MockRepository.IsChannelArchived(ctx, c)
}

func (r *failingRepo) GetStorageQuota(ctx context.Context, t uuid.UUID) (*models.StorageQuota, error) {
	if r.quotaErr != nil {
		return nil, r.quotaErr
	}
	return r.MockRepository.GetStorageQuota(ctx, t)
}

func (r *failingRepo) DeleteFile(ctx context.Context, id uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	return r.MockRepository.DeleteFile(ctx, id)
}

func (r *failingRepo) GetChannelRole(ctx context.Context, c, u uuid.UUID) (models.ChannelRole, error) {
	if r.roleErr != nil {
		return "", r.roleErr
	}
	return r.MockRepository.GetChannelRole(ctx, c, u)
}

func (r *failingRepo) ListChannelFiles(ctx context.Context, c uuid.UUID, limit, offset int) ([]*models.ChatFileWithUploader, int, error) {
	r.listLimit, r.listOffset = limit, offset
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	return r.MockRepository.ListChannelFiles(ctx, c, limit, offset)
}

func (r *failingRepo) IncrementUsedBytes(ctx context.Context, t uuid.UUID, n int64) error {
	if r.incErr != nil {
		return r.incErr
	}
	return r.MockRepository.IncrementUsedBytes(ctx, t, n)
}

func (r *failingRepo) DecrementUsedBytes(ctx context.Context, t uuid.UUID, n int64) error {
	if r.decErr != nil {
		return r.decErr
	}
	return r.MockRepository.DecrementUsedBytes(ctx, t, n)
}

// recordingStore records deletes (the service issues them from a goroutine)
// and can refuse uploads of thumbnail keys.
type recordingStore struct {
	*MockFileStore
	mu          sync.Mutex
	deleted     []string
	failThumb   bool
	failDeleteK string
}

func (s *recordingStore) Upload(ctx context.Context, key string, r io.Reader, size int64, ct string) error {
	if s.failThumb && strings.HasSuffix(key, ".thumb.jpg") {
		return errBoom
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.MockFileStore.Upload(ctx, key, r, size, ct)
}

func (s *recordingStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.MockFileStore.Download(ctx, key)
}

func (s *recordingStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, key)
	if key == s.failDeleteK {
		return errBoom
	}
	return nil
}

func (s *recordingStore) deletedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.deleted...)
}

func newPathsService(t *testing.T, thumbGen ThumbnailGenerator) (*Service, *failingRepo, *recordingStore, uuid.UUID, uuid.UUID) {
	t.Helper()
	repo := &failingRepo{MockRepository: NewMockRepository()}
	store := &recordingStore{MockFileStore: NewMockFileStore()}
	svc := NewService(repo, store, &MockScanner{}, thumbGen, 50)
	channelID, userID := setupChannel(repo.MockRepository)
	return svc, repo, store, channelID, userID
}

func pngUpload(channelID, userID uuid.UUID) UploadInput {
	return UploadInput{
		ChannelID: channelID,
		Filename:  "pic.png",
		MimeType:  "image/png",
		FileSize:  4,
		Reader:    bytes.NewReader([]byte("data")),
		UserID:    userID,
	}
}

func TestService_Upload_RepositoryFailures(t *testing.T) {
	t.Parallel()
	ctx := tenantCtx(uuid.New())

	t.Run("membership lookup fails", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		repo.memberErr = errBoom
		_, err := svc.Upload(ctx, pngUpload(ch, u))
		assert.ErrorIs(t, err, errBoom)
		assert.Empty(t, store.files)
	})

	t.Run("archive lookup fails", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		repo.archivedErr = errBoom
		_, err := svc.Upload(ctx, pngUpload(ch, u))
		assert.ErrorIs(t, err, errBoom)
		assert.Empty(t, store.files)
	})

	t.Run("quota lookup fails with something other than not-found", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		repo.quotaErr = errBoom
		_, err := svc.Upload(ctx, pngUpload(ch, u))
		assert.ErrorIs(t, err, errBoom)
		assert.Empty(t, store.files, "nothing may reach storage when the quota cannot be checked")
	})

	t.Run("first upload of a tenant falls back to the default quota", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		repo.quotaErr = ErrQuotaNotFound
		got, err := svc.Upload(ctx, pngUpload(ch, u))
		require.NoError(t, err)
		assert.Equal(t, "pic.png", got.Filename)
	})

	t.Run("file larger than the default quota is refused for a fresh tenant", func(t *testing.T) {
		t.Parallel()
		repo := &failingRepo{MockRepository: NewMockRepository(), quotaErr: ErrQuotaNotFound}
		svc := NewService(repo, &recordingStore{MockFileStore: NewMockFileStore()}, &MockScanner{}, &MockThumbnailGenerator{}, 50)
		ch, u := setupChannel(repo.MockRepository)
		// The size limit (50 MB) trips first, so lift it above the 10 GB default.
		svc.maxSizeBytes = DefaultMaxQuotaBytes * 2
		in := pngUpload(ch, u)
		in.FileSize = DefaultMaxQuotaBytes + 1
		_, err := svc.Upload(ctx, in)
		assert.ErrorIs(t, err, ErrQuotaExceeded)
	})

	t.Run("missing tenant in the context", func(t *testing.T) {
		t.Parallel()
		svc, _, store, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		_, err := svc.Upload(context.Background(), pngUpload(ch, u))
		require.Error(t, err)
		assert.Empty(t, store.files)
	})

	t.Run("scanner rejection", func(t *testing.T) {
		t.Parallel()
		repo := &failingRepo{MockRepository: NewMockRepository()}
		store := &recordingStore{MockFileStore: NewMockFileStore()}
		svc := NewService(repo, store, &MockScanner{scanErr: errBoom}, &MockThumbnailGenerator{}, 50)
		ch, u := setupChannel(repo.MockRepository)
		_, err := svc.Upload(ctx, pngUpload(ch, u))
		assert.ErrorIs(t, err, ErrScanFailed)
		assert.Empty(t, store.files)
	})

	t.Run("quota counter failure does not fail the upload", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		repo.incErr = errBoom
		got, err := svc.Upload(ctx, pngUpload(ch, u))
		require.NoError(t, err)
		assert.Equal(t, "John", got.UploaderFirstName)
	})

	t.Run("record creation failure removes the stored object", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		repo.createErr = errBoom
		_, err := svc.Upload(ctx, pngUpload(ch, u))
		assert.ErrorIs(t, err, errBoom)
		require.Len(t, store.deletedKeys(), 1)
		assert.Contains(t, store.deletedKeys()[0], "/pic.png")
	})
}

func TestService_Upload_Thumbnails(t *testing.T) {
	t.Parallel()
	ctx := tenantCtx(uuid.New())

	t.Run("thumbnail is stored and referenced", func(t *testing.T) {
		t.Parallel()
		svc, _, store, ch, u := newPathsService(t, &MockThumbnailGenerator{canGen: true})
		got, err := svc.Upload(ctx, pngUpload(ch, u))
		require.NoError(t, err)

		require.NotNil(t, got.ThumbnailKey)
		assert.Equal(t, got.StorageKey+".thumb.jpg", *got.ThumbnailKey)
		assert.Equal(t, []byte("thumbnail-data"), store.files[*got.ThumbnailKey])
	})

	t.Run("generator failure keeps the upload without a thumbnail", func(t *testing.T) {
		t.Parallel()
		svc, _, store, ch, u := newPathsService(t, &MockThumbnailGenerator{canGen: true, genErr: errBoom})
		got, err := svc.Upload(ctx, pngUpload(ch, u))
		require.NoError(t, err)
		assert.Nil(t, got.ThumbnailKey)
		assert.Contains(t, store.files, got.StorageKey)
	})

	t.Run("thumbnail upload failure keeps the upload without a thumbnail", func(t *testing.T) {
		t.Parallel()
		svc, _, store, ch, u := newPathsService(t, &MockThumbnailGenerator{canGen: true})
		store.failThumb = true
		got, err := svc.Upload(ctx, pngUpload(ch, u))
		require.NoError(t, err)
		assert.Nil(t, got.ThumbnailKey)
	})

	t.Run("original upload failure stops before the record", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, ch, u := newPathsService(t, &MockThumbnailGenerator{canGen: true})
		store.uploadErr = errBoom
		_, err := svc.Upload(ctx, pngUpload(ch, u))
		assert.ErrorIs(t, err, errBoom)
		assert.Empty(t, repo.files)
	})

	t.Run("source that cannot be downloaded skips the thumbnail", func(t *testing.T) {
		t.Parallel()
		repo := &failingRepo{MockRepository: NewMockRepository()}
		// A store whose Download always fails, as when MinIO drops between PUT and GET.
		store := &downloadFailStore{recordingStore: &recordingStore{MockFileStore: NewMockFileStore()}}
		svc := NewService(repo, store, &MockScanner{}, &MockThumbnailGenerator{canGen: true}, 50)
		ch, u := setupChannel(repo.MockRepository)
		got, err := svc.Upload(ctx, pngUpload(ch, u))
		require.NoError(t, err)
		assert.Nil(t, got.ThumbnailKey)
	})
}

type downloadFailStore struct{ *recordingStore }

func (s *downloadFailStore) Download(context.Context, string) (io.ReadCloser, error) {
	return nil, errBoom
}

func TestService_GetThumbnailURL_Guards(t *testing.T) {
	t.Parallel()
	thumb := "k/a.thumb.jpg"

	t.Run("deleted file", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, ThumbnailKey: &thumb, IsDeleted: true})
		_, err := svc.GetThumbnailURL(context.Background(), id, u)
		assert.ErrorIs(t, err, ErrFileDeleted)
	})

	t.Run("unknown file", func(t *testing.T) {
		t.Parallel()
		svc, _, _, _, u := newPathsService(t, &MockThumbnailGenerator{})
		_, err := svc.GetThumbnailURL(context.Background(), uuid.New(), u)
		assert.ErrorIs(t, err, ErrFileNotFound)
	})

	t.Run("outsider is refused", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, _ := newPathsService(t, &MockThumbnailGenerator{})
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, ThumbnailKey: &thumb})
		url, err := svc.GetThumbnailURL(context.Background(), id, uuid.New())
		assert.ErrorIs(t, err, ErrNotChannelMember)
		assert.Empty(t, url)
	})

	t.Run("membership lookup fails", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, ThumbnailKey: &thumb})
		repo.memberErr = errBoom
		_, err := svc.GetThumbnailURL(context.Background(), id, u)
		assert.ErrorIs(t, err, errBoom)
	})

	t.Run("presigns the thumbnail key, not the original", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, StorageKey: "k/a", ThumbnailKey: &thumb})
		url, err := svc.GetThumbnailURL(context.Background(), id, u)
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(url, "k/a.thumb.jpg"), url)
	})
}

func TestService_GetDownloadURL_Guards(t *testing.T) {
	t.Parallel()

	t.Run("membership lookup fails", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, StorageKey: "k/a"})
		repo.memberErr = errBoom
		_, err := svc.GetDownloadURL(context.Background(), id, u)
		assert.ErrorIs(t, err, errBoom)
	})
}

func TestService_ListChannelFiles_PagingClamps(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		page, pageSize int
		wantLimit      int
		wantOffset     int
	}{
		{"defaults for zero values", 0, 0, 20, 0},
		{"negative page is page one", -3, 10, 10, 0},
		{"oversized page size falls back to 20", 2, 101, 20, 20},
		{"upper bound of 100 is accepted", 1, 100, 100, 0},
		{"third page of 10", 3, 10, 10, 20},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc, repo, _, ch, u := newPathsService(t, &MockThumbnailGenerator{})
			_, _, err := svc.ListChannelFiles(context.Background(), ch, u, tc.page, tc.pageSize)
			require.NoError(t, err)
			assert.Equal(t, tc.wantLimit, repo.listLimit)
			assert.Equal(t, tc.wantOffset, repo.listOffset)
		})
	}

	t.Run("outsider is refused before any listing", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, _ := newPathsService(t, &MockThumbnailGenerator{})
		files, total, err := svc.ListChannelFiles(context.Background(), ch, uuid.New(), 1, 10)
		assert.ErrorIs(t, err, ErrNotChannelMember)
		assert.Nil(t, files)
		assert.Zero(t, total)
		assert.Zero(t, repo.listLimit, "the repository must not be queried for a non-member")
	})

	t.Run("membership lookup fails", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		repo.memberErr = errBoom
		_, _, err := svc.ListChannelFiles(context.Background(), ch, u, 1, 10)
		assert.ErrorIs(t, err, errBoom)
	})

	t.Run("listing failure is passed on", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		repo.listErr = errBoom
		_, _, err := svc.ListChannelFiles(context.Background(), ch, u, 1, 10)
		assert.ErrorIs(t, err, errBoom)
	})
}

func TestService_Delete_Paths(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("role lookup failure means not authorized", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, UploadedBy: uuid.New(), StorageKey: "k/a"})
		repo.roleErr = errBoom
		err := svc.Delete(ctx, id, u)
		assert.ErrorIs(t, err, ErrNotAuthorized)
		assert.False(t, repo.files[id].IsDeleted)
		assert.Empty(t, store.deletedKeys())
	})

	t.Run("non-member cannot delete someone else's file", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, _ := newPathsService(t, &MockThumbnailGenerator{})
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, UploadedBy: uuid.New(), StorageKey: "k/a"})
		err := svc.Delete(ctx, id, uuid.New())
		assert.ErrorIs(t, err, ErrNotAuthorized)
		assert.False(t, repo.files[id].IsDeleted)
	})

	t.Run("database failure leaves quota and storage untouched", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		repo.quota.UsedBytes = 900
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, UploadedBy: u, StorageKey: "k/a", FileSize: 100})
		repo.deleteErr = errBoom
		err := svc.Delete(ctx, id, u)
		assert.ErrorIs(t, err, errBoom)
		assert.Equal(t, int64(900), repo.quota.UsedBytes)
		assert.Never(t, func() bool { return len(store.deletedKeys()) > 0 }, 50*time.Millisecond, 10*time.Millisecond)
	})

	t.Run("object and thumbnail are removed from storage", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		thumb := "k/a.thumb.jpg"
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, UploadedBy: u, StorageKey: "k/a", ThumbnailKey: &thumb, FileSize: 10})
		require.NoError(t, svc.Delete(ctx, id, u))
		assert.Eventually(t, func() bool { return len(store.deletedKeys()) == 2 }, time.Second, 5*time.Millisecond)
		assert.ElementsMatch(t, []string{"k/a", thumb}, store.deletedKeys())
	})

	t.Run("storage failure on the object still tries the thumbnail", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		store.failDeleteK = "k/a"
		thumb := "k/a.thumb.jpg"
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, UploadedBy: u, StorageKey: "k/a", ThumbnailKey: &thumb})
		require.NoError(t, svc.Delete(ctx, id, u))
		assert.Eventually(t, func() bool { return len(store.deletedKeys()) == 2 }, time.Second, 5*time.Millisecond)
		assert.True(t, repo.files[id].IsDeleted, "the soft delete stands even when storage cleanup fails")
	})

	t.Run("quota decrement failure does not fail the delete", func(t *testing.T) {
		t.Parallel()
		svc, repo, _, ch, u := newPathsService(t, &MockThumbnailGenerator{})
		id := uuid.New()
		repo.AddFile(&models.ChatFile{ID: id, ChannelID: ch, UploadedBy: u, StorageKey: "k/a", FileSize: 10})
		repo.decErr = errBoom
		require.NoError(t, svc.Delete(ctx, id, u))
		assert.True(t, repo.files[id].IsDeleted)
	})
}
