/**
 * React Query hooks for chat channel operations.
 *
 * Provides typed wrappers around the channel REST endpoints with
 * automatic cache invalidation on mutations.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import type { components } from '@/api/types'

type ChannelInfo = components['schemas']['ChannelInfo']
type CreateChannelRequest = components['schemas']['CreateChannelRequest']

/** Query key factory for channel-related queries. */
export const channelKeys = {
  all: ['channels'] as const,
  lists: () => [...channelKeys.all, 'list'] as const,
  list: (params?: { includeArchived?: boolean }) =>
    [...channelKeys.lists(), params] as const,
  details: () => [...channelKeys.all, 'detail'] as const,
  detail: (id: string) => [...channelKeys.details(), id] as const,
  members: (id: string) => [...channelKeys.all, 'members', id] as const,
  dms: () => [...channelKeys.all, 'dms'] as const,
  unread: () => [...channelKeys.all, 'unread'] as const,
}

/** Fetch all channels the current user is a member of. */
export function useChannels(params?: { includeArchived?: boolean }) {
  return useQuery({
    queryKey: channelKeys.list(params),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/channels', {
        params: {
          query: {
            include_archived: params?.includeArchived,
          },
        },
      })
      if (error) throw new Error('Failed to load channels')
      return data
    },
  })
}

/** Fetch a single channel by ID. */
export function useChannel(id: string) {
  return useQuery({
    queryKey: channelKeys.detail(id),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/channels/{id}', {
        params: { path: { id } },
      })
      if (error) throw new Error('Failed to load channel')
      return data
    },
    enabled: !!id,
  })
}

/** Fetch channel members. */
export function useChannelMembers(channelId: string) {
  return useQuery({
    queryKey: channelKeys.members(channelId),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/channels/{id}/members', {
        params: { path: { id: channelId } },
      })
      if (error) throw new Error('Failed to load members')
      return data
    },
    enabled: !!channelId,
  })
}

/** Create a new channel. */
export function useCreateChannel() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (request: CreateChannelRequest) => {
      const { data, error } = await apiClient.POST('/api/v1/channels', {
        body: request,
      })
      if (error) throw new Error('Failed to create channel')
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.lists() })
    },
  })
}

/** Join a public channel. */
export function useJoinChannel() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (channelId: string) => {
      const { data, error } = await apiClient.POST('/api/v1/channels/{id}/join', {
        params: { path: { id: channelId } },
      })
      if (error) throw new Error('Failed to join channel')
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.lists() })
    },
  })
}

/** Leave a channel. */
export function useLeaveChannel() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (channelId: string) => {
      const { error } = await apiClient.POST('/api/v1/channels/{id}/leave', {
        params: { path: { id: channelId } },
      })
      if (error) throw new Error('Failed to leave channel')
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.lists() })
    },
  })
}

/** List DM conversations for the current user. */
export function useDMs() {
  return useQuery({
    queryKey: channelKeys.dms(),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/channels/dm')
      if (error) throw new Error('Failed to load DMs')
      return data
    },
  })
}

/** Get or create a DM channel with another user. */
export function useGetOrCreateDM() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (otherUserId: string) => {
      const { data, error } = await apiClient.POST('/api/v1/channels/dm', {
        body: { other_user_id: otherUserId },
      })
      if (error) throw new Error('Failed to create DM')
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.dms() })
    },
  })
}

/**
 * Start a conversation with one or more users. A single id creates/opens a 1:1
 * DM; multiple ids create a group DM (KO-2). The backend DM request type only
 * declares `other_user_id`, so the group payload is sent via a narrow cast —
 * the demo MSW handler understands `participant_ids`.
 */
export function useStartConversation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (userIds: string[]) => {
      const body =
        userIds.length > 1
          ? ({ participant_ids: userIds } as unknown as { other_user_id: string })
          : { other_user_id: userIds[0] ?? '' }
      const { data, error } = await apiClient.POST('/api/v1/channels/dm', { body })
      if (error) throw new Error('Failed to start conversation')
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.dms() })
    },
  })
}

/** Fetch unread message counts for all channels. */
export function useUnreadCounts() {
  return useQuery({
    queryKey: channelKeys.unread(),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/channels/unread')
      if (error) throw new Error('Failed to load unread counts')
      return data
    },
  })
}

/** Mark a channel as read. */
export function useMarkChannelRead() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ channelId, messageId }: { channelId: string; messageId: string }) => {
      const { error } = await apiClient.POST('/api/v1/channels/{id}/read', {
        params: { path: { id: channelId } },
        body: { message_id: messageId },
      })
      if (error) throw new Error('Failed to mark channel as read')
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.unread() })
    },
  })
}

export type { ChannelInfo, CreateChannelRequest }
