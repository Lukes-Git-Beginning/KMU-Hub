import { useState, useEffect } from 'react'
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
  ArrowUpDown,
} from 'lucide-react'
import { toast } from 'sonner'
import { useContactsStore, type Contact } from '@/stores/contacts'
import { useNavigationStore } from '@/stores/navigation'
import { useMeetingsStore } from '@/stores/meetings'
import { ItemActions, ConfirmDialog, EmptyState, type ActionItem } from '@/components/shared'
import { ContactFormDialog } from './ContactFormDialog'
import { ContactDetailPanel } from './ContactDetailPanel'
import { CallOverlay } from '@/modules/meetings/CallOverlay'

type CategoryFilter = 'all' | 'employee' | 'customer' | 'partner'
type SortField = 'name' | 'company' | 'lastContact'

const categoryLabels: Record<string, string> = {
  employee: 'Mitarbeiter',
  customer: 'Kunde',
  partner: 'Partner',
}

const statusColors: Record<string, string> = {
  active: 'bg-success-light text-success',
  prospect: 'bg-warning-light text-warning',
  inactive: 'bg-secondary text-text-disabled',
}

const statusLabels: Record<string, string> = {
  active: 'Aktiv',
  prospect: 'Interessent',
  inactive: 'Inaktiv',
}

