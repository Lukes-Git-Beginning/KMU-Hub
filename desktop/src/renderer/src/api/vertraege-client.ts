/**
 * Lightweight fetch wrapper for Vertraege (contracts) API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  Contract,
  ContractParty,
  ContractReminder,
  CreateContractInput,
  UpdateContractInput,
  ListContractsParams,
  AddPartyInput,
  CreateReminderInput,
  UpdateReminderInput,
  ListContractsResponse,
  ListPartiesResponse,
  ListRemindersResponse,
  UploadDocumentResponse,
} from './vertraege-types'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

async function getAuthToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new Error('Änderungen sind offline nicht möglich.')
  }

  const url = new URL(`${API_BASE_URL}${opts.path}`)

  if (opts.params) {
    for (const [key, value] of Object.entries(opts.params)) {
      if (value === undefined) continue
      url.searchParams.set(key, String(value))
    }
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = await getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const init: RequestInit = { method: opts.method, headers }

  if (opts.body !== undefined) {
    init.body = JSON.stringify(opts.body)
  }

  const response = await fetch(url.toString(), init)

  if (!response.ok) {
    if (response.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()

      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retryResponse = await fetch(url.toString(), { ...init, headers })

        if (!retryResponse.ok) {
          const errBody = await retryResponse.json().catch(() => ({}))
          throw new Error(
            (errBody as Record<string, string>).error ||
              `Request failed: ${retryResponse.status}`,
          )
        }

        if (retryResponse.status === 204) return {} as T
        return retryResponse.json() as Promise<T>
      }

      store.logout()
      throw new Error('Authentication expired')
    }

    const errBody = await response.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Request failed: ${response.status}`,
    )
  }

  if (response.status === 204) return {} as T
  return response.json() as Promise<T>
}

// ---------------------------------------------------------------------------
// Base path
// ---------------------------------------------------------------------------

const BASE = '/api/v1/vertraege'

// ---------------------------------------------------------------------------
// Contracts
// ---------------------------------------------------------------------------

export function listContracts(params?: ListContractsParams) {
  return request<ListContractsResponse>({
    method: 'GET',
    path: `${BASE}/contracts`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getContract(id: string) {
  return request<{ contract: Contract }>({
    method: 'GET',
    path: `${BASE}/contracts/${id}`,
  })
}

export function createContract(body: CreateContractInput) {
  return request<{ contract: Contract }>({
    method: 'POST',
    path: `${BASE}/contracts`,
    body,
  })
}

export function updateContract(id: string, body: UpdateContractInput) {
  return request<{ contract: Contract }>({
    method: 'PATCH',
    path: `${BASE}/contracts/${id}`,
    body,
  })
}

export function deleteContract(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/contracts/${id}` })
}

export function uploadDocument(contractId: string) {
  return request<UploadDocumentResponse>({
    method: 'POST',
    path: `${BASE}/contracts/${contractId}/document`,
  })
}

export function exportContract(contractId: string) {
  return fetch(`${API_BASE_URL}${BASE}/contracts/${contractId}/export`)
}

// ---------------------------------------------------------------------------
// Parties
// ---------------------------------------------------------------------------

export function listParties(contractId: string) {
  return request<ListPartiesResponse>({
    method: 'GET',
    path: `${BASE}/contracts/${contractId}/parties`,
  })
}

export function addParty(contractId: string, body: AddPartyInput) {
  return request<{ party: ContractParty }>({
    method: 'POST',
    path: `${BASE}/contracts/${contractId}/parties`,
    body,
  })
}

export function removeParty(contractId: string, partyId: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/contracts/${contractId}/parties/${partyId}`,
  })
}

// ---------------------------------------------------------------------------
// Reminders
// ---------------------------------------------------------------------------

export function listReminders(contractId: string, onlyPending?: boolean) {
  return request<ListRemindersResponse>({
    method: 'GET',
    path: `${BASE}/contracts/${contractId}/reminders`,
    params: onlyPending !== undefined ? { only_pending: onlyPending } : undefined,
  })
}

export function createReminder(contractId: string, body: CreateReminderInput) {
  return request<{ reminder: ContractReminder }>({
    method: 'POST',
    path: `${BASE}/contracts/${contractId}/reminders`,
    body,
  })
}

export function updateReminder(contractId: string, reminderId: string, body: UpdateReminderInput) {
  return request<{ reminder: ContractReminder }>({
    method: 'PATCH',
    path: `${BASE}/contracts/${contractId}/reminders/${reminderId}`,
    body,
  })
}

export function deleteReminder(contractId: string, reminderId: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/contracts/${contractId}/reminders/${reminderId}`,
  })
}
