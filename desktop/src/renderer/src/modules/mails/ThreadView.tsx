import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Paperclip, ChevronDown, MoreHorizontal } from 'lucide-react'
import { sanitizeMailBody } from '@/lib/sanitize'
import { formatDate, formatTime } from '@/lib/format'
import type { EmailMessageInfo, EmailAttachmentInfo } from '@/api/email-types'

interface ThreadViewProps {
  /** Thread messages, sorted oldest → newest. */
  messages: EmailMessageInfo[]
  /** The current account address — used to mark our own messages. */
  selfEmail: string
  onDownloadAttachment: (att: EmailAttachmentInfo, subject: string) => void
}

/** Split a mail body into the new content and the quoted history (after the first <hr>). */
function splitQuote(html: string): { main: string; quoted: string | null } {
  const m = html.match(/<hr\s*\/?>/i)
  if (m && m.index !== undefined && /<blockquote/i.test(html.slice(m.index))) {
    return { main: html.slice(0, m.index), quoted: html.slice(m.index) }
  }
  return { main: html, quoted: null }
}

function initials(addr: { name: string; email: string }): string {
  if (addr.name) return addr.name.split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2)
  return addr.email[0].toUpperCase()
}

function MessageCard({
  msg,
  expanded,
  isSelf,
  onToggle,
  onDownloadAttachment,
}: {
  msg: EmailMessageInfo
  expanded: boolean
  isSelf: boolean
  onToggle: () => void
  onDownloadAttachment: (att: EmailAttachmentInfo, subject: string) => void
}) {
  const { t } = useTranslation()
  const [showQuoted, setShowQuoted] = useState(false)
  const { main, quoted } = useMemo(
    () => splitQuote(msg.body_html || `<p>${msg.body_text}</p>`),
    [msg.body_html, msg.body_text],
  )

  return (
    <div
      className={`rounded-xl border transition-colors ${
        expanded ? 'border-border bg-card shadow-sm' : 'border-border-muted bg-card/40'
      } ${isSelf ? 'ml-6' : 'mr-6'}`}
    >
      {/* Header — always visible, toggles expand */}
      <button
        onClick={onToggle}
        className="flex w-full items-start gap-3 px-4 py-3 text-left"
        aria-expanded={expanded}
      >
        <div
          className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-medium ${
            isSelf ? 'bg-primary text-primary-foreground' : 'bg-primary-light text-primary'
          }`}
        >
          {initials(msg.from)}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <span className="text-sm font-medium text-foreground truncate">
              {msg.from.name || msg.from.email}
              {isSelf && (
                <span className="ml-1.5 text-[10px] font-normal text-muted-foreground">
                  {t('mails.thread.you', { defaultValue: 'Du' })}
                </span>
              )}
            </span>
            <span className="text-[10px] text-muted-foreground whitespace-nowrap shrink-0">
              {formatDate(msg.date)} · {formatTime(msg.date)}
            </span>
          </div>
          {expanded ? (
            <p className="text-xs text-muted-foreground truncate">
              {t('mails.detail.to')}: {msg.to.map((a) => a.name || a.email).join(', ')}
            </p>
          ) : (
            <p className="text-xs text-muted-foreground truncate">{msg.preview}</p>
          )}
        </div>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-muted-foreground transition-transform ${expanded ? 'rotate-180' : ''}`}
        />
      </button>

      {/* Body — only when expanded */}
      {expanded && (
        <div className="px-4 pb-4 pl-16">
          <div
            className="text-sm text-text-body leading-relaxed prose prose-sm max-w-none [&_img]:max-w-full [&_img]:rounded-lg"
            dangerouslySetInnerHTML={{ __html: sanitizeMailBody(main) }}
          />

          {/* Quoted history — collapsed behind a "···" toggle */}
          {quoted && (
            <div className="mt-2">
              <button
                onClick={() => setShowQuoted((s) => !s)}
                className="flex items-center gap-1 rounded-md bg-secondary px-2 py-0.5 text-muted-foreground hover:bg-secondary/70 transition-colors"
                aria-label={t('mails.thread.toggleQuote', { defaultValue: 'Zitierten Verlauf anzeigen' })}
                title={t('mails.thread.toggleQuote', { defaultValue: 'Zitierten Verlauf anzeigen' })}
              >
                <MoreHorizontal className="h-3.5 w-3.5" />
              </button>
              {showQuoted && (
                <div
                  className="mt-2 border-l-2 border-border pl-3 text-xs text-muted-foreground prose prose-sm max-w-none"
                  dangerouslySetInnerHTML={{ __html: sanitizeMailBody(quoted) }}
                />
              )}
            </div>
          )}

          {/* Attachments */}
          {msg.attachments && msg.attachments.length > 0 && (
            <div className="mt-4 flex flex-wrap gap-2">
              {msg.attachments.map((att) => (
                <button
                  key={att.id}
                  onClick={() => onDownloadAttachment(att, msg.subject)}
                  className="flex items-center gap-2 rounded-xl border border-border bg-card px-3 py-2 hover:bg-secondary hover:shadow-sm transition-all"
                >
                  <Paperclip className="h-4 w-4 text-muted-foreground" />
                  <div className="text-left">
                    <p className="text-sm text-foreground">{att.filename}</p>
                    <p className="text-[10px] text-muted-foreground">
                      {(att.size_bytes / 1024).toFixed(0)} KB
                    </p>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

/**
 * Conversation view for an email thread. The newest message is expanded by
 * default; older ones collapse to a one-line header (click to expand), mirroring
 * modern mail clients. Quoted reply history hides behind a "···" toggle.
 */
export function ThreadView({ messages, selfEmail, onDownloadAttachment }: ThreadViewProps) {
  const latestId = messages.length ? messages[messages.length - 1].id : ''
  const [expandedIds, setExpandedIds] = useState<Set<string>>(() => new Set([latestId]))

  const toggle = (id: string) =>
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  if (messages.length <= 1) {
    // Single message — render its body inline expanded (no card chrome).
    const only = messages[0]
    if (!only) return null
    return (
      <div data-testid="mail-thread">
        <MessageCard
          msg={only}
          expanded
          isSelf={only.from.email === selfEmail}
          onToggle={() => {}}
          onDownloadAttachment={onDownloadAttachment}
        />
      </div>
    )
  }

  return (
    <div className="space-y-2.5" data-testid="mail-thread">
      {messages.map((msg) => (
        <MessageCard
          key={msg.id}
          msg={msg}
          expanded={expandedIds.has(msg.id)}
          isSelf={msg.from.email === selfEmail}
          onToggle={() => toggle(msg.id)}
          onDownloadAttachment={onDownloadAttachment}
        />
      ))}
    </div>
  )
}
