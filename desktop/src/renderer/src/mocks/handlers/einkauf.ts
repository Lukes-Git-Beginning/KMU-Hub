import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function paginate<T>(arr: T[], page = 1, pageSize = 20): T[] {
  const start = (page - 1) * pageSize
  return arr.slice(start, start + pageSize)
}

function newId(): string {
  return 'ek-' + Math.random().toString(36).slice(2, 10)
}

function now(): string {
  return new Date().toISOString()
}

// ---------------------------------------------------------------------------
// Wire-shape interfaces (snake_case, numeric fields as strings)
// ---------------------------------------------------------------------------

interface WireSupplier {
  id: string
  tenant_id: string
  name: string
  contact_id: string | null
  email: string
  phone: string
  address: string
  payment_terms: string
  notes: string
  created_at: string
  updated_at: string
}

interface WirePOLine {
  id: string
  tenant_id: string
  po_id: string
  product_name: string
  sku: string
  quantity: string
  unit_price: string
  tax_rate: string
  received_quantity: string
  line_position: number
  created_at: string
  updated_at: string
}

interface WirePO {
  id: string
  tenant_id: string
  supplier_id: string
  po_number: string
  status: 'draft' | 'submitted' | 'sent' | 'partially_received' | 'received' | 'closed' | 'cancelled'
  order_date: string
  expected_delivery_date: string | null
  total_amount: string
  currency: string
  notes: string
  created_by: string | null
  created_at: string
  updated_at: string
  lines?: WirePOLine[]
}

interface WireCatalogItem {
  id: string
  tenant_id: string
  supplier_id: string
  name: string
  sku: string
  category: string
  price: string
  currency: string
  unit: string
  available: boolean
  min_order_qty: string
  created_at: string
  updated_at: string
}

interface WireSupplierRating {
  id: string
  tenant_id: string
  supplier_id: string
  category: 'quality' | 'delivery' | 'price'
  rating: number
  comment: string
  rated_by: string | null
  rated_at: string
  created_at: string
  updated_at: string
}

interface WireFrameworkContractItem {
  id: string
  tenant_id: string
  contract_id: string
  name: string
  unit_price: string
  unit: string
  agreed_qty: string
  called_qty: string
  created_at: string
  updated_at: string
}

interface WireFrameworkContractCall {
  id: string
  tenant_id: string
  contract_id: string
  po_id: string | null
  amount: string
  currency: string
  called_at: string
  notes: string
  created_at: string
  updated_at: string
}

interface WireFrameworkContract {
  id: string
  tenant_id: string
  supplier_id: string
  title: string
  contract_nr: string
  start_date: string | null
  end_date: string | null
  total_value: string
  used_value: string
  currency: string
  status: 'draft' | 'active' | 'expired'
  created_at: string
  updated_at: string
  items?: WireFrameworkContractItem[]
}

// ---------------------------------------------------------------------------
// Mutable state — seeded from MOCK_ data in einkauf store
// ---------------------------------------------------------------------------

