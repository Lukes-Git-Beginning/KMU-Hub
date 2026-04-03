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
  const extra = (c.customFields ?? {}) as ExtraFields
  const firstName = c.firstName ?? ''
  const lastName = c.lastName ?? ''
  const initials = `${firstName[0] ?? ''}${lastName[0] ?? ''}`.toUpperCase()

  return {
    id: c.id ?? '',
    salutation: (extra._salutation as 'Herr' | 'Frau' | '') ?? '',
    title: c.title ?? '',
    firstName,
    lastName,
    initials,
    email: c.email ?? '',
    phone: c.phone ?? '',
    mobile: extra._mobile ?? '',
    company: c.companyName ?? '',
    jobTitle: extra._jobTitle ?? '',
    department: extra._department ?? '',
    address: extra._address ?? { street: '', zip: '', city: '', country: 'Deutschland' },
    website: extra._website ?? '',
    category: (extra._category as 'employee' | 'customer' | 'partner') ?? 'customer',
    status: (extra._status as 'active' | 'prospect' | 'inactive') ?? 'active',
    tags: (c.tags ?? []).map((t) => t.name ?? '').filter(Boolean),
    notes: c.notes ?? '',
    socialMedia: extra._socialMedia ?? { linkedin: '', xing: '' },
    lastContact: c.updatedAt?.split('T')[0] ?? new Date().toISOString().split('T')[0],
    projects: extra._projects ?? [],
    createdAt: c.createdAt?.split('T')[0] ?? '',
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

/** Map UI form data → backend CreateContactRequest */
export function uiFormToCreateRequest(
  data: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>,
): CreateContactRequest {
  return {
    first_name: data.firstName,
    last_name: data.lastName,
    email: data.email || undefined,
    phone: data.phone || undefined,
    title: data.title || undefined,
    notes: data.notes || undefined,
    custom_fields: buildExtraFields(data),
  }
}

/** Map UI form data → backend UpdateContactRequest */
export function uiFormToUpdateRequest(
  data: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>,
): UpdateContactRequest {
  return {
    first_name: data.firstName,
    last_name: data.lastName,
    email: data.email || undefined,
    phone: data.phone || undefined,
    title: data.title || undefined,
    notes: data.notes || undefined,
    custom_fields: buildExtraFields(data),
  }
}
