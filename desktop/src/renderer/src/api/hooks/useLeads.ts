/**
 * Lead inbox hooks (frontend mock-first).
 *
 * Leads are the raw, unqualified pre-stage before a real contact. Architecture
 * intent for the backend (Luke): model a lead as a contact lifecycle-status
 * (`lead`) rather than a separate table — see .planning/backend-gaps.md. Until
 * the backend exists, this hook operates on an in-memory array so the inbox is
 * fully interactive in demo mode. Swap the queryFn/mutationFn bodies for
 * apiClient calls once `/api/v1/leads` lands.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

export type LeadSource = 'manual' | 'csv' | 'dialer'
export type LeadStatus = 'new' | 'contacted' | 'qualified' | 'disqualified'
export type LeadTemperature = 'hot' | 'warm' | 'cold'

export interface Lead {
  id: string
  firstName: string
  lastName: string
  company: string
  email: string
  phone: string
  source: LeadSource
  /** Auto-computed score 0–100. */
  score: number
  /** Effective temperature — manual override wins over the score-derived one. */
  temperature: LeadTemperature
  /** Set when the user manually overrides the auto temperature. */
  temperatureOverride?: LeadTemperature
  status: LeadStatus
  notes: string
  createdAt: string
}

// ---------------------------------------------------------------------------
// Scoring — auto-suggestion from source quality + data completeness.
// ---------------------------------------------------------------------------

const SOURCE_BASE: Record<LeadSource, number> = { dialer: 35, manual: 25, csv: 10 }

/** Compute an auto score 0–100 from source and how complete the lead data is. */
export function computeLeadScore(input: {
  source: LeadSource
  email?: string
  phone?: string
  company?: string
  notes?: string
}): number {
  let score = SOURCE_BASE[input.source] ?? 15
  if (input.email?.trim()) score += 20
  if (input.phone?.trim()) score += 15
  if (input.company?.trim()) score += 20
  if (input.notes?.trim()) score += 10
  return Math.min(100, score)
}

export function scoreToTemperature(score: number): LeadTemperature {
  if (score >= 66) return 'hot'
  if (score >= 33) return 'warm'
  return 'cold'
}

/** Effective temperature: manual override if set, else derived from score. */
export function effectiveTemperature(lead: Lead): LeadTemperature {
  return lead.temperatureOverride ?? scoreToTemperature(lead.score)
}

// ---------------------------------------------------------------------------
// In-memory mock store
// ---------------------------------------------------------------------------

function daysAgo(n: number): string {
  return new Date(Date.now() - n * 86_400_000).toISOString()
}

function seed(partial: Omit<Lead, 'score' | 'temperature'>): Lead {
  const score = computeLeadScore(partial)
  return { ...partial, score, temperature: scoreToTemperature(score) }
}

