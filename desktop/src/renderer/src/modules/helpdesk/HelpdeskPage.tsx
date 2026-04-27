import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
  BarChart3,
  AlertCircle,
  Clock,
  CheckCircle2,
  Eye,
  X,
  Send,
  MessageSquare,
  User,
  ChevronDown,
  ArrowLeft,
  Lock,
  Zap,
  Route,
  Settings2,
  Pencil,
  Tag,
  Bot,
  LifeBuoy,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useHelpdeskStore,
  type Ticket as TicketType,
  type KBArticle,
  MOCK_CATEGORIES,
  MOCK_CUSTOM_FIELD_DEFS,
} from '@/stores/helpdesk'
import { SLABadge, SLABreachBanner } from './SLABadge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { CSATWidget, CSATAggregate } from './CSATWidget'
import { CannedResponsesPanel } from './CannedResponsesPanel'
import { CannedResponsePicker } from './CannedResponsePicker'
import { BusinessHoursDialog } from './BusinessHoursDialog'
import { TicketRoutingConfig } from './TicketRoutingConfig'
import { LazyRichTextEditor as RichTextEditor } from '@/components/shared/RichTextEditor'
import { useAIStore } from '@/stores/ai'
import { PageHeader, EmptyState } from '@/components/shared'
import { EmptyHelpdesk } from '@/components/shared/illustrations'
import { formatDate } from '@/lib/format'

type TabKey = 'tickets' | 'wissensdatenbank' | 'statistik'
type StatusFilter = 'all' | TicketType['status']
type PriorityFilter = 'all' | TicketType['priority']
type CategoryFilter = 'all' | string

// ---------------------------------------------------------------------------
// Label / Color Maps
// ---------------------------------------------------------------------------

const priorityColors: Record<string, string> = {
  low: 'bg-secondary text-muted-foreground',
  medium: 'bg-info-light text-info',
  high: 'bg-warning-light text-warning',
  critical: 'bg-error-light text-error',
}

// priorityLabels moved inside components as usePriorityLabels()

const statusColors: Record<string, string> = {
  open: 'bg-warning-light text-warning',
  in_progress: 'bg-primary-light text-primary',
  waiting: 'bg-info-light text-info',
  resolved: 'bg-success-light text-success',
  closed: 'bg-secondary text-muted-foreground',
}

// statusLabels moved inside components as useStatusLabels()

const kbCategoryColors: Record<string, string> = {
  Netzwerk: 'bg-info-light text-info',
  Hardware: 'bg-warning-light text-warning',
  Sicherheit: 'bg-error-light text-error',
  'E-Mail': 'bg-primary-light text-primary',
  Allgemein: 'bg-secondary text-muted-foreground',
}

const categoryColors: Record<string, string> = {
  Netzwerk: 'bg-info-light text-info',
  Hardware: 'bg-warning-light text-warning',
  Software: 'bg-primary-light text-primary',
  Zugang: 'bg-success-light text-success',
  'E-Mail': 'bg-primary-light text-primary',
  Telefonie: 'bg-secondary text-muted-foreground',
  Sicherheit: 'bg-error-light text-error',
  Sonstiges: 'bg-secondary text-muted-foreground',
}

// ---------------------------------------------------------------------------
// Mock message threads per ticket
// ---------------------------------------------------------------------------

interface ThreadMessage {
  id: string
  author: string
  role: 'customer' | 'agent'
  text: string
  timestamp: string
  isInternal?: boolean
}

const MOCK_THREADS: Record<string, ThreadMessage[]> = {
  'tk-1': [
    { id: 'm1', author: 'Brigitte Schaerer', role: 'customer', text: 'Der Drucker im 2. OG zeigt seit heute Morgen "Offline" an. Mehrere Kollegen haben das gleiche Problem.', timestamp: '2026-02-15T08:30:00' },
    { id: 'm2', author: 'Marco Hartmann', role: 'agent', text: 'Danke für die Meldung. Ich prüfe den Druckserver und die Netzwerkverbindung. Bitte kurz Geduld.', timestamp: '2026-02-15T09:15:00' },
    { id: 'm3', author: 'Marco Hartmann', role: 'agent', text: 'Der Druckspooler-Dienst wurde neu gestartet. Bitte testen Sie erneut.', timestamp: '2026-02-15T09:45:00' },
  ],
  'tk-2': [
    { id: 'm1', author: 'Stefan Wenger', role: 'customer', text: 'Meine VPN-Verbindung trennt sich alle 10 Minuten. Arbeiten im Home-Office ist so nicht möglich.', timestamp: '2026-02-14T17:45:00' },
    { id: 'm2', author: 'Marco Hartmann', role: 'agent', text: 'Welchen VPN-Client nutzen Sie? Bitte senden Sie mir die Logdatei.', timestamp: '2026-02-15T08:00:00' },
    { id: 'm3', author: 'Stefan Wenger', role: 'customer', text: 'Cisco AnyConnect 4.10. Logs sind im Anhang.', timestamp: '2026-02-15T08:30:00' },
  ],
  'tk-3': [
    { id: 'm1', author: 'Karin Pfister', role: 'customer', text: 'Neuer Mitarbeiter Lukas Meier startet am 01.03. in der Buchhaltung. Bitte alle Standardzugänge einrichten.', timestamp: '2026-02-14T14:00:00' },
    { id: 'm2', author: 'Sandra Buerki', role: 'agent', text: 'Wird vorbereitet. AD-Konto, E-Mail, ERP und Zeiterfassung.', timestamp: '2026-02-14T15:00:00', isInternal: true },
  ],
}

