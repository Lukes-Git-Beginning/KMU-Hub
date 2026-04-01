import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { WikiArticle, WikiCategory, WikiTemplate } from '@/types/wiki'

// ---------------------------------------------------------------------------
// State interface
// ---------------------------------------------------------------------------

interface WikiState {
  categories: WikiCategory[]
  articles: WikiArticle[]
  templates: WikiTemplate[]

  // UI state
  selectedArticleId: string | null
  selectedCategoryId: string | null
  isEditing: boolean
  searchQuery: string

  // Actions — articles
  addArticle: (article: Omit<WikiArticle, 'id' | 'slug' | 'viewCount' | 'versions' | 'createdAt'>) => void
  updateArticle: (id: string, updates: Partial<WikiArticle>) => void
  deleteArticle: (id: string) => void
  publishArticle: (id: string) => void
  archiveArticle: (id: string) => void
  togglePin: (id: string) => void
  incrementViewCount: (id: string) => void

  // Actions — categories
  addCategory: (category: Omit<WikiCategory, 'id' | 'articleCount'>) => void
  updateCategory: (id: string, updates: Partial<WikiCategory>) => void
  deleteCategory: (id: string) => void

  // Actions — UI
  setSelectedArticle: (id: string | null) => void
  setSelectedCategory: (id: string | null) => void
  setEditing: (editing: boolean) => void
  setSearchQuery: (query: string) => void
}

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

const mockCategories: WikiCategory[] = [
  { id: 'wc1', name: 'Unternehmen', icon: 'Building2', sortOrder: 1, articleCount: 3 },
  { id: 'wc2', name: 'Prozesse', icon: 'GitBranch', sortOrder: 2, articleCount: 4 },
  { id: 'wc3', name: 'IT & Tools', icon: 'Monitor', sortOrder: 3, articleCount: 3 },
  { id: 'wc4', name: 'HR & Onboarding', icon: 'Users', sortOrder: 4, articleCount: 2 },
]

