/**
 * Adapters between backend ContactInfo and the UI Contact type.
 *
 * The backend schema only covers core fields (name, email, phone, title,
 * companyId, notes, tags, customFields). All UI-specific fields (salutation,
 * mobile, address, jobTitle, department, website, category, status,
 * socialMedia, projects) are round-tripped through the backend's
 * custom_fields map with an underscore-prefixed key convention.
 */
import type { components } from '@/api/types'
import { dual } from '@/api/casing'
import { DEMO_MODE } from '@/mocks/demo-mode-flag'
import type {
  Contact,
  ContactAddress,
  ContactSocialMedia,
  CustomFieldValue,
} from '@/stores/contacts'

type BackendContact = components['schemas']['ContactInfo']
type CreateContactRequest = components['schemas']['CreateContactRequest']
type UpdateContactRequest = components['schemas']['UpdateContactRequest']

/** Shape of UI-specific fields stored inside backend customFields */
interface ExtraFields {
  _salutation?: string
  _mobile?: string
  _jobTitle?: string
  _department?: string
  _address?: ContactAddress
  _website?: string
  _category?: string
  _status?: string
  _socialMedia?: ContactSocialMedia
  _projects?: string[]
  _customFields?: Record<string, CustomFieldValue>
}

/** Convert a backend ContactInfo to a full UI Contact. isFavorite is always false
 *  here — callers should overlay it from local Zustand state. */
export function backendContactToUI(c: BackendContact): Contact {
  // Das gRPC-Gateway liefert snake_case (protobuf json-Tags), die OpenAPI-Typen
  // sind camelCase (Spec-Drift, X-3). `dual()` liest beide Casings, bis die Spec
  // auf snake_case konsolidiert ist — sonst bleiben Name/Firma/Datum leer.
  const extra = (dual<ExtraFields>(c, 'customFields') ?? {}) as ExtraFields
  const firstName = dual<string>(c, 'firstName') ?? ''
  const lastName = dual<string>(c, 'lastName') ?? ''
  const updatedAt = dual<string>(c, 'updatedAt')
  const createdAt = dual<string>(c, 'createdAt')
  const initials = `${firstName[0] ?? ''}${lastName[0] ?? ''}`.toUpperCase()

  return {
    id: c.id ?? '',
    salutation: (extra._salutation as 'Herr' | 'Frau' | '') ?? '',
    // Akademischer Titel (Dr./Prof.) — reines Extra-Feld, hat kein echtes
    // Backend-Pendant (gegen das echte Backend leer, im Mock via `title`).
    title: c.title ?? '',
    firstName,
    lastName,
    initials,
    email: c.email ?? '',
    phone: c.phone ?? '',
    mobile: extra._mobile ?? '',
    company: dual<string>(c, 'companyName') ?? '',
    // Job-Position ("Geschäftsführer"): echtes Backend liefert sie im
    // `position`-Feld, der Mock im custom-fields-Extra `_jobTitle` (X-3).
    jobTitle: (dual<string>(c, 'position') ?? extra._jobTitle) ?? '',
    department: extra._department ?? '',
    address: extra._address ?? { street: '', zip: '', city: '', country: 'Deutschland' },
    website: extra._website ?? '',
    category: (extra._category as 'employee' | 'customer' | 'partner') ?? 'customer',
    status: (extra._status as 'active' | 'prospect' | 'inactive') ?? 'active',
    tags: (c.tags ?? []).map((t) => t.name ?? '').filter(Boolean),
    notes: c.notes ?? '',
    socialMedia: extra._socialMedia ?? { linkedin: '', xing: '' },
    lastContact: updatedAt?.split('T')[0] ?? new Date().toISOString().split('T')[0],
    projects: extra._projects ?? [],
    createdAt: createdAt?.split('T')[0] ?? '',
    isFavorite: false,
    activities: [],
    customFields: extra._customFields,
  }
}

/** Build the custom_fields payload that carries all UI-only fields. */
function buildExtraFields(data: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>): Record<string, unknown> {
  const extra: ExtraFields = {
    _salutation: data.salutation,
    _mobile: data.mobile,
    _jobTitle: data.jobTitle,
    _department: data.department,
    _address: data.address,
    _website: data.website,
    _category: data.category,
    _status: data.status,
    _socialMedia: data.socialMedia,
    _projects: data.projects,
  }
  if (data.customFields) {
    extra._customFields = data.customFields
  }
  return extra as Record<string, unknown>
}

type FormData = Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>

/** Kernfelder, die das echte Contact-Schema kennt (snake_case, Gateway-konform). */
function coreContactFields(data: FormData): {
  first_name: string
  last_name: string
  email?: string
  phone?: string
  notes?: string
} {
  return {
    first_name: data.firstName,
    last_name: data.lastName,
    email: data.email || undefined,
    phone: data.phone || undefined,
    notes: data.notes || undefined,
  }
}

/**
 * Map UI form data → Create/Update payload, je nach Modus.
 *
 * Mock (DEMO_MODE): toleriert das `_`-präfixierte `custom_fields`-Objekt und das
 * `title`-Feld → der volle UI-Feldumfang bleibt im Demo-Modus erhalten.
 *
 * Echtes Backend: Das Contact-Schema kennt nur Kernfelder. Job-Position liegt im
 * `position`-Feld (die OpenAPI-Spec nennt es fälschlich `title` und kennt kein
 * `position` → Cast, X-3). `custom_fields` verlangt echte Custom-Field-UUIDs →
 * leeres Array. Die 9 UI-Extra-Felder (mobile/address/jobTitle/…) sind eine
 * Backend-Lücke (Handover Luke) und werden serverseitig noch nicht persistiert.
 */
function uiFormToWriteRequest(data: FormData): CreateContactRequest {
  if (DEMO_MODE) {
    return {
      ...coreContactFields(data),
      title: data.title || undefined,
      custom_fields: buildExtraFields(data),
    }
  }
  return {
    ...coreContactFields(data),
    position: data.jobTitle || undefined,
    custom_fields: [],
  } as unknown as CreateContactRequest
}

/** Map UI form data → backend CreateContactRequest */
export function uiFormToCreateRequest(data: FormData): CreateContactRequest {
  return uiFormToWriteRequest(data)
}

/** Map UI form data → backend UpdateContactRequest */
export function uiFormToUpdateRequest(data: FormData): UpdateContactRequest {
  return uiFormToWriteRequest(data) as unknown as UpdateContactRequest
}