let leadsStore: Lead[] = [
  seed({ id: 'ld-1', firstName: 'Sabine', lastName: 'Vogel', company: 'Vogel Dachtechnik GmbH', email: 'sabine.vogel@vogel-dach.de', phone: '+49 711 4455660', source: 'dialer', status: 'new', notes: 'Rückruf gewünscht — Interesse an CRM für 12 Mitarbeiter.', createdAt: daysAgo(1) }),
  seed({ id: 'ld-2', firstName: 'Markus', lastName: 'Brandt', company: 'Brandt Logistik', email: '', phone: '+49 40 99887766', source: 'dialer', status: 'new', notes: 'Auf der Messe angesprochen.', createdAt: daysAgo(2) }),
  seed({ id: 'ld-3', firstName: 'Lena', lastName: 'Fischer', company: 'Fischer & Partner Steuerberatung', email: 'l.fischer@fischer-partner.de', phone: '+49 89 1234560', source: 'manual', status: 'contacted', notes: 'Erstgespräch geführt, will Angebot.', createdAt: daysAgo(4) }),
  seed({ id: 'ld-4', firstName: 'Tobias', lastName: 'Krüger', company: '', email: 'tobias.krueger@gmail.com', phone: '', source: 'csv', status: 'new', notes: '', createdAt: daysAgo(6) }),
  seed({ id: 'ld-5', firstName: 'Andrea', lastName: 'Schuster', company: 'Schuster Bau AG', email: 'a.schuster@schuster-bau.ch', phone: '+41 44 5566778', source: 'manual', status: 'contacted', notes: 'Sucht Lösung für Baustellen-Doku.', createdAt: daysAgo(8) }),
  seed({ id: 'ld-6', firstName: 'Daniel', lastName: 'Berger', company: 'Berger Elektro', email: '', phone: '', source: 'csv', status: 'new', notes: '', createdAt: daysAgo(10) }),
  seed({ id: 'ld-7', firstName: 'Petra', lastName: 'Lindner', company: 'Lindner Consulting', email: 'petra@lindner-consulting.de', phone: '+49 30 7788990', source: 'dialer', status: 'qualified', notes: 'Qualifiziert — wird zu Kontakt.', createdAt: daysAgo(12) }),
  seed({ id: 'ld-8', firstName: 'Jochen', lastName: 'Maier', company: 'Maier GmbH', email: 'info@maier-gmbh.de', phone: '+49 721 4433221', source: 'manual', status: 'disqualified', notes: 'Kein Budget, nicht weiterverfolgen.', createdAt: daysAgo(15) }),
]

let nextId = leadsStore.length + 1
function delay<T>(value: T): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), 120))
}

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------

export function useLeads() {
  return useQuery({
    queryKey: ['leads'],
    queryFn: () => delay([...leadsStore]),
  })
}

export interface NewLeadInput {
  firstName: string
  lastName: string
  company?: string
  email?: string
  phone?: string
  source: LeadSource
  notes?: string
}

export function useCreateLead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: NewLeadInput) => {
      const score = computeLeadScore(input)
      const lead: Lead = {
        id: `ld-${nextId++}`,
        firstName: input.firstName,
        lastName: input.lastName,
        company: input.company ?? '',
        email: input.email ?? '',
        phone: input.phone ?? '',
        source: input.source,
        score,
        temperature: scoreToTemperature(score),
        status: 'new',
        notes: input.notes ?? '',
        createdAt: new Date().toISOString(),
      }
      leadsStore = [lead, ...leadsStore]
      return delay(lead)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['leads'] }),
  })
}

export function useCreateLeadsBatch() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (rows: NewLeadInput[]) => {
      const created = rows.map((input) => {
        const score = computeLeadScore(input)
        const lead: Lead = {
          id: `ld-${nextId++}`,
          firstName: input.firstName,
          lastName: input.lastName,
          company: input.company ?? '',
          email: input.email ?? '',
          phone: input.phone ?? '',
          source: input.source,
          score,
          temperature: scoreToTemperature(score),
          status: 'new',
          notes: input.notes ?? '',
          createdAt: new Date().toISOString(),
        }
        return lead
      })
      leadsStore = [...created, ...leadsStore]
      return delay(created.length)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['leads'] }),
  })
}

export function useUpdateLeadStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: LeadStatus }) => {
      leadsStore = leadsStore.map((l) => (l.id === id ? { ...l, status } : l))
      return delay(true)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['leads'] }),
  })
}

/** Manually override the lead temperature (sticky over the auto score). */
export function useSetLeadTemperature() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, temperature }: { id: string; temperature: LeadTemperature }) => {
      leadsStore = leadsStore.map((l) =>
        l.id === id ? { ...l, temperatureOverride: temperature, temperature } : l
      )
      return delay(true)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['leads'] }),
  })
}

/** Mark a lead qualified (it leaves the open inbox). Contact/company/deal
 *  creation is handled by the caller via the existing CRM create mocks. */
export function useConvertLead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id }: { id: string; createCompany?: boolean; createDeal?: boolean }) => {
      leadsStore = leadsStore.map((l) => (l.id === id ? { ...l, status: 'qualified' } : l))
      return delay(true)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['leads'] }),
  })
}
