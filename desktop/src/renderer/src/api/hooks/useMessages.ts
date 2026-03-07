/**
 * React Query hooks for chat message operations with WebSocket integration.
 *
 * Provides message CRUD hooks and real-time cache synchronization
 * via WebSocket subscriptions for new, updated, and deleted messages.
 */
import { useCallback, useEffect, useRef } from 'react'
import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import { wsManager } from '@/api/websocket'
import type { components } from '@/api/types'

type MessageInfo = components['schemas']['MessageInfo']

/** Query key factory for message-related queries. */
export const messageKeys = {
  all: ['messages'] as const,
  lists: () => [...messageKeys.all, 'list'] as const,
  list: (channelId: string) => [...messageKeys.lists(), channelId] as const,
  threads: () => [...messageKeys.all, 'thread'] as const,
  thread: (messageId: string) => [...messageKeys.threads(), messageId] as const,
}

interface MessagesPage {
  messages?: MessageInfo[]
  has_more?: boolean
}

/** Fetch messages for a channel with infinite scroll (cursor-based pagination). */
export function useMessages(channelId: string) {
  const query = useInfiniteQuery<MessagesPage>({
    queryKey: messageKeys.list(channelId),
    queryFn: async ({ pageParam }) => {
      const { data, error } = await apiClient.GET('/api/v1/channels/{id}/messages', {
        params: {
          path: { id: channelId },
          query: {
            limit: 50,
            before: pageParam as string | undefined,
          },
        },
      })
      if (error) throw new Error('Failed to load messages')
      return (data ?? { messages: [], has_more: false }) as MessagesPage
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => {
      if (!lastPage.has_more || !lastPage.messages?.length) return undefined
      // The last (oldest) message in the page is the cursor for the next page
      return lastPage.messages[lastPage.messages.length - 1]?.id
    },
    enabled: !!channelId,
  })

  // Flatten all pages into a single array, reversed so newest is last
  const allMessages = query.data?.pages.flatMap((page) => page.messages ?? []).reverse() ?? []

  return { ...query, allMessages }
}

/** Send a message to a channel. */
export function useSendMessage() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      channelId,
      content,
      parentMessageId,
    }: {
      channelId: string
      content: string
      parentMessageId?: string
    }) => {
      const { data, error } = await apiClient.POST('/api/v1/channels/{id}/messages', {
        params: { path: { id: channelId } },
        body: {
          content,
          parent_message_id: parentMessageId,
        },
      })
      if (error) throw new Error('Failed to send message')
      return data
    },
    onSuccess: (_data, variables) => {
      // WebSocket will push the new message for real-time display.
      // Invalidate as a fallback to ensure consistency.
      queryClient.invalidateQueries({ queryKey: messageKeys.list(variables.channelId) })
    },
  })
}

/** Edit a message (author only). */
export function useEditMessage() {
  return useMutation({
    mutationFn: async ({ messageId, content }: { messageId: string; content: string }) => {
      const { data, error } = await apiClient.PUT('/api/v1/messages/{id}', {
        params: { path: { id: messageId } },
        body: { content },
      })
      if (error) throw new Error('Failed to edit message')
      return data
    },
  })
}

/** Delete a message (author or channel admin/owner). */
export function useDeleteMessage() {
  return useMutation({
    mutationFn: async (messageId: string) => {
      const { error } = await apiClient.DELETE('/api/v1/messages/{id}', {
        params: { path: { id: messageId } },
      })
      if (error) throw new Error('Failed to delete message')
    },
  })
}

/**
 * Subscribe to real-time message events for a channel.
 *
 * Handles message.new, message.updated, message.deleted via WebSocket
 * and updates the React Query cache accordingly. Also sends
 * channel.subscribe/unsubscribe on mount/unmount.
 */
export function useMessageWebSocket(channelId: string | null) {
  const queryClient = useQueryClient()

  useEffect(() => {
    if (!channelId) return

    // Subscribe to channel events
    wsManager.send({ type: 'channel.subscribe', channel_id: channelId })

    const unsubNew = wsManager.on('message.new', (data) => {
      const msg = data.message as MessageInfo | undefined
      if (!msg || msg.channel_id !== channelId) return

      // Skip thread replies -- they go in the thread cache
      if (msg.parent_message_id) return

      queryClient.setQueryData(
        messageKeys.list(channelId),
        (old: { pages: MessagesPage[]; pageParams: unknown[] } | undefined) => {
          if (!old) return old
          const firstPage = old.pages[0]
          return {
            ...old,
            pages: [
              {
                ...firstPage,
                messages: [msg, ...(firstPage?.messages ?? [])],
              },
              ...old.pages.slice(1),
            ],
          }
        }
      )
    })

    const unsubUpdated = wsManager.on('message.updated', (data) => {
      const msg = data.message as MessageInfo | undefined
      if (!msg || msg.channel_id !== channelId) return

      queryClient.setQueryData(
        messageKeys.list(channelId),
        (old: { pages: MessagesPage[]; pageParams: unknown[] } | undefined) => {
          if (!old) return old
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages?.map((m) =>
                m.id === msg.id ? msg : m
              ),
            })),
          }
        }
      )
    })

    const unsubDeleted = wsManager.on('message.deleted', (data) => {
      const messageId = data.message_id as string | undefined
      if (!messageId) return

      queryClient.setQueryData(
        messageKeys.list(channelId),
        (old: { pages: MessagesPage[]; pageParams: unknown[] } | undefined) => {
          if (!old) return old
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages?.filter((m) => m.id !== messageId),
            })),
          }
        }
      )
    })

    return () => {
      wsManager.send({ type: 'channel.unsubscribe', channel_id: channelId })
      unsubNew()
      unsubUpdated()
      unsubDeleted()
    }
  }, [channelId, queryClient])
}

