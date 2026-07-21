/**
 * View-as store (RBAC R-5 — Part C.2).
 *
 * Implements the product-level "Als Benutzer anzeigen" feature
 * (admin:impersonate:run capability required). Differs from the dev-only
 * ProfileSwitcher in that:
 *   - It is a product feature, NOT gated on DEV_BYPASS_AUTH.
 *   - It writes an audit event on start.
 *   - It tracks the origin userId so exitViewAs() restores it.
 *   - It is session-only (never persisted to localStorage).
 *
 * Integration points:
 *   1. UserDetailModal — shows "Als Benutzer anzeigen" button.
 *   2. ViewAsBanner — persistent banner in the app shell.
 *   3. permissions store — refreshes via useAuthStore subscription on
 *      setState, identical to the ProfileSwitcher pattern.
 */
import { create } from 'zustand'
import { useAuthStore, type User } from '@/stores/auth'
import { setDemoSessionUserId } from '@/mocks/data/rbac'
import { writeAuditEvent } from '@/mocks/data/audit-events'

export interface ViewAsState {
  /** The user being impersonated; null = not in view-as mode. */
  viewAsUser: User | null
  /** The original admin's userId to restore on exit. */
  originUserId: string | null

  /** Start viewing as the given user. Writes audit event. */
  startViewAs: (targetUser: User) => void
  /** Exit view-as, restoring the original session. */
  exitViewAs: () => void
}

export const useViewAsStore = create<ViewAsState>()((set, get) => ({
  viewAsUser: null,
  originUserId: null,

  startViewAs(targetUser: User) {
    const currentUser = useAuthStore.getState().user
    if (!currentUser) return

    // Capture origin before switching
    const originUserId = currentUser.id

    // Write audit event BEFORE switching so actorId = admin
    writeAuditEvent({
      action: 'user.view_as',
      target: `${targetUser.firstName} ${targetUser.lastName}`,
      targetType: 'user',
      actorId: originUserId,
      extraDetails: { target_user_id: targetUser.id },
    })

    // Switch the MSW demo session + auth store (same pattern as ProfileSwitcher)
    setDemoSessionUserId(targetUser.id)
    useAuthStore.setState({
      user: targetUser,
      isAuthenticated: true,
    })
    // The permissions store subscription on useAuthStore picks up the user change
    // and calls refresh() automatically — no manual invalidation needed.

    set({ viewAsUser: targetUser, originUserId })
  },

  exitViewAs() {
    const { originUserId } = get()
    if (!originUserId) return

    // We need to restore the original user object. The DEMO_PROFILES list
    // in mocks/data/rbac carries the full user objects for preset identities.
    // For the demo, the origin will always be the admin (CURRENT_USER).
    // We restore by importing the auth store reset pattern.
    import('@/mocks/data/rbac').then(({ DEMO_PROFILES }) => {
      const origin = DEMO_PROFILES.find((p) => p.user.id === originUserId)
      if (!origin) return

      setDemoSessionUserId(origin.user.id)
      useAuthStore.setState({
        user: origin.user,
        isAuthenticated: true,
      })

      set({ viewAsUser: null, originUserId: null })
    })
  },
}))
