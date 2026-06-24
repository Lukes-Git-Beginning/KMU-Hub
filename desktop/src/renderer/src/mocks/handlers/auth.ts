import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { CURRENT_USER } from '../data/shared-ids'

const API = API_BASE_URL

const mockUser = {
  id: CURRENT_USER.id,
  email: CURRENT_USER.email,
  first_name: CURRENT_USER.firstName,
  last_name: CURRENT_USER.lastName,
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

  // Password reset (enumeration-safe: always 200)
  http.post(`${API}/api/v1/auth/forgot-password`, () => {
    return HttpResponse.json({ message: { id: 'demo', content: 'reset link sent' } })
  }),

  http.post(`${API}/api/v1/auth/reset-password`, () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  // Sessions: kanonisch in handlers/security.ts (Security-Center „Aktive Sitzungen").
  // Hier bewusst entfernt, um den BE-konformen { sessions: [...] }-Handler dort
  // gewinnen zu lassen (vorher: nacktes Array hier → maskierte den security-Handler).

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