function getThread(ticketId: string): ThreadMessage[] {
  return MOCK_THREADS[ticketId] ?? [
    { id: 'default-1', author: 'System', role: 'agent', text: 'Ticket wurde erstellt und dem zustaendigen Techniker zugewiesen.', timestamp: new Date().toISOString() },
  ]
}

// ---------------------------------------------------------------------------
// KB Article bodies
// ---------------------------------------------------------------------------

const KB_BODIES: Record<string, string> = {
  'kb-1': `Schritt 1: Laden Sie den Cisco AnyConnect Client von unserem Self-Service Portal herunter.\n\nSchritt 2: Installieren Sie den Client und starten Sie ihn. Geben Sie als Server-Adresse "vpn.firma.de" ein.\n\nSchritt 3: Melden Sie sich mit Ihren Active-Directory Zugangsdaten an (gleiche wie Windows-Login). Bei der ersten Verbindung müssen Sie das Zertifikat akzeptieren.\n\nBei Problemen kontaktieren Sie bitte den Helpdesk unter Ticket-Kategorie "Netzwerk".`,
  'kb-2': `Netzwerkdrucker unter Windows hinzufügen:\n\n1. Öffnen Sie die Windows-Einstellungen → Geräte → Drucker und Scanner.\n2. Klicken Sie auf "Drucker oder Scanner hinzufügen".\n3. Falls der Drucker nicht automatisch gefunden wird, klicken Sie auf "Der gewünschte Drucker ist nicht aufgelistet".\n4. Wählen Sie "Freigegebenen Drucker über den Namen auswählen" und geben Sie den Pfad ein (z.B. \\\\printserver\\HP-2OG).\n\nTreiber werden automatisch installiert. Bei macOS verwenden Sie das Druckdienstprogramm unter Systemeinstellungen.`,
  'kb-3': `Passwort-Reset über Self-Service Portal:\n\nBesuchen Sie https://password.firma.de und melden Sie sich mit Ihrem Benutzernamen an. Sie erhalten einen Bestätigungscode per SMS.\n\nGeben Sie den Code ein und setzen Sie ein neues Passwort. Das Passwort muss mindestens 12 Zeichen lang sein.\n\nNach dem Reset müssen Sie sich auf allen Geräten neu anmelden.`,
  'kb-4': `E-Mail Signatur einrichten:\n\nVerwenden Sie die offizielle Vorlage aus dem Intranet unter "Vorlagen → E-Mail Signatur". Kopieren Sie den HTML-Code und fügen Sie ihn in Outlook unter Datei → Optionen → E-Mail → Signaturen ein.\n\nBitte achten Sie auf die korrekte Schreibweise.`,
  'kb-5': `Home-Office IT-Checkliste:\n\n- VPN-Zugang eingerichtet und getestet\n- Softphone oder Rufweiterleitung konfiguriert\n- Laptop mit aktuellem Betriebssystem und Virenscanner\n- Stabile Internetverbindung (mind. 20 Mbit/s empfohlen)\n- Bildschirmsperre aktiviert (max. 5 Min. Timeout)\n- Keine vertraulichen Dokumente ausdrucken\n\nBei Fragen wenden Sie sich an den Helpdesk.`,
}

// ============================================================
// Main Component
// ============================================================

