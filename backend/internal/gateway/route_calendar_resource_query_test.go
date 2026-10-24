package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The desktop hooks (useResources.ts) and the spec name the resource query
// parameters differently; both spellings have to reach the same RPC filter.

func TestHandleListResources_DesktopAndSpecQueryNamesReachTheSameFilter(t *testing.T) {
	cases := map[string]string{
		"spec":    "type=room&tags=a,b&include_inactive=true",
		"desktop": "resource_type=room&tag=a,b&is_active=false",
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			routes, stub := newCalendarStubRoutes(t)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources?"+query, nil)
			req = withAuth(req, uuid.New().String(), testTenantID)
			routes.HandleListResources(rec, req)

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.NotNil(t, stub.gotResources)
			require.NotNil(t, stub.gotResources.ResourceType)
			assert.Equal(t, "room", *stub.gotResources.ResourceType)
			assert.Equal(t, []string{"a", "b"}, stub.gotResources.Tags)
			assert.True(t, stub.gotResources.IncludeInactive)
		})
	}
}

func TestHandleListResources_IsActiveTrueKeepsActiveOnly(t *testing.T) {
	routes, stub := newCalendarStubRoutes(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources?is_active=true", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListResources(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.False(t, stub.gotResources.IncludeInactive)
}

func TestHandleListResourceAvailability_DateParameterCoversOneDay(t *testing.T) {
	routes, stub := newCalendarStubRoutes(t)
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id+"/availability?date=2026-03-02", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListResourceAvailability(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, stub.gotAvail)
	assert.Equal(t, time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), stub.gotAvail.Start.AsTime())
	assert.Equal(t, time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), stub.gotAvail.End.AsTime())
}

func TestHandleListResourceAvailability_StartEndStillWinOverDate(t *testing.T) {
	routes, stub := newCalendarStubRoutes(t)
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id+"/availability?start=2026-03-02T08:00:00Z&end=2026-03-02T12:00:00Z&date=2026-04-01", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListResourceAvailability(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC), stub.gotAvail.Start.AsTime())
	assert.Equal(t, time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC), stub.gotAvail.End.AsTime())
}

func TestHandleListResourceAvailability_InvalidDateAndMissingRangeAre400(t *testing.T) {
	routes, _ := newCalendarStubRoutes(t)
	id := uuid.New().String()
	for _, query := range []string{"date=02.03.2026", ""} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id+"/availability?"+query, nil)
		req = withAuth(req, uuid.New().String(), testTenantID)
		req = withChiURLParam(req, "id", id)
		routes.HandleListResourceAvailability(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code, query)
	}
}
