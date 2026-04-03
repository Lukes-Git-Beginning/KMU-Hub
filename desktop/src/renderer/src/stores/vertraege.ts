import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface ContractHistoryEntry {
  date: string
  action: string
  user: string
}

export interface ContractSigner {
  email: string
  name: string
  status: 'pending' | 'sent' | 'viewed' | 'signed' | 'declined'
  signedAt?: string
  order: number
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
  currency?: string
  reminderDays?: number[]
  templateId?: string
  signers?: ContractSigner[]
}

export type ContractType = Contract['type']
export type ContractStatus = Contract['status']

export interface ContractTemplate {
  id: string
  name: string
  description: string
  type: ContractType
  defaultDuration: string
  defaultNoticePeriodDays: number
  defaultRenewal: 'auto' | 'manual'
  defaultMonthlyCost?: number
}

const MOCK_TEMPLATES: ContractTemplate[] = [
  {
    id: 'tpl-miet',
    name: 'Standard-Mietvertrag',
    description: 'Gewerbliche Mietverträge für Büro- und Lagerräume mit üblichen Konditionen',
    type: 'mietvertrag',
    defaultDuration: '36',
    defaultNoticePeriodDays: 90,
    defaultRenewal: 'manual',
    defaultMonthlyCost: 2500,
  },
  {
    id: 'tpl-service',
    name: 'Servicevertrag (SLA)',
    description: 'IT- und Facility-Serviceverträge mit definierten Service Level Agreements',
    type: 'servicevertrag',
    defaultDuration: '12',
    defaultNoticePeriodDays: 30,
    defaultRenewal: 'auto',
    defaultMonthlyCost: 500,
  },
  {
    id: 'tpl-lizenz',
    name: 'Software-Lizenzvertrag',
    description: 'Jahreslizenzen für Software und Cloud-Dienste mit automatischer Verlängerung',
    type: 'lizenz',
    defaultDuration: '12',
    defaultNoticePeriodDays: 30,
    defaultRenewal: 'auto',
    defaultMonthlyCost: 300,
  },
  {
    id: 'tpl-liefer',
    name: 'Rahmenvertrag Lieferant',
    description: 'Rahmenverträge mit Lieferanten inkl. Mengenrabatte und Zahlungskonditionen',
    type: 'liefervertrag',
    defaultDuration: '12',
    defaultNoticePeriodDays: 30,
    defaultRenewal: 'auto',
  },
  {
    id: 'tpl-arbeit',
    name: 'Arbeitsvertrag (unbefristet)',
    description: 'Standard-Arbeitsvertrag für unbefristete Anstellungen nach deutschem Recht',
    type: 'arbeitsvertrag',
    defaultDuration: '',
    defaultNoticePeriodDays: 90,
    defaultRenewal: 'auto',
  },
  {
    id: 'tpl-versicherung',
    name: 'Betriebsversicherung',
    description: 'Haftpflicht-, Inventar- und Betriebsunterbrechungsversicherungen',
    type: 'versicherung',
    defaultDuration: '12',
    defaultNoticePeriodDays: 90,
    defaultRenewal: 'auto',
    defaultMonthlyCost: 400,
  },
]

interface VertraegeStore {
  contracts: Contract[]
  contractTemplates: ContractTemplate[]
  addContract: (contract: Contract) => void
  addContractFromTemplate: (templateId: string, overrides?: Partial<Contract>) => void
  updateContract: (id: string, updates: Partial<Contract>) => void
  deleteContract: (id: string) => void
  terminateContract: (id: string, reason: string, date: string) => void
}

