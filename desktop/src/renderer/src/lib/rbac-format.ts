/**
 * RBAC formatting helpers (R-2) — one place for capability/module/role/scope
 * display so profile tab, role builder, editor, compare and team views render
 * identically. i18n building blocks: `rbac.subject.<subject>`,
 * `rbac.action.<action>`, `rbac.module.<module>`, `rbac.scope.<scope>`.
 */
import type { TFunction } from 'i18next'
import type { CapabilityScope, Role } from '@/api/rbac-types'
import { roleLabelKey } from '@/config/roles'

/** Human label for a capability key (`work:task:edit` → "Aufgaben — Bearbeiten"). */
export function capabilityLabel(t: TFunction, key: string): string {
  const [, subject, action] = key.split(':')
  if (subject === 'module' && action === 'view') return t('rbac.effective.moduleVisible')
  return `${t(`rbac.subject.${subject}`, { defaultValue: subject })} — ${t(`rbac.action.${action}`, { defaultValue: action })}`
}

/** Module display name. */
export function moduleLabel(t: TFunction, module: string): string {
  return t(`rbac.module.${module}`, { defaultValue: module })
}

/** Display name of a role — presets via i18n, custom roles carry their own name. */
export function roleDisplayName(
  t: TFunction,
  role: Pick<Role, 'id' | 'name' | 'isSystem'>,
): string {
  return role.isSystem ? t(roleLabelKey(role.id)) : role.name
}

/** Scope pill classes (mirrors BerechtigungenTab R-1). */
export const SCOPE_BADGE_CLASS: Record<CapabilityScope, string> = {
  own: 'bg-secondary text-muted-foreground',
  team: 'bg-info-light text-info',
  all: 'bg-success-light text-success',
}

/** True for `<module>:module:view` level-1 keys. */
export function isModuleViewKey(key: string): boolean {
  const parts = key.split(':')
  return parts[1] === 'module' && parts[2] === 'view'
}

const RBAC_ERROR_CODES = new Set([
  'preset_immutable',
  'role_limit_reached',
  'role_name_exists',
  'role_has_members',
  'last_admin',
])

/** Map the mock/BE error contract (`{error: <code>}`) onto i18n messages. */
export function rbacErrorMessage(t: TFunction, err: unknown): string {
  const code = err instanceof Error ? err.message : ''
  return RBAC_ERROR_CODES.has(code)
    ? t(`rbac.builder.errors.${code}`)
    : t('rbac.builder.errors.generic')
}
