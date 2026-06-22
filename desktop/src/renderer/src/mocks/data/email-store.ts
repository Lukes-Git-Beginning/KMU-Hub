/**
 * Stateful in-memory email store for the MSW demo backend.
 *
 * The raw seed data in `emails.ts` is loosely typed and uses a few legacy field
 * names (`snippet`, `type`, `size`). This store normalises everything to the
 * real `EmailMessageInfo` / `EmailFolderInfo` shape AND keeps a mutable copy so
 * that read/star/move/delete actually persist for the lifetime of the session.
 *
 * Folder counters (unread/total) are derived from the live message list, so the
 * unread badge decrements the moment a mail is opened — the visible payoff of a
 * stateful demo. State resets on reload, which is exactly what we want for a
 * deterministic demo.
 */
import type {
  EmailMessageInfo,
  EmailFolderInfo,
  EmailAccountInfo,
  EmailAttachmentInfo,
  EmailSignatureInfo,
  EmailLabelInfo,
  EmailRuleInfo,
  ListMessagesParams,
  ListMessagesResponse,
} from '@/api/email-types'
import {
  mockEmailAccount,
  mockEmailFolders,
  mockEmailMessages,
  mockSignatures,
} from './emails'
import { IDS } from './shared-ids'
import { now, daysAgo, hoursAgo } from './date-helpers'

// ---------------------------------------------------------------------------
// Normalisation
// ---------------------------------------------------------------------------

type Raw = Record<string, unknown>

function str(v: unknown, fallback = ''): string {
  return typeof v === 'string' ? v : fallback
}
function num(v: unknown, fallback = 0): number {
  return typeof v === 'number' ? v : fallback
}
function bool(v: unknown, fallback = false): boolean {
  return typeof v === 'boolean' ? v : fallback
}

function normalizeAttachment(raw: Raw): EmailAttachmentInfo {
  return {
    id: str(raw.id),
    filename: str(raw.filename),
    content_type: str(raw.content_type ?? raw.mime_type, 'application/octet-stream'),
    size_bytes: num(raw.size_bytes ?? raw.size),
    minio_key: str(raw.minio_key),
    content_id: str(raw.content_id),
    is_inline: bool(raw.is_inline),
  }
}

function normalizeMessage(raw: Raw, fallbackAccountId: string): EmailMessageInfo {
  const attachments = Array.isArray(raw.attachments)
    ? (raw.attachments as Raw[]).map(normalizeAttachment)
    : []
  const folderId = str(raw.folder_id)
  return {
    id: str(raw.id),
    account_id: str(raw.account_id, fallbackAccountId),
    folder_id: folderId,
    uid: num(raw.uid),
    message_id_header: str(raw.message_id_header, `<${str(raw.id)}@cosmi.local>`),
    in_reply_to: str(raw.in_reply_to),
    references: Array.isArray(raw.references) ? (raw.references as string[]) : [],
    thread_id: str(raw.thread_id, str(raw.id)),
    from: (raw.from as EmailMessageInfo['from']) ?? { name: '', email: '' },
    to: (raw.to as EmailMessageInfo['to']) ?? [],
    cc: (raw.cc as EmailMessageInfo['cc']) ?? [],
    bcc: (raw.bcc as EmailMessageInfo['bcc']) ?? [],
    subject: str(raw.subject),
    // legacy seeds use `snippet`; the UI reads `preview`
    preview: str(raw.preview ?? raw.snippet),
    body_text: str(raw.body_text),
    body_html: str(raw.body_html),
    is_read: bool(raw.is_read),
    is_starred: bool(raw.is_starred),
    is_draft: bool(raw.is_draft, folderId === IDS.emailFolders.drafts),
    has_attachments: bool(raw.has_attachments, attachments.length > 0),
    date: str(raw.date, now()),
    size_bytes: num(raw.size_bytes, 12000),
    attachments,
    created_at: str(raw.created_at, str(raw.date, now())),
    updated_at: str(raw.updated_at, str(raw.date, now())),
    label_ids: Array.isArray(raw.label_ids) ? (raw.label_ids as string[]) : [],
  }
}

