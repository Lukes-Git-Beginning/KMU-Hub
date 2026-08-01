/**
 * RBAC Phase 1a seed generator.
 *
 * Reads the two frontend RBAC SSOT files and emits the SQL INSERT blocks for
 * the Phase-1a migration on stdout: (a) the full permissions catalogue
 * (module:module:view keys for every module + every fine capability key from
 * the catalogue), (b) the 4 preset roles migration 000002 never seeded
 * (it_admin/hr_admin/readonly/extern) plus a color backfill for the 3 that
 * already exist, (c) role_permissions grants with scope for all 7 presets.
 *
 * Every statement is idempotent so a re-run against a partially-migrated
 * database does not break (many fine keys already exist from earlier
 * per-module permission seeds; the coarse legacy permissions are untouched and
 * keep coexisting).
 *
 * The permissions and roles blocks use ON CONFLICT DO NOTHING; the
 * role_permissions block uses DO UPDATE SET scope on purpose. A pre-existing
 * grant carries the column default 'all', so DO NOTHING would silently leave a
 * catalogue-narrowed grant at tenant-wide scope — member already held
 * schichten:swap:create/read from an earlier per-module seed while the
 * catalogue puts both at 'own'. The catalogue is the source of truth for
 * system presets (tenant_id IS NULL), so the seed converges them.
 *
 * Run from desktop/ (tsx needs --tsconfig to resolve the `@/` alias used
 * inside rbac.ts):
 *
 *   npx tsx --tsconfig tsconfig.web.json scripts/gen-rbac-seed.mjs
 *
 * Output is a SQL fragment meant to be embedded into the Phase-1a migration's
 * .up.sql (p1a-migration), not a standalone runnable file — it references
 * roles.tenant_id/color and role_permissions.scope, which that migration
 * adds before this seed runs.
 */
import capabilitiesPkg from '../src/renderer/src/config/capabilities.ts'
import catalogPkg from '../src/renderer/src/config/capability-catalog.ts'
import rbacPkg from '../src/renderer/src/mocks/data/rbac.ts'

// tsx on this toolchain resolves named exports of .ts files under `default`
// instead of as top-level named exports — destructure from there.
const { MODULE_KEYS, moduleViewKey } = capabilitiesPkg.default ?? capabilitiesPkg
const { CAPABILITY_CATALOG } = catalogPkg.default ?? catalogPkg
const { ROLE_DEFS } = rbacPkg.default ?? rbacPkg

/** Presets migration 000002 already created (admin/manager/member) get a color backfill only. */
const EXISTING_PRESET_ROLE_IDS = ['admin', 'manager', 'member']
/** Presets missing entirely — inserted fresh, tenant_id stays NULL (system preset). */
const NEW_PRESET_ROLE_IDS = ['it_admin', 'hr_admin', 'readonly', 'extern']

// Must match the expression in idx_roles_tenant_name (p1a-migration) exactly,
// or ON CONFLICT below cannot find its arbiter index.
const SYSTEM_TENANT_COALESCE = "COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid)"

function sqlString(value) {
  return `'${String(value).replace(/'/g, "''")}'`
}

/** `modul:subject:action` -> {resource: 'modul:subject', action}. Level-1 keys are `modul:module:view`. */
function permissionRowFor(key) {
  const parts = key.split(':')
  if (parts.length < 2) throw new Error(`capability key without a resource:action shape: ${key}`)
  const action = parts[parts.length - 1]
  const resource = parts.slice(0, -1).join(':')
  return { key, resource, action }
}

/** Union of all level-1 module:view keys and every fine key in the catalogue — the full permissions surface. */
function buildPermissionKeys() {
  const keys = new Set()
  for (const moduleKey of MODULE_KEYS) keys.add(moduleViewKey(moduleKey))
  for (const defs of Object.values(CAPABILITY_CATALOG)) {
    for (const def of defs) keys.add(def.key)
  }
  return [...keys].sort()
}

