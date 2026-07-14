/**
 * Formulare — tenant-weite DSGVO-Vorgaben ("Für alle"-Scope, siehe
 * ModuleSettingsShell): Standard-Einwilligungstext + Datenschutz-Link (beim
 * Hinzufügen eines Consent-Felds), Eingangs-Benachrichtigung + Aufbewahrung.
 * Editierbar nur durch Modul-Leiter/Admin (der PUT wird serverseitig für andere
 * Rollen verworfen; das Panel gatet die UI). Persönliche Prefs liegen in
 * stores/formularePrefs.ts.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/formulare/tenant):
 * initFromServer() hydratet einmal pro Session (via useHydrateModuleSettings);
 * Setter schreiben durch. localStorage bleibt Optimistic-Cache.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

const MODULE_ID = 'formulare'

export const DEFAULT_CONSENT_TEXT =
  'Ich willige ein, dass meine Angaben zur Bearbeitung meiner Anfrage gemäß der Datenschutzerklärung verarbeitet werden. Die Einwilligung kann jederzeit widerrufen werden.'
export const DEFAULT_PRIVACY_URL = 'https://www.zentria.tech/datenschutz'

interface FormulareTenantState {
  defaultConsentText: string
  defaultPrivacyUrl: string
  notifyOnSubmission: boolean
  /** Recipient for the submission notification (when enabled). */
  notifyEmail: string
  /** Default thank-you message proposed for new forms. */
  defaultThankYouMessage: string
  retentionDays: number
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultConsentText: (s: string) => void
  setDefaultPrivacyUrl: (s: string) => void
  setNotifyOnSubmission: (b: boolean) => void
  setNotifyEmail: (s: string) => void
  setDefaultThankYouMessage: (s: string) => void
  setRetentionDays: (n: number) => void
  initFromServer: () => Promise<void>
}

function tenantPayload(s: FormulareTenantState): Record<string, unknown> {
  return {
    defaultConsentText: s.defaultConsentText,
    defaultPrivacyUrl: s.defaultPrivacyUrl,
    notifyOnSubmission: s.notifyOnSubmission,
    notifyEmail: s.notifyEmail,
    defaultThankYouMessage: s.defaultThankYouMessage,
    retentionDays: s.retentionDays,
  }
}

export const useFormulareTenantStore = create<FormulareTenantState>()(
  persist(
    (set, get) => ({
      defaultConsentText: DEFAULT_CONSENT_TEXT,
      defaultPrivacyUrl: DEFAULT_PRIVACY_URL,
      notifyOnSubmission: true,
      notifyEmail: '',
      defaultThankYouMessage: '',
      retentionDays: 365,
      serverInitialized: false,
      setDefaultConsentText: (defaultConsentText) => {
        set({ defaultConsentText })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setDefaultPrivacyUrl: (defaultPrivacyUrl) => {
        set({ defaultPrivacyUrl })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setNotifyOnSubmission: (notifyOnSubmission) => {
        set({ notifyOnSubmission })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setNotifyEmail: (notifyEmail) => {
        set({ notifyEmail })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setDefaultThankYouMessage: (defaultThankYouMessage) => {
        set({ defaultThankYouMessage })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setRetentionDays: (retentionDays) => {
        set({ retentionDays: Math.max(0, Math.round(retentionDays)) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          defaultConsentText:
            typeof map.defaultConsentText === 'string'
              ? map.defaultConsentText
              : s.defaultConsentText,
          defaultPrivacyUrl:
            typeof map.defaultPrivacyUrl === 'string' ? map.defaultPrivacyUrl : s.defaultPrivacyUrl,
          notifyOnSubmission:
            typeof map.notifyOnSubmission === 'boolean'
              ? map.notifyOnSubmission
              : s.notifyOnSubmission,
          notifyEmail: typeof map.notifyEmail === 'string' ? map.notifyEmail : s.notifyEmail,
          defaultThankYouMessage:
            typeof map.defaultThankYouMessage === 'string'
              ? map.defaultThankYouMessage
              : s.defaultThankYouMessage,
          retentionDays:
            typeof map.retentionDays === 'number' ? map.retentionDays : s.retentionDays,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-formulare-tenant' },
  ),
)
