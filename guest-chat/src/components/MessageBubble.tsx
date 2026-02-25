import type { ChatMessage } from '../api/types'

interface MessageBubbleProps {
  message: ChatMessage
  isSelf: boolean
}

function formatTime(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMin = Math.floor(diffMs / 60_000)

  if (diffMin < 1) return 'Jetzt'
  if (diffMin < 60) return `vor ${diffMin} Min.`
  return date.toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })
}

function getSenderName(message: ChatMessage, isSelf: boolean): string {
  if (isSelf) return 'Sie'
  if (message.sender_first_name || message.sender_last_name) {
    return [message.sender_first_name, message.sender_last_name].filter(Boolean).join(' ')
  }
  return 'Kundenservice'
}

function isImageType(contentType: string): boolean {
  return contentType.startsWith('image/')
}

export function MessageBubble({ message, isSelf }: MessageBubbleProps) {
  const isDeleted = !message.content && (!message.files || message.files.length === 0)

  return (
    <div className={`message-row ${isSelf ? 'self' : 'other'}`}>
      <span className="message-sender">{getSenderName(message, isSelf)}</span>
      <div className={`message-bubble ${isDeleted ? 'deleted' : ''}`}>
        {isDeleted ? (
          'Nachricht entfernt'
        ) : (
          <>
            {message.content}
            {message.files?.map((file) =>
              isImageType(file.content_type) ? (
                <img
                  key={file.id}
                  src={file.url}
                  alt={file.filename}
                  className="message-file-preview"
                />
              ) : (
                <a
                  key={file.id}
                  href={file.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="message-file-link"
                >
                  &#128196; {file.filename}
                </a>
              ),
            )}
          </>
        )}
      </div>
      <span className="message-time">{formatTime(message.created_at)}</span>
    </div>
  )
}
