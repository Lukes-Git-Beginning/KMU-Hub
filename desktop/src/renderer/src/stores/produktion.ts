import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface BomItem {
  materialName: string
  quantity: number
  unit: string
}

export interface BOM {
  id: string
  productName: string
  sku: string
  version: string
  active: boolean
  items: BomItem[]
}

export interface ProductionOrder {
  id: string
  orderNr: string
  bomId: string
  product: string
  quantity: number
  status: 'planned' | 'in_progress' | 'paused' | 'completed' | 'cancelled'
  progress: number
  startDate: string
  dueDate: string
}

export interface QualityCheck {
  id: string
  orderId: string
  orderNr: string
  inspector: string
  date: string
  passed: boolean
  defectsFound: number
  notes: string
}

interface ProduktionStore {
  orders: ProductionOrder[]
  boms: BOM[]
  qualityChecks: QualityCheck[]
}

const MOCK_BOMS: BOM[] = [
  {
    id: 'bom-1', productName: 'Schaltschrank Standard 600mm', sku: 'SS-600', version: '3.2', active: true,
    items: [
      { materialName: 'Stahlgehäuse 600x400x200', quantity: 1, unit: 'Stk' },
      { materialName: 'Hauptschalter 63A', quantity: 1, unit: 'Stk' },
      { materialName: 'Sicherungsautomat B16', quantity: 8, unit: 'Stk' },
      { materialName: 'FI-Schutzschalter 40A/30mA', quantity: 2, unit: 'Stk' },
      { materialName: 'Hutschiene TS35 1m', quantity: 3, unit: 'Stk' },
      { materialName: 'Kabelkanal 40x60mm', quantity: 4, unit: 'm' },
      { materialName: 'Klemmleiste PE/N', quantity: 2, unit: 'Stk' },
      { materialName: 'Verdrahtung H07V-K 1.5mm²', quantity: 25, unit: 'm' },
    ],
  },
  {
    id: 'bom-2', productName: 'Steuerungspanel Touch 10"', sku: 'SP-T10', version: '2.1', active: true,
    items: [
      { materialName: 'Touch-Display 10" IPS', quantity: 1, unit: 'Stk' },
      { materialName: 'Raspberry Pi CM4 8GB', quantity: 1, unit: 'Stk' },
      { materialName: 'Aluminiumgehäuse IP54', quantity: 1, unit: 'Stk' },
      { materialName: 'Netzteil 24V/5A', quantity: 1, unit: 'Stk' },
      { materialName: 'Industrial Ethernet Switch', quantity: 1, unit: 'Stk' },
      { materialName: 'Flachbandkabel 40-pol', quantity: 0.5, unit: 'm' },
    ],
  },
  {
    id: 'bom-3', productName: 'Kabelsatz Industrieanlage A', sku: 'KS-IND-A', version: '1.4', active: true,
    items: [
      { materialName: 'Kabel NYM-J 5x2.5mm²', quantity: 50, unit: 'm' },
      { materialName: 'Kabel NYM-J 3x1.5mm²', quantity: 30, unit: 'm' },
      { materialName: 'Steckverbinder M12 4-pol', quantity: 12, unit: 'Stk' },
      { materialName: 'Kabelverschraubung M20', quantity: 24, unit: 'Stk' },
      { materialName: 'Schrumpfschlauch Set', quantity: 1, unit: 'Set' },
    ],
  },
  {
    id: 'bom-4', productName: 'LED-Leuchte IP65 Outdoor', sku: 'LED-IP65', version: '4.0', active: true,
    items: [
      { materialName: 'LED-Modul 50W 5000K', quantity: 1, unit: 'Stk' },
      { materialName: 'Treiber LED 50W dimmbar', quantity: 1, unit: 'Stk' },
      { materialName: 'Gehäuse Druckguss IP65', quantity: 1, unit: 'Stk' },
      { materialName: 'Optik 60° Linse', quantity: 1, unit: 'Stk' },
    ],
  },
  {
    id: 'bom-5', productName: 'Sensormodul Temperatur/Feuchte', sku: 'SM-TF-01', version: '1.0', active: false,
    items: [
      { materialName: 'Sensor SHT40', quantity: 1, unit: 'Stk' },
      { materialName: 'PCB Custom 30x20mm', quantity: 1, unit: 'Stk' },
      { materialName: 'ESP32-C3 Modul', quantity: 1, unit: 'Stk' },
    ],
  },
]

