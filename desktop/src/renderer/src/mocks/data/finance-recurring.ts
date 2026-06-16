/**
 * Mock seed data for recurring invoice schedules (finanzen P1).
 *
 * A recurring invoice is a template that emits draft invoices at a fixed
 * interval. The handler mutates `mockRecurringInvoices.recurring` in place so
 * pause/resume/generate feel real in the demo.
 */
import { daysAgo, daysFromNow, monthsAgo } from './date-helpers'
import type { LineItem, TaxBreakdown, RecurringInvoice } from '@/types/finance-types'

interface SimpleItem {
  description: string
  quantity: number
  unit_price: number
  tax_rate?: number
}

/** Build full line_items + tax_breakdown from a compact item list. */
function build(items: SimpleItem[]): { line_items: LineItem[]; tax_breakdown: TaxBreakdown } {
  const line_items: LineItem[] = items.map((it, idx) => ({
    id: `rec-li-${idx}`,
    position: idx + 1,
    description: it.description,
    quantity: String(it.quantity),
    unit_price: String(it.unit_price),
    tax_rate: String(it.tax_rate ?? 19),
    line_total: (it.quantity * it.unit_price).toFixed(2),
  }))
  const tax_by_rate: Record<string, number> = {}
  let subtotal = 0
  for (const it of items) {
    const net = it.quantity * it.unit_price
    subtotal += net
    const rate = it.tax_rate ?? 19
    tax_by_rate[rate] = (tax_by_rate[rate] ?? 0) + net * (rate / 100)
  }
  const totalTax = Object.values(tax_by_rate).reduce((s, v) => s + v, 0)
  return {
    line_items,
    tax_breakdown: {
      subtotal: subtotal.toFixed(2),
      tax_by_rate: Object.fromEntries(
        Object.entries(tax_by_rate).map(([k, v]) => [k, v.toFixed(2)]),
      ),
      total_tax: totalTax.toFixed(2),
      gross_total: (subtotal + totalTax).toFixed(2),
    },
  }
}

const SEED: RecurringInvoice[] = [
  {
    id: 'rec-001',
    title: 'CRM-Lizenz Enterprise (Jahresabo)',
    customer: {
      name: 'Gruber Maschinenbau GmbH',
      address: 'Industriestraße 8, 80939 München',
      email: 'buchhaltung@gruber-maschinenbau.de',
    },
    ...build([{ description: 'CRM Lizenz Enterprise — monatliche Rate', quantity: 1, unit_price: 790 }]),
    tax_mode: 'standard',
    currency: 'EUR',
    interval: 'monthly',
    status: 'active',
    start_date: monthsAgo(4),
    next_run: daysFromNow(12),
    payment_terms_days: 14,
    generated_count: 4,
    last_generated_at: monthsAgo(0),
    notes: 'Automatische monatliche Lizenzrechnung.',
    created_at: monthsAgo(4),
  },
  {
    id: 'rec-002',
    title: 'Support-Vertrag Premium (Quartal)',
    customer: {
      name: 'Bavaria Elektro AG',
      address: 'Maximilianstraße 14, 90402 Nürnberg',
      email: 'finance@bavaria-elektro.de',
    },
    ...build([
      { description: 'Premium-Support — Quartalspauschale', quantity: 1, unit_price: 1350 },
      { description: 'Reaktionszeit-SLA 4h', quantity: 1, unit_price: 450 },
    ]),
    tax_mode: 'standard',
    currency: 'EUR',
    interval: 'quarterly',
    status: 'active',
    start_date: monthsAgo(6),
    next_run: daysFromNow(3),
    payment_terms_days: 30,
    generated_count: 2,
    last_generated_at: monthsAgo(3),
    notes: '',
    created_at: monthsAgo(6),
  },
  {
    id: 'rec-003',
    title: 'Hosting & Wartung (monatlich)',
    customer: {
      name: 'Helvetia Software AG',
      address: 'Bahnhofstrasse 21, 8001 Zürich',
      email: 'rechnung@helvetia-software.ch',
    },
    ...build([{ description: 'EU-Hosting + Wartung — Monatspauschale', quantity: 1, unit_price: 340, tax_rate: 8.1 }]),
    tax_mode: 'standard',
    currency: 'CHF',
    interval: 'monthly',
    status: 'paused',
    start_date: monthsAgo(8),
    next_run: daysFromNow(8),
    payment_terms_days: 14,
    generated_count: 7,
    last_generated_at: monthsAgo(1),
    notes: 'Pausiert — Kunde prüft Tarifwechsel.',
    created_at: monthsAgo(8),
  },
  {
    id: 'rec-004',
    title: 'Wartungsvertrag (jährlich)',
    customer: {
      name: 'Rhein Consulting GmbH',
      address: 'Rheinuferstraße 9, 50667 Köln',
      email: 'buchhaltung@rhein-consulting.de',
    },
    ...build([{ description: 'Jahres-Wartungsvertrag CRM', quantity: 1, unit_price: 4800 }]),
    tax_mode: 'standard',
    currency: 'EUR',
    interval: 'yearly',
    status: 'ended',
    start_date: monthsAgo(13),
    end_date: daysAgo(20),
    next_run: daysAgo(20),
    payment_terms_days: 30,
    generated_count: 1,
    last_generated_at: monthsAgo(13),
    notes: 'Vertrag ausgelaufen — Verlängerung offen.',
    created_at: monthsAgo(13),
  },
]

export const mockRecurringInvoices: { recurring: RecurringInvoice[]; total: number } = {
  recurring: SEED,
  total: SEED.length,
}

/** Advance a YYYY-MM-DD date by one interval. */
export function advanceByInterval(dateISO: string, interval: RecurringInvoice['interval']): string {
  const d = new Date(dateISO + 'T00:00:00')
  switch (interval) {
    case 'weekly':
      d.setDate(d.getDate() + 7)
      break
    case 'monthly':
      d.setMonth(d.getMonth() + 1)
      break
    case 'quarterly':
      d.setMonth(d.getMonth() + 3)
      break
    case 'yearly':
      d.setFullYear(d.getFullYear() + 1)
      break
  }
  return d.toISOString().split('T')[0]
}
