import { useState, Fragment } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  Plus,
  Phone,
  Mail,
  MessageSquare,
  Building2,
  User,
  Users,
  Briefcase,
  Star,
  Pencil,
  Trash2,
  Copy,
  Download,
  FileDown,
  FolderOpen,
  FolderPlus,
  Settings2,
  Loader2,
  List,
  LayoutGrid,
  Clock,
} from 'lucide-react'
import { toast } from 'sonner'
import { useContacts, useCreateContact, useUpdateContact, useDeleteContact } from '@/api/hooks/useContacts'
import { useContactsStore, type Contact } from '@/stores/contacts'
import { useNavigationStore } from '@/stores/navigation'
import { useMeetingsStore } from '@/stores/meetings'
import { ItemActions, ConfirmDialog, EmptyState, SortMenu, type ActionItem, type SortDirection } from '@/components/shared'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { ContactFormDialog } from './ContactFormDialog'
import { ContactDetailPanel } from './ContactDetailPanel'
import { ImportContactsDialog } from './ImportContactsDialog'
import { GroupManagerDialog } from './GroupManagerDialog'
import { GroupAssignDialog } from './GroupAssignDialog'
import { CallOverlay } from '@/modules/meetings/CallOverlay'
import { backendContactToUI, uiFormToCreateRequest, uiFormToUpdateRequest } from './adapters'
import { formatDate } from '@/lib/format'

type CategoryFilter = 'all' | 'employee' | 'customer' | 'partner' | `group:${string}`
type SortField = 'name' | 'company' | 'lastContact'
type ViewMode = 'list' | 'grid'

const _categoryLabels: Record<string, string> = {
  employee: 'employee',
  customer: 'customer',
  partner: 'partner',
}

const statusColors: Record<string, string> = {
  active: 'bg-success-light text-success',
  prospect: 'bg-warning-light text-warning',
  inactive: 'bg-secondary text-text-disabled',
}

const statusLabelKeys: Record<string, string> = {
  active: 'kontakte.status.active',
  prospect: 'kontakte.status.prospect',
  inactive: 'kontakte.status.inactive',
}

