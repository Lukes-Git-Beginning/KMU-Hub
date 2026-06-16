/**
 * CSV-Export-Builder für die Buchhaltung (finanzen P2.5c).
 *
 * Erzeugt plausible DACH-CSV-Dateien (Semikolon-getrennt), damit die
 * Export-Buttons im Demo eine echte Datei liefern. Das ist NICHT der finale
 * DATEV-EXTF-Standard (Berater-/Mandanten-Nr, Konto-Mapping, Header-Zeile) —
 * der kommt in P3. Hier geht es um „Klick → echte CSV".
 */

interface ExportInvoice {
  invoice_number?: string
  number?: string
  customer?: { name?: string }
  customer_name?: string
  invoice_date?: string
  issue_date?: string
  date?: string
  total_net?: number | string
  total_gross?: number | string
  tax_breakdown?: { subtotal?: string; gross_total?: string }
  account?: string
}

function num(v: unknown): string {
  const n = typeof v === 'string' ? Number(v) : (v as number)
  return (Number.isFinite(n) ? n : 0).toFixed(2).replace('.', ',')
}

function field(v: unknown): string {
  const s = String(v ?? '')
  return /[";\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

function toCsv(rows: (string | number)[][]): string {
  return rows.map((r) => r.map(field).join(';')).join('\r\n') + '\r\n'
}

function invDate(inv: ExportInvoice): string {
  return inv.invoice_date ?? inv.issue_date ?? inv.date ?? ''
}
function invNo(inv: ExportInvoice): string {
  return inv.invoice_number ?? inv.number ?? ''
}
function invCustomer(inv: ExportInvoice): string {
  return inv.customer?.name ?? inv.customer_name ?? ''
}
function invNet(inv: ExportInvoice): number | string {
  return inv.tax_breakdown?.subtotal ?? inv.total_net ?? 0
}
function invGross(inv: ExportInvoice): number | string {
  return inv.tax_breakdown?.gross_total ?? inv.total_gross ?? 0
}

/** DATEV-EXTF-naher Buchungsstapel (vereinfacht — volle Spec = P3). */
export function buildDatevCsv(invoices: ExportInvoice[]): string {
  const rows: (string | number)[][] = [
    ['Umsatz', 'Soll/Haben', 'WKZ', 'Konto', 'Gegenkonto', 'Belegdatum', 'Belegfeld 1', 'Buchungstext'],
  ]
  for (const inv of invoices) {
    rows.push([
      num(invGross(inv)),
      'S',
      'EUR',
      '1400', // Forderungen aLL (Debitor-Sammelkonto, Demo)
      inv.account ?? '8400', // Erlöse 19% USt (Demo-Gegenkonto)
      invDate(inv).replaceAll('-', ''),
      invNo(inv),
      `Rechnung ${invCustomer(inv)}`,
    ])
  }
  return toCsv(rows)
}

/** Bexio-Import-CSV (Rechnungsliste). */
export function buildBexioCsv(invoices: ExportInvoice[]): string {
  const rows: (string | number)[][] = [
    ['Rechnungsnummer', 'Kunde', 'Datum', 'Netto', 'Brutto'],
  ]
  for (const inv of invoices) {
    rows.push([invNo(inv), invCustomer(inv), invDate(inv), num(invNet(inv)), num(invGross(inv))])
  }
  return toCsv(rows)
}

/** BMD-NTCS-CSV (AT). */
export function buildBmdCsv(invoices: ExportInvoice[]): string {
  const rows: (string | number)[][] = [
    ['Belegnr', 'Kunde', 'Belegdatum', 'Betrag', 'Konto'],
  ]
  for (const inv of invoices) {
    rows.push([invNo(inv), invCustomer(inv), invDate(inv), num(invGross(inv)), inv.account ?? '4000'])
  }
  return toCsv(rows)
}

/** Browser-Download eines CSV-Strings als Datei. */
export function downloadCsv(content: string, filename: string): void {
  const blob = new Blob(['﻿' + content], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
