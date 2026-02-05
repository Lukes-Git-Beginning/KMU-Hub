package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

type GatewayHandler struct {
	authClient     authv1.AuthServiceClient
	healthCheckers []health.Checker
}

func NewGatewayHandler(authClient authv1.AuthServiceClient, healthCheckers []health.Checker) *GatewayHandler {
	return &GatewayHandler{
		authClient:     authClient,
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
