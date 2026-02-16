import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type FileType = 'pdf' | 'word' | 'excel' | 'image' | 'video' | 'archive' | 'other'

export interface DocFile {
  id: string
  name: string
  type: FileType
  size: string
  sizeBytes: number
  date: string
  folderId: string
  tags: string[]
  createdBy: string
  isFavorite: boolean
  isShared: boolean
  isVault: boolean
  sharedWith: { name: string; permission: 'view' | 'edit' }[]
  versions: { version: string; date: string; author: string }[]
}

export interface DocFolder {
  id: string
  name: string
  parentId: string | null
  icon: 'folder' | 'share' | 'star' | 'lock'
  isSystem: boolean
}

export interface WikiArticle {
  id: string
  title: string
  categoryId: string
  content: string
  tags: string[]
  author: string
  createdAt: string
  updatedAt: string
  views: number
}

export interface WikiCategory {
  id: string
  name: string
  icon: string
  parentId: string | null
  order: number
}

interface DocumentsState {
  files: DocFile[]
  folders: DocFolder[]
  wikiArticles: WikiArticle[]
  wikiCategories: WikiCategory[]
  addFile: (file: Omit<DocFile, 'id' | 'versions'>) => void
  removeFile: (id: string) => void
  renameFile: (id: string, name: string) => void
  moveFile: (id: string, folderId: string) => void
  toggleFavorite: (id: string) => void
  toggleShare: (id: string) => void
  updateFileTags: (id: string, tags: string[]) => void
  addFolder: (name: string, parentId: string | null) => void
  renameFolder: (id: string, name: string) => void
  deleteFolder: (id: string) => void
  totalStorageUsed: () => number
  addWikiArticle: (article: Omit<WikiArticle, 'id'>) => void
  updateWikiArticle: (id: string, updates: Partial<Omit<WikiArticle, 'id'>>) => void
  deleteWikiArticle: (id: string) => void
  addWikiCategory: (category: Omit<WikiCategory, 'id'>) => void
}

const mockFolders: DocFolder[] = [
  { id: 'root', name: 'Alle Dateien', parentId: null, icon: 'folder', isSystem: true },
  { id: 'projects', name: 'Projekte', parentId: 'root', icon: 'folder', isSystem: false },
  { id: 'contracts', name: 'Verträge', parentId: 'root', icon: 'folder', isSystem: false },
  { id: 'invoices', name: 'Rechnungen', parentId: 'root', icon: 'folder', isSystem: false },
  { id: 'marketing', name: 'Marketing', parentId: 'root', icon: 'folder', isSystem: false },
  { id: 'hr', name: 'HR', parentId: 'root', icon: 'folder', isSystem: false },
  { id: 'shared', name: 'Geteilt mit mir', parentId: null, icon: 'share', isSystem: true },
  { id: 'favorites', name: 'Favoriten', parentId: null, icon: 'star', isSystem: true },
  { id: 'vault', name: 'Tresor', parentId: null, icon: 'lock', isSystem: true },
  { id: 'vault-hr', name: 'HR Dokumente', parentId: 'vault', icon: 'folder', isSystem: false },
  { id: 'vault-finance', name: 'Finanzen', parentId: 'vault', icon: 'folder', isSystem: false },
]

