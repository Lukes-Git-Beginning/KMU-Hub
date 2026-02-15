import { useState, useMemo } from 'react'
import {
  Search,
  Plus,
  Ticket,
  BookOpen,
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
  FileText,
} from 'lucide-react'
import { toast } from 'sonner'
import { useHelpdeskStore, type Ticket as TicketType, type KBArticle } from '@/stores/helpdesk'

type TabKey = 'tickets' | 'wissensdatenbank' | 'statistik'
type StatusFilter = 'all' | TicketType['status']
type PriorityFilter = 'all' | TicketType['priority']

// ---------------------------------------------------------------------------
// Label / Color Maps
// ---------------------------------------------------------------------------

const priorityColors: Record<string, string> = {
  low: 'bg-secondary text-muted-foreground',
  medium: 'bg-info-light text-info',
  high: 'bg-warning-light text-warning',
  critical: 'bg-error-light text-error',
}

const priorityLabels: Record<string, string> = {
  low: 'Niedrig',
  medium: 'Mittel',
  high: 'Hoch',
  critical: 'Kritisch',
}

const statusColors: Record<string, string> = {
  open: 'bg-warning-light text-warning',
  in_progress: 'bg-primary-light text-primary',
  waiting: 'bg-info-light text-info',
  resolved: 'bg-success-light text-success',
  closed: 'bg-secondary text-muted-foreground',
}

const statusLabels: Record<string, string> = {
  open: 'Offen',
  in_progress: 'In Bearbeitung',
  waiting: 'Wartend',
  resolved: 'Gelöst',
  closed: 'Geschlossen',
}

const kbCategoryColors: Record<string, string> = {
  Netzwerk: 'bg-info-light text-info',
  Hardware: 'bg-warning-light text-warning',
  Sicherheit: 'bg-error-light text-error',
  'E-Mail': 'bg-primary-light text-primary',
  Allgemein: 'bg-secondary text-muted-foreground',
}

// ---------------------------------------------------------------------------
// Mock message threads per ticket (inline, not in store)
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
    { id: 'm1', author: 'Brigitte Schärer', role: 'customer', text: 'Der Drucker im 2. OG zeigt seit heute Morgen "Offline" an. Mehrere Kollegen haben das gleiche Problem.', timestamp: '2026-02-15T08:30:00' },
    { id: 'm2', author: 'Marco Hartmann', role: 'agent', text: 'Danke für die Meldung. Ich prüfe den Druckserver und die Netzwerkverbindung. Bitte kurz Geduld.', timestamp: '2026-02-15T09:15:00' },
    { id: 'm3', author: 'Marco Hartmann', role: 'agent', text: 'Der Druckspooler-Dienst wurde neu gestartet. Bitte testen Sie erneut.', timestamp: '2026-02-15T09:45:00', isInternal: false },
  ],
  'tk-2': [
    { id: 'm1', author: 'Stefan Wenger', role: 'customer', text: 'Meine VPN-Verbindung trennt sich alle 10 Minuten. Arbeiten im Home-Office ist so nicht möglich.', timestamp: '2026-02-14T17:45:00' },
    { id: 'm2', author: 'Marco Hartmann', role: 'agent', text: 'Welchen VPN-Client nutzen Sie? Bitte senden Sie mir die Logdatei unter C:\\ProgramData\\Cisco\\logs.', timestamp: '2026-02-15T08:00:00' },
    { id: 'm3', author: 'Stefan Wenger', role: 'customer', text: 'Cisco AnyConnect 4.10. Logs sind im Anhang. Es scheint immer beim gleichen Timeout zu passieren.', timestamp: '2026-02-15T08:30:00' },
  ],
  'tk-3': [
    { id: 'm1', author: 'Karin Pfister', role: 'customer', text: 'Neuer Mitarbeiter Lukas Meier startet am 01.03. in der Buchhaltung. Bitte alle Standardzugänge einrichten.', timestamp: '2026-02-14T14:00:00' },
    { id: 'm2', author: 'Sandra Bürki', role: 'agent', text: 'Wird vorbereitet. AD-Konto, E-Mail, ERP und Zeiterfassung. Ich melde mich wenn alles bereit ist.', timestamp: '2026-02-14T15:00:00', isInternal: true },
  ],
}

