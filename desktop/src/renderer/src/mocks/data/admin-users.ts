/**
 * Admin account roster — the tenant's LOGIN accounts (access layer), distinct
 * from the HR personnel records in the `team` module. Same people, different
 * lens: here we model who can sign in, their role, account status and last
 * login — NOT contracts/payroll (that lives in HR).
 *
 * Seeded from the shared identity source (`IDS.users` + `CURRENT_USER`) so the
 * admin user list and the team roster reference the same people. Roles are NOT
 * stored here (multi-role since R-2): `USER_ROLE_ASSIGNMENTS` in
 * `mocks/data/rbac.ts` is the single source, the admin handlers compose
 * `AdminUser.roles` from it on read. NEVER hardcode the current user's name —
 * reference `CURRENT_USER`.
 *
 * Real backend (account provisioning, invite tokens, SSO) is Luke's track 🔒.
 */
import type { AdminUser } from '@/api/admin-types'
import { IDS, CURRENT_USER } from './shared-ids'
import { hoursAgo, minutesAgo } from './date-helpers'

/** Stored account row — roles are composed from USER_ROLE_ASSIGNMENTS on read. */
export type AdminUserRecord = Omit<AdminUser, 'roles'>

const days = (n: number) => hoursAgo(n * 24)

/**
 * Initial roster: 11 active + 2 invited + 1 deactivated = 14 rows.
 * 13 of these consume a seat (active + pending invites); the deactivated
 * account does not — keeping the count consistent with the tenant's seat usage
 * surfaced in A-3 (Lizenz/Modul).
 */
export const seedAdminUsers = (): AdminUserRecord[] => [
  {
    // Current demo identity — Geschäftsführer / owner. Name from CURRENT_USER.
    id: CURRENT_USER.id,
    firstName: CURRENT_USER.firstName,
    lastName: CURRENT_USER.lastName,
    email: CURRENT_USER.email,
    jobTitle: CURRENT_USER.jobTitle,    status: 'active',
    lastLoginAt: minutesAgo(4),
    invitedAt: null,
  },
  {
    id: IDS.users.sarah,
    firstName: 'Sarah',
    lastName: 'Müller',
    email: 'sarah.mueller@techvision.de',
    jobTitle: 'Teamleiterin Projekte',    status: 'active',
    lastLoginAt: hoursAgo(2),
    invitedAt: null,
  },
  {
    id: IDS.users.thomas,
    firstName: 'Thomas',
    lastName: 'Keller',
    email: 'thomas.keller@techvision.de',
    jobTitle: 'IT-Administrator',    status: 'active',
    lastLoginAt: minutesAgo(38),
    invitedAt: null,
  },
  {
    id: IDS.users.nina,
    firstName: 'Nina',
    lastName: 'Fischer',
    email: 'nina.fischer@techvision.de',
    jobTitle: 'HR-Managerin',    status: 'active',
    lastLoginAt: hoursAgo(5),
    invitedAt: null,
  },
  {
    id: IDS.users.markus,
    firstName: 'Markus',
    lastName: 'Weber',
    email: 'markus.weber@techvision.de',
    jobTitle: 'Senior Developer',    status: 'active',
    lastLoginAt: hoursAgo(1),
    invitedAt: null,
  },
  {
    id: IDS.users.laura,
    firstName: 'Laura',
    lastName: 'Neumann',
    email: 'laura.neumann@techvision.de',
    jobTitle: 'Account Managerin',    status: 'active',
    lastLoginAt: hoursAgo(7),
    invitedAt: null,
  },
  {
    id: IDS.users.felix,
    firstName: 'Felix',
    lastName: 'Krause',
    email: 'felix.krause@techvision.de',
    jobTitle: 'Sales Development Rep',    status: 'active',
    lastLoginAt: days(1),
    invitedAt: null,
  },
  {
    id: IDS.users.julia,
    firstName: 'Julia',
    lastName: 'Hofmann',
    email: 'julia.hofmann@techvision.de',
    jobTitle: 'UX Designerin',    status: 'active',
    lastLoginAt: hoursAgo(9),
    invitedAt: null,
  },
  {
    id: IDS.users.jan,
    firstName: 'Jan',
    lastName: 'Schäfer',
    email: 'jan.schaefer@techvision.de',
    jobTitle: 'Backend Engineer',    status: 'active',
    lastLoginAt: days(2),
    invitedAt: null,
  },
  {
    id: IDS.users.lena,
    firstName: 'Lena',
    lastName: 'Braun',
    email: 'lena.braun@techvision.de',
    jobTitle: 'Support-Spezialistin',    status: 'active',
    lastLoginAt: days(4),
    invitedAt: null,
  },
  {
    id: IDS.users.david,
    firstName: 'David',
    lastName: 'Wagner',
    email: 'david.wagner@techvision.de',
    jobTitle: 'DevOps Engineer',    status: 'active',
    lastLoginAt: days(11),
    invitedAt: null,
  },
  {
    id: IDS.users.lisa,
    firstName: 'Lisa',
    lastName: 'Becker',
    email: 'lisa.becker@techvision.de',
    jobTitle: 'Marketing Managerin',    status: 'invited',
    lastLoginAt: null,
    invitedAt: days(2),
  },
  {
    id: IDS.users.michael,
    firstName: 'Michael',
    lastName: 'Schulz',
    email: 'michael.schulz@techvision.de',
    jobTitle: 'Buchhalter',    status: 'invited',
    lastLoginAt: null,
    invitedAt: hoursAgo(20),
  },
  {
    id: IDS.users.anna,
    firstName: 'Anna',
    lastName: 'Großmann',
    email: 'anna.grossmann@techvision.de',
    jobTitle: 'Office Managerin',    status: 'deactivated',
    lastLoginAt: days(96),
    invitedAt: null,
  },
  {
    // Read-only preset demo (external tax advisor — sees, never edits)
    id: IDS.users.elena,
    firstName: 'Elena',
    lastName: 'Richter',
    email: 'elena.richter@extern.de',
    jobTitle: 'Steuerberaterin (extern)',    status: 'active',
    lastLoginAt: days(3),
    invitedAt: null,
  },
  {
    // Aushilfe/Extern preset demo (Dariens Referenzfall)
    id: IDS.users.max,
    firstName: 'Max',
    lastName: 'Steiner',
    email: 'max.steiner@extern.de',
    jobTitle: 'Aushilfe Lager',    status: 'active',
    lastLoginAt: days(1),
    invitedAt: null,
  },
]
