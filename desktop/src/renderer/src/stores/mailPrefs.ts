import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Mail module preferences (demo-stateful, persisted locally).
 * - personal: default sending account, desktop notifications, conversation view.
 * - tenant (demo): compliance toggles (retention badges, external-image policy).
 * Server config + signature + auto-reply live in the shared settings store
 * (useSettingsStore().mail) and are reused by the tenant section.
 */
interface MailPrefsState {
  // personal
  defaultAccountId: string
  desktopNotifications: boolean
  conversationView: boolean
  // tenant (demo)
  showRetentionBadges: boolean
  loadExternalImages: boolean

  setDefaultAccount: (id: string) => void
  setDesktopNotifications: (v: boolean) => void
  setConversationView: (v: boolean) => void
  setShowRetentionBadges: (v: boolean) => void
  setLoadExternalImages: (v: boolean) => void
}

export const useMailPrefsStore = create<MailPrefsState>()(
  persist(
    (set) => ({
      defaultAccountId: '',
      desktopNotifications: true,
      conversationView: true,
      showRetentionBadges: true,
      loadExternalImages: false,
      setDefaultAccount: (defaultAccountId) => set({ defaultAccountId }),
      setDesktopNotifications: (desktopNotifications) => set({ desktopNotifications }),
      setConversationView: (conversationView) => set({ conversationView }),
      setShowRetentionBadges: (showRetentionBadges) => set({ showRetentionBadges }),
      setLoadExternalImages: (loadExternalImages) => set({ loadExternalImages }),
    }),
    { name: 'cosmi-mail-prefs' },
  ),
)
