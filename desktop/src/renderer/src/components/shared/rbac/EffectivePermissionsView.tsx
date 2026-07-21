/**
 * EffectivePermissionsView (R-2) — the shared "Effektive Rechte" rendering:
 * role chips, filter, grants grouped by module with scope + provenance badges.
 * Used by the own-profile tab (BerechtigungenTab) and the admin/HR per-user
 * view (team member + user detail). The provenance ("aus Rolle X") is the
 * union-transparency no competitor ships (KONZEPT §3).
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, Ban } from 'lucide-react'
import { Input } from '@/components/ui/input'
import type {
  CapabilityGrant,
  CapabilityScope,
  DeniedByOverride,
  EffectivePermissions,
} from '@/api/rbac-types'
import { OVERRIDE_SOURCE } from '@/api/rbac-types'
import { SCOPE_BADGE_CLASS, roleDisplayName } from '@/lib/rbac-format'

interface GrantRow {
  key: string
  module: string
  label: string
  isModuleVisibility: boolean
  scope: CapabilityScope
  sources: string[]
  /** True for a role-granted key a deny override removed (struck-through). */
  denied?: boolean
}

export function EffectivePermissionsView({
  roles,
  capabilities,
  deniedByOverride = [],
}: {
  roles: EffectivePermissions['roles']
  capabilities: Record<string, CapabilityGrant>
  deniedByOverride?: DeniedByOverride[]
}) {
  const { t } = useTranslation()
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

    // R-6: keys the roles granted but a deny override removed — shown struck
    // through instead of silently dropped (the market gap, R6-RECHERCHE §1).
    for (const denied of deniedByOverride) {
      const [module, subject, action] = denied.key.split(':')
      const isModuleVisibility = subject === 'module' && action === 'view'
      const label = isModuleVisibility
        ? t('rbac.effective.moduleVisible')
        : `${t(`rbac.subject.${subject}`, { defaultValue: subject })} — ${t(`rbac.action.${action}`, { defaultValue: action })}`
      rows.push({
        key: denied.key,
        module,
        label,
        isModuleVisibility,
        scope: denied.roleScope,
        sources: denied.sources,
        denied: true,
      })
    }

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
  }, [capabilities, deniedByOverride, query, t])

  return (
    <div className="space-y-4">
      {/* Roles held by this account */}
      <div className="flex flex-wrap gap-2">
        {roles.map((role) => (
          <span
            key={role.id}
            className="inline-flex items-center gap-1.5 rounded-full border border-border bg-card px-2.5 py-1 text-xs font-medium text-foreground"
          >
            <span className="h-2 w-2 rounded-full" style={{ background: role.color }} aria-hidden="true" />
            {roleDisplayName(t, role)}
          </span>
        ))}
      </div>

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
                {rows.map((row) => {
                  const hasOverrideSource = row.sources.includes(OVERRIDE_SOURCE)
                  const roleSources = row.sources.filter((s) => s !== OVERRIDE_SOURCE)
                  // Show provenance chips when multiple roles OR an override is involved.
                  const showSources = multiRole || hasOverrideSource || row.denied
                  return (
                    <li key={`${row.key}${row.denied ? '-denied' : ''}`} className="flex items-center gap-3 px-4 py-2">
                      <span
                        className={`min-w-0 flex-1 truncate text-sm ${
                          row.denied ? 'text-muted-foreground line-through' : 'text-foreground'
                        }`}
                      >
                        {row.label}
                      </span>
                      {!row.isModuleVisibility && !row.denied && (
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${SCOPE_BADGE_CLASS[row.scope]}`}>
                          {t(`rbac.scope.${row.scope}`)}
                        </span>
                      )}
                      {showSources && (
                        <span className="flex shrink-0 items-center gap-1">
                          {row.denied ? (
                            <span className="inline-flex items-center gap-1 rounded-full bg-error-light px-1.5 py-0.5 text-[10px] font-medium text-error">
                              <Ban className="h-2.5 w-2.5" aria-hidden="true" />
                              {t('rbac.override.deniedSource')}
                            </span>
                          ) : (
                            <>
                              {hasOverrideSource && (
                                <span className="inline-flex items-center gap-1 rounded-full bg-info-light px-1.5 py-0.5 text-[10px] font-medium text-info">
                                  {t('rbac.override.source')}
                                </span>
                              )}
                              {multiRole &&
                                roleSources.map((src) => {
                                  const role = roles.find((r) => r.id === src)
                                  const label = role ? roleDisplayName(t, role) : src
                                  return (
                                    <span
                                      key={src}
                                      title={label}
                                      className="inline-flex items-center gap-1 rounded-full bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground"
                                    >
                                      <span
                                        className="h-1.5 w-1.5 rounded-full"
                                        style={{ background: role?.color ?? 'currentColor' }}
                                        aria-hidden="true"
                                      />
                                      {label}
                                    </span>
                                  )
                                })}
                            </>
                          )}
                        </span>
                      )}
                    </li>
                  )
                })}
              </ul>
            </section>
          ))}
        </div>
      )}
    </div>
  )
}
