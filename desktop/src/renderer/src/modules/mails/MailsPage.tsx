import { useState, useEffect } from 'react'
import {
  Mail,
  Inbox,
  Send,
  FileEdit,
  Archive,
  Trash2,
  Star,
  Search,
  Plus,
  Paperclip,
  ChevronLeft,
  Reply,
  ReplyAll,
  Forward,
  AlertCircle,
  FolderPlus,
  Eye,
  EyeOff,
  ArrowRight,
  Pencil,
} from 'lucide-react'
import { toast } from 'sonner'
import { useMailsStore, type Email } from '@/stores/mails'
import { useNavigationStore } from '@/stores/navigation'
import { ItemActions, ConfirmDialog, EmptyState, type ActionItem } from '@/components/shared'
import { ComposeModal, type ComposeMode } from './ComposeModal'

const folderIcons: Record<string, typeof Inbox> = {
  inbox: Inbox,
  drafts: FileEdit,
  sent: Send,
  archive: Archive,
  spam: AlertCircle,
  trash: Trash2,
}

export default function MailsPage() {
  const {
    emails, folders,
    deleteEmail, markRead, markUnread, toggleStar, archiveEmail, moveToFolder,
    addFolder, emptyTrash,
  } = useMailsStore()
  const consumeIntent = useNavigationStore((s) => s.consumeIntent)

  const [activeFolder, setActiveFolder] = useState('inbox')
  const [selectedEmailId, setSelectedEmailId] = useState<string | null>(null)
  const [search, setSearch] = useState('')

  // Compose state
  const [composeOpen, setComposeOpen] = useState(false)
  const [composeMode, setComposeMode] = useState<ComposeMode>('compose')
  const [composeReplyTo, setComposeReplyTo] = useState<Email | null>(null)
  const [composePrefillTo, setComposePrefillTo] = useState<string | undefined>(undefined)

  // Confirm dialogs
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)
  const [emptyTrashConfirm, setEmptyTrashConfirm] = useState(false)

  // Consume navigation intent on mount
  useEffect(() => {
    const intent = consumeIntent()
    if (intent?.type === 'compose-email') {
      setComposePrefillTo(intent.data.to)
      setComposeMode('compose')
      setComposeReplyTo(null)
      setComposeOpen(true)
    }
  }, [])

  const folderEmails = emails.filter((e) => e.folderId === activeFolder)
  const filtered = folderEmails.filter((e) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      e.subject.toLowerCase().includes(q) ||
      e.from.name.toLowerCase().includes(q) ||
      e.body.toLowerCase().includes(q)
    )
  })

  const selectedEmail = emails.find((e) => e.id === selectedEmailId) || null
  const deleteTarget = emails.find((e) => e.id === deleteConfirmId)

  const handleSelectEmail = (email: Email) => {
    setSelectedEmailId(email.id)
    if (!email.isRead) markRead(email.id)
  }

  const openCompose = (mode: ComposeMode = 'compose', email?: Email) => {
    setComposeMode(mode)
    setComposeReplyTo(email || null)
    setComposePrefillTo(undefined)
    setComposeOpen(true)
  }

  const handleDelete = () => {
    if (deleteConfirmId) {
      deleteEmail(deleteConfirmId)
      if (selectedEmailId === deleteConfirmId) setSelectedEmailId(null)
      setDeleteConfirmId(null)
      toast.success('E-Mail geloescht')
    }
  }

  const handleEmptyTrash = () => {
    emptyTrash()
    setEmptyTrashConfirm(false)
    toast.success('Papierkorb geleert')
  }

  const getEmailActions = (e: Email): ActionItem[] => {
    const actions: ActionItem[] = []

    if (e.folderId === 'inbox') {
      actions.push(
        { label: 'Antworten', icon: Reply, onClick: () => openCompose('reply', e) },
        { label: 'Allen antworten', icon: ReplyAll, onClick: () => openCompose('reply-all', e) },
        { label: 'Weiterleiten', icon: Forward, onClick: () => openCompose('forward', e) },
      )
    }

    actions.push({
      label: e.isStarred ? 'Stern entfernen' : 'Stern vergeben',
      icon: Star,
      onClick: () => {
        toggleStar(e.id)
        toast.success(e.isStarred ? 'Stern entfernt' : 'Stern vergeben')
      },
      separator: true,
    })

    actions.push({
      label: e.isRead ? 'Als ungelesen' : 'Als gelesen',
      icon: e.isRead ? EyeOff : Eye,
      onClick: () => {
        e.isRead ? markUnread(e.id) : markRead(e.id)
        toast.success(e.isRead ? 'Als ungelesen markiert' : 'Als gelesen markiert')
      },
    })

    if (e.folderId !== 'archive') {
      actions.push({
        label: 'Archivieren',
        icon: Archive,
        onClick: () => {
          archiveEmail(e.id)
          if (selectedEmailId === e.id) setSelectedEmailId(null)
          toast.success('Archiviert')
        },
        separator: true,
      })
    }

    actions.push({
      label: 'Loeschen',
      icon: Trash2,
      variant: 'destructive',
      onClick: () => setDeleteConfirmId(e.id),
      separator: e.folderId === 'archive',
    })

    return actions
  }

  const folderUnreadCounts = folders.map((f) => ({
    ...f,
    unread: emails.filter((e) => e.folderId === f.id && !e.isRead).length,
  }))

  return (
    <div className="flex h-full overflow-hidden">
      {/* Folder sidebar */}
      <aside className="w-52 shrink-0 border-r border-border bg-card p-3 overflow-y-auto">
        <button
          onClick={() => openCompose()}
          className="flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors mb-4"
        >
          <Plus className="h-4 w-4" />
          Neue E-Mail
        </button>
        <nav className="space-y-0.5">
          {folderUnreadCounts.map((folder) => {
            const Icon = folderIcons[folder.id] || Mail
            return (
              <button
                key={folder.id}
                onClick={() => {
                  setActiveFolder(folder.id)
                  setSelectedEmailId(null)
                }}
                className={`flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors ${
                  activeFolder === folder.id
                    ? 'bg-primary-light text-primary font-medium'
                    : 'text-foreground hover:bg-secondary'
                }`}
              >
                <Icon className="h-4 w-4 shrink-0" />
                <span className="flex-1 text-left truncate">{folder.name}</span>
                {folder.unread > 0 && (
                  <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-primary text-[10px] text-primary-foreground px-1">
                    {folder.unread}
                  </span>
                )}
              </button>
            )
          })}
        </nav>

        {/* Empty trash */}
        {activeFolder === 'trash' && emails.some((e) => e.folderId === 'trash') && (
          <button
            onClick={() => setEmptyTrashConfirm(true)}
            className="mt-4 flex w-full items-center justify-center gap-1.5 rounded-lg border border-red-300 px-3 py-1.5 text-xs text-red-500 hover:bg-red-50 transition-colors"
          >
            <Trash2 className="h-3.5 w-3.5" />
            Papierkorb leeren
          </button>
        )}
      </aside>

      {/* Email list / Detail */}
      <div className="flex flex-1 overflow-hidden">
        {/* Email list */}
        <div
          className={`flex flex-col border-r border-border overflow-hidden ${
            selectedEmail ? 'hidden md:flex w-80' : 'flex-1'
          }`}
        >
          {/* Search bar */}
          <div className="border-b border-border px-4 py-3">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="E-Mails suchen..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-background pl-9 pr-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Email rows */}
          <div className="flex-1 overflow-y-auto">
            {filtered.map((email) => (
              <div
                key={email.id}
                className={`group flex w-full items-start gap-3 border-b border-border-muted px-4 py-3 text-left transition-colors hover:bg-secondary/50 cursor-pointer ${
                  selectedEmailId === email.id ? 'bg-primary-light/50' : ''
                } ${!email.isRead ? 'bg-card' : ''}`}
                onClick={() => handleSelectEmail(email)}
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-light text-xs font-medium text-primary">
                  {email.from.initials}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className={`text-sm truncate ${!email.isRead ? 'font-medium text-foreground' : 'text-text-body'}`}>
                      {email.from.name}
                    </span>
                    <div className="flex items-center gap-1.5 shrink-0">
                      <span className="text-[10px] text-muted-foreground whitespace-nowrap">{email.time}</span>
                    </div>
                  </div>
                  <p className={`text-sm truncate ${!email.isRead ? 'font-medium text-foreground' : 'text-text-body'}`}>
                    {email.subject}
                  </p>
                  <p className="text-xs text-muted-foreground truncate mt-0.5">{email.preview}</p>
                  <div className="flex items-center gap-2 mt-1">
                    {email.attachments.length > 0 && <Paperclip className="h-3 w-3 text-muted-foreground" />}
                    {email.isStarred && (
                      <button
                        onClick={(e) => { e.stopPropagation(); toggleStar(email.id) }}
                        className="p-0"
                      >
                        <Star className="h-3 w-3 text-warning fill-warning" />
                      </button>
                    )}
                  </div>
                </div>
                <div className="opacity-0 group-hover:opacity-100 transition-opacity shrink-0 self-center" onClick={(e) => e.stopPropagation()}>
                  <ItemActions items={getEmailActions(email)} />
                </div>
              </div>
            ))}
            {filtered.length === 0 && (
              <EmptyState
                icon={Mail}
                title="Keine E-Mails"
                description={
                  search
                    ? 'Keine E-Mails fuer diesen Suchbegriff'
                    : activeFolder === 'inbox'
                      ? 'Dein Posteingang ist leer'
                      : `Keine E-Mails in "${folderUnreadCounts.find((f) => f.id === activeFolder)?.name || activeFolder}"`
                }
              />
            )}
          </div>
        </div>

        {/* Detail view */}
        {selectedEmail ? (
          <div className="flex-1 flex flex-col overflow-hidden">
            {/* Detail header */}
            <div className="flex items-center gap-3 border-b border-border px-6 py-3">
              <button
                onClick={() => setSelectedEmailId(null)}
                className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary md:hidden"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
              <div className="flex-1" />
              <div className="flex items-center gap-1">
                <button
                  onClick={() => openCompose('reply', selectedEmail)}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground"
                  title="Antworten"
                >
                  <Reply className="h-4 w-4" />
                </button>
                <button
                  onClick={() => openCompose('reply-all', selectedEmail)}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground"
                  title="Allen antworten"
                >
                  <ReplyAll className="h-4 w-4" />
                </button>
                <button
                  onClick={() => openCompose('forward', selectedEmail)}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground"
                  title="Weiterleiten"
                >
                  <Forward className="h-4 w-4" />
                </button>
                <span className="mx-1 h-5 w-px bg-border" />
                <button
                  onClick={() => {
                    toggleStar(selectedEmail.id)
                    toast.success(selectedEmail.isStarred ? 'Stern entfernt' : 'Stern vergeben')
                  }}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground"
                  title={selectedEmail.isStarred ? 'Stern entfernen' : 'Stern vergeben'}
                >
                  <Star className={`h-4 w-4 ${selectedEmail.isStarred ? 'fill-warning text-warning' : ''}`} />
                </button>
                <button
                  onClick={() => {
                    archiveEmail(selectedEmail.id)
                    setSelectedEmailId(null)
                    toast.success('Archiviert')
                  }}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground"
                  title="Archivieren"
                >
                  <Archive className="h-4 w-4" />
                </button>
                <button
                  onClick={() => setDeleteConfirmId(selectedEmail.id)}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-red-500"
                  title="Loeschen"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>

            {/* Detail body */}
            <div className="flex-1 overflow-y-auto px-6 py-4">
              <h2 className="text-foreground mb-4">{selectedEmail.subject}</h2>
              <div className="flex items-start gap-3 mb-6">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary-light text-sm font-medium text-primary">
                  {selectedEmail.from.initials}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-foreground">{selectedEmail.from.name}</span>
                    <span className="text-xs text-muted-foreground truncate">&lt;{selectedEmail.from.email}&gt;</span>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    An: {selectedEmail.to.join(', ')}
                    {selectedEmail.cc.length > 0 && ` · Cc: ${selectedEmail.cc.join(', ')}`}
                    {' · '}
                    {new Date(selectedEmail.date).toLocaleDateString('de-CH')} um {selectedEmail.time}
                  </p>
                </div>
              </div>
              <div className="text-sm text-text-body whitespace-pre-line leading-relaxed">
                {selectedEmail.body}
              </div>
              {selectedEmail.attachments.length > 0 && (
                <div className="mt-6 border-t border-border pt-4">
                  <p className="text-xs font-medium text-muted-foreground mb-2">
                    Anhaenge ({selectedEmail.attachments.length})
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {selectedEmail.attachments.map((att, i) => (
                      <button
                        key={i}
                        onClick={() => toast.success(`"${att.name}" wird heruntergeladen`)}
                        className="flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2 hover:bg-secondary transition-colors"
                      >
                        <Paperclip className="h-4 w-4 text-muted-foreground" />
                        <div className="text-left">
                          <p className="text-sm text-foreground">{att.name}</p>
                          <p className="text-[10px] text-muted-foreground">{att.size}</p>
                        </div>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Quick reply */}
              <div className="mt-6 border-t border-border pt-4">
                <button
                  onClick={() => openCompose('reply', selectedEmail)}
                  className="flex w-full items-center gap-2 rounded-lg border border-border px-4 py-3 text-sm text-muted-foreground hover:bg-secondary transition-colors"
                >
                  <Reply className="h-4 w-4" />
                  Antworten...
                </button>
              </div>
            </div>
          </div>
        ) : (
          <div className="hidden md:flex flex-1 items-center justify-center text-muted-foreground">
            <div className="text-center">
              <Mail className="h-12 w-12 mx-auto mb-3 opacity-30" />
              <p className="text-sm">Waehle eine E-Mail aus</p>
            </div>
          </div>
        )}
      </div>

      {/* Compose Modal */}
      <ComposeModal
        open={composeOpen}
        onOpenChange={setComposeOpen}
        mode={composeMode}
        replyTo={composeReplyTo}
        prefillTo={composePrefillTo}
      />

      {/* Delete Confirm */}
      <ConfirmDialog
        open={!!deleteConfirmId}
        onOpenChange={(open) => !open && setDeleteConfirmId(null)}
        title="E-Mail loeschen?"
        description={
          deleteTarget?.folderId === 'trash'
            ? `"${deleteTarget?.subject}" wird endgueltig geloescht.`
            : `"${deleteTarget?.subject}" wird in den Papierkorb verschoben.`
        }
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={handleDelete}
      />

      {/* Empty Trash Confirm */}
      <ConfirmDialog
        open={emptyTrashConfirm}
        onOpenChange={setEmptyTrashConfirm}
        title="Papierkorb leeren?"
        description="Alle E-Mails im Papierkorb werden endgueltig geloescht. Diese Aktion kann nicht rueckgaengig gemacht werden."
        confirmLabel="Papierkorb leeren"
        variant="destructive"
        onConfirm={handleEmptyTrash}
      />
    </div>
  )
}
