package file

// Drives MinIOStore against an httptest server that speaks just enough S3
// (bucket HEAD/PUT/location, object PUT/GET/DELETE) for the real minio-go
// client to run its own request signing and retry logic. Only the network end
// is faked; every line of MinIOStore and the FileStore fallbacks runs for real.

import (
	"bufio"
	"bytes"
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeS3 struct {
	mu          sync.Mutex
	srv         *httptest.Server
	bucket      string
	bucketThere bool
	headStatus  int // overrides the bucket HEAD answer when non-zero
	putBucket   int // status for bucket creation, 0 = 200
	objects     map[string][]byte
	types       map[string]string
	deleted     []string
	madeBucket  bool
}

func newFakeS3(t *testing.T, bucket string, bucketThere bool) *fakeS3 {
	t.Helper()
	f := &fakeS3{bucket: bucket, bucketThere: bucketThere, objects: map[string][]byte{}, types: map[string]string{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeS3) host() string { return strings.TrimPrefix(f.srv.URL, "http://") }

func (f *fakeS3) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == f.bucket || path == f.bucket+"/" {
		switch {
		case r.URL.Query().Has("location"):
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<?xml version="1.0"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`)
		case r.Method == http.MethodHead:
			if f.headStatus != 0 {
				w.WriteHeader(f.headStatus)
				return
			}
			if f.bucketThere {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPut:
			if f.putBucket != 0 {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(f.putBucket)
				_, _ = io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>no</Message></Error>`)
				return
			}
			f.madeBucket = true
			f.bucketThere = true
			w.WriteHeader(http.StatusOK)
		}
		return
	}

	key := strings.TrimPrefix(path, f.bucket+"/")
	switch r.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		f.objects[key] = decodeAWSChunked(r, body)
		f.types[key] = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		data, ok := f.objects[key]
		if !ok {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("ETag", `"fake"`)
		_, _ = w.Write(data)
	case http.MethodDelete:
		f.deleted = append(f.deleted, key)
		delete(f.objects, key)
		w.WriteHeader(http.StatusNoContent)
	}
}

// decodeAWSChunked unwraps the streaming-signature body minio-go sends over
// plain HTTP; a body that is not chunk-framed is returned untouched.
func decodeAWSChunked(r *http.Request, body []byte) []byte {
	if !strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
		return body
	}
	var out bytes.Buffer
	br := bufio.NewReader(bytes.NewReader(body))
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return out.Bytes()
		}
		sizeHex := strings.TrimSpace(strings.SplitN(line, ";", 2)[0])
		n, err := strconv.ParseInt(sizeHex, 16, 64)
		if err != nil || n == 0 {
			return out.Bytes()
		}
		chunk := make([]byte, n)
		if _, err := io.ReadFull(br, chunk); err != nil {
			return out.Bytes()
		}
		out.Write(chunk)
		_, _ = br.Discard(2) // trailing CRLF
	}
}

func TestMinIOStore_ObjectRoundTrip(t *testing.T) {
	t.Parallel()
	s3 := newFakeS3(t, "files", true)

	store, err := NewMinIOStore(s3.host(), "key", "secret", "files", false)
	require.NoError(t, err)
	assert.False(t, s3.madeBucket, "an existing bucket must not be created again")

	ctx := context.Background()
	payload := []byte("hello chat file")

	require.NoError(t, store.Upload(ctx, "channels/a/b.txt", bytes.NewReader(payload), int64(len(payload)), "text/plain"))
	assert.Equal(t, payload, s3.objects["channels/a/b.txt"])
	assert.Equal(t, "text/plain", s3.types["channels/a/b.txt"])

	rc, err := store.Download(ctx, "channels/a/b.txt")
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	assert.Equal(t, payload, got)

	require.NoError(t, store.Delete(ctx, "channels/a/b.txt"))
	assert.Equal(t, []string{"channels/a/b.txt"}, s3.deleted)
	assert.NotContains(t, s3.objects, "channels/a/b.txt")
}

func TestMinIOStore_DownloadMissingObjectFailsOnRead(t *testing.T) {
	t.Parallel()
	s3 := newFakeS3(t, "files", true)
	store, err := NewMinIOStore(s3.host(), "key", "secret", "files", false)
	require.NoError(t, err)

	// minio-go GetObject is lazy: the NoSuchKey surfaces on the first read.
	rc, err := store.Download(context.Background(), "nope")
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()
	_, err = io.ReadAll(rc)
	require.Error(t, err)
}

func TestMinIOStore_UploadWithCancelledContextFails(t *testing.T) {
	t.Parallel()
	s3 := newFakeS3(t, "files", true)
	store, err := NewMinIOStore(s3.host(), "key", "secret", "files", false)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = store.Upload(ctx, "k", strings.NewReader("x"), 1, "text/plain")
	require.Error(t, err)
	assert.NotContains(t, s3.objects, "k")
}

func TestNewMinIOStore_CreatesMissingBucket(t *testing.T) {
	t.Parallel()
	s3 := newFakeS3(t, "fresh", false)

	_, err := NewMinIOStore(s3.host(), "key", "secret", "fresh", false)
	require.NoError(t, err)
	assert.True(t, s3.madeBucket, "a missing bucket must be created")
}

