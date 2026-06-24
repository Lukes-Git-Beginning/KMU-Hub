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
} from './vertraege-types'
import { authenticatedRequest, authenticatedBlobRequest } from './utils/authenticatedFetch'
import { normalizeWireTimestamps } from './wire-time'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

async function request<T>(opts: RequestOptions): Promise<T> {
  const raw = await authenticatedRequest<T>(opts)
  return normalizeWireTimestamps(raw)
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

interface PresignUploadResponse {
  upload_url: string
  object_key: string
  expires_at: string
}

export async function uploadDocument(
  contractId: string,
  file: File,
): Promise<{ object_key: string }> {
  // Step 1: obtain presigned PUT URL from the files service
  const presign = await authenticatedRequest<PresignUploadResponse>({
    method: 'POST',
    path: '/api/v1/files/presign-upload',
    body: {
      scope: 'vertraege',
      file_name: file.name,
      content_type: file.type,
      size_bytes: file.size,
    },
  })

  // Step 2: PUT the file directly to storage (no Authorization header — presigned!)
  const uploadResp = await fetch(presign.upload_url, {
    method: 'PUT',
    body: file,
    headers: { 'Content-Type': file.type },
  })
  if (!uploadResp.ok) {
    throw new Error(`Storage upload failed: ${uploadResp.status} ${uploadResp.statusText}`)
  }

  // Step 3: persist the object_key on the contract
  await authenticatedRequest<{ contract: unknown }>({
    method: 'PATCH',
    path: `${BASE}/contracts/${contractId}`,
    body: { document_url: presign.object_key },
  })

  return { object_key: presign.object_key }
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
