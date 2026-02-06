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
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

type GatewayHandler struct {
	authClient     authv1.AuthServiceClient
	crmClient      crmv1.CRMServiceClient
	chatClient     chatv1.ChatServiceClient
	healthCheckers []health.Checker
}

func NewGatewayHandler(authClient authv1.AuthServiceClient, crmClient crmv1.CRMServiceClient, chatClient chatv1.ChatServiceClient, healthCheckers []health.Checker) *GatewayHandler {
	return &GatewayHandler{
		authClient:     authClient,
		crmClient:      crmClient,
		chatClient:     chatClient,
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

	// CRM: Pipeline Stages routes
	r.Route("/api/v1/pipeline-stages", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("pipeline_stages", "read")).Get("/", h.HandleListPipelineStages)
		r.With(middleware.RequirePermission("pipeline_stages", "read")).Get("/{id}", h.HandleGetPipelineStage)
		r.With(middleware.RequirePermission("pipeline_stages", "write")).Post("/", h.HandleCreatePipelineStage)
		r.With(middleware.RequirePermission("pipeline_stages", "write")).Put("/{id}", h.HandleUpdatePipelineStage)
		r.With(middleware.RequirePermission("pipeline_stages", "write")).Post("/reorder", h.HandleReorderPipelineStages)
		r.With(middleware.RequirePermission("pipeline_stages", "delete")).Delete("/{id}", h.HandleDeletePipelineStage)
	})

	// CRM: Deals routes
	r.Route("/api/v1/deals", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("deals", "read")).Get("/", h.HandleListDeals)
		r.With(middleware.RequirePermission("deals", "read")).Get("/{id}", h.HandleGetDeal)
		r.With(middleware.RequirePermission("deals", "write")).Post("/", h.HandleCreateDeal)
		r.With(middleware.RequirePermission("deals", "write")).Put("/{id}", h.HandleUpdateDeal)
		r.With(middleware.RequirePermission("deals", "write")).Post("/{id}/stage", h.HandleMoveDealToStage)
		r.With(middleware.RequirePermission("deals", "write")).Post("/{id}/tags", h.HandleAddDealTags)
		r.With(middleware.RequirePermission("deals", "write")).Delete("/{id}/tags", h.HandleRemoveDealTags)
		r.With(middleware.RequirePermission("deals", "delete")).Delete("/{id}", h.HandleDeleteDeal)
	})

	// CRM: Activities routes
	r.Route("/api/v1/activities", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("activities", "read")).Get("/", h.HandleListActivities)
		r.With(middleware.RequirePermission("activities", "read")).Get("/{id}", h.HandleGetActivity)
		r.With(middleware.RequirePermission("activities", "write")).Post("/", h.HandleCreateActivity)
		r.With(middleware.RequirePermission("activities", "write")).Put("/{id}", h.HandleUpdateActivity)
		r.With(middleware.RequirePermission("activities", "write")).Post("/{id}/complete", h.HandleCompleteActivity)
		r.With(middleware.RequirePermission("activities", "write")).Post("/{id}/tags", h.HandleAddActivityTags)
		r.With(middleware.RequirePermission("activities", "write")).Delete("/{id}/tags", h.HandleRemoveActivityTags)
		r.With(middleware.RequirePermission("activities", "delete")).Delete("/{id}", h.HandleDeleteActivity)
	})

	// CRM: Search route
	r.Route("/api/v1/search", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", h.HandleSearch)
	})

	// CRM: Saved Filters routes
	r.Route("/api/v1/saved-filters", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("saved_filters", "read")).Get("/", h.HandleListSavedFilters)
		r.With(middleware.RequirePermission("saved_filters", "read")).Get("/{id}", h.HandleGetSavedFilter)
		r.With(middleware.RequirePermission("saved_filters", "write")).Post("/", h.HandleCreateSavedFilter)
		r.With(middleware.RequirePermission("saved_filters", "write")).Patch("/{id}", h.HandleUpdateSavedFilter)
		r.With(middleware.RequirePermission("saved_filters", "delete")).Delete("/{id}", h.HandleDeleteSavedFilter)
	})

	// CRM: Reports routes
	r.Route("/api/v1/reports", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("reports", "read")).Get("/pipeline", h.HandleGetPipelineReport)
		r.With(middleware.RequirePermission("reports", "read")).Get("/conversion", h.HandleGetConversionReport)
		r.With(middleware.RequirePermission("reports", "read")).Get("/activities", h.HandleGetActivityReport)
	})

	// Chat: Channels routes
	r.Route("/api/v1/channels", func(r chi.Router) {
		r.Use(authMiddleware)
		// DM routes (before /{id} to avoid conflict)
		r.With(middleware.RequirePermission("channels", "write")).Post("/dm", h.HandleGetOrCreateDM)
		r.With(middleware.RequirePermission("channels", "read")).Get("/dm", h.HandleListDMs)
		// Channel CRUD
		r.With(middleware.RequirePermission("channels", "read")).Get("/", h.HandleListChannels)
		r.With(middleware.RequirePermission("channels", "write")).Post("/", h.HandleCreateChannel)
		r.With(middleware.RequirePermission("channels", "read")).Get("/{id}", h.HandleGetChannel)
		r.With(middleware.RequirePermission("channels", "write")).Put("/{id}", h.HandleUpdateChannel)
		r.With(middleware.RequirePermission("channels", "delete")).Delete("/{id}", h.HandleDeleteChannel)
		r.With(middleware.RequirePermission("channels", "write")).Post("/{id}/archive", h.HandleArchiveChannel)
		// Membership routes
		r.With(middleware.RequirePermission("channels", "write")).Post("/{id}/join", h.HandleJoinChannel)
		r.With(middleware.RequirePermission("channels", "write")).Post("/{id}/leave", h.HandleLeaveChannel)
		r.With(middleware.RequirePermission("channels", "read")).Get("/{id}/members", h.HandleGetChannelMembers)
		r.With(middleware.RequirePermission("channels", "write")).Put("/{id}/members/{userId}/role", h.HandleUpdateMemberRole)
		// Message routes
		r.With(middleware.RequirePermission("messages", "read")).Get("/{id}/messages", h.HandleGetMessages)
		r.With(middleware.RequirePermission("messages", "write")).Post("/{id}/messages", h.HandleSendMessage)
	})

	// Chat: Message routes (for individual message operations)
	r.Route("/api/v1/messages", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("messages", "read")).Get("/{id}/thread", h.HandleGetThreadReplies)
		r.With(middleware.RequirePermission("messages", "write")).Put("/{id}", h.HandleUpdateMessage)
		r.With(middleware.RequirePermission("messages", "delete")).Delete("/{id}", h.HandleDeleteMessage)
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