function getThread(ticketId: string): ThreadMessage[] {
  return MOCK_THREADS[ticketId] ?? [
    { id: 'default-1', author: 'System', role: 'agent', text: 'Ticket wurde erstellt und dem zuständigen Techniker zugewiesen.', timestamp: new Date().toISOString() },
  ]
}

// ---------------------------------------------------------------------------
// KB Article placeholder bodies
// ---------------------------------------------------------------------------

const KB_BODIES: Record<string, string> = {
  'kb-1': `Schritt 1: Laden Sie den Cisco AnyConnect Client von unserem Self-Service Portal herunter.\n\nSchritt 2: Installieren Sie den Client und starten Sie ihn. Geben Sie als Server-Adresse "vpn.firma.ch" ein.\n\nSchritt 3: Melden Sie sich mit Ihren Active-Directory Zugangsdaten an (gleiche wie Windows-Login). Bei der ersten Verbindung müssen Sie das Zertifikat akzeptieren.\n\nBei Problemen kontaktieren Sie bitte den Helpdesk unter Ticket-Kategorie "Netzwerk".`,
  'kb-2': `Netzwerkdrucker unter Windows hinzufügen:\n\n1. Öffnen Sie die Windows-Einstellungen → Geräte → Drucker und Scanner.\n2. Klicken Sie auf "Drucker oder Scanner hinzufügen".\n3. Falls der Drucker nicht automatisch gefunden wird, klicken Sie auf "Der gewünschte Drucker ist nicht aufgelistet".\n4. Wählen Sie "Freigegebenen Drucker über den Namen auswählen" und geben Sie den Pfad ein (z.B. \\\\printserver\\HP-2OG).\n\nTreiber werden automatisch installiert. Bei macOS verwenden Sie das Druckdienstprogramm unter Systemeinstellungen.`,
  'kb-3': `Passwort-Reset über Self-Service Portal:\n\nBesuchen Sie https://password.firma.ch und melden Sie sich mit Ihrem Benutzernamen an. Sie erhalten einen Bestätigungscode per SMS an die hinterlegte Mobilnummer.\n\nGeben Sie den Code ein und setzen Sie ein neues Passwort. Das Passwort muss mindestens 12 Zeichen lang sein und Gross-/Kleinbuchstaben, Zahlen sowie ein Sonderzeichen enthalten.\n\nNach dem Reset müssen Sie sich auf allen Geräten neu anmelden.`,
  'kb-4': `E-Mail Signatur einrichten:\n\nVerwenden Sie die offizielle Vorlage aus dem Intranet unter "Vorlagen → E-Mail Signatur". Kopieren Sie den HTML-Code und fügen Sie ihn in Outlook unter Datei → Optionen → E-Mail → Signaturen ein.\n\nBitte achten Sie auf die korrekte Schreibweise Ihres Namens, Ihrer Position und der Telefonnummer gemäss CI/CD-Richtlinien.`,
  'kb-5': `Home-Office IT-Checkliste:\n\n- VPN-Zugang eingerichtet und getestet\n- Softphone oder Rufweiterleitung konfiguriert\n- Laptop mit aktuellem Betriebssystem und Virenscanner\n- Stabile Internetverbindung (mind. 20 Mbit/s empfohlen)\n- Bildschirmsperre aktiviert (max. 5 Min. Timeout)\n- Keine vertraulichen Dokumente ausdrucken\n\nBei Fragen wenden Sie sich an den Helpdesk.`,
}

// ============================================================
// Main Component
// ============================================================

