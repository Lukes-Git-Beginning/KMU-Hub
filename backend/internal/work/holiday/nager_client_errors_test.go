package holiday

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNagerClient_UsesPublicBaseURL(t *testing.T) {
	c := NewNagerClient()
	assert.Equal(t, defaultBaseURL, c.baseURL)
	assert.NotZero(t, c.httpClient.Timeout)
}

func TestFetchHolidays_RequestPathAndDecode(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[{"date":"2026-01-01","localName":"Neujahr","name":"New Year's Day","countryCode":"DE","global":true,"counties":null,"types":["Public"]}]`))
	}))
	defer srv.Close()

	got, err := NewNagerClientWithURL(srv.URL).FetchHolidays(context.Background(), 2026, "DE")
	require.NoError(t, err)
	assert.Equal(t, "/api/v3/PublicHolidays/2026/DE", gotPath)
	require.Len(t, got, 1)
	assert.Equal(t, "Neujahr", got[0].LocalName)
	assert.True(t, got[0].Global)
}

func TestFetchHolidays_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := NewNagerClientWithURL(srv.URL).FetchHolidays(context.Background(), 2026, "XX")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

func TestFetchHolidays_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer srv.Close()

	_, err := NewNagerClientWithURL(srv.URL).FetchHolidays(context.Background(), 2026, "DE")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestFetchHolidays_TransportAndRequestErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing listens any more

	_, err := NewNagerClientWithURL(url).FetchHolidays(context.Background(), 2026, "DE")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch holidays")

	_, err = NewNagerClientWithURL("http://bad\x7fhost").FetchHolidays(context.Background(), 2026, "DE")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create request")
}
