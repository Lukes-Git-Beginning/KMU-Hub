/**
 * BexioConfigPanel — custom config panel for Bexio integration.
 *
 * Features: OAuth2 mock flow, sync-scope checkboxes, conflict handling,
 * sync interval selection.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Building2, Loader2, Plug, Unlink, RefreshCw, Zap } from 'lucide-react'
import { toast } from 'sonner'
import { useIntegrationStore } from '@/stores/integrations'
import { formatDate } from '@/lib/format'

const STATUS_BADGE_KEYS: Record<string, { labelKey: string; cls: string }> = {
  connected: { labelKey: 'settings.integrations.status.connected', cls: 'bg-success-light text-success' },
  disconnected: { labelKey: 'settings.integrations.status.disconnected', cls: 'bg-secondary text-muted-foreground' },
  syncing: { labelKey: 'settings.integrations.status.syncing', cls: 'bg-info-light text-info' },
  error: { labelKey: 'settings.integrations.status.error', cls: 'bg-error-light text-destructive' },
}

const SYNC_SCOPES = [
  { id: 'contacts', labelKey: 'settings.integrations.bexio.syncContacts', helpTextKey: 'settings.integrations.bexio.syncContactsHelp' },
  { id: 'invoices', labelKey: 'settings.integrations.bexio.syncInvoices', helpTextKey: 'settings.integrations.bexio.syncInvoicesHelp' },
  { id: 'products', labelKey: 'settings.integrations.bexio.syncProducts', helpTextKey: 'settings.integrations.bexio.syncProductsHelp' },
  { id: 'projects', labelKey: 'settings.integrations.bexio.syncProjects', helpTextKey: 'settings.integrations.bexio.syncProjectsHelp' },
]

const CONFLICT_OPTIONS = [
  { value: 'kmuhub', labelKey: 'settings.integrations.bexio.conflictCosmi' },
  { value: 'bexio', labelKey: 'settings.integrations.bexio.conflictBexio' },
  { value: 'manual', labelKey: 'settings.integrations.bexio.conflictManual' },
]

const INTERVAL_OPTIONS = [
  { value: '5', labelKey: 'settings.integrations.interval.5min' },
  { value: '15', labelKey: 'settings.integrations.interval.15min' },
  { value: '60', labelKey: 'settings.integrations.interval.hourly' },
  { value: '1440', labelKey: 'settings.integrations.interval.daily' },
]

interface BexioConfigPanelProps {
  onBack: () => void
}

export function BexioConfigPanel({ onBack }: BexioConfigPanelProps) {
  const { t } = useTranslation()
  const store = useIntegrationStore()
  const status = store.getStatus('bexio')
  const integration = store.integrations['bexio']
  const storedValues = store.getFieldValues('bexio')

  const [oauthState, setOauthState] = useState<'idle' | 'loading'>('idle')
  const [syncContacts, setSyncContacts] = useState((storedValues.syncContacts as boolean) ?? true)
  const [syncInvoices, setSyncInvoices] = useState((storedValues.syncInvoices as boolean) ?? true)
  const [syncProducts, setSyncProducts] = useState((storedValues.syncProducts as boolean) ?? false)
  const [syncProjects, setSyncProjects] = useState((storedValues.syncProjects as boolean) ?? false)
  const [conflictHandling, setConflictHandling] = useState((storedValues.conflictHandling as string) ?? 'kmuhub')
  const [syncInterval, setSyncInterval] = useState((storedValues.syncInterval as string) ?? '15')
  const [testing, setTesting] = useState(false)

  const badgeConfig = STATUS_BADGE_KEYS[status] ?? STATUS_BADGE_KEYS.disconnected

  const scopeStates: Record<string, { value: boolean; set: (v: boolean) => void }> = {
    contacts: { value: syncContacts, set: setSyncContacts },
    invoices: { value: syncInvoices, set: setSyncInvoices },
    products: { value: syncProducts, set: setSyncProducts },
    projects: { value: syncProjects, set: setSyncProjects },
  }

  const handleOAuthConnect = () => {
    setOauthState('loading')
    setTimeout(() => {
      setOauthState('idle')
      store.setFieldValues('bexio', {
        syncContacts,
        syncInvoices,
        syncProducts,
        syncProjects,
        conflictHandling,
        syncInterval,
      })
      store.connect('bexio')
      toast.success(t('settings.integrations.bexio.connected'))
    }, 1500)
  }

  const handleDisconnect = () => {
    store.disconnect('bexio')
    toast.success(t('settings.integrations.bexio.disconnected'))
  }

  const handleSave = () => {
    store.setFieldValues('bexio', {
      syncContacts,
      syncInvoices,
      syncProducts,
      syncProjects,
      conflictHandling,
      syncInterval,
    })
    toast.success(t('settings.integrations.bexio.settingsSaved'))
  }

  const handleSync = () => {
    store.triggerSync('bexio')
    toast.success(t('settings.integrations.bexio.syncStarted'))
  }

  const handleTest = () => {
    setTesting(true)
    setTimeout(() => {
      setTesting(false)
      toast.success(t('settings.integrations.bexio.connectionSuccess'))
    }, 1000)
  }

  return (
    <div className="max-w-2xl">
      {/* Back */}
      <button
        onClick={onBack}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        {t('settings.integrations.backToIntegrations')}
      </button>

      {/* Header */}
      <div className="flex items-start gap-4 mb-6">
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-secondary">
          <Building2 className="h-6 w-6 text-info" />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold text-foreground">Bexio</h2>
            <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-medium ${badgeConfig.cls}`}>
              {t(badgeConfig.labelKey)}
            </span>
          </div>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t('settings.integrations.bexio.description')}
          </p>
        </div>
      </div>

      {/* OAuth2 Connection */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">{t('settings.integrations.connection')}</h3>
        {status === 'connected' ? (
          <div className="rounded-lg border border-success/30 bg-success-light p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-success">
                  {t('settings.integrations.bexio.connectedWith')}
                </p>
                <p className="text-xs text-success/70 mt-0.5">
                  Seit {integration?.connectedAt ? formatDate(integration.connectedAt) : '—'}
                  {integration?.lastSync && (
                    <span className="ml-2">
                      · Letzte Sync: {new Date(integration.lastSync).toLocaleString('de-DE')}
                    </span>
                  )}
                </p>
              </div>
              <button
                onClick={handleDisconnect}
                className="flex items-center gap-1.5 rounded-lg border border-success/30 px-3 py-1.5 text-sm text-success hover:bg-success-light transition-colors"
              >
                <Unlink className="h-3.5 w-3.5" />
                {t('settings.integrations.disconnect')}
              </button>
            </div>
          </div>
        ) : (
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground mb-3">
              {t('settings.integrations.bexio.oauthPrompt')}
            </p>
            <button
              onClick={handleOAuthConnect}
              disabled={oauthState === 'loading'}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-dark transition-colors disabled:opacity-50"
            >
              {oauthState === 'loading' ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Plug className="h-4 w-4" />
              )}
              {oauthState === 'loading' ? t('settings.integrations.bexio.connecting') : t('settings.integrations.bexio.connect')}
            </button>
          </div>
        )}
      </div>

      {/* Sync Scope */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">{t('settings.integrations.syncScopes')}</h3>
        <div className="space-y-2">
          {SYNC_SCOPES.map((scope) => {
            const state = scopeStates[scope.id]
            return (
              <label
                key={scope.id}
                className="flex items-center justify-between rounded-lg border border-border p-3 cursor-pointer hover:bg-secondary/30 transition-colors"
              >
                <div>
                  <span className="text-sm font-medium text-foreground">{t(scope.labelKey)}</span>
                  <p className="text-[11px] text-muted-foreground">{t(scope.helpTextKey)}</p>
                </div>
                <button
                  onClick={(e) => {
                    e.preventDefault()
                    state.set(!state.value)
                  }}
                  className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors ${
                    state.value ? 'bg-primary' : 'bg-muted'
                  }`}
                >
                  <span
                    className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                      state.value ? 'translate-x-4.5' : 'translate-x-0.5'
                    }`}
                  />
                </button>
              </label>
            )
          })}
        </div>
      </div>

      {/* Conflict Handling */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">{t('settings.integrations.conflictHandling')}</h3>
        <select
          value={conflictHandling}
          onChange={(e) => setConflictHandling(e.target.value)}
          className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
        >
          {CONFLICT_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>{t(opt.labelKey)}</option>
          ))}
        </select>
        <p className="text-[11px] text-muted-foreground mt-1.5">
          {t('settings.integrations.conflictHandlingHelp')}
        </p>
      </div>

      {/* Sync Interval */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">{t('settings.integrations.syncInterval')}</h3>
        <select
          value={syncInterval}
          onChange={(e) => setSyncInterval(e.target.value)}
          className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
        >
          {INTERVAL_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>{t(opt.labelKey)}</option>
          ))}
        </select>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2 border-t border-border pt-4 mt-6">
        {status === 'connected' && (
          <button
            onClick={handleSync}
            disabled={status === 'syncing'}
            className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-muted transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`h-4 w-4 ${status === 'syncing' ? 'animate-spin' : ''}`} />
            {t('settings.integrations.syncNow')}
          </button>
        )}
        <button
          onClick={handleSave}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          {t('common.save')}
        </button>
        {status === 'connected' && (
          <button
            onClick={handleTest}
            disabled={testing}
            className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-muted transition-colors disabled:opacity-50 ml-auto"
          >
            {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />}
            {t('settings.integrations.testConnection')}
          </button>
        )}
      </div>
    </div>
  )
}
