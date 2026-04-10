package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

// AuthRoutes handles HTTP routes for the auth backend service.
type AuthRoutes struct {
	registry *ServiceRegistry
}

// NewAuthRoutes creates a new AuthRoutes with the given service registry.
func NewAuthRoutes(registry *ServiceRegistry) *AuthRoutes {
	return &AuthRoutes{registry: registry}
}

// ServiceName returns the backend service name.
func (a *AuthRoutes) ServiceName() string { return "auth" }

// getAuthClient lazily obtains a gRPC client for the auth service.
func (a *AuthRoutes) getAuthClient() (authv1.AuthServiceClient, error) {
	conn, err := a.registry.GetConnection("auth")
	if err != nil {
		return nil, err
	}
	return authv1.NewAuthServiceClient(conn), nil
}

// RegisterRoutes registers all auth, user, invitation, 2FA, and session HTTP routes.
func (a *AuthRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Public auth routes
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", a.HandleRegister)
		r.Post("/login", a.HandleLogin)
		r.Post("/refresh", a.HandleRefresh)
		r.Post("/logout", a.HandleLogout)

		// 2FA validation (uses pending_token, not JWT auth)
		r.Post("/2fa/validate", a.HandleValidate2FALogin)

		// Protected auth routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/me", a.HandleGetProfile)
			r.Post("/change-password", a.HandleChangePassword)

			// 2FA management (protected)
			r.Post("/2fa/setup", a.HandleSetup2FA)
			r.Post("/2fa/verify", a.HandleVerify2FA)
			r.Post("/2fa/disable", a.HandleDisable2FA)
			r.Post("/2fa/regenerate-codes", a.HandleRegenerateRecoveryCodes)

			// 2FA admin operations
			r.With(middleware.RequireRole("admin")).Get("/2fa/policy", a.HandleGetTwoFactorPolicy)
			r.With(middleware.RequireRole("admin")).Put("/2fa/policy", a.HandleUpdateTwoFactorPolicy)
			r.With(middleware.RequireRole("admin")).Post("/2fa/admin-reset", a.HandleAdminReset2FA)

			// Session management
			r.Get("/sessions", a.HandleListSessions)
			r.Delete("/sessions/{id}", a.HandleTerminateSession)
			r.Delete("/sessions", a.HandleTerminateAllSessions)

			// Admin session management
			r.With(middleware.RequireRole("admin")).Get("/sessions/all", a.HandleListAllSessions)
		})
	})

	// Protected user routes
	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequireRole("admin", "manager")).Get("/", a.HandleListUsers)
		r.Get("/{id}", a.HandleGetUser)
		r.With(middleware.RequireRole("admin")).Put("/{id}", a.HandleUpdateUser)
		r.With(middleware.RequireRole("admin")).Post("/{id}/roles", a.HandleAssignRole)
		r.With(middleware.RequireRole("admin")).Delete("/{id}/roles", a.HandleRemoveRole)
	})

	// Invitation routes
	r.Route("/api/v1/invitations", func(r chi.Router) {
		// Public: accept invitation (no auth)
		r.Post("/{token}/accept", a.HandleAcceptInvitation)

		// Protected: create, list, cancel invitations
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.With(middleware.RequireRole("admin", "manager")).Post("/", a.HandleCreateInvitation)
			r.With(middleware.RequireRole("admin", "manager")).Get("/", a.HandleListInvitations)
			r.With(middleware.RequireRole("admin")).Delete("/{id}", a.HandleCancelInvitation)
		})
	})
}

// ============================================================================
// Auth Handlers
// ============================================================================

type registerRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (a *AuthRoutes) HandleRegister(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	resp, err := client.Register(r.Context(), &authv1.RegisterRequest{
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

func (a *AuthRoutes) HandleLogin(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	resp, err := client.Login(r.Context(), &authv1.LoginRequest{
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

func (a *AuthRoutes) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.RefreshToken(r.Context(), &authv1.RefreshTokenRequest{
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

func (a *AuthRoutes) HandleLogout(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.Logout(r.Context(), &authv1.LogoutRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (a *AuthRoutes) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetUser(r.Context(), &authv1.GetUserRequest{UserId: userID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (a *AuthRoutes) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	resp, err := client.ListUsers(r.Context(), &authv1.ListUsersRequest{
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

func (a *AuthRoutes) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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

	resp, err := client.UpdateUser(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type assignRoleRequest struct {
	RoleName string `json:"role_name"`
}

func (a *AuthRoutes) HandleAssignRole(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req assignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.AssignRole(r.Context(), &authv1.AssignRoleRequest{
		UserId:   userID,
		RoleName: req.RoleName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "role assigned"})
}

func (a *AuthRoutes) HandleRemoveRole(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req assignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.RemoveRole(r.Context(), &authv1.RemoveRoleRequest{
		UserId:   userID,
		RoleName: req.RoleName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "role removed"})
}

func (a *AuthRoutes) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetProfile(r.Context(), &authv1.GetProfileRequest{UserId: userID})
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

func (a *AuthRoutes) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		response.Error(w, http.StatusBadRequest, "old_password and new_password are required")
		return
	}

	_, err = client.ChangePassword(r.Context(), &authv1.ChangePasswordRequest{
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

// ============================================================================
// Invitation Handlers
// ============================================================================

type createInvitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (a *AuthRoutes) HandleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Role == "" {
		response.Error(w, http.StatusBadRequest, "email and role are required")
		return
	}

	resp, err := client.CreateInvitation(r.Context(), &authv1.CreateInvitationRequest{
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

func (a *AuthRoutes) HandleListInvitations(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	resp, err := client.ListInvitations(r.Context(), &authv1.ListInvitationsRequest{})
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

func (a *AuthRoutes) HandleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

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

	resp, err := client.AcceptInvitation(r.Context(), &authv1.AcceptInvitationRequest{
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

func (a *AuthRoutes) HandleCancelInvitation(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	invitationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.CancelInvitation(r.Context(), &authv1.CancelInvitationRequest{
		InvitationId: invitationID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "invitation cancelled"})
}

// ============================================================================
// 2FA Handlers
// ============================================================================

func (a *AuthRoutes) HandleSetup2FA(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.Setup2FA(r.Context(), &authv1.Setup2FARequest{UserId: userID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type verify2FARequest struct {
	Code string `json:"code"`
}

func (a *AuthRoutes) HandleVerify2FA(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		response.Error(w, http.StatusBadRequest, "code is required")
		return
	}

	resp, err := client.Verify2FA(r.Context(), &authv1.Verify2FARequest{
		UserId: userID,
		Code:   req.Code,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type validate2FALoginRequest struct {
	PendingToken   string `json:"pending_token"`
	Code           string `json:"code"`
	IsRecoveryCode bool   `json:"is_recovery_code"`
}

func (a *AuthRoutes) HandleValidate2FALogin(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	var req validate2FALoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.PendingToken == "" || req.Code == "" {
		response.Error(w, http.StatusBadRequest, "pending_token and code are required")
		return
	}

	resp, err := client.Validate2FALogin(r.Context(), &authv1.Validate2FALoginRequest{
		PendingToken:   req.PendingToken,
		Code:           req.Code,
		IsRecoveryCode: req.IsRecoveryCode,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type disable2FARequest struct {
	Code string `json:"code"`
}

func (a *AuthRoutes) HandleDisable2FA(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req disable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.Disable2FA(r.Context(), &authv1.Disable2FARequest{
		UserId: userID,
		Code:   req.Code,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "2fa disabled"})
}

type regenerateCodesRequest struct {
	Code string `json:"code"`
}

func (a *AuthRoutes) HandleRegenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req regenerateCodesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.RegenerateRecoveryCodes(r.Context(), &authv1.RegenerateRecoveryCodesRequest{
		UserId: userID,
		Code:   req.Code,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type adminReset2FARequest struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason"`
}

func (a *AuthRoutes) HandleAdminReset2FA(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	adminID := middleware.GetUserID(r.Context())
	if adminID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req adminReset2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" || req.Reason == "" {
		response.Error(w, http.StatusBadRequest, "user_id and reason are required")
		return
	}

	_, err = client.AdminReset2FA(r.Context(), &authv1.AdminReset2FARequest{
		UserId:  req.UserID,
		Reason:  req.Reason,
		AdminId: adminID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "2fa reset"})
}

func (a *AuthRoutes) HandleGetTwoFactorPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	roleName := r.URL.Query().Get("role_name")

	resp, err := client.GetTwoFactorPolicy(r.Context(), &authv1.GetTwoFactorPolicyRequest{
		RoleName: roleName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateTwoFactorPolicyRequest struct {
	RoleName        string `json:"role_name"`
	Enforced        bool   `json:"enforced"`
	GracePeriodDays int32  `json:"grace_period_days"`
}

func (a *AuthRoutes) HandleUpdateTwoFactorPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	adminID := middleware.GetUserID(r.Context())
	if adminID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req updateTwoFactorPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RoleName == "" {
		response.Error(w, http.StatusBadRequest, "role_name is required")
		return
	}

	resp, err := client.UpdateTwoFactorPolicy(r.Context(), &authv1.UpdateTwoFactorPolicyRequest{
		RoleName:        req.RoleName,
		Enforced:        req.Enforced,
		GracePeriodDays: req.GracePeriodDays,
		UpdatedBy:       adminID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Session Handlers
// ============================================================================

func (a *AuthRoutes) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.ListSessions(r.Context(), &authv1.ListSessionsRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (a *AuthRoutes) HandleListAllSessions(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	// Admin endpoint: list sessions for a specific user or all users
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		response.Error(w, http.StatusBadRequest, "user_id query parameter is required")
		return
	}

	resp, err := client.ListSessions(r.Context(), &authv1.ListSessionsRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (a *AuthRoutes) HandleTerminateSession(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	sessionID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.TerminateSession(r.Context(), &authv1.TerminateSessionRequest{
		SessionId: sessionID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "session terminated"})
}

func (a *AuthRoutes) HandleTerminateAllSessions(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	// Optionally keep current session alive
	currentSessionID := r.URL.Query().Get("current_session_id")

	resp, err := client.TerminateAllSessions(r.Context(), &authv1.TerminateAllSessionsRequest{
		UserId:           userID,
		CurrentSessionId: currentSessionID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}
