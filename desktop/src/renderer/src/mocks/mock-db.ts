/**
 * Central mock database — single source of truth for ALL demo data.
 *
 * TechVision GmbH — Software-Agentur, München, 18 Mitarbeiter
 * Deutschland-first: EUR, 19%/7% MwSt, ArbZG, DATEV-kompatibel
 *
 * Replace this with API calls when backend is ready.
 */

// ---------------------------------------------------------------------------
// Company
// ---------------------------------------------------------------------------

export const COMPANY = {
  name: 'TechVision GmbH',
  legalForm: 'GmbH',
  industry: 'Software & IT-Dienstleistungen',
  address: {
    street: 'Leopoldstrasse 120',
    zip: '80802',
    city: 'München',
    state: 'Bayern',
    country: 'Deutschland',
  },
  phone: '+49 89 2468 1350',
  email: 'info@techvision.de',
  website: 'www.techvision.de',
  taxId: 'DE314256789',
  hrNumber: 'HRB 248712',
  founded: '2021-04-01',
  bankAccount: {
    iban: 'DE89 3704 0044 0532 0130 00',
    bic: 'COBADEFFXXX',
    bank: 'Commerzbank München',
  },
}

// ---------------------------------------------------------------------------
// Departments
// ---------------------------------------------------------------------------

export interface MockDepartment {
  id: string
  name: string
  color: string
  headId: string
}

export const DEPARTMENTS: MockDepartment[] = [
  { id: 'mgmt', name: 'Geschäftsführung', color: '#6366f1', headId: 'e1' },
  { id: 'dev', name: 'Entwicklung', color: '#3b82f6', headId: 'e2' },
  { id: 'design', name: 'Design', color: '#ec4899', headId: 'e9' },
  { id: 'sales', name: 'Vertrieb', color: '#f59e0b', headId: 'e3' },
  { id: 'marketing', name: 'Marketing', color: '#8b5cf6', headId: 'e12' },
  { id: 'finance', name: 'Buchhaltung', color: '#10b981', headId: 'e13' },
  { id: 'hr', name: 'Personal', color: '#f97316', headId: 'e15' },
  { id: 'support', name: 'Support', color: '#06b6d4', headId: 'e16' },
  { id: 'pm', name: 'Projektmanagement', color: '#14b8a6', headId: 'e18' },
]

// ---------------------------------------------------------------------------
// Employees (18)
// ---------------------------------------------------------------------------

export type EmployeeRole = 'admin' | 'manager' | 'member'

export interface MockEmployee {
  id: string
  firstName: string
  lastName: string
  initials: string
  email: string
  phone: string
  mobile: string
  role: EmployeeRole
  jobTitle: string
  department: string
  departmentId: string
  contractType: 'full_time' | 'part_time' | 'intern' | 'temporary'
  workload: number
  joinDate: string
  birthday: string
  location: string
  managerId?: string
  salary: number
  skills: string[]
  status: 'online' | 'away' | 'busy' | 'offline'
  currentTask?: string
  avatar?: string
}

