/**
 * Mock seed data for unbilled time entries (Stunden → Rechnung) — finanzen P2.5e.
 *
 * Stands in for the Zeiterfassung service: GET /time-entries?billed=false.
 * Used by HoursToInvoiceDialog to build invoice line items from tracked hours.
 */
import { daysAgo } from './date-helpers'
import type { FinanceTimeEntry } from '@/types/finance-types'

const entries: FinanceTimeEntry[] = [
  { id: 'te1', date: daysAgo(2), project: 'Website Redesign', task: 'Frontend-Entwicklung', employee: 'Anna Müller', hours: 4.5, description: 'React-Komponenten implementiert', billed: false },
  { id: 'te2', date: daysAgo(2), project: 'Website Redesign', task: 'Design-Review', employee: 'Anna Müller', hours: 1.5, description: 'Figma-Prototyp reviewt', billed: false },
  { id: 'te3', date: daysAgo(3), project: 'CRM Integration', task: 'API-Entwicklung', employee: 'Thomas Fischer', hours: 6.0, description: 'REST-Endpoints implementiert', billed: false },
  { id: 'te4', date: daysAgo(3), project: 'CRM Integration', task: 'Testing', employee: 'Thomas Fischer', hours: 2.0, description: 'Integration Tests geschrieben', billed: false },
  { id: 'te5', date: daysAgo(4), project: 'Website Redesign', task: 'Deployment', employee: 'Max Schmidt', hours: 3.0, description: 'Staging-Deployment + DNS', billed: false },
  { id: 'te6', date: daysAgo(5), project: 'Mobile App', task: 'UI-Development', employee: 'Sara Weber', hours: 5.5, description: 'React Native Screens', billed: false },
  { id: 'te7', date: daysAgo(5), project: 'CRM Integration', task: 'Datenbank-Migration', employee: 'Thomas Fischer', hours: 3.5, description: 'PostgreSQL Migrations', billed: false },
  { id: 'te8', date: daysAgo(6), project: 'Website Redesign', task: 'Performance', employee: 'Anna Müller', hours: 2.0, description: 'Lighthouse-Optimierung', billed: false },
  { id: 'te9', date: daysAgo(7), project: 'Beratung', task: 'Workshop', employee: 'Max Schmidt', hours: 4.0, description: 'Onboarding-Workshop beim Kunden', billed: true },
]

export const mockFinanceTimeEntries: { entries: FinanceTimeEntry[] } = { entries }
