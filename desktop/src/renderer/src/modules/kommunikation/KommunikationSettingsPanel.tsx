import { useTranslation } from 'react-i18next'
import { LayoutGrid, MessageSquare, Inbox } from 'lucide-react'
import { ModuleSettingsShell } from '@/components/shared/ModuleSettingsShell'
import type { ModuleSettingsSection } from '@/components/shared/ModuleSettingsShell'
import { useKommunikationPrefs } from '@/stores/kommunikationPrefs'

/**
 * Module settings for the unified Kommunikation module (moduleId 'chat').
 *
 * Phase 2: personal display preferences (default area, density, enter-to-send).
 * Tenant-wide sections (channels, routing rules, team inbox, canned responses,
 * retention) are added in later phases.
 */
export function KommunikationSettingsPanel() {
  const { t } = useTranslation()
  const defaultBereich = useKommunikationPrefs((s) => s.defaultBereich)
  const setDefaultBereich = useKommunikationPrefs((s) => s.setDefaultBereich)
  const density = useKommunikationPrefs((s) => s.density)
  const setDensity = useKommunikationPrefs((s) => s.setDensity)
  const enterToSend = useKommunikationPrefs((s) => s.enterToSend)
  const setEnterToSend = useKommunikationPrefs((s) => s.setEnterToSend)

  const sections: ModuleSettingsSection[] = [
    {
      id: 'display',
      titleKey: 'kommunikation.settings.display.title',
      descriptionKey: 'kommunikation.settings.display.desc',
      scope: 'personal',
      icon: LayoutGrid,
      children: (
        <div className="space-y-5">
          {/* Default area */}
          <div className="space-y-2">
            <p className="text-xs font-medium text-foreground">{t('kommunikation.settings.display.defaultArea')}</p>
            <div className="grid grid-cols-2 gap-2">
              <AreaChoice
                active={defaultBereich === 'team'}
                icon={MessageSquare}
                label={t('kommunikation.bereich.team')}
                onClick={() => setDefaultBereich('team')}
              />
              <AreaChoice
                active={defaultBereich === 'posteingang'}
                icon={Inbox}
                label={t('kommunikation.bereich.posteingang')}
                onClick={() => setDefaultBereich('posteingang')}
              />
            </div>
          </div>

          {/* Density */}
          <label className="flex items-center justify-between gap-4">
            <span className="text-xs text-foreground">{t('kommunikation.settings.display.density')}</span>
            <select
              value={density}
              onChange={(e) => setDensity(e.target.value as 'comfortable' | 'compact')}
              className="rounded border border-border bg-input-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              <option value="comfortable">{t('kommunikation.settings.display.densityComfortable')}</option>
              <option value="compact">{t('kommunikation.settings.display.densityCompact')}</option>
            </select>
          </label>

          {/* Enter to send */}
          <label className="flex items-center justify-between gap-4">
            <span className="text-xs text-foreground">{t('kommunikation.settings.display.enterToSend')}</span>
            <button
              type="button"
              role="switch"
              aria-checked={enterToSend}
              onClick={() => setEnterToSend(!enterToSend)}
              className={`flex h-5 w-9 shrink-0 items-center rounded-full px-0.5 transition-colors ${
                enterToSend ? 'justify-end bg-primary' : 'justify-start bg-border'
              }`}
            >
              <span className="h-4 w-4 rounded-full bg-white shadow-sm" />
            </button>
          </label>
        </div>
      ),
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="chat"
      titleKey="kommunikation.settings.title"
      descriptionKey="kommunikation.settings.desc"
      sections={sections}
    />
  )
}

function AreaChoice({
  active,
  icon: Icon,
  label,
  onClick,
}: {
  active: boolean
  icon: typeof MessageSquare
  label: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`flex items-center gap-2 rounded-lg border px-3 py-2.5 text-xs font-medium transition-colors ${
        active ? 'border-primary bg-primary-light text-primary' : 'border-border text-foreground hover:bg-secondary'
      }`}
    >
      <Icon className="h-4 w-4" aria-hidden="true" />
      {label}
    </button>
  )
}
