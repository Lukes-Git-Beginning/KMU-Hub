import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface Supplier {
  id: string
  name: string
  contactName: string
  email: string
  phone: string
  paymentTerms: string
  isActive: boolean
}

export interface PurchaseOrder {
  id: string
  orderNumber: string
  supplierId: string
  supplierName: string
  status: 'draft' | 'sent' | 'confirmed' | 'partial' | 'received' | 'cancelled'
  total: number
  expectedDelivery: string
  itemCount: number
  createdAt: string
}

export interface PurchaseOrderItem {
  id: string
  orderId: string
  itemName: string
  quantity: number
  unitPrice: number
  receivedQuantity: number
}

interface EinkaufStore {
  suppliers: Supplier[]
  purchaseOrders: PurchaseOrder[]
  purchaseOrderItems: PurchaseOrderItem[]
}

const MOCK_SUPPLIERS: Supplier[] = [
  { id: 'sup-1', name: 'Distrelec AG', contactName: 'Martin Zuercher', email: 'm.zuercher@distrelec.ch', phone: '+41 44 944 99 11', paymentTerms: '30 Tage netto', isActive: true },
  { id: 'sup-2', name: 'Haberkorn GmbH', contactName: 'Stefan Hofer', email: 's.hofer@haberkorn.at', phone: '+43 5574 695 0', paymentTerms: '14 Tage 2% Skonto', isActive: true },
  { id: 'sup-3', name: 'Wuerth Schweiz AG', contactName: 'Claudia Roth', email: 'c.roth@wuerth.ch', phone: '+41 55 464 44 44', paymentTerms: '30 Tage netto', isActive: true },
  { id: 'sup-4', name: 'Reichelt Elektronik', contactName: 'Klaus Bergmann', email: 'k.bergmann@reichelt.de', phone: '+49 4422 955 333', paymentTerms: 'Vorkasse', isActive: true },
  { id: 'sup-5', name: 'Hilti Schweiz AG', contactName: 'Urs Ammann', email: 'u.ammann@hilti.ch', phone: '+41 55 296 42 42', paymentTerms: '45 Tage netto', isActive: false },
]

const MOCK_PURCHASE_ORDERS: PurchaseOrder[] = [
  { id: 'po-1', orderNumber: 'PO-2026-001', supplierId: 'sup-1', supplierName: 'Distrelec AG', status: 'received', total: 3450.80, expectedDelivery: '2026-02-10', itemCount: 4, createdAt: '2026-02-03T08:00:00' },
  { id: 'po-2', orderNumber: 'PO-2026-002', supplierId: 'sup-3', supplierName: 'Wuerth Schweiz AG', status: 'confirmed', total: 1280.50, expectedDelivery: '2026-02-18', itemCount: 3, createdAt: '2026-02-05T14:30:00' },
  { id: 'po-3', orderNumber: 'PO-2026-003', supplierId: 'sup-2', supplierName: 'Haberkorn GmbH', status: 'partial', total: 5670.00, expectedDelivery: '2026-02-15', itemCount: 5, createdAt: '2026-02-06T09:15:00' },
  { id: 'po-4', orderNumber: 'PO-2026-004', supplierId: 'sup-4', supplierName: 'Reichelt Elektronik', status: 'sent', total: 890.20, expectedDelivery: '2026-02-20', itemCount: 6, createdAt: '2026-02-10T11:00:00' },
  { id: 'po-5', orderNumber: 'PO-2026-005', supplierId: 'sup-1', supplierName: 'Distrelec AG', status: 'draft', total: 2150.00, expectedDelivery: '2026-02-25', itemCount: 2, createdAt: '2026-02-12T16:45:00' },
  { id: 'po-6', orderNumber: 'PO-2026-006', supplierId: 'sup-3', supplierName: 'Wuerth Schweiz AG', status: 'received', total: 780.40, expectedDelivery: '2026-02-08', itemCount: 3, createdAt: '2026-02-01T10:20:00' },
  { id: 'po-7', orderNumber: 'PO-2026-007', supplierId: 'sup-5', supplierName: 'Hilti Schweiz AG', status: 'cancelled', total: 4200.00, expectedDelivery: '2026-02-22', itemCount: 2, createdAt: '2026-02-07T13:00:00' },
  { id: 'po-8', orderNumber: 'PO-2026-008', supplierId: 'sup-2', supplierName: 'Haberkorn GmbH', status: 'confirmed', total: 1560.30, expectedDelivery: '2026-02-21', itemCount: 4, createdAt: '2026-02-11T08:30:00' },
  { id: 'po-9', orderNumber: 'PO-2026-009', supplierId: 'sup-1', supplierName: 'Distrelec AG', status: 'sent', total: 3100.00, expectedDelivery: '2026-02-24', itemCount: 5, createdAt: '2026-02-13T15:00:00' },
  { id: 'po-10', orderNumber: 'PO-2026-010', supplierId: 'sup-4', supplierName: 'Reichelt Elektronik', status: 'draft', total: 420.60, expectedDelivery: '2026-02-28', itemCount: 3, createdAt: '2026-02-14T09:00:00' },
]

const MOCK_PO_ITEMS: PurchaseOrderItem[] = [
  { id: 'poi-1', orderId: 'po-1', itemName: 'Sicherungsautomat C16A', quantity: 50, unitPrice: 12.80, receivedQuantity: 50 },
  { id: 'poi-2', orderId: 'po-1', itemName: 'FI/LS-Schalter B16 30mA', quantity: 20, unitPrice: 68.50, receivedQuantity: 20 },
  { id: 'poi-3', orderId: 'po-1', itemName: 'Reihenklemme 4mm²', quantity: 100, unitPrice: 3.90, receivedQuantity: 100 },
  { id: 'poi-4', orderId: 'po-1', itemName: 'Hutschiene 1m', quantity: 10, unitPrice: 8.50, receivedQuantity: 10 },
  { id: 'poi-5', orderId: 'po-2', itemName: 'Duebel-Set 6/8/10mm', quantity: 200, unitPrice: 0.85, receivedQuantity: 0 },
  { id: 'poi-6', orderId: 'po-2', itemName: 'Schrauben TX M5x40', quantity: 500, unitPrice: 0.18, receivedQuantity: 0 },
  { id: 'poi-7', orderId: 'po-2', itemName: 'Kabelbinder 200mm schwarz', quantity: 1000, unitPrice: 1.02, receivedQuantity: 0 },
  { id: 'poi-8', orderId: 'po-3', itemName: 'Arbeitshandschuhe Gr. L', quantity: 50, unitPrice: 8.40, receivedQuantity: 50 },
  { id: 'poi-9', orderId: 'po-3', itemName: 'Schutzbrille klar', quantity: 30, unitPrice: 12.50, receivedQuantity: 30 },
  { id: 'poi-10', orderId: 'po-3', itemName: 'Sicherheitsschuhe S3', quantity: 10, unitPrice: 89.00, receivedQuantity: 0 },
  { id: 'poi-11', orderId: 'po-3', itemName: 'Gehörschutz Peltor', quantity: 15, unitPrice: 42.00, receivedQuantity: 0 },
  { id: 'poi-12', orderId: 'po-3', itemName: 'Warnweste gelb', quantity: 20, unitPrice: 6.50, receivedQuantity: 20 },
]

export const useEinkaufStore = create<EinkaufStore>()(
  persist(
    () => ({
      suppliers: MOCK_SUPPLIERS,
      purchaseOrders: MOCK_PURCHASE_ORDERS,
      purchaseOrderItems: MOCK_PO_ITEMS,
    }),
    { name: 'kmuhub-einkauf' },
  ),
)
