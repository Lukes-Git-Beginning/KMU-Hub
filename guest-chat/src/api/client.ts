import type { ChatFile, ChatMessage, GuestChannelConfig, GuestSession, TokenValidationResult } from './types'

class GuestApiClient {
  private token: string = ''

  setToken(token: string) {
    this.token = token
  }

  private async request<T>(path: string, options?: RequestInit): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(this.token ? { 'X-Guest-Token': this.token } : {}),
    }
    const res = await fetch(`${path}`, {
      ...options,
      headers: { ...headers, ...options?.headers },
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      throw new Error(body.error || `HTTP ${res.status}`)
    }
    return res.json()
  }

  validateToken(token: string): Promise<TokenValidationResult> {
    return this.request('/api/v1/guest/sessions/validate', {
      method: 'POST',
      body: JSON.stringify({ token }),
    })
  }

  createSession(
    channelId: string,
    displayName: string,
    email?: string,
  ): Promise<{ token: string; session: GuestSession; config: GuestChannelConfig }> {
    return this.request('/api/v1/guest/sessions', {
      method: 'POST',
      body: JSON.stringify({ channel_id: channelId, display_name: displayName, email }),
    })
  }

  getMessages(
    channelId: string,
    before?: string,
    limit = 50,
  ): Promise<{ messages: ChatMessage[] }> {
    const params = new URLSearchParams({ limit: String(limit) })
    if (before) params.set('before', before)
    return this.request(`/api/v1/guest/channels/${channelId}/messages?${params}`)
  }

  sendMessage(channelId: string, content: string): Promise<{ message: ChatMessage }> {
    return this.request(`/api/v1/guest/channels/${channelId}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    })
  }

  async uploadFile(channelId: string, file: File): Promise<ChatFile> {
    const formData = new FormData()
    formData.append('file', file)
    const res = await fetch(`/api/v1/guest/channels/${channelId}/files`, {
      method: 'POST',
      headers: { 'X-Guest-Token': this.token },
      body: formData,
    })
    if (!res.ok) throw new Error(`Upload failed: ${res.status}`)
    return res.json()
  }

  getConfig(): Promise<GuestChannelConfig> {
    return this.request('/api/v1/guest/config')
  }
}

export const apiClient = new GuestApiClient()
