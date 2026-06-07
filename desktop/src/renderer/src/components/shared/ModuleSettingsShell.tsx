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

  const hasTenantSections = sections.some((s) => s.scope === 'tenant')

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">{t(titleKey)}</h2>
      {descriptionKey && <p className="text-sm text-muted-foreground mb-6">{t(descriptionKey)}</p>}

      {/* Lock hint when tenant sections exist but the user cannot edit them. */}
      {hasTenantSections && !isLead && (
        <div className="mb-6 flex items-start gap-3 rounded-lg border border-border bg-secondary/40 px-4 py-3">
          <Lock className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
          <p className="text-xs leading-relaxed text-muted-foreground">{t('settings.scope.lockedHint')}</p>
        </div>
      )}

      {intro}

      <div className="space-y-6">
        {sections.map((section) => {
          const editable = section.scope === 'personal' || isLead
          const Icon = section.icon
          return (
            <ModuleSettingsScopeContext.Provider key={section.id} value={{ scope: section.scope, editable }}>
              <section className="rounded-xl border border-border bg-card p-5">
                <header className="mb-4 flex items-start justify-between gap-3">
                  <div className="flex items-start gap-2.5">
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
                  </div>
                  <ScopeBadge scope={section.scope} editable={editable} />
                </header>

                {/* Native fieldset[disabled] cascades to inputs/buttons; the scope
                    context covers custom controls that opt in. */}
                <fieldset disabled={!editable} className={editable ? 'min-w-0' : 'min-w-0 opacity-60'}>
                  {section.children}
                </fieldset>
              </section>
            </ModuleSettingsScopeContext.Provider>
          )
        })}
      </div>

      {footer && <div className="mt-6">{footer}</div>}
    </div>
  )
}

// ─────────────────────────── Scope badge ───────────────────────────

function ScopeBadge({ scope, editable }: { scope: SettingsScope; editable: boolean }) {
  const { t } = useTranslation()

  if (scope === 'personal') {
    return (
      <span className="flex shrink-0 items-center gap-1 rounded-full bg-secondary px-2.5 py-1 text-[10px] font-medium text-muted-foreground">
        <UserIcon className="h-3 w-3" />
        {t('settings.scope.personal')}
      </span>
    )
  }

  return (
    <span
      className={`flex shrink-0 items-center gap-1 rounded-full px-2.5 py-1 text-[10px] font-medium ${
        editable ? 'bg-primary-light text-primary' : 'bg-secondary text-muted-foreground'
      }`}
    >
      {editable ? <Users className="h-3 w-3" /> : <Lock className="h-3 w-3" />}
      {t('settings.scope.tenant')}
    </span>
  )
}