export default function KontaktePage() {
  const { contacts, addContact, updateContact, deleteContact, toggleFavorite, duplicateContact } =
    useContactsStore()
  const { startCall, endCall, activeCallContactId, activeCallContactName } = useMeetingsStore()
  const navigate = useNavigate()
  const setIntent = useNavigationStore((s) => s.setIntent)

  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('all')
  const [sortField, setSortField] = useState<SortField>('name')
  const [selectedContactId, setSelectedContactId] = useState<string | null>(null)

  // Dialog state
  const [formOpen, setFormOpen] = useState(false)
  const [editContact, setEditContact] = useState<Contact | null>(null)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  const filtered = contacts
    .filter((c) => {
      if (categoryFilter !== 'all' && c.category !== categoryFilter) return false
      if (search) {
        const q = search.toLowerCase()
        return (
          c.firstName.toLowerCase().includes(q) ||
          c.lastName.toLowerCase().includes(q) ||
          c.company.toLowerCase().includes(q) ||
          c.email.toLowerCase().includes(q) ||
          c.tags.some((t) => t.toLowerCase().includes(q))
        )
      }
      return true
    })
    .sort((a, b) => {
      if (sortField === 'name') return `${a.lastName} ${a.firstName}`.localeCompare(`${b.lastName} ${b.firstName}`)
      if (sortField === 'company') return a.company.localeCompare(b.company)
      return new Date(b.lastContact).getTime() - new Date(a.lastContact).getTime()
    })

  const selectedContact = contacts.find((c) => c.id === selectedContactId) || null
  const deleteTarget = contacts.find((c) => c.id === deleteConfirmId)

  const categories: { key: CategoryFilter; label: string; icon: typeof User; count: number }[] = [
    { key: 'all', label: 'Alle', icon: Users, count: contacts.length },
    { key: 'employee', label: 'Mitarbeiter', icon: User, count: contacts.filter((c) => c.category === 'employee').length },
    { key: 'customer', label: 'Kunden', icon: Briefcase, count: contacts.filter((c) => c.category === 'customer').length },
    { key: 'partner', label: 'Partner', icon: Building2, count: contacts.filter((c) => c.category === 'partner').length },
  ]

  const handleCreateSubmit = (data: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>) => {
    if (editContact) {
      updateContact(editContact.id, data)
      toast.success('Kontakt aktualisiert')
    } else {
      addContact(data)
      toast.success('Kontakt erstellt')
    }
    setEditContact(null)
  }

  const handleDelete = () => {
    if (deleteConfirmId) {
      deleteContact(deleteConfirmId)
      toast.success('Kontakt geloescht')
      if (selectedContactId === deleteConfirmId) setSelectedContactId(null)
      setDeleteConfirmId(null)
    }
  }

  const handleEmail = (contact: Contact) => {
    setIntent({
      type: 'compose-email',
      data: { to: contact.email, name: `${contact.firstName} ${contact.lastName}` },
    })
    navigate('/mails')
    toast.info(`E-Mail an ${contact.firstName} ${contact.lastName}`)
  }

  const handleCall = (contact: Contact) => {
    startCall(contact.id, `${contact.firstName} ${contact.lastName}`)
    toast.info(`Anruf an ${contact.firstName} ${contact.lastName}...`)
  }

  const handleMessage = (contact: Contact) => {
    setIntent({
      type: 'send-message',
      data: { contactId: contact.id, name: `${contact.firstName} ${contact.lastName}` },
    })
    navigate('/chat')
    toast.info(`Chat mit ${contact.firstName} ${contact.lastName}`)
  }

  const handleExportVCard = (contact: Contact) => {
    toast.success(`vCard fuer ${contact.firstName} ${contact.lastName} exportiert`)
  }

  const getContactActions = (c: Contact): ActionItem[] => [
    {
      label: 'Bearbeiten',
      icon: Pencil,
      onClick: () => {
        setEditContact(c)
        setFormOpen(true)
      },
    },
    {
      label: 'Duplizieren',
      icon: Copy,
      onClick: () => {
        duplicateContact(c.id)
        toast.success('Kontakt dupliziert')
      },
    },
    {
      label: c.isFavorite ? 'Aus Favoriten' : 'Favorisieren',
      icon: Star,
      onClick: () => {
        toggleFavorite(c.id)
        toast.success(c.isFavorite ? 'Aus Favoriten entfernt' : 'Zu Favoriten hinzugefuegt')
      },
    },
    {
      label: 'E-Mail senden',
      icon: Mail,
      onClick: () => handleEmail(c),
      separator: true,
    },
    {
      label: 'Anrufen',
      icon: Phone,
      onClick: () => handleCall(c),
    },
    {
      label: 'Nachricht',
      icon: MessageSquare,
      onClick: () => handleMessage(c),
    },
    {
      label: 'vCard exportieren',
      icon: Download,
      onClick: () => handleExportVCard(c),
      separator: true,
    },
    {
      label: 'Loeschen',
      icon: Trash2,
      variant: 'destructive',
      onClick: () => setDeleteConfirmId(c.id),
      separator: true,
    },
  ]

  return (
    <div className="flex h-full overflow-hidden">
      {/* Category sidebar */}
      <aside className="w-52 shrink-0 border-r border-border bg-card p-3 overflow-y-auto">
        <h3 className="text-sm font-medium text-foreground mb-3 px-2">Kategorien</h3>
        <nav className="space-y-0.5">
          {categories.map((cat) => {
            const Icon = cat.icon
            return (
              <button
                key={cat.key}
                onClick={() => setCategoryFilter(cat.key)}
                className={`flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors ${
                  categoryFilter === cat.key
                    ? 'bg-primary-light text-primary font-medium'
                    : 'text-foreground hover:bg-secondary'
                }`}
              >
                <Icon className="h-4 w-4 shrink-0" />
                <span className="flex-1 text-left">{cat.label}</span>
                <span className="text-xs text-muted-foreground">{cat.count}</span>
              </button>
            )
          })}
        </nav>

        {/* Favorites section */}
        {contacts.some((c) => c.isFavorite) && (
          <div className="mt-6">
            <h3 className="text-sm font-medium text-foreground mb-3 px-2 flex items-center gap-1.5">
              <Star className="h-3.5 w-3.5 text-yellow-500" />
              Favoriten
            </h3>
            <nav className="space-y-0.5">
              {contacts
                .filter((c) => c.isFavorite)
                .map((c) => (
                  <button
                    key={c.id}
                    onClick={() => setSelectedContactId(c.id)}
                    className={`flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-sm transition-colors ${
                      selectedContactId === c.id
                        ? 'bg-primary-light text-primary font-medium'
                        : 'text-foreground hover:bg-secondary'
                    }`}
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

      {/* Contact list */}
      <div
        className={`flex flex-col border-r border-border overflow-hidden ${
          selectedContact ? 'hidden md:flex w-80' : 'flex-1'
        }`}
      >
        {/* Search + New */}
        <div className="flex items-center gap-2 border-b border-border px-4 py-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Kontakt suchen..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-background pl-9 pr-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
          <button
            onClick={() => {
              const next: SortField[] = ['name', 'company', 'lastContact']
              setSortField(next[(next.indexOf(sortField) + 1) % next.length])
            }}
            className="rounded-lg border border-border p-1.5 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
            title={`Sortierung: ${sortField === 'name' ? 'Name' : sortField === 'company' ? 'Firma' : 'Letzter Kontakt'}`}
          >
            <ArrowUpDown className="h-4 w-4" />
          </button>
          <button
            onClick={() => {
              setEditContact(null)
              setFormOpen(true)
            }}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            <span className="hidden lg:inline">Neu</span>
          </button>
        </div>

        {/* Sort indicator */}
        <div className="flex items-center gap-2 px-4 py-1.5 border-b border-border-muted text-xs text-muted-foreground">
          <span>{filtered.length} Kontakte</span>
          <span>·</span>
          <span>
            Sortiert nach {sortField === 'name' ? 'Name' : sortField === 'company' ? 'Firma' : 'Letzter Kontakt'}
          </span>
        </div>

        {/* List */}
        <div className="flex-1 overflow-y-auto">
          {filtered.map((contact) => (
            <div
              key={contact.id}
              className={`group flex w-full items-center gap-3 border-b border-border-muted px-4 py-3 text-left transition-colors hover:bg-secondary/50 cursor-pointer ${
                selectedContactId === contact.id ? 'bg-primary-light/50 border-l-2 border-l-primary' : ''
              }`}
              onClick={() => setSelectedContactId(contact.id)}
            >
              <div className="relative flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary-light text-sm font-medium text-primary">
                {contact.initials}
                {contact.isFavorite && (
                  <Star className="absolute -top-1 -right-1 h-3 w-3 fill-yellow-400 text-yellow-400" />
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-foreground truncate">
                    {contact.firstName} {contact.lastName}
                  </span>
                  <span className={`inline-flex rounded-full px-1.5 py-0.5 text-[10px] ${statusColors[contact.status]}`}>
                    {statusLabels[contact.status]}
                  </span>
                </div>
                <p className="text-xs text-muted-foreground truncate">
                  {contact.jobTitle}{contact.jobTitle && contact.company ? ' · ' : ''}{contact.company}
                </p>
              </div>
              <div className="opacity-0 group-hover:opacity-100 transition-opacity" onClick={(e) => e.stopPropagation()}>
                <ItemActions items={getContactActions(contact)} />
              </div>
            </div>
          ))}

          {filtered.length === 0 && (
            <EmptyState
              icon={Users}
              title="Keine Kontakte gefunden"
              description={
                search
                  ? 'Versuche einen anderen Suchbegriff'
                  : 'Erstelle deinen ersten Kontakt'
              }
              action={
                !search
                  ? {
                      label: 'Neuer Kontakt',
                      onClick: () => {
                        setEditContact(null)
                        setFormOpen(true)
                      },
                    }
                  : undefined
              }
            />
          )}
        </div>
      </div>

      {/* Detail panel */}
      {selectedContact ? (
        <ContactDetailPanel
          contact={selectedContact}
          onClose={() => setSelectedContactId(null)}
          onEdit={(c) => {
            setEditContact(c)
            setFormOpen(true)
          }}
          onDelete={(id) => setDeleteConfirmId(id)}
          onEmail={handleEmail}
          onCall={handleCall}
          onMessage={handleMessage}
          onToggleFavorite={toggleFavorite}
        />
      ) : (
        <div className="hidden md:flex flex-1 items-center justify-center text-muted-foreground">
          <div className="text-center">
            <User className="h-12 w-12 mx-auto mb-3 opacity-30" />
            <p className="text-sm">Waehle einen Kontakt aus</p>
            <p className="text-xs mt-1 text-muted-foreground/60">
              oder erstelle einen neuen mit dem + Button
            </p>
          </div>
        </div>
      )}

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
        title="Kontakt loeschen?"
        description={`"${deleteTarget ? `${deleteTarget.firstName} ${deleteTarget.lastName}` : ''}" wird unwiderruflich geloescht.`}
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={handleDelete}
      />

      {/* Call Overlay */}
      <CallOverlay
        contactName={activeCallContactName || ''}
        open={!!activeCallContactId}
        onEnd={() => {
          endCall()
          toast.info('Anruf beendet')
        }}
      />
    </div>
  )
}