export const EMPLOYEES: MockEmployee[] = [
  // --- Geschäftsführung ---
  {
    id: 'e1', firstName: 'Stefan', lastName: 'Vogel', initials: 'SV',
    email: 'stefan.vogel@techvision.de', phone: '+49 89 2468 1351', mobile: '+49 171 234 5678',
    role: 'admin', jobTitle: 'Geschäftsführer', department: 'Geschäftsführung', departmentId: 'mgmt',
    contractType: 'full_time', workload: 100, joinDate: '2021-04-01', birthday: '1978-06-15',
    location: 'München', salary: 12500,
    skills: ['Strategie', 'Business Development', 'Verhandlung'],
    status: 'online', currentTask: 'Quartalsbericht vorbereiten',
  },
  // --- Entwicklung (5) ---
  {
    id: 'e2', firstName: 'Markus', lastName: 'Weber', initials: 'MW',
    email: 'markus.weber@techvision.de', phone: '+49 89 2468 1352', mobile: '+49 172 345 6789',
    role: 'manager', jobTitle: 'CTO / Technischer Leiter', department: 'Entwicklung', departmentId: 'dev',
    contractType: 'full_time', workload: 100, joinDate: '2021-04-01', birthday: '1985-03-22',
    location: 'München', salary: 9800,
    skills: ['Go', 'TypeScript', 'PostgreSQL', 'Docker', 'Kubernetes'],
    status: 'online', currentTask: 'Architektur-Review API v2',
  },
  {
    id: 'e4', firstName: 'Laura', lastName: 'Neumann', initials: 'LN',
    email: 'laura.neumann@techvision.de', phone: '+49 89 2468 1354', mobile: '+49 176 567 8901',
    role: 'member', jobTitle: 'Senior Developerin', department: 'Entwicklung', departmentId: 'dev',
    contractType: 'full_time', workload: 100, joinDate: '2022-01-15', birthday: '1990-11-08',
    location: 'München', managerId: 'e2', salary: 7200,
    skills: ['React', 'TypeScript', 'Node.js', 'GraphQL'],
    status: 'online', currentTask: 'Dashboard Widgets fertigstellen',
  },
  {
    id: 'e5', firstName: 'Felix', lastName: 'Krause', initials: 'FK',
    email: 'felix.krause@techvision.de', phone: '+49 89 2468 1355', mobile: '+49 157 678 9012',
    role: 'member', jobTitle: 'Backend Developer', department: 'Entwicklung', departmentId: 'dev',
    contractType: 'full_time', workload: 100, joinDate: '2022-06-01', birthday: '1992-07-30',
    location: 'München', managerId: 'e2', salary: 6800,
    skills: ['Go', 'PostgreSQL', 'Redis', 'gRPC'],
    status: 'busy', currentTask: 'API-Endpoint /deals optimieren',
  },
  {
    id: 'e6', firstName: 'Tim', lastName: 'Hartmann', initials: 'TH',
    email: 'tim.hartmann@techvision.de', phone: '+49 89 2468 1356', mobile: '+49 151 789 0123',
    role: 'member', jobTitle: 'Frontend Developer', department: 'Entwicklung', departmentId: 'dev',
    contractType: 'full_time', workload: 100, joinDate: '2023-03-01', birthday: '1995-01-17',
    location: 'München', managerId: 'e2', salary: 6200,
    skills: ['React', 'Tailwind CSS', 'Electron', 'Vite'],
    status: 'online', currentTask: 'Kalender-Modul Bug fixen',
  },
  {
    id: 'e7', firstName: 'Lena', lastName: 'Braun', initials: 'LB',
    email: 'lena.braun@techvision.de', phone: '+49 89 2468 1357', mobile: '+49 160 890 1234',
    role: 'member', jobTitle: 'Junior Developerin', department: 'Entwicklung', departmentId: 'dev',
    contractType: 'intern', workload: 60, joinDate: '2025-10-01', birthday: '2001-09-12',
    location: 'München', managerId: 'e2', salary: 2400,
    skills: ['TypeScript', 'React', 'Jest', 'Git'],
    status: 'away', currentTask: 'Unit Tests für CRM-Modul',
  },
  // --- Vertrieb (3) ---
  {
    id: 'e3', firstName: 'Thomas', lastName: 'Meier', initials: 'TM',
    email: 'thomas.meier@techvision.de', phone: '+49 89 2468 1353', mobile: '+49 173 456 7890',
    role: 'manager', jobTitle: 'Vertriebsleiter', department: 'Vertrieb', departmentId: 'sales',
    contractType: 'full_time', workload: 100, joinDate: '2021-09-01', birthday: '1982-12-03',
    location: 'München', salary: 8500,
    skills: ['Verhandlung', 'CRM', 'Key Account Management'],
    status: 'online', currentTask: 'Angebot Gruber Maschinenbau',
  },
  {
    id: 'e10', firstName: 'Sabine', lastName: 'Fischer', initials: 'SF',
    email: 'sabine.fischer@techvision.de', phone: '+49 89 2468 1360', mobile: '+49 175 012 3456',
    role: 'member', jobTitle: 'Account Managerin', department: 'Vertrieb', departmentId: 'sales',
    contractType: 'full_time', workload: 100, joinDate: '2023-01-15', birthday: '1988-04-25',
    location: 'München', managerId: 'e3', salary: 5800,
    skills: ['Account Management', 'Praesentation', 'SalesForce'],
    status: 'away', currentTask: 'Nachfass-Gespräche Pipeline Q1',
  },
  {
    id: 'e11', firstName: 'Kevin', lastName: 'Baumann', initials: 'KB',
    email: 'kevin.baumann@techvision.de', phone: '+49 89 2468 1361', mobile: '+49 179 123 4567',
    role: 'member', jobTitle: 'Sales Representative', department: 'Vertrieb', departmentId: 'sales',
    contractType: 'full_time', workload: 100, joinDate: '2024-06-01', birthday: '1996-08-19',
    location: 'München', managerId: 'e3', salary: 4800,
    skills: ['Kaltakquise', 'LinkedIn Sales', 'Demos'],
    status: 'online', currentTask: 'Demo für MedTech Innovations',
  },
  // --- Design (2) ---
  {
    id: 'e9', firstName: 'Nina', lastName: 'Richter', initials: 'NR',
    email: 'nina.richter@techvision.de', phone: '+49 89 2468 1359', mobile: '+49 174 901 2345',
    role: 'manager', jobTitle: 'Creative Director', department: 'Design', departmentId: 'design',
    contractType: 'full_time', workload: 100, joinDate: '2021-11-01', birthday: '1987-02-14',
    location: 'München', salary: 7500,
    skills: ['Figma', 'Brand Design', 'Design Systems', 'UX Research'],
    status: 'online', currentTask: 'Design System v2 Components',
  },
  {
    id: 'e8', firstName: 'Sophie', lastName: 'Lang', initials: 'SL',
    email: 'sophie.lang@techvision.de', phone: '+49 89 2468 1358', mobile: '+49 163 890 1234',
    role: 'member', jobTitle: 'UX Designerin', department: 'Design', departmentId: 'design',
    contractType: 'part_time', workload: 80, joinDate: '2023-09-01', birthday: '1993-05-28',
    location: 'München', managerId: 'e9', salary: 4800,
    skills: ['Figma', 'User Research', 'Prototyping', 'Accessibility'],
    status: 'online', currentTask: 'Usability Test Ergebnisse auswerten',
  },
  // --- Marketing (1) ---
  {
    id: 'e12', firstName: 'Julia', lastName: 'Hofmann', initials: 'JH',
    email: 'julia.hofmann@techvision.de', phone: '+49 89 2468 1362', mobile: '+49 152 234 5678',
    role: 'member', jobTitle: 'Marketing Managerin', department: 'Marketing', departmentId: 'marketing',
    contractType: 'full_time', workload: 100, joinDate: '2023-04-01', birthday: '1991-10-06',
    location: 'München', salary: 5500,
    skills: ['SEO', 'Content Marketing', 'Google Analytics', 'HubSpot'],
    status: 'online', currentTask: 'Blog-Artikel: KI im Mittelstand',
  },
  // --- Buchhaltung (1) ---
  {
    id: 'e13', firstName: 'Petra', lastName: 'Zimmermann', initials: 'PZ',
    email: 'petra.zimmermann@techvision.de', phone: '+49 89 2468 1363', mobile: '+49 177 345 6789',
    role: 'member', jobTitle: 'Buchhalterin', department: 'Buchhaltung', departmentId: 'finance',
    contractType: 'full_time', workload: 100, joinDate: '2022-03-01', birthday: '1975-08-20',
    location: 'München', salary: 5200,
    skills: ['DATEV', 'Lohnbuchhaltung', 'UStVA', 'BWA'],
    status: 'offline', currentTask: 'Monatsabschluss Januar',
  },
  // --- Office (1) ---
  {
    id: 'e14', firstName: 'Andrea', lastName: 'Keller', initials: 'AK',
    email: 'andrea.keller@techvision.de', phone: '+49 89 2468 1364', mobile: '+49 178 456 7890',
    role: 'member', jobTitle: 'Office Managerin', department: 'Geschäftsführung', departmentId: 'mgmt',
    contractType: 'part_time', workload: 60, joinDate: '2022-08-01', birthday: '1983-11-30',
    location: 'München', managerId: 'e1', salary: 3200,
    skills: ['Organisation', 'Eventplanung', 'Office 365'],
    status: 'online', currentTask: 'Team-Event März planen',
  },
  // --- Personal (1) ---
  {
    id: 'e15', firstName: 'Elena', lastName: 'Schuster', initials: 'ES',
    email: 'elena.schuster@techvision.de', phone: '+49 89 2468 1365', mobile: '+49 159 567 8901',
    role: 'member', jobTitle: 'HR Managerin', department: 'Personal', departmentId: 'hr',
    contractType: 'full_time', workload: 100, joinDate: '2023-02-01', birthday: '1989-04-03',
    location: 'München', salary: 5800,
    skills: ['Recruiting', 'Arbeitsrecht', 'Personio', 'Onboarding'],
    status: 'online', currentTask: 'Bewerbungsgespräche planen',
  },
  // --- Support (2) ---
  {
    id: 'e16', firstName: 'Martin', lastName: 'Wolf', initials: 'MaW',
    email: 'martin.wolf@techvision.de', phone: '+49 89 2468 1366', mobile: '+49 176 678 9012',
    role: 'member', jobTitle: 'Support Engineer', department: 'Support', departmentId: 'support',
    contractType: 'full_time', workload: 100, joinDate: '2023-07-01', birthday: '1994-12-11',
    location: 'München', salary: 5000,
    skills: ['Linux', 'Troubleshooting', 'Jira', 'Zendesk'],
    status: 'online', currentTask: 'Ticket #472 — Login-Problem Gruber',
  },
  {
    id: 'e17', firstName: 'Jonas', lastName: 'Schmitt', initials: 'JS',
    email: 'jonas.schmitt@techvision.de', phone: '+49 89 2468 1367', mobile: '+49 155 789 0123',
    role: 'member', jobTitle: 'Helpdesk Agent', department: 'Support', departmentId: 'support',
    contractType: 'full_time', workload: 100, joinDate: '2024-09-01', birthday: '1998-03-07',
    location: 'München', managerId: 'e16', salary: 3800,
    skills: ['Kundenkommunikation', 'Ticketing', 'FAQ-Pflege'],
    status: 'busy', currentTask: 'FAQ-Artikel aktualisieren',
  },
  // --- Projektmanagement (1) ---
  {
    id: 'e18', firstName: 'Sarah', lastName: 'Beck', initials: 'SB',
    email: 'sarah.beck@techvision.de', phone: '+49 89 2468 1368', mobile: '+49 170 890 1234',
    role: 'member', jobTitle: 'Projektleiterin', department: 'Projektmanagement', departmentId: 'pm',
    contractType: 'full_time', workload: 100, joinDate: '2022-09-01', birthday: '1986-07-22',
    location: 'München', salary: 6500,
    skills: ['Scrum', 'Jira', 'Confluence', 'Stakeholder Management'],
    status: 'away', currentTask: 'Sprint Review vorbereiten',
  },
]

// ---------------------------------------------------------------------------
// External Contacts (20 — customers, partners, prospects)
// ---------------------------------------------------------------------------

export interface MockExternalContact {
  id: string
  salutation: 'Herr' | 'Frau'
  title: string
  firstName: string
  lastName: string
  initials: string
  email: string
  phone: string
  mobile: string
  company: string
  jobTitle: string
  category: 'customer' | 'partner' | 'prospect'
  status: 'active' | 'prospect' | 'inactive'
  city: string
  tags: string[]
  notes: string
  lastContact: string
}

