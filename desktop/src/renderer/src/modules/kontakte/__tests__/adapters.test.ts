import { describe, it, expect, vi } from 'vitest'

// DEMO_MODE off: this file is about the real-backend path, which is the one
// that used to drop nine visible form fields on save (G0-6).
vi.mock('@/mocks/demo-mode-flag', () => ({ DEMO_MODE: false }))

import { uiFormToCreateRequest, backendContactToUI } from '../adapters'
import type { Contact } from '@/stores/contacts'

type FormData = Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>

function form(overrides: Partial<FormData> = {}): FormData {
  return {
    salutation: 'Herr',
    title: 'Dr.',
    firstName: 'John',
    lastName: 'Doe',
    email: 'john@example.com',
    phone: '+49 6131 123456',
    mobile: '+49 170 1234567',
    company: 'Acme GmbH',
    jobTitle: 'Geschäftsführer',
    department: 'Einkauf',
    address: { street: 'Hauptstraße 1', zip: '55131', city: 'Mainz', country: 'Deutschland' },
    website: 'https://example.com',
    category: 'partner',
    status: 'prospect',
    tags: [],
    notes: 'Wichtiger Kontakt',
    socialMedia: { linkedin: 'https://linkedin.com/in/jdoe', xing: 'https://xing.com/profile/jdoe' },
    lastContact: '2026-08-16',
    projects: [],
    isFavorite: false,
    ...overrides,
  }
}

describe('uiFormToCreateRequest (real backend)', () => {
  it('carries every visible form field into the request', () => {
    const req = uiFormToCreateRequest(form()) as unknown as Record<string, unknown>

    expect(req).toMatchObject({
      first_name: 'John',
      last_name: 'Doe',
      email: 'john@example.com',
      phone: '+49 6131 123456',
      notes: 'Wichtiger Kontakt',
      position: 'Geschäftsführer',
      salutation: 'Herr',
      title: 'Dr.',
      mobile: '+49 170 1234567',
      department: 'Einkauf',
      address_street: 'Hauptstraße 1',
      address_zip: '55131',
      address_city: 'Mainz',
      address_country: 'Deutschland',
      website: 'https://example.com',
      linkedin: 'https://linkedin.com/in/jdoe',
      xing: 'https://xing.com/profile/jdoe',
      category: 'partner',
      status: 'prospect',
    })
  })

  it('sends an empty string for cleared fields so the update clears the column', () => {
    const req = uiFormToCreateRequest(
      form({ mobile: '', department: '', website: '' }),
    ) as unknown as Record<string, unknown>

    // Not undefined: an omitted field means "leave alone" on update, which
    // would silently keep the old value the user just deleted.
    expect(req.mobile).toBe('')
    expect(req.department).toBe('')
    expect(req.website).toBe('')
  })

  it('leaves custom_fields empty — those need real field UUIDs', () => {
    const req = uiFormToCreateRequest(form()) as unknown as Record<string, unknown>
    expect(req.custom_fields).toEqual([])
  })
})

describe('backendContactToUI', () => {
  it('reads the profile columns back', () => {
    const ui = backendContactToUI({
      id: 'c1',
      firstName: 'John',
      lastName: 'Doe',
      salutation: 'Herr',
      title: 'Dr.',
      mobile: '+49 170 1234567',
      department: 'Einkauf',
      address_street: 'Hauptstraße 1',
      address_zip: '55131',
      address_city: 'Mainz',
      address_country: 'Deutschland',
      website: 'https://example.com',
      linkedin: 'https://linkedin.com/in/jdoe',
      xing: 'https://xing.com/profile/jdoe',
      category: 'partner',
      status: 'prospect',
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any)

    expect(ui.salutation).toBe('Herr')
    expect(ui.title).toBe('Dr.')
    expect(ui.mobile).toBe('+49 170 1234567')
    expect(ui.department).toBe('Einkauf')
    expect(ui.address).toEqual({
      street: 'Hauptstraße 1',
      zip: '55131',
      city: 'Mainz',
      country: 'Deutschland',
    })
    expect(ui.website).toBe('https://example.com')
    expect(ui.socialMedia).toEqual({
      linkedin: 'https://linkedin.com/in/jdoe',
      xing: 'https://xing.com/profile/jdoe',
    })
    expect(ui.category).toBe('partner')
    expect(ui.status).toBe('prospect')
  })

  it('falls back to the pre-000314 custom_fields extras', () => {
    const ui = backendContactToUI({
      id: 'c1',
      firstName: 'John',
      lastName: 'Doe',
      customFields: {
        _mobile: '+49 170 9999999',
        _address: { street: 'Altweg 2', zip: '10115', city: 'Berlin', country: 'Deutschland' },
        _category: 'customer',
      },
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any)

    expect(ui.mobile).toBe('+49 170 9999999')
    expect(ui.address.city).toBe('Berlin')
    expect(ui.category).toBe('customer')
  })

  it('defaults an empty contact without throwing', () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const ui = backendContactToUI({ id: 'c1', firstName: 'A', lastName: 'B' } as any)

    expect(ui.address).toEqual({ street: '', zip: '', city: '', country: 'Deutschland' })
    expect(ui.socialMedia).toEqual({ linkedin: '', xing: '' })
    expect(ui.category).toBe('customer')
    expect(ui.status).toBe('active')
  })
})
