package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

type GatewayHandler struct {
	authClient     authv1.AuthServiceClient
	crmClient      crmv1.CRMServiceClient
	healthCheckers []health.Checker
}

func NewGatewayHandler(authClient authv1.AuthServiceClient, crmClient crmv1.CRMServiceClient, healthCheckers []health.Checker) *GatewayHandler {
	return &GatewayHandler{
		authClient:     authClient,
		crmClient:      crmClient,
		healthCheckers: healthCheckers,
	}
}

func (h *GatewayHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Public auth routes
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", h.HandleRegister)
		r.Post("/login", h.HandleLogin)
		r.Post("/refresh", h.HandleRefresh)
		r.Post("/logout", h.HandleLogout)

		// Protected auth routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/me", h.HandleGetProfile)
			r.Post("/change-password", h.HandleChangePassword)
		})
	})

	// Protected user routes
	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequireRole("admin", "manager")).Get("/", h.HandleListUsers)
		r.Get("/{id}", h.HandleGetUser)
		r.With(middleware.RequireRole("admin")).Put("/{id}", h.HandleUpdateUser)
		r.With(middleware.RequireRole("admin")).Post("/{id}/roles", h.HandleAssignRole)
		r.With(middleware.RequireRole("admin")).Delete("/{id}/roles", h.HandleRemoveRole)
	})

	// Invitation routes
	r.Route("/api/v1/invitations", func(r chi.Router) {
		// Public: accept invitation (no auth)
		r.Post("/{token}/accept", h.HandleAcceptInvitation)

		// Protected: create, list, cancel invitations
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.With(middleware.RequireRole("admin", "manager")).Post("/", h.HandleCreateInvitation)
			r.With(middleware.RequireRole("admin", "manager")).Get("/", h.HandleListInvitations)
			r.With(middleware.RequireRole("admin")).Delete("/{id}", h.HandleCancelInvitation)
		})
	})

	// CRM: Custom Fields routes
	r.Route("/api/v1/custom-fields", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("custom_fields", "read")).Get("/", h.HandleListCustomFields)
		r.With(middleware.RequirePermission("custom_fields", "read")).Get("/{id}", h.HandleGetCustomField)
		r.With(middleware.RequirePermission("custom_fields", "write")).Post("/", h.HandleCreateCustomField)
		r.With(middleware.RequirePermission("custom_fields", "write")).Put("/{id}", h.HandleUpdateCustomField)
		r.With(middleware.RequirePermission("custom_fields", "delete")).Delete("/{id}", h.HandleDeleteCustomField)
	})

	// CRM: Tags routes
	r.Route("/api/v1/tags", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tags", "read")).Get("/", h.HandleListTags)
		r.With(middleware.RequirePermission("tags", "read")).Get("/{id}", h.HandleGetTag)
		r.With(middleware.RequirePermission("tags", "write")).Post("/", h.HandleCreateTag)
		r.With(middleware.RequirePermission("tags", "write")).Put("/{id}", h.HandleUpdateTag)
		r.With(middleware.RequirePermission("tags", "delete")).Delete("/{id}", h.HandleDeleteTag)
	})

	// CRM: Contacts routes
	r.Route("/api/v1/contacts", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("contacts", "read")).Get("/", h.HandleListContacts)
		r.With(middleware.RequirePermission("contacts", "read")).Get("/{id}", h.HandleGetContact)
		r.With(middleware.RequirePermission("contacts", "write")).Post("/", h.HandleCreateContact)
		r.With(middleware.RequirePermission("contacts", "write")).Put("/{id}", h.HandleUpdateContact)
		r.With(middleware.RequirePermission("contacts", "delete")).Delete("/{id}", h.HandleDeleteContact)
		r.With(middleware.RequirePermission("contacts", "write")).Post("/{id}/tags", h.HandleAddContactTags)
		r.With(middleware.RequirePermission("contacts", "write")).Delete("/{id}/tags", h.HandleRemoveContactTags)
	})

	// CRM: Companies routes
	r.Route("/api/v1/companies", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("companies", "read")).Get("/", h.HandleListCompanies)
		r.With(middleware.RequirePermission("companies", "read")).Get("/{id}", h.HandleGetCompany)
		r.With(middleware.RequirePermission("companies", "read")).Get("/{id}/contacts", h.HandleGetCompanyContacts)
		r.With(middleware.RequirePermission("companies", "write")).Post("/", h.HandleCreateCompany)
		r.With(middleware.RequirePermission("companies", "write")).Put("/{id}", h.HandleUpdateCompany)
		r.With(middleware.RequirePermission("companies", "delete")).Delete("/{id}", h.HandleDeleteCompany)
	})

	// Health check
	r.Get("/health", h.HandleHealth)
}