const mockFiles: DocFile[] = [
  {
    id: 'd1', name: 'Projektplan_Q1_2026.pdf', type: 'pdf', size: '2.4 MB', sizeBytes: 2516582,
    date: '2026-02-07', folderId: 'projects', tags: ['Q1', 'Planung'], createdBy: 'Anna Mueller',
    isFavorite: true, isShared: true, isVault: false,
    sharedWith: [{ name: 'Michael Berg', permission: 'edit' }, { name: 'Sarah Klein', permission: 'view' }],
    versions: [{ version: '1.2', date: '2026-02-07', author: 'Anna Mueller' }, { version: '1.1', date: '2026-02-03', author: 'Anna Mueller' }, { version: '1.0', date: '2026-01-28', author: 'Michael Berg' }],
  },
  {
    id: 'd2', name: 'Vertrag_Kunde_ABC.pdf', type: 'pdf', size: '1.8 MB', sizeBytes: 1887437,
    date: '2026-02-06', folderId: 'contracts', tags: ['Vertrag', 'ABC GmbH'], createdBy: 'Peter Koch',
    isFavorite: false, isShared: false, isVault: true,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-02-06', author: 'Peter Koch' }],
  },
  {
    id: 'd3', name: 'Budget_2026.xlsx', type: 'excel', size: '540 KB', sizeBytes: 552960,
    date: '2026-02-05', folderId: 'invoices', tags: ['Budget', 'Finanzen'], createdBy: 'Michael Berg',
    isFavorite: true, isShared: true, isVault: false,
    sharedWith: [{ name: 'Anna Mueller', permission: 'edit' }],
    versions: [{ version: '2.0', date: '2026-02-05', author: 'Michael Berg' }, { version: '1.0', date: '2026-01-15', author: 'Michael Berg' }],
  },
  {
    id: 'd4', name: 'Logo_Redesign_v3.png', type: 'image', size: '4.2 MB', sizeBytes: 4404019,
    date: '2026-02-04', folderId: 'marketing', tags: ['Design', 'Branding'], createdBy: 'Sarah Klein',
    isFavorite: false, isShared: true, isVault: false,
    sharedWith: [{ name: 'Lisa Schmidt', permission: 'view' }],
    versions: [{ version: '3.0', date: '2026-02-04', author: 'Sarah Klein' }, { version: '2.0', date: '2026-01-20', author: 'Sarah Klein' }],
  },
  {
    id: 'd5', name: 'Onboarding_Guide.docx', type: 'word', size: '890 KB', sizeBytes: 911360,
    date: '2026-02-03', folderId: 'hr', tags: ['HR', 'Onboarding'], createdBy: 'Lisa Schmidt',
    isFavorite: false, isShared: true, isVault: false,
    sharedWith: [{ name: 'Anna Mueller', permission: 'edit' }, { name: 'Peter Koch', permission: 'view' }],
    versions: [{ version: '1.1', date: '2026-02-03', author: 'Lisa Schmidt' }],
  },
  {
    id: 'd6', name: 'Produktvideo_Final.mp4', type: 'video', size: '156 MB', sizeBytes: 163577856,
    date: '2026-02-02', folderId: 'marketing', tags: ['Video', 'Marketing'], createdBy: 'Jonas Diaz',
    isFavorite: false, isShared: false, isVault: false,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-02-02', author: 'Jonas Diaz' }],
  },
  {
    id: 'd7', name: 'Archiv_2025.zip', type: 'archive', size: '340 MB', sizeBytes: 356515840,
    date: '2026-01-31', folderId: 'root', tags: ['Archiv', '2025'], createdBy: 'Peter Koch',
    isFavorite: false, isShared: false, isVault: false,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-01-31', author: 'Peter Koch' }],
  },
  {
    id: 'd8', name: 'Mitarbeiterakte_vertraulich.pdf', type: 'pdf', size: '3.1 MB', sizeBytes: 3250586,
    date: '2026-02-01', folderId: 'vault-hr', tags: ['HR', 'Vertraulich'], createdBy: 'Anna Mueller',
    isFavorite: false, isShared: false, isVault: true,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-02-01', author: 'Anna Mueller' }],
  },
  {
    id: 'd9', name: 'Sprint_Retro_Notes.docx', type: 'word', size: '120 KB', sizeBytes: 122880,
    date: '2026-02-07', folderId: 'projects', tags: ['Sprint', 'Retro'], createdBy: 'Michael Berg',
    isFavorite: false, isShared: true, isVault: false,
    sharedWith: [{ name: 'Anna Mueller', permission: 'view' }],
    versions: [{ version: '1.0', date: '2026-02-07', author: 'Michael Berg' }],
  },
  {
    id: 'd10', name: 'Kampagne_Assets.zip', type: 'archive', size: '89 MB', sizeBytes: 93323264,
    date: '2026-02-06', folderId: 'marketing', tags: ['Marketing', 'Assets'], createdBy: 'Sarah Klein',
    isFavorite: true, isShared: false, isVault: false,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-02-06', author: 'Sarah Klein' }],
  },
  {
    id: 'd11', name: 'Angebot_Steiner_Bau.pdf', type: 'pdf', size: '1.5 MB', sizeBytes: 1572864,
    date: '2026-02-08', folderId: 'contracts', tags: ['Angebot', 'Steiner Bau'], createdBy: 'Anna Mueller',
    isFavorite: false, isShared: true, isVault: false,
    sharedWith: [{ name: 'Thomas Weber', permission: 'view' }],
    versions: [{ version: '1.0', date: '2026-02-08', author: 'Anna Mueller' }],
  },
  {
    id: 'd12', name: 'Bilanz_Q4_2025.xlsx', type: 'excel', size: '780 KB', sizeBytes: 798720,
    date: '2026-01-15', folderId: 'vault-finance', tags: ['Bilanz', 'Q4'], createdBy: 'Michael Berg',
    isFavorite: false, isShared: false, isVault: true,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-01-15', author: 'Michael Berg' }],
  },
]

