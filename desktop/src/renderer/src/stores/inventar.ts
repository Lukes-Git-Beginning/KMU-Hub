import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface InventoryItem {
  id: string
  name: string
  sku: string
  category: string
  unit: string
  currentStock: number
  minStock: number
  locationId: string
  locationName: string
  barcode?: string
  price: number
  description?: string
  isActive: boolean
  lastMovement: string
}

export interface InventoryLocation {
  id: string
  name: string
  address: string
  type: 'warehouse' | 'store' | 'vehicle'
  itemCount: number
}

export interface InventoryMovement {
  id: string
  itemId: string
  itemName: string
  type: 'in' | 'out' | 'transfer' | 'adjustment'
  quantity: number
  locationFrom?: string
  locationTo?: string
  reference: string
  notes?: string
  createdBy: string
  createdAt: string
}

interface InventarStore {
  items: InventoryItem[]
  locations: InventoryLocation[]
  movements: InventoryMovement[]
}

const MOCK_LOCATIONS: InventoryLocation[] = [
  { id: 'loc-1', name: 'Hauptlager', address: 'Industriestrasse 12, 8005 Zürich', type: 'warehouse', itemCount: 142 },
  { id: 'loc-2', name: 'Filiale Bern', address: 'Marktgasse 5, 3011 Bern', type: 'store', itemCount: 67 },
  { id: 'loc-3', name: 'Lieferwagen #1', address: 'Mobil', type: 'vehicle', itemCount: 36 },
]

const MOCK_ITEMS: InventoryItem[] = [
  { id: 'inv-1', name: 'Kabelkanal 20x10mm', sku: 'KK-2010', category: 'Elektromaterial', unit: 'Meter', currentStock: 450, minStock: 100, locationId: 'loc-1', locationName: 'Hauptlager', barcode: '7610000000001', price: 2.50, isActive: true, lastMovement: '2026-02-14' },
  { id: 'inv-2', name: 'LED Panel 60x60cm', sku: 'LED-6060', category: 'Beleuchtung', unit: 'Stück', currentStock: 24, minStock: 10, locationId: 'loc-1', locationName: 'Hauptlager', barcode: '7610000000002', price: 89.90, isActive: true, lastMovement: '2026-02-13' },
  { id: 'inv-3', name: 'Steckdose T13', sku: 'SD-T13', category: 'Elektromaterial', unit: 'Stück', currentStock: 8, minStock: 50, locationId: 'loc-1', locationName: 'Hauptlager', barcode: '7610000000003', price: 4.20, isActive: true, lastMovement: '2026-02-12' },
  { id: 'inv-4', name: 'Sicherung 16A', sku: 'SI-16A', category: 'Sicherungen', unit: 'Stück', currentStock: 120, minStock: 30, locationId: 'loc-1', locationName: 'Hauptlager', price: 3.80, isActive: true, lastMovement: '2026-02-14' },
  { id: 'inv-5', name: 'NYM-J 3x1.5mm²', sku: 'NYM-315', category: 'Kabel', unit: 'Meter', currentStock: 1200, minStock: 500, locationId: 'loc-1', locationName: 'Hauptlager', barcode: '7610000000005', price: 1.10, isActive: true, lastMovement: '2026-02-14' },
  { id: 'inv-6', name: 'Verteilerdose AP', sku: 'VD-AP', category: 'Elektromaterial', unit: 'Stück', currentStock: 45, minStock: 20, locationId: 'loc-2', locationName: 'Filiale Bern', price: 6.50, isActive: true, lastMovement: '2026-02-11' },
  { id: 'inv-7', name: 'Bewegungsmelder', sku: 'BM-180', category: 'Sensoren', unit: 'Stück', currentStock: 15, minStock: 5, locationId: 'loc-2', locationName: 'Filiale Bern', barcode: '7610000000007', price: 34.90, isActive: true, lastMovement: '2026-02-10' },
  { id: 'inv-8', name: 'Thermostat digital', sku: 'TH-DIG', category: 'Heizung', unit: 'Stück', currentStock: 3, minStock: 5, locationId: 'loc-2', locationName: 'Filiale Bern', price: 79.00, isActive: true, lastMovement: '2026-02-09' },
  { id: 'inv-9', name: 'FI-Schutzschalter 30mA', sku: 'FI-30', category: 'Sicherungen', unit: 'Stück', currentStock: 18, minStock: 10, locationId: 'loc-1', locationName: 'Hauptlager', price: 42.50, isActive: true, lastMovement: '2026-02-14' },
  { id: 'inv-10', name: 'Leerrohr 25mm', sku: 'LR-25', category: 'Rohre', unit: 'Meter', currentStock: 350, minStock: 100, locationId: 'loc-1', locationName: 'Hauptlager', price: 1.80, isActive: true, lastMovement: '2026-02-13' },
  { id: 'inv-11', name: 'Schrauben-Set M4', sku: 'SS-M4', category: 'Befestigung', unit: 'Packung', currentStock: 67, minStock: 20, locationId: 'loc-3', locationName: 'Lieferwagen #1', price: 8.90, isActive: true, lastMovement: '2026-02-12' },
  { id: 'inv-12', name: 'Rauchmelder EN 14604', sku: 'RM-EN14', category: 'Sicherheit', unit: 'Stück', currentStock: 28, minStock: 10, locationId: 'loc-1', locationName: 'Hauptlager', barcode: '7610000000012', price: 19.90, isActive: true, lastMovement: '2026-02-11' },
  { id: 'inv-13', name: 'Klebeband Isolier', sku: 'KB-ISO', category: 'Verbrauch', unit: 'Rolle', currentStock: 92, minStock: 30, locationId: 'loc-3', locationName: 'Lieferwagen #1', price: 3.20, isActive: true, lastMovement: '2026-02-14' },
  { id: 'inv-14', name: 'Kabelschuh 6mm²', sku: 'KS-6', category: 'Kabelzubehör', unit: 'Stück', currentStock: 200, minStock: 100, locationId: 'loc-1', locationName: 'Hauptlager', price: 0.35, isActive: true, lastMovement: '2026-02-13' },
  { id: 'inv-15', name: 'Smart Switch Zigbee', sku: 'SS-ZB', category: 'Smart Home', unit: 'Stück', currentStock: 12, minStock: 5, locationId: 'loc-2', locationName: 'Filiale Bern', barcode: '7610000000015', price: 54.90, isActive: true, lastMovement: '2026-02-10' },
]

