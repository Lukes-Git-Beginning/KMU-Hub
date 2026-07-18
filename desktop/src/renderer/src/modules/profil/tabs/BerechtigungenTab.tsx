/**
 * BerechtigungenTab — read-only "Effektive Rechte" view of the own account
 * (RBAC R-1 preview; the full per-user admin view ships with the role builder).
 *
 * Shows the resolved multi-role union grouped by module, each grant with its
 * data scope and — when the account holds more than one role — the provenance
 * ("aus Rolle X"), the capability no competitor exposes (KONZEPT §3).
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, ShieldCheck } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { usePermissionsStore } from '@/stores/permissions'
import { roleLabelKey } from '@/config/roles'
import type { CapabilityGrant, CapabilityScope } from '@/api/rbac-types'

interface GrantRow {
  key: string
  module: string
  label: string
  isModuleVisibility: boolean
  scope: CapabilityScope
  sources: string[]
}

const SCOPE_BADGE: Record<CapabilityScope, string> = {
  own: 'bg-secondary text-muted-foreground',
  team: 'bg-info-light text-info',
  all: 'bg-success-light text-success',
}

export default function BerechtigungenTab() {
  const { t } = useTranslation()
  const roles = usePermissionsStore((s) => s.roles)
  const capabilities = usePermissionsStore((s) => s.capabilities)
  const [query, setQuery] = useState('')

  const multiRole = roles.length > 1

  const grouped = useMemo(() => {
    const rows: GrantRow[] = Object.entries(capabilities).map(([key, grant]: [string, CapabilityGrant]) => {
      const [module, subject, action] = key.split(':')
      const isModuleVisibility = subject === 'module' && action === 'view'
      const label = isModuleVisibility
        ? t('rbac.effective.moduleVisible')
        : `${t(`rbac.subject.${subject}`, { defaultValue: subject })} — ${t(`rbac.action.${action}`, { defaultValue: action })}`
      return { key, module, label, isModuleVisibility, scope: grant.scope, sources: grant.sources }
    })

    const q = query.trim().toLowerCase()
    const filtered = q
      ? rows.filter(
          (r) =>
            r.label.toLowerCase().includes(q) ||
            t(`rbac.module.${r.module}`, { defaultValue: r.module }).toLowerCase().includes(q),
        )
      : rows

    const byModule = new Map<string, GrantRow[]>()
    for (const row of filtered) {
      const list = byModule.get(row.module) ?? []
      list.push(row)
      byModule.set(row.module, list)
    }
    // Module visibility first, then alphabetically by label
    for (const list of byModule.values()) {
      list.sort((a, b) =>
        a.isModuleVisibility === b.isModuleVisibility
          ? a.label.localeCompare(b.label)
          : a.isModuleVisibility ? -1 : 1,
      )
    }
    return [...byModule.entries()].sort(([a], [b]) =>
      t(`rbac.module.${a}`, { defaultValue: a }).localeCompare(t(`rbac.module.${b}`, { defaultValue: b })),
    )
  }, [capabilities, query, t])

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
      {/* Roles held by this account */}
      <section className="space-y-3">
        <div className="flex items-center gap-2">
          <ShieldCheck className="h-4 w-4 text-primary" aria-hidden="true" />
          <h2 className="text-sm font-semibold text-foreground">{t('rbac.effective.title')}</h2>
        </div>
        <p className="text-xs text-muted-foreground">{t('rbac.effective.intro')}</p>
        <div className="flex flex-wrap gap-2">
          {roles.map((role) => (
            <span
              key={role.id}
              className="inline-flex items-center gap-1.5 rounded-full border border-border bg-card px-2.5 py-1 text-xs font-medium text-foreground"
            >
              <span className="h-2 w-2 rounded-full" style={{ background: role.color }} aria-hidden="true" />
              {role.isSystem ? t(roleLabelKey(role.id)) : role.name}
            </span>
          ))}
        </div>
      </section>

      {/* Filter */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('rbac.effective.filterPlaceholder')}
          className="pl-9"
        />
      </div>

      {/* Grouped grants */}
      {grouped.length === 0 ? (
        <p className="py-10 text-center text-sm text-muted-foreground">{t('rbac.effective.noMatches')}</p>
      ) : (
        <div className="space-y-4">
          {grouped.map(([module, rows]) => (
            <section key={module} className="rounded-xl border border-border bg-card">
              <header className="flex items-center justify-between border-b border-border px-4 py-2.5">
                <h3 className="text-sm font-medium text-foreground">
                  {t(`rbac.module.${module}`, { defaultValue: module })}
                </h3>
                <span className="text-xs text-muted-foreground">
                  {t('rbac.effective.grantCount', { count: rows.length })}
                </span>
              </header>
              <ul className="divide-y divide-border">
                {rows.map((row) => (
                  <li key={row.key} className="flex items-center gap-3 px-4 py-2">
                    <span className="min-w-0 flex-1 truncate text-sm text-foreground">{row.label}</span>
                    {!row.isModuleVisibility && (
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${SCOPE_BADGE[row.scope]}`}>
                        {t(`rbac.scope.${row.scope}`)}
                      </span>
                    )}
                    {multiRole && (
                      <span className="flex shrink-0 items-center gap-1">
                        {row.sources.map((src) => {
                          const role = roles.find((r) => r.id === src)
                          return (
                            <span
                              key={src}
                              title={role?.isSystem ? t(roleLabelKey(src)) : (role?.name ?? src)}
                              className="inline-flex items-center gap-1 rounded-full bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground"
                            >
                              <span
                                className="h-1.5 w-1.5 rounded-full"
                                style={{ background: role?.color ?? 'currentColor' }}
                                aria-hidden="true"
                              />
                              {role?.isSystem ? t(roleLabelKey(src)) : (role?.name ?? src)}
                            </span>
                          )
                        })}
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      )}
    </div>
  )
}
