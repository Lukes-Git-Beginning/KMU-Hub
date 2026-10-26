package attachment_test

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestStore_DownloadAndPresignRejectInvalidInput(t *testing.T) {
	store, _ := newStoreAgainstStub(t)

	rc, err := store.Download(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "failed to download attachment") {
		if rc != nil {
			_ = rc.Close()
		}
		t.Fatalf("Download with an empty key = %v, want failed-to-download error", err)
	}

	// Presigning is local, but minio-go refuses an expiry beyond seven days.
	_, err = store.GetPresignedURL(context.Background(), "email-attachments/a/1/x.pdf", 8*24*time.Hour)
	if err == nil || !strings.Contains(err.Error(), "failed to generate presigned URL") {
		t.Fatalf("GetPresignedURL with an 8 day expiry = %v, want failed-to-generate error", err)
	}
}