export const EXTERNAL_CONTACTS: MockExternalContact[] = [
  // --- Kunden (10) ---
  {
    id: 'x1', salutation: 'Herr', title: 'Dr.', firstName: 'Hans', lastName: 'Gruber', initials: 'HG',
    email: 'gruber@gruber-maschinenbau.de', phone: '+49 711 123 4567', mobile: '+49 170 111 2233',
    company: 'Gruber Maschinenbau GmbH', jobTitle: 'Geschäftsführer',
    category: 'customer', status: 'active', city: 'Stuttgart',
    tags: ['VIP', 'Maschinenbau', 'Entscheider'], notes: 'Hauptkunde seit 2022. Website-Relaunch + Wartungsvertrag.',
    lastContact: '2026-02-20',
  },
  {
    id: 'x2', salutation: 'Frau', title: '', firstName: 'Claudia', lastName: 'Hartwig', initials: 'CH',
    email: 'hartwig@rn-logistik.de', phone: '+49 621 234 5678', mobile: '+49 171 222 3344',
    company: 'Rhein-Neckar Logistik AG', jobTitle: 'IT-Leiterin',
    category: 'customer', status: 'active', city: 'Mannheim',
    tags: ['Logistik', 'CRM-Migration'], notes: 'CRM-Migration von Salesforce. Go-Live geplant April 2026.',
    lastContact: '2026-02-18',
  },
  {
    id: 'x3', salutation: 'Herr', title: '', firstName: 'Michael', lastName: 'Berger', initials: 'MB',
    email: 'berger@berger-soehne.de', phone: '+49 911 345 6789', mobile: '+49 172 333 4455',
    company: 'Berger & Söhne Metallverarbeitung', jobTitle: 'CEO',
    category: 'customer', status: 'active', city: 'Nürnberg',
    tags: ['Produktion', 'Mobile App'], notes: 'Mobile App für Werkstatt-Terminals. Wichtiger Referenzkunde.',
    lastContact: '2026-02-15',
  },
  {
    id: 'x4', salutation: 'Frau', title: '', firstName: 'Katrin', lastName: 'Weiss', initials: 'KW',
    email: 'weiss@weiss-consulting.de', phone: '+49 69 456 7890', mobile: '+49 173 444 5566',
    company: 'Weiss Consulting GmbH', jobTitle: 'Geschäftsführerin',
    category: 'customer', status: 'active', city: 'Frankfurt am Main',
    tags: ['Beratung', 'KMU'], notes: 'Nutzt Cosmi für internes Projektmanagement. Sehr zufrieden.',
    lastContact: '2026-02-12',
  },
  {
    id: 'x5', salutation: 'Herr', title: '', firstName: 'Juergen', lastName: 'Becker', initials: 'JB',
    email: 'becker@alpen-tourismus.de', phone: '+49 8821 567 890', mobile: '+49 174 555 6677',
    company: 'Alpen Tourismus AG', jobTitle: 'COO',
    category: 'customer', status: 'active', city: 'Garmisch-Partenkirchen',
    tags: ['Tourismus', 'Buchungssystem'], notes: 'Buchungssystem-Integration. Saisongeschaeft beachten.',
    lastContact: '2026-02-10',
  },
  {
    id: 'x6', salutation: 'Frau', title: '', firstName: 'Monika', lastName: 'Lehmann', initials: 'ML',
    email: 'lehmann@dataflow.de', phone: '+49 30 678 9012', mobile: '+49 175 666 7788',
    company: 'DataFlow GmbH', jobTitle: 'CTO',
    category: 'customer', status: 'active', city: 'Berlin',
    tags: ['SaaS', 'Analytics'], notes: 'Data Analytics Dashboard Projekt. Technisch sehr versiert.',
    lastContact: '2026-02-19',
  },
  {
    id: 'x7', salutation: 'Herr', title: '', firstName: 'Robert', lastName: 'Stadler', initials: 'RS',
    email: 'stadler@stadler-bau.de', phone: '+49 821 789 0123', mobile: '+49 176 777 8899',
    company: 'Stadler Bauunternehmen GmbH', jobTitle: 'Geschäftsführer',
    category: 'customer', status: 'active', city: 'Augsburg',
    tags: ['Bau', 'Intranet'], notes: 'Intranet-Portal für 120 Mitarbeiter. Budget bestätigt.',
    lastContact: '2026-02-14',
  },
  {
    id: 'x8', salutation: 'Frau', title: '', firstName: 'Ingrid', lastName: 'Schwarz', initials: 'IS',
    email: 'schwarz@schwarz-handel.de', phone: '+49 40 890 1234', mobile: '+49 177 888 9900',
    company: 'Schwarz Einzelhandel AG', jobTitle: 'Einkaufsleiterin',
    category: 'customer', status: 'active', city: 'Hamburg',
    tags: ['Einzelhandel', 'E-Commerce'], notes: 'E-Commerce-Plattform. Multi-Channel-Anbindung gewünscht.',
    lastContact: '2026-02-08',
  },
  {
    id: 'x9', salutation: 'Herr', title: '', firstName: 'Frank', lastName: 'Neubauer', initials: 'FN',
    email: 'neubauer@biotech-solutions.de', phone: '+49 761 901 2345', mobile: '+49 178 999 0011',
    company: 'BioTech Solutions GmbH', jobTitle: 'IT-Manager',
    category: 'customer', status: 'active', city: 'Freiburg',
    tags: ['BioTech', 'Security'], notes: 'Security Audit + DSGVO-Beratung. Sehr datenschutzbewusst.',
    lastContact: '2026-02-17',
  },
  {
    id: 'x10', salutation: 'Frau', title: '', firstName: 'Susanne', lastName: 'Kramer', initials: 'SK',
    email: 'kramer@kramer-automotive.de', phone: '+49 841 012 3456', mobile: '+49 179 000 1122',
    company: 'Kramer Automotive GmbH', jobTitle: 'Personalchefin',
    category: 'customer', status: 'active', city: 'Ingolstadt',
    tags: ['Automotive', 'HR-Modul'], notes: 'HR-Modul Pilotkundin. 85 Mitarbeiter. Feedback sehr wertvoll.',
    lastContact: '2026-02-06',
  },
  // --- Partner (5) ---
  {
    id: 'x11', salutation: 'Herr', title: 'Prof. Dr.', firstName: 'Werner', lastName: 'Braun', initials: 'WB',
    email: 'braun@braun-kollegen.de', phone: '+49 89 111 2233', mobile: '+49 170 112 2334',
    company: 'Braun & Kollegen Rechtsanwaelte', jobTitle: 'IT-Rechtsanwalt',
    category: 'partner', status: 'active', city: 'München',
    tags: ['Recht', 'DSGVO', 'Verträge'], notes: 'Externe Rechtsberatung. DSGVO + AGB + Arbeitsrecht.',
    lastContact: '2026-02-11',
  },
  {
    id: 'x12', salutation: 'Frau', title: '', firstName: 'Anke', lastName: 'Dietrich', initials: 'AD',
    email: 'dietrich@cloudfirst.de', phone: '+49 69 222 3344', mobile: '+49 171 223 3445',
    company: 'CloudFirst Hosting GmbH', jobTitle: 'Account Managerin',
    category: 'partner', status: 'active', city: 'Frankfurt am Main',
    tags: ['Hosting', 'SLA', 'Infrastruktur'], notes: 'Hosting-Partner. Hetzner-basiert. SLA 99.9%. Vertrag bis 2028.',
    lastContact: '2026-02-13',
  },
  {
    id: 'x13', salutation: 'Herr', title: '', firstName: 'Dirk', lastName: 'Hoffmann', initials: 'DH',
    email: 'hoffmann@hoffmann-it.de', phone: '+49 911 333 4455', mobile: '+49 172 334 4556',
    company: 'Hoffmann IT — DATEV-Partner', jobTitle: 'Senior Consultant',
    category: 'partner', status: 'active', city: 'Nürnberg',
    tags: ['DATEV', 'Buchhaltung', 'Schnittstellen'], notes: 'DATEV-Integrationspartner. Hilft bei Export/Import-Schnittstellen.',
    lastContact: '2026-02-07',
  },
  {
    id: 'x14', salutation: 'Frau', title: '', firstName: 'Carla', lastName: 'Ruiz', initials: 'CR',
    email: 'ruiz@linguaplus.de', phone: '+49 89 444 5566', mobile: '+49 173 445 5667',
    company: 'LinguaPlus GmbH', jobTitle: 'Übersetzerin DE/ES/EN',
    category: 'partner', status: 'active', city: 'München',
    tags: ['Übersetzung', 'Lokalisierung'], notes: 'Übersetzt Marketingmaterial und Docs. Schnell und zuverlässig.',
    lastContact: '2026-01-28',
  },
  {
    id: 'x15', salutation: 'Herr', title: '', firstName: 'Maximilian', lastName: 'Reuter', initials: 'MR',
    email: 'reuter@reuter-design.de', phone: '+49 30 555 6677', mobile: '+49 174 556 6778',
    company: 'Reuter Design Studio', jobTitle: 'Grafikdesigner',
    category: 'partner', status: 'active', city: 'Berlin',
    tags: ['Grafik', 'Print', 'Branding'], notes: 'Externer Grafikdesigner für Print + Messematerial.',
    lastContact: '2026-02-03',
  },
  // --- Prospects (5) ---
  {
    id: 'x16', salutation: 'Frau', title: 'Dr.', firstName: 'Astrid', lastName: 'Vogel', initials: 'AV',
    email: 'vogel@medtech-innovations.de', phone: '+49 7071 666 7788', mobile: '+49 175 667 7889',
    company: 'MedTech Innovations GmbH', jobTitle: 'Geschäftsführerin',
    category: 'prospect', status: 'prospect', city: 'Tuebingen',
    tags: ['MedTech', 'Prospect', 'Demo'], notes: 'Erstkontakt Messe DMEA. Demo am 28.02. geplant. 45 MA.',
    lastContact: '2026-02-16',
  },
  {
    id: 'x17', salutation: 'Herr', title: '', firstName: 'Henrik', lastName: 'Svensson', initials: 'HS',
    email: 'svensson@nordsoft.de', phone: '+49 40 777 8899', mobile: '+49 176 778 8990',
    company: 'NordSoft GmbH', jobTitle: 'VP Sales DACH',
    category: 'prospect', status: 'prospect', city: 'Hamburg',
    tags: ['Software', 'Prospect', 'DACH'], notes: 'Schwedisches Unternehmen, DACH-Expansion. Interesse an CRM-Modul.',
    lastContact: '2026-02-09',
  },
  {
    id: 'x18', salutation: 'Frau', title: '', firstName: 'Barbara', lastName: 'Engel', initials: 'BE',
    email: 'engel@engel-stb.de', phone: '+49 89 888 9900', mobile: '+49 177 889 9001',
    company: 'Engel Steuerberatung', jobTitle: 'Inhaberin',
    category: 'prospect', status: 'prospect', city: 'München',
    tags: ['Steuerberatung', 'Prospect'], notes: 'Empfehlung von Hoffmann IT. 12 Mitarbeiter. Bucht evtl. Buchhaltungsmodul.',
    lastContact: '2026-02-04',
  },
  {
    id: 'x19', salutation: 'Herr', title: '', firstName: 'Lukas', lastName: 'Pfeifer', initials: 'LP',
    email: 'pfeifer@greentech-sol.de', phone: '+49 6151 999 0011', mobile: '+49 178 990 0112',
    company: 'GreenTech Solutions UG', jobTitle: 'Gründer',
    category: 'prospect', status: 'prospect', city: 'Darmstadt',
    tags: ['Startup', 'GreenTech', 'Prospect'], notes: 'Startup, 8 MA. Sucht günstige All-in-One Lösung. Preissensibel.',
    lastContact: '2026-02-01',
  },
  {
    id: 'x20', salutation: 'Frau', title: '', firstName: 'Christine', lastName: 'Wolf', initials: 'CW',
    email: 'wolf@bvg-medien.de', phone: '+49 89 000 1122', mobile: '+49 179 001 1223',
    company: 'Bayerische Verlagsgruppe AG', jobTitle: 'Marketing-Leiterin',
    category: 'prospect', status: 'prospect', city: 'München',
    tags: ['Verlag', 'Medien', 'Prospect'], notes: 'Kontakt über LinkedIn. Interesse an Marketing-Automatisierung.',
    lastContact: '2026-01-25',
  },
]