const MOCK_MOVEMENTS: InventoryMovement[] = [
  { id: 'mov-1', itemId: 'inv-1', itemName: 'Kabelkanal 20x10mm', type: 'in', quantity: 200, locationTo: 'Hauptlager', reference: 'PO-2024-031', createdBy: 'Markus Weber', createdAt: '2026-02-14T09:30:00' },
  { id: 'mov-2', itemId: 'inv-3', itemName: 'Steckdose T13', type: 'out', quantity: 12, locationFrom: 'Hauptlager', reference: 'Auftrag #A-445', createdBy: 'Lukas Brunner', createdAt: '2026-02-14T08:15:00' },
  { id: 'mov-3', itemId: 'inv-5', itemName: 'NYM-J 3x1.5mm²', type: 'transfer', quantity: 100, locationFrom: 'Hauptlager', locationTo: 'Lieferwagen #1', reference: 'Transfer #T-12', createdBy: 'Thomas Keller', createdAt: '2026-02-13T16:45:00' },
  { id: 'mov-4', itemId: 'inv-8', itemName: 'Thermostat digital', type: 'out', quantity: 2, locationFrom: 'Filiale Bern', reference: 'Auftrag #A-443', createdBy: 'Sarah Mueller', createdAt: '2026-02-13T14:20:00' },
  { id: 'mov-5', itemId: 'inv-2', itemName: 'LED Panel 60x60cm', type: 'in', quantity: 10, locationTo: 'Hauptlager', reference: 'PO-2024-030', createdBy: 'Markus Weber', createdAt: '2026-02-13T10:00:00' },
  { id: 'mov-6', itemId: 'inv-12', itemName: 'Rauchmelder EN 14604', type: 'adjustment', quantity: -2, locationFrom: 'Hauptlager', reference: 'Inventur-Korrektur', notes: 'Defekte Ware aussortiert', createdBy: 'Nina Fischer', createdAt: '2026-02-12T11:30:00' },
]

export const useInventarStore = create<InventarStore>()(
  persist(
    () => ({
      items: MOCK_ITEMS,
      locations: MOCK_LOCATIONS,
      movements: MOCK_MOVEMENTS,
    }),
    { name: 'kmuhub-inventar' },
  ),
)
