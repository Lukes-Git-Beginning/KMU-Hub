/**
 * CSV-Exports für das Inventar-Modul (Demo-Tiefe: Klick → echte Datei).
 *
 * Marktüblich (weclapp/Zoho/myfactory): Artikel-/Bestandsliste, Bewegungsjournal
 * und Inventur-Zählliste als CSV. Semikolon-getrennt wie die Buchhaltungs-Exports
 * (DACH-Excel öffnet das direkt); Download-Helfer wird wiederverwendet.
 */
import type { InventarItem, InventarMovement, InventurSession } from '@/api/inventar-types'
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

/** Artikel-/Bestandsliste (respektiert die aktive Filterung der Tabelle). */
export function buildItemsCsv(items: InventarItem[]): string {
  const rows: (string | number)[][] = [
    ['Name', 'SKU', 'Kategorie', 'Bestand', 'Einheit', 'Mindestbestand', 'Lagerort', 'Preis', 'Waehrung', 'Barcode', 'Charge'],
  ]
  for (const item of items) {
    rows.push([
      item.name,
      item.sku,
      item.category ?? '',
      Number(item.quantity),
      item.unit,
      Number(item.min_quantity),
      item.location ?? '',
      money(item.price),
      item.currency ?? 'EUR',
      item.barcode ?? '',
      item.batch_number ?? '',
    ])
  }
  return toCsv(rows)
}

/** Bewegungsjournal eines Artikels. */
export function buildMovementsCsv(movements: InventarMovement[], itemName: string): string {
  const rows: (string | number)[][] = [
    ['Datum', 'Artikel', 'Typ', 'Menge', 'Von', 'Nach', 'Referenz', 'Grund', 'Erfasst von'],
  ]
  for (const mov of movements) {
    rows.push([
      mov.created_at,
      itemName,
      mov.movement_type,
      Number(mov.quantity),
      mov.location_from ?? '',
      mov.location_to ?? '',
      mov.reference ?? '',
      mov.reason,
      mov.performed_by ?? '',
    ])
  }
  return toCsv(rows)
}

/** Inventur-Zählliste (Soll/Ist/Differenz) einer Session. */
export function buildInventurCsv(
  session: InventurSession,
  itemsById: Map<string, InventarItem>,
): string {
  const rows: (string | number)[][] = [
    ['Artikel', 'SKU', 'Soll', 'Ist', 'Differenz'],
  ]
  for (const count of session.counts) {
    const item = itemsById.get(count.item_id)
    const expected = Number(count.expected)
    const counted = count.counted === null ? null : Number(count.counted)
    rows.push([
      item?.name ?? count.item_id,
      item?.sku ?? '',
      expected,
      counted ?? '',
      counted === null ? '' : counted - expected,
    ])
  }
  return toCsv(rows)
}

/** yyyy-mm-dd für Dateinamen. */
export function csvDateStamp(): string {
  return new Date().toISOString().slice(0, 10)
}
