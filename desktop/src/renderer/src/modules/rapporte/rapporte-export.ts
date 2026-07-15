/**
 * Echte (non-stub) Exports für das Rapporte-Modul.
 *
 * PDF: dependency-freier Mini-Generator nach dem mail-export.ts-Muster
 * (makeDemoPdf), erweitert um /WinAnsiEncoding + Latin-1-Byte-Ausgabe, damit
 * Umlaute im Tagesrapport korrekt erscheinen. Aufbau folgt dem markt-üblichen
 * Rapport-PDF (HERO/mfr/Craftboxx): Kopf, Zeiten, Arbeiter, Tätigkeiten,
 * Material, Notizen, Unterschrifts-Status.
 * CSV: Sammel-Export der Rapportliste (Semikolon, DACH-Excel).
 */
import type { FieldReport } from '@/stores/rapporte'
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

function netMinutes(r: FieldReport): number {
  const [sh, sm] = r.workStart.split(':').map(Number)
  const [eh, em] = r.workEnd.split(':').map(Number)
  return (eh * 60 + em) - (sh * 60 + sm) - r.breakMinutes
}

/** Sammel-Export der (gefilterten) Rapportliste. */
export function buildReportsCsv(reports: FieldReport[]): string {
  const rows: (string | number)[][] = [
    ['Datum', 'Projekt', 'Autor', 'Status', 'Arbeitsbeginn', 'Arbeitsende', 'Pause (min)', 'Netto (min)', 'Arbeiter', 'Taetigkeiten', 'Material'],
  ]
  for (const r of reports) {
    rows.push([
      r.date,
      r.projectName,
      r.author,
      r.approvalStatus,
      r.workStart,
      r.workEnd,
      r.breakMinutes,
      netMinutes(r),
      r.workers.map((w) => `${w.name} (${w.hours}h)`).join(', '),
      r.activities.map((a) => a.description).join('; '),
      r.materials.map((m) => `${m.article} ${m.quantity} ${m.unit}`).join('; '),
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

function fmtDate(dateStr: string): string {
  return new Date(dateStr + 'T00:00:00').toLocaleDateString('de-DE', {
    day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

/** Tagesrapport als echtes PDF (Markt-Layout: Kopf, Zeiten, Positionen, Unterschrift). */
export function buildReportPdf(report: FieldReport, labels: {
  title: string
  time: string
  workers: string
  activities: string
  materials: string
  notes: string
  signature: string
  signed: string
  unsigned: string
  status: string
}): Blob {
  const [sh, sm] = report.workStart.split(':').map(Number)
  const [eh, em] = report.workEnd.split(':').map(Number)
  const totalMin = (eh * 60 + em) - (sh * 60 + sm) - report.breakMinutes
  const net = `${Math.floor(totalMin / 60)}h ${totalMin % 60 > 0 ? (totalMin % 60) + 'min' : ''}`.trim()

  const lines: PdfLine[] = [
    { text: labels.title, size: 16, bold: true, gap: 10 },
    { text: `${report.projectName} · ${fmtDate(report.date)} · ${report.author}`, size: 10, gap: 20 },
    { text: `${labels.status}: ${report.approvalStatus}`, size: 9, gap: 14 },
    { text: labels.time, size: 12, bold: true, gap: 24 },
    { text: `${report.workStart} - ${report.workEnd} · Pause ${report.breakMinutes} min · Netto ${net}`, size: 10 },
    { text: labels.workers, size: 12, bold: true, gap: 24 },
    ...report.workers.map((w) => ({ text: `- ${w.name} (${w.role}) · ${w.hours}h`, size: 10 })),
    { text: labels.activities, size: 12, bold: true, gap: 24 },
    ...report.activities.map((a) => ({ text: `- ${a.description}${a.category ? ` [${a.category}]` : ''}`, size: 10 })),
  ]
  if (report.materials.length > 0) {
    lines.push({ text: labels.materials, size: 12, bold: true, gap: 24 })
    lines.push(...report.materials.map((m) => ({ text: `- ${m.article}: ${m.quantity} ${m.unit}`, size: 10 })))
  }
  if (report.notes) {
    lines.push({ text: labels.notes, size: 12, bold: true, gap: 24 })
    lines.push({ text: report.notes, size: 10 })
  }
  lines.push({ text: labels.signature, size: 12, bold: true, gap: 24 })
  lines.push({ text: report.signatureStatus === 'signed' ? labels.signed : labels.unsigned, size: 10 })

  return makePdf(lines)
}

/** yyyy-mm-dd für Dateinamen. */
export function csvDateStamp(): string {
  return new Date().toISOString().slice(0, 10)
}
