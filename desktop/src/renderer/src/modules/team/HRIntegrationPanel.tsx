import { useState } from 'react'
import {
  Link2,
  Link2Off,
  RefreshCw,
  Check,
  AlertTriangle,
  ExternalLink,
  Settings,
  ChevronDown,
  ChevronUp,
  Globe,
  Banknote,
  Building2,
  FileText,
  Loader2,
} from 'lucide-react'
import { toast } from 'sonner'

// ============================================================
// Types
// ============================================================

type ConnectionStatus = 'connected' | 'disconnected' | 'syncing' | 'error'
type Country = 'DE' | 'CH' | 'AT'

interface IntegrationItem {
  id: string
  name: string
  description: string
  status: ConnectionStatus
  lastSync?: string
  syncCount?: number
  icon: typeof Building2
  category: 'payroll' | 'hr' | 'other'
}

interface CountryDeduction {
  label: string
  rate: string
  amount: number
  type: 'employee' | 'employer' | 'both'
}

// ============================================================
// Mock Data
// ============================================================

const INTEGRATIONS: IntegrationItem[] = [
  {
    id: 'datev-lohn',
    name: 'DATEV Lohn & Gehalt',
    description: 'Automatischer Lohndaten-Export an DATEV Unternehmen Online',
    status: 'connected',
    lastSync: '2026-02-21T08:30:00',
    syncCount: 247,
    icon: Building2,
    category: 'payroll',
  },
  {
    id: 'abacus-hr',
    name: 'Abacus HR',
    description: 'Personalstammdaten-Synchronisation mit Abacus',
    status: 'disconnected',
    icon: Building2,
    category: 'hr',
  },
  {
    id: 'sage-hr',
    name: 'Sage HR',
    description: 'Personalmanagement und Abrechnung mit Sage',
    status: 'disconnected',
    icon: Building2,
    category: 'hr',
  },
  {
    id: 'personio',
    name: 'Personio',
    description: 'All-in-One HR-Software fuer KMUs',
    status: 'disconnected',
    icon: Building2,
    category: 'hr',
  },
]

const COUNTRY_DEDUCTIONS: Record<Country, { label: string; currency: string; grossExample: number; deductions: CountryDeduction[] }> = {
  DE: {
    label: 'Deutschland',
    currency: 'EUR',
    grossExample: 4200,
    deductions: [
      { label: 'Lohnsteuer', rate: '~18,5%', amount: 777, type: 'employee' },
      { label: 'Solidaritaetszuschlag', rate: '5,5% d. LSt', amount: 42, type: 'employee' },
      { label: 'Kirchensteuer', rate: '8% d. LSt', amount: 62, type: 'employee' },
      { label: 'Rentenversicherung', rate: '9,3%', amount: 391, type: 'both' },
      { label: 'Krankenversicherung', rate: '7,3% + Zusatz', amount: 340, type: 'both' },
      { label: 'Pflegeversicherung', rate: '1,7%', amount: 71, type: 'both' },
      { label: 'Arbeitslosenversicherung', rate: '1,3%', amount: 55, type: 'both' },
    ],
  },
  CH: {
    label: 'Schweiz',
    currency: 'CHF',
    grossExample: 6500,
    deductions: [
      { label: 'AHV / IV / EO', rate: '5,3%', amount: 345, type: 'both' },
      { label: 'BVG (Pensionskasse)', rate: '~7%', amount: 455, type: 'both' },
      { label: 'Quellensteuer', rate: 'Tarif A', amount: 520, type: 'employee' },
      { label: 'NBU (Nichtberufsunfall)', rate: '~1,4%', amount: 91, type: 'employee' },
      { label: 'ALV', rate: '1,1%', amount: 72, type: 'both' },
    ],
  },
  AT: {
    label: 'Oesterreich',
    currency: 'EUR',
    grossExample: 3800,
    deductions: [
      { label: 'Lohnsteuer', rate: '~20%', amount: 760, type: 'employee' },
      { label: 'Sozialversicherung (SV)', rate: '18,12%', amount: 689, type: 'employee' },
      { label: 'Krankenversicherung', rate: '3,87%', amount: 147, type: 'both' },
      { label: 'Pensionsversicherung', rate: '10,25%', amount: 390, type: 'both' },
      { label: 'Arbeitslosenversicherung', rate: '3%', amount: 114, type: 'both' },
    ],
  },
}

