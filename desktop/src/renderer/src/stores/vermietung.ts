import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type RentalObjectType = 'geraet' | 'raum' | 'fahrzeug' | 'werkzeug'

export interface RentalObject {
  id: string
  name: string
  type: RentalObjectType
  description: string
  location: string
  serialNumber?: string
  dailyRate: number
  weeklyRate?: number
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
}

interface VermietungStore {
  objects: RentalObject[]
  reservations: Reservation[]
  addObject: (obj: RentalObject) => void
  updateObject: (id: string, data: Partial<RentalObject>) => void
  deleteObject: (id: string) => void
  addReservation: (res: Reservation) => void
  updateReservation: (id: string, data: Partial<Reservation>) => void
  cancelReservation: (id: string) => void
}

const MOCK_OBJECTS: RentalObject[] = [
  {
    id: 'obj-1',
    name: 'Beamer Epson EB-W49',
    type: 'geraet',
    description: 'Full-HD Beamer mit 3800 Lumen, HDMI & USB, inkl. Fernbedienung und Tragetasche',
    location: 'Buero Zuerich',
    serialNumber: 'EP-W49-2024-0871',
    dailyRate: 45,
    status: 'available',
  },
  {
    id: 'obj-2',
    name: 'Konferenzraum A',
    type: 'raum',
    description: 'Grosser Konferenzraum fuer bis zu 20 Personen, Beamer & Whiteboard vorhanden',
    location: '2. OG',
    dailyRate: 150,
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
    status: 'available',
  },
  {
    id: 'obj-5',
    name: 'Hebebuehne 12m',
    type: 'geraet',
    description: 'Selbstfahrende Scherenbuehne, max. Arbeitshoehe 12m, Tragkraft 350kg',
    location: 'Aussenlager',
    serialNumber: 'HB-12M-0042',
    dailyRate: 280,
    weeklyRate: 1500,
    status: 'maintenance',
  },
  {
    id: 'obj-6',
    name: 'Schulungsraum B',
    type: 'raum',
    description: 'Schulungsraum fuer bis zu 12 Personen, Flipchart & Beamer',
    location: '1. OG',
    dailyRate: 100,
    status: 'available',
  },
  {
    id: 'obj-7',
    name: 'Laptop-Pool (5 Stk.)',
    type: 'geraet',
    description: '5x Lenovo ThinkPad T14s, 16GB RAM, 512GB SSD, Windows 11 Pro',
    location: 'IT-Buero',
    dailyRate: 25,
    status: 'reserved',
  },
  {
    id: 'obj-8',
    name: 'PKW-Anhaenger',
    type: 'fahrzeug',
    description: 'Einachser-Anhaenger 750kg, Plane & Spriegel, Stuetzrad',
    location: 'Tiefgarage',
    serialNumber: 'ZH-ANH-1105',
    dailyRate: 55,
    weeklyRate: 280,
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
  },
  {
    id: 'res-2',
    objectId: 'obj-7',
    objectName: 'Laptop-Pool (5 Stk.)',
    startDate: '2026-02-13',
    endDate: '2026-02-17',
    renter: 'Sandra Mueller',
    renterType: 'employee',
    notes: 'Workshop-Woche Digitalisierung',
    status: 'active',
    pickupLocation: 'IT-Buero',
    returnLocation: 'IT-Buero',
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
    pickupLocation: 'Buero Zuerich',
    returnLocation: 'Buero Zuerich',
  },
  {
    id: 'res-5',
    objectId: 'obj-4',
    objectName: 'Hilti Bohrhammer',
    startDate: '2026-02-20',
    endDate: '2026-02-22',
    renter: 'Reto Aeschlimann',
    renterType: 'employee',
    notes: 'Montage Kabelkanaele Neubau',
    status: 'upcoming',
    pickupLocation: 'Lager Winterthur',
    returnLocation: 'Lager Winterthur',
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
  },
  {
    id: 'res-7',
    objectId: 'obj-8',
    objectName: 'PKW-Anhaenger',
    startDate: '2026-02-05',
    endDate: '2026-02-07',
    renter: 'Baumann GmbH',
    renterType: 'customer',
    notes: 'Entsorgung Altmaterial',
    status: 'completed',
    pickupLocation: 'Tiefgarage',
    returnLocation: 'Tiefgarage',
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
    pickupLocation: 'Buero Zuerich',
    returnLocation: 'Buero Zuerich',
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
  },
]

export const useVermietungStore = create<VermietungStore>()(
  persist(
    (set) => ({
      objects: MOCK_OBJECTS,
      reservations: MOCK_RESERVATIONS,

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
    }),
    { name: 'kmuhub-vermietung' },
  ),
)