export default function HelpdeskPage() {
  const { tickets, kbArticles, stats } = useHelpdeskStore()

  // Tab & filters
  const [tab, setTab] = useState<TabKey>('tickets')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [priorityFilter, setPriorityFilter] = useState<PriorityFilter>('all')

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

  // KB article detail
  const [selectedArticleId, setSelectedArticleId] = useState<string | null>(null)

  // Computed
  const openTickets = tickets.filter((t) => t.status !== 'closed' && t.status !== 'resolved')

  const filteredTickets = useMemo(() => {
    return tickets.filter((t) => {
      if (statusFilter !== 'all' && t.status !== statusFilter) return false
      if (priorityFilter !== 'all' && t.priority !== priorityFilter) return false
      if (search) {
        const q = search.toLowerCase()
        return (
          t.ticketNr.toLowerCase().includes(q) ||
          t.subject.toLowerCase().includes(q) ||
          t.assignedTo.toLowerCase().includes(q)
        )
      }
      return true
    })
  }, [tickets, statusFilter, priorityFilter, search])

  const selectedTicket = tickets.find((t) => t.id === selectedTicketId) ?? null
  const selectedArticle = kbArticles.find((a) => a.id === selectedArticleId) ?? null

  // Handlers
  const handleOpenNewTicket = () => {
    setNtSubject('')
    setNtDescription('')
    setNtPriority('medium')
    setNtAssignee('Marco Hartmann')
    setNtContact('')
    setNewTicketOpen(true)
  }

  const handleSaveNewTicket = () => {
    if (!ntSubject.trim()) {
      toast.error('Bitte Betreff eingeben')
      return
    }
    toast.success(`Ticket erstellt: ${ntSubject}`)
    setNewTicketOpen(false)
  }

  const handleSendReply = () => {
    if (!replyText.trim()) return
    toast.success(showInternalNotes ? 'Interne Notiz gespeichert' : 'Antwort gesendet')
    setReplyText('')
  }

  const handleStatusChange = (newStatus: TicketType['status']) => {
    if (selectedTicket) {
      toast.info(`Status geändert: ${statusLabels[newStatus]}`)
    }
  }

  const handleTicketRowClick = (id: string) => {
    setSelectedTicketId(id)
    setReplyText('')
    setShowInternalNotes(false)
  }

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div>
          <h1 className="text-foreground">Helpdesk</h1>
          <p className="text-sm text-muted-foreground">
            {openTickets.length} offene Tickets &middot; {kbArticles.length} Wissensartikel
          </p>
        </div>
        <button
          onClick={handleOpenNewTicket}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Plus className="h-4 w-4" />
          Neues Ticket
        </button>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'tickets' as const, label: `Tickets (${openTickets.length} offen)` },
          { key: 'wissensdatenbank' as const, label: 'Wissensdatenbank' },
          { key: 'statistik' as const, label: 'Statistik' },
        ]).map((t) => (
          <button
            key={t.key}
            onClick={() => { setTab(t.key); setSelectedTicketId(null); setSelectedArticleId(null) }}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key ? 'border-primary text-primary font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'
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
        <>
          {/* Filters row */}
          <div className="flex flex-wrap items-center gap-3 mb-4">
            {/* Search */}
            <div className="relative flex-1 min-w-[200px] max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Ticket suchen..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>

            {/* Status filter */}
            <div className="relative">
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
                className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
              >
                <option value="all">Alle Status</option>
                <option value="open">Offen</option>
                <option value="in_progress">In Bearbeitung</option>
                <option value="waiting">Wartend</option>
                <option value="resolved">Gelöst</option>
                <option value="closed">Geschlossen</option>
              </select>
              <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            </div>

            {/* Priority filter */}
            <div className="relative">
              <select
                value={priorityFilter}
                onChange={(e) => setPriorityFilter(e.target.value as PriorityFilter)}
                className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
              >
                <option value="all">Alle Prioritäten</option>
                <option value="critical">Kritisch</option>
                <option value="high">Hoch</option>
                <option value="medium">Mittel</option>
                <option value="low">Niedrig</option>
              </select>
              <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            </div>

            {/* Active filter chips */}
            {(statusFilter !== 'all' || priorityFilter !== 'all') && (
              <button
                onClick={() => { setStatusFilter('all'); setPriorityFilter('all') }}
                className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
              >
                Filter zurücksetzen
              </button>
            )}
          </div>

          {/* Ticket table */}
          <div className="rounded-lg border border-border bg-card overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Ticket-Nr</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Betreff</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Priorität</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Zugewiesen an</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">SLA</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Erstellt am</th>
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
                      <td className="px-4 py-3 text-foreground font-medium max-w-[300px] truncate">{ticket.subject}</td>
                      <td className="px-4 py-3">
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${priorityColors[ticket.priority]}`}>
                          {priorityLabels[ticket.priority] ?? ticket.priority}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${statusColors[ticket.status]}`}>
                          {statusLabels[ticket.status] ?? ticket.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{ticket.assignedTo}</td>
                      <td className="px-4 py-3">
                        <SlaIndicator overdue={ticket.slaOverdue} remaining={ticket.slaRemaining} />
                      </td>
                      <td className="px-4 py-3 text-xs text-muted-foreground">
                        {new Date(ticket.createdAt).toLocaleDateString('de-CH')}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {filteredTickets.length === 0 && (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <Ticket className="h-10 w-10 mb-3 opacity-40" />
                <p className="text-sm font-medium">Keine Tickets gefunden</p>
                <p className="text-xs mt-1">{search || statusFilter !== 'all' || priorityFilter !== 'all' ? 'Passe deine Filter an' : 'Erstelle ein neues Ticket'}</p>
              </div>
            )}
          </div>
        </>
      )}

      {/* ================================================================== */}
      {/* WISSENSDATENBANK TAB                                                */}
      {/* ================================================================== */}
      {tab === 'wissensdatenbank' && (
        <>
          {selectedArticle ? (
            <KBArticleDetail
              article={selectedArticle}
              onBack={() => setSelectedArticleId(null)}
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {kbArticles.length === 0 ? (
                <div className="col-span-full flex flex-col items-center justify-center py-12 text-muted-foreground">
                  <BookOpen className="h-10 w-10 mb-3 opacity-40" />
                  <p className="text-sm font-medium">Keine Artikel vorhanden</p>
                  <p className="text-xs mt-1">Erstelle den ersten Wissensartikel</p>
                </div>
              ) : (
                kbArticles.map((article) => (
                  <div
                    key={article.id}
                    onClick={() => setSelectedArticleId(article.id)}
                    className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer"
                  >
                    <div className="flex items-start justify-between mb-2">
                      <h4 className="text-sm font-medium text-foreground line-clamp-2">{article.title}</h4>
                      {article.published ? (
                        <span className="rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium shrink-0 ml-2">
                          Veröffentlicht
                        </span>
                      ) : (
                        <span className="rounded-full bg-secondary text-muted-foreground px-2 py-0.5 text-[10px] font-medium shrink-0 ml-2">
                          Entwurf
                        </span>
                      )}
                    </div>
                    <span className={`inline-block rounded-full px-2 py-0.5 text-[10px] font-medium mb-2 ${kbCategoryColors[article.category] ?? 'bg-secondary text-muted-foreground'}`}>
                      {article.category}
                    </span>
                    <p className="text-xs text-muted-foreground line-clamp-3 mb-3">{article.excerpt}</p>
                    <div className="flex items-center gap-3 text-xs text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <Eye className="h-3 w-3" />
                        {article.views}
                      </span>
                      <span>{new Date(article.updatedAt).toLocaleDateString('de-CH')}</span>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </>
      )}

      {/* ================================================================== */}
      {/* STATISTIK TAB                                                       */}
      {/* ================================================================== */}
      {tab === 'statistik' && (
        <>
          {/* KPI Cards */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
            <StatCard
              icon={AlertCircle}
              label="Offene Tickets"
              value={stats.openTickets}
              iconColor="text-warning"
              iconBg="bg-warning-light"
            />
            <StatCard
              icon={Clock}
              label="Durchschnittl. Antwortzeit"
              value={stats.avgResponseTime}
              iconColor="text-info"
              iconBg="bg-info-light"
            />
            <StatCard
              icon={CheckCircle2}
              label="Gelöst diese Woche"
              value={stats.resolvedThisWeek}
              iconColor="text-success"
              iconBg="bg-success-light"
            />
            <StatCard
              icon={BarChart3}
              label="Kundenzufriedenheit"
              value={stats.customerSatisfaction}
              iconColor="text-primary"
              iconBg="bg-primary-light"
            />
          </div>

          {/* Bar Chart */}
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-sm font-medium text-foreground mb-4">Tickets pro Wochentag</h3>
            <div className="flex items-end gap-3 h-40">
              {stats.weeklyBreakdown.map((day) => {
                const maxCount = Math.max(...stats.weeklyBreakdown.map((d) => d.count), 1)
                return (
                  <div key={day.label} className="flex-1 flex flex-col items-center gap-1">
                    <span className="text-xs text-muted-foreground">{day.count}</span>
                    <div
                      className="w-full rounded-t bg-primary/80 transition-all"
                      style={{ height: `${Math.max(8, (day.count / maxCount) * 100)}%` }}
                    />
                    <span className="text-[10px] text-muted-foreground">{day.label}</span>
                  </div>
                )
              })}
            </div>
          </div>

          {/* Breakdown by status */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-6">
            <div className="rounded-lg border border-border bg-card p-4">
              <h3 className="text-sm font-medium text-foreground mb-3">Nach Status</h3>
              <div className="space-y-2">
                {(['open', 'in_progress', 'waiting', 'resolved', 'closed'] as const).map((s) => {
                  const count = tickets.filter((t) => t.status === s).length
                  const pct = tickets.length > 0 ? Math.round((count / tickets.length) * 100) : 0
                  return (
                    <div key={s} className="flex items-center gap-3">
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium w-28 text-center ${statusColors[s]}`}>
                        {statusLabels[s]}
                      </span>
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
              <h3 className="text-sm font-medium text-foreground mb-3">Nach Priorität</h3>
              <div className="space-y-2">
                {(['critical', 'high', 'medium', 'low'] as const).map((p) => {
                  const count = tickets.filter((t) => t.priority === p).length
                  const pct = tickets.length > 0 ? Math.round((count / tickets.length) * 100) : 0
                  return (
                    <div key={p} className="flex items-center gap-3">
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium w-28 text-center ${priorityColors[p]}`}>
                        {priorityLabels[p]}
                      </span>
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
        </>
      )}

      {/* ================================================================== */}
      {/* TICKET DETAIL PANEL (slide-over right)                              */}
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
      {newTicketOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setNewTicketOpen(false)}>
          <div className="w-full max-w-lg rounded-xl bg-card border border-border shadow-xl" onClick={(e) => e.stopPropagation()}>
            {/* Dialog header */}
            <div className="flex items-center justify-between border-b border-border px-6 py-4">
              <h2 className="text-base font-semibold text-foreground">Neues Ticket erstellen</h2>
              <button onClick={() => setNewTicketOpen(false)} className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors">
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Dialog body */}
            <div className="space-y-4 px-6 py-5">
              {/* Betreff */}
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">Betreff</label>
                <input
                  type="text"
                  value={ntSubject}
                  onChange={(e) => setNtSubject(e.target.value)}
                  placeholder="Kurze Beschreibung des Problems"
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>

              {/* Beschreibung */}
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">Beschreibung</label>
                <textarea
                  value={ntDescription}
                  onChange={(e) => setNtDescription(e.target.value)}
                  placeholder="Detaillierte Problembeschreibung..."
                  rows={4}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
                />
              </div>

              {/* Priorität */}
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">Priorität</label>
                <div className="flex gap-2">
                  {(['low', 'medium', 'high', 'critical'] as const).map((p) => (
                    <label
                      key={p}
                      className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs cursor-pointer transition-colors ${
                        ntPriority === p
                          ? 'border-primary bg-primary/10 text-primary font-medium'
                          : 'border-border text-muted-foreground hover:bg-secondary'
                      }`}
                    >
                      <input
                        type="radio"
                        name="priority"
                        value={p}
                        checked={ntPriority === p}
                        onChange={() => setNtPriority(p)}
                        className="sr-only"
                      />
                      {priorityLabels[p]}
                    </label>
                  ))}
                </div>
              </div>

              {/* Zuweisen an */}
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">Zuweisen an</label>
                <div className="relative">
                  <select
                    value={ntAssignee}
                    onChange={(e) => setNtAssignee(e.target.value)}
                    className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
                  >
                    <option value="Marco Hartmann">Marco Hartmann</option>
                    <option value="Sandra Bürki">Sandra Bürki</option>
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                </div>
              </div>

              {/* Kontakt */}
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">Kontakt</label>
                <input
                  type="text"
                  value={ntContact}
                  onChange={(e) => setNtContact(e.target.value)}
                  placeholder="Name des Kontakts"
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
            </div>

            {/* Dialog footer */}
            <div className="flex items-center justify-end gap-3 border-t border-border px-6 py-4">
              <button
                onClick={() => setNewTicketOpen(false)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleSaveNewTicket}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                Ticket erstellen
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// ============================================================
// Sub-Components
// ============================================================

/** SLA timer badge with color coding */
function SlaIndicator({ overdue, remaining }: { overdue: boolean; remaining: string }) {
  // Parse hours for yellow warning
  const isWarning = !overdue && remaining.includes('h') && !remaining.includes('d')
  const hours = parseInt(remaining, 10)
  const isYellow = isWarning && !isNaN(hours) && hours < 4

  let colorClass = 'text-muted-foreground'
  if (overdue) colorClass = 'text-error font-medium'
  else if (isYellow) colorClass = 'text-warning font-medium'

  return (
    <span className={`flex items-center gap-1 text-xs ${colorClass}`}>
      <Clock className="h-3 w-3" />
      {remaining}
    </span>
  )
}

/** Stat card for Statistik tab */
function StatCard({
  icon: Icon,
  label,
  value,
  iconColor,
  iconBg,
}: {
  icon: typeof AlertCircle
  label: string
  value: string | number
  iconColor: string
  iconBg: string
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center gap-3 mb-2">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${iconBg}`}>
          <Icon className={`h-5 w-5 ${iconColor}`} />
        </div>
      </div>
      <p className="text-2xl font-semibold text-foreground">{value}</p>
      <p className="text-xs text-muted-foreground mt-1">{label}</p>
    </div>
  )
}

/** Ticket detail slide-over panel */
function TicketDetailPanel({
  ticket,
  replyText,
  onReplyChange,
  showInternalNotes,
  onToggleInternal,
  onSendReply,
  onStatusChange,
  onClose,
}: {
  ticket: TicketType
  replyText: string
  onReplyChange: (v: string) => void
  showInternalNotes: boolean
  onToggleInternal: (v: boolean) => void
  onSendReply: () => void
  onStatusChange: (s: TicketType['status']) => void
  onClose: () => void
}) {
  const [statusDropdownOpen, setStatusDropdownOpen] = useState(false)
  const thread = getThread(ticket.id)

  // SLA color for the timer block
  const isWarningHours = !ticket.slaOverdue && ticket.slaRemaining.includes('h') && !ticket.slaRemaining.includes('d')
  const parsedHours = parseInt(ticket.slaRemaining, 10)
  const isYellow = isWarningHours && !isNaN(parsedHours) && parsedHours < 4

  let slaBgClass = 'bg-success-light text-success'
  if (ticket.slaOverdue) slaBgClass = 'bg-error-light text-error'
  else if (isYellow) slaBgClass = 'bg-warning-light text-warning'

  return (
    <div className="fixed inset-y-0 right-0 z-40 w-[420px] max-w-full border-l border-border bg-card shadow-xl flex flex-col overflow-hidden">
      {/* Panel header */}
      <div className="flex items-center justify-between border-b border-border px-5 py-4 shrink-0">
        <div className="min-w-0">
          <p className="text-xs font-mono text-muted-foreground">{ticket.ticketNr}</p>
          <h3 className="text-sm font-semibold text-foreground truncate">{ticket.subject}</h3>
        </div>
        <button onClick={onClose} className="rounded-lg p-1.5 text-muted-foreground hover:bg-secondary transition-colors shrink-0 ml-2">
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto px-5 py-4 space-y-5">
        {/* Badges row */}
        <div className="flex flex-wrap items-center gap-2">
          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${priorityColors[ticket.priority]}`}>
            {priorityLabels[ticket.priority]}
          </span>
          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${statusColors[ticket.status]}`}>
            {statusLabels[ticket.status]}
          </span>
        </div>

        {/* SLA Timer block */}
        <div className={`rounded-lg px-3 py-2.5 ${slaBgClass}`}>
          <div className="flex items-center gap-2 text-xs font-medium">
            <Clock className="h-3.5 w-3.5" />
            <span>SLA: {ticket.slaRemaining}</span>
          </div>
        </div>

        {/* Description */}
        <div>
          <h4 className="text-xs font-medium text-muted-foreground mb-1">Beschreibung</h4>
          <p className="text-sm text-foreground leading-relaxed">{ticket.description}</p>
        </div>

        {/* Meta info */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">Kontakt</p>
            <div className="flex items-center gap-1.5 text-sm text-foreground">
              <User className="h-3.5 w-3.5 text-muted-foreground" />
              {ticket.contactName}
            </div>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">Zugewiesen an</p>
            <div className="flex items-center gap-1.5 text-sm text-foreground">
              <User className="h-3.5 w-3.5 text-muted-foreground" />
              {ticket.assignedTo}
            </div>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">Erstellt</p>
            <p className="text-sm text-foreground">{new Date(ticket.createdAt).toLocaleString('de-CH', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })}</p>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">Aktualisiert</p>
            <p className="text-sm text-foreground">{new Date(ticket.updatedAt).toLocaleString('de-CH', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })}</p>
          </div>
        </div>

        {/* Status change */}
        <div>
          <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5">Status ändern</p>
          <div className="relative">
            <button
              onClick={() => setStatusDropdownOpen(!statusDropdownOpen)}
              className="flex w-full items-center justify-between rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <span className="flex items-center gap-2">
                <span className={`inline-block h-2 w-2 rounded-full ${statusColors[ticket.status].split(' ')[0]}`} />
                {statusLabels[ticket.status]}
              </span>
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            </button>
            {statusDropdownOpen && (
              <div className="absolute top-full left-0 right-0 z-10 mt-1 rounded-lg border border-border bg-card py-1 shadow-lg">
                {(['open', 'in_progress', 'waiting', 'resolved', 'closed'] as const).map((s) => (
                  <button
                    key={s}
                    onClick={() => { onStatusChange(s); setStatusDropdownOpen(false) }}
                    className={`flex w-full items-center gap-2 px-3 py-1.5 text-sm transition-colors hover:bg-secondary ${
                      ticket.status === s ? 'text-primary font-medium' : 'text-foreground'
                    }`}
                  >
                    <span className={`inline-block h-2 w-2 rounded-full ${statusColors[s].split(' ')[0]}`} />
                    {statusLabels[s]}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Separator */}
        <div className="border-t border-border" />

        {/* Message thread */}
        <div>
          <h4 className="text-xs font-medium text-muted-foreground mb-3 flex items-center gap-1.5">
            <MessageSquare className="h-3.5 w-3.5" />
            Nachrichtenverlauf ({thread.length})
          </h4>
          <div className="space-y-3">
            {thread.map((msg) => (
              <div
                key={msg.id}
                className={`flex flex-col ${msg.role === 'agent' ? 'items-end' : 'items-start'}`}
              >
                <div
                  className={`max-w-[85%] rounded-lg px-3 py-2 text-sm ${
                    msg.isInternal
                      ? 'bg-warning-light/50 border border-warning/30'
                      : msg.role === 'agent'
                        ? 'bg-primary/10 text-foreground'
                        : 'bg-secondary text-foreground'
                  }`}
                >
                  <div className="flex items-center gap-2 mb-0.5">
                    <span className="text-[10px] font-medium text-muted-foreground">{msg.author}</span>
                    {msg.isInternal && (
                      <span className="flex items-center gap-0.5 text-[9px] text-warning font-medium">
                        <Lock className="h-2.5 w-2.5" />
                        Intern
                      </span>
                    )}
                  </div>
                  <p className="text-xs leading-relaxed">{msg.text}</p>
                  <p className="text-[10px] text-muted-foreground mt-1">
                    {new Date(msg.timestamp).toLocaleString('de-CH', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Reply area (fixed at bottom) */}
      <div className="border-t border-border px-5 py-3 shrink-0">
        {/* Internal note toggle */}
        <div className="flex items-center gap-2 mb-2">
          <button
            onClick={() => onToggleInternal(false)}
            className={`rounded-lg px-2.5 py-1 text-xs transition-colors ${
              !showInternalNotes ? 'bg-primary/10 text-primary font-medium' : 'text-muted-foreground hover:bg-secondary'
            }`}
          >
            Antworten
          </button>
          <button
            onClick={() => onToggleInternal(true)}
            className={`flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs transition-colors ${
              showInternalNotes ? 'bg-warning-light text-warning font-medium' : 'text-muted-foreground hover:bg-secondary'
            }`}
          >
            <Lock className="h-3 w-3" />
            Interne Notiz
          </button>
        </div>
        <div className="flex gap-2">
          <textarea
            value={replyText}
            onChange={(e) => onReplyChange(e.target.value)}
            placeholder={showInternalNotes ? 'Interne Notiz schreiben...' : 'Antwort schreiben...'}
            rows={2}
            className={`flex-1 rounded-lg border px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 resize-none ${
              showInternalNotes
                ? 'border-warning/40 bg-warning-light/20 focus:ring-warning/30'
                : 'border-border bg-card focus:ring-focus-ring'
            }`}
          />
          <button
            onClick={onSendReply}
            disabled={!replyText.trim()}
            className="self-end rounded-lg bg-primary p-2.5 text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <Send className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  )
}

/** KB Article full detail view */
function KBArticleDetail({ article, onBack }: { article: KBArticle; onBack: () => void }) {
  const body = KB_BODIES[article.id] ?? 'Dieser Artikel hat noch keinen Inhalt. Bitte wenden Sie sich an den Helpdesk-Administrator.'

  return (
    <div className="max-w-3xl">
      <button
        onClick={onBack}
        className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors mb-5"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Zurück zur Übersicht
      </button>

      <div className="rounded-lg border border-border bg-card p-6">
        {/* Header */}
        <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground mb-2">{article.title}</h2>
            <div className="flex items-center gap-3">
              <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${kbCategoryColors[article.category] ?? 'bg-secondary text-muted-foreground'}`}>
                {article.category}
              </span>
              {article.published ? (
                <span className="rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium">
                  Veröffentlicht
                </span>
              ) : (
                <span className="rounded-full bg-secondary text-muted-foreground px-2 py-0.5 text-[10px] font-medium">
                  Entwurf
                </span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-4 text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <Eye className="h-3.5 w-3.5" />
              {article.views} Aufrufe
            </span>
            <span className="flex items-center gap-1">
              <FileText className="h-3.5 w-3.5" />
              {new Date(article.updatedAt).toLocaleDateString('de-CH')}
            </span>
          </div>
        </div>

        {/* Separator */}
        <div className="border-t border-border mb-4" />

        {/* Body */}
        <div className="prose prose-sm max-w-none">
          {body.split('\n\n').map((paragraph, i) => (
            <p key={i} className="text-sm text-foreground leading-relaxed mb-3">
              {paragraph}
            </p>
          ))}
        </div>

        {/* Footer */}
        <div className="border-t border-border mt-6 pt-4 flex items-center justify-between">
          <p className="text-xs text-muted-foreground">
            Zuletzt aktualisiert: {new Date(article.updatedAt).toLocaleDateString('de-CH', { day: '2-digit', month: 'long', year: 'numeric' })}
          </p>
          <button
            onClick={() => toast.info('Artikel-Feedback gesendet')}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            War dieser Artikel hilfreich?
          </button>
        </div>
      </div>
    </div>
  )
}