export default function HelpdeskPage() {
  const { t } = useTranslation()
  const { tickets, kbArticles, stats } = useHelpdeskStore()

  const priorityLabels: Record<string, string> = {
    low: t('helpdesk.priority.low'), medium: t('helpdesk.priority.medium'),
    high: t('helpdesk.priority.high'), critical: t('helpdesk.priority.critical'),
  }
  const statusLabels: Record<string, string> = {
    open: t('helpdesk.status.open'), in_progress: t('helpdesk.status.inProgress'),
    waiting: t('helpdesk.status.waiting'), resolved: t('helpdesk.status.resolved'),
    closed: t('helpdesk.status.closed'),
  }

  // Tab & filters
  const [tab, setTab] = useState<TabKey>('tickets')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [priorityFilter, setPriorityFilter] = useState<PriorityFilter>('all')
  const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('all')

  // Detail panel
  const [selectedTicketId, setSelectedTicketId] = useState<string | null>(null)
  const [replyText, setReplyText] = useState('')
  const [showInternalNotes, setShowInternalNotes] = useState(false)

  // New ticket dialog
  const [newTicketOpen, setNewTicketOpen] = useState(false)
  const [ntSubject, setNtSubject] = useState('')
  const [ntDescription, setNtDescription] = useState('')
  const [ntPriority, setNtPriority] = useState<TicketType['priority']>('medium')
  const [ntAssignee, setNtAssignee] = useState('Marco Hartmann')
  const [ntContact, setNtContact] = useState('')
  const [ntCategory, setNtCategory] = useState<string>('Sonstiges')

  // KB article detail
  const [selectedArticleId, setSelectedArticleId] = useState<string | null>(null)

  // Dialogs (5.6, 5.8, 5.11)
  const [cannedResponsesOpen, setCannedResponsesOpen] = useState(false)
  const [businessHoursOpen, setBusinessHoursOpen] = useState(false)
  const [routingConfigOpen, setRoutingConfigOpen] = useState(false)

  // Computed
  const openTickets = tickets.filter((t) => t.status !== 'closed' && t.status !== 'resolved')

  const filteredTickets = useMemo(() => {
    return tickets.filter((t) => {
      if (statusFilter !== 'all' && t.status !== statusFilter) return false
      if (priorityFilter !== 'all' && t.priority !== priorityFilter) return false
      if (categoryFilter !== 'all' && t.category !== categoryFilter) return false
      if (search) {
        const q = search.toLowerCase()
        return (
          t.ticketNr.toLowerCase().includes(q) ||
          t.subject.toLowerCase().includes(q) ||
          t.assignedTo.toLowerCase().includes(q) ||
          (t.category ?? '').toLowerCase().includes(q)
        )
      }
      return true
    })
  }, [tickets, statusFilter, priorityFilter, categoryFilter, search])

  const selectedTicket = tickets.find((t) => t.id === selectedTicketId) ?? null
  const selectedArticle = kbArticles.find((a) => a.id === selectedArticleId) ?? null

  // Handlers
  const handleOpenNewTicket = () => {
    setNtSubject(''); setNtDescription(''); setNtPriority('medium')
    setNtAssignee('Marco Hartmann'); setNtContact(''); setNtCategory('Sonstiges')
    setNewTicketOpen(true)
  }

  const handleSaveNewTicket = () => {
    if (!ntSubject.trim()) { toast.error(t('helpdesk.newTicket.subjectRequired')); return }
    toast.success(t('helpdesk.newTicket.created', { subject: ntSubject }))
    setNewTicketOpen(false)
  }

  const handleSendReply = () => {
    if (!replyText.trim()) return
    toast.success(showInternalNotes ? t('helpdesk.ticket.internalNoteSaved') : t('helpdesk.ticket.replySent'))
    setReplyText('')
  }

  const handleStatusChange = (newStatus: TicketType['status']) => {
    if (selectedTicket) toast.info(t('helpdesk.ticket.statusChanged', { status: statusLabels[newStatus] }))
  }

  const handleTicketRowClick = (id: string) => {
    setSelectedTicketId(id); setReplyText(''); setShowInternalNotes(false)
  }

  const hasActiveFilters = statusFilter !== 'all' || priorityFilter !== 'all' || categoryFilter !== 'all'

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <PageHeader
        title="Helpdesk"
        description={t('helpdesk.header.description', { openCount: openTickets.length, articleCount: kbArticles.length })}
        icon={LifeBuoy}
        moduleId="helpdesk"
        actions={
          <div className="flex items-center gap-2">
            <button
              onClick={() => setBusinessHoursOpen(true)}
              className="flex items-center gap-1.5 rounded-xl border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              <Clock className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">{t('helpdesk.header.businessHours')}</span>
            </button>
            <button
              onClick={() => setRoutingConfigOpen(true)}
              className="flex items-center gap-1.5 rounded-xl border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              <Route className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">Routing</span>
            </button>
            <button
              onClick={() => setCannedResponsesOpen(true)}
              className="flex items-center gap-1.5 rounded-xl border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              <Zap className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">{t('helpdesk.header.cannedResponses')}</span>
            </button>
            <button
              onClick={handleOpenNewTicket}
              className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-4 w-4" />
              {t('helpdesk.header.newTicket')}
            </button>
          </div>
        }
        className="mb-6"
      />

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'tickets' as const, label: t('helpdesk.tabs.tickets', { count: openTickets.length }) },
          { key: 'wissensdatenbank' as const, label: t('helpdesk.tabs.knowledgeBase') },
          { key: 'statistik' as const, label: t('helpdesk.tabs.statistics') },
        ]).map((t) => (
          <button
            key={t.key}
            onClick={() => { setTab(t.key); setSelectedTicketId(null); setSelectedArticleId(null) }}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key ? 'border-primary text-primary font-medium tab-accent-active' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* ================================================================== */}
      {/* TICKETS TAB                                                         */}
      {/* ================================================================== */}
      {tab === 'tickets' && (
        <div className="animate-fade-up">
          {/* Filters row */}
          <div className="flex flex-wrap items-center gap-3 mb-4">
            <div className="relative flex-1 min-w-[200px] max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder={t('helpdesk.filter.searchTicket')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>

            {/* Status */}
            <div className="relative">
              <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value as StatusFilter)} className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer">
                <option value="all">{t('helpdesk.filter.allStatuses')}</option>
                <option value="open">{t('helpdesk.status.open')}</option>
                <option value="in_progress">{t('helpdesk.status.inProgress')}</option>
                <option value="waiting">{t('helpdesk.status.waiting')}</option>
                <option value="resolved">{t('helpdesk.status.resolved')}</option>
                <option value="closed">{t('helpdesk.status.closed')}</option>
              </select>
              <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            </div>

            {/* Priority */}
            <div className="relative">
              <select value={priorityFilter} onChange={(e) => setPriorityFilter(e.target.value as PriorityFilter)} className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer">
                <option value="all">{t('helpdesk.filter.allPriorities')}</option>
                <option value="critical">{t('helpdesk.priority.critical')}</option>
                <option value="high">{t('helpdesk.priority.high')}</option>
                <option value="medium">{t('helpdesk.priority.medium')}</option>
                <option value="low">{t('helpdesk.priority.low')}</option>
              </select>
              <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            </div>

            {/* Category (5.10) */}
            <div className="relative">
              <select value={categoryFilter} onChange={(e) => setCategoryFilter(e.target.value)} className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer">
                <option value="all">{t('helpdesk.filter.allCategories')}</option>
                {MOCK_CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
              </select>
              <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            </div>

            {hasActiveFilters && (
              <button
                onClick={() => { setStatusFilter('all'); setPriorityFilter('all'); setCategoryFilter('all') }}
                className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.resetFilters')}
              </button>
            )}
          </div>

          {/* Ticket table */}
          <div className="rounded-lg border border-border bg-card overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.table.ticketNr')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.table.subject')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.table.category')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.table.priority')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('common.status')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.table.assignedTo')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">SLA</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.table.createdAt')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredTickets.map((ticket) => (
                    <tr
                      key={ticket.id}
                      onClick={() => handleTicketRowClick(ticket.id)}
                      className={`border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer ${
                        selectedTicketId === ticket.id ? 'bg-secondary/70' : ''
                      }`}
                    >
                      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{ticket.ticketNr}</td>
                      <td className="px-4 py-3 text-foreground font-medium max-w-[250px] truncate">
                        {ticket.autoRouted && <Route className="inline h-3 w-3 text-primary mr-1" />}
                        {ticket.subject}
                      </td>
                      <td className="px-4 py-3">
                        {ticket.category && (
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${categoryColors[ticket.category] ?? 'bg-secondary text-muted-foreground'}`}>
                            {ticket.category}
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${priorityColors[ticket.priority]}`}>
                          {priorityLabels[ticket.priority]}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${statusColors[ticket.status]}`}>
                          {statusLabels[ticket.status]}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{ticket.assignedTo}</td>
                      <td className="px-4 py-3">
                        <SLABadge overdue={ticket.slaOverdue} remaining={ticket.slaRemaining} dueAt={ticket.slaDueAt} compact />
                      </td>
                      <td className="px-4 py-3 text-xs text-muted-foreground">
                        {formatDate(ticket.createdAt)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {filteredTickets.length === 0 && (
              <EmptyState
                illustration={<EmptyHelpdesk />}
                title={t('helpdesk.empty.noTickets')}
                description={hasActiveFilters || search ? t('helpdesk.empty.adjustFilters') : t('helpdesk.empty.createTicket')}
                action={!hasActiveFilters && !search ? { label: t('helpdesk.empty.createFirstTicket'), onClick: handleOpenNewTicket } : undefined}
              />
            )}
          </div>
        </div>
      )}

      {/* ================================================================== */}
      {/* WISSENSDATENBANK TAB                                                */}
      {/* ================================================================== */}
      {tab === 'wissensdatenbank' && (
        <div className="animate-fade-up">
          {selectedArticle ? (
            <KBArticleDetail article={selectedArticle} onBack={() => setSelectedArticleId(null)} />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {kbArticles.length === 0 ? (
                <div className="col-span-full">
                  <EmptyState
                    illustration={<EmptyHelpdesk />}
                    title={t('helpdesk.kb.noArticles')}
                  />
                </div>
              ) : (
                kbArticles.map((article) => (
                  <button type="button" key={article.id} onClick={() => setSelectedArticleId(article.id)} className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer text-left w-full">
                    <div className="flex items-start justify-between mb-2">
                      <h4 className="text-sm font-medium text-foreground line-clamp-2">{article.title}</h4>
                      {article.published ? (
                        <span className="rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium shrink-0 ml-2">{t('helpdesk.kb.published')}</span>
                      ) : (
                        <span className="rounded-full bg-secondary text-muted-foreground px-2 py-0.5 text-[10px] font-medium shrink-0 ml-2">{t('helpdesk.kb.draft')}</span>
                      )}
                    </div>
                    <span className={`inline-block rounded-full px-2 py-0.5 text-[10px] font-medium mb-2 ${kbCategoryColors[article.category] ?? 'bg-secondary text-muted-foreground'}`}>{article.category}</span>
                    <p className="text-xs text-muted-foreground line-clamp-3 mb-3">{article.excerpt}</p>
                    <div className="flex items-center gap-3 text-xs text-muted-foreground">
                      <span className="flex items-center gap-1"><Eye className="h-3 w-3" />{article.views}</span>
                      <span>{formatDate(article.updatedAt)}</span>
                    </div>
                  </button>
                ))
              )}
            </div>
          )}
        </div>
      )}

      {/* ================================================================== */}
      {/* STATISTIK TAB                                                       */}
      {/* ================================================================== */}
      {tab === 'statistik' && (
        <div className="animate-fade-up">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
            <StatCard icon={AlertCircle} label={t('helpdesk.stats.openTickets')} value={stats.openTickets} iconColor="text-warning" iconBg="bg-warning-light" />
            <StatCard icon={Clock} label={t('helpdesk.stats.avgResponseTime')} value={stats.avgResponseTime} iconColor="text-info" iconBg="bg-info-light" />
            <StatCard icon={CheckCircle2} label={t('helpdesk.stats.resolvedThisWeek')} value={stats.resolvedThisWeek} iconColor="text-success" iconBg="bg-success-light" />
            <StatCard icon={BarChart3} label={t('helpdesk.stats.customerSatisfaction')} value={stats.customerSatisfaction} iconColor="text-primary" iconBg="bg-primary-light" />
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
            {/* Bar Chart */}
            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="text-sm font-medium text-foreground mb-4">{t('helpdesk.stats.ticketsPerDay')}</h3>
              <div className="flex items-end gap-3 h-40">
                {stats.weeklyBreakdown.map((day) => {
                  const maxCount = Math.max(...stats.weeklyBreakdown.map((d) => d.count), 1)
                  return (
                    <div key={day.label} className="flex-1 flex flex-col items-center gap-1">
                      <span className="text-xs text-muted-foreground">{day.count}</span>
                      <div className="w-full rounded-t bg-primary/80 transition-all" style={{ height: `${Math.max(8, (day.count / maxCount) * 100)}%` }} />
                      <span className="text-[10px] text-muted-foreground">{day.label}</span>
                    </div>
                  )
                })}
              </div>
            </div>

            {/* CSAT Aggregate (5.12) */}
            <CSATAggregate />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="rounded-lg border border-border bg-card p-4">
              <h3 className="text-sm font-medium text-foreground mb-3">{t('helpdesk.stats.byStatus')}</h3>
              <div className="space-y-2">
                {(['open', 'in_progress', 'waiting', 'resolved', 'closed'] as const).map((s) => {
                  const count = tickets.filter((t) => t.status === s).length
                  const pct = tickets.length > 0 ? Math.round((count / tickets.length) * 100) : 0
                  return (
                    <div key={s} className="flex items-center gap-3">
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium w-28 text-center ${statusColors[s]}`}>{statusLabels[s]}</span>
                      <div className="flex-1 h-2 rounded-full bg-secondary overflow-hidden">
                        <div className="h-full rounded-full bg-primary/70 transition-all" style={{ width: `${pct}%` }} />
                      </div>
                      <span className="text-xs text-muted-foreground w-10 text-right">{count}</span>
                    </div>
                  )
                })}
              </div>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <h3 className="text-sm font-medium text-foreground mb-3">{t('helpdesk.stats.byPriority')}</h3>
              <div className="space-y-2">
                {(['critical', 'high', 'medium', 'low'] as const).map((p) => {
                  const count = tickets.filter((t) => t.priority === p).length
                  const pct = tickets.length > 0 ? Math.round((count / tickets.length) * 100) : 0
                  return (
                    <div key={p} className="flex items-center gap-3">
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium w-28 text-center ${priorityColors[p]}`}>{priorityLabels[p]}</span>
                      <div className="flex-1 h-2 rounded-full bg-secondary overflow-hidden">
                        <div className="h-full rounded-full bg-primary/70 transition-all" style={{ width: `${pct}%` }} />
                      </div>
                      <span className="text-xs text-muted-foreground w-10 text-right">{count}</span>
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ================================================================== */}
      {/* TICKET DETAIL PANEL                                                 */}
      {/* ================================================================== */}
      {selectedTicket && (
        <TicketDetailPanel
          ticket={selectedTicket}
          replyText={replyText}
          onReplyChange={setReplyText}
          showInternalNotes={showInternalNotes}
          onToggleInternal={setShowInternalNotes}
          onSendReply={handleSendReply}
          onStatusChange={handleStatusChange}
          onClose={() => setSelectedTicketId(null)}
        />
      )}

      {/* ================================================================== */}
      {/* NEW TICKET DIALOG                                                   */}
      {/* ================================================================== */}
      <Dialog open={newTicketOpen} onOpenChange={setNewTicketOpen}>
        <DialogContent className="max-w-lg gap-0 p-0">
          <DialogHeader className="px-6 py-4 border-b border-border">
            <DialogTitle>{t('helpdesk.newTicket.title')}</DialogTitle>
          </DialogHeader>
            <div className="space-y-4 px-6 py-5">
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">{t('helpdesk.newTicket.subject')}</label>
                <input type="text" value={ntSubject} onChange={(e) => setNtSubject(e.target.value)} placeholder={t('helpdesk.newTicket.subjectPlaceholder')} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring" />
              </div>
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">{t('helpdesk.newTicket.description')}</label>
                <textarea value={ntDescription} onChange={(e) => setNtDescription(e.target.value)} placeholder={t('helpdesk.newTicket.descriptionPlaceholder')} rows={4} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none" />
              </div>
              {/* Category (5.10) */}
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">{t('helpdesk.newTicket.category')}</label>
                <div className="relative">
                  <select value={ntCategory} onChange={(e) => setNtCategory(e.target.value)} className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer">
                    {MOCK_CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                </div>
              </div>
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">{t('helpdesk.newTicket.priorityLabel')}</label>
                <div className="flex gap-2">
                  {(['low', 'medium', 'high', 'critical'] as const).map((p) => (
                    <label key={p} className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs cursor-pointer transition-colors ${ntPriority === p ? 'border-primary bg-primary/10 text-primary font-medium' : 'border-border text-muted-foreground hover:bg-secondary'}`}>
                      <input type="radio" name="priority" value={p} checked={ntPriority === p} onChange={() => setNtPriority(p)} className="sr-only" />
                      {priorityLabels[p]}
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">{t('helpdesk.newTicket.assignTo')}</label>
                <div className="relative">
                  <select value={ntAssignee} onChange={(e) => setNtAssignee(e.target.value)} className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer">
                    <option value="Marco Hartmann">Marco Hartmann</option>
                    <option value="Sandra Buerki">Sandra Buerki</option>
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                </div>
              </div>
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">{t('helpdesk.newTicket.contact')}</label>
                <input type="text" value={ntContact} onChange={(e) => setNtContact(e.target.value)} placeholder={t('helpdesk.newTicket.contactPlaceholder')} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring" />
              </div>
            </div>
            <DialogFooter className="px-6 py-4 border-t border-border">
              <button onClick={() => setNewTicketOpen(false)} className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors">{t('common.cancel')}</button>
              <button onClick={handleSaveNewTicket} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">{t('helpdesk.newTicket.createButton')}</button>
            </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* External Dialogs */}
      <CannedResponsesPanel open={cannedResponsesOpen} onClose={() => setCannedResponsesOpen(false)} onInsert={(content) => { setReplyText(content.replace(/<[^>]+>/g, '')); setCannedResponsesOpen(false) }} />
      <BusinessHoursDialog open={businessHoursOpen} onClose={() => setBusinessHoursOpen(false)} />
      <TicketRoutingConfig open={routingConfigOpen} onClose={() => setRoutingConfigOpen(false)} />
    </div>
  )
}