func (h *GatewayHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	overall, checks := health.CheckAll(ctx, h.healthCheckers)

	statusCode := http.StatusOK
	if overall == health.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	response.JSON(w, statusCode, map[string]interface{}{
		"status": string(overall),
		"checks": checks,
	})
}

type registerRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (h *GatewayHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	resp, err := h.authClient.Register(r.Context(), &authv1.RegisterRequest{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *GatewayHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	resp, err := h.authClient.Login(r.Context(), &authv1.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *GatewayHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authClient.RefreshToken(r.Context(), &authv1.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *GatewayHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err := h.authClient.Logout(r.Context(), &authv1.LogoutRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (h *GatewayHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(userID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	resp, err := h.authClient.GetUser(r.Context(), &authv1.GetUserRequest{UserId: userID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	resp, err := h.authClient.ListUsers(r.Context(), &authv1.ListUsersRequest{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateUserRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

func (h *GatewayHandler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(userID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &authv1.UpdateUserRequest{UserId: userID}
	if req.FirstName != nil {
		grpcReq.FirstName = req.FirstName
	}
	if req.LastName != nil {
		grpcReq.LastName = req.LastName
	}
	if req.IsActive != nil {
		grpcReq.IsActive = req.IsActive
	}

	resp, err := h.authClient.UpdateUser(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type assignRoleRequest struct {
	RoleName string `json:"role_name"`
}

func (h *GatewayHandler) HandleAssignRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(userID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req assignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err := h.authClient.AssignRole(r.Context(), &authv1.AssignRoleRequest{
		UserId:   userID,
		RoleName: req.RoleName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "role assigned"})
}

func (h *GatewayHandler) HandleRemoveRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(userID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req assignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err := h.authClient.RemoveRole(r.Context(), &authv1.RemoveRoleRequest{
		UserId:   userID,
		RoleName: req.RoleName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "role removed"})
}

func (h *GatewayHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := h.authClient.GetProfile(r.Context(), &authv1.GetProfileRequest{UserId: userID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *GatewayHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		response.Error(w, http.StatusBadRequest, "old_password and new_password are required")
		return
	}

	_, err := h.authClient.ChangePassword(r.Context(), &authv1.ChangePasswordRequest{
		UserId:      userID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "password changed"})
}

type createInvitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *GatewayHandler) HandleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Role == "" {
		response.Error(w, http.StatusBadRequest, "email and role are required")
		return
	}

	resp, err := h.authClient.CreateInvitation(r.Context(), &authv1.CreateInvitationRequest{
		Email:     req.Email,
		Role:      req.Role,
		CreatedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleListInvitations(w http.ResponseWriter, r *http.Request) {
	resp, err := h.authClient.ListInvitations(r.Context(), &authv1.ListInvitationsRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type acceptInvitationRequest struct {
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (h *GatewayHandler) HandleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		response.Error(w, http.StatusBadRequest, "token is required")
		return
	}

	var req acceptInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" || req.FirstName == "" || req.LastName == "" {
		response.Error(w, http.StatusBadRequest, "password, first_name, and last_name are required")
		return
	}

	resp, err := h.authClient.AcceptInvitation(r.Context(), &authv1.AcceptInvitationRequest{
		Token:     token,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleCancelInvitation(w http.ResponseWriter, r *http.Request) {
	invitationID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(invitationID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid invitation id")
		return
	}

	_, err := h.authClient.CancelInvitation(r.Context(), &authv1.CancelInvitationRequest{
		InvitationId: invitationID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "invitation cancelled"})
}

// HealthHandler returns an http.HandlerFunc that runs the given health checkers.
// Used by services that need a standalone health endpoint (e.g. auth service).
func HealthHandler(checkers []health.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		overall, checks := health.CheckAll(ctx, checkers)

		statusCode := http.StatusOK
		if overall == health.StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		response.JSON(w, statusCode, map[string]interface{}{
			"status": string(overall),
			"checks": checks,
		})
	}
}

func respondGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	switch st.Code() {
	case codes.NotFound:
		response.Error(w, http.StatusNotFound, st.Message())
	case codes.AlreadyExists:
		response.Error(w, http.StatusConflict, st.Message())
	case codes.Unauthenticated:
		response.Error(w, http.StatusUnauthorized, st.Message())
	case codes.PermissionDenied:
		response.Error(w, http.StatusForbidden, st.Message())
	case codes.InvalidArgument:
		response.Error(w, http.StatusBadRequest, st.Message())
	case codes.FailedPrecondition:
		response.Error(w, http.StatusGone, st.Message())
	default:
		response.Error(w, http.StatusInternalServerError, "internal error")
	}
}

// ============================================================================
// CRM: Custom Fields Handlers
// ============================================================================

type createCustomFieldRequest struct {
	EntityType   string   `json:"entity_type"`
	FieldName    string   `json:"field_name"`
	FieldLabel   string   `json:"field_label"`
	FieldType    string   `json:"field_type"`
	Options      []string `json:"options,omitempty"`
	DefaultValue *string  `json:"default_value,omitempty"`
	IsRequired   bool     `json:"is_required"`
	SortOrder    int      `json:"sort_order"`
}

func (h *GatewayHandler) HandleCreateCustomField(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createCustomFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.EntityType == "" || req.FieldName == "" || req.FieldLabel == "" || req.FieldType == "" {
		response.Error(w, http.StatusBadRequest, "entity_type, field_name, field_label, and field_type are required")
		return
	}

	grpcReq := &crmv1.CreateCustomFieldRequest{
		EntityType: req.EntityType,
		FieldName:  req.FieldName,
		FieldLabel: req.FieldLabel,
		FieldType:  req.FieldType,
		Options:    req.Options,
		IsRequired: req.IsRequired,
		SortOrder:  int32(req.SortOrder),
		CreatedBy:  userID,
	}
	if req.DefaultValue != nil {
		grpcReq.DefaultValue = req.DefaultValue
	}

	resp, err := h.crmClient.CreateCustomField(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleGetCustomField(w http.ResponseWriter, r *http.Request) {
	fieldID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(fieldID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid custom field id")
		return
	}

	resp, err := h.crmClient.GetCustomField(r.Context(), &crmv1.GetCustomFieldRequest{Id: fieldID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleListCustomFields(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entity_type")
	page := 1
	pageSize := 20

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

	resp, err := h.crmClient.ListCustomFields(r.Context(), &crmv1.ListCustomFieldsRequest{
		EntityType: entityType,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateCustomFieldRequest struct {
	FieldLabel   *string  `json:"field_label,omitempty"`
	Options      []string `json:"options,omitempty"`
	DefaultValue *string  `json:"default_value,omitempty"`
	IsRequired   *bool    `json:"is_required,omitempty"`
	SortOrder    *int32   `json:"sort_order,omitempty"`
}

func (h *GatewayHandler) HandleUpdateCustomField(w http.ResponseWriter, r *http.Request) {
	fieldID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(fieldID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid custom field id")
		return
	}

	var req updateCustomFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateCustomFieldRequest{
		Id:      fieldID,
		Options: req.Options,
	}
	if req.FieldLabel != nil {
		grpcReq.FieldLabel = req.FieldLabel
	}
	if req.DefaultValue != nil {
		grpcReq.DefaultValue = req.DefaultValue
	}
	if req.IsRequired != nil {
		grpcReq.IsRequired = req.IsRequired
	}
	if req.SortOrder != nil {
		grpcReq.SortOrder = req.SortOrder
	}

	resp, err := h.crmClient.UpdateCustomField(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleDeleteCustomField(w http.ResponseWriter, r *http.Request) {
	fieldID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(fieldID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid custom field id")
		return
	}

	_, err := h.crmClient.DeleteCustomField(r.Context(), &crmv1.DeleteCustomFieldRequest{Id: fieldID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "custom field deleted"})
}

// ============================================================================
// CRM: Tags Handlers
// ============================================================================

type createTagRequest struct {
	Name       string `json:"name"`
	Color      string `json:"color"`
	EntityType string `json:"entity_type"`
}

func (h *GatewayHandler) HandleCreateTag(w http.ResponseWriter, r *http.Request) {
	var req createTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.EntityType == "" {
		response.Error(w, http.StatusBadRequest, "name and entity_type are required")
		return
	}

	resp, err := h.crmClient.CreateTag(r.Context(), &crmv1.CreateTagRequest{
		Name:       req.Name,
		Color:      req.Color,
		EntityType: req.EntityType,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleGetTag(w http.ResponseWriter, r *http.Request) {
	tagID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(tagID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid tag id")
		return
	}

	resp, err := h.crmClient.GetTag(r.Context(), &crmv1.GetTagRequest{Id: tagID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleListTags(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entity_type")
	page := 1
	pageSize := 50

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

	resp, err := h.crmClient.ListTags(r.Context(), &crmv1.ListTagsRequest{
		EntityType: entityType,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateTagRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
}

func (h *GatewayHandler) HandleUpdateTag(w http.ResponseWriter, r *http.Request) {
	tagID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(tagID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid tag id")
		return
	}

	var req updateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateTagRequest{Id: tagID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Color != nil {
		grpcReq.Color = req.Color
	}

	resp, err := h.crmClient.UpdateTag(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleDeleteTag(w http.ResponseWriter, r *http.Request) {
	tagID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(tagID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid tag id")
		return
	}

	_, err := h.crmClient.DeleteTag(r.Context(), &crmv1.DeleteTagRequest{Id: tagID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "tag deleted"})
}

// ============================================================================
// CRM: Contacts Handlers
// ============================================================================

type createContactRequest struct {
	FirstName    string                    `json:"first_name"`
	LastName     string                    `json:"last_name"`
	Email        string                    `json:"email"`
	Phone        string                    `json:"phone"`
	CompanyID    *string                   `json:"company_id,omitempty"`
	Position     string                    `json:"position"`
	Notes        string                    `json:"notes"`
	TagIDs       []string                  `json:"tag_ids"`
	CustomFields []customFieldValueRequest `json:"custom_fields"`
}

type customFieldValueRequest struct {
	FieldID string `json:"field_id"`
	Value   string `json:"value"` // JSON-encoded value
}

func (h *GatewayHandler) HandleCreateContact(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FirstName == "" || req.LastName == "" {
		response.Error(w, http.StatusBadRequest, "first_name and last_name are required")
		return
	}

	grpcReq := &crmv1.CreateContactRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Phone:     req.Phone,
		Position:  req.Position,
		Notes:     req.Notes,
		TagIds:    req.TagIDs,
		CreatedBy: userID,
	}

	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}

	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := h.crmClient.CreateContact(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleGetContact(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(contactID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	resp, err := h.crmClient.GetContact(r.Context(), &crmv1.GetContactRequest{Id: contactID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleListContacts(w http.ResponseWriter, r *http.Request) {
	companyID := r.URL.Query().Get("company_id")
	search := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sort_by")
	sortDesc := r.URL.Query().Get("sort_desc") == "true"
	tagIDs := r.URL.Query()["tag_ids"]
	page := 1
	pageSize := 20

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

	grpcReq := &crmv1.ListContactsRequest{
		Search:   search,
		Page:     int32(page),
		PageSize: int32(pageSize),
		SortBy:   sortBy,
		SortDesc: sortDesc,
		TagIds:   tagIDs,
	}

	if companyID != "" {
		grpcReq.CompanyId = &companyID
	}

	resp, err := h.crmClient.ListContacts(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateContactRequest struct {
	FirstName    *string                   `json:"first_name,omitempty"`
	LastName     *string                   `json:"last_name,omitempty"`
	Email        *string                   `json:"email,omitempty"`
	Phone        *string                   `json:"phone,omitempty"`
	CompanyID    *string                   `json:"company_id,omitempty"`
	Position     *string                   `json:"position,omitempty"`
	Notes        *string                   `json:"notes,omitempty"`
	CustomFields []customFieldValueRequest `json:"custom_fields"`
}

func (h *GatewayHandler) HandleUpdateContact(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(contactID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	var req updateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateContactRequest{Id: contactID}
	if req.FirstName != nil {
		grpcReq.FirstName = req.FirstName
	}
	if req.LastName != nil {
		grpcReq.LastName = req.LastName
	}
	if req.Email != nil {
		grpcReq.Email = req.Email
	}
	if req.Phone != nil {
		grpcReq.Phone = req.Phone
	}
	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}
	if req.Position != nil {
		grpcReq.Position = req.Position
	}
	if req.Notes != nil {
		grpcReq.Notes = req.Notes
	}
	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := h.crmClient.UpdateContact(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleDeleteContact(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(contactID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	_, err := h.crmClient.DeleteContact(r.Context(), &crmv1.DeleteContactRequest{Id: contactID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "contact deleted"})
}

type modifyContactTagsRequest struct {
	TagIDs []string `json:"tag_ids"`
}

func (h *GatewayHandler) HandleAddContactTags(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(contactID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	var req modifyContactTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.crmClient.AddContactTags(r.Context(), &crmv1.AddContactTagsRequest{
		ContactId: contactID,
		TagIds:    req.TagIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleRemoveContactTags(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(contactID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	var req modifyContactTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.crmClient.RemoveContactTags(r.Context(), &crmv1.RemoveContactTagsRequest{
		ContactId: contactID,
		TagIds:    req.TagIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// CRM: Companies Handlers
// ============================================================================

type createCompanyRequest struct {
	Name          string                    `json:"name"`
	Domain        string                    `json:"domain"`
	Industry      string                    `json:"industry"`
	EmployeeCount int32                     `json:"employee_count"`
	Address       string                    `json:"address"`
	City          string                    `json:"city"`
	Country       string                    `json:"country"`
	Notes         string                    `json:"notes"`
	TagIDs        []string                  `json:"tag_ids"`
	CustomFields  []customFieldValueRequest `json:"custom_fields"`
}

func (h *GatewayHandler) HandleCreateCompany(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	grpcReq := &crmv1.CreateCompanyRequest{
		Name:          req.Name,
		Domain:        req.Domain,
		Industry:      req.Industry,
		EmployeeCount: req.EmployeeCount,
		Address:       req.Address,
		City:          req.City,
		Country:       req.Country,
		Notes:         req.Notes,
		TagIds:        req.TagIDs,
		CreatedBy:     userID,
	}

	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := h.crmClient.CreateCompany(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleGetCompany(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(companyID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid company id")
		return
	}

	resp, err := h.crmClient.GetCompany(r.Context(), &crmv1.GetCompanyRequest{Id: companyID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleListCompanies(w http.ResponseWriter, r *http.Request) {
	industry := r.URL.Query().Get("industry")
	search := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sort_by")
	sortDesc := r.URL.Query().Get("sort_desc") == "true"
	tagIDs := r.URL.Query()["tag_ids"]
	page := 1
	pageSize := 20

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

	resp, err := h.crmClient.ListCompanies(r.Context(), &crmv1.ListCompaniesRequest{
		Industry: industry,
		Search:   search,
		Page:     int32(page),
		PageSize: int32(pageSize),
		SortBy:   sortBy,
		SortDesc: sortDesc,
		TagIds:   tagIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateCompanyRequest struct {
	Name          *string                   `json:"name,omitempty"`
	Domain        *string                   `json:"domain,omitempty"`
	Industry      *string                   `json:"industry,omitempty"`
	EmployeeCount *int32                    `json:"employee_count,omitempty"`
	Address       *string                   `json:"address,omitempty"`
	City          *string                   `json:"city,omitempty"`
	Country       *string                   `json:"country,omitempty"`
	Notes         *string                   `json:"notes,omitempty"`
	CustomFields  []customFieldValueRequest `json:"custom_fields"`
}

func (h *GatewayHandler) HandleUpdateCompany(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(companyID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid company id")
		return
	}

	var req updateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateCompanyRequest{Id: companyID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Domain != nil {
		grpcReq.Domain = req.Domain
	}
	if req.Industry != nil {
		grpcReq.Industry = req.Industry
	}
	if req.EmployeeCount != nil {
		grpcReq.EmployeeCount = req.EmployeeCount
	}
	if req.Address != nil {
		grpcReq.Address = req.Address
	}
	if req.City != nil {
		grpcReq.City = req.City
	}
	if req.Country != nil {
		grpcReq.Country = req.Country
	}
	if req.Notes != nil {
		grpcReq.Notes = req.Notes
	}
	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := h.crmClient.UpdateCompany(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleDeleteCompany(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(companyID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid company id")
		return
	}

	_, err := h.crmClient.DeleteCompany(r.Context(), &crmv1.DeleteCompanyRequest{Id: companyID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "company deleted"})
}

func (h *GatewayHandler) HandleGetCompanyContacts(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(companyID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid company id")
		return
	}

	page := 1
	pageSize := 20

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

	resp, err := h.crmClient.GetCompanyContacts(r.Context(), &crmv1.GetCompanyContactsRequest{
		CompanyId: companyID,
		Page:      int32(page),
		PageSize:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}