// ---------------------------------------------------------------------------
// Projects (8) — linked to employees + external contacts
// ---------------------------------------------------------------------------

export interface MockProject {
  id: string
  name: string
  clientId: string
  clientName: string
  status: 'aktiv' | 'geplant' | 'abgeschlossen' | 'pausiert'
  startDate: string
  endDate: string
  budget: number
  progress: number
  managerId: string
  teamMemberIds: string[]
  description: string
}

export const PROJECTS: MockProject[] = [
  {
    id: 'p1', name: 'Website Relaunch', clientId: 'x1', clientName: 'Gruber Maschinenbau GmbH',
    status: 'aktiv', startDate: '2025-11-01', endDate: '2026-04-30', budget: 48000, progress: 65,
    managerId: 'e18', teamMemberIds: ['e4', 'e6', 'e9', 'e8'],
    description: 'Kompletter Relaunch der Unternehmenswebsite inkl. Produktkonfigurator.',
  },
  {
    id: 'p2', name: 'CRM Migration', clientId: 'x2', clientName: 'Rhein-Neckar Logistik AG',
    status: 'aktiv', startDate: '2025-12-01', endDate: '2026-04-15', budget: 35000, progress: 40,
    managerId: 'e18', teamMemberIds: ['e2', 'e5', 'e4'],
    description: 'Migration von Salesforce zu Cosmi CRM. Datentransfer + Schulung.',
  },
  {
    id: 'p3', name: 'Mobile App', clientId: 'x3', clientName: 'Berger & Söhne Metallverarbeitung',
    status: 'aktiv', startDate: '2026-01-15', endDate: '2026-06-30', budget: 62000, progress: 20,
    managerId: 'e18', teamMemberIds: ['e6', 'e4', 'e9'],
    description: 'Native App für Werkstatt-Terminals mit Barcode-Scanner und Auftragsverwaltung.',
  },
  {
    id: 'p4', name: 'E-Commerce Plattform', clientId: 'x8', clientName: 'Schwarz Einzelhandel AG',
    status: 'geplant', startDate: '2026-03-01', endDate: '2026-09-30', budget: 85000, progress: 0,
    managerId: 'e18', teamMemberIds: ['e2', 'e5', 'e6', 'e4'],
    description: 'Multi-Channel E-Commerce mit ERP-Anbindung. Shopify + eigenes Backend.',
  },
  {
    id: 'p5', name: 'Data Analytics Dashboard', clientId: 'x6', clientName: 'DataFlow GmbH',
    status: 'aktiv', startDate: '2026-01-01', endDate: '2026-03-31', budget: 28000, progress: 70,
    managerId: 'e18', teamMemberIds: ['e5', 'e4', 'e7'],
    description: 'Interaktives Dashboard für Echtzeit-Datenvisualisierung mit Drill-Down.',
  },
  {
    id: 'p6', name: 'Intranet Portal', clientId: 'x7', clientName: 'Stadler Bauunternehmen GmbH',
    status: 'aktiv', startDate: '2025-10-01', endDate: '2026-03-15', budget: 42000, progress: 85,
    managerId: 'e18', teamMemberIds: ['e6', 'e8', 'e7'],
    description: 'Mitarbeiterportal mit News, Dokumenten-Management und Urlaubsantraegen.',
  },
  {
    id: 'p7', name: 'Marketing Automation', clientId: 'e1', clientName: 'TechVision GmbH (intern)',
    status: 'aktiv', startDate: '2026-02-01', endDate: '2026-05-31', budget: 15000, progress: 15,
    managerId: 'e12', teamMemberIds: ['e12', 'e9', 'e8'],
    description: 'Internes Projekt: Newsletter-Automatisierung, Lead-Scoring, Landing Pages.',
  },
  {
    id: 'p8', name: 'Security Audit', clientId: 'x9', clientName: 'BioTech Solutions GmbH',
    status: 'aktiv', startDate: '2026-02-10', endDate: '2026-03-10', budget: 18000, progress: 35,
    managerId: 'e2', teamMemberIds: ['e2', 'e5'],
    description: 'Umfassender Security Audit: Penetrationstest, DSGVO-Check, Infrastruktur-Review.',
  },
]

// ---------------------------------------------------------------------------
// Deals (22) — linked to external contacts
// ---------------------------------------------------------------------------

export type DealStage = 'lead' | 'qualifiziert' | 'angebot' | 'verhandlung' | 'gewonnen' | 'verloren'

export interface MockDeal {
  id: string
  name: string
  contactId: string
  contactName: string
  company: string
  stage: DealStage
  value: number
  probability: number
  expectedClose: string
  ownerId: string
  ownerName: string
  createdAt: string
  notes: string
}

