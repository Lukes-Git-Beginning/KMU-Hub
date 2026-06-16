import { IDS } from './shared-ids'
import { daysAgo, daysFromNow, monthsAgo, today } from './date-helpers'

// ---------------------------------------------------------------------------
// Recurring-generated invoices (finanzen P2.5e)
// Link a few seed invoices back to their recurring profile (rec-00x) so the
// "Erzeugte Rechnungen"-Liste im RecurringDetailPanel is populated on load.
// ---------------------------------------------------------------------------

interface RecurringSeed {
  seq: number
  recurring_id: string
  date: string
  status: 'paid' | 'sent'
  customer: { id: string; name: string }
  currency: string
  tax_rate: number
  items: { description: string; quantity: number; unit_price: number; total: number }[]
}

function buildRecurringInvoice(s: RecurringSeed) {
  const net = s.items.reduce((sum, it) => sum + it.total, 0)
  const gross = Math.round(net * (1 + s.tax_rate / 100) * 100) / 100
  const num = `RE-2026-${String(s.seq).padStart(3, '0')}`
  return {
    id: `inv-${s.seq}`,
    number: num,
    invoice_number: num,
    status: s.status,
    customer: s.customer,
    customer_name: s.customer.name,
    customer_id: s.customer.id,
    issue_date: s.date,
    invoice_date: s.date,
    due_date: s.date,
    total_net: net,
    total_gross: gross,
    tax_rate: s.tax_rate,
    currency: s.currency,
    recurring_id: s.recurring_id,
    items: s.items,
    notes: 'Automatisch erzeugt aus Abo',
    created_at: s.date,
  }
}

const recurringGeneratedInvoices = [
  buildRecurringInvoice({ seq: 13, recurring_id: 'rec-001', date: monthsAgo(2), status: 'paid', customer: { id: IDS.companies.gruberMaschinenbau, name: 'Gruber Maschinenbau GmbH' }, currency: 'EUR', tax_rate: 19, items: [{ description: 'CRM Lizenz Enterprise — monatliche Rate', quantity: 1, unit_price: 790, total: 790 }] }),
  buildRecurringInvoice({ seq: 14, recurring_id: 'rec-001', date: monthsAgo(1), status: 'paid', customer: { id: IDS.companies.gruberMaschinenbau, name: 'Gruber Maschinenbau GmbH' }, currency: 'EUR', tax_rate: 19, items: [{ description: 'CRM Lizenz Enterprise — monatliche Rate', quantity: 1, unit_price: 790, total: 790 }] }),
  buildRecurringInvoice({ seq: 15, recurring_id: 'rec-002', date: monthsAgo(3), status: 'paid', customer: { id: IDS.companies.bavariaElektro, name: 'Bavaria Elektro AG' }, currency: 'EUR', tax_rate: 19, items: [{ description: 'Premium-Support — Quartalspauschale', quantity: 1, unit_price: 1350, total: 1350 }, { description: 'Reaktionszeit-SLA 4h', quantity: 1, unit_price: 450, total: 450 }] }),
  buildRecurringInvoice({ seq: 16, recurring_id: 'rec-002', date: today(), status: 'sent', customer: { id: IDS.companies.bavariaElektro, name: 'Bavaria Elektro AG' }, currency: 'EUR', tax_rate: 19, items: [{ description: 'Premium-Support — Quartalspauschale', quantity: 1, unit_price: 1350, total: 1350 }, { description: 'Reaktionszeit-SLA 4h', quantity: 1, unit_price: 450, total: 450 }] }),
  buildRecurringInvoice({ seq: 17, recurring_id: 'rec-003', date: monthsAgo(2), status: 'paid', customer: { id: IDS.companies.helvetiaSoftware, name: 'Helvetia Software AG' }, currency: 'CHF', tax_rate: 8.1, items: [{ description: 'EU-Hosting + Wartung — Monatspauschale', quantity: 1, unit_price: 340, total: 340 }] }),
  buildRecurringInvoice({ seq: 18, recurring_id: 'rec-003', date: monthsAgo(1), status: 'paid', customer: { id: IDS.companies.helvetiaSoftware, name: 'Helvetia Software AG' }, currency: 'CHF', tax_rate: 8.1, items: [{ description: 'EU-Hosting + Wartung — Monatspauschale', quantity: 1, unit_price: 340, total: 340 }] }),
  buildRecurringInvoice({ seq: 19, recurring_id: 'rec-004', date: monthsAgo(13), status: 'paid', customer: { id: IDS.companies.rheinConsulting, name: 'Rhein Consulting GmbH' }, currency: 'EUR', tax_rate: 19, items: [{ description: 'Jahres-Wartungsvertrag CRM', quantity: 1, unit_price: 4800, total: 4800 }] }),
]

