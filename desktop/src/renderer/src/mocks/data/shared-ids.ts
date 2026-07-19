/**
 * Central cross-reference IDs — single source of truth for all mock data.
 * Every data file imports from here to ensure referential integrity.
 */

export const IDS = {
  // Users (map to mock-db employees)
  users: {
    admin: 'usr-001',
    stefan: 'usr-e1',
    markus: 'usr-e2',
    thomas: 'usr-e3',
    julia: 'usr-e4',
    laura: 'usr-e5',
    felix: 'usr-e6',
    sarah: 'usr-e7',
    jan: 'usr-e8',
    lena: 'usr-e9',
    david: 'usr-e10',
    lisa: 'usr-e11',
    anna: 'usr-e12',
    michael: 'usr-e13',
    sophie: 'usr-e14',
    nina: 'usr-e15',
    max: 'usr-e16',
    christian: 'usr-e17',
    elena: 'usr-e18',
  },

  // CRM Contacts (external people)
  contacts: {
    mueller: 'ct-001',
    weber: 'ct-002',
    gruber: 'ct-003',
    schneider: 'ct-004',
    huber: 'ct-005',
    bauer: 'ct-006',
    wagner: 'ct-007',
    berger: 'ct-008',
    koch: 'ct-009',
    richter: 'ct-010',
    schmid: 'ct-011',
    brunner: 'ct-012',
    steiner: 'ct-013',
    hofmann: 'ct-014',
    keller: 'ct-015',
    lang: 'ct-016',
    maier: 'ct-017',
    peters: 'ct-018',
    roth: 'ct-019',
    zimmermann: 'ct-020',
    egger: 'ct-021',
    hofer: 'ct-022',
    winkler: 'ct-023',
    schwarz: 'ct-024',
  },

  // CRM Companies
  companies: {
    gruberMaschinenbau: 'co-001',
    helvetiaSoftware: 'co-002',
    alpenLogistik: 'co-003',
    rheinConsulting: 'co-004',
    bavariaElektro: 'co-005',
    zurichFintech: 'co-006',
    wienerDesign: 'co-007',
    schwarzwaldHolz: 'co-008',
    nordlichtMedia: 'co-009',
    donauPharma: 'co-010',
    bernSolar: 'co-011',
    hanseatischIT: 'co-012',
  },

  // CRM Deals
  deals: {
    crmLizenz: 'dl-001',
    supportVertrag: 'dl-002',
    webRedesign: 'dl-003',
    erpMigration: 'dl-004',
    cloudSetup: 'dl-005',
    schulungsPaket: 'dl-006',
    apiIntegration: 'dl-007',
    sicherheitsAudit: 'dl-008',
    mobileApp: 'dl-009',
    datenbankOptimierung: 'dl-010',
    hostingVertrag: 'dl-011',
    consultingProjekt: 'dl-012',
    iotPlattform: 'dl-013',
    hrDigitalisierung: 'dl-014',
    ecommercePlattform: 'dl-015',
    kiBeratung: 'dl-016',
    backupLoesung: 'dl-017',
    netzwerkUpgrade: 'dl-018',
  },

  // Pipeline Stages
  stages: {
    lead: 'stg-lead',
    qualified: 'stg-qual',
    proposal: 'stg-prop',
    negotiation: 'stg-neg',
    won: 'stg-won',
    lost: 'stg-lost',
  },

  // Projects
  projects: {
    hubV2: 'prj-001',
    websiteRelaunch: 'prj-002',
    mobileApp: 'prj-003',
    infrastruktur: 'prj-004',
    datenschutz: 'prj-005',
    onboarding: 'prj-006',
  },

  // Chat Channels
  channels: {
    allgemein: 'ch-001',
    entwicklung: 'ch-002',
    vertrieb: 'ch-003',
    design: 'ch-004',
    support: 'ch-005',
    random: 'ch-006',
    projektAlpha: 'ch-007',
    announcements: 'ch-008',
  },

  // DM Channels
  dms: {
    stefanMarkus: 'dm-001',
    stefanJulia: 'dm-002',
    stefanThomas: 'dm-003',
  },

  // Calendar
  calendars: {
    personal: 'cal-001',
    team: 'cal-002',
    meetings: 'cal-003',
  },

  // Email
  emailAccounts: {
    main: 'email-acc-001',
  },
  emailFolders: {
    inbox: 'ef-inbox',
    sent: 'ef-sent',
    drafts: 'ef-drafts',
    trash: 'ef-trash',
    archive: 'ef-archive',
  },

  // Documents
  folders: {
    root: 'fld-root',
    projekte: 'fld-projekte',
    verträge: 'fld-verträge',
    rechnungen: 'fld-rechnungen',
    marketing: 'fld-marketing',
    vorlagen: 'fld-vorlagen',
    personal: 'fld-personal',
  },

  // Finance
  invoices: {
    inv001: 'inv-001',
    inv002: 'inv-002',
    inv003: 'inv-003',
    inv004: 'inv-004',
    inv005: 'inv-005',
    inv006: 'inv-006',
    inv007: 'inv-007',
    inv008: 'inv-008',
    inv009: 'inv-009',
    inv010: 'inv-010',
  },

  // Automation
  workflows: {
    leadScoring: 'wf-001',
    onboardingEmail: 'wf-002',
    ticketEskalation: 'wf-003',
    rechnungsMahnung: 'wf-004',
    dailyReport: 'wf-005',
    backupCheck: 'wf-006',
    vertragsErinnerung: 'wf-007',
    feedbackSammlung: 'wf-008',
  },
  // Dialer
  dialer: {
    // Campaigns
    kampagneKaltAkquise: 'dlr-camp-001',
    kampagneBestandskunden: 'dlr-camp-002',

    // Campaign contacts
    cc001: 'dlr-cc-001',
    cc002: 'dlr-cc-002',
    cc003: 'dlr-cc-003',
    cc004: 'dlr-cc-004',
    cc005: 'dlr-cc-005',
    cc006: 'dlr-cc-006',
    cc007: 'dlr-cc-007',
    cc008: 'dlr-cc-008',

    // Call outcomes
    outcomeErreicht: 'dlr-out-001',
    outcomeNichtErreicht: 'dlr-out-002',
    outcomeWiedervorlage: 'dlr-out-003',
    outcomeTermin: 'dlr-out-004',

    // Call sessions
    session001: 'dlr-sess-001',
  },
} as const

