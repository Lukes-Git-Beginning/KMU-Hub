import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type RentalObjectType = 'gerät' | 'raum' | 'fahrzeug' | 'werkzeug'

export interface RentalObject {
  id: string
  name: string
  type: RentalObjectType
  description: string
  location: string
  serialNumber?: string
  dailyRate: number
  weeklyRate?: number
  currency: string
  deposit?: number
  status: 'available' | 'reserved' | 'maintenance'
  imageUrl?: string
}

export interface Reservation {
  id: string
  objectId: string
  objectName: string
  startDate: string
  endDate: string
  renter: string
  renterType: 'employee' | 'customer'
  notes: string
  status: 'active' | 'upcoming' | 'completed' | 'cancelled'
  pickupLocation: string
  returnLocation: string
  currency: string
  depositAmount?: number
  depositStatus?: 'none' | 'collected' | 'returned'
  totalPrice?: number
}

// Wave 9 — Zustandsprotokoll
export interface ZustandsprotokollItem {
  label: string
  condition: 'ok' | 'damaged' | 'missing'
  note?: string
}

export interface Zustandsprotokoll {
  id: string
  reservationId: string
  type: 'pickup' | 'return'
  date: string
  checklist: ZustandsprotokollItem[]
  photoCount: number
  signatureDataUrl?: string
  notes: string
  createdBy: string
}

interface VermietungStore {
  objects: RentalObject[]
  reservations: Reservation[]
  zustandsprotokolle: Zustandsprotokoll[]
  addObject: (obj: RentalObject) => void
  updateObject: (id: string, data: Partial<RentalObject>) => void
  deleteObject: (id: string) => void
  addReservation: (res: Reservation) => void
  updateReservation: (id: string, data: Partial<Reservation>) => void
  cancelReservation: (id: string) => void
  addZustandsprotokoll: (z: Zustandsprotokoll) => void
}

const MOCK_OBJECTS: RentalObject[] = [
  {
    id: 'obj-1',
    name: 'Beamer Epson EB-W49',
    type: 'gerät',
    description: 'Full-HD Beamer mit 3800 Lumen, HDMI & USB, inkl. Fernbedienung und Tragetasche',
    location: 'Büro Zürich',
    serialNumber: 'EP-W49-2024-0871',
    dailyRate: 45,
    currency: 'EUR',
    status: 'available',
  },
  {
    id: 'obj-2',
    name: 'Konferenzraum A',
    type: 'raum',
    description: 'Grosser Konferenzraum für bis zu 20 Personen, Beamer & Whiteboard vorhanden',
    location: '2. OG',
    dailyRate: 150,
    currency: 'EUR',
    status: 'reserved',
  },
  {
    id: 'obj-3',
    name: 'VW Transporter',
    type: 'fahrzeug',
    description: 'VW T6.1 Transporter, Ladeflaeche 6m3, Anhängerkupplung',
    location: 'Tiefgarage',
    serialNumber: 'ZH-482731',
    dailyRate: 120,
    weeklyRate: 650,
    currency: 'EUR',
    deposit: 500,
    status: 'available',
  },
  {
    id: 'obj-4',
    name: 'Hilti Bohrhammer',
    type: 'werkzeug',
    description: 'Hilti TE 30-A36, 36V Akku-Bohrhammer mit SDS-plus, inkl. 2 Akkus und Koffer',
    location: 'Lager Winterthur',
    serialNumber: 'HI-TE30-5523',
    dailyRate: 35,
    currency: 'EUR',
    deposit: 200,
    status: 'available',
  },
  {
    id: 'obj-5',
    name: 'Hebebuehne 12m',
    type: 'gerät',
    description: 'Selbstfahrende Scherenbuehne, max. Arbeitshoehe 12m, Tragkraft 350kg',
    location: 'Aussenlager',
    serialNumber: 'HB-12M-0042',
    dailyRate: 280,
    weeklyRate: 1500,
    currency: 'EUR',
    deposit: 1000,
    status: 'maintenance',
  },
  {
    id: 'obj-6',
    name: 'Schulungsraum B',
    type: 'raum',
    description: 'Schulungsraum für bis zu 12 Personen, Flipchart & Beamer',
    location: '1. OG',
    dailyRate: 100,
    currency: 'EUR',
    status: 'available',
  },
  {
    id: 'obj-7',
    name: 'Laptop-Pool (5 Stk.)',
    type: 'gerät',
    description: '5x Lenovo ThinkPad T14s, 16GB RAM, 512GB SSD, Windows 11 Pro',
    location: 'IT-Büro',
    dailyRate: 25,
    currency: 'EUR',
    status: 'reserved',
  },
  {
    id: 'obj-8',
    name: 'PKW-Anhänger',
    type: 'fahrzeug',
    description: 'Einachser-Anhänger 750kg, Plane & Spriegel, Stuetzrad',
    location: 'Tiefgarage',
    serialNumber: 'ZH-ANH-1105',
    dailyRate: 55,
    weeklyRate: 280,
    currency: 'EUR',
    deposit: 300,
    status: 'available',
  },
]

