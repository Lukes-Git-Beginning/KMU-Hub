import { useState, useMemo, useEffect } from 'react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
  BarChart3,
  AlertCircle,
  Clock,
  CheckCircle2,
  Send,
  MessageSquare,
  User,
  ChevronDown,
  ArrowLeft,
  ArrowUp,
  UserPlus,
  GitMerge,
  Lock,
  Zap,
  Route,
  Pencil,
  Tag,
  Bot,
  LifeBuoy,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  slaLabel,
  MOCK_CATEGORIES,
  DEMO_TICKET_CUSTOM_FIELDS,
} from '@/stores/helpdesk'
import type { CustomFieldDefinition } from '@/mocks/data/custom-fields'
import {
  useTickets,
  useTicketMessages,
  useCreateTicket,
  useUpdateTicket,
  useCloseTicket,
  useReopenTicket,
  useAssignTicket,
  useMergeTickets,
  useAddMessage,
  useKBArticles,
  useCreateKBArticle,
  useUpdateKBArticle,
  useHelpdeskStats,
} from '@/api/hooks/useHelpdesk'
import type { KBArticle } from '@/api/helpdesk-types'
import {
  wireTicketToDisplay,
  displayStatusToWire,
  displayPriorityToWire,
  type DisplayTicket,
} from '@/api/helpdesk-adapters'
import type { TicketMessage } from '@/api/helpdesk-types'
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
import { DocumentBlockEditor, DocumentReader, isDocEmpty, type DocRow } from '@/components/shared/document'
import { kbBlockRegistry, kbContentToRows, kbRowsToContent, kbContentPreview } from './kb-blocks'
import { useAIStore } from '@/stores/ai'
import { useHelpdeskPrefsStore } from '@/stores/helpdeskPrefs'
import { PageHeader, EmptyState, DetailModal, SortMenu, AbbrTooltip, SkeletonTable, SkeletonText, type SortDirection } from '@/components/shared'
import { EditableText, useEditorGuard, useModuleValueSet, useModuleAreas, useValueSetMigration, useModuleCustomFields } from '@/components/customization/EditorSurface'
import { useCapabilitySet, useCapability, useScopedCapability, useHasCapability } from '@/hooks/useCapability'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
import { useAuthStore } from '@/stores/auth'
import { EmptyHelpdesk } from '@/components/shared/illustrations'
import { formatDate } from '@/lib/format'

type TabKey = 'tickets' | 'wissensdatenbank' | 'statistik'
type StatusFilter = 'all' | DisplayTicket['status']
type PriorityFilter = 'all' | DisplayTicket['priority']
type CategoryFilter = 'all' | string

/** Demo agent pool for assign/escalate (no real team lookup — that is CRM/backend). */
const HELPDESK_AGENTS = ['Marco Hartmann', 'Sandra Bürki'] as const

/** Sort fields for the ticket list (H-7). */
const SORT_FIELDS = [
  { value: 'createdAt', labelKey: 'helpdesk.sort.createdAt' },
  { value: 'priority', labelKey: 'helpdesk.sort.priority' },
  { value: 'status', labelKey: 'helpdesk.sort.status' },
  { value: 'sla', labelKey: 'helpdesk.sort.sla' },
] as const
const PRIORITY_RANK: Record<DisplayTicket['priority'], number> = { low: 0, medium: 1, high: 2, critical: 3 }
const STATUS_RANK: Record<DisplayTicket['status'], number> = { open: 0, in_progress: 1, waiting: 2, resolved: 3, closed: 4 }

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

/**
 * A status/priority chip coloured from the customizable value-set. When the
 * value-set option carries a colour we tint the pill with it (so recolouring in
 * the Wertelisten editor shows live); otherwise we fall back to the semantic
 * Tailwind class. Keeps one look whether or not a tenant has customised colours.
 */
