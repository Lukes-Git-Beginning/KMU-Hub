import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { getAllContactsForCRM } from '@/mocks/mock-db'

export interface ContactAddress {
  street: string
  zip: string
  city: string
  country: string
}

export interface ContactSocialMedia {
  linkedin: string
  xing: string
}

export interface ContactActivity {
  id: string
  type: 'email' | 'call' | 'meeting' | 'note'
  description: string
  date: string
}

export interface Contact {
  id: string
  salutation: 'Herr' | 'Frau' | ''
  title: string // Academic title: Dr., Prof., Prof. Dr., Dipl.-Ing., etc.
  firstName: string
  lastName: string
  initials: string
  email: string
  phone: string
  mobile: string
  company: string
  jobTitle: string
  department: string
  address: ContactAddress
  website: string
  category: 'employee' | 'customer' | 'partner'
  status: 'active' | 'prospect' | 'inactive'
  tags: string[]
  notes: string
  socialMedia: ContactSocialMedia
  lastContact: string
  projects: string[]
  createdAt: string
  isFavorite: boolean
  activities: ContactActivity[]
  customFields?: Record<string, CustomFieldValue>
}

export interface ContactGroup {
  id: string
  name: string
  color: string
  contactIds: string[]
}

// ---------------------------------------------------------------------------
// Custom fields
// ---------------------------------------------------------------------------

export type CustomFieldType = 'text' | 'number' | 'date' | 'dropdown' | 'checkbox' | 'url'

export interface CustomFieldDefinition {
  id: string
  name: string
  type: CustomFieldType
  options?: string[] // for dropdown type
  required: boolean
  sortOrder: number
}

export type CustomFieldValue = string | number | boolean | null

interface ContactsState {
  contacts: Contact[]
  groups: ContactGroup[]
  customFieldDefinitions: CustomFieldDefinition[]

