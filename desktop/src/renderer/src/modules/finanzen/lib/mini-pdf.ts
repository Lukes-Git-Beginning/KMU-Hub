/**
 * Dependency-free minimal PDF generator (finanzen P2.5c).
 *
 * Erzeugt ein gültiges einseitiges PDF (Helvetica/WinAnsi) aus Textzeilen —
 * gut genug, damit „PDF herunterladen" im Demo eine echte, im Viewer öffnbare
 * Datei liefert, ohne eine PDF-Bibliothek zu bündeln. Beim Backend-Anbindung
 * ersetzt Lukes serverseitiges PDF (ZUGFeRD/XRechnung, P4) diesen Mock.
 *
 * Alle Nicht-ASCII-Zeichen werden auf ihren WinAnsi-Oktalcode escaped, damit
 * (a) Umlaute/€ korrekt im PDF erscheinen und (b) String-Länge == Byte-Länge
 * bleibt (für korrekte xref-Offsets).
 */

const WINANSI: Record<string, string> = {
  'ä': '\\344', 'ö': '\\366', 'ü': '\\374', 'Ä': '\\304', 'Ö': '\\326', 'Ü': '\\334',
  'ß': '\\337', '€': '\\200', '§': '\\247', '°': '\\260', '·': '\\267', '–': '-', '—': '-',
  '„': '"', '“': '"', '”': '"', '‚': "'", '‘': "'", '’': "'", ' ': ' ',
}

/** Escape a string for a PDF text literal (parentheses, backslash, non-ASCII). */
function pdfEscape(s: string): string {
  let out = ''
  for (const ch of s) {
    if (ch === '(' || ch === ')' || ch === '\\') out += '\\' + ch
    else if (ch in WINANSI) out += WINANSI[ch]
    else if (ch.charCodeAt(0) < 128) out += ch
    else out += '?' // unbekanntes Zeichen — Platzhalter statt kaputtes Encoding
  }
  return out
}

export interface PdfLine {
  text: string
  /** Optional: größer/fett rendern (z. B. Titel). */
  size?: number
  /** Vertikaler Extra-Abstand vor dieser Zeile (in pt). */
  gap?: number
}

/** Build a one-page A4 PDF from text lines. Returns the raw bytes. */
export function buildSimplePdf(lines: PdfLine[]): Uint8Array {
  // Content stream: start near top-left, step down per line.
  let content = 'BT\n'
  let first = true
  for (const line of lines) {
    const size = line.size ?? 11
    const lead = (line.gap ?? 0) + size + 6
    if (first) {
      content += `/F1 ${size} Tf\n1 0 0 1 56 786 Tm\n`
      first = false
    } else {
      content += `/F1 ${size} Tf\n`
    }
    content += `0 -${lead} Td\n(${pdfEscape(line.text)}) Tj\n`
  }
  content += 'ET'

  const objects = [
    '<</Type/Catalog/Pages 2 0 R>>',
    '<</Type/Pages/Kids[3 0 R]/Count 1>>',
    '<</Type/Page/Parent 2 0 R/MediaBox[0 0 595 842]/Resources<</Font<</F1 4 0 R>>>>/Contents 5 0 R>>',
    '<</Type/Font/Subtype/Type1/BaseFont/Helvetica/Encoding/WinAnsiEncoding>>',
    `<</Length ${content.length}>>\nstream\n${content}\nendstream`,
  ]

  let pdf = '%PDF-1.4\n'
  const offsets: number[] = []
  objects.forEach((body, i) => {
    offsets.push(pdf.length)
    pdf += `${i + 1} 0 obj\n${body}\nendobj\n`
  })
  const xrefStart = pdf.length
  pdf += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`
  for (const off of offsets) {
    pdf += `${String(off).padStart(10, '0')} 00000 n \n`
  }
  pdf += `trailer\n<</Size ${objects.length + 1}/Root 1 0 R>>\nstartxref\n${xrefStart}\n%%EOF`

  // String ist reines ASCII (Nicht-ASCII wurde escaped) → 1 Zeichen == 1 Byte.
  const bytes = new Uint8Array(pdf.length)
  for (let i = 0; i < pdf.length; i++) bytes[i] = pdf.charCodeAt(i) & 0xff
  return bytes
}
