import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type FormFieldType = 'text' | 'textarea' | 'select' | 'checkbox' | 'radio' | 'date' | 'number' | 'rating' | 'file'

export interface FormField {
  id: string
  type: FormFieldType
  label: string
  required: boolean
  placeholder?: string
  options?: string[] // for select, radio
  conditionalLogic?: { fieldId: string; operator: 'equals' | 'not_equals' | 'contains'; value: string }
  page?: number
}

export interface Form {
  id: string
  name: string
  description: string
  status: 'active' | 'draft' | 'archived'
  fields: FormField[]
  createdAt: string
  updatedAt: string
  createdBy: string
  submissionCount: number
  isTemplate: boolean
  pageCount?: number
  actions?: { type: 'email' | 'task' | 'crm_contact'; config: Record<string, string> }[]
  isPublic?: boolean
}

export interface FormSubmission {
  id: string
  formId: string
  formName: string
  submittedAt: string
  submittedBy: string
  status: 'new' | 'read' | 'archived'
  answers: Record<string, string | string[] | number | boolean>
}

// ---------------------------------------------------------------------------
// Mock Forms
// ---------------------------------------------------------------------------

const MOCK_FORMS: Form[] = [
  {
    id: 'form-1',
    name: 'Kundenzufriedenheit',
    description: 'Umfrage zur Zufriedenheit unserer Kunden mit unseren Dienstleistungen und Produkten.',
    status: 'active',
    createdAt: '2026-01-15T09:00:00',
    updatedAt: '2026-02-10T14:30:00',
    createdBy: 'Anna Müller',
    submissionCount: 8,
    isTemplate: false,
    pageCount: 1,
    actions: [],
    isPublic: false,
    fields: [
      { id: 'f1-1', type: 'text', label: 'Name', required: true, placeholder: 'Ihr vollstaendiger Name' },
      { id: 'f1-2', type: 'text', label: 'E-Mail', required: true, placeholder: 'ihre.email@beispiel.ch' },
      { id: 'f1-3', type: 'rating', label: 'Bewertung', required: true },
      { id: 'f1-4', type: 'textarea', label: 'Was hat Ihnen gefallen?', required: false, placeholder: 'Erzaehlen Sie uns, was gut war...' },
      { id: 'f1-5', type: 'textarea', label: 'Verbesserungsvorschlaege', required: false, placeholder: 'Was könnten wir besser machen?' },
      { id: 'f1-6', type: 'radio', label: 'Weiterempfehlung', required: true, options: ['Ja', 'Nein', 'Vielleicht'] },
    ],
  },
  {
    id: 'form-2',
    name: 'Schadensmeldung',
    description: 'Formular zur Erfassung und Dokumentation von Schadensfaellen im Betrieb.',
    status: 'active',
    createdAt: '2026-01-20T11:00:00',
    updatedAt: '2026-02-08T16:15:00',
    createdBy: 'Peter Koch',
    submissionCount: 4,
    isTemplate: false,
    pageCount: 1,
    actions: [],
    isPublic: false,
    fields: [
      { id: 'f2-1', type: 'text', label: 'Melder', required: true, placeholder: 'Name der meldenden Person' },
      { id: 'f2-2', type: 'date', label: 'Datum', required: true },
      { id: 'f2-3', type: 'text', label: 'Ort', required: true, placeholder: 'Wo ist der Schaden aufgetreten?' },
      { id: 'f2-4', type: 'textarea', label: 'Beschreibung', required: true, placeholder: 'Beschreiben Sie den Schaden detailliert...', conditionalLogic: { fieldId: 'f2-5', operator: 'not_equals', value: '' } },
      { id: 'f2-5', type: 'select', label: 'Schwere', required: true, options: ['Gering', 'Mittel', 'Schwer', 'Kritisch'] },
      { id: 'f2-6', type: 'file', label: 'Fotos', required: false },
      { id: 'f2-7', type: 'textarea', label: 'Sofortmassnahmen', required: false, placeholder: 'Welche Massnahmen wurden bereits ergriffen?' },
    ],
  },
  {
    id: 'form-3',
    name: 'Bewerbungsformular',
    description: 'Online-Bewerbungsformular für offene Stellen in unserem Unternehmen.',
    status: 'active',
    createdAt: '2026-01-25T08:30:00',
    updatedAt: '2026-02-12T10:00:00',
    createdBy: 'Lisa Schmidt',
    submissionCount: 3,
    isTemplate: false,
    pageCount: 1,
    isPublic: false,
    fields: [
      { id: 'f3-1', type: 'text', label: 'Name', required: true, placeholder: 'Vor- und Nachname' },
      { id: 'f3-2', type: 'text', label: 'E-Mail', required: true, placeholder: 'ihre.email@beispiel.ch' },
      { id: 'f3-3', type: 'select', label: 'Position', required: true, options: ['Entwickler', 'Designer', 'Marketing', 'Vertrieb', 'HR'] },
      { id: 'f3-4', type: 'number', label: 'Erfahrung (Jahre)', required: true },
      { id: 'f3-5', type: 'textarea', label: 'Motivation', required: true, placeholder: 'Warum moechten Sie bei uns arbeiten?' },
      { id: 'f3-6', type: 'file', label: 'Lebenslauf', required: true },
    ],
  },
  {
    id: 'form-4',
    name: 'Inventur-Checkliste',
    description: 'Checkliste für die jaehrliche Inventur aller Lagerbestaende und Betriebsmittel.',
    status: 'draft',
    createdAt: '2026-02-01T13:00:00',
    updatedAt: '2026-02-01T13:00:00',
    createdBy: 'Michael Berg',
    submissionCount: 0,
    isTemplate: false,
    pageCount: 1,
    isPublic: false,
    fields: [
      { id: 'f4-1', type: 'text', label: 'Bereich', required: true, placeholder: 'Lager, Büro, Werkstatt...' },
      { id: 'f4-2', type: 'number', label: 'Artikel-Anzahl', required: true },
      { id: 'f4-3', type: 'textarea', label: 'Abweichungen', required: false, placeholder: 'Festgestellte Abweichungen zum Soll-Bestand' },
      { id: 'f4-4', type: 'select', label: 'Zustand', required: true, options: ['Gut', 'Beschaedigt', 'Fehlend'] },
      { id: 'f4-5', type: 'text', label: 'Prüfer', required: true, placeholder: 'Name des Prüfers' },
    ],
  },
  {
    id: 'form-5',
    name: 'Mitarbeiter-Feedback',
    description: 'Anonymes Feedbackformular für Mitarbeiter zur Verbesserung der Arbeitskultur.',
    status: 'active',
    createdAt: '2026-01-28T10:00:00',
    updatedAt: '2026-02-14T09:00:00',
    createdBy: 'Lisa Schmidt',
    submissionCount: 5,
    isTemplate: false,
    pageCount: 1,
    isPublic: false,
    fields: [
      { id: 'f5-1', type: 'select', label: 'Abteilung', required: true, options: ['IT', 'HR', 'Marketing', 'Vertrieb', 'Produktion'] },
      { id: 'f5-2', type: 'rating', label: 'Zufriedenheit', required: true },
      { id: 'f5-3', type: 'textarea', label: 'Stärken', required: false, placeholder: 'Was laeuft gut im Unternehmen?' },
      { id: 'f5-4', type: 'textarea', label: 'Verbesserungen', required: false, placeholder: 'Was könnte verbessert werden?' },
      { id: 'f5-5', type: 'checkbox', label: 'Anonym', required: false },
    ],
  },
]

