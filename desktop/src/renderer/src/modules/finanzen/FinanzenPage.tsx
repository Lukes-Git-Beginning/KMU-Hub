import { useState, useEffect } from 'react'
import {
  FileText,
  Search,
  Download,
  Plus,
  CheckCircle2,
  Clock,
  AlertCircle,
  XCircle,
  BarChart3,
  Gavel,
  Receipt,
  Link2,
  Landmark,
  Timer,
} from 'lucide-react'
import { toast } from 'sonner'
import { ItemActions, ConfirmDialog, EmptyState, PageHeader } from '@/components/shared'
import { useFinanceUIStore, formatEUR, type FinanceTabKey } from '@/stores/finance'
import {
  useInvoices,
  useQuotes,
  useCreditNotes,
  useSendInvoice,
  useCancelInvoice,
  useSendQuote,
  useAcceptQuote,
  useRejectQuote,
  useConvertQuoteToInvoice,
  useDeleteQuote,
  useSendCreditNote,
  useDownloadInvoicePDF,
  useDownloadQuotePDF,
  useDownloadCreditNotePDF,
} from '@/api/hooks/useFinance'
import type {
  Invoice,
  InvoiceStatus,
  Quote,
  QuoteStatus,
} from '@/types/finance-types'
import { InvoiceFormDialog } from './InvoiceFormDialog'
import { QuoteFormDialog } from './QuoteFormDialog'
import { CreditNoteDialog } from './CreditNoteDialog'
import { PaymentRecordDialog } from './PaymentRecordDialog'
import { InvoiceDetailPanel } from './InvoiceDetailPanel'
import { ExportDialog } from './ExportDialog'
import { DunningPanel } from './DunningPanel'
import { FinanceDashboard } from './FinanceDashboard'
import { BelegketteTab } from './BelegketteTab'
import { QRRechnungPreview, QRBillIndicator } from './QRRechnungPreview'
import { EInvoiceBadge, EInvoiceDetailDialog } from './EInvoiceIndicator'
import { HoursToInvoiceDialog } from './HoursToInvoiceDialog'
import { BankingWidget } from './BankingWidget'
import { AnimatedCheckmark } from '@/components/shared/AnimatedCheckmark'

// ---------------------------------------------------------------------------
// Status badge config
// ---------------------------------------------------------------------------

const invoiceStatusConfig: Record<
  InvoiceStatus,
  { label: string; colors: string; icon: typeof CheckCircle2 }
> = {
  draft: {
    label: 'Entwurf',
    colors: 'bg-secondary text-muted-foreground',
    icon: FileText,
  },
  sent: {
    label: 'Gesendet',
    colors: 'bg-info-light text-info',
    icon: Clock,
  },
  paid: {
    label: 'Bezahlt',
    colors: 'bg-success-light text-success',
    icon: CheckCircle2,
  },
  overdue: {
    label: 'Ueberfällig',
    colors: 'bg-error-light text-error',
    icon: AlertCircle,
  },
  cancelled: {
    label: 'Storniert',
    colors: 'bg-secondary text-muted-foreground',
    icon: XCircle,
  },
}

const quoteStatusConfig: Record<
  QuoteStatus,
  { label: string; colors: string; icon: typeof CheckCircle2 }