// ---------------------------------------------------------------------------
// Invoices (12 base + recurring-generated)
// ---------------------------------------------------------------------------

export const mockInvoices = {
  invoices: [
    // --- 4 PAID ---
    {
      id: IDS.invoices.inv001,
      number: 'RE-2026-001',
      invoice_number: 'RE-2026-001',
      status: 'paid' as const,
      customer: { id: IDS.companies.gruberMaschinenbau, name: 'Gruber Maschinenbau GmbH' },
      customer_name: 'Gruber Maschinenbau GmbH',
      customer_id: IDS.companies.gruberMaschinenbau,
      issue_date: daysAgo(45),
      due_date: daysAgo(15),
      total_net: 12500.0,
      total_gross: 14875.0,
      tax_rate: 19,
      currency: 'EUR',
      items: [
        { description: 'CRM Lizenz Enterprise (Jahresabo)', quantity: 1, unit_price: 9500.0, total: 9500.0 },
        { description: 'Onsite-Prozessanalyse (5 Tage)', quantity: 5, unit_price: 600.0, total: 3000.0 },
      ],
      notes: 'Zahlung erhalten am ' + daysAgo(18),
      created_at: daysAgo(45),
    },
    {
      id: IDS.invoices.inv002,
      number: 'RE-2026-002',
      invoice_number: 'RE-2026-002',
      status: 'paid' as const,
      customer: { id: IDS.companies.helvetiaSoftware, name: 'Helvetia Software AG' },
      customer_name: 'Helvetia Software AG',
      customer_id: IDS.companies.helvetiaSoftware,
      issue_date: daysAgo(40),
      due_date: daysAgo(10),
      total_net: 8400.0,
      total_gross: 9996.0,
      tax_rate: 19,
      currency: 'CHF',
      exchange_rate: '1.06',
      items: [
        { description: 'CRM Lizenz Professional (Jahresabo)', quantity: 1, unit_price: 5400.0, total: 5400.0 },
        { description: 'Schulungspaket (3 Tage)', quantity: 3, unit_price: 1000.0, total: 3000.0 },
      ],
      notes: '',
      created_at: daysAgo(40),
    },
    {
      id: IDS.invoices.inv003,
      number: 'RE-2026-003',
      invoice_number: 'RE-2026-003',
      status: 'paid' as const,
      customer: { id: IDS.companies.alpenLogistik, name: 'Alpen Logistik KG' },
      customer_name: 'Alpen Logistik KG',
      customer_id: IDS.companies.alpenLogistik,
      issue_date: daysAgo(35),
      due_date: daysAgo(5),
      total_net: 3200.0,
      total_gross: 3808.0,
      tax_rate: 19,
      currency: 'EUR',
      items: [
        { description: 'API-Integration Bestandssystem', quantity: 1, unit_price: 2400.0, total: 2400.0 },
        { description: 'Dokumentation & Uebergabe', quantity: 1, unit_price: 800.0, total: 800.0 },
      ],
      notes: '',
      created_at: daysAgo(35),
    },
    {
      id: IDS.invoices.inv004,
      number: 'RE-2026-004',
      invoice_number: 'RE-2026-004',
      status: 'paid' as const,
      customer: { id: IDS.companies.rheinConsulting, name: 'Rhein Consulting GmbH' },
      customer_name: 'Rhein Consulting GmbH',
      customer_id: IDS.companies.rheinConsulting,
      issue_date: daysAgo(30),
      due_date: daysAgo(0),
      total_net: 6800.0,
      total_gross: 8092.0,
      tax_rate: 19,
      currency: 'EUR',
      items: [
        { description: 'Consulting-Paket Digitalisierung', quantity: 1, unit_price: 4800.0, total: 4800.0 },
        { description: 'Hosting Setup (EU-Cloud)', quantity: 1, unit_price: 2000.0, total: 2000.0 },
      ],
      notes: '',
      created_at: daysAgo(30),
    },

    // --- 3 SENT ---
    {
      id: IDS.invoices.inv005,
      number: 'RE-2026-005',
      invoice_number: 'RE-2026-005',
      status: 'sent' as const,
      customer: { id: IDS.companies.bavariaElektro, name: 'Bavaria Elektro AG' },
      customer_name: 'Bavaria Elektro AG',
      customer_id: IDS.companies.bavariaElektro,
      issue_date: daysAgo(10),
      due_date: daysFromNow(20),
      total_net: 15000.0,
      total_gross: 17850.0,
      tax_rate: 19,
      currency: 'EUR',
      items: [
        { description: 'CRM Lizenz Enterprise (Jahresabo)', quantity: 1, unit_price: 9500.0, total: 9500.0 },
        { description: 'WASM-Plugin Entwicklung', quantity: 1, unit_price: 3500.0, total: 3500.0 },
        { description: 'Daten-Migration', quantity: 1, unit_price: 2000.0, total: 2000.0 },
      ],
      notes: 'Zahlungsziel 30 Tage netto',
      created_at: daysAgo(10),
    },
    {
      id: IDS.invoices.inv006,
      number: 'RE-2026-006',
      invoice_number: 'RE-2026-006',
      status: 'sent' as const,
      customer: { id: IDS.companies.zurichFintech, name: 'Zürich Fintech Solutions' },
      customer_name: 'Zürich Fintech Solutions',
      customer_id: IDS.companies.zurichFintech,
      issue_date: daysAgo(7),
      due_date: daysFromNow(23),
      total_net: 4200.0,
      total_gross: 4998.0,
      tax_rate: 19,
      currency: 'CHF',
      exchange_rate: '1.06',
      items: [
        { description: 'CRM Lizenz Professional (Jahresabo)', quantity: 1, unit_price: 4200.0, total: 4200.0 },
      ],
      notes: '',
      created_at: daysAgo(7),
    },
    {
      id: IDS.invoices.inv007,
      number: 'RE-2026-007',
      invoice_number: 'RE-2026-007',
      status: 'sent' as const,
      customer: { id: IDS.companies.wienerDesign, name: 'Wiener Design Studio' },
      customer_name: 'Wiener Design Studio',
      customer_id: IDS.companies.wienerDesign,
      issue_date: daysAgo(5),
      due_date: daysFromNow(25),
      total_net: 2800.0,
      total_gross: 3332.0,
      tax_rate: 19,
      currency: 'EUR',
      items: [
        { description: 'CRM Lizenz Starter (Jahresabo)', quantity: 1, unit_price: 1800.0, total: 1800.0 },
        { description: 'Einrichtung & Konfiguration', quantity: 1, unit_price: 1000.0, total: 1000.0 },
      ],
      notes: '',
      created_at: daysAgo(5),
    },

    // --- 2 OVERDUE ---
    {
      id: IDS.invoices.inv008,
      number: 'RE-2026-008',
      invoice_number: 'RE-2026-008',
      status: 'overdue' as const,
      customer: { id: IDS.companies.schwarzwaldHolz, name: 'Schwarzwald Holz GmbH' },
      customer_name: 'Schwarzwald Holz GmbH',
      customer_id: IDS.companies.schwarzwaldHolz,
      issue_date: daysAgo(60),
      due_date: daysAgo(30),
      total_net: 5600.0,
      total_gross: 6664.0,
      tax_rate: 19,
      currency: 'EUR',
      items: [
        { description: 'CRM Lizenz Professional (Jahresabo)', quantity: 1, unit_price: 5400.0, total: 5400.0 },
        { description: 'Zusatz-Benutzer (2x)', quantity: 2, unit_price: 100.0, total: 200.0 },
      ],
      notes: '1. Mahnung versendet',
      created_at: daysAgo(60),
    },
    {
      id: IDS.invoices.inv009,
      number: 'RE-2026-009',
      invoice_number: 'RE-2026-009',
      status: 'overdue' as const,
      customer: { id: IDS.companies.nordlichtMedia, name: 'Nordlicht Media GmbH' },
      customer_name: 'Nordlicht Media GmbH',
      customer_id: IDS.companies.nordlichtMedia,
      issue_date: daysAgo(50),
      due_date: daysAgo(20),
      total_net: 3300.0,
      total_gross: 3927.0,
      tax_rate: 19,
      currency: 'EUR',
      items: [
        { description: 'Schulungspaket (2 Tage)', quantity: 2, unit_price: 1000.0, total: 2000.0 },
        { description: 'Support-Vertrag (Quartal)', quantity: 1, unit_price: 1300.0, total: 1300.0 },
      ],
      notes: 'Telefonische Erinnerung am ' + daysAgo(5),
      created_at: daysAgo(50),
    },

    // --- 2 DRAFT ---
    {
      id: IDS.invoices.inv010,
      number: 'RE-2026-010',
      invoice_number: 'RE-2026-010',
      status: 'draft' as const,
      customer: { id: IDS.companies.donauPharma, name: 'Donau Pharma AG' },
      customer_name: 'Donau Pharma AG',
      customer_id: IDS.companies.donauPharma,
      issue_date: today(),
      due_date: daysFromNow(30),
      total_net: 22000.0,
      total_gross: 26180.0,
      tax_rate: 19,
      currency: 'EUR',
      items: [
        { description: 'CRM Lizenz Enterprise (Jahresabo)', quantity: 1, unit_price: 9500.0, total: 9500.0 },
        { description: 'DSGVO-Compliance Paket', quantity: 1, unit_price: 5500.0, total: 5500.0 },
        { description: 'Onsite-Prozessanalyse (5 Tage)', quantity: 5, unit_price: 600.0, total: 3000.0 },
        { description: 'Daten-Migration (komplex)', quantity: 1, unit_price: 4000.0, total: 4000.0 },
      ],
      notes: 'Entwurf — noch nicht freigegeben',
      created_at: today(),
    },
    {
      id: 'inv-011',
      number: 'RE-2026-011',
      invoice_number: 'RE-2026-011',
      status: 'draft' as const,
      customer: { id: IDS.companies.bernSolar, name: 'Bern Solar GmbH' },
      customer_name: 'Bern Solar GmbH',
      customer_id: IDS.companies.bernSolar,
      issue_date: today(),
      due_date: daysFromNow(30),
      total_net: 7600.0,
      total_gross: 9044.0,
      tax_rate: 19,
      currency: 'EUR',
      items: [
        { description: 'CRM Lizenz Professional (Jahresabo)', quantity: 1, unit_price: 5400.0, total: 5400.0 },
        { description: 'Bexio-Integration', quantity: 1, unit_price: 2200.0, total: 2200.0 },
      ],
      notes: '',
      created_at: today(),
    },

    // --- 1 CANCELLED ---
    {
      id: 'inv-012',
      number: 'RE-2026-012',
      invoice_number: 'RE-2026-012',
      status: 'cancelled' as const,
      customer: { id: IDS.companies.hanseatischIT, name: 'Hanseatisch IT Services' },
      customer_name: 'Hanseatisch IT Services',
      customer_id: IDS.companies.hanseatischIT,
      issue_date: daysAgo(25),
      due_date: daysFromNow(5),
      total_net: 4500.0,
      total_gross: 5355.0,
      tax_rate: 19,
      currency: 'EUR',
      items: [
        { description: 'Support-Vertrag Premium (Jahresabo)', quantity: 1, unit_price: 4500.0, total: 4500.0 },
      ],
      notes: 'Storniert — Kunde hat Projekt verschoben',
      created_at: daysAgo(25),
    },

    // --- Recurring-generated (link back to rec-00x profiles) ---
    ...recurringGeneratedInvoices,
  ],
  total: 12 + recurringGeneratedInvoices.length,
  page: 1,
  per_page: 50,
}

