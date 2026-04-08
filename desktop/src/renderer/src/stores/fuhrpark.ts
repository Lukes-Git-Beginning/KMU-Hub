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
  { id: 'veh-1', licensePlate: 'M-AB 3456', make: 'VW', model: 'Caddy Cargo', year: 2024, type: 'van', currentDriver: 'Thomas Keller', mileage: 18420, nextInspection: '2026-08-15', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-2', licensePlate: 'M-CD 1122', make: 'Skoda', model: 'Octavia Combi', year: 2023, type: 'car', currentDriver: 'Lukas Brunner', mileage: 34560, nextInspection: '2026-05-20', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-3', licensePlate: 'B-EF 4567', make: 'Mercedes-Benz', model: 'Sprinter 314', year: 2022, type: 'truck', currentDriver: 'Reto Aeschlimann', mileage: 67890, nextInspection: '2026-03-10', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-4', licensePlate: 'M-GH 9988', make: 'Renault', model: 'Kangoo E-Tech', year: 2025, type: 'van', currentDriver: 'Sandra Müller', mileage: 5230, nextInspection: '2027-02-01', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-5', licensePlate: 'HH-JK 2233', make: 'Toyota', model: 'Proace City', year: 2023, type: 'van', currentDriver: 'Daniel Frei', mileage: 41200, nextInspection: '2026-06-30', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-6', licensePlate: 'M-LM 6677', make: 'Iveco', model: 'Daily 35S14', year: 2021, type: 'truck', currentDriver: '', mileage: 89450, nextInspection: '2026-04-15', insuranceExpiry: '2027-01-01', isActive: false },
]

const MOCK_MAINTENANCE: MaintenanceRecord[] = [
  { id: 'mnt-1', vehicleId: 'veh-1', vehiclePlate: 'M-AB 3456', type: 'service', date: '2026-01-20', mileage: 15000, cost: 480.00, notes: 'Jährlicher Service, Ölwechsel, Filterwechsel' },
  { id: 'mnt-2', vehicleId: 'veh-2', vehiclePlate: 'M-CD 1122', type: 'tires', date: '2025-11-05', mileage: 30200, cost: 1120.00, notes: 'Winterreifen montiert, 4x Continental WinterContact TS 870' },
  { id: 'mnt-3', vehicleId: 'veh-3', vehiclePlate: 'B-EF 4567', type: 'repair', date: '2026-01-08', mileage: 65400, cost: 2340.00, notes: 'Bremsen vorne komplett erneuert, Bremsscheiben + Beläge' },
  { id: 'mnt-4', vehicleId: 'veh-3', vehiclePlate: 'B-EF 4567', type: 'inspection', date: '2025-03-10', mileage: 52000, cost: 180.00, notes: 'HU/AU bestanden ohne Mängel' },
  { id: 'mnt-5', vehicleId: 'veh-5', vehiclePlate: 'HH-JK 2233', type: 'service', date: '2026-02-01', mileage: 40000, cost: 520.00, notes: '40.000 km Service gemäß Herstellervorgabe' },
  { id: 'mnt-6', vehicleId: 'veh-6', vehiclePlate: 'M-LM 6677', type: 'repair', date: '2025-12-15', mileage: 88000, cost: 3800.00, notes: 'Getriebeschaden, Fahrzeug stillgelegt bis Reparaturentscheid' },
  { id: 'mnt-7', vehicleId: 'veh-4', vehiclePlate: 'M-GH 9988', type: 'service', date: '2026-02-10', mileage: 5000, cost: 150.00, notes: 'Erster Service Elektrofahrzeug, Bremsflüssigkeit geprüft' },
  { id: 'mnt-8', vehicleId: 'veh-2', vehiclePlate: 'M-CD 1122', type: 'repair', date: '2026-02-05', mileage: 33800, cost: 290.00, notes: 'Steinschlag Windschutzscheibe repariert' },
]

