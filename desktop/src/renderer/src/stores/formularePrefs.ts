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

interface FormularePrefsState {
  // personal
  defaultTab: FormulareTab
  defaultExportFormat: ExportFormat
  // tenant
  defaultConsentText: string
  defaultPrivacyUrl: string
  notifyOnSubmission: boolean
  retentionDays: number

  setDefaultTab: (t: FormulareTab) => void
  setDefaultExportFormat: (f: ExportFormat) => void
  setDefaultConsentText: (s: string) => void
  setDefaultPrivacyUrl: (s: string) => void
  setNotifyOnSubmission: (b: boolean) => void
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
      defaultConsentText: DEFAULT_CONSENT_TEXT,
      defaultPrivacyUrl: DEFAULT_PRIVACY_URL,
      notifyOnSubmission: true,
      retentionDays: 365,

      setDefaultTab: (defaultTab) => set({ defaultTab }),
      setDefaultExportFormat: (defaultExportFormat) => set({ defaultExportFormat }),
      setDefaultConsentText: (defaultConsentText) => set({ defaultConsentText }),
      setDefaultPrivacyUrl: (defaultPrivacyUrl) => set({ defaultPrivacyUrl }),
      setNotifyOnSubmission: (notifyOnSubmission) => set({ notifyOnSubmission }),
      setRetentionDays: (retentionDays) =>
        set({ retentionDays: Math.max(0, Math.round(retentionDays)) }),
    }),
    { name: 'cosmi-formulare-prefs' },
  ),
)
