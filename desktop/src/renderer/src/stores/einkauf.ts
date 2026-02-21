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
  currency: string
  expectedDelivery: string
  itemCount: number
  createdAt: string
  requiresApproval?: boolean
  approvedBy?: string
}

export interface CatalogItem {
  id: string
  name: string
  sku: string
  category: string
  price: number
  currency: string
  unit: string
  supplierId: string
  supplierName: string
  available: boolean
  minOrder: number
}

export interface SupplierRating {
  id: string
  supplierId: string
  category: 'quality' | 'delivery' | 'price'
  rating: number
  comment: string
  date: string
}

export interface FrameworkContract {
  id: string
  supplierId: string
  supplierName: string
  title: string
  contractNr: string
  startDate: string
  endDate: string
  totalValue: number
  usedValue: number
  currency: string
  status: 'active' | 'expired' | 'draft'
  items: { name: string; unitPrice: number; unit: string; agreedQty: number; calledQty: number }[]
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
  catalogItems: CatalogItem[]
  supplierRatings: SupplierRating[]
  frameworkContracts: FrameworkContract[]
  approvalThreshold: number
}

const MOCK_SUPPLIERS: Supplier[] = [
  { id: 'sup-1', name: 'Distrelec AG', contactName: 'Martin Zuercher', email: 'm.zuercher@distrelec.ch', phone: '+41 44 944 99 11', paymentTerms: '30 Tage netto', isActive: true },
  { id: 'sup-2', name: 'Haberkorn GmbH', contactName: 'Stefan Hofer', email: 's.hofer@haberkorn.at', phone: '+43 5574 695 0', paymentTerms: '14 Tage 2% Skonto', isActive: true },
  { id: 'sup-3', name: 'Wuerth Schweiz AG', contactName: 'Claudia Roth', email: 'c.roth@wuerth.ch', phone: '+41 55 464 44 44', paymentTerms: '30 Tage netto', isActive: true },
  { id: 'sup-4', name: 'Reichelt Elektronik', contactName: 'Klaus Bergmann', email: 'k.bergmann@reichelt.de', phone: '+49 4422 955 333', paymentTerms: 'Vorkasse', isActive: true },
  { id: 'sup-5', name: 'Hilti Schweiz AG', contactName: 'Urs Ammann', email: 'u.ammann@hilti.ch', phone: '+41 55 296 42 42', paymentTerms: '45 Tage netto', isActive: false },
]

const MOCK_PURCHASE_ORDERS: PurchaseOrder[] = [
  { id: 'po-1', orderNumber: 'PO-2026-001', supplierId: 'sup-1', supplierName: 'Distrelec AG', status: 'received', total: 3450.80, currency: 'EUR', expectedDelivery: '2026-02-10', itemCount: 4, createdAt: '2026-02-03T08:00:00' },
  { id: 'po-2', orderNumber: 'PO-2026-002', supplierId: 'sup-3', supplierName: 'Wuerth Schweiz AG', status: 'confirmed', total: 1280.50, currency: 'EUR', expectedDelivery: '2026-02-18', itemCount: 3, createdAt: '2026-02-05T14:30:00' },
  { id: 'po-3', orderNumber: 'PO-2026-003', supplierId: 'sup-2', supplierName: 'Haberkorn GmbH', status: 'partial', total: 5670.00, currency: 'EUR', expectedDelivery: '2026-02-15', itemCount: 5, createdAt: '2026-02-06T09:15:00', requiresApproval: true, approvedBy: 'Max Mustermann' },
  { id: 'po-4', orderNumber: 'PO-2026-004', supplierId: 'sup-4', supplierName: 'Reichelt Elektronik', status: 'sent', total: 890.20, currency: 'EUR', expectedDelivery: '2026-02-20', itemCount: 6, createdAt: '2026-02-10T11:00:00' },
  { id: 'po-5', orderNumber: 'PO-2026-005', supplierId: 'sup-1', supplierName: 'Distrelec AG', status: 'draft', total: 2150.00, currency: 'EUR', expectedDelivery: '2026-02-25', itemCount: 2, createdAt: '2026-02-12T16:45:00' },
  { id: 'po-6', orderNumber: 'PO-2026-006', supplierId: 'sup-3', supplierName: 'Wuerth Schweiz AG', status: 'received', total: 780.40, currency: 'EUR', expectedDelivery: '2026-02-08', itemCount: 3, createdAt: '2026-02-01T10:20:00' },
  { id: 'po-7', orderNumber: 'PO-2026-007', supplierId: 'sup-5', supplierName: 'Hilti Schweiz AG', status: 'cancelled', total: 4200.00, currency: 'CHF', expectedDelivery: '2026-02-22', itemCount: 2, createdAt: '2026-02-07T13:00:00' },
  { id: 'po-8', orderNumber: 'PO-2026-008', supplierId: 'sup-2', supplierName: 'Haberkorn GmbH', status: 'confirmed', total: 1560.30, currency: 'EUR', expectedDelivery: '2026-02-21', itemCount: 4, createdAt: '2026-02-11T08:30:00' },
  { id: 'po-9', orderNumber: 'PO-2026-009', supplierId: 'sup-1', supplierName: 'Distrelec AG', status: 'sent', total: 3100.00, currency: 'EUR', expectedDelivery: '2026-02-24', itemCount: 5, createdAt: '2026-02-13T15:00:00', requiresApproval: true, approvedBy: 'Max Mustermann' },
  { id: 'po-10', orderNumber: 'PO-2026-010', supplierId: 'sup-4', supplierName: 'Reichelt Elektronik', status: 'draft', total: 420.60, currency: 'EUR', expectedDelivery: '2026-02-28', itemCount: 3, createdAt: '2026-02-14T09:00:00' },
]

