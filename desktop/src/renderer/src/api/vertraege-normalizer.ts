/**
 * Normalizers for the Vertraege (contracts) API wire-shape.
 *
 * The gateway serializes via response.JSON (encoding/json over proto structs),
 * which emits google.protobuf.Timestamp as {seconds, nanos} instead of ISO
 * strings. normalizeWireTimestamps() from wire-time.ts fixes that.
 *
 * Additionally, the backend uses English contract_type values
 * (rental|service|employment|nda|other) while the UI uses German slugs
 * (mietvertrag|liefervertrag|servicevertrag|arbeitsvertrag|lizenz|versicherung).
 * This module provides bidirectional mapping + a converter from the API
 * Contract type (vertraege-types.ts) to the store Contract type
 * (stores/vertraege.ts) that the VertraegePage components consume.
 *
 * Fields that exist only in the mock store (partner, noticePeriodDays,
 * renewal, monthlyCost, totalValue, currency, documents, history,
 * reminderDays, signers, contactId/Name, dealId/Title, invoiceIds/Names)
 * are filled with safe empty defaults since the current backend does not
 * persist them. They remain editable via the dialog (which calls the backend
 * for the persisted fields and keeps the rest as local UI state via the
 * Zustand store for the session).
 */

import { normalizeWireTimestamps } from './wire-time'
import type { Contract as ApiContract } from './vertraege-types'
import type { Contract as StoreContract } from '@/stores/vertraege'

// ---------------------------------------------------------------------------
// ContractType mapping: backend (English) ↔ UI (German slugs)
// ---------------------------------------------------------------------------

type BackendContractType = 'rental' | 'service' | 'employment' | 'nda' | 'other'
type UiContractType = StoreContract['type']

const BACKEND_TO_UI_TYPE: Record<BackendContractType, UiContractType> = {
  rental: 'mietvertrag',
  service: 'servicevertrag',
  employment: 'arbeitsvertrag',
  nda: 'lizenz',    // closest semantic match; nda has no exact UI slug
  other: 'lizenz',  // fallback
}

const UI_TO_BACKEND_TYPE: Record<UiContractType, BackendContractType> = {
  mietvertrag: 'rental',
  liefervertrag: 'service',  // liefervertrag → service (closest match)
  servicevertrag: 'service',
  arbeitsvertrag: 'employment',
  lizenz: 'nda',
  versicherung: 'other',
}

export function backendTypeToUi(type: string): UiContractType {
  return BACKEND_TO_UI_TYPE[type as BackendContractType] ?? 'servicevertrag'
}

export function uiTypeToBackend(type: UiContractType): BackendContractType {
  return UI_TO_BACKEND_TYPE[type] ?? 'other'
}

// ---------------------------------------------------------------------------
// ContractStatus mapping
// Backend: draft|active|expired|terminated
// UI:      active|expiring|terminated|expired
// ---------------------------------------------------------------------------

type BackendStatus = 'draft' | 'active' | 'expired' | 'terminated'
type UiStatus = StoreContract['status']

const BACKEND_TO_UI_STATUS: Record<BackendStatus, UiStatus> = {
  draft: 'active',       // draft → treat as active in UI (no "draft" tab)
  active: 'active',
  expired: 'expired',
  terminated: 'terminated',
}

export function backendStatusToUi(status: string): UiStatus {
  return BACKEND_TO_UI_STATUS[status as BackendStatus] ?? 'active'
}

// ---------------------------------------------------------------------------
// Timestamp normalization
// ---------------------------------------------------------------------------

/**
 * Normalizes a single API contract response, converting {seconds, nanos}
 * proto timestamps to ISO strings.
 */
export function normalizeApiContract(raw: unknown): ApiContract {
  return normalizeWireTimestamps(raw) as ApiContract
}

/**
 * Normalizes a full ListContractsResponse or single ContractResponse,
 * handling the timestamp conversion recursively.
 */
export function normalizeListContractsResponse(raw: unknown): {
  contracts: ApiContract[]
  total: number
} {
  const normalized = normalizeWireTimestamps(raw) as {
    contracts?: ApiContract[]
    total?: number
  }
  return {
    contracts: normalized.contracts ?? [],
    total: normalized.total ?? 0,
  }
}

// ---------------------------------------------------------------------------
// API Contract → Store Contract adapter
// ---------------------------------------------------------------------------

/**
 * Converts a backend API Contract (vertraege-types.ts) to the store Contract
 * type (stores/vertraege.ts) that VertraegePage components consume.
 *
 * Missing backend fields receive safe defaults. Date fields are sliced to
 * YYYY-MM-DD since the UI always uses date strings (not full ISO datetimes)
 * for startDate/endDate display.
 */
export function apiContractToStore(c: ApiContract): StoreContract {
  const startDate = c.starts_on ? c.starts_on.split('T')[0] : ''
  const endDate = c.ends_on ? c.ends_on.split('T')[0] : ''

  return {
    id: c.id,
    contractNumber: c.contract_number,
    title: c.title,
    partner: '',            // not in backend — remains empty until Phase D
    type: backendTypeToUi(c.contract_type),
    status: backendStatusToUi(c.status),
    startDate,
    endDate,
    noticePeriodDays: 0,   // not in backend
    renewal: 'manual',     // not in backend — safe default
    monthlyCost: 0,        // not in backend
    totalValue: 0,         // not in backend
    documentRef: c.document_url ?? undefined,
    documents: [],         // not in backend (only document_url string)
    notes: c.notes ?? '',
    currency: 'EUR',       // not in backend — default EUR
    reminderDays: [],      // not in backend (reminders are separate entities)
    history: [],           // not in backend
    signers: [],           // not in backend
  }
}

/**
 * Converts a list of API contracts to store contracts.
 */
export function apiContractsToStore(contracts: ApiContract[]): StoreContract[] {
  return contracts.map(apiContractToStore)
}
