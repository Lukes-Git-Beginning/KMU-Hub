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
  currency: string
  description?: string
  isActive: boolean
  lastMovement: string
  batchNumber?: string
  serialNumbers?: string[]
  linkedPurchaseOrder?: string
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

export interface InventurCount {
  itemId: string
  itemName: string
  sku: string
  expected: number
  counted: number | null
}

export interface InventurSession {
  id: string
  name: string
  date: string
  status: 'open' | 'counting' | 'review' | 'completed'
  locationId: string
  locationName: string
  counts: InventurCount[]
  createdBy: string
}

interface InventarStore {
  items: InventoryItem[]
  locations: InventoryLocation[]
  movements: InventoryMovement[]
  inventurSessions: InventurSession[]
}

const MOCK_LOCATIONS: InventoryLocation[] = [
  { id: 'loc-1', name: 'Hauptlager', address: 'Industriestrasse 12, 8005 Zürich', type: 'warehouse', itemCount: 142 },
  { id: 'loc-2', name: 'Filiale Bern', address: 'Marktgasse 5, 3011 Bern', type: 'store', itemCount: 67 },
  { id: 'loc-3', name: 'Lieferwagen #1', address: 'Mobil', type: 'vehicle', itemCount: 36 },
]

const MOCK_ITEMS: InventoryItem[] = [
  { id: 'inv-1', name: 'Kabelkanal 20x10mm', sku: 'KK-2010', category: 'Elektromaterial', unit: 'Meter', currentStock: 450, minStock: 100, locationId: 'loc-1', locationName: 'Hauptlager', barcode: '7610000000001', price: 2.50, currency: 'EUR', isActive: true, lastMovement: '2026-02-14', batchNumber: 'CH-2026-0142' },
  { id: 'inv-2', name: 'LED Panel 60x60cm', sku: 'LED-6060', category: 'Beleuchtung', unit: 'Stück', currentStock: 24, minStock: 10, locationId: 'loc-1', locationName: 'Hauptlager', barcode: '7610000000002', price: 89.90, currency: 'EUR', isActive: true, lastMovement: '2026-02-13', serialNumbers: ['LP-60-001', 'LP-60-002', 'LP-60-003'] },
  { id: 'inv-3', name: 'Steckdose T13', sku: 'SD-T13', category: 'Elektromaterial', unit: 'Stück', currentStock: 8, minStock: 50, locationId: 'loc-1', locationName: 'Hauptlager', barcode: '7610000000003', price: 4.20, currency: 'EUR', isActive: true, lastMovement: '2026-02-12', linkedPurchaseOrder: 'PO-2026-004' },
  { id: 'inv-4', name: 'Sicherung 16A', sku: 'SI-16A', category: 'Sicherungen', unit: 'Stück', currentStock: 120, minStock: 30, locationId: 'loc-1', locationName: 'Hauptlager', price: 3.80, currency: 'EUR', isActive: true, lastMovement: '2026-02-14', batchNumber: 'SI-2026-0088' },
  { id: 'inv-5', name: 'NYM-J 3x1.5mm²', sku: 'NYM-315', category: 'Kabel', unit: 'Meter', currentStock: 1200, minStock: 500, locationId: 'loc-1', locationName: 'Hauptlager', barcode: '7610000000005', price: 1.10, currency: 'EUR', isActive: true, lastMovement: '2026-02-14' },
  { id: 'inv-6', name: 'Verteilerdose AP', sku: 'VD-AP', category: 'Elektromaterial', unit: 'Stück', currentStock: 45, minStock: 20, locationId: 'loc-2', locationName: 'Filiale Bern', price: 6.50, currency: 'EUR', isActive: true, lastMovement: '2026-02-11' },
  { id: 'inv-7', name: 'Bewegungsmelder', sku: 'BM-180', category: 'Sensoren', unit: 'Stück', currentStock: 15, minStock: 5, locationId: 'loc-2', locationName: 'Filiale Bern', barcode: '7610000000007', price: 34.90, currency: 'EUR', isActive: true, lastMovement: '2026-02-10', serialNumbers: ['BM-180-A01', 'BM-180-A02'] },
  { id: 'inv-8', name: 'Thermostat digital', sku: 'TH-DIG', category: 'Heizung', unit: 'Stück', currentStock: 3, minStock: 5, locationId: 'loc-2', locationName: 'Filiale Bern', price: 79.00, currency: 'EUR', isActive: true, lastMovement: '2026-02-09', serialNumbers: ['TH-DIG-2025-001', 'TH-DIG-2025-002', 'TH-DIG-2025-003'] },
  { id: 'inv-9', name: 'FI-Schutzschalter 30mA', sku: 'FI-30', category: 'Sicherungen', unit: 'Stück', currentStock: 18, minStock: 10, locationId: 'loc-1', locationName: 'Hauptlager', price: 42.50, currency: 'EUR', isActive: true, lastMovement: '2026-02-14', batchNumber: 'FI-2026-0034' },
  { id: 'inv-10', name: 'Leerrohr 25mm', sku: 'LR-25', category: 'Rohre', unit: 'Meter', currentStock: 350, minStock: 100, locationId: 'loc-1', locationName: 'Hauptlager', price: 1.80, currency: 'EUR', isActive: true, lastMovement: '2026-02-13' },
  { id: 'inv-11', name: 'Schrauben-Set M4', sku: 'SS-M4', category: 'Befestigung', unit: 'Packung', currentStock: 67, minStock: 20, locationId: 'loc-3', locationName: 'Lieferwagen #1', price: 8.90, currency: 'EUR', isActive: true, lastMovement: '2026-02-12' },
  { id: 'inv-12', name: 'Rauchmelder EN 14604', sku: 'RM-EN14', category: 'Sicherheit', unit: 'Stück', currentStock: 28, minStock: 10, locationId: 'loc-1', locationName: 'Hauptlager', barcode: '7610000000012', price: 19.90, currency: 'EUR', isActive: true, lastMovement: '2026-02-11', batchNumber: 'RM-2026-0012', serialNumbers: ['RM-EN14-001', 'RM-EN14-002'] },
  { id: 'inv-13', name: 'Klebeband Isolier', sku: 'KB-ISO', category: 'Verbrauch', unit: 'Rolle', currentStock: 92, minStock: 30, locationId: 'loc-3', locationName: 'Lieferwagen #1', price: 3.20, currency: 'EUR', isActive: true, lastMovement: '2026-02-14' },
  { id: 'inv-14', name: 'Kabelschuh 6mm²', sku: 'KS-6', category: 'Kabelzubehör', unit: 'Stück', currentStock: 200, minStock: 100, locationId: 'loc-1', locationName: 'Hauptlager', price: 0.35, currency: 'EUR', isActive: true, lastMovement: '2026-02-13', batchNumber: 'KS-2026-0200' },
  { id: 'inv-15', name: 'Smart Switch Zigbee', sku: 'SS-ZB', category: 'Smart Home', unit: 'Stück', currentStock: 12, minStock: 5, locationId: 'loc-2', locationName: 'Filiale Bern', barcode: '7610000000015', price: 54.90, currency: 'CHF', isActive: true, lastMovement: '2026-02-10', serialNumbers: ['SSZ-2026-A01'] },
]

