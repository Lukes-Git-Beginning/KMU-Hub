/**
 * TanStack Query hooks for the Unified Inbox (Kommunikation) module.
 *
 * Query keys follow the pattern ['inbox', domain, ...params] for consistent
 * cache invalidation. Mutations use optimistic updates for triage actions
 * (mark read, star, archive) and invalidation for team/rule management.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import * as inboxClient from '../inbox-client'
import type {
  InboxListFilter,
  InboxMessageList,
  InboxMessage,
  TeamInbox,
  RoutingRule,
  Condition,
  TeamMemberRole,
} from '../inbox-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const inboxKeys = {
  all: ['inbox'] as const,
  messages: (filter: InboxListFilter) => ['inbox', 'messages', filter] as const,
  message: (id: string) => ['inbox', 'message', id] as const,
  unread: () => ['inbox', 'unread'] as const,
  teams: () => ['inbox', 'teams'] as const,
  teamMembers: (teamId: string) =>
    ['inbox', 'teams', teamId, 'members'] as const,
  rules: () => ['inbox', 'rules'] as const,
}

// ---------------------------------------------------------------------------
// Message queries
// ---------------------------------------------------------------------------

export function useInboxMessages(filter: InboxListFilter) {
  return useQuery({
    queryKey: inboxKeys.messages(filter),
    queryFn: () => inboxClient.listMessages(filter),
    staleTime: 30 * 1000, // 30s for near-real-time
  })
}

export function useInboxMessage(id: string) {
  return useQuery({
    queryKey: inboxKeys.message(id),
    queryFn: () => inboxClient.getMessage(id),
    enabled: !!id,
  })
}

export function useUnreadCount() {
  return useQuery({
    queryKey: inboxKeys.unread(),
    queryFn: () => inboxClient.getUnreadCount(),
    staleTime: 15 * 1000, // 15s for sidebar badge
  })
}

// ---------------------------------------------------------------------------
// Team queries
// ---------------------------------------------------------------------------

export function useTeamInboxes() {
  return useQuery({
    queryKey: inboxKeys.teams(),
    queryFn: () => inboxClient.listTeamInboxes(),
    staleTime: 60 * 1000,
  })
}

export function useTeamMembers(teamId: string) {
  return useQuery({
    queryKey: inboxKeys.teamMembers(teamId),
    queryFn: () => inboxClient.listTeamMembers(teamId),
    enabled: !!teamId,
  })
}

// ---------------------------------------------------------------------------
// Routing rules query
// ---------------------------------------------------------------------------

export function useRoutingRules() {
  return useQuery({
    queryKey: inboxKeys.rules(),
    queryFn: () => inboxClient.listRoutingRules(),
    staleTime: 60 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Message mutations with optimistic updates
// ---------------------------------------------------------------------------

export function useMarkRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => inboxClient.markRead(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: ['inbox', 'messages'] })
      // Optimistic: set is_read=true in all message list caches
      qc.setQueriesData<InboxMessageList>(
        { queryKey: ['inbox', 'messages'] },
        (old) => {
          if (!old) return old
          return {
            ...old,
            messages: old.messages.map((m) =>
              m.id === id ? { ...m, is_read: true } : m,
            ),
          }
        },
      )
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: inboxKeys.unread() })
    },
    onError: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      toast.error('Fehler beim Markieren als gelesen')
    },
  })
}

export function useMarkUnread() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => inboxClient.markUnread(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: ['inbox', 'messages'] })
      qc.setQueriesData<InboxMessageList>(
        { queryKey: ['inbox', 'messages'] },
        (old) => {
          if (!old) return old
          return {
            ...old,
            messages: old.messages.map((m) =>
              m.id === id ? { ...m, is_read: false } : m,
            ),
          }
        },
      )
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: inboxKeys.unread() })
    },
    onError: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      toast.error('Fehler beim Markieren als ungelesen')
    },
  })
}

export function useToggleStar() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => inboxClient.toggleStar(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: ['inbox', 'messages'] })
      qc.setQueriesData<InboxMessageList>(
        { queryKey: ['inbox', 'messages'] },
        (old) => {
          if (!old) return old
          return {
            ...old,
            messages: old.messages.map((m) =>
              m.id === id ? { ...m, is_starred: !m.is_starred } : m,
            ),
          }
        },
      )
    },
    onError: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      toast.error('Fehler beim Aendern des Sterns')
    },
  })
}

export function useArchiveMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => inboxClient.archiveMessage(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: ['inbox', 'messages'] })
      qc.setQueriesData<InboxMessageList>(
        { queryKey: ['inbox', 'messages'] },
        (old) => {
          if (!old) return old
          return {
            ...old,
            messages: old.messages.filter((m) => m.id !== id),
            total_count: old.total_count - 1,
          }
        },
      )
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      qc.invalidateQueries({ queryKey: inboxKeys.unread() })
      toast.success('Archiviert')
    },
    onError: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      toast.error('Fehler beim Archivieren')
    },
  })
}

export function useSnoozeMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, snoozeUntil }: { id: string; snoozeUntil: string }) =>
      inboxClient.snoozeMessage(id, snoozeUntil),
    onMutate: async ({ id }) => {
      await qc.cancelQueries({ queryKey: ['inbox', 'messages'] })
      // Remove from list (snoozed items hidden from active view)
      qc.setQueriesData<InboxMessageList>(
        { queryKey: ['inbox', 'messages'] },
        (old) => {
          if (!old) return old
          return {
            ...old,
            messages: old.messages.filter((m) => m.id !== id),
            total_count: old.total_count - 1,
          }
        },
      )
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      qc.invalidateQueries({ queryKey: inboxKeys.unread() })
    },
    onError: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      toast.error('Fehler beim Verschieben')
    },
  })
}

export function useReplyToMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: string }) =>
      inboxClient.replyToMessage(id, body),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: inboxKeys.message(vars.id) })
      toast.success('Antwort gesendet')
    },
    onError: () => {
      toast.error('Fehler beim Senden der Antwort')
    },
  })
}

export function useAssignMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      assigneeId,
    }: {
      id: string
      assigneeId: string
    }) => inboxClient.assignMessage(id, assigneeId),
    onMutate: async ({ id, assigneeId }) => {
      await qc.cancelQueries({ queryKey: ['inbox', 'messages'] })
      qc.setQueriesData<InboxMessageList>(
        { queryKey: ['inbox', 'messages'] },
        (old) => {
          if (!old) return old
          return {
            ...old,
            messages: old.messages.map((m) =>
              m.id === id ? { ...m, assigned_to: assigneeId } : m,
            ),
          }
        },
      )
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      toast.success('Zugewiesen')
    },
    onError: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      toast.error('Fehler beim Zuweisen')
    },
  })
}

export function useClaimMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => inboxClient.claimMessage(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      toast.success('Uebernommen')
    },
    onError: () => {
      toast.error('Fehler beim Uebernehmen')
    },
  })
}

export function useBulkMarkRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (ids: string[]) => inboxClient.bulkMarkRead(ids),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      qc.invalidateQueries({ queryKey: inboxKeys.unread() })
      toast.success('Als gelesen markiert')
    },
    onError: () => {
      toast.error('Fehler beim Markieren')
    },
  })
}

export function useBulkArchive() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (ids: string[]) => inboxClient.bulkArchive(ids),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'messages'] })
      qc.invalidateQueries({ queryKey: inboxKeys.unread() })
      toast.success('Archiviert')
    },
    onError: () => {
      toast.error('Fehler beim Archivieren')
    },
  })
}

// ---------------------------------------------------------------------------
// Team inbox mutations
// ---------------------------------------------------------------------------

export function useCreateTeamInbox() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<TeamInbox>) =>
      inboxClient.createTeamInbox(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: inboxKeys.teams() })
      toast.success('Team-Postfach erstellt')
    },
    onError: () => {
      toast.error('Fehler beim Erstellen des Team-Postfachs')
    },
  })
}

export function useUpdateTeamInbox() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<TeamInbox> & { id: string }) =>
      inboxClient.updateTeamInbox(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: inboxKeys.teams() })
      toast.success('Team-Postfach aktualisiert')
    },
    onError: () => {
      toast.error('Fehler beim Aktualisieren')
    },
  })
}

export function useDeleteTeamInbox() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => inboxClient.deleteTeamInbox(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: inboxKeys.teams() })
      toast.success('Team-Postfach geloescht')
    },
    onError: () => {
      toast.error('Fehler beim Loeschen')
    },
  })
}

export function useAddTeamMember() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      teamId,
      userId,
      role,
    }: {
      teamId: string
      userId: string
      role: TeamMemberRole
    }) => inboxClient.addTeamMember(teamId, userId, role),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({
        queryKey: inboxKeys.teamMembers(vars.teamId),
      })
      toast.success('Mitglied hinzugefuegt')
    },
    onError: () => {
      toast.error('Fehler beim Hinzufuegen')
    },
  })
}

export function useRemoveTeamMember() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ teamId, userId }: { teamId: string; userId: string }) =>
      inboxClient.removeTeamMember(teamId, userId),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({
        queryKey: inboxKeys.teamMembers(vars.teamId),
      })
      toast.success('Mitglied entfernt')
    },
    onError: () => {
      toast.error('Fehler beim Entfernen')
    },
  })
}

// ---------------------------------------------------------------------------
// Routing rule mutations
// ---------------------------------------------------------------------------

export function useCreateRoutingRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<RoutingRule>) =>
      inboxClient.createRoutingRule(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: inboxKeys.rules() })
      toast.success('Regel erstellt')
    },
    onError: () => {
      toast.error('Fehler beim Erstellen der Regel')
    },
  })
}

export function useUpdateRoutingRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<RoutingRule> & { id: string }) =>
      inboxClient.updateRoutingRule(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: inboxKeys.rules() })
      toast.success('Regel aktualisiert')
    },
    onError: () => {
      toast.error('Fehler beim Aktualisieren der Regel')
    },
  })
}

export function useDeleteRoutingRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => inboxClient.deleteRoutingRule(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: inboxKeys.rules() })
      toast.success('Regel geloescht')
    },
    onError: () => {
      toast.error('Fehler beim Loeschen der Regel')
    },
  })
}

export function useTestRoutingRule() {
  return useMutation({
    mutationFn: ({
      conditions,
      message,
    }: {
      conditions: Condition
      message: Partial<InboxMessage>
    }) => inboxClient.testRoutingRule(conditions, message),
  })
}
