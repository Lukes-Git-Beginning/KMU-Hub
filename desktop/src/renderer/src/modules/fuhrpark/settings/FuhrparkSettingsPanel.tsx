import { useTranslation } from 'react-i18next'
import { Truck, ShieldCheck, ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { Switch } from '@/components/ui/switch'
import {
  useFuhrparkPrefsStore,
  type FuhrparkTab,
  type TripCategory,
} from '@/stores/fuhrparkPrefs'
import {
  useFuhrparkTenantStore,
  FUHRPARK_CURRENCY_OPTIONS,
  FUHRPARK_FUEL_OPTIONS,
  type FuhrparkFuelType,
} from '@/stores/fuhrparkTenant'

// ─── Persönlich ──────────────────────────────────────────────────

function FuhrparkPersonalPrefs() {
  const { t } = useTranslation()
  const defaultTab = useFuhrparkPrefsStore((s) => s.defaultTab)
  const showTireReminder = useFuhrparkPrefsStore((s) => s.showTireReminder)
  const defaultTripCategory = useFuhrparkPrefsStore((s) => s.defaultTripCategory)
  const setDefaultTab = useFuhrparkPrefsStore((s) => s.setDefaultTab)
  const setShowTireReminder = useFuhrparkPrefsStore((s) => s.setShowTireReminder)
  const setDefaultTripCategory = useFuhrparkPrefsStore((s) => s.setDefaultTripCategory)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('fuhrpark.settings.personal.defaultTab')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultTab}
            onChange={(e) => setDefaultTab(e.target.value as FuhrparkTab)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            <option value="fahrzeuge">{t('fuhrpark.settings.personal.tabFahrzeuge')}</option>
            <option value="wartung">{t('fuhrpark.settings.personal.tabWartung')}</option>
            <option value="tankprotokoll">{t('fuhrpark.settings.personal.tabTanken')}</option>
            <option value="fahrtenbuch">{t('fuhrpark.settings.personal.tabFahrtenbuch')}</option>
            <option value="tracking">{t('fuhrpark.tab.tracking')}</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('fuhrpark.settings.personal.defaultTabHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('fuhrpark.settings.personal.defaultTripCategory')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultTripCategory}
            onChange={(e) => setDefaultTripCategory(e.target.value as TripCategory)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            <option value="business">{t('fuhrpark.logbook.business')}</option>
            <option value="private">{t('fuhrpark.logbook.private')}</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('fuhrpark.settings.personal.defaultTripCategoryHint')}</p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{t('fuhrpark.settings.personal.showTireReminder')}</p>
          <p className="text-xs text-muted-foreground">{t('fuhrpark.settings.personal.showTireReminderHint')}</p>
        </div>
        <Switch checked={showTireReminder} onCheckedChange={setShowTireReminder} />
      </div>
    </div>
  )
}

// ─── Für alle (Tenant) ───────────────────────────────────────────

function FuhrparkTenantSettings() {
  const { t } = useTranslation()
  const reminderLeadDays = useFuhrparkTenantStore((s) => s.reminderLeadDays)
  const currency = useFuhrparkTenantStore((s) => s.currency)
  const defaultFuelType = useFuhrparkTenantStore((s) => s.defaultFuelType)
  const privateTripsEnabled = useFuhrparkTenantStore((s) => s.privateTripsEnabled)
  const setReminderLeadDays = useFuhrparkTenantStore((s) => s.setReminderLeadDays)
  const setCurrency = useFuhrparkTenantStore((s) => s.setCurrency)
  const setDefaultFuelType = useFuhrparkTenantStore((s) => s.setDefaultFuelType)
  const setPrivateTripsEnabled = useFuhrparkTenantStore((s) => s.setPrivateTripsEnabled)

  const fuelLabels: Record<FuhrparkFuelType, string> = {
    diesel: t('fuhrpark.fuelType.diesel'),
    petrol: t('fuhrpark.fuelType.petrol'),
    electric: t('fuhrpark.fuelType.electric'),
    hybrid: t('fuhrpark.fuelType.hybrid'),
  }

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('fuhrpark.settings.tenant.reminderLeadDays')}</label>
        <input
          type="number"
          min={7}
          max={180}
          value={reminderLeadDays}
          onChange={(e) => setReminderLeadDays(Number(e.target.value))}
          className="w-24 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
        />
        <p className="text-xs text-muted-foreground">{t('fuhrpark.settings.tenant.reminderLeadDaysHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('fuhrpark.settings.tenant.currency')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={currency}
            onChange={(e) => setCurrency(e.target.value)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            {FUHRPARK_CURRENCY_OPTIONS.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('fuhrpark.settings.tenant.currencyHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('fuhrpark.settings.tenant.defaultFuelType')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultFuelType}
            onChange={(e) => setDefaultFuelType(e.target.value as FuhrparkFuelType)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            {FUHRPARK_FUEL_OPTIONS.map((f) => (
              <option key={f} value={f}>{fuelLabels[f]}</option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('fuhrpark.settings.tenant.defaultFuelTypeHint')}</p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{t('fuhrpark.settings.tenant.privateTrips')}</p>
          <p className="text-xs text-muted-foreground">{t('fuhrpark.settings.tenant.privateTripsHint')}</p>
        </div>
        <Switch checked={privateTripsEnabled} onCheckedChange={setPrivateTripsEnabled} />
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * FuhrparkSettingsPanel — the "Fuhrpark" entry of the module-settings overlay.
 * Personal prefs (default tab, trip category, tire banner) apply per user;
 * the tenant block (reminder lead, currency, fuel default, private-trip
 * policy) is editable only by a Fuhrpark-Modul-Leiter / admin.
 */
export function FuhrparkSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'fuhrpark.settings.personal.title',
      descriptionKey: 'fuhrpark.settings.personal.desc',
      scope: 'personal',
      icon: Truck,
      children: <FuhrparkPersonalPrefs />,
    },
    {
      id: 'tenant',
      titleKey: 'fuhrpark.settings.tenant.title',
      descriptionKey: 'fuhrpark.settings.tenant.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <FuhrparkTenantSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="fuhrpark"
      titleKey="fuhrpark.settings.panel.title"
      descriptionKey="fuhrpark.settings.panel.subtitle"
      sections={sections}
    />
  )
}
