/**
 * Adapters between the backend CompanyInfo wire shape and the camelCase shape
 * the companies UI reads. Same drift family as kontakte (X-3):
 *  - casing: gateway emits snake_case (`created_at`, `contact_count`), the
 *    OpenAPI types are camelCase → read via `dual()`.
 *  - field name: the wire field is `domain`, the UI/spec call it `website`.
 *  - write: `custom_fields` must be `[{field_id,value}]` against the real
 *    backend (object → 400); UI-only extras (_phone/_email/_size/_tags) have no
 *    backend home (handover) → dropped in real mode, kept in DEMO_MODE.
 */
import type { components } from '@/api/types'
import { dual } from '@/api/casing'

type CompanyInfo = components['schemas']['CompanyInfo']

/** Normalize a backend company object (snake_case wire OR camelCase mock) to the
 *  camelCase shape the UI components read. */
export function backendCompanyToUI(c: unknown): CompanyInfo {
  const raw = (c ?? {}) as Record<string, unknown>
  return {
    ...raw,
    id: (raw.id as string) ?? '',
    name: (raw.name as string) ?? '',
    // Field-name drift: wire `domain`, UI `website`.
    website: (dual<string>(c, 'website') ?? dual<string>(c, 'domain')) ?? '',
    industry: (raw.industry as string) ?? '',
    address: (raw.address as string) ?? '',
    notes: (raw.notes as string) ?? '',
    contactCount: dual<number>(c, 'contactCount') ?? 0,
    createdAt: dual<string>(c, 'createdAt') ?? '',
    customFields: (dual(c, 'customFields') ?? {}) as CompanyInfo['customFields'],
    tags: (raw.tags as CompanyInfo['tags']) ?? [],
  } as CompanyInfo
}
