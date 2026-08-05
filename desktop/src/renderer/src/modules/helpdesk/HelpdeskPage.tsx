import { useState, useMemo, useEffect, useRef } from 'react'
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
  GripVertical,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  slaLabel,
  MOCK_CATEGORIES,
  useHelpdeskStore,
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
import type { KBArticle, TicketChannel, CreateTicketInput } from '@/api/helpdesk-types'
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
import { EditableText, useEditorGuard, useModuleValueSet, useModuleAreas, useValueSetMigration, useModuleCustomFields, useEditorFocusEffect, useFieldOptions, useModuleColumnLayout, orderColumns, columnWidthStyle } from '@/components/customization/EditorSurface'
import { mapSubmissionToRecord, IntakeFieldInputs } from '@/components/shared/intake'
import { useFormSchema } from '@/api/hooks/useFormulare'
import type { FormField } from '@/api/formulare-types'
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
  // CSAT feature gate = the tenant Helpdesk setting (intake P3, auto-after-close).
  const csatEnabled = useHelpdeskStore((s) => s.csatEnabled)
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
        // Custom-field values come from the wire ticket (intake P1b). Unsaved /
        // just-saved in-session edits layer on top for instant, flicker-free
        // typing; the commit (onBlur for text, on-change for select/checkbox)
        // persists them to the wire via PUT.
        customFields: createdCustomFields[tk.id]
          ? { ...tk.customFields, ...createdCustomFields[tk.id] }
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
  // Custom-field defs drive the dynamic statistics breakdowns (B1): every select
  // field with a value list gets its own "distribution by <field>" widget, so a
  // new value list added in the editor shows up in the stats automatically.
  const statFieldDefs = useModuleCustomFields('helpdesk_ticket')
  // Choices per field — bound value lists supply labels AND colours here, so a list
  // created in the editor shows up as a breakdown with its own palette (Darien 2026-08-04).
  const statFieldOptions = useFieldOptions(statFieldDefs)
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

  // Ticket-Intake P6/§7 — origin (Herkunft) sub-tabs: when several creation
  // channels are active, tickets split by where they came in through, with a
  // "zusammenführen" toggle to collapse back to one list.
  const intakeChannels = useHelpdeskStore((s) => s.intakeChannels)
  const [sourceTab, setSourceTab] = useState<'all' | TicketChannel>('all')
  const [mergeSources, setMergeSources] = useState(false)
  const activeChannels = (['agent', 'selfservice', 'external'] as const).filter(
    (c) => intakeChannels[c],
  )
  const showSourceTabs = !mergeSources && activeChannels.length >= 2

  // Detail panel
  const [selectedTicketId, setSelectedTicketId] = useState<string | null>(null)
  const [replyText, setReplyText] = useState('')
  const [showInternalNotes, setShowInternalNotes] = useState(false)

  // New ticket dialog (agent channel). The bound agent form (intakeForms.agent)
  // drives the submitter-facing fields (subject/description/category + extras);
  // the professional tools below (priority/assignee/contact) stay fixed — agents
  // keep contact search, priority and assignment that a plain form can't express.
  const [newTicketOpen, setNewTicketOpen] = useState(false)
  const [ntPriority, setNtPriority] = useState<DisplayTicket['priority']>('medium')
  const [ntAssignee, setNtAssignee] = useState('Marco Hartmann')
  const [ntContact, setNtContact] = useState('')
  // Agent-form answers keyed by form field id (fed through the intake engine on save).
  const [ntFormValues, setNtFormValues] = useState<Record<string, unknown>>({})
  const [ntErrors, setNtErrors] = useState<Record<string, string>>({})
  const agentFormId = useHelpdeskStore((s) => s.intakeForms.agent)
  const { data: agentForm } = useFormSchema(agentFormId)
  // Fields the agent fills: drop page breaks and requester roles (the agent notes
  // the contact via the pro tool, not as a form field).
  const agentFields = useMemo(() => {
    const fields = (agentForm?.fields ?? []) as FormField[]
    return fields.filter(
      (f) => f.label !== '__page_break__' && f.role !== 'requester_name' && f.role !== 'requester_email',
    )
  }, [agentForm])

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
  // Order/width of the list columns — same draft layer, layout half of it.
  const columnLayout = useModuleColumnLayout('helpdesk')
  const ticketTableRef = useRef<HTMLTableElement>(null)
  // Column being dragged right now — drives the live percentage readout, because
  // setting a width by eye needs a number to compare against (Darien 2026-08-05).
  const [resizingColumn, setResizingColumn] = useState<string | null>(null)
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

  // Editor: let the preview follow the left rail (Darien 2026-08-04) — selecting a
  // dimension navigates here to where it is actually visible, instead of making the
  // user hunt for it. Zusatzfelder/Wertelisten live in a ticket detail, statistics
  // in their own tab, Kanäle in the origin sub-tabs on the list.
  useEditorFocusEffect({
    statistik: () => { setSelectedTicketId(null); setTab('statistik') },
    felder: () => { setTab('tickets'); setSelectedTicketId(baseTickets[0]?.id ?? null) },
    // Value sets show densest as chips across the list (status/priority/category/SLA).
    wertelisten: () => { setSelectedTicketId(null); setTab('tickets') },
    bereiche: () => { setSelectedTicketId(null); setTab('tickets') },
    begriffe: () => { setSelectedTicketId(null); setTab('tickets') },
    // Columns are only judgeable on the list itself.
    spalten: () => { setSelectedTicketId(null); setTab('tickets') },
    'kanäle': () => { setSelectedTicketId(null); setTab('tickets') },
  })

  const filteredTickets = useMemo(() => {
    return baseTickets.filter((t) => {
      // Origin sub-tab (Herkunft) — only narrows when the tabs are actually shown.
      if (showSourceTabs && sourceTab !== 'all' && (t.channel ?? 'agent') !== sourceTab)
        return false
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
  }, [baseTickets, statusFilter, priorityFilter, categoryFilter, search, showSourceTabs, sourceTab])

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
    setNtFormValues({}); setNtErrors({})
    setNtPriority('medium'); setNtAssignee('Marco Hartmann'); setNtContact('')
    setNewTicketOpen(true)
  }

  const handleSaveNewTicket = () => {
    // Validate required agent-form fields (the bound template drives them).
    const errs: Record<string, string> = {}
    for (const f of agentFields) {
      const raw = ntFormValues[f.id]
      const missing =
        f.type === 'consent' || f.type === 'checkbox' ? raw !== true : String(raw ?? '').trim() === ''
      if (f.required && missing) errs[f.id] = t('intake.fill.required')
    }
    setNtErrors(errs)
    if (Object.keys(errs).length > 0) return

    // Map the agent-form answers onto a ticket via the shared intake engine, then
    // layer the professional tools (priority/assignee/contact) on top.
    let record: CreateTicketInput
    try {
      record = mapSubmissionToRecord<CreateTicketInput>(
        'helpdesk_ticket',
        (agentForm?.fields ?? []) as FormField[],
        ntFormValues,
        { channel: 'agent' },
      ).record
    } catch {
      toast.error(t('helpdesk.newTicket.createError'))
      return
    }
    if (!record.subject?.trim()) { toast.error(t('helpdesk.newTicket.subjectRequired')); return }

    createTicketMut.mutate(
      {
        ...record,
        priority: displayPriorityToWire(ntPriority),
        assignee_id: ntAssignee || undefined,
        // The agent notes the contact manually; keep it internal (not external) so
        // the new ticket stays visible to the creator (scope=own).
        requester_name: ntContact.trim() || record.requester_name,
        requester_is_external: false,
      },
      {
        onSuccess: () => {
          toast.success(t('helpdesk.newTicket.created', { subject: record.subject }))
          setNewTicketOpen(false)
          setNtFormValues({})
          setNtErrors({})
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

  // ── List columns (Darien 2026-08-04) ────────────────────────────────────────
  // Priority/status are readable straight from the list without opening a ticket;
  // a value list added in the editor had no way to get that same treatment, and the
  // built-in columns had no way to be dropped. Columns are therefore configurable,
  // riding the moduleAreas machinery under a `col:` prefix (draft/deploy/undo come
  // for free). Built-ins default to ON, custom-field columns to OFF — otherwise
  // every new field would silently widen the table.
  const columnDefs: { key: string; header: React.ReactNode; cell: (tk: DisplayTicket) => React.ReactNode; optIn?: boolean }[] = [
    { key: 'ticketNr', header: <EditableText as="span" dkey="helpdesk.table.ticketNr" />, cell: (tk) => <span className="font-mono text-xs text-muted-foreground">{tk.ticketNr}</span> },
    {
      key: 'subject',
      header: <EditableText as="span" dkey="helpdesk.table.subject" />,
      cell: (tk) => (
        <span className="block max-w-[250px] truncate font-medium text-foreground">
          {tk.autoRouted && <Route className="inline h-3 w-3 text-primary mr-1" />}
          {tk.subject}
        </span>
      ),
    },
    {
      key: 'category',
      header: <EditableText as="span" dkey="helpdesk.table.category" />,
      cell: (tk) => tk.category ? (
        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${categoryColors[tk.category] ?? 'bg-secondary text-muted-foreground'}`}>{tk.category}</span>
      ) : null,
    },
    { key: 'priority', header: <EditableText as="span" dkey="helpdesk.table.priority" />, cell: (tk) => <VsChip label={priorityLabels[tk.priority]} color={priorityColorOf(tk.priority)} fallbackClass={priorityColors[tk.priority]} /> },
    // Own key instead of the shared common.status — renaming this column must not
    // retitle every status label in the app.
    { key: 'status', header: <EditableText as="span" dkey="helpdesk.table.status" />, cell: (tk) => <VsChip label={statusLabels[tk.status]} color={statusColorOf(tk.status)} fallbackClass={statusColors[tk.status]} /> },
    { key: 'assignedTo', header: <EditableText as="span" dkey="helpdesk.table.assignedTo" />, cell: (tk) => <span className="text-muted-foreground">{tk.assignedTo}</span> },
    // Renamable AND still explained: the abbreviation tooltip wraps the editable
    // label rather than replacing it.
    { key: 'sla', header: <AbbrTooltip term="SLA"><EditableText as="span" dkey="helpdesk.table.sla" /></AbbrTooltip>, cell: (tk) => <SLABadge overdue={tk.slaOverdue} days={tk.slaDays} hours={tk.slaHours} dueAt={tk.slaDueAt} compact /> },
    { key: 'createdAt', header: <EditableText as="span" dkey="helpdesk.table.createdAt" />, cell: (tk) => <span className="text-xs text-muted-foreground">{formatDate(tk.createdAt)}</span> },
    // One opt-in column per custom field. Select fields bound to a value list render
    // as the same chip as priority/status — that is what "make my new value list
    // show in the overview" means in practice.
    ...statFieldDefs.map((f) => ({
      key: `field:${f.key}`,
      optIn: true,
      header: f.label,
      cell: (tk: DisplayTicket) => {
        const raw = tk.customFields?.[f.key]
        if (raw == null || raw === '') return null
        const opt = statFieldOptions[f.key]?.find((o) => o.id === String(raw))
        return opt
          ? <VsChip label={opt.label} color={opt.color} fallbackClass="bg-secondary text-muted-foreground" />
          : <span className="text-xs text-muted-foreground">{String(raw)}</span>
      },
    })),
  ]
  // Order and width live in the same `col:` draft layer as visibility (Darien
  // 2026-08-05). Unconfigured columns keep their code position, so the list only
  // rearranges once someone actually drags something.
  const visibleColumns = orderColumns(
    columnDefs.filter((c) =>
      c.optIn ? areaEnabled[`col:${c.key}`] === true : areaEnabled[`col:${c.key}`] !== false,
    ),
    columnLayout.layout,
  )
  // `fixed` only once a width exists: with `auto` the browser overrides the width
  // as soon as a cell holds long text, which is exactly what dragging must beat.
  const hasColumnWidths = visibleColumns.some((c) => columnWidthStyle(columnLayout.layout, c.key).width)

  /**
   * Drag a column edge to resize it (editor sandbox only). Stored as a share of
   * the table so the result survives a narrower window; 8% floor keeps a column
   * from being dragged into nothing.
   */
  const startColumnResize = (columnKey: string, event: React.PointerEvent<HTMLElement>): void => {
    event.preventDefault()
    event.stopPropagation()
    const header = event.currentTarget.closest('th')
    const table = ticketTableRef.current
    if (!header || !table) return
    const startX = event.clientX
    const startWidth = header.getBoundingClientRect().width
    const tableWidth = table.getBoundingClientRect().width
    if (tableWidth === 0) return
    // First drag: keep the table exactly as it looks right now. Switching to
    // `table-layout: fixed` gives every unsized column an equal share, so without
    // this the whole list would jump on the first pixel of the first drag.
    if (!hasColumnWidths) {
      const cells = Array.from(table.querySelectorAll('thead th'))
      const frozen: Record<string, number> = {}
      cells.forEach((cell, index) => {
        const column = visibleColumns[index]
        if (column) frozen[column.key] = cell.getBoundingClientRect().width / tableWidth
      })
      columnLayout.freezeWidths(frozen, columnKey)
    }
    setResizingColumn(columnKey)
    const onMove = (moveEvent: PointerEvent): void => {
      const next = (startWidth + moveEvent.clientX - startX) / tableWidth
      columnLayout.setWidth(columnKey, Math.min(0.8, Math.max(0.08, next)))
    }
    const onUp = (): void => {
      setResizingColumn(null)
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }

  // Fill/change a custom-field value on a ticket — instant, flicker-free session
  // buffer (merged over the wire value in the `tickets` memo) so typing feels
  // responsive without a request per keystroke.
  const handleCustomFieldChange = (ticketId: string, key: string, value: string) => {
    setCreatedCustomFields((m) => ({ ...m, [ticketId]: { ...(m[ticketId] ?? {}), [key]: value } }))
  }
  // Persist a custom-field value to the wire ticket (intake P1b). Fired on blur
  // for text/number inputs and immediately for select/checkbox — the commit
  // never runs mid-keystroke, so the input keeps focus through the refetch.
  const handleCustomFieldCommit = (ticketId: string, key: string, value: string) => {
    updateTicketMut.mutate({ id: ticketId, custom_fields: { [key]: value } })
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
          {/* Herkunfts-Reiter (§7): split tickets by origin channel when several
              creation channels are active, with a "zusammenführen" toggle. */}
          {activeChannels.length >= 2 && (
            <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
              {showSourceTabs ? (
                <div className="flex items-center gap-1 rounded-lg bg-secondary/60 p-0.5">
                  {(['all', ...activeChannels] as const).map((src) => {
                    const count =
                      src === 'all'
                        ? baseTickets.length
                        : baseTickets.filter((t) => (t.channel ?? 'agent') === src).length
                    const isActive = sourceTab === src
                    return (
                      <button
                        key={src}
                        onClick={() => setSourceTab(src)}
                        className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                          isActive
                            ? 'bg-card text-foreground shadow-sm'
                            : 'text-muted-foreground hover:text-foreground'
                        }`}
                      >
                        {t(`helpdesk.source.${src}`)}
                        <span className="rounded-full bg-secondary px-1.5 text-[11px] tabular-nums text-muted-foreground">
                          {count}
                        </span>
                      </button>
                    )
                  })}
                </div>
              ) : (
                <span className="text-sm text-muted-foreground">{t('helpdesk.source.mergedHint')}</span>
              )}
              <button
                onClick={() => setMergeSources((m) => !m)}
                role="switch"
                aria-checked={mergeSources}
                className="flex items-center gap-2 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
              >
                <span
                  className={`relative inline-flex h-4 w-7 items-center rounded-full transition-colors ${
                    mergeSources ? 'bg-primary' : 'bg-secondary'
                  }`}
                >
                  <span
                    className={`inline-block h-3 w-3 transform rounded-full bg-white transition-transform ${
                      mergeSources ? 'translate-x-3.5' : 'translate-x-0.5'
                    }`}
                  />
                </span>
                {t('helpdesk.source.merge')}
              </button>
            </div>
          )}
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
              <table
                ref={ticketTableRef}
                className="w-full text-sm"
                style={hasColumnWidths ? { tableLayout: 'fixed' } : undefined}
              >
                <thead>
                  <tr className="border-b border-border">
                    {visibleColumns.map((col) => (
                      <th
                        key={col.key}
                        style={columnWidthStyle(columnLayout.layout, col.key)}
                        className={`relative px-4 py-3 text-left text-xs font-medium text-muted-foreground${
                          columnLayout.showDividers ? ' bg-[var(--accent-1)]/[0.04]' : ''
                        }`}
                      >
                        <span className={hasColumnWidths ? 'block truncate' : undefined}>{col.header}</span>
                        {/* Resize grip — editor sandbox only: in the live app the
                            list is operated, not configured. It only becomes VISIBLE
                            while the Spalten section is open, because on a white
                            header you cannot tell where the edge is (Darien). */}
                        {columnLayout.editing && (
                          <span
                            role="separator"
                            aria-orientation="vertical"
                            aria-label={t('customization.editor.spalten.resize', { column: col.key })}
                            data-column-resize={col.key}
                            onPointerDown={(e) => startColumnResize(col.key, e)}
                            className={`group absolute inset-y-1 right-0 z-10 flex w-3 cursor-col-resize items-center justify-center ${
                              columnLayout.showDividers ? '' : 'opacity-0'
                            }`}
                          >
                            <span className="h-full w-px bg-[var(--accent-1)]/25 transition-colors group-hover:w-0.5 group-hover:bg-[var(--accent-1)]" />
                            <GripVertical
                              className="pointer-events-none absolute h-3 w-3 text-[var(--accent-1)]/50 transition-colors group-hover:text-[var(--accent-1)]"
                              aria-hidden="true"
                            />
                          </span>
                        )}
                        {/* Live readout while dragging: the panel carries the saved
                            value, but the eye is on the table — without a number
                            here you are setting a width blind. */}
                        {resizingColumn === col.key && (
                          <span className="pointer-events-none absolute -top-1 right-1 z-20 rounded bg-[var(--accent-1)] px-1.5 py-0.5 text-[10px] font-semibold tabular-nums text-white shadow-sm">
                            {Math.round((columnLayout.layout[`col:${col.key}`]?.width ?? 0) * 100)} %
                          </span>
                        )}
                      </th>
                    ))}
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
                      {visibleColumns.map((col) => (
                        // Once widths are fixed, a long value must clip instead of
                        // pushing the column back open.
                        <td key={col.key} className={`px-4 py-3${hasColumnWidths ? ' truncate' : ''}`}>
                          {col.cell(ticket)}
                        </td>
                      ))}
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
        // A missing key means visible. CSAT now reads real ratings off the wire
        // (intake P0/P3), so the card + chart are enabled; the tenant on/off + delay
        // survey setting lands with the Helpdesk settings step.
        const showStat = (key: string): boolean => areaEnabled[`stat:${key}`] !== false
        const CSAT_FEATURE_ENABLED = csatEnabled
        // Generic value-list breakdown (B1): one row per option, label + colour
        // pulled from the value set, so renaming/adding options in the editor is
        // reflected here live. Used by status/priority AND every select custom field.
        const renderBreakdown = (
          keyId: string,
          heading: React.ReactNode,
          rows: { id: string; label: string; color?: string; fallbackClass?: string; count: number }[],
        ) => (
          <div key={keyId} className="rounded-lg border border-border bg-card p-4">
            <h3 className="text-sm font-medium text-foreground mb-3">{heading}</h3>
            <div className="space-y-2">
              {rows.map((r) => {
                const pct = tickets.length > 0 ? Math.round((r.count / tickets.length) * 100) : 0
                return (
                  <div key={r.id} className="flex items-center gap-3">
                    <VsChip label={r.label} color={r.color} fallbackClass={r.fallbackClass} className="w-28 text-center" />
                    <div className="flex-1 h-2 rounded-full bg-secondary overflow-hidden">
                      <div className="h-full rounded-full bg-primary/70 transition-all" style={{ width: `${pct}%` }} />
                    </div>
                    <span className="text-xs text-muted-foreground w-10 text-right">{r.count}</span>
                  </div>
                )
              })}
            </div>
          </div>
        )
        // Dynamic breakdowns: every select custom field with options becomes a
        // "distribution by <field>" widget (togglable via stat:field:<key>).
        const fieldBreakdowns = statFieldDefs
          .filter((f) => f.type === 'select' && (statFieldOptions[f.key]?.length ?? 0) > 0)
          .map((f) =>
            showStat(`field:${f.key}`)
              ? renderBreakdown(
                  `f-${f.key}`,
                  t('helpdesk.stats.byField', { field: f.label }),
                  (statFieldOptions[f.key] ?? []).map((opt) => ({
                    id: opt.id,
                    label: opt.label,
                    color: opt.color,
                    count: tickets.filter(
                      (tk) => tk.customFields?.[f.key] != null && String(tk.customFields[f.key]) === opt.id,
                    ).length,
                  })),
                )
              : false,
          )
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
          showStat('byStatus') && renderBreakdown(
            'bs',
            <EditableText as="span" dkey="helpdesk.stats.byStatus" />,
            activeStatusOptions.map((o) => ({
              id: o.id, label: o.label, color: o.color, fallbackClass: statusColors[o.id],
              count: tickets.filter((tk) => tk.status === o.id).length,
            })),
          ),
          showStat('byPriority') && renderBreakdown(
            'bp',
            <EditableText as="span" dkey="helpdesk.stats.byPriority" />,
            activePriorityOptions.map((o) => ({
              id: o.id, label: o.label, color: o.color, fallbackClass: priorityColors[o.id],
              count: tickets.filter((tk) => tk.priority === o.id).length,
            })),
          ),
          ...fieldBreakdowns,
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
          onCustomFieldCommit={handleCustomFieldCommit}
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
              {/* Submitter-facing fields from the bound agent form (intakeForms.agent). */}
              {agentFields.length > 0 ? (
                <IntakeFieldInputs
                  fields={agentFields}
                  values={ntFormValues}
                  errors={ntErrors}
                  onChange={(id, v) => setNtFormValues((m) => ({ ...m, [id]: v }))}
                />
              ) : (
                <p className="rounded-lg bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                  {t('helpdesk.newTicket.noAgentForm')}
                </p>
              )}

              {/* Internal tools — agents keep contact search, priority and assignment.
                  These are not part of the form and never shown to self-service. */}
              <div className="space-y-4 border-t border-border pt-4">
                <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{t('helpdesk.newTicket.internalSection')}</p>
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
              </div>
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
  def, value, onChange, onCommit, options,
}: {
  def: CustomFieldDefinition
  value: string
  onChange: (v: string) => void
  /** Persist the value (intake P1b). Discrete controls (select/checkbox) commit
   *  on change; free-text inputs commit on blur so typing isn't interrupted. */
  onCommit?: (v: string) => void
  /** Resolved choices — from a bound value list or the field's own options
   *  (useFieldOptions). Falls back to the raw options when not supplied. */
  options?: { id: string; label: string; color?: string }[]
}) {
  const { t } = useTranslation()
  const inputCls = 'w-full rounded-lg border border-border bg-card px-2.5 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring'
  if (def.type === 'boolean') {
    return (
      <label className="flex items-center gap-2 text-sm text-foreground">
        <input type="checkbox" checked={value === 'true'} onChange={(e) => { const v = e.target.checked ? 'true' : 'false'; onChange(v); onCommit?.(v) }} className="h-4 w-4 rounded border-border" />
        {value === 'true' ? t('common.yes') : t('common.no')}
      </label>
    )
  }
  const choices = options ?? def.options.map((o) => ({ id: o, label: o }))
  if (def.type === 'select' && choices.length > 0) {
    return (
      <div className="relative">
        <select value={value} onChange={(e) => { onChange(e.target.value); onCommit?.(e.target.value) }} className={`${inputCls} appearance-none pr-7 cursor-pointer`}>
          <option value="">{t('helpdesk.newTicket.selectPlaceholder')}</option>
          {choices.map((o) => <option key={o.id} value={o.id}>{o.label}</option>)}
        </select>
        <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
      </div>
    )
  }
  const inputType = def.type === 'number' ? 'number' : def.type === 'email' ? 'email' : def.type === 'url' ? 'url' : def.type === 'date' ? 'date' : 'text'
  return <input type={inputType} value={value} onChange={(e) => onChange(e.target.value)} onBlur={(e) => onCommit?.(e.target.value)} placeholder={t('helpdesk.ticket.customFieldPlaceholder')} className={inputCls} />
}

function TicketDetailPanel({
  ticket, allTickets, replyText, onReplyChange, showInternalNotes, onToggleInternal,
  onSendReply, onStatusChange, onClose, onCustomFieldChange, onCustomFieldCommit, assignTicketMut, updateTicketMut, mergeTicketsMut,
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
  onCustomFieldCommit: (ticketId: string, key: string, value: string) => void
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
  // A select field may draw its choices from a shared value list — then renaming an
  // option in the Wertelisten panel reaches this dropdown live.
  const customFieldOptions = useFieldOptions(customFieldDefs)

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
                    options={customFieldOptions[def.key]}
                    value={String(ticket.customFields?.[def.key] ?? '')}
                    onChange={(v) => onCustomFieldChange(ticket.id, def.key, v)}
                    onCommit={(v) => onCustomFieldCommit(ticket.id, def.key, v)}
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
