package attachment_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/kmuhub/kmuhub/internal/email/attachment"
)

// s3Stub is the smallest S3 that Store talks to: it records what arrived and
// serves the object back. Region is pinned in the client so no bucket-location
// lookup precedes a request.
type s3Stub struct {
	mu          sync.Mutex
	objects     map[string][]byte
	contentType map[string]string
	deleted     []string
}

func newStoreAgainstStub(t *testing.T) (*attachment.Store, *s3Stub) {
	t.Helper()

	stub := &s3Stub{objects: map[string][]byte{}, contentType: map[string]string{}}
	srv := httptest.NewServer(http.HandlerFunc(stub.serve))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse stub url: %v", err)
	}
	client, err := minio.New(u.Host, &minio.Options{
		Creds:        credentials.NewStaticV4("test-access", "test-secret", ""),
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		t.Fatalf("minio client: %v", err)
	}
	return attachment.NewStore(client, "mail"), stub
}

func (s *s3Stub) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimPrefix(r.URL.Path, "/")
	switch r.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		// Over plain HTTP minio-go signs the body as aws-chunked frames.
		if strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
			body = decodeAWSChunked(body)
		}
		s.objects[key] = body
		s.contentType[key] = r.Header.Get("Content-Type")
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		body, ok := s.objects[key]
		if !ok {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `<?xml version="1.0"?><Error><Code>NoSuchKey</Code><Message>nope</Message></Error>`)
			return
		}
		w.Header().Set("Content-Type", s.contentType[key])
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("ETag", `"abc"`)
		_, _ = w.Write(body)
	case http.MethodDelete:
		s.deleted = append(s.deleted, key)
		delete(s.objects, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// decodeAWSChunked strips the aws-chunked framing (a hex size and signature
// line, the data, a CRLF) and returns the concatenated payload.
func decodeAWSChunked(raw []byte) []byte {
	var out []byte
	for len(raw) > 0 {
		header, rest, ok := strings.Cut(string(raw), "\r\n")
		if !ok {
			break
		}
		sizeHex, _, _ := strings.Cut(header, ";")
		size, err := strconv.ParseInt(sizeHex, 16, 64)
		if err != nil || size == 0 {
			break
		}
		out = append(out, rest[:size]...)
		raw = []byte(rest[size+2:])
	}
	return out
}

func (s *s3Stub) object(key string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	body, ok := s.objects[key]
	return body, ok
}

func TestStore_UploadWritesUnderAccountScopedKey(t *testing.T) {
	store, stub := newStoreAgainstStub(t)
	accountID := uuid.New()
	payload := "%PDF-1.7 rechnung"

	key, err := store.Upload(context.Background(), accountID, 42, "rechnung.pdf",
		strings.NewReader(payload), int64(len(payload)), "application/pdf")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	wantKey := "email-attachments/" + accountID.String() + "/42/rechnung.pdf"
	if key != wantKey {
		t.Fatalf("key = %q, want %q", key, wantKey)
	}
	stored, ok := stub.object("mail/" + wantKey)
	if !ok || string(stored) != payload {
		t.Fatalf("object at mail/%s = %q (present=%v), want %q", wantKey, stored, ok, payload)
	}
	stub.mu.Lock()
	gotType := stub.contentType["mail/"+wantKey]
	stub.mu.Unlock()
	if gotType != "application/pdf" {
		t.Fatalf("stored Content-Type = %q, want application/pdf", gotType)
	}
}

func TestStore_UploadSameFilenameOfDifferentMessagesDoesNotCollide(t *testing.T) {
	store, stub := newStoreAgainstStub(t)
	accountID := uuid.New()

	k1, err := store.Upload(context.Background(), accountID, 1, "scan.pdf", strings.NewReader("eins"), 4, "application/pdf")
	if err != nil {
		t.Fatalf("Upload 1: %v", err)
	}
	k2, err := store.Upload(context.Background(), accountID, 2, "scan.pdf", strings.NewReader("zwei"), 4, "application/pdf")
	if err != nil {
		t.Fatalf("Upload 2: %v", err)
	}

	if k1 == k2 {
		t.Fatalf("both messages got key %q, the second upload would overwrite the first", k1)
	}
	b1, _ := stub.object("mail/" + k1)
	b2, _ := stub.object("mail/" + k2)
	if string(b1) != "eins" || string(b2) != "zwei" {
		t.Fatalf("objects = %q / %q, want eins / zwei", b1, b2)
	}
}

func TestStore_UploadFailureIsReportedWithoutKey(t *testing.T) {
	store, _ := newStoreAgainstStub(t)

	// A cancelled context makes PutObject fail before any byte is sent.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	key, err := store.Upload(ctx, uuid.New(), 1, "x.pdf", strings.NewReader("x"), 1, "application/pdf")
	if err == nil {
		t.Fatal("Upload with a cancelled context succeeded, want error")
	}
	if key != "" {
		t.Fatalf("failed Upload returned key %q, want empty", key)
	}
	if !strings.Contains(err.Error(), "failed to upload attachment") {
		t.Fatalf("error %q does not name the failed operation", err)
	}
}

func TestStore_DownloadStreamsStoredBytes(t *testing.T) {
	store, _ := newStoreAgainstStub(t)
	key, err := store.Upload(context.Background(), uuid.New(), 5, "notiz.txt", strings.NewReader("Hallo Welt"), 10, "text/plain")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	rc, err := store.Download(context.Background(), key)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer func() { _ = rc.Close() }()

	body, err := io.ReadAll(rc)
	if err != nil || string(body) != "Hallo Welt" {
		t.Fatalf("downloaded %q, %v, want Hallo Welt", body, err)
	}
}

func TestStore_GetPresignedURLTargetsKeyAndHonoursExpiry(t *testing.T) {
	store, _ := newStoreAgainstStub(t)
	key := "email-attachments/acc/9/angebot.pdf"

	raw, err := store.GetPresignedURL(context.Background(), key, 0)
	if err != nil {
		t.Fatalf("GetPresignedURL default: %v", err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse presigned url: %v", err)
	}
	if u.Path != "/mail/"+key {
		t.Fatalf("path = %q, want /mail/%s", u.Path, key)
	}
	// The zero expiry falls back to one hour.
	if got := u.Query().Get("X-Amz-Expires"); got != "3600" {
		t.Fatalf("default X-Amz-Expires = %q, want 3600", got)
	}

	raw, err = store.GetPresignedURL(context.Background(), key, 5*time.Minute)
	if err != nil {
		t.Fatalf("GetPresignedURL 5m: %v", err)
	}
	u, err = url.Parse(raw)
	if err != nil {
		t.Fatalf("parse presigned url: %v", err)
	}
	if got := u.Query().Get("X-Amz-Expires"); got != "300" {
		t.Fatalf("explicit X-Amz-Expires = %q, want 300", got)
	}
}

func TestStore_DeleteRemovesTheObject(t *testing.T) {
	store, stub := newStoreAgainstStub(t)
	key, err := store.Upload(context.Background(), uuid.New(), 8, "weg.pdf", strings.NewReader("x"), 1, "application/pdf")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, still := stub.object("mail/" + key); still {
		t.Fatalf("object mail/%s still stored after Delete", key)
	}
	stub.mu.Lock()
	deleted := append([]string(nil), stub.deleted...)
	stub.mu.Unlock()
	if len(deleted) != 1 || deleted[0] != "mail/"+key {
		t.Fatalf("deleted = %v, want exactly [mail/%s]", deleted, key)
	}
}

func TestStore_DeleteFailureIsReported(t *testing.T) {
	store, _ := newStoreAgainstStub(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.Delete(ctx, "email-attachments/acc/1/x.pdf")
	if err == nil || !strings.Contains(err.Error(), "failed to delete attachment") {
		t.Fatalf("Delete with a cancelled context = %v, want failed-to-delete error", err)
	}
}