const statusConfig: Record<ConnectionStatus, { label: string; color: string; bg: string; icon: typeof Check }> = {
  connected: { label: 'Verbunden', color: 'text-success', bg: 'bg-success-light', icon: Check },
  disconnected: { label: 'Nicht verbunden', color: 'text-muted-foreground', bg: 'bg-secondary', icon: Link2Off },
  syncing: { label: 'Synchronisiert...', color: 'text-info', bg: 'bg-info-light', icon: RefreshCw },
  error: { label: 'Fehler', color: 'text-error', bg: 'bg-error-light', icon: AlertTriangle },
}

const formatCurrency = (amount: number, currency: string) =>
  new Intl.NumberFormat('de-DE', { style: 'currency', currency, minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(amount)

// ============================================================
// Component
// ============================================================

export function HRIntegrationPanel() {
  const [expandedId, setExpandedId] = useState<string | null>('datev-lohn')
  const [previewCountry, setPreviewCountry] = useState<Country>('DE')
  const [syncing, setSyncing] = useState<string | null>(null)

  const handleSync = (id: string) => {
    setSyncing(id)
    setTimeout(() => {
      setSyncing(null)
      toast.success('Synchronisation abgeschlossen', { description: `${INTEGRATIONS.find(i => i.id === id)?.name}` })
    }, 1500)
  }

  const handleConnect = (name: string) => {
    toast.info(`${name} Verbindung wird hergestellt...`, { description: 'OAuth-Flow wird gestartet (Mock)' })
  }

  const countryData = COUNTRY_DEDUCTIONS[previewCountry]
  const totalDeductions = countryData.deductions.reduce((s, d) => s + d.amount, 0)
  const netSalary = countryData.grossExample - totalDeductions

  return (
    <div className="space-y-6">
      {/* Integration Cards */}
      <div>
        <h3 className="text-sm font-medium text-foreground mb-3">HR-Integrationen</h3>
        <div className="space-y-3">
          {INTEGRATIONS.map((integration) => {
            const status = syncing === integration.id ? statusConfig.syncing : statusConfig[integration.status]
            const StatusIcon = syncing === integration.id ? Loader2 : status.icon
            const IntIcon = integration.icon
            const isExpanded = expandedId === integration.id

            return (
              <div key={integration.id} className="rounded-lg border border-border bg-card overflow-hidden">
                <div
                  className="flex items-center gap-4 p-4 cursor-pointer hover:bg-secondary/30 transition-colors"
                  onClick={() => setExpandedId(isExpanded ? null : integration.id)}
                >
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light flex-shrink-0">
                    <IntIcon className="h-5 w-5 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <h4 className="text-sm font-medium text-foreground">{integration.name}</h4>
                      <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${status.bg} ${status.color}`}>
                        <StatusIcon className={`h-3 w-3 ${syncing === integration.id ? 'animate-spin' : ''}`} />
                        {status.label}
                      </span>
                    </div>
                    <p className="text-xs text-muted-foreground mt-0.5">{integration.description}</p>
                  </div>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    {integration.status === 'connected' && (
                      <button
                        onClick={(e) => { e.stopPropagation(); handleSync(integration.id) }}
                        disabled={!!syncing}
                        className="flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors disabled:opacity-50"
                      >
                        <RefreshCw className={`h-3 w-3 ${syncing === integration.id ? 'animate-spin' : ''}`} />
                        Sync
                      </button>
                    )}
                    {integration.status === 'disconnected' && (
                      <button
                        onClick={(e) => { e.stopPropagation(); handleConnect(integration.name) }}
                        className="flex items-center gap-1.5 rounded-md bg-primary px-2.5 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
                      >
                        <Link2 className="h-3 w-3" />
                        Verbinden
                      </button>
                    )}
                    {isExpanded ? <ChevronUp className="h-4 w-4 text-muted-foreground" /> : <ChevronDown className="h-4 w-4 text-muted-foreground" />}
                  </div>
                </div>

                {isExpanded && integration.status === 'connected' && (
                  <div className="border-t border-border px-4 py-3 bg-secondary/20">
                    <div className="grid grid-cols-3 gap-4">
                      <div>
                        <p className="text-[11px] text-muted-foreground">Letzte Synchronisation</p>
                        <p className="text-sm font-medium text-foreground">
                          {integration.lastSync ? new Date(integration.lastSync).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' }) : '-'}
                        </p>
                      </div>
                      <div>
                        <p className="text-[11px] text-muted-foreground">Synchronisierte Datensaetze</p>
                        <p className="text-sm font-medium text-foreground">{integration.syncCount ?? 0}</p>
                      </div>
                      <div>
                        <p className="text-[11px] text-muted-foreground">Aktionen</p>
                        <div className="flex items-center gap-2 mt-0.5">
                          <button className="text-xs text-primary hover:underline flex items-center gap-1">
                            <Settings className="h-3 w-3" />
                            Einstellungen
                          </button>
                          <button className="text-xs text-primary hover:underline flex items-center gap-1">
                            <ExternalLink className="h-3 w-3" />
                            Dashboard
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {isExpanded && integration.status === 'disconnected' && (
                  <div className="border-t border-border px-4 py-3 bg-secondary/20">
                    <p className="text-xs text-muted-foreground">
                      Verbinden Sie {integration.name}, um Personalstammdaten und Lohndaten automatisch zu synchronisieren.
                      Nach der Verbindung werden Aenderungen in Echtzeit uebertragen.
                    </p>
                  </div>
                )}
              </div>
            )
          })}
        </div>

        <div className="mt-3 rounded-lg border border-dashed border-border bg-card/50 p-4 text-center">
          <p className="text-sm text-muted-foreground">
            Weitere HR-Systeme (Lexware, Agenda, BMD, etc.) in Vorbereitung
          </p>
          <button className="mt-2 text-xs text-primary hover:underline">
            Integration anfragen
          </button>
        </div>
      </div>

      {/* Multi-Country Deduction Preview (7.2) */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-foreground flex items-center gap-2">
            <Globe className="h-4 w-4 text-muted-foreground" />
            Lohn-Vorschau nach Land
          </h3>
          <div className="flex items-center gap-1 rounded-lg border border-border bg-card p-0.5">
            {(['DE', 'CH', 'AT'] as Country[]).map((c) => (
              <button
                key={c}
                onClick={() => setPreviewCountry(c)}
                className={`rounded-md px-3 py-1 text-xs font-medium transition-colors ${
                  previewCountry === c ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {c}
              </button>
            ))}
          </div>
        </div>

        <div className="rounded-lg border border-border bg-card overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-secondary/30">
            <div className="flex items-center gap-2">
              <Banknote className="h-4 w-4 text-primary" />
              <span className="text-sm font-medium text-foreground">{countryData.label}</span>
            </div>
            <span className="text-xs text-muted-foreground">
              Beispiel: {formatCurrency(countryData.grossExample, countryData.currency)} brutto
            </span>
          </div>

          <div className="divide-y divide-border-muted">
            {countryData.deductions.map((ded, i) => (
              <div key={i} className="flex items-center justify-between px-4 py-2.5">
                <div className="flex items-center gap-2">
                  <span className="text-sm text-foreground">{ded.label}</span>
                  <span className={`rounded-full px-1.5 py-0.5 text-[9px] font-medium ${
                    ded.type === 'employee' ? 'bg-warning-light text-warning' : ded.type === 'employer' ? 'bg-info-light text-info' : 'bg-secondary text-muted-foreground'
                  }`}>
                    {ded.type === 'employee' ? 'AN' : ded.type === 'employer' ? 'AG' : 'AN+AG'}
                  </span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-xs text-muted-foreground tabular-nums">{ded.rate}</span>
                  <span className="text-sm font-medium text-error tabular-nums w-20 text-right">
                    -{formatCurrency(ded.amount, countryData.currency)}
                  </span>
                </div>
              </div>
            ))}
          </div>

          <div className="border-t border-border px-4 py-3 bg-secondary/20">
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm text-muted-foreground">Bruttolohn</span>
              <span className="text-sm font-medium text-foreground tabular-nums">{formatCurrency(countryData.grossExample, countryData.currency)}</span>
            </div>
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm text-muted-foreground">Abzuege gesamt</span>
              <span className="text-sm font-medium text-error tabular-nums">-{formatCurrency(totalDeductions, countryData.currency)}</span>
            </div>
            <div className="flex items-center justify-between border-t border-border pt-2 mt-2">
              <span className="text-sm font-semibold text-foreground">Nettolohn (ca.)</span>
              <span className="text-base font-bold text-foreground tabular-nums">{formatCurrency(netSalary, countryData.currency)}</span>
            </div>
          </div>
        </div>

        <p className="mt-2 text-[11px] text-muted-foreground flex items-center gap-1">
          <FileText className="h-3 w-3" />
          Werte sind Richtwerte. Exakte Berechnung erfolgt ueber die verbundene Lohnbuchhaltung.
        </p>
      </div>
    </div>
  )
}
