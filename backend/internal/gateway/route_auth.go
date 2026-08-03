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
			r.Get("/me/permissions", a.HandleGetMyPermissions)
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

	// Role administration (RBAC Phase 1 wave 1b). roles:manage is the coarse
	// legacy key every admin token has always carried (migration 000002);
	// admin:role:read is the fine-grained successor seeded for admin/it_admin/
	// hr_admin (migration 000256). RequirePermissionAny keeps a session minted
	// before 000256 working until it refreshes, instead of a hard 403.
	r.Route("/api/v1/admin/roles", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermissionAny(
			[2]string{"roles", "manage"},
			[2]string{"admin:role", "read"},
		)).Get("/", a.HandleListRoles)
		r.With(middleware.RequirePermissionAny(
			[2]string{"roles", "manage"},
			[2]string{"admin:role", "create"},
		)).Post("/", a.HandleCreateRole)
		r.With(middleware.RequirePermissionAny(
			[2]string{"roles", "manage"},
			[2]string{"admin:role", "edit"},
		)).Patch("/{id}", a.HandleUpdateRole)
		r.With(middleware.RequirePermissionAny(
			[2]string{"roles", "manage"},
			[2]string{"admin:role", "delete"},
		)).Delete("/{id}", a.HandleDeleteRole)
		r.With(middleware.RequirePermissionAny(
			[2]string{"roles", "manage"},
			[2]string{"admin:role", "read"},
		)).Get("/{id}/permissions", a.HandleGetRolePermissions)
		r.With(middleware.RequirePermissionAny(
			[2]string{"roles", "manage"},
			[2]string{"admin:role", "edit"},
		)).Put("/{id}/permissions", a.HandleSetRolePermissions)
	})

	// Protected user routes
	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequireRole("admin", "manager")).Get("/", a.HandleListUsers)
		r.Get("/{id}", a.HandleGetUser)
		r.With(middleware.RequireRole("admin")).Put("/{id}", a.HandleUpdateUser)
		// Role assignment, the same additive guard pair as /admin/roles: every
		// admin token carries roles:manage (migration 000002 grants the admin
		// preset the full catalogue), admin:role:assign is the fine successor
		// seeded for admin/it_admin/hr_admin in 000256. The pair widens the
		// route rather than narrowing it — hr_admin assigns roles without being
		// allowed to edit them, which the previous RequireRole("admin") could
		// not express.
		r.With(middleware.RequirePermissionAny(
			[2]string{"roles", "manage"},
			[2]string{"admin:role", "assign"},
		)).Post("/{id}/roles", a.HandleAssignRole)
		r.With(middleware.RequirePermissionAny(
			[2]string{"roles", "manage"},
			[2]string{"admin:role", "assign"},
		)).Delete("/{id}/roles/{roleId}", a.HandleRemoveRole)
	})

	// Effective-permission audit view of any account, and the account roster
	// itself. Lives under /admin/users rather than /users because that is
	// where the RBAC/account admin frontend addresses both (rbac-client.ts,
	// useAdminUsers.ts) — RequireRole("admin") throughout, since the roster
	// carries every tenant member's e-mail address and rights.
	r.Route("/api/v1/admin/users", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequireRole("admin")).Get("/", a.HandleListAdminUsers)
		r.With(middleware.RequireRole("admin")).Post("/invite", a.HandleInviteAdminUser)
		r.With(middleware.RequireRole("admin")).Patch("/{id}", a.HandleUpdateAdminUser)
		r.With(middleware.RequireRole("admin")).Post("/{id}/resend-invite", a.HandleResendAdminUserInvite)
		r.With(middleware.RequireRole("admin")).Get("/{id}/permissions", a.HandleGetUserPermissions)
		// Per-user overrides (RBAC R-6) guard with their own catalogue key
		// rather than RequireRole("admin") like their neighbours: fine-tuning
		// an individual is deliberately NOT part of the hr_admin delegation
		// that hands out roles (R6 briefing §0.3), and only the admin preset
		// carries admin:user_override:manage since migration 000256. A key
		// instead of a role name also lets a tenant that clones the preset
		// into its own "Geschäftsführung" keep the right.
		r.With(middleware.RequirePermission("admin:user_override", "manage")).Get("/{id}/overrides", a.HandleGetUserOverrides)
		r.With(middleware.RequirePermission("admin:user_override", "manage")).Put("/{id}/overrides", a.HandleSetUserOverrides)
		r.With(middleware.RequirePermission("admin:user_override", "manage")).Delete("/{id}/overrides", a.HandleClearUserOverrides)
	})

	// Tenant provisioning. Not RequireRole("admin"): a tenant admin
	// administers their own tenant, creating one is a platform operation.
	// tenants:write is held by the platform_admin role alone (migration
	// 000251), and that role cannot be handed out over the API — the invite
	// and role-assign handlers accept oneof=admin manager member.
	r.Route("/api/v1/tenants", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tenants", "write")).Post("/", a.HandleProvisionTenant)
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

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

// adminUserBody mirrors AdminUser in admin-types.ts. Two fields carry a
// deliberate gap, documented rather than invented:
//   - jobTitle is always "" — the backend has no job-title data on an
//     account (that lives in the HR employee record, a different service,
//     addressed by a different id). Inventing one would be worse than empty.
//   - hasOverrides is omitted (omitempty): per-user permission overrides
//     (R-6) do not exist server-side yet.
type adminUserBody struct {
	ID           string   `json:"id"`
	FirstName    string   `json:"firstName"`
	LastName     string   `json:"lastName"`
	Email        string   `json:"email"`
	JobTitle     string   `json:"jobTitle"`
	Roles        []string `json:"roles"`
	HasOverrides bool     `json:"hasOverrides,omitempty"`
	Status       string   `json:"status"`
	LastLoginAt  *string  `json:"lastLoginAt"`
	InvitedAt    *string  `json:"invitedAt"`
}

type adminUsersBody struct {
	Users []adminUserBody `json:"users"`
}

func toAdminUserBody(u *authv1.AdminUserInfo) adminUserBody {
	roles := u.RoleIds
	if roles == nil {
		roles = []string{}
	}
	return adminUserBody{
		ID:          u.Id,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Email:       u.Email,
		Roles:       roles,
		Status:      u.Status,
		LastLoginAt: nullIfEmpty(u.LastLoginAt),
		InvitedAt:   nullIfEmpty(u.InvitedAt),
	}
}

// HandleListAdminUsers returns the tenant's account roster: every real
// account plus every still-open invitation, merged into the one list the
// account admin surface shows.
func (a *AuthRoutes) HandleListAdminUsers(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	resp, err := client.ListAdminUsers(r.Context(), &authv1.ListAdminUsersRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	body := adminUsersBody{Users: make([]adminUserBody, len(resp.Users))}
	for i, u := range resp.Users {
		body.Users[i] = toAdminUserBody(u)
	}

	response.JSON(w, http.StatusOK, body)
}

// adminUserResultBody is the answer of the three writing roster routes. The
// frontend reads `user` alone (useAdminUsers.ts); inviteToken rides along on
// the two invitation routes because nothing server-side dispatches invitation
// mail — the caller builds the accept link, exactly as POST /api/v1/invitations
// already works.
type adminUserResultBody struct {
	User        adminUserBody `json:"user"`
	InviteToken string        `json:"inviteToken,omitempty"`
}

func respondAdminUser(w http.ResponseWriter, code int, resp *authv1.AdminUserResponse) {
	response.JSON(w, code, adminUserResultBody{
		User:        toAdminUserBody(resp.User),
		InviteToken: resp.InviteToken,
	})
}

type inviteAdminUserRequest struct {
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"firstName" validate:"omitempty,max=100"`
	LastName  string `json:"lastName" validate:"omitempty,max=100"`
	// No oneof on the ids: role ids are resolved dynamically against the roles
	// table, which is the only way a custom role can ever be invited. min=1
	// because an account with no role has no rights and cannot fix that itself.
	Roles []string `json:"roles" validate:"required,min=1,dive,uuid"`
}

// HandleInviteAdminUser opens an invitation for an address and answers with the
// roster row it produced, so the surface can insert it without refetching.
func (a *AuthRoutes) HandleInviteAdminUser(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[inviteAdminUserRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.InviteAdminUser(r.Context(), &authv1.InviteAdminUserRequest{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		RoleIds:   req.Roles,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondAdminUser(w, http.StatusCreated, resp)
}

type updateAdminUserRequest struct {
	// A pointer to a slice, not a slice: an absent `roles` leaves the account's
	// roles alone, while `"roles": []` strips them. Collapsing the two would
	// make every status-only PATCH silently clear the account's rights.
	Roles  *[]string `json:"roles" validate:"omitempty,dive,uuid"`
	Status *string   `json:"status" validate:"omitempty,oneof=active deactivated"`
}

// HandleUpdateAdminUser changes an account's roles and/or its active state.
func (a *AuthRoutes) HandleUpdateAdminUser(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[updateAdminUserRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &authv1.UpdateAdminUserRequest{
		UserId: chi.URLParam(r, "id"),
		Status: req.Status,
	}
	if req.Roles != nil {
		grpcReq.HasRoleIds = true
		grpcReq.RoleIds = *req.Roles
	}

	resp, err := client.UpdateAdminUser(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondAdminUser(w, http.StatusOK, resp)
}

// HandleResendAdminUserInvite re-issues a pending invitation. The path id is
// the INVITATION id, which is what an invited roster row carries — no account
// exists for it yet.
func (a *AuthRoutes) HandleResendAdminUserInvite(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	resp, err := client.ResendAdminUserInvite(r.Context(), &authv1.ResendAdminUserInviteRequest{
		InvitationId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondAdminUser(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

// assignRoleRequest mirrors AssignRoleInput in rbac-types.ts. The role travels
// as an id, not a name: the previous `role_name` with its
// `oneof=admin manager member` tag could only ever name the three seed presets,
// which made every custom role of wave 1b structurally unassignable. Whether
// the id exists and whether the caller may see it is decided against the roles
// table in the service — a static tag cannot know a tenant's own roles.
type assignRoleRequest struct {
	RoleID string `json:"roleId" validate:"required,uuid"`
}

// userRolesResponseBody mirrors UserRolesResponse in rbac-types.ts: the role
// ids the account holds after the mutation. Built through toUserRolesBody so
// an account left without any role marshals as [], not null.
type userRolesResponseBody struct {
	Roles []string `json:"roles"`
}

func toUserRolesBody(roleIDs []string) userRolesResponseBody {
	if roleIDs == nil {
		return userRolesResponseBody{Roles: []string{}}
	}
	return userRolesResponseBody{Roles: roleIDs}
}

// HandleAssignRole grants a role to an account. Roles are n:m — an assignment
// adds to what the account already holds instead of replacing it.
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

	resp, err := client.AssignUserRole(r.Context(), &authv1.AssignUserRoleRequest{
		UserId: userID,
		RoleId: req.RoleID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toUserRolesBody(resp.RoleIds))
}

// HandleRemoveRole takes one role off an account. The role travels in the path
// rather than in a body: a DELETE with a payload is awkward for clients, and
// the frontend addresses the assignment as a resource of its own
// (rbac-client.ts, removeUserRole).
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

	roleID, ok := validateUUIDParam(w, r, "roleId")
	if !ok {
		return
	}

	resp, err := client.RevokeUserRole(r.Context(), &authv1.RevokeUserRoleRequest{
		UserId: userID,
		RoleId: roleID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toUserRolesBody(resp.RoleIds))
}

// ============================================================================
// Effective Permissions Handlers
// ============================================================================

// effectivePermissionsBody is the wire shape the RBAC frontend consumes
// (rbac-types.ts, EffectivePermissionsResponse). It is deliberately not the
// proto response: capabilities travels as a map keyed by the capability key,
// while the proto carries a repeated list. A key that is absent means denied
// (default-deny), so an entry with an empty scope is never emitted.
type effectivePermissionsBody struct {
	Permissions effectivePermissions `json:"permissions"`
}

type effectivePermissions struct {
	Roles        []effectiveRole                `json:"roles"`
	Capabilities map[string]effectiveCapability `json:"capabilities"`
	// Both optional in rbac-types.ts and both omitted whenever they carry
	// nothing — for ?base=1 and for every account without an override. That is
	// not a shortcut: it makes the answer for an account without overrides
	// byte-identical to the one this route gave before R-6 existed, and the
	// frontend defaults both (`deniedByOverride = []`, `hasOverrides = false`).
	HasOverrides     bool                   `json:"hasOverrides,omitempty"`
	DeniedByOverride []deniedCapabilityBody `json:"deniedByOverride,omitempty"`
}

// deniedCapabilityBody is a key the roles grant and a deny override revokes.
// roleScope is what the roles would have given — the effective view strikes
// the row through and still shows it.
type deniedCapabilityBody struct {
	Key       string   `json:"key"`
	RoleScope string   `json:"roleScope"`
	Sources   []string `json:"sources"`
}

// effectiveRole spells isSystem in camelCase because the frontend type does
// (rbac-types.ts, Role). This block is app-owned contract rather than proto
// passthrough — do not "correct" it to snake_case.
type effectiveRole struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsSystem bool   `json:"isSystem"`
	Color    string `json:"color"`
}

type effectiveCapability struct {
	Scope string `json:"scope"`
	// Role IDs, resolvable against the roles list of the same response — that
	// is the lookup the provenance chip in EffectivePermissionsView performs.
	Sources []string `json:"sources"`
}

func toEffectivePermissionsBody(resp *authv1.GetEffectivePermissionsResponse) effectivePermissionsBody {
	roles := make([]effectiveRole, len(resp.Roles))
	for i, role := range resp.Roles {
		roles[i] = effectiveRole{
			ID:       role.Id,
			Name:     role.Name,
			IsSystem: role.IsSystem,
			Color:    role.Color,
		}
	}

	caps := make(map[string]effectiveCapability, len(resp.Capabilities))
	for _, c := range resp.Capabilities {
		sources := c.Sources
		if sources == nil {
			// proto3 drops an empty repeated field; the frontend iterates this
			// unconditionally, so it must marshal as [] and never as null.
			sources = []string{}
		}
		caps[c.Key] = effectiveCapability{Scope: c.Scope, Sources: sources}
	}

	var denied []deniedCapabilityBody
	for _, d := range resp.DeniedByOverride {
		sources := d.Sources
		if sources == nil {
			sources = []string{}
		}
		denied = append(denied, deniedCapabilityBody{
			Key:       d.Key,
			RoleScope: d.RoleScope,
			Sources:   sources,
		})
	}

	return effectivePermissionsBody{
		Permissions: effectivePermissions{
			Roles:            roles,
			Capabilities:     caps,
			HasOverrides:     resp.HasOverrides,
			DeniedByOverride: denied,
		},
	}
}

// HandleGetMyPermissions returns the authenticated account's resolved
// capabilities. No permission guard on purpose: every signed-in user may read
// their own rights — the UI needs them before it can hide anything, and a new
// guard here would demand a seed of its own.
func (a *AuthRoutes) HandleGetMyPermissions(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	// Empty user_id means "me": the auth service resolves the caller from the
	// identity metadata the tenant interceptor forwards, so no client-supplied
	// id is ever trusted on this route.
	resp, err := client.GetEffectivePermissions(r.Context(), &authv1.GetEffectivePermissionsRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toEffectivePermissionsBody(resp))
}

// HandleGetUserPermissions returns any account's resolved capabilities for the
// admin audit view.
//
// Fetching the target user first is a security step, not a convenience:
// user_roles carries neither tenant_id nor an RLS policy, so the resolver
// filters the roles but not the membership — handed a foreign tenant's user id
// it would resolve that user's preset roles just fine. GetUser runs under the
// users RLS policy and turns a foreign id into a clean 404.
func (a *AuthRoutes) HandleGetUserPermissions(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if _, err := client.GetUser(r.Context(), &authv1.GetUserRequest{UserId: userID}); err != nil {
		respondGRPCError(w, err)
		return
	}

	// ?base=1 asks for the role union without the account's per-user overrides
	// (RBAC R-6) — the inherited baseline the override editor greys out behind
	// the deviations. Only the exact "1" switches it: an unknown value means a
	// caller who did not mean to ask, and the effective set is the safer answer
	// to hand someone who guessed wrong.
	base := r.URL.Query().Get("base") == "1"

	resp, err := client.GetEffectivePermissions(r.Context(), &authv1.GetEffectivePermissionsRequest{
		UserId: userID,
		Base:   base,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toEffectivePermissionsBody(resp))
}

// ============================================================================
// Role Administration Handlers (RBAC Phase 1 wave 1b)
// ============================================================================

// roleBody is the wire shape the RBAC frontend consumes (rbac-types.ts,
// Role). It is app-owned rather than proto passthrough: tenantId/basedOn must
// render as JSON null on a system preset, which proto3's empty string cannot
// express, and the frontend spells every field camelCase.
type roleBody struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	TenantID        *string `json:"tenantId"`
	BasedOn         *string `json:"basedOn"`
	IsSystem        bool    `json:"isSystem"`
	Color           string  `json:"color"`
	MemberCount     int32   `json:"memberCount"`
	CapabilityCount int32   `json:"capabilityCount"`
}

type rolesBody struct {
	Roles []roleBody `json:"roles"`
}

// roleResponseBody wraps a single role — the frontend's RoleResponse reads
// `resp.role`, so the entity must not go out bare.
type roleResponseBody struct {
	Role roleBody `json:"role"`
}

// nullIfEmpty turns the proto3 zero value for a nullable string field back
// into JSON null. preset_id is empty on presets themselves as well as on
// every role with no clone lineage — both cases are "no value", not "".
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toRoleBody(r *authv1.Role) roleBody {
	return roleBody{
		ID:              r.Id,
		Name:            r.Name,
		Description:     r.Description,
		TenantID:        nullIfEmpty(r.TenantId),
		BasedOn:         nullIfEmpty(r.PresetId),
		IsSystem:        r.IsSystem,
		Color:           r.Color,
		MemberCount:     r.MemberCount,
		CapabilityCount: r.CapabilityCount,
	}
}

// HandleListRoles returns the system presets plus the calling tenant's custom
// roles.
func (a *AuthRoutes) HandleListRoles(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	resp, err := client.ListRoles(r.Context(), &authv1.ListRolesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	body := rolesBody{Roles: make([]roleBody, len(resp.Roles))}
	for i, role := range resp.Roles {
		body.Roles[i] = toRoleBody(role)
	}

	response.JSON(w, http.StatusOK, body)
}

// createRoleRequest mirrors CreateRoleInput in rbac-types.ts. The length
// bounds match the columns (name VARCHAR(50), color VARCHAR(40)) so an
// oversized value fails as a 400 instead of a 22001 turned 500. name and
// basedOn carry no `required` tag on purpose — they are checked below to
// answer with the 422 the frontend's contract specifies for a missing
// mandatory field, rather than decodeAndValidate's 400.
type createRoleRequest struct {
	Name        string `json:"name" validate:"omitempty,max=50"`
	Description string `json:"description" validate:"omitempty,max=1000"`
	Color       string `json:"color" validate:"omitempty,max=40"`
	BasedOn     string `json:"basedOn" validate:"omitempty,uuid"`
}

// HandleCreateRole clones an existing role into a new tenant-owned one.
func (a *AuthRoutes) HandleCreateRole(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createRoleRequest](w, r)
	if !ok {
		return
	}
	if strings.TrimSpace(req.Name) == "" || req.BasedOn == "" {
		response.Error(w, http.StatusUnprocessableEntity, "not_found")
		return
	}

	resp, err := client.CreateRole(r.Context(), &authv1.CreateRoleRequest{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		BasedOn:     req.BasedOn,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, roleResponseBody{Role: toRoleBody(resp.Role)})
}

// updateRoleRequest mirrors UpdateRoleInput in rbac-types.ts: every field is
// optional, absence means "leave unchanged". The length bounds match
// createRoleRequest's — same columns, same overflow-to-500 concern.
type updateRoleRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,max=50"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
	Color       *string `json:"color,omitempty" validate:"omitempty,max=40"`
}

// HandleUpdateRole renames/re-describes/recolors a tenant-owned role.
func (a *AuthRoutes) HandleUpdateRole(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	roleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateRoleRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateRole(r.Context(), &authv1.UpdateRoleRequest{
		RoleId:      roleID,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, roleResponseBody{Role: toRoleBody(resp.Role)})
}

// HandleDeleteRole removes a tenant-owned role. Presets are immutable and a
// role still worn by a member cannot be deleted out from under its holders —
// both answer through respondGRPCError, not a bare 500.
func (a *AuthRoutes) HandleDeleteRole(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	roleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if _, err := client.DeleteRole(r.Context(), &authv1.DeleteRoleRequest{RoleId: roleID}); err != nil {
		respondGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// roleGrantBody is one capability grant on the wire — RoleGrants in
// rbac-types.ts is a plain `Record<string, {scope}>`, the key never travels
// as a field of its own.
type roleGrantBody struct {
	Scope string `json:"scope"`
}

// rolePermissionsResponseBody mirrors RolePermissionsResponse in
// rbac-types.ts. Grants is built through toRoleGrantsBody so it always
// marshals as `{}`, never `null`, on a role with zero grants.
type rolePermissionsResponseBody struct {
	RoleID string                   `json:"roleId"`
	Grants map[string]roleGrantBody `json:"grants"`
}

func toRoleGrantsBody(grants []*authv1.RoleGrant) map[string]roleGrantBody {
	body := make(map[string]roleGrantBody, len(grants))
	for _, g := range grants {
		body[g.Key] = roleGrantBody{Scope: g.Scope}
	}
	return body
}

// HandleGetRolePermissions returns the grant set of a role visible to the
// caller — presets included, so the builder can read what it is cloning.
func (a *AuthRoutes) HandleGetRolePermissions(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	roleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetRolePermissions(r.Context(), &authv1.GetRolePermissionsRequest{RoleId: roleID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, rolePermissionsResponseBody{
		RoleID: resp.RoleId,
		Grants: toRoleGrantsBody(resp.Grants),
	})
}

// putRolePermissionsRequest mirrors UpdateRolePermissionsInput in
// rbac-types.ts. Scope shape (own|team|all) is not a validate tag here: the
// catalogue check that also rejects an unknown key needs the database, so
// both travel to the service together instead of splitting the validation
// across two layers.
type putRolePermissionsRequest struct {
	Grants map[string]roleGrantBody `json:"grants"`
}

// HandleSetRolePermissions replaces the entire grant set of a tenant-owned
// role. Presets and unknown capability keys are rejected in the service —
// this handler only reshapes the map the frontend sends into the repeated
// RoleGrant the proto carries.
func (a *AuthRoutes) HandleSetRolePermissions(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	roleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[putRolePermissionsRequest](w, r)
	if !ok {
		return
	}

	grants := make([]*authv1.RoleGrant, 0, len(req.Grants))
	for key, g := range req.Grants {
		grants = append(grants, &authv1.RoleGrant{Key: key, Scope: g.Scope})
	}

	resp, err := client.SetRolePermissions(r.Context(), &authv1.SetRolePermissionsRequest{
		RoleId: roleID,
		Grants: grants,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, rolePermissionsResponseBody{
		RoleID: resp.RoleId,
		Grants: toRoleGrantsBody(resp.Grants),
	})
}

// overrideBody is one per-user override on the wire. Like roleGrantBody the
// capability key never travels as a field of its own — UserOverrides in
// rbac-types.ts is a plain `Record<string, {mode, scope}>`.
type overrideBody struct {
	Mode  string `json:"mode"`
	Scope string `json:"scope"`
}

// userOverridesResponseBody mirrors UserOverridesResponse in rbac-types.ts.
// Overrides is built through toOverridesBody so an account without any
// deviation marshals as `{}`, never `null`.
type userOverridesResponseBody struct {
	UserID    string                  `json:"userId"`
	Overrides map[string]overrideBody `json:"overrides"`
}

func toOverridesBody(overrides []*authv1.CapabilityOverride) map[string]overrideBody {
	body := make(map[string]overrideBody, len(overrides))
	for _, o := range overrides {
		body[o.Key] = overrideBody{Mode: o.Mode, Scope: o.Scope}
	}
	return body
}

// HandleGetUserOverrides returns the per-user permission overrides of an
// account — the deviations the editor renders on top of the inherited role
// stand.
func (a *AuthRoutes) HandleGetUserOverrides(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetUserOverrides(r.Context(), &authv1.GetUserOverridesRequest{UserId: userID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, userOverridesResponseBody{
		UserID:    resp.UserId,
		Overrides: toOverridesBody(resp.Overrides),
	})
}

// putUserOverridesRequest mirrors UpdateUserOverridesInput in rbac-types.ts.
// mode and scope carry no validate tags: the catalogue check that rejects an
// unknown key needs the database, so all three travel to the service together
// instead of splitting the validation across two layers — the same call
// putRolePermissionsRequest makes.
type putUserOverridesRequest struct {
	Overrides map[string]overrideBody `json:"overrides"`
}

// HandleSetUserOverrides replaces the entire override map of an account. An
// empty map clears every deviation, which is the editor's "back to the role
// stand". Guardrails (self-edit, escalation, last admin) and the catalogue
// check live in the service.
func (a *AuthRoutes) HandleSetUserOverrides(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[putUserOverridesRequest](w, r)
	if !ok {
		return
	}

	overrides := make([]*authv1.CapabilityOverride, 0, len(req.Overrides))
	for key, o := range req.Overrides {
		overrides = append(overrides, &authv1.CapabilityOverride{Key: key, Mode: o.Mode, Scope: o.Scope})
	}

	resp, err := client.SetUserOverrides(r.Context(), &authv1.SetUserOverridesRequest{
		UserId:    userID,
		Overrides: overrides,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, userOverridesResponseBody{
		UserID:    resp.UserId,
		Overrides: toOverridesBody(resp.Overrides),
	})
}

// HandleClearUserOverrides drops every override of an account.
func (a *AuthRoutes) HandleClearUserOverrides(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	userID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if _, err := client.ClearUserOverrides(r.Context(), &authv1.ClearUserOverridesRequest{UserId: userID}); err != nil {
		respondGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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
// Tenant Provisioning Handler
// ============================================================================

// provisionTenantRequest mirrors the proto message. seat_limit is a pointer so
// "not booked" (omitted) stays distinguishable from a cap of zero, which is
// rejected — the administrator's invitation takes the first seat.
type provisionTenantRequest struct {
	Name             string   `json:"name" validate:"required,max=200"`
	PlanType         string   `json:"plan_type" validate:"omitempty,oneof=cosmi orbit"`
	SupportTier      string   `json:"support_tier" validate:"omitempty,oneof=standard priority enterprise"`
	SeatLimit        *int32   `json:"seat_limit" validate:"omitempty,min=1"`
	BillingPeriodEnd string   `json:"billing_period_end" validate:"omitempty,datetime=2006-01-02"`
	Modules          []string `json:"modules"`
	AdminEmail       string   `json:"admin_email" validate:"required,email"`
}

func (a *AuthRoutes) HandleProvisionTenant(w http.ResponseWriter, r *http.Request) {
	client, err := a.getAuthClient()
	if err != nil {
		respondServiceUnavailable(w, a.ServiceName())
		return
	}

	req, ok := decodeAndValidate[provisionTenantRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &authv1.ProvisionTenantRequest{
		Name:             req.Name,
		PlanType:         req.PlanType,
		SupportTier:      req.SupportTier,
		BillingPeriodEnd: req.BillingPeriodEnd,
		Modules:          req.Modules,
		AdminEmail:       req.AdminEmail,
		CreatedBy:        middleware.GetUserID(r.Context()),
	}
	if req.SeatLimit != nil {
		grpcReq.SeatLimit = *req.SeatLimit
		grpcReq.HasSeatLimit = true
	}

	resp, err := client.ProvisionTenant(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}
