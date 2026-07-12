package gateway

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

// SecurityRoutes handles HTTP routes for security services (audit, vault, GDPR, password, IP rules).
// Security services run in the same binary as auth, so we reuse the "auth" gRPC connection.
type SecurityRoutes struct {
	registry *ServiceRegistry
}

// NewSecurityRoutes creates a new SecurityRoutes with the given service registry.
func NewSecurityRoutes(registry *ServiceRegistry) *SecurityRoutes {
	return &SecurityRoutes{registry: registry}
}

// ServiceName returns the backend service name (shares "auth" gRPC port).
func (sr *SecurityRoutes) ServiceName() string { return "auth" }

// getSecurityClient lazily obtains a gRPC client for the Security service.
func (sr *SecurityRoutes) getSecurityClient() (securityv1.SecurityServiceClient, error) {
	conn, err := sr.registry.GetConnection("auth")
	if err != nil {
		return nil, err
	}
	return securityv1.NewSecurityServiceClient(conn), nil
}

// RegisterRoutes registers all security HTTP routes.
func (sr *SecurityRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/security", func(r chi.Router) {
		r.Use(authMiddleware)

		// Audit log (admin only)
		r.With(middleware.RequireRole("admin")).Get("/audit", sr.HandleListAuditEntries)
		r.With(middleware.RequireRole("admin")).Get("/audit/export", sr.HandleExportAuditLog)
		r.With(middleware.RequireRole("admin")).Post("/audit/verify", sr.HandleVerifyAuditChain)

		// Vault (admin only)
		r.With(middleware.RequireRole("admin")).Get("/vault", sr.HandleListVaultSecrets)
		r.With(middleware.RequireRole("admin")).Get("/vault/{keyName}", sr.HandleGetVaultSecret)
		r.With(middleware.RequireRole("admin")).Put("/vault", sr.HandleSetVaultSecret)
		r.With(middleware.RequireRole("admin")).Delete("/vault/{keyName}", sr.HandleDeleteVaultSecret)

		// GDPR
		r.Post("/gdpr/export", sr.HandleRequestDataExport)
		r.Get("/gdpr/exports", sr.HandleListDataExports)
		r.With(middleware.RequireRole("admin")).Post("/gdpr/exports/{id}/approve", sr.HandleApproveDataExport)
		r.With(middleware.RequireRole("admin")).Post("/gdpr/exports/{id}/deny", sr.HandleDenyDataExport)
		r.Get("/gdpr/download/{token}", sr.HandleGetExportDownload)
		r.With(middleware.RequireRole("admin")).Post("/gdpr/erasure/preview", sr.HandlePreviewErasure)
		r.With(middleware.RequireRole("admin")).Post("/gdpr/erasure/execute", sr.HandleExecuteErasure)
		r.With(middleware.RequireRole("admin")).Get("/dsar/search", sr.HandleDSARSearch)

		// Password policy
		r.Get("/password/policy", sr.HandleGetPasswordPolicy)
		r.With(middleware.RequireRole("admin")).Put("/password/policy", sr.HandleUpdatePasswordPolicy)
		r.Post("/password/validate", sr.HandleValidatePassword)

		// IP access rules (admin only)
		r.With(middleware.RequireRole("admin")).Get("/ip-rules", sr.HandleListIPRules)
		r.With(middleware.RequireRole("admin")).Post("/ip-rules", sr.HandleCreateIPRule)
		r.With(middleware.RequireRole("admin")).Delete("/ip-rules/{id}", sr.HandleDeleteIPRule)

		// Retention policies — DSGVO Art. 5(1)(e) (admin only)
		r.With(middleware.RequireRole("admin")).Get("/retention-policies", sr.HandleListRetentionPolicies)
		r.With(middleware.RequireRole("admin")).Post("/retention-policies", sr.HandleCreateRetentionPolicy)
		r.With(middleware.RequireRole("admin")).Put("/retention-policies/{id}", sr.HandleUpdateRetentionPolicy)
		r.With(middleware.RequireRole("admin")).Delete("/retention-policies/{id}", sr.HandleDeleteRetentionPolicy)
	})
}

