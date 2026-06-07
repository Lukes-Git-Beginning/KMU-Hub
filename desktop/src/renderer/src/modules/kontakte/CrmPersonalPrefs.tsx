/**
 * CrmPersonalPrefs — personal-scope CRM settings (everyone can adapt these to
 * their own workflow). Embedded as the "Persönlich" section of the CRM panel.
 */
import { useTranslation } from 'react-i18next'
import { List, LayoutGrid, Rows3, Rows4 } from 'lucide-react'
import { useCrmPrefsStore, type ContactView, type Density } from '@/stores/crmPrefs'

export function CrmPersonalPrefs() {
  const { t } = useTranslation()
  const defaultContactView = useCrmPrefsStore((s) => s.defaultContactView)
  const density = useCrmPrefsStore((s) => s.density)
  const showAvatars = useCrmPrefsStore((s) => s.showAvatars)
  const setDefaultContactView = useCrmPrefsStore((s) => s.setDefaultContactView)
  const setDensity = useCrmPrefsStore((s) => s.setDensity)
  const setShowAvatars = useCrmPrefsStore((s) => s.setShowAvatars)

  const viewOptions: { id: ContactView; labelKey: string; icon: typeof List }[] = [
    { id: 'list', labelKey: 'crm.settings.personal.viewList', icon: List },
    { id: 'grid', labelKey: 'crm.settings.personal.viewGrid', icon: LayoutGrid },
  ]
  const densityOptions: { id: Density; labelKey: string; icon: typeof Rows3 }[] = [
    { id: 'comfortable', labelKey: 'crm.settings.personal.densityComfortable', icon: Rows3 },
    { id: 'compact', labelKey: 'crm.settings.personal.densityCompact', icon: Rows4 },
  ]

  return (
    <div className="space-y-5">
      {/* Default view */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('crm.settings.personal.defaultView')}</label>
        <div className="flex gap-2">
          {viewOptions.map((opt) => {
            const Icon = opt.icon
            const active = defaultContactView === opt.id
            return (
              <button
                key={opt.id}
                onClick={() => setDefaultContactView(opt.id)}
                className={`flex flex-1 items-center justify-center gap-2 rounded-lg border py-2 text-sm transition-colors ${
                  active ? 'border-primary bg-primary/5 font-medium text-primary' : 'border-border text-foreground hover:bg-secondary'
                }`}
              >
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
      </div>

      {/* Density */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('crm.settings.personal.density')}</label>
        <div className="flex gap-2">
          {densityOptions.map((opt) => {
            const Icon = opt.icon
            const active = density === opt.id
            return (
              <button
                key={opt.id}
                onClick={() => setDensity(opt.id)}
                className={`flex flex-1 items-center justify-center gap-2 rounded-lg border py-2 text-sm transition-colors ${
                  active ? 'border-primary bg-primary/5 font-medium text-primary' : 'border-border text-foreground hover:bg-secondary'
                }`}
              >
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
      </div>

      {/* Show avatars */}
      <label className="flex cursor-pointer items-center justify-between gap-3 rounded-lg border border-border px-3 py-2.5">
        <span className="text-sm text-foreground">{t('crm.settings.personal.showAvatars')}</span>
        <button
          type="button"
          role="switch"
          aria-checked={showAvatars}
          onClick={() => setShowAvatars(!showAvatars)}
          className={`flex h-5 w-9 shrink-0 items-center rounded-full px-0.5 transition-colors ${
            showAvatars ? 'justify-end bg-primary' : 'justify-start bg-border'
          }`}
        >
          <span className="h-4 w-4 rounded-full bg-white shadow-sm" />
        </button>
      </label>
    </div>
  )
}
