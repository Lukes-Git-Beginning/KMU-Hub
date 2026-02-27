import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// Re-export types for components that import them from here
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
  title: string
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
  options?: string[]
  required: boolean
  sortOrder: number
}

export type CustomFieldValue = string | number | boolean | null

interface ContactsState {
  /** Contact IDs the user has marked as favorite (local UI state). */
  favoriteIds: string[]
  groups: ContactGroup[]
  customFieldDefinitions: CustomFieldDefinition[]

  toggleFavorite: (id: string) => void
  addGroup: (name: string, color: string) => void
  updateGroup: (id: string, updates: Partial<Omit<ContactGroup, 'id'>>) => void
  deleteGroup: (id: string) => void
  addContactToGroup: (groupId: string, contactId: string) => void
  removeContactFromGroup: (groupId: string, contactId: string) => void

  addCustomFieldDefinition: (field: Omit<CustomFieldDefinition, 'id' | 'sortOrder'>) => void
  updateCustomFieldDefinition: (id: string, updates: Partial<CustomFieldDefinition>) => void
  deleteCustomFieldDefinition: (id: string) => void
}

let nextCfId = 6

export const useContactsStore = create<ContactsState>()(
  persist(
    (set) => ({
      favoriteIds: [],
      customFieldDefinitions: [
        { id: 'cf1', name: 'Kundennummer', type: 'text', required: false, sortOrder: 1 },
        { id: 'cf2', name: 'Jahresumsatz', type: 'number', required: false, sortOrder: 2 },
        { id: 'cf3', name: 'Branche', type: 'dropdown', options: ['IT', 'Bau', 'Handel', 'Dienstleistung', 'Produktion', 'Gesundheit', 'Bildung', 'Logistik', 'Tourismus', 'Automotive', 'Sonstige'], required: false, sortOrder: 3 },
        { id: 'cf4', name: 'Newsletter', type: 'checkbox', required: false, sortOrder: 4 },
        { id: 'cf5', name: 'Vertragsbeginn', type: 'date', required: false, sortOrder: 5 },
      ],
      groups: [
        { id: 'g1', name: 'VIP-Kunden', color: '#F59E0B', contactIds: [] },
        { id: 'g2', name: 'Entwicklerteam', color: '#3B82F6', contactIds: [] },
        { id: 'g3', name: 'Externe Partner', color: '#8B5CF6', contactIds: [] },
        { id: 'g4', name: 'Prospects', color: '#10B981', contactIds: [] },
      ],

      toggleFavorite: (id) =>
        set((state) => ({
          favoriteIds: state.favoriteIds.includes(id)
            ? state.favoriteIds.filter((fid) => fid !== id)
            : [...state.favoriteIds, id],
        })),

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
        })),
    }),
    { name: 'kmuhub-contacts' }
  )
)
