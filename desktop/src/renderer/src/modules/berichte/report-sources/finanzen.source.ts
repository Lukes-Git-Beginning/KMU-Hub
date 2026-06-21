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
const DOC_TYPE = [
  { value: 'invoice', label: 'Rechnung' },
  { value: 'quote', label: 'Angebot' },
  { value: 'credit_note', label: 'Gutschrift' },
  { value: 'reminder', label: 'Mahnung' },
]
const PAYMENT_TERMS = [
  { value: '14', label: '14 Tage' },
  { value: '30', label: '30 Tage' },
  { value: '60', label: '60 Tage' },
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
    { key: 'due_date', labelKey: 'berichte.fields.finanzen.due_date', label: 'Fälligkeitsdatum', dataType: 'date', role: 'dimension' },
    { key: 'doc_type', labelKey: 'berichte.fields.finanzen.doc_type', label: 'Belegart', dataType: 'enum', role: 'dimension', enumValues: DOC_TYPE },
    { key: 'status', labelKey: 'berichte.fields.finanzen.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'customer_name', labelKey: 'berichte.fields.finanzen.customer_name', label: 'Kunde', dataType: 'string', role: 'dimension' },
    { key: 'currency', labelKey: 'berichte.fields.finanzen.currency', label: 'Währung', dataType: 'enum', role: 'dimension', enumValues: CURRENCY },
    { key: 'tax_rate', labelKey: 'berichte.fields.finanzen.tax_rate', label: 'Steuersatz', dataType: 'enum', role: 'dimension', enumValues: TAX },
    { key: 'payment_terms', labelKey: 'berichte.fields.finanzen.payment_terms', label: 'Zahlungsziel', dataType: 'enum', role: 'dimension', enumValues: PAYMENT_TERMS },
    { key: 'total_net', labelKey: 'berichte.fields.finanzen.total_net', label: 'Netto-Betrag', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'total_gross', labelKey: 'berichte.fields.finanzen.total_gross', label: 'Brutto-Betrag', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'tax', labelKey: 'berichte.fields.finanzen.tax', label: 'Steuerbetrag', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'days_to_payment', labelKey: 'berichte.fields.finanzen.days_to_payment', label: 'Zahlungsdauer (Tage)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
  ],
  sampleRows: (count = 140) =>
    generateRows(101, count, (r) => {
      const net = floatBetween(r(), 250, 18000, 2)
      const taxOpt = pick(TAX, r())
      const tax = Math.round(net * (Number(taxOpt.value) / 100) * 100) / 100
      const gross = Math.round((net + tax) * 100) / 100
      const terms = pick(PAYMENT_TERMS, r())
      const issueDaysAgo = intBetween(r(), 0, 364)
      return {
        issue_date: isoDaysAgo(issueDaysAgo),
        due_date: isoDaysAgo(issueDaysAgo - Number(terms.value)),
        doc_type: pick(DOC_TYPE, r()).value,
        status: pick(STATUS, r()).value,
        customer_name: pick(CUSTOMERS, r()),
        currency: pick(CURRENCY, r()).value,
        tax_rate: taxOpt.value,
        payment_terms: terms.value,
        total_net: net,
        total_gross: gross,
        tax,
        days_to_payment: intBetween(r(), -5, 45),
      }
    }),
}