// ---------------------------------------------------------------------------
// Mock Templates
// ---------------------------------------------------------------------------

const MOCK_TEMPLATES: Form[] = [
  {
    id: 'tmpl-1',
    name: 'Feedback-Vorlage',
    description: 'Allgemeine Vorlage für Feedback-Formulare mit Bewertung und Freitextfeldern.',
    status: 'active',
    createdAt: '2026-01-10T08:00:00',
    updatedAt: '2026-01-10T08:00:00',
    createdBy: 'System',
    submissionCount: 0,
    isTemplate: true,
    fields: [
      { id: 'tmpl1-1', type: 'text', label: 'Name', required: false, placeholder: 'Optional' },
      { id: 'tmpl1-2', type: 'rating', label: 'Gesamtbewertung', required: true },
      { id: 'tmpl1-3', type: 'textarea', label: 'Kommentar', required: false, placeholder: 'Ihr Feedback...' },
      { id: 'tmpl1-4', type: 'radio', label: 'Weiterempfehlung', required: false, options: ['Ja', 'Nein', 'Vielleicht'] },
    ],
  },
  {
    id: 'tmpl-2',
    name: 'Checkliste-Vorlage',
    description: 'Einfache Checkliste mit Bereich, Status und Anmerkungen.',
    status: 'active',
    createdAt: '2026-01-10T08:00:00',
    updatedAt: '2026-01-10T08:00:00',
    createdBy: 'System',
    submissionCount: 0,
    isTemplate: true,
    fields: [
      { id: 'tmpl2-1', type: 'text', label: 'Bereich', required: true, placeholder: 'Zu prüfender Bereich' },
      { id: 'tmpl2-2', type: 'select', label: 'Status', required: true, options: ['Offen', 'In Bearbeitung', 'Erledigt'] },
      { id: 'tmpl2-3', type: 'textarea', label: 'Anmerkungen', required: false, placeholder: 'Zusaetzliche Anmerkungen...' },
    ],
  },
  {
    id: 'tmpl-3',
    name: 'Anmeldeformular-Vorlage',
    description: 'Vorlage für Event-Anmeldungen mit persönlichen Daten und Praeferenzen.',
    status: 'active',
    createdAt: '2026-01-10T08:00:00',
    updatedAt: '2026-01-10T08:00:00',
    createdBy: 'System',
    submissionCount: 0,
    isTemplate: true,
    fields: [
      { id: 'tmpl3-1', type: 'text', label: 'Vorname', required: true, placeholder: 'Ihr Vorname' },
      { id: 'tmpl3-2', type: 'text', label: 'Nachname', required: true, placeholder: 'Ihr Nachname' },
      { id: 'tmpl3-3', type: 'text', label: 'E-Mail', required: true, placeholder: 'ihre.email@beispiel.ch' },
      { id: 'tmpl3-4', type: 'select', label: 'Teilnahme', required: true, options: ['Vor Ort', 'Online', 'Unsicher'] },
      { id: 'tmpl3-5', type: 'textarea', label: 'Besondere Wünsche', required: false, placeholder: 'Allergien, Barrierefreiheit etc.' },
    ],
  },
]

