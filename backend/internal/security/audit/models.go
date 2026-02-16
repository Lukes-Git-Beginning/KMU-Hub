package audit

// AuditLockID is the PostgreSQL advisory lock ID used to serialize audit log writes.
// This ensures hash chain integrity under concurrent writes.
const AuditLockID int64 = 8675309

// Security audit actions
const (
	ActionLogin             = "login"
	ActionLogout            = "logout"
	ActionLoginFailed       = "login_failed"
	Action2FAEnabled        = "2fa_enabled"
	Action2FADisabled       = "2fa_disabled"
	Action2FAAdminReset     = "2fa_admin_reset"
	ActionPasswordChanged   = "password_changed"
	ActionRoleChanged       = "role_changed"
	ActionSessionTerminated = "session_terminated"
)

// Admin audit actions
const (
	ActionUserInvited              = "user_invited"
	ActionUserDeactivated          = "user_deactivated"
	ActionSettingsChanged          = "settings_changed"
	ActionVaultAccess              = "vault_access"
	ActionEnforcementPolicyChanged = "enforcement_policy_changed"
)

// Data audit actions
const (
	ActionSensitiveDataViewed   = "sensitive_data_viewed"
	ActionBulkDataExport        = "bulk_data_export"
	ActionDataExportRequested   = "data_export_requested"
	ActionDataExportApproved    = "data_export_approved"
	ActionDataDeletionExecuted  = "data_deletion_executed"
)

// Result constants
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)
