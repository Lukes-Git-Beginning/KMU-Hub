/**
 * Mock seed data for the Ausgaben/Transaktionen ledger (finanzen P2).
 *
 * Swap-ready: there is no `/api/v1/finance/expenses` backend yet (siehe
 * backend-gaps.md §finanzen). The MSW handler mutates `mockExpenses` /
 * `mockTransactions` in place so approve/reject/create/Kontierung/Beleg-Upload
 * feel real in the demo. Replace the handler bodies with apiClient calls once
 * the endpoints land — the shapes already match the `Expense`/`Transaction`
 * types.
 *
 * Expenses carry an SKR03-Sachkonto (`account`) and an optional receipt name so
 * the Kontierung + Beleg-Indikator have data on first load.
 */
import { daysAgo } from './date-helpers'
import type { Expense, Transaction } from '@/stores/finance'

const EXPENSE_SEED: Expense[] = [
  { id: 'exp-1', description: 'Bürobedarf Q2', amount: 234.5, date: daysAgo(3), category: 'Büromaterial', supplier: 'Staples', account: '4930', receipt: true, receiptName: 'staples-rechnung-q2.pdf', status: 'pending' },
  { id: 'exp-2', description: 'Server-Hosting Hetzner', amount: 89.0, date: daysAgo(7), category: 'IT-Infrastruktur', supplier: 'Hetzner Online GmbH', account: '4900', receipt: true, receiptName: 'hetzner-2026-05.pdf', status: 'approved' },
  // Bewusst unkontiert + ohne Beleg → zeigt den "Kontieren"-Bedarf in der Liste.
  { id: 'exp-3', description: 'Geschäftsessen Kunde Müller', amount: 142.8, date: daysAgo(12), category: 'Bewirtung', supplier: 'Restaurant Adler', project: 'Projekt Müller-CRM', status: 'pending' },
  { id: 'exp-4', description: 'Software-Lizenz JetBrains', amount: 199.0, date: daysAgo(18), category: 'Software', supplier: 'JetBrains s.r.o.', account: '4806', receipt: true, receiptName: 'jetbrains-invoice.pdf', status: 'approved' },
  { id: 'exp-5', description: 'Werbeanzeigen Q2', amount: 850.0, date: daysAgo(25), category: 'Marketing', supplier: 'Google Ireland Ltd.', account: '4600', receipt: true, receiptName: 'google-ads-mai.pdf', status: 'rejected' },
  { id: 'exp-6', description: 'Bahnfahrt Berlin–Hamburg', amount: 78.4, date: daysAgo(34), category: 'Reisekosten', supplier: 'Deutsche Bahn AG', project: 'Kundentermin DB', account: '4660', status: 'approved' },
  { id: 'exp-7', description: 'Fortbildung Excel-Schulung', amount: 320.0, date: daysAgo(48), category: 'Weiterbildung', supplier: 'Akademie Online GmbH', account: '4945', receipt: true, receiptName: 'akademie-teilnahme.pdf', status: 'approved' },
  { id: 'exp-8', description: 'Steuerberatung Jahresabschluss', amount: 1450.0, date: daysAgo(52), category: 'Beratung', supplier: 'Kanzlei Bergmann', account: '4950', status: 'approved' },
]

const TRANSACTION_SEED: Transaction[] = [
  // ~5 Monate Verlauf für den Einnahmen-/Ausgaben-Chart (Berichte-Tab).
  { id: 'tx-1', type: 'income', description: 'Rechnung 2026-042 Müller GmbH', amount: 4280.0, date: daysAgo(2), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-042' },
  { id: 'tx-2', type: 'expense', description: 'Bürobedarf Staples', amount: 234.5, date: daysAgo(3), category: 'Büromaterial', status: 'completed' },
  { id: 'tx-3', type: 'expense', description: 'Server-Hosting Hetzner', amount: 89.0, date: daysAgo(7), category: 'IT-Infrastruktur', status: 'completed' },
  { id: 'tx-4', type: 'income', description: 'Rechnung 2026-039 Bavaria Elektro', amount: 2750.0, date: daysAgo(15), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-039' },
  { id: 'tx-5', type: 'expense', description: 'Werbeanzeigen Google', amount: 850.0, date: daysAgo(25), category: 'Marketing', status: 'completed' },
  { id: 'tx-6', type: 'income', description: 'Rechnung 2026-031 Gruber GmbH', amount: 3940.0, date: daysAgo(40), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-031' },
  { id: 'tx-7', type: 'expense', description: 'Bahnfahrt Berlin–Hamburg', amount: 78.4, date: daysAgo(34), category: 'Reisekosten', status: 'completed' },
  { id: 'tx-8', type: 'expense', description: 'Software-Lizenz JetBrains', amount: 199.0, date: daysAgo(48), category: 'Software', status: 'completed' },
  { id: 'tx-9', type: 'income', description: 'Rechnung 2026-024 Rhein Consulting', amount: 5100.0, date: daysAgo(70), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-024' },
  { id: 'tx-10', type: 'expense', description: 'Steuerberatung Q1', amount: 680.0, date: daysAgo(72), category: 'Beratung', status: 'completed' },
  { id: 'tx-11', type: 'income', description: 'Rechnung 2026-018 Helvetia AG', amount: 3420.0, date: daysAgo(98), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-018' },
  { id: 'tx-12', type: 'expense', description: 'Fortbildung Excel', amount: 320.0, date: daysAgo(105), category: 'Weiterbildung', status: 'completed' },
  { id: 'tx-13', type: 'income', description: 'Rechnung 2026-009 Bavaria Elektro', amount: 2980.0, date: daysAgo(128), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-009' },
  { id: 'tx-14', type: 'expense', description: 'Bewirtung Kundentermin', amount: 142.8, date: daysAgo(132), category: 'Bewirtung', status: 'completed' },
]

/** Mutable in place — der MSW-Handler verändert diese Arrays direkt. */
export const mockExpenses: Expense[] = EXPENSE_SEED
export const mockTransactions: Transaction[] = TRANSACTION_SEED
