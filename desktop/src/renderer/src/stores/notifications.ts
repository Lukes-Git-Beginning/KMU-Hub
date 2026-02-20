import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type NotificationType =
  | 'message'
  | 'email'
  | 'task_assigned'
  | 'task_due'
  | 'meeting_soon'
  | 'ticket_new'
  | 'ticket_reply'
  | 'contract_expiry'
  | 'invoice_overdue'
  | 'mention'
  | 'system'

export type NotificationPriority = 'low' | 'normal' | 'high'

export interface Notification {
  id: string
  type: NotificationType
  title: string
  body: string
  icon: string
  priority: NotificationPriority
  isRead: boolean
  actionUrl?: string
  actorName?: string
  actorInitials?: string
  groupKey?: string
  createdAt: string
}

// ---------------------------------------------------------------------------
// State interface
// ---------------------------------------------------------------------------

interface NotificationsState {
  notifications: Notification[]
  isDropdownOpen: boolean

  // Computed-like
  unreadCount: () => number

  // Actions
  markAsRead: (id: string) => void
  markAllAsRead: () => void
  deleteNotification: (id: string) => void
  clearAll: () => void
  addNotification: (notification: Omit<Notification, 'id' | 'createdAt' | 'isRead'>) => void
  toggleDropdown: () => void
  closeDropdown: () => void
}

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

const mockNotifications: Notification[] = [
  {
    id: 'n1',
    type: 'message',
    title: 'Neue Nachricht von Anna Mueller',
    body: 'Hey, hast du die Entwürfe für das Dashboard gesehen?',
    icon: 'MessageSquare',
    priority: 'normal',
    isRead: false,
    actionUrl: '/chat',
    actorName: 'Anna Mueller',
    actorInitials: 'AM',
    createdAt: '2026-02-20T09:30:00',
  },
  {
    id: 'n2',
    type: 'task_assigned',
    title: 'Aufgabe zugewiesen',
    body: 'Peter Schmidt hat dir "API Dokumentation schreiben" zugewiesen',
    icon: 'CheckSquare',
    priority: 'normal',
    isRead: false,
    actionUrl: '/work',
    actorName: 'Peter Schmidt',
    actorInitials: 'PS',
    createdAt: '2026-02-20T08:45:00',
  },
  {
    id: 'n3',
    type: 'meeting_soon',
    title: 'Meeting in 15 Minuten',
    body: 'Sprint Planning — Raum Alpen, 10:00 Uhr',
    icon: 'Video',
    priority: 'high',
    isRead: false,
    actionUrl: '/meetings',
    createdAt: '2026-02-20T09:45:00',
  },
  {
    id: 'n4',
    type: 'ticket_new',
    title: 'Neues Support-Ticket #247',
    body: 'Kunde: Weber GmbH — "Login funktioniert nicht"',
    icon: 'LifeBuoy',
    priority: 'high',
    isRead: false,
    actionUrl: '/helpdesk',
    actorName: 'Weber GmbH',
    createdAt: '2026-02-20T08:15:00',
  },
  {
    id: 'n5',
    type: 'email',
    title: 'Neue E-Mail',
    body: 'Von: info@lieferant.de — Betreff: Bestellbestätigung #4521',
    icon: 'Mail',
    priority: 'normal',
    isRead: true,
    actionUrl: '/mails',
    createdAt: '2026-02-20T07:30:00',
  },
  {
    id: 'n6',
    type: 'contract_expiry',
    title: 'Vertrag läuft bald ab',
    body: 'Wartungsvertrag mit Tech Solutions AG — Ablauf in 30 Tagen',
    icon: 'FileWarning',
    priority: 'normal',
    isRead: true,
    actionUrl: '/vertraege',
    createdAt: '2026-02-19T16:00:00',
  },
  {
    id: 'n7',
    type: 'invoice_overdue',
    title: 'Rechnung überfällig',
    body: 'RE-2026-0042 — Müller & Partner — CHF 3.450,00 (fällig seit 7 Tagen)',
    icon: 'AlertCircle',
    priority: 'high',
    isRead: true,
    actionUrl: '/buchhaltung',
    createdAt: '2026-02-19T10:00:00',
  },
  {
    id: 'n8',
    type: 'mention',
    title: '@Erwähnung im Chat',
    body: 'Thomas Weber hat dich in #projekt-alpha erwähnt',
    icon: 'AtSign',
    priority: 'normal',
    isRead: true,
    actionUrl: '/chat',
    actorName: 'Thomas Weber',
    actorInitials: 'TW',
    createdAt: '2026-02-19T09:20:00',
  },
  {
    id: 'n9',
    type: 'task_due',
    title: 'Aufgabe fällig morgen',
    body: '"Kundenpräsentation vorbereiten" — Projekt: Website Relaunch',
    icon: 'Clock',
    priority: 'normal',
    isRead: true,
    actionUrl: '/work',
    createdAt: '2026-02-18T17:00:00',
  },
  {
    id: 'n10',
    type: 'system',
    title: 'System-Update verfügbar',
    body: 'KMU Hub v0.2.0 — Neue Features: Wiki, Kommunikation, verbesserte Suche',
    icon: 'Download',
    priority: 'low',
    isRead: true,
    createdAt: '2026-02-18T08:00:00',
  },
]

// ---------------------------------------------------------------------------
// ID counter
// ---------------------------------------------------------------------------

let nextId = 11

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useNotificationsStore = create<NotificationsState>()(
  persist(
    (set, get) => ({
      notifications: mockNotifications,
      isDropdownOpen: false,

      unreadCount: () => get().notifications.filter((n) => !n.isRead).length,

      markAsRead: (id) =>
        set((state) => ({
          notifications: state.notifications.map((n) =>
            n.id === id ? { ...n, isRead: true } : n,
          ),
        })),

      markAllAsRead: () =>
        set((state) => ({
          notifications: state.notifications.map((n) => ({ ...n, isRead: true })),
        })),

      deleteNotification: (id) =>
        set((state) => ({
          notifications: state.notifications.filter((n) => n.id !== id),
        })),

      clearAll: () => set({ notifications: [] }),

      addNotification: (notification) => {
        const id = `n${nextId++}`
        set((state) => ({
          notifications: [
            { ...notification, id, isRead: false, createdAt: new Date().toISOString() },
            ...state.notifications,
          ],
        }))
      },

      toggleDropdown: () =>
        set((state) => ({ isDropdownOpen: !state.isDropdownOpen })),

      closeDropdown: () => set({ isDropdownOpen: false }),
    }),
    {
      name: 'kmuhub-notifications',
    },
  ),
)