const MOCK_FUEL_RECORDS: FuelRecord[] = [
  { id: 'fuel-1', vehicleId: 'veh-1', vehiclePlate: 'M-AB 3456', date: '2026-02-14', liters: 42.5, cost: 78.63, mileage: 18420 },
  { id: 'fuel-2', vehicleId: 'veh-2', vehiclePlate: 'M-CD 1122', date: '2026-02-13', liters: 38.2, cost: 70.67, mileage: 34560 },
  { id: 'fuel-3', vehicleId: 'veh-3', vehiclePlate: 'B-EF 4567', date: '2026-02-14', liters: 65.0, cost: 123.50, mileage: 67890 },
  { id: 'fuel-4', vehicleId: 'veh-5', vehiclePlate: 'HH-JK 2233', date: '2026-02-12', liters: 35.8, cost: 66.23, mileage: 41200 },
  { id: 'fuel-5', vehicleId: 'veh-1', vehiclePlate: 'M-AB 3456', date: '2026-02-07', liters: 40.1, cost: 74.19, mileage: 17850 },
  { id: 'fuel-6', vehicleId: 'veh-3', vehiclePlate: 'B-EF 4567', date: '2026-02-07', liters: 62.3, cost: 118.37, mileage: 67100 },
  { id: 'fuel-7', vehicleId: 'veh-2', vehiclePlate: 'M-CD 1122', date: '2026-02-05', liters: 36.9, cost: 68.27, mileage: 33800 },
  { id: 'fuel-8', vehicleId: 'veh-5', vehiclePlate: 'HH-JK 2233', date: '2026-02-04', liters: 33.5, cost: 61.98, mileage: 40600 },
  { id: 'fuel-9', vehicleId: 'veh-1', vehiclePlate: 'M-AB 3456', date: '2026-01-30', liters: 43.0, cost: 79.55, mileage: 17200 },
  { id: 'fuel-10', vehicleId: 'veh-3', vehiclePlate: 'B-EF 4567', date: '2026-01-29', liters: 60.8, cost: 115.52, mileage: 66300 },
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
      { lat: 52.5200, lng: 13.4050, address: 'Friedrichstraße 42, 10117 Berlin', timestamp: '2026-02-15T07:15:00' },
      { lat: 52.3906, lng: 13.0645, address: 'Industriestraße 18, 14473 Potsdam', timestamp: '2026-02-15T08:05:00' },
      { lat: 52.1300, lng: 11.6167, address: 'Domplatz 1, 39104 Magdeburg', timestamp: '2026-02-15T09:30:00' },
      { lat: 52.3667, lng: 13.5033, address: 'BER Flughafen, Cargo-Terminal', timestamp: '2026-02-15T11:20:00' },
      { lat: 52.5069, lng: 13.2846, address: 'Kantstraße 201, 10623 Berlin', timestamp: '2026-02-15T12:45:00' },
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
      { lat: 48.1351, lng: 11.5820, address: 'Marienplatz 1, 80331 München', timestamp: '2026-02-15T07:45:00' },
      { lat: 48.2603, lng: 11.6681, address: 'Schleißheimer Str. 20, 85748 Garching', timestamp: '2026-02-15T08:40:00' },
      { lat: 47.8749, lng: 11.7280, address: 'Marktstraße 5, 83607 Holzkirchen', timestamp: '2026-02-15T10:15:00' },
      { lat: 47.8749, lng: 11.7280, address: 'Marktstraße 5, 83607 Holzkirchen (geparkt)', timestamp: '2026-02-15T10:20:00' },
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
      { lat: 53.5511, lng: 9.9937, address: 'Jungfernstieg 12, 20354 Hamburg', timestamp: '2026-02-15T06:50:00' },
      { lat: 53.4653, lng: 9.9688, address: 'Harburger Ring 25, 21073 Hamburg', timestamp: '2026-02-15T07:30:00' },
      { lat: 53.5544, lng: 10.0077, address: 'Hauptbahnhof 2, 20099 Hamburg', timestamp: '2026-02-15T09:10:00' },
      { lat: 53.5877, lng: 10.0510, address: 'Wandsbeker Marktstraße 68, 22041 Hamburg', timestamp: '2026-02-15T10:45:00' },
      { lat: 53.5311, lng: 10.0183, address: 'Industriepark Billbrook', timestamp: '2026-02-15T11:55:00' },
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
      { lat: 48.1351, lng: 11.5820, address: 'Sendlinger Straße 15, 80331 München', timestamp: '2026-02-15T08:00:00' },
      { lat: 48.1392, lng: 11.5781, address: 'Karlsplatz 14, 80335 München', timestamp: '2026-02-15T08:10:00' },
      { lat: 48.1392, lng: 11.5781, address: 'Karlsplatz 14, 80335 München (geparkt seit 08:15)', timestamp: '2026-02-15T08:15:00' },
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
      { lat: 48.3705, lng: 10.8978, address: 'Rathausplatz 5, 86150 Augsburg', timestamp: '2026-02-15T07:30:00' },
      { lat: 48.4011, lng: 10.9482, address: 'Berliner Allee 10, 86153 Augsburg', timestamp: '2026-02-15T08:15:00' },
      { lat: 48.7258, lng: 11.4350, address: 'Theresienstraße 33, 85049 Ingolstadt', timestamp: '2026-02-15T09:45:00' },
      { lat: 48.7665, lng: 11.4258, address: 'Manchinger Straße 2, 85053 Ingolstadt', timestamp: '2026-02-15T11:10:00' },
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
      { lat: 48.1351, lng: 11.5820, address: 'Lagerstraße 104, 80335 München (letzte Position)', timestamp: '2026-02-14T17:30:00' },
    ],
  },
]

