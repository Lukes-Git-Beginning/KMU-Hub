import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { toast } from 'sonner'

export interface Vehicle {
  id: string
  licensePlate: string
  make: string
  model: string
  year: number
  type: 'car' | 'van' | 'truck'
  currentDriver: string
  mileage: number
  nextInspection: string
  insuranceExpiry: string
  isActive: boolean
}

export interface MaintenanceRecord {
  id: string
  vehicleId: string
  vehiclePlate: string
  type: 'service' | 'repair' | 'inspection' | 'tires'
  date: string
  mileage: number
  cost: number
  notes: string
}

export interface FuelRecord {
  id: string
  vehicleId: string
  vehiclePlate: string
  date: string
  liters: number
  cost: number
  mileage: number
}

export interface VehiclePosition {
  lat: number
  lng: number
  address: string
  timestamp: string
}

export interface VehicleRoute {
  id: string
  vehicleId: string
  vehicleName: string
  date: string
  positions: VehiclePosition[]
  dailyKm: number
  status: 'driving' | 'parked' | 'unknown'
  driver: string
}

// Wave 9 — Fahrtenbuch
export interface LogbookEntry {
  id: string
  vehicleId: string
  vehiclePlate: string
  date: string
  startLocation: string
  endLocation: string
  purpose: string
  startKm: number
  endKm: number
  km: number
  isPrivate: boolean
  driver: string
}

// Wave 9 — Dokumente pro Fahrzeug
export interface VehicleDocument {
  id: string
  vehicleId: string
  type: 'registration' | 'insurance' | 'tuev' | 'other'
  name: string
  uploadDate: string
  expiryDate?: string
}

// Wave 9 — Schadensmeldung
export interface DamageReport {
  id: string
  vehicleId: string
  vehiclePlate: string
  date: string
  description: string
  severity: 'minor' | 'moderate' | 'major'
  location: string
  photoCount: number
  reportedBy: string
  status: 'open' | 'in_progress' | 'resolved'
}

interface FuhrparkStore {
  vehicles: Vehicle[]
  maintenanceRecords: MaintenanceRecord[]
  fuelRecords: FuelRecord[]
  vehicleRoutes: VehicleRoute[]
  logbookEntries: LogbookEntry[]
  vehicleDocuments: VehicleDocument[]
  damageReports: DamageReport[]
  refreshTracking: () => void
}

