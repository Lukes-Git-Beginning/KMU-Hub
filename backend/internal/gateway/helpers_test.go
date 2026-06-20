package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGrpcStatusToHTTP(t *testing.T) {
	tests := []struct {
		code     codes.Code
		expected int
	}{
		{codes.NotFound, http.StatusNotFound},
		{codes.AlreadyExists, http.StatusConflict},
		{codes.Unauthenticated, http.StatusUnauthorized},
		{codes.PermissionDenied, http.StatusForbidden},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.FailedPrecondition, http.StatusConflict},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.DeadlineExceeded, http.StatusGatewayTimeout},
		{codes.ResourceExhausted, http.StatusTooManyRequests},
		{codes.Unimplemented, http.StatusNotImplemented},
		{codes.Internal, http.StatusInternalServerError},
		{codes.Unknown, http.StatusInternalServerError},
		{codes.DataLoss, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			got := grpcStatusToHTTP(tt.code)
			if got != tt.expected {
				t.Errorf("grpcStatusToHTTP(%v) = %d, want %d", tt.code, got, tt.expected)
			}
		})
	}
}

func TestRespondGRPCError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantMsg    string
		wantMsgNot string // message should NOT contain this
	}{
		{
			name:     "not_found",
			err:      status.Error(codes.NotFound, "contact not found"),
			wantCode: http.StatusNotFound,
			wantMsg:  "contact not found",
		},
		{
			name:     "invalid_argument",
			err:      status.Error(codes.InvalidArgument, "email is required"),
			wantCode: http.StatusBadRequest,
			wantMsg:  "email is required",
		},
		{
			name:     "internal_hides_message",
			err:      status.Error(codes.Internal, "secret database error"),
			wantCode: http.StatusInternalServerError,
			wantMsg:  "internal error",
		},
		{
			name:     "unauthenticated",
			err:      status.Error(codes.Unauthenticated, "invalid token"),
			wantCode: http.StatusUnauthorized,
			wantMsg:  "invalid token",
		},
		{
			name:     "unavailable",
			err:      status.Error(codes.Unavailable, "service down"),
			wantCode: http.StatusServiceUnavailable,
			wantMsg:  "service down",
		},
		{
			name:     "non_grpc_error",
			err:      http.ErrNoCookie,
			wantCode: http.StatusInternalServerError,
			wantMsg:  "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			respondGRPCError(rec, tt.err)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}

			var resp map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if resp["error"] != tt.wantMsg {
				t.Errorf("message = %q, want %q", resp["error"], tt.wantMsg)
			}
		})
	}
}

func TestRespondServiceUnavailable(t *testing.T) {
	rec := httptest.NewRecorder()
	respondServiceUnavailable(rec, "auth")

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "auth service unavailable" {
		t.Errorf("message = %q, want %q", resp["error"], "auth service unavailable")
	}
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		defPage      int
		defPageSize  int
		wantPage     int
		wantPageSize int
	}{
		{"defaults", "", 1, 20, 1, 20},
		{"custom_page", "?page=3", 1, 20, 3, 20},
		{"custom_page_size", "?page_size=50", 1, 20, 1, 50},
		{"both", "?page=2&page_size=10", 1, 20, 2, 10},
		{"invalid_page", "?page=abc", 1, 20, 1, 20},
		{"negative_page", "?page=-1", 1, 20, 1, 20},
		{"zero_page", "?page=0", 1, 20, 1, 20},
		{"page_size_over_max", "?page_size=200", 1, 20, 1, 20},
		{"page_size_zero", "?page_size=0", 1, 20, 1, 20},
		{"page_size_negative", "?page_size=-5", 1, 20, 1, 20},
		{"page_size_at_max", "?page_size=100", 1, 20, 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/"+tt.query, nil)
			page, pageSize := parsePagination(r, tt.defPage, tt.defPageSize)
			if page != tt.wantPage || pageSize != tt.wantPageSize {
				t.Errorf("parsePagination() = (%d, %d), want (%d, %d)",
					page, pageSize, tt.wantPage, tt.wantPageSize)
			}
		})
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		defLimit int
		maxLimit int
		want     int
	}{
		{"default", "", 10, 100, 10},
		{"custom", "?limit=25", 10, 100, 25},
		{"at_max", "?limit=100", 10, 100, 100},
		{"over_max", "?limit=200", 10, 100, 10},
		{"invalid", "?limit=abc", 10, 100, 10},
		{"negative", "?limit=-1", 10, 100, 10},
		{"zero", "?limit=0", 10, 100, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/"+tt.query, nil)
			got := parseLimit(r, tt.defLimit, tt.maxLimit)
			if got != tt.want {
				t.Errorf("parseLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}