function normalizeFolder(raw: Raw, accountId: string): EmailFolderInfo {
  const ts = now()
  return {
    id: str(raw.id),
    account_id: str(raw.account_id, accountId),
    name: str(raw.name),
    imap_name: str(raw.imap_name, str(raw.name)),
    // legacy seeds use `type`; the UI reads `folder_type`
    folder_type: (str(raw.folder_type ?? raw.type) || 'custom') as EmailFolderInfo['folder_type'],
    uid_validity: num(raw.uid_validity, 1),
    message_count: num(raw.message_count ?? raw.total_count),
    unread_count: num(raw.unread_count),
    sort_order: num(raw.sort_order),
    created_at: str(raw.created_at, ts),
    updated_at: str(raw.updated_at, ts),
  }
}

function normalizeAccount(raw: Raw): EmailAccountInfo {
  const ts = now()
  return {
    id: str(raw.id),
    user_id: str(raw.user_id, 'user-self'),
    email_address: str(raw.email_address ?? raw.email),
    display_name: str(raw.display_name ?? raw.name),
    imap_host: str(raw.imap_host),
    imap_port: num(raw.imap_port, 993),
    smtp_host: str(raw.smtp_host),
    smtp_port: num(raw.smtp_port, 587),
    username: str(raw.username ?? raw.email_address ?? raw.email),
    use_ssl: bool(raw.use_ssl, true),
    sync_enabled: bool(raw.sync_enabled ?? raw.is_active, true),
    last_sync_at: str(raw.last_sync_at, ts),
    created_at: str(raw.created_at, ts),
    updated_at: str(raw.updated_at, ts),
  }
}

// ---------------------------------------------------------------------------
// State (mutable, module-level)
// ---------------------------------------------------------------------------

interface EmailState {
  accounts: EmailAccountInfo[]
  folders: EmailFolderInfo[]
  messages: EmailMessageInfo[]
  signatures: EmailSignatureInfo[]
  labels: EmailLabelInfo[]
  rules: EmailRuleInfo[]
}

const seedLabels: EmailLabelInfo[] = [
  { id: 'lbl-wichtig', name: 'Wichtig', color: '#EF4444' },
  { id: 'lbl-kunde', name: 'Kunde', color: '#10B981' },
  { id: 'lbl-rechnung', name: 'Rechnung', color: '#F59E0B' },
  { id: 'lbl-intern', name: 'Intern', color: '#6B7280' },
]

const seedRules: EmailRuleInfo[] = [
  { id: 'rule-1', name: 'Rechnungen markieren', field: 'subject', op: 'contains', value: 'Rechnung', action_type: 'label', action_target: 'lbl-rechnung' },
  { id: 'rule-2', name: 'Bewerbungen archivieren', field: 'subject', op: 'contains', value: 'Bewerbung', action_type: 'label', action_target: 'lbl-wichtig' },
]

// Messages that start out tagged so labels are visible immediately.
const seedMessageLabels: Record<string, string[]> = {
  'em-001': ['lbl-kunde'],
  'em-003': ['lbl-rechnung'],
  'em-009': ['lbl-kunde', 'lbl-wichtig'],
  'em-011': ['lbl-intern'],
}

/**
 * Secondary demo accounts (shared mailboxes) so the account switcher and the
 * unified inbox have something to show. Each gets its own inbox/sent/archive
 * folder set plus a few messages.
 */
