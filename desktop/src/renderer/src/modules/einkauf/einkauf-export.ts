/**
 * Echte (non-stub) Exports für das Einkauf-Modul.
 *
 * PDF: dependency-freier Mini-Generator nach dem rapporte/fuhrpark-Muster
 * (/WinAnsiEncoding + Latin-1-Bytes für Umlaute). Das Bestell-PDF folgt dem
 * markt-üblichen Layout (weclapp/Xentral: Beleg an den Lieferanten): Kopf mit
 * Bestellnummer/Lieferant/Terminen/Zahlungsbedingung, Positionstabelle,
 * Summenblock Netto/USt/Brutto.
 * CSV: Bestell- und Lieferantenlisten-Export (Semikolon, DACH-Excel).
 */
import type { PurchaseOrder, POLine, Supplier } from '@/api/einkauf-types'
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

function fmtAmount(v: number): string {
  return v.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

/** Bestellliste als CSV — eine Zeile pro Bestellung. */
export function buildPOsCsv(
  pos: PurchaseOrder[],
  supplierNameById: Map<string, string>,
  statusLabels: Record<string, string>,
): string {
  const rows: (string | number)[][] = [
    ['Bestellnummer', 'Lieferant', 'Status', 'Bestelldatum', 'Liefertermin', 'Betrag', 'Währung'],
  ]
  for (const po of pos) {
    rows.push([
      po.po_number,
      supplierNameById.get(po.supplier_id) ?? po.supplier_id,
      statusLabels[po.status] ?? po.status,
      fmtDate(po.order_date),
      fmtDate(po.expected_delivery_date),
      fmtAmount(parseFloat(po.total_amount)),
      po.currency || 'EUR',
    ])
  }
  return toCsv(rows)
}

/** Lieferantenliste als CSV. */
export function buildSuppliersCsv(
  suppliers: Supplier[],
  orderCountById: Map<string, number>,
): string {
  const rows: (string | number)[][] = [
    ['Name', 'E-Mail', 'Telefon', 'Adresse', 'Zahlungsbedingung', 'Notizen', 'Bestellungen'],
  ]
  for (const s of suppliers) {
    rows.push([
      s.name,
      s.email,
      s.phone,
      s.address,
      s.payment_terms,
      s.notes,
      orderCountById.get(s.id) ?? 0,
    ])
  }
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

export interface POPdfLabels {
  title: string
  supplier: string
  orderDate: string
  deliveryDate: string
  paymentTerms: string
  positions: string
  qty: string
  net: string
  vat: string
  gross: string
  notes: string
  noPositions: string
  morePositions: (count: number) => string
}

/** Bestellung als echtes PDF (Beleg-Layout: Kopf, Positionen, Summenblock). */
export function buildPOPdf(
  po: PurchaseOrder,
  lines: POLine[],
  supplierName: string,
  paymentTerms: string,
  labels: POPdfLabels,
): Blob {
  const currency = po.currency || 'EUR'
  const net = lines.reduce((s, l) => s + parseFloat(l.quantity) * parseFloat(l.unit_price), 0)
  const vat = lines.reduce(
    (s, l) => s + parseFloat(l.quantity) * parseFloat(l.unit_price) * parseFloat(l.tax_rate || '0'),
    0,
  )

  const pdfLines: PdfLine[] = [
    { text: `${labels.title} ${po.po_number}`, size: 16, bold: true, gap: 10 },
    { text: `${labels.supplier}: ${supplierName}`, size: 10, gap: 18 },
    { text: `${labels.orderDate}: ${fmtDate(po.order_date)} - ${labels.deliveryDate}: ${fmtDate(po.expected_delivery_date)}`, size: 9 },
    ...(paymentTerms ? [{ text: `${labels.paymentTerms}: ${paymentTerms}`, size: 9 }] : []),
    { text: labels.positions, size: 11, bold: true, gap: 22 },
  ]

  if (lines.length === 0) {
    pdfLines.push({ text: labels.noPositions, size: 10, gap: 16 })
  } else {
    // Single-page generator — cap positions to stay on the page.
    const MAX_LINES = 22
    const sorted = [...lines].sort((a, b) => a.line_position - b.line_position)
    for (const l of sorted.slice(0, MAX_LINES)) {
      const qty = parseFloat(l.quantity)
      const price = parseFloat(l.unit_price)
      pdfLines.push({
        text: `${l.product_name}${l.sku ? ` (${l.sku})` : ''}  -  ${labels.qty} ${qty.toLocaleString('de-DE')} x ${fmtAmount(price)} ${currency} = ${fmtAmount(qty * price)} ${currency}`,
        size: 9,
        gap: 14,
      })
    }
    if (sorted.length > MAX_LINES) {
      pdfLines.push({ text: labels.morePositions(sorted.length - MAX_LINES), size: 9, gap: 14 })
    }
  }

  pdfLines.push(
    { text: `${labels.net}: ${fmtAmount(net)} ${currency}`, size: 10, gap: 22 },
    { text: `${labels.vat}: ${fmtAmount(vat)} ${currency}`, size: 10 },
    { text: `${labels.gross}: ${fmtAmount(net + vat)} ${currency}`, size: 11, bold: true },
  )

  if (po.notes) {
    pdfLines.push({ text: `${labels.notes}: ${po.notes.slice(0, 100)}`, size: 9, gap: 20 })
  }

  return makePdf(pdfLines)
}

/** yyyy-mm-dd für Dateinamen. */
export function csvDateStamp(): string {
  return new Date().toISOString().slice(0, 10)
}
