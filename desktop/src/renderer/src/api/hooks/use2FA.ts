/**
 * TanStack Query hooks for two-factor authentication operations.
 *
 * Provides mutations for 2FA lifecycle (setup, verify, disable) and
 * queries for admin policy management.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  setup2FA,
  verify2FA,
  validate2FALogin,
  disable2FA,
  regenerateRecoveryCodes,
  adminReset2FA,
  getTwoFactorPolicies,
  updateTwoFactorPolicy,
} from '../security-client'

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

/** List 2FA enforcement policies per role (admin only). */
export function useTwoFactorPolicies() {
  return useQuery({
    queryKey: ['2fa', 'policies'],
    queryFn: getTwoFactorPolicies,
  })
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

/** Begin 2FA setup -- returns QR code and manual secret. */
export function useSetup2FA() {
  return useMutation({
    mutationFn: () => setup2FA(),
  })
}

/** Verify 2FA setup with a TOTP code -- returns recovery codes. */
export function useVerify2FA() {
  return useMutation({
    mutationFn: (code: string) => verify2FA(code),
  })
}

/** Validate 2FA during login with pending token and TOTP/recovery code. */
export function useValidate2FALogin() {
  return useMutation({
    mutationFn: ({ pendingToken, code }: { pendingToken: string; code: string }) =>
      validate2FALogin(pendingToken, code),
  })
}

/** Disable 2FA on the current user's account. */
export function useDisable2FA() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (code: string) => disable2FA(code),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth'] })
    },
  })
}

/** Regenerate recovery codes (invalidates existing ones). */
export function useRegenerateRecoveryCodes() {
  return useMutation({
    mutationFn: (code: string) => regenerateRecoveryCodes(code),
  })
}

/** Admin: reset another user's 2FA (requires reason for audit trail). */
export function useAdminReset2FA() {
  return useMutation({
    mutationFn: ({ userId, reason }: { userId: string; reason: string }) =>
      adminReset2FA(userId, reason),
  })
}

/** Admin: update 2FA enforcement policy for a role. */
export function useUpdateTwoFactorPolicy() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      roleName,
      enforced,
      gracePeriodDays,
    }: {
      roleName: string
      enforced: boolean
      gracePeriodDays: number
    }) => updateTwoFactorPolicy(roleName, enforced, gracePeriodDays),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['2fa', 'policies'] })
    },
  })
}
