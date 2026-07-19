import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type WeatherType = 'sunny' | 'cloudy' | 'rainy' | 'snowy'

export interface ReportWorker {
  name: string
  role: string
  hours: number
}

export interface ReportActivity {
  description: string
  category: string
}

export interface ReportMaterial {
  article: string
  quantity: number
  unit: string
}

export interface FieldReport {
  id: string
  date: string
  projectId: string
  projectName: string
  author: string
  /** Raw user id for scope-own checks (API reports only; legacy seeds omit it). */
  authorId?: string
  weather: WeatherType
  temperature: number
  workStart: string
  workEnd: string
  breakMinutes: number
  workers: ReportWorker[]
  activities: ReportActivity[]
  materials: ReportMaterial[]
  photos: { id: string; caption: string; thumbnailUrl?: string }[]
  notes: string
  signatureStatus: 'pending' | 'signed'
  signatureDataUrl?: string
  approvalStatus: 'draft' | 'submitted' | 'approved' | 'rejected'
  approvalComment?: string
  approvedBy?: string
}

export interface MeasurementPosition {
  id: string
  label: string
  length: number
  width: number
  height: number
  area: number
  volume: number
}

export interface Measurement {
  id: string
  name: string
  projectName: string
  positions: MeasurementPosition[]
  createdAt: string
  author: string
}

export interface ReportTemplate {
  id: string
  name: string
  description: string
  defaultActivities: string[]
  defaultMaterials: string[]
}

// ---------------------------------------------------------------------------
// Mock Data
// ---------------------------------------------------------------------------

