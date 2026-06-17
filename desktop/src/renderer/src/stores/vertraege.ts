import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/** Stable action codes written by store mutations. Legacy mock entries use
 *  free-form German text which the renderer displays as-is (fallback). */
export type ContractHistoryActionCode =
  | 'contract_created'
  | 'contract_updated'
  | 'contract_terminated'
  | 'contract_signed'
  | 'reminder_triggered'
  | 'document_added'
  | 'document_removed'
  | 'contact_linked'
  | 'contact_unlinked'
  | 'deal_linked'
  | 'deal_unlinked'
  | 'invoice_linked'
  | 'invoice_unlinked'

/** A document reference attached to a contract (linked via Dokumente-Modul). */
export interface ContractDocument {
  fileId: string
  name: string
  mimeType?: string
  size?: number
  addedAt: string
}

export interface ContractHistoryEntry {
  date: string
  /** Either a ContractHistoryActionCode or legacy free-form text. */
  action: string
  user: string
  /** Optional extra payload for the action label (e.g. termination reason). */
  meta?: string
}

export interface ContractSigner {
  email: string
  name: string
  status: 'pending' | 'sent' | 'viewed' | 'signed' | 'declined'
  signedAt?: string
  order: number
  /** Base64 PNG data URL — present when signed via the on-site canvas (EES). */
  signatureDataUrl?: string
  /** How the signer completed (or will complete) the signing step. */
  signedVia?: 'canvas' | 'dispatch'
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
  /** @deprecated Use `documents` instead — kept for legacy migration only */
  documentRef?: string
  documents?: ContractDocument[]
  notes: string
  history: ContractHistoryEntry[]
  currency?: string
  reminderDays?: number[]
  templateId?: string
  signers?: ContractSigner[]
  /** CRM entity links (Phase 10 — mock-first, 1 contact + 1 deal + n invoices) */
  contactId?: string
  contactName?: string
  dealId?: string
  dealTitle?: string
  /** Snapshot list of linked invoice IDs */
  invoiceIds?: string[]
  /** Snapshot names for display (parallel to invoiceIds) */
  invoiceNames?: string[]
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
  addDocument: (contractId: string, doc: ContractDocument) => void
  removeDocument: (contractId: string, fileId: string) => void
  /** Sign a single signer via canvas or mark as sent via dispatch.
   *  Always appends a contract_signed history entry. */
  signSigner: (contractId: string, signerIndex: number, opts: {
    via: 'canvas' | 'dispatch'
    signatureDataUrl?: string
  }) => void
  /** Replace the signers array without adding a spurious contract_updated entry. */
  updateSigners: (contractId: string, signers: ContractSigner[]) => void
  /**
   * Dispatch for remote signing: persist the given signer list and move all
   * pending/viewed signers to `sent` (signedVia `dispatch`), appending one
   * `contract_sent` audit entry. Accepts the list so locally-added signers are
   * synced in the same write.
   */
  dispatchSigners: (contractId: string, signers: ContractSigner[]) => void
  /**
   * Demo: advance one dispatched signer one step along the simulated return
   * flow (`sent` → `viewed` → `signed`), appending a matching audit entry.
   * Returns the new status, or null if no transition applied.
   */
  advanceSignerReturn: (contractId: string, signerIndex: number) => ContractSigner['status'] | null
  /** Link a contact to a contract (1 contact max). */
  linkContact: (contractId: string, contactId: string, contactName: string) => void
  /** Unlink the contact from a contract. */
  unlinkContact: (contractId: string) => void
  /** Link a deal to a contract (1 deal max). */
  linkDeal: (contractId: string, dealId: string, dealTitle: string) => void
  /** Unlink the deal from a contract. */
  unlinkDeal: (contractId: string) => void
  /** Link an invoice to a contract (n invoices). */
  linkInvoice: (contractId: string, invoiceId: string, invoiceName: string) => void
  /** Unlink an invoice from a contract by invoiceId. */
  unlinkInvoice: (contractId: string, invoiceId: string) => void
}

/**
 * Demo: keep a few contracts genuinely "expiring soon" relative to today so
 * the Auslaufend-Tab and the Fristen-Reminder notifications always have data,
 * regardless of when the demo is opened. Returns an ISO date (YYYY-MM-DD).
 */
