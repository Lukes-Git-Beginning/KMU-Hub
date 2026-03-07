/**
 * BankingWidget — FinAPI bank connection placeholder + transaction matching.
 *
 * Shows bank account connection card, recent transactions, and
 * automatic invoice-to-payment matching UI.
 * Mock data for design — backend: FinAPI integration.
 */
import { useState } from 'react'
import {
  Landmark,
  Link2,
  RefreshCw,
  Check,
  X,
  ArrowUpRight,
  ArrowDownRight,
  Search,
  CheckCircle2,
  HelpCircle,
  Zap,
} from 'lucide-react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface BankAccount {
  id: string
  bankName: string
  iban: string
  bic: string
  balance: number
  currency: string
  connected: boolean
  lastSync: string | null
}

type MatchStatus = 'matched' | 'suggested' | 'unmatched'

interface BankTransaction {
  id: string
  date: string
  description: string
  amount: number
  type: 'credit' | 'debit'
  counterpart: string
  matchStatus: MatchStatus
  matchedInvoice?: string
}

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

// TODO: Replace mock data with API call — Backend needed: FinAPI integration endpoints
// GET /api/v1/finance/bank-accounts, GET /api/v1/finance/bank-transactions
const mockAccounts: BankAccount[] = [
  {
    id: 'ba1',
    bankName: 'Commerzbank',
    iban: 'DE89 3704 0044 0532 0130 00',
    bic: 'COBADEFFXXX',
    balance: 142850.75,
    currency: 'EUR',
    connected: true,
    lastSync: '2026-02-24 09:15',
  },
  {
    id: 'ba2',
    bankName: 'Sparkasse Muenchen',
    iban: 'DE72 7015 0000 0012 3456 78',
    bic: 'SSKMDEMMXXX',
    balance: 85000.00,
    currency: 'EUR',
    connected: false,
    lastSync: null,
  },
]

