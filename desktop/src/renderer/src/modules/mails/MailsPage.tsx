import { useState, useEffect, useMemo } from 'react'
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
  Eye,
  EyeOff,
  Printer,
  Download,
  RefreshCw,
  Users,
  Loader2,
  Settings,
  ShieldCheck,
} from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth'
import { useMailsStore } from '@/stores/mails'
import { useNavigationStore } from '@/stores/navigation'
import {
  useEmailAccount,
  useEmailFolders,
  useEmailMessages,
  useMarkEmailRead,
  useMarkEmailUnread,
  useToggleEmailStar,
  useMoveEmailToFolder,
  useDeleteEmailMessage,
  useTriggerSync,
  useSyncStatus,
  useEmailContactLinks,
} from '@/api/hooks/useEmail'
import type { EmailMessageInfo } from '@/api/email-types'
import type { ComposeMode } from '@/stores/mails'
import { ComposeInline } from './ComposeInline'
import { ItemActions, ConfirmDialog, EmptyState, type ActionItem } from '@/components/shared'

const folderIcons: Record<string, typeof Inbox> = {
  inbox: Inbox,
  drafts: FileEdit,
  sent: Send,
  archive: Archive,
  spam: AlertCircle,
  trash: Trash2,
}

// ---------------------------------------------------------------------------
// DSGVO Aufbewahrungsfristen
// ---------------------------------------------------------------------------

const DSGVO_BUSINESS_DOMAINS = [
  'gmbh', 'ag', 'sarl', 'srl', 'ltd', 'inc', 'corp',
]

interface RetentionInfo {
  label: string
  years: number
  basis: string
}

/** Determine if an email likely has DSGVO retention requirements */
function getRetentionInfo(msg: EmailMessageInfo): RetentionInfo | null {
  const allEmails = [msg.from.email, ...msg.to.map((a) => a.email)]
  const hasBusiness = allEmails.some((e) => {
    const domain = e.split('@')[1]?.toLowerCase() ?? ''
    return DSGVO_BUSINESS_DOMAINS.some((d) => domain.includes(d)) || !domain.includes('gmail') && !domain.includes('outlook') && !domain.includes('yahoo') && !domain.includes('hotmail') && !domain.includes('gmx') && !domain.includes('web.de')
  })
  if (!hasBusiness) return null

  // Check subject for invoice/contract keywords (stronger retention)
  const subjectLower = msg.subject.toLowerCase()
  const isFinancial = ['rechnung', 'invoice', 'zahlung', 'payment', 'vertrag', 'contract', 'angebot', 'quote', 'bestellung', 'order', 'mahnung'].some((kw) => subjectLower.includes(kw))

  if (isFinancial) {
    return {
      label: 'CH: 10 J. / DE: 10 J.',
      years: 10,
      basis: 'Geschaeftliche Buchungsbelege — OR Art. 958f (CH) / AO §147 (DE)',
    }
  }

  return {
    label: 'CH: 10 J. / DE: 6 J.',
    years: 6,
    basis: 'Geschaeftskorrespondenz — OR Art. 958f (CH) / HGB §257 (DE)',
  }
}