const MOCK_REPORTS: FieldReport[] = [
  {
    id: 'rpt-1',
    date: '2026-02-14',
    projectId: 'prj-1',
    projectName: 'Rohbau Mehrfamilienhaus Zürich',
    author: 'Marco Brunner',
    weather: 'sunny',
    temperature: 8,
    workStart: '07:00',
    workEnd: '17:00',
    breakMinutes: 60,
    workers: [
      { name: 'Marco Brunner', role: 'Polier', hours: 9 },
      { name: 'Luca Meier', role: 'Maurer', hours: 9 },
      { name: 'Jonas Widmer', role: 'Maurer', hours: 9 },
      { name: 'Patrick Steiner', role: 'Eisenleger', hours: 9 },
      { name: 'David Keller', role: 'Hilfsarbeiter', hours: 9 },
      { name: 'Fabio Huber', role: 'Hilfsarbeiter', hours: 9 },
    ],
    activities: [
      { description: 'Schalung 2. OG Decke Achse A-C aufgestellt', category: 'Schalung' },
      { description: 'Bewehrung Decke 2. OG verlegt und abgenommen', category: 'Bewehrung' },
      { description: 'Betonieren Decke 1. OG Nachbehandlung', category: 'Betonarbeiten' },
      { description: 'Baustelle aufgeräumt, Material sortiert', category: 'Aufräumen' },
    ],
    materials: [
      { article: 'Beton C25/30', quantity: 12, unit: 'm³' },
      { article: 'Bewehrungsstahl B500B', quantity: 2.4, unit: 't' },
      { article: 'Schalungsplatten 27mm', quantity: 45, unit: 'm²' },
    ],
    photos: [
      { id: 'ph-1', caption: 'Bewehrung Decke 2. OG vor Betonage' },
      { id: 'ph-2', caption: 'Schalung Achse A-C fertig' },
    ],
    notes: 'Betonpumpe ab 13:00 im Einsatz. Nächster Guss geplant für Mittwoch.',
    signatureStatus: 'signed',
    approvalStatus: 'approved',
    approvedBy: 'Peter Widmer',
  },
  {
    id: 'rpt-2',
    date: '2026-02-13',
    projectId: 'prj-2',
    projectName: 'Fassadensanierung Bern',
    author: 'Stefan Gerber',
    weather: 'cloudy',
    temperature: 4,
    workStart: '07:30',
    workEnd: '16:30',
    breakMinutes: 45,
    workers: [
      { name: 'Stefan Gerber', role: 'Vorarbeiter', hours: 8 },
      { name: 'Reto Bühler', role: 'Fassadenmonteur', hours: 8 },
      { name: 'Simon Aebi', role: 'Fassadenmonteur', hours: 8 },
      { name: 'Kevin Wyss', role: 'Hilfsarbeiter', hours: 8 },
    ],
    activities: [
      { description: 'Alte Fassadenverkleidung Südseite demontiert', category: 'Rückbau' },
      { description: 'Dämmplatten EPS 160mm montiert (32 m²)', category: 'Dämmung' },
      { description: 'Armierungsmörtel auf Westseite aufgetragen', category: 'Verputz' },
    ],
    materials: [
      { article: 'EPS Dämmplatten 160mm', quantity: 32, unit: 'm²' },
      { article: 'Armierungsmörtel', quantity: 12, unit: 'Sack' },
      { article: 'Armierungsgewebe 165g/m²', quantity: 35, unit: 'm²' },
      { article: 'Dübel STR-U 200mm', quantity: 180, unit: 'Stk' },
    ],
    photos: [],
    notes: 'Südseite komplett entfernt. Untergrund stellenweise schadhaft — Bauherr informiert.',
    signatureStatus: 'signed',
    approvalStatus: 'approved',
    approvedBy: 'Hans Meier',
  },
  {
    id: 'rpt-3',
    date: '2026-02-13',
    projectId: 'prj-3',
    projectName: 'Innenausbau Büro Winterthur',
    author: 'Thomas Lehmann',
    weather: 'sunny',
    temperature: 12,
    workStart: '08:00',
    workEnd: '17:00',
    breakMinutes: 60,
    workers: [
      { name: 'Thomas Lehmann', role: 'Projektleiter', hours: 8 },
      { name: 'Markus Frei', role: 'Schreiner', hours: 8 },
      { name: 'Nico Berger', role: 'Schreiner', hours: 8 },
      { name: 'Andrea Moser', role: 'Malerin', hours: 8 },
      { name: 'Lukas Bauer', role: 'Elektriker', hours: 8 },
    ],
    activities: [
      { description: 'Trennwände Grossraumbüro montiert (Trockenbau)', category: 'Trockenbau' },
      { description: 'Bodenbelag Vinyl in Büro 3+4 verlegt', category: 'Bodenbelag' },
      { description: 'Malerarbeiten Empfangsbereich abgeschlossen', category: 'Malerarbeiten' },
    ],
    materials: [
      { article: 'Gipskartonplatten 12.5mm', quantity: 28, unit: 'm²' },
      { article: 'CW-Profile 75mm', quantity: 24, unit: 'Stk' },
      { article: 'Vinyl-Bodenbelag Eiche', quantity: 42, unit: 'm²' },
      { article: 'Wandfarbe weiss matt', quantity: 15, unit: 'Liter' },
    ],
    photos: [
      { id: 'ph-3', caption: 'Empfangsbereich nach Malerarbeiten' },
    ],
    notes: 'Büromöbel-Lieferung am Donnerstag erwartet. Bodenleger morgen für Korridor eingeplant.',
    signatureStatus: 'pending',
    approvalStatus: 'submitted',
  },
  {
    id: 'rpt-4',
    date: '2026-02-12',
    projectId: 'prj-4',
    projectName: 'Dacherneuerung Luzern',
    author: 'Beat Zimmermann',
    weather: 'rainy',
    temperature: 6,
    workStart: '07:00',
    workEnd: '14:00',
    breakMinutes: 30,
    workers: [
      { name: 'Beat Zimmermann', role: 'Dachdeckermeister', hours: 6.5 },
      { name: 'Tobias Roth', role: 'Dachdecker', hours: 6.5 },
      { name: 'Samuel Bieri', role: 'Lehrling', hours: 6.5 },
    ],
    activities: [
      { description: 'Unterdachbahn auf Ostseite verlegt (18 m²)', category: 'Dacharbeiten' },
      { description: 'Arbeit wegen Starkregen um 14:00 eingestellt', category: 'Unterbruch' },
    ],
    materials: [
      { article: 'Unterdachbahn Delta-Maxx', quantity: 20, unit: 'm²' },
      { article: 'Konterlattung 30x50mm', quantity: 36, unit: 'lfm' },
    ],
    photos: [
      { id: 'ph-4', caption: 'Unterdachbahn Ostseite vor Regenunterbruch' },
    ],
    notes: 'Arbeiten wegen starkem Regen vorzeitig beendet. Wetterprognose für morgen besser — Weiterarbeit geplant.',
    signatureStatus: 'signed',
    approvalStatus: 'approved',
    approvedBy: 'Beat Zimmermann',
  },
  {
    id: 'rpt-5',
    date: '2026-02-12',
    projectId: 'prj-5',
    projectName: 'Elektrische Installation St. Gallen',
    author: 'Christian Sutter',
    weather: 'cloudy',
    temperature: 9,
    workStart: '07:30',
    workEnd: '17:00',
    breakMinutes: 45,
    workers: [
      { name: 'Christian Sutter', role: 'Elektroinstallateur', hours: 8.75 },
      { name: 'Florian Ammann', role: 'Elektromonteur', hours: 8.75 },
      { name: 'Yannick Wenger', role: 'Lehrling', hours: 8.75 },
    ],
    activities: [
      { description: 'Leerrohrverlegung 1. OG (42 Stk)', category: 'Rohinstallation' },
      { description: 'Unterverteilung UG montiert und verdrahtet', category: 'Verteilung' },
      { description: 'Steckdosen/Schalter EG gesetzt (26 Stk)', category: 'Apparatemontage' },
      { description: 'Prüfprotokoll Erdung erstellt', category: 'Prüfung' },
    ],
    materials: [
      { article: 'Leerrohr M20 grau', quantity: 120, unit: 'm' },
      { article: 'Kabel TT 3x1.5mm²', quantity: 85, unit: 'm' },
      { article: 'Kabel TT 5x2.5mm²', quantity: 40, unit: 'm' },
      { article: 'Steckdosen Feller EDIZIOdue', quantity: 18, unit: 'Stk' },
      { article: 'Schalter Feller EDIZIOdue', quantity: 8, unit: 'Stk' },
    ],
    photos: [],
    notes: 'NIV-Kontrolle für nächste Woche angemeldet. Unterverteilung UG bereit für Abnahme.',
    signatureStatus: 'pending',
    approvalStatus: 'rejected',
    approvalComment: 'Prüfprotokoll Erdung fehlt als Anlage',
    approvedBy: 'Peter Widmer',
  },
  {
    id: 'rpt-6',
    date: '2026-02-11',
    projectId: 'prj-6',
    projectName: 'Gartenanlage Basel',
    author: 'Adrian Steffen',
    weather: 'sunny',
    temperature: 15,
    workStart: '07:00',
    workEnd: '17:30',
    breakMinutes: 60,
    workers: [
      { name: 'Adrian Steffen', role: 'Gartenbauleiter', hours: 9.5 },
      { name: 'Manuel Kunz', role: 'Landschaftsgärtner', hours: 9.5 },
      { name: 'Daniel Egli', role: 'Landschaftsgärtner', hours: 9.5 },
      { name: 'Pascal Wirth', role: 'Pflästerer', hours: 9.5 },
      { name: 'Tim Vogt', role: 'Hilfsarbeiter', hours: 9.5 },
      { name: 'Sara Kern', role: 'Hilfsarbeiterin', hours: 9.5 },
      { name: 'Michael Fuchs', role: 'Baggerführer', hours: 4 },
    ],
    activities: [
      { description: 'Aushub Schwimmteich (ca. 18 m³)', category: 'Erdarbeiten' },
      { description: 'Randsteine Zufahrt gesetzt (24 lfm)', category: 'Pflasterarbeiten' },
      { description: 'Rollrasen Südseite verlegt (85 m²)', category: 'Bepflanzung' },
      { description: 'Bewässerungsleitungen Beete West verlegt', category: 'Bewässerung' },
      { description: 'Natursteinmauer Terrasse aufgeschichtet', category: 'Steinarbeiten' },
    ],
    materials: [
      { article: 'Rollrasen Premium', quantity: 85, unit: 'm²' },
      { article: 'Granit-Randsteine 100x25x8', quantity: 24, unit: 'Stk' },
      { article: 'Teichfolie EPDM 1.1mm', quantity: 30, unit: 'm²' },
      { article: 'Bewässerungsrohr PE 25mm', quantity: 45, unit: 'm' },
      { article: 'Naturstein Tessiner Gneis', quantity: 3.2, unit: 't' },
      { article: 'Kies 16/32 Wandkies', quantity: 8, unit: 'm³' },
    ],
    photos: [
      { id: 'ph-5', caption: 'Aushub Schwimmteich' },
      { id: 'ph-6', caption: 'Rollrasen Südseite fertig verlegt' },
      { id: 'ph-7', caption: 'Natursteinmauer in Arbeit' },
    ],
    notes: 'Bagger morgen nicht verfügbar — Erdarbeiten am Mittwoch weiterführen. Pflanzlieferung für Freitag bestätigt.',
    signatureStatus: 'signed',
    approvalStatus: 'approved',
    approvedBy: 'Adrian Steffen',
  },
  {
    id: 'rpt-7',
    date: '2026-02-11',
    projectId: 'prj-7',
    projectName: 'Badezimmer-Renovation Thun',
    author: 'René Lüthi',
    weather: 'cloudy',
    temperature: 7,
    workStart: '08:00',
    workEnd: '16:00',
    breakMinutes: 30,
    workers: [
      { name: 'René Lüthi', role: 'Plattenleger', hours: 7.5 },
      { name: 'Oliver Hess', role: 'Sanitärinstallateur', hours: 7.5 },
    ],
    activities: [
      { description: 'Abdichtung Duschbereich mit Flüssigfolie', category: 'Abdichtung' },
      { description: 'Bodenplatten 60x60 Feinsteinzeug verlegt', category: 'Plattenarbeiten' },
      { description: 'Wasseranschlüsse Dusche/Lavabo vorbereitet', category: 'Sanitär' },
    ],
    materials: [
      { article: 'Feinsteinzeug 60x60 anthrazit', quantity: 8.5, unit: 'm²' },
      { article: 'Flüssigfolie Mapelastic', quantity: 6, unit: 'kg' },
      { article: 'Flexkleber C2TE', quantity: 2, unit: 'Sack' },
      { article: 'Sanitärsilikon transparent', quantity: 3, unit: 'Stk' },
    ],
    photos: [
      { id: 'ph-8', caption: 'Abdichtung Duschbereich fertig' },
    ],
    notes: 'Bodenplatten ca. 70% verlegt. Fugenarbeiten morgen. Duscharmatur-Lieferung ausstehend (erwartet Donnerstag).',
    signatureStatus: 'pending',
    approvalStatus: 'draft',
  },
  {
    id: 'rpt-8',
    date: '2026-02-10',
    projectId: 'prj-8',
    projectName: 'Tiefgarage Abdichtung Aarau',
    author: 'Marco Brunner',
    weather: 'snowy',
    temperature: -2,
    workStart: '08:00',
    workEnd: '15:00',
    breakMinutes: 45,
    workers: [
      { name: 'Marco Brunner', role: 'Polier', hours: 6.25 },
      { name: 'Luca Meier', role: 'Abdichter', hours: 6.25 },
      { name: 'Jonas Widmer', role: 'Abdichter', hours: 6.25 },
      { name: 'Patrick Steiner', role: 'Hilfsarbeiter', hours: 6.25 },
    ],
    activities: [
      { description: 'Bitumenbahn Bodenplatte verschweisst (45 m²)', category: 'Abdichtung' },
      { description: 'Schutzschicht Noppenbahn auf Wände montiert', category: 'Schutzschicht' },
    ],
    materials: [
      { article: 'Bitumenbahn V60 S4', quantity: 50, unit: 'm²' },
      { article: 'Noppenbahn Delta-MS 20', quantity: 28, unit: 'm²' },
      { article: 'Bitumen-Voranstrich', quantity: 10, unit: 'Liter' },
    ],
    photos: [
      { id: 'ph-9', caption: 'Abdichtung Bodenplatte Abschnitt Nord' },
      { id: 'ph-10', caption: 'Noppenbahn Wand Ost' },
    ],
    notes: 'Trotz Schneefall konnte in der Tiefgarage gearbeitet werden (überdacht). Temperatur grenzwertig für Bitumenverarbeitung — Heizgebläse eingesetzt.',
    signatureStatus: 'signed',
    approvalStatus: 'approved',
    approvedBy: 'Marco Brunner',
  },
]

