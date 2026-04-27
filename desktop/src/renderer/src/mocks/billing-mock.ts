/**
 * billing-mock.ts — Plausible Demo-Daten für Billing-Insights (Sprint 1).
 *
 * 14 deutsche Demo-User, 2 erwartete Recommendations:
 *   1. Buchhaltung (finance): 3 von 4 inaktiv → −18 €/M
 *   2. Dialer: 6 von 8 inaktiv → −30 €/M
 */
import type { UserModuleGrant, ModuleUsageStats, ModuleId } from '@/lib/pricing'

// ───────────────────────── Helpers ─────────────────────────

/** Gibt ISO-String von vor n Tagen zurück. */
function daysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString()
}

// ───────────────────────── Mock-User ─────────────────────────

export const MOCK_USERS = [
  { userId: 'u-01', userName: 'Anna Müller' },
  { userId: 'u-02', userName: 'Bernd Schmidt' },
  { userId: 'u-03', userName: 'Carla Hoffmann' },
  { userId: 'u-04', userName: 'Daniel Weber' },
  { userId: 'u-05', userName: 'Eva König' },
  { userId: 'u-06', userName: 'Felix Bauer' },
  { userId: 'u-07', userName: 'Greta Schulz' },
  { userId: 'u-08', userName: 'Hannah Becker' },
  { userId: 'u-09', userName: 'Ingo Vogel' },
  { userId: 'u-10', userName: 'Julia Lehmann' },
  { userId: 'u-11', userName: 'Klaus Werner' },
  { userId: 'u-12', userName: 'Lina Friedrich' },
  { userId: 'u-13', userName: 'Max Wolf' },
  { userId: 'u-14', userName: 'Nora Sommer' },
] as const

// ───────────────────────── Mock-Grants ─────────────────────────
// Achtung: lastActiveAt in daysAgo() — deterministische Zahlen,
// damit generateMockUsage() mit DEFAULT_INSIGHT_SETTINGS exakt 2 Recommendations liefert.

