package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	"github.com/kmuhub/kmuhub/internal/validation"
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
		// 409 Conflict: the request conflicts with the resource's current state
		// (consent pending, invitation expired/used, invalid state transition).
		// Not 410 Gone — the resource is not permanently removed.
		return http.StatusConflict
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	// OutOfRange carries "semantically valid request, unprocessable value"
	// errors (currently: an unknown RBAC capability key) — 422, distinct from
	// InvalidArgument's 400 malformed-request meaning.
	case codes.OutOfRange:
		return http.StatusUnprocessableEntity
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

// validateUUIDParam extracts a chi URL parameter and validates it as a UUID.
// Returns the raw string value and true on success. On failure, writes a 400
// error response and returns false. The raw string is returned (not a uuid.UUID)
// because gRPC handlers accept string IDs.
func validateUUIDParam(w http.ResponseWriter, r *http.Request, param string) (string, bool) {
	raw := chi.URLParam(r, param)
	if _, err := uuid.Parse(raw); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid "+param)
		return "", false
	}
	return raw, true
}

// ownerFilterForScope resolves the data scope of one permission into a list
// filter. It returns a non-nil user id exactly when the caller's grant on
// resource:action is narrowed to "own", and nil when they may see the whole
// tenant. On failure it writes a 401 and returns false, and the caller must
// return immediately.
//
//	ownerID, ok := ownerFilterForScope(w, r, "rapporte:report", "read")
//	if !ok {
//	    return
//	}
//	grpcReq.AuthorId = ownerID
//
// Assigning the result unconditionally is the point: at scope "own" it must
// override whatever owner filter the client asked for, or the narrowing is a
// suggestion rather than a boundary.
//
// The id then travels to the service and ends up in the repository's WHERE
// clause — never in a post-filter over the response, which would leave the
// total count and the page boundaries describing rows the caller may not see.
// Scope "team" needs a reporting-line resolver that does not exist yet and
// currently behaves like "all".
func ownerFilterForScope(w http.ResponseWriter, r *http.Request, resource, action string) (*string, bool) {
	if middleware.PermissionScope(r.Context(), resource, action) != auth.ScopeOwn {
		return nil, true
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		// Refusing beats falling through: an empty id would either filter on
		// nothing and hand back the full list, or reach the service as an
		// unparsable filter.
		response.Error(w, http.StatusUnauthorized, "missing user in token")
		return nil, false
	}
	return &userID, true
}

// decodeAndValidate decodes the JSON request body into a value of type T and
// validates it against its `validate` struct tags via the shared validation
// package. On a decode error it writes a 400 with the contract-stable
// {"error": "invalid request body"} shape; on a validation error it writes a
// 400 with the structured {"error", "code": "validation_failed", "details": [...]}
// body. In both failure cases it returns (zero, false) and the caller must
// return immediately. On success it returns (value, true).
//
// This is the single entry point for "Parse → Validate" in thin handlers:
//
//	req, ok := decodeAndValidate[createContactRequest](w, r)
//	if !ok {
//	    return
//	}
func decodeAndValidate[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return req, false
	}
	if err := validation.Validate(&req); err != nil {
		if body, ok := validation.ErrorBody(err); ok {
			response.JSON(w, http.StatusBadRequest, body)
		} else {
			response.Error(w, http.StatusBadRequest, "validation failed")
		}
		return req, false
	}
	return req, true
}

// validateCustomerVAT rejects a customer VAT identification number that is
// present but invalid (DE/AT/CH check digits are verified; the rest of the EU
// stays structural-only). It mirrors decodeAndValidate's {error, code, details}
// response shape and returns false after writing the response on failure. An
// empty id passes — the field is optional. Use it after decodeAndValidate in
// the invoice/quote/credit-note create handlers, whose customer is a generated
// proto type that cannot carry validate tags.
func validateCustomerVAT(w http.ResponseWriter, vatID string) bool {
	body := struct {
		UstID string `json:"customer.ust_id_nr" validate:"omitempty,ustid_eu"`
	}{UstID: vatID}
	if err := validation.Validate(&body); err != nil {
		if errBody, ok := validation.ErrorBody(err); ok {
			response.JSON(w, http.StatusBadRequest, errBody)
		} else {
			response.Error(w, http.StatusBadRequest, "validation failed")
		}
		return false
	}
	return true
}

// RequireAuthenticated is a chi middleware that rejects requests without
// a valid user ID in the context. Apply as a route-group middleware to
// eliminate per-handler auth guards.
func RequireAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if middleware.GetUserID(r.Context()) == "" {
			response.Error(w, http.StatusUnauthorized, "user not authenticated")
			return
		}
		next.ServeHTTP(w, r)
	})
}
