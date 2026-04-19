/**
 * TypeScript types for the Formulare (Forms) module.
 *
 * Mirrors the backend gateway surface for /api/v1/formulare.
 * UUIDs are represented as strings; timestamps as ISO 8601 strings.
 * JSONB fields (answers, payload) are typed as Record<string, unknown>.
 */

// ---------------------------------------------------------------------------
// Enum-like string unions
// ---------------------------------------------------------------------------

export type FormSchemaStatus = 'draft' | 'active' | 'archived'
export type FormSubmissionStatus = 'new' | 'read' | 'archived'
export type WebhookDeliveryStatus = 'pending' | 'delivered' | 'failed' | 'dead'
export type ExportFormat = 'csv' | 'xlsx'

/** Field types supported by the backend schema. Note: 'rating' is frontend-only (not in backend proto). */
export type FormFieldType =
  | 'text'
  | 'textarea'
  | 'email'
  | 'number'
  | 'select'
  | 'radio'
  | 'checkbox'
  | 'date'
  | 'file'

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export interface FormField {
  id: string
  type: FormFieldType
  label: string
  required: boolean
  placeholder?: string
  options?: string[] // für select/radio
  conditionalLogic?: unknown // TBD in späterem Sprint
  page?: number
}

export interface FormSchema {
  id: string
  tenantId: string
  title: string
  description: string
  fields: FormField[]
  status: FormSchemaStatus
  isTemplate: boolean
  isPublic: boolean
  pageCount: number
  submissionCount: number
  createdBy: string | null
  createdAt: string
  updatedAt: string
  deletedAt: string | null
}

export interface FormSubmission {
  id: string
  formSchemaId: string | null
  tenantId: string
  answers: Record<string, unknown>
  status: FormSubmissionStatus
  submittedBy: string | null
  ipAddress: string | null
  submittedAt: string
}

export interface FormWebhook {
  id: string
  formSchemaId: string
  tenantId: string
  url: string
  secret: string // masked im GET-Response (z.B. "....ab12")
  events: string[]
  active: boolean
  lastTriggeredAt: string | null
  lastStatus: string | null
  createdAt: string
  updatedAt: string
}

export interface WebhookDelivery {
  id: string
  webhookId: string
  submissionId: string
  tenantId: string
  payload: Record<string, unknown>
  status: WebhookDeliveryStatus
  attemptCount: number
  maxAttempts: number
  nextAttemptAt: string
  lastError: string | null
  lastResponseCode: number | null
  createdAt: string
  deliveredAt: string | null
}

// ---------------------------------------------------------------------------
// Request inputs
// ---------------------------------------------------------------------------

export interface CreateFormSchemaInput {
  title: string
  description?: string
  fields?: FormField[]
  status?: FormSchemaStatus
  isTemplate?: boolean
  isPublic?: boolean
  pageCount?: number
}

export interface UpdateFormSchemaInput {
  title?: string
  description?: string
  fields?: FormField[]
  status?: FormSchemaStatus
  isTemplate?: boolean
  isPublic?: boolean
  pageCount?: number
}

export interface DuplicateFormSchemaInput {
  title?: string
}

export interface CreateSubmissionInput {
  answers: Record<string, unknown>
  submittedBy?: string
}

export interface UpdateSubmissionStatusInput {
  status: FormSubmissionStatus
}

export interface CreateWebhookInput {
  url: string
  secret?: string
  events: string[]
  active?: boolean
}

export interface UpdateWebhookInput {
  url?: string
  secret?: string
  events?: string[]
  active?: boolean
}

export interface ListSchemasQuery {
  status?: FormSchemaStatus
  isTemplate?: boolean
  search?: string
  limit?: number
  offset?: number
}

export interface ListSubmissionsQuery {
  status?: FormSubmissionStatus
  limit?: number
  offset?: number
}

export interface ListDeliveriesQuery {
  webhook_id?: string
  submission_id?: string
  status?: WebhookDeliveryStatus
  limit?: number
  offset?: number
}

// ---------------------------------------------------------------------------
// Response envelopes
// ---------------------------------------------------------------------------

export interface ListFormSchemasResponse {
  items: FormSchema[]
  total: number
}

export interface ListSubmissionsResponse {
  items: FormSubmission[]
  total: number
}

/** Bundled export download with filename parsed from Content-Disposition. */
export interface ExportedSubmissions {
  blob: Blob
  filename: string
  contentType: string
}