export default function KontaktePage() {
  const { t } = useTranslation()
  const { groups, favoriteIds, toggleFavorite } = useContactsStore()
  const { startCall, endCall, activeCallContactId, activeCallContactName } = useMeetingsStore()
  const navigate = useNavigate()
  const setIntent = useNavigationStore((s) => s.setIntent)

  // React Query
  const { data: contactsData, isLoading, isError } = useContacts()
  const createContactMutation = useCreateContact()
  const updateContactMutation = useUpdateContact()
  const deleteContactMutation = useDeleteContact()

  // Map backend contacts to UI contacts, overlaying local favoriteIds
  const contacts: Contact[] = (contactsData?.contacts ?? [])
    .map(backendContactToUI)
    .map((c) => ({ ...c, isFavorite: favoriteIds.includes(c.id) }))

  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('all')
  const [sortField, setSortField] = useState<SortField>('name')
  const [sortDir, setSortDir] = useState<SortDirection>('asc')
  const [viewMode, setViewMode] = useState<ViewMode>('list')

  // Modal state — which contact is open in the detail modal
  const [detailContactId, setDetailContactId] = useState<string | null>(null)

  // Dialog state
  const [formOpen, setFormOpen] = useState(false)
  const [editContact, setEditContact] = useState<Contact | null>(null)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)
  const [importOpen, setImportOpen] = useState(false)
  const [groupManagerOpen, setGroupManagerOpen] = useState(false)
  const [groupAssignContact, setGroupAssignContact] = useState<Contact | null>(null)

  const filtered = contacts
    .filter((c) => {
      if (categoryFilter.startsWith('group:')) {
        const groupId = categoryFilter.slice(6)
        const group = groups.find((g) => g.id === groupId)
        if (group && !group.contactIds.includes(c.id)) return false
      } else if (categoryFilter !== 'all' && c.category !== categoryFilter) {
        return false
      }
      if (search) {
        const q = search.toLowerCase()
        return (
          c.firstName.toLowerCase().includes(q) ||
          c.lastName.toLowerCase().includes(q) ||
          c.company.toLowerCase().includes(q) ||
          c.email.toLowerCase().includes(q) ||
          c.tags.some((tag) => tag.toLowerCase().includes(q))
        )
      }
      return true
    })
    .sort((a, b) => {
      let cmp: number
      if (sortField === 'name') cmp = `${a.lastName} ${a.firstName}`.localeCompare(`${b.lastName} ${b.firstName}`)
      else if (sortField === 'company') cmp = a.company.localeCompare(b.company)
      else cmp = new Date(a.lastContact).getTime() - new Date(b.lastContact).getTime()
      return sortDir === 'asc' ? cmp : -cmp
    })

  const detailContact = contacts.find((c) => c.id === detailContactId) || null
  const deleteTarget = contacts.find((c) => c.id === deleteConfirmId)

  const sortFieldLabel = (f: SortField) =>
    f === 'name' ? t('kontakte.sort.name') : f === 'company' ? t('kontakte.sort.company') : t('kontakte.sort.lastContact')

  const categories: { key: CategoryFilter; label: string; icon: typeof User; count: number }[] = [
    { key: 'all', label: t('kontakte.category.all'), icon: Users, count: contacts.length },
    { key: 'employee', label: t('kontakte.category.employee'), icon: User, count: contacts.filter((c) => c.category === 'employee').length },
    { key: 'customer', label: t('kontakte.category.customers'), icon: Briefcase, count: contacts.filter((c) => c.category === 'customer').length },
    { key: 'partner', label: t('kontakte.category.partner'), icon: Building2, count: contacts.filter((c) => c.category === 'partner').length },
  ]

  const handleCreateSubmit = async (data: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>) => {
    if (editContact) {
      await updateContactMutation.mutateAsync({
        id: editContact.id,
        ...uiFormToUpdateRequest(data),
      })
      toast.success(t('kontakte.toast.contactUpdated'))
    } else {
      await createContactMutation.mutateAsync(uiFormToCreateRequest(data))
      toast.success(t('kontakte.toast.contactCreated'))
    }
    setEditContact(null)
  }

  const handleDelete = async () => {
    if (deleteConfirmId) {
      await deleteContactMutation.mutateAsync(deleteConfirmId)
      toast.success(t('kontakte.toast.contactDeleted'))
      if (detailContactId === deleteConfirmId) setDetailContactId(null)
      setDeleteConfirmId(null)
    }
  }

  const handleEmail = (contact: Contact) => {
    setIntent({
      type: 'compose-email',
      data: { to: contact.email, name: `${contact.firstName} ${contact.lastName}` },
    })
    navigate('/mails')
    toast.info(t('kontakte.toast.emailTo', { name: `${contact.firstName} ${contact.lastName}` }))
  }

  const handleCall = (contact: Contact) => {
    startCall(contact.id, `${contact.firstName} ${contact.lastName}`)
    toast.info(t('kontakte.toast.callTo', { name: `${contact.firstName} ${contact.lastName}` }))
  }

  const handleMessage = (contact: Contact) => {
    setIntent({
      type: 'send-message',
      data: { contactId: contact.id, name: `${contact.firstName} ${contact.lastName}` },
    })
    navigate('/chat')
    toast.info(t('kontakte.toast.chatWith', { name: `${contact.firstName} ${contact.lastName}` }))
  }

  const handleExportVCard = (contact: Contact) => {
    toast.success(t('kontakte.toast.vcardExported', { name: `${contact.firstName} ${contact.lastName}` }))
  }

  const handleDuplicate = async (contact: Contact) => {
    await createContactMutation.mutateAsync(
      uiFormToCreateRequest({ ...contact, firstName: `${contact.firstName} (${t('kontakte.action.copy')})`, isFavorite: false })
    )
    toast.success(t('kontakte.toast.contactDuplicated'))
  }

  const getContactActions = (c: Contact): ActionItem[] => [
    {
      label: t('common.edit'),
      icon: Pencil,
      onClick: () => {
        setEditContact(c)
        setFormOpen(true)
      },
    },
    {
      label: t('kontakte.action.duplicate'),
      icon: Copy,
      onClick: () => handleDuplicate(c),
    },
    {
      label: c.isFavorite ? t('kontakte.action.unfavorite') : t('kontakte.action.favorite'),
      icon: Star,
      onClick: () => {
        toggleFavorite(c.id)
        toast.success(c.isFavorite ? t('kontakte.toast.removedFromFavorites') : t('kontakte.toast.addedToFavorites'))
      },
    },
    {
      label: t('kontakte.action.assignToGroup'),
      icon: FolderPlus,
      onClick: () => setGroupAssignContact(c),
    },
    {
      label: t('kontakte.action.sendEmail'),
      icon: Mail,
      onClick: () => handleEmail(c),
      separator: true,
    },
    {
      label: t('kontakte.action.call'),
      icon: Phone,
      onClick: () => handleCall(c),
    },
    {
      label: t('kontakte.action.message'),
      icon: MessageSquare,
      onClick: () => handleMessage(c),
    },
    {
      label: t('kontakte.action.exportVCard'),
      icon: Download,
      onClick: () => handleExportVCard(c),
      separator: true,
    },
    {
      label: t('common.delete'),
      icon: Trash2,
      variant: 'destructive',
      onClick: () => setDeleteConfirmId(c.id),
      separator: true,
    },
  ]

  return (
    <div className="flex h-full overflow-hidden animate-fade-in">
      {/* Category sidebar */}
      <aside className="w-52 shrink-0 border-r border-border bg-card p-3 overflow-y-auto">
        <h3 className="text-sm font-medium text-foreground mb-3 px-2">{t('kontakte.sidebar.categories')}</h3>
        <nav className="space-y-0.5">
          {categories.map((cat) => {
            const Icon = cat.icon
            return (
              <button
                key={cat.key}
                onClick={() => setCategoryFilter(cat.key)}
                className={`flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition-colors ${
                  categoryFilter === cat.key
                    ? 'bg-primary-light text-primary font-medium'
                    : 'text-foreground hover:bg-secondary'
                }`}
              >
                <Icon className="h-4 w-4 shrink-0" />
                <span className="flex-1 text-left">{cat.label}</span>
                <span className="text-xs text-muted-foreground badge-accent">{cat.count}</span>
              </button>
            )
          })}
        </nav>

        {/* Groups section */}
        {groups.length > 0 && (
          <div className="mt-6">
            <div className="flex items-center justify-between px-2 mb-2">
              <h3 className="text-sm font-medium text-foreground flex items-center gap-1.5">
                <FolderOpen className="h-3.5 w-3.5" />
                {t('kontakte.sidebar.groups')}
              </h3>
              <button
                onClick={() => setGroupManagerOpen(true)}
                className="rounded p-0.5 text-muted-foreground hover:text-foreground transition-colors"
                title={t('kontakte.sidebar.manageGroups')}
              >
                <Settings2 className="h-3.5 w-3.5" />
              </button>
            </div>
            <nav className="space-y-0.5">
              {groups.map((g) => (
                <button
                  key={g.id}
                  onClick={() => setCategoryFilter(`group:${g.id}`)}
                  className={`flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-sm transition-colors ${
                    categoryFilter === `group:${g.id}`
                      ? 'bg-primary-light text-primary font-medium'
                      : 'text-foreground hover:bg-secondary'
                  }`}
                >
                  <span className="h-2.5 w-2.5 rounded-full shrink-0" style={{ backgroundColor: g.color }} />
                  <span className="flex-1 text-left truncate">{g.name}</span>
                  <span className="text-xs text-muted-foreground">{g.contactIds.length}</span>
                </button>
              ))}
            </nav>
          </div>
        )}

        {/* Favorites section */}
        {contacts.some((c) => c.isFavorite) && (
          <div className="mt-6">
            <h3 className="text-sm font-medium text-foreground mb-3 px-2 flex items-center gap-1.5">
              <Star className="h-3.5 w-3.5 text-yellow-500" />
              {t('kontakte.sidebar.favorites')}
            </h3>
            <nav className="space-y-0.5">
              {contacts
                .filter((c) => c.isFavorite)
                .map((c) => (
                  <button
                    key={c.id}
                    onClick={() => setDetailContactId(c.id)}
                    className="flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-sm transition-colors text-foreground hover:bg-secondary"
                  >
                    <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-light text-[10px] font-medium text-primary">
                      {c.initials}
                    </span>
                    <span className="truncate">{c.firstName} {c.lastName}</span>
                  </button>
                ))}
            </nav>
          </div>
        )}
      </aside>

      {/* Main content — full width */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Toolbar: Search (fills) + controls cluster (right) */}
        <div className="flex items-center gap-3 border-b border-border px-4 py-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder={t('kontakte.search.placeholder')}
              aria-label={t('kontakte.search.placeholder')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-background pl-9 pr-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Right-aligned controls cluster */}
          <div className="flex items-center gap-2 shrink-0">

          {/* Sort — field + direction (reusable) */}
          <SortMenu
            options={[
              { value: 'name', label: t('kontakte.sort.name') },
              { value: 'company', label: t('kontakte.sort.company') },
              { value: 'lastContact', label: t('kontakte.sort.lastContact') },
            ]}
            field={sortField}
            direction={sortDir}
            onChange={(f, d) => { setSortField(f as SortField); setSortDir(d) }}
          />

          {/* View mode toggle — segmented control */}
          <div
            role="group"
            aria-label={t('kontakte.view.toggleAriaLabel')}
            className="flex rounded-lg border border-border overflow-hidden"
          >
            <button
              onClick={() => setViewMode('list')}
              className={`flex items-center gap-1 px-2.5 py-1.5 text-xs transition-colors ${
                viewMode === 'list'
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground hover:bg-secondary'
              }`}
              aria-pressed={viewMode === 'list'}
              title={t('kontakte.view.list')}
            >
              <List className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">{t('kontakte.view.list')}</span>
            </button>
            <button
              onClick={() => setViewMode('grid')}
              className={`flex items-center gap-1 px-2.5 py-1.5 text-xs transition-colors border-l border-border ${
                viewMode === 'grid'
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground hover:bg-secondary'
              }`}
              aria-pressed={viewMode === 'grid'}
              title={t('kontakte.view.grid')}
            >
              <LayoutGrid className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">{t('kontakte.view.grid')}</span>
            </button>
          </div>

          <button
            onClick={() => setImportOpen(true)}
            className="rounded-lg border border-border p-1.5 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
            title={t('kontakte.action.importContacts')}
            aria-label={t('kontakte.action.importContacts')}
          >
            <FileDown className="h-4 w-4" />
          </button>
          <button
            onClick={() => {
              setEditContact(null)
              setFormOpen(true)
            }}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            <span>{t('kontakte.action.new')}</span>
          </button>
          </div>
        </div>

        {/* Count + sort indicator */}
        <div className="flex items-center gap-2 px-4 py-1.5 border-b border-border-muted text-xs text-muted-foreground">
          <span>{isLoading ? '…' : filtered.length} {t('kontakte.list.contacts')}</span>
          <span>·</span>
          <span>
            {t('kontakte.sort.sortedBy', { field: sortFieldLabel(sortField) })}
          </span>
        </div>

        {/* Contact list / grid */}
        <div className="flex-1 overflow-y-auto">
          {isLoading && (
            <div className="flex items-center justify-center py-16 text-muted-foreground gap-2">
              <Loader2 className="h-5 w-5 animate-spin" />
              <span className="text-sm">{t('kontakte.list.loading')}</span>
            </div>
          )}

          {isError && !isLoading && (
            <div className="flex flex-col items-center justify-center py-16 gap-2">
              <p className="text-sm text-destructive">{t('kontakte.error.backendUnavailable')}</p>
              <p className="text-xs text-muted-foreground">{t('kontakte.error.startGateway')}</p>
            </div>
          )}

          {!isLoading && !isError && filtered.length === 0 && (
            <EmptyState
              icon={Users}
              title={t('kontakte.empty.title')}
              description={
                search
                  ? t('kontakte.empty.tryDifferentSearch')
                  : t('kontakte.empty.createFirst')
              }
              action={
                !search
                  ? {
                      label: t('kontakte.action.newContact'),
                      onClick: () => {
                        setEditContact(null)
                        setFormOpen(true)
                      },
                    }
                  : undefined
              }
            />
          )}

          {/* ---------------------------------------------------------------- */}
          {/* LIST VIEW                                                         */}
          {/* ---------------------------------------------------------------- */}
          {!isLoading && !isError && filtered.length > 0 && viewMode === 'list' && (
            <div>
              {filtered.map((contact, idx) => {
                const letter = (contact.lastName?.[0] ?? '#').toUpperCase()
                const prevLetter = idx > 0 ? (filtered[idx - 1].lastName?.[0] ?? '#').toUpperCase() : null
                const showDivider = sortField === 'name' && letter !== prevLetter
                const phone = contact.phone || contact.mobile
                return (
                  <Fragment key={contact.id}>
                    {showDivider && (
                      <div className="sticky top-0 z-10 border-b border-border-muted bg-secondary/80 px-4 py-1 text-xs font-semibold text-muted-foreground backdrop-blur">
                        {letter}
                      </div>
                    )}
                    <div
                      role="button"
                      tabIndex={0}
                      className="group flex w-full items-center gap-3 border-b border-border-muted px-4 py-3 text-left transition-colors hover:bg-secondary/50 cursor-pointer focus-visible:outline-none focus-visible:bg-secondary/50"
                      onClick={() => setDetailContactId(contact.id)}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setDetailContactId(contact.id) } }}
                    >
                      <div className="relative flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary-light text-sm font-medium text-primary">
                        {contact.initials}
                        {contact.isFavorite && (
                          <Star className="absolute -top-1 -right-1 h-3 w-3 fill-yellow-400 text-yellow-400" />
                        )}
                      </div>
                      {/* Name + company */}
                      <div className="min-w-0 flex-1 sm:flex-none sm:w-52 lg:w-64">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-foreground truncate">
                            {contact.firstName} {contact.lastName}
                          </span>
                          <span className={`inline-flex shrink-0 rounded-full px-1.5 py-0.5 text-[10px] ${statusColors[contact.status]}`}>
                            {t(statusLabelKeys[contact.status])}
                          </span>
                        </div>
                        <p className="text-xs text-muted-foreground truncate">
                          {contact.jobTitle}{contact.jobTitle && contact.company ? ' · ' : ''}{contact.company}
                        </p>
                      </div>
                      {/* Contact info — fills the middle */}
                      <div className="hidden min-w-0 flex-1 flex-col justify-center gap-0.5 text-xs text-muted-foreground sm:flex">
                        {contact.email && (
                          <span className="flex items-center gap-1.5 truncate">
                            <Mail className="h-3 w-3 shrink-0" />
                            <span className="truncate">{contact.email}</span>
                          </span>
                        )}
                        {phone && (
                          <span className="flex items-center gap-1.5 truncate">
                            <Phone className="h-3 w-3 shrink-0" />
                            <span className="truncate">{phone}</span>
                          </span>
                        )}
                      </div>
                      {/* Last contact */}
                      <div
                        className="hidden w-28 shrink-0 items-center justify-end gap-1.5 text-xs text-muted-foreground lg:flex"
                        title={t('kontakte.sort.lastContact')}
                      >
                        <Clock className="h-3 w-3 shrink-0" />
                        {formatDate(contact.lastContact)}
                      </div>
                      {/* Quick-action pills — visible on hover */}
                      <div className="hidden group-hover:flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                        <button
                          onClick={() => handleCall(contact)}
                          className="rounded-md border border-border p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
                          title={t('kontakte.action.call')}
                        >
                          <Phone className="h-3.5 w-3.5" />
                        </button>
                        <button
                          onClick={() => handleEmail(contact)}
                          className="rounded-md border border-border p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
                          title={t('kontakte.action.sendEmail')}
                        >
                          <Mail className="h-3.5 w-3.5" />
                        </button>
                        <div onClick={(e) => e.stopPropagation()}>
                          <ItemActions items={getContactActions(contact)} />
                        </div>
                      </div>
                      {/* Fallback: three-dot menu only (no hover on touch) */}
                      <div className="group-hover:hidden shrink-0" onClick={(e) => e.stopPropagation()}>
                        <ItemActions items={getContactActions(contact)} />
                      </div>
                    </div>
                  </Fragment>
                )
              })}
            </div>
          )}

          {/* ---------------------------------------------------------------- */}
          {/* GRID VIEW                                                         */}
          {/* ---------------------------------------------------------------- */}
          {!isLoading && !isError && filtered.length > 0 && viewMode === 'grid' && (
            <div className="p-4 grid gap-3 grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
              {filtered.map((contact) => (
                <div
                  key={contact.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => setDetailContactId(contact.id)}
                  onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setDetailContactId(contact.id) } }}
                  className="group relative flex flex-col items-center gap-2.5 overflow-hidden rounded-xl border border-border bg-card p-4 text-center transition-colors hover:bg-secondary/60 hover:border-primary/30 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
                >
                  {/* Favorite star */}
                  {contact.isFavorite && (
                    <Star className="absolute top-2 right-2 h-3.5 w-3.5 fill-yellow-400 text-yellow-400" />
                  )}

                  {/* Avatar */}
                  <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-primary-light text-base font-medium text-primary">
                    {contact.initials}
                  </div>

                  {/* Name */}
                  <div className="w-full min-w-0">
                    <p className="text-sm font-semibold text-foreground truncate leading-tight">
                      {contact.firstName} {contact.lastName}
                    </p>
                    {contact.jobTitle || contact.company ? (
                      <p className="text-[11px] text-muted-foreground truncate mt-0.5">
                        {contact.jobTitle}
                        {contact.jobTitle && contact.company ? ' · ' : ''}
                        {contact.company}
                      </p>
                    ) : null}
                  </div>

                  {/* Status badge */}
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-[10px] leading-tight ${statusColors[contact.status]}`}>
                    {t(statusLabelKeys[contact.status])}
                  </span>

                  {/* Contact quick-info */}
                  <div className="w-full min-w-0 space-y-1">
                    {contact.email && (
                      <p className="flex items-center gap-1 text-[11px] text-muted-foreground truncate">
                        <Mail className="h-3 w-3 shrink-0" />
                        <span className="truncate">{contact.email}</span>
                      </p>
                    )}
                    {(contact.phone || contact.mobile) && (
                      <p className="flex items-center gap-1 text-[11px] text-muted-foreground truncate">
                        <Phone className="h-3 w-3 shrink-0" />
                        <span className="truncate">{contact.phone || contact.mobile}</span>
                      </p>
                    )}
                  </div>

                  {/* Tags (max 2 visible) */}
                  {contact.tags.length > 0 && (
                    <div className="flex flex-wrap justify-center gap-1 w-full">
                      {contact.tags.slice(0, 2).map((tag) => (
                        <span
                          key={tag}
                          className="rounded-full border border-border bg-secondary px-1.5 py-0.5 text-[10px] text-foreground truncate max-w-[70px]"
                        >
                          {tag}
                        </span>
                      ))}
                      {contact.tags.length > 2 && (
                        <span className="rounded-full border border-border bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground">
                          +{contact.tags.length - 2}
                        </span>
                      )}
                    </div>
                  )}

                  {/* Hover quick-actions — overlay at the bottom, no layout shift */}
                  <div
                    className="absolute inset-x-0 bottom-0 hidden items-center justify-center gap-1 border-t border-border bg-card/95 px-2 py-2 backdrop-blur-sm group-hover:flex group-focus-within:flex"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <button
                      onClick={() => handleCall(contact)}
                      className="rounded-md border border-border p-1 text-muted-foreground hover:text-foreground hover:bg-background transition-colors"
                      title={t('kontakte.action.call')}
                    >
                      <Phone className="h-3.5 w-3.5" />
                    </button>
                    <button
                      onClick={() => handleEmail(contact)}
                      className="rounded-md border border-border p-1 text-muted-foreground hover:text-foreground hover:bg-background transition-colors"
                      title={t('kontakte.action.sendEmail')}
                    >
                      <Mail className="h-3.5 w-3.5" />
                    </button>
                    <button
                      onClick={() => handleMessage(contact)}
                      className="rounded-md border border-border p-1 text-muted-foreground hover:text-foreground hover:bg-background transition-colors"
                      title={t('kontakte.action.message')}
                    >
                      <MessageSquare className="h-3.5 w-3.5" />
                    </button>
                    <ItemActions items={getContactActions(contact)} />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* -------------------------------------------------------------------- */}
      {/* Contact Detail Modal                                                  */}
      {/* -------------------------------------------------------------------- */}
      <Dialog
        open={!!detailContact}
        onOpenChange={(open) => { if (!open) setDetailContactId(null) }}
      >
        <DialogContent showCloseButton={false} className="max-w-4xl p-0 overflow-hidden max-h-[90vh] flex flex-col rounded-2xl">
          {detailContact && (
            <ContactDetailPanel
              contact={detailContact}
              onClose={() => setDetailContactId(null)}
              onEdit={(c) => {
                setEditContact(c)
                setFormOpen(true)
              }}
              onDelete={(id) => {
                setDeleteConfirmId(id)
                setDetailContactId(null)
              }}
              onEmail={handleEmail}
              onCall={handleCall}
              onMessage={handleMessage}
              onToggleFavorite={toggleFavorite}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Form Dialog */}
      <ContactFormDialog
        open={formOpen}
        onOpenChange={(open) => {
          setFormOpen(open)
          if (!open) setEditContact(null)
        }}
        contact={editContact}
        onSubmit={handleCreateSubmit}
      />

      {/* Delete Confirm */}
      <ConfirmDialog
        open={!!deleteConfirmId}
        onOpenChange={(open) => !open && setDeleteConfirmId(null)}
        title={t('kontakte.confirm.deleteTitle')}
        description={t('kontakte.confirm.deleteDescription', { name: deleteTarget ? `${deleteTarget.firstName} ${deleteTarget.lastName}` : '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />

      {/* Import Dialog */}
      <ImportContactsDialog
        open={importOpen}
        onOpenChange={setImportOpen}
        onImport={async (data) => {
          const results = await Promise.allSettled(
            data.map((c) => createContactMutation.mutateAsync(uiFormToCreateRequest(c as Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>)))
          )
          const succeeded = results.filter((r) => r.status === 'fulfilled').length
          toast.success(t('kontakte.toast.contactsImported', { count: succeeded }))
          return succeeded
        }}
      />

      {/* Group Manager */}
      <GroupManagerDialog
        open={groupManagerOpen}
        onOpenChange={setGroupManagerOpen}
      />

      {/* Assign contact to groups */}
      <GroupAssignDialog
        contact={groupAssignContact}
        open={!!groupAssignContact}
        onOpenChange={(o) => { if (!o) setGroupAssignContact(null) }}
        onManageGroups={() => setGroupManagerOpen(true)}
      />

      {/* Call Overlay */}
      <CallOverlay
        contactName={activeCallContactName || ''}
        open={!!activeCallContactId}
        onEnd={() => {
          endCall()
          toast.info(t('kontakte.toast.callEnded'))
        }}
      />
    </div>
  )
}
