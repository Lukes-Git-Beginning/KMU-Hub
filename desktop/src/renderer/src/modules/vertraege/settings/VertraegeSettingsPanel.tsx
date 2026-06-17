import { useTranslation } from 'react-i18next'
import {
  SlidersHorizontal,
  Shapes,
  CalendarClock,
  Hash,
  Bell,
  Rows3,
  Rows4,
  Building,
  Truck,
  Wrench,
  UserCheck,
  KeyRound,
  Shield,
  FileText,
  Hourglass,
  Archive,
  LayoutTemplate,
} from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import {
  useVertraegePrefsStore,
  type VertraegeTab,
  type VertraegeDensity,
} from '@/stores/vertraegePrefs'
import {
  useVertraegeSettingsStore,
  ALL_CONTRACT_TYPES,
  formatContractNumber,
} from '@/stores/vertraegeSettings'
import type { ContractType } from '@/stores/vertraege'

const REMINDER_CHOICES = [30, 60, 90]

const segBtn = (active: boolean) =>
  `flex flex-1 items-center justify-center gap-2 rounded-lg border py-2 text-sm transition-colors ${
    active
      ? 'border-primary bg-primary/5 font-medium text-primary'
      : 'border-border text-foreground hover:bg-secondary'
  }`

const typeIcons: Record<ContractType, typeof Building> = {
  mietvertrag: Building,
  liefervertrag: Truck,
  servicevertrag: Wrench,
  arbeitsvertrag: UserCheck,
  lizenz: KeyRound,
  versicherung: Shield,
}

// ─── Persönlich ──────────────────────────────────────────────────