const MOCK_RESERVATIONS: Reservation[] = [
  {
    id: 'res-1',
    objectId: 'obj-2',
    objectName: 'Konferenzraum A',
    startDate: '2026-02-14',
    endDate: '2026-02-14',
    renter: 'Thomas Keller',
    renterType: 'employee',
    notes: 'Kundenpraesentation Q1',
    status: 'active',
    pickupLocation: '2. OG',
    returnLocation: '2. OG',
    currency: 'EUR',
  },
  {
    id: 'res-2',
    objectId: 'obj-7',
    objectName: 'Laptop-Pool (5 Stk.)',
    startDate: '2026-02-13',
    endDate: '2026-02-17',
    renter: 'Sandra Müller',
    renterType: 'employee',
    notes: 'Workshop-Woche Digitalisierung',
    status: 'active',
    pickupLocation: 'IT-Büro',
    returnLocation: 'IT-Büro',
    currency: 'EUR',
  },
  {
    id: 'res-3',
    objectId: 'obj-3',
    objectName: 'VW Transporter',
    startDate: '2026-02-18',
    endDate: '2026-02-20',
    renter: 'Meier Elektro AG',
    renterType: 'customer',
    notes: 'Materialtransport Baustelle Winterthur',
    status: 'upcoming',
    pickupLocation: 'Tiefgarage',
    returnLocation: 'Tiefgarage',
    currency: 'EUR',
  },
  {
    id: 'res-4',
    objectId: 'obj-1',
    objectName: 'Beamer Epson EB-W49',
    startDate: '2026-02-19',
    endDate: '2026-02-19',
    renter: 'Lukas Brunner',
    renterType: 'employee',
    notes: 'Praesentation Teammeeting',
    status: 'upcoming',
    pickupLocation: 'Büro Zürich',
    returnLocation: 'Büro Zürich',
    currency: 'EUR',
  },
  {
    id: 'res-5',
    objectId: 'obj-4',
    objectName: 'Hilti Bohrhammer',
    startDate: '2026-02-20',
    endDate: '2026-02-22',
    renter: 'Reto Aeschlimann',
    renterType: 'employee',
    notes: 'Montage Kabelkanäle Neubau',
    status: 'upcoming',
    pickupLocation: 'Lager Winterthur',
    returnLocation: 'Lager Winterthur',
    currency: 'EUR',
  },
  {
    id: 'res-6',
    objectId: 'obj-6',
    objectName: 'Schulungsraum B',
    startDate: '2026-02-10',
    endDate: '2026-02-10',
    renter: 'Nicole Berger',
    renterType: 'employee',
    notes: 'Erste-Hilfe-Kurs',
    status: 'completed',
    pickupLocation: '1. OG',
    returnLocation: '1. OG',
    currency: 'EUR',
  },
  {
    id: 'res-7',
    objectId: 'obj-8',
    objectName: 'PKW-Anhänger',
    startDate: '2026-02-05',
    endDate: '2026-02-07',
    renter: 'Baumann GmbH',
    renterType: 'customer',
    notes: 'Entsorgung Altmaterial',
    status: 'completed',
    pickupLocation: 'Tiefgarage',
    returnLocation: 'Tiefgarage',
    currency: 'EUR',
  },
  {
    id: 'res-8',
    objectId: 'obj-1',
    objectName: 'Beamer Epson EB-W49',
    startDate: '2026-02-03',
    endDate: '2026-02-03',
    renter: 'Marco Hartmann',
    renterType: 'employee',
    notes: 'Kundenschulung',
    status: 'completed',
    pickupLocation: 'Büro Zürich',
    returnLocation: 'Büro Zürich',
    currency: 'EUR',
  },
  {
    id: 'res-9',
    objectId: 'obj-3',
    objectName: 'VW Transporter',
    startDate: '2026-02-08',
    endDate: '2026-02-09',
    renter: 'Daniel Frei',
    renterType: 'employee',
    notes: 'Umzug Lagermaterial',
    status: 'completed',
    pickupLocation: 'Tiefgarage',
    returnLocation: 'Tiefgarage',
    currency: 'EUR',
  },
  {
    id: 'res-10',
    objectId: 'obj-2',
    objectName: 'Konferenzraum A',
    startDate: '2026-02-21',
    endDate: '2026-02-21',
    renter: 'Irene Graf',
    renterType: 'employee',
    notes: 'Budget-Review Meeting',
    status: 'upcoming',
    pickupLocation: '2. OG',
    returnLocation: '2. OG',
    currency: 'EUR',
  },
  {
    id: 'res-11',
    objectId: 'obj-4',
    objectName: 'Hilti Bohrhammer',
    startDate: '2026-02-01',
    endDate: '2026-02-02',
    renter: 'Weber Bau AG',
    renterType: 'customer',
    notes: '',
    status: 'cancelled',
    pickupLocation: 'Lager Winterthur',
    returnLocation: 'Lager Winterthur',
    currency: 'EUR',
  },
  {
    id: 'res-12',
    objectId: 'obj-6',
    objectName: 'Schulungsraum B',
    startDate: '2026-02-25',
    endDate: '2026-02-26',
    renter: 'Thomas Keller',
    renterType: 'employee',
    notes: 'Sicherheitsschulung Neues Personal',
    status: 'upcoming',
    pickupLocation: '1. OG',
    returnLocation: '1. OG',
    currency: 'EUR',
  },
]