// Wave 9 — Fahrtenbuch mock data
const MOCK_LOGBOOK: LogbookEntry[] = [
  { id: 'lb-1', vehicleId: 'veh-1', vehiclePlate: 'M-AB 3456', date: '2026-02-14', startLocation: 'Büro München', endLocation: 'Kunde Meier AG, Augsburg', purpose: 'Kundenbesuch Angebot #2045', startKm: 18320, endKm: 18370, km: 50, isPrivate: false, driver: 'Thomas Keller' },
  { id: 'lb-2', vehicleId: 'veh-1', vehiclePlate: 'M-AB 3456', date: '2026-02-14', startLocation: 'Kunde Meier AG, Augsburg', endLocation: 'Büro München', purpose: 'Rückfahrt', startKm: 18370, endKm: 18420, km: 50, isPrivate: false, driver: 'Thomas Keller' },
  { id: 'lb-3', vehicleId: 'veh-2', vehiclePlate: 'M-CD 1122', date: '2026-02-13', startLocation: 'Wohnort', endLocation: 'Büro München', purpose: 'Arbeitsweg', startKm: 34480, endKm: 34520, km: 40, isPrivate: true, driver: 'Lukas Brunner' },
  { id: 'lb-4', vehicleId: 'veh-2', vehiclePlate: 'M-CD 1122', date: '2026-02-13', startLocation: 'Büro München', endLocation: 'Baustelle Pasing', purpose: 'Baustellenbesichtigung', startKm: 34520, endKm: 34540, km: 20, isPrivate: false, driver: 'Lukas Brunner' },
  { id: 'lb-5', vehicleId: 'veh-3', vehiclePlate: 'B-EF 4567', date: '2026-02-14', startLocation: 'Depot Berlin', endLocation: 'Baustelle Potsdam', purpose: 'Materialtransport Baustahl', startKm: 67750, endKm: 67820, km: 70, isPrivate: false, driver: 'Reto Aeschlimann' },
  { id: 'lb-6', vehicleId: 'veh-3', vehiclePlate: 'B-EF 4567', date: '2026-02-14', startLocation: 'Baustelle Potsdam', endLocation: 'Depot Berlin', purpose: 'Rückfahrt leer', startKm: 67820, endKm: 67890, km: 70, isPrivate: false, driver: 'Reto Aeschlimann' },
  { id: 'lb-7', vehicleId: 'veh-4', vehiclePlate: 'M-GH 9988', date: '2026-02-12', startLocation: 'Büro München', endLocation: 'Messe München', purpose: 'Messeaufbau', startKm: 5180, endKm: 5230, km: 50, isPrivate: false, driver: 'Sandra Müller' },
  { id: 'lb-8', vehicleId: 'veh-5', vehiclePlate: 'HH-JK 2233', date: '2026-02-13', startLocation: 'Büro Hamburg', endLocation: 'Kunde Weber, Bremen', purpose: 'Servicebesuch', startKm: 41100, endKm: 41200, km: 100, isPrivate: false, driver: 'Daniel Frei' },
]

