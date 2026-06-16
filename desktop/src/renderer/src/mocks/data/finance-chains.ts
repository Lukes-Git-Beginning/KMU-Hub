/**
 * Mock seed data for document chains (Belegkette) — finanzen P2.5e.
 *
 * A chain links the lifecycle of a financial document: Angebot → Rechnung →
 * Zahlung → Gutschrift/Mahnung. Read-only aggregation for the demo.
 */
import type { DocumentChain } from '@/types/finance-types'

const chains: DocumentChain[] = [
  {
    id: 'chain-1',
    customer: 'Muster AG',
    totalValue: 'CHF 12.450,00',
    isComplete: true,
    nodes: [
      { type: 'quote', number: 'AN-2026-001', date: '2026-01-10', amount: 'CHF 12.450,00', status: 'completed' },
      { type: 'invoice', number: 'RE-2026-003', date: '2026-01-25', amount: 'CHF 12.450,00', status: 'completed' },
      { type: 'payment', number: 'ZA-2026-007', date: '2026-02-10', amount: 'CHF 12.450,00', status: 'completed' },
    ],
  },
  {
    id: 'chain-2',
    customer: 'Digital Solutions GmbH',
    totalValue: 'EUR 8.900,00',
    isComplete: false,
    nodes: [
      { type: 'quote', number: 'AN-2026-004', date: '2026-01-20', amount: 'EUR 8.900,00', status: 'completed' },
      { type: 'invoice', number: 'RE-2026-008', date: '2026-02-01', amount: 'EUR 8.900,00', status: 'active' },
      { type: 'dunning', number: 'MA-2026-001', date: '2026-02-18', amount: 'EUR 8.900,00', status: 'active' },
      { type: 'payment', number: '—', date: '—', amount: 'EUR 8.900,00', status: 'pending' },
    ],
  },
  {
    id: 'chain-3',
    customer: 'Weber & Partner',
    totalValue: 'CHF 5.200,00',
    isComplete: false,
    nodes: [
      { type: 'invoice', number: 'RE-2026-012', date: '2026-02-05', amount: 'CHF 5.200,00', status: 'overdue' },
      { type: 'payment', number: '—', date: '—', amount: 'CHF 5.200,00', status: 'pending' },
    ],
  },
  {
    id: 'chain-4',
    customer: 'TechStart Zürich',
    totalValue: 'CHF 3.750,00',
    isComplete: true,
    nodes: [
      { type: 'invoice', number: 'RE-2026-006', date: '2026-01-15', amount: 'CHF 3.750,00', status: 'completed' },
      { type: 'payment', number: 'ZA-2026-012', date: '2026-01-28', amount: 'CHF 2.750,00', status: 'completed' },
      { type: 'credit-note', number: 'GU-2026-001', date: '2026-02-01', amount: 'CHF 1.000,00', status: 'completed' },
    ],
  },
  {
    id: 'chain-5',
    customer: 'Alpen Logistik AG',
    totalValue: 'EUR 24.800,00',
    isComplete: false,
    nodes: [
      { type: 'quote', number: 'AN-2026-009', date: '2026-02-10', amount: 'EUR 24.800,00', status: 'completed' },
      { type: 'invoice', number: 'RE-2026-015', date: '2026-02-14', amount: 'EUR 24.800,00', status: 'active' },
      { type: 'payment', number: '—', date: '—', amount: 'EUR 24.800,00', status: 'pending' },
    ],
  },
  {
    id: 'chain-6',
    customer: 'Innovate Labs',
    totalValue: 'CHF 1.950,00',
    isComplete: true,
    nodes: [
      { type: 'quote', number: 'AN-2026-002', date: '2026-01-05', amount: 'CHF 1.950,00', status: 'cancelled' },
    ],
  },
]

export const mockDocumentChains: { chains: DocumentChain[] } = { chains }
