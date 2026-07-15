/**
 * Echte (non-stub) Exports für das Schichten-Modul.
 *
 * PDF: dependency-freier Mini-Generator nach dem rapporte-export.ts-Muster
 * (/WinAnsiEncoding + Latin-1-Bytes für Umlaute). Layout folgt dem markt-
 * üblichen Dienstplan-PDF (Planday/Deputy/Shiftbase): Kopf mit KW/Zeitraum,
 * pro Mitarbeiter die zugewiesenen Schichten der Woche, Summenzeile.
 * CSV: eine Zeile pro Zuweisung (Semikolon, DACH-Excel).
 */
import type { ShiftAssignment, ShiftTemplate } from '@/stores/schichten'
import {
  getSurchargeLabel,
  isHoliday,
  computeShiftHours,
  WEEKDAYS,
  type SchichtenEmployee,
} from './schichten-shared'
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

function weekdayLabel(dateStr: string): string {
  const idx = (new Date(dateStr + 'T00:00:00').getDay() + 6) % 7
  return WEEKDAYS[idx]
}

function fmtDate(dateStr: string): string {
  return new Date(dateStr + 'T00:00:00').toLocaleDateString('de-DE', {
    day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

/** Wochen-Zuweisungen als CSV — eine Zeile pro Schicht (Marktstandard). */
export function buildWeekCsv(
  assignments: ShiftAssignment[],
  employees: SchichtenEmployee[],
  templateMap: Map<string, ShiftTemplate>,
): string {
  const roleById = new Map(employees.map((e) => [e.id, e.role]))
  const rows: (string | number)[][] = [
    ['Datum', 'Wochentag', 'Mitarbeiter', 'Rolle', 'Schicht', 'Beginn', 'Ende', 'Pause (min)', 'Stunden', 'Zuschlag', 'Status'],
  ]
  const sorted = [...assignments].sort(
    (a, b) => a.date.localeCompare(b.date) || a.userName.localeCompare(b.userName),
  )
  for (const a of sorted) {
    const tpl = templateMap.get(a.templateId)
    const surcharge = getSurchargeLabel(a.templateId, a.date)
    rows.push([
      fmtDate(a.date),
      weekdayLabel(a.date),
      a.userName,
      roleById.get(a.userId) ?? '',
      a.templateName,
      tpl?.startTime ?? '',
      tpl?.endTime ?? '',
      tpl?.breakMinutes ?? '',
      computeShiftHours(a.templateId).toFixed(2).replace('.', ','),
      surcharge ? `${surcharge.rate} ${surcharge.label}` : '',
      a.status === 'confirmed' ? 'Veröffentlicht' : 'Entwurf',
    ])
  }
  return toCsv(rows)
}

// ---------------------------------------------------------------------------
// PDF (dependency-frei, WinAnsi für Umlaute)
// ---------------------------------------------------------------------------

/** Latin-1-safe: Zeichen außerhalb von WinAnsi werden ersetzt. */
function winAnsi(s: string): string {
  return s.replace(/[–—]/g, '-').replace(/[^\x20-\xFF]/g, '?')
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

export interface WeekPlanPdfLabels {
  title: string
  employees: string
  hours: string
  occupancy: string
  noShifts: string
  draft: string
  published: string
}

/** Wochen-Dienstplan als echtes PDF (Kopf, pro Mitarbeiter die Schichten, Summen). */
export function buildWeekPlanPdf(
  kw: number,
  dateRange: string,
  employees: SchichtenEmployee[],
  assignments: ShiftAssignment[],
  templateMap: Map<string, ShiftTemplate>,
  stats: { occupancyRate: number; totalHours: number },
  labels: WeekPlanPdfLabels,
): Blob {
  const lines: PdfLine[] = [
    { text: `${labels.title} — KW ${kw}`, size: 16, bold: true, gap: 10 },
    { text: dateRange, size: 10, gap: 16 },
    {
      text: `${labels.employees}: ${employees.length} · ${labels.hours}: ${stats.totalHours.toFixed(1)}h · ${labels.occupancy}: ${stats.occupancyRate}%`,
      size: 9,
      gap: 14,
    },
  ]

  for (const emp of employees) {
    const empAssignments = assignments
      .filter((a) => a.userId === emp.id)
      .sort((a, b) => a.date.localeCompare(b.date))
    const weekHours = empAssignments.reduce((s, a) => s + computeShiftHours(a.templateId), 0)

    lines.push({
      text: `${emp.name} (${emp.role}) · ${weekHours.toFixed(1)}h`,
      size: 11,
      bold: true,
      gap: 20,
    })
    if (empAssignments.length === 0) {
      lines.push({ text: `  ${labels.noShifts}`, size: 9 })
      continue
    }
    for (const a of empAssignments) {
      const tpl = templateMap.get(a.templateId)
      const surcharge = getSurchargeLabel(a.templateId, a.date)
      const holiday = isHoliday(a.date)
      const parts = [
        `${weekdayLabel(a.date)} ${fmtDate(a.date)}:`,
        a.templateName,
        tpl ? `${tpl.startTime}-${tpl.endTime}` : '',
        a.status === 'confirmed' ? `[${labels.published}]` : `[${labels.draft}]`,
      ]
      if (surcharge) parts.push(`${surcharge.rate} ${surcharge.label}`)
      if (holiday) parts.push(`(${holiday})`)
      lines.push({ text: `  ${parts.filter(Boolean).join(' ')}`, size: 9 })
    }
  }

  return makePdf(lines)
}

/** yyyy-mm-dd für Dateinamen. */
export function csvDateStamp(): string {
  return new Date().toISOString().slice(0, 10)
}
