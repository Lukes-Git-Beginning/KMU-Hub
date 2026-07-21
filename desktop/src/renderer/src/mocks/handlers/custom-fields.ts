/**
 * Custom Fields handlers (v1.1) — MSW endpoints for the unified custom
 * field definition CRUD across all 5 entities.
 *
 * Contract:
 *   GET    /api/v1/customization/fields[?entity=<entity>]
 *   POST   /api/v1/customization/fields
 *   GET    /api/v1/customization/fields/:id
 *   PUT    /api/v1/customization/fields/:id
 *   DELETE /api/v1/customization/fields/:id
 *
 * 🔒 BE note (Luke's track, backend-gaps §Customization): once the real
 * backend endpoints are up these handlers are replaced by the real network
 * layer. The shapes below are the agreed contract.
 */
import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import type {
  CustomFieldEntity,
  CreateCustomFieldInput,
  UpdateCustomFieldInput,
} from '../data/custom-fields'
import {
  listCustomFields,
  getCustomField,
  createCustomField,
  updateCustomField,
  deleteCustomField,
} from '../data/custom-fields'

const API = API_BASE_URL

const notFound = (msg: string) => HttpResponse.json({ error: msg }, { status: 404 })
const badRequest = (msg: string) => HttpResponse.json({ error: msg }, { status: 400 })

export const customFieldHandlers = [
  /**
   * GET /api/v1/customization/fields[?entity=<entity>]
   * Returns all field definitions, optionally filtered by entity.
   */
  http.get(`${API}/api/v1/customization/fields`, ({ request }) => {
    const url = new URL(request.url)
    const entity = url.searchParams.get('entity') as CustomFieldEntity | null
    const fields = listCustomFields(entity ?? undefined)
    return HttpResponse.json({ fields })
  }),

  /**
   * POST /api/v1/customization/fields
   * Creates a new custom field definition.
   * Body: CreateCustomFieldInput
   */
  http.post(`${API}/api/v1/customization/fields`, async ({ request }) => {
    const input = (await request.json()) as CreateCustomFieldInput
    if (!input?.entity || !input?.label || !input?.type) {
      return badRequest('entity, label and type are required')
    }
    const field = createCustomField(input)
    return HttpResponse.json({ field }, { status: 201 })
  }),

  /**
   * GET /api/v1/customization/fields/:id
   * Returns a single field definition.
   */
  http.get(`${API}/api/v1/customization/fields/:id`, ({ params }) => {
    const id = String(params.id)
    const field = getCustomField(id)
    if (!field) return notFound(`custom_field_not_found: ${id}`)
    return HttpResponse.json({ field })
  }),

  /**
   * PUT /api/v1/customization/fields/:id
   * Updates an existing field definition.
   * Body: UpdateCustomFieldInput (partial)
   */
  http.put(`${API}/api/v1/customization/fields/:id`, async ({ params, request }) => {
    const id = String(params.id)
    const input = (await request.json()) as UpdateCustomFieldInput
    const field = updateCustomField(id, input)
    if (!field) return notFound(`custom_field_not_found: ${id}`)
    return HttpResponse.json({ field })
  }),

  /**
   * DELETE /api/v1/customization/fields/:id
   * Deletes a field definition.
   * The UI layer must show the consequence dialog for inUse=true fields
   * before calling this endpoint — the API deletes unconditionally.
   */
  http.delete(`${API}/api/v1/customization/fields/:id`, ({ params }) => {
    const id = String(params.id)
    const ok = deleteCustomField(id)
    if (!ok) return notFound(`custom_field_not_found: ${id}`)
    return HttpResponse.json({ success: true })
  }),
]