// ---------------------------------------------------------------------------
// Mock Submissions
// ---------------------------------------------------------------------------

const MOCK_SUBMISSIONS: FormSubmission[] = [
  // Kundenzufriedenheit (form-1) — 8 submissions
  { id: 'sub-1', formId: 'form-1', formName: 'Kundenzufriedenheit', submittedAt: '2026-02-14T10:30:00', submittedBy: 'Hans Meier', status: 'new', answers: { 'f1-1': 'Hans Meier', 'f1-2': 'hans.meier@example.ch', 'f1-3': 5, 'f1-4': 'Sehr schnelle Bearbeitung und freundlicher Kundenservice.', 'f1-5': '', 'f1-6': 'Ja' } },
  { id: 'sub-2', formId: 'form-1', formName: 'Kundenzufriedenheit', submittedAt: '2026-02-13T16:45:00', submittedBy: 'Klara Fischer', status: 'new', answers: { 'f1-1': 'Klara Fischer', 'f1-2': 'k.fischer@bluewin.ch', 'f1-3': 4, 'f1-4': 'Gute Produktqualitaet.', 'f1-5': 'Lieferzeiten könnten kuerzer sein.', 'f1-6': 'Ja' } },
  { id: 'sub-3', formId: 'form-1', formName: 'Kundenzufriedenheit', submittedAt: '2026-02-12T09:15:00', submittedBy: 'Thomas Weber', status: 'read', answers: { 'f1-1': 'Thomas Weber', 'f1-2': 'weber@firma.de', 'f1-3': 3, 'f1-4': 'Ordentliche Leistung insgesamt.', 'f1-5': 'Bessere Kommunikation bei Verzoegerungen.', 'f1-6': 'Vielleicht' } },
  { id: 'sub-4', formId: 'form-1', formName: 'Kundenzufriedenheit', submittedAt: '2026-02-11T14:00:00', submittedBy: 'Susanne Baumann', status: 'read', answers: { 'f1-1': 'Susanne Baumann', 'f1-2': 's.baumann@gmx.ch', 'f1-3': 5, 'f1-4': 'Alles bestens, bin sehr zufrieden!', 'f1-5': '', 'f1-6': 'Ja' } },
  { id: 'sub-5', formId: 'form-1', formName: 'Kundenzufriedenheit', submittedAt: '2026-02-10T11:30:00', submittedBy: 'Reto Brunner', status: 'read', answers: { 'f1-1': 'Reto Brunner', 'f1-2': 'r.brunner@swisscom.ch', 'f1-3': 4, 'f1-4': 'Preis-Leistung stimmt.', 'f1-5': 'Online-Portal könnte moderner sein.', 'f1-6': 'Ja' } },
  { id: 'sub-6', formId: 'form-1', formName: 'Kundenzufriedenheit', submittedAt: '2026-02-08T08:00:00', submittedBy: 'Maria Keller', status: 'archived', answers: { 'f1-1': 'Maria Keller', 'f1-2': 'm.keller@sunrise.ch', 'f1-3': 2, 'f1-4': '', 'f1-5': 'Reklamation wurde nicht zufriedenstellend bearbeitet.', 'f1-6': 'Nein' } },
  { id: 'sub-7', formId: 'form-1', formName: 'Kundenzufriedenheit', submittedAt: '2026-02-06T15:20:00', submittedBy: 'Andreas Huber', status: 'archived', answers: { 'f1-1': 'Andreas Huber', 'f1-2': 'a.huber@outlook.ch', 'f1-3': 4, 'f1-4': 'Gute Beratung vor Ort.', 'f1-5': 'Mehr Parkplaetze waeren toll.', 'f1-6': 'Ja' } },
  { id: 'sub-8', formId: 'form-1', formName: 'Kundenzufriedenheit', submittedAt: '2026-02-04T12:00:00', submittedBy: 'Eva Schmid', status: 'archived', answers: { 'f1-1': 'Eva Schmid', 'f1-2': 'e.schmid@gmail.com', 'f1-3': 3, 'f1-4': 'Insgesamt okay.', 'f1-5': 'Wartezeiten am Telefon reduzieren.', 'f1-6': 'Vielleicht' } },

  // Schadensmeldung (form-2) — 4 submissions
  { id: 'sub-9', formId: 'form-2', formName: 'Schadensmeldung', submittedAt: '2026-02-13T08:45:00', submittedBy: 'Michael Berg', status: 'new', answers: { 'f2-1': 'Michael Berg', 'f2-2': '2026-02-13', 'f2-3': 'Lager Halle B', 'f2-4': 'Wasserschaden an der Decke, Tropfbildung ueber Regal 4.', 'f2-5': 'Mittel', 'f2-6': 'foto_schaden_01.jpg', 'f2-7': 'Eimer aufgestellt, Bereich abgesperrt.' } },
  { id: 'sub-10', formId: 'form-2', formName: 'Schadensmeldung', submittedAt: '2026-02-10T14:20:00', submittedBy: 'Jonas Diaz', status: 'read', answers: { 'f2-1': 'Jonas Diaz', 'f2-2': '2026-02-10', 'f2-3': 'Parkplatz Einfahrt', 'f2-4': 'Schranke defekt, schliesst nicht mehr vollstaendig.', 'f2-5': 'Gering', 'f2-6': '', 'f2-7': 'Schranke manuell gesichert.' } },
  { id: 'sub-11', formId: 'form-2', formName: 'Schadensmeldung', submittedAt: '2026-02-07T09:00:00', submittedBy: 'Sarah Klein', status: 'read', answers: { 'f2-1': 'Sarah Klein', 'f2-2': '2026-02-07', 'f2-3': 'Büro 2. OG', 'f2-4': 'Fensterscheibe gesprungen, vermutlich durch Temperaturschwankung.', 'f2-5': 'Schwer', 'f2-6': 'fenster_schaden.jpg', 'f2-7': 'Fenster mit Folie abgedichtet, Handwerker benachrichtigt.' } },
  { id: 'sub-12', formId: 'form-2', formName: 'Schadensmeldung', submittedAt: '2026-02-03T16:30:00', submittedBy: 'Peter Koch', status: 'archived', answers: { 'f2-1': 'Peter Koch', 'f2-2': '2026-02-03', 'f2-3': 'Werkstatt', 'f2-4': 'Druckluftleitung undicht an Verbindungsstück.', 'f2-5': 'Mittel', 'f2-6': '', 'f2-7': 'Leitung provisorisch abgedichtet.' } },

  // Bewerbungsformular (form-3) — 3 submissions
  { id: 'sub-13', formId: 'form-3', formName: 'Bewerbungsformular', submittedAt: '2026-02-12T11:00:00', submittedBy: 'Laura Steiner', status: 'new', answers: { 'f3-1': 'Laura Steiner', 'f3-2': 'l.steiner@gmail.com', 'f3-3': 'Designer', 'f3-4': 4, 'f3-5': 'Ich bringe Erfahrung in UX/UI Design mit und moechte in einem innovativen KMU mitgestalten.', 'f3-6': 'lebenslauf_steiner.pdf' } },
  { id: 'sub-14', formId: 'form-3', formName: 'Bewerbungsformular', submittedAt: '2026-02-09T09:30:00', submittedBy: 'Marco Frei', status: 'read', answers: { 'f3-1': 'Marco Frei', 'f3-2': 'm.frei@protonmail.ch', 'f3-3': 'Entwickler', 'f3-4': 7, 'f3-5': 'Als Full-Stack-Entwickler mit Go- und React-Erfahrung suche ich eine neue Herausforderung.', 'f3-6': 'cv_frei_marco.pdf' } },
  { id: 'sub-15', formId: 'form-3', formName: 'Bewerbungsformular', submittedAt: '2026-02-05T14:15:00', submittedBy: 'Nina Hofer', status: 'read', answers: { 'f3-1': 'Nina Hofer', 'f3-2': 'n.hofer@outlook.ch', 'f3-3': 'Marketing', 'f3-4': 3, 'f3-5': 'Ich habe Erfahrung im digitalen Marketing und moechte den DACH-Markt aktiv mitgestalten.', 'f3-6': 'bewerbung_hofer.pdf' } },

  // Mitarbeiter-Feedback (form-5) — 5 submissions (no submitter names due to anonymity)
  { id: 'sub-16', formId: 'form-5', formName: 'Mitarbeiter-Feedback', submittedAt: '2026-02-14T08:00:00', submittedBy: 'Anonym', status: 'new', answers: { 'f5-1': 'IT', 'f5-2': 4, 'f5-3': 'Gute Teamkultur und moderne Arbeitsmittel.', 'f5-4': 'Mehr Budget für Weiterbildungen.', 'f5-5': true } },
  { id: 'sub-17', formId: 'form-5', formName: 'Mitarbeiter-Feedback', submittedAt: '2026-02-13T12:00:00', submittedBy: 'Anonym', status: 'new', answers: { 'f5-1': 'Vertrieb', 'f5-2': 3, 'f5-3': 'Kollegiales Miteinander.', 'f5-4': 'Klarere Zielvorgaben und regelmaessigere Meetings.', 'f5-5': true } },
  { id: 'sub-18', formId: 'form-5', formName: 'Mitarbeiter-Feedback', submittedAt: '2026-02-11T10:00:00', submittedBy: 'Anonym', status: 'read', answers: { 'f5-1': 'HR', 'f5-2': 5, 'f5-3': 'Sehr gute Work-Life-Balance und flexible Arbeitszeiten.', 'f5-4': 'Kantine waere ein Plus.', 'f5-5': true } },
  { id: 'sub-19', formId: 'form-5', formName: 'Mitarbeiter-Feedback', submittedAt: '2026-02-09T15:30:00', submittedBy: 'Anonym', status: 'read', answers: { 'f5-1': 'Produktion', 'f5-2': 2, 'f5-3': 'Stabiler Arbeitsplatz.', 'f5-4': 'Arbeitsbelastung besser verteilen, mehr Personal einstellen.', 'f5-5': true } },
  { id: 'sub-20', formId: 'form-5', formName: 'Mitarbeiter-Feedback', submittedAt: '2026-02-07T11:00:00', submittedBy: 'Anonym', status: 'archived', answers: { 'f5-1': 'Marketing', 'f5-2': 4, 'f5-3': 'Kreative Freiheit und spannende Projekte.', 'f5-4': 'Bessere Tools für Social Media Management.', 'f5-5': true } },
]

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