const mockWikiCategories: WikiCategory[] = [
  { id: 'wc-general', name: 'Allgemein', icon: 'BookOpen', parentId: null, order: 1 },
  { id: 'wc-tech', name: 'IT & Technik', icon: 'Monitor', parentId: null, order: 2 },
  { id: 'wc-hr', name: 'Personal & HR', icon: 'Users', parentId: null, order: 3 },
  { id: 'wc-process', name: 'Prozesse', icon: 'GitBranch', parentId: null, order: 4 },
  { id: 'wc-templates', name: 'Vorlagen', icon: 'FileText', parentId: null, order: 5 },
]

const mockWikiArticles: WikiArticle[] = [
  {
    id: 'wa-1',
    title: 'Willkommen im KMU Hub Wiki',
    categoryId: 'wc-general',
    content: 'Das KMU Hub Wiki ist die zentrale Wissensdatenbank unseres Unternehmens. Hier findest du alle wichtigen Informationen, Anleitungen und Richtlinien, die du fuer deine taegliche Arbeit brauchst.\n\nAlle Mitarbeiter sind eingeladen, aktiv zum Wiki beizutragen. Wenn du neues Wissen hast oder bestehende Artikel aktualisieren moechtest, nutze einfach die Bearbeitungsfunktion.\n\nBei Fragen zum Wiki wende dich bitte an das IT-Team oder deinen Vorgesetzten.',
    tags: ['Einstieg', 'Uebersicht', 'Willkommen'],
    author: 'Anna Mueller',
    createdAt: '2026-01-10',
    updatedAt: '2026-02-01',
    views: 342,
  },
  {
    id: 'wa-2',
    title: 'Onboarding neuer Mitarbeiter',
    categoryId: 'wc-hr',
    content: 'Checkliste fuer das Onboarding neuer Mitarbeiter:\n\n1. Arbeitsvertrag unterschreiben und Personalbogen ausfuellen\n2. IT-Zugaenge beantragen (E-Mail, VPN, KMU Hub)\n3. Arbeitsplatz einrichten und Schluessel uebergeben\n4. Einweisungen durchfuehren (Datenschutz, Arbeitssicherheit)\n5. Mentor zuweisen und Einarbeitungsplan erstellen\n6. Probezeitgespraech nach 3 Monaten terminieren',
    tags: ['Onboarding', 'Checkliste', 'Neue Mitarbeiter'],
    author: 'Lisa Schmidt',
    createdAt: '2026-01-12',
    updatedAt: '2026-02-05',
    views: 189,
  },
  {
    id: 'wa-3',
    title: 'VPN-Zugang einrichten',
    categoryId: 'wc-tech',
    content: 'Anleitung zum Einrichten des VPN-Zugangs:\n\n1. WireGuard Client herunterladen (wireguard.com/install)\n2. Konfigurationsdatei beim IT-Team anfordern\n3. Konfiguration importieren: WireGuard oeffnen > Tunnel hinzufuegen > Datei importieren\n4. Verbindung aktivieren und Zugang testen\n5. Bei Problemen: IT-Helpdesk kontaktieren (Ticket erstellen)\n\nWichtig: VPN-Zugang ist Pflicht fuer Remote-Arbeit und den Zugriff auf interne Systeme.',
    tags: ['VPN', 'Anleitung', 'Remote'],
    author: 'Jonas Diaz',
    createdAt: '2026-01-15',
    updatedAt: '2026-01-28',
    views: 256,
  },
  {
    id: 'wa-4',
    title: 'Urlaubsantrag stellen',
    categoryId: 'wc-hr',
    content: 'So stellst du einen Urlaubsantrag:\n\nMelde dich im KMU Hub an und navigiere zum Modul "Personal". Waehle dort "Urlaubsantrag" und gib den gewuenschten Zeitraum ein. Der Antrag wird automatisch an deinen Vorgesetzten weitergeleitet. Die Bearbeitung dauert in der Regel 2 Werktage. Bei dringenden Anfragen sprich bitte direkt mit deiner Fuehrungskraft. Resturlaub muss bis zum 31. Maerz des Folgejahres genommen werden.',
    tags: ['Urlaub', 'Antrag', 'Personal'],
    author: 'Lisa Schmidt',
    createdAt: '2026-01-18',
    updatedAt: '2026-02-10',
    views: 412,
  },
  {
    id: 'wa-5',
    title: 'IT-Sicherheitsrichtlinien',
    categoryId: 'wc-tech',
    content: 'Verbindliche IT-Sicherheitsrichtlinien fuer alle Mitarbeiter:\n\nPasswoerter muessen mindestens 12 Zeichen lang sein und Gross-/Kleinbuchstaben, Zahlen sowie Sonderzeichen enthalten. Passwoerter alle 90 Tage aendern. Zwei-Faktor-Authentifizierung ist fuer alle Systeme Pflicht. Verdaechtige E-Mails nicht oeffnen und sofort an security@firma.ch weiterleiten. USB-Sticks duerfen nicht an Firmengeraete angeschlossen werden. Software darf nur durch das IT-Team installiert werden.',
    tags: ['Sicherheit', 'Richtlinien', 'Passwort'],
    author: 'Jonas Diaz',
    createdAt: '2026-01-20',
    updatedAt: '2026-02-08',
    views: 178,
  },
  {
    id: 'wa-6',
    title: 'Reisekostenabrechnung',
    categoryId: 'wc-process',
    content: 'Anleitung zur Reisekostenabrechnung:\n\nAlle Belege muessen als Scan oder Foto eingereicht werden. Die Abrechnung erfolgt ueber das Buchhaltungsmodul unter "Ausgaben > Reisekosten". Tagespauschalen: Inland CHF 45, EU CHF 60, uebrige Laender CHF 80. Hotelkosten werden bis max. CHF 180/Nacht erstattet. Fahrtkosten mit Privatfahrzeug: CHF 0.70/km. Einreichungsfrist: 30 Tage nach Reiseende.',
    tags: ['Reisekosten', 'Abrechnung', 'Spesen'],
    author: 'Michael Berg',
    createdAt: '2026-01-22',
    updatedAt: '2026-02-03',
    views: 134,
  },
  {
    id: 'wa-7',
    title: 'E-Mail Signaturen',
    categoryId: 'wc-tech',
    content: 'Vorlage fuer einheitliche E-Mail-Signaturen:\n\nMit freundlichen Gruessen\n[Vorname Nachname]\n[Position/Abteilung]\n\nFirma GmbH\nMusterstrasse 42 | 8001 Zuerich\nTel: +41 44 123 45 67\nwww.firma.ch\n\nDie Signatur muss in allen geschaeftlichen E-Mails verwendet werden. Persoenliche Zitate oder Bilder sind nicht gestattet. Bei Fragen zur Einrichtung hilft das IT-Team.',
    tags: ['E-Mail', 'Signatur', 'Vorlage'],
    author: 'Sarah Klein',
    createdAt: '2026-01-25',
    updatedAt: '2026-01-25',
    views: 98,
  },
  {
    id: 'wa-8',
    title: 'Datenschutz (DSGVO)',
    categoryId: 'wc-general',
    content: 'Uebersicht zu den Datenschutzrichtlinien gemaess DSGVO und Schweizer DSG:\n\nPersonenbezogene Daten duerfen nur zweckgebunden erhoben und verarbeitet werden. Jeder Mitarbeiter ist verpflichtet, die Datenschutzschulung jaehrlich zu absolvieren. Datenpannen muessen innerhalb von 72 Stunden gemeldet werden. Kundenanfragen zur Datenauskunft oder Loeschung sind umgehend an den Datenschutzbeauftragten weiterzuleiten. Dokumente mit personenbezogenen Daten muessen im Tresor-Bereich gespeichert werden.',
    tags: ['Datenschutz', 'DSGVO', 'Compliance'],
    author: 'Peter Koch',
    createdAt: '2026-01-28',
    updatedAt: '2026-02-12',
    views: 267,
  },
  {
    id: 'wa-9',
    title: 'Meeting-Protokoll Vorlage',
    categoryId: 'wc-templates',
    content: 'Vorlage fuer Meeting-Protokolle:\n\nDatum: [TT.MM.JJJJ]\nTeilnehmer: [Namen]\nProtokollant: [Name]\n\nAgenda:\n1. [Thema 1]\n2. [Thema 2]\n\nBeschlossene Massnahmen:\n- [Massnahme] — Verantwortlich: [Name] — Frist: [Datum]\n\nNaechster Termin: [Datum/Uhrzeit]\n\nDas Protokoll ist innerhalb von 24 Stunden nach dem Meeting an alle Teilnehmer zu versenden.',
    tags: ['Meeting', 'Protokoll', 'Vorlage'],
    author: 'Anna Mueller',
    createdAt: '2026-02-01',
    updatedAt: '2026-02-01',
    views: 76,
  },
  {
    id: 'wa-10',
    title: 'Notfallkontakte',
    categoryId: 'wc-general',
    content: 'Wichtige Notfallkontakte:\n\nIT-Notfall (Systemausfall, Sicherheitsvorfall): +41 44 123 45 00 / it-notfall@firma.ch\nGebaeudemanagement (Wasser, Strom, Einbruch): +41 44 123 45 10\nGeschaeftsfuehrung: +41 79 234 56 78\nDatenschutzbeauftragter: datenschutz@firma.ch\nBetriebsarzt: +41 44 987 65 43\n\nBei lebensbedrohlichen Notfaellen immer zuerst 144 (Sanitaet) oder 117 (Polizei) anrufen.',
    tags: ['Notfall', 'Kontakte', 'Wichtig'],
    author: 'Peter Koch',
    createdAt: '2026-02-05',
    updatedAt: '2026-02-14',
    views: 523,
  },
]

