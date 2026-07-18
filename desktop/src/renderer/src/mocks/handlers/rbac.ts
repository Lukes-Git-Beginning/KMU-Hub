/**
 * RBAC handlers (R-1) — effective permissions of the demo session account and
 * the role list. Stateful against the seed in ../data/rbac.ts; the union
 * resolution mirrors what Luke's `GET /me/permissions` will do server-side
 * (JWT `perms` claim already exists — backend-gaps §RBAC).
 */
import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import type { EffectivePermissionsResponse, RolesResponse } from '@/api/rbac-types'
import {
  ROLE_DEFS,
  getDemoSessionUserId,
  resolveCapabilities,
  rolesForUser,
  seedRoles,
} from '../data/rbac'

const API = API_BASE_URL

export const rbacHandlers = [
  // Effective permissions of the authenticated account (multi-role union).
  http.get(`${API}/api/v1/auth/me/permissions`, () => {
    const userId = getDemoSessionUserId()
    const roleIds = rolesForUser(userId)
    const body: EffectivePermissionsResponse = {
      permissions: {
        roles: roleIds.map((id) => ({
          id,
          name: ROLE_DEFS[id].name,
          isSystem: true,
          color: ROLE_DEFS[id].color,
        })),
        capabilities: resolveCapabilities(roleIds),
      },
    }
    return HttpResponse.json(body)
  }),

  // Role list (R-1: presets only; R-2 adds tenant custom roles + CRUD).
  http.get(`${API}/api/v1/admin/roles`, () => {
    const body: RolesResponse = { roles: seedRoles() }
    return HttpResponse.json(body)
  }),
]
