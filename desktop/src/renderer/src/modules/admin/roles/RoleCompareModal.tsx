/**
 * RoleCompareModal (R-2) — two roles side by side with difference
 * highlighting. No market leader ships this (research 2026-07-18); the
 * "only differences" default keeps it scannable across 100+ keys.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, GitCompareArrows, Minus } from 'lucide-react'
import { DetailModal } from '@/components/shared'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Role, RoleGrants } from '@/api/rbac-types'
import { useRoleGrants } from '@/api/hooks/useRbacRoles'
import { MODULE_KEYS } from '@/config/capabilities'
import { SCOPE_BADGE_CLASS, capabilityLabel, isModuleViewKey, moduleLabel, roleDisplayName } from '@/lib/rbac-format'

export default function RoleCompareModal({
  open,
  onClose,
  roles,
}: {
  open: boolean
  onClose: () => void
  roles: Role[]
}) {
  const { t } = useTranslation()
  const [aId, setAId] = useState<string>('member')
  const [bId, setBId] = useState<string>('extern')
  const [onlyDiff, setOnlyDiff] = useState(true)

  const { data: aGrants } = useRoleGrants(open ? aId : null)
  const { data: bGrants } = useRoleGrants(open ? bId : null)

  const roleA = roles.find((r) => r.id === aId)
  const roleB = roles.find((r) => r.id === bId)

  const sections = useMemo(() => {
    if (!aGrants || !bGrants) return []
    return buildSections(aGrants, bGrants, onlyDiff)
  }, [aGrants, bGrants, onlyDiff])

  return (
    <DetailModal
      open={open}
      onClose={onClose}
      title={t('rbac.compare.title')}
      subtitle={t('rbac.compare.subtitle')}
      maxWidth="max-w-3xl"
      badge={
        <span className="inline-flex items-center gap-1.5 rounded-full bg-secondary px-2.5 py-1 text-xs font-medium text-muted-foreground">
          <GitCompareArrows className="h-3.5 w-3.5" aria-hidden="true" />
          {t('rbac.compare.diffCount', { count: sections.reduce((n, s) => n + s.rows.filter((r) => r.differs).length, 0) })}
        </span>
      }
    >
      {/* Role pickers + filter */}
      <div className="mb-4 flex flex-wrap items-end gap-3">
        <RolePicker
          label={t('rbac.compare.roleA')}
          value={aId}
          onChange={setAId}
          roles={roles}
        />
        <RolePicker
          label={t('rbac.compare.roleB')}
          value={bId}
          onChange={setBId}
          roles={roles}
        />
        <label className="ml-auto flex items-center gap-2 pb-1 text-xs text-muted-foreground">
          {t('rbac.compare.onlyDifferences')}
          <Switch checked={onlyDiff} onCheckedChange={setOnlyDiff} aria-label={t('rbac.compare.onlyDifferences')} />
        </label>
      </div>

      {sections.length === 0 ? (
        <p className="py-10 text-center text-sm text-muted-foreground">
          {onlyDiff ? t('rbac.compare.identical') : t('rbac.compare.empty')}
        </p>
      ) : (
        <div className="space-y-4">
          {/* Column header */}
          <div className="grid grid-cols-[1fr_7rem_7rem] items-center gap-2 px-1 text-xs font-medium text-muted-foreground">
            <span />
            <span className="flex items-center justify-center gap-1.5 truncate">
              <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: roleA?.color }} aria-hidden="true" />
              {roleA ? roleDisplayName(t, roleA) : ''}
            </span>
            <span className="flex items-center justify-center gap-1.5 truncate">
              <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: roleB?.color }} aria-hidden="true" />
              {roleB ? roleDisplayName(t, roleB) : ''}
            </span>
          </div>
          {sections.map((section) => (
            <section key={section.module} className="rounded-xl border border-border bg-card">
              <header className="border-b border-border px-4 py-2">
                <h3 className="text-sm font-medium text-foreground">{moduleLabel(t, section.module)}</h3>
              </header>
              <ul className="divide-y divide-border/60">
                {section.rows.map((row) => (
                  <li
                    key={row.key}
                    className={`grid grid-cols-[1fr_7rem_7rem] items-center gap-2 px-4 py-1.5 ${
                      row.differs ? 'bg-info-light/40' : ''
                    }`}
                  >
                    <span className="min-w-0 truncate text-sm text-foreground">
                      {capabilityLabel(t, row.key)}
                    </span>
                    <GrantCell grant={row.a} />
                    <GrantCell grant={row.b} />
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      )}
    </DetailModal>
  )
}

interface CompareRow {
  key: string
  a: RoleGrants[string] | undefined
  b: RoleGrants[string] | undefined
  differs: boolean
}

function buildSections(a: RoleGrants, b: RoleGrants, onlyDiff: boolean) {
  const keys = [...new Set([...Object.keys(a), ...Object.keys(b)])]
  const byModule = new Map<string, CompareRow[]>()
  for (const key of keys) {
    const row: CompareRow = {
      key,
      a: a[key],
      b: b[key],
      differs: (a[key]?.scope ?? null) !== (b[key]?.scope ?? null),
    }
    if (onlyDiff && !row.differs) continue
    const module = key.split(':')[0]
    const list = byModule.get(module) ?? []
    list.push(row)
    byModule.set(module, list)
  }
  // Module-visibility first, keep MODULE_KEYS order for the sections.
  for (const rows of byModule.values()) {
    rows.sort((x, y) =>
      isModuleViewKey(x.key) === isModuleViewKey(y.key)
        ? x.key.localeCompare(y.key)
        : isModuleViewKey(x.key) ? -1 : 1,
    )
  }
  const order = [...MODULE_KEYS] as string[]
  return [...byModule.entries()]
    .sort(([x], [y]) => order.indexOf(x) - order.indexOf(y))
    .map(([module, rows]) => ({ module, rows }))
}

function GrantCell({ grant }: { grant: RoleGrants[string] | undefined }) {
  const { t } = useTranslation()
  if (!grant) {
    return (
      <span className="flex items-center justify-center text-muted-foreground/60">
        <Minus className="h-3.5 w-3.5" aria-hidden="true" />
      </span>
    )
  }
  return (
    <span className="flex items-center justify-center gap-1">
      <Check className="h-3.5 w-3.5 text-success" aria-hidden="true" />
      <span className={`rounded-full px-1.5 py-0.5 text-[10px] font-medium ${SCOPE_BADGE_CLASS[grant.scope]}`}>
        {t(`rbac.scope.${grant.scope}`)}
      </span>
    </span>
  )
}

function RolePicker({
  label,
  value,
  onChange,
  roles,
}: {
  label: string
  value: string
  onChange: (id: string) => void
  roles: Role[]
}) {
  const { t } = useTranslation()
  return (
    <div className="space-y-1">
      <span className="text-xs font-medium text-muted-foreground">{label}</span>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger className="h-8 w-44 text-sm">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {roles.map((role) => (
            <SelectItem key={role.id} value={role.id}>
              <span className="flex items-center gap-2">
                <span className="h-2 w-2 rounded-full" style={{ background: role.color }} aria-hidden="true" />
                {roleDisplayName(t, role)}
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
