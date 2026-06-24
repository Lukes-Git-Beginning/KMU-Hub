/**
 * Settings store — Zustand with localStorage persistence + server sync.
 *
 * Server-sync strategy:
 *   - On app boot call `initFromServer()` once per session to hydrate the store
 *     from backend (user-scope settings, modules: appearance/language/notifications/calendar).
 *   - On every action that changes a server-synced section, call the returned
 *     `serverSync` helper (debounced PUT via useUserSettings hook or direct fetch).
 *   - Sections not yet server-backed (mail, finance, teamAdmin, security) stay
 *     localStorage-only until the corresponding backend endpoints exist.
 *
 * Wire: GET/PUT /api/v1/settings/{module_id}/user
 *   Response: { entries: [{ key: string; value: unknown }] }
 *   Body:     { settings: { [key: string]: unknown } }
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// ---- Notification matrix ----
export interface NotificationPrefs {
  email: boolean
  push: boolean
  inApp: boolean
}

const DEFAULT_NOTIF: NotificationPrefs = { email: true, push: true, inApp: true }

export type NotificationModule = 'messages' | 'tasks' | 'meetings' | 'mails' | 'calendar' | 'team' | 'finance'

// ---- Sub-setting interfaces ----
export interface ProfileSettings {
  firstName: string
  lastName: string
  email: string
  phone: string
  position: string
  bio: string
  avatarUrl: string | null
  /** Demo "member since" date (ISO). Mirrors the authenticated user's join
   *  date; the real backend will supply this via /auth/me in production. */
  joinedAt: string
}

export interface AppearanceSettings {
  theme: 'light' | 'dark' | 'auto'
  fontSize: number
  deskTheme: string
}

export interface LanguageSettings {
  locale: 'de' | 'en' | 'fr'
  timezone: string
  dateFormat: string
}

export interface MailSettings {
  imapHost: string
  imapPort: number
  smtpHost: string
  smtpPort: number
  username: string
  signature: string
  autoReplyEnabled: boolean
  autoReplyMessage: string
}

export interface CalendarSettings {
  defaultView: 'week' | 'day' | 'month'
  workStartHour: number
  workEndHour: number
  defaultReminder: number
  holidayRegion: string
  weekStartsOn: 'monday' | 'sunday'
}

export interface FinanceSettings {
  companyName: string
  companyAddress: string
  bankName: string
  iban: string
  bic: string
  vatNumber: string
  defaultVatRate: number
  invoicePrefix: string
  nextInvoiceNumber: number
  defaultPaymentTerms: string
  datevClientNumber: string
  datevConsultantNumber: string
}

export interface LeaveType {
  name: string
  days: number
  color: string
}

export interface TeamAdminSettings {
  departments: string[]
  roles: string[]
  leaveTypes: LeaveType[]
  workHoursPerWeek: number
  overtimeEnabled: boolean
}

export interface SecuritySettings {
  twoFactorEnabled: boolean
  twoFactorSecret: string | null
  backupCodes: string[]
  passwordLastChanged: string // ISO date
  passwordExpiryDays: number // 0 = no expiry
}

// ---- Full state ----
interface SettingsState {
  profile: ProfileSettings
  appearance: AppearanceSettings
  language: LanguageSettings
  notifications: Record<NotificationModule, NotificationPrefs>
  mail: MailSettings
  calendar: CalendarSettings
  finance: FinanceSettings
  teamAdmin: TeamAdminSettings
  security: SecuritySettings

  /** Whether the store has been hydrated from the backend at least once this session. */
  serverInitialized: boolean

  // Actions
  updateProfile: (data: Partial<ProfileSettings>) => void
  updateAppearance: (data: Partial<AppearanceSettings>) => void
  updateLanguage: (data: Partial<LanguageSettings>) => void
  updateNotification: (module: NotificationModule, channel: keyof NotificationPrefs, value: boolean) => void
  updateMail: (data: Partial<MailSettings>) => void
  updateCalendar: (data: Partial<CalendarSettings>) => void
  updateFinance: (data: Partial<FinanceSettings>) => void
  updateTeamAdmin: (data: Partial<TeamAdminSettings>) => void
  updateSecurity: (data: Partial<SecuritySettings>) => void
  enable2FA: () => void
  disable2FA: () => void

  /**
   * Hydrates appearance / language / notifications / calendar from the backend.
   * Call once on app boot (e.g. in AppShell after auth). Idempotent within a session.
   *
   * Server wins over localStorage for these sections so that cross-device settings
   * propagate correctly. Sections not yet server-backed (mail, finance, teamAdmin,
   * security) are left as-is.
   */
  initFromServer: () => Promise<void>