const MOCK_CONTRACTS: Contract[] = [
  {
    id: 'v-1',
    contractNumber: 'MV-2024-001',
    title: 'Büro-Mietvertrag München',
    partner: 'Immobilien München GmbH',
    type: 'mietvertrag',
    status: 'active',
    startDate: '2024-01-01',
    endDate: '2029-01-01',
    noticePeriodDays: 180,
    renewal: 'manual',
    monthlyCost: 4500,
    totalValue: 270000,
    documentRef: 'DOC-MV-001',
    notes: 'Hauptsitz in Schwabing. Miete inkl. Nebenkosten und 2 Parkplätze in der Tiefgarage. Kaution EUR 13500 hinterlegt bei der Commerzbank.',
    currency: 'EUR',
    reminderDays: [30, 60, 90],
    history: [
      { date: '2024-01-01', action: 'Vertrag unterzeichnet', user: 'Markus Weber' },
      { date: '2024-01-15', action: 'Büroeinrichtung abgeschlossen', user: 'Sandra Bürki' },
      { date: '2025-06-01', action: 'Nebenkostenabrechnung geprüft', user: 'Markus Weber' },
    ],
  },
  {
    id: 'v-2',
    contractNumber: 'SV-2025-003',
    title: 'Telekom Business Internet',
    partner: 'Deutsche Telekom GmbH',
    type: 'servicevertrag',
    status: 'active',
    startDate: '2025-01-01',
    endDate: '2027-01-01',
    noticePeriodDays: 60,
    renewal: 'auto',
    monthlyCost: 189,
    totalValue: 4536,
    documentRef: 'DOC-SV-003',
    notes: 'Business Internet XL mit 10 Gbit/s symmetrisch, inkl. Managed Router und SLA 99.9%. Störungshotline 24/7.',
    currency: 'EUR',
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
    notes: '25 Lizenzen Microsoft 365 Business Premium. Inkl. Exchange Online, Teams, SharePoint und Intune. Verlängerung muss manuell bestätigt werden.',
    currency: 'EUR',
    reminderDays: [30, 60, 90],
    history: [
      { date: '2025-04-01', action: 'Lizenz aktiviert', user: 'Thomas Keller' },
      { date: '2025-04-05', action: 'Migration von G Suite abgeschlossen', user: 'Thomas Keller' },
      { date: '2025-10-15', action: 'Lizenzen von 20 auf 25 erhöht', user: 'Markus Weber' },
    ],
  },
  {
    id: 'v-4',
    contractNumber: 'VS-2024-004',
    title: 'Allianz Betriebsversicherung',
    partner: 'Allianz Versicherungs-AG',
    type: 'versicherung',
    status: 'active',
    startDate: '2024-06-01',
    endDate: '2027-06-01',
    noticePeriodDays: 90,
    renewal: 'auto',
    monthlyCost: 890,
    totalValue: 32040,
    documentRef: 'DOC-VS-004',
    notes: 'Kombinierte Betriebsversicherung: Haftpflicht (EUR 5 Mio.), Inventar (EUR 500000), Betriebsunterbrechung (12 Monate). Selbstbehalt EUR 1000 pro Schadensfall.',
    currency: 'EUR',
    history: [
      { date: '2024-06-01', action: 'Police ausgestellt', user: 'Markus Weber' },
      { date: '2024-12-15', action: 'Jährliche Prämienanpassung +2.1%', user: 'Sandra Bürki' },
      { date: '2025-06-01', action: 'Inventarwert aktualisiert', user: 'Markus Weber' },
    ],
  },
  {
    id: 'v-5',
    contractNumber: 'LF-2025-001',
    title: 'Müller Metallbau Rahmenvertrag',
    partner: 'Müller Metallbau GmbH',
    type: 'liefervertrag',
    status: 'active',
    startDate: '2025-03-01',
    endDate: '2026-03-01',
    noticePeriodDays: 30,
    renewal: 'auto',
    monthlyCost: 2200,
    totalValue: 26400,
    notes: 'Rahmenvertrag für Stahlkomponenten und Sonderanfertigungen. Lieferzeit max. 10 Werktage, Zahlungsziel 30 Tage netto. Mengenrabatt ab EUR 5000 pro Bestellung.',
    currency: 'EUR',
    reminderDays: [30, 60, 90],
    signers: [
      { email: 'l.brunner@firma.de', name: 'Lukas Brunner', status: 'signed', signedAt: '2025-02-28T14:30:00', order: 1 },
      { email: 'h.mueller@metallbau.de', name: 'Hans Müller', status: 'pending', order: 2 },
    ],
    history: [
      { date: '2025-03-01', action: 'Rahmenvertrag unterzeichnet', user: 'Lukas Brunner' },
      { date: '2025-05-20', action: 'Erste Bestellung ausgelöst', user: 'Lukas Brunner' },
    ],
  },
  {
    id: 'v-6',
    contractNumber: 'SV-2025-006',
    title: 'Reinigungsfirma Clean AG',
    partner: 'Clean AG Gebäudereinigung',
    type: 'servicevertrag',
    status: 'active',
    startDate: '2025-06-01',
    endDate: '2026-06-01',
    noticePeriodDays: 30,
    renewal: 'auto',
    monthlyCost: 750,
    totalValue: 9000,
    notes: 'Büroreinigung 3x wöchentlich (Mo/Mi/Fr), Grundreinigung 1x monatlich. Reinigungsmittel inkl. Fensterreinigung quartalsweise.',
    currency: 'EUR',
    history: [
      { date: '2025-06-01', action: 'Servicevertrag gestartet', user: 'Sandra Bürki' },
      { date: '2025-08-10', action: 'Zusätzliche Fensterreinigung vereinbart', user: 'Sandra Bürki' },
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
    notes: 'Unbefristeter Arbeitsvertrag, Senior Entwickler. 100% Pensum, 30 Tage Urlaub. Homeoffice 2 Tage/Woche gemäß Betriebsvereinbarung.',
    currency: 'EUR',
    history: [
      { date: '2023-03-01', action: 'Arbeitsvertrag unterzeichnet', user: 'Markus Weber' },
      { date: '2024-03-01', action: 'Lohnerhöhung +3.5%', user: 'Markus Weber' },
      { date: '2025-03-01', action: 'Zweites Dienstjubiläum', user: 'Sandra Bürki' },
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
    notes: '10 Lizenzen Creative Cloud All Apps. Nutzung für Marketing-Team und Design-Abteilung. Schulung über Adobe Learning inbegriffen.',
    currency: 'EUR',
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
    notes: 'Flottenversicherung für 3 Firmenfahrzeuge. Vollkasko mit EUR 500 Selbstbehalt. Pannenhilfe Deutschland und Europa inkl.',
    currency: 'EUR',
    history: [
      { date: '2025-01-01', action: 'Police erneuert', user: 'Sandra Bürki' },
      { date: '2025-04-12', action: 'Neues Fahrzeug hinzugefügt (M-AB 3456)', user: 'Sandra Bürki' },
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
    notes: 'Rahmenvertrag für regelmäßige Warentransporte. Gekündigt per 31.12.2025 wegen Wechsel zu günstigerem Anbieter. Restliche Lieferungen werden noch abgewickelt.',
    currency: 'EUR',
    history: [
      { date: '2024-01-01', action: 'Rahmenvertrag unterzeichnet', user: 'Lukas Brunner' },
      { date: '2025-06-15', action: 'Kündigung eingereicht', user: 'Markus Weber' },
      { date: '2025-06-20', action: 'Kündigung bestätigt vom Partner', user: 'Sandra Bürki' },
    ],
  },
  {
    id: 'v-11',
    contractNumber: 'MV-2024-002',
    title: 'Lagerraum Augsburg',
    partner: 'Immo-Invest Augsburg GmbH',
    type: 'mietvertrag',
    status: 'expiring',
    startDate: '2024-06-01',
    endDate: '2026-06-01',
    noticePeriodDays: 90,
    renewal: 'manual',
    monthlyCost: 1200,
    totalValue: 28800,
    documentRef: 'DOC-MV-002',
    notes: 'Lagerraum 120m2 im Industriegebiet Augsburg-Lechhausen. Zugang 6-22 Uhr, Rampe für LKW-Anlieferung vorhanden. Heizung und Strom inkl.',
    currency: 'EUR',
    history: [
      { date: '2024-06-01', action: 'Mietvertrag unterzeichnet', user: 'Markus Weber' },
      { date: '2025-01-15', action: 'Lagerregale installiert', user: 'Lukas Brunner' },
      { date: '2026-01-10', action: 'Erinnerung: Kündigungsfrist läuft', user: 'System' },
    ],
  },
  {
    id: 'v-12',
    contractNumber: 'LZ-2025-007',
    title: 'SAP Business One Lizenz',
    partner: 'SAP Deutschland SE & Co. KG',
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
    currency: 'EUR',
    history: [
      { date: '2025-07-01', action: 'Lizenzvertrag aktiviert', user: 'Thomas Keller' },
      { date: '2025-07-20', action: 'Go-Live nach 3 Wochen Implementierung', user: 'Thomas Keller' },
      { date: '2025-09-15', action: 'Schulung für Buchhaltungsteam abgeschlossen', user: 'Sandra Bürki' },
    ],
  },
]

export const useVertraegeStore = create<VertraegeStore>()(
  persist(
    (set, get) => ({
      contracts: MOCK_CONTRACTS,
      contractTemplates: MOCK_TEMPLATES,

      addContract: (contract) =>
        set((state) => ({
          contracts: [...state.contracts, contract],
        })),

      addContractFromTemplate: (templateId, overrides) => {
        const template = get().contractTemplates.find((t) => t.id === templateId)
        if (!template) return
        const today = new Date().toISOString().split('T')[0]
        let endDate = ''
        if (template.defaultDuration) {
          const end = new Date()
          end.setMonth(end.getMonth() + Number(template.defaultDuration))
          endDate = end.toISOString().split('T')[0]
        }
        const months = template.defaultDuration ? Number(template.defaultDuration) : 0
        const monthlyCost = template.defaultMonthlyCost ?? 0
        const contract: Contract = {
          id: `v-${Date.now()}`,
          contractNumber: '',
          title: '',
          partner: '',
          type: template.type,
          status: 'active',
          startDate: today,
          endDate,
          noticePeriodDays: template.defaultNoticePeriodDays,
          renewal: template.defaultRenewal,
          monthlyCost,
          totalValue: monthlyCost * months,
          notes: '',
          templateId,
          currency: 'EUR',
          history: [
            { date: today, action: `Aus Vorlage "${template.name}" erstellt`, user: 'Aktueller Benutzer' },
          ],
          ...overrides,
        }
        set((state) => ({ contracts: [...state.contracts, contract] }))
      },

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
                      action: `Kündigung eingeleitet: ${reason}`,
                      user: 'Aktueller Benutzer',
                    },
                  ],
                }
              : c
          ),
        })),
    }),
    { name: 'kmuhub-verträge' },
  ),
)
