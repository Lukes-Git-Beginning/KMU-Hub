/**
 * Mail — personal preferences (user scope, see ModuleSettingsShell): default
 * sending account, desktop notifications, conversation view. The tenant-wide
 * compliance toggles live in stores/mailTenant.ts.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/mail/user):
 * initFromServer() hydrates once per session (via useHydrateModuleSettings);
 * each setter writes through. localStorage stays the optimistic cache.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

const MODULE_ID = 'mail'

interface MailPrefsState {
  defaultAccountId: string
  desktopNotifications: boolean
  conversationView: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultAccount: (id: string) => void
  setDesktopNotifications: (v: boolean) => void
  setConversationView: (v: boolean) => void
  initFromServer: () => Promise<void>
}

function userPayload(s: MailPrefsState): Record<string, unknown> {
  return {
    defaultAccountId: s.defaultAccountId,
    desktopNotifications: s.desktopNotifications,
    conversationView: s.conversationView,
  }
}

export const useMailPrefsStore = create<MailPrefsState>()(
  persist(
    (set, get) => ({
      defaultAccountId: '',
      desktopNotifications: true,
      conversationView: true,
      serverInitialized: false,
      setDefaultAccount: (defaultAccountId) => {
        set({ defaultAccountId })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDesktopNotifications: (desktopNotifications) => {
        set({ desktopNotifications })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setConversationView: (conversationView) => {
        set({ conversationView })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultAccountId:
            typeof map.defaultAccountId === 'string' ? map.defaultAccountId : s.defaultAccountId,
          desktopNotifications:
            typeof map.desktopNotifications === 'boolean'
              ? map.desktopNotifications
              : s.desktopNotifications,
          conversationView:
            typeof map.conversationView === 'boolean' ? map.conversationView : s.conversationView,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-mail-prefs' },
  ),
)