function emitPermissions(keys) {
  const rows = keys.map(permissionRowFor)
  const lines = rows.map(({ key, resource, action }, i) => {
    const sep = i === rows.length - 1 ? '' : ','
    return `    (${sqlString(key)}, ${sqlString(resource)}, ${sqlString(action)})${sep}`
  })
  return [
    `-- ${rows.length} keys: level-1 module:view visibility for every module +`,
    '-- every fine capability key from capability-catalog.ts.',
    'INSERT INTO permissions (name, resource, action) VALUES',
    ...lines,
    'ON CONFLICT (name) DO NOTHING;',
  ].join('\n')
}

function emitNewPresetRoles() {
  const lines = NEW_PRESET_ROLE_IDS.map((roleId, i) => {
    const def = ROLE_DEFS[roleId]
    const sep = i === NEW_PRESET_ROLE_IDS.length - 1 ? '' : ','
    return `    (${sqlString(roleId)}, ${sqlString(def.description)}, ${sqlString(def.color)})${sep}`
  })
  return [
    '-- Preset roles migration 000002 never seeded. tenant_id stays NULL — system',
    '-- preset, visible to every tenant via the roles RLS policy.',
    'INSERT INTO roles (name, description, color) VALUES',
    ...lines,
    `ON CONFLICT (${SYSTEM_TENANT_COALESCE}, name) DO NOTHING;`,
  ].join('\n')
}

function emitExistingPresetColors() {
  const lines = EXISTING_PRESET_ROLE_IDS.map((roleId) => {
    const def = ROLE_DEFS[roleId]
    return `UPDATE roles SET color = ${sqlString(def.color)} WHERE name = ${sqlString(roleId)} AND tenant_id IS NULL;`
  })
  return ['-- Backfill color on the 3 presets migration 000002 already created.', ...lines].join('\n')
}

/** role_permissions rows, grouped per role by scope so each block is one INSERT...SELECT with an IN-list. */
function emitRolePermissions() {
  const blocks = []
  for (const [roleId, def] of Object.entries(ROLE_DEFS)) {
    const byScope = new Map()
    for (const [key, scope] of Object.entries(def.grants)) {
      if (!byScope.has(scope)) byScope.set(scope, [])
      byScope.get(scope).push(key)
    }
    for (const scope of ['all', 'team', 'own']) {
      const keys = byScope.get(scope)
      if (!keys || keys.length === 0) continue
      keys.sort()
      const nameList = keys.map(sqlString).join(', ')
      blocks.push(
        [
          `-- ${roleId}: ${keys.length} grant(s) at scope '${scope}'`,
          'INSERT INTO role_permissions (role_id, permission_id, scope)',
          `SELECT r.id, p.id, ${sqlString(scope)}`,
          'FROM roles r, permissions p',
          `WHERE r.name = ${sqlString(roleId)} AND r.tenant_id IS NULL AND p.name IN (${nameList})`,
          'ON CONFLICT (role_id, permission_id) DO UPDATE SET scope = EXCLUDED.scope;',
        ].join('\n'),
      )
    }
  }
  return blocks.join('\n\n')
}

function main() {
  const permissionKeys = buildPermissionKeys()

  const sections = [
    '-- ============================================================',
    '-- Generated by desktop/scripts/gen-rbac-seed.mjs — do not hand-edit.',
    '-- Source of truth: mocks/data/rbac.ts (ROLE_DEFS) + config/capability-catalog.ts.',
    '-- ============================================================',
    '',
    '-- (a) permissions',
    emitPermissions(permissionKeys),
    '',
    '-- (b) the 4 missing preset roles + color backfill on the 3 existing ones',
    emitNewPresetRoles(),
    '',
    emitExistingPresetColors(),
    '',
    '-- (c) role_permissions with scope, all 7 presets',
    emitRolePermissions(),
  ]

  process.stdout.write(sections.join('\n') + '\n')

  const grantCounts = Object.fromEntries(
    Object.entries(ROLE_DEFS).map(([id, def]) => [id, Object.keys(def.grants).length]),
  )
  process.stderr.write(`# permissions: ${permissionKeys.length}\n# grants: ${JSON.stringify(grantCounts)}\n`)
}

main()
