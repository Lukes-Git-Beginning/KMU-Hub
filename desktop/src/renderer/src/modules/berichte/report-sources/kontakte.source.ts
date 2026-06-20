import { Contact } from 'lucide-react'
import type { ReportSource } from './types'
import { floatBetween, generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const STAGE = [
  { value: 'lead', label: 'Lead' },
  { value: 'qualified', label: 'Qualifiziert' },
  { value: 'proposal', label: 'Angebot' },
  { value: 'negotiation', label: 'Verhandlung' },
  { value: 'won', label: 'Gewonnen' },
  { value: 'lost', label: 'Verloren' },
]
const INDUSTRY = [
  { value: 'maschinenbau', label: 'Maschinenbau' },
  { value: 'fintech', label: 'Fintech' },
  { value: 'design', label: 'Design' },
  { value: 'medien', label: 'Medien' },
  { value: 'pharma', label: 'Pharma' },
  { value: 'handwerk', label: 'Handwerk' },
]
const COUNTRY = [
  { value: 'DE', label: 'Deutschland' },
  { value: 'CH', label: 'Schweiz' },
  { value: 'AT', label: 'Österreich' },
]
const OWNERS = ['Stefan Vogel', 'Lena Hofer', 'Marco Reis', 'Julia Brandt', 'Tom Keller']

export const kontakteSource: ReportSource = {
  id: 'kontakte',
  module: 'crm',
  labelKey: 'berichte.sources.kontakte.label',
  label: 'Kontakte & Deals',
  icon: Contact,
  description: 'Pipeline, Leads und Abschlüsse',
  defaultViz: 'bar',
  fields: [
    { key: 'created_at', labelKey: 'berichte.fields.kontakte.created_at', label: 'Erstellt am', dataType: 'date', role: 'dimension' },
    { key: 'stage', labelKey: 'berichte.fields.kontakte.stage', label: 'Pipeline-Stufe', dataType: 'enum', role: 'dimension', enumValues: STAGE },
    { key: 'industry', labelKey: 'berichte.fields.kontakte.industry', label: 'Branche', dataType: 'enum', role: 'dimension', enumValues: INDUSTRY },
    { key: 'country', labelKey: 'berichte.fields.kontakte.country', label: 'Land', dataType: 'enum', role: 'dimension', enumValues: COUNTRY },
    { key: 'owner', labelKey: 'berichte.fields.kontakte.owner', label: 'Verantwortlich', dataType: 'string', role: 'dimension' },
    { key: 'value', labelKey: 'berichte.fields.kontakte.value', label: 'Deal-Wert', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'probability', labelKey: 'berichte.fields.kontakte.probability', label: 'Wahrscheinlichkeit', dataType: 'number', role: 'measure', format: 'percent', defaultAgg: 'avg' },
  ],
  sampleRows: (count = 120) =>
    generateRows(202, count, (r) => {
      const stage = pick(STAGE, r())
      const prob =
        stage.value === 'won' ? 100 : stage.value === 'lost' ? 0 : intBetween(r(), 10, 90)
      return {
        created_at: isoDaysAgo(intBetween(r(), 0, 364)),
        stage: stage.value,
        industry: pick(INDUSTRY, r()).value,
        country: pick(COUNTRY, r()).value,
        owner: pick(OWNERS, r()),
        value: floatBetween(r(), 1500, 95000, 0),
        probability: prob,
      }
    }),
}
