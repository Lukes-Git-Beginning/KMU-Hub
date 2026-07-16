/**
 * Echte (non-stub) Exports für das Produktions-Modul.
 *
 * PDF: dependency-freier Mini-Generator nach dem rapporte/einkauf-Muster
 * (/WinAnsiEncoding + Latin-1-Bytes für Umlaute). Die Laufkarte folgt dem
 * markt-üblichen Arbeitspapier (Katana/MRPeasy/weclapp: Produktionsauftrag
 * drucken): Kopf mit PA-Nummer/Produkt/Menge/Terminen/Priorität, Stückliste
 * (Materialbedarf für die Auftragsmenge), Arbeitsschritte mit Zeitvorgaben.
 * CSV: Auftragslisten-Export (Controlling) + Stücklisten-Positionsexport
 * (Kommissionierung), Semikolon/BOM für DACH-Excel.
 */
import type {
  ProductionOrder,
  BOMResponse,
  WorkStepResponse,
  QualityCheckResponse,
} from '@/api/produktion-types'
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

function fmtDate(dateStr: string | null | undefined): string {
  if (!dateStr) return '—'
  return new Date(dateStr + (dateStr.length === 10 ? 'T00:00:00' : '')).toLocaleDateString('de-DE', {
    day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

/** Auftragsliste als CSV — eine Zeile pro Produktionsauftrag. */
export function buildOrdersCsv(
  orders: ProductionOrder[],
  statusLabels: Record<string, string>,
  priorityLabels: Record<number, string>,
  progressById: Map<string, number>,
): string {
  const rows: (string | number)[][] = [
    ['Auftragsnummer', 'Produkt', 'Menge', 'Status', 'Priorität', 'Fortschritt %', 'Start geplant', 'Fällig', 'Start Ist', 'Ende Ist', 'Notizen'],
  ]
  for (const o of orders) {
    rows.push([
      o.order_number,
      o.product_name,
      o.quantity,
      statusLabels[o.status] ?? o.status,
      priorityLabels[o.priority] ?? `P${o.priority}`,
      progressById.get(o.id) ?? (o.status === 'completed' ? 100 : 0),
      fmtDate(o.planned_start),
      fmtDate(o.planned_end),
      fmtDate(o.actual_start),
      fmtDate(o.actual_end),
      o.notes,
    ])
  }
  return toCsv(rows)
}

/** Stücklisten-Positionen als CSV (Kommissionierliste light). */
export function buildBomItemsCsv(bom: BOMResponse, orderQuantity?: number): string {
  const withDemand = typeof orderQuantity === 'number' && orderQuantity > 0
  const header: (string | number)[] = ['Nr.', 'Material', 'Menge je Stück', 'Einheit']
  if (withDemand) header.push(`Bedarf für ${orderQuantity} Stk`)
  const rows: (string | number)[][] = [header]
  const sorted = [...bom.items].sort((a, b) => a.sort_order - b.sort_order)
  sorted.forEach((item, idx) => {
    const row: (string | number)[] = [idx + 1, item.material_name, item.quantity, item.unit]
    if (withDemand) row.push(Math.round(item.quantity * (orderQuantity as number) * 100) / 100)
    rows.push(row)
  })
  return toCsv(rows)
}

// ---------------------------------------------------------------------------
// PDF (dependency-frei, WinAnsi für Umlaute)
// ---------------------------------------------------------------------------

/** Latin-1-safe: Zeichen außerhalb von WinAnsi werden ersetzt. */
function winAnsi(s: string): string {
  return s.replace(/[–—→·]/g, '-').replace(/[^\x20-\xFF]/g, '?')
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

export interface OrderPdfLabels {
  title: string
  product: string
  quantity: string
  status: string
  priority: string
  plannedStart: string
  plannedEnd: string
  bomSection: string
  bomEmpty: string
  perPiece: string
  demand: string
  stepsSection: string
  stepsEmpty: string
  duration: string
  assignee: string
  signature: string
  qualitySection: string
  qcPassed: string
  qcFailed: string
  qcDefects: (count: number) => string
  notes: string
  moreLines: (count: number) => string
}

/**
 * Laufkarte / Arbeitspapier als echtes PDF: Kopf, Materialbedarf laut
 * Stückliste, Arbeitsschritte mit Zeitvorgabe + Unterschriftslinie, QS-Historie.
 */
export function buildOrderPdf(
  order: ProductionOrder,
  bom: BOMResponse | undefined,
  steps: WorkStepResponse[],
  checks: QualityCheckResponse[],
  statusLabel: string,
  priorityLabel: string,
  labels: OrderPdfLabels,
): Blob {
  const pdfLines: PdfLine[] = [
    { text: `${labels.title} ${order.order_number}`, size: 16, bold: true, gap: 10 },
    { text: `${labels.product}: ${order.product_name}`, size: 11, gap: 18 },
    { text: `${labels.quantity}: ${order.quantity.toLocaleString('de-DE')} Stk - ${labels.status}: ${statusLabel} - ${labels.priority}: ${priorityLabel}`, size: 9 },
    { text: `${labels.plannedStart}: ${fmtDate(order.planned_start)} - ${labels.plannedEnd}: ${fmtDate(order.planned_end)}`, size: 9 },
  ]

  // Stückliste (Materialbedarf für die Auftragsmenge)
  pdfLines.push({ text: labels.bomSection, size: 11, bold: true, gap: 20 })
  if (!bom || bom.items.length === 0) {
    pdfLines.push({ text: labels.bomEmpty, size: 9, gap: 14 })
  } else {
    const MAX_BOM = 10
    const sorted = [...bom.items].sort((a, b) => a.sort_order - b.sort_order)
    for (const item of sorted.slice(0, MAX_BOM)) {
      const demand = Math.round(item.quantity * order.quantity * 100) / 100
      pdfLines.push({
        text: `${item.material_name}  -  ${labels.perPiece}: ${item.quantity} ${item.unit}  -  ${labels.demand}: ${demand.toLocaleString('de-DE')} ${item.unit}`,
        size: 9,
        gap: 13,
      })
    }
    if (sorted.length > MAX_BOM) {
      pdfLines.push({ text: labels.moreLines(sorted.length - MAX_BOM), size: 9, gap: 13 })
    }
  }

  // Arbeitsschritte mit Unterschriftslinie (Werkstatt-Laufkarte)
  pdfLines.push({ text: labels.stepsSection, size: 11, bold: true, gap: 20 })
  if (steps.length === 0) {
    pdfLines.push({ text: labels.stepsEmpty, size: 9, gap: 14 })
  } else {
    const MAX_STEPS = 8
    const sorted = [...steps].sort((a, b) => a.step_nr - b.step_nr)
    for (const step of sorted.slice(0, MAX_STEPS)) {
      const assignee = step.assignee ? ` - ${labels.assignee}: ${step.assignee}` : ''
      pdfLines.push({
        text: `${step.step_nr}. ${step.name} - ${labels.duration}: ${step.duration_minutes} min${assignee}`,
        size: 9,
        gap: 14,
      })
      pdfLines.push({ text: `   ${labels.signature}: ______________________`, size: 8, gap: 11 })
    }
    if (sorted.length > MAX_STEPS) {
      pdfLines.push({ text: labels.moreLines(sorted.length - MAX_STEPS), size: 9, gap: 13 })
    }
  }

  // QS-Historie (kompakt)
  if (checks.length > 0) {
    pdfLines.push({ text: labels.qualitySection, size: 11, bold: true, gap: 20 })
    for (const qc of checks.slice(0, 4)) {
      const result = qc.passed ? labels.qcPassed : `${labels.qcFailed}, ${labels.qcDefects(qc.defects_found)}`
      pdfLines.push({ text: `${fmtDate(qc.checked_at)} - ${qc.inspector} - ${result}`, size: 9, gap: 13 })
    }
  }

  if (order.notes) {
    pdfLines.push({ text: `${labels.notes}: ${order.notes.slice(0, 100)}`, size: 9, gap: 18 })
  }

  return makePdf(pdfLines)
}

/** yyyy-mm-dd für Dateinamen. */
export function csvDateStamp(): string {
  return new Date().toISOString().slice(0, 10)
}
