/**
 * Mock seed data for banking (FinAPI placeholder) — finanzen P2.5e.
 *
 * Bank accounts + recent transactions with invoice-to-payment match status.
 * Handlers mutate `mockBanking.transactions` in place so accept/reject a
 * suggested match feels real in the demo.
 */
import { daysAgo } from './date-helpers'
import type { BankAccount, BankTransaction } from '@/types/finance-types'

const accounts: BankAccount[] = [
  {
    id: 'ba1',
    bankName: 'Commerzbank',
    iban: 'DE89 3704 0044 0532 0130 00',
    bic: 'COBADEFFXXX',
    balance: 142850.75,
    currency: 'EUR',
    connected: true,
    lastSync: daysAgo(0),
  },
  {
    id: 'ba2',
    bankName: 'Sparkasse München',
    iban: 'DE72 7015 0000 0012 3456 78',
    bic: 'SSKMDEMMXXX',
    balance: 85000.0,
    currency: 'EUR',
    connected: false,
    lastSync: null,
  },
]

const transactions: BankTransaction[] = [
  { id: 'bt1', date: daysAgo(1), description: 'Eingang Gruber Maschinenbau — Abschlag 3', amount: 16000.0, type: 'credit', counterpart: 'Gruber Maschinenbau GmbH', matchStatus: 'matched', matchedInvoice: 'RE-2026-003' },
  { id: 'bt2', date: daysAgo(3), description: 'Eingang DataFlow — Analytics Dashboard', amount: 14000.0, type: 'credit', counterpart: 'DataFlow GmbH', matchStatus: 'matched', matchedInvoice: 'RE-2026-011' },
  { id: 'bt3', date: daysAgo(6), description: 'Eingang Stadler Bau — Intranet Portal', amount: 21000.0, type: 'credit', counterpart: 'Stadler Bauunternehmen GmbH', matchStatus: 'suggested', matchedInvoice: 'RE-2026-008' },
  { id: 'bt4', date: daysAgo(2), description: 'CloudFirst Hosting — Monatsrechnung Feb', amount: -1890.0, type: 'debit', counterpart: 'CloudFirst Hosting GmbH', matchStatus: 'unmatched' },
  { id: 'bt5', date: daysAgo(4), description: 'Gehälter Februar 2026', amount: -78500.0, type: 'debit', counterpart: 'Sammelüberweisung', matchStatus: 'unmatched' },
  { id: 'bt6', date: daysAgo(0), description: 'Eingang Berger — Mobile App Anzahlung', amount: 20000.0, type: 'credit', counterpart: 'Berger & Soehne', matchStatus: 'suggested', matchedInvoice: 'RE-2026-015' },
  { id: 'bt7', date: daysAgo(7), description: 'Adobe Creative Cloud — Jahresrechnung', amount: -4188.0, type: 'debit', counterpart: 'Adobe Inc.', matchStatus: 'unmatched' },
  // Offener Eingang ohne automatische Zuordnung — für manuelle Zuordnung zur
  // überfälligen Rechnung RE-2026-008 (Schwarzwald Holz).
  { id: 'bt8', date: daysAgo(2), description: 'Eingang Schwarzwald Holz — Überweisung', amount: 6664.0, type: 'credit', counterpart: 'Schwarzwald Holz GmbH', matchStatus: 'unmatched' },
]

export const mockBanking: { accounts: BankAccount[]; transactions: BankTransaction[] } = {
  accounts,
  transactions,
}
