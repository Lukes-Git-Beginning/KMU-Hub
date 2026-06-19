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
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
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
        vatNumber: 'DE123456789',
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

      // ---- Actions ----
      updateProfile: (data) => set((s) => ({ profile: { ...s.profile, ...data } })),
      updateAppearance: (data) => set((s) => ({ appearance: { ...s.appearance, ...data } })),
      updateLanguage: (data) => set((s) => ({ language: { ...s.language, ...data } })),

      updateNotification: (module, channel, value) =>
        set((s) => ({
          notifications: {
            ...s.notifications,
            [module]: { ...s.notifications[module], [channel]: value },
          },
        })),

      updateMail: (data) => set((s) => ({ mail: { ...s.mail, ...data } })),
      updateCalendar: (data) => set((s) => ({ calendar: { ...s.calendar, ...data } })),
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
    }),
    { name: 'cosmi-settings' },
  ),
)
