/**
 * Lightweight fetch wrapper for Vertraege (contracts) API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
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
import { authenticatedRequest, authenticatedBlobRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
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
  return authenticatedBlobRequest({
    method: 'GET',
    path: `${BASE}/contracts/${contractId}/export`,
  })
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