export const DEALS: MockDeal[] = [
  // Gewonnene Deals (project-linked)
  { id: 'd1', name: 'Website Relaunch Gruber', contactId: 'x1', contactName: 'Dr. Hans Gruber', company: 'Gruber Maschinenbau GmbH', stage: 'gewonnen', value: 48000, probability: 100, expectedClose: '2025-10-15', ownerId: 'e3', ownerName: 'Thomas Meier', createdAt: '2025-08-01', notes: 'Vertrag unterschrieben. Projekt läuft.' },
  { id: 'd2', name: 'CRM Migration RN Logistik', contactId: 'x2', contactName: 'Claudia Hartwig', company: 'Rhein-Neckar Logistik AG', stage: 'gewonnen', value: 35000, probability: 100, expectedClose: '2025-11-20', ownerId: 'e3', ownerName: 'Thomas Meier', createdAt: '2025-09-10', notes: 'Vertrag + Wartung 12 Monate.' },
  { id: 'd3', name: 'Mobile App Berger', contactId: 'x3', contactName: 'Michael Berger', company: 'Berger & Söhne', stage: 'gewonnen', value: 62000, probability: 100, expectedClose: '2026-01-10', ownerId: 'e10', ownerName: 'Sabine Fischer', createdAt: '2025-10-15', notes: 'Großprojekt. Phase 1 ab Januar.' },
  { id: 'd4', name: 'Intranet Stadler', contactId: 'x7', contactName: 'Robert Stadler', company: 'Stadler Bauunternehmen GmbH', stage: 'gewonnen', value: 42000, probability: 100, expectedClose: '2025-09-15', ownerId: 'e3', ownerName: 'Thomas Meier', createdAt: '2025-07-01', notes: 'Langzeit-Kunde. Weitere Projekte möglich.' },
  { id: 'd5', name: 'Analytics Dashboard DataFlow', contactId: 'x6', contactName: 'Monika Lehmann', company: 'DataFlow GmbH', stage: 'gewonnen', value: 28000, probability: 100, expectedClose: '2025-12-20', ownerId: 'e10', ownerName: 'Sabine Fischer', createdAt: '2025-11-01', notes: 'Technisch anspruchsvoll aber spannend.' },
  { id: 'd6', name: 'Security Audit BioTech', contactId: 'x9', contactName: 'Frank Neubauer', company: 'BioTech Solutions GmbH', stage: 'gewonnen', value: 18000, probability: 100, expectedClose: '2026-02-05', ownerId: 'e3', ownerName: 'Thomas Meier', createdAt: '2026-01-15', notes: 'Dringend. DSGVO-Audit vor Zertifizierung.' },
  // Aktive Pipeline
  { id: 'd7', name: 'E-Commerce Schwarz AG', contactId: 'x8', contactName: 'Ingrid Schwarz', company: 'Schwarz Einzelhandel AG', stage: 'verhandlung', value: 85000, probability: 75, expectedClose: '2026-02-28', ownerId: 'e3', ownerName: 'Thomas Meier', createdAt: '2025-12-01', notes: 'Vertragsverhandlung läuft. Budget bestätigt.' },
  { id: 'd8', name: 'HR-Modul Kramer', contactId: 'x10', contactName: 'Susanne Kramer', company: 'Kramer Automotive GmbH', stage: 'angebot', value: 32000, probability: 60, expectedClose: '2026-03-15', ownerId: 'e10', ownerName: 'Sabine Fischer', createdAt: '2026-01-20', notes: 'Angebot versendet. Warten auf Feedback.' },
  { id: 'd9', name: 'Consulting Paket Weiss', contactId: 'x4', contactName: 'Katrin Weiss', company: 'Weiss Consulting GmbH', stage: 'angebot', value: 15000, probability: 50, expectedClose: '2026-03-31', ownerId: 'e11', ownerName: 'Kevin Baumann', createdAt: '2026-02-01', notes: 'Erweiterung um Schulungspaket.' },
  { id: 'd10', name: 'Buchungssystem Alpen', contactId: 'x5', contactName: 'Jürgen Becker', company: 'Alpen Tourismus AG', stage: 'qualifiziert', value: 55000, probability: 40, expectedClose: '2026-05-30', ownerId: 'e3', ownerName: 'Thomas Meier', createdAt: '2026-01-10', notes: 'Anforderungsanalyse geplant für März.' },
  { id: 'd11', name: 'Cosmi Lizenz MedTech', contactId: 'x16', contactName: 'Dr. Astrid Vogel', company: 'MedTech Innovations GmbH', stage: 'qualifiziert', value: 24000, probability: 35, expectedClose: '2026-04-30', ownerId: 'e11', ownerName: 'Kevin Baumann', createdAt: '2026-02-10', notes: 'Demo am 28.02. Entscheidung im März.' },
  { id: 'd12', name: 'CRM Lizenz NordSoft', contactId: 'x17', contactName: 'Henrik Svensson', company: 'NordSoft GmbH', stage: 'lead', value: 18000, probability: 20, expectedClose: '2026-06-30', ownerId: 'e11', ownerName: 'Kevin Baumann', createdAt: '2026-02-09', notes: 'Erstkontakt. Brauchen DACH-spezifisches CRM.' },
  { id: 'd13', name: 'Buchhaltung Engel STB', contactId: 'x18', contactName: 'Barbara Engel', company: 'Engel Steuerberatung', stage: 'lead', value: 8000, probability: 25, expectedClose: '2026-04-15', ownerId: 'e10', ownerName: 'Sabine Fischer', createdAt: '2026-02-04', notes: 'Klein aber gute Referenz für Steuerberater-Branche.' },
  { id: 'd14', name: 'Starter-Paket GreenTech', contactId: 'x19', contactName: 'Lukas Pfeifer', company: 'GreenTech Solutions UG', stage: 'lead', value: 6000, probability: 15, expectedClose: '2026-05-31', ownerId: 'e11', ownerName: 'Kevin Baumann', createdAt: '2026-02-01', notes: 'Startup-Rabatt möglich. Preisverhandlung.' },
  { id: 'd15', name: 'Marketing Suite BVG', contactId: 'x20', contactName: 'Christine Wolf', company: 'Bayerische Verlagsgruppe AG', stage: 'lead', value: 45000, probability: 10, expectedClose: '2026-07-31', ownerId: 'e10', ownerName: 'Sabine Fischer', createdAt: '2026-01-25', notes: 'Sehr fruehe Phase. Erst mal Bedarfsanalyse.' },
  // Verloren
  { id: 'd16', name: 'IT-Infrastruktur Maier AG', contactId: 'x1', contactName: 'Dr. Hans Gruber', company: 'Maier Elektrotechnik AG', stage: 'verloren', value: 38000, probability: 0, expectedClose: '2025-12-15', ownerId: 'e3', ownerName: 'Thomas Meier', createdAt: '2025-09-01', notes: 'Verloren an Wettbewerber. Preis war entscheidend.' },
  // Extra pipeline deals
  { id: 'd17', name: 'App Erweiterung Gruber Ph.2', contactId: 'x1', contactName: 'Dr. Hans Gruber', company: 'Gruber Maschinenbau GmbH', stage: 'qualifiziert', value: 25000, probability: 60, expectedClose: '2026-06-30', ownerId: 'e3', ownerName: 'Thomas Meier', createdAt: '2026-02-15', notes: 'Phase 2 Website: Kundenportal + Konfigurator.' },
  { id: 'd18', name: 'Wartungsvertrag RN Logistik', contactId: 'x2', contactName: 'Claudia Hartwig', company: 'Rhein-Neckar Logistik AG', stage: 'verhandlung', value: 12000, probability: 80, expectedClose: '2026-04-01', ownerId: 'e3', ownerName: 'Thomas Meier', createdAt: '2026-02-10', notes: 'Verlaengerung Wartungsvertrag. Fast sicher.' },
  { id: 'd19', name: 'Schulungspaket Berger', contactId: 'x3', contactName: 'Michael Berger', company: 'Berger & Söhne', stage: 'angebot', value: 8500, probability: 70, expectedClose: '2026-03-15', ownerId: 'e10', ownerName: 'Sabine Fischer', createdAt: '2026-02-12', notes: 'Schulung für 20 Werkstatt-Mitarbeiter.' },
  { id: 'd20', name: 'Zusatzmodul Weiss Consulting', contactId: 'x4', contactName: 'Katrin Weiss', company: 'Weiss Consulting GmbH', stage: 'angebot', value: 9500, probability: 55, expectedClose: '2026-04-15', ownerId: 'e11', ownerName: 'Kevin Baumann', createdAt: '2026-02-05', notes: 'Helpdesk + Chat-Modul Erweiterung.' },
  { id: 'd21', name: 'DSGVO Audit Kramer', contactId: 'x10', contactName: 'Susanne Kramer', company: 'Kramer Automotive GmbH', stage: 'qualifiziert', value: 14000, probability: 45, expectedClose: '2026-05-15', ownerId: 'e10', ownerName: 'Sabine Fischer', createdAt: '2026-02-08', notes: 'Kombination mit HR-Modul möglich.' },
  { id: 'd22', name: 'Analytics Erweiterung DataFlow', contactId: 'x6', contactName: 'Monika Lehmann', company: 'DataFlow GmbH', stage: 'verhandlung', value: 19000, probability: 70, expectedClose: '2026-04-30', ownerId: 'e10', ownerName: 'Sabine Fischer', createdAt: '2026-02-18', notes: 'Phase 2: Predictive Analytics + ML-Pipeline.' },
]

// ---------------------------------------------------------------------------
// Tasks — linked to projects + employees
// ---------------------------------------------------------------------------

export type TaskPriority = 'high' | 'normal' | 'low'
export type TaskStatus = 'todo' | 'in_progress' | 'done' | 'blocked'

export interface MockTask {
  id: string
  title: string
  projectId: string
  projectName: string
  assigneeId: string
  assigneeName: string
  priority: TaskPriority
  status: TaskStatus
  deadline: string
  description?: string
}