let nextFormId = 6
let nextFieldId = 100
let nextSubmissionId = 21

interface FormulareState {
  forms: Form[]
  templates: Form[]
  submissions: FormSubmission[]

  addForm: (form: Omit<Form, 'id' | 'createdAt' | 'updatedAt' | 'submissionCount'>) => void
  updateForm: (id: string, updates: Partial<Omit<Form, 'id'>>) => void
  deleteForm: (id: string) => void
  duplicateForm: (id: string) => void
  addSubmission: (submission: Omit<FormSubmission, 'id'>) => void
  updateSubmissionStatus: (id: string, status: FormSubmission['status']) => void
  addField: (formId: string, field: Omit<FormField, 'id'>) => void
  removeField: (formId: string, fieldId: string) => void
  reorderFields: (formId: string, fields: FormField[]) => void
}

export const useFormulareStore = create<FormulareState>()(
  persist(
    (set) => ({
      forms: MOCK_FORMS,
      templates: MOCK_TEMPLATES,
      submissions: MOCK_SUBMISSIONS,

      addForm: (form) => {
        const now = new Date().toISOString()
        set((state) => ({
          forms: [
            {
              ...form,
              id: `form-${nextFormId++}`,
              createdAt: now,
              updatedAt: now,
              submissionCount: 0,
            },
            ...state.forms,
          ],
        }))
      },

      updateForm: (id, updates) =>
        set((state) => ({
          forms: state.forms.map((f) =>
            f.id === id ? { ...f, ...updates, updatedAt: new Date().toISOString() } : f
          ),
        })),

      deleteForm: (id) =>
        set((state) => ({
          forms: state.forms.filter((f) => f.id !== id),
          submissions: state.submissions.filter((s) => s.formId !== id),
        })),

      duplicateForm: (id) =>
        set((state) => {
          const source = [...state.forms, ...state.templates].find((f) => f.id === id)
          if (!source) return state
          const now = new Date().toISOString()
          const newFields = source.fields.map((field) => ({
            ...field,
            id: `field-${nextFieldId++}`,
          }))
          return {
            forms: [
              {
                ...source,
                id: `form-${nextFormId++}`,
                name: `${source.name} (Kopie)`,
                status: 'draft' as const,
                createdAt: now,
                updatedAt: now,
                submissionCount: 0,
                isTemplate: false,
                fields: newFields,
              },
              ...state.forms,
            ],
          }
        }),

      addSubmission: (submission) =>
        set((state) => ({
          submissions: [
            { ...submission, id: `sub-${nextSubmissionId++}` },
            ...state.submissions,
          ],
          forms: state.forms.map((f) =>
            f.id === submission.formId
              ? { ...f, submissionCount: f.submissionCount + 1 }
              : f
          ),
        })),

      updateSubmissionStatus: (id, status) =>
        set((state) => ({
          submissions: state.submissions.map((s) =>
            s.id === id ? { ...s, status } : s
          ),
        })),

      addField: (formId, field) =>
        set((state) => ({
          forms: state.forms.map((f) =>
            f.id === formId
              ? {
                  ...f,
                  fields: [...f.fields, { ...field, id: `field-${nextFieldId++}` }],
                  updatedAt: new Date().toISOString(),
                }
              : f
          ),
        })),

      removeField: (formId, fieldId) =>
        set((state) => ({
          forms: state.forms.map((f) =>
            f.id === formId
              ? {
                  ...f,
                  fields: f.fields.filter((fld) => fld.id !== fieldId),
                  updatedAt: new Date().toISOString(),
                }
              : f
          ),
        })),

      reorderFields: (formId, fields) =>
        set((state) => ({
          forms: state.forms.map((f) =>
            f.id === formId
              ? { ...f, fields, updatedAt: new Date().toISOString() }
              : f
          ),
        })),
    }),
    { name: 'kmuhub-formulare' }
  )
)
