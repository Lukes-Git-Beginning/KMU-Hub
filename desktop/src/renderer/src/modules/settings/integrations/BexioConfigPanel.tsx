/**
 * BexioConfigPanel — custom config panel for Bexio integration.
 *
 * Features: OAuth2 mock flow, sync-scope checkboxes, conflict handling,
 * sync interval selection.
 */
import { useState } from 'react'
import { ArrowLeft, Building2, Loader2, Plug, Unlink, RefreshCw, Zap } from 'lucide-react'
import { toast } from 'sonner'
import { useIntegrationStore } from '@/stores/integrations'

const STATUS_BADGE: Record<string, { label: string; cls: string }> = {
  connected: { label: 'Verbunden', cls: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
  disconnected: { label: 'Nicht verbunden', cls: 'bg-secondary text-muted-foreground' },
  syncing: { label: 'Synchronisiert...', cls: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' },
  error: { label: 'Fehler', cls: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' },
}

const SYNC_SCOPES = [
  { id: 'contacts', label: 'Kontakte synchronisieren', helpText: 'Kunden und Lieferanten mit Bexio abgleichen' },
  { id: 'invoices', label: 'Rechnungen synchronisieren', helpText: 'Ausgangsrechnungen an Bexio übertragen' },
  { id: 'products', label: 'Produkte synchronisieren', helpText: 'Artikelstamm zwischen Systemen abgleichen' },
  { id: 'projects', label: 'Projekte synchronisieren', helpText: 'Projektdaten bidirektional synchronisieren' },
]

const CONFLICT_OPTIONS = [
  { value: 'kmuhub', label: 'Cosmi hat Vorrang' },
  { value: 'bexio', label: 'Bexio hat Vorrang' },
  { value: 'manual', label: 'Manuell abgleichen' },
]

const INTERVAL_OPTIONS = [
  { value: '5', label: 'Alle 5 Minuten' },
  { value: '15', label: 'Alle 15 Minuten' },
  { value: '60', label: 'Jede Stunde' },
  { value: '1440', label: 'Taeglich' },
]

interface BexioConfigPanelProps {
  onBack: () => void
}

export function BexioConfigPanel({ onBack }: BexioConfigPanelProps) {
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

  const badge = STATUS_BADGE[status] ?? STATUS_BADGE.disconnected

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
      toast.success('Bexio erfolgreich verbunden')
    }, 1500)
  }

  const handleDisconnect = () => {
    store.disconnect('bexio')
    toast.success('Bexio getrennt')
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
    toast.success('Bexio-Einstellungen gespeichert')
  }

  const handleSync = () => {
    store.triggerSync('bexio')
    toast.success('Bexio-Synchronisation gestartet')
  }

  const handleTest = () => {
    setTesting(true)
    setTimeout(() => {
      setTesting(false)
      toast.success('Bexio-Verbindung erfolgreich')
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
        Zurück zu Integrationen
      </button>

      {/* Header */}
      <div className="flex items-start gap-4 mb-6">
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-secondary">
          <Building2 className="h-6 w-6 text-blue-600" />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold text-foreground">Bexio</h2>
            <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-medium ${badge.cls}`}>
              {badge.label}
            </span>
          </div>
          <p className="text-sm text-muted-foreground mt-0.5">
            Buchhaltungssoftware mit Kontakt-, Rechnungs- und Produkt-Sync
          </p>
        </div>
      </div>

      {/* OAuth2 Connection */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Verbindung</h3>
        {status === 'connected' ? (
          <div className="rounded-lg border border-green-200 bg-green-50 dark:border-green-900/50 dark:bg-green-900/10 p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-green-700 dark:text-green-400">
                  Verbunden mit Bexio
                </p>
                <p className="text-xs text-green-600/70 dark:text-green-500/70 mt-0.5">
                  Seit {integration?.connectedAt ? new Date(integration.connectedAt).toLocaleDateString('de-DE') : '—'}
                  {integration?.lastSync && (
                    <span className="ml-2">
                      · Letzte Sync: {new Date(integration.lastSync).toLocaleString('de-DE')}
                    </span>
                  )}
                </p>
              </div>
              <button
                onClick={handleDisconnect}
                className="flex items-center gap-1.5 rounded-lg border border-green-300 dark:border-green-800 px-3 py-1.5 text-sm text-green-700 dark:text-green-400 hover:bg-green-100 dark:hover:bg-green-900/30 transition-colors"
              >
                <Unlink className="h-3.5 w-3.5" />
                Trennen
              </button>
            </div>
          </div>
        ) : (
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground mb-3">
              Verbinde dein Bexio-Konto ueber OAuth2 um Daten automatisch zu synchronisieren.
            </p>
            <button
              onClick={handleOAuthConnect}
              disabled={oauthState === 'loading'}
              className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              {oauthState === 'loading' ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Plug className="h-4 w-4" />
              )}
              {oauthState === 'loading' ? 'Verbinde mit Bexio...' : 'Mit Bexio verbinden'}
            </button>
          </div>
        )}
      </div>

      {/* Sync Scope */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Sync-Bereiche</h3>
        <div className="space-y-2">
          {SYNC_SCOPES.map((scope) => {
            const state = scopeStates[scope.id]
            return (
              <label
                key={scope.id}
                className="flex items-center justify-between rounded-lg border border-border p-3 cursor-pointer hover:bg-secondary/30 transition-colors"
              >
                <div>
                  <span className="text-sm font-medium text-foreground">{scope.label}</span>
                  <p className="text-[11px] text-muted-foreground">{scope.helpText}</p>
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
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Konfliktbehandlung</h3>
        <select
          value={conflictHandling}
          onChange={(e) => setConflictHandling(e.target.value)}
          className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
        >
          {CONFLICT_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
        <p className="text-[11px] text-muted-foreground mt-1.5">
          Bestimmt welche Daten bei Konflikten bevorzugt werden
        </p>
      </div>

      {/* Sync Interval */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Sync-Intervall</h3>
        <select
          value={syncInterval}
          onChange={(e) => setSyncInterval(e.target.value)}
          className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
        >
          {INTERVAL_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
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
            Sync jetzt ausführen
          </button>
        )}
        <button
          onClick={handleSave}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          Speichern
        </button>
        {status === 'connected' && (
          <button
            onClick={handleTest}
            disabled={testing}
            className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-muted transition-colors disabled:opacity-50 ml-auto"
          >
            {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />}
            Verbindung testen
          </button>
        )}
      </div>
    </div>
  )
}