> = {
  draft: {
    label: 'Entwurf',
    colors: 'bg-secondary text-muted-foreground',
    icon: FileText,
  },
  sent: {
    label: 'Gesendet',
    colors: 'bg-info-light text-info',
    icon: Clock,
  },
  accepted: {
    label: 'Angenommen',
    colors: 'bg-success-light text-success',
    icon: CheckCircle2,
  },
  rejected: {
    label: 'Abgelehnt',
    colors: 'bg-error-light text-error',
    icon: XCircle,
  },
  expired: {
    label: 'Abgelaufen',
    colors: 'bg-secondary text-muted-foreground',
    icon: Clock,
  },
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function FinanzenPage() {
  const {
    activeTab,
    setActiveTab,
    invoiceFilter,
    setInvoiceFilter,
    quoteFilter,
  } = useFinanceUIStore()

  const [search, setSearch] = useState('')

  // Data hooks
  const { data: invoicesData, isLoading: invoicesLoading } = useInvoices(
    invoiceFilter.status ? { status: invoiceFilter.status } : undefined,
  )
  const { data: quotesData, isLoading: quotesLoading } = useQuotes(
    quoteFilter.status ? { status: quoteFilter.status } : undefined,
  )
  const { data: creditNotesData, isLoading: creditNotesLoading } =
    useCreditNotes()

  // Mutations
  const sendInvoice = useSendInvoice()
  const cancelInvoice = useCancelInvoice()
  const sendQuote = useSendQuote()
  const acceptQuote = useAcceptQuote()
  const rejectQuote = useRejectQuote()
  const convertQuote = useConvertQuoteToInvoice()
  const deleteQuote = useDeleteQuote()
  const sendCreditNote = useSendCreditNote()
  const downloadInvoicePDF = useDownloadInvoicePDF()
  const downloadQuotePDF = useDownloadQuotePDF()
  const downloadCreditNotePDF = useDownloadCreditNotePDF()

  // Dialog state
  const [showInvoiceForm, setShowInvoiceForm] = useState(false)
  const [editInvoice, setEditInvoice] = useState<Invoice | null>(null)
  const [showQuoteForm, setShowQuoteForm] = useState(false)
  const [editQuote, setEditQuote] = useState<Quote | null>(null)
  const [showCreditNote, setShowCreditNote] = useState(false)
  const [paymentInvoiceId, setPaymentInvoiceId] = useState<string | null>(null)
  const [selectedInvoiceId, setSelectedInvoiceId] = useState<string | null>(
    null,
  )
  const [showExport, setShowExport] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState<{
    type: string
    id: string
    label: string
  } | null>(null)
  const [qrPreviewInvoice, setQrPreviewInvoice] = useState<string | null>(null)
  const [eInvoiceDetailNumber, setEInvoiceDetailNumber] = useState<string | null>(null)
  const [showHoursToInvoice, setShowHoursToInvoice] = useState(false)
  const [sentAnimation, setSentAnimation] = useState<string | null>(null)

  // Auto-dismiss invoice sent animation
  useEffect(() => {
    if (!sentAnimation) return
    const timer = setTimeout(() => setSentAnimation(null), 2000)
    return () => clearTimeout(timer)
  }, [sentAnimation])

  // Filtered data
  const invoices = invoicesData?.invoices ?? []
  const quotes = quotesData?.quotes ?? []
  const creditNotes = creditNotesData?.credit_notes ?? []

  const filteredInvoices = invoices.filter((inv) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      inv.customer.name.toLowerCase().includes(q) ||
      inv.invoice_number.toLowerCase().includes(q)
    )
  })

  const filteredQuotes = quotes.filter((q) => {
    if (!search) return true
    const s = search.toLowerCase()
    return (
      q.customer.name.toLowerCase().includes(s) ||
      q.quote_number.toLowerCase().includes(s)
    )
  })

  const filteredCreditNotes = creditNotes.filter((cn) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      cn.customer.name.toLowerCase().includes(q) ||
      cn.credit_note_number.toLowerCase().includes(q)
    )
  })

  // Handlers
  const handleNewInvoice = () => {
    setEditInvoice(null)
    setShowInvoiceForm(true)
  }

  const handleEditInvoice = (inv: Invoice) => {
    setEditInvoice(inv)
    setShowInvoiceForm(true)
  }

  const handleNewQuote = () => {
    setEditQuote(null)
    setShowQuoteForm(true)
  }

  const handleDeleteConfirm = () => {
    if (!confirmDelete) return
    if (confirmDelete.type === 'quote') {
      deleteQuote.mutate(confirmDelete.id, {
        onSuccess: () => toast.success(`${confirmDelete.label} gelöscht`),
        onError: (err) => toast.error(err.message),
      })
    }
    setConfirmDelete(null)
  }

  const getInvoiceActions = (inv: Invoice) => {
    const actions: {
      label: string
      onClick: () => void
      variant?: 'destructive'
      separator?: true
    }[] = [
      { label: 'Details ansehen', onClick: () => setSelectedInvoiceId(inv.id) },
      {
        label: 'PDF herunterladen',
        onClick: () => downloadInvoicePDF.mutate(inv.id),
      },
    ]
    if (inv.status === 'draft') {
      actions.push({
        label: 'Bearbeiten',
        onClick: () => handleEditInvoice(inv),
      })
      actions.push({
        label: 'Senden',
        onClick: () => {
          sendInvoice.mutate(inv.id, {
            onSuccess: () => {
              setSentAnimation(inv.invoice_number)
              toast.success(`${inv.invoice_number} gesendet`)
            },
            onError: (err) => toast.error(err.message),
          })
        },
      })
    }
    if (
      inv.status !== 'paid' &&
      inv.status !== 'cancelled'
    ) {
      actions.push({
        label: 'Zahlung erfassen',
        onClick: () => setPaymentInvoiceId(inv.id),
      })
    }
    if (inv.status !== 'cancelled' && inv.status !== 'paid') {
      actions.push({
        separator: true as const,
        label: '',
        onClick: () => {},
      })
      actions.push({
        label: 'Stornieren',
        variant: 'destructive' as const,
        onClick: () => {
          cancelInvoice.mutate(inv.id, {
            onSuccess: () =>
              toast.success(`${inv.invoice_number} storniert`),
            onError: (err) => toast.error(err.message),
          })
        },
      })
    }
    return actions
  }

  const getQuoteActions = (q: Quote) => {
    const actions: {
      label: string
      onClick: () => void
      variant?: 'destructive'
      separator?: true
    }[] = [
      {
        label: 'PDF herunterladen',
        onClick: () => downloadQuotePDF.mutate(q.id),
      },
    ]
    if (q.status === 'draft') {
      actions.push({
        label: 'Senden',
        onClick: () => {
          sendQuote.mutate(q.id, {
            onSuccess: () =>
              toast.success(`${q.quote_number} gesendet`),
            onError: (err) => toast.error(err.message),
          })
        },
      })
    }
    if (q.status === 'sent') {
      actions.push({
        label: 'Annehmen',
        onClick: () => {
          acceptQuote.mutate(q.id, {
            onSuccess: () =>
              toast.success(`${q.quote_number} angenommen`),
            onError: (err) => toast.error(err.message),
          })
        },
      })
      actions.push({
        label: 'Ablehnen',
        onClick: () => {
          rejectQuote.mutate(q.id, {
            onSuccess: () =>
              toast.success(`${q.quote_number} abgelehnt`),
            onError: (err) => toast.error(err.message),
          })
        },
      })
    }
    if (q.status === 'accepted') {
      actions.push({
        label: 'In Rechnung umwandeln',
        onClick: () => {
          convertQuote.mutate(q.id, {
            onSuccess: () =>
              toast.success('Rechnung aus Angebot erstellt'),
            onError: (err) => toast.error(err.message),
          })
        },
      })
    }
    if (q.status === 'draft') {
      actions.push({
        separator: true as const,
        label: '',
        onClick: () => {},
      })
      actions.push({
        label: 'Löschen',
        variant: 'destructive' as const,
        onClick: () =>
          setConfirmDelete({
            type: 'quote',
            id: q.id,
            label: q.quote_number,
          }),
      })
    }
    return actions
  }

  // Tabs config
  const tabs: {
    key: FinanceTabKey
    label: string
    icon: typeof FileText
    count?: number
  }[] = [
    { key: 'dashboard', label: 'Dashboard', icon: BarChart3 },
    {
      key: 'invoices',
      label: `Rechnungen (${invoicesData?.total ?? 0})`,
      icon: FileText,
    },
    {
      key: 'quotes',
      label: `Angebote (${quotesData?.total ?? 0})`,
      icon: FileText,
    },
    {
      key: 'credit-notes',
      label: `Gutschriften (${creditNotesData?.total ?? 0})`,
      icon: Receipt,
    },
    { key: 'dunning', label: 'Mahnungen', icon: Gavel },
    { key: 'belegkette', label: 'Belegkette', icon: Link2 },
    { key: 'banking', label: 'Banking', icon: Landmark },
    { key: 'export', label: 'Export', icon: Download },
  ]

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <PageHeader
        title="Rechnungen & Finanzen"
        description="Rechnungen, Angebote, Gutschriften, Belegketten und Mahnwesen"
        icon={Receipt}
        moduleId="finance"
        className="mb-6"
        actions={
          <div className="flex gap-2">
            <button
              onClick={() => setShowHoursToInvoice(true)}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <Timer className="h-4 w-4" />
              Stunden abrechnen
            </button>
            <button
              onClick={() => setShowExport(true)}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <Download className="h-4 w-4" />
              DATEV Export
            </button>
            <button
              onClick={handleNewInvoice}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-4 w-4" />
              Neue Rechnung
            </button>
          </div>
        }
      />

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6 overflow-x-auto animate-fade-up stagger-1">
        {tabs.map((t) => {
          const Icon = t.icon
          return (
            <button
              key={t.key}
              onClick={() => setActiveTab(t.key)}
              className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm whitespace-nowrap transition-colors tab-accent-active ${
                activeTab === t.key
                  ? 'border-primary text-primary font-medium'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Icon className="h-4 w-4" />
              {t.label}
            </button>
          )
        })}
      </div>

      {/* Search (for list tabs) */}
      {(activeTab === 'invoices' ||
        activeTab === 'quotes' ||
        activeTab === 'credit-notes') && (
        <div className="flex items-center gap-3 mb-4">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Suchen..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
          {activeTab === 'quotes' && (
            <button
              onClick={handleNewQuote}
              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
              Neues Angebot
            </button>
          )}
          {activeTab === 'credit-notes' && (
            <button
              onClick={() => setShowCreditNote(true)}
              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
              Neue Gutschrift
            </button>
          )}
          {activeTab === 'invoices' && (
            <div className="flex items-center gap-1.5">
              {(
                [undefined, 'draft', 'sent', 'paid', 'overdue', 'cancelled'] as const
              ).map((st) => {
                const labels: Record<string, string> = {
                  '': 'Alle',
                  draft: 'Entwurf',
                  sent: 'Gesendet',
                  paid: 'Bezahlt',
                  overdue: 'Ueberfällig',
                  cancelled: 'Storniert',
                }
                return (
                  <button
                    key={st ?? ''}
                    onClick={() =>
                      setInvoiceFilter({
                        status: st as InvoiceStatus | undefined,
                      })
                    }
                    className={`rounded-md px-2 py-1 text-xs transition-colors ${
                      invoiceFilter.status === st
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-secondary text-muted-foreground hover:text-foreground'
                    }`}
                  >
                    {labels[st ?? '']}
                  </button>
                )
              })}
            </div>
          )}
        </div>
      )}

      {/* Dashboard Tab */}
      {activeTab === 'dashboard' && <FinanceDashboard />}

      {/* Invoices Tab */}
      {activeTab === 'invoices' &&
        (invoicesLoading ? (
          <LoadingState />
        ) : filteredInvoices.length === 0 ? (
          <EmptyState
            icon={FileText}
            title="Keine Rechnungen"
            description="Erstelle deine erste Rechnung"
            action={{ label: 'Neue Rechnung', onClick: handleNewInvoice }}
          />
        ) : (
          <div className="rounded-xl border border-border bg-card overflow-hidden animate-fade-up stagger-2">
            <div className="grid grid-cols-[100px_1fr_100px_100px_100px_160px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
              <span>Nr.</span>
              <span>Kunde</span>
              <span>Betrag</span>
              <span>Fällig</span>
              <span>Offen</span>
              <span>Status</span>
              <span />
            </div>
            {filteredInvoices.map((inv) => {
              const sc = invoiceStatusConfig[inv.status]
              const StatusIcon = sc.icon
              const grossTotal = Number(inv.tax_breakdown.gross_total)
              return (
                <div
                  key={inv.id}
                  className="grid grid-cols-[100px_1fr_100px_100px_100px_160px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors"
                >
                  <button
                    onClick={() => setSelectedInvoiceId(inv.id)}
                    className="text-sm font-mono text-primary hover:underline text-left"
                  >
                    {inv.invoice_number}
                  </button>
                  <span className="text-sm text-foreground truncate">
                    {inv.customer.name}
                  </span>
                  <span className="text-sm font-medium text-foreground stat-accent">
                    {formatEUR(grossTotal)}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {new Date(inv.due_date).toLocaleDateString('de-DE')}
                  </span>
                  <span
                    className={`text-xs font-medium ${
                      inv.status !== 'paid' ? 'text-warning' : 'text-success'
                    }`}
                  >
                    {inv.status !== 'paid'
                      ? formatEUR(inv.tax_breakdown.gross_total)
                      : '--'}
                  </span>
                  <div className="flex items-center gap-1 flex-wrap">
                    <span
                      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.colors}`}
                    >
                      <StatusIcon className="h-3 w-3" />
                      {sc.label}
                    </span>
                    <QRBillIndicator
                      hasQRBill={inv.invoice_number === 'RE-2026-003' || inv.invoice_number === 'RE-2026-008'}
                      invoiceNumber={inv.invoice_number}
                      onPreview={() => setQrPreviewInvoice(inv.invoice_number)}
                    />
                    <EInvoiceBadge
                      invoiceNumber={inv.invoice_number}
                      onClick={() => setEInvoiceDetailNumber(inv.invoice_number)}
                    />
                  </div>
                  <ItemActions items={getInvoiceActions(inv)} />
                </div>
              )
            })}
          </div>
        ))}

      {/* Quotes Tab */}
      {activeTab === 'quotes' &&
        (quotesLoading ? (
          <LoadingState />
        ) : filteredQuotes.length === 0 ? (
          <EmptyState
            icon={FileText}
            title="Keine Angebote"
            description="Erstelle dein erstes Angebot"
            action={{ label: 'Neues Angebot', onClick: handleNewQuote }}
          />
        ) : (
          <div className="rounded-xl border border-border bg-card overflow-hidden animate-fade-up stagger-2">
            <div className="grid grid-cols-[100px_1fr_100px_100px_90px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
              <span>Nr.</span>
              <span>Kunde</span>
              <span>Betrag</span>
              <span>Gültig bis</span>
              <span>Status</span>
              <span />
            </div>
            {filteredQuotes.map((q) => {
              const sc = quoteStatusConfig[q.status]
              const StatusIcon = sc.icon
              return (
                <div
                  key={q.id}
                  className="grid grid-cols-[100px_1fr_100px_100px_90px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors"
                >
                  <span className="text-sm font-mono text-primary">
                    {q.quote_number}
                  </span>
                  <span className="text-sm text-foreground truncate">
                    {q.customer.name}
                  </span>
                  <span className="text-sm font-medium text-foreground">
                    {formatEUR(q.tax_breakdown.gross_total)}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {new Date(q.valid_until).toLocaleDateString('de-DE')}
                  </span>
                  <span
                    className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.colors}`}
                  >
                    <StatusIcon className="h-3 w-3" />
                    {sc.label}
                  </span>
                  <ItemActions items={getQuoteActions(q)} />
                </div>
              )
            })}
          </div>
        ))}

      {/* Credit Notes Tab */}
      {activeTab === 'credit-notes' &&
        (creditNotesLoading ? (
          <LoadingState />
        ) : filteredCreditNotes.length === 0 ? (
          <EmptyState
            icon={Receipt}
            title="Keine Gutschriften"
            description="Erstelle eine Gutschrift zu einer bestehenden Rechnung"
            action={{
              label: 'Neue Gutschrift',
              onClick: () => setShowCreditNote(true),
            }}
          />
        ) : (
          <div className="rounded-xl border border-border bg-card overflow-hidden animate-fade-up stagger-2">
            <div className="grid grid-cols-[100px_1fr_120px_100px_90px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
              <span>Nr.</span>
              <span>Kunde</span>
              <span>Originalrechnung</span>
              <span>Betrag</span>
              <span>Status</span>
              <span />
            </div>
            {filteredCreditNotes.map((cn) => {
              const isDraft = cn.status === 'draft'
              return (
                <div
                  key={cn.id}
                  className="grid grid-cols-[100px_1fr_120px_100px_90px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors"
                >
                  <span className="text-sm font-mono text-primary">
                    {cn.credit_note_number}
                  </span>
                  <span className="text-sm text-foreground truncate">
                    {cn.customer.name}
                  </span>
                  <span className="text-xs text-muted-foreground font-mono">
                    {cn.original_invoice_id.slice(0, 8)}...
                  </span>
                  <span className="text-sm font-medium text-foreground">
                    {formatEUR(cn.tax_breakdown.gross_total)}
                  </span>
                  <span
                    className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${
                      isDraft
                        ? 'bg-secondary text-muted-foreground'
                        : 'bg-info-light text-info'
                    }`}
                  >
                    {isDraft ? 'Entwurf' : 'Gesendet'}
                  </span>
                  <ItemActions
                    items={[
                      {
                        label: 'PDF herunterladen',
                        onClick: () =>
                          downloadCreditNotePDF.mutate(cn.id),
                      },
                      ...(isDraft
                        ? [
                            {
                              label: 'Senden',
                              onClick: () => {
                                sendCreditNote.mutate(cn.id, {
                                  onSuccess: () =>
                                    toast.success('Gutschrift gesendet'),
                                  onError: (err: Error) =>
                                    toast.error(err.message),
                                })
                              },
                            },
                          ]
                        : []),
                    ]}
                  />
                </div>
              )
            })}
          </div>
        ))}

      {/* Dunning Tab */}
      {activeTab === 'dunning' && <DunningPanel />}

      {/* Belegkette Tab */}
      {activeTab === 'belegkette' && <BelegketteTab />}

      {/* Banking Tab */}
      {activeTab === 'banking' && <BankingWidget />}

      {/* Export Tab */}
      {activeTab === 'export' && (
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Exportiere Finanzdaten in verschiedene Buchhaltungsformate.
          </p>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            {/* DATEV */}
            <div className="rounded-lg border border-border p-4 space-y-3">
              <div className="flex items-center gap-2">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                  <Download className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">DATEV</p>
                  <p className="text-[10px] text-muted-foreground">EXTF-Format (CSV)</p>
                </div>
              </div>
              <p className="text-xs text-muted-foreground">
                Buchungsstapel für DATEV Unternehmen online, Kanzlei-Rechnungswesen und Steuerberater-Software.
              </p>
              <button
                onClick={() => setShowExport(true)}
                className="w-full flex items-center justify-center gap-1.5 rounded-md bg-primary px-3 py-2 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <Download className="h-3.5 w-3.5" />
                DATEV Export
              </button>
            </div>
            {/* Bexio */}
            <div className="rounded-lg border border-border p-4 space-y-3">
              <div className="flex items-center gap-2">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info-light">
                  <Download className="h-5 w-5 text-info" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">Bexio</p>
                  <p className="text-[10px] text-muted-foreground">CSV-Import</p>
                </div>
              </div>
              <p className="text-xs text-muted-foreground">
                Export im Bexio-kompatiblen CSV-Format. Rechnungen, Kontakte und Artikel separat exportierbar.
              </p>
              <button
                onClick={() => toast.success('Bexio-Export heruntergeladen')}
                className="w-full flex items-center justify-center gap-1.5 rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-accent transition-colors"
              >
                <Download className="h-3.5 w-3.5" />
                Bexio Export
              </button>
            </div>
            {/* BMD */}
            <div className="rounded-lg border border-border p-4 space-y-3">
              <div className="flex items-center gap-2">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-success-light">
                  <Download className="h-5 w-5 text-success" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">BMD</p>
                  <p className="text-[10px] text-muted-foreground">NTCS-Format</p>
                </div>
              </div>
              <p className="text-xs text-muted-foreground">
                Export für BMD NTCS Buchhaltung. Standard in Österreich für Steuerberater und KMUs.
              </p>
              <button
                onClick={() => toast.success('BMD-Export heruntergeladen')}
                className="w-full flex items-center justify-center gap-1.5 rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-accent transition-colors"
              >
                <Download className="h-3.5 w-3.5" />
                BMD Export
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Invoice Detail Panel */}
      {selectedInvoiceId && (
        <InvoiceDetailPanel
          invoiceId={selectedInvoiceId}
          onClose={() => setSelectedInvoiceId(null)}
          onEdit={() => {
            const inv = invoices.find((i) => i.id === selectedInvoiceId)
            if (inv) handleEditInvoice(inv)
            setSelectedInvoiceId(null)
          }}
          onRecordPayment={() => {
            setPaymentInvoiceId(selectedInvoiceId)
            setSelectedInvoiceId(null)
          }}
        />
      )}

      {/* Invoice Form */}
      <InvoiceFormDialog
        open={showInvoiceForm}
        onOpenChange={setShowInvoiceForm}
        editInvoice={editInvoice}
      />

      {/* Quote Form */}
      <QuoteFormDialog
        open={showQuoteForm}
        onOpenChange={setShowQuoteForm}
        editQuote={editQuote}
      />

      {/* Credit Note Dialog */}
      <CreditNoteDialog
        open={showCreditNote}
        onOpenChange={setShowCreditNote}
      />

      {/* Payment Record */}
      <PaymentRecordDialog
        open={!!paymentInvoiceId}
        onOpenChange={() => setPaymentInvoiceId(null)}
        invoiceId={paymentInvoiceId}
      />

      {/* Export */}
      <ExportDialog open={showExport} onOpenChange={setShowExport} />

      {/* QR-Rechnung Preview */}
      <QRRechnungPreview
        open={!!qrPreviewInvoice}
        onOpenChange={() => setQrPreviewInvoice(null)}
        invoiceNumber={qrPreviewInvoice ?? undefined}
      />

      {/* E-Invoice Detail */}
      <EInvoiceDetailDialog
        open={!!eInvoiceDetailNumber}
        onOpenChange={() => setEInvoiceDetailNumber(null)}
        invoiceNumber={eInvoiceDetailNumber ?? ''}
      />

      {/* Hours to Invoice */}
      <HoursToInvoiceDialog
        open={showHoursToInvoice}
        onOpenChange={setShowHoursToInvoice}
      />

      {/* Confirm Delete */}
      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title="Eintrag löschen?"
        description={`"${confirmDelete?.label}" wird dauerhaft gelöscht.`}
        confirmLabel="Löschen"
        variant="destructive"
        onConfirm={handleDeleteConfirm}
      />

      {/* Invoice sent success overlay */}
      {sentAnimation && (
        <div
          className="fixed inset-0 z-[100] flex items-center justify-center bg-black/10 animate-fade-in"
          onClick={() => setSentAnimation(null)}
        >
          <div className="flex flex-col items-center gap-3 rounded-2xl bg-card p-8 shadow-large animate-scale-in-bounce">
            <AnimatedCheckmark size={56} />
            <p className="text-sm font-medium text-foreground">{sentAnimation} gesendet</p>
          </div>
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Loading state
// ---------------------------------------------------------------------------

function LoadingState() {
  return (
    <div className="space-y-3 py-4">
      {[1, 2, 3].map((i) => (
        <div key={i} className="h-14 rounded-xl bg-secondary/50 animate-shimmer" />
      ))}
    </div>
  )
}
