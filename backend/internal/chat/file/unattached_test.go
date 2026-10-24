package file

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTenantUploadRejectsLookalikes(t *testing.T) {
	tenant := uuid.New()
	prefix := UnattachedUploadPrefix(tenant)
	for key, want := range map[string]bool{
		prefix + uuid.New().String():                                  true,
		prefix:                                                        false,
		prefix + "../../channels/x/files/y":                           false,
		prefix + uuid.New().String() + "/extra":                       false,
		UnattachedUploadPrefix(uuid.New()) + uuid.NewString():         false,
		"channels/" + uuid.NewString() + "/files/" + uuid.NewString(): false,
	} {
		if got := IsTenantUpload(tenant, key); got != want {
			t.Errorf("IsTenantUpload(%q) = %v, want %v", key, got, want)
		}
	}
}

func TestService_UploadUnattached(t *testing.T) {
	t.Run("stores under the tenant and charges the quota", func(t *testing.T) {
		svc, repo, store, _, _ := setupTestService()
		tenant := uuid.New()

		up, err := svc.UploadUnattached(tenantCtx(tenant), UploadInput{
			Filename: "angebot.pdf", MimeType: "application/pdf", FileSize: 512,
			Reader: bytes.NewReader([]byte("pdf")), UserID: uuid.New(),
		})

		require.NoError(t, err)
		assert.True(t, IsTenantUpload(tenant, up.StorageKey), up.StorageKey)
		assert.Contains(t, store.files, up.StorageKey)
		assert.Equal(t, int64(512), repo.quota.UsedBytes)
		assert.Equal(t, "angebot.pdf", up.Filename)
	})

	t.Run("same admission checks as a channel upload", func(t *testing.T) {
		svc, _, _, _, _ := setupTestService()
		_, err := svc.UploadUnattached(tenantCtx(uuid.New()), UploadInput{
			Filename: "x.exe", MimeType: "application/x-msdownload", FileSize: 1, Reader: bytes.NewReader([]byte("x")),
		})
		assert.ErrorIs(t, err, ErrInvalidMimeType)
	})

	t.Run("store failure is returned", func(t *testing.T) {
		svc, _, store, _, _ := setupTestService()
		store.uploadErr = errors.New("minio down")
		_, err := svc.UploadUnattached(tenantCtx(uuid.New()), UploadInput{
			Filename: "a.pdf", MimeType: "application/pdf", FileSize: 1, Reader: bytes.NewReader([]byte("x")),
		})
		assert.ErrorIs(t, err, store.uploadErr)
	})
}