const MOCK_MEASUREMENTS: Measurement[] = [
  {
    id: 'ms-1',
    name: 'Wohnzimmer EG',
    projectName: 'Innenausbau Büro Winterthur',
    positions: [
      { id: 'mp-1', label: 'Hauptfläche L-Form', length: 6.2, width: 4.8, height: 2.55, area: 29.76, volume: 75.89 },
      { id: 'mp-2', label: 'Erker Süd', length: 2.1, width: 1.4, height: 2.55, area: 2.94, volume: 7.50 },
      { id: 'mp-3', label: 'Nische Kamin', length: 1.8, width: 0.9, height: 2.55, area: 1.62, volume: 4.13 },
    ],
    createdAt: '2026-02-12',
    author: 'Thomas Lehmann',
  },
  {
    id: 'ms-2',
    name: 'Büro 1. OG',
    projectName: 'Innenausbau Büro Winterthur',
    positions: [
      { id: 'mp-4', label: 'Büro A', length: 5.4, width: 3.6, height: 2.7, area: 19.44, volume: 52.49 },
      { id: 'mp-5', label: 'Büro B', length: 4.8, width: 3.6, height: 2.7, area: 17.28, volume: 46.66 },
    ],
    createdAt: '2026-02-11',
    author: 'Thomas Lehmann',
  },
  {
    id: 'ms-3',
    name: 'Lagerraum UG',
    projectName: 'Tiefgarage Abdichtung Aarau',
    positions: [
      { id: 'mp-6', label: 'Hauptlager', length: 12.5, width: 8.2, height: 3.1, area: 102.5, volume: 317.75 },
    ],
    createdAt: '2026-02-10',
    author: 'Marco Brunner',
  },
  {
    id: 'ms-4',
    name: 'Terrasse Anbau',
    projectName: 'Gartenanlage Basel',
    positions: [
      { id: 'mp-7', label: 'Terrasse Hauptfläche', length: 6.0, width: 3.5, height: 0.15, area: 21.0, volume: 3.15 },
      { id: 'mp-8', label: 'Stufe/Absatz', length: 6.0, width: 0.4, height: 0.18, area: 2.4, volume: 0.43 },
    ],
    createdAt: '2026-02-09',
    author: 'Adrian Steffen',
  },
  {
    id: 'ms-5',
    name: 'Garage',
    projectName: 'Rohbau Mehrfamilienhaus Zürich',
    positions: [
      { id: 'mp-9', label: 'Einstellhalle', length: 15.0, width: 6.0, height: 2.8, area: 90.0, volume: 252.0 },
      { id: 'mp-10', label: 'Technikraum', length: 3.2, width: 2.4, height: 2.8, area: 7.68, volume: 21.5 },
    ],
    createdAt: '2026-02-08',
    author: 'Marco Brunner',
  },
]

