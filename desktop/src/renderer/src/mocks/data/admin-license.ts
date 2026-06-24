/**
 * Tenant module-activation seed (A-3 Lizenz/Modul).
 *
 * The provisioning layer: which Cosmi modules are licensed/active tenant-wide
 * (distinct from the read-only cost breakdown in the Billing tab, and from the
 * per-user grants in team's module-assignment matrix). Module ids match
 * `ModuleId` in @/lib/pricing; real licensing = Luke's track 🔒.
 */
import type { TenantModule, ModuleGroupId } from '@/api/admin-types'

interface Seed {
  moduleId: string
  group: ModuleGroupId
  active: boolean
  assignedSeats: number
}

// Realistic mix: most core/comm/team modules active with seats; several
// industry/tools modules left inactive (dimmed) so the toggle has something to
// switch on. Seats roughly track the 14-user tenant.
const SEED: Seed[] = [
  // Kern
  { moduleId: 'crm', group: 'core', active: true, assignedSeats: 14 },
  { moduleId: 'tasks', group: 'core', active: true, assignedSeats: 14 },
  { moduleId: 'finance', group: 'core', active: true, assignedSeats: 4 },
  { moduleId: 'calendar', group: 'core', active: true, assignedSeats: 14 },
  { moduleId: 'documents', group: 'core', active: true, assignedSeats: 14 },
  // Kommunikation
  { moduleId: 'chat', group: 'comm', active: true, assignedSeats: 14 },
  { moduleId: 'meetings', group: 'comm', active: true, assignedSeats: 8 },
  { moduleId: 'mail', group: 'comm', active: true, assignedSeats: 14 },
  { moduleId: 'dialer', group: 'comm', active: true, assignedSeats: 8 },
  // Team
  { moduleId: 'team', group: 'team', active: true, assignedSeats: 14 },
  { moduleId: 'zeiterfassung', group: 'team', active: true, assignedSeats: 14 },
  { moduleId: 'schichten', group: 'team', active: false, assignedSeats: 0 },
  // Branche / Projekte
  { moduleId: 'projects', group: 'industry', active: true, assignedSeats: 12 },
  { moduleId: 'vertraege', group: 'industry', active: true, assignedSeats: 6 },
  { moduleId: 'helpdesk', group: 'industry', active: true, assignedSeats: 5 },
  { moduleId: 'inventar', group: 'industry', active: false, assignedSeats: 0 },
  { moduleId: 'einkauf', group: 'industry', active: false, assignedSeats: 0 },
  { moduleId: 'fuhrpark', group: 'industry', active: false, assignedSeats: 0 },
  { moduleId: 'produktion', group: 'industry', active: false, assignedSeats: 0 },
  { moduleId: 'vermietung', group: 'industry', active: false, assignedSeats: 0 },
  // Tools
  { moduleId: 'berichte', group: 'tools', active: true, assignedSeats: 6 },
  { moduleId: 'formulare', group: 'tools', active: true, assignedSeats: 4 },
  { moduleId: 'wiki', group: 'tools', active: true, assignedSeats: 9 },
  { moduleId: 'rapporte', group: 'tools', active: false, assignedSeats: 0 },
]

export const seedTenantModules = (): TenantModule[] => SEED.map((m) => ({ ...m }))
