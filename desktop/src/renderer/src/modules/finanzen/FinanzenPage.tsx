import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
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
  Repeat,
  Wallet,
  PieChart,
} from 'lucide-react'
import { toast } from 'sonner'
import { ItemActions, ConfirmDialog, EmptyState, PageHeader } from '@/components/shared'
import { useFinanceUIStore, formatMoney, type FinanceTabKey } from '@/stores/finance'
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
import { QuoteDetailPanel } from './QuoteDetailPanel'
import { CreditNoteDetailPanel } from './CreditNoteDetailPanel'
import { FinanceDetailNavProvider, useFinanceDetailNav } from './FinanceDetailNav'
import { buildBexioCsv, buildBmdCsv, downloadCsv } from './lib/finance-export'
import { ExportDialog } from './ExportDialog'
import { DunningPanel } from './DunningPanel'
import { FinanceDashboard } from './FinanceDashboard'
import { BelegketteTab } from './BelegketteTab'
import { ExpensesTab } from './tabs/ExpensesTab'
import { TransactionsTab } from './tabs/TransactionsTab'
import { BerichteTab } from './tabs/BerichteTab'
import { RecurringInvoicesTab } from './RecurringInvoicesTab'
import { OpenItemsTab } from './OpenItemsTab'
import { QRRechnungPreview, QRBillIndicator } from './QRRechnungPreview'
import { EInvoiceBadge, EInvoiceDetailDialog } from './EInvoiceIndicator'
import { HoursToInvoiceDialog } from './HoursToInvoiceDialog'
import { BankingWidget } from './BankingWidget'
import { AnimatedCheckmark } from '@/components/shared/AnimatedCheckmark'
import { useFinancePrefsStore } from '@/stores/financePrefs'
import { formatDate } from '@/lib/format'

// ---------------------------------------------------------------------------
// Status badge config
// ---------------------------------------------------------------------------

const invoiceStatusConfig: Record<
  InvoiceStatus,
  { labelKey: string; colors: string; icon: typeof CheckCircle2 }
> = {
  draft: {
    labelKey: 'finanzen.status.draft',
    colors: 'bg-secondary text-muted-foreground',
    icon: FileText,
  },
  sent: {
    labelKey: 'finanzen.status.sent',
    colors: 'bg-info-light text-info',
    icon: Clock,
  },
  paid: {
    labelKey: 'finanzen.status.paid',
    colors: 'bg-success-light text-success',
    icon: CheckCircle2,
  },
  overdue: {
    labelKey: 'finanzen.status.overdue',
    colors: 'bg-error-light text-error',
    icon: AlertCircle,
  },
  cancelled: {
    labelKey: 'finanzen.status.cancelled',
    colors: 'bg-secondary text-muted-foreground',
    icon: XCircle,
  },
}

const quoteStatusConfig: Record<
  QuoteStatus,
  { labelKey: string; colors: string; icon: typeof CheckCircle2 }