// ---------------------------------------------------------------------------
// Quotes (6)
// ---------------------------------------------------------------------------

export const mockQuotes = {
  quotes: [
    {
      id: 'qt-001',
      number: 'AN-2026-001',
      quote_number: 'AN-2026-001',
      status: 'accepted' as const,
      customer: { id: IDS.companies.gruberMaschinenbau, name: 'Gruber Maschinenbau GmbH' },
      customer_name: 'Gruber Maschinenbau GmbH',
      customer_id: IDS.companies.gruberMaschinenbau,
      valid_until: daysAgo(10),
      total_net: 12500.0,
      total_gross: 14875.0,
      items: [
        { description: 'CRM Lizenz Enterprise (Jahresabo)', quantity: 1, unit_price: 9500.0, total: 9500.0 },
        { description: 'Onsite-Prozessanalyse (5 Tage)', quantity: 5, unit_price: 600.0, total: 3000.0 },
      ],
      created_at: daysAgo(60),
    },
    {
      id: 'qt-002',
      number: 'AN-2026-002',
      quote_number: 'AN-2026-002',
      status: 'sent' as const,
      customer: { id: IDS.companies.donauPharma, name: 'Donau Pharma AG' },
      customer_name: 'Donau Pharma AG',
      customer_id: IDS.companies.donauPharma,
      valid_until: daysFromNow(14),
      total_net: 22000.0,
      total_gross: 26180.0,
      items: [
        { description: 'CRM Lizenz Enterprise (Jahresabo)', quantity: 1, unit_price: 9500.0, total: 9500.0 },
        { description: 'DSGVO-Compliance Paket', quantity: 1, unit_price: 5500.0, total: 5500.0 },
        { description: 'Onsite-Prozessanalyse (5 Tage)', quantity: 5, unit_price: 600.0, total: 3000.0 },
        { description: 'Daten-Migration (komplex)', quantity: 1, unit_price: 4000.0, total: 4000.0 },
      ],
      created_at: daysAgo(7),
    },
    {
      id: 'qt-003',
      number: 'AN-2026-003',
      quote_number: 'AN-2026-003',
      status: 'draft' as const,
      customer: { id: IDS.companies.bernSolar, name: 'Bern Solar GmbH' },
      customer_name: 'Bern Solar GmbH',
      customer_id: IDS.companies.bernSolar,
      valid_until: daysFromNow(30),
      currency: 'CHF',
      exchange_rate: '1.06',
      total_net: 9800.0,
      total_gross: 11662.0,
      items: [
        { description: 'CRM Lizenz Professional (Jahresabo)', quantity: 1, unit_price: 5400.0, total: 5400.0 },
        { description: 'Bexio-Integration', quantity: 1, unit_price: 2200.0, total: 2200.0 },
        { description: 'Schulungspaket (2 Tage)', quantity: 2, unit_price: 1100.0, total: 2200.0 },
      ],
      created_at: daysAgo(2),
    },
    {
      id: 'qt-004',
      number: 'AN-2026-004',
      quote_number: 'AN-2026-004',
      status: 'rejected' as const,
      customer: { id: IDS.companies.hanseatischIT, name: 'Hanseatisch IT Services' },
      customer_name: 'Hanseatisch IT Services',
      customer_id: IDS.companies.hanseatischIT,
      valid_until: daysAgo(5),
      total_net: 18500.0,
      total_gross: 22015.0,
      items: [
        { description: 'CRM Lizenz Enterprise (Jahresabo)', quantity: 1, unit_price: 9500.0, total: 9500.0 },
        { description: 'Self-Hosted Setup', quantity: 1, unit_price: 5000.0, total: 5000.0 },
        { description: 'Daten-Migration', quantity: 1, unit_price: 4000.0, total: 4000.0 },
      ],
      created_at: daysAgo(30),
    },
    {
      id: 'qt-005',
      number: 'AN-2026-005',
      quote_number: 'AN-2026-005',
      status: 'expired' as const,
      customer: { id: IDS.companies.nordlichtMedia, name: 'Nordlicht Media GmbH' },
      customer_name: 'Nordlicht Media GmbH',
      customer_id: IDS.companies.nordlichtMedia,
      valid_until: daysAgo(15),
      total_net: 6200.0,
      total_gross: 7378.0,
      items: [
        { description: 'CRM Lizenz Professional (Jahresabo)', quantity: 1, unit_price: 5400.0, total: 5400.0 },
        { description: 'Einrichtung & Konfiguration', quantity: 1, unit_price: 800.0, total: 800.0 },
      ],
      created_at: daysAgo(50),
    },
    {
      id: 'qt-006',
      number: 'AN-2026-006',
      quote_number: 'AN-2026-006',
      status: 'sent' as const,
      customer: { id: IDS.companies.bavariaElektro, name: 'Bavaria Elektro AG' },
      customer_name: 'Bavaria Elektro AG',
      customer_id: IDS.companies.bavariaElektro,
      valid_until: daysFromNow(21),
      total_net: 8500.0,
      total_gross: 10115.0,
      items: [
        { description: 'Support-Vertrag Premium (Jahresabo)', quantity: 1, unit_price: 4500.0, total: 4500.0 },
        { description: 'WASM-Plugin Wartung', quantity: 1, unit_price: 4000.0, total: 4000.0 },
      ],
      created_at: daysAgo(3),
    },
  ],
  total: 6,
  page: 1,
  per_page: 50,
}