// TODO: Replace mock data with API call — Backend needed: FinAPI integration endpoints
// GET /api/v1/finance/bank-transactions (with filtering, pagination, match status)
const mockTransactions: BankTransaction[] = [
  { id: 'bt1', date: '2026-02-24', description: 'Eingang Gruber Maschinenbau — Abschlag 3', amount: 16000.00, type: 'credit', counterpart: 'Gruber Maschinenbau GmbH', matchStatus: 'matched', matchedInvoice: 'RE-2026-003' },
  { id: 'bt2', date: '2026-02-22', description: 'Eingang DataFlow — Analytics Dashboard', amount: 14000.00, type: 'credit', counterpart: 'DataFlow GmbH', matchStatus: 'matched', matchedInvoice: 'RE-2026-011' },
  { id: 'bt3', date: '2026-02-19', description: 'Eingang Stadler Bau — Intranet Portal', amount: 21000.00, type: 'credit', counterpart: 'Stadler Bauunternehmen GmbH', matchStatus: 'suggested', matchedInvoice: 'RE-2026-008' },
  { id: 'bt4', date: '2026-02-23', description: 'CloudFirst Hosting — Monatsrechnung Feb', amount: -1890.00, type: 'debit', counterpart: 'CloudFirst Hosting GmbH', matchStatus: 'unmatched' },
  { id: 'bt5', date: '2026-02-21', description: 'Gehaelter Februar 2026', amount: -78500.00, type: 'debit', counterpart: 'Sammelueberweisung', matchStatus: 'unmatched' },
  { id: 'bt6', date: '2026-02-25', description: 'Eingang Berger — Mobile App Anzahlung', amount: 20000.00, type: 'credit', counterpart: 'Berger & Soehne', matchStatus: 'suggested', matchedInvoice: 'RE-2026-015' },
  { id: 'bt7', date: '2026-02-18', description: 'Adobe Creative Cloud — Jahresrechnung', amount: -4188.00, type: 'debit', counterpart: 'Adobe Inc.', matchStatus: 'unmatched' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function BankingWidget() {
  const [syncing, setSyncing] = useState(false)
  const [filter, setFilter] = useState<'all' | 'matched' | 'suggested' | 'unmatched'>('all')

  const connectedAccount = mockAccounts.find((a) => a.connected)

  const handleSync = () => {
    setSyncing(true)
    setTimeout(() => {
      setSyncing(false)
      toast.success('Kontostand aktualisiert')
    }, 2000)
  }

  const handleAcceptMatch = (_txId: string) => {
    toast.success('Zuordnung bestaetigt')
  }

  const handleRejectMatch = (_txId: string) => {
    toast.success('Zuordnung abgelehnt')
  }

  const filteredTx = mockTransactions.filter((tx) => {
    if (filter === 'all') return true
    return tx.matchStatus === filter
  })

  const matchedCount = mockTransactions.filter((t) => t.matchStatus === 'matched').length
  const suggestedCount = mockTransactions.filter((t) => t.matchStatus === 'suggested').length
  const unmatchedCount = mockTransactions.filter((t) => t.matchStatus === 'unmatched').length

  const formatEUR = (v: number) =>
    new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(v)

  return (
    <div className="space-y-4">
      {/* Bank accounts */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {mockAccounts.map((acc) => (
          <div
            key={acc.id}
            className={`rounded-lg border p-4 ${
              acc.connected ? 'border-success/30 bg-success-light/10' : 'border-border'
            }`}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${
                  acc.connected ? 'bg-success-light' : 'bg-secondary'
                }`}>
                  <Landmark className={`h-5 w-5 ${acc.connected ? 'text-success' : 'text-muted-foreground'}`} />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">{acc.bankName}</p>
                  <p className="text-[10px] text-muted-foreground font-mono">{acc.iban}</p>
                </div>
              </div>
              {acc.connected ? (
                <CheckCircle2 className="h-4 w-4 text-success" />
              ) : (
                <button className="flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors">
                  <Link2 className="h-3 w-3" />
                  Verbinden
                </button>
              )}
            </div>

            {acc.connected && (
              <div className="flex items-end justify-between">
                <div>
                  <p className="text-[10px] text-muted-foreground">Kontostand</p>
                  <p className="text-lg font-semibold text-foreground">{formatEUR(acc.balance)}</p>
                </div>
                <div className="flex items-center gap-2">
                  {acc.lastSync && (
                    <span className="text-[9px] text-muted-foreground">
                      Letzte Sync: {acc.lastSync}
                    </span>
                  )}
                  <button
                    onClick={handleSync}
                    disabled={syncing}
                    className="rounded-md border border-border p-1.5 text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
                  >
                    <RefreshCw className={`h-3.5 w-3.5 ${syncing ? 'animate-spin' : ''}`} />
                  </button>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* FinAPI info */}
      <div className="flex items-start gap-2 rounded-md bg-secondary/50 px-3 py-2">
        <Zap className="h-3.5 w-3.5 text-muted-foreground mt-0.5 shrink-0" />
        <p className="text-[11px] text-muted-foreground">
          Bankverbindung via <strong>FinAPI</strong> — Automatischer Kontoabgleich und Zahlungs-Matching.
          PSD2-konform, verschluesselte Verbindung. Keine Kontodaten auf KMU Hub Servern gespeichert.
        </p>
      </div>

      {/* Transaction matching */}
      {connectedAccount && (
        <>
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium text-foreground">Transaktions-Matching</h3>
            <div className="flex items-center gap-1.5">
              {([
                ['all', `Alle (${mockTransactions.length})`],
                ['matched', `Zugeordnet (${matchedCount})`],
                ['suggested', `Vorschlaege (${suggestedCount})`],
                ['unmatched', `Offen (${unmatchedCount})`],
              ] as const).map(([key, label]) => (
                <button
                  key={key}
                  onClick={() => setFilter(key)}
                  className={`rounded-md px-2 py-1 text-[10px] transition-colors ${
                    filter === key
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-secondary text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>

          <div className="rounded-lg border border-border overflow-hidden">
            <div className="grid grid-cols-[80px_1fr_120px_90px_100px_70px] gap-2 items-center bg-secondary/30 px-3 py-2 text-[10px] font-medium text-muted-foreground">
              <span>Datum</span>
              <span>Beschreibung</span>
              <span>Gegenpartei</span>
              <span className="text-right">Betrag</span>
              <span>Zuordnung</span>
              <span />
            </div>
            {filteredTx.map((tx) => (
              <div
                key={tx.id}
                className={`grid grid-cols-[80px_1fr_120px_90px_100px_70px] gap-2 items-center px-3 py-2.5 border-t border-border transition-colors hover:bg-accent/30 ${
                  tx.matchStatus === 'suggested' ? 'bg-warning-light/10' : ''
                }`}
              >
                <span className="text-xs text-muted-foreground">
                  {new Date(tx.date).toLocaleDateString('de-DE')}
                </span>
                <span className="text-xs text-foreground truncate">{tx.description}</span>
                <span className="text-xs text-muted-foreground truncate">{tx.counterpart}</span>
                <span
                  className={`text-xs font-medium text-right flex items-center justify-end gap-1 ${
                    tx.type === 'credit' ? 'text-success' : 'text-foreground'
                  }`}
                >
                  {tx.type === 'credit' ? (
                    <ArrowDownRight className="h-3 w-3" />
                  ) : (
                    <ArrowUpRight className="h-3 w-3 text-muted-foreground" />
                  )}
                  {formatEUR(Math.abs(tx.amount))}
                </span>
                <div>
                  {tx.matchStatus === 'matched' && (
                    <span className="flex items-center gap-1 text-[10px] text-success font-medium">
                      <CheckCircle2 className="h-3 w-3" />
                      {tx.matchedInvoice}
                    </span>
                  )}
                  {tx.matchStatus === 'suggested' && (
                    <span className="flex items-center gap-1 text-[10px] text-warning font-medium">
                      <HelpCircle className="h-3 w-3" />
                      {tx.matchedInvoice}?
                    </span>
                  )}
                  {tx.matchStatus === 'unmatched' && (
                    <span className="text-[10px] text-muted-foreground">—</span>
                  )}
                </div>
                <div className="flex items-center gap-1">
                  {tx.matchStatus === 'suggested' && (
                    <>
                      <button
                        onClick={() => handleAcceptMatch(tx.id)}
                        className="rounded p-1 text-success hover:bg-success-light transition-colors"
                        title="Zuordnung bestaetigen"
                      >
                        <Check className="h-3 w-3" />
                      </button>
                      <button
                        onClick={() => handleRejectMatch(tx.id)}
                        className="rounded p-1 text-muted-foreground hover:bg-accent transition-colors"
                        title="Zuordnung ablehnen"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </>
                  )}
                  {tx.matchStatus === 'unmatched' && tx.type === 'credit' && (
                    <button
                      onClick={() => toast.success('Manuelle Zuordnung geoeffnet')}
                      className="rounded p-1 text-primary hover:bg-primary/10 transition-colors"
                      title="Manuell zuordnen"
                    >
                      <Search className="h-3 w-3" />
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