function VsChip({ label, color, fallbackClass, className = '' }: {
  label: string; color?: string; fallbackClass?: string; className?: string
}) {
  const base = 'inline-block rounded-full px-2 py-0.5 text-[10px] font-medium'
  if (color) {
    return (
      <span className={`${base} ${className}`} style={{ backgroundColor: `color-mix(in srgb, ${color} 16%, transparent)`, color }}>
        {label}
      </span>
    )
  }
  return <span className={`${base} ${fallbackClass ?? ''} ${className}`}>{label}</span>
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
  // Editor sandbox: mutating/out-navigating actions become no-ops (R-1).
  const guard = useEditorGuard()

  // API hooks — Tickets (core data)
  const { data: ticketsResponse, isLoading: ticketsLoading, error: ticketsError } = useTickets()
  // Editor draft may delete a status/priority and reassign its records; remap the
  // display values through that migration so the preview is honest (empty in the
  // live app — the real record migration runs on deploy in the backend).
  const statusMigration = useValueSetMigration('ticket_status')
  const priorityMigration = useValueSetMigration('ticket_priority')
  // Custom-field values for tickets created this session (keyed by ticket id) —
  // display-layer overlay merged in the `tickets` memo (wire has no custom fields).
  const [createdCustomFields, setCreatedCustomFields] = useState<Record<string, Record<string, string>>>({})
  const rawTickets: DisplayTicket[] = useMemo(
    () => (ticketsResponse?.tickets ?? []).map(wireTicketToDisplay),
    [ticketsResponse],
  )
  const tickets: DisplayTicket[] = useMemo(
    () =>
      rawTickets.map((tk) => ({
        ...tk,
        status: (statusMigration[tk.status] ?? tk.status) as DisplayTicket['status'],
        priority: (priorityMigration[tk.priority] ?? tk.priority) as DisplayTicket['priority'],
        // Custom-field values live in a display-layer overlay (wire type has none
        // yet — backend-gaps §Customization): demo seed ⊕ in-session edits, so the
        // detail renders them and edits stick within the session.
        customFields: (DEMO_TICKET_CUSTOM_FIELDS[tk.id] || createdCustomFields[tk.id])
          ? { ...DEMO_TICKET_CUSTOM_FIELDS[tk.id], ...createdCustomFields[tk.id] }
          : tk.customFields,
      })),
    [rawTickets, statusMigration, priorityMigration, createdCustomFields],
  )

  // Ticket mutations
  const createTicketMut = useCreateTicket()
  const updateTicketMut = useUpdateTicket()
  const closeTicketMut = useCloseTicket()
  const reopenTicketMut = useReopenTicket()
  const assignTicketMut = useAssignTicket()
  const mergeTicketsMut = useMergeTickets()
  const addMessageMut = useAddMessage()

  // API hooks — KB articles + Stats
  const { data: kbArticlesData } = useKBArticles()
  const kbArticles: KBArticle[] = kbArticlesData ?? []
  const { data: statsData } = useHelpdeskStats()
  const stats = statsData ?? {
    open_tickets: 0,
    avg_response_time: '–',
    resolved_this_week: 0,
    customer_satisfaction: '–',
    weekly_breakdown: [],
  }

  // Priority AND status labels + colours come from the customizable value-sets
  // (ticket_priority / ticket_status) so the Wertelisten editor drives label,
  // colour, order and add/remove; fall back to the i18n enum + Tailwind class
  // when an option is missing. The editor sandbox layers the draft on top → live.
  const priorityValueSet = useModuleValueSet('ticket_priority')
  const statusValueSet = useModuleValueSet('ticket_status')
  // Tenant-defined custom fields for tickets — rendered as inputs in the new-ticket
  // dialog so values can be captured on creation (draft ⊕ live; previews in editor).
  const ticketFieldDefs = useModuleCustomFields('helpdesk_ticket')
  const prioBy = new Map((priorityValueSet?.options ?? []).map((o) => [o.id, o]))
  const statusBy = new Map((statusValueSet?.options ?? []).map((o) => [o.id, o]))
  const priorityColorOf = (id: string): string | undefined => prioBy.get(id)?.color
  const statusColorOf = (id: string): string | undefined => statusBy.get(id)?.color
  const activePriorityOptions = (priorityValueSet?.options ?? []).filter((o) => o.active)
  const activeStatusOptions = (statusValueSet?.options ?? []).filter((o) => o.active)
  const priorityLabels: Record<string, string> = {
    low: prioBy.get('low')?.label ?? t('helpdesk.priority.low'),
    medium: prioBy.get('medium')?.label ?? t('helpdesk.priority.medium'),
    high: prioBy.get('high')?.label ?? t('helpdesk.priority.high'),
    critical: prioBy.get('critical')?.label ?? t('helpdesk.priority.critical'),
  }
  const statusLabels: Record<string, string> = {
    open: statusBy.get('open')?.label ?? t('helpdesk.status.open'),
    in_progress: statusBy.get('in_progress')?.label ?? t('helpdesk.status.inProgress'),
    waiting: statusBy.get('waiting')?.label ?? t('helpdesk.status.waiting'),
    resolved: statusBy.get('resolved')?.label ?? t('helpdesk.status.resolved'),
    closed: statusBy.get('closed')?.label ?? t('helpdesk.status.closed'),
  }

  // RBAC R-3 capability checks (default hidden per gating convention)
  const { has: capHas, ready: capReady } = useCapabilitySet()
  const ticketRead = useCapability('helpdesk:ticket:read')
  const canCreateTicket = useHasCapability('helpdesk:ticket:create')
  const canManageCanned = useHasCapability('helpdesk:canned:manage')
  const currentUserId = useAuthStore((s) => s.user?.id ?? null)

  // Tab & filters — seeded from personal prefs (H-6)
  const startTab = useHelpdeskPrefsStore((s) => s.startTab)
  const defaultStatusFilter = useHelpdeskPrefsStore((s) => s.defaultStatusFilter)
  const [tab, setTab] = useState<TabKey>(startTab)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>(defaultStatusFilter)
  const [priorityFilter, setPriorityFilter] = useState<PriorityFilter>('all')
  const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('all')
  const [sortField, setSortField] = useState<string>('createdAt')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')

  // Detail panel
  const [selectedTicketId, setSelectedTicketId] = useState<string | null>(null)
  const [replyText, setReplyText] = useState('')
  const [showInternalNotes, setShowInternalNotes] = useState(false)

  // New ticket dialog
  const [newTicketOpen, setNewTicketOpen] = useState(false)
  const [ntSubject, setNtSubject] = useState('')
  const [ntDescription, setNtDescription] = useState('')
  const [ntPriority, setNtPriority] = useState<DisplayTicket['priority']>('medium')
  const [ntAssignee, setNtAssignee] = useState('Marco Hartmann')
  const [ntContact, setNtContact] = useState('')
  const [ntCategory, setNtCategory] = useState<string>('Sonstiges')
  // Custom-field values entered in the new-ticket dialog (keyed by field.key).
  const [ntCustomFields, setNtCustomFields] = useState<Record<string, string>>({})

  // KB article detail
  const [selectedArticleId, setSelectedArticleId] = useState<string | null>(null)
  // Track a just-created article so it opens straight in the block editor.
  const [newArticleEditId, setNewArticleEditId] = useState<string | null>(null)
  const createKBArticle = useCreateKBArticle()
  const canManageKb = useHasCapability('helpdesk:kb:manage')

  // Dialogs (5.6) — business hours + routing now live in the settings panel (H-6)
  const [cannedResponsesOpen, setCannedResponsesOpen] = useState(false)

  // RBAC tab gating — wissensdatenbank is always free (self-service KB)
  const TAB_CAPABILITY: Record<TabKey, string | null> = {
    tickets: 'helpdesk:ticket:read',
    wissensdatenbank: null,
    statistik: 'helpdesk:stats:view',
  }
  // R4: tenant can hide whole tabs via the editor (moduleAreas). A tab shows only
  // when RBAC-visible AND its area is enabled.
  const areaEnabled = useModuleAreas('helpdesk')
  const visibleTabs = (Object.keys(TAB_CAPABILITY) as TabKey[]).filter(
    (key) =>
      (!capReady || TAB_CAPABILITY[key] === null || capHas(TAB_CAPABILITY[key] as string)) &&
      areaEnabled[key] !== false,
  )
  useEffect(() => {
    if (!capReady) return
    if (visibleTabs.length > 0 && !visibleTabs.includes(tab)) setTab(visibleTabs[0])
  }, [capReady, visibleTabs, tab])

  // Computed
  // Requester-Modell: ticket:read scope=own → nur eigene Tickets sehen
  const baseTickets = useMemo(() => {
    if (ticketRead.allowed && ticketRead.scope === 'own') {
      return tickets.filter(
        (t) => t.requesterId === currentUserId || t.assigneeId === currentUserId,
      )
    }
    return tickets
  }, [tickets, ticketRead.allowed, ticketRead.scope, currentUserId])

  const openTickets = baseTickets.filter((t) => t.status !== 'closed' && t.status !== 'resolved')

  const filteredTickets = useMemo(() => {
    return baseTickets.filter((t) => {
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
  }, [baseTickets, statusFilter, priorityFilter, categoryFilter, search])

  const sortedTickets = useMemo(() => {
    const arr = [...filteredTickets]
    arr.sort((a, b) => {
      let cmp = 0
      switch (sortField) {
        case 'priority': cmp = PRIORITY_RANK[a.priority] - PRIORITY_RANK[b.priority]; break
        case 'status': cmp = STATUS_RANK[a.status] - STATUS_RANK[b.status]; break
        case 'sla': cmp = new Date(a.slaDueAt).getTime() - new Date(b.slaDueAt).getTime(); break
        default: cmp = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
      }
      return sortDir === 'asc' ? cmp : -cmp
    })
    return arr
  }, [filteredTickets, sortField, sortDir])

  const sortOptions = SORT_FIELDS.map((f) => ({ value: f.value, label: t(f.labelKey) }))

  const selectedTicket: DisplayTicket | null = tickets.find((t) => t.id === selectedTicketId) ?? null
  const selectedArticle: KBArticle | null = kbArticles.find((a) => a.id === selectedArticleId) ?? null

  // Handlers
  const handleOpenNewTicket = () => {
    setNtSubject(''); setNtDescription(''); setNtPriority('medium')
    setNtAssignee('Marco Hartmann'); setNtContact(''); setNtCategory('Sonstiges')
    setNtCustomFields({})
    setNewTicketOpen(true)
  }

  const handleSaveNewTicket = () => {
    if (!ntSubject.trim()) { toast.error(t('helpdesk.newTicket.subjectRequired')); return }
    // Intake P1 (agent channel): the whole form flows onto the wire ticket —
    // description, category, contact and the custom-field values — so nothing
    // is lost on creation and no session overlay is needed for new tickets.
    const enteredFields = Object.fromEntries(
      Object.entries(ntCustomFields).filter(([, v]) => v !== undefined && v !== ''),
    )
    createTicketMut.mutate(
      {
        subject: ntSubject.trim(),
        description: ntDescription.trim() || undefined,
        category: ntCategory || undefined,
        priority: displayPriorityToWire(ntPriority),
        assignee_id: ntAssignee || undefined,
        channel: 'agent',
        // The agent notes the contact manually; ownership stays with the creator
        // (scope=own) so the new ticket remains visible to them.
        requester_name: ntContact.trim() || undefined,
        custom_fields: Object.keys(enteredFields).length > 0 ? enteredFields : undefined,
      },
      {
        onSuccess: () => {
          toast.success(t('helpdesk.newTicket.created', { subject: ntSubject.trim() }))
          setNewTicketOpen(false)
        },
        onError: () => toast.error(t('helpdesk.newTicket.createError')),
      },
    )
  }

  const handleSendReply = () => {
    if (!replyText.trim() || !selectedTicket) return
    addMessageMut.mutate(
      { ticketId: selectedTicket.id, body: replyText.trim(), internal: showInternalNotes },
      {
        onSuccess: () => {
          toast.success(showInternalNotes ? t('helpdesk.ticket.internalNoteSaved') : t('helpdesk.ticket.replySent'))
          setReplyText('')
        },
        onError: () => toast.error(t('helpdesk.ticket.replyError')),
      },
    )
  }

  const handleStatusChange = (newStatus: DisplayTicket['status']) => {
    if (!selectedTicket) return
    const id = selectedTicket.id
    const onError = () => toast.error(t('helpdesk.ticket.statusChangeError'))
    if (newStatus === 'closed') {
      closeTicketMut.mutate(id, {
        onSuccess: () => toast.info(t('helpdesk.ticket.statusChanged', { status: statusLabels[newStatus] })),
        onError,
      })
    } else if (
      (selectedTicket.status === 'closed' || selectedTicket.status === 'resolved') &&
      newStatus === 'open'
    ) {
      reopenTicketMut.mutate(id, {
        onSuccess: () => toast.info(t('helpdesk.ticket.statusChanged', { status: statusLabels[newStatus] })),
        onError,
      })
    } else {
      updateTicketMut.mutate({ id, status: displayStatusToWire(newStatus) }, {
        onSuccess: () => toast.info(t('helpdesk.ticket.statusChanged', { status: statusLabels[newStatus] })),
        onError,
      })
    }
  }

  const handleTicketRowClick = (id: string) => {
    setSelectedTicketId(id); setReplyText(''); setShowInternalNotes(false)
  }

  // Fill/change a custom-field value on a ticket (accumulated per ticket; merged
  // over the demo seed in the `tickets` memo). Session-only until BE persistence.
  const handleCustomFieldChange = (ticketId: string, key: string, value: string) => {
    setCreatedCustomFields((m) => ({ ...m, [ticketId]: { ...(m[ticketId] ?? {}), [key]: value } }))
  }

  const handleCreateKBArticle = () => {
    createKBArticle.mutate(
      { title: t('helpdesk.kb.newArticleTitle'), category: 'Allgemein', content: '', status: 'draft' },
      {
        onSuccess: (created) => {
          if (created?.id) { setNewArticleEditId(created.id); setSelectedArticleId(created.id) }
        },
        onError: () => toast.error(t('common.errorGeneric')),
      },
    )
  }

  const hasActiveFilters = statusFilter !== 'all' || priorityFilter !== 'all' || categoryFilter !== 'all'

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <PageHeader
        title={t('rbac.module.helpdesk')}
        description={t('helpdesk.header.description', { openCount: openTickets.length, articleCount: kbArticles.length })}
        icon={LifeBuoy}
        moduleId="helpdesk"
        actions={
          <div className="flex items-center gap-2">
            <RestrictedModeBadge module="helpdesk" />
            {canManageCanned && (
              <button
                onClick={() => setCannedResponsesOpen(true)}
                className="flex items-center gap-1.5 rounded-xl border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Zap className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">{t('helpdesk.header.cannedResponses')}</span>
              </button>
            )}
            {canCreateTicket && (
              <button
                onClick={guard(handleOpenNewTicket)}
                className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-4 w-4" />
                {t('helpdesk.header.newTicket')}
              </button>
            )}
          </div>
        }
        className="mb-6"
      />

      {/* Tabs — labels are edit-in-place (interactive: single click switches the tab,
          double click renames it). The Tickets counter renders separately so the
          noun stays renamable. */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {(([
          { key: 'tickets', dkey: 'helpdesk.tabs.ticketsLabel', count: openTickets.length },
          { key: 'wissensdatenbank', dkey: 'helpdesk.tabs.knowledgeBase' },
          { key: 'statistik', dkey: 'helpdesk.tabs.statistics' },
        ] as { key: TabKey; dkey: string; count?: number }[]).filter((item) => !capReady || visibleTabs.includes(item.key))).map((item) => (
          <button
            key={item.key}
            onClick={() => { setTab(item.key); setSelectedTicketId(null); setSelectedArticleId(null) }}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === item.key ? 'border-primary text-primary font-medium tab-accent-active' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <EditableText as="span" dkey={item.dkey} interactive />
            {item.count !== undefined && <span className="ml-1">({item.count})</span>}
          </button>
        ))}
      </div>

      {/* ================================================================== */}
      {/* TICKETS TAB                                                         */}
      {/* ================================================================== */}
      {tab === 'tickets' && (
        <div className="animate-fade-up">
          {ticketsLoading && (
            <SkeletonTable rows={8} cols={6} className="mb-4" />
          )}
          {ticketsError && (
            <div className="p-4 rounded-lg bg-error-light text-error text-sm mb-4">{t('helpdesk.loadError')}</div>
          )}
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
                {activeStatusOptions.map((o) => <option key={o.id} value={o.id}>{o.label}</option>)}
              </select>
              <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            </div>

            {/* Priority */}
            <div className="relative">
              <select value={priorityFilter} onChange={(e) => setPriorityFilter(e.target.value as PriorityFilter)} className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer">
                <option value="all">{t('helpdesk.filter.allPriorities')}</option>
                {activePriorityOptions.map((o) => <option key={o.id} value={o.id}>{o.label}</option>)}
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

            <div className="ml-auto">
              <SortMenu
                options={sortOptions}
                field={sortField}
                direction={sortDir}
                onChange={(f, d) => { setSortField(f); setSortDir(d) }}
              />
            </div>
          </div>

          {/* Ticket table */}
          <div className="rounded-lg border border-border bg-card overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground"><EditableText as="span" dkey="helpdesk.table.ticketNr" /></th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground"><EditableText as="span" dkey="helpdesk.table.subject" /></th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground"><EditableText as="span" dkey="helpdesk.table.category" /></th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground"><EditableText as="span" dkey="helpdesk.table.priority" /></th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('common.status')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground"><EditableText as="span" dkey="helpdesk.table.assignedTo" /></th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground"><AbbrTooltip term="SLA" /></th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground"><EditableText as="span" dkey="helpdesk.table.createdAt" /></th>
                  </tr>
                </thead>
                <tbody>
                  {sortedTickets.map((ticket) => (
                    <tr
                      key={ticket.id}
                      role="button"
                      tabIndex={0}
                      aria-label={`${ticket.ticketNr} — ${ticket.subject}`}
                      onClick={() => handleTicketRowClick(ticket.id)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          handleTicketRowClick(ticket.id)
                        }
                      }}
                      className={`border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring focus-visible:ring-inset ${
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
                        <VsChip label={priorityLabels[ticket.priority]} color={priorityColorOf(ticket.priority)} fallbackClass={priorityColors[ticket.priority]} />
                      </td>
                      <td className="px-4 py-3">
                        <VsChip label={statusLabels[ticket.status]} color={statusColorOf(ticket.status)} fallbackClass={statusColors[ticket.status]} />
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{ticket.assignedTo}</td>
                      <td className="px-4 py-3">
                        <SLABadge overdue={ticket.slaOverdue} days={ticket.slaDays} hours={ticket.slaHours} dueAt={ticket.slaDueAt} compact />
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
                action={!hasActiveFilters && !search && canCreateTicket ? { label: t('helpdesk.empty.createFirstTicket'), onClick: guard(handleOpenNewTicket) } : undefined}
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
            <KBArticleDetail
              key={selectedArticle.id}
              article={selectedArticle}
              startEditing={selectedArticle.id === newArticleEditId}
              onBack={() => { setSelectedArticleId(null); setNewArticleEditId(null) }}
            />
          ) : (
            <>
              {canManageKb && kbArticles.length > 0 && (
                <div className="mb-4 flex justify-end">
                  <button onClick={guard(handleCreateKBArticle)} className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">
                    <Plus className="h-4 w-4" />{t('helpdesk.kb.newArticle')}
                  </button>
                </div>
              )}
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {kbArticles.length === 0 ? (
                <div className="col-span-full">
                  <EmptyState
                    illustration={<EmptyHelpdesk />}
                    title={t('helpdesk.kb.noArticles')}
                    action={canManageKb ? { label: t('helpdesk.kb.newArticle'), onClick: guard(handleCreateKBArticle) } : undefined}
                  />
                </div>
              ) : (
                kbArticles.map((article) => (
                  <button type="button" key={article.id} onClick={() => setSelectedArticleId(article.id)} className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer text-left w-full">
                    <div className="flex items-start justify-between mb-2">
                      <h4 className="text-sm font-medium text-foreground line-clamp-2">{article.title}</h4>
                      {article.status === 'published' ? (
                        <span className="rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium shrink-0 ml-2">{t('helpdesk.kb.published')}</span>
                      ) : (
                        <span className="rounded-full bg-secondary text-muted-foreground px-2 py-0.5 text-[10px] font-medium shrink-0 ml-2">{t('helpdesk.kb.draft')}</span>
                      )}
                    </div>
                    <span className={`inline-block rounded-full px-2 py-0.5 text-[10px] font-medium mb-2 ${kbCategoryColors[article.category] ?? 'bg-secondary text-muted-foreground'}`}>{article.category}</span>
                    <p className="text-xs text-muted-foreground line-clamp-3 mb-3">{kbContentPreview(article.content || KB_BODIES[article.id] || '')}</p>
                    <div className="flex items-center gap-3 text-xs text-muted-foreground">
                      <span>{formatDate(article.updated_at)}</span>
                    </div>
                  </button>
                ))
              )}
              </div>
            </>
          )}
        </div>
      )}

      {/* ================================================================== */}
      {/* STATISTIK TAB                                                       */}
      {/* ================================================================== */}
      {tab === 'statistik' && (() => {
        // Stats widgets are toggleable via the editor (moduleAreas, `stat:` prefix).
        // A missing key means visible. CSAT has no real data source yet → its card
        // + chart stay hidden until the CSAT survey feature ships (backend-gaps);
        // the editor still lists them (locked) so the tenant sees they exist.
        const showStat = (key: string): boolean => areaEnabled[`stat:${key}`] !== false
        const CSAT_FEATURE_ENABLED = false
        const statCards = [
          showStat('openTickets') && <StatCard key="ot" icon={AlertCircle} label={<EditableText dkey="helpdesk.stats.openTickets" />} value={stats.open_tickets} iconColor="text-warning" iconBg="bg-warning-light" />,
          showStat('avgResponseTime') && <StatCard key="art" icon={Clock} label={<EditableText dkey="helpdesk.stats.avgResponseTime" />} value={stats.avg_response_time} iconColor="text-info" iconBg="bg-info-light" />,
          showStat('resolvedThisWeek') && <StatCard key="rtw" icon={CheckCircle2} label={<EditableText dkey="helpdesk.stats.resolvedThisWeek" />} value={stats.resolved_this_week} iconColor="text-success" iconBg="bg-success-light" />,
          CSAT_FEATURE_ENABLED && showStat('csat') && <StatCard key="csat" icon={BarChart3} label={<EditableText dkey="helpdesk.stats.customerSatisfaction" />} value={stats.customer_satisfaction} iconColor="text-primary" iconBg="bg-primary-light" />,
        ].filter(Boolean)
        const charts = [
          showStat('ticketsPerDay') && (
            <div key="tpd" className="rounded-lg border border-border bg-card p-6">
              <h3 className="text-sm font-medium text-foreground mb-4"><EditableText as="span" dkey="helpdesk.stats.ticketsPerDay" /></h3>
              <div className="flex items-end gap-3 h-40">
                {stats.weekly_breakdown.map((day) => {
                  const maxCount = Math.max(...stats.weekly_breakdown.map((d) => d.count), 1)
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
          ),
          CSAT_FEATURE_ENABLED && showStat('csatChart') && <CSATAggregate key="csat-agg" />,
          showStat('byStatus') && (
            <div key="bs" className="rounded-lg border border-border bg-card p-4">
              <h3 className="text-sm font-medium text-foreground mb-3"><EditableText as="span" dkey="helpdesk.stats.byStatus" /></h3>
              <div className="space-y-2">
                {activeStatusOptions.map((o) => {
                  const count = tickets.filter((t) => t.status === o.id).length
                  const pct = tickets.length > 0 ? Math.round((count / tickets.length) * 100) : 0
                  return (
                    <div key={o.id} className="flex items-center gap-3">
                      <VsChip label={o.label} color={o.color} fallbackClass={statusColors[o.id]} className="w-28 text-center" />
                      <div className="flex-1 h-2 rounded-full bg-secondary overflow-hidden">
                        <div className="h-full rounded-full bg-primary/70 transition-all" style={{ width: `${pct}%` }} />
                      </div>
                      <span className="text-xs text-muted-foreground w-10 text-right">{count}</span>
                    </div>
                  )
                })}
              </div>
            </div>
          ),
          showStat('byPriority') && (
            <div key="bp" className="rounded-lg border border-border bg-card p-4">
              <h3 className="text-sm font-medium text-foreground mb-3"><EditableText as="span" dkey="helpdesk.stats.byPriority" /></h3>
              <div className="space-y-2">
                {activePriorityOptions.map((o) => {
                  const count = tickets.filter((t) => t.priority === o.id).length
                  const pct = tickets.length > 0 ? Math.round((count / tickets.length) * 100) : 0
                  return (
                    <div key={o.id} className="flex items-center gap-3">
                      <VsChip label={o.label} color={o.color} fallbackClass={priorityColors[o.id]} className="w-28 text-center" />
                      <div className="flex-1 h-2 rounded-full bg-secondary overflow-hidden">
                        <div className="h-full rounded-full bg-primary/70 transition-all" style={{ width: `${pct}%` }} />
                      </div>
                      <span className="text-xs text-muted-foreground w-10 text-right">{count}</span>
                    </div>
                  )
                })}
              </div>
            </div>
          ),
        ].filter(Boolean)
        return (
          <div className="animate-fade-up">
            {statCards.length > 0 && (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 mb-8">{statCards}</div>
            )}
            {charts.length > 0 && (
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">{charts}</div>
            )}
            {statCards.length === 0 && charts.length === 0 && (
              <div className="flex flex-col items-center justify-center gap-2 py-16 text-center text-muted-foreground">
                <BarChart3 className="h-8 w-8" aria-hidden="true" />
                <p className="text-sm">{t('helpdesk.stats.allHidden')}</p>
              </div>
            )}
          </div>
        )
      })()}

      {/* ================================================================== */}
      {/* TICKET DETAIL PANEL                                                 */}
      {/* ================================================================== */}
      {selectedTicket && (
        <TicketDetailPanel
          ticket={selectedTicket}
          allTickets={tickets}
          replyText={replyText}
          onReplyChange={setReplyText}
          showInternalNotes={showInternalNotes}
          onToggleInternal={setShowInternalNotes}
          onSendReply={guard(handleSendReply)}
          onStatusChange={guard(handleStatusChange)}
          onClose={() => setSelectedTicketId(null)}
          onCustomFieldChange={handleCustomFieldChange}
          assignTicketMut={assignTicketMut}
          updateTicketMut={updateTicketMut}
          mergeTicketsMut={mergeTicketsMut}
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
                  {(['low', 'medium', 'high', 'critical'] as DisplayTicket['priority'][]).map((p) => (
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
              {/* Zusatzfelder — tenant-defined custom fields as inputs (G2). */}
              {ticketFieldDefs.length > 0 && (
                <div className="space-y-3 border-t border-border pt-4">
                  <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{t('helpdesk.ticket.customFields')}</p>
                  {ticketFieldDefs.map((def) => (
                    <div key={def.id}>
                      <label className="mb-1.5 block text-xs font-medium text-foreground">
                        {def.label}{def.required && <span className="text-destructive"> *</span>}
                      </label>
                      <CustomFieldControl
                        def={def}
                        value={ntCustomFields[def.key] ?? ''}
                        onChange={(v) => setNtCustomFields((m) => ({ ...m, [def.key]: v }))}
                      />
                    </div>
                  ))}
                </div>
              )}
            </div>
            <DialogFooter className="px-6 py-4 border-t border-border">
              <button onClick={() => setNewTicketOpen(false)} className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors">{t('common.cancel')}</button>
              <button onClick={guard(handleSaveNewTicket)} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">{t('helpdesk.newTicket.createButton')}</button>
            </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* External Dialogs */}
      <CannedResponsesPanel open={cannedResponsesOpen} onClose={() => setCannedResponsesOpen(false)} onInsert={(content) => { setReplyText(content.replace(/<[^>]+>/g, '')); setCannedResponsesOpen(false) }} />
    </div>
  )
}

// ============================================================
// Sub-Components
// ============================================================

function StatCard({ icon: Icon, label, value, iconColor, iconBg }: {
  icon: typeof AlertCircle; label: ReactNode; value: string | number; iconColor: string; iconBg: string
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

/**
 * A single custom ("Zusatzfeld") value control, rendered by its definition type.
 * Shared by the ticket detail (fill values in place) and the new-ticket dialog.
 * select → dropdown of its options · boolean → checkbox · else a typed input.
 */
function CustomFieldControl({
  def, value, onChange,
}: {
  def: CustomFieldDefinition
  value: string
  onChange: (v: string) => void
}) {
  const { t } = useTranslation()
  const inputCls = 'w-full rounded-lg border border-border bg-card px-2.5 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring'
  if (def.type === 'boolean') {
    return (
      <label className="flex items-center gap-2 text-sm text-foreground">
        <input type="checkbox" checked={value === 'true'} onChange={(e) => onChange(e.target.checked ? 'true' : 'false')} className="h-4 w-4 rounded border-border" />
        {value === 'true' ? t('common.yes') : t('common.no')}
      </label>
    )
  }
  if (def.type === 'select' && def.options.length > 0) {
    return (
      <div className="relative">
        <select value={value} onChange={(e) => onChange(e.target.value)} className={`${inputCls} appearance-none pr-7 cursor-pointer`}>
          <option value="">{t('helpdesk.newTicket.selectPlaceholder')}</option>
          {def.options.map((o) => <option key={o} value={o}>{o}</option>)}
        </select>
        <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
      </div>
    )
  }
  const inputType = def.type === 'number' ? 'number' : def.type === 'email' ? 'email' : def.type === 'url' ? 'url' : def.type === 'date' ? 'date' : 'text'
  return <input type={inputType} value={value} onChange={(e) => onChange(e.target.value)} placeholder={t('helpdesk.ticket.customFieldPlaceholder')} className={inputCls} />
}

function TicketDetailPanel({
  ticket, allTickets, replyText, onReplyChange, showInternalNotes, onToggleInternal,
  onSendReply, onStatusChange, onClose, onCustomFieldChange, assignTicketMut, updateTicketMut, mergeTicketsMut,
}: {
  ticket: DisplayTicket
  allTickets: DisplayTicket[]
  replyText: string
  onReplyChange: (v: string) => void
  showInternalNotes: boolean
  onToggleInternal: (v: boolean) => void
  onSendReply: () => void
  onStatusChange: (s: DisplayTicket['status']) => void
  onClose: () => void
  onCustomFieldChange: (ticketId: string, key: string, value: string) => void
  assignTicketMut: ReturnType<typeof useAssignTicket>
  updateTicketMut: ReturnType<typeof useUpdateTicket>
  mergeTicketsMut: ReturnType<typeof useMergeTickets>
}) {
  const { t } = useTranslation()
  const guard = useEditorGuard()
  const priorityValueSet = useModuleValueSet('ticket_priority')
  const statusValueSet = useModuleValueSet('ticket_status')
  // Custom "Zusatzfelder": the tenant-defined helpdesk_ticket fields (draft ⊕ live).
  // Editing/adding one in the Felder panel previews live here (G2).
  const customFieldDefs = useModuleCustomFields('helpdesk_ticket')

  // RBAC R-3: agent-action gating — ticket:edit scope=own → only when assigned to me
  const canEditTicket = useScopedCapability('helpdesk:ticket:edit', ticket.assigneeId)
  const canReplyPanel = useHasCapability('helpdesk:ticket:reply')
  const canEditInternal = useHasCapability('helpdesk:ticket:edit')

  const [statusDropdownOpen, setStatusDropdownOpen] = useState(false)
  const [aiSuggestionLoading, setAISuggestionLoading] = useState(false)
  const aiHelpdeskEnabled = useAIStore((s) => s.isModuleEnabled('helpdesk'))

  const prioBy = new Map((priorityValueSet?.options ?? []).map((o) => [o.id, o]))
  const statusBy = new Map((statusValueSet?.options ?? []).map((o) => [o.id, o]))
  const priorityColorOf = (id: string): string | undefined => prioBy.get(id)?.color
  const statusColorOf = (id: string): string | undefined => statusBy.get(id)?.color
  const activeStatusOptions = (statusValueSet?.options ?? []).filter((o) => o.active)
  const priorityLabels: Record<string, string> = {
    low: prioBy.get('low')?.label ?? t('helpdesk.priority.low'),
    medium: prioBy.get('medium')?.label ?? t('helpdesk.priority.medium'),
    high: prioBy.get('high')?.label ?? t('helpdesk.priority.high'),
    critical: prioBy.get('critical')?.label ?? t('helpdesk.priority.critical'),
  }
  const statusLabels: Record<string, string> = {
    open: statusBy.get('open')?.label ?? t('helpdesk.status.open'),
    in_progress: statusBy.get('in_progress')?.label ?? t('helpdesk.status.inProgress'),
    waiting: statusBy.get('waiting')?.label ?? t('helpdesk.status.waiting'),
    resolved: statusBy.get('resolved')?.label ?? t('helpdesk.status.resolved'),
    closed: statusBy.get('closed')?.label ?? t('helpdesk.status.closed'),
  }

  const handleAISuggestion = () => {
    setAISuggestionLoading(true)
    setTimeout(() => {
      const suggestion = 'Vielen Dank für Ihre Anfrage. Wir haben Ihr Anliegen geprüft und arbeiten an einer Lösung. Wir melden uns kurzfristig mit weiteren Informationen.\n\nMit freundlichen Grüßen'
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

  // Messages via API hook
  const { data: wireMessages = [], isLoading: messagesLoading } = useTicketMessages(ticket.id)
  const internalNoteCount = wireMessages.filter((m: TicketMessage) => m.internal).length

  // Agent actions (H-4)
  const mergeTargets = useMemo(
    () => allTickets.filter((x) => x.id !== ticket.id && x.status !== 'closed'),
    [allTickets, ticket.id],
  )

  const handleAssign = (agent: string) => {
    if (agent === ticket.assignedTo) return
    assignTicketMut.mutate(
      { ticketId: ticket.id, assigneeId: agent },
      { onSuccess: () => toast.success(t('helpdesk.ticket.assigned', { ticket: ticket.ticketNr, agent })),
        onError: () => toast.error(t('helpdesk.ticket.assignError')) },
    )
  }
  const handleEscalate = () => {
    if (ticket.priority === 'critical') { toast.info(t('helpdesk.ticket.escalateMax')); return }
    const ladder: DisplayTicket['priority'][] = ['low', 'medium', 'high', 'critical']
    const next = ladder[Math.min(ladder.indexOf(ticket.priority) + 1, ladder.length - 1)]
    updateTicketMut.mutate(
      { id: ticket.id, priority: 'urgent' },
      { onSuccess: () => toast.success(t('helpdesk.ticket.escalated', { ticket: ticket.ticketNr, priority: priorityLabels[next] })),
        onError: () => toast.error(t('helpdesk.ticket.escalateError')) },
    )
  }
  const handleMerge = (targetId: string) => {
    const target = allTickets.find((x) => x.id === targetId)
    mergeTicketsMut.mutate(
      { sourceId: ticket.id, targetId },
      { onSuccess: () => {
          toast.success(t('helpdesk.ticket.merged', { source: ticket.ticketNr, target: target?.ticketNr ?? '' }))
          onClose()
        },
        onError: () => toast.error(t('helpdesk.ticket.mergeError')) },
    )
  }

  const isYellow = !ticket.slaOverdue && ticket.slaDays === 0 && ticket.slaHours < 4
  let slaBgClass = 'bg-success-light text-success'
  if (ticket.slaOverdue) slaBgClass = 'bg-error-light text-error'
  else if (isYellow) slaBgClass = 'bg-warning-light text-warning'

  const headerBadge = (
    <div className="flex items-center gap-1.5">
      <VsChip label={priorityLabels[ticket.priority]} color={priorityColorOf(ticket.priority)} fallbackClass={priorityColors[ticket.priority]} />
      <VsChip label={statusLabels[ticket.status]} color={statusColorOf(ticket.status)} fallbackClass={statusColors[ticket.status]} />
    </div>
  )

  // Reply footer — nur wenn ticket:reply gewährt; Interner-Notizen-Toggle zusätzlich ticket:edit
  const footer = canReplyPanel ? (
    <div>
      {/* Internal note banner (5.7) */}
      {showInternalNotes && (
        <div className="flex items-center gap-1.5 rounded-lg bg-warning-light/30 border border-warning/20 px-2.5 py-1.5 text-[10px] text-warning font-medium mb-2">
          <Lock className="h-3 w-3" /> {t('helpdesk.ticket.internalOnlyBanner')}
        </div>
      )}

      <div className="flex items-center gap-2 mb-2">
        <button onClick={() => onToggleInternal(false)} className={`rounded-lg px-2.5 py-1 text-xs transition-colors ${!showInternalNotes ? 'bg-primary/10 text-primary font-medium' : 'text-muted-foreground hover:bg-secondary'}`}>{t('helpdesk.ticket.reply')}</button>
        {canEditInternal && (
          <button onClick={() => onToggleInternal(true)} className={`flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs transition-colors ${showInternalNotes ? 'bg-warning-light text-warning font-medium' : 'text-muted-foreground hover:bg-secondary'}`}>
            <Lock className="h-3 w-3" />{t('helpdesk.ticket.internalNote')}{internalNoteCount > 0 && ` (${internalNoteCount})`}
          </button>
        )}
        <div className="flex-1" />
        {aiHelpdeskEnabled && (
          <button
            onClick={guard(handleAISuggestion)}
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
        <CannedResponsePicker onSelect={(content) => onReplyChange(content.replace(/<[^>]+>/g, ''))} />
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
  ) : undefined

  return (
    <DetailModal
      open
      onClose={onClose}
      title={ticket.subject}
      subtitle={ticket.ticketNr}
      badge={headerBadge}
      footer={footer}
      maxWidth="max-w-3xl"
    >
      <div className="space-y-5">
        {/* SLA Breach Banner (5.9) */}
        {ticket.slaOverdue && <SLABreachBanner remaining={slaLabel(t, ticket)} />}

        {/* Secondary badges: category + auto-routing (status/priority live in the header) */}
        {(ticket.category || ticket.autoRouted) && (
          <div className="flex flex-wrap items-center gap-2">
            {ticket.category && (
              <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${categoryColors[ticket.category] ?? 'bg-secondary text-muted-foreground'}`}>
                <Tag className="inline h-2.5 w-2.5 mr-0.5" />{ticket.category}
              </span>
            )}
            {ticket.autoRouted && (
              <span className="rounded-full bg-primary/10 text-primary px-2 py-0.5 text-[10px] font-medium">
                <Route className="inline h-2.5 w-2.5 mr-0.5" />{t('helpdesk.ticket.autoRouting')}
              </span>
            )}
          </div>
        )}

        {/* SLA Timer */}
        <div className={`rounded-lg px-3 py-2.5 ${slaBgClass}`}>
          <div className="flex items-center gap-2 text-xs font-medium">
            <Clock className="h-3.5 w-3.5" />
            <span><AbbrTooltip term="SLA">{t('helpdesk.ticket.slaLabel')}</AbbrTooltip>: {slaLabel(t, ticket)}</span>
          </div>
        </div>

        {/* Description */}
        {ticket.description && (
          <div>
            <h4 className="text-xs font-medium text-muted-foreground mb-1"><EditableText as="span" dkey="helpdesk.ticket.description" /></h4>
            <p className="text-sm text-foreground leading-relaxed">{ticket.description}</p>
          </div>
        )}

        {/* Zusatzfelder — tenant-defined custom fields (customization layer),
            editable in place: pick a value from the dropdown, type a note, etc.
            Renaming/adding a field in the editor's Felder panel is visible here
            in context (G2). Values persist in the session overlay (backend-gaps). */}
        {customFieldDefs.length > 0 && (
          <div>
            <h4 className="text-xs font-medium text-muted-foreground mb-2"><EditableText as="span" dkey="helpdesk.ticket.customFields" /></h4>
            <div className="grid grid-cols-2 gap-3">
              {customFieldDefs.map((def) => (
                <div key={def.id} className="space-y-1">
                  <label className="text-[10px] text-muted-foreground">
                    {def.label}{def.required && <span className="text-destructive"> *</span>}
                  </label>
                  <CustomFieldControl
                    def={def}
                    value={String(ticket.customFields?.[def.key] ?? '')}
                    onChange={(v) => onCustomFieldChange(ticket.id, def.key, v)}
                  />
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Meta info */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5"><EditableText as="span" dkey="helpdesk.ticket.contact" /></p>
            <div className="flex items-center gap-1.5 text-sm text-foreground"><User className="h-3.5 w-3.5 text-muted-foreground" />{ticket.contactName}</div>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5"><EditableText as="span" dkey="helpdesk.ticket.assignedTo" /></p>
            <div className="flex items-center gap-1.5 text-sm text-foreground"><User className="h-3.5 w-3.5 text-muted-foreground" />{ticket.assignedTo}</div>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5"><EditableText as="span" dkey="helpdesk.ticket.created" /></p>
            <p className="text-sm text-foreground">{new Date(ticket.createdAt).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })}</p>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5"><EditableText as="span" dkey="helpdesk.ticket.updated" /></p>
            <p className="text-sm text-foreground">{new Date(ticket.updatedAt).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })}</p>
          </div>
        </div>

        {/* Agent actions (H-4): assign / escalate / merge — only for ticket:edit-capable users */}
        {canEditTicket && (
          <>
            <div className="rounded-lg border border-border bg-secondary/20 p-3">
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-2"><EditableText as="span" dkey="helpdesk.ticket.actions" /></p>
              <div className="flex flex-wrap items-center gap-2">
                {/* Assign */}
                <div className="relative">
                  <select
                    aria-label={t('helpdesk.ticket.assign')}
                    value={ticket.assignedTo}
                    onChange={(e) => guard(handleAssign)(e.target.value)}
                    className="appearance-none rounded-lg border border-border bg-card pl-7 pr-7 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
                  >
                    {!HELPDESK_AGENTS.includes(ticket.assignedTo as (typeof HELPDESK_AGENTS)[number]) && (
                      <option value={ticket.assignedTo}>{ticket.assignedTo}</option>
                    )}
                    {HELPDESK_AGENTS.map((a) => <option key={a} value={a}>{a}</option>)}
                  </select>
                  <UserPlus className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                  <ChevronDown className="pointer-events-none absolute right-1.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                </div>

                {/* Escalate */}
                <button
                  onClick={guard(handleEscalate)}
                  disabled={ticket.priority === 'critical'}
                  className="flex items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  <ArrowUp className="h-3.5 w-3.5" />{t('helpdesk.ticket.escalate')}
                </button>

                {/* Merge */}
                <div className="relative">
                  <select
                    aria-label={t('helpdesk.ticket.mergeInto')}
                    value=""
                    disabled={mergeTargets.length === 0}
                    onChange={(e) => { if (e.target.value) guard(handleMerge)(e.target.value) }}
                    className="appearance-none rounded-lg border border-border bg-card pl-7 pr-7 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    <option value="">{mergeTargets.length === 0 ? t('helpdesk.ticket.mergeNoTarget') : t('helpdesk.ticket.mergeInto')}</option>
                    {mergeTargets.map((tk) => <option key={tk.id} value={tk.id}>{tk.ticketNr} — {tk.subject}</option>)}
                  </select>
                  <GitMerge className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                  <ChevronDown className="pointer-events-none absolute right-1.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                </div>
              </div>
            </div>

            {/* Status change */}
            <div>
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5"><EditableText as="span" dkey="helpdesk.ticket.changeStatus" /></p>
              <div className="relative">
                <button onClick={() => setStatusDropdownOpen(!statusDropdownOpen)} className="flex w-full items-center justify-between rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                  <span className="flex items-center gap-2">
                    <span className="inline-block h-2 w-2 rounded-full" style={{ backgroundColor: statusColorOf(ticket.status) }} />
                    {statusLabels[ticket.status]}
                  </span>
                  <ChevronDown className="h-4 w-4 text-muted-foreground" />
                </button>
                {statusDropdownOpen && (
                  <div className="absolute top-full left-0 right-0 z-10 mt-1 rounded-lg border border-border bg-card py-1 shadow-lg">
                    {activeStatusOptions.map((o) => (
                      <button key={o.id} onClick={() => { onStatusChange(o.id as DisplayTicket['status']); setStatusDropdownOpen(false) }} className={`flex w-full items-center gap-2 px-3 py-1.5 text-sm transition-colors hover:bg-secondary ${ticket.status === o.id ? 'text-primary font-medium' : 'text-foreground'}`}>
                        <span className="inline-block h-2 w-2 rounded-full" style={{ backgroundColor: o.color }} />{o.label}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </>
        )}

        <div className="border-t border-border" />

        {/* CSAT Widget (5.12) */}
        <CSATWidget key={ticket.id} ticketId={ticket.id} ticketStatus={ticket.status} csatRating={ticket.csatRating} csatComment={ticket.csatComment} />

        {/* Message thread */}
        <div>
          <h4 className="text-xs font-medium text-muted-foreground mb-3 flex items-center gap-1.5">
            <MessageSquare className="h-3.5 w-3.5" />
            {t('helpdesk.ticket.messageThread', { count: wireMessages.length })}
            {internalNoteCount > 0 && (
              <span className="rounded-full bg-warning-light text-warning px-1.5 py-0.5 text-[9px] font-medium ml-1">
                {t('helpdesk.ticket.internalCount', { count: internalNoteCount })}
              </span>
            )}
          </h4>
          {messagesLoading && (
            <SkeletonText lines={3} className="py-2" />
          )}
          <div className="space-y-3">
            {wireMessages.map((msg: TicketMessage) => (
              <div key={msg.id} className="flex flex-col items-start">
                <div className={`max-w-[85%] rounded-lg px-3 py-2 text-sm ${
                  msg.internal ? 'bg-warning-light/50 border border-warning/30' : 'bg-secondary text-foreground'
                }`}>
                  <div className="flex items-center gap-2 mb-0.5">
                    <span className="text-[10px] font-medium text-muted-foreground">{msg.author_id}</span>
                    {msg.internal && (
                      <span className="flex items-center gap-0.5 text-[9px] text-warning font-medium"><Lock className="h-2.5 w-2.5" />{t('helpdesk.ticket.internal')}</span>
                    )}
                  </div>
                  <p className="text-xs leading-relaxed">{msg.body}</p>
                  <p className="text-[10px] text-muted-foreground mt-1">{new Date(msg.created_at).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </DetailModal>
  )
}

function KBArticleDetail({ article, onBack, startEditing = false }: { article: KBArticle; onBack: () => void; startEditing?: boolean }) {
  const { t } = useTranslation()
  const guard = useEditorGuard()
  const canManageKbArticle = useHasCapability('helpdesk:kb:manage')
  const updateKBArticle = useUpdateKBArticle()
  // KB articles are authored/read on the shared block-document engine (G3). Legacy
  // HTML/plain-text seeds are wrapped into a single text block by the adapter, so
  // they open cleanly and can be enriched with structural blocks.
  const [editing, setEditing] = useState(startEditing)
  const [title, setTitle] = useState(article.title)
  const [rows, setRows] = useState<DocRow[]>(() =>
    kbContentToRows(article.content || KB_BODIES[article.id] || ''),
  )
  const viewRows = useMemo(
    () => kbContentToRows(article.content || KB_BODIES[article.id] || ''),
    [article.content, article.id],
  )
  const viewEmpty = isDocEmpty(viewRows)

  const handleSaveBody = () => {
    updateKBArticle.mutate(
      { id: article.id, title: title.trim() || article.title, content: kbRowsToContent(rows) },
      {
        onSuccess: () => {
          setEditing(false)
          toast.success(t('helpdesk.kb.articleSaved'))
        },
        onError: () => {
          toast.error(t('common.errorGeneric'))
        },
      },
    )
  }

  return (
    <div className="max-w-3xl mx-auto">
      <button onClick={onBack} className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors mb-5">
        <ArrowLeft className="h-3.5 w-3.5" />{t('helpdesk.kb.backToOverview')}
      </button>

      <div className="rounded-lg border border-border bg-card p-6">
        <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
          <div className="min-w-0 flex-1">
            {editing ? (
              <input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder={t('helpdesk.kb.titlePlaceholder')}
                aria-label={t('helpdesk.kb.titlePlaceholder')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-lg font-semibold text-foreground mb-2 focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            ) : (
              <h2 className="text-lg font-semibold text-foreground mb-2">{article.title}</h2>
            )}
            <div className="flex items-center gap-3">
              <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${kbCategoryColors[article.category] ?? 'bg-secondary text-muted-foreground'}`}>{article.category}</span>
              {article.status === 'published' ? (
                <span className="rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium"><EditableText as="span" dkey="helpdesk.kb.published" /></span>
              ) : (
                <span className="rounded-full bg-secondary text-muted-foreground px-2 py-0.5 text-[10px] font-medium"><EditableText as="span" dkey="helpdesk.kb.draft" /></span>
              )}
            </div>
          </div>
          {canManageKbArticle && (
            <div className="flex items-center gap-2">
              {!editing ? (
                <button onClick={guard(() => setEditing(true))} className="flex items-center gap-1 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors">
                  <Pencil className="h-3 w-3" />{t('common.edit')}
                </button>
              ) : (
                <div className="flex items-center gap-1.5">
                  <button onClick={() => { setEditing(false); setTitle(article.title); setRows(kbContentToRows(article.content || KB_BODIES[article.id] || '')) }} className="rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors">{t('common.cancel')}</button>
                  <button onClick={guard(handleSaveBody)} className="rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors">{t('common.save')}</button>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="border-t border-border mb-4" />

        {/* Body — shared block-document engine (G3): edit via DocumentBlockEditor,
            read via DocumentReader. */}
        {editing ? (
          <DocumentBlockEditor rows={rows} onChange={setRows} registry={kbBlockRegistry} widthClass="max-w-none" />
        ) : viewEmpty ? (
          <p className="text-sm text-muted-foreground">{t('helpdesk.kb.noContent')}</p>
        ) : (
          <DocumentReader rows={viewRows} registry={kbBlockRegistry} />
        )}

        <div className="border-t border-border mt-6 pt-4 flex items-center justify-between">
          <p className="text-xs text-muted-foreground">
            {t('helpdesk.kb.lastUpdated')}: {formatDate(article.updated_at, { day: '2-digit', month: 'long', year: 'numeric' })}
          </p>
          <button onClick={guard(() => toast.info(t('helpdesk.kb.feedbackSent')))} className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors">
            {t('helpdesk.kb.wasHelpful')}
          </button>
        </div>
      </div>
    </div>
  )
}
