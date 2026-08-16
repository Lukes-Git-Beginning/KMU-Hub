/**
 * Adapters between backend ContactInfo and the UI Contact type.
 *
 * Since migration 000314 the backend stores the whole contact form: salutation,
 * academic title, mobile, department, the four address parts, website,
 * LinkedIn, Xing, category and status are real columns. Before that they only
 * survived in DEMO_MODE, and saving against the real backend dropped them
 * without a word (G0-6).
 *
 * `projects` is the one field still carried in custom_fields: it is a list of
 * links to the work module, not a scalar, and wiring it properly is its own
 * piece of work.
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

  // Real columns win; the `_`-prefixed extras remain as the fallback so
  // DEMO_MODE and any contact saved before 000314 still render.
  const street = dual<string>(c, 'addressStreet') ?? extra._address?.street ?? ''
  const zip = dual<string>(c, 'addressZip') ?? extra._address?.zip ?? ''
  const city = dual<string>(c, 'addressCity') ?? extra._address?.city ?? ''
  const country = dual<string>(c, 'addressCountry') ?? extra._address?.country ?? 'Deutschland'

  return {
    id: c.id ?? '',
    salutation: ((c.salutation ?? extra._salutation) as 'Herr' | 'Frau' | '') ?? '',
    // Akademischer Titel (Dr./Prof.) — seit 000314 eine echte Spalte, getrennt
    // von `position` (Job-Rolle).
    title: c.title ?? '',
    firstName,
    lastName,
    initials,
    email: c.email ?? '',
    phone: c.phone ?? '',
    mobile: c.mobile ?? extra._mobile ?? '',
    company: dual<string>(c, 'companyName') ?? '',
    // Job-Position ("Geschäftsführer"). Das Extra `_jobTitle` bleibt als
    // Rückfall für Datensätze, die vor 000314 gespeichert wurden.
    jobTitle: (dual<string>(c, 'position') ?? extra._jobTitle) ?? '',
    department: c.department ?? extra._department ?? '',
    address: { street, zip, city, country },
    website: c.website ?? extra._website ?? '',
    category: ((c.category ?? extra._category) as 'employee' | 'customer' | 'partner') ?? 'customer',
    status: ((c.status ?? extra._status) as 'active' | 'prospect' | 'inactive') ?? 'active',
    tags: (c.tags ?? []).map((t) => t.name ?? '').filter(Boolean),
    notes: c.notes ?? '',
    socialMedia: {
      linkedin: c.linkedin ?? extra._socialMedia?.linkedin ?? '',
      xing: c.xing ?? extra._socialMedia?.xing ?? '',
    },
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

/**
 * Alle Felder, die das echte Contact-Schema kennt (snake_case, Gateway-konform).
 *
 * Leerstring statt `undefined` bei den Profilfeldern: Der Nutzer, der ein Feld
 * leert, will es geleert haben — beim Update bedeutet `undefined` „nicht
 * mitgeschickt", der Wert bliebe also stehen.
 */
function coreContactFields(data: FormData): Record<string, unknown> {
  return {
    first_name: data.firstName,
    last_name: data.lastName,
    email: data.email || undefined,
    phone: data.phone || undefined,
    notes: data.notes || undefined,
    position: data.jobTitle ?? '',
    salutation: data.salutation ?? '',
    title: data.title ?? '',
    mobile: data.mobile ?? '',
    department: data.department ?? '',
    address_street: data.address?.street ?? '',
    address_zip: data.address?.zip ?? '',
    address_city: data.address?.city ?? '',
    address_country: data.address?.country ?? '',
    website: data.website ?? '',
    linkedin: data.socialMedia?.linkedin ?? '',
    xing: data.socialMedia?.xing ?? '',
    category: data.category ?? '',
    status: data.status ?? '',
  }
}

/**
 * Map UI form data → Create/Update payload, je nach Modus.
 *
 * Mock (DEMO_MODE): reicht zusätzlich das `_`-präfixierte `custom_fields`-Objekt
 * durch, damit die Demo-Handler unverändert weiterlaufen.
 *
 * Echtes Backend: seit Migration 000314 nimmt das Schema alle Formularfelder
 * entgegen. `custom_fields` bleibt leer — es verlangt echte Custom-Field-UUIDs,
 * und `projects` (das einzige verbliebene Extra) ist eine Verknüpfung ins
 * work-Modul, kein Skalar.
 */
function uiFormToWriteRequest(data: FormData): CreateContactRequest {
  const core = coreContactFields(data)
  if (DEMO_MODE) {
    return {
      ...core,
      custom_fields: buildExtraFields(data),
    } as unknown as CreateContactRequest
  }
  return {
    ...core,
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
