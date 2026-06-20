/**
 * Email module TypeScript types mirroring the proto/email/v1 messages.
 *
 * These types are manually maintained to match the backend gRPC service.
 * Field names use snake_case to match protobuf JSON serialization.
 */

// ---------------------------------------------------------------------------
// Common
// ---------------------------------------------------------------------------

export interface EmailAddress {
  name: string
  email: string
}

export interface EmailAttachmentInfo {
  id: string
  filename: string
  content_type: string
  size_bytes: number
  minio_key: string
  content_id: string
  is_inline: boolean
}

// ---------------------------------------------------------------------------
// Accounts
// ---------------------------------------------------------------------------

export interface EmailAccountInfo {
  id: string
  user_id: string
  email_address: string
  display_name: string
  imap_host: string
  imap_port: number
  smtp_host: string
  smtp_port: number
  username: string
  use_ssl: boolean
  sync_enabled: boolean
  last_sync_at: string
  created_at: string
  updated_at: string
}

export interface CreateEmailAccountRequest {
  user_id: string
  email_address: string
  display_name: string
  imap_host: string
  imap_port: number
  smtp_host: string
  smtp_port: number
  username: string
  password: string
  use_ssl: boolean
  sync_enabled: boolean
}

export interface UpdateEmailAccountRequest {
  id: string
  display_name?: string
  imap_host?: string
  imap_port?: number
  smtp_host?: string
  smtp_port?: number
  username?: string
  password?: string
  use_ssl?: boolean
  sync_enabled?: boolean
}

export interface TestEmailConnectionRequest {
  imap_host: string
  imap_port: number
  smtp_host: string
  smtp_port: number
  username: string
  password: string
  use_ssl: boolean
}

export interface TestEmailConnectionResponse {
  imap_ok: boolean
  smtp_ok: boolean
  imap_error: string
  smtp_error: string
}

// ---------------------------------------------------------------------------
// Folders
// ---------------------------------------------------------------------------

export type FolderType = 'inbox' | 'sent' | 'drafts' | 'trash' | 'spam' | 'archive' | 'custom'

export interface EmailFolderInfo {
  id: string
  account_id: string
  name: string
  imap_name: string
  folder_type: FolderType
  uid_validity: number
  message_count: number
  unread_count: number
  sort_order: number
  created_at: string
  updated_at: string
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

export interface EmailMessageInfo {
  id: string
  account_id: string
  folder_id: string
  uid: number
  message_id_header: string
  in_reply_to: string
  references: string[]
  thread_id: string
  from: EmailAddress
  to: EmailAddress[]
  cc: EmailAddress[]
  bcc: EmailAddress[]
  subject: string
  preview: string
  body_text: string
  body_html: string
  is_read: boolean
  is_starred: boolean
  is_draft: boolean
  has_attachments: boolean
  date: string
  size_bytes: number
  attachments: EmailAttachmentInfo[]
  created_at: string
  updated_at: string
}

export interface ListMessagesParams {
  folder_id: string
  page?: number
  per_page?: number
  search?: string
  sort_by?: string
  sort_desc?: boolean
}

export interface ListMessagesResponse {
  messages: EmailMessageInfo[]
  total: number
}

// ---------------------------------------------------------------------------
// Send / Compose
// ---------------------------------------------------------------------------

export interface SendEmailRequest {
  account_id: string
  to: EmailAddress[]
  cc?: EmailAddress[]
  bcc?: EmailAddress[]
  subject: string
  body_html: string
  body_text: string
  attachment_ids?: string[]
  in_reply_to_message_id?: string
  signature_id?: string
  // contact_id, when set, makes the backend enforce marketing consent (UWG §7)
  // for the recipient. Set only for outreach to a CRM contact, not for
  // transactional or ad-hoc mail.
  contact_id?: string
}

export interface SaveDraftRequest {
  account_id: string
  draft_id?: string
  to: EmailAddress[]
  cc?: EmailAddress[]
  bcc?: EmailAddress[]
  subject: string
  body_html: string
  body_text: string
  attachment_ids?: string[]
  in_reply_to_message_id?: string
}

export interface ReplyEmailRequest {
  account_id: string
  original_message_id: string
  body_html: string
  body_text: string
  reply_all: boolean
  attachment_ids?: string[]
}

export interface ForwardEmailRequest {
  account_id: string
  original_message_id: string
  to: EmailAddress[]
  body_html: string
  body_text: string
  attachment_ids?: string[]
}

// ---------------------------------------------------------------------------
// Signatures
// ---------------------------------------------------------------------------

export interface EmailSignatureInfo {
  id: string
  user_id: string
  name: string
  html_content: string
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface CreateSignatureRequest {
  user_id: string
  name: string
  html_content: string
  is_default: boolean
}

export interface UpdateSignatureRequest {
  id: string
  name?: string
  html_content?: string
  is_default?: boolean
}

// ---------------------------------------------------------------------------
// CRM Links
// ---------------------------------------------------------------------------

export interface EmailContactLinkInfo {
  id: string
  message_id: string
  contact_id: string
  link_type: 'auto' | 'manual'
  created_at: string
}

export interface GetContactEmailsResponse {
  messages: EmailMessageInfo[]
  total: number
}

// ---------------------------------------------------------------------------
// Sync
// ---------------------------------------------------------------------------

export type SyncStatus = 'idle' | 'syncing' | 'error'

export interface SyncStatusResponse {
  status: SyncStatus
  last_sync_at: string
  error_message: string
}

export interface TriggerSyncResponse {
  status: 'started' | 'already_syncing'
}

// ---------------------------------------------------------------------------
// Attachments
// ---------------------------------------------------------------------------

export interface UploadAttachmentResponse {
  id: string
  minio_key: string
  size_bytes: number
}

export interface AttachmentDownloadURLResponse {
  download_url: string
  filename: string
  content_type: string
  size_bytes: number
}
