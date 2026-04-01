/**
 * Tour system store.
 *
 * Manages interactive guided tours that highlight UI elements with explanations.
 * Tours are manually startable from "Über Cosmi" or per-module help menus.
 */
import { create } from 'zustand'

export interface TourStep {
  /** CSS selector for the element to highlight (null = centered modal) */
  target: string | null
  title: string
  description: string
  /** Which side of the element to show the popover */
  placement?: 'top' | 'bottom' | 'left' | 'right'
}

export interface TourDefinition {
  id: string
  name: string
  description: string
  /** Module id this tour belongs to (null = general) */
  moduleId: string | null
  steps: TourStep[]
}

interface TourState {
  /** Currently active tour (null if no tour running) */
  activeTour: TourDefinition | null
  /** Current step index */
  currentStep: number
  /** All available tours */
  tours: TourDefinition[]

  startTour: (tourId: string) => void
  nextStep: () => void
  prevStep: () => void
  skipTour: () => void
  goToStep: (index: number) => void
}

// ---------------------------------------------------------------------------
// Tour definitions
// ---------------------------------------------------------------------------

const TOURS: TourDefinition[] = [
  {
    id: 'general',
    name: 'Cosmi kennenlernen',
    description: 'Kurze Einführung in die wichtigsten Bereiche der App',
    moduleId: null,
    steps: [
      {
        target: '[data-tour="sidebar"]',
        title: 'Navigation',
        description: 'Hier findest du alle Module. Klicke auf ein Modul um es zu öffnen. Du kannst die Sidebar auch einklappen.',
        placement: 'right',
      },
      {
        target: '[data-tour="header"]',
        title: 'Kopfleiste',
        description: 'Hier siehst du die Suchleiste, deine Widgets und den Zeiterfassungs-Button. Die Widgets kannst du in den Einstellungen anpassen.',
        placement: 'bottom',
      },
      {
        target: '[data-tour="time-tracker"]',
        title: 'Zeiterfassung',
        description: 'Stempel dich ein, tracke Aufgaben und erfasse Pausen — alles an einem Ort.',
        placement: 'bottom',
      },
      {
        target: null,
        title: 'Module',
        description: 'Cosmi bietet Module für CRM, Projekte, Aufgaben, Kalender, Buchhaltung, Team-Management und mehr. Jedes Modul hat eigene Einstellungen.',
      },
      {
        target: null,
        title: 'Einstellungen',
        description: 'Unter Einstellungen kannst du Farben, Sprache, Sicherheit und vieles mehr anpassen. Modul-spezifische Einstellungen findest du direkt im jeweiligen Modul.',
      },
    ],
  },
  {
    id: 'dashboard',
    name: 'Dashboard',
    description: 'Überblick ueber dein persönliches Dashboard',
    moduleId: 'dashboard',
    steps: [
      {
        target: null,
        title: 'Dein Dashboard',
        description: 'Das Dashboard zeigt dir auf einen Blick deine wichtigsten KPIs, anstehende Termine, offene Aufgaben und Team-Aktivitaeten.',
      },
      {
        target: null,
        title: 'Widgets',
        description: 'Jedes Widget zeigt Echtzeit-Daten aus den jeweiligen Modulen. Du kannst ueber die Einstellungen festlegen, welche Header-Widgets angezeigt werden.',
      },
    ],
  },
  {
    id: 'team',
    name: 'Team & HR',
    description: 'Mitarbeiter verwalten, Abwesenheiten, Schulungen',
    moduleId: 'team',
    steps: [
      {
        target: null,
        title: 'Team-Übersicht',
        description: 'Hier siehst du alle Mitarbeiter nach Abteilung. Du kannst zwischen Karten- und Listenansicht wechseln.',
      },
      {
        target: null,
        title: 'Anfragen & Abwesenheiten',
        description: 'Urlaubsantraege und Krankmeldungen können hier genehmigt oder abgelehnt werden. Der Abwesenheitskalender zeigt dir wer wann fehlt.',
      },
      {
        target: null,
        title: 'Einstellungen',
        description: 'Im Tab "Einstellungen" kannst du Abteilungen, Rollen, Abwesenheitsarten und Arbeitszeitregeln verwalten.',
      },
    ],
  },
  {
    id: 'finance',
    name: 'Buchhaltung',
    description: 'Rechnungen, Angebote und Finanzen',
    moduleId: 'finance',
    steps: [
      {
        target: null,
        title: 'Finanz-Übersicht',
        description: 'Verwalte Rechnungen, Angebote und Ausgaben. Die KPI-Karten oben zeigen dir den aktuellen Stand.',
      },
      {
        target: null,
        title: 'Export & Berichte',
        description: 'Du kannst Daten als CSV/PDF exportieren und mit DATEV synchronisieren.',
      },
    ],
  },
  {
    id: 'kommunikation',
    name: 'Kommunikation',
    description: 'Unified Inbox für alle Kanäle',
    moduleId: 'kommunikation',
    steps: [
      {
        target: null,
        title: 'Unified Inbox',
        description: 'Alle Kundenanfragen aus E-Mail, WhatsApp, Teams, Portal und Widget landen hier in einer zentralen Inbox.',
      },
      {
        target: null,
        title: 'Konversationen',
        description: 'Links wähle eine Konversation, in der Mitte siehst du den Nachrichtenverlauf, rechts den CRM-Kontext des Kontakts.',
      },
    ],
  },
]

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useTourStore = create<TourState>()((set, get) => ({
  activeTour: null,
  currentStep: 0,
  tours: TOURS,

  startTour: (tourId) => {
    const tour = get().tours.find((t) => t.id === tourId)
    if (tour) set({ activeTour: tour, currentStep: 0 })
  },

  nextStep: () => {
    const { activeTour, currentStep } = get()
    if (!activeTour) return
    if (currentStep < activeTour.steps.length - 1) {
      set({ currentStep: currentStep + 1 })
    } else {
      set({ activeTour: null, currentStep: 0 })
    }
  },

  prevStep: () => {
    const { currentStep } = get()
    if (currentStep > 0) set({ currentStep: currentStep - 1 })
  },

  skipTour: () => set({ activeTour: null, currentStep: 0 }),

  goToStep: (index) => {
    const { activeTour } = get()
    if (activeTour && index >= 0 && index < activeTour.steps.length) {
      set({ currentStep: index })
    }
  },
}))
