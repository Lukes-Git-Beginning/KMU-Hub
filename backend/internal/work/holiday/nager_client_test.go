package holiday

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchHolidays_SendsCosmiUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	_, err := NewNagerClientWithURL(srv.URL).FetchHolidays(context.Background(), 2026, "DE")
	require.NoError(t, err)
	assert.Equal(t, "Cosmi/1.0", gotUA)
}
