import { create } from 'zustand'
import { persist } from 'zustand/middleware'

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
}

export interface ContactGroup {
  id: string
  name: string
  color: string
  contactIds: string[]
}

interface ContactsState {
  contacts: Contact[]
  groups: ContactGroup[]
  addContact: (contact: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>) => void
  bulkAddContacts: (contacts: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>[]) => number
  updateContact: (id: string, updates: Partial<Contact>) => void
  deleteContact: (id: string) => void
  toggleFavorite: (id: string) => void
  duplicateContact: (id: string) => void
  addGroup: (name: string, color: string) => void
  updateGroup: (id: string, updates: Partial<Omit<ContactGroup, 'id'>>) => void
  deleteGroup: (id: string) => void
  addContactToGroup: (groupId: string, contactId: string) => void
  removeContactFromGroup: (groupId: string, contactId: string) => void
}

const mockContacts: Contact[] = [
  {
    id: 'c1',
    salutation: 'Frau',
    firstName: 'Anna',
    lastName: 'Mueller',
    initials: 'AM',
    email: 'anna.mueller@kmuhub.ch',
    phone: '+41 44 123 45 67',
    mobile: '+41 79 123 45 67',
    company: 'KMU Hub AG',
    jobTitle: 'Projektleiterin',
    department: 'Entwicklung',
    address: { street: 'Bahnhofstrasse 10', zip: '8001', city: 'Zürich', country: 'Schweiz' },
    website: 'www.kmuhub.ch',
    category: 'employee',
    status: 'active',
    tags: ['Technik', 'Projektleitung'],
    notes: 'Verantwortlich für alle Kundenprojekte. Bevorzugt Kommunikation per E-Mail.',
    socialMedia: { linkedin: 'anna-mueller-ch', xing: 'Anna_Mueller' },
    lastContact: '2026-02-08',
    projects: ['Website Relaunch', 'Mobile App'],
    createdAt: '2025-06-15',
    isFavorite: true,
    activities: [
      { id: 'a1', type: 'meeting', description: 'Sprint Planning Q1', date: '2026-02-09' },
      { id: 'a2', type: 'email', description: 'Projektupdate versendet', date: '2026-02-08' },
      { id: 'a3', type: 'call', description: 'Rueckruf wegen Deadline', date: '2026-02-07' },
    ],
  },
  {
    id: 'c2',
    salutation: 'Herr',
    firstName: 'Michael',
    lastName: 'Berg',
    initials: 'MB',
    email: 'michael.berg@kmuhub.ch',
    phone: '+41 44 123 45 68',
    mobile: '+41 79 234 56 78',
    company: 'KMU Hub AG',
    jobTitle: 'Senior Developer',
    department: 'Entwicklung',
    address: { street: 'Limmatquai 42', zip: '8001', city: 'Zürich', country: 'Schweiz' },
    website: 'www.kmuhub.ch',
    category: 'employee',
    status: 'active',
    tags: ['Technik', 'Backend'],
    notes: 'Backend-Spezialist, Go und PostgreSQL. Zuständig für API-Architektur.',
    socialMedia: { linkedin: 'michael-berg-dev', xing: '' },
    lastContact: '2026-02-08',
    projects: ['CRM Integration', 'Security Audit'],
    createdAt: '2025-06-15',
    isFavorite: false,
    activities: [
      { id: 'a4', type: 'meeting', description: 'API Review', date: '2026-02-07' },
      { id: 'a5', type: 'note', description: 'Neuen Microservice besprochen', date: '2026-02-06' },
    ],
  },
  {
    id: 'c3',
    salutation: 'Frau',
    firstName: 'Sarah',
    lastName: 'Klein',
    initials: 'SK',
    email: 'sarah@designstudio.ch',
    phone: '+41 41 567 89 01',
    mobile: '+41 79 345 67 89',
    company: 'Design Studio GmbH',
    jobTitle: 'Creative Director',
    department: 'Design',
    address: { street: 'Seestrasse 5', zip: '6003', city: 'Luzern', country: 'Schweiz' },
    website: 'www.designstudio.ch',
    category: 'partner',
    status: 'active',
    tags: ['Design', 'UX', 'Extern'],
    notes: 'Externe Design-Partnerin. Arbeitet remote, bevorzugt Videocalls.',
    socialMedia: { linkedin: 'sarah-klein-design', xing: 'Sarah_Klein' },
    lastContact: '2026-02-07',
    projects: ['Website Relaunch'],
    createdAt: '2025-08-22',
    isFavorite: true,
    activities: [
      { id: 'a6', type: 'meeting', description: 'Design Review Mobile App', date: '2026-02-09' },
      { id: 'a7', type: 'email', description: 'Mockups v3 erhalten', date: '2026-02-05' },
    ],
  },
  {
    id: 'c4',
    salutation: 'Herr',
    firstName: 'Thomas',
    lastName: 'Weber',
    initials: 'TW',
    email: 'thomas.weber@abc-gmbh.ch',
    phone: '+41 31 456 78 90',
    mobile: '+41 79 456 78 90',
    company: 'ABC GmbH',
    jobTitle: 'Geschäftsführer',
    department: 'Management',
    address: { street: 'Bundesplatz 1', zip: '3011', city: 'Bern', country: 'Schweiz' },
    website: 'www.abc-gmbh.ch',
    category: 'customer',
    status: 'active',
    tags: ['VIP', 'Entscheider'],
    notes: 'Hauptkunde, sehr zufrieden. Vertrag läuft bis Ende 2027. Potenzial für Upselling.',
    socialMedia: { linkedin: 'thomas-weber-bern', xing: 'Thomas_Weber' },
    lastContact: '2026-02-05',
    projects: ['CRM Integration'],
    createdAt: '2025-03-10',
    isFavorite: true,
    activities: [
      { id: 'a8', type: 'meeting', description: 'Quartalsbesprechung', date: '2026-02-05' },
      { id: 'a9', type: 'email', description: 'Angebot für Phase 2 versendet', date: '2026-02-03' },
      { id: 'a10', type: 'call', description: 'Feedback zum Prototyp', date: '2026-01-28' },
    ],
  },
  {
    id: 'c5',
    salutation: 'Frau',
    firstName: 'Lisa',
    lastName: 'Schmidt',
    initials: 'LS',
    email: 'lisa.schmidt@kmuhub.ch',
    phone: '+41 44 123 45 69',
    mobile: '+41 79 567 89 01',
    company: 'KMU Hub AG',
    jobTitle: 'Marketing Manager',
    department: 'Marketing',
    address: { street: 'Bahnhofstrasse 10', zip: '8001', city: 'Zürich', country: 'Schweiz' },
    website: 'www.kmuhub.ch',
    category: 'employee',
    status: 'active',
    tags: ['Marketing', 'Content'],
    notes: 'Verantwortlich für Social Media und Content-Strategie.',
    socialMedia: { linkedin: 'lisa-schmidt-marketing', xing: 'Lisa_Schmidt' },
    lastContact: '2026-02-06',
    projects: ['Marketing Kampagne'],
    createdAt: '2025-07-01',
    isFavorite: false,
    activities: [
      { id: 'a11', type: 'email', description: 'Newsletter-Entwurf versendet', date: '2026-02-06' },
      { id: 'a12', type: 'meeting', description: 'Content-Planung Februar', date: '2026-02-04' },
    ],
  },
  {
    id: 'c6',
    salutation: 'Herr',
    firstName: 'Peter',
    lastName: 'Koch',
    initials: 'PK',
    email: 'peter.koch@kmuhub.ch',
    phone: '+41 44 123 45 70',
    mobile: '+41 79 678 90 12',
    company: 'KMU Hub AG',
    jobTitle: 'Security Engineer',
    department: 'IT',
    address: { street: 'Bahnhofstrasse 10', zip: '8001', city: 'Zürich', country: 'Schweiz' },
    website: 'www.kmuhub.ch',
    category: 'employee',
    status: 'active',
    tags: ['Technik', 'Security', 'CISSP'],
    notes: 'CISSP-zertifiziert. Zuständig für Security Audits und Penetrationstests.',
    socialMedia: { linkedin: 'peter-koch-security', xing: '' },
    lastContact: '2026-02-07',
    projects: ['Security Audit', 'Datenmigration'],
    createdAt: '2025-06-15',
    isFavorite: false,
    activities: [
      { id: 'a13', type: 'meeting', description: 'Security Briefing', date: '2026-02-06' },
      { id: 'a14', type: 'note', description: 'Audit-Report fertiggestellt', date: '2026-02-05' },
    ],
  },
  {
    id: 'c7',
    salutation: 'Frau',
    firstName: 'Maria',
    lastName: 'Huber',
    initials: 'MH',
    email: 'maria.huber@xyz-ag.ch',
    phone: '+41 61 789 01 23',
    mobile: '+41 79 789 01 23',
    company: 'XYZ AG',
    jobTitle: 'CFO',
    department: 'Finanzen',
    address: { street: 'Marktgasse 15', zip: '4001', city: 'Basel', country: 'Schweiz' },
    website: 'www.xyz-ag.ch',
    category: 'customer',
    status: 'prospect',
    tags: ['Finanzen', 'Entscheider'],
    notes: 'Potentieller Neukunde. Erstgespräch am 15.02 geplant. Interesse an CRM-Lösung.',
    socialMedia: { linkedin: 'maria-huber-finance', xing: 'Maria_Huber' },
    lastContact: '2026-01-28',
    projects: [],
    createdAt: '2026-01-15',
    isFavorite: false,
    activities: [
      { id: 'a15', type: 'email', description: 'Infomaterial versendet', date: '2026-01-28' },
      { id: 'a16', type: 'call', description: 'Erstkontakt per Telefon', date: '2026-01-20' },
    ],
  },
  {
    id: 'c8',
    salutation: 'Herr',
    firstName: 'David',
    lastName: 'Meier',
    initials: 'DM',
    email: 'david.meier@swisshosting.ch',
    phone: '+41 52 890 12 34',
    mobile: '+41 79 890 12 34',
    company: 'Swiss Hosting AG',
    jobTitle: 'Account Manager',
    department: 'Sales',
    address: { street: 'Technopark 3', zip: '8400', city: 'Winterthur', country: 'Schweiz' },
    website: 'www.swisshosting.ch',
    category: 'partner',
    status: 'active',
    tags: ['Hosting', 'SLA'],
    notes: 'Hosting-Provider. SLA-Vertrag bis Ende 2027. Sehr zuverlässig.',
    socialMedia: { linkedin: 'david-meier-hosting', xing: '' },
    lastContact: '2026-02-04',
    projects: [],
    createdAt: '2025-04-20',
    isFavorite: false,
    activities: [
      { id: 'a17', type: 'email', description: 'SLA-Verlängerung besprochen', date: '2026-02-04' },
    ],
  },
  {
    id: 'c9',
    salutation: 'Frau',
    firstName: 'Sandra',
    lastName: 'Braun',
    initials: 'SB',
    email: 'sandra.braun@oldclient.ch',
    phone: '+41 62 901 23 45',
    mobile: '+41 79 901 23 45',
    company: 'Old Client GmbH',
    jobTitle: 'IT-Leiterin',
    department: 'IT',
    address: { street: 'Industriestrasse 8', zip: '5000', city: 'Aarau', country: 'Schweiz' },
    website: 'www.oldclient.ch',
    category: 'customer',
    status: 'inactive',
    tags: ['IT'],
    notes: 'Vertrag im November 2025 ausgelaufen. Mögliche Verlängerung in Diskussion.',
    socialMedia: { linkedin: '', xing: 'Sandra_Braun' },
    lastContact: '2025-11-15',
    projects: [],
    createdAt: '2024-09-01',
    isFavorite: false,
    activities: [
      { id: 'a18', type: 'email', description: 'Follow-up Vertragsverlängerung', date: '2025-11-15' },
    ],
  },
  {
    id: 'c10',
    salutation: 'Herr',
    firstName: 'Jonas',
    lastName: 'Diaz',
    initials: 'JD',
    email: 'jonas.diaz@kmuhub.ch',
    phone: '+41 44 123 45 71',
    mobile: '+41 79 012 34 56',
    company: 'KMU Hub AG',
    jobTitle: 'Frontend Developer',
    department: 'Entwicklung',
    address: { street: 'Bahnhofstrasse 10', zip: '8001', city: 'Zürich', country: 'Schweiz' },
    website: 'www.kmuhub.ch',
    category: 'employee',
    status: 'active',
    tags: ['Technik', 'Frontend', 'React'],
    notes: 'React-Spezialist. Arbeitet eng mit Sarah Klein zusammen.',
    socialMedia: { linkedin: 'jonas-diaz-dev', xing: '' },
    lastContact: '2026-02-08',
    projects: ['Website Relaunch', 'Mobile App'],
    createdAt: '2025-09-01',
    isFavorite: false,
    activities: [
      { id: 'a19', type: 'meeting', description: 'Code Review Frontend', date: '2026-02-08' },
      { id: 'a20', type: 'note', description: 'Neue Komponenten-Library evaluiert', date: '2026-02-06' },
    ],
  },
  {
    id: 'c11',
    salutation: 'Frau',
    firstName: 'Eva',
    lastName: 'Brunner',
    initials: 'EB',
    email: 'eva@brunner-partner.ch',
    phone: '+41 71 234 56 78',
    mobile: '+41 79 111 22 33',
    company: 'Brunner & Partner',
    jobTitle: 'Rechtsanwältin',
    department: 'Recht',
    address: { street: 'Oberer Graben 12', zip: '9000', city: 'St. Gallen', country: 'Schweiz' },
    website: 'www.brunner-partner.ch',
    category: 'partner',
    status: 'active',
    tags: ['Recht', 'DSGVO', 'Verträge'],
    notes: 'Externe Rechtsberatung. Spezialisiert auf IT-Recht und DSGVO.',
    socialMedia: { linkedin: 'eva-brunner-law', xing: 'Eva_Brunner' },
    lastContact: '2026-02-01',
    projects: ['Security Audit'],
    createdAt: '2025-05-10',
    isFavorite: false,
    activities: [
      { id: 'a21', type: 'email', description: 'Datenschutz-Gutachten erhalten', date: '2026-02-01' },
      { id: 'a22', type: 'meeting', description: 'DSGVO-Beratung', date: '2026-01-25' },
    ],
  },
  {
    id: 'c12',
    salutation: 'Herr',
    firstName: 'Markus',
    lastName: 'Steiner',
    initials: 'MS',
    email: 'markus@steiner-bau.ch',
    phone: '+41 33 345 67 89',
    mobile: '+41 79 222 33 44',
    company: 'Steiner Bauunternehmen AG',
    jobTitle: 'CEO',
    department: 'Management',
    address: { street: 'Hauptstrasse 25', zip: '3600', city: 'Thun', country: 'Schweiz' },
    website: 'www.steiner-bau.ch',
    category: 'customer',
    status: 'active',
    tags: ['Bau', 'VIP', 'Entscheider'],
    notes: 'Neukunde seit Januar 2026. Bauunternehmen mit 80 Mitarbeitern. Sehr interessiert an Projektmanagement-Modul.',
    socialMedia: { linkedin: 'markus-steiner-bau', xing: 'Markus_Steiner' },
    lastContact: '2026-02-03',
    projects: [],
    createdAt: '2026-01-05',
    isFavorite: true,
    activities: [
      { id: 'a23', type: 'meeting', description: 'Onboarding-Workshop', date: '2026-02-03' },
      { id: 'a24', type: 'email', description: 'Vertrag unterzeichnet', date: '2026-01-20' },
      { id: 'a25', type: 'call', description: 'Anforderungen besprochen', date: '2026-01-10' },
    ],
  },
  {
    id: 'c13',
    salutation: 'Frau',
    firstName: 'Claudia',
    lastName: 'Frei',
    initials: 'CF',
    email: 'claudia.frei@techventures.at',
    phone: '+43 1 456 78 90',
    mobile: '+43 660 123 45 67',
    company: 'TechVentures GmbH',
    jobTitle: 'Head of Sales',
    department: 'Vertrieb',
    address: { street: 'Kärntner Strasse 30', zip: '1010', city: 'Wien', country: 'Österreich' },
    website: 'www.techventures.at',
    category: 'customer',
    status: 'prospect',
    tags: ['Sales', 'Österreich'],
    notes: 'Kontakt über Messe. Interesse an KMU Hub für ihr 50-Personen-Team. Demo am 20.02 geplant.',
    socialMedia: { linkedin: 'claudia-frei-sales', xing: 'Claudia_Frei' },
    lastContact: '2026-02-02',
    projects: [],
    createdAt: '2026-01-28',
    isFavorite: false,
    activities: [
      { id: 'a26', type: 'email', description: 'Demo-Termin bestätigt', date: '2026-02-02' },
      { id: 'a27', type: 'call', description: 'Erstgespräch nach Messe', date: '2026-01-28' },
    ],
  },
  {
    id: 'c14',
    salutation: 'Herr',
    firstName: 'Andreas',
    lastName: 'Hofer',
    initials: 'AH',
    email: 'andreas@alpina-consulting.de',
    phone: '+49 89 567 89 01',
    mobile: '+49 170 123 45 67',
    company: 'Alpina Consulting',
    jobTitle: 'Berater',
    department: 'Beratung',
    address: { street: 'Maximilianstrasse 14', zip: '80539', city: 'Muenchen', country: 'Deutschland' },
    website: 'www.alpina-consulting.de',
    category: 'partner',
    status: 'active',
    tags: ['Beratung', 'Deutschland', 'Prozesse'],
    notes: 'Externer Berater für Prozessoptimierung. Hilft bei Onsite-Analysen für Kunden.',
    socialMedia: { linkedin: 'andreas-hofer-consulting', xing: 'Andreas_Hofer' },
    lastContact: '2026-02-06',
    projects: ['CRM Integration'],
    createdAt: '2025-10-15',
    isFavorite: false,
    activities: [
      { id: 'a28', type: 'meeting', description: 'Prozessanalyse ABC GmbH', date: '2026-02-06' },
      { id: 'a29', type: 'email', description: 'Bericht Prozessoptimierung', date: '2026-02-01' },
    ],
  },
]

let nextId = 15

export const useContactsStore = create<ContactsState>()(
  persist(
    (set, get) => ({
      contacts: mockContacts,
      groups: [
        { id: 'g1', name: 'VIP-Kunden', color: '#F59E0B', contactIds: ['c4', 'c12'] },
        { id: 'g2', name: 'Entwicklerteam', color: '#3B82F6', contactIds: ['c2', 'c6', 'c10'] },
        { id: 'g3', name: 'Externe Partner', color: '#8B5CF6', contactIds: ['c3', 'c8', 'c11', 'c14'] },
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
    }),
    { name: 'kmuhub-contacts' }
  )
)
