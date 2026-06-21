import { ShoppingCart } from 'lucide-react'
import type { ReportSource } from './types'
import { floatBetween, generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const STATUS = [
  { value: 'received', label: 'Erhalten' },
  { value: 'ordered', label: 'Bestellt' },
  { value: 'partial', label: 'Teilweise erhalten' },
  { value: 'draft', label: 'Entwurf' },
  { value: 'cancelled', label: 'Storniert' },
]
const CATEGORY = [
  { value: 'material', label: 'Material' },
  { value: 'service', label: 'Dienstleistung' },
  { value: 'it', label: 'IT & Software' },
  { value: 'office', label: 'Büro' },
  { value: 'machines', label: 'Maschinen' },
]
const CURRENCY = [
  { value: 'EUR', label: 'EUR' },
  { value: 'CHF', label: 'CHF' },
]
const TERMS = [
  { value: '14', label: '14 Tage' },
  { value: '30', label: '30 Tage' },
  { value: '60', label: '60 Tage' },
]
const SUPPLIERS = [
  'Stahl & Metall AG',
  'TechSupply GmbH',
  'Büroprofi Handel',
  'Industrie Komponenten KG',
  'Logistik Rhein-Main',
  'CloudWorks Software',
  'Werkzeug Schäfer e.K.',
  'Verpackung Nord GmbH',
]

export const einkaufSource: ReportSource = {
  id: 'einkauf',
  module: 'einkauf',
  labelKey: 'berichte.sources.einkauf.label',
  label: 'Bestellungen',
  icon: ShoppingCart,
  description: 'Beschaffung, Lieferanten und Volumen',
  defaultViz: 'bar',
  fields: [
    { key: 'order_date', labelKey: 'berichte.fields.einkauf.order_date', label: 'Bestelldatum', dataType: 'date', role: 'dimension' },
    { key: 'status', labelKey: 'berichte.fields.einkauf.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'category', labelKey: 'berichte.fields.einkauf.category', label: 'Kategorie', dataType: 'enum', role: 'dimension', enumValues: CATEGORY },
    { key: 'supplier', labelKey: 'berichte.fields.einkauf.supplier', label: 'Lieferant', dataType: 'string', role: 'dimension' },
    { key: 'currency', labelKey: 'berichte.fields.einkauf.currency', label: 'Währung', dataType: 'enum', role: 'dimension', enumValues: CURRENCY },
    { key: 'payment_terms', labelKey: 'berichte.fields.einkauf.payment_terms', label: 'Zahlungsziel', dataType: 'enum', role: 'dimension', enumValues: TERMS },
    { key: 'total_amount', labelKey: 'berichte.fields.einkauf.total_amount', label: 'Bestellwert', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'item_count', labelKey: 'berichte.fields.einkauf.item_count', label: 'Positionen', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'delivery_days', labelKey: 'berichte.fields.einkauf.delivery_days', label: 'Lieferzeit (Tage)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
  ],
  sampleRows: (count = 130) =>
    generateRows(640, count, (r) => ({
      order_date: isoDaysAgo(intBetween(r(), 0, 364)),
      status: pick(STATUS, r()).value,
      category: pick(CATEGORY, r()).value,
      supplier: pick(SUPPLIERS, r()),
      currency: pick(CURRENCY, r()).value,
      payment_terms: pick(TERMS, r()).value,
      total_amount: floatBetween(r(), 120, 48000, 2),
      item_count: intBetween(r(), 1, 40),
      delivery_days: intBetween(r(), 1, 45),
    })),
}
