/**
 * Lightweight fetch wrapper for Security API endpoints.
 *
 * Follows the same pattern as calendar-client.ts / video-client.ts:
 * manual typed fetch helper with auth header injection and 401 retry.
 * Once security routes are added to openapi.yaml and types regenerated,
 * hooks can migrate to the typed apiClient.
 */
import type {
  TwoFactorSetup,
  RecoveryCodes,
  TwoFactorPolicy,
  UserSession,
  AuditFilter,
  AuditListResponse,
  AuditExportFormat,
  AuditChainResult,
  VaultSecret,
  VaultSecretValue,
  GDPRExportRequest,
  ErasurePreview,
  PasswordPolicy,
  PasswordValidationResult,
  IPAccessRule,
} from './security-types'
import { authenticatedRequest, authenticatedBlobRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Internal fetch helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | string[] | undefined>
  /** If true, return the raw Response instead of parsing JSON (for blob downloads). */
  raw?: boolean
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (opts.raw) {
    const response = await authenticatedBlobRequest(opts)
    if (!response.ok) {
      const errBody = await response.json().catch(() => ({}))
      throw new Error(
        (errBody as Record<string, string>).error ||
          `Request failed: ${response.status}`,
      )
    }
    return response as unknown as T
  }
  return authenticatedRequest<T>(opts)
}

// Convenience methods
function get<T>(path: string, params?: RequestOptions['params']): Promise<T> {
  return request<T>({ method: 'GET', path, params })
}

function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>({ method: 'POST', path, body })
}

function put<T>(path: string, body?: unknown): Promise<T> {
  return request<T>({ method: 'PUT', path, body })
}

function del<T = void>(path: string): Promise<T> {
  return request<T>({ method: 'DELETE', path })
}

function getRaw(path: string, params?: RequestOptions['params']): Promise<Response> {
  return request<Response>({ method: 'GET', path, params, raw: true })
}

// ---------------------------------------------------------------------------
// 2FA
// ---------------------------------------------------------------------------

export const setup2FA = () =>
  post<TwoFactorSetup>('/api/v1/auth/2fa/setup')

export const verify2FA = (code: string) =>
  post<RecoveryCodes>('/api/v1/auth/2fa/verify', { code })

export const validate2FALogin = (pendingToken: string, code: string) =>
  post<{ access_token: string; refresh_token: string; user: { id: string; email: string; first_name: string; last_name: string; roles: string[] } }>(
    '/api/v1/auth/2fa/validate',
    { pending_token: pendingToken, code },
  )

export const disable2FA = (code: string) =>
  post<void>('/api/v1/auth/2fa/disable', { code })

// BE requires the current TOTP/recovery code to confirm regeneration (route_auth.go regenerateCodesRequest).
export const regenerateRecoveryCodes = (code: string) =>
  post<RecoveryCodes>('/api/v1/auth/2fa/regenerate-codes', { code })

// BE route has no path param — user_id travels in the body (route_auth.go adminReset2FARequest).
export const adminReset2FA = (userId: string, reason: string) =>
  post<void>('/api/v1/auth/2fa/admin-reset', { user_id: userId, reason })

// BE wraps lists in an object (encoding/json over protobuf): `{ policies: [...] }`.
export const getTwoFactorPolicies = () =>
  get<{ policies: TwoFactorPolicy[] }>('/api/v1/auth/2fa/policy').then((r) => r.policies ?? [])

export const updateTwoFactorPolicy = (
  roleName: string,
  enforced: boolean,
  gracePeriodDays: number,
) =>
  put<TwoFactorPolicy>('/api/v1/auth/2fa/policy', {
    role_name: roleName,
    enforced,
    grace_period_days: gracePeriodDays,
  })

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

export const listMySessions = () =>
  get<{ sessions: UserSession[] }>('/api/v1/auth/sessions').then((r) => r.sessions ?? [])

// BE requires user_id as a query param — it lists sessions for one user, not a
// cross-user aggregate (route_auth.go HandleListAllSessions 400s without it).
// lean: aktuell ungenutzt — die Admin-"Alle Sessions"-Ansicht wurde entfernt (kein User-Picker,
// kein Cross-User-Aggregat im BE). Reaktivieren, sobald eine der beiden Voraussetzungen existiert.
export const listAllSessions = (userId: string) =>
  get<{ sessions: UserSession[]; total: number }>('/api/v1/auth/sessions/all', { user_id: userId })

