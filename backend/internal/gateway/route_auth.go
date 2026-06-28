package gateway

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

// forgotRateLimitWindow is the sliding window for per-email forgot-password attempts.
const forgotRateLimitWindow = 10 * time.Minute

// forgotRateLimitMax is the maximum number of requests allowed per email per window.
const forgotRateLimitMax = 5

// forgotBucket tracks rate-limit state for a single email address. mu guards
// count/windowEnd: sync.Map protects the map itself but not concurrent access to
// a stored *forgotBucket, so two simultaneous requests for the same email would
// otherwise race on count.
type forgotBucket struct {
	mu        sync.Mutex
	count     int
	windowEnd time.Time
}

// AuthRoutes handles HTTP routes for the auth backend service.
type AuthRoutes struct {
	registry *ServiceRegistry
	// forgotLimiter provides per-email sliding-window rate limiting for the
	// forgot-password endpoint. In-memory; resets on gateway restart. Good
	// enough for pilot scale — upgrade to Redis-backed when needed.
	forgotLimiter sync.Map // key: email string → *forgotBucket
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

		// Password reset (public — no JWT required)
		r.Post("/forgot-password", a.HandleForgotPassword)
		r.Post("/reset-password", a.HandleResetPassword)

		// 2FA validation (uses pending_token, not JWT auth)
		r.Post("/2fa/validate", a.HandleValidate2FALogin)

		// Protected auth routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/me", a.HandleGetProfile)
			r.Patch("/profile", a.HandleUpdateProfile)
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
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (a *AuthRoutes) HandleRegister(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[registerRequest](w, r)
	if !ok {
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
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (a *AuthRoutes) HandleLogin(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[loginRequest](w, r)
	if !ok {
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
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (a *AuthRoutes) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[refreshRequest](w, r)
	if !ok {
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
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (a *AuthRoutes) HandleLogout(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[logoutRequest](w, r)
	if !ok {
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
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,min=1"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,min=1"`
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

	req, ok := decodeAndValidate[updateUserRequest](w, r)
	if !ok {
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
	RoleName string `json:"role_name" validate:"required,oneof=admin manager member"`
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

	req, ok := decodeAndValidate[assignRoleRequest](w, r)
	if !ok {
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

	req, ok := decodeAndValidate[assignRoleRequest](w, r)
	if !ok {
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

// updateProfileRequest is the self-service profile patch. All fields optional;
// user_id comes from the JWT context, never the body.
type updateProfileRequest struct {
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,max=100"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,max=100"`
	AvatarURL *string `json:"avatar_url,omitempty" validate:"omitempty,max=512"`
}

func (a *AuthRoutes) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[updateProfileRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateProfile(r.Context(), &authv1.UpdateProfileRequest{
		UserId:    userID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		AvatarUrl: req.AvatarURL,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

func (a *AuthRoutes) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[changePasswordRequest](w, r)
	if !ok {
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
// Password Reset Handlers
// ============================================================================

type forgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// allowForgotAttempt returns true if the email is under the rate limit.
// Thread-safe via sync.Map + per-bucket mutex.
func (a *AuthRoutes) allowForgotAttempt(email string) bool {
	// Normalize so mixed-case retries (User@x vs user@x) share one bucket.
	email = strings.ToLower(strings.TrimSpace(email))
	now := time.Now()
	actual, _ := a.forgotLimiter.LoadOrStore(email, &forgotBucket{windowEnd: now.Add(forgotRateLimitWindow)})
	b := actual.(*forgotBucket)

	// Guard count/windowEnd with the bucket's own mutex. Reset the fields in
	// place (rather than Store-ing a fresh bucket) so a concurrent caller holding
	// the same pointer observes the reset under the same lock.
	b.mu.Lock()
	defer b.mu.Unlock()
	if now.After(b.windowEnd) {
		// Window expired — reset.
		b.count = 1
		b.windowEnd = now.Add(forgotRateLimitWindow)
		return true
	}
	b.count++
	return b.count <= forgotRateLimitMax
}

func (a *AuthRoutes) HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[forgotPasswordRequest](w, r)
	if !ok {
		return
	}

	if !a.allowForgotAttempt(req.Email) {
		w.Header().Set("Retry-After", "600")
		response.Error(w, http.StatusTooManyRequests, "too many password reset requests, try again later")
		return
	}

	// Always return 200 — enumeration-safe.
	_, _ = client.RequestPasswordReset(r.Context(), &authv1.RequestPasswordResetRequest{
		Email: req.Email,
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "If an account with that email exists, a reset link has been sent.",
	})
}

type resetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

func (a *AuthRoutes) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[resetPasswordRequest](w, r)
	if !ok {
		return
	}

	_, err = client.ResetPassword(r.Context(), &authv1.ResetPasswordRequest{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "password reset successful"})
}

// ============================================================================
// Invitation Handlers
// ============================================================================

type createInvitationRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin manager member"`
}

func (a *AuthRoutes) HandleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createInvitationRequest](w, r)
	if !ok {
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
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
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

	req, ok := decodeAndValidate[acceptInvitationRequest](w, r)
	if !ok {
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
	Code string `json:"code" validate:"required,len=6"`
}

func (a *AuthRoutes) HandleVerify2FA(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[verify2FARequest](w, r)
	if !ok {
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
	PendingToken   string `json:"pending_token" validate:"required"`
	Code           string `json:"code" validate:"required"`
	IsRecoveryCode bool   `json:"is_recovery_code"`
}

func (a *AuthRoutes) HandleValidate2FALogin(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[validate2FALoginRequest](w, r)
	if !ok {
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
	Code string `json:"code" validate:"required"`
}

func (a *AuthRoutes) HandleDisable2FA(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[disable2FARequest](w, r)
	if !ok {
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
	Code string `json:"code" validate:"required"`
}

func (a *AuthRoutes) HandleRegenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[regenerateCodesRequest](w, r)
	if !ok {
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
	UserID string `json:"user_id" validate:"required,uuid"`
	Reason string `json:"reason" validate:"required"`
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

	req, ok := decodeAndValidate[adminReset2FARequest](w, r)
	if !ok {
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
	RoleName        string `json:"role_name" validate:"required,oneof=admin manager member"`
	Enforced        bool   `json:"enforced"`
	GracePeriodDays int32  `json:"grace_period_days" validate:"omitempty,gte=0,lte=365"`
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

	req, ok := decodeAndValidate[updateTwoFactorPolicyRequest](w, r)
	if !ok {
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
