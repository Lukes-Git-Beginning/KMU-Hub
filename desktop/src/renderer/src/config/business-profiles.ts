/**
 * Business profile definitions for different industries.
 *
 * Each profile determines which modules are visible by default
 * and which can be optionally enabled. The profile is selected
 * during initial setup (onsite configuration).
 *
 * Module IDs must match the `id` field in nav-items.ts.
 */

export type BusinessProfileId =
  | 'allgemein'
  | 'handwerk'
  | 'gastronomie'
  | 'einzelhandel'
  | 'dienstleistung'
  | 'it_tech'
  | 'produktion'
  | 'logistik'
  | 'gesundheit'
  | 'bau'

export interface BusinessProfile {
  id: BusinessProfileId
  name: string
  description: string
  emoji: string
  color: string
  defaultModules: string[]
  optionalModules: string[]
  examples: string[]
}

/** Always-visible modules that every profile includes implicitly */
export const ALWAYS_VISIBLE_MODULES = ['dashboard', 'settings', 'security-admin', 'profil', 'notifications', 'contacts']

export const BUSINESS_PROFILES: BusinessProfile[] = [
  {
    id: 'allgemein',
    name: 'config.businessProfile.allgemein.name',
    description: 'config.businessProfile.allgemein.description',
    emoji: '\u{1F3E2}',
    color: 'hsl(217 91% 60%)',
    defaultModules: [
      'crm', 'projects', 'tasks', 'chat', 'calendar', 'meetings',
      'documents', 'mail', 'contacts', 'team', 'finance', 'zeiterfassung',
    ],
    optionalModules: ['berichte', 'helpdesk', 'verträge', 'formulare'],
    examples: ['config.businessProfile.allgemein.example.0', 'config.businessProfile.allgemein.example.1', 'config.businessProfile.allgemein.example.2', 'config.businessProfile.allgemein.example.3'],
  },
  {
    id: 'handwerk',
    name: 'config.businessProfile.handwerk.name',
    description: 'config.businessProfile.handwerk.description',
    emoji: '\u{1F527}',
    color: 'hsl(25 95% 53%)',
    defaultModules: [
      'crm', 'projects', 'calendar', 'einkauf', 'inventar',
      'fuhrpark', 'documents', 'team', 'finance', 'rapporte', 'zeiterfassung',
    ],
    optionalModules: ['schichten', 'chat', 'meetings', 'verträge', 'vermietung', 'formulare'],
    examples: ['config.businessProfile.handwerk.example.0', 'config.businessProfile.handwerk.example.1', 'config.businessProfile.handwerk.example.2', 'config.businessProfile.handwerk.example.3', 'config.businessProfile.handwerk.example.4'],
  },
  {
    id: 'gastronomie',
    name: 'config.businessProfile.gastronomie.name',
    description: 'config.businessProfile.gastronomie.description',
    emoji: '\u{1F373}',
    color: 'hsl(0 72% 51%)',
    defaultModules: [
      'inventar', 'schichten', 'einkauf',
      'team', 'finance', 'calendar', 'zeiterfassung',
    ],
    optionalModules: ['crm', 'chat', 'berichte', 'verträge', 'formulare'],
    examples: ['config.businessProfile.gastronomie.example.0', 'config.businessProfile.gastronomie.example.1', 'config.businessProfile.gastronomie.example.2', 'config.businessProfile.gastronomie.example.3', 'config.businessProfile.gastronomie.example.4'],
  },
  {
    id: 'einzelhandel',
    name: 'config.businessProfile.einzelhandel.name',
    description: 'config.businessProfile.einzelhandel.description',
    emoji: '\u{1F6D2}',
    color: 'hsl(262 83% 58%)',
    defaultModules: [
      'inventar', 'crm', 'einkauf', 'schichten',
      'team', 'finance', 'zeiterfassung',
    ],
    optionalModules: ['chat', 'berichte', 'meetings', 'verträge', 'vermietung', 'formulare'],
    examples: ['config.businessProfile.einzelhandel.example.0', 'config.businessProfile.einzelhandel.example.1', 'config.businessProfile.einzelhandel.example.2', 'config.businessProfile.einzelhandel.example.3'],
  },
  {
    id: 'dienstleistung',
    name: 'config.businessProfile.dienstleistung.name',
    description: 'config.businessProfile.dienstleistung.description',
    emoji: '\u{1F485}',
    color: 'hsl(330 81% 60%)',
    defaultModules: [
      'calendar', 'crm', 'mail', 'documents',
      'team', 'finance', 'chat', 'zeiterfassung', 'verträge',
    ],
    optionalModules: ['projects', 'meetings', 'helpdesk', 'berichte', 'vermietung', 'formulare', 'rapporte'],
    examples: ['config.businessProfile.dienstleistung.example.0', 'config.businessProfile.dienstleistung.example.1', 'config.businessProfile.dienstleistung.example.2', 'config.businessProfile.dienstleistung.example.3'],
  },
  {
    id: 'it_tech',
    name: 'config.businessProfile.it_tech.name',
    description: 'config.businessProfile.it_tech.description',
    emoji: '\u{1F4BB}',
    color: 'hsl(142 71% 45%)',
    defaultModules: [
      'crm', 'projects', 'tasks', 'helpdesk', 'chat', 'meetings',
      'documents', 'mail', 'team', 'infrastructure', 'finance', 'zeiterfassung', 'verträge',
    ],
    optionalModules: ['berichte', 'inventar', 'formulare'],
    examples: ['config.businessProfile.it_tech.example.0', 'config.businessProfile.it_tech.example.1', 'config.businessProfile.it_tech.example.2', 'config.businessProfile.it_tech.example.3'],
  },
  {
    id: 'produktion',
    name: 'config.businessProfile.produktion.name',
    description: 'config.businessProfile.produktion.description',
    emoji: '\u{1F3ED}',
    color: 'hsl(45 93% 47%)',
    defaultModules: [
      'produktion', 'inventar', 'einkauf', 'schichten',
      'team', 'finance', 'calendar', 'zeiterfassung',
    ],
    optionalModules: ['crm', 'fuhrpark', 'berichte', 'documents', 'rapporte', 'formulare'],
    examples: ['config.businessProfile.produktion.example.0', 'config.businessProfile.produktion.example.1', 'config.businessProfile.produktion.example.2', 'config.businessProfile.produktion.example.3'],
  },
  {
    id: 'logistik',
    name: 'config.businessProfile.logistik.name',
    description: 'config.businessProfile.logistik.description',
    emoji: '\u{1F69A}',
    color: 'hsl(199 89% 48%)',
    defaultModules: [
      'fuhrpark', 'inventar', 'schichten', 'crm',
      'team', 'finance', 'calendar', 'zeiterfassung',
    ],
    optionalModules: ['einkauf', 'chat', 'berichte', 'documents', 'rapporte', 'vermietung', 'formulare'],
    examples: ['config.businessProfile.logistik.example.0', 'config.businessProfile.logistik.example.1', 'config.businessProfile.logistik.example.2'],
  },
  {
    id: 'gesundheit',
    name: 'config.businessProfile.gesundheit.name',
    description: 'config.businessProfile.gesundheit.description',
    emoji: '\u{1FA7A}',
    color: 'hsl(174 62% 47%)',
    defaultModules: [
      'calendar', 'crm', 'documents',
      'team', 'finance', 'mail', 'zeiterfassung', 'verträge',
    ],
    optionalModules: ['schichten', 'inventar', 'meetings', 'helpdesk', 'berichte', 'formulare'],
    examples: ['config.businessProfile.gesundheit.example.0', 'config.businessProfile.gesundheit.example.1', 'config.businessProfile.gesundheit.example.2', 'config.businessProfile.gesundheit.example.3'],
  },
  {
    id: 'bau',
    name: 'config.businessProfile.bau.name',
    description: 'config.businessProfile.bau.description',
    emoji: '\u{1F3D7}\uFE0F',
    color: 'hsl(32 95% 44%)',
    defaultModules: [
      'projects', 'inventar', 'einkauf', 'fuhrpark',
      'team', 'schichten', 'finance', 'calendar', 'rapporte', 'zeiterfassung',
    ],
    optionalModules: ['crm', 'documents', 'chat', 'berichte', 'vermietung', 'formulare'],
    examples: ['config.businessProfile.bau.example.0', 'config.businessProfile.bau.example.1', 'config.businessProfile.bau.example.2'],
  },
]

export function getProfileById(id: BusinessProfileId): BusinessProfile | undefined {
  return BUSINESS_PROFILES.find((p) => p.id === id)
}

/**
 * Check if a module is allowed for a given profile.
 * Returns true if:
 * - No profile is set (show everything)
 * - Module is in ALWAYS_VISIBLE_MODULES
 * - Module is in defaultModules
 * - Module is in optionalModules AND explicitly enabled
 */
export function isModuleAllowedForProfile(
  moduleId: string,
  profileId: BusinessProfileId | null,
  enabledOptionals: string[],
): boolean {
  // No profile = show all (backwards compat / dev)
  if (!profileId) return true

  // Always-visible modules
  if (ALWAYS_VISIBLE_MODULES.includes(moduleId)) return true

  const profile = getProfileById(profileId)
  if (!profile) return true

  // Default modules are always visible
  if (profile.defaultModules.includes(moduleId)) return true

  // Optional modules only if explicitly enabled
  if (profile.optionalModules.includes(moduleId) && enabledOptionals.includes(moduleId)) {
    return true
  }

  return false
}