/** Fetch thread replies for a message. */
export function useThreadReplies(messageId: string | null) {
  return useQuery({
    queryKey: messageKeys.thread(messageId ?? ''),
    queryFn: async () => {
      if (!messageId) throw new Error('No message ID')
      const { data, error } = await apiClient.GET('/api/v1/messages/{id}/thread', {
        params: {
          path: { id: messageId },
          query: { limit: 50 },
        },
      })
      if (error) throw new Error('Failed to load thread')
      return data
    },
    enabled: !!messageId,
  })
}

/**
 * Subscribe to real-time thread reply events.
 * Updates the thread cache when new replies arrive.
 */
export function useThreadWebSocket(messageId: string | null) {
  const queryClient = useQueryClient()

  useEffect(() => {
    if (!messageId) return

    const unsub = wsManager.on('thread.reply.new', (data) => {
      const msg = data.message as MessageInfo | undefined
      if (!msg || msg.parent_message_id !== messageId) return

      queryClient.setQueryData(
        messageKeys.thread(messageId),
        (old: { parent?: MessageInfo; replies?: MessageInfo[]; has_more?: boolean } | undefined) => {
          if (!old) return old
          return {
            ...old,
            replies: [...(old.replies ?? []), msg],
            parent: old.parent
              ? { ...old.parent, reply_count: (old.parent.reply_count ?? 0) + 1 }
              : old.parent,
          }
        }
      )

      // Also update the reply_count on the parent message in the channel list
      const channelId = msg.channel_id
      if (channelId) {
        queryClient.setQueryData(
          messageKeys.list(channelId),
          (old: { pages: MessagesPage[]; pageParams: unknown[] } | undefined) => {
            if (!old) return old
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                messages: page.messages?.map((m) =>
                  m.id === messageId
                    ? { ...m, reply_count: (m.reply_count ?? 0) + 1 }
                    : m
                ),
              })),
            }
          }
        )
      }
    })

    return unsub
  }, [messageId, queryClient])
}

/**
 * Typing indicator hook.
 *
 * Subscribes to typing.start and typing.stop WebSocket events,
 * maintaining a map of currently-typing users with auto-expiry.
 */
export function useTypingIndicator(channelId: string | null) {
  const typingUsersRef = useRef<Map<string, { name: string; timeout: ReturnType<typeof setTimeout> }>>(new Map())
  const queryClient = useQueryClient()

  // Use a dummy query to trigger re-renders when typing state changes
  const { data: typingUsers = [] } = useQuery<string[]>({
    queryKey: ['typing', channelId],
    queryFn: () => [],
    enabled: !!channelId,
    staleTime: Infinity,
    gcTime: 0,
  })

  useEffect(() => {
    if (!channelId) return

    const updateTypingState = () => {
      const names = Array.from(typingUsersRef.current.values()).map((v) => v.name)
      queryClient.setQueryData(['typing', channelId], names)
    }

    const unsubStart = wsManager.on('typing.start', (data) => {
      if (data.channel_id !== channelId) return
      const userId = data.user_id as string
      const name = `${data.first_name ?? ''} ${data.last_name ?? ''}`.trim() || 'Someone'

      // Clear existing timeout for this user
      const existing = typingUsersRef.current.get(userId)
      if (existing) clearTimeout(existing.timeout)

      // Set new timeout (auto-expire after 4 seconds)
      const timeout = setTimeout(() => {
        typingUsersRef.current.delete(userId)
        updateTypingState()
      }, 4000)

      typingUsersRef.current.set(userId, { name, timeout })
      updateTypingState()
    })

    const unsubStop = wsManager.on('typing.stop', (data) => {
      if (data.channel_id !== channelId) return
      const userId = data.user_id as string
      const existing = typingUsersRef.current.get(userId)
      if (existing) {
        clearTimeout(existing.timeout)
        typingUsersRef.current.delete(userId)
        updateTypingState()
      }
    })

    // Copy ref to local variable for cleanup (avoids stale ref in unmount)
    const currentTypingUsers = typingUsersRef.current
    return () => {
      unsubStart()
      unsubStop()
      // Clean up all timeouts
      for (const entry of currentTypingUsers.values()) {
        clearTimeout(entry.timeout)
      }
      currentTypingUsers.clear()
    }
  }, [channelId, queryClient])

  return { typingUsers }
}

/**
 * Returns a debounced function that sends typing.start indicators
 * via WebSocket. Only sends once per 3 seconds of continuous typing.
 */
export function useSendTypingIndicator(channelId: string | null) {
  const lastSentRef = useRef(0)

  return useCallback(() => {
    if (!channelId) return
    const now = Date.now()
    if (now - lastSentRef.current < 3000) return
    lastSentRef.current = now
    wsManager.send({ type: 'typing.start', channel_id: channelId })
  }, [channelId])
}

export type { MessageInfo }
