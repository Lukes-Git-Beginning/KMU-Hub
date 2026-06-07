import { type ReactNode } from 'react'
import { Lock, Users, User as UserIcon, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { ModuleId } from '@/lib/pricing'
import type { SettingsScope } from '@/lib/module-settings'
import { useIsModuleLead } from '@/hooks/useModuleSettings'
import { ModuleSettingsScopeContext } from './module-settings-scope'

// ─────────────────────────── Public API ───────────────────────────

export interface ModuleSettingsSection {
  id: string
  /** i18n key for the section heading. */
  titleKey: string
  /** Optional i18n key for the section description. */
  descriptionKey?: string
  /** 'personal' = always editable; 'tenant' = only Modul-Leiter/admin may edit. */
  scope: SettingsScope
  icon?: LucideIcon
  children: ReactNode
}

export interface ModuleSettingsShellProps {
  /** Module whose Modul-Leiter rights gate the tenant sections. */
  moduleId: ModuleId
  titleKey: string
  descriptionKey?: string
  sections: ModuleSettingsSection[]
  /** Optional content rendered above the sections (banner, hero, etc.). */
  intro?: ReactNode
  /** Optional content rendered below the sections (e.g. a save button). */
  footer?: ReactNode
}

/**
 * ModuleSettingsShell — the reusable scope-aware container for module settings.
 *
 * Renders a list of declared sections. Personal sections are always editable;
 * tenant-scoped sections are editable only for a Modul-Leiter (or admin) and are
 * otherwise shown read-only behind a lock, with an explanatory hint at the top.
 *
 * Part of the Settings-Fundament — see lib/module-settings.ts.
 */
export function ModuleSettingsShell({
  moduleId,
  titleKey,
  descriptionKey,
  sections,
  intro,
  footer,
}: ModuleSettingsShellProps) {
  const { t } = useTranslation()
  const isLead = useIsModuleLead(moduleId)

  const personalSections = sections.filter((s) => s.scope === 'personal')
  const tenantSections = sections.filter((s) => s.scope === 'tenant')

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">{t(titleKey)}</h2>
      {descriptionKey && <p className="text-sm text-muted-foreground mb-6">{t(descriptionKey)}</p>}

      {intro}

      <div className="space-y-8">
        {/* Persönlich — everyone can adapt the module to their own workflow. */}
        {personalSections.length > 0 && (
          <div>
            <ScopeGroupHeader scope="personal" editable />
            <div className="space-y-6">
              {personalSections.map((section) => renderSection(section, true, t))}
            </div>
          </div>
        )}

        {/* Für alle — tenant-wide; only a Modul-Leiter / admin may change these. */}
        {tenantSections.length > 0 && (
          <div>
            <ScopeGroupHeader scope="tenant" editable={isLead} />
            {!isLead && (
              <div className="mb-4 flex items-start gap-3 rounded-lg border border-border bg-secondary/40 px-4 py-3">
                <Lock className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                <p className="text-xs leading-relaxed text-muted-foreground">{t('settings.scope.lockedHint')}</p>
              </div>
            )}
            <div className="space-y-6">
              {tenantSections.map((section) => renderSection(section, isLead, t))}
            </div>
          </div>
        )}
      </div>

      {footer && <div className="mt-6">{footer}</div>}
    </div>
  )
}

function renderSection(section: ModuleSettingsSection, editable: boolean, t: (k: string) => string) {
  const Icon = section.icon
  return (
    <ModuleSettingsScopeContext.Provider key={section.id} value={{ scope: section.scope, editable }}>
      <section className="rounded-xl border border-border bg-card p-5">
        <header className="mb-4 flex items-start gap-2.5">
          {Icon && (
            <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary-light">
              <Icon className="h-4 w-4 text-primary" />
            </span>
          )}
          <div>
            <h3 className="text-sm font-medium text-foreground">{t(section.titleKey)}</h3>
            {section.descriptionKey && (
              <p className="mt-0.5 text-xs text-muted-foreground">{t(section.descriptionKey)}</p>
            )}
          </div>
        </header>

        {/* Native fieldset[disabled] cascades to inputs/buttons; the scope
            context covers custom controls that opt in. */}
        <fieldset disabled={!editable} className={editable ? 'min-w-0' : 'min-w-0 opacity-60'}>
          {section.children}
        </fieldset>
      </section>
    </ModuleSettingsScopeContext.Provider>
  )
}

// ─────────────────────── Scope group header ───────────────────────

function ScopeGroupHeader({ scope, editable }: { scope: SettingsScope; editable: boolean }) {
  const { t } = useTranslation()
  const isPersonal = scope === 'personal'
  const Icon = isPersonal ? UserIcon : editable ? Users : Lock
  return (
    <div className="mb-3 flex items-center gap-2">
      <span className="flex h-6 w-6 items-center justify-center rounded-md bg-secondary">
        <Icon className="h-3.5 w-3.5 text-muted-foreground" />
      </span>
      <div>
        <p className="text-xs font-semibold uppercase tracking-wider text-foreground">
          {t(isPersonal ? 'settings.scope.personal' : 'settings.scope.tenant')}
        </p>
        <p className="text-[11px] text-muted-foreground">
          {t(isPersonal ? 'settings.scope.personalGroupDesc' : 'settings.scope.tenantGroupDesc')}
        </p>
      </div>
    </div>
  )
}
