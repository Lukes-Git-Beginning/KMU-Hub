/**
 * Security module TypeScript types.
 *
 * Manually defined types for the Phase 9 security API entities.
 * These types mirror the backend auth/security service models
 * (2FA, sessions, audit, vault, GDPR, password, IP rules).
 */

// ---------------------------------------------------------------------------
// 2FA types
// ---------------------------------------------------------------------------

export interface TwoFactorSetup {
  qr_code_png: string
  manual_secret: string
}

export interface RecoveryCodes {
  recovery_codes: string[]
}

export interface TwoFactorPolicy {
  role_name: string
  enforced: boolean
  grace_period_days: number
}

// ---------------------------------------------------------------------------
// Session types
// ---------------------------------------------------------------------------

export interface UserSession {
  id: string
  user_id: string
  device_name: string
  device_type: string
  ip_address: string
  location: string
  user_agent: string
  is_current: boolean
  last_active_at: string
  created_at: string
}

// ---------------------------------------------------------------------------
// Audit types
// ---------------------------------------------------------------------------

export interface AuditEntry {
  id: string
  /** BE wire: int64 → protojson serializes as string; MSW mocks still send number. */
  sequence_num: number | string
  timestamp: string
  user_id: string
  user_name: string
  action: string
  target: string
  target_type: string
  details: Record<string, unknown>
  ip_address: string
  user_agent: string
  result: 'success' | 'failure'
}

export interface AuditFilter {
  date_from?: string
  date_to?: string
  user_id?: string
  action?: string
  result?: 'success' | 'failure'
  offset: number
  limit: number
}

export type AuditExportFormat = 'csv' | 'json'

export interface AuditListResponse {
  entries: AuditEntry[]
  total: number
}

export interface AuditChainResult {
  valid: boolean
  /** BE wire: int64 → protojson serializes as string; MSW mocks still send number. */
  first_invalid_sequence?: number | string
}

// ---------------------------------------------------------------------------
// Vault types
// ---------------------------------------------------------------------------

export interface VaultSecret {
  id: string
  key_name: string
  description: string
  key_version: number
  created_by: string
  created_at: string
  updated_at: string
}

export interface VaultSecretValue {
  key_name: string
  value: string
}

// ---------------------------------------------------------------------------
// GDPR types
// ---------------------------------------------------------------------------

export type GDPRExportStatus = 'pending' | 'approved' | 'denied' | 'processing' | 'ready' | 'expired'

export interface GDPRExportRequest {
  id: string
  user_id: string
  /** Optional display names joined by the backend (or demo data). */
  user_name?: string | null
  reviewer_name?: string | null
  status: GDPRExportStatus
  requested_at: string
  reviewed_by: string | null
  reviewed_at: string | null
  review_note: string | null
  download_token: string | null
  download_expires_at: string | null
}

export interface ModuleErasurePreview {
  module_name: string
  record_count: number
  action: string
}

export interface ErasurePreview {
  modules: ModuleErasurePreview[]
}

// ---------------------------------------------------------------------------
// Password policy types
// ---------------------------------------------------------------------------

export interface PasswordPolicy {
  id: string
  min_length: number
  min_entropy: number
  max_age_days: number | null
  prevent_reuse_count: number
  require_uppercase: boolean
  require_lowercase: boolean
  require_digit: boolean
  require_special: boolean
}

export interface PasswordValidationResult {
  valid: boolean
  failures: string[]
}

// ---------------------------------------------------------------------------
// IP access rule types
// ---------------------------------------------------------------------------

export type IPRuleType = 'allow' | 'block'

export interface IPAccessRule {
  id: string
  ip_cidr: string
  rule_type: IPRuleType
  description: string
  created_by: string | null
  created_at: string
}

// ---------------------------------------------------------------------------
// Login response extension (2FA flow)
// ---------------------------------------------------------------------------

export interface LoginResult {
  requires_two_factor: boolean
  pending_token: string | null
  access_token: string | null
  refresh_token: string | null
  user: {
    id: string
    email: string
    first_name: string
    last_name: string
    roles: string[]
  } | null
}