export const MOCK_GRANTS: UserModuleGrant[] = [
  // CRM — alle 14, alle aktiv (0–7 Tage)
  ...MOCK_USERS.map((u, i) => ({
    userId: u.userId,
    userName: u.userName,
    moduleId: 'crm' as ModuleId,
    grantedAt: daysAgo(180),
    lastActiveAt: daysAgo(i % 7),
  })),

  // Tasks — 11 User (nicht Greta, Klaus, Nora), alle aktiv
  ...MOCK_USERS.filter((u) => !['u-07', 'u-11', 'u-14'].includes(u.userId)).map((u, i) => ({
    userId: u.userId,
    userName: u.userName,
    moduleId: 'tasks' as ModuleId,
    grantedAt: daysAgo(180),
    lastActiveAt: daysAgo(i % 10),
  })),

  // Finance (Buchhaltung) — Anna, Bernd, Carla, Daniel
  // Daniel aktiv (5 Tage), Anna/Bernd/Carla inaktiv (95 Tage) → Recommendation!
  { userId: 'u-01', userName: 'Anna Müller', moduleId: 'finance', grantedAt: daysAgo(180), lastActiveAt: daysAgo(95) },
  { userId: 'u-02', userName: 'Bernd Schmidt', moduleId: 'finance', grantedAt: daysAgo(180), lastActiveAt: daysAgo(100) },
  { userId: 'u-03', userName: 'Carla Hoffmann', moduleId: 'finance', grantedAt: daysAgo(180), lastActiveAt: daysAgo(98) },
  { userId: 'u-04', userName: 'Daniel Weber', moduleId: 'finance', grantedAt: daysAgo(180), lastActiveAt: daysAgo(5) },

  // Calendar — alle 14, alle aktiv
  ...MOCK_USERS.map((u, i) => ({
    userId: u.userId,
    userName: u.userName,
    moduleId: 'calendar' as ModuleId,
    grantedAt: daysAgo(180),
    lastActiveAt: daysAgo(i % 5),
  })),

  // Documents — alle 14, alle aktiv
  ...MOCK_USERS.map((u, i) => ({
    userId: u.userId,
    userName: u.userName,
    moduleId: 'documents' as ModuleId,
    grantedAt: daysAgo(180),
    lastActiveAt: daysAgo(i % 6),
  })),

  // Chat — alle 14, alle aktiv
  ...MOCK_USERS.map((u, i) => ({
    userId: u.userId,
    userName: u.userName,
    moduleId: 'chat' as ModuleId,
    grantedAt: daysAgo(180),
    lastActiveAt: daysAgo(i % 3),
  })),

  // Meetings — 8 User, alle aktiv
  ...MOCK_USERS.slice(0, 8).map((u, i) => ({
    userId: u.userId,
    userName: u.userName,
    moduleId: 'meetings' as ModuleId,
    grantedAt: daysAgo(180),
    lastActiveAt: daysAgo(i % 14),
  })),

  // Mail — alle 14, alle aktiv
  ...MOCK_USERS.map((u, i) => ({
    userId: u.userId,
    userName: u.userName,
    moduleId: 'mail' as ModuleId,
    grantedAt: daysAgo(180),
    lastActiveAt: daysAgo(i % 4),
  })),

  // Dialer — Anna, Bernd, Eva, Felix, Greta, Hannah, Ingo, Klaus (8)
  // Anna + Bernd aktiv (2 und 4 Tage), 6 andere inaktiv (65–110 Tage) → Recommendation!
  { userId: 'u-01', userName: 'Anna Müller', moduleId: 'dialer', grantedAt: daysAgo(180), lastActiveAt: daysAgo(2) },
  { userId: 'u-02', userName: 'Bernd Schmidt', moduleId: 'dialer', grantedAt: daysAgo(180), lastActiveAt: daysAgo(4) },
  { userId: 'u-05', userName: 'Eva König', moduleId: 'dialer', grantedAt: daysAgo(180), lastActiveAt: daysAgo(72) },
  { userId: 'u-06', userName: 'Felix Bauer', moduleId: 'dialer', grantedAt: daysAgo(180), lastActiveAt: daysAgo(85) },
  { userId: 'u-07', userName: 'Greta Schulz', moduleId: 'dialer', grantedAt: daysAgo(180), lastActiveAt: daysAgo(65) },
  { userId: 'u-08', userName: 'Hannah Becker', moduleId: 'dialer', grantedAt: daysAgo(180), lastActiveAt: daysAgo(91) },
  { userId: 'u-09', userName: 'Ingo Vogel', moduleId: 'dialer', grantedAt: daysAgo(180), lastActiveAt: daysAgo(110) },
  { userId: 'u-11', userName: 'Klaus Werner', moduleId: 'dialer', grantedAt: daysAgo(180), lastActiveAt: daysAgo(78) },

  // Team — alle 14
  ...MOCK_USERS.map((u, i) => ({
    userId: u.userId,
    userName: u.userName,
    moduleId: 'team' as ModuleId,
    grantedAt: daysAgo(180),
    lastActiveAt: daysAgo(i % 8),
  })),

  // Zeiterfassung — alle 14
  ...MOCK_USERS.map((u, i) => ({
    userId: u.userId,
    userName: u.userName,
    moduleId: 'zeiterfassung' as ModuleId,
    grantedAt: daysAgo(180),
    lastActiveAt: daysAgo(i % 5),
  })),

  // Projects — 12 User (nicht Greta, Klaus), 1 inaktiv (>60T) → Utilization ~92% → keine Recommendation
  ...MOCK_USERS.filter((u) => !['u-07', 'u-11'].includes(u.userId)).map((u, i) => ({
    userId: u.userId,
    userName: u.userName,
    moduleId: 'projects' as ModuleId,
    grantedAt: daysAgo(180),
    // User 8 (Hannah) inaktiv, alle anderen aktiv
    lastActiveAt: u.userId === 'u-08' ? daysAgo(65) : daysAgo(i % 12),
  })),

  // Vertraege — 6 User, 5 aktiv (1 inaktiv >60T aber Utilization ~83% → über 30%)
  { userId: 'u-01', userName: 'Anna Müller', moduleId: 'vertraege', grantedAt: daysAgo(180), lastActiveAt: daysAgo(3) },
  { userId: 'u-02', userName: 'Bernd Schmidt', moduleId: 'vertraege', grantedAt: daysAgo(180), lastActiveAt: daysAgo(7) },
  { userId: 'u-03', userName: 'Carla Hoffmann', moduleId: 'vertraege', grantedAt: daysAgo(180), lastActiveAt: daysAgo(12) },
  { userId: 'u-04', userName: 'Daniel Weber', moduleId: 'vertraege', grantedAt: daysAgo(180), lastActiveAt: daysAgo(2) },
  { userId: 'u-05', userName: 'Eva König', moduleId: 'vertraege', grantedAt: daysAgo(180), lastActiveAt: daysAgo(9) },
  { userId: 'u-06', userName: 'Felix Bauer', moduleId: 'vertraege', grantedAt: daysAgo(180), lastActiveAt: daysAgo(65) },
]