let suppliers: WireSupplier[] = [
  { id: 'sup-1', tenant_id: 'tenant-001', name: 'Distrelec GmbH', contact_id: null, email: 'm.zuercher@distrelec.de', phone: '+49 30 944 99 11', address: '', payment_terms: '30 Tage netto', notes: '', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'sup-2', tenant_id: 'tenant-001', name: 'Haberkorn GmbH', contact_id: null, email: 's.hofer@haberkorn.de', phone: '+49 821 695 0', address: '', payment_terms: '14 Tage 2% Skonto', notes: '', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'sup-3', tenant_id: 'tenant-001', name: 'Würth Deutschland GmbH', contact_id: null, email: 'c.roth@wuerth.de', phone: '+49 7940 15 0', address: '', payment_terms: '30 Tage netto', notes: '', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'sup-4', tenant_id: 'tenant-001', name: 'Reichelt Elektronik', contact_id: null, email: 'k.bergmann@reichelt.de', phone: '+49 4422 955 333', address: '', payment_terms: 'Vorkasse', notes: '', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'sup-5', tenant_id: 'tenant-001', name: 'Hilti Deutschland AG', contact_id: null, email: 'u.ammann@hilti.de', phone: '+49 800 888 5555', address: '', payment_terms: '45 Tage netto', notes: '', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
]

let poLines: WirePOLine[] = [
  { id: 'poi-1', tenant_id: 'tenant-001', po_id: 'po-1', product_name: 'Sicherungsautomat C16A', sku: 'SA-C16-3P', quantity: '50', unit_price: '12.80', tax_rate: '0.19', received_quantity: '50', line_position: 1, created_at: '2026-02-03T08:00:00Z', updated_at: '2026-02-03T08:00:00Z' },
  { id: 'poi-2', tenant_id: 'tenant-001', po_id: 'po-1', product_name: 'FI/LS-Schalter B16 30mA', sku: 'FILS-B16-2P', quantity: '20', unit_price: '68.50', tax_rate: '0.19', received_quantity: '20', line_position: 2, created_at: '2026-02-03T08:00:00Z', updated_at: '2026-02-03T08:00:00Z' },
  { id: 'poi-3', tenant_id: 'tenant-001', po_id: 'po-1', product_name: 'Reihenklemme 4mm²', sku: '', quantity: '100', unit_price: '3.90', tax_rate: '0.19', received_quantity: '100', line_position: 3, created_at: '2026-02-03T08:00:00Z', updated_at: '2026-02-03T08:00:00Z' },
  { id: 'poi-4', tenant_id: 'tenant-001', po_id: 'po-1', product_name: 'Hutschiene 1m', sku: '', quantity: '10', unit_price: '8.50', tax_rate: '0.19', received_quantity: '10', line_position: 4, created_at: '2026-02-03T08:00:00Z', updated_at: '2026-02-03T08:00:00Z' },
  { id: 'poi-5', tenant_id: 'tenant-001', po_id: 'po-2', product_name: 'Dübel-Set 6/8/10mm', sku: '', quantity: '200', unit_price: '0.85', tax_rate: '0.19', received_quantity: '0', line_position: 1, created_at: '2026-02-05T14:30:00Z', updated_at: '2026-02-05T14:30:00Z' },
  { id: 'poi-6', tenant_id: 'tenant-001', po_id: 'po-2', product_name: 'Schrauben TX M5x40', sku: '', quantity: '500', unit_price: '0.18', tax_rate: '0.19', received_quantity: '0', line_position: 2, created_at: '2026-02-05T14:30:00Z', updated_at: '2026-02-05T14:30:00Z' },
  { id: 'poi-7', tenant_id: 'tenant-001', po_id: 'po-2', product_name: 'Kabelbinder 200mm schwarz', sku: '', quantity: '1000', unit_price: '1.02', tax_rate: '0.19', received_quantity: '0', line_position: 3, created_at: '2026-02-05T14:30:00Z', updated_at: '2026-02-05T14:30:00Z' },
  { id: 'poi-8', tenant_id: 'tenant-001', po_id: 'po-3', product_name: 'Arbeitshandschuhe Gr. L', sku: '', quantity: '50', unit_price: '8.40', tax_rate: '0.19', received_quantity: '50', line_position: 1, created_at: '2026-02-06T09:15:00Z', updated_at: '2026-02-06T09:15:00Z' },
  { id: 'poi-9', tenant_id: 'tenant-001', po_id: 'po-3', product_name: 'Schutzbrille klar', sku: '', quantity: '30', unit_price: '12.50', tax_rate: '0.19', received_quantity: '30', line_position: 2, created_at: '2026-02-06T09:15:00Z', updated_at: '2026-02-06T09:15:00Z' },
  { id: 'poi-10', tenant_id: 'tenant-001', po_id: 'po-3', product_name: 'Sicherheitsschuhe S3', sku: '', quantity: '10', unit_price: '89.00', tax_rate: '0.19', received_quantity: '0', line_position: 3, created_at: '2026-02-06T09:15:00Z', updated_at: '2026-02-06T09:15:00Z' },
  { id: 'poi-11', tenant_id: 'tenant-001', po_id: 'po-3', product_name: 'Gehörschutz Peltor', sku: '', quantity: '15', unit_price: '42.00', tax_rate: '0.19', received_quantity: '0', line_position: 4, created_at: '2026-02-06T09:15:00Z', updated_at: '2026-02-06T09:15:00Z' },
  { id: 'poi-12', tenant_id: 'tenant-001', po_id: 'po-3', product_name: 'Warnweste gelb', sku: '', quantity: '20', unit_price: '6.50', tax_rate: '0.19', received_quantity: '20', line_position: 5, created_at: '2026-02-06T09:15:00Z', updated_at: '2026-02-06T09:15:00Z' },
]

let purchaseOrders: WirePO[] = [
  { id: 'po-1', tenant_id: 'tenant-001', supplier_id: 'sup-1', po_number: 'PO-2026-001', status: 'received', order_date: '2026-02-03', expected_delivery_date: '2026-02-10', total_amount: '3450.80', currency: 'EUR', notes: '', created_by: null, created_at: '2026-02-03T08:00:00Z', updated_at: '2026-02-03T08:00:00Z' },
  { id: 'po-2', tenant_id: 'tenant-001', supplier_id: 'sup-3', po_number: 'PO-2026-002', status: 'sent', order_date: '2026-02-05', expected_delivery_date: '2026-02-18', total_amount: '1280.50', currency: 'EUR', notes: '', created_by: null, created_at: '2026-02-05T14:30:00Z', updated_at: '2026-02-05T14:30:00Z' },
  { id: 'po-3', tenant_id: 'tenant-001', supplier_id: 'sup-2', po_number: 'PO-2026-003', status: 'partially_received', order_date: '2026-02-06', expected_delivery_date: '2026-02-15', total_amount: '5670.00', currency: 'EUR', notes: 'Freigabe erforderlich', created_by: null, created_at: '2026-02-06T09:15:00Z', updated_at: '2026-02-06T09:15:00Z' },
  { id: 'po-4', tenant_id: 'tenant-001', supplier_id: 'sup-4', po_number: 'PO-2026-004', status: 'sent', order_date: '2026-02-10', expected_delivery_date: '2026-02-20', total_amount: '890.20', currency: 'EUR', notes: '', created_by: null, created_at: '2026-02-10T11:00:00Z', updated_at: '2026-02-10T11:00:00Z' },
  { id: 'po-5', tenant_id: 'tenant-001', supplier_id: 'sup-1', po_number: 'PO-2026-005', status: 'draft', order_date: '2026-02-12', expected_delivery_date: '2026-02-25', total_amount: '2150.00', currency: 'EUR', notes: '', created_by: null, created_at: '2026-02-12T16:45:00Z', updated_at: '2026-02-12T16:45:00Z' },
  { id: 'po-6', tenant_id: 'tenant-001', supplier_id: 'sup-3', po_number: 'PO-2026-006', status: 'received', order_date: '2026-02-01', expected_delivery_date: '2026-02-08', total_amount: '780.40', currency: 'EUR', notes: '', created_by: null, created_at: '2026-02-01T10:20:00Z', updated_at: '2026-02-01T10:20:00Z' },
  { id: 'po-7', tenant_id: 'tenant-001', supplier_id: 'sup-5', po_number: 'PO-2026-007', status: 'cancelled', order_date: '2026-02-07', expected_delivery_date: '2026-02-22', total_amount: '4200.00', currency: 'EUR', notes: '', created_by: null, created_at: '2026-02-07T13:00:00Z', updated_at: '2026-02-07T13:00:00Z' },
  { id: 'po-8', tenant_id: 'tenant-001', supplier_id: 'sup-2', po_number: 'PO-2026-008', status: 'sent', order_date: '2026-02-11', expected_delivery_date: '2026-02-21', total_amount: '1560.30', currency: 'EUR', notes: '', created_by: null, created_at: '2026-02-11T08:30:00Z', updated_at: '2026-02-11T08:30:00Z' },
  { id: 'po-9', tenant_id: 'tenant-001', supplier_id: 'sup-1', po_number: 'PO-2026-009', status: 'sent', order_date: '2026-02-13', expected_delivery_date: '2026-02-24', total_amount: '3100.00', currency: 'EUR', notes: 'Freigabe erteilt: Max Mustermann', created_by: null, created_at: '2026-02-13T15:00:00Z', updated_at: '2026-02-13T15:00:00Z' },
  { id: 'po-10', tenant_id: 'tenant-001', supplier_id: 'sup-4', po_number: 'PO-2026-010', status: 'draft', order_date: '2026-02-14', expected_delivery_date: '2026-02-28', total_amount: '420.60', currency: 'EUR', notes: '', created_by: null, created_at: '2026-02-14T09:00:00Z', updated_at: '2026-02-14T09:00:00Z' },
]

let catalogItems: WireCatalogItem[] = [
  { id: 'cat-1', tenant_id: 'tenant-001', supplier_id: 'sup-1', name: 'Sicherungsautomat C16A 3-polig', sku: 'SA-C16-3P', category: 'Sicherungen', price: '14.90', currency: 'EUR', unit: 'Stk', available: true, min_order_qty: '10', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-2', tenant_id: 'tenant-001', supplier_id: 'sup-1', name: 'FI/LS-Schalter B16 30mA 2-polig', sku: 'FILS-B16-2P', category: 'Sicherungen', price: '72.50', currency: 'EUR', unit: 'Stk', available: true, min_order_qty: '5', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-3', tenant_id: 'tenant-001', supplier_id: 'sup-3', name: 'Kabelbinder 200mm schwarz (100er)', sku: 'KB-200-BK', category: 'Befestigung', price: '8.90', currency: 'EUR', unit: 'Pkg', available: true, min_order_qty: '1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-4', tenant_id: 'tenant-001', supplier_id: 'sup-2', name: 'Arbeitshandschuhe Nitril Gr. L', sku: 'AH-NIT-L', category: 'Schutzausrüstung', price: '9.50', currency: 'EUR', unit: 'Paar', available: true, min_order_qty: '10', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-5', tenant_id: 'tenant-001', supplier_id: 'sup-4', name: 'LED-Roehre T8 150cm 24W', sku: 'LED-T8-150', category: 'Beleuchtung', price: '12.80', currency: 'EUR', unit: 'Stk', available: true, min_order_qty: '5', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-6', tenant_id: 'tenant-001', supplier_id: 'sup-1', name: 'Verteilerdose AP IP54', sku: 'VD-AP-IP54', category: 'Elektromaterial', price: '7.20', currency: 'EUR', unit: 'Stk', available: false, min_order_qty: '10', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-7', tenant_id: 'tenant-001', supplier_id: 'sup-3', name: 'Kabelkanal 40x60mm weiss (2m)', sku: 'KK-4060-W', category: 'Kabelmanagement', price: '6.50', currency: 'EUR', unit: 'Stk', available: true, min_order_qty: '5', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-8', tenant_id: 'tenant-001', supplier_id: 'sup-3', name: 'Schrauben TX M5x40 (200er)', sku: 'SR-TX-M540', category: 'Befestigung', price: '18.90', currency: 'EUR', unit: 'Pkg', available: true, min_order_qty: '1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-9', tenant_id: 'tenant-001', supplier_id: 'sup-2', name: 'Sicherheitsschuhe S3 Gr. 43', sku: 'SS-S3-43', category: 'Schutzausrüstung', price: '89.00', currency: 'EUR', unit: 'Paar', available: true, min_order_qty: '1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-10', tenant_id: 'tenant-001', supplier_id: 'sup-4', name: 'Multimeter Digital CAT III', sku: 'MM-DIG-C3', category: 'Werkzeug', price: '149.00', currency: 'EUR', unit: 'Stk', available: true, min_order_qty: '1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-11', tenant_id: 'tenant-001', supplier_id: 'sup-1', name: 'NYM-J 5x2.5mm² (50m Ring)', sku: 'NYM-525-50', category: 'Kabel', price: '89.50', currency: 'EUR', unit: 'Ring', available: true, min_order_qty: '1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-12', tenant_id: 'tenant-001', supplier_id: 'sup-5', name: 'Bohrhammer SDS-Plus 800W', sku: 'BH-SDS-800', category: 'Werkzeug', price: '289.00', currency: 'EUR', unit: 'Stk', available: true, min_order_qty: '1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
]

let supplierRatings: WireSupplierRating[] = [
  { id: 'sr-1', tenant_id: 'tenant-001', supplier_id: 'sup-1', category: 'quality', rating: 5, comment: 'Immer einwandfreie Ware', rated_by: null, rated_at: '2026-02-10T00:00:00Z', created_at: '2026-02-10T00:00:00Z', updated_at: '2026-02-10T00:00:00Z' },
  { id: 'sr-2', tenant_id: 'tenant-001', supplier_id: 'sup-1', category: 'delivery', rating: 4, comment: 'Meist pünktlich, selten 1 Tag Verspätung', rated_by: null, rated_at: '2026-02-10T00:00:00Z', created_at: '2026-02-10T00:00:00Z', updated_at: '2026-02-10T00:00:00Z' },
  { id: 'sr-3', tenant_id: 'tenant-001', supplier_id: 'sup-1', category: 'price', rating: 3, comment: 'Marktdurchschnitt', rated_by: null, rated_at: '2026-02-10T00:00:00Z', created_at: '2026-02-10T00:00:00Z', updated_at: '2026-02-10T00:00:00Z' },
  { id: 'sr-4', tenant_id: 'tenant-001', supplier_id: 'sup-2', category: 'quality', rating: 4, comment: 'Gute Qualität', rated_by: null, rated_at: '2026-02-08T00:00:00Z', created_at: '2026-02-08T00:00:00Z', updated_at: '2026-02-08T00:00:00Z' },
  { id: 'sr-5', tenant_id: 'tenant-001', supplier_id: 'sup-2', category: 'delivery', rating: 5, comment: 'Immer pünktlich', rated_by: null, rated_at: '2026-02-08T00:00:00Z', created_at: '2026-02-08T00:00:00Z', updated_at: '2026-02-08T00:00:00Z' },
  { id: 'sr-6', tenant_id: 'tenant-001', supplier_id: 'sup-2', category: 'price', rating: 4, comment: 'Guter Skonto', rated_by: null, rated_at: '2026-02-08T00:00:00Z', created_at: '2026-02-08T00:00:00Z', updated_at: '2026-02-08T00:00:00Z' },
  { id: 'sr-7', tenant_id: 'tenant-001', supplier_id: 'sup-3', category: 'quality', rating: 5, comment: 'Topqualität', rated_by: null, rated_at: '2026-02-05T00:00:00Z', created_at: '2026-02-05T00:00:00Z', updated_at: '2026-02-05T00:00:00Z' },
  { id: 'sr-8', tenant_id: 'tenant-001', supplier_id: 'sup-3', category: 'delivery', rating: 3, comment: 'Manchmal verzögert', rated_by: null, rated_at: '2026-02-05T00:00:00Z', created_at: '2026-02-05T00:00:00Z', updated_at: '2026-02-05T00:00:00Z' },
  { id: 'sr-9', tenant_id: 'tenant-001', supplier_id: 'sup-3', category: 'price', rating: 3, comment: 'Etwas teurer als Wettbewerb', rated_by: null, rated_at: '2026-02-05T00:00:00Z', created_at: '2026-02-05T00:00:00Z', updated_at: '2026-02-05T00:00:00Z' },
  { id: 'sr-10', tenant_id: 'tenant-001', supplier_id: 'sup-4', category: 'quality', rating: 4, comment: 'Solide Ware', rated_by: null, rated_at: '2026-01-20T00:00:00Z', created_at: '2026-01-20T00:00:00Z', updated_at: '2026-01-20T00:00:00Z' },
  { id: 'sr-11', tenant_id: 'tenant-001', supplier_id: 'sup-4', category: 'delivery', rating: 4, comment: 'Schnelle Lieferung', rated_by: null, rated_at: '2026-01-20T00:00:00Z', created_at: '2026-01-20T00:00:00Z', updated_at: '2026-01-20T00:00:00Z' },
  { id: 'sr-12', tenant_id: 'tenant-001', supplier_id: 'sup-4', category: 'price', rating: 5, comment: 'Sehr günstig', rated_by: null, rated_at: '2026-01-20T00:00:00Z', created_at: '2026-01-20T00:00:00Z', updated_at: '2026-01-20T00:00:00Z' },
]

let frameworkContractItems: WireFrameworkContractItem[] = [
  { id: 'fci-1', tenant_id: 'tenant-001', contract_id: 'fc-1', name: 'Sicherungsautomat C16A', unit_price: '12.80', unit: 'Stk', agreed_qty: '500', called_qty: '120', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'fci-2', tenant_id: 'tenant-001', contract_id: 'fc-1', name: 'FI-Schutzschalter 30mA', unit_price: '42.50', unit: 'Stk', agreed_qty: '200', called_qty: '45', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'fci-3', tenant_id: 'tenant-001', contract_id: 'fc-1', name: 'Reihenklemme 4mm²', unit_price: '3.90', unit: 'Stk', agreed_qty: '1000', called_qty: '350', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'fci-4', tenant_id: 'tenant-001', contract_id: 'fc-2', name: 'Dübel-Set 6/8/10mm', unit_price: '0.85', unit: 'Stk', agreed_qty: '5000', called_qty: '3200', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'fci-5', tenant_id: 'tenant-001', contract_id: 'fc-2', name: 'Schrauben TX M5x40', unit_price: '0.18', unit: 'Stk', agreed_qty: '10000', called_qty: '6500', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'fci-6', tenant_id: 'tenant-001', contract_id: 'fc-2', name: 'Kabelbinder 200mm', unit_price: '1.02', unit: 'Stk', agreed_qty: '3000', called_qty: '1800', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'fci-7', tenant_id: 'tenant-001', contract_id: 'fc-3', name: 'Arbeitshandschuhe Gr. L', unit_price: '8.40', unit: 'Paar', agreed_qty: '200', called_qty: '195', created_at: '2025-07-01T00:00:00Z', updated_at: '2025-07-01T00:00:00Z' },
  { id: 'fci-8', tenant_id: 'tenant-001', contract_id: 'fc-3', name: 'Sicherheitsschuhe S3', unit_price: '89.00', unit: 'Paar', agreed_qty: '30', called_qty: '28', created_at: '2025-07-01T00:00:00Z', updated_at: '2025-07-01T00:00:00Z' },
]

let frameworkContracts: WireFrameworkContract[] = [
  { id: 'fc-1', tenant_id: 'tenant-001', supplier_id: 'sup-1', title: 'Jahresvertrag Elektromaterial 2026', contract_nr: 'RV-2026-001', start_date: '2026-01-01', end_date: '2026-12-31', total_value: '50000.00', used_value: '12500.00', currency: 'EUR', status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'fc-2', tenant_id: 'tenant-001', supplier_id: 'sup-3', title: 'Befestigungsmaterial 2026', contract_nr: 'RV-2026-002', start_date: '2026-01-01', end_date: '2026-06-30', total_value: '15000.00', used_value: '8200.00', currency: 'EUR', status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'fc-3', tenant_id: 'tenant-001', supplier_id: 'sup-2', title: 'Schutzausrüstung Q1/Q2', contract_nr: 'RV-2025-008', start_date: '2025-07-01', end_date: '2025-12-31', total_value: '8000.00', used_value: '7850.00', currency: 'EUR', status: 'expired', created_at: '2025-07-01T00:00:00Z', updated_at: '2025-07-01T00:00:00Z' },
]

let contractCalls: WireFrameworkContractCall[] = []

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const einkaufHandlers = [

  // ── Suppliers ──────────────────────────────────────────────────────────────

  http.get(`${API}/api/v1/einkauf/suppliers`, ({ request }) => {
    const url = new URL(request.url)
    const search = url.searchParams.get('search') ?? ''
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    let result = [...suppliers]
    if (search) {
      const q = search.toLowerCase()
      result = result.filter(s => s.name.toLowerCase().includes(q) || s.email.toLowerCase().includes(q))
    }
    return HttpResponse.json({ suppliers: paginate(result, page, pageSize), total: result.length })
  }),

  http.get(`${API}/api/v1/einkauf/suppliers/:id`, ({ params }) => {
    const supplier = suppliers.find(s => s.id === params.id)
    if (!supplier) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    return HttpResponse.json({ supplier })
  }),

  http.post(`${API}/api/v1/einkauf/suppliers`, async ({ request }) => {
    const body = await request.json() as Partial<WireSupplier>
    const supplier: WireSupplier = {
      id: newId(),
      tenant_id: 'tenant-001',
      name: body.name ?? '',
      contact_id: body.contact_id ?? null,
      email: body.email ?? '',
      phone: body.phone ?? '',
      address: body.address ?? '',
      payment_terms: body.payment_terms ?? '',
      notes: body.notes ?? '',
      created_at: now(),
      updated_at: now(),
    }
    suppliers = [...suppliers, supplier]
    return HttpResponse.json({ supplier }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/einkauf/suppliers/:id`, async ({ params, request }) => {
    const body = await request.json() as Partial<WireSupplier>
    const idx = suppliers.findIndex(s => s.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    suppliers[idx] = { ...suppliers[idx], ...body, updated_at: now() }
    return HttpResponse.json({ supplier: suppliers[idx] })
  }),

  http.delete(`${API}/api/v1/einkauf/suppliers/:id`, ({ params }) => {
    suppliers = suppliers.filter(s => s.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // ── Supplier Ratings ───────────────────────────────────────────────────────

  http.get(`${API}/api/v1/einkauf/suppliers/:supplierId/ratings`, ({ params }) => {
    const result = supplierRatings.filter(r => r.supplier_id === params.supplierId)
    return HttpResponse.json({ ratings: result })
  }),

  http.post(`${API}/api/v1/einkauf/suppliers/:supplierId/ratings`, async ({ params, request }) => {
    const body = await request.json() as Partial<WireSupplierRating>
    const rating: WireSupplierRating = {
      id: newId(),
      tenant_id: 'tenant-001',
      supplier_id: String(params.supplierId),
      category: body.category ?? 'quality',
      rating: body.rating ?? 1,
      comment: body.comment ?? '',
      rated_by: null,
      rated_at: now(),
      created_at: now(),
      updated_at: now(),
    }
    supplierRatings = [...supplierRatings, rating]
    return HttpResponse.json({ rating }, { status: 201 })
  }),

  http.delete(`${API}/api/v1/einkauf/suppliers/:supplierId/ratings/:ratingId`, ({ params }) => {
    supplierRatings = supplierRatings.filter(r => r.id !== params.ratingId)
    return new HttpResponse(null, { status: 204 })
  }),

  // ── Purchase Orders ────────────────────────────────────────────────────────

  http.get(`${API}/api/v1/einkauf/pos`, ({ request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    const statusFilter = url.searchParams.get('status') ?? ''
    const supplierFilter = url.searchParams.get('supplier_id') ?? ''
    let result = [...purchaseOrders]
    if (statusFilter) result = result.filter(p => p.status === statusFilter)
    if (supplierFilter) result = result.filter(p => p.supplier_id === supplierFilter)
    return HttpResponse.json({ pos: paginate(result, page, pageSize), total: result.length })
  }),

  http.get(`${API}/api/v1/einkauf/pos/:id`, ({ params }) => {
    const po = purchaseOrders.find(p => p.id === params.id)
    if (!po) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    const lines = poLines.filter(l => l.po_id === params.id)
    return HttpResponse.json({ po: { ...po, lines } })
  }),

  http.post(`${API}/api/v1/einkauf/pos`, async ({ request }) => {
    const body = await request.json() as Partial<WirePO>
    const po: WirePO = {
      id: newId(),
      tenant_id: 'tenant-001',
      supplier_id: body.supplier_id ?? '',
      po_number: body.po_number ?? `PO-${Date.now()}`,
      status: 'draft',
      order_date: body.order_date ?? new Date().toISOString().split('T')[0],
      expected_delivery_date: body.expected_delivery_date ?? null,
      total_amount: '0.00',
      currency: body.currency ?? 'EUR',
      notes: body.notes ?? '',
      created_by: null,
      created_at: now(),
      updated_at: now(),
    }
    purchaseOrders = [...purchaseOrders, po]
    return HttpResponse.json({ po }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/einkauf/pos/:id`, async ({ params, request }) => {
    const body = await request.json() as Partial<WirePO>
    const idx = purchaseOrders.findIndex(p => p.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    purchaseOrders[idx] = { ...purchaseOrders[idx], ...body, updated_at: now() }
    return HttpResponse.json({ po: purchaseOrders[idx] })
  }),

  http.delete(`${API}/api/v1/einkauf/pos/:id`, ({ params }) => {
    purchaseOrders = purchaseOrders.filter(p => p.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${API}/api/v1/einkauf/pos/:id/submit`, ({ params }) => {
    const idx = purchaseOrders.findIndex(p => p.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    purchaseOrders[idx] = { ...purchaseOrders[idx], status: 'submitted', updated_at: now() }
    return HttpResponse.json({ po: purchaseOrders[idx] })
  }),

  http.post(`${API}/api/v1/einkauf/pos/:id/receive`, ({ params }) => {
    const idx = purchaseOrders.findIndex(p => p.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    purchaseOrders[idx] = { ...purchaseOrders[idx], status: 'received', updated_at: now() }
    // Mark all lines as fully received
    poLines = poLines.map(l =>
      l.po_id === params.id ? { ...l, received_quantity: l.quantity, updated_at: now() } : l
    )
    return HttpResponse.json({ po: purchaseOrders[idx] })
  }),

  http.post(`${API}/api/v1/einkauf/pos/:id/partial-receive`, async ({ params, request }) => {
    const body = await request.json() as { items: { line_id: string; received_quantity: string }[] }
    const idx = purchaseOrders.findIndex(p => p.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    if (body.items) {
      body.items.forEach(item => {
        const lineIdx = poLines.findIndex(l => l.id === item.line_id)
        if (lineIdx !== -1) {
          poLines[lineIdx] = { ...poLines[lineIdx], received_quantity: item.received_quantity, updated_at: now() }
        }
      })
    }
    purchaseOrders[idx] = { ...purchaseOrders[idx], status: 'partially_received', updated_at: now() }
    return HttpResponse.json({ po: purchaseOrders[idx] })
  }),

  // ── PO Lines ───────────────────────────────────────────────────────────────

  http.get(`${API}/api/v1/einkauf/pos/:poId/lines`, ({ params }) => {
    const lines = poLines.filter(l => l.po_id === params.poId)
    return HttpResponse.json({ lines })
  }),

  http.post(`${API}/api/v1/einkauf/pos/:poId/lines`, async ({ params, request }) => {
    const body = await request.json() as Partial<WirePOLine>
    const existingLines = poLines.filter(l => l.po_id === params.poId)
    const line: WirePOLine = {
      id: newId(),
      tenant_id: 'tenant-001',
      po_id: String(params.poId),
      product_name: body.product_name ?? '',
      sku: body.sku ?? '',
      quantity: body.quantity ?? '0',
      unit_price: body.unit_price ?? '0',
      tax_rate: body.tax_rate ?? '0.19',
      received_quantity: '0',
      line_position: body.line_position ?? existingLines.length + 1,
      created_at: now(),
      updated_at: now(),
    }
    poLines = [...poLines, line]
    return HttpResponse.json({ line }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/einkauf/pos/:poId/lines/:lineId`, async ({ params, request }) => {
    const body = await request.json() as Partial<WirePOLine>
    const idx = poLines.findIndex(l => l.id === params.lineId && l.po_id === params.poId)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    poLines[idx] = { ...poLines[idx], ...body, updated_at: now() }
    return HttpResponse.json({ line: poLines[idx] })
  }),

  http.delete(`${API}/api/v1/einkauf/pos/:poId/lines/:lineId`, ({ params }) => {
    poLines = poLines.filter(l => !(l.id === params.lineId && l.po_id === params.poId))
    return new HttpResponse(null, { status: 204 })
  }),

  // ── Catalog Items ──────────────────────────────────────────────────────────

  http.get(`${API}/api/v1/einkauf/catalog`, ({ request }) => {
    const url = new URL(request.url)
    const search = url.searchParams.get('search') ?? ''
    const category = url.searchParams.get('category') ?? ''
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    let result = [...catalogItems]
    if (search) {
      const q = search.toLowerCase()
      result = result.filter(c => c.name.toLowerCase().includes(q) || c.sku.toLowerCase().includes(q))
    }
    if (category) {
      result = result.filter(c => c.category === category)
    }
    return HttpResponse.json({ catalog_items: paginate(result, page, pageSize), total: result.length })
  }),

  http.get(`${API}/api/v1/einkauf/catalog/:id`, ({ params }) => {
    const catalog_item = catalogItems.find(c => c.id === params.id)
    if (!catalog_item) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    return HttpResponse.json({ catalog_item })
  }),

  http.post(`${API}/api/v1/einkauf/catalog`, async ({ request }) => {
    const body = await request.json() as Partial<WireCatalogItem>
    const catalog_item: WireCatalogItem = {
      id: newId(),
      tenant_id: 'tenant-001',
      supplier_id: body.supplier_id ?? '',
      name: body.name ?? '',
      sku: body.sku ?? '',
      category: body.category ?? '',
      price: body.price ?? '0',
      currency: body.currency ?? 'EUR',
      unit: body.unit ?? 'Stk',
      available: body.available ?? true,
      min_order_qty: body.min_order_qty ?? '1',
      created_at: now(),
      updated_at: now(),
    }
    catalogItems = [...catalogItems, catalog_item]
    return HttpResponse.json({ catalog_item }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/einkauf/catalog/:id`, async ({ params, request }) => {
    const body = await request.json() as Partial<WireCatalogItem>
    const idx = catalogItems.findIndex(c => c.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    catalogItems[idx] = { ...catalogItems[idx], ...body, updated_at: now() }
    return HttpResponse.json({ catalog_item: catalogItems[idx] })
  }),

  http.delete(`${API}/api/v1/einkauf/catalog/:id`, ({ params }) => {
    catalogItems = catalogItems.filter(c => c.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // ── Framework Contracts ────────────────────────────────────────────────────

  http.get(`${API}/api/v1/einkauf/contracts`, ({ request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    const statusFilter = url.searchParams.get('status') ?? ''
    let result = [...frameworkContracts]
    if (statusFilter) result = result.filter(c => c.status === statusFilter)
    // Attach items
    const withItems = paginate(result, page, pageSize).map(c => ({
      ...c,
      items: frameworkContractItems.filter(i => i.contract_id === c.id),
    }))
    return HttpResponse.json({ contracts: withItems, total: result.length })
  }),

  http.get(`${API}/api/v1/einkauf/contracts/:id`, ({ params }) => {
    const contract = frameworkContracts.find(c => c.id === params.id)
    if (!contract) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    const items = frameworkContractItems.filter(i => i.contract_id === params.id)
    return HttpResponse.json({ contract: { ...contract, items } })
  }),

  http.post(`${API}/api/v1/einkauf/contracts`, async ({ request }) => {
    const body = await request.json() as Partial<WireFrameworkContract>
    const contract: WireFrameworkContract = {
      id: newId(),
      tenant_id: 'tenant-001',
      supplier_id: body.supplier_id ?? '',
      title: body.title ?? '',
      contract_nr: body.contract_nr ?? '',
      start_date: body.start_date ?? null,
      end_date: body.end_date ?? null,
      total_value: body.total_value ?? '0',
      used_value: '0.00',
      currency: body.currency ?? 'EUR',
      status: body.status ?? 'draft',
      created_at: now(),
      updated_at: now(),
    }
    frameworkContracts = [...frameworkContracts, contract]
    return HttpResponse.json({ contract }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/einkauf/contracts/:id`, async ({ params, request }) => {
    const body = await request.json() as Partial<WireFrameworkContract>
    const idx = frameworkContracts.findIndex(c => c.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    frameworkContracts[idx] = { ...frameworkContracts[idx], ...body, updated_at: now() }
    return HttpResponse.json({ contract: frameworkContracts[idx] })
  }),

  http.delete(`${API}/api/v1/einkauf/contracts/:id`, ({ params }) => {
    frameworkContracts = frameworkContracts.filter(c => c.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${API}/api/v1/einkauf/contracts/:contractId/items`, async ({ params, request }) => {
    const body = await request.json() as Partial<WireFrameworkContractItem>
    const item: WireFrameworkContractItem = {
      id: newId(),
      tenant_id: 'tenant-001',
      contract_id: String(params.contractId),
      name: body.name ?? '',
      unit_price: body.unit_price ?? '0',
      unit: body.unit ?? 'Stk',
      agreed_qty: body.agreed_qty ?? '0',
      called_qty: '0',
      created_at: now(),
      updated_at: now(),
    }
    frameworkContractItems = [...frameworkContractItems, item]
    return HttpResponse.json({ item }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/einkauf/contracts/:contractId/items/:itemId`, async ({ params, request }) => {
    const body = await request.json() as Partial<WireFrameworkContractItem>
    const idx = frameworkContractItems.findIndex(i => i.id === params.itemId && i.contract_id === params.contractId)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    frameworkContractItems[idx] = { ...frameworkContractItems[idx], ...body, updated_at: now() }
    return HttpResponse.json({ item: frameworkContractItems[idx] })
  }),

  http.delete(`${API}/api/v1/einkauf/contracts/:contractId/items/:itemId`, ({ params }) => {
    frameworkContractItems = frameworkContractItems.filter(
      i => !(i.id === params.itemId && i.contract_id === params.contractId)
    )
    return new HttpResponse(null, { status: 204 })
  }),

  http.get(`${API}/api/v1/einkauf/contracts/:contractId/calls`, ({ params }) => {
    const calls = contractCalls.filter(c => c.contract_id === params.contractId)
    return HttpResponse.json({ calls })
  }),

  http.post(`${API}/api/v1/einkauf/contracts/:contractId/calls`, async ({ params, request }) => {
    const body = await request.json() as Partial<WireFrameworkContractCall>
    const call: WireFrameworkContractCall = {
      id: newId(),
      tenant_id: 'tenant-001',
      contract_id: String(params.contractId),
      po_id: body.po_id ?? null,
      amount: body.amount ?? '0',
      currency: body.currency ?? 'EUR',
      called_at: now(),
      notes: body.notes ?? '',
      created_at: now(),
      updated_at: now(),
    }
    contractCalls = [...contractCalls, call]
    // Update used_value on contract
    const contractIdx = frameworkContracts.findIndex(c => c.id === params.contractId)
    if (contractIdx !== -1) {
      const newUsed = (parseFloat(frameworkContracts[contractIdx].used_value) + parseFloat(call.amount)).toFixed(2)
      frameworkContracts[contractIdx] = { ...frameworkContracts[contractIdx], used_value: newUsed, updated_at: now() }
    }
    return HttpResponse.json({ call }, { status: 201 })
  }),
]
