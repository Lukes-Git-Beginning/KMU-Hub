import { useTranslation } from 'react-i18next'
import { ShoppingCart, ShieldCheck, ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import {
  useEinkaufPrefsStore,
  type EinkaufTab,
  type EinkaufStatusFilter,
} from '@/stores/einkaufPrefs'
import { useEinkaufTenantStore, EINKAUF_CURRENCY_OPTIONS } from '@/stores/einkaufTenant'
import { PAYMENT_TERMS_OPTIONS, orderStatusLabels } from '../einkauf-shared'

const selectCls =
  'w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer'

// ─── Persönlich ──────────────────────────────────────────────────

const STATUS_FILTER_VALUES: EinkaufStatusFilter[] = [
  'all', 'draft', 'submitted', 'sent', 'partially_received', 'received', 'cancelled',
]

function EinkaufPersonalPrefs() {
  const { t } = useTranslation()
  const defaultTab = useEinkaufPrefsStore((s) => s.defaultTab)
  const defaultStatusFilter = useEinkaufPrefsStore((s) => s.defaultStatusFilter)
  const setDefaultTab = useEinkaufPrefsStore((s) => s.setDefaultTab)
  const setDefaultStatusFilter = useEinkaufPrefsStore((s) => s.setDefaultStatusFilter)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('einkauf.settings.personal.defaultTab')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultTab}
            onChange={(e) => setDefaultTab(e.target.value as EinkaufTab)}
            className={selectCls}
          >
            <option value="bestellungen">{t('einkauf.settings.personal.tabBestellungen')}</option>
            <option value="lieferanten">{t('einkauf.settings.personal.tabLieferanten')}</option>
            <option value="katalog">{t('einkauf.settings.personal.tabKatalog')}</option>
            <option value="rahmenverträge">{t('einkauf.settings.personal.tabRahmenvertraege')}</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('einkauf.settings.personal.defaultTabHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('einkauf.settings.personal.defaultStatusFilter')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultStatusFilter}
            onChange={(e) => setDefaultStatusFilter(e.target.value as EinkaufStatusFilter)}
            className={selectCls}
          >
            {STATUS_FILTER_VALUES.map((v) => (
              <option key={v} value={v}>
                {v === 'all' ? t('einkauf.statusFilter.all') : t(orderStatusLabels[v] ?? v)}
              </option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('einkauf.settings.personal.defaultStatusFilterHint')}</p>
      </div>
    </div>
  )
}

// ─── Für alle (Tenant) ───────────────────────────────────────────

function EinkaufTenantSettings() {
  const { t } = useTranslation()
  const approvalThreshold = useEinkaufTenantStore((s) => s.approvalThreshold)
  const currency = useEinkaufTenantStore((s) => s.currency)
  const defaultPaymentTerms = useEinkaufTenantStore((s) => s.defaultPaymentTerms)
  const poNumberPrefix = useEinkaufTenantStore((s) => s.poNumberPrefix)
  const setApprovalThreshold = useEinkaufTenantStore((s) => s.setApprovalThreshold)
  const setCurrency = useEinkaufTenantStore((s) => s.setCurrency)
  const setDefaultPaymentTerms = useEinkaufTenantStore((s) => s.setDefaultPaymentTerms)
  const setPoNumberPrefix = useEinkaufTenantStore((s) => s.setPoNumberPrefix)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('einkauf.settings.tenant.approvalThreshold')}</label>
        <input
          type="number"
          min={0}
          step={100}
          value={approvalThreshold}
          onChange={(e) => setApprovalThreshold(Number(e.target.value))}
          className="w-32 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
        />
        <p className="text-xs text-muted-foreground">{t('einkauf.settings.tenant.approvalThresholdHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('einkauf.settings.tenant.currency')}</label>
        <div className="relative w-64 max-w-full">
          <select value={currency} onChange={(e) => setCurrency(e.target.value)} className={selectCls}>
            {EINKAUF_CURRENCY_OPTIONS.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('einkauf.settings.tenant.currencyHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('einkauf.settings.tenant.defaultPaymentTerms')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultPaymentTerms}
            onChange={(e) => setDefaultPaymentTerms(e.target.value)}
            className={selectCls}
          >
            {(PAYMENT_TERMS_OPTIONS.includes(defaultPaymentTerms)
              ? PAYMENT_TERMS_OPTIONS
              : [defaultPaymentTerms, ...PAYMENT_TERMS_OPTIONS]
            ).map((pt) => (
              <option key={pt} value={pt}>{pt}</option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('einkauf.settings.tenant.defaultPaymentTermsHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('einkauf.settings.tenant.poNumberPrefix')}</label>
        <input
          type="text"
          maxLength={8}
          value={poNumberPrefix}
          onChange={(e) => setPoNumberPrefix(e.target.value)}
          className="w-32 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground font-mono uppercase focus:outline-none focus:ring-2 focus:ring-focus-ring"
        />
        <p className="text-xs text-muted-foreground">
          {t('einkauf.settings.tenant.poNumberPrefixHint', {
            example: `${poNumberPrefix}-${new Date().getFullYear()}-011`,
          })}
        </p>
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * EinkaufSettingsPanel — the "Einkauf" entry of the module-settings overlay.
 * Personal prefs (default tab, saved status filter) apply per user; the
 * tenant block (approval threshold, currency, payment-terms default, PO
 * number prefix) is editable only by an Einkauf-Modul-Leiter / admin.
 */
export function EinkaufSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'einkauf.settings.personal.title',
      descriptionKey: 'einkauf.settings.personal.desc',
      scope: 'personal',
      icon: ShoppingCart,
      children: <EinkaufPersonalPrefs />,
    },
    {
      id: 'tenant',
      titleKey: 'einkauf.settings.tenant.title',
      descriptionKey: 'einkauf.settings.tenant.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <EinkaufTenantSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="einkauf"
      titleKey="einkauf.settings.panel.title"
      descriptionKey="einkauf.settings.panel.subtitle"
      sections={sections}
    />
  )
}
