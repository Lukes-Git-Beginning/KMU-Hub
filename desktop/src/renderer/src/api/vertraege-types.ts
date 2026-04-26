/**
 * TypeScript types for the Vertraege (contracts) module.
 *
 * Mirrors backend/internal/vertraege/models.go and service.go input types.
 * UUIDs are represented as strings; dates as ISO 8601 strings.
 */

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

export type ContractType = 'rental' | 'service' | 'employment' | 'nda' | 'other'
export type ContractStatus = 'draft' | 'active' | 'expired' | 'terminated'
export type PartyType = 'contact' | 'company' | 'external'
export type ReminderType = 'renewal' | 'expiry' | 'payment' | 'custom'
export type ReminderStatus = 'pending' | 'sent' | 'cancelled'

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export interface Contract {
  id: string
  tenant_id: string
  contract_number: string
  title: string
  contract_type: ContractType
  status: ContractStatus
  starts_on: string   // ISO date (YYYY-MM-DD)
  ends_on: string | null
  document_url: string | null
  notes: string
  created_by: string | null
  /** Phase D: Skribble placeholder — currently null in all environments. */
  signature_provider: string | null
  created_at: string
  updated_at: string
  /** Populated only by GetContract. */
  parties?: ContractParty[]
  /** Populated only by GetContract. */
  reminders?: ContractReminder[]
}

export interface ContractParty {
  id: string
  tenant_id: string
  contract_id: string
  party_type: PartyType
  contact_id: string | null
  company_id: string | null
  external_name: string | null
  role_in_contract: string
  signed_on: string | null
  created_at: string
}

export interface ContractReminder {
  id: string
  tenant_id: string
  contract_id: string
  remind_at: string
  reminder_type: ReminderType
  subject: string
  message: string
  status: ReminderStatus
  created_at: string
  sent_at: string | null
}

// ---------------------------------------------------------------------------
// Request input types
// ---------------------------------------------------------------------------

export interface CreateContractInput {
  contract_number: string
  title: string
  contract_type: ContractType
  starts_on: string
  ends_on?: string
  document_url?: string
  notes?: string
}

export interface UpdateContractInput {
  title?: string
  contract_type?: ContractType
  status?: ContractStatus
  starts_on?: string
  ends_on?: string
  clear_ends_on?: boolean
  document_url?: string
  notes?: string
}

export interface ListContractsParams {
  status?: ContractStatus
  contract_type?: ContractType
  starts_after?: string
  starts_before?: string
  ends_after?: string
  ends_before?: string
  page?: number
  page_size?: number
}

export interface AddPartyInput {
  party_type: PartyType
  contact_id?: string
  company_id?: string
  external_name?: string
  role_in_contract: string
  signed_on?: string
}

export interface CreateReminderInput {
  remind_at: string
  reminder_type: ReminderType
  subject: string
  message?: string
}

export interface UpdateReminderInput {
  remind_at?: string
  reminder_type?: ReminderType
  subject?: string
  message?: string
  status?: ReminderStatus
}

// ---------------------------------------------------------------------------
// Response wrapper types
// ---------------------------------------------------------------------------

export interface ListContractsResponse {
  contracts: Contract[]
  total: number
}

export interface ListPartiesResponse {
  parties: ContractParty[]
}

export interface ListRemindersResponse {
  reminders: ContractReminder[]
}

export interface UploadDocumentResponse {
  upload_url: string
}