const MOCK_VEHICLES: Vehicle[] = [
  { id: 'veh-1', licensePlate: 'ZH 345 678', make: 'VW', model: 'Caddy Cargo', year: 2024, type: 'van', currentDriver: 'Thomas Keller', mileage: 18420, nextInspection: '2026-08-15', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-2', licensePlate: 'ZH 112 233', make: 'Skoda', model: 'Octavia Combi', year: 2023, type: 'car', currentDriver: 'Lukas Brunner', mileage: 34560, nextInspection: '2026-05-20', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-3', licensePlate: 'BE 456 789', make: 'Mercedes-Benz', model: 'Sprinter 314', year: 2022, type: 'truck', currentDriver: 'Reto Aeschlimann', mileage: 67890, nextInspection: '2026-03-10', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-4', licensePlate: 'ZH 998 877', make: 'Renault', model: 'Kangoo E-Tech', year: 2025, type: 'van', currentDriver: 'Sandra Müller', mileage: 5230, nextInspection: '2027-02-01', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-5', licensePlate: 'AG 223 344', make: 'Toyota', model: 'Proace City', year: 2023, type: 'van', currentDriver: 'Daniel Frei', mileage: 41200, nextInspection: '2026-06-30', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-6', licensePlate: 'ZH 667 788', make: 'Iveco', model: 'Daily 35S14', year: 2021, type: 'truck', currentDriver: '', mileage: 89450, nextInspection: '2026-04-15', insuranceExpiry: '2027-01-01', isActive: false },
]

const MOCK_MAINTENANCE: MaintenanceRecord[] = [
  { id: 'mnt-1', vehicleId: 'veh-1', vehiclePlate: 'ZH 345 678', type: 'service', date: '2026-01-20', mileage: 15000, cost: 480.00, notes: 'Jährlicher Service, Ölwechsel, Filterwechsel' },
  { id: 'mnt-2', vehicleId: 'veh-2', vehiclePlate: 'ZH 112 233', type: 'tires', date: '2025-11-05', mileage: 30200, cost: 1120.00, notes: 'Winterreifen montiert, 4x Continental WinterContact TS 870' },
  { id: 'mnt-3', vehicleId: 'veh-3', vehiclePlate: 'BE 456 789', type: 'repair', date: '2026-01-08', mileage: 65400, cost: 2340.00, notes: 'Bremsen vorne komplett erneuert, Bremsscheiben + Beläge' },
  { id: 'mnt-4', vehicleId: 'veh-3', vehiclePlate: 'BE 456 789', type: 'inspection', date: '2025-03-10', mileage: 52000, cost: 180.00, notes: 'MFK bestanden ohne Mängel' },
  { id: 'mnt-5', vehicleId: 'veh-5', vehiclePlate: 'AG 223 344', type: 'service', date: '2026-02-01', mileage: 40000, cost: 520.00, notes: '40\'000 km Service gemäß Herstellervorgabe' },
  { id: 'mnt-6', vehicleId: 'veh-6', vehiclePlate: 'ZH 667 788', type: 'repair', date: '2025-12-15', mileage: 88000, cost: 3800.00, notes: 'Getriebeschaden, Fahrzeug stillgelegt bis Reparaturentscheid' },
  { id: 'mnt-7', vehicleId: 'veh-4', vehiclePlate: 'ZH 998 877', type: 'service', date: '2026-02-10', mileage: 5000, cost: 150.00, notes: 'Erster Service Elektrofahrzeug, Bremsflüssigkeit geprüft' },
  { id: 'mnt-8', vehicleId: 'veh-2', vehiclePlate: 'ZH 112 233', type: 'repair', date: '2026-02-05', mileage: 33800, cost: 290.00, notes: 'Steinschlag Windschutzscheibe repariert' },
]

const MOCK_FUEL_RECORDS: FuelRecord[] = [
  { id: 'fuel-1', vehicleId: 'veh-1', vehiclePlate: 'ZH 345 678', date: '2026-02-14', liters: 42.5, cost: 78.63, mileage: 18420 },
  { id: 'fuel-2', vehicleId: 'veh-2', vehiclePlate: 'ZH 112 233', date: '2026-02-13', liters: 38.2, cost: 70.67, mileage: 34560 },
  { id: 'fuel-3', vehicleId: 'veh-3', vehiclePlate: 'BE 456 789', date: '2026-02-14', liters: 65.0, cost: 123.50, mileage: 67890 },
  { id: 'fuel-4', vehicleId: 'veh-5', vehiclePlate: 'AG 223 344', date: '2026-02-12', liters: 35.8, cost: 66.23, mileage: 41200 },
  { id: 'fuel-5', vehicleId: 'veh-1', vehiclePlate: 'ZH 345 678', date: '2026-02-07', liters: 40.1, cost: 74.19, mileage: 17850 },
  { id: 'fuel-6', vehicleId: 'veh-3', vehiclePlate: 'BE 456 789', date: '2026-02-07', liters: 62.3, cost: 118.37, mileage: 67100 },
  { id: 'fuel-7', vehicleId: 'veh-2', vehiclePlate: 'ZH 112 233', date: '2026-02-05', liters: 36.9, cost: 68.27, mileage: 33800 },
  { id: 'fuel-8', vehicleId: 'veh-5', vehiclePlate: 'AG 223 344', date: '2026-02-04', liters: 33.5, cost: 61.98, mileage: 40600 },
  { id: 'fuel-9', vehicleId: 'veh-1', vehiclePlate: 'ZH 345 678', date: '2026-01-30', liters: 43.0, cost: 79.55, mileage: 17200 },
  { id: 'fuel-10', vehicleId: 'veh-3', vehiclePlate: 'BE 456 789', date: '2026-01-29', liters: 60.8, cost: 115.52, mileage: 66300 },
]

const MOCK_VEHICLE_ROUTES: VehicleRoute[] = [
  {
    id: 'route-1',
    vehicleId: 'veh-3',
    vehicleName: 'Mercedes-Benz Sprinter 314',
    date: '2026-02-15',
    dailyKm: 142,
    status: 'driving',
    driver: 'Thomas Berger',
    positions: [
      { lat: 47.3769, lng: 8.5417, address: 'Bahnhofstrasse 42, 8001 Zürich', timestamp: '2026-02-15T07:15:00' },
      { lat: 47.4245, lng: 8.6507, address: 'Industriestrasse 18, 8404 Winterthur', timestamp: '2026-02-15T08:05:00' },
      { lat: 47.4979, lng: 8.7271, address: 'Muensterplatz 1, 8200 Schaffhausen', timestamp: '2026-02-15T09:30:00' },
      { lat: 47.4508, lng: 8.6843, address: 'Zürich Flughafen, Cargo-Terminal', timestamp: '2026-02-15T11:20:00' },
      { lat: 47.3895, lng: 8.5185, address: 'Hardstrasse 201, 8005 Zürich', timestamp: '2026-02-15T12:45:00' },
    ],
  },
  {
    id: 'route-2',
    vehicleId: 'veh-1',
    vehicleName: 'VW Caddy Cargo',
    date: '2026-02-15',
    dailyKm: 95,
    status: 'parked',
    driver: 'Sarah Weber',
    positions: [
      { lat: 46.9480, lng: 7.4474, address: 'Bundesplatz 3, 3011 Bern', timestamp: '2026-02-15T07:45:00' },
      { lat: 46.7580, lng: 7.6280, address: 'Allmendstrasse 20, 3600 Thun', timestamp: '2026-02-15T08:40:00' },
      { lat: 46.6863, lng: 7.8632, address: 'Hoehematte, 3800 Interlaken', timestamp: '2026-02-15T10:15:00' },
      { lat: 46.6863, lng: 7.8632, address: 'Hoehematte, 3800 Interlaken (geparkt)', timestamp: '2026-02-15T10:20:00' },
    ],
  },
  {
    id: 'route-3',
    vehicleId: 'veh-5',
    vehicleName: 'Toyota Proace City',
    date: '2026-02-15',
    dailyKm: 118,
    status: 'driving',
    driver: 'Marco Fischer',
    positions: [
      { lat: 47.5596, lng: 7.5886, address: 'Steinenvorstadt 12, 4051 Basel', timestamp: '2026-02-15T06:50:00' },
      { lat: 47.4840, lng: 7.7302, address: 'Rheinstrasse 25, 4410 Liestal', timestamp: '2026-02-15T07:30:00' },
      { lat: 47.3925, lng: 8.0441, address: 'Bahnhofplatz 2, 5000 Aarau', timestamp: '2026-02-15T09:10:00' },
      { lat: 47.3521, lng: 7.9075, address: 'Hauptgasse 68, 4600 Olten', timestamp: '2026-02-15T10:45:00' },
      { lat: 47.3740, lng: 7.9570, address: 'Industriepark Olten-West', timestamp: '2026-02-15T11:55:00' },
    ],
  },
  {
    id: 'route-4',
    vehicleId: 'veh-4',
    vehicleName: 'Renault Kangoo E-Tech',
    date: '2026-02-15',
    dailyKm: 12,
    status: 'parked',
    driver: 'Anna Müller',
    positions: [
      { lat: 47.0502, lng: 8.3093, address: 'Pilatusstrasse 15, 6003 Luzern', timestamp: '2026-02-15T08:00:00' },
      { lat: 47.0378, lng: 8.3080, address: 'Bundesplatz 14, 6003 Luzern', timestamp: '2026-02-15T08:10:00' },
      { lat: 47.0378, lng: 8.3080, address: 'Bundesplatz 14, 6003 Luzern (geparkt seit 08:15)', timestamp: '2026-02-15T08:15:00' },
    ],
  },
  {
    id: 'route-5',
    vehicleId: 'veh-2',
    vehicleName: 'Skoda Octavia Combi',
    date: '2026-02-15',
    dailyKm: 87,
    status: 'driving',
    driver: 'David Keller',
    positions: [
      { lat: 47.4245, lng: 9.3767, address: 'Marktplatz 5, 9000 St. Gallen', timestamp: '2026-02-15T07:30:00' },
      { lat: 47.4489, lng: 9.2750, address: 'Gossau SG, Wilerstrasse 10', timestamp: '2026-02-15T08:15:00' },
      { lat: 47.5571, lng: 8.8981, address: 'Bahnhofstrasse 33, 8500 Frauenfeld', timestamp: '2026-02-15T09:45:00' },
      { lat: 47.6633, lng: 9.1756, address: 'Hussenstrasse 2, 78462 Konstanz (DE)', timestamp: '2026-02-15T11:10:00' },
    ],
  },
  {
    id: 'route-6',
    vehicleId: 'veh-6',
    vehicleName: 'Iveco Daily 35S14',
    date: '2026-02-14',
    dailyKm: 0,
    status: 'unknown',
    driver: '\u2014',
    positions: [
      { lat: 47.3769, lng: 8.5417, address: 'Lagerstrasse 104, 8004 Zürich (letzte Position)', timestamp: '2026-02-14T17:30:00' },
    ],
  },
]

// Wave 9 — Fahrtenbuch mock data
const MOCK_LOGBOOK: LogbookEntry[] = [
  { id: 'lb-1', vehicleId: 'veh-1', vehiclePlate: 'ZH 345 678', date: '2026-02-14', startLocation: 'Büro Zürich', endLocation: 'Kunde Meier AG, Winterthur', purpose: 'Kundenbesuch Angebot #2045', startKm: 18320, endKm: 18370, km: 50, isPrivate: false, driver: 'Thomas Keller' },
  { id: 'lb-2', vehicleId: 'veh-1', vehiclePlate: 'ZH 345 678', date: '2026-02-14', startLocation: 'Kunde Meier AG, Winterthur', endLocation: 'Büro Zürich', purpose: 'Rueckfahrt', startKm: 18370, endKm: 18420, km: 50, isPrivate: false, driver: 'Thomas Keller' },
  { id: 'lb-3', vehicleId: 'veh-2', vehiclePlate: 'ZH 112 233', date: '2026-02-13', startLocation: 'Wohnort', endLocation: 'Büro Zürich', purpose: 'Arbeitsweg', startKm: 34480, endKm: 34520, km: 40, isPrivate: true, driver: 'Lukas Brunner' },
  { id: 'lb-4', vehicleId: 'veh-2', vehiclePlate: 'ZH 112 233', date: '2026-02-13', startLocation: 'Büro Zürich', endLocation: 'Baustelle Altstetten', purpose: 'Baustellenbesichtigung', startKm: 34520, endKm: 34540, km: 20, isPrivate: false, driver: 'Lukas Brunner' },
  { id: 'lb-5', vehicleId: 'veh-3', vehiclePlate: 'BE 456 789', date: '2026-02-14', startLocation: 'Depot Bern', endLocation: 'Baustelle Thun', purpose: 'Materialtransport Baustahl', startKm: 67750, endKm: 67820, km: 70, isPrivate: false, driver: 'Reto Aeschlimann' },
  { id: 'lb-6', vehicleId: 'veh-3', vehiclePlate: 'BE 456 789', date: '2026-02-14', startLocation: 'Baustelle Thun', endLocation: 'Depot Bern', purpose: 'Rueckfahrt leer', startKm: 67820, endKm: 67890, km: 70, isPrivate: false, driver: 'Reto Aeschlimann' },
  { id: 'lb-7', vehicleId: 'veh-4', vehiclePlate: 'ZH 998 877', date: '2026-02-12', startLocation: 'Büro Luzern', endLocation: 'Messe Zürich', purpose: 'Messeaufbau', startKm: 5180, endKm: 5230, km: 50, isPrivate: false, driver: 'Sandra Müller' },
  { id: 'lb-8', vehicleId: 'veh-5', vehiclePlate: 'AG 223 344', date: '2026-02-13', startLocation: 'Büro Aarau', endLocation: 'Kunde Weber, Basel', purpose: 'Servicebesuch', startKm: 41100, endKm: 41200, km: 100, isPrivate: false, driver: 'Daniel Frei' },
]

// Wave 9 — Dokumente mock data
const MOCK_DOCUMENTS: VehicleDocument[] = [
  { id: 'doc-1', vehicleId: 'veh-1', type: 'registration', name: 'Fahrzeugausweis ZH 345 678', uploadDate: '2024-03-15' },
  { id: 'doc-2', vehicleId: 'veh-1', type: 'insurance', name: 'Vollkasko Zurich Vers.', uploadDate: '2025-01-01', expiryDate: '2027-01-01' },
  { id: 'doc-3', vehicleId: 'veh-2', type: 'registration', name: 'Fahrzeugausweis ZH 112 233', uploadDate: '2023-06-01' },
  { id: 'doc-4', vehicleId: 'veh-2', type: 'tuev', name: 'MFK-Bericht 2025', uploadDate: '2025-05-20', expiryDate: '2026-05-20' },
  { id: 'doc-5', vehicleId: 'veh-3', type: 'registration', name: 'Fahrzeugausweis BE 456 789', uploadDate: '2022-09-01' },
  { id: 'doc-6', vehicleId: 'veh-3', type: 'insurance', name: 'Haftpflicht AXA', uploadDate: '2025-01-01', expiryDate: '2027-01-01' },
  { id: 'doc-7', vehicleId: 'veh-3', type: 'tuev', name: 'MFK-Bericht 2025', uploadDate: '2025-03-10', expiryDate: '2026-03-10' },
  { id: 'doc-8', vehicleId: 'veh-4', type: 'registration', name: 'Fahrzeugausweis ZH 998 877', uploadDate: '2025-01-15' },
  { id: 'doc-9', vehicleId: 'veh-5', type: 'insurance', name: 'Vollkasko Mobiliar', uploadDate: '2023-07-01', expiryDate: '2027-01-01' },
]

// Wave 9 — Schadensmeldungen mock data
const MOCK_DAMAGE_REPORTS: DamageReport[] = [
  { id: 'dmg-1', vehicleId: 'veh-3', vehiclePlate: 'BE 456 789', date: '2026-01-22', description: 'Streifschaden rechte Seite beim Rangieren auf Baustelle', severity: 'moderate', location: 'Rechte Fahrzeugseite, hinteres Drittel', photoCount: 3, reportedBy: 'Reto Aeschlimann', status: 'in_progress' },
  { id: 'dmg-2', vehicleId: 'veh-2', vehiclePlate: 'ZH 112 233', date: '2026-02-05', description: 'Steinschlag Windschutzscheibe A1 Hoehe Baden', severity: 'minor', location: 'Windschutzscheibe links oben', photoCount: 1, reportedBy: 'Lukas Brunner', status: 'resolved' },
  { id: 'dmg-3', vehicleId: 'veh-5', vehiclePlate: 'AG 223 344', date: '2026-02-10', description: 'Delle Heckklappe — Parkplatz Einkaufszentrum', severity: 'minor', location: 'Heckklappe mittig', photoCount: 2, reportedBy: 'Daniel Frei', status: 'open' },
]

export const useFuhrparkStore = create<FuhrparkStore>()(
  persist(
    (_set) => ({
      vehicles: MOCK_VEHICLES,
      maintenanceRecords: MOCK_MAINTENANCE,
      fuelRecords: MOCK_FUEL_RECORDS,
      vehicleRoutes: MOCK_VEHICLE_ROUTES,
      logbookEntries: MOCK_LOGBOOK,
      vehicleDocuments: MOCK_DOCUMENTS,
      damageReports: MOCK_DAMAGE_REPORTS,
      refreshTracking: () => {
        toast.success('Tracking-Daten aktualisiert')
      },
    }),
    { name: 'kmuhub-fuhrpark' },
  ),
)
