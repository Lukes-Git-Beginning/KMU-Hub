/**
 * BelegketteTab — Document chain visualization (Angebot → Rechnung → Zahlung → Gutschrift).
 *
 * Shows the lifecycle of a financial document from quote through to payment/credit note.
 * Each chain is a horizontal pipeline with status indicators.
 * Mock data for design — backend swap: replace with API hooks.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  FileText,
  ArrowRight,
  CheckCircle2,
  Clock,
  AlertCircle,
  CreditCard,
  Receipt,
  Search,
  ChevronDown,
  ChevronUp,
  ExternalLink,
  Filter,
} from 'lucide-react'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type DocType = 'quote' | 'invoice' | 'payment' | 'credit-note' | 'dunning'
type DocStatus = 'completed' | 'active' | 'pending' | 'cancelled' | 'overdue'

interface ChainNode {
  type: DocType
  number: string
  date: string
  amount: string
  status: DocStatus
}

interface DocumentChain {
  id: string
  customer: string
  totalValue: string
  nodes: ChainNode[]
  isComplete: boolean
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

const docTypeConfig: Record<DocType, { labelKey: string; icon: typeof FileText; color: string }> = {
  quote: { labelKey: 'finanzen.docChain.docType.quote', icon: FileText, color: 'text-info' },
  invoice: { labelKey: 'finanzen.docChain.docType.invoice', icon: FileText, color: 'text-primary' },
  payment: { labelKey: 'finanzen.docChain.docType.payment', icon: CreditCard, color: 'text-success' },
  'credit-note': { labelKey: 'finanzen.docChain.docType.creditNote', icon: Receipt, color: 'text-warning' },
  dunning: { labelKey: 'finanzen.docChain.docType.dunning', icon: AlertCircle, color: 'text-error' },
}

const statusConfig: Record<DocStatus, { labelKey: string; bg: string; text: string }> = {
  completed: { labelKey: 'finanzen.docChain.status.completed', bg: 'bg-success-light', text: 'text-success' },
  active: { labelKey: 'finanzen.docChain.status.active', bg: 'bg-info-light', text: 'text-info' },
  pending: { labelKey: 'finanzen.docChain.status.pending', bg: 'bg-secondary', text: 'text-muted-foreground' },
  cancelled: { labelKey: 'finanzen.docChain.status.cancelled', bg: 'bg-secondary', text: 'text-muted-foreground' },
  overdue: { labelKey: 'finanzen.docChain.status.overdue', bg: 'bg-error-light', text: 'text-error' },
}

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

// TODO: Replace mock data with API call — Backend needed: GET /api/v1/finance/document-chains
// This aggregation endpoint needs to link invoices, payments, and receipts into chains.
const mockChains: DocumentChain[] = [
  {
    id: 'chain-1',
    customer: 'Muster AG',
    totalValue: 'CHF 12.450,00',
    isComplete: true,
    nodes: [
      { type: 'quote', number: 'AN-2026-001', date: '2026-01-10', amount: 'CHF 12.450,00', status: 'completed' },
      { type: 'invoice', number: 'RE-2026-003', date: '2026-01-25', amount: 'CHF 12.450,00', status: 'completed' },
      { type: 'payment', number: 'ZA-2026-007', date: '2026-02-10', amount: 'CHF 12.450,00', status: 'completed' },
    ],
  },
  {
    id: 'chain-2',
    customer: 'Digital Solutions GmbH',
    totalValue: 'EUR 8.900,00',
    isComplete: false,
    nodes: [
      { type: 'quote', number: 'AN-2026-004', date: '2026-01-20', amount: 'EUR 8.900,00', status: 'completed' },
      { type: 'invoice', number: 'RE-2026-008', date: '2026-02-01', amount: 'EUR 8.900,00', status: 'active' },
      { type: 'dunning', number: 'MA-2026-001', date: '2026-02-18', amount: 'EUR 8.900,00', status: 'active' },
      { type: 'payment', number: '—', date: '—', amount: 'EUR 8.900,00', status: 'pending' },
    ],
  },
  {
    id: 'chain-3',
    customer: 'Weber & Partner',
    totalValue: 'CHF 5.200,00',
    isComplete: false,
    nodes: [
      { type: 'invoice', number: 'RE-2026-012', date: '2026-02-05', amount: 'CHF 5.200,00', status: 'overdue' },
      { type: 'payment', number: '—', date: '—', amount: 'CHF 5.200,00', status: 'pending' },
    ],
  },
  {
    id: 'chain-4',
    customer: 'TechStart Zürich',
    totalValue: 'CHF 3.750,00',
    isComplete: true,
    nodes: [
      { type: 'invoice', number: 'RE-2026-006', date: '2026-01-15', amount: 'CHF 3.750,00', status: 'completed' },
      { type: 'payment', number: 'ZA-2026-012', date: '2026-01-28', amount: 'CHF 2.750,00', status: 'completed' },
      { type: 'credit-note', number: 'GU-2026-001', date: '2026-02-01', amount: 'CHF 1.000,00', status: 'completed' },
    ],
  },
  {
    id: 'chain-5',
    customer: 'Alpen Logistik AG',
    totalValue: 'EUR 24.800,00',
    isComplete: false,
    nodes: [
      { type: 'quote', number: 'AN-2026-009', date: '2026-02-10', amount: 'EUR 24.800,00', status: 'completed' },
      { type: 'invoice', number: 'RE-2026-015', date: '2026-02-14', amount: 'EUR 24.800,00', status: 'active' },
      { type: 'payment', number: '—', date: '—', amount: 'EUR 24.800,00', status: 'pending' },
    ],
  },
  {
    id: 'chain-6',
    customer: 'Innovate Labs',
    totalValue: 'CHF 1.950,00',
    isComplete: true,
    nodes: [
      { type: 'quote', number: 'AN-2026-002', date: '2026-01-05', amount: 'CHF 1.950,00', status: 'cancelled' },
    ],
  },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

type FilterOption = 'all' | 'open' | 'complete' | 'overdue'

export function BelegketteTab() {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<FilterOption>('all')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const filtered = mockChains.filter((chain) => {
    if (search) {
      const q = search.toLowerCase()
      if (
        !chain.customer.toLowerCase().includes(q) &&
        !chain.nodes.some((n) => n.number.toLowerCase().includes(q))
      ) {
        return false
      }
    }
    if (filter === 'complete') return chain.isComplete
    if (filter === 'open') return !chain.isComplete
    if (filter === 'overdue') return chain.nodes.some((n) => n.status === 'overdue')
    return true
  })

  const stats = {
    total: mockChains.length,
    open: mockChains.filter((c) => !c.isComplete).length,
    complete: mockChains.filter((c) => c.isComplete).length,
    overdue: mockChains.filter((c) => c.nodes.some((n) => n.status === 'overdue')).length,
  }

  return (
    <div className="space-y-4">
      {/* Stats */}
      <div className="grid grid-cols-4 gap-3">
        {([
          ['total', t('finanzen.docChain.statsTotal'), stats.total, 'bg-secondary text-foreground'],
          ['open', t('finanzen.docChain.statsOpen'), stats.open, 'bg-info-light text-info'],
          ['complete', t('finanzen.docChain.statsComplete'), stats.complete, 'bg-success-light text-success'],
          ['overdue', t('finanzen.docChain.statsOverdue'), stats.overdue, 'bg-error-light text-error'],
        ] as const).map(([key, label, count, _colors]) => (
          <button
            key={key}
            onClick={() => setFilter(key === filter ? 'all' : key)}
            className={`rounded-lg border p-3 text-left transition-colors ${
              filter === key ? 'border-primary ring-1 ring-primary/20' : 'border-border hover:bg-accent/50'
            }`}
          >
            <p className="text-2xl font-semibold text-foreground">{count}</p>
            <p className="text-xs text-muted-foreground">{label}</p>
          </button>
        ))}
      </div>

      {/* Search + filter */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder={t('finanzen.docChain.searchPlaceholder')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        </div>
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <Filter className="h-3.5 w-3.5" />
          {filtered.length} von {mockChains.length}
        </div>
      </div>

      {/* Chain list */}
      <div className="space-y-3">
        {filtered.length === 0 ? (
          <div className="py-12 text-center">
            <Search className="mx-auto h-10 w-10 text-muted-foreground/30" />
            <p className="mt-3 text-sm font-medium text-foreground">{t('finanzen.docChain.noResults')}</p>
            <p className="mt-1 text-xs text-muted-foreground">{t('finanzen.docChain.adjustFilters')}</p>
          </div>
        ) : (
          filtered.map((chain) => {
            const isExpanded = expandedId === chain.id
            return (
              <div
                key={chain.id}
                className={`rounded-lg border transition-colors ${
                  chain.nodes.some((n) => n.status === 'overdue')
                    ? 'border-error/30'
                    : chain.isComplete
                      ? 'border-success/30'
                      : 'border-border'
                }`}
              >
                {/* Chain header */}
                <button
                  onClick={() => setExpandedId(isExpanded ? null : chain.id)}
                  className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-accent/30 transition-colors"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-foreground truncate">
                        {chain.customer}
                      </span>
                      {chain.isComplete ? (
                        <span className="flex items-center gap-1 rounded-full bg-success-light px-2 py-0.5 text-[10px] font-medium text-success">
                          <CheckCircle2 className="h-3 w-3" />
                          {t('finanzen.docChain.status.completed')}
                        </span>
                      ) : chain.nodes.some((n) => n.status === 'overdue') ? (
                        <span className="flex items-center gap-1 rounded-full bg-error-light px-2 py-0.5 text-[10px] font-medium text-error">
                          <AlertCircle className="h-3 w-3" />
                          {t('finanzen.docChain.status.overdue')}
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 rounded-full bg-info-light px-2 py-0.5 text-[10px] font-medium text-info">
                          <Clock className="h-3 w-3" />
                          {t('finanzen.docChain.statsOpen')}
                        </span>
                      )}
                    </div>
                    <span className="text-xs text-muted-foreground">{chain.totalValue}</span>
                  </div>

                  {/* Mini pipeline */}
                  <div className="hidden sm:flex items-center gap-1">
                    {chain.nodes.map((node, i) => {
                      const cfg = docTypeConfig[node.type]
                      const Icon = cfg.icon
                      const sCfg = statusConfig[node.status]
                      return (
                        <div key={i} className="flex items-center gap-1">
                          {i > 0 && <ArrowRight className="h-3 w-3 text-border" />}
                          <div
                            className={`flex h-7 w-7 items-center justify-center rounded-full ${sCfg.bg}`}
                            title={`${t(cfg.labelKey)}: ${node.number}`}
                          >
                            <Icon className={`h-3.5 w-3.5 ${sCfg.text}`} />
                          </div>
                        </div>
                      )
                    })}
                  </div>

                  {isExpanded ? (
                    <ChevronUp className="h-4 w-4 text-muted-foreground shrink-0" />
                  ) : (
                    <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
                  )}
                </button>

                {/* Expanded detail */}
                {isExpanded && (
                  <div className="border-t border-border px-4 py-3 space-y-3">
                    {/* Pipeline visualization */}
                    <div className="flex items-start gap-0 overflow-x-auto pb-2">
                      {chain.nodes.map((node, i) => {
                        const cfg = docTypeConfig[node.type]
                        const Icon = cfg.icon
                        const sCfg = statusConfig[node.status]
                        return (
                          <div key={i} className="flex items-start">
                            {i > 0 && (
                              <div className="flex items-center pt-5 px-1">
                                <div className={`h-0.5 w-8 ${
                                  node.status === 'pending' ? 'bg-border border-dashed' : 'bg-primary/30'
                                }`} />
                                <ArrowRight className={`h-4 w-4 -ml-1.5 ${
                                  node.status === 'pending' ? 'text-border' : 'text-primary/30'
                                }`} />
                              </div>
                            )}
                            <div className={`flex flex-col items-center gap-1.5 rounded-lg border p-3 min-w-[130px] ${
                              node.status === 'active'
                                ? 'border-primary/40 bg-primary/5'
                                : node.status === 'overdue'
                                  ? 'border-error/40 bg-error-light/20'
                                  : node.status === 'pending'
                                    ? 'border-dashed border-border bg-secondary/30'
                                    : 'border-border'
                            }`}>
                              <div className={`flex h-10 w-10 items-center justify-center rounded-full ${sCfg.bg}`}>
                                <Icon className={`h-5 w-5 ${sCfg.text}`} />
                              </div>
                              <span className={`text-xs font-medium ${cfg.color}`}>{t(cfg.labelKey)}</span>
                              <span className="text-[11px] font-mono text-foreground">{node.number}</span>
                              <span className="text-[10px] text-muted-foreground">{node.date}</span>
                              <span className="text-[10px] font-medium text-foreground">{node.amount}</span>
                              <span className={`rounded-full px-2 py-0.5 text-[9px] font-medium ${sCfg.bg} ${sCfg.text}`}>
                                {t(sCfg.labelKey)}
                              </span>
                            </div>
                          </div>
                        )
                      })}
                    </div>

                    {/* Actions */}
                    <div className="flex items-center justify-end gap-2 pt-1">
                      <button className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-accent transition-colors">
                        <ExternalLink className="h-3 w-3" />
                        {t('finanzen.docChain.goToCustomer')}
                      </button>
                      {!chain.isComplete && (
                        <button className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors">
                          <ArrowRight className="h-3 w-3" />
                          {t('finanzen.docChain.nextStep')}
                        </button>
                      )}
                    </div>
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