/**
 * The currently authenticated demo user — single source of truth for the
 * logged-in identity across ALL mock data (auth/me, seed authors, owners,
 * email from/to, chat senders, calendar attendees, ...).
 *
 * Maps to mock-db employee `e1`. In production this is replaced by the real
 * authenticated user from the backend, so NEVER hardcode the name/email in
 * individual handlers or seeds — reference CURRENT_USER instead.
 */
export const CURRENT_USER = {
  id: IDS.users.stefan,
  firstName: 'Stefan',
  lastName: 'Vogel',
  name: 'Stefan Vogel',
  email: 'stefan.vogel@techvision.de',
  jobTitle: 'Geschäftsführer',
  company: 'TechVision GmbH',
} as const

/**
 * Canonical display names per user id — reverse of the roster mapping in
 * handlers/team.ts (NAME_TO_USER_ID). Seeds that reference people by id
 * resolve their display label here instead of hardcoding names; ids without
 * an entry (or free-text legacy values) pass through unchanged.
 */
export const USER_DISPLAY_NAMES: Record<string, string> = {
  [IDS.users.stefan]: 'Stefan Vogel',
  [IDS.users.markus]: 'Markus Weber',
  [IDS.users.thomas]: 'Thomas Keller',
  [IDS.users.julia]: 'Julia Hofmann',
  [IDS.users.laura]: 'Laura Neumann',
  [IDS.users.felix]: 'Felix Krause',
  [IDS.users.sarah]: 'Sarah Müller',
  [IDS.users.jan]: 'Jan Schäfer',
  [IDS.users.lena]: 'Lena Braun',
  [IDS.users.david]: 'David Wagner',
  [IDS.users.anna]: 'Anna Großmann',
  [IDS.users.michael]: 'Michael Schulz',
  [IDS.users.sophie]: 'Sophie Lang',
  [IDS.users.nina]: 'Nina Fischer',
  [IDS.users.max]: 'Max Steiner',
  [IDS.users.elena]: 'Elena Richter',
}

/** Display name for a user id; free-text legacy values pass through. */
export function displayUserName(idOrName: string | null | undefined): string {
  if (!idOrName) return ''
  return USER_DISPLAY_NAMES[idOrName] ?? idOrName
}