function buildSecondary(): { accounts: EmailAccountInfo[]; folders: EmailFolderInfo[]; messages: EmailMessageInfo[] } {
  const accountsRaw: Raw[] = [
    { id: 'email-acc-002', email: 'info@techvision.de', name: 'TechVision Info', imap_host: 'imap.techvision.de', smtp_host: 'smtp.techvision.de' },
    { id: 'email-acc-003', email: 'support@techvision.de', name: 'TechVision Support', imap_host: 'imap.techvision.de', smtp_host: 'smtp.techvision.de' },
  ]
  const accounts = accountsRaw.map(normalizeAccount)

  const folders: EmailFolderInfo[] = []
  const messages: EmailMessageInfo[] = []
  const folderTypes: EmailFolderInfo['folder_type'][] = ['inbox', 'sent', 'archive']
  const folderNames: Record<string, string> = { inbox: 'Posteingang', sent: 'Gesendet', archive: 'Archiv' }

  for (const acc of accounts) {
    const short = acc.id.slice(-3) // '002' | '003'
    folderTypes.forEach((type, i) => {
      folders.push(
        normalizeFolder(
          { id: `ef-${short}-${type}`, name: folderNames[type], folder_type: type, sort_order: i },
          acc.id,
        ),
      )
    })
  }

  // info@ inbox — generic enquiries
  const infoInbox = 'ef-002-inbox'
  const infoMsgs: Raw[] = [
    { id: 'em-i-001', subject: 'Frage zu Ihren Preisen', from: { name: 'Markus Lang', email: 'm.lang@beispiel.de' }, to: [{ name: 'Info', email: 'info@techvision.de' }], date: hoursAgo(3), snippet: 'Guten Tag, ich interessiere mich für Cosmi und hätte eine Frage zu den Preismodellen...', body_html: '<p>Guten Tag,</p><p>ich interessiere mich für Cosmi und hätte eine Frage zu den Preismodellen. Gibt es eine Übersicht?</p><p>Beste Grüße<br/>Markus Lang</p>', folder_id: infoInbox, is_read: false, is_starred: false, thread_id: 'th-i-001' },
    { id: 'em-i-002', subject: 'Newsletter-Anmeldung', from: { name: 'Julia Berg', email: 'j.berg@example.com' }, to: [{ name: 'Info', email: 'info@techvision.de' }], date: hoursAgo(20), snippet: 'Bitte tragen Sie mich in Ihren Newsletter ein. Vielen Dank!', body_html: '<p>Bitte tragen Sie mich in Ihren Newsletter ein. Vielen Dank!</p><p>Julia Berg</p>', folder_id: infoInbox, is_read: true, is_starred: false, thread_id: 'th-i-002' },
    { id: 'em-i-003', subject: 'Pressekontakt gesucht', from: { name: 'Redaktion KMU-Magazin', email: 'redaktion@kmu-magazin.de' }, to: [{ name: 'Info', email: 'info@techvision.de' }], date: daysAgo(2), snippet: 'Sehr geehrte Damen und Herren, für einen Artikel über DACH-CRM-Anbieter...', body_html: '<p>Sehr geehrte Damen und Herren,</p><p>für einen Artikel über DACH-CRM-Anbieter würden wir gerne ein kurzes Statement von Ihnen einholen.</p>', folder_id: infoInbox, is_read: false, is_starred: true, thread_id: 'th-i-003' },
  ]

  // support@ inbox — support tickets
  const supInbox = 'ef-003-inbox'
  const supMsgs: Raw[] = [
    { id: 'em-su-001', subject: '[Ticket #4412] Login funktioniert nicht', from: { name: 'Petra Frank', email: 'p.frank@kunde-gmbh.de' }, to: [{ name: 'Support', email: 'support@techvision.de' }], date: hoursAgo(1), snippet: 'Hallo, seit heute morgen kann ich mich nicht mehr einloggen. Fehlermeldung: ungültige Sitzung...', body_html: '<p>Hallo,</p><p>seit heute Morgen kann ich mich nicht mehr einloggen. Fehlermeldung: „ungültige Sitzung". Können Sie helfen?</p><p>Petra Frank</p>', folder_id: supInbox, is_read: false, is_starred: false, thread_id: 'th-su-001' },
    { id: 'em-su-002', subject: '[Ticket #4408] Export nach DATEV', from: { name: 'Bernd Klose', email: 'b.klose@handwerk-klose.de' }, to: [{ name: 'Support', email: 'support@techvision.de' }], date: hoursAgo(6), snippet: 'Der DATEV-Export erzeugt eine leere Datei. Anbei der Screenshot...', body_html: '<p>Der DATEV-Export erzeugt eine leere Datei. Was kann ich tun?</p><p>Bernd Klose</p>', folder_id: supInbox, is_read: true, is_starred: false, thread_id: 'th-su-002' },
    { id: 'em-su-003', subject: '[Ticket #4399] Feature-Wunsch: Dunkelmodus', from: { name: 'Sarah Vogt', email: 's.vogt@vogt-design.ch' }, to: [{ name: 'Support', email: 'support@techvision.de' }], date: daysAgo(1), snippet: 'Gibt es bereits einen Dunkelmodus oder ist das geplant?', body_html: '<p>Gibt es bereits einen Dunkelmodus oder ist das geplant? Das wäre für uns sehr hilfreich.</p>', folder_id: supInbox, is_read: true, is_starred: false, thread_id: 'th-su-003' },
  ]

  for (const raw of [...infoMsgs, ...supMsgs]) {
    const accId = (raw.folder_id as string).startsWith('ef-002') ? 'email-acc-002' : 'email-acc-003'
    messages.push(normalizeMessage({ account_id: accId, ...raw }, accId))
  }

  return { accounts, folders, messages }
}

