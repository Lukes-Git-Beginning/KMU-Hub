import { FileText, Image, File, Reply, Forward, Copy, MoreHorizontal } from 'lucide-react'
import { toast } from 'sonner'
import type { ConversationMessage } from '@/types/communication'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatTime(timestamp: string): string {
  try {
    return new Date(timestamp).toLocaleTimeString('de-CH', {
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return ''
  }
}

function formatDate(timestamp: string): string {
  try {
    return new Date(timestamp).toLocaleDateString('de-CH', {
      weekday: 'short',
      day: '2-digit',
      month: 'short',
    })
  } catch {
    return ''
  }
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
  const isOutbound = msg.direction === 'outbound'
  const isInternal = msg.direction === 'internal'

  return (
    <>
      {/* Date separator */}
      {showDate && (
        <div className="flex items-center gap-3 py-2">
          <div className="flex-1 border-t border-border" />
          <span className="text-[10px] text-muted-foreground shrink-0">
            {formatDate(msg.timestamp)}
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
              onClick={() => toast.info('Antwort wird vorbereitet...')}
              className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              title="Antworten"
            >
              <Reply className="h-3 w-3" />
            </button>
            <button
              onClick={() => toast.info('Weiterleitung wird vorbereitet...')}
              className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              title="Weiterleiten"
            >
              <Forward className="h-3 w-3" />
            </button>
            <button
              onClick={() => {
                navigator.clipboard.writeText(msg.content)
                toast.success('Kopiert')
              }}
              className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              title="Kopieren"
            >
              <Copy className="h-3 w-3" />
            </button>
          </div>
          {/* Sender + time */}
          <div className={`flex items-center gap-2 mb-0.5 ${isOutbound ? 'flex-row-reverse' : ''}`}>
            <span className="text-[11px] font-medium text-foreground">{msg.senderName}</span>
            {isInternal && (
              <span className="rounded bg-warning/15 px-1.5 py-0.5 text-[9px] font-semibold text-warning">
                Interne Notiz
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
            <p className="text-[13px] text-foreground/90 whitespace-pre-line leading-relaxed">
              {msg.content}
            </p>

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
