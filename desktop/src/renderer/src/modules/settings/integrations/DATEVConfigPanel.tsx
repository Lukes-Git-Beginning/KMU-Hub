/**
 * DATEVConfigPanel — custom config panel for DATEV Rechnungswesen.
 *
 * Features: Mandant/Berater numbers, export method selection,
 * Konten-Mapping table, BWA/SuSa toggle.
 * Different from HRIntegrationPanel's "DATEV Lohn" (payroll).
 */
import { useState } from 'react'
import { ArrowLeft, Building2, Loader2, Plug, Unlink, RefreshCw, Zap, Plus, Pencil, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useIntegrationStore } from '@/stores/integrations'

interface KontoMapping {
  id: string
  kmuLabel: string
  datevKonto: string
}

const DEFAULT_MAPPINGS: KontoMapping[] = [
  { id: 'km-1', kmuLabel: 'Erloese 19%', datevKonto: '8400' },
  { id: 'km-2', kmuLabel: 'Erloese 7%', datevKonto: '8300' },
  { id: 'km-3', kmuLabel: 'Betriebsausgaben', datevKonto: '4000' },
  { id: 'km-4', kmuLabel: 'Lohn & Gehalt', datevKonto: '6000' },
]

const STATUS_BADGE: Record<string, { label: string; cls: string }> = {
  connected: { label: 'Verbunden', cls: 'bg-success-light text-success' },
  disconnected: { label: 'Nicht verbunden', cls: 'bg-secondary text-muted-foreground' },
  syncing: { label: 'Synchronisiert...', cls: 'bg-info-light text-info' },
  error: { label: 'Fehler', cls: 'bg-error-light text-destructive' },
}

interface DATEVConfigPanelProps {
  onBack: () => void
}