export const TASKS: MockTask[] = [
  // p1 — Website Relaunch
  { id: 't1', title: 'Hero-Sektion responsive machen', projectId: 'p1', projectName: 'Website Relaunch', assigneeId: 'e6', assigneeName: 'Tim Hartmann', priority: 'high', status: 'in_progress', deadline: '2026-02-25' },
  { id: 't2', title: 'Produktkonfigurator Prototyp', projectId: 'p1', projectName: 'Website Relaunch', assigneeId: 'e4', assigneeName: 'Laura Neumann', priority: 'high', status: 'in_progress', deadline: '2026-03-01' },
  { id: 't3', title: 'SEO Meta-Tags einpflegen', projectId: 'p1', projectName: 'Website Relaunch', assigneeId: 'e6', assigneeName: 'Tim Hartmann', priority: 'normal', status: 'todo', deadline: '2026-03-10' },
  { id: 't4', title: 'Design Review Startseite', projectId: 'p1', projectName: 'Website Relaunch', assigneeId: 'e9', assigneeName: 'Nina Richter', priority: 'normal', status: 'done', deadline: '2026-02-15' },
  // p2 — CRM Migration
  { id: 't5', title: 'Salesforce-Datenexport validieren', projectId: 'p2', projectName: 'CRM Migration', assigneeId: 'e5', assigneeName: 'Felix Krause', priority: 'high', status: 'done', deadline: '2026-02-10' },
  { id: 't6', title: 'Kontakt-Import Mapping erstellen', projectId: 'p2', projectName: 'CRM Migration', assigneeId: 'e4', assigneeName: 'Laura Neumann', priority: 'high', status: 'in_progress', deadline: '2026-02-26' },
  { id: 't7', title: 'Testmigration durchführen', projectId: 'p2', projectName: 'CRM Migration', assigneeId: 'e2', assigneeName: 'Markus Weber', priority: 'normal', status: 'todo', deadline: '2026-03-05' },
  // p3 — Mobile App
  { id: 't8', title: 'Barcode-Scanner Integration', projectId: 'p3', projectName: 'Mobile App', assigneeId: 'e6', assigneeName: 'Tim Hartmann', priority: 'high', status: 'in_progress', deadline: '2026-03-01' },
  { id: 't9', title: 'Offline-Modus implementieren', projectId: 'p3', projectName: 'Mobile App', assigneeId: 'e4', assigneeName: 'Laura Neumann', priority: 'normal', status: 'todo', deadline: '2026-03-15' },
  { id: 't10', title: 'UI Mockups Auftragsansicht', projectId: 'p3', projectName: 'Mobile App', assigneeId: 'e9', assigneeName: 'Nina Richter', priority: 'normal', status: 'in_progress', deadline: '2026-02-28' },
  // p5 — Data Analytics
  { id: 't11', title: 'Chart-Bibliothek evaluieren', projectId: 'p5', projectName: 'Data Analytics Dashboard', assigneeId: 'e5', assigneeName: 'Felix Krause', priority: 'low', status: 'done', deadline: '2026-02-01' },
  { id: 't12', title: 'Echtzeit-Daten Pipeline', projectId: 'p5', projectName: 'Data Analytics Dashboard', assigneeId: 'e5', assigneeName: 'Felix Krause', priority: 'high', status: 'in_progress', deadline: '2026-02-28' },
  { id: 't13', title: 'Drill-Down Navigation bauen', projectId: 'p5', projectName: 'Data Analytics Dashboard', assigneeId: 'e7', assigneeName: 'Lena Braun', priority: 'normal', status: 'todo', deadline: '2026-03-10' },
  // p6 — Intranet
  { id: 't14', title: 'Dokumenten-Upload finalisieren', projectId: 'p6', projectName: 'Intranet Portal', assigneeId: 'e6', assigneeName: 'Tim Hartmann', priority: 'high', status: 'done', deadline: '2026-02-10' },
  { id: 't15', title: 'Urlaubsantrag-Workflow testen', projectId: 'p6', projectName: 'Intranet Portal', assigneeId: 'e8', assigneeName: 'Sophie Lang', priority: 'normal', status: 'in_progress', deadline: '2026-02-24' },
  { id: 't16', title: 'Go-Live Checklist erstellen', projectId: 'p6', projectName: 'Intranet Portal', assigneeId: 'e7', assigneeName: 'Lena Braun', priority: 'low', status: 'todo', deadline: '2026-03-01' },
  // p7 — Marketing Automation (intern)
  { id: 't17', title: 'Newsletter-Template designen', projectId: 'p7', projectName: 'Marketing Automation', assigneeId: 'e9', assigneeName: 'Nina Richter', priority: 'normal', status: 'in_progress', deadline: '2026-02-27' },
  { id: 't18', title: 'Lead-Scoring Regeln definieren', projectId: 'p7', projectName: 'Marketing Automation', assigneeId: 'e12', assigneeName: 'Julia Hofmann', priority: 'high', status: 'todo', deadline: '2026-03-05' },
  // p8 — Security Audit
  { id: 't19', title: 'Penetrationstest durchführen', projectId: 'p8', projectName: 'Security Audit', assigneeId: 'e2', assigneeName: 'Markus Weber', priority: 'high', status: 'in_progress', deadline: '2026-02-28' },
  { id: 't20', title: 'DSGVO-Checkliste abarbeiten', projectId: 'p8', projectName: 'Security Audit', assigneeId: 'e5', assigneeName: 'Felix Krause', priority: 'high', status: 'todo', deadline: '2026-03-05' },
  // Nicht-Projekt (interne Aufgaben)
  { id: 't21', title: 'Monatsabschluss Januar', projectId: '', projectName: 'Intern', assigneeId: 'e13', assigneeName: 'Petra Zimmermann', priority: 'high', status: 'in_progress', deadline: '2026-02-25' },
  { id: 't22', title: 'Bewerbungsgespräche KW 9', projectId: '', projectName: 'Intern', assigneeId: 'e15', assigneeName: 'Elena Schuster', priority: 'normal', status: 'todo', deadline: '2026-02-28' },
  { id: 't23', title: 'Team-Event März organisieren', projectId: '', projectName: 'Intern', assigneeId: 'e14', assigneeName: 'Andrea Keller', priority: 'low', status: 'in_progress', deadline: '2026-03-07' },
  { id: 't24', title: 'FAQ-Artikel aktualisieren', projectId: '', projectName: 'Intern', assigneeId: 'e17', assigneeName: 'Jonas Schmitt', priority: 'normal', status: 'in_progress', deadline: '2026-02-24' },
]

// ---------------------------------------------------------------------------
// Calendar Events — for today (2026-02-24)
// ---------------------------------------------------------------------------

export interface MockCalendarEvent {
  id: string
  title: string
  time: string
  duration: string
  location?: string
  type: 'meeting' | 'focus' | 'call' | 'break' | 'workshop'
  attendeeIds: string[]
  color: string
}

export const TODAY_EVENTS: MockCalendarEvent[] = [
  { id: 'ev1', title: 'Daily Standup', time: '09:00', duration: '15 Min', location: 'Raum Isar', type: 'meeting', attendeeIds: ['e2', 'e4', 'e5', 'e6', 'e7'], color: '#3b82f6' },
  { id: 'ev2', title: 'Sprint Review', time: '10:00', duration: '1 Std', location: 'Raum Alpen', type: 'meeting', attendeeIds: ['e1', 'e2', 'e18', 'e4', 'e5', 'e6'], color: '#6366f1' },
  { id: 'ev3', title: 'Design Review Mobile App', time: '11:30', duration: '45 Min', location: 'Raum Isar', type: 'meeting', attendeeIds: ['e9', 'e8', 'e6'], color: '#ec4899' },
  { id: 'ev4', title: 'Mittagspause', time: '12:30', duration: '1 Std', type: 'break', attendeeIds: [], color: '#94a3b8' },
  { id: 'ev5', title: 'Fokuszeit — Deep Work', time: '13:30', duration: '2 Std', type: 'focus', attendeeIds: [], color: '#10b981' },
  { id: 'ev6', title: 'Call: Claudia Hartwig (RN Logistik)', time: '15:30', duration: '30 Min', type: 'call', attendeeIds: ['e3', 'e10'], color: '#f59e0b' },
  { id: 'ev7', title: 'Team Retro', time: '16:00', duration: '1 Std', location: 'Raum Alpen', type: 'workshop', attendeeIds: ['e2', 'e4', 'e5', 'e6', 'e7', 'e18'], color: '#14b8a6' },
  { id: 'ev8', title: '1:1 mit Stefan (Quartalsplanung)', time: '17:00', duration: '30 Min', location: 'Büro GF', type: 'meeting', attendeeIds: ['e1', 'e2'], color: '#6366f1' },
]

