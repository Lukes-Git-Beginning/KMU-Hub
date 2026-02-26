export interface GuestSession {
  id: string
  channel_id: string
  display_name: string
  email?: string
  is_active: boolean
  created_at: string
  expires_at: string
}

export interface GuestChannelConfig {
  welcome_message: string
  logo_url?: string
  primary_color: string
  max_file_size_mb: number
  allowed_file_types: string
}

export interface ChatMessage {
  id: string
  channel_id: string
  content: string
  created_by?: string
  guest_session_id?: string
  guest_display_name?: string
  sender_first_name?: string
  sender_last_name?: string
  created_at: string
  files?: ChatFile[]
}

export interface ChatFile {
  id: string
  filename: string
  content_type: string
  size: number
  url: string
}

export interface TokenValidationResult {
  session: GuestSession
  config: GuestChannelConfig
}

export interface WSMessage {
  type: string
  payload: Record<string, unknown>
}
