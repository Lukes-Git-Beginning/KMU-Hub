/**
 * RestrictedModeBadge (R-3) — the shared module-header chip that tells a role
 * WHY action buttons are absent (market pattern: Asana "Comment only", Google
 * "Viewing"; the alternative — buttons silently missing — is the documented
 * Confluence weakness).
 *
 * Two tiers, derived from the capability catalogue:
 * - "Nur Ansicht": the role holds no mutating capability in the module at all.
 * - "Eingeschränkt": only fine-grained mutations (comment, be_assigned,
 *   time:log …), no base create/edit/delete.
 * Roles with any base mutation render nothing — the chip must never nag
 * daily-use roles.
 */
import { useTranslation } from 'react-i18next'
import { Eye, ShieldAlert } from 'lucide-react'
import { CAPABILITY_CATALOG } from '@/config/capability-catalog'
import type { ModuleKey } from '@/config/capabilities'
import { useCapabilitySet } from '@/hooks/useCapability'

const READ_ACTIONS = new Set(['read', 'view'])

function isMutating(key: string): boolean {
  return !READ_ACTIONS.has(key.split(':')[2])
}

export function RestrictedModeBadge({ module }: { module: ModuleKey }) {
  const { t } = useTranslation()
  const { has, ready } = useCapabilitySet()
  if (!ready) return null

  const defs = CAPABILITY_CATALOG[module]
  if (defs.length === 0) return null

  const held = defs.filter((d) => has(d.key))
  const heldMutating = held.filter((d) => isMutating(d.key))
  if (heldMutating.some((d) => !d.fine)) return null

  const readOnly = heldMutating.length === 0
  const label = readOnly ? t('rbac.gate.readOnlyChip') : t('rbac.gate.limitedChip')
  const hint = readOnly ? t('rbac.gate.readOnlyHint') : t('rbac.gate.limitedHint')
  const Icon = readOnly ? Eye : ShieldAlert

  return (
    <span
      role="status"
      title={hint}
      className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-border bg-secondary px-2.5 py-1 text-[11px] font-medium text-muted-foreground"
    >
      <Icon className="h-3 w-3" aria-hidden="true" />
      {label}
    </span>
  )
}
