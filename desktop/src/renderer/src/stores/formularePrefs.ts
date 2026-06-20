import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ExportFormat } from '@/api/formulare-types'

/**
 * Formulare settings.
 * - personal: default tab when the module opens + preferred export format
 *   (both apply for real in FormularePage / the ExportMenu).
 * - tenant (demo, persisted locally — no real backend): the default DSGVO
 *   consent text + privacy link used when a consent field is added, plus a
 *   submission-notification toggle and a retention period.
 */
export type FormulareTab = 'formulare' | 'eingänge' | 'vorlagen'
export type FormulareView = 'grid' | 'list'

interface FormularePrefsState {
  // personal
  defaultTab: FormulareTab
  defaultExportFormat: ExportFormat
  /** FT-5 — persists the FT-4 forms-tab grid/list toggle. */
  defaultFormView: FormulareView
  // tenant
  defaultConsentText: string
  defaultPrivacyUrl: string
  notifyOnSubmission: boolean
  /** FT-5 — recipient for the submission notification (when enabled). */
  notifyEmail: string
  /** FT-5 — default thank-you message proposed for new forms. */
  defaultThankYouMessage: string
  retentionDays: number

  setDefaultTab: (t: FormulareTab) => void
  setDefaultExportFormat: (f: ExportFormat) => void
  setDefaultFormView: (v: FormulareView) => void
  setDefaultConsentText: (s: string) => void
  setDefaultPrivacyUrl: (s: string) => void
  setNotifyOnSubmission: (b: boolean) => void
  setNotifyEmail: (s: string) => void
  setDefaultThankYouMessage: (s: string) => void
  setRetentionDays: (n: number) => void
}

export const DEFAULT_CONSENT_TEXT =
  'Ich willige ein, dass meine Angaben zur Bearbeitung meiner Anfrage gemäß der Datenschutzerklärung verarbeitet werden. Die Einwilligung kann jederzeit widerrufen werden.'
export const DEFAULT_PRIVACY_URL = 'https://www.zentria.tech/datenschutz'

export const useFormularePrefsStore = create<FormularePrefsState>()(
  persist(
    (set) => ({
      defaultTab: 'formulare',
      defaultExportFormat: 'csv',
      defaultFormView: 'grid',
      defaultConsentText: DEFAULT_CONSENT_TEXT,
      defaultPrivacyUrl: DEFAULT_PRIVACY_URL,
      notifyOnSubmission: true,
      notifyEmail: '',
      defaultThankYouMessage: '',
      retentionDays: 365,

      setDefaultTab: (defaultTab) => set({ defaultTab }),
      setDefaultExportFormat: (defaultExportFormat) => set({ defaultExportFormat }),
      setDefaultFormView: (defaultFormView) => set({ defaultFormView }),
      setDefaultConsentText: (defaultConsentText) => set({ defaultConsentText }),
      setDefaultPrivacyUrl: (defaultPrivacyUrl) => set({ defaultPrivacyUrl }),
      setNotifyOnSubmission: (notifyOnSubmission) => set({ notifyOnSubmission }),
      setNotifyEmail: (notifyEmail) => set({ notifyEmail }),
      setDefaultThankYouMessage: (defaultThankYouMessage) =>
        set({ defaultThankYouMessage }),
      setRetentionDays: (retentionDays) =>
        set({ retentionDays: Math.max(0, Math.round(retentionDays)) }),
    }),
    { name: 'cosmi-formulare-prefs' },
  ),
)
