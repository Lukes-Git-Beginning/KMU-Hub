package gateway

import (
	"log/slog"
	"net/http"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/server/response"
)

// respondGRPCError translates a gRPC error into an appropriate HTTP error response.
// If the error is a gRPC Unavailable error (service down), it returns 503.
func respondGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpCode := grpcStatusToHTTP(st.Code())
	msg := st.Message()

	// For internal errors, don't leak the message
	if httpCode == http.StatusInternalServerError {
		msg = "internal error"
	}

	response.Error(w, httpCode, msg)
}

// grpcStatusToHTTP converts a gRPC status code to the corresponding HTTP status code.
func grpcStatusToHTTP(code codes.Code) int {
	switch code {
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.FailedPrecondition:
		return http.StatusGone
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

// respondServiceUnavailable sends a 503 response indicating the named service is unavailable.
func respondServiceUnavailable(w http.ResponseWriter, serviceName string) {
	slog.Warn("service unavailable", "service", serviceName)
	response.Error(w, http.StatusServiceUnavailable, serviceName+" service unavailable")
}

// parsePagination extracts page and page_size from query parameters with defaults.
func parsePagination(r *http.Request, defaultPage, defaultPageSize int) (page, pageSize int) {
	page = defaultPage
	pageSize = defaultPageSize

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}
	return page, pageSize
}

// parseLimit extracts a limit query parameter with default and max bounds.
func parseLimit(r *http.Request, defaultLimit, maxLimit int) int {
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= maxLimit {
			return parsed
		}
	}
	return defaultLimit
}
