/**
 * Customization handlers (v1.0 Fundament) — MSW endpoints for the
 * label-override and value-set overlay layers.
 *
 * Mirrors the resolver logic from ../data/customization.ts; stateful
 * against that module. The real backend (Luke's track 🔒,
 * backend-gaps §Customization) will conform to the same shapes once
 * the tenant_label_overrides + tenant_value_sets tables land.
 *
 * Endpoint contract:
 *   GET  /api/v1/customization/labels?locale=de[&base=1]
 *   PUT  /api/v1/customization/labels
 *   GET  /api/v1/customization/value-sets[?base=1]
 *   GET  /api/v1/customization/value-sets/:id[?base=1]
 *   PUT  /api/v1/customization/value-sets/:id
 */
import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import type {
  LabelOverridesResponse,
  UpdateLabelOverridesInput,
  UpsertValueSetInput,
  ValueSetResponse,
  ValueSetsResponse,
} from '@/api/customization-types'
import {
  activeConfigLayer,
  clearLabelOverride,
  listResolvedValueSets,
  resolveLabelOverrides,
  resolveValueSet,
  setLabelOverride,
  upsertValueSet,
  LABEL_WHITELIST,
} from '../data/customization'

const API = API_BASE_URL

const notFound = (msg: string) => HttpResponse.json({ error: msg }, { status: 404 })
const badRequest = (msg: string) => HttpResponse.json({ error: msg }, { status: 400 })

export const customizationHandlers = [
  // ── B · Label-Overrides ──────────────────────────────────────────────────

  /**
   * GET /api/v1/customization/labels?locale=de[&base=1]
   * Returns the resolved label map for a locale (all LABEL_WHITELIST keys).
   * ?base=1 returns code-default provenance for every key (editor baseline).
   */
  http.get(`${API}/api/v1/customization/labels`, ({ request }) => {
    const url = new URL(request.url)
    const locale = url.searchParams.get('locale') ?? 'de'
    const base = url.searchParams.get('base') === '1'

    const labels = resolveLabelOverrides(locale, base)
    const body: LabelOverridesResponse = { locale, labels }
    return HttpResponse.json(body)
  }),

  /**
   * PUT /api/v1/customization/labels
   * Upserts label overrides for a locale on the active layer.
   * Body: { locale, layer, overrides: Record<string, string> }
   *
   * Only LABEL_WHITELIST keys are accepted; unknown keys are silently
   * filtered (defensive against future whitelist shrinkage).
   */
  http.put(`${API}/api/v1/customization/labels`, async ({ request }) => {
    const input = (await request.json()) as UpdateLabelOverridesInput
    if (!input?.locale || !input?.overrides) {
      return badRequest('locale and overrides are required')
    }

    const layer = input.layer ?? activeConfigLayer()

    for (const [key, value] of Object.entries(input.overrides)) {
      if (!LABEL_WHITELIST.includes(key)) continue
      if (value === null || value === '') {
        clearLabelOverride(layer, input.locale, key)
      } else {
        setLabelOverride(layer, input.locale, key, value)
      }
    }

    // Return the freshly merged map after applying the changes
    const labels = resolveLabelOverrides(input.locale)
    const body: LabelOverridesResponse = { locale: input.locale, labels }
    return HttpResponse.json(body)
  }),

  // ── M · Value-Sets ───────────────────────────────────────────────────────

  /**
   * GET /api/v1/customization/value-sets[?base=1]
   * Returns all known value-sets, resolved across layers.
   * ?base=1 returns code-default values only (editor baseline).
   */
  http.get(`${API}/api/v1/customization/value-sets`, ({ request }) => {
    const base = new URL(request.url).searchParams.get('base') === '1'
    const valueSets = listResolvedValueSets(base)
    const body: ValueSetsResponse = { valueSets }
    return HttpResponse.json(body)
  }),

  /**
   * GET /api/v1/customization/value-sets/:id[?base=1]
   * Returns a single resolved value-set with per-option provenance.
   */
  http.get(`${API}/api/v1/customization/value-sets/:id`, ({ params, request }) => {
    const base = new URL(request.url).searchParams.get('base') === '1'
    const id = String(params.id)
    const valueSet = resolveValueSet(id, base)
    if (!valueSet) return notFound(`value_set_not_found: ${id}`)
    const body: ValueSetResponse = { valueSet }
    return HttpResponse.json(body)
  }),

  /**
   * PUT /api/v1/customization/value-sets/:id
   * Replaces a value-set's options on the active (or specified) layer.
   * Body: { layer?, name?, options: ValueSetOption[] }
   */
  http.put(`${API}/api/v1/customization/value-sets/:id`, async ({ params, request }) => {
    const id = String(params.id)
    const input = (await request.json()) as UpsertValueSetInput
    if (!input?.options) return badRequest('options are required')

    const layer = input.layer ?? activeConfigLayer()

    upsertValueSet(layer, {
      id,
      name: input.name ?? id,
      options: input.options,
    })

    const valueSet = resolveValueSet(id)
    if (!valueSet) return notFound(`value_set_not_found: ${id}`)
    const body: ValueSetResponse = { valueSet }
    return HttpResponse.json(body)
  }),
]