// ============================================================================
// CRM: Pipeline Stages Handlers
// ============================================================================

type createPipelineStageRequest struct {
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	IsWon       bool    `json:"is_won"`
	IsLost      bool    `json:"is_lost"`
	Probability float64 `json:"probability"`
}

func (h *GatewayHandler) HandleCreatePipelineStage(w http.ResponseWriter, r *http.Request) {
	var req createPipelineStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	resp, err := h.crmClient.CreatePipelineStage(r.Context(), &crmv1.CreatePipelineStageRequest{
		Name:        req.Name,
		Color:       req.Color,
		IsWon:       req.IsWon,
		IsLost:      req.IsLost,
		Probability: req.Probability,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleGetPipelineStage(w http.ResponseWriter, r *http.Request) {
	stageID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(stageID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid pipeline stage id")
		return
	}

	resp, err := h.crmClient.GetPipelineStage(r.Context(), &crmv1.GetPipelineStageRequest{Id: stageID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleListPipelineStages(w http.ResponseWriter, r *http.Request) {
	resp, err := h.crmClient.ListPipelineStages(r.Context(), &crmv1.ListPipelineStagesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updatePipelineStageRequest struct {
	Name        *string  `json:"name,omitempty"`
	Color       *string  `json:"color,omitempty"`
	IsWon       *bool    `json:"is_won,omitempty"`
	IsLost      *bool    `json:"is_lost,omitempty"`
	Probability *float64 `json:"probability,omitempty"`
}

func (h *GatewayHandler) HandleUpdatePipelineStage(w http.ResponseWriter, r *http.Request) {
	stageID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(stageID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid pipeline stage id")
		return
	}

	var req updatePipelineStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdatePipelineStageRequest{Id: stageID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Color != nil {
		grpcReq.Color = req.Color
	}
	if req.IsWon != nil {
		grpcReq.IsWon = req.IsWon
	}
	if req.IsLost != nil {
		grpcReq.IsLost = req.IsLost
	}
	if req.Probability != nil {
		grpcReq.Probability = req.Probability
	}

	resp, err := h.crmClient.UpdatePipelineStage(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleDeletePipelineStage(w http.ResponseWriter, r *http.Request) {
	stageID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(stageID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid pipeline stage id")
		return
	}

	_, err := h.crmClient.DeletePipelineStage(r.Context(), &crmv1.DeletePipelineStageRequest{Id: stageID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "pipeline stage deleted"})
}

type reorderPipelineStagesRequest struct {
	StageIDs []string `json:"stage_ids"`
}

func (h *GatewayHandler) HandleReorderPipelineStages(w http.ResponseWriter, r *http.Request) {
	var req reorderPipelineStagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.StageIDs) == 0 {
		response.Error(w, http.StatusBadRequest, "stage_ids is required")
		return
	}

	resp, err := h.crmClient.ReorderPipelineStages(r.Context(), &crmv1.ReorderPipelineStagesRequest{
		StageIds: req.StageIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// CRM: Deals Handlers
// ============================================================================

type createDealRequest struct {
	Name              string                    `json:"name"`
	Value             float64                   `json:"value"`
	Currency          string                    `json:"currency"`
	StageID           string                    `json:"stage_id"`
	ContactID         *string                   `json:"contact_id,omitempty"`
	CompanyID         *string                   `json:"company_id,omitempty"`
	OwnerID           *string                   `json:"owner_id,omitempty"`
	ExpectedCloseDate string                    `json:"expected_close_date"`
	Notes             string                    `json:"notes"`
	TagIDs            []string                  `json:"tag_ids"`
	CustomFields      []customFieldValueRequest `json:"custom_fields"`
}

func (h *GatewayHandler) HandleCreateDeal(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.StageID == "" {
		response.Error(w, http.StatusBadRequest, "name and stage_id are required")
		return
	}

	grpcReq := &crmv1.CreateDealRequest{
		Name:              req.Name,
		Value:             req.Value,
		Currency:          req.Currency,
		StageId:           req.StageID,
		ExpectedCloseDate: req.ExpectedCloseDate,
		Notes:             req.Notes,
		TagIds:            req.TagIDs,
		CreatedBy:         userID,
	}

	if req.ContactID != nil {
		grpcReq.ContactId = req.ContactID
	}
	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}
	if req.OwnerID != nil {
		grpcReq.OwnerId = req.OwnerID
	}

	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := h.crmClient.CreateDeal(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleGetDeal(w http.ResponseWriter, r *http.Request) {
	dealID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(dealID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid deal id")
		return
	}

	resp, err := h.crmClient.GetDeal(r.Context(), &crmv1.GetDealRequest{Id: dealID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleListDeals(w http.ResponseWriter, r *http.Request) {
	stageID := r.URL.Query().Get("stage_id")
	contactID := r.URL.Query().Get("contact_id")
	companyID := r.URL.Query().Get("company_id")
	ownerID := r.URL.Query().Get("owner_id")
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

	grpcReq := &crmv1.ListDealsRequest{
		Search:   search,
		Page:     int32(page),
		PageSize: int32(pageSize),
		SortBy:   sortBy,
		SortDesc: sortDesc,
		TagIds:   tagIDs,
	}

	if stageID != "" {
		grpcReq.StageId = &stageID
	}
	if contactID != "" {
		grpcReq.ContactId = &contactID
	}
	if companyID != "" {
		grpcReq.CompanyId = &companyID
	}
	if ownerID != "" {
		grpcReq.OwnerId = &ownerID
	}

	resp, err := h.crmClient.ListDeals(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateDealRequest struct {
	Name              *string                   `json:"name,omitempty"`
	Value             *float64                  `json:"value,omitempty"`
	Currency          *string                   `json:"currency,omitempty"`
	ContactID         *string                   `json:"contact_id,omitempty"`
	CompanyID         *string                   `json:"company_id,omitempty"`
	OwnerID           *string                   `json:"owner_id,omitempty"`
	ExpectedCloseDate *string                   `json:"expected_close_date,omitempty"`
	Notes             *string                   `json:"notes,omitempty"`
	CustomFields      []customFieldValueRequest `json:"custom_fields"`
}

func (h *GatewayHandler) HandleUpdateDeal(w http.ResponseWriter, r *http.Request) {
	dealID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(dealID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid deal id")
		return
	}

	var req updateDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateDealRequest{Id: dealID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Value != nil {
		grpcReq.Value = req.Value
	}
	if req.Currency != nil {
		grpcReq.Currency = req.Currency
	}
	if req.ContactID != nil {
		grpcReq.ContactId = req.ContactID
	}
	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}
	if req.OwnerID != nil {
		grpcReq.OwnerId = req.OwnerID
	}
	if req.ExpectedCloseDate != nil {
		grpcReq.ExpectedCloseDate = req.ExpectedCloseDate
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

	resp, err := h.crmClient.UpdateDeal(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleDeleteDeal(w http.ResponseWriter, r *http.Request) {
	dealID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(dealID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid deal id")
		return
	}

	_, err := h.crmClient.DeleteDeal(r.Context(), &crmv1.DeleteDealRequest{Id: dealID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "deal deleted"})
}

type moveDealToStageRequest struct {
	StageID string `json:"stage_id"`
}

func (h *GatewayHandler) HandleMoveDealToStage(w http.ResponseWriter, r *http.Request) {
	dealID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(dealID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid deal id")
		return
	}

	var req moveDealToStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.StageID == "" {
		response.Error(w, http.StatusBadRequest, "stage_id is required")
		return
	}

	resp, err := h.crmClient.MoveDealToStage(r.Context(), &crmv1.MoveDealToStageRequest{
		DealId:  dealID,
		StageId: req.StageID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type modifyDealTagsRequest struct {
	TagIDs []string `json:"tag_ids"`
}

func (h *GatewayHandler) HandleAddDealTags(w http.ResponseWriter, r *http.Request) {
	dealID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(dealID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid deal id")
		return
	}

	var req modifyDealTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Note: We don't have AddDealTags in proto, so we update the deal with tags
	// For now, respond with a not implemented error
	response.Error(w, http.StatusNotImplemented, "add deal tags not implemented via HTTP, use gRPC")
}

func (h *GatewayHandler) HandleRemoveDealTags(w http.ResponseWriter, r *http.Request) {
	dealID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(dealID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid deal id")
		return
	}

	var req modifyDealTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Note: We don't have RemoveDealTags in proto, so we update the deal with tags
	// For now, respond with a not implemented error
	response.Error(w, http.StatusNotImplemented, "remove deal tags not implemented via HTTP, use gRPC")
}

// ============================================================================
// CRM: Activities Handlers
// ============================================================================

type createActivityRequest struct {
	ActivityType string                    `json:"activity_type"`
	Subject      string                    `json:"subject"`
	Description  string                    `json:"description"`
	ContactID    *string                   `json:"contact_id,omitempty"`
	CompanyID    *string                   `json:"company_id,omitempty"`
	DealID       *string                   `json:"deal_id,omitempty"`
	AssignedTo   *string                   `json:"assigned_to,omitempty"`
	DueDate      string                    `json:"due_date"`
	TagIDs       []string                  `json:"tag_ids"`
	CustomFields []customFieldValueRequest `json:"custom_fields"`
}

func (h *GatewayHandler) HandleCreateActivity(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ActivityType == "" || req.Subject == "" {
		response.Error(w, http.StatusBadRequest, "activity_type and subject are required")
		return
	}

	grpcReq := &crmv1.CreateActivityRequest{
		ActivityType: req.ActivityType,
		Subject:      req.Subject,
		Description:  req.Description,
		DueDate:      req.DueDate,
		CreatedBy:    userID,
	}

	if req.ContactID != nil {
		grpcReq.ContactId = req.ContactID
	}
	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}
	if req.DealID != nil {
		grpcReq.DealId = req.DealID
	}
	if req.AssignedTo != nil {
		grpcReq.AssignedTo = req.AssignedTo
	}

	resp, err := h.crmClient.CreateActivity(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleGetActivity(w http.ResponseWriter, r *http.Request) {
	activityID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(activityID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid activity id")
		return
	}

	resp, err := h.crmClient.GetActivity(r.Context(), &crmv1.GetActivityRequest{Id: activityID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleListActivities(w http.ResponseWriter, r *http.Request) {
	activityType := r.URL.Query().Get("activity_type")
	contactID := r.URL.Query().Get("contact_id")
	companyID := r.URL.Query().Get("company_id")
	dealID := r.URL.Query().Get("deal_id")
	assignedTo := r.URL.Query().Get("assigned_to")
	isCompletedStr := r.URL.Query().Get("is_completed")
	sortBy := r.URL.Query().Get("sort_by")
	sortDesc := r.URL.Query().Get("sort_desc") == "true"
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

	grpcReq := &crmv1.ListActivitiesRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		SortBy:   sortBy,
		SortDesc: sortDesc,
	}

	if activityType != "" {
		grpcReq.ActivityType = &activityType
	}

	if contactID != "" {
		grpcReq.ContactId = &contactID
	}
	if companyID != "" {
		grpcReq.CompanyId = &companyID
	}
	if dealID != "" {
		grpcReq.DealId = &dealID
	}
	if assignedTo != "" {
		grpcReq.AssignedTo = &assignedTo
	}
	if isCompletedStr != "" {
		isCompleted := isCompletedStr == "true"
		grpcReq.IsCompleted = &isCompleted
	}

	resp, err := h.crmClient.ListActivities(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateActivityRequest struct {
	Subject      *string                   `json:"subject,omitempty"`
	Description  *string                   `json:"description,omitempty"`
	ContactID    *string                   `json:"contact_id,omitempty"`
	CompanyID    *string                   `json:"company_id,omitempty"`
	DealID       *string                   `json:"deal_id,omitempty"`
	AssignedTo   *string                   `json:"assigned_to,omitempty"`
	DueDate      *string                   `json:"due_date,omitempty"`
	CustomFields []customFieldValueRequest `json:"custom_fields"`
}

func (h *GatewayHandler) HandleUpdateActivity(w http.ResponseWriter, r *http.Request) {
	activityID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(activityID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid activity id")
		return
	}

	var req updateActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateActivityRequest{Id: activityID}
	if req.Subject != nil {
		grpcReq.Subject = req.Subject
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}
	if req.ContactID != nil {
		grpcReq.ContactId = req.ContactID
	}
	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}
	if req.DealID != nil {
		grpcReq.DealId = req.DealID
	}
	if req.AssignedTo != nil {
		grpcReq.AssignedTo = req.AssignedTo
	}
	if req.DueDate != nil {
		grpcReq.DueDate = req.DueDate
	}

	resp, err := h.crmClient.UpdateActivity(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleDeleteActivity(w http.ResponseWriter, r *http.Request) {
	activityID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(activityID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid activity id")
		return
	}

	_, err := h.crmClient.DeleteActivity(r.Context(), &crmv1.DeleteActivityRequest{Id: activityID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "activity deleted"})
}

func (h *GatewayHandler) HandleCompleteActivity(w http.ResponseWriter, r *http.Request) {
	activityID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(activityID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid activity id")
		return
	}

	resp, err := h.crmClient.CompleteActivity(r.Context(), &crmv1.CompleteActivityRequest{Id: activityID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type modifyActivityTagsRequest struct {
	TagIDs []string `json:"tag_ids"`
}

func (h *GatewayHandler) HandleAddActivityTags(w http.ResponseWriter, r *http.Request) {
	activityID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(activityID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid activity id")
		return
	}

	var req modifyActivityTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Note: Activity tags are managed via gRPC for now
	response.Error(w, http.StatusNotImplemented, "add activity tags not implemented via HTTP, use gRPC")
}

func (h *GatewayHandler) HandleRemoveActivityTags(w http.ResponseWriter, r *http.Request) {
	activityID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(activityID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid activity id")
		return
	}

	var req modifyActivityTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Note: Activity tags are managed via gRPC for now
	response.Error(w, http.StatusNotImplemented, "remove activity tags not implemented via HTTP, use gRPC")
}

// ============================================================================
// CRM: Search Handler
// ============================================================================

func (h *GatewayHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		response.Error(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	entityTypes := r.URL.Query()["types"]
	limit := 20

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	resp, err := h.crmClient.Search(r.Context(), &crmv1.SearchRequest{
		Query:       query,
		EntityTypes: entityTypes,
		Limit:       int32(limit),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// CRM: Saved Filters Handlers
// ============================================================================

type createSavedFilterRequest struct {
	Name       string `json:"name"`
	EntityType string `json:"entity_type"`
	FilterJSON string `json:"filter_json"`
	IsDefault  bool   `json:"is_default"`
}

func (h *GatewayHandler) HandleCreateSavedFilter(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createSavedFilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.EntityType == "" || req.FilterJSON == "" {
		response.Error(w, http.StatusBadRequest, "name, entity_type, and filter_json are required")
		return
	}

	resp, err := h.crmClient.CreateSavedFilter(r.Context(), &crmv1.CreateSavedFilterRequest{
		Name:       req.Name,
		EntityType: req.EntityType,
		FilterJson: req.FilterJSON,
		IsDefault:  req.IsDefault,
		CreatedBy:  userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleGetSavedFilter(w http.ResponseWriter, r *http.Request) {
	filterID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(filterID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid saved filter id")
		return
	}

	resp, err := h.crmClient.GetSavedFilter(r.Context(), &crmv1.GetSavedFilterRequest{Id: filterID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleListSavedFilters(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityType := r.URL.Query().Get("entity_type")

	resp, err := h.crmClient.ListSavedFilters(r.Context(), &crmv1.ListSavedFiltersRequest{
		EntityType: entityType,
		UserId:     userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateSavedFilterRequest struct {
	Name       *string `json:"name,omitempty"`
	FilterJSON *string `json:"filter_json,omitempty"`
	IsDefault  *bool   `json:"is_default,omitempty"`
}

func (h *GatewayHandler) HandleUpdateSavedFilter(w http.ResponseWriter, r *http.Request) {
	filterID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(filterID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid saved filter id")
		return
	}

	var req updateSavedFilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateSavedFilterRequest{Id: filterID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.FilterJSON != nil {
		grpcReq.FilterJson = req.FilterJSON
	}
	if req.IsDefault != nil {
		grpcReq.IsDefault = req.IsDefault
	}

	resp, err := h.crmClient.UpdateSavedFilter(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleDeleteSavedFilter(w http.ResponseWriter, r *http.Request) {
	filterID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(filterID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid saved filter id")
		return
	}

	_, err := h.crmClient.DeleteSavedFilter(r.Context(), &crmv1.DeleteSavedFilterRequest{Id: filterID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "saved filter deleted"})
}

// ============================================================================
// CRM: Reports Handlers
// ============================================================================

func (h *GatewayHandler) HandleGetPipelineReport(w http.ResponseWriter, r *http.Request) {
	ownerID := r.URL.Query().Get("owner_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" || endDate == "" {
		response.Error(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	grpcReq := &crmv1.GetPipelineReportRequest{
		StartDate: startDate,
		EndDate:   endDate,
	}
	if ownerID != "" {
		grpcReq.OwnerId = &ownerID
	}

	resp, err := h.crmClient.GetPipelineReport(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleGetConversionReport(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" || endDate == "" {
		response.Error(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	resp, err := h.crmClient.GetConversionReport(r.Context(), &crmv1.GetConversionReportRequest{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleGetActivityReport(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" || endDate == "" {
		response.Error(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	grpcReq := &crmv1.GetActivityReportRequest{
		StartDate: startDate,
		EndDate:   endDate,
	}
	if userID != "" {
		grpcReq.UserId = &userID
	}

	resp, err := h.crmClient.GetActivityReport(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Chat: Channels Handlers
// ============================================================================

type createChannelRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	IsPrivate        bool     `json:"is_private"`
	InitialMemberIDs []string `json:"initial_member_ids"`
}

func (h *GatewayHandler) HandleCreateChannel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	grpcReq := &chatv1.CreateChannelRequest{
		Name:             req.Name,
		IsPrivate:        req.IsPrivate,
		CreatedBy:        userID,
		InitialMemberIds: req.InitialMemberIDs,
	}
	if req.Description != "" {
		grpcReq.Description = &req.Description
	}

	resp, err := h.chatClient.CreateChannel(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleGetChannel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	resp, err := h.chatClient.GetChannel(r.Context(), &chatv1.GetChannelRequest{
		Id:     channelID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleListChannels(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	includeArchived := r.URL.Query().Get("include_archived") == "true"
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

	resp, err := h.chatClient.ListChannels(r.Context(), &chatv1.ListChannelsRequest{
		UserId:          userID,
		IncludeArchived: includeArchived,
		Page:            int32(page),
		PageSize:        int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateChannelRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (h *GatewayHandler) HandleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	var req updateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &chatv1.UpdateChannelRequest{
		Id:     channelID,
		UserId: userID,
	}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}

	resp, err := h.chatClient.UpdateChannel(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	_, err := h.chatClient.DeleteChannel(r.Context(), &chatv1.DeleteChannelRequest{
		Id:     channelID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "channel deleted"})
}

func (h *GatewayHandler) HandleArchiveChannel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	resp, err := h.chatClient.ArchiveChannel(r.Context(), &chatv1.ArchiveChannelRequest{
		Id:     channelID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Chat: Channel Membership Handlers
// ============================================================================

func (h *GatewayHandler) HandleJoinChannel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	resp, err := h.chatClient.JoinChannel(r.Context(), &chatv1.JoinChannelRequest{
		ChannelId: channelID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleLeaveChannel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	_, err := h.chatClient.LeaveChannel(r.Context(), &chatv1.LeaveChannelRequest{
		ChannelId: channelID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "left channel"})
}

func (h *GatewayHandler) HandleGetChannelMembers(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

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

	resp, err := h.chatClient.GetChannelMembers(r.Context(), &chatv1.GetChannelMembersRequest{
		ChannelId: channelID,
		Page:      int32(page),
		PageSize:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

func (h *GatewayHandler) HandleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.GetUserID(r.Context())
	if requesterID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	targetUserID := chi.URLParam(r, "userId")
	if _, err := uuid.Parse(targetUserID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role == "" {
		response.Error(w, http.StatusBadRequest, "role is required")
		return
	}

	resp, err := h.chatClient.UpdateMemberRole(r.Context(), &chatv1.UpdateMemberRoleRequest{
		ChannelId:       channelID,
		TargetUserId:    targetUserID,
		RequesterUserId: requesterID,
		NewRole:         req.Role,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Chat: Messages Handlers
// ============================================================================

type sendMessageRequest struct {
	Content         string  `json:"content"`
	ParentMessageID *string `json:"parent_message_id,omitempty"`
}

func (h *GatewayHandler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		response.Error(w, http.StatusBadRequest, "content is required")
		return
	}

	grpcReq := &chatv1.SendMessageRequest{
		ChannelId: channelID,
		Content:   req.Content,
		CreatedBy: userID,
	}

	if req.ParentMessageID != nil && *req.ParentMessageID != "" {
		if _, parseErr := uuid.Parse(*req.ParentMessageID); parseErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid parent_message_id")
			return
		}
		grpcReq.ParentMessageId = req.ParentMessageID
	}

	resp, err := h.chatClient.SendMessage(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	grpcReq := &chatv1.GetMessagesRequest{
		ChannelId: channelID,
		UserId:    userID,
		Limit:     int32(limit),
	}

	if before := r.URL.Query().Get("before"); before != "" {
		grpcReq.Before = &before
	}
	if after := r.URL.Query().Get("after"); after != "" {
		grpcReq.After = &after
	}

	resp, err := h.chatClient.GetMessages(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateMessageRequest struct {
	Content string `json:"content"`
}

func (h *GatewayHandler) HandleUpdateMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	messageID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(messageID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}

	var req updateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		response.Error(w, http.StatusBadRequest, "content is required")
		return
	}

	resp, err := h.chatClient.UpdateMessage(r.Context(), &chatv1.UpdateMessageRequest{
		Id:      messageID,
		UserId:  userID,
		Content: req.Content,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) HandleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	messageID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(messageID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}

	_, err := h.chatClient.DeleteMessage(r.Context(), &chatv1.DeleteMessageRequest{
		Id:     messageID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "message deleted"})
}

// ============================================================================
// Chat: DM Handlers (Sprint 2)
// ============================================================================

type getOrCreateDMRequest struct {
	OtherUserID string `json:"other_user_id"`
}

func (h *GatewayHandler) HandleGetOrCreateDM(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req getOrCreateDMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OtherUserID == "" {
		response.Error(w, http.StatusBadRequest, "other_user_id is required")
		return
	}

	if _, err := uuid.Parse(req.OtherUserID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid other_user_id")
		return
	}

	resp, err := h.chatClient.GetOrCreateDM(r.Context(), &chatv1.GetOrCreateDMRequest{
		UserId:      userID,
		OtherUserId: req.OtherUserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	statusCode := http.StatusOK
	if resp.Created {
		statusCode = http.StatusCreated
	}

	response.JSON(w, statusCode, resp)
}

func (h *GatewayHandler) HandleListDMs(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 20
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	resp, err := h.chatClient.ListDMs(r.Context(), &chatv1.ListDMsRequest{
		UserId:   userID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Chat: Thread Handlers (Sprint 2)
// ============================================================================

func (h *GatewayHandler) HandleGetThreadReplies(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	messageID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(messageID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	grpcReq := &chatv1.GetThreadRepliesRequest{
		ParentMessageId: messageID,
		UserId:          userID,
		Limit:           int32(limit),
	}

	if before := r.URL.Query().Get("before"); before != "" {
		grpcReq.Before = &before
	}
	if after := r.URL.Query().Get("after"); after != "" {
		grpcReq.After = &after
	}

	resp, err := h.chatClient.GetThreadReplies(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}
