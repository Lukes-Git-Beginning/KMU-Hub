import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface ContractHistoryEntry {
  date: string
  action: string
  user: string
}

export interface Contract {
  id: string
  contractNumber: string
  title: string
  partner: string
  type: 'mietvertrag' | 'liefervertrag' | 'servicevertrag' | 'arbeitsvertrag' | 'lizenz' | 'versicherung'
  status: 'active' | 'expiring' | 'terminated' | 'expired'
  startDate: string
  endDate: string
  noticePeriodDays: number
  renewal: 'auto' | 'manual'
  monthlyCost: number
  totalValue: number
  documentRef?: string
  notes: string
  history: ContractHistoryEntry[]
}

export type ContractType = Contract['type']
export type ContractStatus = Contract['status']

interface VertraegeStore {
  contracts: Contract[]
  addContract: (contract: Contract) => void
  updateContract: (id: string, updates: Partial<Contract>) => void
  deleteContract: (id: string) => void
  terminateContract: (id: string, reason: string, date: string) => void
}

const MOCK_CONTRACTS: Contract[] = [
  {
    id: 'v-1',
    contractNumber: 'MV-2024-001',
    title: 'Buero-Mietvertrag Zuerich',
    partner: 'Immobilien Zuerich AG',
    type: 'mietvertrag',
    status: 'active',
    startDate: '2024-01-01',
    endDate: '2029-01-01',
    noticePeriodDays: 180,
    renewal: 'manual',
    monthlyCost: 4500,
    totalValue: 270000,
    documentRef: 'DOC-MV-001',
    notes: 'Hauptsitz im Kreis 5. Miete inkl. Nebenkosten und 2 Parkplaetze in der Tiefgarage. Kaution CHF 13500 hinterlegt bei der Zuercher Kantonalbank.',
    history: [
      { date: '2024-01-01', action: 'Vertrag unterzeichnet', user: 'Markus Weber' },
      { date: '2024-01-15', action: 'Bueroeinrichtung abgeschlossen', user: 'Sandra Buerki' },
      { date: '2025-06-01', action: 'Nebenkostenabrechnung geprueft', user: 'Markus Weber' },
    ],
  },
  {
    id: 'v-2',
    contractNumber: 'SV-2025-003',
    title: 'Swisscom Business Internet',
    partner: 'Swisscom (Schweiz) AG',
    type: 'servicevertrag',
    status: 'active',
    startDate: '2025-01-01',
    endDate: '2027-01-01',
    noticePeriodDays: 60,
    renewal: 'auto',
    monthlyCost: 189,
    totalValue: 4536,
    documentRef: 'DOC-SV-003',
    notes: 'Business Internet XL mit 10 Gbit/s symmetrisch, inkl. Managed Router und SLA 99.9%. Stoerungshotline 24/7.',
    history: [
      { date: '2025-01-01', action: 'Vertrag aktiviert', user: 'Thomas Keller' },
      { date: '2025-01-10', action: 'Router installiert und konfiguriert', user: 'Thomas Keller' },
    ],
  },
  {
    id: 'v-3',
    contractNumber: 'LZ-2025-002',
    title: 'Microsoft 365 Business',
    partner: 'Microsoft Ireland Operations Ltd.',
    type: 'lizenz',
    status: 'expiring',
    startDate: '2025-04-01',
    endDate: '2026-04-01',
    noticePeriodDays: 30,
    renewal: 'manual',
    monthlyCost: 450,
    totalValue: 5400,
    documentRef: 'DOC-LZ-002',
    notes: '25 Lizenzen Microsoft 365 Business Premium. Inkl. Exchange Online, Teams, SharePoint und Intune. Verlaengerung muss manuell bestaetigt werden.',
    history: [
      { date: '2025-04-01', action: 'Lizenz aktiviert', user: 'Thomas Keller' },
      { date: '2025-04-05', action: 'Migration von G Suite abgeschlossen', user: 'Thomas Keller' },
      { date: '2025-10-15', action: 'Lizenzen von 20 auf 25 erhoeht', user: 'Markus Weber' },
    ],
  },
  {
    id: 'v-4',
    contractNumber: 'VS-2024-004',
    title: 'Helvetia Betriebsversicherung',
    partner: 'Helvetia Versicherungen AG',
    type: 'versicherung',
    status: 'active',
    startDate: '2024-06-01',
    endDate: '2027-06-01',
    noticePeriodDays: 90,
    renewal: 'auto',
    monthlyCost: 890,
    totalValue: 32040,
    documentRef: 'DOC-VS-004',
    notes: 'Kombinierte Betriebsversicherung: Haftpflicht (CHF 5 Mio.), Inventar (CHF 500000), Betriebsunterbrechung (12 Monate). Selbstbehalt CHF 1000 pro Schadensfall.',
    history: [
      { date: '2024-06-01', action: 'Police ausgestellt', user: 'Markus Weber' },
      { date: '2024-12-15', action: 'Jaehrliche Praemienanpassung +2.1%', user: 'Sandra Buerki' },
      { date: '2025-06-01', action: 'Inventarwert aktualisiert', user: 'Markus Weber' },
    ],
  },
  {
    id: 'v-5',
    contractNumber: 'LF-2025-001',
    title: 'Mueller Metallbau Rahmenvertrag',
    partner: 'Mueller Metallbau GmbH',
    type: 'liefervertrag',
    status: 'active',
    startDate: '2025-03-01',
    endDate: '2026-03-01',
    noticePeriodDays: 30,
    renewal: 'auto',
    monthlyCost: 2200,
    totalValue: 26400,
    notes: 'Rahmenvertrag fuer Stahlkomponenten und Sonderanfertigungen. Lieferzeit max. 10 Werktage, Zahlungsziel 30 Tage netto. Mengenrabatt ab CHF 5000 pro Bestellung.',
    history: [
      { date: '2025-03-01', action: 'Rahmenvertrag unterzeichnet', user: 'Lukas Brunner' },
      { date: '2025-05-20', action: 'Erste Bestellung ausgeloest', user: 'Lukas Brunner' },
    ],
  },
  {
    id: 'v-6',
    contractNumber: 'SV-2025-006',
    title: 'Reinigungsfirma Clean AG',
    partner: 'Clean AG Gebaeudereinigung',
    type: 'servicevertrag',
    status: 'active',
    startDate: '2025-06-01',
    endDate: '2026-06-01',
    noticePeriodDays: 30,
    renewal: 'auto',
    monthlyCost: 750,
    totalValue: 9000,
    notes: 'Buroreinigung 3x woechtentlich (Mo/Mi/Fr), Grundreinigung 1x monatlich. Reinigungsmittel inkl. Fensterreinigung quartalsweise.',
    history: [
      { date: '2025-06-01', action: 'Servicevertrag gestartet', user: 'Sandra Buerki' },
      { date: '2025-08-10', action: 'Zusaetzliche Fensterreinigung vereinbart', user: 'Sandra Buerki' },
    ],
  },
  {
    id: 'v-7',
    contractNumber: 'AV-2023-001',
    title: 'Thomas Berger Arbeitsvertrag',
    partner: 'Thomas Berger',
    type: 'arbeitsvertrag',
    status: 'active',
    startDate: '2023-03-01',
    endDate: '',
    noticePeriodDays: 90,
    renewal: 'auto',
    monthlyCost: 6500,
    totalValue: 0,
    notes: 'Unbefristeter Arbeitsvertrag, Senior Entwickler. 100% Pensum, 5 Wochen Ferien. 13. Monatslohn. Homeoffice 2 Tage/Woche gemaess Reglement.',
    history: [
      { date: '2023-03-01', action: 'Arbeitsvertrag unterzeichnet', user: 'Markus Weber' },
      { date: '2024-03-01', action: 'Lohnerhoehung +3.5%', user: 'Markus Weber' },
      { date: '2025-03-01', action: 'Zweites Dienstjubilaeum', user: 'Sandra Buerki' },
    ],
  },
  {
    id: 'v-8',
    contractNumber: 'LZ-2025-005',
    title: 'Adobe Creative Cloud',
    partner: 'Adobe Systems Software Ireland Ltd.',
    type: 'lizenz',
    status: 'active',
    startDate: '2025-01-01',
    endDate: '2026-01-01',
    noticePeriodDays: 30,
    renewal: 'auto',
    monthlyCost: 680,
    totalValue: 8160,
    documentRef: 'DOC-LZ-005',
    notes: '10 Lizenzen Creative Cloud All Apps. Nutzung fuer Marketing-Team und Design-Abteilung. Schulung ueber Adobe Learning inbegriffen.',
    history: [
      { date: '2025-01-01', action: 'Jahreslizenz erneuert', user: 'Thomas Keller' },
      { date: '2025-01-15', action: 'Lizenzen zugewiesen an Team', user: 'Thomas Keller' },
    ],
  },
  {
    id: 'v-9',
    contractNumber: 'VS-2025-002',
    title: 'AXA Fahrzeugversicherung',
    partner: 'AXA Versicherungen AG',
    type: 'versicherung',
    status: 'active',
    startDate: '2025-01-01',
    endDate: '2026-01-01',
    noticePeriodDays: 30,
    renewal: 'auto',
    monthlyCost: 320,
    totalValue: 3840,
    notes: 'Flottenversicherung fuer 3 Firmenfahrzeuge. Vollkasko mit CHF 500 Selbstbehalt. Pannenhilfe Schweiz und Europa inkl.',
    history: [
      { date: '2025-01-01', action: 'Police erneuert', user: 'Sandra Buerki' },
      { date: '2025-04-12', action: 'Neues Fahrzeug hinzugefuegt (ZH-345678)', user: 'Sandra Buerki' },
    ],
  },
  {
    id: 'v-10',
    contractNumber: 'LF-2024-003',
    title: 'Weber Transport Rahmenvertrag',
    partner: 'Weber Transport & Logistik AG',
    type: 'liefervertrag',
    status: 'terminated',
    startDate: '2024-01-01',
    endDate: '2025-12-31',
    noticePeriodDays: 60,
    renewal: 'manual',
    monthlyCost: 1800,
    totalValue: 43200,
    notes: 'Rahmenvertrag fuer regelmaessige Warentransporte. Gekuendigt per 31.12.2025 wegen Wechsel zu guenstigerem Anbieter. Restliche Lieferungen werden noch abgewickelt.',
    history: [
      { date: '2024-01-01', action: 'Rahmenvertrag unterzeichnet', user: 'Lukas Brunner' },
      { date: '2025-06-15', action: 'Kuendigung eingereicht', user: 'Markus Weber' },
      { date: '2025-06-20', action: 'Kuendigung bestaetigt vom Partner', user: 'Sandra Buerki' },
    ],
  },
  {
    id: 'v-11',
    contractNumber: 'MV-2024-002',
    title: 'Lagerraum Winterthur',
    partner: 'Immo-Invest Winterthur AG',
    type: 'mietvertrag',
    status: 'expiring',
    startDate: '2024-06-01',
    endDate: '2026-06-01',
    noticePeriodDays: 90,
    renewal: 'manual',
    monthlyCost: 1200,
    totalValue: 28800,
    documentRef: 'DOC-MV-002',
    notes: 'Lagerraum 120m2 im Industriegebiet Winterthur-Toess. Zugang 6-22 Uhr, Rampe fuer LKW-Anlieferung vorhanden. Heizung und Strom inkl.',
    history: [
      { date: '2024-06-01', action: 'Mietvertrag unterzeichnet', user: 'Markus Weber' },
      { date: '2025-01-15', action: 'Lagerregale installiert', user: 'Lukas Brunner' },
      { date: '2026-01-10', action: 'Erinnerung: Kuendigungsfrist laeuft', user: 'System' },
    ],
  },
  {
    id: 'v-12',
    contractNumber: 'LZ-2025-007',
    title: 'SAP Business One Lizenz',
    partner: 'SAP (Schweiz) AG',
    type: 'lizenz',
    status: 'active',
    startDate: '2025-07-01',
    endDate: '2028-07-01',
    noticePeriodDays: 180,
    renewal: 'manual',
    monthlyCost: 1950,
    totalValue: 70200,
    documentRef: 'DOC-LZ-007',
    notes: '15 Named-User-Lizenzen SAP Business One. Inkl. Finanzwesen, Einkauf, Vertrieb und Lagerverwaltung. Support Level: Enterprise (Reaktionszeit 4h).',
    history: [
      { date: '2025-07-01', action: 'Lizenzvertrag aktiviert', user: 'Thomas Keller' },
      { date: '2025-07-20', action: 'Go-Live nach 3 Wochen Implementierung', user: 'Thomas Keller' },
      { date: '2025-09-15', action: 'Schulung fuer Buchhaltungsteam abgeschlossen', user: 'Sandra Buerki' },
    ],
  },
]

export const useVertraegeStore = create<VertraegeStore>()(
  persist(
    (set) => ({
      contracts: MOCK_CONTRACTS,

      addContract: (contract) =>
        set((state) => ({
          contracts: [...state.contracts, contract],
        })),

      updateContract: (id, updates) =>
        set((state) => ({
          contracts: state.contracts.map((c) =>
            c.id === id ? { ...c, ...updates } : c
          ),
        })),

      deleteContract: (id) =>
        set((state) => ({
          contracts: state.contracts.filter((c) => c.id !== id),
        })),

      terminateContract: (id, reason, date) =>
        set((state) => ({
          contracts: state.contracts.map((c) =>
            c.id === id
              ? {
                  ...c,
                  status: 'terminated' as const,
                  history: [
                    ...c.history,
                    {
                      date,
                      action: `Kuendigung eingeleitet: ${reason}`,
                      user: 'Aktueller Benutzer',
                    },
                  ],
                }
              : c
          ),
        })),
    }),
    { name: 'kmuhub-vertraege' },
  ),
)
