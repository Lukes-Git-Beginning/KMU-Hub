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
      profile: {
        firstName: 'Darien',
        lastName: 'Morales',
        email: 'darien@firma.ch',
        phone: '+41 79 123 45 67',
        position: 'UI/UX Designer',
        bio: 'Designer mit Leidenschaft fuer schoene und funktionale Interfaces.',
        avatarUrl: null,
      },

      appearance: {
        theme: 'light',
        fontSize: 16,
        deskTheme: 'default',
      },

      language: {
        locale: 'de',
        timezone: 'Europe/Zurich',
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
        imapHost: 'imap.firma.ch',
        imapPort: 993,
        smtpHost: 'smtp.firma.ch',
        smtpPort: 587,
        username: 'darien@firma.ch',
        signature: '<p>Mit freundlichen Gruessen<br/>Darien Morales<br/>UI/UX Designer</p>',
        autoReplyEnabled: false,
        autoReplyMessage: 'Vielen Dank fuer Ihre Nachricht. Ich bin derzeit nicht im Buero und werde mich nach meiner Rueckkehr bei Ihnen melden.',
      },

      calendar: {
        defaultView: 'week',
        workStartHour: 8,
        workEndHour: 17,
        defaultReminder: 15,
        holidayRegion: 'CH-ZH',
        weekStartsOn: 'monday',
      },

      finance: {
        companyName: 'Morales Design GmbH',
        companyAddress: 'Bahnhofstrasse 42, 8001 Zuerich',
        bankName: 'Zuercher Kantonalbank',
        iban: 'CH93 0070 0110 0000 5000 6',
        bic: 'ZKBKCHZZ80A',
        vatNumber: 'CHE-123.456.789 MWST',
        defaultVatRate: 8.1,
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
        set({
          security: {
            twoFactorEnabled: true,
            twoFactorSecret: 'JBSWY3DPEHPK3PXP',
            backupCodes: ['A1B2-C3D4', 'E5F6-G7H8', 'J9K0-L1M2', 'N3P4-Q5R6', 'S7T8-U9V0', 'W1X2-Y3Z4'],
          },
        }),

      disable2FA: () =>
        set({
          security: {
            twoFactorEnabled: false,
            twoFactorSecret: null,
            backupCodes: [],
          },
        }),
    }),
    { name: 'kmuhub-settings' },
  ),
)