// ---------------------------------------------------------------------------
// Credit Notes (3)
// ---------------------------------------------------------------------------

export const mockCreditNotes = {
  credit_notes: [
    {
      id: 'cn-001',
      number: 'GS-2026-001',
      credit_note_number: 'GS-2026-001',
      status: 'issued' as const,
      invoice_number: 'RE-2026-012',
      customer: { id: IDS.companies.hanseatischIT, name: 'Hanseatisch IT Services' },
      customer_name: 'Hanseatisch IT Services',
      customer_id: IDS.companies.hanseatischIT,
      issue_date: daysAgo(20),
      total_net: 4500.0,
      total_gross: 5355.0,
      reason: 'Stornierung — Projekt verschoben',
      created_at: daysAgo(20),
    },
    {
      id: 'cn-002',
      number: 'GS-2026-002',
      credit_note_number: 'GS-2026-002',
      status: 'issued' as const,
      invoice_number: 'RE-2026-003',
      customer: { id: IDS.companies.alpenLogistik, name: 'Alpen Logistik KG' },
      customer_name: 'Alpen Logistik KG',
      customer_id: IDS.companies.alpenLogistik,
      issue_date: daysAgo(28),
      total_net: 800.0,
      total_gross: 952.0,
      reason: 'Teilgutschrift — Dokumentation entfaellt',
      created_at: daysAgo(28),
    },
    {
      id: 'cn-003',
      number: 'GS-2026-003',
      credit_note_number: 'GS-2026-003',
      status: 'draft' as const,
      invoice_number: 'RE-2026-008',
      customer: { id: IDS.companies.schwarzwaldHolz, name: 'Schwarzwald Holz GmbH' },
      customer_name: 'Schwarzwald Holz GmbH',
      customer_id: IDS.companies.schwarzwaldHolz,
      issue_date: today(),
      total_net: 200.0,
      total_gross: 238.0,
      reason: 'Rabatt — Zahlungsverzug-Kulanz',
      created_at: today(),
    },
  ],
  total: 3,
}