> = {
  draft: {
    labelKey: 'finanzen.status.draft',
    colors: 'bg-secondary text-muted-foreground',
    icon: FileText,
  },
  sent: {
    labelKey: 'finanzen.status.sent',
    colors: 'bg-info-light text-info',
    icon: Clock,
  },
  accepted: {
    labelKey: 'finanzen.quoteStatus.accepted',
    colors: 'bg-success-light text-success',
    icon: CheckCircle2,
  },
  rejected: {
    labelKey: 'finanzen.quoteStatus.rejected',
    colors: 'bg-error-light text-error',
    icon: XCircle,
  },
  expired: {
    labelKey: 'finanzen.quoteStatus.expired',
    colors: 'bg-secondary text-muted-foreground',
    icon: Clock,
  },
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function FinanzenPage() {
  return (
    <FinanceDetailNavProvider>
      <FinanzenPageContent />
    </FinanceDetailNavProvider>
  )
}

function FinanzenPageContent() {
  const { t } = useTranslation()
  const { current: navCurrent, depth: navDepth, open: openDetail, back: navBack, close: navClose } =
    useFinanceDetailNav()
  const {
    activeTab,
    setActiveTab,
    invoiceFilter,
    setInvoiceFilter,
    quoteFilter,
  } = useFinanceUIStore()

  // Settings/Stammdaten/Integrationen moved to the module-settings overlay
  // (FinanceSettingsPanel). Guard persisted state pointing at a removed tab.
  const retiredTabs: FinanceTabKey[] = ['settings', 'stammdaten', 'finanz-integrationen']
  const effectiveTab: FinanceTabKey = retiredTabs.includes(activeTab) ? 'dashboard' : activeTab

  // Personal pref: open at a fixed start tab (unless "last").
  const startTab = useFinancePrefsStore((s) => s.startTab)
  useEffect(() => {
    if (startTab !== 'last' && !retiredTabs.includes(startTab)) {
      setActiveTab(startTab)
    }
    // run once on mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

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
  const [creditNote, setCreditNote] = useState<{ invoice: Invoice | null; storno: boolean } | null>(null)
  const [paymentInvoiceId, setPaymentInvoiceId] = useState<string | null>(null)

  // Phase 10 (vertraege): consume ?invoice=<id> from CRM/Vertraege navigation chip.
  // Effect is reactive (depends on searchParams value) so re-fires when navigating
  // from the same page — intentional, not a mount-only effect.
  // Guard: only set selectedInvoiceId once invoices have loaded AND the ID is known.
  // Unknown IDs (e.g. stale links) are silently ignored — no crash.
  const [searchParams, setSearchParams] = useSearchParams()
  useEffect(() => {
    const invoiceParam = searchParams.get('invoice')
    if (!invoiceParam) return
    // Wait until invoices are loaded before checking existence
    if (invoicesLoading) return
    // Accept the param if the ID exists among loaded invoices, or if the list
    // is empty (backend may have the invoice even if the filter returns nothing)
    const loadedInvoices = invoicesData?.invoices ?? []
    const known = loadedInvoices.length === 0 || loadedInvoices.some((inv) => inv.id === invoiceParam)
    if (known) {
      openDetail('invoice', invoiceParam)
    }
    // Always remove the query param so the URL is clean
    const next = new URLSearchParams(searchParams)
    next.delete('invoice')
    setSearchParams(next, { replace: true })
  }, [searchParams, setSearchParams, invoicesLoading, invoicesData, openDetail])

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
      inv.customer?.name?.toLowerCase().includes(q) ||
      inv.invoice_number?.toLowerCase().includes(q)
    )
  })

  const filteredQuotes = quotes.filter((q) => {
    if (!search) return true
    const s = search.toLowerCase()
    return (
      q.customer?.name?.toLowerCase().includes(s) ||
      q.quote_number?.toLowerCase().includes(s)
    )
  })

  const filteredCreditNotes = creditNotes.filter((cn) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      cn.customer?.name?.toLowerCase().includes(q) ||
      cn.credit_note_number?.toLowerCase().includes(q)
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
        onSuccess: () => toast.success(t('common.delete') + ': ' + confirmDelete.label),
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
      { label: t('common.details'), onClick: () => openDetail('invoice', inv.id) },
      {
        label: t('finanzen.pdf.downloadPdf'),
        onClick: () => downloadInvoicePDF.mutate(inv.id),
      },
    ]
    if (inv.status === 'draft') {
      actions.push({
        label: t('common.edit'),
        onClick: () => handleEditInvoice(inv),
      })
      actions.push({
        label: t('finanzen.dunning.send'),
        onClick: () => {
          sendInvoice.mutate(inv.id, {
            onSuccess: () => {
              setSentAnimation(inv.invoice_number)
              toast.success(`${inv.invoice_number} ${t('finanzen.dunning.sent')}`)
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
        label: t('finanzen.invoiceDetail.recordPayment'),
        onClick: () => setPaymentInvoiceId(inv.id),
      })
    }
    // Issued invoices are storniert via a linked credit note (GoBD); drafts
    // can be cancelled directly since they were never issued.
    if (inv.status === 'sent' || inv.status === 'overdue') {
      actions.push({
        separator: true as const,
        label: '',
        onClick: () => {},
      })
      actions.push({
        label: t('finanzen.creditNote.createForInvoice'),
        onClick: () => setCreditNote({ invoice: inv, storno: false }),
      })
      actions.push({
        label: t('finanzen.invoiceDetail.storno'),
        variant: 'destructive' as const,
        onClick: () => setCreditNote({ invoice: inv, storno: true }),
      })
    } else if (inv.status === 'draft') {
      actions.push({
        separator: true as const,
        label: '',
        onClick: () => {},
      })
      actions.push({
        label: t('finanzen.invoiceDetail.cancel'),
        variant: 'destructive' as const,
        onClick: () => {
          cancelInvoice.mutate(inv.id, {
            onSuccess: () =>
              toast.success(`${inv.invoice_number} ${t('finanzen.status.cancelled')}`),
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
        label: t('common.details'),
        onClick: () => openDetail('quote', q.id),
      },
      {
        label: t('finanzen.pdf.downloadPdf'),
        onClick: () => downloadQuotePDF.mutate(q.id),
      },
    ]
    if (q.status === 'draft') {
      actions.push({
        label: t('finanzen.dunning.send'),
        onClick: () => {
          sendQuote.mutate(q.id, {
            onSuccess: () =>
              toast.success(`${q.quote_number} ${t('finanzen.dunning.sent')}`),
            onError: (err) => toast.error(err.message),
          })
        },
      })
    }
    if (q.status === 'sent') {
      actions.push({
        label: t('finanzen.page.accept'),
        onClick: () => {
          acceptQuote.mutate(q.id, {
            onSuccess: () =>
              toast.success(`${q.quote_number} ${t('finanzen.quoteStatus.accepted')}`),
            onError: (err) => toast.error(err.message),
          })
        },
      })
      actions.push({
        label: t('finanzen.page.reject'),
        onClick: () => {
          rejectQuote.mutate(q.id, {
            onSuccess: () =>
              toast.success(`${q.quote_number} ${t('finanzen.quoteStatus.rejected')}`),
            onError: (err) => toast.error(err.message),
          })
        },
      })
    }
    if (q.status === 'accepted') {
      actions.push({
        label: t('finanzen.page.convertToInvoice'),
        onClick: () => {
          convertQuote.mutate(q.id, {
            onSuccess: () =>
              toast.success(t('finanzen.page.invoiceFromQuoteCreated')),
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
        label: t('common.delete'),
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
      label: t('finanzen.tabs.invoices', { count: invoicesData?.total ?? 0 }),
      icon: FileText,
    },
    {
      key: 'open-items',
      label: t('finanzen.tabs.openItems'),
      icon: Wallet,
    },
    {
      key: 'quotes',
      label: t('finanzen.tabs.quotes', { count: quotesData?.total ?? 0 }),
      icon: FileText,
    },
    {
      key: 'recurring',
      label: t('finanzen.tabs.recurring'),
      icon: Repeat,
    },
    {
      key: 'credit-notes',
      label: t('finanzen.tabs.creditNotes', { count: creditNotesData?.total ?? 0 }),
      icon: Receipt,
    },
    {
      key: 'expenses',
      label: t('buchhaltung.tabs.expenses'),
      icon: Receipt,
    },
    {
      key: 'transactions',
      label: t('buchhaltung.tabs.transactions'),
      icon: Timer,
    },
    { key: 'berichte', label: t('buchhaltung.tabs.reports'), icon: PieChart },
    { key: 'dunning', label: t('finanzen.tabs.dunning'), icon: Gavel },
    { key: 'belegkette', label: t('finanzen.tabs.belegkette'), icon: Link2 },
    { key: 'banking', label: t('finanzen.tabs.banking'), icon: Landmark },
    { key: 'export', label: t('finanzen.tabs.export'), icon: Download },
  ]

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <PageHeader
        title={t('finanzen.page.title')}
        description={t('finanzen.page.description')}
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
              {t('finanzen.hours.title')}
            </button>
            <button
              onClick={() => setShowExport(true)}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <Download className="h-4 w-4" />
              {t('finanzen.page.datevExport')}
            </button>
            <button
              onClick={handleNewInvoice}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-4 w-4" />
              {t('finanzen.invoices.create')}
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
                effectiveTab === t.key
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
      {(effectiveTab === 'invoices' ||
        effectiveTab === 'quotes' ||
        effectiveTab === 'credit-notes') && (
        <div className="flex items-center gap-3 mb-4">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder={t('finanzen.page.searchPlaceholder')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
          {effectiveTab === 'quotes' && (
            <button
              onClick={handleNewQuote}
              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
              {t('finanzen.quotes.create')}
            </button>
          )}
          {effectiveTab === 'credit-notes' && (
            <button
              onClick={() => setCreditNote({ invoice: null, storno: false })}
              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
              {t('finanzen.creditNote.createTitle')}
            </button>
          )}
          {effectiveTab === 'invoices' && (
            <div className="flex items-center gap-1.5">
              {(
                [undefined, 'draft', 'sent', 'paid', 'overdue', 'cancelled'] as const
              ).map((st) => {
                const labels: Record<string, string> = {
                  '': t('finanzen.filterAll'),
                  draft: t('finanzen.status.draft'),
                  sent: t('finanzen.status.sent'),
                  paid: t('finanzen.status.paid'),
                  overdue: t('finanzen.status.overdue'),
                  cancelled: t('finanzen.status.cancelled'),
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
      {effectiveTab === 'dashboard' && (
        <FinanceDashboard
          onOpenInvoice={(id) => openDetail('invoice', id)}
          onOpenQuote={(id) => openDetail('quote', id)}
          onOpenDunnings={() => setActiveTab('dunning')}
        />
      )}

      {/* Recurring invoices Tab */}
      {effectiveTab === 'recurring' && <RecurringInvoicesTab />}

      {/* Open items (OP-Liste) Tab */}
      {effectiveTab === 'open-items' && <OpenItemsTab onOpenInvoice={(id) => openDetail('invoice', id)} />}

      {/* Invoices Tab */}
      {effectiveTab === 'invoices' &&
        (invoicesLoading ? (
          <LoadingState />
        ) : filteredInvoices.length === 0 ? (
          <EmptyState
            icon={FileText}
            title={t('finanzen.invoices.emptyTitle')}
            description={t('finanzen.invoices.emptyDescription')}
            action={{ label: t('finanzen.invoices.create'), onClick: handleNewInvoice }}
          />
        ) : (
          <div className="rounded-xl border border-border bg-card overflow-hidden animate-fade-up stagger-2">
            <div className="grid grid-cols-[100px_1fr_100px_100px_100px_160px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
              <span>{t('finanzen.col.nr')}</span>
              <span>{t('finanzen.col.customer')}</span>
              <span>{t('finanzen.col.amount')}</span>
              <span>{t('finanzen.col.dueDate')}</span>
              <span>{t('finanzen.col.open')}</span>
              <span>{t('common.status')}</span>
              <span />
            </div>
            {filteredInvoices.map((inv) => {
              const sc = invoiceStatusConfig[inv.status]
              const StatusIcon = sc.icon
              const grossTotal = Number(inv.tax_breakdown?.gross_total ?? inv.total_gross ?? 0)
              return (
                <div
                  key={inv.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => openDetail('invoice', inv.id)}
                  onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openDetail('invoice', inv.id) } }}
                  className="grid cursor-pointer grid-cols-[100px_1fr_100px_100px_100px_160px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors focus-visible:bg-secondary/40 focus-visible:outline-none"
                >
                  <span className="text-sm font-mono text-primary">{inv.invoice_number}</span>
                  <span className="text-sm text-foreground truncate">
                    {inv.customer.name}
                  </span>
                  <span className="text-sm font-medium text-foreground stat-accent">
                    {formatMoney(grossTotal, inv.currency)}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {formatDate(inv.due_date)}
                  </span>
                  <span
                    className={`text-xs font-medium ${
                      inv.status !== 'paid' ? 'text-warning' : 'text-success'
                    }`}
                  >
                    {inv.status !== 'paid'
                      ? formatMoney(inv.tax_breakdown?.gross_total ?? inv.total_gross ?? 0, inv.currency)
                      : '--'}
                  </span>
                  <div className="flex items-center gap-1 flex-wrap">
                    <span
                      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.colors}`}
                    >
                      <StatusIcon className="h-3 w-3" />
                      {t(sc.labelKey)}
                    </span>
                    <span onClick={(e) => e.stopPropagation()} className="contents">
                      <QRBillIndicator
                        hasQRBill={inv.invoice_number === 'RE-2026-003' || inv.invoice_number === 'RE-2026-008'}
                        invoiceNumber={inv.invoice_number}
                        onPreview={() => setQrPreviewInvoice(inv.invoice_number)}
                      />
                      <EInvoiceBadge
                        invoiceNumber={inv.invoice_number}
                        onClick={() => setEInvoiceDetailNumber(inv.invoice_number)}
                      />
                    </span>
                  </div>
                  <div onClick={(e) => e.stopPropagation()}>
                    <ItemActions items={getInvoiceActions(inv)} />
                  </div>
                </div>
              )
            })}
          </div>
        ))}

      {/* Quotes Tab */}
      {effectiveTab === 'quotes' &&
        (quotesLoading ? (
          <LoadingState />
        ) : filteredQuotes.length === 0 ? (
          <EmptyState
            icon={FileText}
            title={t('finanzen.quotes.emptyTitle')}
            description={t('finanzen.quotes.emptyDescription')}
            action={{ label: t('finanzen.quotes.create'), onClick: handleNewQuote }}
          />
        ) : (
          <div className="rounded-xl border border-border bg-card overflow-hidden animate-fade-up stagger-2">
            <div className="grid grid-cols-[100px_1fr_100px_100px_90px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
              <span>{t('finanzen.col.nr')}</span>
              <span>{t('finanzen.col.customer')}</span>
              <span>{t('finanzen.col.amount')}</span>
              <span>{t('finanzen.quoteForm.validUntil')}</span>
              <span>{t('common.status')}</span>
              <span />
            </div>
            {filteredQuotes.map((q) => {
              const sc = quoteStatusConfig[q.status]
              const StatusIcon = sc.icon
              return (
                <div
                  key={q.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => openDetail('quote', q.id)}
                  onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openDetail('quote', q.id) } }}
                  className="grid cursor-pointer grid-cols-[100px_1fr_100px_100px_90px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors focus-visible:bg-secondary/40 focus-visible:outline-none"
                >
                  <span className="text-sm font-mono text-primary">{q.quote_number}</span>
                  <span className="text-sm text-foreground truncate">
                    {q.customer.name}
                  </span>
                  <span className="text-sm font-medium text-foreground">
                    {formatMoney(q.tax_breakdown?.gross_total ?? q.total_gross ?? 0, q.currency)}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {formatDate(q.valid_until)}
                  </span>
                  <span
                    className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.colors}`}
                  >
                    <StatusIcon className="h-3 w-3" />
                    {t(sc.labelKey)}
                  </span>
                  <div onClick={(e) => e.stopPropagation()}>
                    <ItemActions items={getQuoteActions(q)} />
                  </div>
                </div>
              )
            })}
          </div>
        ))}

      {/* Credit Notes Tab */}
      {effectiveTab === 'credit-notes' &&
        (creditNotesLoading ? (
          <LoadingState />
        ) : filteredCreditNotes.length === 0 ? (
          <EmptyState
            icon={Receipt}
            title={t('finanzen.creditNotes.emptyTitle')}
            description={t('finanzen.creditNotes.emptyDescription')}
            action={{
              label: t('finanzen.creditNote.createTitle'),
              onClick: () => setCreditNote({ invoice: null, storno: false }),
            }}
          />
        ) : (
          <div className="rounded-xl border border-border bg-card overflow-hidden animate-fade-up stagger-2">
            <div className="grid grid-cols-[100px_1fr_120px_100px_90px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
              <span>{t('finanzen.col.nr')}</span>
              <span>{t('finanzen.col.customer')}</span>
              <span>{t('finanzen.creditNote.originalInvoice')}</span>
              <span>{t('finanzen.col.amount')}</span>
              <span>{t('common.status')}</span>
              <span />
            </div>
            {filteredCreditNotes.map((cn) => {
              const isDraft = cn.status === 'draft'
              return (
                <div
                  key={cn.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => openDetail('creditNote', cn.id)}
                  onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openDetail('creditNote', cn.id) } }}
                  className="grid cursor-pointer grid-cols-[100px_1fr_120px_100px_90px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors focus-visible:bg-secondary/40 focus-visible:outline-none"
                >
                  <span className="text-sm font-mono text-primary">{cn.credit_note_number}</span>
                  <span className="text-sm text-foreground truncate">
                    {cn.customer.name}
                  </span>
                  <span className="text-xs text-muted-foreground font-mono">
                    {cn.invoice_number ?? '--'}
                    {cn.is_storno && (
                      <span className="ml-1 rounded bg-error-light px-1 py-0.5 text-[9px] font-medium text-error not-italic">
                        {t('finanzen.creditNote.stornoBadge')}
                      </span>
                    )}
                  </span>
                  <span className="text-sm font-medium text-foreground">
                    {formatMoney(cn.tax_breakdown?.gross_total ?? cn.total_gross ?? 0, cn.currency)}
                  </span>
                  <span
                    className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${
                      isDraft
                        ? 'bg-secondary text-muted-foreground'
                        : 'bg-info-light text-info'
                    }`}
                  >
                    {isDraft ? t('finanzen.status.draft') : t('finanzen.status.sent')}
                  </span>
                  <div onClick={(e) => e.stopPropagation()}>
                  <ItemActions
                    items={[
                      {
                        label: t('common.details'),
                        onClick: () => openDetail('creditNote', cn.id),
                      },
                      {
                        label: t('finanzen.pdf.downloadPdf'),
                        onClick: () =>
                          downloadCreditNotePDF.mutate(cn.id),
                      },
                      ...(isDraft
                        ? [
                            {
                              label: t('finanzen.dunning.send'),
                              onClick: () => {
                                sendCreditNote.mutate(cn.id, {
                                  onSuccess: () =>
                                    toast.success(t('finanzen.creditNote.sent')),
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
                </div>
              )
            })}
          </div>
        ))}

      {/* Expenses Tab (migriert aus altem buchhaltung-Modul) */}
      {effectiveTab === 'expenses' && <ExpensesTab />}

      {/* Transactions Tab (migriert aus altem buchhaltung-Modul) */}
      {effectiveTab === 'transactions' && <TransactionsTab />}

      {/* Berichte Tab (Einnahmen/Ausgaben + Kategorien, finanzen P2) */}
      {effectiveTab === 'berichte' && <BerichteTab />}

      {/* Dunning Tab */}
      {effectiveTab === 'dunning' && <DunningPanel />}

      {/* Belegkette Tab */}
      {effectiveTab === 'belegkette' && <BelegketteTab />}

      {/* Banking Tab */}
      {effectiveTab === 'banking' && <BankingWidget />}

      {/* Export Tab */}
      {effectiveTab === 'export' && (
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {t('finanzen.exportTab.description')}
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
                {t('finanzen.exportTab.datevDescription')}
              </p>
              <button
                onClick={() => setShowExport(true)}
                className="w-full flex items-center justify-center gap-1.5 rounded-md bg-primary px-3 py-2 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <Download className="h-3.5 w-3.5" />
                {t('finanzen.page.datevExport')}
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
                {t('finanzen.exportTab.bexioDescription')}
              </p>
              <button
                onClick={() => {
                  downloadCsv(buildBexioCsv(invoices as never), 'bexio-rechnungen.csv')
                  toast.success(t('finanzen.exportTab.bexioDownloaded'))
                }}
                className="w-full flex items-center justify-center gap-1.5 rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-accent transition-colors"
              >
                <Download className="h-3.5 w-3.5" />
                {t('finanzen.exportTab.bexioExport')}
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
                {t('finanzen.exportTab.bmdDescription')}
              </p>
              <button
                onClick={() => {
                  downloadCsv(buildBmdCsv(invoices as never), 'bmd-ntcs-export.csv')
                  toast.success(t('finanzen.exportTab.bmdDownloaded'))
                }}
                className="w-full flex items-center justify-center gap-1.5 rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-accent transition-colors"
              >
                <Download className="h-3.5 w-3.5" />
                {t('finanzen.exportTab.bmdExport')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Beleg-Detail-Modals — zentral aus dem Navigations-Stack gerendert.
          „Zurück" erscheint, sobald die Navigations-Kette tiefer als 1 ist. */}
      {navCurrent?.kind === 'invoice' && (
        <InvoiceDetailPanel
          invoiceId={navCurrent.id}
          onClose={navClose}
          onBack={navDepth > 1 ? navBack : undefined}
          onEdit={() => {
            const inv = invoices.find((i) => i.id === navCurrent.id)
            if (inv) handleEditInvoice(inv)
            navClose()
          }}
          onRecordPayment={() => {
            setPaymentInvoiceId(navCurrent.id)
            navClose()
          }}
          onStorno={() => {
            const inv = invoices.find((i) => i.id === navCurrent.id)
            if (inv) setCreditNote({ invoice: inv, storno: true })
            navClose()
          }}
        />
      )}

      {navCurrent?.kind === 'quote' && (
        <QuoteDetailPanel
          quoteId={navCurrent.id}
          onClose={navClose}
          onBack={navDepth > 1 ? navBack : undefined}
          onEdit={() => {
            const q = (quotesData?.quotes ?? []).find((x) => x.id === navCurrent.id)
            if (q) {
              setEditQuote(q)
              setShowQuoteForm(true)
            }
            navClose()
          }}
          onConverted={() => {
            navClose()
            setActiveTab('invoices')
          }}
        />
      )}

      {navCurrent?.kind === 'creditNote' && (
        <CreditNoteDetailPanel
          creditNoteId={navCurrent.id}
          onClose={navClose}
          onBack={navDepth > 1 ? navBack : undefined}
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
        open={!!creditNote}
        onOpenChange={(open) => !open && setCreditNote(null)}
        preselectedInvoice={creditNote?.invoice ?? null}
        isStorno={creditNote?.storno ?? false}
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
        title={t('finanzen.confirm.deleteTitle')}
        description={t('finanzen.confirm.deleteDescription', { label: confirmDelete?.label })}
        confirmLabel={t('common.delete')}
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
            <p className="text-sm font-medium text-foreground">{t('finanzen.page.invoiceSentOverlay', { number: sentAnimation })}</p>
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