// ---------------------------------------------------------------------------
// Chat Messages — for team-chat widget
// ---------------------------------------------------------------------------

export interface MockChatMessage {
  id: string
  channel: string
  senderId: string
  senderName: string
  senderInitials: string
  text: string
  time: string
  unread: boolean
}

export const CHAT_MESSAGES: MockChatMessage[] = [
  { id: 'msg1', channel: 'allgemein', senderId: 'e18', senderName: 'Sarah Beck', senderInitials: 'SB', text: 'Reminder: Sprint Review heute um 10 Uhr in Raum Alpen!', time: '08:45', unread: true },
  { id: 'msg2', channel: 'entwicklung', senderId: 'e2', senderName: 'Markus Weber', senderInitials: 'MW', text: 'PR #312 ist ready for review — API v2 Endpunkte. Wer kann drüberschauen?', time: '08:32', unread: true },
  { id: 'msg3', channel: 'allgemein', senderId: 'e9', senderName: 'Nina Richter', senderInitials: 'NR', text: 'Neue Mockups für die Mobile App sind im Figma. Link im Thread.', time: '08:15', unread: false },
  { id: 'msg4', channel: 'vertrieb', senderId: 'e3', senderName: 'Thomas Meier', senderInitials: 'TM', text: 'Schwarz Einzelhandel hat den E-Commerce-Vertrag unterschrieben! 🎉', time: 'Gestern', unread: false },
  { id: 'msg5', channel: 'random', senderId: 'e14', senderName: 'Andrea Keller', senderInitials: 'AK', text: 'Wer hat Lust auf Mittagessen beim Italiener um die Ecke?', time: 'Gestern', unread: false },
  { id: 'msg6', channel: 'entwicklung', senderId: 'e5', senderName: 'Felix Krause', senderInitials: 'FK', text: 'Heads up: Redis-Migration auf v7 ist durch. Bitte lokales Setup updaten.', time: 'Gestern', unread: false },
  { id: 'msg7', channel: 'design', senderId: 'e8', senderName: 'Sophie Lang', senderInitials: 'SL', text: 'Usability-Test Ergebnisse sind zusammengefasst. Praesentation morgen.', time: 'Gestern', unread: false },
]

// ---------------------------------------------------------------------------
// Absences — who is out today
// ---------------------------------------------------------------------------

export type AbsenceType = 'urlaub' | 'krank' | 'elternzeit' | 'weiterbildung' | 'homeoffice'

export interface MockAbsence {
  id: string
  employeeId: string
  name: string
  initials: string
  type: AbsenceType
  until: string
  department: string
}

export const ABSENCES_TODAY: MockAbsence[] = [
  { id: 'abs1', employeeId: 'e10', name: 'Sabine Fischer', initials: 'SF', type: 'urlaub', until: '28.02.', department: 'Vertrieb' },
  { id: 'abs2', employeeId: 'e13', name: 'Petra Zimmermann', initials: 'PZ', type: 'krank', until: 'unbekannt', department: 'Buchhaltung' },
  { id: 'abs3', employeeId: 'e7', name: 'Lena Braun', initials: 'LB', type: 'weiterbildung', until: '25.02.', department: 'Entwicklung' },
  { id: 'abs4', employeeId: 'e8', name: 'Sophie Lang', initials: 'SL', type: 'homeoffice', until: '24.02.', department: 'Design' },
]

// ---------------------------------------------------------------------------
// Monthly Revenue (12 months for chart)
// ---------------------------------------------------------------------------

export interface MockMonthlyRevenue {
  month: string
  label: string
  revenue: number
}

export const MONTHLY_REVENUE: MockMonthlyRevenue[] = [
  { month: '2025-03', label: 'Mär', revenue: 52000 },
  { month: '2025-04', label: 'Apr', revenue: 61000 },
  { month: '2025-05', label: 'Mai', revenue: 58000 },
  { month: '2025-06', label: 'Jun', revenue: 67000 },
  { month: '2025-07', label: 'Jul', revenue: 54000 },
  { month: '2025-08', label: 'Aug', revenue: 48000 },
  { month: '2025-09', label: 'Sep', revenue: 72000 },
  { month: '2025-10', label: 'Okt', revenue: 79000 },
  { month: '2025-11', label: 'Nov', revenue: 83000 },
  { month: '2025-12', label: 'Dez', revenue: 71000 },
  { month: '2026-01', label: 'Jan', revenue: 88000 },
  { month: '2026-02', label: 'Feb', revenue: 84750 },
]

// ---------------------------------------------------------------------------
// Birthdays (upcoming from today 2026-02-24)
// ---------------------------------------------------------------------------

export interface MockBirthday {
  employeeId: string
  name: string
  initials: string
  department: string
  birthday: string
  displayDate: string
  daysUntil: number
}

/** Compute upcoming birthdays dynamically from employee data. */
export function getUpcomingBirthdays(): MockBirthday[] {
  const MONTH_NAMES = ['Jan', 'Feb', 'Maer', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez']
  const now = new Date()
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate())

  return EMPLOYEES
    .filter((e) => e.birthday)
    .map((e) => {
      const [, m, d] = e.birthday.split('-').map(Number)
      let nextBirthday = new Date(now.getFullYear(), m - 1, d)
      if (nextBirthday < todayStart) {
        nextBirthday = new Date(now.getFullYear() + 1, m - 1, d)
      }
      const daysUntil = Math.round((nextBirthday.getTime() - todayStart.getTime()) / 86_400_000)
      const displayDate = `${String(d).padStart(2, '0')}. ${MONTH_NAMES[m - 1]}`
      return {
        employeeId: e.id,
        name: `${e.firstName} ${e.lastName}`,
        initials: e.initials,
        department: e.department,
        birthday: e.birthday,
        displayDate,
        daysUntil,
      }
    })
    .sort((a, b) => a.daysUntil - b.daysUntil)
    .slice(0, 5)
}

// ---------------------------------------------------------------------------
// Helpdesk tickets
// ---------------------------------------------------------------------------

export interface MockTicket {
  id: string
  subject: string
  status: 'open' | 'in_progress' | 'waiting' | 'resolved' | 'closed'
  priority: 'low' | 'medium' | 'high' | 'urgent'
  contactId: string
  contactName: string
  company: string
  assigneeId: string
  assigneeName: string
  createdAt: string
  category: string
}

export const HELPDESK_TICKETS: MockTicket[] = [
  { id: 'tk-1', subject: 'Login funktioniert nicht nach Passwort-Reset', status: 'in_progress', priority: 'high', contactId: 'x1', contactName: 'Dr. Hans Gruber', company: 'Gruber Maschinenbau GmbH', assigneeId: 'e16', assigneeName: 'Martin Wolf', createdAt: '2026-02-23', category: 'Authentifizierung' },
  { id: 'tk-2', subject: 'Datenimport bricht bei Sonderzeichen ab', status: 'open', priority: 'high', contactId: 'x2', contactName: 'Claudia Hartwig', company: 'Rhein-Neckar Logistik AG', assigneeId: 'e17', assigneeName: 'Jonas Schmitt', createdAt: '2026-02-22', category: 'Datenimport' },
  { id: 'tk-3', subject: 'Dashboard laedt langsam bei vielen Widgets', status: 'in_progress', priority: 'medium', contactId: 'x7', contactName: 'Robert Stadler', company: 'Stadler Bauunternehmen GmbH', assigneeId: 'e16', assigneeName: 'Martin Wolf', createdAt: '2026-02-21', category: 'Performance' },
  { id: 'tk-4', subject: 'E-Mail-Benachrichtigungen kommen doppelt', status: 'waiting', priority: 'medium', contactId: 'x4', contactName: 'Katrin Weiss', company: 'Weiss Consulting GmbH', assigneeId: 'e17', assigneeName: 'Jonas Schmitt', createdAt: '2026-02-20', category: 'Benachrichtigungen' },
  { id: 'tk-5', subject: 'PDF-Export zeigt falsches Datum', status: 'resolved', priority: 'low', contactId: 'x5', contactName: 'Jürgen Becker', company: 'Alpen Tourismus AG', assigneeId: 'e16', assigneeName: 'Martin Wolf', createdAt: '2026-02-18', category: 'Export' },
  { id: 'tk-6', subject: 'API Timeout bei grossen Datenmengen', status: 'open', priority: 'urgent', contactId: 'x6', contactName: 'Monika Lehmann', company: 'DataFlow GmbH', assigneeId: 'e16', assigneeName: 'Martin Wolf', createdAt: '2026-02-24', category: 'API' },
  { id: 'tk-7', subject: 'Benutzerrolle kann nicht geändert werden', status: 'open', priority: 'medium', contactId: 'x9', contactName: 'Frank Neubauer', company: 'BioTech Solutions GmbH', assigneeId: 'e17', assigneeName: 'Jonas Schmitt', createdAt: '2026-02-23', category: 'Benutzerverwaltung' },
  { id: 'tk-8', subject: 'Kalender-Sync mit Outlook fehlerhaft', status: 'in_progress', priority: 'high', contactId: 'x10', contactName: 'Susanne Kramer', company: 'Kramer Automotive GmbH', assigneeId: 'e16', assigneeName: 'Martin Wolf', createdAt: '2026-02-19', category: 'Kalender' },
]