func TestNewMinIOStore_Errors(t *testing.T) {
	t.Parallel()

	t.Run("invalid endpoint", func(t *testing.T) {
		t.Parallel()
		_, err := NewMinIOStore("http://not a host/", "key", "secret", "files", false)
		require.Error(t, err)
	})

	t.Run("bucket check refused", func(t *testing.T) {
		t.Parallel()
		s3 := newFakeS3(t, "files", true)
		s3.headStatus = http.StatusForbidden
		_, err := NewMinIOStore(s3.host(), "key", "secret", "files", false)
		require.Error(t, err)
	})

	t.Run("bucket creation refused", func(t *testing.T) {
		t.Parallel()
		s3 := newFakeS3(t, "files", false)
		s3.putBucket = http.StatusForbidden
		_, err := NewMinIOStore(s3.host(), "key", "secret", "files", false)
		require.Error(t, err)
		assert.False(t, s3.madeBucket)
	})
}

func TestNewMinIOStoreWithPublicEndpoint_Errors(t *testing.T) {
	t.Parallel()

	t.Run("internal store fails", func(t *testing.T) {
		t.Parallel()
		s3 := newFakeS3(t, "files", true)
		s3.headStatus = http.StatusForbidden
		_, err := NewMinIOStoreWithPublicEndpoint(s3.host(), "key", "secret", "files", false, "s3.example.test", true)
		require.Error(t, err)
	})

	t.Run("invalid public endpoint", func(t *testing.T) {
		t.Parallel()
		s3 := newFakeS3(t, "files", true)
		_, err := NewMinIOStoreWithPublicEndpoint(s3.host(), "key", "secret", "files", false, "http://bad host/", false)
		require.Error(t, err)
	})
}

func TestMinIOStore_PresignedURLs(t *testing.T) {
	t.Parallel()
	s3 := newFakeS3(t, "files", true)

	t.Run("internal host without a public endpoint", func(t *testing.T) {
		t.Parallel()
		store, err := NewMinIOStoreWithPublicEndpoint(s3.host(), "key", "secret", "files", false, "", false)
		require.NoError(t, err)
		assert.Nil(t, store.publicClient)

		raw, err := store.GetPresignedURL(context.Background(), "channels/a/b.pdf", 15*time.Minute)
		require.NoError(t, err)
		u, err := url.Parse(raw)
		require.NoError(t, err)
		assert.Equal(t, s3.host(), u.Host)
		assert.Equal(t, "/files/channels/a/b.pdf", u.Path)
		assert.Equal(t, "900", u.Query().Get("X-Amz-Expires"))
		assert.NotEmpty(t, u.Query().Get("X-Amz-Signature"))
	})

	t.Run("public host for download and upload", func(t *testing.T) {
		t.Parallel()
		// Same fake, but reached under another name so the URL host proves
		// which client signed it. The public client does its own region
		// lookup against that host, so it has to be reachable here.
		publicHost := "localhost:" + strings.TrimPrefix(s3.host(), "127.0.0.1:")
		store, err := NewMinIOStoreWithPublicEndpoint(s3.host(), "key", "secret", "files", false, publicHost, false)
		require.NoError(t, err)

		down, err := store.GetPresignedURL(context.Background(), "k/get.png", time.Hour)
		require.NoError(t, err)
		du, err := url.Parse(down)
		require.NoError(t, err)
		assert.Equal(t, publicHost, du.Host)
		assert.Equal(t, "3600", du.Query().Get("X-Amz-Expires"))

		up, err := store.GetPresignedUploadURL(context.Background(), "k/put.png", 5*time.Minute)
		require.NoError(t, err)
		uu, err := url.Parse(up)
		require.NoError(t, err)
		assert.Equal(t, publicHost, uu.Host)
		assert.Equal(t, "/files/k/put.png", uu.Path)
		assert.Equal(t, "300", uu.Query().Get("X-Amz-Expires"))
	})

	t.Run("expiry beyond seven days is rejected", func(t *testing.T) {
		t.Parallel()
		store, err := NewMinIOStore(s3.host(), "key", "secret", "files", false)
		require.NoError(t, err)

		_, err = store.GetPresignedURL(context.Background(), "k", 8*24*time.Hour)
		require.Error(t, err)
		_, err = store.GetPresignedUploadURL(context.Background(), "k", 8*24*time.Hour)
		require.Error(t, err)
	})
}

func TestNewMinIOStoreOrUnavailable(t *testing.T) {
	t.Parallel()

	t.Run("reachable minio gives a real store", func(t *testing.T) {
		t.Parallel()
		s3 := newFakeS3(t, "files", true)
		store := NewMinIOStoreOrUnavailable(s3.host(), "key", "secret", "files", false)
		assert.IsType(t, &MinIOStore{}, store)
	})

	t.Run("refused minio degrades to an unavailable store", func(t *testing.T) {
		t.Parallel()
		s3 := newFakeS3(t, "files", true)
		s3.headStatus = http.StatusForbidden
		store := NewMinIOStoreOrUnavailable(s3.host(), "key", "secret", "files", false)
		assert.IsType(t, &UnavailableStore{}, store)
		assert.Error(t, store.Delete(context.Background(), "k"), "erasure must fail and be retried, not pass silently")
	})
}

func TestUnavailableStore_EveryOperationFails(t *testing.T) {
	t.Parallel()
	store := NewUnavailableStore()
	ctx := context.Background()

	assert.EqualError(t, store.Upload(ctx, "k", strings.NewReader("x"), 1, "text/plain"), "file storage unavailable")

	rc, err := store.Download(ctx, "k")
	assert.EqualError(t, err, "file storage unavailable")
	assert.Nil(t, rc)

	assert.EqualError(t, store.Delete(ctx, "k"), "file storage unavailable")

	u, err := store.GetPresignedURL(ctx, "k", time.Minute)
	assert.EqualError(t, err, "file storage unavailable")
	assert.Empty(t, u)

	u, err = store.GetPresignedUploadURL(ctx, "k", time.Minute)
	assert.EqualError(t, err, "file storage unavailable")
	assert.Empty(t, u)
}