const mockArticles: WikiArticle[] = [
  {
    id: 'wa1',
    title: 'Willkommen bei Cosmi',
    slug: 'willkommen',
    content: '<h2>Willkommen im internen Wiki</h2><p>Hier findest du alle wichtigen Informationen rund um unser Unternehmen, Prozesse und Tools.</p><h3>Erste Schritte</h3><ul><li>Lies die Unternehmensrichtlinien</li><li>Richte dein Profil ein</li><li>Tritt den relevanten Chat-Kanälen bei</li></ul>',
    categoryId: 'wc1',
    status: 'published',
    authorId: 'c1',
    authorName: 'Anna Müller',
    tags: ['onboarding', 'start'],
    isPinned: true,
    viewCount: 247,
    lastEditedBy: 'Anna Müller',
    lastEditedAt: '2026-02-18',
    createdAt: '2025-09-01',
    versions: [
      { id: 'wv1', articleId: 'wa1', version: 3, editorName: 'Anna Müller', editedAt: '2026-02-18', changeNote: 'Neue Schritte hinzugefügt' },
      { id: 'wv2', articleId: 'wa1', version: 2, editorName: 'Thomas Weber', editedAt: '2026-01-10', changeNote: 'Links aktualisiert' },
      { id: 'wv3', articleId: 'wa1', version: 1, editorName: 'Anna Müller', editedAt: '2025-09-01', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa2',
    title: 'Organigramm & Ansprechpartner',
    slug: 'organigramm',
    content: '<h2>Organigramm</h2><p>Unser Unternehmen ist in folgende Abteilungen gegliedert:</p><h3>Geschäftsleitung</h3><p>Max Mustermann — CEO</p><h3>Entwicklung</h3><p>Anna Müller — Head of Engineering</p><h3>Vertrieb</h3><p>Peter Schmidt — Head of Sales</p>',
    categoryId: 'wc1',
    status: 'published',
    authorId: 'c3',
    authorName: 'Thomas Weber',
    tags: ['organisation', 'team'],
    isPinned: false,
    viewCount: 182,
    lastEditedBy: 'Thomas Weber',
    lastEditedAt: '2026-02-05',
    createdAt: '2025-10-15',
    versions: [
      { id: 'wv4', articleId: 'wa2', version: 2, editorName: 'Thomas Weber', editedAt: '2026-02-05', changeNote: 'Neue Abteilung eingefügt' },
      { id: 'wv5', articleId: 'wa2', version: 1, editorName: 'Thomas Weber', editedAt: '2025-10-15', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa3',
    title: 'Homeoffice-Richtlinien',
    slug: 'homeoffice',
    content: '<h2>Homeoffice-Regelung</h2><p>Mitarbeiter können bis zu 3 Tage pro Woche im Homeoffice arbeiten.</p><h3>Voraussetzungen</h3><ul><li>Absprache mit dem Teamlead</li><li>Erreichbarkeit während Kernarbeitszeit (09:00–16:00)</li><li>Sichere Internetverbindung</li></ul><h3>Equipment</h3><p>Laptop und Headset werden gestellt. Bildschirm auf Anfrage.</p>',
    categoryId: 'wc1',
    status: 'published',
    authorId: 'c5',
    authorName: 'Lisa Braun',
    tags: ['homeoffice', 'richtlinien'],
    isPinned: false,
    viewCount: 95,
    lastEditedBy: 'Lisa Braun',
    lastEditedAt: '2026-01-20',
    createdAt: '2025-11-01',
    versions: [
      { id: 'wv6', articleId: 'wa3', version: 1, editorName: 'Lisa Braun', editedAt: '2025-11-01', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa4',
    title: 'Angebotsprozess Schritt für Schritt',
    slug: 'angebotsprozess',
    content: '<h2>Angebotsprozess</h2><ol><li><strong>Lead qualifizieren:</strong> Budget, Zeitrahmen, Entscheider klären</li><li><strong>Anforderungen aufnehmen:</strong> Bedarfsanalyse durchführen</li><li><strong>Angebot erstellen:</strong> Vorlage in Dokumente verwenden</li><li><strong>Review:</strong> Teamlead prüft Angebot</li><li><strong>Versand:</strong> Per E-Mail mit PDF</li><li><strong>Follow-Up:</strong> Nach 5 Tagen nachfassen</li></ol>',
    categoryId: 'wc2',
    status: 'published',
    authorId: 'c4',
    authorName: 'Peter Schmidt',
    tags: ['vertrieb', 'angebot', 'prozess'],
    isPinned: true,
    viewCount: 134,
    lastEditedBy: 'Peter Schmidt',
    lastEditedAt: '2026-02-10',
    createdAt: '2025-10-01',
    versions: [
      { id: 'wv7', articleId: 'wa4', version: 2, editorName: 'Peter Schmidt', editedAt: '2026-02-10', changeNote: 'Follow-Up Schritt ergänzt' },
      { id: 'wv8', articleId: 'wa4', version: 1, editorName: 'Peter Schmidt', editedAt: '2025-10-01', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa5',
    title: 'Rechnungsstellung & Mahnwesen',
    slug: 'rechnungsstellung',
    content: '<h2>Rechnungsstellung</h2><p>Rechnungen werden über das Modul "Rechnungen & Finanzen" erstellt.</p><h3>Ablauf</h3><ol><li>Rechnung aus Angebot/Auftrag generieren</li><li>Positionen prüfen, MWSt kontrollieren</li><li>PDF generieren und per E-Mail versenden</li></ol><h3>Mahnwesen</h3><p>Nach 30 Tagen: 1. Mahnung (freundlich). Nach 45 Tagen: 2. Mahnung. Nach 60 Tagen: 3. Mahnung mit Fristsetzung.</p>',
    categoryId: 'wc2',
    status: 'published',
    authorId: 'c6',
    authorName: 'Markus Keller',
    tags: ['finanzen', 'rechnung', 'mahnung'],
    isPinned: false,
    viewCount: 78,
    lastEditedBy: 'Markus Keller',
    lastEditedAt: '2026-01-15',
    createdAt: '2025-11-20',
    versions: [
      { id: 'wv9', articleId: 'wa5', version: 1, editorName: 'Markus Keller', editedAt: '2025-11-20', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa6',
    title: 'Kundensupport-Leitfaden',
    slug: 'support-leitfaden',
    content: '<h2>Support-Leitfaden</h2><p>Dieser Leitfaden beschreibt den Standard-Ablauf für Kundensupport-Anfragen.</p><h3>Prioritäten</h3><table><tr><th>Prio</th><th>Beschreibung</th><th>Reaktionszeit</th></tr><tr><td>Kritisch</td><td>System nicht nutzbar</td><td>1 Stunde</td></tr><tr><td>Hoch</td><td>Wichtige Funktion eingeschränkt</td><td>4 Stunden</td></tr><tr><td>Normal</td><td>Einzelne Funktion betroffen</td><td>1 Arbeitstag</td></tr><tr><td>Niedrig</td><td>Wunsch / Verbesserung</td><td>3 Arbeitstage</td></tr></table>',
    categoryId: 'wc2',
    status: 'published',
    authorId: 'c2',
    authorName: 'Sarah Meier',
    tags: ['support', 'helpdesk', 'sla'],
    isPinned: false,
    viewCount: 156,
    lastEditedBy: 'Sarah Meier',
    lastEditedAt: '2026-02-12',
    createdAt: '2025-09-15',
    versions: [
      { id: 'wv10', articleId: 'wa6', version: 3, editorName: 'Sarah Meier', editedAt: '2026-02-12', changeNote: 'SLA-Tabelle ergänzt' },
      { id: 'wv11', articleId: 'wa6', version: 2, editorName: 'Sarah Meier', editedAt: '2025-12-01', changeNote: 'Prioritäten überarbeitet' },
      { id: 'wv12', articleId: 'wa6', version: 1, editorName: 'Sarah Meier', editedAt: '2025-09-15', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa7',
    title: 'Datenschutz & DSGVO intern',
    slug: 'datenschutz',
    content: '<h2>Datenschutz im Unternehmen</h2><p>Alle Mitarbeiter sind verpflichtet, die DSGVO-Richtlinien einzuhalten.</p><h3>Grundregeln</h3><ul><li>Personenbezogene Daten nur zweckgebunden verarbeiten</li><li>Keine Kundendaten per unverschlüsselter E-Mail</li><li>Bildschirm sperren bei Verlassen des Arbeitsplatzes</li><li>Datenschutzvorfall sofort an datenschutz@firma.de melden</li></ul>',
    categoryId: 'wc2',
    status: 'published',
    authorId: 'c3',
    authorName: 'Thomas Weber',
    tags: ['dsgvo', 'datenschutz', 'compliance'],
    isPinned: false,
    viewCount: 203,
    lastEditedBy: 'Thomas Weber',
    lastEditedAt: '2026-01-30',
    createdAt: '2025-09-10',
    versions: [
      { id: 'wv13', articleId: 'wa7', version: 1, editorName: 'Thomas Weber', editedAt: '2025-09-10', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa8',
    title: 'VPN & Remote-Zugang einrichten',
    slug: 'vpn-einrichten',
    content: '<h2>VPN-Zugang</h2><p>Für den Zugriff auf interne Systeme von ausserhalb des Büros wird ein VPN benötigt.</p><h3>Einrichtung</h3><ol><li>WireGuard herunterladen (wireguard.com)</li><li>Konfiguration von IT anfordern (it-support@firma.de)</li><li>Config-Datei importieren</li><li>Verbinden und testen</li></ol><h3>Fehlerbehebung</h3><p>Bei Problemen: IT-Helpdesk Ticket erstellen mit Fehlermeldung und Screenshot.</p>',
    categoryId: 'wc3',
    status: 'published',
    authorId: 'c7',
    authorName: 'Felix Wagner',
    tags: ['it', 'vpn', 'remote'],
    isPinned: false,
    viewCount: 89,
    lastEditedBy: 'Felix Wagner',
    lastEditedAt: '2026-02-01',
    createdAt: '2025-10-20',
    versions: [
      { id: 'wv14', articleId: 'wa8', version: 1, editorName: 'Felix Wagner', editedAt: '2025-10-20', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa9',
    title: 'Cosmi — Tipps & Tricks',
    slug: 'kmuhub-tipps',
    content: '<h2>Tipps für den Alltag mit Cosmi</h2><h3>Tastenkürzel</h3><ul><li><kbd>Ctrl+K</kbd> — Globale Suche</li><li><kbd>Ctrl+N</kbd> — Neuer Eintrag</li><li><kbd>Ctrl+Shift+M</kbd> — Neues Meeting</li></ul><h3>Desk-Personalisierung</h3><p>Unter Einstellungen > Desk kannst du dein Theme, Hintergrundbild und Dekorationen anpassen.</p>',
    categoryId: 'wc3',
    status: 'published',
    authorId: 'c1',
    authorName: 'Anna Müller',
    tags: ['kmuhub', 'tipps', 'shortcuts'],
    isPinned: false,
    viewCount: 312,
    lastEditedBy: 'Anna Müller',
    lastEditedAt: '2026-02-15',
    createdAt: '2025-11-10',
    versions: [
      { id: 'wv15', articleId: 'wa9', version: 2, editorName: 'Anna Müller', editedAt: '2026-02-15', changeNote: 'Neue Shortcuts ergänzt' },
      { id: 'wv16', articleId: 'wa9', version: 1, editorName: 'Anna Müller', editedAt: '2025-11-10', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa10',
    title: 'Passwort-Richtlinien & 2FA',
    slug: 'passwort-richtlinien',
    content: '<h2>Passwort-Richtlinien</h2><ul><li>Mindestens 12 Zeichen</li><li>Gross-/Kleinbuchstaben + Zahl + Sonderzeichen</li><li>Kein Passwort-Recycling</li><li>Passwort-Manager empfohlen (Bitwarden)</li></ul><h3>Zwei-Faktor-Authentifizierung</h3><p>2FA ist Pflicht für alle Mitarbeiter. Einrichtung: Profil > Sicherheit > 2FA aktivieren.</p>',
    categoryId: 'wc3',
    status: 'published',
    authorId: 'c7',
    authorName: 'Felix Wagner',
    tags: ['sicherheit', 'passwort', '2fa'],
    isPinned: false,
    viewCount: 145,
    lastEditedBy: 'Felix Wagner',
    lastEditedAt: '2026-01-25',
    createdAt: '2025-09-20',
    versions: [
      { id: 'wv17', articleId: 'wa10', version: 1, editorName: 'Felix Wagner', editedAt: '2025-09-20', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa11',
    title: 'Onboarding-Checkliste neue Mitarbeiter',
    slug: 'onboarding-checkliste',
    content: '<h2>Onboarding-Checkliste</h2><h3>Tag 1</h3><ul><li>Begrüssung & Bürotour</li><li>IT-Equipment abholen</li><li>Cosmi Account einrichten</li><li>Team-Chat beitreten</li></ul><h3>Woche 1</h3><ul><li>Wiki durchlesen (diese Artikel!)</li><li>Prozessdokumentation lesen</li><li>Buddy-Gespräch</li></ul><h3>Monat 1</h3><ul><li>Selbständig erste Aufgaben bearbeiten</li><li>Feedback-Gespräch mit Teamlead</li></ul>',
    categoryId: 'wc4',
    status: 'published',
    authorId: 'c5',
    authorName: 'Lisa Braun',
    tags: ['onboarding', 'hr', 'checkliste'],
    isPinned: true,
    viewCount: 67,
    lastEditedBy: 'Lisa Braun',
    lastEditedAt: '2026-02-08',
    createdAt: '2025-12-01',
    versions: [
      { id: 'wv18', articleId: 'wa11', version: 1, editorName: 'Lisa Braun', editedAt: '2025-12-01', changeNote: 'Erstellt' },
    ],
  },
  {
    id: 'wa12',
    title: 'Abwesenheiten & Urlaubsanträge',
    slug: 'abwesenheiten',
    content: '<h2>Urlaubsanträge</h2><p>Urlaub wird über das Team-Modul beantragt.</p><h3>Ablauf</h3><ol><li>Team > Abwesenheiten > "Neuer Antrag"</li><li>Zeitraum und Vertretung auswählen</li><li>Teamlead genehmigt</li></ol><h3>Regeln</h3><ul><li>Mindestens 2 Wochen Vorlauf</li><li>Brückentage frühzeitig beantragen</li><li>Resturlaub bis 31. März Folgejahr nehmen</li></ul>',
    categoryId: 'wc4',
    status: 'published',
    authorId: 'c5',
    authorName: 'Lisa Braun',
    tags: ['hr', 'urlaub', 'abwesenheit'],
    isPinned: false,
    viewCount: 91,
    lastEditedBy: 'Lisa Braun',
    lastEditedAt: '2026-01-18',
    createdAt: '2025-11-15',
    versions: [
      { id: 'wv19', articleId: 'wa12', version: 1, editorName: 'Lisa Braun', editedAt: '2025-11-15', changeNote: 'Erstellt' },
    ],
  },
]

const mockTemplates: WikiTemplate[] = [
  {
    id: 'wt1',
    name: 'Meeting-Protokoll',
    description: 'Standardvorlage für Meeting-Notizen',
    content: '<h2>Meeting: [Titel]</h2><p><strong>Datum:</strong> [Datum]<br><strong>Teilnehmer:</strong> [Namen]</p><h3>Agenda</h3><ol><li>[Punkt 1]</li><li>[Punkt 2]</li></ol><h3>Beschlüsse</h3><ul><li>[Beschluss]</li></ul><h3>Nächste Schritte</h3><ul><li>[Aktion] — [Verantwortlich] — [Frist]</li></ul>',
    icon: 'FileText',
    category: 'meetings',
  },
  {
    id: 'wt2',
    name: 'Post-Mortem',
    description: 'Analyse nach einem Vorfall oder Problem',
    content: '<h2>Post-Mortem: [Vorfall]</h2><p><strong>Datum:</strong> [Datum]<br><strong>Schweregrad:</strong> [Kritisch/Hoch/Mittel]</p><h3>Zusammenfassung</h3><p>[Was ist passiert?]</p><h3>Timeline</h3><ul><li>[HH:MM] — [Ereignis]</li></ul><h3>Root Cause</h3><p>[Ursache]</p><h3>Massnahmen</h3><ul><li>[Massnahme] — [Verantwortlich] — [Frist]</li></ul><h3>Lessons Learned</h3><ul><li>[Erkenntnis]</li></ul>',
    icon: 'AlertTriangle',
    category: 'incidents',
  },
  {
    id: 'wt3',
    name: 'How-To Anleitung',
    description: 'Schritt-für-Schritt-Anleitung für Prozesse',
    content: '<h2>How-To: [Titel]</h2><p>[Kurzbeschreibung — was wird erreicht?]</p><h3>Voraussetzungen</h3><ul><li>[Voraussetzung 1]</li></ul><h3>Schritte</h3><ol><li>[Schritt 1]</li><li>[Schritt 2]</li><li>[Schritt 3]</li></ol><h3>Häufige Probleme</h3><ul><li><strong>Problem:</strong> [Beschreibung]<br><strong>Lösung:</strong> [Lösung]</li></ul>',
    icon: 'BookOpen',
    category: 'howto',
  },
]

// ---------------------------------------------------------------------------
// ID counter
// ---------------------------------------------------------------------------

let nextArticleId = 13
let nextCategoryId = 5
let nextVersionId = 20

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useWikiStore = create<WikiState>()(
  persist(
    (set, _get) => ({
      categories: mockCategories,
      articles: mockArticles,
      templates: mockTemplates,

      selectedArticleId: null,
      selectedCategoryId: null,
      isEditing: false,
      searchQuery: '',

      // -- Article actions --

      addArticle: (article) => {
        const id = `wa${nextArticleId++}`
        const slug = article.title
          .toLowerCase()
          .replace(/[äöü]/g, (c) => ({ ä: 'ae', ö: 'oe', ü: 'ue' })[c] || c)
          .replace(/[^a-z0-9]+/g, '-')
          .replace(/(^-|-$)/g, '')
        const now = new Date().toISOString().split('T')[0]
        set((state) => ({
          articles: [
            {
              ...article,
              id,
              slug,
              viewCount: 0,
              createdAt: now,
              versions: [
                {
                  id: `wv${nextVersionId++}`,
                  articleId: id,
                  version: 1,
                  editorName: article.authorName,
                  editedAt: now,
                  changeNote: 'Erstellt',
                },
              ],
            },
            ...state.articles,
          ],
          categories: state.categories.map((c) =>
            c.id === article.categoryId ? { ...c, articleCount: c.articleCount + 1 } : c,
          ),
        }))
      },

      updateArticle: (id, updates) => {
        const now = new Date().toISOString().split('T')[0]
        set((state) => ({
          articles: state.articles.map((a) =>
            a.id === id
              ? {
                  ...a,
                  ...updates,
                  lastEditedAt: now,
                  versions: [
                    {
                      id: `wv${nextVersionId++}`,
                      articleId: id,
                      version: a.versions.length + 1,
                      editorName: updates.lastEditedBy || a.lastEditedBy,
                      editedAt: now,
                      changeNote: 'Bearbeitet',
                    },
                    ...a.versions,
                  ],
                }
              : a,
          ),
        }))
      },

      deleteArticle: (id) =>
        set((state) => {
          const article = state.articles.find((a) => a.id === id)
          return {
            articles: state.articles.filter((a) => a.id !== id),
            categories: article
              ? state.categories.map((c) =>
                  c.id === article.categoryId
                    ? { ...c, articleCount: Math.max(0, c.articleCount - 1) }
                    : c,
                )
              : state.categories,
            selectedArticleId: state.selectedArticleId === id ? null : state.selectedArticleId,
          }
        }),

      publishArticle: (id) =>
        set((state) => ({
          articles: state.articles.map((a) => (a.id === id ? { ...a, status: 'published' as const } : a)),
        })),

      archiveArticle: (id) =>
        set((state) => ({
          articles: state.articles.map((a) => (a.id === id ? { ...a, status: 'archived' as const } : a)),
        })),

      togglePin: (id) =>
        set((state) => ({
          articles: state.articles.map((a) => (a.id === id ? { ...a, isPinned: !a.isPinned } : a)),
        })),

      incrementViewCount: (id) =>
        set((state) => ({
          articles: state.articles.map((a) => (a.id === id ? { ...a, viewCount: a.viewCount + 1 } : a)),
        })),

      // -- Category actions --

      addCategory: (category) => {
        const id = `wc${nextCategoryId++}`
        set((state) => ({
          categories: [...state.categories, { ...category, id, articleCount: 0 }],
        }))
      },

      updateCategory: (id, updates) =>
        set((state) => ({
          categories: state.categories.map((c) => (c.id === id ? { ...c, ...updates } : c)),
        })),

      deleteCategory: (id) =>
        set((state) => ({
          categories: state.categories.filter((c) => c.id !== id),
          selectedCategoryId: state.selectedCategoryId === id ? null : state.selectedCategoryId,
        })),

      // -- UI actions --

      setSelectedArticle: (id) => set({ selectedArticleId: id }),
      setSelectedCategory: (id) => set({ selectedCategoryId: id }),
      setEditing: (editing) => set({ isEditing: editing }),
      setSearchQuery: (query) => set({ searchQuery: query }),
    }),
    {
      name: 'kmuhub-wiki',
      partialize: (state) => ({
        selectedCategoryId: state.selectedCategoryId,
      }),
    },
  ),
)