function isoDaysFromNow(days: number): string {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  d.setDate(d.getDate() + days)
  // Lokale Datums-Komponenten (nicht toISOString → UTC-Off-by-one in DE-Zeitzonen).
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
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
    documents: [
      { fileId: 'file-005', name: 'Vertrag_Gruber_Maschinenbau.pdf', mimeType: 'application/pdf', size: 540000, addedAt: '2024-01-01' },
      { fileId: 'file-007', name: 'NDA_Rhein_Consulting.pdf', mimeType: 'application/pdf', size: 180000, addedAt: '2024-01-15' },
    ],
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
    documents: [
      { fileId: 'file-006', name: 'SLA_Helvetia_Software.pdf', mimeType: 'application/pdf', size: 320000, addedAt: '2025-01-01' },
    ],
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
    status: 'active',
    startDate: '2025-04-01',
    endDate: isoDaysFromNow(18),
    noticePeriodDays: 30,
    renewal: 'manual',
    monthlyCost: 450,
    totalValue: 5400,
    documentRef: 'DOC-LZ-002',
    documents: [
      { fileId: 'file-021', name: 'Wartungsvertrag_Bavaria_Elektro.pdf', mimeType: 'application/pdf', size: 410000, addedAt: '2025-04-01' },
    ],
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
    documents: [
      { fileId: 'file-018', name: 'Datenschutzerklaerung.pdf', mimeType: 'application/pdf', size: 340000, addedAt: '2024-06-01' },
    ],
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
    endDate: isoDaysFromNow(47),
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
    documents: [
      { fileId: 'file-014', name: 'Arbeitsvertrag_Muster.pdf', mimeType: 'application/pdf', size: 290000, addedAt: '2023-03-01' },
    ],
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
    status: 'active',
    startDate: '2024-06-01',
    endDate: isoDaysFromNow(82),
    noticePeriodDays: 90,
    renewal: 'manual',
    monthlyCost: 1200,
    totalValue: 28800,
    documentRef: 'DOC-MV-002',
    notes: 'Lagerraum 120m2 im Industriegebiet Augsburg-Lechhausen. Zugang 6-22 Uhr, Rampe für LKW-Anlieferung vorhanden. Heizung und Strom inkl.',
    currency: 'EUR',
    reminderDays: [30, 60, 90],
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
            { date: today, action: 'contract_created', meta: template.name, user: 'Aktueller Benutzer' },
          ],
          ...overrides,
        }
        set((state) => ({ contracts: [...state.contracts, contract] }))
      },

      updateContract: (id, updates) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== id) return c
            // Append a history entry unless the caller already supplied one
            // (ContractDialog passes an updated history array explicitly).
            const callerProvidesHistory = Array.isArray(updates.history)
            if (callerProvidesHistory) {
              return { ...c, ...updates }
            }
            return {
              ...c,
              ...updates,
              history: [
                ...c.history,
                {
                  date: new Date().toISOString().split('T')[0],
                  action: 'contract_updated',
                  user: 'Aktueller Benutzer',
                },
              ],
            }
          }),
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
                      action: 'contract_terminated',
                      meta: reason,
                      user: 'Aktueller Benutzer',
                    },
                  ],
                }
              : c
          ),
        })),

      addDocument: (contractId, doc) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            return {
              ...c,
              documents: [...(c.documents ?? []), doc],
              history: [
                ...c.history,
                {
                  date: new Date().toISOString().split('T')[0],
                  action: 'document_added',
                  meta: doc.name,
                  user: 'Aktueller Benutzer',
                },
              ],
            }
          }),
        })),

      removeDocument: (contractId, fileId) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            const removed = (c.documents ?? []).find((d) => d.fileId === fileId)
            return {
              ...c,
              documents: (c.documents ?? []).filter((d) => d.fileId !== fileId),
              history: [
                ...c.history,
                {
                  date: new Date().toISOString().split('T')[0],
                  action: 'document_removed',
                  meta: removed?.name ?? fileId,
                  user: 'Aktueller Benutzer',
                },
              ],
            }
          }),
        })),

      signSigner: (contractId, signerIndex, { via, signatureDataUrl }) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            const signers = (c.signers ?? []).map((s, i) => {
              if (i !== signerIndex) return s
              return {
                ...s,
                status: 'signed' as const,
                signedAt: new Date().toISOString(),
                signedVia: via,
                ...(signatureDataUrl != null ? { signatureDataUrl } : {}),
              }
            })
            const signedSigner = signers[signerIndex]
            return {
              ...c,
              signers,
              history: [
                ...c.history,
                {
                  date: new Date().toISOString().split('T')[0],
                  action: 'contract_signed' as const,
                  meta: signedSigner?.name ?? '',
                  user: 'Aktueller Benutzer',
                },
              ],
            }
          }),
        })),

      updateSigners: (contractId, signers) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            return { ...c, signers }
          }),
        })),

      dispatchSigners: (contractId, incoming) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            const dispatched = incoming.filter(
              (s) => s.status === 'pending' || s.status === 'viewed',
            )
            if (dispatched.length === 0) return { ...c, signers: incoming }
            const signers = incoming.map((s) =>
              s.status === 'pending' || s.status === 'viewed'
                ? { ...s, status: 'sent' as const, signedVia: 'dispatch' as const }
                : s,
            )
            return {
              ...c,
              signers,
              history: [
                ...c.history,
                {
                  date: new Date().toISOString().split('T')[0],
                  action: 'contract_sent' as const,
                  meta: dispatched.map((s) => s.name).join(', '),
                  user: 'Aktueller Benutzer',
                },
              ],
            }
          }),
        })),

      advanceSignerReturn: (contractId, signerIndex) => {
        let nextStatus: ContractSigner['status'] | null = null
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            const current = (c.signers ?? [])[signerIndex]
            if (!current) return c
            // State machine: sent → viewed → signed. Anything else: no-op.
            if (current.status !== 'sent' && current.status !== 'viewed') return c
            nextStatus = current.status === 'sent' ? 'viewed' : 'signed'
            const action = nextStatus === 'viewed' ? 'contract_viewed' : 'contract_signed'
            const signers = (c.signers ?? []).map((s, i) => {
              if (i !== signerIndex) return s
              return nextStatus === 'signed'
                ? { ...s, status: 'signed' as const, signedAt: new Date().toISOString(), signedVia: 'dispatch' as const }
                : { ...s, status: 'viewed' as const }
            })
            return {
              ...c,
              signers,
              history: [
                ...c.history,
                {
                  date: new Date().toISOString().split('T')[0],
                  action,
                  meta: current.name,
                  // Rücklauf = Aktion des Unterzeichners, nicht des aktuellen Nutzers.
                  user: current.name,
                },
              ],
            }
          }),
        }))
        return nextStatus
      },

      linkContact: (contractId, contactId, contactName) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            return {
              ...c,
              contactId,
              contactName,
              history: [
                ...c.history,
                { date: new Date().toISOString().split('T')[0], action: 'contact_linked' as const, meta: contactName, user: 'Aktueller Benutzer' },
              ],
            }
          }),
        })),

      unlinkContact: (contractId) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            const name = c.contactName ?? ''
            return {
              ...c,
              contactId: undefined,
              contactName: undefined,
              history: [
                ...c.history,
                { date: new Date().toISOString().split('T')[0], action: 'contact_unlinked' as const, meta: name, user: 'Aktueller Benutzer' },
              ],
            }
          }),
        })),

      linkDeal: (contractId, dealId, dealTitle) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            return {
              ...c,
              dealId,
              dealTitle,
              history: [
                ...c.history,
                { date: new Date().toISOString().split('T')[0], action: 'deal_linked' as const, meta: dealTitle, user: 'Aktueller Benutzer' },
              ],
            }
          }),
        })),

      unlinkDeal: (contractId) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            const title = c.dealTitle ?? ''
            return {
              ...c,
              dealId: undefined,
              dealTitle: undefined,
              history: [
                ...c.history,
                { date: new Date().toISOString().split('T')[0], action: 'deal_unlinked' as const, meta: title, user: 'Aktueller Benutzer' },
              ],
            }
          }),
        })),

      linkInvoice: (contractId, invoiceId, invoiceName) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            if ((c.invoiceIds ?? []).includes(invoiceId)) return c
            return {
              ...c,
              invoiceIds: [...(c.invoiceIds ?? []), invoiceId],
              invoiceNames: [...(c.invoiceNames ?? []), invoiceName],
              history: [
                ...c.history,
                { date: new Date().toISOString().split('T')[0], action: 'invoice_linked' as const, meta: invoiceName, user: 'Aktueller Benutzer' },
              ],
            }
          }),
        })),

      unlinkInvoice: (contractId, invoiceId) =>
        set((state) => ({
          contracts: state.contracts.map((c) => {
            if (c.id !== contractId) return c
            const idx = (c.invoiceIds ?? []).indexOf(invoiceId)
            const name = idx >= 0 ? (c.invoiceNames ?? [])[idx] ?? invoiceId : invoiceId
            return {
              ...c,
              invoiceIds: (c.invoiceIds ?? []).filter((id) => id !== invoiceId),
              invoiceNames: (c.invoiceNames ?? []).filter((_, i) => i !== idx),
              history: [
                ...c.history,
                { date: new Date().toISOString().split('T')[0], action: 'invoice_unlinked' as const, meta: name, user: 'Aktueller Benutzer' },
              ],
            }
          }),
        })),
    }),
    { name: 'cosmi-verträge' },
  ),
)