const MOCK_ORDERS: ProductionOrder[] = [
  { id: 'prod-1', orderNr: 'PRD-2026-041', bomId: 'bom-1', product: 'Schaltschrank Standard 600mm', quantity: 10, status: 'in_progress', progress: 70, startDate: '2026-02-03', dueDate: '2026-02-17' },
  { id: 'prod-2', orderNr: 'PRD-2026-042', bomId: 'bom-2', product: 'Steuerungspanel Touch 10"', quantity: 25, status: 'in_progress', progress: 40, startDate: '2026-02-10', dueDate: '2026-02-24' },
  { id: 'prod-3', orderNr: 'PRD-2026-043', bomId: 'bom-3', product: 'Kabelsatz Industrieanlage A', quantity: 50, status: 'completed', progress: 100, startDate: '2026-01-20', dueDate: '2026-02-07' },
  { id: 'prod-4', orderNr: 'PRD-2026-044', bomId: 'bom-4', product: 'LED-Leuchte IP65 Outdoor', quantity: 200, status: 'planned', progress: 0, startDate: '2026-02-18', dueDate: '2026-03-07' },
  { id: 'prod-5', orderNr: 'PRD-2026-045', bomId: 'bom-1', product: 'Schaltschrank Standard 600mm', quantity: 5, status: 'paused', progress: 20, startDate: '2026-02-05', dueDate: '2026-02-20' },
  { id: 'prod-6', orderNr: 'PRD-2026-046', bomId: 'bom-3', product: 'Kabelsatz Industrieanlage A', quantity: 30, status: 'planned', progress: 0, startDate: '2026-02-20', dueDate: '2026-03-10' },
  { id: 'prod-7', orderNr: 'PRD-2026-047', bomId: 'bom-2', product: 'Steuerungspanel Touch 10"', quantity: 15, status: 'in_progress', progress: 85, startDate: '2026-01-27', dueDate: '2026-02-16' },
  { id: 'prod-8', orderNr: 'PRD-2026-048', bomId: 'bom-4', product: 'LED-Leuchte IP65 Outdoor', quantity: 100, status: 'cancelled', progress: 10, startDate: '2026-02-01', dueDate: '2026-02-15' },
]

const MOCK_QUALITY_CHECKS: QualityCheck[] = [
  { id: 'qc-1', orderId: 'prod-1', orderNr: 'PRD-2026-041', inspector: 'Werner Stöckli', date: '2026-02-12', passed: true, defectsFound: 0, notes: 'Stichprobe 3 von 7 Schränken geprüft, alle i.O.' },
  { id: 'qc-2', orderId: 'prod-3', orderNr: 'PRD-2026-043', inspector: 'Werner Stöckli', date: '2026-02-07', passed: true, defectsFound: 1, notes: '1 Kabelsatz mit falschem Steckertyp, nachgearbeitet' },
  { id: 'qc-3', orderId: 'prod-2', orderNr: 'PRD-2026-042', inspector: 'Irene Graf', date: '2026-02-14', passed: false, defectsFound: 3, notes: 'Touchscreen-Kalibrierung bei 3 Panels außerhalb Toleranz. Nachkalibrierung erforderlich.' },
  { id: 'qc-4', orderId: 'prod-7', orderNr: 'PRD-2026-047', inspector: 'Irene Graf', date: '2026-02-13', passed: true, defectsFound: 0, notes: 'Endkontrolle 12 von 15 Panels. Display, Touch, Kommunikation alles i.O.' },
  { id: 'qc-5', orderId: 'prod-5', orderNr: 'PRD-2026-045', inspector: 'Werner Stöckli', date: '2026-02-08', passed: false, defectsFound: 2, notes: 'Fehlende Erdungsanschlüsse bei 2 Schränken. Produktion pausiert bis Korrektur.' },
  { id: 'qc-6', orderId: 'prod-3', orderNr: 'PRD-2026-043', inspector: 'Irene Graf', date: '2026-02-06', passed: true, defectsFound: 0, notes: 'Zwischenprüfung Charge 2, alle 25 Kabelsätze in Ordnung' },
]

export const useProduktionStore = create<ProduktionStore>()(
  persist(
    () => ({
      orders: MOCK_ORDERS,
      boms: MOCK_BOMS,
      qualityChecks: MOCK_QUALITY_CHECKS,
    }),
    { name: 'kmuhub-produktion' },
  ),
)
