/**
 * InCallChat — In-call chat panel (Wave 2).
 *
 * Architecture:
 *  - On mount: load persisted history once via useMeetingChatHistory (TanStack Query,
 *    staleTime=Infinity → no mid-session refetch).
 *  - Live messages during the session: LiveKit useChat DataChannel (instant P2P delivery).
 *  - On send: useChat.send() for live delivery + useSaveMeetingChatMessage for persistence.
 *    Persistence is fire-and-forget with an error toast on failure.
 *
 * Identity: sender_id is authoritative; matched against useParticipants roster.
 * sender_name from history is only used as a fallback for participants no longer in the room.
 *
 * Must be rendered inside a LiveKitRoom (uses useChat, useLocalParticipant, useParticipants).
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useChat, useLocalParticipant, useParticipants } from '@livekit/components-react'
import type { ReceivedChatMessage } from '@livekit/components-core'
import { toast } from 'sonner'

import { useMeetingChatHistory, useSaveMeetingChatMessage } from '@/api/hooks/useVideo'
import type { MeetingChatMessage } from '@/api/video-types'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface InCallChatProps {
  meetingId: string
  /** Display name for the local participant (resolved from LiveKit localParticipant). */
  localDisplayName: string
  onClose: () => void
}

// A unified message shape used for rendering (history + live)
interface DisplayMessage {
  id: string
  /** Resolved display name (roster-first, sender_name fallback) */
  senderName: string
  message: string
  /** Unix ms timestamp for sorting */
  timestamp: number
  isLocal: boolean
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function InCallChat({ meetingId, localDisplayName, onClose }: InCallChatProps) {
  const { t } = useTranslation()
  const { localParticipant } = useLocalParticipant()
  const participants = useParticipants()

  // LiveKit DataChannel chat — live delivery to all peers
  const { chatMessages, send: sendLiveKitMessage } = useChat()

  // Persisted history (fetched once on mount)
  const { data: historyMessages = [], isLoading: isHistoryLoading } = useMeetingChatHistory(meetingId)
  const saveChatMessage = useSaveMeetingChatMessage(meetingId)

  const [inputValue, setInputValue] = useState('')
  const [isSending, setIsSending] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)

  // Build an identity map: participant identity → display name (from LiveKit roster)
  const identityToName = useMemo(() => {
    const map = new Map<string, string>()
    for (const p of participants) {
      map.set(p.identity, p.name ?? p.identity)
    }
    return map
  }, [participants])

  // Resolve sender display name: roster-first, sender_name/identity fallback
  const resolveName = useCallback(
    (senderId: string, senderNameFallback: string) =>
      (identityToName.get(senderId) ?? senderNameFallback) || senderId,
    [identityToName],
  )

  // Convert persisted history to DisplayMessage
  const historyDisplayMessages = useMemo((): DisplayMessage[] =>
    historyMessages.map((m: MeetingChatMessage) => ({
      id: `hist-${m.id}`,
      senderName: resolveName(m.sender_id, m.sender_name),
      message: m.message,
      timestamp: new Date(m.created_at).getTime(),
      isLocal: m.sender_id === localParticipant.identity,
    })),
    [historyMessages, resolveName, localParticipant.identity],
  )

  // Convert LiveKit live messages to DisplayMessage
  // sender.identity is the authoritative identity in LiveKit
  const liveDisplayMessages = useMemo((): DisplayMessage[] =>
    chatMessages.map((m: ReceivedChatMessage) => ({
      id: `live-${m.id}`,
      senderName: resolveName(m.from?.identity ?? '', m.from?.name ?? ''),
      message: m.message,
      timestamp: m.timestamp,
      isLocal: m.from?.identity === localParticipant.identity,
    })),
    [chatMessages, resolveName, localParticipant.identity],
  )

  // Merge history + live, deduplicated by message content+timestamp proximity.
  // Strategy: live messages that arrived after history cutoff are appended.
  // Since we never refetch history mid-session, no overlap handling is needed.
  const historyLoadTimestamp = useRef<number | null>(null)
  useEffect(() => {
    if (historyMessages.length > 0 && historyLoadTimestamp.current === null) {
      const latest = Math.max(...historyMessages.map((m) => new Date(m.created_at).getTime()))
      historyLoadTimestamp.current = latest
    }
  }, [historyMessages])