export const terminateSession = (sessionId: string) =>
  del<void>(`/api/v1/auth/sessions/${sessionId}`)

export const terminateAllSessions = () =>
  del<void>('/api/v1/auth/sessions')

// ---------------------------------------------------------------------------
// Audit
// ---------------------------------------------------------------------------

export const listAuditEntries = (filter: AuditFilter) =>
  get<AuditListResponse>('/api/v1/security/audit', filter as unknown as RequestOptions['params'])

export const exportAuditLog = (
  filter: Omit<AuditFilter, 'offset' | 'limit'>,
  format: AuditExportFormat,
) =>
  getRaw('/api/v1/security/audit/export', {
    ...filter,
    format,
  } as unknown as RequestOptions['params'])

export const verifyAuditChain = (fromSeq: number, toSeq: number) =>
  post<AuditChainResult>('/api/v1/security/audit/verify', {
    from_sequence: fromSeq,
    to_sequence: toSeq,
  })

// ---------------------------------------------------------------------------
// Vault
// ---------------------------------------------------------------------------

export const listVaultSecrets = () =>
  get<{ secrets: VaultSecret[] }>('/api/v1/security/vault').then((r) => r.secrets ?? [])

export const getVaultSecret = (keyName: string) =>
  get<VaultSecretValue>(`/api/v1/security/vault/${keyName}`)

export const setVaultSecret = (
  keyName: string,
  value: string,
  description?: string,
) =>
  put<VaultSecret>(`/api/v1/security/vault/${keyName}`, { value, description })

export const deleteVaultSecret = (keyName: string) =>
  del<void>(`/api/v1/security/vault/${keyName}`)

// ---------------------------------------------------------------------------
// GDPR
// ---------------------------------------------------------------------------

export const requestGDPRExport = () =>
  post<GDPRExportRequest>('/api/v1/security/gdpr/export/request')

export const listGDPRExports = () =>
  get<{ export_requests: GDPRExportRequest[] }>('/api/v1/security/gdpr/exports').then(
    (r) => r.export_requests ?? [],
  )

export const approveGDPRExport = (id: string, note?: string) =>
  post<GDPRExportRequest>(`/api/v1/security/gdpr/export/${id}/approve`, { note })

export const denyGDPRExport = (id: string, note?: string) =>
  post<GDPRExportRequest>(`/api/v1/security/gdpr/export/${id}/deny`, { note })

export const downloadGDPRExport = (token: string) =>
  getRaw(`/api/v1/security/gdpr/export/${token}/download`)

export const previewErasure = (userId: string) =>
  post<ErasurePreview>('/api/v1/security/gdpr/erasure/preview', { user_id: userId })

export const executeErasure = (userId: string) =>
  post<void>('/api/v1/security/gdpr/erasure/execute', { user_id: userId })

// ---------------------------------------------------------------------------
// Password Policy
// ---------------------------------------------------------------------------

export const getPasswordPolicy = () =>
  get<{ policy: PasswordPolicy }>('/api/v1/security/password/policy').then((r) => r.policy)

export const updatePasswordPolicy = (policy: Partial<Omit<PasswordPolicy, 'id'>>) =>
  put<PasswordPolicy>('/api/v1/security/password/policy', policy)

export const validatePassword = (password: string) =>
  post<PasswordValidationResult>('/api/v1/security/password/validate', { password })

// ---------------------------------------------------------------------------
// IP Access Rules
// ---------------------------------------------------------------------------

export const listIPRules = () =>
  get<{ rules: IPAccessRule[] }>('/api/v1/security/ip-rules').then((r) => r.rules ?? [])

export const createIPRule = (
  ipCidr: string,
  ruleType: 'allow' | 'block',
  description: string,
) =>
  post<IPAccessRule>('/api/v1/security/ip-rules', {
    ip_cidr: ipCidr,
    rule_type: ruleType,
    description,
  })

export const deleteIPRule = (id: string) =>
  del<void>(`/api/v1/security/ip-rules/${id}`)