const MOCK_INVENTUR_SESSIONS: InventurSession[] = [
  {
    id: 'inv-s-1', name: 'Jahresinventur Hauptlager 2026', date: '2026-02-15', status: 'review',
    locationId: 'loc-1', locationName: 'Hauptlager', createdBy: 'Markus Weber',
    counts: [
      { itemId: 'inv-1', itemName: 'Kabelkanal 20x10mm', sku: 'KK-2010', expected: 450, counted: 448 },
      { itemId: 'inv-4', itemName: 'Sicherung 16A', sku: 'SI-16A', expected: 120, counted: 120 },
      { itemId: 'inv-5', itemName: 'NYM-J 3x1.5mm²', sku: 'NYM-315', expected: 1200, counted: 1185 },
      { itemId: 'inv-9', itemName: 'FI-Schutzschalter 30mA', sku: 'FI-30', expected: 18, counted: 18 },
      { itemId: 'inv-10', itemName: 'Leerrohr 25mm', sku: 'LR-25', expected: 350, counted: 352 },
      { itemId: 'inv-12', itemName: 'Rauchmelder EN 14604', sku: 'RM-EN14', expected: 28, counted: 26 },
      { itemId: 'inv-14', itemName: 'Kabelschuh 6mm²', sku: 'KS-6', expected: 200, counted: 198 },
      { itemId: 'inv-3', itemName: 'Steckdose T13', sku: 'SD-T13', expected: 8, counted: 8 },
    ],
  },
  {
    id: 'inv-s-2', name: 'Stichprobe Filiale Bern', date: '2026-01-20', status: 'completed',
    locationId: 'loc-2', locationName: 'Filiale Bern', createdBy: 'Sarah Mueller',
    counts: [
      { itemId: 'inv-6', itemName: 'Verteilerdose AP', sku: 'VD-AP', expected: 45, counted: 45 },
      { itemId: 'inv-7', itemName: 'Bewegungsmelder', sku: 'BM-180', expected: 15, counted: 15 },
      { itemId: 'inv-8', itemName: 'Thermostat digital', sku: 'TH-DIG', expected: 5, counted: 5 },
      { itemId: 'inv-15', itemName: 'Smart Switch Zigbee', sku: 'SS-ZB', expected: 12, counted: 12 },
    ],
  },
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
      inventurSessions: MOCK_INVENTUR_SESSIONS,
    }),
    { name: 'kmuhub-inventar' },
  ),
)