  const allMessages = useMemo((): DisplayMessage[] => {
    const cutoff = historyLoadTimestamp.current ?? 0
    // Only include live messages newer than the history cutoff to avoid duplicates
    // (own messages sent during this session are echoed back via LiveKit DataChannel)
    const newLive = liveDisplayMessages.filter((m) => m.timestamp > cutoff)
    const combined = [...historyDisplayMessages, ...newLive]
    combined.sort((a, b) => a.timestamp - b.timestamp)
    return combined
  }, [historyDisplayMessages, liveDisplayMessages])

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [allMessages.length])

  const handleSend = useCallback(async () => {
    const text = inputValue.trim()
    if (!text || isSending) return

    setIsSending(true)
    try {
      // 1. Live delivery via LiveKit DataChannel
      await sendLiveKitMessage(text)

      // 2. Persist to backend (fire-and-forget)
      saveChatMessage.mutate(
        { message: text, sender_name: localDisplayName },
        {
          onError: () => {
            toast.error(t('features.video.chat.saveError'))
          },
        },
      )

      setInputValue('')
    } catch {
      toast.error(t('features.video.chat.sendError'))
    } finally {
      setIsSending(false)
    }
  }, [inputValue, isSending, sendLiveKitMessage, saveChatMessage, localDisplayName, t])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSend()
      }
    },
    [handleSend],
  )

  return (
    <div className="flex w-72 flex-shrink-0 flex-col border-l border-zinc-800 bg-zinc-900/95 animate-slide-in-right">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-800 px-3 py-2.5">
        <span className="text-sm font-medium text-zinc-200">
          {t('features.video.chat.title')}
        </span>
        <button
          onClick={onClose}
          className="flex h-6 w-6 items-center justify-center rounded text-zinc-400 transition-colors hover:bg-zinc-700 hover:text-zinc-200"
          aria-label={t('features.video.chat.close')}
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
            <line x1="1" y1="1" x2="11" y2="11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <line x1="11" y1="1" x2="1" y2="11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          </svg>
        </button>
      </div>

      {/* Message list */}
      <div className="flex-1 overflow-y-auto p-3 space-y-3">
        {isHistoryLoading && (
          <p className="text-center text-xs text-zinc-500">{t('features.video.chat.loading')}</p>
        )}

        {!isHistoryLoading && allMessages.length === 0 && (
          <p className="text-center text-xs text-zinc-500">{t('features.video.chat.empty')}</p>
        )}

        {allMessages.map((msg) => (
          <div
            key={msg.id}
            className={cn(
              'flex flex-col gap-0.5',
              msg.isLocal ? 'items-end' : 'items-start',
            )}
          >
            <span className="text-[10px] font-medium text-zinc-500">
              {msg.isLocal ? t('features.video.chat.you') : msg.senderName}
            </span>
            <div
              className={cn(
                'max-w-[220px] rounded-2xl px-3 py-1.5 text-sm leading-snug',
                msg.isLocal
                  ? 'rounded-tr-sm bg-primary text-primary-foreground'
                  : 'rounded-tl-sm bg-zinc-800 text-zinc-200',
              )}
            >
              {msg.message}
            </div>
          </div>
        ))}

        {/* Scroll anchor */}
        <div ref={bottomRef} />
      </div>

      {/* Input area */}
      <div className="border-t border-zinc-800 p-3">
        <div className="flex items-end gap-2">
          <textarea
            className="flex-1 resize-none rounded-xl border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-200 placeholder-zinc-500 outline-none transition-colors focus:border-zinc-500 focus:ring-0"
            placeholder={t('features.video.chat.placeholder')}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            rows={1}
            maxLength={2000}
            aria-label={t('features.video.chat.inputLabel')}
          />
          <button
            onClick={handleSend}
            disabled={!inputValue.trim() || isSending}
            className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-40"
            aria-label={t('features.video.chat.send')}
          >
            {/* Send icon — custom SVG, no emoji */}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M14.536 21.686a.5.5 0 0 0 .937-.024l6.5-19a.496.496 0 0 0-.635-.635l-19 6.5a.5.5 0 0 0-.024.937l7.93 3.18a2 2 0 0 1 1.112 1.11z" />
              <path d="m21.854 2.147-10.94 10.939" />
            </svg>
          </button>
        </div>
        <p className="mt-1 text-right text-[10px] text-zinc-600">
          {t('features.video.chat.hint')}
        </p>
      </div>
    </div>
  )
}