function seed(): EmailState {
  const account = normalizeAccount((mockEmailAccount.account as Raw) ?? {})
  const accountId = account.id
  const folders = (mockEmailFolders.folders as Raw[]).map((f, i) =>
    normalizeFolder({ sort_order: i, ...f }, accountId),
  )
  const messages: EmailMessageInfo[] = []
  for (const bucket of Object.values(mockEmailMessages)) {
    for (const raw of bucket.messages) {
      messages.push(normalizeMessage(raw as Raw, accountId))
    }
  }
  const signatures = (mockSignatures.signatures as Raw[]).map((s) => ({
    id: str(s.id),
    user_id: str(s.user_id, 'user-self'),
    name: str(s.name),
    html_content: str(s.html_content),
    is_default: bool(s.is_default),
    created_at: now(),
    updated_at: now(),
  }))
  const secondary = buildSecondary()
  const allMessages = [...messages, ...secondary.messages]
  for (const msg of allMessages) {
    if (seedMessageLabels[msg.id]) msg.label_ids = [...seedMessageLabels[msg.id]]
  }
  return {
    accounts: [account, ...secondary.accounts],
    folders: [...folders, ...secondary.folders],
    messages: allMessages,
    signatures,
    labels: seedLabels.map((l) => ({ ...l })),
    rules: seedRules.map((r) => ({ ...r })),
  }
}

const state: EmailState = seed()

/** Reset to seed — used by tests / demo reset. */
export function resetEmailStore(): void {
  const fresh = seed()
  state.accounts = fresh.accounts
  state.folders = fresh.folders
  state.messages = fresh.messages
  state.signatures = fresh.signatures
  state.labels = fresh.labels
  state.rules = fresh.rules
}

// ---------------------------------------------------------------------------
// Derived counters
// ---------------------------------------------------------------------------

function recomputeFolderCounts(): void {
  for (const folder of state.folders) {
    const inFolder = state.messages.filter((m) => m.folder_id === folder.id)
    folder.message_count = inFolder.length
    folder.unread_count = inFolder.filter((m) => !m.is_read).length
  }
}
recomputeFolderCounts()

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

export function getAccounts(): EmailAccountInfo[] {
  return state.accounts
}

export function getFolders(accountId?: string): EmailFolderInfo[] {
  recomputeFolderCounts()
  if (!accountId) return state.folders
  return state.folders.filter((f) => f.account_id === accountId)
}

export function getMessage(id: string): EmailMessageInfo | undefined {
  return state.messages.find((m) => m.id === id)
}

export function getThread(threadId: string): EmailMessageInfo[] {
  return state.messages
    .filter((m) => m.thread_id === threadId)
    .sort((a, b) => a.date.localeCompare(b.date))
}

