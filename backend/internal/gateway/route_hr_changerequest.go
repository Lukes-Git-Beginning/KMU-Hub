package gateway

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// ============================================================================
// Profile Change Request Handlers
// ============================================================================

// changeRequestBody mirrors desktop's ProfileChangeRequest
// (api/hr-change-requests.ts) field for field. It is hand-written rather than
// protojson-marshalled because the gateway's protojson options use proto names
// (snake_case) while this contract is camelCase — marshalling the message would
// hand the frontend user_id where it reads userId.
type changeRequestBody struct {
	ID            string  `json:"id"`
	UserID        string  `json:"userId"`
	UserName      string  `json:"userName"`
	Drawer        string  `json:"drawer"`
	Field         string  `json:"field"`
	FieldLabel    string  `json:"fieldLabel"`
	OldValue      string  `json:"oldValue"`
	NewValue      string  `json:"newValue"`
	Status        string  `json:"status"`
	Reason        string  `json:"reason,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	DecidedAt     *string `json:"decidedAt,omitempty"`
	DecidedByName string  `json:"decidedByName,omitempty"`
}

func toChangeRequestBody(r *hrv1.ProfileChangeRequest) changeRequestBody {
	body := changeRequestBody{
		ID:            r.GetId(),
		UserID:        r.GetUserId(),
		UserName:      r.GetUserName(),
		Drawer:        r.GetDrawer(),
		Field:         r.GetField(),
		FieldLabel:    r.GetFieldLabel(),
		OldValue:      r.GetOldValue(),
		NewValue:      r.GetNewValue(),
		Status:        r.GetStatus(),
		Reason:        r.GetReason(),
		CreatedAt:     r.GetCreatedAt().AsTime().UTC().Format(time.RFC3339),
		DecidedByName: r.GetDecidedByName(),
	}
	if r.GetDecidedAt() != nil {
		decided := r.GetDecidedAt().AsTime().UTC().Format(time.RFC3339)
		body.DecidedAt = &decided
	}
	return body
}

// respondChangeRequest answers the four single-entity routes with the wrapped
// shape the frontend reads (data.request).
func respondChangeRequest(w http.ResponseWriter, code int, req *hrv1.ProfileChangeRequest) {
	response.JSON(w, code, map[string]any{"request": toChangeRequestBody(req)})
}

// callerHasPermission reports whether the token carries an exact capability key.
// PermissionScope cannot answer this: an absent key resolves to "all" there, on
// purpose, so it would report full reach for a caller who holds nothing.
func callerHasPermission(r *http.Request, key string) bool {
	for _, p := range middleware.GetUserPermissions(r.Context()) {
		if p == key {
			return true
		}
	}
	return false
}

// changeRequestListReach decides how far a list read may see. A caller who
// cannot decide gets their own proposals whatever the query says; the guard on
// the route admits proposers so they can see THEIR requests, not the tenant's.
func changeRequestListReach(r *http.Request) (ownOnly bool, actorScope string) {
	if !callerHasPermission(r, "team:data_personal:edit") {
		return true, ""
	}
	return r.URL.Query().Get("scope") == "mine",
		middleware.PermissionScope(r.Context(), "team:data_personal", "edit")
}

// HandleListChangeRequests answers both the HR inbox and an employee's own list.
//
// ?scope=mine asks for the caller's own proposals. A caller without
// team:data_personal:edit gets that regardless of the query parameter — the
// route is readable for proposers so they can see their own requests, not so
// they can read the whole tenant's.
func (h *HRRoutes) HandleListChangeRequests(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	statusFilter := r.URL.Query().Get("status")
	switch statusFilter {
	case "", "pending", "approved", "rejected", "cancelled":
	default:
		response.Error(w, http.StatusBadRequest, "invalid status filter")
		return
	}

	ownOnly, actorScope := changeRequestListReach(r)

	resp, err := client.ListProfileChangeRequests(r.Context(), &hrv1.ListProfileChangeRequestsReq{
		TenantId:   tenantID,
		ActorId:    middleware.GetUserID(r.Context()),
		OwnOnly:    ownOnly,
		Status:     statusFilter,
		ActorScope: actorScope,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	requests := make([]changeRequestBody, 0, len(resp.GetRequests()))
	for _, cr := range resp.GetRequests() {
		requests = append(requests, toChangeRequestBody(cr))
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"requests": requests,
		"total":    resp.GetTotal(),
	})
}

type createChangeRequestHTTPReq struct {
	Drawer     string `json:"drawer"      validate:"omitempty,oneof=personal job"`
	Field      string `json:"field"       validate:"required,max=64"`
	FieldLabel string `json:"fieldLabel"  validate:"omitempty,max=120"`
	OldValue   string `json:"oldValue"    validate:"omitempty,max=500"`
	NewValue   string `json:"newValue"    validate:"required,max=500"`
}

// HandleCreateChangeRequest opens a proposal for the caller's own profile. The
// proposer comes from the token, never from the body.
func (h *HRRoutes) HandleCreateChangeRequest(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[createChangeRequestHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateProfileChangeRequest(r.Context(), &hrv1.CreateProfileChangeRequestReq{
		TenantId:   tenantID,
		ActorId:    middleware.GetUserID(r.Context()),
		Drawer:     req.Drawer,
		Field:      req.Field,
		FieldLabel: req.FieldLabel,
		OldValue:   req.OldValue,
		NewValue:   req.NewValue,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondChangeRequest(w, http.StatusCreated, resp.GetRequest())
}

// HandleApproveChangeRequest applies the proposal to the employee profile.
func (h *HRRoutes) HandleApproveChangeRequest(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ApproveProfileChangeRequest(r.Context(), &hrv1.ApproveProfileChangeRequestReq{
		TenantId:   tenantID,
		ActorId:    middleware.GetUserID(r.Context()),
		RequestId:  chi.URLParam(r, "id"),
		ActorScope: middleware.PermissionScope(r.Context(), "team:data_personal", "edit"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondChangeRequest(w, http.StatusOK, resp.GetRequest())
}

type rejectChangeRequestHTTPReq struct {
	// Not validate:"required": the validator answers 400, and a missing reason
	// is a semantic refusal (422), not a malformed request. The service raises
	// ErrReasonRequired, which maps to codes.OutOfRange and therefore to 422.
	Reason string `json:"reason" validate:"omitempty,max=500"`
}

// HandleRejectChangeRequest closes a proposal with a mandatory reason.
func (h *HRRoutes) HandleRejectChangeRequest(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[rejectChangeRequestHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.RejectProfileChangeRequest(r.Context(), &hrv1.RejectProfileChangeRequestReq{
		TenantId:   tenantID,
		ActorId:    middleware.GetUserID(r.Context()),
		RequestId:  chi.URLParam(r, "id"),
		ActorScope: middleware.PermissionScope(r.Context(), "team:data_personal", "edit"),
		Reason:     req.Reason,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondChangeRequest(w, http.StatusOK, resp.GetRequest())
}

// HandleCancelChangeRequest withdraws the caller's own pending proposal.
func (h *HRRoutes) HandleCancelChangeRequest(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.CancelProfileChangeRequest(r.Context(), &hrv1.CancelProfileChangeRequestReq{
		TenantId:  tenantID,
		ActorId:   middleware.GetUserID(r.Context()),
		RequestId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondChangeRequest(w, http.StatusOK, resp.GetRequest())
}
