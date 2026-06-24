/**
 * TanStack Query hooks for security operations.
 *
 * Covers audit log, vault secrets, GDPR data export/erasure,
 * password policy, and IP access rules.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listAuditEntries,
  exportAuditLog,
  verifyAuditChain,
  listVaultSecrets,
  getVaultSecret,
  setVaultSecret,
  deleteVaultSecret,
  requestGDPRExport,
  listGDPRExports,
  approveGDPRExport,
  denyGDPRExport,
  previewErasure,
  executeErasure,
  getPasswordPolicy,
  updatePasswordPolicy,
  validatePassword,
  listIPRules,
  createIPRule,
  deleteIPRule,
} from '../security-client'
import type { AuditFilter, AuditExportFormat, PasswordPolicy } from '../security-types'

// ---------------------------------------------------------------------------
// Audit Log
// ---------------------------------------------------------------------------

/** Query paginated audit log entries with filters. */
export function useAuditLog(filter: AuditFilter) {
  return useQuery({
    queryKey: ['audit', filter],
    queryFn: () => listAuditEntries(filter),
  })
}

/** Export audit log as CSV or JSON (triggers browser download). */
export function useExportAuditLog() {
  return useMutation({
    mutationFn: async ({
      filter,
      format,
    }: {
      filter: Omit<AuditFilter, 'offset' | 'limit'>
      format: AuditExportFormat
    }) => {
      const response = await exportAuditLog(filter, format)
      const blob = await response.blob()
      const ext = format === 'csv' ? 'csv' : 'json'
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `audit-log-${new Date().toISOString().slice(0, 10)}.${ext}`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    },
  })
}

/** Verify audit chain integrity between sequence numbers. */
export function useVerifyAuditChain() {
  return useMutation({
    mutationFn: ({ fromSeq, toSeq }: { fromSeq: number; toSeq: number }) =>
      verifyAuditChain(fromSeq, toSeq),
  })
}

// ---------------------------------------------------------------------------
// Vault
// ---------------------------------------------------------------------------

/** List vault secrets (metadata only, no values). */
export function useVaultSecrets() {
  return useQuery({
    queryKey: ['vault', 'secrets'],
    queryFn: listVaultSecrets,
  })
}

/**
 * Get a vault secret's decrypted value.
 * This is a mutation (not query) to avoid caching sensitive decrypted values.
 */
export function useGetVaultSecret() {
  return useMutation({
    mutationFn: (keyName: string) => getVaultSecret(keyName),
  })
}

/** Create or update a vault secret. */
export function useSetVaultSecret() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      keyName,
      value,
      description,
    }: {
      keyName: string
      value: string
      description?: string
    }) => setVaultSecret(keyName, value, description),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['vault', 'secrets'] })
    },
  })
}

/** Delete a vault secret. */
export function useDeleteVaultSecret() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (keyName: string) => deleteVaultSecret(keyName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['vault', 'secrets'] })
    },
  })
}

// ---------------------------------------------------------------------------
// GDPR
// ---------------------------------------------------------------------------

/** List all GDPR export requests. */
export function useGDPRExports() {
  return useQuery({
    queryKey: ['gdpr', 'exports'],
    queryFn: listGDPRExports,
  })
}

/** Request a GDPR data export. */
export function useRequestExport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => requestGDPRExport(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gdpr', 'exports'] })
    },
  })
}

/** Admin: approve a GDPR export request. */
export function useApproveExport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, note }: { id: string; note?: string }) =>
      approveGDPRExport(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gdpr', 'exports'] })
    },
  })
}

/** Admin: deny a GDPR export request. */
export function useDenyExport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, note }: { id: string; note?: string }) =>
      denyGDPRExport(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gdpr', 'exports'] })
    },
  })
}

/** Preview which modules/records an erasure would affect (Art. 17). */
export function usePreviewErasure() {
  return useMutation({
    mutationFn: (userId: string) => previewErasure(userId),
  })
}

/** Execute the erasure (anonymize/delete per module). */
export function useExecuteErasure() {
  return useMutation({
    mutationFn: (userId: string) => executeErasure(userId),
  })
}

// ---------------------------------------------------------------------------
// Password Policy
// ---------------------------------------------------------------------------

/** Get the current password policy. */
export function usePasswordPolicy() {
  return useQuery({
    queryKey: ['security', 'password-policy'],
    queryFn: getPasswordPolicy,
  })
}

/** Update the password policy. */
export function useUpdatePasswordPolicy() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (policy: Partial<Omit<PasswordPolicy, 'id'>>) =>
      updatePasswordPolicy(policy),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['security', 'password-policy'] })
    },
  })
}

/** Validate a password against the current policy (real-time feedback). */
export function useValidatePassword() {
  return useMutation({
    mutationFn: (password: string) => validatePassword(password),
  })
}

// ---------------------------------------------------------------------------
// IP Access Rules
// ---------------------------------------------------------------------------

/** List all IP access rules. */
export function useIPRules() {
  return useQuery({
    queryKey: ['security', 'ip-rules'],
    queryFn: listIPRules,
  })
}

/** Create a new IP access rule. */
export function useCreateIPRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      ipCidr,
      ruleType,
      description,
    }: {
      ipCidr: string
      ruleType: 'allow' | 'block'
      description: string
    }) => createIPRule(ipCidr, ruleType, description),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['security', 'ip-rules'] })
    },
  })
}

/** Delete an IP access rule. */
export function useDeleteIPRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteIPRule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['security', 'ip-rules'] })
    },
  })
}
