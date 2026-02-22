import { useState } from 'react'
import {
  TrendingUp,
  TrendingDown,
  DollarSign,
  CreditCard,
  FileText,
  ArrowUpRight,
  ArrowDownRight,
  Search,
  Download,
  Plus,
  CheckCircle2,
  Clock,
  AlertCircle,
  AlertTriangle,
  BarChart3,
  PieChart,
  Receipt,
  XCircle,
  Gavel,
  Send,
  ChevronRight,
  Link2,
  Landmark,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useFinanceStore,
  type Invoice,
  type Dunning,
  calcInvoiceTotal,
  calcRemainingAmount,
} from '@/stores/finance'
import { ItemActions, ConfirmDialog, EmptyState, PageHeader } from '@/components/shared'
import { formatCurrency } from '@/lib/format'
import { InvoiceFormDialog } from './InvoiceFormDialog'
import { ExpenseFormDialog } from './ExpenseFormDialog'
import { PaymentRecordDialog } from './PaymentRecordDialog'
import { InvoiceDetailPanel } from './InvoiceDetailPanel'
import { ExportDialog } from './ExportDialog'
import { BelegketteTab } from '@/modules/finanzen/BelegketteTab'
import { BankingWidget } from '@/modules/finanzen/BankingWidget'

type TabKey = 'transactions' | 'invoices' | 'quotes' | 'expenses' | 'mahnungen' | 'reports' | 'belegkette' | 'banking'

const invoiceStatusConfig: Record<string, { label: string; colors: string; icon: typeof CheckCircle2 }> = {
  draft: { label: 'Entwurf', colors: 'bg-secondary text-muted-foreground', icon: FileText },
  sent: { label: 'Gesendet', colors: 'bg-info-light text-info', icon: Clock },
  paid: { label: 'Bezahlt', colors: 'bg-success-light text-success', icon: CheckCircle2 },
  overdue: { label: 'Überfällig', colors: 'bg-error-light text-error', icon: AlertCircle },
  cancelled: { label: 'Storniert', colors: 'bg-secondary text-muted-foreground', icon: XCircle },
}

