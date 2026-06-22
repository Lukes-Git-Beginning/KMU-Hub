import { useTranslation } from 'react-i18next'
import { FileText, Image, File, Reply, Forward, Copy, BarChart3, AlarmClock, Check } from 'lucide-react'
import { toast } from 'sonner'
import type { ConversationMessage, ConversationPoll, ConversationReminder } from '@/types/communication'
import { useInboxThread } from '@/stores/inboxThread'
import { formatDate as libFormatDate, formatTime as libFormatTime } from '@/lib/format'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatTime(timestamp: string): string {
  return libFormatTime(timestamp, { hour: '2-digit', minute: '2-digit' })
}

function formatDayHeading(timestamp: string): string {
  return libFormatDate(timestamp, { weekday: 'short', day: '2-digit', month: 'short' })
}

function getFileIcon(type: string) {
  if (type.startsWith('image/')) return Image
  if (type.includes('pdf')) return FileText
  return File
}

function getInitials(name: string): string {
  return name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface MessageItemProps {
  message: ConversationMessage
  showDate?: boolean
}

export function MessageItem({ message: msg, showDate }: MessageItemProps) {
  const { t } = useTranslation()
  const isOutbound = msg.direction === 'outbound'
  const isInternal = msg.direction === 'internal'

  return (
    <>
      {/* Date separator */}
      {showDate && (
        <div className="flex items-center gap-3 py-2">
          <div className="flex-1 border-t border-border" />
          <span className="text-[10px] text-muted-foreground shrink-0">
            {formatDayHeading(msg.timestamp)}
          </span>
          <div className="flex-1 border-t border-border" />
        </div>
      )}

      <div className={`group/msg flex gap-2.5 ${isOutbound ? 'flex-row-reverse' : ''}`}>
        {/* Avatar */}
        <div
          className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-[10px] font-medium ${
            isInternal
              ? 'bg-warning/20 text-warning'
              : isOutbound
                ? 'bg-primary/15 text-primary'
                : 'bg-secondary text-muted-foreground'
          }`}
        >
          {getInitials(msg.senderName)}
        </div>

        {/* Bubble */}
        <div className={`relative max-w-[70%] min-w-[140px] ${isOutbound ? 'items-end' : ''}`}>
          {/* Hover action toolbar */}
          <div className={`absolute -top-3 ${isOutbound ? 'left-0' : 'right-0'} hidden group-hover/msg:flex items-center gap-0.5 rounded-md border border-border bg-card px-1 py-0.5 shadow-sm z-10`}>
            <button
              onClick={() => toast.info(t('kommunikation.message.preparingReply'))}
              className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              title={t('kommunikation.message.reply')}
            >
              <Reply className="h-3 w-3" />
            </button>
            <button
              onClick={() => toast.info(t('kommunikation.message.preparingForward'))}
              className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              title={t('kommunikation.message.forward')}
            >
              <Forward className="h-3 w-3" />
            </button>
            <button
              onClick={() => {
                navigator.clipboard.writeText(msg.content)
                toast.success(t('kommunikation.message.copied'))
              }}
              className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              title={t('kommunikation.message.copy')}
            >
              <Copy className="h-3 w-3" />
            </button>
          </div>
          {/* Sender + time */}
          <div className={`flex items-center gap-2 mb-0.5 ${isOutbound ? 'flex-row-reverse' : ''}`}>
            <span className="text-[11px] font-medium text-foreground">{msg.senderName}</span>
            {isInternal && (
              <span className="rounded bg-warning/15 px-1.5 py-0.5 text-[9px] font-semibold text-warning">
                {t('kommunikation.note.internalNote')}
              </span>
            )}
            <span className="text-[10px] text-muted-foreground">{formatTime(msg.timestamp)}</span>
          </div>

          {/* Content */}
          <div
            className={`rounded-lg px-3 py-2 ${
              isInternal
                ? 'bg-warning/8 border border-warning/20'
                : isOutbound
                  ? 'bg-primary/8 border border-primary/10'
                  : 'bg-secondary/80'
            }`}
          >
            {msg.poll ? (
              <PollView poll={msg.poll} convId={msg.conversationId} messageId={msg.id} />
            ) : msg.reminder ? (
              <ReminderView reminder={msg.reminder} />
            ) : (
              <p className="text-[13px] text-foreground/90 whitespace-pre-line leading-relaxed">
                {msg.content}
              </p>
            )}

            {/* Attachments */}
            {msg.attachments.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1.5">
                {msg.attachments.map((att) => {
                  const AttIcon = getFileIcon(att.type)
                  return (
                    <button
                      key={att.id}
                      className="inline-flex items-center gap-1.5 rounded-md bg-background border border-border px-2 py-1 text-[11px] text-foreground hover:bg-accent transition-colors"
                    >
                      <AttIcon className="h-3 w-3 text-muted-foreground" />
                      <span className="max-w-[120px] truncate">{att.name}</span>
                      <span className="text-muted-foreground/60">{att.size}</span>
                    </button>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  )
}

// ---------------------------------------------------------------------------
// Slash-command blocks (KO-8)
// ---------------------------------------------------------------------------

function PollView({ poll, convId, messageId }: { poll: ConversationPoll; convId: string; messageId: string }) {
  const { t } = useTranslation()
  const votePoll = useInboxThread((s) => s.votePoll)
  const total = poll.options.reduce((sum, o) => sum + o.votes, 0)

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        <BarChart3 className="h-3.5 w-3.5 text-primary" />
        <span className="text-[13px] font-medium text-foreground">{poll.question}</span>
      </div>
      <div className="space-y-1">
        {poll.options.map((o) => {
          const pct = total ? Math.round((o.votes / total) * 100) : 0
          const voted = poll.votedOptionId === o.id
          return (
            <button
              key={o.id}
              type="button"
              onClick={() => votePoll(convId, messageId, o.id)}
              className={`relative flex w-full items-center justify-between overflow-hidden rounded-md border px-2.5 py-1.5 text-left text-[12px] transition-colors ${
                voted ? 'border-primary/50' : 'border-border hover:bg-accent'
              }`}
            >
              <span className="absolute inset-y-0 left-0 bg-primary/10" style={{ width: `${pct}%` }} aria-hidden />
              <span className="relative flex items-center gap-1.5 text-foreground">
                {voted && <Check className="h-3 w-3 text-primary" />}
                {o.label}
              </span>
              <span className="relative shrink-0 text-[11px] tabular-nums text-muted-foreground">
                {o.votes} · {pct}%
              </span>
            </button>
          )
        })}
      </div>
      <p className="text-[10px] text-muted-foreground">{t('kommunikation.poll.totalVotes', { count: total })}</p>
    </div>
  )
}

function ReminderView({ reminder }: { reminder: ConversationReminder }) {
  const { t } = useTranslation()
  return (
    <div className="flex items-start gap-2">
      <AlarmClock className="mt-0.5 h-4 w-4 shrink-0 text-warning" />
      <div className="min-w-0">
        <p className="text-[13px] text-foreground">{reminder.text}</p>
        <p className="text-[10px] text-muted-foreground">
          {t('kommunikation.reminder.due', {
            date: libFormatDate(reminder.dueAt, { weekday: 'short', day: '2-digit', month: 'short' }),
            time: libFormatTime(reminder.dueAt, { hour: '2-digit', minute: '2-digit' }),
          })}
        </p>
      </div>
    </div>
  )
}
