/**
 * Admin module — app-owned API contract types (account/access layer).
 *
 * These are the shapes the FE consumes; the MSW mock conforms to them today and
 * the real backend (Luke's track 🔒) will conform tomorrow. Keep this the single
 * source of truth so swapping the data source needs no UI change.
 */
import type { RoleId } from '@/config/roles'

export type AdminUserStatus = 'active' | 'invited' | 'deactivated'

export interface AdminUser {
  id: string
  firstName: string
  lastName: string
  email: string
  jobTitle: string
  role: RoleId
  status: AdminUserStatus
  /** ISO timestamp of the last successful sign-in. `null` while invite is pending. */
  lastLoginAt: string | null
  /** ISO timestamp the invite was sent (only meaningful for `invited`). */
  invitedAt: string | null
}

export interface InviteUserInput {
  email: string
  firstName?: string
  lastName?: string
  role: RoleId
}

// ── RBAC (A-2) ──────────────────────────────────────────────────────────────

/** A functional capability group with its ordered capability ids. */
export interface PermissionGroup {
  id: string
  capabilities: string[]
}

/** capabilityId → which roles hold it. Admin is implicitly always true. */
export type PermissionMatrix = Record<string, Partial<Record<RoleId, boolean>>>

export interface PermissionsResponse {
  groups: PermissionGroup[]
  matrix: PermissionMatrix
}

export interface SetPermissionInput {
  capabilityId: string
  role: RoleId
  granted: boolean
}
