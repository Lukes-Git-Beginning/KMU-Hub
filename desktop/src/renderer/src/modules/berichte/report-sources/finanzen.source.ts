import { Receipt } from 'lucide-react'
import type { ReportSource } from './types'
import { floatBetween, generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const STATUS = [
  { value: 'paid', label: 'Bezahlt' },
  { value: 'sent', label: 'Versendet' },
  { value: 'draft', label: 'Entwurf' },
  { value: 'overdue', label: 'Überfällig' },
  { value: 'cancelled', label: 'Storniert' },
]
const CURRENCY = [
  { value: 'EUR', label: 'EUR' },
  { value: 'CHF', label: 'CHF' },
]
const TAX = [
  { value: '19', label: '19 %' },
  { value: '7', label: '7 %' },
  { value: '8.1', label: '8,1 %' },
  { value: '0', label: '0 %' },
]
const CUSTOMERS = [
  'Müller Maschinenbau GmbH',
  'Fintech Solutions AG',
  'Atelier Nord Design',
  'Pharma Vitalis GmbH',
  'Medienwerk Süd',
  'Bau & Plan AG',
  'Greentech Energie GmbH',
  'Logistik Rhein-Main',
  'Praxis Dr. Berger',
  'Handwerk Schmidt e.K.',
]

export const finanzenSource: ReportSource = {
  id: 'finanzen',
  module: 'finanzen',
  labelKey: 'berichte.sources.finanzen.label',
  label: 'Rechnungen',
  icon: Receipt,
  description: 'Faktura, Umsatz und Zahlungsstatus',
  defaultViz: 'bar',
  fields: [
    { key: 'issue_date', labelKey: 'berichte.fields.finanzen.issue_date', label: 'Rechnungsdatum', dataType: 'date', role: 'dimension' },
    { key: 'status', labelKey: 'berichte.fields.finanzen.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'customer_name', labelKey: 'berichte.fields.finanzen.customer_name', label: 'Kunde', dataType: 'string', role: 'dimension' },
    { key: 'currency', labelKey: 'berichte.fields.finanzen.currency', label: 'Währung', dataType: 'enum', role: 'dimension', enumValues: CURRENCY },
    { key: 'tax_rate', labelKey: 'berichte.fields.finanzen.tax_rate', label: 'Steuersatz', dataType: 'enum', role: 'dimension', enumValues: TAX },
    { key: 'total_net', labelKey: 'berichte.fields.finanzen.total_net', label: 'Netto-Betrag', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'total_gross', labelKey: 'berichte.fields.finanzen.total_gross', label: 'Brutto-Betrag', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
  ],
  sampleRows: (count = 140) =>
    generateRows(101, count, (r) => {
      const net = floatBetween(r(), 250, 18000, 2)
      const taxOpt = pick(TAX, r())
      const gross = Math.round(net * (1 + Number(taxOpt.value) / 100) * 100) / 100
      return {
        issue_date: isoDaysAgo(intBetween(r(), 0, 364)),
        status: pick(STATUS, r()).value,
        customer_name: pick(CUSTOMERS, r()),
        currency: pick(CURRENCY, r()).value,
        tax_rate: taxOpt.value,
        total_net: net,
        total_gross: gross,
      }
    }),
}
