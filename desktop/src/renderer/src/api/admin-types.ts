/**
 * Admin module — app-owned API contract types (account/access layer).
 *
 * These are the shapes the FE consumes; the MSW mock conforms to them today and
 * the real backend (Luke's track 🔒) will conform tomorrow. Keep this the single
 * source of truth so swapping the data source needs no UI change.
 */
export type AdminUserStatus = 'active' | 'invited' | 'deactivated'

export interface AdminUser {
  id: string
  firstName: string
  lastName: string
  email: string
  jobTitle: string
  /**
   * Role ids the account holds (n:m, union semantics — RBAC R-2 multi-role).
   * Preset ids (`admin`, `member`, …) or custom role ids (`role-c*`). The
   * single source in the mock layer is USER_ROLE_ASSIGNMENTS (mocks/data/rbac).
   */
  roles: string[]
  /** Whether the account carries per-user permission overrides (R-6 "Angepasst" badge). */
  hasOverrides?: boolean
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
  roles: string[]
}

// ── Licensing / module activation (A-3) ─────────────────────────────────────

export type ModuleGroupId = 'core' | 'comm' | 'team' | 'industry' | 'tools'

/** Tenant-wide activation state for one Cosmi module (the provisioning layer). */
export interface TenantModule {
  /** Matches `ModuleId` from @/lib/pricing. */
  moduleId: string
  group: ModuleGroupId
  active: boolean
  /** How many users currently have this module assigned. */
  assignedSeats: number
}

export interface LicenseResponse {
  modules: TenantModule[]
}

export interface ToggleModuleInput {
  moduleId: string
  active: boolean
}