const MOCK_TEMPLATES: ReportTemplate[] = [
  {
    id: 'tpl-1',
    name: 'Standard Tagesbericht',
    description: 'Allgemeiner Tagesbericht für Bauprojekte mit Standardkategorien',
    defaultActivities: ['Baustelleneinrichtung', 'Hauptarbeiten', 'Aufräumen', 'Materialtransport'],
    defaultMaterials: ['Beton', 'Mörtel', 'Holz', 'Schrauben/Dübel'],
  },
  {
    id: 'tpl-2',
    name: 'Renovierungsbericht',
    description: 'Für Sanierungs- und Umbauarbeiten mit Rückbau und Entsorgung',
    defaultActivities: ['Rückbau', 'Entsorgung', 'Neuaufbau', 'Malerarbeiten', 'Reinigung'],
    defaultMaterials: ['Gipskarton', 'Farbe', 'Spachtel', 'Abdeckmaterial'],
  },
  {
    id: 'tpl-3',
    name: 'Installationsbericht',
    description: 'Für Elektro-, Sanitär- und HVAC-Installationen',
    defaultActivities: ['Rohinstallation', 'Leitungsverlegung', 'Apparatemontage', 'Prüfung/Inbetriebnahme'],
    defaultMaterials: ['Kabel/Rohre', 'Verbindungsmaterial', 'Apparate', 'Isolationsmaterial'],
  },
]

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

