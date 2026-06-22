/**
 * Real (non-stub) export/download/print helpers for the mail module.
 *
 * The demo has no real attachment bytes, so attachment downloads generate a
 * genuinely openable placeholder file (a valid one-page PDF for PDF types, a
 * text file otherwise). Message export produces a standard `.eml` file, and
 * print opens the browser print dialog with a clean, message-only layout —
 * all real artefacts, no toast theatre.
 */
import type { EmailMessageInfo, EmailAttachmentInfo } from '@/api/email-types'

// ---------------------------------------------------------------------------
// Low-level blob download
// ---------------------------------------------------------------------------

export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

// ---------------------------------------------------------------------------
// Minimal valid PDF generator (no external dependency)
// ---------------------------------------------------------------------------

/** Build a minimal, spec-valid single-page PDF. Returns a Blob you can download. */
export function makeDemoPdf(title: string, bodyLines: string[]): Blob {
  // PDF string literals escape backslash and parentheses; Helvetica is ASCII/WinAnsi.
  const esc = (s: string) => s.replace(/[\\()]/g, (c) => '\\' + c)
  const ascii = (s: string) => s.replace(/[^\x20-\x7E]/g, '?')
  const lines = [title, '', ...bodyLines].map((l) => ascii(l).slice(0, 95))

  let content = `BT\n/F1 16 Tf\n50 790 Td\n(${esc(lines[0])}) Tj\n/F1 11 Tf\n`
  for (let i = 1; i < lines.length; i++) {
    content += `0 -18 Td\n(${esc(lines[i])}) Tj\n`
  }
  content += 'ET'

  const objects: Record<number, string> = {
    1: '<< /Type /Catalog /Pages 2 0 R >>',
    2: '<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
    3: '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>',
    4: '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>',
    5: `<< /Length ${content.length} >>\nstream\n${content}\nendstream`,
  }

  let pdf = '%PDF-1.4\n'
  const offsets: number[] = []
  for (let i = 1; i <= 5; i++) {
    offsets[i] = pdf.length
    pdf += `${i} 0 obj\n${objects[i]}\nendobj\n`
  }
  const xrefStart = pdf.length
  pdf += 'xref\n0 6\n0000000000 65535 f \n'
  for (let i = 1; i <= 5; i++) {
    pdf += String(offsets[i]).padStart(10, '0') + ' 00000 n \n'
  }
  pdf += `trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n${xrefStart}\n%%EOF`
  return new Blob([pdf], { type: 'application/pdf' })
}

// ---------------------------------------------------------------------------
// Attachment download
// ---------------------------------------------------------------------------

/** Download a (demo) attachment as a real, openable file. */
export function downloadAttachment(att: EmailAttachmentInfo, fromMessageSubject?: string): void {
  const isPdf = att.content_type?.includes('pdf') || att.filename.toLowerCase().endsWith('.pdf')
  if (isPdf) {
    const blob = makeDemoPdf(att.filename.replace(/\.pdf$/i, ''), [
      fromMessageSubject ? `Aus E-Mail: ${fromMessageSubject}` : 'Cosmi Mail — Demo-Anhang',
      '',
      'Dies ist ein generierter Demo-Anhang.',
      'Im Produktivbetrieb wird hier die Originaldatei heruntergeladen.',
      '',
      `Dateigröße (Demo): ${(att.size_bytes / 1024).toFixed(0)} KB`,
    ])
    downloadBlob(blob, att.filename)
  } else {
    const blob = new Blob(
      [`${att.filename}\n\nCosmi Mail — Demo-Anhang.\nIm Produktivbetrieb steht hier der Originalinhalt.`],
      { type: att.content_type || 'text/plain' },
    )
    downloadBlob(blob, att.filename)
  }
}

// ---------------------------------------------------------------------------
// Message export (.eml)
// ---------------------------------------------------------------------------

function fmtAddr(a: { name: string; email: string }): string {
  return a.name ? `${a.name} <${a.email}>` : a.email
}

/** Export a message as a standard RFC-822 `.eml` file. */
export function exportMessageEml(msg: EmailMessageInfo): void {
  const headers = [
    `Date: ${new Date(msg.date).toUTCString()}`,
    `From: ${fmtAddr(msg.from)}`,
    `To: ${msg.to.map(fmtAddr).join(', ')}`,
    msg.cc.length ? `Cc: ${msg.cc.map(fmtAddr).join(', ')}` : '',
    `Subject: ${msg.subject}`,
    `Message-ID: ${msg.message_id_header}`,
    'MIME-Version: 1.0',
    'Content-Type: text/html; charset=utf-8',
  ].filter(Boolean)
  const body = msg.body_html || `<pre>${msg.body_text}</pre>`
  const eml = `${headers.join('\r\n')}\r\n\r\n${body}\r\n`
  const safe = (msg.subject || 'email').replace(/[^\p{L}\p{N}\-_ ]/gu, '').slice(0, 60).trim() || 'email'
  downloadBlob(new Blob([eml], { type: 'message/rfc822' }), `${safe}.eml`)
}

// ---------------------------------------------------------------------------
// Print
// ---------------------------------------------------------------------------

/** Open the browser print dialog with a clean, message-only layout. */
export function printMessage(msg: EmailMessageInfo): void {
  const frame = document.createElement('iframe')
  frame.style.position = 'fixed'
  frame.style.right = '0'
  frame.style.bottom = '0'
  frame.style.width = '0'
  frame.style.height = '0'
  frame.style.border = '0'
  document.body.appendChild(frame)

  const doc = frame.contentWindow?.document
  if (!doc) {
    document.body.removeChild(frame)
    return
  }

  const meta = `${msg.from.name || msg.from.email} · ${new Date(msg.date).toLocaleString('de-DE')}`
  doc.open()
  doc.write(`<!doctype html><html><head><meta charset="utf-8"><title>${msg.subject}</title>
    <style>
      body { font-family: -apple-system, system-ui, sans-serif; color: #1a1a1a; margin: 40px; line-height: 1.55; }
      h1 { font-size: 20px; margin: 0 0 4px; }
      .meta { color: #666; font-size: 13px; margin-bottom: 24px; border-bottom: 1px solid #ddd; padding-bottom: 16px; }
      .body { font-size: 14px; }
    </style></head><body>
    <h1>${msg.subject}</h1>
    <div class="meta">${meta}<br>An: ${msg.to.map((a) => a.name || a.email).join(', ')}</div>
    <div class="body">${msg.body_html || `<pre>${msg.body_text}</pre>`}</div>
  </body></html>`)
  doc.close()

  const win = frame.contentWindow
  if (win) {
    win.focus()
    win.print()
  }
  // Give the print dialog time to read the document before teardown.
  setTimeout(() => {
    if (frame.parentNode) document.body.removeChild(frame)
  }, 1000)
}