export default function BuchhaltungPage() {
  const {
    invoices, transactions, expenses, dunnings,
    deleteInvoice, duplicateInvoice, sendInvoice, cancelInvoice,
    deleteTransaction, deleteExpense, approveExpense, rejectExpense,
    sendDunning, escalateDunning,
  } = useFinanceStore()

  const [tab, setTab] = useState<TabKey>('invoices')
  const [search, setSearch] = useState('')

  // Dialogs
  const [showInvoiceForm, setShowInvoiceForm] = useState(false)
  const [editInvoice, setEditInvoice] = useState<Invoice | null>(null)
  const [invoiceFormType, setInvoiceFormType] = useState<'invoice' | 'quote'>('invoice')
  const [showExpenseForm, setShowExpenseForm] = useState(false)
  const [paymentInvoice, setPaymentInvoice] = useState<Invoice | null>(null)
  const [selectedInvoice, setSelectedInvoice] = useState<Invoice | null>(null)
  const [showExport, setShowExport] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState<{ type: string; id: string; label: string } | null>(null)
  const [dunningLevelFilter, setDunningLevelFilter] = useState<'all' | 1 | 2 | 3>('all')
  const [dunningStatusFilter, setDunningStatusFilter] = useState<'all' | 'pending' | 'sent' | 'inkasso'>('all')

  // Stats from actual data
  const totalIncome = transactions.filter((t) => t.type === 'income').reduce((sum, t) => sum + t.amount, 0)
  const totalExpense = transactions.filter((t) => t.type === 'expense').reduce((sum, t) => sum + Math.abs(t.amount), 0)
  const balance = totalIncome - totalExpense
  const openInvoiceAmount = invoices
    .filter((i) => i.type === 'invoice' && i.status !== 'paid' && i.status !== 'cancelled')
    .reduce((sum, i) => sum + calcRemainingAmount(i), 0)

  const openDunningCount = dunnings.filter((d) => d.status !== 'paid').length

  const stats = [
    { label: 'Einnahmen', value: formatCurrency(totalIncome), icon: TrendingUp, color: 'text-success', bg: 'bg-success-light' },
    { label: 'Ausgaben', value: formatCurrency(totalExpense), icon: TrendingDown, color: 'text-error', bg: 'bg-error-light' },
    { label: 'Saldo', value: formatCurrency(balance), icon: DollarSign, color: 'text-primary', bg: 'bg-primary-light' },
    { label: 'Offene Rechnungen', value: formatCurrency(openInvoiceAmount), icon: CreditCard, color: 'text-warning', bg: 'bg-warning-light' },
    { label: 'Offene Mahnungen', value: String(openDunningCount), icon: AlertTriangle, color: 'text-error', bg: 'bg-error-light' },
  ]

  // Filtered data
  const filteredInvoices = invoices.filter((i) => i.type === 'invoice').filter((i) => {
    if (!search) return true
    const q = search.toLowerCase()
    return i.client.toLowerCase().includes(q) || i.number.toLowerCase().includes(q)
  })
  const filteredQuotes = invoices.filter((i) => i.type === 'quote').filter((i) => {
    if (!search) return true
    const q = search.toLowerCase()
    return i.client.toLowerCase().includes(q) || i.number.toLowerCase().includes(q)
  })
  const filteredTransactions = transactions.filter((t) => {
    if (!search) return true
    const q = search.toLowerCase()
    return t.description.toLowerCase().includes(q) || t.category.toLowerCase().includes(q)
  })
  const filteredExpenses = expenses.filter((e) => {
    if (!search) return true
    const q = search.toLowerCase()
    return e.description.toLowerCase().includes(q) || e.supplier.toLowerCase().includes(q)
  })

  // Handlers
  const handleNewInvoice = () => {
    setEditInvoice(null)
    setInvoiceFormType('invoice')
    setShowInvoiceForm(true)
  }

  const handleNewQuote = () => {
    setEditInvoice(null)
    setInvoiceFormType('quote')
    setShowInvoiceForm(true)
  }

  const handleEditInvoice = (inv: Invoice) => {
    setEditInvoice(inv)
    setInvoiceFormType(inv.type)
    setShowInvoiceForm(true)
  }

  const handleDeleteConfirm = () => {
    if (!confirmDelete) return
    if (confirmDelete.type === 'invoice') {
      deleteInvoice(confirmDelete.id)
    } else if (confirmDelete.type === 'transaction') {
      deleteTransaction(confirmDelete.id)
    } else if (confirmDelete.type === 'expense') {
      deleteExpense(confirmDelete.id)
    }
    toast.success(`${confirmDelete.label} gelöscht`)
    setConfirmDelete(null)
  }

  const getInvoiceActions = (inv: Invoice) => {
    const actions: { label: string; onClick: () => void; variant?: 'destructive'; separator?: true }[] = [
      { label: 'Details ansehen', onClick: () => setSelectedInvoice(inv) },
    ]
    if (inv.status === 'draft') {
      actions.push({ label: 'Bearbeiten', onClick: () => handleEditInvoice(inv) })
      actions.push({ label: 'Senden', onClick: () => { sendInvoice(inv.id); toast.success(`${inv.number} gesendet`) } })
    }
    actions.push({ label: 'Duplizieren', onClick: () => { duplicateInvoice(inv.id); toast.success('Kopie erstellt') } })
    if (inv.type === 'invoice' && inv.status !== 'paid' && inv.status !== 'cancelled') {
      actions.push({ label: 'Zahlung erfassen', onClick: () => setPaymentInvoice(inv) })
    }
    if (inv.status !== 'cancelled' && inv.status !== 'paid') {
      actions.push({ separator: true as const, label: '', onClick: () => {} })
      actions.push({ label: 'Stornieren', variant: 'destructive' as const, onClick: () => { cancelInvoice(inv.id); toast.success(`${inv.number} storniert`) } })
    }
    if (inv.status === 'draft') {
      actions.push({ label: 'Löschen', variant: 'destructive' as const, onClick: () => setConfirmDelete({ type: 'invoice', id: inv.id, label: inv.number }) })
    }
    return actions
  }

  return (
    <div className="flex-1 overflow-y-auto p-6 animate-fade-up">
      <PageHeader
        title="Buchhaltung"
        description="Rechnungen, Transaktionen und Ausgaben"
        gradient
        className="mb-6"
        actions={
          <div className="flex gap-2">
            <button
              onClick={() => setShowExport(true)}
              className="flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <Download className="h-4 w-4" />
              Export
            </button>
            <button
              onClick={handleNewInvoice}
              className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-4 w-4" />
              Neue Rechnung
            </button>
          </div>
        }
      />

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
        {stats.map((stat) => {
          const Icon = stat.icon
          return (
            <div key={stat.label} className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
              <div className="flex items-center justify-between mb-2">
                <p className="text-xs text-muted-foreground">{stat.label}</p>
                <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${stat.bg}`}>
                  <Icon className={`h-4 w-4 ${stat.color}`} />
                </div>
              </div>
              <p className={`text-xl font-semibold ${stat.color}`}>{stat.value}</p>
            </div>
          )
        })}
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6 overflow-x-auto">
        {([
          { key: 'invoices' as const, label: `Rechnungen (${invoices.filter((i) => i.type === 'invoice').length})`, icon: FileText },
          { key: 'quotes' as const, label: `Angebote (${invoices.filter((i) => i.type === 'quote').length})`, icon: FileText },
          { key: 'expenses' as const, label: `Ausgaben (${expenses.length})`, icon: Receipt },
          { key: 'mahnungen' as const, label: `Mahnungen (${dunnings.length})`, icon: Gavel },
          { key: 'transactions' as const, label: `Transaktionen (${transactions.length})`, icon: Receipt },
          { key: 'reports' as const, label: 'Berichte', icon: BarChart3 },
          { key: 'belegkette' as const, label: 'Belegkette', icon: Link2 },
          { key: 'banking' as const, label: 'Banking', icon: Landmark },
        ]).map((t) => {
          const Icon = t.icon
          return (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm whitespace-nowrap transition-colors ${
                tab === t.key ? 'border-primary text-primary font-medium tab-accent-active' : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Icon className="h-4 w-4" />
              {t.label}
            </button>
          )
        })}
      </div>

      {/* Search (for non-reports tabs) */}
      {tab !== 'reports' && tab !== 'mahnungen' && (
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
          {tab === 'expenses' && (
            <button onClick={() => setShowExpenseForm(true)} className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors">
              <Plus className="h-3.5 w-3.5" />
              Neue Ausgabe
            </button>
          )}
          {tab === 'quotes' && (
            <button onClick={handleNewQuote} className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors">
              <Plus className="h-3.5 w-3.5" />
              Neues Angebot
            </button>
          )}
        </div>
      )}

      {/* Invoices Tab */}
      {tab === 'invoices' && (
        filteredInvoices.length === 0 ? (
          <EmptyState icon={FileText} title="Keine Rechnungen" description="Erstelle deine erste Rechnung" actionLabel="Neue Rechnung" onAction={handleNewInvoice} />
        ) : (
          <div className="rounded-lg border border-border bg-card overflow-hidden">
            <div className="grid grid-cols-[80px_1fr_100px_100px_100px_80px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
              <span>Nr.</span><span>Kunde</span><span>Betrag</span><span>Fällig</span><span>Offen</span><span>Status</span><span />
            </div>
            {filteredInvoices.map((inv) => {
              const sc = invoiceStatusConfig[inv.status]
              const StatusIcon = sc.icon
              const total = calcInvoiceTotal(inv.items)
              const remaining = calcRemainingAmount(inv)
              return (
                <div key={inv.id} className="grid grid-cols-[80px_1fr_100px_100px_100px_80px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors">
                  <button onClick={() => setSelectedInvoice(inv)} className="text-sm font-mono text-primary hover:underline text-left">{inv.number}</button>
                  <span className="text-sm text-foreground truncate">{inv.client}</span>
                  <span className="text-sm font-medium text-foreground">{formatCurrency(total)}</span>
                  <span className="text-xs text-muted-foreground">{new Date(inv.dueDate).toLocaleDateString('de-CH')}</span>
                  <span className={`text-xs font-medium ${remaining > 0 ? 'text-warning' : 'text-success'}`}>
                    {remaining > 0 ? formatCurrency(remaining) : '—'}
                  </span>
                  <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.colors}`}>
                    <StatusIcon className="h-3 w-3" />
                    {sc.label}
                  </span>
                  <ItemActions items={getInvoiceActions(inv)} />
                </div>
              )
            })}
          </div>
        )
      )}

      {/* Quotes Tab */}
      {tab === 'quotes' && (
        filteredQuotes.length === 0 ? (
          <EmptyState icon={FileText} title="Keine Angebote" description="Erstelle dein erstes Angebot" actionLabel="Neues Angebot" onAction={handleNewQuote} />
        ) : (
          <div className="rounded-lg border border-border bg-card overflow-hidden">
            <div className="grid grid-cols-[80px_1fr_100px_100px_80px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
              <span>Nr.</span><span>Kunde</span><span>Betrag</span><span>Gueltig bis</span><span>Status</span><span />
            </div>
            {filteredQuotes.map((inv) => {
              const sc = invoiceStatusConfig[inv.status]
              const StatusIcon = sc.icon
              const total = calcInvoiceTotal(inv.items)
              return (
                <div key={inv.id} className="grid grid-cols-[80px_1fr_100px_100px_80px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors">
                  <button onClick={() => setSelectedInvoice(inv)} className="text-sm font-mono text-primary hover:underline text-left">{inv.number}</button>
                  <span className="text-sm text-foreground truncate">{inv.client}</span>
                  <span className="text-sm font-medium text-foreground">{formatCurrency(total)}</span>
                  <span className="text-xs text-muted-foreground">{new Date(inv.dueDate).toLocaleDateString('de-CH')}</span>
                  <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.colors}`}>
                    <StatusIcon className="h-3 w-3" />
                    {sc.label}
                  </span>
                  <ItemActions items={getInvoiceActions(inv)} />
                </div>
              )
            })}
          </div>
        )
      )}

      {/* Expenses Tab */}
      {tab === 'expenses' && (
        filteredExpenses.length === 0 ? (
          <EmptyState icon={Receipt} title="Keine Ausgaben" description="Erfasse Ausgaben und Belege" actionLabel="Neue Ausgabe" onAction={() => setShowExpenseForm(true)} />
        ) : (
          <div className="rounded-lg border border-border bg-card overflow-hidden">
            <div className="grid grid-cols-[1fr_100px_80px_100px_80px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
              <span>Beschreibung</span><span>Kategorie</span><span>Betrag</span><span>Lieferant</span><span>Status</span><span />
            </div>
            {filteredExpenses.map((exp) => {
              const statusColors = { pending: 'bg-warning-light text-warning', approved: 'bg-success-light text-success', rejected: 'bg-error-light text-error' }
              const statusLabels = { pending: 'Offen', approved: 'OK', rejected: 'Abgelehnt' }
              return (
                <div key={exp.id} className="grid grid-cols-[1fr_100px_80px_100px_80px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors">
                  <div className="min-w-0">
                    <p className="text-sm text-foreground truncate">{exp.description}</p>
                    <p className="text-[10px] text-muted-foreground">{new Date(exp.date).toLocaleDateString('de-CH')}{exp.project && ` · ${exp.project}`}</p>
                  </div>
                  <span className="text-xs text-muted-foreground">{exp.category}</span>
                  <span className="text-sm font-medium text-error">{formatCurrency(exp.amount)}</span>
                  <span className="text-xs text-muted-foreground truncate">{exp.supplier}</span>
                  <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${statusColors[exp.status]}`}>
                    {statusLabels[exp.status]}
                  </span>
                  <ItemActions items={[
                    ...(exp.status === 'pending' ? [
                      { label: 'Genehmigen', onClick: () => { approveExpense(exp.id); toast.success('Genehmigt') } },
                      { label: 'Ablehnen', onClick: () => { rejectExpense(exp.id); toast.success('Abgelehnt') } },
                    ] : []),
                    { label: 'Löschen', variant: 'destructive' as const, onClick: () => setConfirmDelete({ type: 'expense', id: exp.id, label: exp.description }) },
                  ]} />
                </div>
              )
            })}
          </div>
        )
      )}

      {/* Transactions Tab */}
      {tab === 'transactions' && (
        <div className="rounded-lg border border-border bg-card overflow-hidden">
          <div className="grid grid-cols-[1fr_120px_120px_100px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
            <span>Beschreibung</span><span>Kategorie</span><span className="text-right">Betrag</span><span className="text-right">Datum</span><span />
          </div>
          {filteredTransactions.map((tx) => (
            <div key={tx.id} className="grid grid-cols-[1fr_120px_120px_100px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors">
              <div className="flex items-center gap-3 min-w-0">
                <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${tx.type === 'income' ? 'bg-success-light' : 'bg-error-light'}`}>
                  {tx.type === 'income' ? <ArrowUpRight className="h-4 w-4 text-success" /> : <ArrowDownRight className="h-4 w-4 text-error" />}
                </div>
                <div className="min-w-0">
                  <p className="text-sm text-foreground truncate">{tx.description}</p>
                  {tx.reference && <p className="text-[10px] text-muted-foreground">{tx.reference}</p>}
                </div>
              </div>
              <span className="text-xs text-muted-foreground">{tx.category}</span>
              <span className={`text-sm font-medium text-right ${tx.type === 'income' ? 'text-success' : 'text-error'}`}>
                {tx.type === 'income' ? '+' : ''}{formatCurrency(tx.amount)}
              </span>
              <span className="text-xs text-muted-foreground text-right">{new Date(tx.date).toLocaleDateString('de-CH')}</span>
              <ItemActions items={[
                { label: 'Löschen', variant: 'destructive' as const, onClick: () => setConfirmDelete({ type: 'transaction', id: tx.id, label: tx.description }) },
              ]} />
            </div>
          ))}
        </div>
      )}

      {/* Mahnungen Tab */}
      {tab === 'mahnungen' && (() => {
        const filtered = dunnings
          .filter((d) => dunningLevelFilter === 'all' || d.level === dunningLevelFilter)
          .filter((d) => {
            if (dunningStatusFilter === 'all') return true
            return d.status === dunningStatusFilter
          })

        const totalDueAmount = dunnings.filter((d) => d.status !== 'paid').reduce((s, d) => s + d.dueAmount, 0)
        const inkassoCount = dunnings.filter((d) => d.status === 'inkasso').length
        const avgOverdueDays = Math.round(
          dunnings
            .filter((d) => d.status !== 'paid')
            .reduce((s, d) => {
              const sent = new Date(d.sentAt).getTime()
              const now = Date.now()
              return s + Math.max(0, Math.floor((now - sent) / (1000 * 60 * 60 * 24)))
            }, 0) / Math.max(1, dunnings.filter((d) => d.status !== 'paid').length),
        )

        const dunningStats = [
          { label: 'Offene Mahnungen', value: String(openDunningCount) },
          { label: 'Mahnbetrag gesamt', value: formatCurrency(totalDueAmount) },
          { label: 'Inkasso-Fälle', value: String(inkassoCount) },
          { label: 'Ø Verzugstage', value: `${avgOverdueDays} Tage` },
        ]

        const levelColors: Record<number, string> = {
          1: 'bg-warning-light text-warning',
          2: 'bg-orange-100 text-orange-600 dark:bg-orange-900/30 dark:text-orange-400',
          3: 'bg-error-light text-error',
        }

        const statusConfig: Record<string, { label: string; colors: string }> = {
          pending: { label: 'Ausstehend', colors: 'bg-warning-light text-warning' },
          sent: { label: 'Gesendet', colors: 'bg-info-light text-info' },
          reminded: { label: 'Erinnert', colors: 'bg-primary-light text-primary' },
          paid: { label: 'Bezahlt', colors: 'bg-success-light text-success' },
          inkasso: { label: 'Inkasso', colors: 'bg-error-light text-error' },
        }

        return (
          <div className="space-y-4">
            {/* Dunning stats row */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
              {dunningStats.map((ds) => (
                <div key={ds.label} className="rounded-lg border border-border bg-card p-3">
                  <p className="text-[10px] text-muted-foreground mb-1">{ds.label}</p>
                  <p className="text-lg font-semibold text-foreground">{ds.value}</p>
                </div>
              ))}
            </div>

            {/* Filter bar */}
            <div className="flex items-center gap-3 flex-wrap">
              <div className="flex items-center gap-1.5">
                <span className="text-xs text-muted-foreground">Mahnstufe:</span>
                {(['all', 1, 2, 3] as const).map((lvl) => (
                  <button
                    key={String(lvl)}
                    onClick={() => setDunningLevelFilter(lvl)}
                    className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                      dunningLevelFilter === lvl
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-secondary text-muted-foreground hover:text-foreground'
                    }`}
                  >
                    {lvl === 'all' ? 'Alle' : `Stufe ${lvl}`}
                  </button>
                ))}
              </div>
              <div className="h-4 w-px bg-border" />
              <div className="flex items-center gap-1.5">
                <span className="text-xs text-muted-foreground">Status:</span>
                {(['all', 'pending', 'sent', 'inkasso'] as const).map((st) => {
                  const labels: Record<string, string> = { all: 'Alle', pending: 'Ausstehend', sent: 'Gesendet', inkasso: 'Inkasso' }
                  return (
                    <button
                      key={st}
                      onClick={() => setDunningStatusFilter(st)}
                      className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                        dunningStatusFilter === st
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-secondary text-muted-foreground hover:text-foreground'
                      }`}
                    >
                      {labels[st]}
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Dunning table */}
            {filtered.length === 0 ? (
              <EmptyState icon={Gavel} title="Keine Mahnungen" description="Keine Mahnungen mit diesen Filtern gefunden" />
            ) : (
              <div className="rounded-lg border border-border bg-card overflow-hidden">
                <div className="grid grid-cols-[80px_1fr_100px_100px_100px_80px_160px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
                  <span>Rechnung</span><span>Kunde</span><span>Betrag</span><span>Mahnstufe</span><span>Gesendet</span><span>Status</span><span className="text-right">Aktionen</span>
                </div>
                {filtered.map((d) => {
                  const sc = statusConfig[d.status]
                  return (
                    <div key={d.id} className="grid grid-cols-[80px_1fr_100px_100px_100px_80px_160px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors">
                      <span className="text-sm font-mono text-primary">{d.invoiceNumber}</span>
                      <span className="text-sm text-foreground truncate">{d.client}</span>
                      <span className="text-sm font-medium text-foreground">{formatCurrency(d.dueAmount)}</span>
                      {/* Mahnstufe visualization */}
                      <div className="flex flex-col gap-1">
                        <span className={`inline-flex items-center self-start rounded-full px-2 py-0.5 text-[10px] font-medium ${levelColors[d.level]}`}>
                          Stufe {d.level}
                        </span>
                        <div className="flex items-center gap-0.5">
                          {[1, 2, 3].map((step) => (
                            <div key={step} className="flex items-center">
                              <div
                                className={`h-1.5 w-1.5 rounded-full ${
                                  step <= d.level
                                    ? step === 3 ? 'bg-error' : step === 2 ? 'bg-orange-500' : 'bg-warning'
                                    : 'bg-secondary'
                                }`}
                              />
                              {step < 3 && (
                                <ChevronRight className={`h-2.5 w-2.5 ${step < d.level ? 'text-muted-foreground' : 'text-secondary'}`} />
                              )}
                            </div>
                          ))}
                        </div>
                      </div>
                      <span className="text-xs text-muted-foreground">{new Date(d.sentAt).toLocaleDateString('de-CH')}</span>
                      <span className={`inline-flex items-center self-start rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.colors}`}>
                        {sc.label}
                      </span>
                      <div className="flex items-center justify-end gap-1.5">
                        {d.status === 'pending' && (
                          <button
                            onClick={() => { sendDunning(d.id); toast.success(`Mahnung ${d.invoiceNumber} gesendet`) }}
                            className="flex items-center gap-1 rounded-md bg-primary px-2 py-1 text-[10px] text-primary-foreground hover:bg-button-primary-hover transition-colors"
                          >
                            <Send className="h-3 w-3" />
                            Senden
                          </button>
                        )}
                        {d.level < 3 && d.status !== 'paid' && (
                          <button
                            onClick={() => { escalateDunning(d.id); toast.success(`${d.invoiceNumber} eskaliert auf Stufe ${Math.min(d.level + 1, 3)}`) }}
                            className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-[10px] text-foreground hover:bg-secondary transition-colors"
                          >
                            <AlertTriangle className="h-3 w-3" />
                            Eskalieren
                          </button>
                        )}
                        {d.status === 'inkasso' && (
                          <span className="text-[10px] text-error font-medium">Inkasso aktiv</span>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )
      })()}

      {/* Reports Tab */}
      {tab === 'reports' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-2 mb-4">
              <BarChart3 className="h-5 w-5 text-primary" />
              <h3 className="text-sm font-medium text-foreground">Einnahmen vs. Ausgaben</h3>
            </div>
            <div className="flex items-end gap-3 h-40">
              {['Okt', 'Nov', 'Dez', 'Jan', 'Feb'].map((month, i) => {
                const incomeHeight = [55, 70, 80, 65, 90][i]
                const expenseHeight = [35, 40, 45, 30, 25][i]
                return (
                  <div key={month} className="flex-1 flex flex-col items-center gap-1">
                    <div className="flex gap-1 items-end h-32 w-full">
                      <div className="flex-1 rounded-t-sm bg-primary" style={{ height: `${incomeHeight}%` }} />
                      <div className="flex-1 rounded-t-sm bg-error/60" style={{ height: `${expenseHeight}%` }} />
                    </div>
                    <span className="text-[10px] text-muted-foreground">{month}</span>
                  </div>
                )
              })}
            </div>
            <div className="flex items-center gap-4 mt-3 text-xs text-muted-foreground">
              <span className="flex items-center gap-1"><span className="h-2 w-2 rounded-full bg-primary" /> Einnahmen</span>
              <span className="flex items-center gap-1"><span className="h-2 w-2 rounded-full bg-error/60" /> Ausgaben</span>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-2 mb-4">
              <PieChart className="h-5 w-5 text-primary" />
              <h3 className="text-sm font-medium text-foreground">Ausgaben nach Kategorie</h3>
            </div>
            <div className="space-y-3">
              {(() => {
                const cats = new Map<string, number>()
                transactions.filter((t) => t.type === 'expense').forEach((t) => {
                  cats.set(t.category, (cats.get(t.category) ?? 0) + Math.abs(t.amount))
                })
                const total = Array.from(cats.values()).reduce((s, v) => s + v, 0) || 1
                const colors = ['bg-primary', 'bg-info', 'bg-warning', 'bg-error', 'bg-success']
                return Array.from(cats.entries())
                  .sort((a, b) => b[1] - a[1])
                  .slice(0, 5)
                  .map(([name, amount], idx) => (
                    <div key={name}>
                      <div className="flex items-center justify-between text-sm mb-1">
                        <span className="text-foreground">{name}</span>
                        <span className="text-muted-foreground">{formatCurrency(amount)}</span>
                      </div>
                      <div className="h-2 rounded-full bg-secondary">
                        <div className={`h-full rounded-full ${colors[idx % colors.length]}`} style={{ width: `${(amount / total) * 100}%` }} />
                      </div>
                    </div>
                  ))
              })()}
            </div>
          </div>
        </div>
      )}

      {/* Belegkette Tab */}
      {tab === 'belegkette' && <BelegketteTab />}

      {/* Banking Tab */}
      {tab === 'banking' && <BankingWidget />}

      {/* Invoice Detail Panel */}
      {selectedInvoice && (
        <InvoiceDetailPanel
          invoice={selectedInvoice}
          onClose={() => setSelectedInvoice(null)}
          onEdit={() => { handleEditInvoice(selectedInvoice); setSelectedInvoice(null) }}
          onRecordPayment={() => { setPaymentInvoice(selectedInvoice); setSelectedInvoice(null) }}
        />
      )}

      {/* Invoice Form */}
      <InvoiceFormDialog
        open={showInvoiceForm}
        onOpenChange={setShowInvoiceForm}
        editInvoice={editInvoice}
        defaultType={invoiceFormType}
      />

      {/* Expense Form */}
      <ExpenseFormDialog open={showExpenseForm} onOpenChange={setShowExpenseForm} />

      {/* Payment Record */}
      <PaymentRecordDialog
        open={!!paymentInvoice}
        onOpenChange={() => setPaymentInvoice(null)}
        invoice={paymentInvoice}
      />

      {/* Export */}
      <ExportDialog open={showExport} onOpenChange={setShowExport} />

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
    </div>
  )
}
