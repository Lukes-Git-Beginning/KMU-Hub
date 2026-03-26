import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'

const API = API_BASE_URL

const mockUser = {
  id: IDS.users.stefan,
  email: 'stefan.mueller@techvision.de',
  first_name: 'Stefan',
  last_name: 'Mueller',
  roles: ['admin'],
  avatar_url: null,
}

const mockTokens = {
  access_token: 'demo-access-token-000',
  refresh_token: 'demo-refresh-token-000',
  user: mockUser,
}

export const authHandlers = [
  http.get(`${API}/api/v1/auth/me`, () => {
    return HttpResponse.json({ user: mockUser })
  }),

  http.post(`${API}/api/v1/auth/login`, () => {
    return HttpResponse.json(mockTokens)
  }),

  http.post(`${API}/api/v1/auth/logout`, () => {
    return HttpResponse.json({ success: true })
  }),

  http.post(`${API}/api/v1/auth/refresh`, () => {
    return HttpResponse.json(mockTokens)
  }),

  http.post(`${API}/api/v1/auth/2fa/validate-login`, () => {
    return HttpResponse.json(mockTokens)
  }),

  // Sessions
  http.get(`${API}/api/v1/auth/sessions`, () => {
    return HttpResponse.json([
      {
        id: 'sess-001',
        user_id: IDS.users.stefan,
        ip_address: '192.168.1.100',
        user_agent: 'KMU Hub Desktop/1.0',
        created_at: new Date().toISOString(),
        last_active_at: new Date().toISOString(),
        is_current: true,
      },
    ])
  }),

  http.get(`${API}/api/v1/auth/sessions/all`, () => {
    return HttpResponse.json({ sessions: [], total: 0 })
  }),

  // Presence
  http.get(`${API}/api/v1/video/presence/:userId`, () => {
    return HttpResponse.json({ status: 'online', last_seen: new Date().toISOString() })
  }),

  http.post(`${API}/api/v1/video/presence/bulk`, () => {
    return HttpResponse.json({})
  }),

  http.post(`${API}/api/v1/video/presence/status`, () => {
    return HttpResponse.json({})
  }),

  http.get(`${API}/api/v1/video/presence/config`, () => {
    return HttpResponse.json({ away_timeout_seconds: 300 })
  }),
]
