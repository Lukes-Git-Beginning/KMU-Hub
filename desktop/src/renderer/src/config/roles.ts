/**
 * Role-based access control configuration.
 *
 * Defines 5 roles with different module visibility:
 * - Admin: Full access to everything
 * - Projektleiter: Projects + team overview, no finance/infra
 * - Mitarbeiter: Own work only, minimal management
 * - HR-Manager: Like Mitarbeiter + full team/HR access
 * - IT-Support: Like Mitarbeiter + infrastructure/system
 *
 * IMPORTANT: Items without roles in the nav are visible to ALL.
 * Restricted items are completely INVISIBLE (not greyed out).
 */
import type { User } from '@/stores/auth'

// ---- Role IDs ----
export type RoleId = 'admin' | 'manager' | 'member' | 'hr' | 'it_support'

// ---- Which sidebar nav-item IDs each role can see ----
// Items NOT listed here are visible to everyone (dashboard, projects, tasks, chat, meetings, calendar, documents, mail, contacts)
export const RESTRICTED_NAV_ITEMS: Record<string, RoleId[]> = {
  team:           ['admin', 'manager', 'hr', 'member', 'it_support'],
  finance:        ['admin'],
  infrastructure: ['admin', 'it_support'],
  'security-admin': ['admin'],
  settings:       ['admin', 'it_support'],
}

// ---- Which settings tabs each role can see ----
export const SETTINGS_TAB_ROLES: Record<string, RoleId[]> = {
  // Everyone sees: profile, appearance, language, security, notifications, about
  mail:     ['admin', 'manager', 'member', 'hr'],
  calendar: ['admin', 'manager', 'member', 'hr'],
  finance:  ['admin'],
  company:       ['admin'],
  billing:       ['admin'],
  integrations:  ['admin'],
  business:      ['admin'],
  team:     ['admin', 'hr'],
  privacy:  ['admin', 'it_support'],
  ai:       ['admin'],
}

// ---- Which team-module tabs each role can see ----
// Tabs NOT listed = visible to all (members, absences, orgchart, selfservice).
// Personalakte and HR-Integrationen are hr_only (DSGVO). Korrekturen + Schulungen + Settings = manager+hr+admin.
export const TEAM_TAB_ROLES: Record<string, RoleId[]> = {
  requests:      ['admin', 'manager', 'hr'],
  korrekturen:   ['admin', 'manager', 'hr'],
  personalakte:  ['admin', 'hr'],
  integrationen: ['admin', 'hr'],
  schulungen:    ['admin', 'manager', 'hr'],
  einstellungen: ['admin', 'hr'],
  onboarding:    ['admin', 'manager', 'hr'],
}

// ---- Mock user profiles for design testing ----
export interface DevProfile {
  id: RoleId
  user: User
  label: string
  description: string
  color: string
  initials: string
}

export const DEV_PROFILES: DevProfile[] = [
  {
    id: 'admin',
    user: {
      id: 'u-admin',
      firstName: 'Markus',
      lastName: 'Weber',
      email: 'markus.weber@firma.de',
      roles: ['admin'],
    },
    label: 'config.roles.admin.label',
    description: 'config.roles.admin.description',
    color: 'hsl(0 72% 51%)',
    initials: 'MW',
  },
  {
    id: 'manager',
    user: {
      id: 'u-pm',
      firstName: 'Sarah',
      lastName: 'Müller',
      email: 'sarah.mueller@firma.de',
      roles: ['manager'],
    },
    label: 'config.roles.manager.label',
    description: 'config.roles.manager.description',
    color: 'hsl(217 91% 60%)',
    initials: 'SM',
  },
  {
    id: 'member',
    user: {
      id: 'u-dev',
      firstName: 'Lukas',
      lastName: 'Brunner',
      email: 'lukas.brunner@firma.de',
      roles: ['member'],
    },
    label: 'config.roles.member.label',
    description: 'config.roles.member.description',
    color: 'hsl(142 71% 45%)',
    initials: 'LB',
  },
  {
    id: 'hr',
    user: {
      id: 'u-hr',
      firstName: 'Nina',
      lastName: 'Fischer',
      email: 'nina.fischer@firma.de',
      roles: ['hr'],
    },
    label: 'config.roles.hr.label',
    description: 'config.roles.hr.description',
    color: 'hsl(270 76% 55%)',
    initials: 'NF',
  },
  {
    id: 'it_support',
    user: {
      id: 'u-it',
      firstName: 'Thomas',
      lastName: 'Keller',
      email: 'thomas.keller@firma.de',
      roles: ['it_support'],
    },
    label: 'config.roles.it_support.label',
    description: 'config.roles.it_support.description',
    color: 'hsl(25 95% 53%)',
    initials: 'TK',
  },
]

// ---- Helper ----
export function userHasRole(user: User | null, allowedRoles: RoleId[]): boolean {
  if (!user?.roles) return false
  return allowedRoles.some((r) => user.roles.includes(r))
}

export function canSeeNavItem(user: User | null, navItemId: string): boolean {
  const restriction = RESTRICTED_NAV_ITEMS[navItemId]
  if (!restriction) return true // no restriction = visible to all
  return userHasRole(user, restriction)
}

export function canSeeSettingsTab(user: User | null, tabKey: string): boolean {
  const restriction = SETTINGS_TAB_ROLES[tabKey]
  if (!restriction) return true // no restriction = visible to all
  return userHasRole(user, restriction)
}

export function canSeeTeamTab(user: User | null, tabKey: string): boolean {
  const restriction = TEAM_TAB_ROLES[tabKey]
  if (!restriction) return true // no restriction = visible to all
  return userHasRole(user, restriction)
}