// ───────────────────────── Mock-Invoices ─────────────────────────

export interface MockInvoice {
  id: string
  number: string
  periodStart: string
  periodEnd: string
  amount: number
  status: 'paid' | 'open' | 'overdue'
  paidAt: string | null
}

export const MOCK_INVOICES: MockInvoice[] = [
  { id: 'inv-1', number: 'COSMI-2026-04', periodStart: '2026-04-01', periodEnd: '2026-04-30', amount: 287.5, status: 'open', paidAt: null },
  { id: 'inv-2', number: 'COSMI-2026-03', periodStart: '2026-03-01', periodEnd: '2026-03-31', amount: 287.5, status: 'paid', paidAt: '2026-04-02' },
  { id: 'inv-3', number: 'COSMI-2026-02', periodStart: '2026-02-01', periodEnd: '2026-02-28', amount: 264.2, status: 'paid', paidAt: '2026-03-02' },
  { id: 'inv-4', number: 'COSMI-2026-01', periodStart: '2026-01-01', periodEnd: '2026-01-31', amount: 264.2, status: 'paid', paidAt: '2026-02-02' },
]

// ───────────────────────── generateMockUsage ─────────────────────────

/**
 * Generiert ModuleUsageStats aus UserModuleGrants.
 * Deterministisch: User gilt als aktiv wenn lastActiveAt innerhalb von observationDays.
 */
export function generateMockUsage(
  grants: UserModuleGrant[],
  observationDays: number,
): ModuleUsageStats[] {
  const now = Date.now()
  const windowMs = observationDays * 24 * 60 * 60 * 1000

  // Gruppiere nach ModuleId
  const byModule = new Map<ModuleId, UserModuleGrant[]>()
  for (const grant of grants) {
    const list = byModule.get(grant.moduleId) ?? []
    list.push(grant)
    byModule.set(grant.moduleId, list)
  }

  const result: ModuleUsageStats[] = []

  for (const [moduleId, moduleGrants] of byModule) {
    const assignedUserCount = moduleGrants.length

    const activeGrants = moduleGrants.filter((g) => {
      if (g.lastActiveAt === null) return false
      const lastActive = new Date(g.lastActiveAt).getTime()
      return now - lastActive <= windowMs
    })

    const inactiveGrants = moduleGrants.filter((g) => {
      if (g.lastActiveAt === null) return true
      const lastActive = new Date(g.lastActiveAt).getTime()
      return now - lastActive > windowMs
    })

    const activeUserCount = activeGrants.length
    const utilizationPercent =
      assignedUserCount > 0 ? Math.round((activeUserCount / assignedUserCount) * 100) : 0

    result.push({
      moduleId,
      assignedUserCount,
      activeUserCount,
      utilizationPercent,
      inactiveUsers: inactiveGrants.map((g) => ({
        userId: g.userId,
        userName: g.userName,
        lastActiveAt: g.lastActiveAt,
      })),
    })
  }

  return result
}
