import { useTranslation } from 'react-i18next'
import { ChevronDown, ShieldCheck, SlidersHorizontal } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import {
  useFormularePrefsStore,
  type FormulareTab,
} from '@/stores/formularePrefs'
import type { ExportFormat } from '@/api/formulare-types'

const TABS: FormulareTab[] = ['formulare', 'eingänge', 'vorlagen']
const FORMATS: ExportFormat[] = ['csv', 'xlsx']

// ─── Persönlich ──────────────────────────────────────────────────

function PersonalPrefs() {
  const { t } = useTranslation()
  const defaultTab = useFormularePrefsStore((s) => s.defaultTab)
  const defaultExportFormat = useFormularePrefsStore((s) => s.defaultExportFormat)
  const setDefaultTab = useFormularePrefsStore((s) => s.setDefaultTab)
  const setDefaultExportFormat = useFormularePrefsStore((s) => s.setDefaultExportFormat)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('formulare.settings.personal.defaultTab')}
        </label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultTab}
            onChange={(e) => setDefaultTab(e.target.value as FormulareTab)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            {TABS.map((tab) => (
              <option key={tab} value={tab}>
                {t(`formulare.settings.tabOption.${tab}`)}
              </option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">
          {t('formulare.settings.personal.defaultTabHint')}
        </p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('formulare.settings.personal.exportFormat')}
        </label>
        <div className="flex gap-2">
          {FORMATS.map((f) => {
            const on = defaultExportFormat === f
            return (
              <button
                key={f}
                type="button"
                onClick={() => setDefaultExportFormat(f)}
                className={`rounded-lg border px-4 py-2 text-sm uppercase transition-colors ${
                  on
                    ? 'border-primary bg-primary-light text-primary'
                    : 'border-border text-muted-foreground hover:bg-secondary'
                }`}
                aria-pressed={on}
              >
                {f}
              </button>
            )
          })}
        </div>
        <p className="text-xs text-muted-foreground">
          {t('formulare.settings.personal.exportFormatHint')}
        </p>
      </div>
    </div>
  )
}

// ─── Tenant ──────────────────────────────────────────────────────

function TenantPrefs() {
  const { t } = useTranslation()
  const defaultConsentText = useFormularePrefsStore((s) => s.defaultConsentText)
  const defaultPrivacyUrl = useFormularePrefsStore((s) => s.defaultPrivacyUrl)
  const notifyOnSubmission = useFormularePrefsStore((s) => s.notifyOnSubmission)
  const retentionDays = useFormularePrefsStore((s) => s.retentionDays)
  const setDefaultConsentText = useFormularePrefsStore((s) => s.setDefaultConsentText)
  const setDefaultPrivacyUrl = useFormularePrefsStore((s) => s.setDefaultPrivacyUrl)
  const setNotifyOnSubmission = useFormularePrefsStore((s) => s.setNotifyOnSubmission)
  const setRetentionDays = useFormularePrefsStore((s) => s.setRetentionDays)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('formulare.settings.tenant.consentText')}
        </label>
        <textarea
          value={defaultConsentText}
          onChange={(e) => setDefaultConsentText(e.target.value)}
          rows={3}
          className="w-full resize-none rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
        />
        <p className="text-xs text-muted-foreground">
          {t('formulare.settings.tenant.consentTextHint')}
        </p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('formulare.settings.tenant.privacyUrl')}
        </label>
        <input
          type="url"
          value={defaultPrivacyUrl}
          onChange={(e) => setDefaultPrivacyUrl(e.target.value)}
          placeholder="https://…/datenschutz"
          className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
        />
        <p className="text-xs text-muted-foreground">
          {t('formulare.settings.tenant.privacyUrlHint')}
        </p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <label className="text-sm font-medium text-foreground">
            {t('formulare.settings.tenant.notify')}
          </label>
          <p className="text-xs text-muted-foreground">
            {t('formulare.settings.tenant.notifyHint')}
          </p>
        </div>
        <button
          type="button"
          onClick={() => setNotifyOnSubmission(!notifyOnSubmission)}
          className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors ${
            notifyOnSubmission ? 'bg-primary' : 'bg-secondary'
          }`}
          aria-pressed={notifyOnSubmission}
        >
          <span
            className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
              notifyOnSubmission ? 'translate-x-4.5' : 'translate-x-0.5'
            }`}
          />
        </button>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('formulare.settings.tenant.retention')}
        </label>
        <div className="flex items-center gap-2">
          <input
            type="number"
            min={0}
            value={retentionDays}
            onChange={(e) => setRetentionDays(Number(e.target.value))}
            className="w-28 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
          <span className="text-sm text-muted-foreground">
            {t('formulare.settings.tenant.retentionUnit')}
          </span>
        </div>
        <p className="text-xs text-muted-foreground">
          {t('formulare.settings.tenant.retentionHint')}
        </p>
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * FormulareSettingsPanel — the "Formulare" entry of the module-settings overlay.
 * Personal prefs (default tab + export format) apply for real in FormularePage.
 * The tenant section (consent text/link, submission notification, retention) is
 * a demo-stateful preference set — backend enforcement stays Luke's lane.
 */
export function FormulareSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'formulare.settings.personal.title',
      descriptionKey: 'formulare.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <PersonalPrefs />,
    },
    {
      id: 'privacy',
      titleKey: 'formulare.settings.tenant.title',
      descriptionKey: 'formulare.settings.tenant.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <TenantPrefs />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="formulare"
      titleKey="formulare.settings.title"
      descriptionKey="formulare.settings.subtitle"
      sections={sections}
    />
  )
}