// ---------------------------------------------------------------------------
// Finance: Banking mock
// ---------------------------------------------------------------------------

export interface MockBankAccount {
  id: string
  name: string
  iban: string
  balance: number
  currency: string
}

export interface MockTransaction {
  id: string
  date: string
  description: string
  amount: number
  type: 'eingang' | 'ausgang'
  category: string
  counterparty: string
  status: 'gebucht' | 'ausstehend'
}

export const BANK_ACCOUNTS: MockBankAccount[] = [
  { id: 'ba1', name: 'Geschäftskonto', iban: 'DE89 3704 0044 0532 0130 00', balance: 142850.75, currency: 'EUR' },
  { id: 'ba2', name: 'Rücklagenkonto', iban: 'DE72 3704 0044 0532 0131 17', balance: 85000.00, currency: 'EUR' },
]

export const BANK_TRANSACTIONS: MockTransaction[] = [
  { id: 'bt1', date: '2026-02-24', description: 'Eingang Gruber Maschinenbau — Abschlag 3', amount: 16000, type: 'eingang', category: 'Projekterlös', counterparty: 'Gruber Maschinenbau GmbH', status: 'gebucht' },
  { id: 'bt2', date: '2026-02-23', description: 'CloudFirst Hosting — Monatsrechnung Feb', amount: -1890, type: 'ausgang', category: 'Hosting', counterparty: 'CloudFirst Hosting GmbH', status: 'gebucht' },
  { id: 'bt3', date: '2026-02-22', description: 'Eingang DataFlow — Analytics Dashboard', amount: 14000, type: 'eingang', category: 'Projekterlös', counterparty: 'DataFlow GmbH', status: 'gebucht' },
  { id: 'bt4', date: '2026-02-21', description: 'Gehälter Februar 2026', amount: -78500, type: 'ausgang', category: 'Gehälter', counterparty: 'Sammelüberweisung', status: 'gebucht' },
  { id: 'bt5', date: '2026-02-20', description: 'Büromiete Leopoldstrasse', amount: -4200, type: 'ausgang', category: 'Miete', counterparty: 'Bayerische Hausverwaltung', status: 'gebucht' },
  { id: 'bt6', date: '2026-02-19', description: 'Eingang Stadler Bau — Intranet Portal', amount: 21000, type: 'eingang', category: 'Projekterlös', counterparty: 'Stadler Bauunternehmen GmbH', status: 'gebucht' },
  { id: 'bt7', date: '2026-02-18', description: 'Adobe Creative Cloud — Jahresrechnung', amount: -4188, type: 'ausgang', category: 'Software-Lizenzen', counterparty: 'Adobe Inc.', status: 'gebucht' },
  { id: 'bt8', date: '2026-02-25', description: 'Eingang Berger — Mobile App Anzahlung', amount: 20000, type: 'eingang', category: 'Projekterlös', counterparty: 'Berger & Söhne', status: 'ausstehend' },
  { id: 'bt9', date: '2026-02-26', description: 'DATEV Lizenzgebühr Q1/2026', amount: -890, type: 'ausgang', category: 'Software-Lizenzen', counterparty: 'DATEV eG', status: 'ausstehend' },
]

// ---------------------------------------------------------------------------
// KPI Summary (for dashboard KPI widgets)
// ---------------------------------------------------------------------------

export const KPI = {
  revenue: {
    current: 84750,
    previous: 71000,
    changePercent: 19.4,
    ytdTotal: 172750,
  },
  tasks: {
    total: 24,
    done: 17,
    inProgress: 5,
    overdue: 2,
  },
  deals: {
    pipelineValue: 237500,
    activeDeals: 16,
    wonThisMonth: 2,
    winRate: 80,
    avgDealSize: 31200,
  },
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

export function getEmployee(id: string): MockEmployee | undefined {
  return EMPLOYEES.find((e) => e.id === id)
}

export function getExternalContact(id: string): MockExternalContact | undefined {
  return EXTERNAL_CONTACTS.find((c) => c.id === id)
}

export function getProject(id: string): MockProject | undefined {
  return PROJECTS.find((p) => p.id === id)
}

export function getTasksForEmployee(employeeId: string): MockTask[] {
  return TASKS.filter((t) => t.assigneeId === employeeId)
}

export function getTasksForProject(projectId: string): MockTask[] {
  return TASKS.filter((t) => t.projectId === projectId)
}

export function getDealsForContact(contactId: string): MockDeal[] {
  return DEALS.filter((d) => d.contactId === contactId)
}

export function getProjectsForEmployee(employeeId: string): MockProject[] {
  return PROJECTS.filter((p) => p.teamMemberIds.includes(employeeId) || p.managerId === employeeId)
}

/** Get all contacts (employees as contacts + external) for CRM store */
export function getAllContactsForCRM() {
  return [
    ...EMPLOYEES.map((e) => ({
      id: e.id,
      salutation: '' as const,
      title: '',
      firstName: e.firstName,
      lastName: e.lastName,
      initials: e.initials,
      email: e.email,
      phone: e.phone,
      mobile: e.mobile,
      company: COMPANY.name,
      jobTitle: e.jobTitle,
      department: e.department,
      address: { street: COMPANY.address.street, zip: COMPANY.address.zip, city: COMPANY.address.city, country: COMPANY.address.country },
      website: COMPANY.website,
      category: 'employee' as const,
      status: 'active' as const,
      tags: e.skills.slice(0, 3),
      notes: `${e.jobTitle} — ${e.department}`,
      socialMedia: { linkedin: `${e.firstName.toLowerCase()}-${e.lastName.toLowerCase()}`, xing: '' },
      lastContact: '2026-02-20',
      projects: getProjectsForEmployee(e.id).map((p) => p.name),
      createdAt: e.joinDate,
      isFavorite: e.role !== 'member',
      activities: [],
    })),
    ...EXTERNAL_CONTACTS.map((c) => ({
      id: c.id,
      salutation: c.salutation,
      title: c.title,
      firstName: c.firstName,
      lastName: c.lastName,
      initials: c.initials,
      email: c.email,
      phone: c.phone,
      mobile: c.mobile,
      company: c.company,
      jobTitle: c.jobTitle,
      department: '',
      address: { street: '', zip: '', city: c.city, country: 'Deutschland' },
      website: '',
      category: c.category,
      status: c.status,
      tags: c.tags,
      notes: c.notes,
      socialMedia: { linkedin: `${c.firstName.toLowerCase()}-${c.lastName.toLowerCase()}`, xing: '' },
      lastContact: c.lastContact,
      projects: PROJECTS.filter((p) => p.clientId === c.id).map((p) => p.name),
      createdAt: '2025-06-01',
      isFavorite: c.tags.includes('VIP'),
      activities: [],
    })),
  ]
}

/** Get all team members for Team store */
export function getAllTeamMembers() {
  return EMPLOYEES.map((e) => ({
    id: e.id,
    firstName: e.firstName,
    lastName: e.lastName,
    initials: e.initials,
    email: e.email,
    phone: e.phone,
    mobile: e.mobile,
    role: e.jobTitle,
    department: e.department,
    status: e.status as 'online' | 'away' | 'offline' | 'dnd',
    contractType: e.contractType,
    workload: e.workload,
    joinDate: e.joinDate,
    manager: e.managerId ? `${getEmployee(e.managerId)?.firstName} ${getEmployee(e.managerId)?.lastName}` : undefined,
    location: e.location,
    currentTask: e.currentTask,
    projects: getProjectsForEmployee(e.id).map((p) => p.name),
    skills: e.skills,
    isActive: true,
  }))
}
