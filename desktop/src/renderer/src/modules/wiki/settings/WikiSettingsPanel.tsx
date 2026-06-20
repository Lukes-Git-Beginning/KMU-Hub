import { useTranslation } from 'react-i18next'
import { ShieldCheck, SlidersHorizontal, Info } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import {
  useWikiPrefsStore,
  type WikiView,
  type WikiReadingWidth,
} from '@/stores/wikiPrefs'
import {
  useWikiSettingsStore,
  type WikiShareDefault,
} from '@/stores/wikiSettings'

const VIEWS: WikiView[] = ['tree', 'flat']
const WIDTHS: WikiReadingWidth[] = ['normal', 'wide']
const SHARE_DEFAULTS: WikiShareDefault[] = ['internal', 'public']

// ─── Reusable controls ───────────────────────────────────────────

function Segmented<T extends string>({
  options,
  value,
  onChange,
  labelFor,
}: {
  options: T[]
  value: T
  onChange: (v: T) => void
  labelFor: (v: T) => string
}) {
  return (
    <div className="flex gap-2">
      {options.map((opt) => {
        const on = value === opt
        return (
          <button
            key={opt}
            type="button"
            onClick={() => onChange(opt)}
            aria-pressed={on}
            className={`rounded-lg border px-4 py-2 text-sm transition-colors ${
              on
                ? 'border-primary bg-primary-light text-primary'
                : 'border-border text-muted-foreground hover:bg-secondary'
            }`}
          >
            {labelFor(opt)}
          </button>
        )
      })}
    </div>
  )
}

function Toggle({ on, onToggle }: { on: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-pressed={on}
      className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors ${
        on ? 'bg-primary' : 'bg-secondary'
      }`}
    >
      <span
        className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
          on ? 'translate-x-4.5' : 'translate-x-0.5'
        }`}
      />
    </button>
  )
}

// ─── Persönlich ──────────────────────────────────────────────────

function PersonalPrefs() {
  const { t } = useTranslation()
  const defaultView = useWikiPrefsStore((s) => s.defaultView)
  const readingWidth = useWikiPrefsStore((s) => s.readingWidth)
  const sidebarDefaultOpen = useWikiPrefsStore((s) => s.sidebarDefaultOpen)
  const setDefaultView = useWikiPrefsStore((s) => s.setDefaultView)
  const setReadingWidth = useWikiPrefsStore((s) => s.setReadingWidth)
  const setSidebarDefaultOpen = useWikiPrefsStore((s) => s.setSidebarDefaultOpen)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('wiki.settings.personal.view')}
        </label>
        <Segmented
          options={VIEWS}
          value={defaultView}
          onChange={setDefaultView}
          labelFor={(v) => t(`wiki.settings.viewOption.${v}`)}
        />
        <p className="text-xs text-muted-foreground">{t('wiki.settings.personal.viewHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('wiki.settings.personal.width')}
        </label>
        <Segmented
          options={WIDTHS}
          value={readingWidth}
          onChange={setReadingWidth}
          labelFor={(v) => t(`wiki.settings.widthOption.${v}`)}
        />
        <p className="text-xs text-muted-foreground">{t('wiki.settings.personal.widthHint')}</p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <label className="text-sm font-medium text-foreground">
            {t('wiki.settings.personal.sidebar')}
          </label>
          <p className="text-xs text-muted-foreground">{t('wiki.settings.personal.sidebarHint')}</p>
        </div>
        <Toggle on={sidebarDefaultOpen} onToggle={() => setSidebarDefaultOpen(!sidebarDefaultOpen)} />
      </div>
    </div>
  )
}

// ─── Für alle (mock-first) ───────────────────────────────────────

function TenantPrefs() {
  const { t } = useTranslation()
  const shareDefault = useWikiSettingsStore((s) => s.shareDefault)
  const publicModeEnabled = useWikiSettingsStore((s) => s.publicModeEnabled)
  const setShareDefault = useWikiSettingsStore((s) => s.setShareDefault)
  const setPublicModeEnabled = useWikiSettingsStore((s) => s.setPublicModeEnabled)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('wiki.settings.tenant.shareDefault')}
        </label>
        <Segmented
          options={SHARE_DEFAULTS}
          value={shareDefault}
          onChange={setShareDefault}
          labelFor={(v) => t(`wiki.settings.shareOption.${v}`)}
        />
        <p className="text-xs text-muted-foreground">{t('wiki.settings.tenant.shareDefaultHint')}</p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <label className="text-sm font-medium text-foreground">
            {t('wiki.settings.tenant.publicMode')}
          </label>
          <p className="text-xs text-muted-foreground">{t('wiki.settings.tenant.publicModeHint')}</p>
        </div>
        <Toggle on={publicModeEnabled} onToggle={() => setPublicModeEnabled(!publicModeEnabled)} />
      </div>

      <div className="flex items-start gap-3 rounded-lg border border-border bg-info-light/40 px-4 py-3">
        <Info className="mt-0.5 h-4 w-4 shrink-0 text-info" />
        <p className="text-xs leading-relaxed text-muted-foreground">
          {t('wiki.settings.tenant.rbacNote')}
        </p>
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * WikiSettingsPanel — the "Wiki" entry of the module-settings overlay.
 * Personal prefs (view / reading width / sidebar default) apply for real in the
 * module. The tenant section (share defaults, public mode, category RBAC) is a
 * mock-first preference set — backend enforcement stays Luke's lane.
 */
export function WikiSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'wiki.settings.personal.title',
      descriptionKey: 'wiki.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <PersonalPrefs />,
    },
    {
      id: 'sharing',
      titleKey: 'wiki.settings.tenant.title',
      descriptionKey: 'wiki.settings.tenant.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <TenantPrefs />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="wiki"
      titleKey="wiki.settings.title"
      descriptionKey="wiki.settings.subtitle"
      sections={sections}
    />
  )
}
