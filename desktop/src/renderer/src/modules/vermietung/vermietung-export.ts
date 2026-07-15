/**
 * CSV-Exports für das Vermietung-Modul (Demo-Tiefe: Klick → echte Datei).
 *
 * Marktüblich (Booqable/Rentman): Buchungs-/Objektliste als CSV.
 * Semikolon-getrennt (DACH-Excel), Download-Helfer wiederverwendet.
 */
import type { Rental, RentalObject } from '@/api/vermietung-types'
import { daysBetween } from './vermietung-shared'
export { downloadCsv } from '@/modules/finanzen/lib/finance-export'

function field(v: unknown): string {
  const s = String(v ?? '')
  return /[";\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

function toCsv(rows: (string | number)[][]): string {
  return rows.map((r) => r.map(field).join(';')).join('\r\n') + '\r\n'
}

function money(v: number | null | undefined): string {
  return (Number.isFinite(v as number) ? (v as number) : 0).toFixed(2).replace('.', ',')
}

/** Objektliste (respektiert die aktive Filterung). */
export function buildObjectsCsv(objects: RentalObject[]): string {
  const rows: (string | number)[][] = [
    ['Name', 'Typ', 'Standort', 'Tagespreis', 'Kaution', 'Aktiv', 'Beschreibung'],
  ]
  for (const obj of objects) {
    rows.push([
      obj.name,
      obj.category,
      obj.location ?? '',
      money(obj.daily_rate),
      money(obj.deposit),
      obj.active ? 'ja' : 'nein',
      obj.description ?? '',
    ])
  }
  return toCsv(rows)
}

/** Buchungsliste (respektiert Filter + Sortierung der Tabelle). */
export function buildRentalsCsv(rentals: Rental[], objectsById: Map<string, RentalObject>): string {
  const rows: (string | number)[][] = [
    ['Objekt', 'Mieter', 'Von', 'Bis', 'Tage', 'Status', 'Gesamtpreis', 'Kaution bezahlt', 'Notizen'],
  ]
  for (const r of rentals) {
    const startStr = r.start_date.slice(0, 10)
    const endStr = r.end_date.slice(0, 10)
    rows.push([
      objectsById.get(r.object_id)?.name ?? r.object_id,
      r.renter_name,
      startStr,
      endStr,
      daysBetween(startStr, endStr),
      r.status,
      money(r.total_price),
      r.deposit_paid ? 'ja' : 'nein',
      r.notes ?? '',
    ])
  }
  return toCsv(rows)
}

/** yyyy-mm-dd für Dateinamen. */
export function csvDateStamp(): string {
  return new Date().toISOString().slice(0, 10)
}