function matchesSearch(m: EmailMessageInfo, q: string): boolean {
  const needle = q.toLowerCase()
  return (
    m.subject.toLowerCase().includes(needle) ||
    m.preview.toLowerCase().includes(needle) ||
    m.body_text.toLowerCase().includes(needle) ||
    m.from.name.toLowerCase().includes(needle) ||
    m.from.email.toLowerCase().includes(needle) ||
    m.to.some((a) => a.name.toLowerCase().includes(needle) || a.email.toLowerCase().includes(needle))
  )
}

export interface ListMessagesExtra extends ListMessagesParams {
  /** 'unified' returns the inbox of every account merged. */
  view?: 'folder' | 'unified'
  account_id?: string
  filter?: 'all' | 'unread' | 'starred'
  label_id?: string
}

export function listMessages(params: ListMessagesExtra): ListMessagesResponse {
  let list = [...state.messages]

  // Scope: unified inbox, single folder, or all-of-account
  if (params.view === 'unified') {
    const inboxIds = new Set(
      state.folders.filter((f) => f.folder_type === 'inbox').map((f) => f.id),
    )
    list = list.filter((m) => inboxIds.has(m.folder_id))
  } else if (params.folder_id) {
    list = list.filter((m) => m.folder_id === params.folder_id)
  }

  if (params.account_id) {
    list = list.filter((m) => m.account_id === params.account_id)
  }
  if (params.filter === 'unread') list = list.filter((m) => !m.is_read)
  if (params.filter === 'starred') list = list.filter((m) => m.is_starred)
  if (params.label_id) list = list.filter((m) => m.label_ids?.includes(params.label_id!))
  if (params.search) list = list.filter((m) => matchesSearch(m, params.search!))

  // Sort
  const sortBy = params.sort_by ?? 'date'
  const desc = params.sort_desc ?? true
  list.sort((a, b) => {
    let cmp = 0
    if (sortBy === 'subject') cmp = a.subject.localeCompare(b.subject)
    else if (sortBy === 'from') cmp = (a.from.name || a.from.email).localeCompare(b.from.name || b.from.email)
    else cmp = a.date.localeCompare(b.date)
    return desc ? -cmp : cmp
  })

  const total = list.length
  const page = params.page ?? 1
  const perPage = params.per_page ?? 50
  const start = (page - 1) * perPage
  const slice = list.slice(start, start + perPage).map((m) => ({
    ...m,
    thread_count: state.messages.filter((x) => x.thread_id === m.thread_id).length,
  }))
  return { messages: slice, total }
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

function touch(m: EmailMessageInfo): void {
  m.updated_at = now()
}

export function setRead(id: string, isRead: boolean): EmailMessageInfo | undefined {
  const m = getMessage(id)
  if (m) {
    m.is_read = isRead
    touch(m)
    recomputeFolderCounts()
  }
  return m
}

export function toggleStar(id: string): boolean {
  const m = getMessage(id)
  if (!m) return false
  m.is_starred = !m.is_starred
  touch(m)
  return m.is_starred
}

export function moveToFolder(id: string, targetFolderId: string): EmailMessageInfo | undefined {
  const m = getMessage(id)
  if (m) {
    m.folder_id = targetFolderId
    touch(m)
    recomputeFolderCounts()
  }
  return m
}

/** Soft-delete: move to trash; hard-delete if already in trash. */
export function deleteMessage(id: string): void {
  const m = getMessage(id)
  if (!m) return
  if (m.folder_id === IDS.emailFolders.trash) {
    state.messages = state.messages.filter((x) => x.id !== id)
  } else {
    m.folder_id = IDS.emailFolders.trash
    touch(m)
  }
  recomputeFolderCounts()
}

export interface AppendMessageInput {
  subject?: string
  to?: EmailMessageInfo['to']
  cc?: EmailMessageInfo['cc']
  bcc?: EmailMessageInfo['bcc']
  body_html?: string
  body_text?: string
  folderId: string
  is_draft?: boolean
  thread_id?: string
  in_reply_to?: string
}

let appendCounter = 0

/** Append a sent/draft message and return it (used by send/draft/reply/forward). */
export function appendMessage(input: AppendMessageInput): EmailMessageInfo {
  const account = state.accounts[0]
  appendCounter += 1
  const id = `em-${input.is_draft ? 'draft' : 'sent'}-new-${appendCounter}`
  const text = input.body_text ?? ''
  const msg: EmailMessageInfo = {
    id,
    account_id: account.id,
    folder_id: input.folderId,
    uid: 0,
    message_id_header: `<${id}@cosmi.local>`,
    in_reply_to: input.in_reply_to ?? '',
    references: [],
    thread_id: input.thread_id ?? id,
    from: { name: account.display_name, email: account.email_address },
    to: input.to ?? [],
    cc: input.cc ?? [],
    bcc: input.bcc ?? [],
    subject: input.subject ?? '(Kein Betreff)',
    preview: text.slice(0, 140),
    body_text: text,
    body_html: input.body_html ?? '',
    is_read: true,
    is_starred: false,
    is_draft: input.is_draft ?? false,
    has_attachments: false,
    date: now(),
    size_bytes: Math.max(2000, text.length * 8),
    attachments: [],
    created_at: now(),
    updated_at: now(),
    label_ids: [],
  }
  state.messages.unshift(msg)
  recomputeFolderCounts()
  return msg
}

export function getSignatures(): EmailSignatureInfo[] {
  return state.signatures
}

// ---------------------------------------------------------------------------
// Labels
// ---------------------------------------------------------------------------

let labelCounter = 0
let ruleCounter = 0

export function getLabels(): EmailLabelInfo[] {
  return state.labels
}

export function createLabel(name: string, color: string): EmailLabelInfo {
  labelCounter += 1
  const label: EmailLabelInfo = { id: `lbl-new-${labelCounter}`, name, color }
  state.labels.push(label)
  return label
}

export function updateLabel(id: string, patch: Partial<Pick<EmailLabelInfo, 'name' | 'color'>>): void {
  const label = state.labels.find((l) => l.id === id)
  if (label) Object.assign(label, patch)
}

export function deleteLabel(id: string): void {
  state.labels = state.labels.filter((l) => l.id !== id)
  for (const m of state.messages) {
    if (m.label_ids?.includes(id)) m.label_ids = m.label_ids.filter((x) => x !== id)
  }
}

/** Replace the label set of a message. */
export function setMessageLabels(messageId: string, labelIds: string[]): EmailMessageInfo | undefined {
  const m = getMessage(messageId)
  if (m) {
    m.label_ids = [...labelIds]
    touch(m)
  }
  return m
}

// ---------------------------------------------------------------------------
// Rules
// ---------------------------------------------------------------------------

export function getRules(): EmailRuleInfo[] {
  return state.rules
}

export function createRule(rule: Omit<EmailRuleInfo, 'id'>): EmailRuleInfo {
  ruleCounter += 1
  const created: EmailRuleInfo = { id: `rule-new-${ruleCounter}`, ...rule }
  state.rules.push(created)
  return created
}

export function deleteRule(id: string): void {
  state.rules = state.rules.filter((r) => r.id !== id)
}

function ruleMatches(rule: EmailRuleInfo, m: EmailMessageInfo): boolean {
  const haystack =
    rule.field === 'from'
      ? `${m.from.name} ${m.from.email}`.toLowerCase()
      : m.subject.toLowerCase()
  return haystack.includes(rule.value.toLowerCase())
}

/** Run every rule over the non-trash messages; returns how many messages changed. */
export function applyRules(): number {
  let affected = 0
  for (const m of state.messages) {
    if (m.folder_id === IDS.emailFolders.trash) continue
    let changed = false
    for (const rule of state.rules) {
      if (!ruleMatches(rule, m)) continue
      if (rule.action_type === 'label') {
        if (!m.label_ids?.includes(rule.action_target)) {
          m.label_ids = [...(m.label_ids ?? []), rule.action_target]
          changed = true
        }
      } else if (rule.action_type === 'move') {
        if (m.folder_id !== rule.action_target) {
          m.folder_id = rule.action_target
          changed = true
        }
      }
    }
    if (changed) {
      touch(m)
      affected += 1
    }
  }
  recomputeFolderCounts()
  return affected
}
