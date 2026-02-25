import { useCallback, useEffect, useRef, useState } from 'react'
import { apiClient } from '../api/client'
import type { ChatMessage, GuestSession, WSMessage } from '../api/types'
import { useWebSocket } from './useWebSocket'

export function useGuestChat(token: string, session: GuestSession) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSending, setIsSending] = useState(false)
  const [hasMore, setHasMore] = useState(false)
  const [typingUser, setTypingUser] = useState<string | null>(null)
  const typingTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const handleWSMessage = useCallback(
    (msg: WSMessage) => {
      switch (msg.type) {
        case 'message.new': {
          const newMsg = msg.payload as unknown as ChatMessage
          // Skip our own optimistic messages
          if (newMsg.guest_session_id === session.id) return
          setMessages((prev) => [...prev, newMsg])
          break
        }
        case 'typing': {
          const name = msg.payload.display_name as string
          if (name) {
            setTypingUser(name)
            if (typingTimerRef.current) clearTimeout(typingTimerRef.current)
            typingTimerRef.current = setTimeout(() => setTypingUser(null), 3000)
          }
          break
        }
      }
    },
    [session.id],
  )

  const { connectionState, send } = useWebSocket(token, session.channel_id, handleWSMessage)

  // Load initial messages
  useEffect(() => {
    let cancelled = false

    async function loadMessages() {
      setIsLoading(true)
      try {
        const res = await apiClient.getMessages(session.channel_id)
        if (!cancelled) {
          const msgs = res.messages ?? []
          setMessages(msgs)
          setHasMore(msgs.length >= 50)
        }
      } catch {
        // initial load failed -- messages stay empty
      } finally {
        if (!cancelled) setIsLoading(false)
      }
    }

    loadMessages()
    return () => {
      cancelled = true
    }
  }, [session.channel_id])

  const sendMessage = useCallback(
    async (content: string) => {
      if (!content.trim() || isSending) return
      setIsSending(true)

      // Optimistic message
      const optimistic: ChatMessage = {
        id: `pending-${Date.now()}`,
        channel_id: session.channel_id,
        content,
        guest_session_id: session.id,
        guest_display_name: session.display_name,
        created_at: new Date().toISOString(),
      }
      setMessages((prev) => [...prev, optimistic])

      try {
        const res = await apiClient.sendMessage(session.channel_id, content)
        // Replace optimistic with real
        setMessages((prev) =>
          prev.map((m) => (m.id === optimistic.id ? res.message : m)),
        )
      } catch {
        // Remove optimistic on failure
        setMessages((prev) => prev.filter((m) => m.id !== optimistic.id))
      } finally {
        setIsSending(false)
      }
    },
    [session, isSending],
  )

  const loadMore = useCallback(async () => {
    if (!hasMore || isLoading || messages.length === 0) return

    const oldest = messages[0]
    try {
      const res = await apiClient.getMessages(session.channel_id, oldest.id)
      const older = res.messages ?? []
      setMessages((prev) => [...older, ...prev])
      setHasMore(older.length >= 50)
    } catch {
      // ignore load more errors
    }
  }, [hasMore, isLoading, messages, session.channel_id])

  return {
    messages,
    isLoading,
    isSending,
    sendMessage,
    loadMore,
    hasMore,
    typingUser,
    connectionState,
    send,
  }
}
