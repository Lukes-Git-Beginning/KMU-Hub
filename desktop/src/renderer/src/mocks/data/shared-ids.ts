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
    vertraege: 'fld-vertraege',
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
} as const
