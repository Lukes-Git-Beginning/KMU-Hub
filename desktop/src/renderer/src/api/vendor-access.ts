/**
 * Vendor Access API — Client + React-Query-Hooks (RBAC R-5 B).
 *
 * Endpoint-Präfix: /api/v1/vendor-access
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import i18next from 'i18next'
import { authenticatedRequest } from './utils/authenticatedFetch'
import type {
  VendorAccessListResponse,
  VendorAccessRequest,
  ApproveVendorAccessInput,
  CounterProposeVendorAccessInput,
} from './vendor-access-types'

// ---------------------------------------------------------------------------
// API-Client
// ---------------------------------------------------------------------------

const BASE = '/api/v1/vendor-access'

async function request<T>(path: string, options: { method?: string; body?: unknown } = {}): Promise<T> {
  return authenticatedRequest<T>({
    method: options.method ?? 'GET',
    path,
    body: options.body,
  })
}

const vendorAccessApi = {
  list: (): Promise<VendorAccessListResponse> => request(BASE),

  approve: (id: string, data?: ApproveVendorAccessInput): Promise<{ request: VendorAccessRequest }> =>
    request(`${BASE}/${id}/approve`, { method: 'POST', body: data ?? {} }),

  decline: (id: string): Promise<{ request: VendorAccessRequest }> =>
    request(`${BASE}/${id}/decline`, { method: 'POST' }),

  counterPropose: (
    id: string,
    data: CounterProposeVendorAccessInput,
  ): Promise<{ request: VendorAccessRequest }> =>
    request(`${BASE}/${id}/counter-propose`, { method: 'POST', body: data }),

  revoke: (id: string): Promise<{ request: VendorAccessRequest }> =>
    request(`${BASE}/${id}/revoke`, { method: 'POST' }),
}

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const vendorAccessKeys = {
  all: ['vendor-access'] as const,
  list: () => ['vendor-access', 'list'] as const,
}

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------

/** Alle Vendor-Access-Anfragen. */
export function useVendorAccessList() {
  return useQuery({
    queryKey: vendorAccessKeys.list(),
    queryFn: () => vendorAccessApi.list(),
    select: (data) => data.requests,
  })
}

/** Genehmigen (mit optionalem sensitive_ack). */
export function useApproveVendorAccess() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data?: ApproveVendorAccessInput }) =>
      vendorAccessApi.approve(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: vendorAccessKeys.all })
      toast.success(i18next.t('rbac.vendorAccess.toast.approved'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('common.error'))
    },
  })
}

/** Ablehnen. */
export function useDeclineVendorAccess() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => vendorAccessApi.decline(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: vendorAccessKeys.all })
      toast.success(i18next.t('rbac.vendorAccess.toast.declined'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('common.error'))
    },
  })
}

/** Anderen Starttermin vorschlagen. */
export function useCounterProposeVendorAccess() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: CounterProposeVendorAccessInput }) =>
      vendorAccessApi.counterPropose(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: vendorAccessKeys.all })
      toast.success(i18next.t('rbac.vendorAccess.toast.counterProposed'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('common.error'))
    },
  })
}

/** Zugang entziehen. */
export function useRevokeVendorAccess() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => vendorAccessApi.revoke(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: vendorAccessKeys.all })
      toast.success(i18next.t('rbac.vendorAccess.toast.revoked'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('common.error'))
    },
  })
}
