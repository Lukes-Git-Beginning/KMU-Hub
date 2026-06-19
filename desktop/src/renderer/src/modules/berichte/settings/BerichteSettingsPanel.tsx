import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronDown, Plus, Send, SlidersHorizontal, X } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { useBerichtePrefsStore, type BerichtePeriod } from '@/stores/berichtePrefs'
import type { ReportFormat } from '@/api/berichte-types'

const FORMATS: ReportFormat[] = ['pdf', 'xlsx', 'csv']
const PERIODS: { value: BerichtePeriod; defaultLabel: string }[] = [
  { value: 'last30', defaultLabel: 'Letzte 30 Tage' },
  { value: 'thisMonth', defaultLabel: 'Aktueller Monat' },
  { value: 'thisQuarter', defaultLabel: 'Aktuelles Quartal' },
  { value: 'thisYear', defaultLabel: 'Aktuelles Jahr' },
]

// ─── Persönlich ──────────────────────────────────────────────────

function PersonalPrefs() {
  const { t } = useTranslation()
  const defaultFormat = useBerichtePrefsStore((s) => s.defaultFormat)
  const defaultPeriod = useBerichtePrefsStore((s) => s.defaultPeriod)
  const setDefaultFormat = useBerichtePrefsStore((s) => s.setDefaultFormat)
  const setDefaultPeriod = useBerichtePrefsStore((s) => s.setDefaultPeriod)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('berichte.settings.personal.format', { defaultValue: 'Standard-Format' })}
        </label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultFormat}
            onChange={(e) => setDefaultFormat(e.target.value as ReportFormat)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            {FORMATS.map((f) => (
              <option key={f} value={f}>
                {f.toUpperCase()}
              </option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">
          {t('berichte.settings.personal.formatHint', {
            defaultValue: 'Wird beim Erstellen neuer Berichte vorausgewählt.',
          })}
        </p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('berichte.settings.personal.period', { defaultValue: 'Standard-Zeitraum' })}
        </label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultPeriod}
            onChange={(e) => setDefaultPeriod(e.target.value as BerichtePeriod)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            {PERIODS.map((p) => (
              <option key={p.value} value={p.value}>
                {t(`berichte.period.${p.value}`, { defaultValue: p.defaultLabel })}
              </option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">
          {t('berichte.settings.personal.periodHint', {
            defaultValue: 'Vorbelegung für den Zeitraum-Filter neuer Berichte.',
          })}
        </p>
      </div>
    </div>
  )
}

// ─── Tenant ──────────────────────────────────────────────────────

function TenantPrefs() {
  const { t } = useTranslation()
  const allowedFormats = useBerichtePrefsStore((s) => s.allowedFormats)
  const scheduleDomains = useBerichtePrefsStore((s) => s.scheduleDomains)
  const toggleAllowedFormat = useBerichtePrefsStore((s) => s.toggleAllowedFormat)
  const addScheduleDomain = useBerichtePrefsStore((s) => s.addScheduleDomain)
  const removeScheduleDomain = useBerichtePrefsStore((s) => s.removeScheduleDomain)
  const [domainInput, setDomainInput] = useState('')

  const submitDomain = () => {
    addScheduleDomain(domainInput)
    setDomainInput('')
  }

  return (
    <div className="space-y-5">
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">
          {t('berichte.settings.tenant.allowedFormats', { defaultValue: 'Erlaubte Export-Formate' })}
        </label>
        <div className="flex gap-2">
          {FORMATS.map((f) => {
            const on = allowedFormats.includes(f)
            return (
              <button
                key={f}
                type="button"
                onClick={() => toggleAllowedFormat(f)}
                className={`flex-1 rounded-lg border px-3 py-2 text-sm uppercase transition-colors ${
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
          {t('berichte.settings.tenant.allowedFormatsHint', {
            defaultValue: 'Formate, die im gesamten Arbeitsbereich exportiert werden dürfen.',
          })}
        </p>
      </div>

      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">
          {t('berichte.settings.tenant.domains', { defaultValue: 'Zulässige E-Mail-Domains' })}
        </label>
        <div className="flex gap-2">
          <input
            type="text"
            value={domainInput}
            onChange={(e) => setDomainInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                submitDomain()
              }
            }}
            placeholder={t('berichte.settings.tenant.domainsPlaceholder', { defaultValue: 'z. B. firma.de' })}
            className="flex-1 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
          <button
            onClick={submitDomain}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <Plus className="h-3.5 w-3.5" />
            {t('berichte.settings.tenant.domainAdd', { defaultValue: 'Hinzufügen' })}
          </button>
        </div>
        {scheduleDomains.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {scheduleDomains.map((d) => (
              <span
                key={d}
                className="flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-xs text-foreground"
              >
                @{d}
                <button
                  onClick={() => removeScheduleDomain(d)}
                  className="ml-0.5 rounded-full p-0.5 transition-colors hover:bg-error-light hover:text-error"
                  aria-label={t('common.delete', { defaultValue: 'Löschen' })}
                >
                  <X className="h-2.5 w-2.5" />
                </button>
              </span>
            ))}
          </div>
        )}
        <p className="text-xs text-muted-foreground">
          {t('berichte.settings.tenant.domainsHint', {
            defaultValue: 'Geplante Berichte dürfen nur an diese Domains versendet werden.',
          })}
        </p>
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * BerichteSettingsPanel — the "Berichte" entry of the module-settings overlay.
 * Personal prefs (default format/period) apply for real in the report builder
 * and schedule dialog. Tenant section (allowed formats, schedule domains) is a
 * demo-stateful preference set — backend enforcement is Luke's lane.
 */
export function BerichteSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'berichte.settings.personal.title',
      descriptionKey: 'berichte.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <PersonalPrefs />,
    },
    {
      id: 'delivery',
      titleKey: 'berichte.settings.tenant.title',
      descriptionKey: 'berichte.settings.tenant.desc',
      scope: 'tenant',
      icon: Send,
      children: <TenantPrefs />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="berichte"
      titleKey="berichte.settings.title"
      descriptionKey="berichte.settings.subtitle"
      sections={sections}
    />
  )
}