function VertraegePersonalPrefs() {
  const { t } = useTranslation()
  const defaultTab = useVertraegePrefsStore((s) => s.defaultTab)
  const density = useVertraegePrefsStore((s) => s.density)
  const defaultReminderDays = useVertraegePrefsStore((s) => s.defaultReminderDays)
  const setDefaultTab = useVertraegePrefsStore((s) => s.setDefaultTab)
  const setDensity = useVertraegePrefsStore((s) => s.setDensity)
  const setDefaultReminderDays = useVertraegePrefsStore((s) => s.setDefaultReminderDays)
  const tenantReminderDays = useVertraegeSettingsStore((s) => s.tenantReminderDays)

  // Literal labelKey inference keeps t() happy under typed i18n keys.
  const tabOptions = [
    { id: 'aktiv', labelKey: 'vertraege.tab.aktiv', icon: FileText },
    { id: 'auslaufend', labelKey: 'vertraege.tab.auslaufend', icon: Hourglass },
    { id: 'archiv', labelKey: 'vertraege.tab.archiv', icon: Archive },
    { id: 'vorlagen', labelKey: 'vertraege.tab.vorlagen', icon: LayoutTemplate },
  ] as const satisfies ReadonlyArray<{ id: VertraegeTab; labelKey: string; icon: typeof FileText }>
  const densityOptions = [
    { id: 'comfortable', labelKey: 'vertraege.settings.personal.densityComfortable', icon: Rows3 },
    { id: 'compact', labelKey: 'vertraege.settings.personal.densityCompact', icon: Rows4 },
  ] as const satisfies ReadonlyArray<{ id: VertraegeDensity; labelKey: string; icon: typeof Rows3 }>

  const usesTenantDefault = defaultReminderDays === null
  const effectiveReminders = defaultReminderDays ?? tenantReminderDays

  const toggleOverrideDay = (days: number) => {
    const base = defaultReminderDays ?? tenantReminderDays
    const next = base.includes(days)
      ? base.filter((d) => d !== days)
      : [...base, days].sort((a, b) => a - b)
    setDefaultReminderDays(next)
  }

  return (
    <div className="space-y-5">
      {/* Default tab */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('vertraege.settings.personal.defaultTab')}
        </label>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          {tabOptions.map((opt) => {
            const Icon = opt.icon
            return (
              <button key={opt.id} onClick={() => setDefaultTab(opt.id)} className={segBtn(defaultTab === opt.id)}>
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
        <p className="text-xs text-muted-foreground">{t('vertraege.settings.personal.defaultTabHint')}</p>
      </div>

      {/* Density */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('vertraege.settings.personal.density')}
        </label>
        <div className="flex gap-2">
          {densityOptions.map((opt) => {
            const Icon = opt.icon
            return (
              <button key={opt.id} onClick={() => setDensity(opt.id)} className={segBtn(density === opt.id)}>
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
      </div>

      {/* Personal reminder pre-selection */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('vertraege.settings.personal.reminders')}
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={usesTenantDefault}
            onChange={() => setDefaultReminderDays(usesTenantDefault ? [...tenantReminderDays] : null)}
            className="h-4 w-4 rounded border-border accent-primary"
          />
          <span className="text-sm text-foreground">
            {t('vertraege.settings.personal.useTenantReminders')}
          </span>
        </label>
        <div className="flex gap-4 pt-1">
          {REMINDER_CHOICES.map((days) => (
            <label
              key={days}
              className={`flex items-center gap-2 ${usesTenantDefault ? 'opacity-50' : 'cursor-pointer'}`}
            >
              <input
                type="checkbox"
                disabled={usesTenantDefault}
                checked={effectiveReminders.includes(days)}
                onChange={() => toggleOverrideDay(days)}
                className="h-4 w-4 rounded border-border accent-primary"
              />
              <span className="text-sm text-foreground">
                {t('vertraege.settings.reminderDays', { days })}
              </span>
            </label>
          ))}
        </div>
        <p className="text-xs text-muted-foreground">{t('vertraege.settings.personal.remindersHint')}</p>
      </div>
    </div>
  )
}

// ─── Für alle: Vertragsarten ─────────────────────────────────────

function ContractTypesSettings() {
  const { t } = useTranslation()
  const enabledTypes = useVertraegeSettingsStore((s) => s.enabledTypes)
  const toggleType = useVertraegeSettingsStore((s) => s.toggleType)

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
        {ALL_CONTRACT_TYPES.map((type) => {
          const Icon = typeIcons[type]
          const active = enabledTypes.includes(type)
          return (
            <button key={type} onClick={() => toggleType(type)} className={segBtn(active)}>
              <Icon className="h-4 w-4" />
              {t(`vertraege.type.${type}`)}
            </button>
          )
        })}
      </div>
      <p className="text-xs text-muted-foreground">{t('vertraege.settings.types.hint')}</p>
    </div>
  )
}

// ─── Für alle: Standardwerte ─────────────────────────────────────

function ContractDefaultsSettings() {
  const { t } = useTranslation()
  const defaultDurationMonths = useVertraegeSettingsStore((s) => s.defaultDurationMonths)
  const defaultNoticePeriodDays = useVertraegeSettingsStore((s) => s.defaultNoticePeriodDays)
  const defaultRenewal = useVertraegeSettingsStore((s) => s.defaultRenewal)
  const setDefaultDurationMonths = useVertraegeSettingsStore((s) => s.setDefaultDurationMonths)
  const setDefaultNoticePeriodDays = useVertraegeSettingsStore((s) => s.setDefaultNoticePeriodDays)
  const setDefaultRenewal = useVertraegeSettingsStore((s) => s.setDefaultRenewal)

  const numInput =
    'h-9 w-full rounded-lg border border-border bg-transparent px-2 text-sm text-foreground outline-none focus:border-primary'

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">
            {t('vertraege.settings.defaults.durationMonths')}
          </label>
          <input
            type="number"
            min={1}
            value={defaultDurationMonths}
            onChange={(e) => setDefaultDurationMonths(Math.max(1, Number(e.target.value)))}
            className={numInput}
          />
        </div>
        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">
            {t('vertraege.settings.defaults.noticePeriodDays')}
          </label>
          <input
            type="number"
            min={0}
            value={defaultNoticePeriodDays}
            onChange={(e) => setDefaultNoticePeriodDays(Math.max(0, Number(e.target.value)))}
            className={numInput}
          />
        </div>
      </div>
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('vertraege.settings.defaults.renewal')}
        </label>
        <div className="flex gap-2">
          <button onClick={() => setDefaultRenewal('auto')} className={segBtn(defaultRenewal === 'auto')}>
            {t('vertraege.settings.defaults.renewalAuto')}
          </button>
          <button onClick={() => setDefaultRenewal('manual')} className={segBtn(defaultRenewal === 'manual')}>
            {t('vertraege.settings.defaults.renewalManual')}
          </button>
        </div>
      </div>
      <p className="text-xs text-muted-foreground">{t('vertraege.settings.defaults.hint')}</p>
    </div>
  )
}

// ─── Für alle: Nummernkreis ──────────────────────────────────────

function NumberFormatSettings() {
  const { t } = useTranslation()
  const numberFormat = useVertraegeSettingsStore((s) => s.numberFormat)
  const setNumberFormat = useVertraegeSettingsStore((s) => s.setNumberFormat)
  const numberCounter = useVertraegeSettingsStore((s) => s.numberCounter)

  // Live preview of the actual next number (format + running counter).
  const preview = formatContractNumber(numberFormat, numberCounter)

  return (
    <div className="space-y-2">
      <input
        type="text"
        value={numberFormat}
        onChange={(e) => setNumberFormat(e.target.value)}
        placeholder="V-{JAHR}-{NR}"
        className="h-9 w-full rounded-lg border border-border bg-transparent px-2 font-mono text-sm text-foreground outline-none focus:border-primary"
      />
      <p className="text-xs text-muted-foreground">
        {t('vertraege.settings.numberFormat.preview', { preview })}
      </p>
      <p className="text-xs text-muted-foreground">{t('vertraege.settings.numberFormat.hint')}</p>
    </div>
  )
}

// ─── Für alle: Erinnerungs-Defaults ──────────────────────────────

function TenantReminderSettings() {
  const { t } = useTranslation()
  const tenantReminderDays = useVertraegeSettingsStore((s) => s.tenantReminderDays)
  const toggleTenantReminderDay = useVertraegeSettingsStore((s) => s.toggleTenantReminderDay)

  return (
    <div className="space-y-2">
      <div className="flex gap-4">
        {REMINDER_CHOICES.map((days) => (
          <label key={days} className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={tenantReminderDays.includes(days)}
              onChange={() => toggleTenantReminderDay(days)}
              className="h-4 w-4 rounded border-border accent-primary"
            />
            <span className="text-sm text-foreground">
              {t('vertraege.settings.reminderDays', { days })}
            </span>
          </label>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">{t('vertraege.settings.tenantReminders.hint')}</p>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * VertraegeSettingsPanel — the "Verträge" entry of the module-settings overlay.
 * Personal prefs are always editable and apply for real inside the module
 * (default tab, table density, reminder pre-selection). Tenant sections run
 * mock-first on stores/vertraegeSettings until the backend persists them
 * (backend-handover-luke.md) — enabled types and defaults already drive the
 * new-contract dialog.
 */
export function VertraegeSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'vertraege.settings.personal.title',
      descriptionKey: 'vertraege.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <VertraegePersonalPrefs />,
    },
    {
      id: 'types',
      titleKey: 'vertraege.settings.types.title',
      descriptionKey: 'vertraege.settings.types.desc',
      scope: 'tenant',
      icon: Shapes,
      children: <ContractTypesSettings />,
    },
    {
      id: 'defaults',
      titleKey: 'vertraege.settings.defaults.title',
      descriptionKey: 'vertraege.settings.defaults.desc',
      scope: 'tenant',
      icon: CalendarClock,
      children: <ContractDefaultsSettings />,
    },
    {
      id: 'numberFormat',
      titleKey: 'vertraege.settings.numberFormat.title',
      descriptionKey: 'vertraege.settings.numberFormat.desc',
      scope: 'tenant',
      icon: Hash,
      children: <NumberFormatSettings />,
    },
    {
      id: 'reminders',
      titleKey: 'vertraege.settings.tenantReminders.title',
      descriptionKey: 'vertraege.settings.tenantReminders.desc',
      scope: 'tenant',
      icon: Bell,
      children: <TenantReminderSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="vertraege"
      titleKey="vertraege.settings.title"
      descriptionKey="vertraege.settings.subtitle"
      sections={sections}
    />
  )
}