let nextFileId = 13
let nextFolderId = 12
let nextWikiArticleId = 11
let nextWikiCategoryId = 6

export const useDocumentsStore = create<DocumentsState>()(
  persist(
    (set, get) => ({
      files: mockFiles,
      folders: mockFolders,
      wikiArticles: mockWikiArticles,
      wikiCategories: mockWikiCategories,

      addFile: (file) =>
        set((state) => ({
          files: [
            { ...file, id: `d${nextFileId++}`, versions: [{ version: '1.0', date: file.date, author: file.createdBy }] },
            ...state.files,
          ],
        })),

      removeFile: (id) =>
        set((state) => ({ files: state.files.filter((f) => f.id !== id) })),

      renameFile: (id, name) =>
        set((state) => ({
          files: state.files.map((f) => (f.id === id ? { ...f, name } : f)),
        })),

      moveFile: (id, folderId) =>
        set((state) => ({
          files: state.files.map((f) => (f.id === id ? { ...f, folderId } : f)),
        })),

      toggleFavorite: (id) =>
        set((state) => ({
          files: state.files.map((f) =>
            f.id === id ? { ...f, isFavorite: !f.isFavorite } : f
          ),
        })),

      toggleShare: (id) =>
        set((state) => ({
          files: state.files.map((f) =>
            f.id === id ? { ...f, isShared: !f.isShared } : f
          ),
        })),

      updateFileTags: (id, tags) =>
        set((state) => ({
          files: state.files.map((f) => (f.id === id ? { ...f, tags } : f)),
        })),

      addFolder: (name, parentId) =>
        set((state) => ({
          folders: [
            ...state.folders,
            { id: `folder-${nextFolderId++}`, name, parentId: parentId || 'root', icon: 'folder', isSystem: false },
          ],
        })),

      renameFolder: (id, name) =>
        set((state) => ({
          folders: state.folders.map((f) => (f.id === id ? { ...f, name } : f)),
        })),

      deleteFolder: (id) =>
        set((state) => ({
          folders: state.folders.filter((f) => f.id !== id),
          files: state.files.map((f) => (f.folderId === id ? { ...f, folderId: 'root' } : f)),
        })),

      addWikiArticle: (article) =>
        set((state) => ({
          wikiArticles: [
            { ...article, id: `wa-${nextWikiArticleId++}` },
            ...state.wikiArticles,
          ],
        })),

      updateWikiArticle: (id, updates) =>
        set((state) => ({
          wikiArticles: state.wikiArticles.map((a) =>
            a.id === id ? { ...a, ...updates } : a
          ),
        })),

      deleteWikiArticle: (id) =>
        set((state) => ({
          wikiArticles: state.wikiArticles.filter((a) => a.id !== id),
        })),

      addWikiCategory: (category) =>
        set((state) => ({
          wikiCategories: [
            ...state.wikiCategories,
            { ...category, id: `wc-${nextWikiCategoryId++}` },
          ],
        })),

      totalStorageUsed: () => get().files.reduce((sum, f) => sum + f.sizeBytes, 0),
    }),
    { name: 'kmuhub-documents' }
  )
)