  /**
   * Persist a single section to the backend (user-scope settings PUT).
   * Debounce is the caller's responsibility (e.g. call inside updateAppearance etc.).
   * Internal use — exported so components can trigger a manual save if needed.
   */
  saveSection: (moduleId: string, data: Record<string, unknown>) => Promise<void>
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set, get) => ({
      // Seed mirrors the authenticated demo user (Stefan Vogel / usr-e1, the
      // single source of truth in mocks/data/shared-ids.ts → CURRENT_USER and
      // /auth/me). Kept as plain literals so mock data never enters the prod
      // bundle; the real backend overwrites these from the user profile.
      profile: {
        firstName: 'Stefan',
        lastName: 'Vogel',
        email: 'stefan.vogel@techvision.de',
        phone: '+49 151 2345 6789',
        position: 'Geschäftsführer',
        bio: 'Geschäftsführer der TechVision GmbH — Fokus auf nachhaltige Digitalisierung im Mittelstand.',
        avatarUrl: null,
        joinedAt: '2021-03-01',
      },

      appearance: {
        theme: 'light',
        fontSize: 16,
        deskTheme: 'default',
      },

      language: {
        locale: 'de',
        timezone: 'Europe/Berlin',
        dateFormat: 'DD.MM.YYYY',
      },

      notifications: {
        messages: { ...DEFAULT_NOTIF },
        tasks: { email: true, push: false, inApp: true },
        meetings: { email: false, push: true, inApp: true },
        mails: { email: true, push: false, inApp: true },
        calendar: { email: false, push: true, inApp: true },
        team: { email: true, push: false, inApp: true },
        finance: { email: true, push: false, inApp: false },
      },

      mail: {
        imapHost: 'imap.techvision.de',
        imapPort: 993,
        smtpHost: 'smtp.techvision.de',
        smtpPort: 587,
        username: 'stefan.vogel@techvision.de',
        signature: '<p>Mit freundlichen Grüßen<br/>Stefan Vogel<br/>Geschäftsführer</p>',
        autoReplyEnabled: false,
        autoReplyMessage: 'Vielen Dank für Ihre Nachricht. Ich bin derzeit nicht im Büro und werde mich nach meiner Rückkehr bei Ihnen melden.',
      },

      calendar: {
        defaultView: 'week',
        workStartHour: 8,
        workEndHour: 17,
        defaultReminder: 15,
        holidayRegion: 'DE-BY',
        weekStartsOn: 'monday',
      },

      finance: {
        companyName: 'Morales Design GmbH',
        companyAddress: 'Leopoldstraße 42, 80802 München',
        bankName: 'Commerzbank AG',
        iban: 'DE89 3704 0044 0532 0130 00',
        bic: 'COBADEFFXXX',
        vatNumber: 'DE136695976',
        defaultVatRate: 19,
        invoicePrefix: 'RE-',
        nextInvoiceNumber: 2026001,
        defaultPaymentTerms: '30 Tage netto',
        datevClientNumber: '',
        datevConsultantNumber: '',
      },

      teamAdmin: {
        departments: ['Entwicklung', 'Design', 'Marketing', 'Vertrieb', 'HR', 'Finanzen'],
        roles: ['Admin', 'Projektleiter', 'Mitarbeiter', 'Praktikant', 'Extern'],
        leaveTypes: [
          { name: 'Urlaub', days: 25, color: '#3b82f6' },
          { name: 'Krankheit', days: 0, color: '#ef4444' },
          { name: 'Homeoffice', days: 0, color: '#8b5cf6' },
          { name: 'Weiterbildung', days: 5, color: '#f59e0b' },
          { name: 'Sonderurlaub', days: 3, color: '#06b6d4' },
        ],
        workHoursPerWeek: 42,
        overtimeEnabled: true,
      },

      security: {
        twoFactorEnabled: false,
        twoFactorSecret: null,
        backupCodes: [],
        passwordLastChanged: new Date(Date.now() - 45 * 24 * 60 * 60 * 1000).toISOString(),
        passwordExpiryDays: 90,
      },

      serverInitialized: false,

      // ---- Actions ----
      updateProfile: (data) => set((s) => ({ profile: { ...s.profile, ...data } })),
      updateAppearance: (data) => {
        const next = { ...get().appearance, ...data }
        set({ appearance: next })
        void get().saveSection('appearance', next as Record<string, unknown>)
      },
      updateLanguage: (data) => {
        const next = { ...get().language, ...data }
        set({ language: next })
        void get().saveSection('language', next as Record<string, unknown>)
      },

      updateNotification: (module, channel, value) => {
        const next = {
          ...get().notifications,
          [module]: { ...get().notifications[module], [channel]: value },
        }
        set({ notifications: next })
        void get().saveSection('notifications', next as Record<string, unknown>)
      },

      updateMail: (data) => set((s) => ({ mail: { ...s.mail, ...data } })),
      updateCalendar: (data) => {
        const next = { ...get().calendar, ...data }
        set({ calendar: next })
        void get().saveSection('calendar', next as Record<string, unknown>)
      },
      updateFinance: (data) => set((s) => ({ finance: { ...s.finance, ...data } })),
      updateTeamAdmin: (data) => set((s) => ({ teamAdmin: { ...s.teamAdmin, ...data } })),
      updateSecurity: (data) => set((s) => ({ security: { ...s.security, ...data } })),

      enable2FA: () =>
        set((s) => ({
          security: {
            ...s.security,
            twoFactorEnabled: true,
            twoFactorSecret: 'JBSWY3DPEHPK3PXP',
            backupCodes: ['A1B2-C3D4', 'E5F6-G7H8', 'J9K0-L1M2', 'N3P4-Q5R6', 'S7T8-U9V0', 'W1X2-Y3Z4'],
          },
        })),

      disable2FA: () =>
        set((s) => ({
          security: {
            ...s.security,
            twoFactorEnabled: false,
            twoFactorSecret: null,
            backupCodes: [],
          },
        })),

      // ---- Server sync ----

      saveSection: async (moduleId, data) => {
        try {
          const { authenticatedRequest } = await import('@/api/utils/authenticatedFetch')
          await authenticatedRequest({
            method: 'PUT',
            path: `/api/v1/settings/${moduleId}/user`,
            body: { settings: data },
          })
        } catch {
          // Silent: settings save failures are non-critical.
          // The store already updated; next initFromServer will re-sync on next session.
        }
      },

      initFromServer: async () => {
        if (get().serverInitialized) return

        try {
          const { authenticatedRequest } = await import('@/api/utils/authenticatedFetch')

          type EntriesResp = { entries?: Array<{ key: string; value: unknown }> }

          const toMap = (resp: EntriesResp): Record<string, unknown> => {
            if (!resp.entries || resp.entries.length === 0) return {}
            return Object.fromEntries(resp.entries.map((e) => [e.key, e.value]))
          }

          const [appearanceResp, languageResp, notificationsResp, calendarResp] = await Promise.allSettled([
            authenticatedRequest<EntriesResp>({ method: 'GET', path: '/api/v1/settings/appearance/user' }),
            authenticatedRequest<EntriesResp>({ method: 'GET', path: '/api/v1/settings/language/user' }),
            authenticatedRequest<EntriesResp>({ method: 'GET', path: '/api/v1/settings/notifications/user' }),
            authenticatedRequest<EntriesResp>({ method: 'GET', path: '/api/v1/settings/calendar/user' }),
          ])

          set((s) => {
            const updates: Partial<SettingsState> = { serverInitialized: true }

            if (appearanceResp.status === 'fulfilled') {
              const m = toMap(appearanceResp.value)
              if (Object.keys(m).length > 0) {
                updates.appearance = { ...s.appearance, ...m } as AppearanceSettings
              }
            }

            if (languageResp.status === 'fulfilled') {
              const m = toMap(languageResp.value)
              if (Object.keys(m).length > 0) {
                updates.language = { ...s.language, ...m } as LanguageSettings
                // Sync the active i18n locale store so the UI language reflects the
                // server value without triggering a PUT back (quiet = no side-effect).
                if (typeof m.locale === 'string' && m.locale) {
                  const { useLocaleStore } = await import('./locale')
                  const { SUPPORTED_LOCALES } = await import('./locale')
                  if ((SUPPORTED_LOCALES as readonly string[]).includes(m.locale)) {
                    useLocaleStore.getState().setLocaleQuiet(m.locale as import('./locale').SupportedLocale)
                  }
                }
              }
            }

            if (notificationsResp.status === 'fulfilled') {
              const m = toMap(notificationsResp.value)
              if (Object.keys(m).length > 0) {
                // Backend stores notifications as a flat object keyed by module name.
                // Each value is itself an object { email, push, inApp }.
                updates.notifications = {
                  ...s.notifications,
                  ...(m as Record<string, NotificationPrefs>),
                }
              }
            }

            if (calendarResp.status === 'fulfilled') {
              const m = toMap(calendarResp.value)
              if (Object.keys(m).length > 0) {
                updates.calendar = { ...s.calendar, ...m } as CalendarSettings
              }
            }

            return updates
          })
        } catch {
          // Silent: initFromServer failure leaves the store at localStorage values.
          set({ serverInitialized: true })
        }
      },
    }),
    { name: 'cosmi-settings' },
  ),
)