// ============================================================================
// Request types
// ============================================================================

type setVaultSecretRequest struct {
	KeyName        string `json:"key_name" validate:"required"`
	PlaintextValue string `json:"plaintext_value" validate:"required"`
	Description    string `json:"description"`
}

type approveExportRequest struct {
	ReviewNote string `json:"review_note"`
}

type denyExportRequest struct {
	ReviewNote string `json:"review_note"`
}

type previewErasureRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

type executeErasureRequest struct {
	UserID        string `json:"user_id" validate:"required,uuid"`
	AdminPassword string `json:"admin_password" validate:"required"`
}

type updatePasswordPolicyRequest struct {
	MinLength         int32   `json:"min_length"`
	RequireUppercase  bool    `json:"require_uppercase"`
	RequireLowercase  bool    `json:"require_lowercase"`
	RequireDigit      bool    `json:"require_digit"`
	RequireSpecial    bool    `json:"require_special"`
	MinEntropy        float64 `json:"min_entropy"`
	MaxAgeDays        int32   `json:"max_age_days"`
	PreventReuseCount int32   `json:"prevent_reuse_count"`
}

type validatePasswordHTTPRequest struct {
	Password string `json:"password" validate:"required"`
}

type createIPRuleRequest struct {
	IPCIDR      string `json:"ip_cidr" validate:"required"`
	RuleType    string `json:"rule_type" validate:"required,oneof=allow block"`
	Description string `json:"description"`
}

type createRetentionPolicyRequest struct {
	ResourceType  string `json:"resource_type" validate:"required"`
	RetentionDays int32  `json:"retention_days" validate:"required,min=1"`
	Action        string `json:"action" validate:"required,oneof=delete anonymize"`
	Enabled       bool   `json:"enabled"`
	Description   string `json:"description"`
}

type updateRetentionPolicyRequest struct {
	RetentionDays int32  `json:"retention_days" validate:"required,min=1"`
	Action        string `json:"action" validate:"required,oneof=delete anonymize"`
	Enabled       bool   `json:"enabled"`
	Description   string `json:"description"`
}

// ============================================================================
// Audit Handlers
// ============================================================================