export default function MailsPage() {
  const user = useAuthStore((s) => s.user)
  const userId = user?.id ?? ''
  const consumeIntent = useNavigationStore((s) => s.consumeIntent)
  const { setComposeDraft: _setComposeDraft } = useMailsStore()

  // Account
  const { data: accountData } = useEmailAccount(userId)
  const accountId = accountData?.account?.id ?? ''

  // Folders
  const { data: foldersData } = useEmailFolders(accountId)
  const folders = useMemo(() => foldersData?.folders ?? [], [foldersData?.folders])

  // UI state
  const [activeFolderId, setActiveFolderId] = useState<string>('')
  const [selectedMessageId, setSelectedMessageId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [page, setPage] = useState(1)

  // Compose state
  const [composeOpen, setComposeOpen] = useState(false)
  const [composeMode, setComposeMode] = useState<ComposeMode>('compose')
  const [composeReplyTo, setComposeReplyTo] = useState<EmailMessageInfo | null>(null)
  const [composePrefillTo, setComposePrefillTo] = useState<string | undefined>(undefined)

  // Confirm dialogs
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  // Auto-select inbox folder when folders load
  useEffect(() => {
    if (folders.length > 0 && !activeFolderId) {
      const inbox = folders.find((f) => f.folder_type === 'inbox')
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync local editable state from prop
      setActiveFolderId(inbox?.id ?? folders[0].id)
    }
  }, [folders, activeFolderId])

  // Navigation intent (e.g., compose-email from CRM)
   
  useEffect(() => {
    const intent = consumeIntent()
    if (intent?.type === 'compose-email') {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- navigation intent handler
      setComposePrefillTo(intent.data.to)
      setComposeMode('compose')
      setComposeReplyTo(null)
      setComposeOpen(true)
    }
  }, [consumeIntent])

  // Messages for active folder
  const messagesParams = useMemo(() => ({
    folder_id: activeFolderId,
    page,
    per_page: 50,
    search: searchQuery || undefined,
    sort_by: 'date',
    sort_desc: true,
  }), [activeFolderId, page, searchQuery])

  const { data: messagesData, isLoading: messagesLoading } = useEmailMessages(messagesParams)
  const messages = messagesData?.messages ?? []

  // Sync
  const { data: syncStatusData } = useSyncStatus(accountId)
  const triggerSync = useTriggerSync()

  // Mutations
  const markRead = useMarkEmailRead()
  const markUnread = useMarkEmailUnread()
  const toggleStar = useToggleEmailStar()
  const moveToFolder = useMoveEmailToFolder()
  const deleteMessage = useDeleteEmailMessage()

  // CRM links for selected message
  const { data: linksData } = useEmailContactLinks(selectedMessageId ?? '')

  const selectedMessage = messages.find((m) => m.id === selectedMessageId) ?? null
  const deleteTarget = messages.find((m) => m.id === deleteConfirmId)

  const handleSelectMessage = (msg: EmailMessageInfo) => {
    setSelectedMessageId(msg.id)
    if (composeOpen) setComposeOpen(false)
    if (!msg.is_read) markRead.mutate(msg.id)
  }

  const openCompose = (mode: ComposeMode = 'compose', msg?: EmailMessageInfo) => {
    setComposeMode(mode)
    setComposeReplyTo(msg ?? null)
    setComposePrefillTo(undefined)
    setComposeOpen(true)
    setSelectedMessageId(null)
  }

  const handleDelete = () => {
    if (deleteConfirmId) {
      deleteMessage.mutate(deleteConfirmId)
      if (selectedMessageId === deleteConfirmId) setSelectedMessageId(null)
      setDeleteConfirmId(null)
      toast.success('E-Mail gelöscht')
    }
  }

  const handleArchive = (id: string) => {
    const archiveFolder = folders.find((f) => f.folder_type === 'archive')
    if (archiveFolder) {
      moveToFolder.mutate({ messageId: id, targetFolderId: archiveFolder.id })
      if (selectedMessageId === id) setSelectedMessageId(null)
      toast.success('Archiviert')
    }
  }

  const getMessageActions = (msg: EmailMessageInfo): ActionItem[] => {
    const actions: ActionItem[] = []
    const currentFolder = folders.find((f) => f.id === msg.folder_id)

    if (currentFolder?.folder_type === 'inbox') {
      actions.push(
        { label: 'Antworten', icon: Reply, onClick: () => openCompose('reply', msg) },
        { label: 'Allen antworten', icon: ReplyAll, onClick: () => openCompose('reply-all', msg) },
        { label: 'Weiterleiten', icon: Forward, onClick: () => openCompose('forward', msg) },
      )
    }

    actions.push({
      label: msg.is_starred ? 'Stern entfernen' : 'Stern vergeben',
      icon: Star,
      onClick: () => {
        toggleStar.mutate(msg.id)
        toast.success(msg.is_starred ? 'Stern entfernt' : 'Stern vergeben')
      },
      separator: true,
    })

    actions.push({
      label: msg.is_read ? 'Als ungelesen' : 'Als gelesen',
      icon: msg.is_read ? EyeOff : Eye,
      onClick: () => {
        if (msg.is_read) markUnread.mutate(msg.id)
        else markRead.mutate(msg.id)
        toast.success(msg.is_read ? 'Als ungelesen markiert' : 'Als gelesen markiert')
      },
    })

    if (currentFolder?.folder_type !== 'archive') {
      actions.push({
        label: 'Archivieren',
        icon: Archive,
        onClick: () => handleArchive(msg.id),
        separator: true,
      })
    }

    actions.push(
      { label: 'Drucken', icon: Printer, onClick: () => toast.success('Druckvorschau wird geöffnet...'), separator: true },
      { label: 'Exportieren', icon: Download, onClick: () => toast.success('PDF Export wird vorbereitet...') },
    )

    actions.push({
      label: 'Löschen',
      icon: Trash2,
      variant: 'destructive',
      onClick: () => setDeleteConfirmId(msg.id),
      separator: true,
    })

    return actions
  }

  // Initials helper
  const getInitials = (addr: { name: string; email: string }) => {
    if (addr.name) {
      return addr.name.split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2)
    }
    return addr.email[0].toUpperCase()
  }

  const formatDate = (dateStr: string) => {
    try {
      const d = new Date(dateStr)
      return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })
    } catch {
      return dateStr
    }
  }

  const formatTime = (dateStr: string) => {
    try {
      const d = new Date(dateStr)
      return d.toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })
    } catch {
      return ''
    }
  }

  const showInlineCompose = composeOpen
  const showMessageDetail = selectedMessage && !showInlineCompose
  const crmLinks = linksData?.links ?? []

  // No account configured
  if (!accountId && !accountData) {
    return (
      <div className="flex h-full items-center justify-center">
        <EmptyState
          icon={Settings}
          title="Kein E-Mail-Konto eingerichtet"
          description="Richte dein E-Mail-Konto in den Einstellungen ein, um E-Mails zu senden und empfangen."
        />
      </div>
    )
  }

  return (
    <div className="flex h-full overflow-hidden animate-fade-in">
      {/* Folder sidebar */}
      <aside className="w-52 shrink-0 border-r border-border bg-card p-3 overflow-y-auto">
        <button
          onClick={() => openCompose()}
          className="flex w-full items-center justify-center gap-2 rounded-xl bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors mb-4"
        >
          <Plus className="h-4 w-4" />
          Neue E-Mail
        </button>

        <nav className="space-y-0.5">
          {folders.map((folder) => {
            const Icon = folderIcons[folder.folder_type] || Mail
            return (
              <button
                key={folder.id}
                onClick={() => {
                  setActiveFolderId(folder.id)
                  setSelectedMessageId(null)
                  setPage(1)
                }}
                className={`flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition-colors ${
                  activeFolderId === folder.id
                    ? 'bg-primary-light text-primary font-medium'
                    : 'text-foreground hover:bg-secondary'
                }`}
              >
                <Icon className="h-4 w-4 shrink-0" />
                <span className="flex-1 text-left truncate">{folder.name}</span>
                {folder.unread_count > 0 && (
                  <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-primary text-[10px] text-primary-foreground px-1 badge-accent">
                    {folder.unread_count}
                  </span>
                )}
              </button>
            )
          })}
        </nav>

        {/* Sync button */}
        {accountId && (
          <button
            onClick={() => triggerSync.mutate(accountId)}
            disabled={triggerSync.isPending || syncStatusData?.status === 'syncing'}
            className="mt-4 flex w-full items-center justify-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${syncStatusData?.status === 'syncing' ? 'animate-spin' : ''}`} />
            {syncStatusData?.status === 'syncing' ? 'Synchronisiere...' : 'Synchronisieren'}
          </button>
        )}
      </aside>

      {/* Message list / Detail */}
      <div className="flex flex-1 overflow-hidden">
        {/* Message list */}
        <div
          className={`flex flex-col border-r border-border overflow-hidden ${
            (showMessageDetail || showInlineCompose) ? 'hidden md:flex w-80' : 'flex-1'
          }`}
        >
          {/* Search bar */}
          <div className="border-b border-border px-4 py-3">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="E-Mails suchen..."
                value={searchQuery}
                onChange={(e) => { setSearchQuery(e.target.value); setPage(1) }}
                className="w-full rounded-lg border border-border bg-background pl-9 pr-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Message rows */}
          <div className="flex-1 overflow-y-auto">
            {messagesLoading && (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            )}
            {!messagesLoading && messages.map((msg) => (
              <div
                key={msg.id}
                className={`group flex w-full items-start gap-3 border-b border-border-muted px-4 py-3 text-left transition-colors hover:bg-secondary/50 cursor-pointer ${
                  selectedMessageId === msg.id ? 'bg-primary-light/50' : ''
                } ${!msg.is_read ? 'bg-card' : ''}`}
                onClick={() => handleSelectMessage(msg)}
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-light text-xs font-medium text-primary">
                  {getInitials(msg.from)}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className={`text-sm truncate ${!msg.is_read ? 'font-medium text-foreground' : 'text-text-body'}`}>
                      {msg.from.name || msg.from.email}
                    </span>
                    <div className="flex items-center gap-1.5 shrink-0">
                      <span className="text-[10px] text-muted-foreground whitespace-nowrap">{formatTime(msg.date)}</span>
                    </div>
                  </div>
                  <p className={`text-sm truncate ${!msg.is_read ? 'font-medium text-foreground' : 'text-text-body'}`}>
                    {msg.subject}
                  </p>
                  <p className="text-xs text-muted-foreground truncate mt-0.5">{msg.preview}</p>
                  <div className="flex items-center gap-2 mt-1">
                    {msg.has_attachments && <Paperclip className="h-3 w-3 text-muted-foreground" />}
                    {(() => {
                      const retention = getRetentionInfo(msg)
                      return retention ? (
                        <span
                          title={`DSGVO Aufbewahrungspflicht: ${retention.basis}`}
                          className="inline-flex items-center gap-0.5 rounded-full bg-blue-50 dark:bg-blue-950/40 px-1.5 py-0 text-[9px] font-medium text-blue-600 dark:text-blue-400"
                        >
                          <ShieldCheck className="h-2.5 w-2.5" />
                          {retention.label}
                        </span>
                      ) : null
                    })()}
                    {msg.is_starred && (
                      <button
                        onClick={(e) => { e.stopPropagation(); toggleStar.mutate(msg.id) }}
                        className="p-0"
                      >
                        <Star className="h-3 w-3 text-warning fill-warning" />
                      </button>
                    )}
                  </div>
                </div>
                <div className="opacity-0 group-hover:opacity-100 transition-opacity shrink-0 self-center" onClick={(e) => e.stopPropagation()}>
                  <ItemActions items={getMessageActions(msg)} />
                </div>
              </div>
            ))}
            {!messagesLoading && messages.length === 0 && (
              <EmptyState
                icon={Mail}
                title="Keine E-Mails"
                description={
                  searchQuery
                    ? 'Keine E-Mails für diesen Suchbegriff'
                    : 'Dieser Ordner ist leer'
                }
              />
            )}
          </div>
        </div>

        {/* Detail area */}
        {showInlineCompose ? (
          <ComposeInline
            mode={composeMode}
            replyTo={composeReplyTo}
            prefillTo={composePrefillTo}
            accountId={accountId}
            onClose={() => setComposeOpen(false)}
          />
        ) : showMessageDetail ? (
          <div className="flex-1 flex flex-col overflow-hidden">
            {/* Detail header */}
            <div className="flex items-center gap-3 border-b border-border px-6 py-3">
              <button
                onClick={() => setSelectedMessageId(null)}
                className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary md:hidden"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
              <div className="flex-1" />
              <div className="flex items-center gap-1">
                <button onClick={() => openCompose('reply', selectedMessage)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground" title="Antworten" aria-label="Antworten">
                  <Reply className="h-4 w-4" />
                </button>
                <button onClick={() => openCompose('reply-all', selectedMessage)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground" title="Allen antworten" aria-label="Allen antworten">
                  <ReplyAll className="h-4 w-4" />
                </button>
                <button onClick={() => openCompose('forward', selectedMessage)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground" title="Weiterleiten" aria-label="Weiterleiten">
                  <Forward className="h-4 w-4" />
                </button>
                <span className="mx-1 h-5 w-px bg-border" />
                <button
                  onClick={() => {
                    toggleStar.mutate(selectedMessage.id)
                    toast.success(selectedMessage.is_starred ? 'Stern entfernt' : 'Stern vergeben')
                  }}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground"
                  title={selectedMessage.is_starred ? 'Stern entfernen' : 'Stern vergeben'}
                >
                  <Star className={`h-4 w-4 ${selectedMessage.is_starred ? 'fill-warning text-warning' : ''}`} />
                </button>
                <button
                  onClick={() => handleArchive(selectedMessage.id)}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground"
                  title="Archivieren"
                >
                  <Archive className="h-4 w-4" />
                </button>
                <button
                  onClick={() => setDeleteConfirmId(selectedMessage.id)}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-red-500"
                  title="Löschen"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>

            {/* Detail body */}
            <div className="flex-1 overflow-y-auto px-6 py-4">
              <h2 className="text-foreground mb-4">{selectedMessage.subject}</h2>
              <div className="flex items-start gap-3 mb-6">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary-light text-sm font-medium text-primary">
                  {getInitials(selectedMessage.from)}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-foreground">{selectedMessage.from.name || selectedMessage.from.email}</span>
                    <span className="text-xs text-muted-foreground truncate">&lt;{selectedMessage.from.email}&gt;</span>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    An: {selectedMessage.to.map((a) => a.name || a.email).join(', ')}
                    {selectedMessage.cc.length > 0 && ` · Cc: ${selectedMessage.cc.map((a) => a.name || a.email).join(', ')}`}
                    {' · '}
                    {formatDate(selectedMessage.date)} um {formatTime(selectedMessage.date)}
                  </p>
                </div>
              </div>

              {/* CRM link badge */}
              {crmLinks.length > 0 && (
                <div className="flex items-center gap-2 mb-4 px-3 py-2 rounded-lg bg-primary/5 border border-primary/20">
                  <Users className="h-4 w-4 text-primary" />
                  <span className="text-xs text-primary font-medium">
                    Mit {crmLinks.length} CRM-Kontakt{crmLinks.length > 1 ? 'en' : ''} verknüpft
                  </span>
                </div>
              )}

              {/* DSGVO Aufbewahrungsfrist badge */}
              {(() => {
                const retention = getRetentionInfo(selectedMessage)
                return retention ? (
                  <div className="flex items-center gap-2 mb-4 px-3 py-2 rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800/40">
                    <ShieldCheck className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                    <div>
                      <span className="text-xs font-medium text-blue-700 dark:text-blue-300">
                        Aufbewahrungspflicht: {retention.label}
                      </span>
                      <p className="text-[10px] text-blue-600/70 dark:text-blue-400/70 mt-0.5">
                        {retention.basis}
                      </p>
                    </div>
                  </div>
                ) : null
              })()}

              {/* HTML body or text fallback */}
              {selectedMessage.body_html ? (
                <div
                  className="text-sm text-text-body leading-relaxed prose prose-sm max-w-none"
                  dangerouslySetInnerHTML={{ __html: selectedMessage.body_html }}
                />
              ) : (
                <div className="text-sm text-text-body whitespace-pre-line leading-relaxed">
                  {selectedMessage.body_text}
                </div>
              )}

              {/* Attachments */}
              {selectedMessage.attachments && selectedMessage.attachments.length > 0 && (
                <div className="mt-6 border-t border-border pt-4">
                  <p className="text-xs font-medium text-muted-foreground mb-2">
                    Anhänge ({selectedMessage.attachments.length})
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {selectedMessage.attachments.map((att) => (
                      <button
                        key={att.id}
                        onClick={() => toast.success(`"${att.filename}" wird heruntergeladen`)}
                        className="flex items-center gap-2 rounded-xl border border-border bg-card px-3 py-2 hover:bg-secondary hover:shadow-sm transition-all"
                      >
                        <Paperclip className="h-4 w-4 text-muted-foreground" />
                        <div className="text-left">
                          <p className="text-sm text-foreground">{att.filename}</p>
                          <p className="text-[10px] text-muted-foreground">{(att.size_bytes / 1024).toFixed(0)} KB</p>
                        </div>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Quick reply */}
              <div className="mt-6 border-t border-border pt-4">
                <button
                  onClick={() => openCompose('reply', selectedMessage)}
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
              <p className="text-sm">Wähle eine E-Mail aus</p>
            </div>
          </div>
        )}
      </div>

      {/* Delete Confirm */}
      <ConfirmDialog
        open={!!deleteConfirmId}
        onOpenChange={(open) => !open && setDeleteConfirmId(null)}
        title="E-Mail löschen?"
        description={`"${deleteTarget?.subject}" wird gelöscht.`}
        confirmLabel="Löschen"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}
