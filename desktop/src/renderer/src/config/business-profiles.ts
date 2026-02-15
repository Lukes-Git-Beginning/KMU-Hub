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
export const ALWAYS_VISIBLE_MODULES = ['dashboard', 'settings', 'profil', 'notifications']

export const BUSINESS_PROFILES: BusinessProfile[] = [
  {
    id: 'allgemein',
    name: 'Allgemein (Büro)',
    description: 'Standard-Büroarbeit mit CRM, Projekten und Administration',
    emoji: '\u{1F3E2}',
    color: 'hsl(217 91% 60%)',
    defaultModules: [
      'crm', 'projects', 'tasks', 'chat', 'calendar', 'meetings',
      'documents', 'mail', 'contacts', 'team', 'finance',
    ],
    optionalModules: ['berichte', 'helpdesk'],
    examples: ['Beratungsfirmen', 'Agenturen', 'Kanzleien', 'Treuhänder'],
  },
  {
    id: 'handwerk',
    name: 'Handwerk',
    description: 'Aussendienst, Projekte, Werkzeuge & Materialverwaltung',
    emoji: '\u{1F527}',
    color: 'hsl(25 95% 53%)',
    defaultModules: [
      'crm', 'projects', 'calendar', 'einkauf', 'inventar',
      'fuhrpark', 'documents', 'team', 'finance',
    ],
    optionalModules: ['schichten', 'chat', 'meetings'],
    examples: ['Elektriker', 'Sanitär', 'Schreiner', 'Maler', 'HVAC'],
  },
  {
    id: 'gastronomie',
    name: 'Gastronomie',
    description: 'Schichtbetrieb, Inventar mit Ablaufdaten',
    emoji: '\u{1F373}',
    color: 'hsl(0 72% 51%)',
    defaultModules: [
      'inventar', 'schichten', 'einkauf',
      'team', 'finance', 'calendar',
    ],
    optionalModules: ['crm', 'chat', 'berichte'],
    examples: ['Restaurants', 'Cafes', 'Catering', 'Bäckereien', 'Hotels'],
  },
  {
    id: 'einzelhandel',
    name: 'Einzelhandel',
    description: 'Lagerverwaltung, Kundenbetreuung, Bestellwesen',
    emoji: '\u{1F6D2}',
    color: 'hsl(262 83% 58%)',
    defaultModules: [
      'inventar', 'crm', 'einkauf', 'schichten',
      'team', 'finance',
    ],
    optionalModules: ['chat', 'berichte', 'meetings'],
    examples: ['Elektronik', 'Boutique', 'Möbel', 'Buchhandlung'],
  },
  {
    id: 'dienstleistung',
    name: 'Dienstleistung',
    description: 'Terminbasiert, kundenorientiert, wenig Inventar',
    emoji: '\u{1F485}',
    color: 'hsl(330 81% 60%)',
    defaultModules: [
      'calendar', 'crm', 'mail', 'documents',
      'team', 'finance', 'chat',
    ],
    optionalModules: ['projects', 'meetings', 'helpdesk', 'berichte'],
    examples: ['Friseursalon', 'Kosmetik', 'Reinigung', 'Berater'],
  },
  {
    id: 'it_tech',
    name: 'IT / Tech',
    description: 'Projekte, Helpdesk, Infrastruktur-Management',
    emoji: '\u{1F4BB}',
    color: 'hsl(142 71% 45%)',
    defaultModules: [
      'crm', 'projects', 'tasks', 'helpdesk', 'chat', 'meetings',
      'documents', 'mail', 'team', 'infrastructure', 'finance',
    ],
    optionalModules: ['berichte', 'inventar'],
    examples: ['Softwarefirmen', 'IT-Support', 'SaaS', 'Web-Agenturen'],
  },
  {
    id: 'produktion',
    name: 'Produktion',
    description: 'Fertigungsplanung, Rohstoffe, Qualitätskontrolle',
    emoji: '\u{1F3ED}',
    color: 'hsl(45 93% 47%)',
    defaultModules: [
      'produktion', 'inventar', 'einkauf', 'schichten',
      'team', 'finance', 'calendar',
    ],
    optionalModules: ['crm', 'fuhrpark', 'berichte', 'documents'],
    examples: ['Lebensmittel', 'Metallbau', 'Textil', 'Elektronik-Montage'],
  },
  {
    id: 'logistik',
    name: 'Logistik',
    description: 'Fuhrpark, Lager, Routenplanung',
    emoji: '\u{1F69A}',
    color: 'hsl(199 89% 48%)',
    defaultModules: [
      'fuhrpark', 'inventar', 'schichten', 'crm',
      'team', 'finance', 'calendar',
    ],
    optionalModules: ['einkauf', 'chat', 'berichte', 'documents'],
    examples: ['Kurierdienste', 'Speditionen', 'Lagerhaltung'],
  },
  {
    id: 'gesundheit',
    name: 'Gesundheit',
    description: 'Terminverwaltung, Patientenakten, Compliance',
    emoji: '\u{1FA7A}',
    color: 'hsl(174 62% 47%)',
    defaultModules: [
      'calendar', 'crm', 'documents',
      'team', 'finance', 'mail',
    ],
    optionalModules: ['schichten', 'inventar', 'meetings', 'helpdesk', 'berichte'],
    examples: ['Arztpraxen', 'Zahnärzte', 'Physiotherapie', 'Tierärzte'],
  },
  {
    id: 'bau',
    name: 'Bau',
    description: 'Projektbasiert, Material, Fahrzeuge, Feldteams',
    emoji: '\u{1F3D7}\uFE0F',
    color: 'hsl(32 95% 44%)',
    defaultModules: [
      'projects', 'inventar', 'einkauf', 'fuhrpark',
      'team', 'schichten', 'finance', 'calendar',
    ],
    optionalModules: ['crm', 'documents', 'chat', 'berichte'],
    examples: ['Bauunternehmen', 'Renovation', 'Landschaftsbau'],
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