const MOCK_CATALOG_ITEMS: CatalogItem[] = [
  { id: 'cat-1', name: 'Sicherungsautomat C16A 3-polig', sku: 'SA-C16-3P', category: 'Sicherungen', price: 14.90, currency: 'EUR', unit: 'Stk', supplierId: 'sup-1', supplierName: 'Distrelec AG', available: true, minOrder: 10 },
  { id: 'cat-2', name: 'FI/LS-Schalter B16 30mA 2-polig', sku: 'FILS-B16-2P', category: 'Sicherungen', price: 72.50, currency: 'EUR', unit: 'Stk', supplierId: 'sup-1', supplierName: 'Distrelec AG', available: true, minOrder: 5 },
  { id: 'cat-3', name: 'Kabelbinder 200mm schwarz (100er)', sku: 'KB-200-BK', category: 'Befestigung', price: 8.90, currency: 'EUR', unit: 'Pkg', supplierId: 'sup-3', supplierName: 'Wuerth Schweiz AG', available: true, minOrder: 1 },
  { id: 'cat-4', name: 'Arbeitshandschuhe Nitril Gr. L', sku: 'AH-NIT-L', category: 'Schutzausruestung', price: 9.50, currency: 'EUR', unit: 'Paar', supplierId: 'sup-2', supplierName: 'Haberkorn GmbH', available: true, minOrder: 10 },
  { id: 'cat-5', name: 'LED-Roehre T8 150cm 24W', sku: 'LED-T8-150', category: 'Beleuchtung', price: 12.80, currency: 'EUR', unit: 'Stk', supplierId: 'sup-4', supplierName: 'Reichelt Elektronik', available: true, minOrder: 5 },
  { id: 'cat-6', name: 'Verteilerdose AP IP54', sku: 'VD-AP-IP54', category: 'Elektromaterial', price: 7.20, currency: 'EUR', unit: 'Stk', supplierId: 'sup-1', supplierName: 'Distrelec AG', available: false, minOrder: 10 },
  { id: 'cat-7', name: 'Kabelkanal 40x60mm weiss (2m)', sku: 'KK-4060-W', category: 'Kabelmanagement', price: 6.50, currency: 'EUR', unit: 'Stk', supplierId: 'sup-3', supplierName: 'Wuerth Schweiz AG', available: true, minOrder: 5 },
  { id: 'cat-8', name: 'Schrauben TX M5x40 (200er)', sku: 'SR-TX-M540', category: 'Befestigung', price: 18.90, currency: 'EUR', unit: 'Pkg', supplierId: 'sup-3', supplierName: 'Wuerth Schweiz AG', available: true, minOrder: 1 },
  { id: 'cat-9', name: 'Sicherheitsschuhe S3 Gr. 43', sku: 'SS-S3-43', category: 'Schutzausruestung', price: 89.00, currency: 'EUR', unit: 'Paar', supplierId: 'sup-2', supplierName: 'Haberkorn GmbH', available: true, minOrder: 1 },
  { id: 'cat-10', name: 'Multimeter Digital CAT III', sku: 'MM-DIG-C3', category: 'Werkzeug', price: 149.00, currency: 'EUR', unit: 'Stk', supplierId: 'sup-4', supplierName: 'Reichelt Elektronik', available: true, minOrder: 1 },
  { id: 'cat-11', name: 'NYM-J 5x2.5mm² (50m Ring)', sku: 'NYM-525-50', category: 'Kabel', price: 89.50, currency: 'EUR', unit: 'Ring', supplierId: 'sup-1', supplierName: 'Distrelec AG', available: true, minOrder: 1 },
  { id: 'cat-12', name: 'Bohrhammer SDS-Plus 800W', sku: 'BH-SDS-800', category: 'Werkzeug', price: 289.00, currency: 'CHF', unit: 'Stk', supplierId: 'sup-5', supplierName: 'Hilti Schweiz AG', available: true, minOrder: 1 },
]

