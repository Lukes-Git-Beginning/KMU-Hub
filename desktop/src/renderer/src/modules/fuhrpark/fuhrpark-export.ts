/**
 * Echte (non-stub) Exports für das Fuhrpark-Modul.
 *
 * PDF: dependency-freier Mini-Generator nach dem rapporte-export.ts-Muster
 * (/WinAnsiEncoding + Latin-1-Bytes für Umlaute). Das Fahrtenbuch-PDF folgt
 * dem finanzamts-üblichen Layout (Vimcar/Fleetster): Zeitraum, Summen
 * geschäftlich/privat, pro Fahrt Datum/Route/Zweck/km-Stände/Art/Fahrer.
 * CSV: Fahrtenbuch- und Fahrzeuglisten-Export (Semikolon, DACH-Excel).
 */
import type { Vehicle, LogbookEntry } from '@/stores/fuhrpark'
export { downloadBlob } from '@/modules/mails/lib/mail-export'

// ---------------------------------------------------------------------------
// CSV
// ---------------------------------------------------------------------------

function field(v: unknown): string {
  const s = String(v ?? '')
  return /[";\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

function toCsv(rows: (string | number)[][]): string {
  return rows.map((r) => r.map(field).join(';')).join('\r\n') + '\r\n'
}

function fmtDate(dateStr: string): string {
  return new Date(dateStr + (dateStr.length === 10 ? 'T00:00:00' : '')).toLocaleDateString('de-DE', {
    day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

export interface LogbookCsvLabels {
  business: string
  privateTrip: string
}

/** Fahrtenbuch als CSV — eine Zeile pro Fahrt (finanzamts-Spalten). */
export function buildLogbookCsv(
  entries: LogbookEntry[],
  plateByVehicleId: Map<string, string>,
  labels: LogbookCsvLabels,
): string {
  const rows: (string | number)[][] = [
    ['Datum', 'Fahrzeug', 'Fahrer', 'Start', 'Ziel', 'Zweck', 'km-Stand Beginn', 'km-Stand Ende', 'km', 'Art'],
  ]
  const sorted = [...entries].sort((a, b) => a.date.localeCompare(b.date))
  for (const e of sorted) {
    rows.push([
      fmtDate(e.date),
      plateByVehicleId.get(e.vehicleId) ?? e.vehiclePlate,
      e.driver,
      e.startLocation,
      e.endLocation,
      e.purpose,
      e.startKm,
      e.endKm,
      e.km,
      e.isPrivate ? labels.privateTrip : labels.business,
    ])
  }
  return toCsv(rows)
}

/** Fahrzeugliste als CSV. */
export function buildVehiclesCsv(vehicles: Vehicle[], typeLabels: Record<string, string>): string {
  const rows: (string | number)[][] = [
    ['Kennzeichen', 'Marke', 'Modell', 'Baujahr', 'Typ', 'Fahrer', 'km-Stand', 'Nächste Prüfung', 'Versicherung bis', 'Status'],
  ]
  for (const v of vehicles) {
    rows.push([
      v.licensePlate,
      v.make,
      v.model,
      v.year,
      typeLabels[v.type] ?? v.type,
      v.currentDriver,
      v.mileage,
      fmtDate(v.nextInspection),
      fmtDate(v.insuranceExpiry),
      v.isActive ? 'Aktiv' : 'Inaktiv',
    ])
  }
  return toCsv(rows)
}

// ---------------------------------------------------------------------------
// PDF (dependency-frei, WinAnsi für Umlaute)
// ---------------------------------------------------------------------------

/** Latin-1-safe: Zeichen außerhalb von WinAnsi werden ersetzt. */
function winAnsi(s: string): string {
  return s.replace(/[–—→]/g, '-').replace(/[^\x20-\xFF]/g, '?')
}

function escPdf(s: string): string {
  return s.replace(/[\\()]/g, (c) => '\\' + c)
}

interface PdfLine {
  text: string
  size?: number
  bold?: boolean
  gap?: number
}

/** Build a single-page, spec-valid PDF from styled text lines (Latin-1). */
function makePdf(lines: PdfLine[]): Blob {
  let content = 'BT\n50 800 Td\n'
  for (const line of lines) {
    const size = line.size ?? 10
    const font = line.bold ? '/F2' : '/F1'
    const gap = line.gap ?? size + 5
    content += `0 -${gap} Td\n${font} ${size} Tf\n(${escPdf(winAnsi(line.text).slice(0, 110))}) Tj\n`
  }
  content += 'ET'

  const objects: Record<number, string> = {
    1: '<< /Type /Catalog /Pages 2 0 R >>',
    2: '<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
    3: '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R /F2 6 0 R >> >> /Contents 5 0 R >>',
    4: '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>',
    5: `<< /Length ${content.length} >>\nstream\n${content}\nendstream`,
    6: '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>',
  }

  let pdf = '%PDF-1.4\n'
  const offsets: number[] = []
  for (let i = 1; i <= 6; i++) {
    offsets[i] = pdf.length
    pdf += `${i} 0 obj\n${objects[i]}\nendobj\n`
  }
  const xrefStart = pdf.length
  pdf += 'xref\n0 7\n0000000000 65535 f \n'
  for (let i = 1; i <= 6; i++) {
    pdf += String(offsets[i]).padStart(10, '0') + ' 00000 n \n'
  }
  pdf += `trailer\n<< /Size 7 /Root 1 0 R >>\nstartxref\n${xrefStart}\n%%EOF`

  // Emit as Latin-1 bytes (a plain string Blob would be UTF-8-encoded and
  // break the WinAnsi umlauts).
  const bytes = new Uint8Array(pdf.length)
  for (let i = 0; i < pdf.length; i++) {
    const code = pdf.charCodeAt(i)
    bytes[i] = code > 255 ? 63 : code
  }
  return new Blob([bytes], { type: 'application/pdf' })
}

export interface LogbookPdfLabels {
  title: string
  period: string
  businessKm: string
  privateKm: string
  totalKm: string
  business: string
  privateTrip: string
  noEntries: string
}

/** Fahrtenbuch als echtes PDF (finanzamts-Layout: Kopf, Summen, pro Fahrt zwei Zeilen). */
export function buildLogbookPdf(
  entries: LogbookEntry[],
  plateByVehicleId: Map<string, string>,
  labels: LogbookPdfLabels,
): Blob {
  const sorted = [...entries].sort((a, b) => a.date.localeCompare(b.date))
  const businessKm = sorted.filter((e) => !e.isPrivate).reduce((s, e) => s + e.km, 0)
  const privateKm = sorted.filter((e) => e.isPrivate).reduce((s, e) => s + e.km, 0)
  const from = sorted[0]?.date
  const to = sorted[sorted.length - 1]?.date

  const lines: PdfLine[] = [
    { text: labels.title, size: 16, bold: true, gap: 10 },
    { text: from && to ? `${labels.period}: ${fmtDate(from)} - ${fmtDate(to)}` : labels.period, size: 10, gap: 16 },
    {
      text: `${labels.businessKm}: ${businessKm.toLocaleString('de-DE')} km · ${labels.privateKm}: ${privateKm.toLocaleString('de-DE')} km · ${labels.totalKm}: ${(businessKm + privateKm).toLocaleString('de-DE')} km`,
      size: 9,
      gap: 14,
    },
  ]

  if (sorted.length === 0) {
    lines.push({ text: labels.noEntries, size: 10, gap: 20 })
    return makePdf(lines)
  }

  // Single-page generator — cap at 18 trips (2 lines each) to stay on the page.
  const MAX_TRIPS = 18
  for (const e of sorted.slice(0, MAX_TRIPS)) {
    const plate = plateByVehicleId.get(e.vehicleId) ?? e.vehiclePlate
    lines.push({
      text: `${fmtDate(e.date)} · ${plate} · ${e.startKm.toLocaleString('de-DE')} - ${e.endKm.toLocaleString('de-DE')} km (${e.km} km) · ${e.isPrivate ? labels.privateTrip : labels.business}`,
      size: 10,
      bold: true,
      gap: 18,
    })
    lines.push({
      text: `  ${e.startLocation} -> ${e.endLocation} · ${e.purpose}${e.driver ? ` · ${e.driver}` : ''}`,
      size: 9,
    })
  }
  if (sorted.length > MAX_TRIPS) {
    lines.push({ text: `+ ${sorted.length - MAX_TRIPS} weitere Fahrten (siehe CSV-Export)`, size: 9, gap: 16 })
  }

  return makePdf(lines)
}

/** yyyy-mm-dd für Dateinamen. */
export function csvDateStamp(): string {
  return new Date().toISOString().slice(0, 10)
}