// ============================================================
// Sub-Components
// ============================================================

function StatCard({ icon: Icon, label, value, iconColor, iconBg }: {
  icon: typeof AlertCircle; label: string; value: string | number; iconColor: string; iconBg: string
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-4 transition-all duration-200 hover:shadow-md hover:-translate-y-0.5">
      <div className="flex items-center gap-3 mb-2">
        <div className={`flex h-10 w-10 items-center justify-center rounded-xl icon-accent ${iconBg}`}>
          <Icon className={`h-5 w-5 ${iconColor}`} />
        </div>
      </div>
      <p className="text-2xl font-semibold text-foreground stat-accent">{value}</p>
      <p className="text-xs text-muted-foreground mt-1">{label}</p>
    </div>
  )
}

function TicketDetailPanel({ ticket, replyText, onReplyChange, showInternalNotes, onToggleInternal, onSendReply, onStatusChange, onClose }: {
  ticket: TicketType; replyText: string; onReplyChange: (v: string) => void; showInternalNotes: boolean
  onToggleInternal: (v: boolean) => void; onSendReply: () => void; onStatusChange: (s: TicketType['status']) => void; onClose: () => void
}) {
  const { t } = useTranslation()
  const [statusDropdownOpen, setStatusDropdownOpen] = useState(false)
  const [aiSuggestionLoading, setAISuggestionLoading] = useState(false)
  const aiHelpdeskEnabled = useAIStore((s) => s.isModuleEnabled('helpdesk'))

  const priorityLabels: Record<string, string> = {
    low: t('helpdesk.priority.low'), medium: t('helpdesk.priority.medium'),
    high: t('helpdesk.priority.high'), critical: t('helpdesk.priority.critical'),
  }
  const statusLabels: Record<string, string> = {
    open: t('helpdesk.status.open'), in_progress: t('helpdesk.status.inProgress'),
    waiting: t('helpdesk.status.waiting'), resolved: t('helpdesk.status.resolved'),
    closed: t('helpdesk.status.closed'),
  }

  const handleAISuggestion = () => {
    setAISuggestionLoading(true)
    setTimeout(() => {
      const suggestions: Record<string, string> = {
        'tk-1': 'Guten Tag,\n\nder Drucker im 2. OG wurde erfolgreich neu konfiguriert. Bitte testen Sie den Druckvorgang erneut. Falls das Problem weiterhin besteht, prüfen Sie bitte die Netzwerkverbindung des Druckers (Kabel am Port 3 im Patchfeld).\n\nBei weiteren Fragen stehe ich Ihnen gerne zur Verfügung.',
        'tk-2': 'Hallo,\n\nbasierend auf den Logs liegt das Problem an einem veralteten VPN-Profil. Bitte führen Sie folgende Schritte aus:\n\n1. Öffnen Sie AnyConnect → Einstellungen → Profile\n2. Löschen Sie das bestehende Profil "Firma-VPN"\n3. Verbinden Sie sich erneut mit vpn.firma.de\n\nDas neue Profil wird automatisch heruntergeladen.',
        'tk-3': 'Hallo,\n\nalle Zugänge für den neuen Mitarbeiter wurden eingerichtet:\n\n- Active Directory Konto\n- E-Mail-Konto\n- ERP-Zugang: Standardrolle\n- Zeiterfassung: Profil angelegt\n\nDie Zugangsdaten werden am ersten Arbeitstag persönlich übergeben.',
      }
      const suggestion = suggestions[ticket.id] ?? 'Vielen Dank für Ihre Anfrage. Wir haben Ihr Anliegen geprüft und arbeiten an einer Lösung. Wir melden uns kurzfristig mit weiteren Informationen.\n\nMit freundlichen Grüßen'
      onReplyChange(suggestion)
      setAISuggestionLoading(false)
      useAIStore.getState().addActivityLog({
        module: 'Helpdesk',
        action: 'Antwort vorgeschlagen',
        inputPreview: `${ticket.id}: ${ticket.subject.slice(0, 40)}`,
        outputPreview: suggestion.slice(0, 50) + '...',
      })
      toast.success(t('helpdesk.ticket.aiSuggestionInserted'))
    }, 1800)
  }
  const [showCannedPicker, setShowCannedPicker] = useState(false)
  const thread = getThread(ticket.id)
  const internalNoteCount = thread.filter((m) => m.isInternal).length

  const isWarningHours = !ticket.slaOverdue && ticket.slaRemaining.includes('h') && !ticket.slaRemaining.includes('d')
  const parsedHours = parseInt(ticket.slaRemaining, 10)
  const isYellow = isWarningHours && !isNaN(parsedHours) && parsedHours < 4
  let slaBgClass = 'bg-success-light text-success'
  if (ticket.slaOverdue) slaBgClass = 'bg-error-light text-error'
  else if (isYellow) slaBgClass = 'bg-warning-light text-warning'

  return (
    <div className="fixed inset-y-0 right-0 z-40 w-[440px] max-w-full border-l border-border bg-card shadow-xl flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-5 py-4 shrink-0">
        <div className="min-w-0">
          <p className="text-xs font-mono text-muted-foreground">{ticket.ticketNr}</p>
          <h3 className="text-sm font-semibold text-foreground truncate">{ticket.subject}</h3>
        </div>
        <button onClick={onClose} className="rounded-lg p-1.5 text-muted-foreground hover:bg-secondary transition-colors shrink-0 ml-2"><X className="h-4 w-4" /></button>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto px-5 py-4 space-y-5">
        {/* SLA Breach Banner (5.9) */}
        {ticket.slaOverdue && <SLABreachBanner remaining={ticket.slaRemaining} />}

        {/* Badges row */}
        <div className="flex flex-wrap items-center gap-2">
          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${priorityColors[ticket.priority]}`}>{priorityLabels[ticket.priority]}</span>
          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${statusColors[ticket.status]}`}>{statusLabels[ticket.status]}</span>
          {ticket.category && (
            <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${categoryColors[ticket.category] ?? 'bg-secondary text-muted-foreground'}`}>
              <Tag className="inline h-2.5 w-2.5 mr-0.5" />{ticket.category}
            </span>
          )}
          {ticket.autoRouted && (
            <span className="rounded-full bg-primary/10 text-primary px-2 py-0.5 text-[10px] font-medium">
              <Route className="inline h-2.5 w-2.5 mr-0.5" />Auto-Routing
            </span>
          )}
        </div>

        {/* SLA Timer */}
        <div className={`rounded-lg px-3 py-2.5 ${slaBgClass}`}>
          <div className="flex items-center gap-2 text-xs font-medium">
            <Clock className="h-3.5 w-3.5" />
            <span>SLA: {ticket.slaRemaining}</span>
          </div>
        </div>

        {/* Description */}
        <div>
          <h4 className="text-xs font-medium text-muted-foreground mb-1">{t('helpdesk.ticket.description')}</h4>
          <p className="text-sm text-foreground leading-relaxed">{ticket.description}</p>
        </div>

        {/* Custom Fields (5.13) */}
        {ticket.customFields && Object.keys(ticket.customFields).length > 0 && (
          <div>
            <h4 className="text-xs font-medium text-muted-foreground mb-2 flex items-center gap-1">
              <Settings2 className="h-3 w-3" /> {t('helpdesk.ticket.customFields')}
            </h4>
            <div className="grid grid-cols-2 gap-2">
              {MOCK_CUSTOM_FIELD_DEFS.map((def) => {
                const val = ticket.customFields?.[def.name]
                if (val === undefined) return null
                return (
                  <div key={def.id} className="rounded-lg border border-border bg-secondary/30 px-2.5 py-1.5">
                    <p className="text-[10px] text-muted-foreground">{def.name}</p>
                    <p className="text-xs font-medium text-foreground">
                      {def.type === 'checkbox' ? (val ? t('common.yes') : t('common.no')) : String(val)}
                    </p>
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* Meta info */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">{t('helpdesk.ticket.contact')}</p>
            <div className="flex items-center gap-1.5 text-sm text-foreground"><User className="h-3.5 w-3.5 text-muted-foreground" />{ticket.contactName}</div>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">{t('helpdesk.ticket.assignedTo')}</p>
            <div className="flex items-center gap-1.5 text-sm text-foreground"><User className="h-3.5 w-3.5 text-muted-foreground" />{ticket.assignedTo}</div>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">{t('helpdesk.ticket.created')}</p>
            <p className="text-sm text-foreground">{new Date(ticket.createdAt).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })}</p>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">{t('helpdesk.ticket.updated')}</p>
            <p className="text-sm text-foreground">{new Date(ticket.updatedAt).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })}</p>
          </div>
        </div>

        {/* Status change */}
        <div>
          <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5">{t('helpdesk.ticket.changeStatus')}</p>
          <div className="relative">
            <button onClick={() => setStatusDropdownOpen(!statusDropdownOpen)} className="flex w-full items-center justify-between rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
              <span className="flex items-center gap-2">
                <span className={`inline-block h-2 w-2 rounded-full ${statusColors[ticket.status].split(' ')[0]}`} />
                {statusLabels[ticket.status]}
              </span>
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            </button>
            {statusDropdownOpen && (
              <div className="absolute top-full left-0 right-0 z-10 mt-1 rounded-lg border border-border bg-card py-1 shadow-lg">
                {(['open', 'in_progress', 'waiting', 'resolved', 'closed'] as const).map((s) => (
                  <button key={s} onClick={() => { onStatusChange(s); setStatusDropdownOpen(false) }} className={`flex w-full items-center gap-2 px-3 py-1.5 text-sm transition-colors hover:bg-secondary ${ticket.status === s ? 'text-primary font-medium' : 'text-foreground'}`}>
                    <span className={`inline-block h-2 w-2 rounded-full ${statusColors[s].split(' ')[0]}`} />{statusLabels[s]}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="border-t border-border" />

        {/* CSAT Widget (5.12) */}
        <CSATWidget ticketId={ticket.id} ticketStatus={ticket.status} />

        {/* Message thread */}
        <div>
          <h4 className="text-xs font-medium text-muted-foreground mb-3 flex items-center gap-1.5">
            <MessageSquare className="h-3.5 w-3.5" />
            {t('helpdesk.ticket.messageThread', { count: thread.length })}
            {internalNoteCount > 0 && (
              <span className="rounded-full bg-warning-light text-warning px-1.5 py-0.5 text-[9px] font-medium ml-1">
                {t('helpdesk.ticket.internalCount', { count: internalNoteCount })}
              </span>
            )}
          </h4>
          <div className="space-y-3">
            {thread.map((msg) => (
              <div key={msg.id} className={`flex flex-col ${msg.role === 'agent' ? 'items-end' : 'items-start'}`}>
                <div className={`max-w-[85%] rounded-lg px-3 py-2 text-sm ${
                  msg.isInternal ? 'bg-warning-light/50 border border-warning/30' : msg.role === 'agent' ? 'bg-primary/10 text-foreground' : 'bg-secondary text-foreground'
                }`}>
                  <div className="flex items-center gap-2 mb-0.5">
                    <span className="text-[10px] font-medium text-muted-foreground">{msg.author}</span>
                    {msg.isInternal && (
                      <span className="flex items-center gap-0.5 text-[9px] text-warning font-medium"><Lock className="h-2.5 w-2.5" />{t('helpdesk.ticket.internal')}</span>
                    )}
                  </div>
                  <p className="text-xs leading-relaxed">{msg.text}</p>
                  <p className="text-[10px] text-muted-foreground mt-1">{new Date(msg.timestamp).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Reply area */}
      <div className="border-t border-border px-5 py-3 shrink-0">
        {/* Internal note banner (5.7) */}
        {showInternalNotes && (
          <div className="flex items-center gap-1.5 rounded-lg bg-warning-light/30 border border-warning/20 px-2.5 py-1.5 text-[10px] text-warning font-medium mb-2">
            <Lock className="h-3 w-3" /> {t('helpdesk.ticket.internalOnlyBanner')}
          </div>
        )}

        <div className="flex items-center gap-2 mb-2">
          <button onClick={() => onToggleInternal(false)} className={`rounded-lg px-2.5 py-1 text-xs transition-colors ${!showInternalNotes ? 'bg-primary/10 text-primary font-medium' : 'text-muted-foreground hover:bg-secondary'}`}>{t('helpdesk.ticket.reply')}</button>
          <button onClick={() => onToggleInternal(true)} className={`flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs transition-colors ${showInternalNotes ? 'bg-warning-light text-warning font-medium' : 'text-muted-foreground hover:bg-secondary'}`}>
            <Lock className="h-3 w-3" />{t('helpdesk.ticket.internalNote')}{internalNoteCount > 0 && ` (${internalNoteCount})`}
          </button>
          <div className="flex-1" />
          {aiHelpdeskEnabled && (
            <button
              onClick={handleAISuggestion}
              disabled={aiSuggestionLoading}
              className="flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-primary hover:bg-primary-light transition-colors disabled:opacity-40"
            >
              {aiSuggestionLoading ? (
                <span className="h-3 w-3 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              ) : (
                <Bot className="h-3 w-3" />
              )}
              {t('helpdesk.ticket.aiSuggestion')}
            </button>
          )}
          <div className="relative">
            <button onClick={() => setShowCannedPicker(!showCannedPicker)} className="flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-muted-foreground hover:bg-secondary transition-colors">
              <Zap className="h-3 w-3" />{t('helpdesk.ticket.cannedResponse')}
            </button>
            {showCannedPicker && (
              <CannedResponsePicker
                onSelect={(content) => { onReplyChange(content.replace(/<[^>]+>/g, '')); setShowCannedPicker(false) }}
                onClose={() => setShowCannedPicker(false)}
              />
            )}
          </div>
        </div>
        <div className="flex gap-2">
          <textarea
            value={replyText}
            onChange={(e) => onReplyChange(e.target.value)}
            placeholder={showInternalNotes ? t('helpdesk.ticket.internalNotePlaceholder') : t('helpdesk.ticket.replyPlaceholder')}
            rows={2}
            className={`flex-1 rounded-lg border px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 resize-none ${
              showInternalNotes ? 'border-warning/40 bg-warning-light/20 focus:ring-warning/30' : 'border-border bg-card focus:ring-focus-ring'
            }`}
          />
          <button onClick={onSendReply} disabled={!replyText.trim()} className="self-end rounded-lg bg-primary p-2.5 text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
            <Send className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  )
}

function KBArticleDetail({ article, onBack }: { article: KBArticle; onBack: () => void }) {
  const { t } = useTranslation()
  const body = KB_BODIES[article.id] ?? t('helpdesk.kb.noContent')
  const [editing, setEditing] = useState(false)
  const [editContent, setEditContent] = useState(() => body.split('\n\n').map((p) => `<p>${p}</p>`).join(''))

  return (
    <div className="max-w-3xl mx-auto">
      <button onClick={onBack} className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors mb-5">
        <ArrowLeft className="h-3.5 w-3.5" />{t('helpdesk.kb.backToOverview')}
      </button>

      <div className="rounded-lg border border-border bg-card p-6">
        <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground mb-2">{article.title}</h2>
            <div className="flex items-center gap-3">
              <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${kbCategoryColors[article.category] ?? 'bg-secondary text-muted-foreground'}`}>{article.category}</span>
              {article.published ? (
                <span className="rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium">{t('helpdesk.kb.published')}</span>
              ) : (
                <span className="rounded-full bg-secondary text-muted-foreground px-2 py-0.5 text-[10px] font-medium">{t('helpdesk.kb.draft')}</span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="flex items-center gap-1 text-xs text-muted-foreground"><Eye className="h-3.5 w-3.5" />{article.views}</span>
            {!editing ? (
              <button onClick={() => setEditing(true)} className="flex items-center gap-1 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors">
                <Pencil className="h-3 w-3" />{t('common.edit')}
              </button>
            ) : (
              <div className="flex items-center gap-1.5">
                <button onClick={() => setEditing(false)} className="rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors">{t('common.cancel')}</button>
                <button onClick={() => { setEditing(false); toast.success(t('helpdesk.kb.articleSaved')) }} className="rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors">{t('common.save')}</button>
              </div>
            )}
          </div>
        </div>

        <div className="border-t border-border mb-4" />

        {/* Body — TipTap in edit mode (5.14) */}
        {editing ? (
          <RichTextEditor
            content={editContent}
            onChange={setEditContent}
            placeholder={t('helpdesk.kb.articleContentPlaceholder')}
            showToolbar
            minHeight="200px"
          />
        ) : (
          <div className="prose prose-sm max-w-none">
            {body.split('\n\n').map((paragraph, i) => (
              <p key={i} className="text-sm text-foreground leading-relaxed mb-3">{paragraph}</p>
            ))}
          </div>
        )}

        <div className="border-t border-border mt-6 pt-4 flex items-center justify-between">
          <p className="text-xs text-muted-foreground">
            {t('helpdesk.kb.lastUpdated')}: {formatDate(article.updatedAt, { day: '2-digit', month: 'long', year: 'numeric' })}
          </p>
          <button onClick={() => toast.info(t('helpdesk.kb.feedbackSent'))} className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors">
            {t('helpdesk.kb.wasHelpful')}
          </button>
        </div>
      </div>
    </div>
  )
}
