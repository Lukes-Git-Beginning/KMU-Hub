/**
 * MSW handlers for the account-link status endpoint
 * (GET /api/v1/integrations/link/:platform/status).
 *
 * Mirrors the flat response shape returned by the gateway's
 * HandleGetLinkStatus (backend/internal/gateway/route_integration.go):
 * { linked, platform, external_display_name?, linked_at? }.
 */
import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'

const API = API_BASE_URL

const linkStatusByPlatform: Record<string, Record<string, unknown>> = {
  teams: {
    linked: true,
    platform: 'teams',
    external_display_name: 'Markus Weber (Teams)',
    linked_at: '2026-05-01T09:00:00Z',
  },
  slack: {
    linked: false,
    platform: 'slack',
  },
}

export const integrationHandlers = [
  http.get(`${API}/api/v1/integrations/link/:platform/status`, ({ params }) => {
    const platform = String(params.platform)
    const status = linkStatusByPlatform[platform] ?? { linked: false, platform }
    return HttpResponse.json(status)
  }),
]
