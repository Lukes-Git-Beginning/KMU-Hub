/**
 * Admin module — stateful MSW handlers for the tenant administration areas
 * (Benutzer / Rollen / Lizenz / Branding). Mock-first: state lives in
 * module-level variables and survives navigation within a session. The real
 * endpoints (account provisioning, RBAC persistence, licensing) are Luke's
 * track 🔒 — keep request/response shapes swap-ready.
 *
 * Endpoints in this file (A-1 Benutzerverwaltung):
 *   GET    /api/v1/admin/users                  → { users }
 *   POST   /api/v1/admin/users/invite           → { user }   (creates pending invite)
 *   PATCH  /api/v1/admin/users/:id              → { user }   (role / status change)
 *   POST   /api/v1/admin/users/:id/resend-invite → { user }
 */
import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import type { RoleId } from '@/config/roles'
import type { AdminUser, AdminUserStatus, PermissionMatrix, TenantModule } from '@/api/admin-types'
import { seedAdminUsers } from '../data/admin-users'
import { PERMISSION_GROUPS, seedPermissionMatrix } from '../data/admin-permissions'
import { seedTenantModules } from '../data/admin-license'

const API = API_BASE_URL

// ── In-memory state (stateful for the session) ──────────────────────────────
let adminUsers: AdminUser[] = seedAdminUsers()
let permissionMatrix: PermissionMatrix = seedPermissionMatrix()
const tenantModules: TenantModule[] = seedTenantModules()

const VALID_ROLES: RoleId[] = ['admin', 'manager', 'member', 'hr', 'it_support']

/** Derive a display name from an e-mail local part, e.g. "max.muster" → "Max Muster". */
function nameFromEmail(email: string): { firstName: string; lastName: string } {
  const local = email.split('@')[0] ?? ''
  const parts = local.split(/[._-]+/).filter(Boolean)
  const cap = (s: string) => (s ? s[0].toUpperCase() + s.slice(1) : s)
  const firstName = cap(parts[0] ?? 'Neuer')
  const lastName = parts.slice(1).map(cap).join(' ') || 'Nutzer'
  return { firstName, lastName }
}

export const adminHandlers = [
  // ── List ──────────────────────────────────────────────────────────────────
  http.get(`${API}/api/v1/admin/users`, () => {
    return HttpResponse.json({ users: adminUsers })
  }),

  // ── Invite (creates a pending account) ─────────────────────────────────────
  http.post(`${API}/api/v1/admin/users/invite`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      email?: string
      firstName?: string
      lastName?: string
      role?: RoleId
    }

    const email = (body.email ?? '').trim().toLowerCase()
    if (!email) {
      return HttpResponse.json({ error: 'email_required' }, { status: 400 })
    }
    if (adminUsers.some((u) => u.email.toLowerCase() === email)) {
      return HttpResponse.json({ error: 'email_exists' }, { status: 409 })
    }

    const derived = nameFromEmail(email)
    const role: RoleId = VALID_ROLES.includes(body.role as RoleId) ? (body.role as RoleId) : 'member'
    const user: AdminUser = {
      id: `usr-inv-${Date.now()}`,
      firstName: (body.firstName ?? derived.firstName).trim() || derived.firstName,
      lastName: (body.lastName ?? derived.lastName).trim() || derived.lastName,
      email,
      jobTitle: '',
      role,
      status: 'invited',
      lastLoginAt: null,
      invitedAt: new Date().toISOString(),
    }
    adminUsers = [user, ...adminUsers]
    return HttpResponse.json({ user }, { status: 201 })
  }),

  // ── Resend invite ──────────────────────────────────────────────────────────
  http.post(`${API}/api/v1/admin/users/:id/resend-invite`, ({ params }) => {
    const id = String(params.id)
    const user = adminUsers.find((u) => u.id === id)
    if (!user) return HttpResponse.json({ error: 'not_found' }, { status: 404 })
    user.invitedAt = new Date().toISOString()
    return HttpResponse.json({ user })
  }),

  // ── Update (role / status) ──────────────────────────────────────────────────
  http.patch(`${API}/api/v1/admin/users/:id`, async ({ params, request }) => {
    const id = String(params.id)
    const patch = (await request.json().catch(() => ({}))) as {
      role?: RoleId
      status?: AdminUserStatus
    }
    const user = adminUsers.find((u) => u.id === id)
    if (!user) return HttpResponse.json({ error: 'not_found' }, { status: 404 })

    if (patch.role && VALID_ROLES.includes(patch.role)) {
      user.role = patch.role
    }
    if (patch.status && ['active', 'invited', 'deactivated'].includes(patch.status)) {
      user.status = patch.status
      // Reactivating a never-logged-in invite keeps lastLoginAt null; an active
      // account that gets reactivated keeps its prior login timestamp.
    }
    return HttpResponse.json({ user })
  }),

  // ── RBAC: permission matrix (A-2) ───────────────────────────────────────────
  http.get(`${API}/api/v1/admin/permissions`, () => {
    return HttpResponse.json({ groups: PERMISSION_GROUPS, matrix: permissionMatrix })
  }),

  http.patch(`${API}/api/v1/admin/permissions`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      capabilityId?: string
      role?: RoleId
      granted?: boolean
    }
    const { capabilityId, role, granted } = body
    // Admin always holds every capability — its grants are not editable.
    if (capabilityId && role && role !== 'admin' && typeof granted === 'boolean') {
      const row = { ...(permissionMatrix[capabilityId] ?? { admin: true }) }
      if (granted) row[role] = true
      else delete row[role]
      permissionMatrix = { ...permissionMatrix, [capabilityId]: row }
    }
    return HttpResponse.json({ matrix: permissionMatrix })
  }),

  // ── Licensing: tenant module activation (A-3) ───────────────────────────────
  http.get(`${API}/api/v1/admin/license`, () => {
    return HttpResponse.json({ modules: tenantModules })
  }),

  http.patch(`${API}/api/v1/admin/license`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      moduleId?: string
      active?: boolean
    }
    const mod = tenantModules.find((m) => m.moduleId === body.moduleId)
    if (!mod) return HttpResponse.json({ error: 'not_found' }, { status: 404 })
    if (typeof body.active === 'boolean') {
      mod.active = body.active
      // Deactivating a module releases its assigned seats in the demo model.
      if (!body.active) mod.assignedSeats = 0
    }
    return HttpResponse.json({ module: mod })
  }),
]