interface RapporteStore {
  reports: FieldReport[]
  measurements: Measurement[]
  templates: ReportTemplate[]

  addReport: (report: FieldReport) => void
  updateReport: (id: string, updates: Partial<FieldReport>) => void
  deleteReport: (id: string) => void
  addMeasurement: (measurement: Measurement) => void
  deleteMeasurement: (id: string) => void
  addMeasurementPosition: (measurementId: string, position: MeasurementPosition) => void
}

export const useRapporteStore = create<RapporteStore>()(
  persist(
    (set) => ({
      reports: MOCK_REPORTS,
      measurements: MOCK_MEASUREMENTS,
      templates: MOCK_TEMPLATES,

      addReport: (report) =>
        set((state) => ({ reports: [report, ...state.reports] })),

      updateReport: (id, updates) =>
        set((state) => ({
          reports: state.reports.map((r) => (r.id === id ? { ...r, ...updates } : r)),
        })),

      deleteReport: (id) =>
        set((state) => ({ reports: state.reports.filter((r) => r.id !== id) })),

      addMeasurement: (measurement) =>
        set((state) => ({ measurements: [measurement, ...state.measurements] })),

      deleteMeasurement: (id) =>
        set((state) => ({ measurements: state.measurements.filter((m) => m.id !== id) })),

      addMeasurementPosition: (measurementId, position) =>
        set((state) => ({
          measurements: state.measurements.map((m) =>
            m.id === measurementId
              ? { ...m, positions: [...m.positions, position] }
              : m,
          ),
        })),
    }),
    { name: 'cosmi-rapporte' },
  ),
)
