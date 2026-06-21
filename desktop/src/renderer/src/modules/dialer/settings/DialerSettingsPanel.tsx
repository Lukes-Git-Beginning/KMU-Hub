import { useTranslation } from 'react-i18next'
import { Headset, Users, ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { Switch } from '@/components/ui/switch'
import { useDialerPrefsStore } from '@/stores/dialerPrefs'
import { useOutcomes } from '@/api/hooks/useDialer'

const WRAP_UP_OPTIONS = [15, 30, 45, 60]
const CONCURRENT_OPTIONS = [1, 2, 3]

// ─── Reusable controls ───────────────────────────────────────────

function SelectRow({
  label,
  hint,
  value,
  onChange,
  children,
}: {
  label: string
  hint: string
  value: string | number
  onChange: (v: string) => void
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <label className="text-sm font-medium text-foreground">{label}</label>
      <div className="relative w-64 max-w-full">
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
        >
          {children}
        </select>
        <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
      </div>
      <p className="text-xs text-muted-foreground">{hint}</p>
    </div>
  )
}

function SwitchRow({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string
  hint: string
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="space-y-0.5">
        <p className="text-sm font-medium text-foreground">{label}</p>
        <p className="text-xs text-muted-foreground">{hint}</p>
      </div>
      <Switch checked={checked} onCheckedChange={onChange} className="mt-0.5 shrink-0" />
    </div>
  )
}

// ─── Persönlich ──────────────────────────────────────────────────

function DialerPersonalPrefs() {
  const { t } = useTranslation()
  const defaultWrapUpSeconds = useDialerPrefsStore((s) => s.defaultWrapUpSeconds)
  const autoAdvance = useDialerPrefsStore((s) => s.autoAdvance)
  const setDefaultWrapUpSeconds = useDialerPrefsStore((s) => s.setDefaultWrapUpSeconds)
  const setAutoAdvance = useDialerPrefsStore((s) => s.setAutoAdvance)

  return (
    <div className="space-y-5">
      <SelectRow
        label={t('dialer.settings.personal.wrapUpTime')}
        hint={t('dialer.settings.personal.wrapUpTimeHint')}
        value={defaultWrapUpSeconds}
        onChange={(v) => setDefaultWrapUpSeconds(Number(v))}
      >
        {WRAP_UP_OPTIONS.map((s) => (
          <option key={s} value={s}>
            {t('dialer.settings.personal.seconds', { count: s })}
          </option>
        ))}
      </SelectRow>

      <SwitchRow
        label={t('dialer.settings.personal.autoAdvance')}
        hint={t('dialer.settings.personal.autoAdvanceHint')}
        checked={autoAdvance}
        onChange={setAutoAdvance}
      />
    </div>
  )
}

// ─── Für alle (Tenant) ───────────────────────────────────────────

function DialerTenantSettings() {
  const { t } = useTranslation()
  const { data: outcomesData } = useOutcomes()
  const outcomes = outcomesData?.outcomes ?? []

  const maxConcurrentCalls = useDialerPrefsStore((s) => s.maxConcurrentCalls)
  const recordingConsentDefault = useDialerPrefsStore((s) => s.recordingConsentDefault)
  const defaultOutcomeId = useDialerPrefsStore((s) => s.defaultOutcomeId)
  const setMaxConcurrentCalls = useDialerPrefsStore((s) => s.setMaxConcurrentCalls)
  const setRecordingConsentDefault = useDialerPrefsStore((s) => s.setRecordingConsentDefault)
  const setDefaultOutcomeId = useDialerPrefsStore((s) => s.setDefaultOutcomeId)

  return (
    <div className="space-y-5">
      <SelectRow
        label={t('dialer.settings.tenant.maxConcurrent')}
        hint={t('dialer.settings.tenant.maxConcurrentHint')}
        value={maxConcurrentCalls}
        onChange={(v) => setMaxConcurrentCalls(Number(v))}
      >
        {CONCURRENT_OPTIONS.map((n) => (
          <option key={n} value={n}>
            {t('dialer.settings.tenant.callsCount', { count: n })}
          </option>
        ))}
      </SelectRow>

      <SelectRow
        label={t('dialer.settings.tenant.defaultOutcome')}
        hint={t('dialer.settings.tenant.defaultOutcomeHint')}
        value={defaultOutcomeId ?? ''}
        onChange={(v) => setDefaultOutcomeId(v || null)}
      >
        <option value="">{t('dialer.settings.tenant.noDefaultOutcome')}</option>
        {outcomes.map((o) => (
          <option key={o.id} value={o.id}>
            {o.label}
          </option>
        ))}
      </SelectRow>

      <SwitchRow
        label={t('dialer.settings.tenant.recordingConsent')}
        hint={t('dialer.settings.tenant.recordingConsentHint')}
        checked={recordingConsentDefault}
        onChange={setRecordingConsentDefault}
      />
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * DialerSettingsPanel — the "Dialer" entry of the module-settings overlay.
 * Personal prefs (wrap-up time, auto-advance) apply per user; the tenant block
 * (max concurrent calls, recording-consent default, default outcome) is editable
 * only by a Dialer-Modul-Leiter / admin. Call outcomes themselves stay in the
 * in-module Dialer settings page.
 */
export function DialerSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'dialer.settings.personal.workflowTitle',
      descriptionKey: 'dialer.settings.personal.workflowDesc',
      scope: 'personal',
      icon: Headset,
      children: <DialerPersonalPrefs />,
    },
    {
      id: 'tenant',
      titleKey: 'dialer.settings.tenant.title',
      descriptionKey: 'dialer.settings.tenant.desc',
      scope: 'tenant',
      icon: Users,
      children: <DialerTenantSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="dialer"
      titleKey="dialer.settings.panel.title"
      descriptionKey="dialer.settings.panel.subtitle"
      sections={sections}
    />
  )
}
