/**
 * TanStack Query hooks for the Helpdesk module.
 *
 * Queries for tickets, messages, queues, canned responses, and SLA policies.
 * Mutations for ticket lifecycle, assignment, merging, message creation,
 * queue/canned-response management, and SLA operations.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listTickets,
  getTicket,
  createTicket,
  updateTicket,
  closeTicket,
  reopenTicket,
  assignTicket,
  mergeTickets,
  listMessages,
  addMessage,
  listQueues,
  createQueue,
  updateQueue,
  deleteQueue,
  listCannedResponses,
  createCannedResponse,
  updateCannedResponse,
  deleteCannedResponse,
  listSLAPolicies,
  createSLAPolicy,
  updateSLAPolicy,
  deleteSLAPolicy,
  applySLAPolicy,
  getSLAStatus,
} from '../helpdesk-client'
import type {
  CreateTicketInput,
  UpdateTicketInput,
  AddMessageInput,
  CreateQueueInput,
  UpdateQueueInput,
  CreateCannedResponseInput,
  UpdateCannedResponseInput,
  CreateSLAPolicyInput,
  UpdateSLAPolicyInput,
  TicketStatus,
} from '../helpdesk-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const helpdeskKeys = {
  all: ['helpdesk'] as const,
  tickets: (params?: { status?: TicketStatus; page?: number; page_size?: number }) =>
    ['helpdesk', 'tickets', params] as const,
  ticket: (id: string) => ['helpdesk', 'tickets', id] as const,
  messages: (ticketId: string) => ['helpdesk', 'tickets', ticketId, 'messages'] as const,
  slaStatus: (ticketId: string) => ['helpdesk', 'tickets', ticketId, 'sla'] as const,
  queues: () => ['helpdesk', 'queues'] as const,
  cannedResponses: () => ['helpdesk', 'canned-responses'] as const,
  slaPolicies: () => ['helpdesk', 'sla-policies'] as const,
}

// ---------------------------------------------------------------------------
// Queries — Tickets
// ---------------------------------------------------------------------------

export function useTickets(params?: {
  status?: TicketStatus
  page?: number
  page_size?: number
}) {
  return useQuery({
    queryKey: helpdeskKeys.tickets(params),
    queryFn: () => listTickets(params),
  })
}

export function useTicket(id: string) {
  return useQuery({
    queryKey: helpdeskKeys.ticket(id),
    queryFn: () => getTicket(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — Messages
// ---------------------------------------------------------------------------

export function useTicketMessages(ticketId: string) {
  return useQuery({
    queryKey: helpdeskKeys.messages(ticketId),
    queryFn: () => listMessages(ticketId),
    enabled: !!ticketId,
  })
}

// ---------------------------------------------------------------------------
// Queries — SLA
// ---------------------------------------------------------------------------

export function useSLAStatus(ticketId: string) {
  return useQuery({
    queryKey: helpdeskKeys.slaStatus(ticketId),
    queryFn: () => getSLAStatus(ticketId),
    enabled: !!ticketId,
  })
}

// ---------------------------------------------------------------------------
// Queries — Queues, Canned Responses, SLA Policies
// ---------------------------------------------------------------------------

export function useQueues() {
  return useQuery({
    queryKey: helpdeskKeys.queues(),
    queryFn: listQueues,
  })
}

export function useCannedResponses() {
  return useQuery({
    queryKey: helpdeskKeys.cannedResponses(),
    queryFn: listCannedResponses,
  })
}

export function useSLAPolicies() {
  return useQuery({
    queryKey: helpdeskKeys.slaPolicies(),
    queryFn: listSLAPolicies,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Ticket lifecycle
// ---------------------------------------------------------------------------

export function useCreateTicket() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateTicketInput) => createTicket(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['helpdesk', 'tickets'] }),
  })
}

export function useUpdateTicket() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateTicketInput & { id: string }) =>
      updateTicket(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: helpdeskKeys.ticket(variables.id) })
      qc.invalidateQueries({ queryKey: ['helpdesk', 'tickets'] })
    },
  })
}

export function useCloseTicket() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => closeTicket(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: helpdeskKeys.ticket(id) })
      qc.invalidateQueries({ queryKey: ['helpdesk', 'tickets'] })
    },
  })
}

export function useReopenTicket() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => reopenTicket(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: helpdeskKeys.ticket(id) })
      qc.invalidateQueries({ queryKey: ['helpdesk', 'tickets'] })
    },
  })
}

export function useAssignTicket() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ ticketId, assigneeId }: { ticketId: string; assigneeId: string }) =>
      assignTicket(ticketId, assigneeId),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: helpdeskKeys.ticket(variables.ticketId) })
      qc.invalidateQueries({ queryKey: ['helpdesk', 'tickets'] })
    },
  })
}

export function useMergeTickets() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ sourceId, targetId }: { sourceId: string; targetId: string }) =>
      mergeTickets(sourceId, targetId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['helpdesk', 'tickets'] }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Messages
// ---------------------------------------------------------------------------

export function useAddMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      ticketId,
      ...input
    }: AddMessageInput & { ticketId: string }) => addMessage(ticketId, input),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: helpdeskKeys.messages(variables.ticketId) }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Queues
// ---------------------------------------------------------------------------

export function useCreateQueue() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateQueueInput) => createQueue(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: helpdeskKeys.queues() }),
  })
}

export function useUpdateQueue() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateQueueInput & { id: string }) =>
      updateQueue(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: helpdeskKeys.queues() }),
  })
}

export function useDeleteQueue() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteQueue(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: helpdeskKeys.queues() }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Canned responses
// ---------------------------------------------------------------------------

export function useCreateCannedResponse() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateCannedResponseInput) => createCannedResponse(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: helpdeskKeys.cannedResponses() }),
  })
}

export function useUpdateCannedResponse() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateCannedResponseInput & { id: string }) =>
      updateCannedResponse(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: helpdeskKeys.cannedResponses() }),
  })
}

export function useDeleteCannedResponse() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteCannedResponse(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: helpdeskKeys.cannedResponses() }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — SLA policies
// ---------------------------------------------------------------------------

export function useCreateSLAPolicy() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateSLAPolicyInput) => createSLAPolicy(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: helpdeskKeys.slaPolicies() }),
  })
}

export function useUpdateSLAPolicy() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateSLAPolicyInput & { id: string }) =>
      updateSLAPolicy(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: helpdeskKeys.slaPolicies() }),
  })
}

export function useDeleteSLAPolicy() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteSLAPolicy(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: helpdeskKeys.slaPolicies() }),
  })
}

export function useApplySLA() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ ticketId, policyId }: { ticketId: string; policyId: string }) =>
      applySLAPolicy(ticketId, policyId),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: helpdeskKeys.ticket(variables.ticketId) })
      qc.invalidateQueries({ queryKey: helpdeskKeys.slaStatus(variables.ticketId) })
    },
  })
}