const MOCK_ZUSTANDSPROTOKOLLE: Zustandsprotokoll[] = [
  {
    id: 'zp-1',
    reservationId: 'res-7',
    type: 'pickup',
    date: '2026-02-05',
    checklist: [
      { label: 'Allgemeinzustand', condition: 'ok' },
      { label: 'Reifen/Raeder', condition: 'ok' },
      { label: 'Beleuchtung', condition: 'ok' },
      { label: 'Plane/Spriegel', condition: 'damaged', note: 'Kleine Rissbildung an der Seite' },
      { label: 'Ladungssicherung', condition: 'ok' },
    ],
    photoCount: 2,
    notes: 'Plane hat leichten Riss, Mieter informiert',
    createdBy: 'Thomas Keller',
  },
  {
    id: 'zp-2',
    reservationId: 'res-7',
    type: 'return',
    date: '2026-02-07',
    checklist: [
      { label: 'Allgemeinzustand', condition: 'ok' },
      { label: 'Reifen/Raeder', condition: 'ok' },
      { label: 'Beleuchtung', condition: 'ok' },
      { label: 'Plane/Spriegel', condition: 'damaged', note: 'Riss unveraendert' },
      { label: 'Ladungssicherung', condition: 'ok' },
    ],
    photoCount: 1,
    notes: 'Rückgabe ohne neue Schaeden',
    createdBy: 'Thomas Keller',
  },
]

export const useVermietungStore = create<VermietungStore>()(
  persist(
    (set) => ({
      objects: MOCK_OBJECTS,
      reservations: MOCK_RESERVATIONS,
      zustandsprotokolle: MOCK_ZUSTANDSPROTOKOLLE,

      addObject: (obj) =>
        set((state) => ({ objects: [...state.objects, obj] })),

      updateObject: (id, data) =>
        set((state) => ({
          objects: state.objects.map((o) => (o.id === id ? { ...o, ...data } : o)),
        })),

      deleteObject: (id) =>
        set((state) => ({
          objects: state.objects.filter((o) => o.id !== id),
        })),

      addReservation: (res) =>
        set((state) => ({ reservations: [...state.reservations, res] })),

      updateReservation: (id, data) =>
        set((state) => ({
          reservations: state.reservations.map((r) =>
            r.id === id ? { ...r, ...data } : r
          ),
        })),

      cancelReservation: (id) =>
        set((state) => ({
          reservations: state.reservations.map((r) =>
            r.id === id ? { ...r, status: 'cancelled' as const } : r
          ),
        })),

      addZustandsprotokoll: (z) =>
        set((state) => ({ zustandsprotokolle: [...state.zustandsprotokolle, z] })),
    }),
    { name: 'kmuhub-vermietung' },
  ),
)