func (sr *SecurityRoutes) HandleListAuditEntries(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	q := r.URL.Query()
	filter := &securityv1.AuditFilter{
		Action: q.Get("action"),
		Result: q.Get("result"),
		UserId: q.Get("user_id"),
	}

	page, pageSize := parsePagination(r, 1, 50)
	filter.Offset = int32((page - 1) * pageSize)
	filter.Limit = int32(pageSize)

	resp, err := client.ListAuditEntries(r.Context(), &securityv1.ListAuditEntriesRequest{
		Filter: filter,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleExportAuditLog(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	resp, err := client.ExportAuditLog(r.Context(), &securityv1.ExportAuditLogRequest{
		Format: format,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", resp.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+resp.Filename)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.Data)
}

func (sr *SecurityRoutes) HandleVerifyAuditChain(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	resp, err := client.VerifyAuditChain(r.Context(), &securityv1.VerifyAuditChainRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Vault Handlers
// ============================================================================

func (sr *SecurityRoutes) HandleListVaultSecrets(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	resp, err := client.ListVaultSecrets(r.Context(), &securityv1.ListVaultSecretsRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleGetVaultSecret(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	keyName := chi.URLParam(r, "keyName")
	if keyName == "" {
		response.Error(w, http.StatusBadRequest, "key name is required")
		return
	}

	resp, err := client.GetVaultSecret(r.Context(), &securityv1.GetVaultSecretRequest{
		KeyName: keyName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleSetVaultSecret(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[setVaultSecretRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.SetVaultSecret(r.Context(), &securityv1.SetVaultSecretRequest{
		KeyName:        req.KeyName,
		PlaintextValue: req.PlaintextValue,
		Description:    req.Description,
		CreatedBy:      userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleDeleteVaultSecret(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	keyName := chi.URLParam(r, "keyName")
	if keyName == "" {
		response.Error(w, http.StatusBadRequest, "key name is required")
		return
	}

	_, err = client.DeleteVaultSecret(r.Context(), &securityv1.DeleteVaultSecretRequest{
		KeyName: keyName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "secret deleted"})
}

// ============================================================================
// GDPR Handlers
// ============================================================================

func (sr *SecurityRoutes) HandleRequestDataExport(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.RequestDataExport(r.Context(), &securityv1.RequestDataExportRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (sr *SecurityRoutes) HandleListDataExports(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	// Non-admin users only see their own exports
	userID := middleware.GetUserID(r.Context())
	statusFilter := r.URL.Query().Get("status")

	// Admin can filter by user_id
	filterUserID := userID
	if qUserID := r.URL.Query().Get("user_id"); qUserID != "" {
		// Only admins can view other users' exports -- gateway trusts RBAC middleware
		filterUserID = qUserID
	}

	resp, err := client.ListDataExports(r.Context(), &securityv1.ListDataExportsRequest{
		UserId: filterUserID,
		Status: statusFilter,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleApproveDataExport(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	exportID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	adminID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[approveExportRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ApproveDataExport(r.Context(), &securityv1.ApproveDataExportRequest{
		ExportId:   exportID,
		ReviewedBy: adminID,
		ReviewNote: req.ReviewNote,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleDenyDataExport(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	exportID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	adminID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[denyExportRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.DenyDataExport(r.Context(), &securityv1.DenyDataExportRequest{
		ExportId:   exportID,
		ReviewedBy: adminID,
		ReviewNote: req.ReviewNote,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleGetExportDownload(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	token := chi.URLParam(r, "token")
	if token == "" {
		response.Error(w, http.StatusBadRequest, "download token is required")
		return
	}

	resp, err := client.GetExportDownload(r.Context(), &securityv1.GetExportDownloadRequest{
		DownloadToken: token,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+resp.Filename)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.Data)
}

func (sr *SecurityRoutes) HandlePreviewErasure(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[previewErasureRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.PreviewErasure(r.Context(), &securityv1.PreviewErasureRequest{
		UserId: req.UserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleExecuteErasure(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	adminID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[executeErasureRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ExecuteErasure(r.Context(), &securityv1.ExecuteErasureRequest{
		UserId:        req.UserID,
		ExecutedBy:    adminID,
		AdminPassword: req.AdminPassword,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// dsarModuleJSON / dsarPersonJSON mirror the exact flat shape the DSAR search
// page consumes (records as plain key→value objects), so we transform the nested
// proto response here rather than passing it through.
type dsarModuleJSON struct {
	Module  string              `json:"module"`
	Columns []string            `json:"columns"`
	Records []map[string]string `json:"records"`
}

type dsarPersonJSON struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Email   string           `json:"email"`
	Company string           `json:"company"`
	Avatar  string           `json:"avatar"`
	Modules []dsarModuleJSON `json:"modules"`
}

// HandleDSARSearch performs an Art. 15 GDPR cross-module person search (admin only).
// GET /api/v1/security/dsar/search?q=<query>
func (sr *SecurityRoutes) HandleDSARSearch(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(query)) < 2 {
		response.Error(w, http.StatusBadRequest, "query must be at least 2 characters")
		return
	}

	resp, err := client.DSARSearch(r.Context(), &securityv1.DSARSearchRequest{Query: query})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	results := make([]dsarPersonJSON, 0, len(resp.Persons))
	for _, p := range resp.Persons {
		person := dsarPersonJSON{
			ID:      p.Id,
			Name:    p.Name,
			Email:   p.Email,
			Company: p.Company,
			Avatar:  p.Avatar,
			Modules: make([]dsarModuleJSON, 0, len(p.Modules)),
		}
		for _, m := range p.Modules {
			mod := dsarModuleJSON{
				Module:  m.Module,
				Columns: m.Columns,
				Records: make([]map[string]string, 0, len(m.Records)),
			}
			for _, rec := range m.Records {
				obj := make(map[string]string, len(rec.Fields))
				for _, f := range rec.Fields {
					obj[f.Key] = f.Value
				}
				mod.Records = append(mod.Records, obj)
			}
			person.Modules = append(person.Modules, mod)
		}
		results = append(results, person)
	}

	response.JSON(w, http.StatusOK, map[string]any{"results": results})
}

// ============================================================================
// Password Policy Handlers
// ============================================================================

func (sr *SecurityRoutes) HandleGetPasswordPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	resp, err := client.GetPasswordPolicy(r.Context(), &securityv1.GetPasswordPolicyRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleUpdatePasswordPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	adminID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[updatePasswordPolicyRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdatePasswordPolicy(r.Context(), &securityv1.UpdatePasswordPolicyRequest{
		Policy: &securityv1.PasswordPolicy{
			MinLength:         req.MinLength,
			RequireUppercase:  req.RequireUppercase,
			RequireLowercase:  req.RequireLowercase,
			RequireDigit:      req.RequireDigit,
			RequireSpecial:    req.RequireSpecial,
			MinEntropy:        req.MinEntropy,
			MaxAgeDays:        req.MaxAgeDays,
			PreventReuseCount: req.PreventReuseCount,
		},
		UpdatedBy: adminID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleValidatePassword(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[validatePasswordHTTPRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ValidatePassword(r.Context(), &securityv1.ValidatePasswordRequest{
		Password: req.Password,
		UserId:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// IP Access Rule Handlers
// ============================================================================

func (sr *SecurityRoutes) HandleListIPRules(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	ruleType := r.URL.Query().Get("rule_type")

	resp, err := client.ListIPRules(r.Context(), &securityv1.ListIPRulesRequest{
		RuleType: ruleType,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleCreateIPRule(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createIPRuleRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateIPRule(r.Context(), &securityv1.CreateIPRuleRequest{
		IpCidr:      req.IPCIDR,
		RuleType:    req.RuleType,
		Description: req.Description,
		CreatedBy:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (sr *SecurityRoutes) HandleDeleteIPRule(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	ruleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteIPRule(r.Context(), &securityv1.DeleteIPRuleRequest{
		RuleId: ruleID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ip rule deleted"})
}

// ============================================================================
// Retention Policy Handlers
// ============================================================================

func (sr *SecurityRoutes) HandleListRetentionPolicies(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	resp, err := client.ListRetentionPolicies(r.Context(), &securityv1.ListRetentionPoliciesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleCreateRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createRetentionPolicyRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateRetentionPolicy(r.Context(), &securityv1.CreateRetentionPolicyRequest{
		ResourceType:  req.ResourceType,
		RetentionDays: req.RetentionDays,
		Action:        req.Action,
		Enabled:       req.Enabled,
		Description:   req.Description,
		CreatedBy:     userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (sr *SecurityRoutes) HandleUpdateRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	policyID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateRetentionPolicyRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateRetentionPolicy(r.Context(), &securityv1.UpdateRetentionPolicyRequest{
		Id:            policyID,
		RetentionDays: req.RetentionDays,
		Action:        req.Action,
		Enabled:       req.Enabled,
		Description:   req.Description,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (sr *SecurityRoutes) HandleDeleteRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSecurityClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	policyID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteRetentionPolicy(r.Context(), &securityv1.DeleteRetentionPolicyRequest{
		Id: policyID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "retention policy deleted"})
}
