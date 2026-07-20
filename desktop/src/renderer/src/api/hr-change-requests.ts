/**
 * HR Self-Service Change Request API — types, client, react-query hooks.
 *
 * R-4 P3: employees with `team:self:propose` submit profile change requests
 * instead of editing directly. HR users with `data_personal:edit` approve or
 * reject from the ChangeRequestInbox.
 *
 * Endpoint prefix: /api/v1/hr/change-requests
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import i18next from 'i18next'
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type ChangeRequestStatus = 'pending' | 'approved' | 'rejected' | 'cancelled'
export type ChangeRequestDrawer = 'personal' | 'job'

export interface ProfileChangeRequest {
  id: string
  userId: string
  userName: string
  drawer: ChangeRequestDrawer
  field: string
  fieldLabel: string
  oldValue: string
  newValue: string
  status: ChangeRequestStatus
  reason?: string
  createdAt: string
  decidedAt?: string
  decidedByName?: string
}

export interface CreateChangeRequestInput {
  drawer: ChangeRequestDrawer
  field: string
  fieldLabel: string
  oldValue: string
  newValue: string
}

export interface RejectChangeRequestInput {
  reason: string
}

// ---------------------------------------------------------------------------
// API client helpers
// ---------------------------------------------------------------------------

const BASE = '/api/v1/hr/change-requests'

async function request<T>(path: string, options: { method?: string; body?: unknown } = {}): Promise<T> {
  return authenticatedRequest<T>({
    method: options.method ?? 'GET',
    path,
    body: options.body,
  })
}

const changeRequestsApi = {
  list: (status?: ChangeRequestStatus): Promise<{ requests: ProfileChangeRequest[] }> =>
    request(status ? `${BASE}?status=${status}` : BASE),

  listMine: (): Promise<{ requests: ProfileChangeRequest[] }> =>
    request(`${BASE}?scope=mine`),

  create: (data: CreateChangeRequestInput): Promise<{ request: ProfileChangeRequest }> =>
    request(BASE, { method: 'POST', body: data }),

  approve: (id: string): Promise<{ request: ProfileChangeRequest }> =>
    request(`${BASE}/${id}/approve`, { method: 'POST' }),

  reject: (id: string, data: RejectChangeRequestInput): Promise<{ request: ProfileChangeRequest }> =>
    request(`${BASE}/${id}/reject`, { method: 'POST', body: data }),

  cancel: (id: string): Promise<{ request: ProfileChangeRequest }> =>
    request(`${BASE}/${id}/cancel`, { method: 'POST' }),
}

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const changeRequestKeys = {
  all: ['hr', 'change-requests'] as const,
  list: (status?: ChangeRequestStatus) =>
    ['hr', 'change-requests', 'list', status ?? 'all'] as const,
  mine: () => ['hr', 'change-requests', 'mine'] as const,
}

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------

/** All pending change requests — HR inbox view. */
export function useChangeRequests(status?: ChangeRequestStatus) {
  return useQuery({
    queryKey: changeRequestKeys.list(status),
    queryFn: () => changeRequestsApi.list(status),
    select: (data) => data.requests,
  })
}

/** Only the current user's own change requests. */
export function useMyChangeRequests() {
  return useQuery({
    queryKey: changeRequestKeys.mine(),
    queryFn: () => changeRequestsApi.listMine(),
    select: (data) => data.requests,
  })
}

/** Submit a new change request for the current user's own profile. */
export function useCreateChangeRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateChangeRequestInput) => changeRequestsApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: changeRequestKeys.mine() })
      qc.invalidateQueries({ queryKey: changeRequestKeys.list() })
      toast.success(i18next.t('team.changeRequest.submitted'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('common.error'))
    },
  })
}

/** HR: approve a pending request. Also invalidates employee data since mock mutates employee. */
export function useApproveChangeRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => changeRequestsApi.approve(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: changeRequestKeys.all })
      qc.invalidateQueries({ queryKey: ['hr', 'employees'] })
      toast.success(i18next.t('team.changeRequest.approved'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('common.error'))
    },
  })
}

/** HR: reject a pending request with a mandatory reason. */
export function useRejectChangeRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: RejectChangeRequestInput }) =>
      changeRequestsApi.reject(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: changeRequestKeys.all })
      toast.success(i18next.t('team.changeRequest.rejected'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('common.error'))
    },
  })
}

/** Employee: cancel own pending request. */
export function useCancelChangeRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => changeRequestsApi.cancel(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: changeRequestKeys.mine() })
      qc.invalidateQueries({ queryKey: changeRequestKeys.list() })
      toast.success(i18next.t('team.changeRequest.cancelled'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('common.error'))
    },
  })
}