  addContact: (contact: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'> & { title?: string }) => void
  bulkAddContacts: (contacts: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>[] ) => number
  updateContact: (id: string, updates: Partial<Contact>) => void
  deleteContact: (id: string) => void
  toggleFavorite: (id: string) => void
  duplicateContact: (id: string) => void
  addGroup: (name: string, color: string) => void
  updateGroup: (id: string, updates: Partial<Omit<ContactGroup, 'id'>>) => void
  deleteGroup: (id: string) => void
  addContactToGroup: (groupId: string, contactId: string) => void
  removeContactFromGroup: (groupId: string, contactId: string) => void

  // Custom field actions
  addCustomFieldDefinition: (field: Omit<CustomFieldDefinition, 'id' | 'sortOrder'>) => void
  updateCustomFieldDefinition: (id: string, updates: Partial<CustomFieldDefinition>) => void
  deleteCustomFieldDefinition: (id: string) => void
  setCustomFieldValue: (contactId: string, fieldId: string, value: CustomFieldValue) => void
}

// ---------------------------------------------------------------------------
// Initial data from central mock-db (TechVision GmbH, 18 employees + 20 external)
// ---------------------------------------------------------------------------

const mockContacts: Contact[] = getAllContactsForCRM().map((c) => {
  // Add custom field values to select customers
  if (c.id === 'x1') return { ...c, customFields: { cf1: 'K-2022-001', cf2: 2800000, cf3: 'Produktion', cf4: true, cf5: '2022-05-10' } }
  if (c.id === 'x3') return { ...c, customFields: { cf1: 'K-2023-003', cf2: 5200000, cf3: 'Produktion', cf4: true } }
  if (c.id === 'x7') return { ...c, customFields: { cf1: 'K-2023-007', cf2: 12000000, cf3: 'Bau', cf4: true, cf5: '2023-09-01' } }
  if (c.id === 'x8') return { ...c, customFields: { cf1: 'K-2024-008', cf2: 45000000, cf3: 'Handel', cf4: false } }
  return c
}) as Contact[]

const mockCustomFieldDefinitions: CustomFieldDefinition[] = [
  { id: 'cf1', name: 'Kundennummer', type: 'text', required: false, sortOrder: 1 },
  { id: 'cf2', name: 'Jahresumsatz', type: 'number', required: false, sortOrder: 2 },
  { id: 'cf3', name: 'Branche', type: 'dropdown', options: ['IT', 'Bau', 'Handel', 'Dienstleistung', 'Produktion', 'Gesundheit', 'Bildung', 'Logistik', 'Tourismus', 'Automotive', 'Sonstige'], required: false, sortOrder: 3 },
  { id: 'cf4', name: 'Newsletter', type: 'checkbox', required: false, sortOrder: 4 },
  { id: 'cf5', name: 'Vertragsbeginn', type: 'date', required: false, sortOrder: 5 },
]

let nextId = 100
let nextCfId = 6

export const useContactsStore = create<ContactsState>()(
  persist(
    (set, get) => ({
      contacts: mockContacts,
      customFieldDefinitions: mockCustomFieldDefinitions,
      groups: [
        { id: 'g1', name: 'VIP-Kunden', color: '#F59E0B', contactIds: ['x1', 'x3', 'x7', 'x8'] },
        { id: 'g2', name: 'Entwicklerteam', color: '#3B82F6', contactIds: ['e2', 'e4', 'e5', 'e6', 'e7'] },
        { id: 'g3', name: 'Externe Partner', color: '#8B5CF6', contactIds: ['x11', 'x12', 'x13', 'x14', 'x15'] },
        { id: 'g4', name: 'Prospects', color: '#10B981', contactIds: ['x16', 'x17', 'x18', 'x19', 'x20'] },
      ],

      addContact: (contact) => {
        const id = `c${nextId++}`
        const initials = `${contact.firstName[0] || ''}${contact.lastName[0] || ''}`.toUpperCase()
        set((state) => ({
          contacts: [
            {
              ...contact,
              id,
              initials,
              title: contact.title ?? '',
              createdAt: new Date().toISOString().split('T')[0],
              activities: [],
            },
            ...state.contacts,
          ],
        }))
      },

      bulkAddContacts: (contacts) => {
        const newContacts = contacts.map((c) => ({
          ...c,
          id: `c${nextId++}`,
          initials: `${c.firstName[0] || ''}${c.lastName[0] || ''}`.toUpperCase(),
          createdAt: new Date().toISOString().split('T')[0],
          activities: [] as ContactActivity[],
        }))
        set((state) => ({ contacts: [...newContacts, ...state.contacts] }))
        return newContacts.length
      },

      updateContact: (id, updates) =>
        set((state) => ({
          contacts: state.contacts.map((c) =>
            c.id === id
              ? {
                  ...c,
                  ...updates,
                  initials: updates.firstName || updates.lastName
                    ? `${(updates.firstName || c.firstName)[0]}${(updates.lastName || c.lastName)[0]}`.toUpperCase()
                    : c.initials,
                }
              : c
          ),
        })),

      deleteContact: (id) =>
        set((state) => ({
          contacts: state.contacts.filter((c) => c.id !== id),
          groups: state.groups.map((g) => ({
            ...g,
            contactIds: g.contactIds.filter((cId) => cId !== id),
          })),
        })),

      toggleFavorite: (id) =>
        set((state) => ({
          contacts: state.contacts.map((c) =>
            c.id === id ? { ...c, isFavorite: !c.isFavorite } : c
          ),
        })),

      duplicateContact: (id) => {
        const original = get().contacts.find((c) => c.id === id)
        if (!original) return
        const newId = `c${nextId++}`
        set((state) => ({
          contacts: [
            {
              ...original,
              id: newId,
              firstName: `${original.firstName} (Kopie)`,
              initials: original.initials,
              createdAt: new Date().toISOString().split('T')[0],
              isFavorite: false,
            },
            ...state.contacts,
          ],
        }))
      },

      addGroup: (name, color) =>
        set((state) => ({
          groups: [...state.groups, { id: `g${Date.now()}`, name, color, contactIds: [] }],
        })),

      updateGroup: (id, updates) =>
        set((state) => ({
          groups: state.groups.map((g) => (g.id === id ? { ...g, ...updates } : g)),
        })),

      deleteGroup: (id) =>
        set((state) => ({
          groups: state.groups.filter((g) => g.id !== id),
        })),

      addContactToGroup: (groupId, contactId) =>
        set((state) => ({
          groups: state.groups.map((g) =>
            g.id === groupId && !g.contactIds.includes(contactId)
              ? { ...g, contactIds: [...g.contactIds, contactId] }
              : g
          ),
        })),

      removeContactFromGroup: (groupId, contactId) =>
        set((state) => ({
          groups: state.groups.map((g) =>
            g.id === groupId
              ? { ...g, contactIds: g.contactIds.filter((id) => id !== contactId) }
              : g
          ),
        })),

      // -- Custom field actions --

      addCustomFieldDefinition: (field) => {
        const id = `cf${nextCfId++}`
        set((state) => ({
          customFieldDefinitions: [
            ...state.customFieldDefinitions,
            { ...field, id, sortOrder: state.customFieldDefinitions.length + 1 },
          ],
        }))
      },

      updateCustomFieldDefinition: (id, updates) =>
        set((state) => ({
          customFieldDefinitions: state.customFieldDefinitions.map((f) =>
            f.id === id ? { ...f, ...updates } : f,
          ),
        })),

      deleteCustomFieldDefinition: (id) =>
        set((state) => ({
          customFieldDefinitions: state.customFieldDefinitions.filter((f) => f.id !== id),
          contacts: state.contacts.map((c) => {
            if (!c.customFields || !(id in c.customFields)) return c
            const { [id]: _, ...rest } = c.customFields
            return { ...c, customFields: Object.keys(rest).length > 0 ? rest : undefined }
          }),
        })),

      setCustomFieldValue: (contactId, fieldId, value) =>
        set((state) => ({
          contacts: state.contacts.map((c) =>
            c.id === contactId
              ? {
                  ...c,
                  customFields: {
                    ...c.customFields,
                    [fieldId]: value,
                  },
                }
              : c,
          ),
        })),
    }),
    { name: 'kmuhub-contacts' }
  )
)
