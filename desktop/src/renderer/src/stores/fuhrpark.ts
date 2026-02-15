import { create } from 'zustand'
import { persist } from 'zustand/middleware'

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

interface FuhrparkStore {
  vehicles: Vehicle[]
  maintenanceRecords: MaintenanceRecord[]
  fuelRecords: FuelRecord[]
}

const MOCK_VEHICLES: Vehicle[] = [
  { id: 'veh-1', licensePlate: 'ZH 345 678', make: 'VW', model: 'Caddy Cargo', year: 2024, type: 'van', currentDriver: 'Thomas Keller', mileage: 18420, nextInspection: '2026-08-15', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-2', licensePlate: 'ZH 112 233', make: 'Skoda', model: 'Octavia Combi', year: 2023, type: 'car', currentDriver: 'Lukas Brunner', mileage: 34560, nextInspection: '2026-05-20', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-3', licensePlate: 'BE 456 789', make: 'Mercedes-Benz', model: 'Sprinter 314', year: 2022, type: 'truck', currentDriver: 'Reto Aeschlimann', mileage: 67890, nextInspection: '2026-03-10', insuranceExpiry: '2027-01-01', isActive: true },
  { id: 'veh-4', licensePlate: 'ZH 998 877', make: 'Renault', model: 'Kangoo E-Tech', year: 2025, type: 'van', currentDriver: 'Sandra Mueller', mileage: 5230, nextInspection: '2027-02-01', insuranceExpiry: '2027-01-01', isActive: true },
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

export const useFuhrparkStore = create<FuhrparkStore>()(
  persist(
    () => ({
      vehicles: MOCK_VEHICLES,
      maintenanceRecords: MOCK_MAINTENANCE,
      fuelRecords: MOCK_FUEL_RECORDS,
    }),
    { name: 'kmuhub-fuhrpark' },
  ),
)
