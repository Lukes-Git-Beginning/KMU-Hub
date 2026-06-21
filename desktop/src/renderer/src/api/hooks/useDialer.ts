/**
 * TanStack Query hooks for the Dialer module.
 *
 * Queries for campaigns, contacts, agent status, dashboards, and outcomes.
 * Mutations for campaign lifecycle, call flow, and outcome management.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listCampaigns,
  getCampaign,
  createCampaign,
  updateCampaign,
  startCampaign,
  pauseCampaign,
  archiveCampaign,
  addContactsToCampaign,
  listCampaignContacts,
  getNextContact,
  skipContact,
  requeueContact,
  initiateCall,
  logCallOutcome,
  saveCallNotes,
  completeWrapUp,
  getAgentStatus,
  setAgentStatus,
  getCampaignAgents,
  getSupervisorOverview,
  getContactCalls,
  getAgentDashboard,
  getCampaignDashboard,
  listOutcomes,
  createOutcome,
  updateOutcome,
  deleteOutcome,
} from '../dialer-client'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const dialerKeys = {
  all: ['dialer'] as const,
  campaigns: (params?: { page?: number; page_size?: number; status_filter?: number }) =>
    ['dialer', 'campaigns', params] as const,
  campaign: (id: string) => ['dialer', 'campaigns', id] as const,
  campaignContacts: (
    id: string,
    params?: { page?: number; page_size?: number; status_filter?: number },
  ) => ['dialer', 'campaigns', id, 'contacts', params] as const,
  campaignDashboard: (id: string) => ['dialer', 'campaigns', id, 'dashboard'] as const,
  campaignAgents: (id: string) => ['dialer', 'campaigns', id, 'agents'] as const,
  agentStatus: () => ['dialer', 'agent-status'] as const,
  agentDashboard: () => ['dialer', 'dashboard', 'agent'] as const,
  supervisor: () => ['dialer', 'supervisor'] as const,
  contactCalls: (ccId: string) => ['dialer', 'contacts', ccId, 'calls'] as const,
  outcomes: (includeInactive?: boolean) => ['dialer', 'outcomes', includeInactive] as const,
}

// ---------------------------------------------------------------------------
// Queries — Campaigns
// ---------------------------------------------------------------------------

export function useCampaigns(params?: {
  page?: number
  page_size?: number
  status_filter?: number
}) {
  return useQuery({
    queryKey: dialerKeys.campaigns(params),
    queryFn: () => listCampaigns(params),
  })
}

export function useCampaign(id: string) {
  return useQuery({
    queryKey: dialerKeys.campaign(id),
    queryFn: () => getCampaign(id),
    enabled: !!id,
  })
}

export function useCampaignContacts(
  campaignId: string,
  params?: { page?: number; page_size?: number; status_filter?: number },
) {
  return useQuery({
    queryKey: dialerKeys.campaignContacts(campaignId, params),
    queryFn: () => listCampaignContacts(campaignId, params),
    enabled: !!campaignId,
  })
}

export function useCampaignDashboard(campaignId: string) {
  return useQuery({
    queryKey: dialerKeys.campaignDashboard(campaignId),
    queryFn: () => getCampaignDashboard(campaignId),
    enabled: !!campaignId,
    refetchInterval: 30_000,
  })
}

export function useCampaignAgents(campaignId: string) {
  return useQuery({
    queryKey: dialerKeys.campaignAgents(campaignId),
    queryFn: () => getCampaignAgents(campaignId),
    enabled: !!campaignId,
    refetchInterval: 15_000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Agent
// ---------------------------------------------------------------------------

export function useAgentStatus() {
  return useQuery({
    queryKey: dialerKeys.agentStatus(),
    queryFn: () => getAgentStatus(),
    refetchInterval: 10_000,
  })
}

export function useAgentDashboard() {
  return useQuery({
    queryKey: dialerKeys.agentDashboard(),
    queryFn: () => getAgentDashboard(),
    refetchInterval: 30_000,
  })
}

export function useSupervisorOverview() {
  return useQuery({
    queryKey: dialerKeys.supervisor(),
    queryFn: () => getSupervisorOverview(),
    refetchInterval: 15_000,
  })
}

export function useContactCalls(campaignContactId: string | undefined) {
  return useQuery({
    queryKey: dialerKeys.contactCalls(campaignContactId ?? ''),
    queryFn: () => getContactCalls(campaignContactId!),
    enabled: !!campaignContactId,
  })
}

// ---------------------------------------------------------------------------
// Queries — Outcomes
// ---------------------------------------------------------------------------

export function useOutcomes(includeInactive?: boolean) {
  return useQuery({
    queryKey: dialerKeys.outcomes(includeInactive),
    queryFn: () => listOutcomes(includeInactive),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Campaigns
// ---------------------------------------------------------------------------

export function useCreateCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createCampaign,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'campaigns'] }),
  })
}

export function useUpdateCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: Parameters<typeof updateCampaign>[1] & { id: string }) =>
      updateCampaign(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'campaigns'] }),
  })
}

export function useStartCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: startCampaign,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'campaigns'] }),
  })
}

export function usePauseCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: pauseCampaign,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'campaigns'] }),
  })
}

export function useArchiveCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: archiveCampaign,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'campaigns'] }),
  })
}

export function useAddContacts() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      campaignId,
      ...body
    }: { campaignId: string; contact_ids?: string[]; filter_id?: string }) =>
      addContactsToCampaign(campaignId, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'campaigns'] }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Call flow (side-effectful, hence useMutation)
// ---------------------------------------------------------------------------

export function useGetNextContact() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: getNextContact,
    onSuccess: (_data, campaignId) =>
      qc.invalidateQueries({
        queryKey: dialerKeys.campaignContacts(campaignId),
      }),
  })
}

export function useSkipContact() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      campaignId,
      contactId,
    }: {
      campaignId: string
      contactId: string
    }) => skipContact(campaignId, contactId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'campaigns'] }),
  })
}

export function useRequeueContact() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      campaignId,
      contactId,
    }: {
      campaignId: string
      contactId: string
    }) => requeueContact(campaignId, contactId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'campaigns'] }),
  })
}

export function useInitiateCall() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: initiateCall,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dialerKeys.agentStatus() })
      qc.invalidateQueries({ queryKey: dialerKeys.agentDashboard() })
    },
  })
}

export function useLogOutcome() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      callId,
      ...body
    }: {
      callId: string
      outcome_id: string
      notes?: string
      next_action?: string
      callback_at?: string
      appointment_id?: string
    }) => logCallOutcome(callId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dialerKeys.agentStatus() })
      qc.invalidateQueries({ queryKey: dialerKeys.agentDashboard() })
      qc.invalidateQueries({ queryKey: ['dialer', 'campaigns'] })
    },
  })
}

export function useSaveNotes() {
  return useMutation({
    mutationFn: ({ callId, notes }: { callId: string; notes: string }) =>
      saveCallNotes(callId, notes),
  })
}

export function useCompleteWrapUp() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ callId, agentId }: { callId: string; agentId?: string }) =>
      completeWrapUp(callId, agentId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dialerKeys.agentStatus() })
      qc.invalidateQueries({ queryKey: dialerKeys.agentDashboard() })
      qc.invalidateQueries({ queryKey: ['dialer', 'campaigns'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Agent Status
// ---------------------------------------------------------------------------

export function useSetAgentStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: setAgentStatus,
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: dialerKeys.agentStatus() }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Outcomes
// ---------------------------------------------------------------------------

export function useCreateOutcome() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createOutcome,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'outcomes'] }),
  })
}

export function useUpdateOutcome() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: Parameters<typeof updateOutcome>[1] & { id: string }) =>
      updateOutcome(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'outcomes'] }),
  })
}

export function useDeleteOutcome() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: deleteOutcome,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dialer', 'outcomes'] }),
  })
}
