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

/**
 * Lifecycle status. `closed` (Darien decision 2026-06-20) sits between `active`
 * and `archived`: the form stops accepting new submissions but stays
 * re-activatable and its evaluation remains visible. Only `active` forms are
 * shareable; `draft` must be published first.
 */
export type FormSchemaStatus = 'draft' | 'active' | 'closed' | 'archived'
export type FormSubmissionStatus = 'new' | 'read' | 'archived'
export type WebhookDeliveryStatus = 'pending' | 'delivered' | 'failed' | 'dead'
export type ExportFormat = 'csv' | 'xlsx'

/** Distribution channel a share link was created for (FD-1). */
export type ShareChannel = 'link' | 'email' | 'embed' | 'qr'
/** Lifecycle of a single share link. `expired` is derived (date / answer limit). */
export type ShareLinkStatus = 'active' | 'expired' | 'disabled'

/**
 * Field types supported by the backend schema. Note: 'rating' and 'consent' are
 * frontend/demo pseudo-types (not in the backend proto whitelist) — 'consent'
 * renders a purpose-bound DSGVO checkbox with a linked privacy policy.
 */
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
  | 'consent'
  | 'rating'

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

/** FT-2a — preset pattern for text-field validation (regex derived centrally). */
export type FieldPatternType = 'free' | 'plz' | 'phone' | 'iban'

/** FT-2a — optional per-field validation rules (length/value/pattern). */
export interface FieldValidation {
  minLength?: number
  maxLength?: number
  min?: number
  max?: number
  patternType?: FieldPatternType
}

export interface FormField {
  id: string
  type: FormFieldType
  label: string
  required: boolean
  placeholder?: string
  options?: string[] // für select/radio
  consentText?: string // consent only: purpose-binding / privacy text
  privacyUrl?: string // consent only: link to the Datenschutzerklärung
  conditionalLogic?: unknown // TBD in späterem Sprint
  validation?: FieldValidation // FT-2a — length / value / pattern rules
  ratingScale?: number // rating only (FT-2b): 5 = stars, 10 = NPS scale
  pageTitle?: string // page-break only (FT-2b): title of the following page
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
  /** Message shown on the public fill-out page once the form is `closed`. */
  closedMessage?: string
  /** FT-2b — custom thank-you message shown after a successful submit. */
  thankYouMessage?: string
  /** FT-2b — redirect URL applied after submit on the public page. */
  redirectUrl?: string
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

/**
 * A shareable distribution link for a form (FD-1 / FD-2). MSW-stateful in demo;
 * the real token + public route are Luke's lane (FD-4). `views`/`submissions`
 * drive the conversion overview; `status` is derived on read from expiry,
 * answer limit and a manual disable flag.
 */
export interface FormShareLink {
  id: string
  formSchemaId: string
  tenantId: string
  token: string
  /** Public fill-out URL — placeholder host until FD-4 wires the real route. */
  url: string
  channel: ShareChannel
  expiresAt: string | null
  maxSubmissions: number | null
  views: number
  submissions: number
  status: ShareLinkStatus
  createdBy: string | null
  createdAt: string
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
  closedMessage?: string
  thankYouMessage?: string
  redirectUrl?: string
}

export interface UpdateFormSchemaInput {
  title?: string
  description?: string
  fields?: FormField[]
  status?: FormSchemaStatus
  isTemplate?: boolean
  isPublic?: boolean
  pageCount?: number
  closedMessage?: string
  thankYouMessage?: string
  redirectUrl?: string
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

export interface CreateShareLinkInput {
  channel: ShareChannel
  expiresAt?: string | null
  maxSubmissions?: number | null
}

export interface UpdateShareLinkInput {
  status?: ShareLinkStatus
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