const MOCK_SUPPLIER_RATINGS: SupplierRating[] = [
  { id: 'sr-1', supplierId: 'sup-1', category: 'quality', rating: 5, comment: 'Immer einwandfreie Ware', date: '2026-02-10' },
  { id: 'sr-2', supplierId: 'sup-1', category: 'delivery', rating: 4, comment: 'Meist puenktlich, selten 1 Tag Verspaetung', date: '2026-02-10' },
  { id: 'sr-3', supplierId: 'sup-1', category: 'price', rating: 3, comment: 'Marktdurchschnitt', date: '2026-02-10' },
  { id: 'sr-4', supplierId: 'sup-2', category: 'quality', rating: 4, comment: 'Gute Qualitaet', date: '2026-02-08' },
  { id: 'sr-5', supplierId: 'sup-2', category: 'delivery', rating: 5, comment: 'Immer puenktlich', date: '2026-02-08' },
  { id: 'sr-6', supplierId: 'sup-2', category: 'price', rating: 4, comment: 'Guter Skonto', date: '2026-02-08' },
  { id: 'sr-7', supplierId: 'sup-3', category: 'quality', rating: 5, comment: 'Topqualitaet', date: '2026-02-05' },
  { id: 'sr-8', supplierId: 'sup-3', category: 'delivery', rating: 3, comment: 'Manchmal verzoegert', date: '2026-02-05' },
  { id: 'sr-9', supplierId: 'sup-3', category: 'price', rating: 3, comment: 'Etwas teurer als Wettbewerb', date: '2026-02-05' },
  { id: 'sr-10', supplierId: 'sup-4', category: 'quality', rating: 4, comment: 'Solide Ware', date: '2026-01-20' },
  { id: 'sr-11', supplierId: 'sup-4', category: 'delivery', rating: 4, comment: 'Schnelle Lieferung', date: '2026-01-20' },
  { id: 'sr-12', supplierId: 'sup-4', category: 'price', rating: 5, comment: 'Sehr guenstig', date: '2026-01-20' },
]

const MOCK_FRAMEWORK_CONTRACTS: FrameworkContract[] = [
  {
    id: 'fc-1', supplierId: 'sup-1', supplierName: 'Distrelec AG', title: 'Jahresvertrag Elektromaterial 2026',
    contractNr: 'RV-2026-001', startDate: '2026-01-01', endDate: '2026-12-31', totalValue: 50000, usedValue: 12500,
    currency: 'EUR', status: 'active',
    items: [
      { name: 'Sicherungsautomat C16A', unitPrice: 12.80, unit: 'Stk', agreedQty: 500, calledQty: 120 },
      { name: 'FI-Schutzschalter 30mA', unitPrice: 42.50, unit: 'Stk', agreedQty: 200, calledQty: 45 },
      { name: 'Reihenklemme 4mm²', unitPrice: 3.90, unit: 'Stk', agreedQty: 1000, calledQty: 350 },
    ],
  },
  {
    id: 'fc-2', supplierId: 'sup-3', supplierName: 'Wuerth Schweiz AG', title: 'Befestigungsmaterial 2026',
    contractNr: 'RV-2026-002', startDate: '2026-01-01', endDate: '2026-06-30', totalValue: 15000, usedValue: 8200,
    currency: 'EUR', status: 'active',
    items: [
      { name: 'Duebel-Set 6/8/10mm', unitPrice: 0.85, unit: 'Stk', agreedQty: 5000, calledQty: 3200 },
      { name: 'Schrauben TX M5x40', unitPrice: 0.18, unit: 'Stk', agreedQty: 10000, calledQty: 6500 },
      { name: 'Kabelbinder 200mm', unitPrice: 1.02, unit: 'Stk', agreedQty: 3000, calledQty: 1800 },
    ],
  },
  {
    id: 'fc-3', supplierId: 'sup-2', supplierName: 'Haberkorn GmbH', title: 'Schutzausruestung Q1/Q2',
    contractNr: 'RV-2025-008', startDate: '2025-07-01', endDate: '2025-12-31', totalValue: 8000, usedValue: 7850,
    currency: 'EUR', status: 'expired',
    items: [
      { name: 'Arbeitshandschuhe Gr. L', unitPrice: 8.40, unit: 'Paar', agreedQty: 200, calledQty: 195 },
      { name: 'Sicherheitsschuhe S3', unitPrice: 89.00, unit: 'Paar', agreedQty: 30, calledQty: 28 },
    ],
  },
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
      catalogItems: MOCK_CATALOG_ITEMS,
      supplierRatings: MOCK_SUPPLIER_RATINGS,
      frameworkContracts: MOCK_FRAMEWORK_CONTRACTS,
      approvalThreshold: 5000,
    }),
    { name: 'kmuhub-einkauf' },
  ),
)