// Wave 9 — Dokumente mock data
const MOCK_DOCUMENTS: VehicleDocument[] = [
  { id: 'doc-1', vehicleId: 'veh-1', type: 'registration', name: 'Fahrzeugschein M-AB 3456', uploadDate: '2024-03-15' },
  { id: 'doc-2', vehicleId: 'veh-1', type: 'insurance', name: 'Vollkasko Allianz', uploadDate: '2025-01-01', expiryDate: '2027-01-01' },
  { id: 'doc-3', vehicleId: 'veh-2', type: 'registration', name: 'Fahrzeugschein M-CD 1122', uploadDate: '2023-06-01' },
  { id: 'doc-4', vehicleId: 'veh-2', type: 'tuev', name: 'TÜV-Bericht 2025', uploadDate: '2025-05-20', expiryDate: '2026-05-20' },
  { id: 'doc-5', vehicleId: 'veh-3', type: 'registration', name: 'Fahrzeugschein B-EF 4567', uploadDate: '2022-09-01' },
  { id: 'doc-6', vehicleId: 'veh-3', type: 'insurance', name: 'Haftpflicht AXA', uploadDate: '2025-01-01', expiryDate: '2027-01-01' },
  { id: 'doc-7', vehicleId: 'veh-3', type: 'tuev', name: 'TÜV-Bericht 2025', uploadDate: '2025-03-10', expiryDate: '2026-03-10' },
  { id: 'doc-8', vehicleId: 'veh-4', type: 'registration', name: 'Fahrzeugschein M-GH 9988', uploadDate: '2025-01-15' },
  { id: 'doc-9', vehicleId: 'veh-5', type: 'insurance', name: 'Vollkasko HUK-Coburg', uploadDate: '2023-07-01', expiryDate: '2027-01-01' },
]

// Wave 9 — Schadensmeldungen mock data
const MOCK_DAMAGE_REPORTS: DamageReport[] = [
  { id: 'dmg-1', vehicleId: 'veh-3', vehiclePlate: 'B-EF 4567', date: '2026-01-22', description: 'Streifschaden rechte Seite beim Rangieren auf Baustelle', severity: 'moderate', location: 'Rechte Fahrzeugseite, hinteres Drittel', photoCount: 3, reportedBy: 'Reto Aeschlimann', status: 'in_progress' },
  { id: 'dmg-2', vehicleId: 'veh-2', vehiclePlate: 'M-CD 1122', date: '2026-02-05', description: 'Steinschlag Windschutzscheibe A8 Höhe Rosenheim', severity: 'minor', location: 'Windschutzscheibe links oben', photoCount: 1, reportedBy: 'Lukas Brunner', status: 'resolved' },
  { id: 'dmg-3', vehicleId: 'veh-5', vehiclePlate: 'HH-JK 2233', date: '2026-02-10', description: 'Delle Heckklappe — Parkplatz Einkaufszentrum', severity: 'minor', location: 'Heckklappe mittig', photoCount: 2, reportedBy: 'Daniel Frei', status: 'open' },
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
    { name: 'cosmi-fuhrpark' },
  ),
)