export function DATEVConfigPanel({ onBack }: DATEVConfigPanelProps) {
  const store = useIntegrationStore()
  const status = store.getStatus('datev-rechnungswesen')
  const integration = store.integrations['datev-rechnungswesen']
  const storedValues = store.getFieldValues('datev-rechnungswesen')

  const [mandantNr, setMandantNr] = useState((storedValues.mandantNr as string) ?? '')
  const [beraterNr, setBeraterNr] = useState((storedValues.beraterNr as string) ?? '')
  const [exportMethod, setExportMethod] = useState<'connect' | 'file'>((storedValues.exportMethod as 'connect' | 'file') ?? 'connect')
  const [bwaEnabled, setBwaEnabled] = useState((storedValues.bwaEnabled as boolean) ?? true)
  const [susaEnabled, setSusaEnabled] = useState((storedValues.susaEnabled as boolean) ?? true)
  const [kontenblaetter, setKontenblaetter] = useState((storedValues.kontenblaetter as boolean) ?? false)
  const [mappings, setMappings] = useState<KontoMapping[]>(DEFAULT_MAPPINGS)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editKonto, setEditKonto] = useState('')
  const [testing, setTesting] = useState(false)

  const badge = STATUS_BADGE[status] ?? STATUS_BADGE.disconnected

  const handleSave = () => {
    store.setFieldValues('datev-rechnungswesen', {
      mandantNr,
      beraterNr,
      exportMethod,
      bwaEnabled,
      susaEnabled,
      kontenblaetter,
    })
    toast.success('DATEV-Einstellungen gespeichert')
  }

  const handleConnect = () => {
    handleSave()
    store.connect('datev-rechnungswesen')
    toast.success('DATEV Rechnungswesen verbunden')
  }

  const handleDisconnect = () => {
    store.disconnect('datev-rechnungswesen')
    toast.success('DATEV Rechnungswesen getrennt')
  }

  const handleTest = () => {
    setTesting(true)
    setTimeout(() => {
      setTesting(false)
      toast.success('DATEV-Verbindung erfolgreich')
    }, 1000)
  }

  const handleSync = () => {
    store.triggerSync('datev-rechnungswesen')
    toast.success('DATEV-Synchronisation gestartet')
  }

  const startEdit = (mapping: KontoMapping) => {
    setEditingId(mapping.id)
    setEditKonto(mapping.datevKonto)
  }

  const saveEdit = () => {
    if (!editingId) return
    setMappings((prev) =>
      prev.map((m) => (m.id === editingId ? { ...m, datevKonto: editKonto } : m)),
    )
    setEditingId(null)
    setEditKonto('')
  }

  const addMapping = () => {
    const newId = `km-${Date.now()}`
    setMappings((prev) => [...prev, { id: newId, kmuLabel: 'Neues Konto', datevKonto: '' }])
    setEditingId(newId)
    setEditKonto('')
  }

  const removeMapping = (id: string) => {
    setMappings((prev) => prev.filter((m) => m.id !== id))
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
          <Building2 className="h-6 w-6 text-success" />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold text-foreground">DATEV Rechnungswesen</h2>
            <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-medium ${badge.cls}`}>
              {badge.label}
            </span>
          </div>
          <p className="text-sm text-muted-foreground mt-0.5">
            Konten-Mapping, BWA/SuSa-Export, DATEV-Connect oder Dateiexport
          </p>
        </div>
      </div>

      {/* Connected info */}
      {status === 'connected' && integration?.connectedAt && (
        <div className="rounded-lg border border-success/30 bg-success-light px-4 py-2.5 mb-6 text-sm text-success">
          Verbunden seit {new Date(integration.connectedAt).toLocaleDateString('de-DE')}
          {integration.lastSync && (
            <span className="ml-3 text-success/70">
              Letzte Sync: {new Date(integration.lastSync).toLocaleString('de-DE')}
            </span>
          )}
        </div>
      )}

      {/* Mandant */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Mandant</h3>
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Mandantennummer</label>
            <input
              type="text"
              value={mandantNr}
              onChange={(e) => setMandantNr(e.target.value)}
              placeholder="z.B. 12345"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground font-mono placeholder:text-muted-foreground"
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Beraternummer</label>
            <input
              type="text"
              value={beraterNr}
              onChange={(e) => setBeraterNr(e.target.value)}
              placeholder="z.B. 67890"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground font-mono placeholder:text-muted-foreground"
            />
          </div>
        </div>
      </div>

      {/* Export method */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Exportmethode</h3>
        <div className="space-y-2">
          <label className="flex items-center gap-3 rounded-lg border border-border p-3 cursor-pointer hover:bg-secondary/30 transition-colors">
            <input
              type="radio"
              name="exportMethod"
              checked={exportMethod === 'connect'}
              onChange={() => setExportMethod('connect')}
              className="accent-primary"
            />
            <div>
              <span className="text-sm font-medium text-foreground">DATEV-Connect Online (API)</span>
              <p className="text-[11px] text-muted-foreground">Direkte Übertragung an DATEV Rechenzentrum</p>
            </div>
          </label>
          <label className="flex items-center gap-3 rounded-lg border border-border p-3 cursor-pointer hover:bg-secondary/30 transition-colors">
            <input
              type="radio"
              name="exportMethod"
              checked={exportMethod === 'file'}
              onChange={() => setExportMethod('file')}
              className="accent-primary"
            />
            <div>
              <span className="text-sm font-medium text-foreground">Dateiexport (ASCII/CSV)</span>
              <p className="text-[11px] text-muted-foreground">Export-Dateien zum manuellen Import in DATEV</p>
            </div>
          </label>
        </div>
      </div>

      {/* Konten-Mapping */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Konten-Mapping</h3>
        <div className="rounded-lg border border-border overflow-hidden">
          <div className="grid grid-cols-[1fr_120px_80px] gap-3 px-4 py-2 bg-secondary/50 text-xs font-medium text-muted-foreground">
            <span>Cosmi Konto</span>
            <span>DATEV Konto</span>
            <span className="text-right">Aktion</span>
          </div>
          {mappings.map((m) => (
            <div key={m.id} className="grid grid-cols-[1fr_120px_80px] gap-3 items-center px-4 py-2.5 border-t border-border">
              <span className="text-sm text-foreground">{m.kmuLabel}</span>
              {editingId === m.id ? (
                <input
                  type="text"
                  value={editKonto}
                  onChange={(e) => setEditKonto(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && saveEdit()}
                  onBlur={saveEdit}
                  autoFocus
                  className="rounded border border-primary bg-card px-2 py-1 text-sm font-mono text-foreground"
                />
              ) : (
                <span className="text-sm font-mono text-foreground">{m.datevKonto || '—'}</span>
              )}
              <div className="flex justify-end gap-1">
                <button
                  onClick={() => startEdit(m)}
                  className="rounded p-1 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                >
                  <Pencil className="h-3.5 w-3.5" />
                </button>
                <button
                  onClick={() => removeMapping(m.id)}
                  className="rounded p-1 text-muted-foreground hover:bg-error-light hover:text-destructive transition-colors"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          ))}
        </div>
        <button
          onClick={addMapping}
          className="flex items-center gap-1.5 mt-2 text-xs text-primary hover:text-primary/80 transition-colors"
        >
          <Plus className="h-3.5 w-3.5" />
          Zuordnung hinzufügen
        </button>
      </div>

      {/* Auswertungen */}
      <div className="mb-6">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Auswertungen</h3>
        <div className="space-y-2">
          {[
            { label: 'BWA (Betriebswirtschaftliche Auswertung)', value: bwaEnabled, set: setBwaEnabled },
            { label: 'SuSa (Summen- und Saldenliste)', value: susaEnabled, set: setSusaEnabled },
            { label: 'Kontenblaetter', value: kontenblaetter, set: setKontenblaetter },
          ].map((item) => (
            <label key={item.label} className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={item.value}
                onChange={(e) => item.set(e.target.checked)}
                className="accent-primary h-4 w-4 rounded"
              />
              <span className="text-sm text-foreground">{item.label}</span>
            </label>
          ))}
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2 border-t border-border pt-4 mt-6">
        {status === 'connected' ? (
          <>
            <button
              onClick={handleDisconnect}
              className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-muted transition-colors"
            >
              <Unlink className="h-4 w-4" />
              Trennen
            </button>
            <button
              onClick={handleSync}
              className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-muted transition-colors"
            >
              <RefreshCw className={`h-4 w-4 ${status === 'syncing' ? 'animate-spin' : ''}`} />
              Synchronisieren
            </button>
          </>
        ) : (
          <button
            onClick={handleConnect}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plug className="h-4 w-4" />
            Verbinden
          </button>
        )}
        <button
          onClick={handleSave}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          Speichern
        </button>
        <button
          onClick={handleTest}
          disabled={testing}
          className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-muted transition-colors disabled:opacity-50 ml-auto"
        >
          {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />}
          Verbindung testen
        </button>
      </div>
    </div>
  )
}
