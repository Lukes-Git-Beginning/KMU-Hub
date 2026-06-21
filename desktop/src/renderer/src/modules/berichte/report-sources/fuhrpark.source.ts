import { Car } from 'lucide-react'
import type { ReportSource } from './types'
import { floatBetween, generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const FUEL = [
  { value: 'diesel', label: 'Diesel' },
  { value: 'petrol', label: 'Benzin' },
  { value: 'electric', label: 'Elektro' },
  { value: 'hybrid', label: 'Hybrid' },
]
const STATUS = [
  { value: 'active', label: 'Aktiv' },
  { value: 'service', label: 'In Werkstatt' },
  { value: 'idle', label: 'Stillgelegt' },
  { value: 'sold', label: 'Verkauft' },
]
const VCLASS = [
  { value: 'car', label: 'PKW' },
  { value: 'van', label: 'Transporter' },
  { value: 'truck', label: 'LKW' },
  { value: 'trailer', label: 'Anhänger' },
]
const LOCATION = [
  { value: 'muenchen', label: 'München' },
  { value: 'berlin', label: 'Berlin' },
  { value: 'hamburg', label: 'Hamburg' },
]
const MAKES = ['VW', 'Mercedes-Benz', 'BMW', 'Audi', 'Ford', 'Renault', 'Tesla', 'MAN']

export const fuhrparkSource: ReportSource = {
  id: 'fuhrpark',
  module: 'fuhrpark',
  labelKey: 'berichte.sources.fuhrpark.label',
  label: 'Fuhrpark',
  icon: Car,
  description: 'Fahrzeuge, Laufleistung und Kosten',
  defaultViz: 'bar',
  fields: [
    { key: 'make', labelKey: 'berichte.fields.fuhrpark.make', label: 'Hersteller', dataType: 'string', role: 'dimension' },
    { key: 'fuel_type', labelKey: 'berichte.fields.fuhrpark.fuel_type', label: 'Antrieb', dataType: 'enum', role: 'dimension', enumValues: FUEL },
    { key: 'vehicle_class', labelKey: 'berichte.fields.fuhrpark.vehicle_class', label: 'Fahrzeugklasse', dataType: 'enum', role: 'dimension', enumValues: VCLASS },
    { key: 'status', labelKey: 'berichte.fields.fuhrpark.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'location', labelKey: 'berichte.fields.fuhrpark.location', label: 'Standort', dataType: 'enum', role: 'dimension', enumValues: LOCATION },
    { key: 'acquisition_date', labelKey: 'berichte.fields.fuhrpark.acquisition_date', label: 'Anschaffung', dataType: 'date', role: 'dimension' },
    { key: 'mileage_km', labelKey: 'berichte.fields.fuhrpark.mileage_km', label: 'Kilometerstand', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'fuel_cost', labelKey: 'berichte.fields.fuhrpark.fuel_cost', label: 'Kraftstoffkosten', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'maintenance_cost', labelKey: 'berichte.fields.fuhrpark.maintenance_cost', label: 'Wartungskosten', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'insurance_cost', labelKey: 'berichte.fields.fuhrpark.insurance_cost', label: 'Versicherungskosten', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'co2_g_km', labelKey: 'berichte.fields.fuhrpark.co2_g_km', label: 'CO₂ (g/km)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
  ],
  sampleRows: (count = 40) =>
    generateRows(650, count, (r) => {
      const fuel = pick(FUEL, r())
      const electric = fuel.value === 'electric'
      return {
        make: pick(MAKES, r()),
        fuel_type: fuel.value,
        vehicle_class: pick(VCLASS, r()).value,
        status: pick(STATUS, r()).value,
        location: pick(LOCATION, r()).value,
        acquisition_date: isoDaysAgo(intBetween(r(), 90, 2920)),
        mileage_km: intBetween(r(), 5000, 220000),
        fuel_cost: electric ? floatBetween(r(), 300, 2400, 2) : floatBetween(r(), 1200, 9000, 2),
        maintenance_cost: floatBetween(r(), 200, 6000, 2),
        insurance_cost: floatBetween(r(), 600, 3500, 2),
        co2_g_km: electric ? 0 : intBetween(r(), 95, 210),
      }
    }),
}