// ---------------------------------------------------------------------------
// Finance Dashboard
// ---------------------------------------------------------------------------

export const mockFinanceDashboard = {
  total_invoiced: '189600',
  total_paid: '165800',
  total_outstanding: '23800',
  overdue_amount: '8900',
  quotes_pending: 3,
  conversion_rate: 68,
  average_deal_size: '42500',
  revenue_forecast: '52000',
  revenue_this_month: 47500,
  revenue_last_month: 42300,
  status_breakdown: { draft: 2, sent: 3, paid: 4, overdue: 2 } as Record<string, number>,
  recent_invoices: [] as unknown[],
  expiring_quotes: [] as unknown[],
  pending_dunnings: [] as unknown[],
  monthly_revenue: [
    { month: monthsAgo(11), revenue: 28400 },
    { month: monthsAgo(10), revenue: 31200 },
    { month: monthsAgo(9), revenue: 29800 },
    { month: monthsAgo(8), revenue: 35600 },
    { month: monthsAgo(7), revenue: 33100 },
    { month: monthsAgo(6), revenue: 38900 },
    { month: monthsAgo(5), revenue: 36400 },
    { month: monthsAgo(4), revenue: 41200 },
    { month: monthsAgo(3), revenue: 39800 },
    { month: monthsAgo(2), revenue: 44100 },
    { month: monthsAgo(1), revenue: 42300 },
    { month: monthsAgo(0), revenue: 47500 },
  ],
}

// ---------------------------------------------------------------------------
// Finance Settings (company)
// ---------------------------------------------------------------------------

export const mockFinanceSettings = {
  settings: {
    company_name: 'TechVision GmbH',
    tax_id: 'DE314256789',
    vat_id: 'DE314256789',
    address: {
      street: 'Innovationsweg 12',
      zip: '80331',
      city: 'München',
      country: 'DE',
    },
    bank_account: {
      bank_name: 'Commerzbank AG',
      iban: 'DE89 3704 0044 0532 0130 00',
      bic: 'COBADEFFXXX',
    },
    default_payment_terms: 30,
    default_tax_rate: 19,
    invoice_prefix: 'RE',
    quote_prefix: 'AN',
    credit_note_prefix: 'GS',
    next_invoice_number: 13,
    next_quote_number: 7,
    footer_text: 'Vielen Dank für Ihr Vertrauen. Zahlbar innerhalb von 30 Tagen netto.',
  },
}
