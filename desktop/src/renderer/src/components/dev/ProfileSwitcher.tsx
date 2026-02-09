/**
 * Floating dev tool for switching between role profiles.
 *
 * Shows a small button in the bottom-left corner. Clicking opens
 * a panel with 5 mock profiles. Switching sets the auth store
 * user directly (Zustand external setState).
 *
 * Only visible when DEV_BYPASS_AUTH is enabled.
 */
import { useState } from 'react'
import { Users, ChevronUp, Check, X } from 'lucide-react'
import { useAuthStore } from '@/stores/auth'
import { DEV_PROFILES, type DevProfile } from '@/config/roles'

export function ProfileSwitcher() {
  const [isOpen, setIsOpen] = useState(false)
  const currentUser = useAuthStore((s) => s.user)

  const currentProfileId = DEV_PROFILES.find(
    (p) => p.user.id === currentUser?.id
  )?.id

  const switchProfile = (profile: DevProfile) => {
    useAuthStore.setState({
      user: profile.user,
      isAuthenticated: true,
    })
    setIsOpen(false)
  }

  return (
    <>
      {/* Floating trigger button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="fixed bottom-4 left-4 z-[100] flex items-center gap-2 rounded-full border border-border bg-card px-3 py-2 shadow-lg hover:shadow-xl transition-all hover:scale-105"
        title="Profil wechseln (Dev-Tool)"
      >
        {currentProfileId ? (
          <div
            className="flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-bold text-white"
            style={{ background: DEV_PROFILES.find((p) => p.id === currentProfileId)?.color }}
          >
            {DEV_PROFILES.find((p) => p.id === currentProfileId)?.initials}
          </div>
        ) : (
          <Users className="h-4 w-4 text-muted-foreground" />
        )}
        <span className="text-xs font-medium text-foreground hidden sm:inline">
          {currentProfileId
            ? DEV_PROFILES.find((p) => p.id === currentProfileId)?.label
            : 'Profil waehlen'}
        </span>
        <ChevronUp className={`h-3.5 w-3.5 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {/* Profile panel */}
      {isOpen && (
        <>
          {/* Backdrop */}
          <div className="fixed inset-0 z-[99]" onClick={() => setIsOpen(false)} />

          {/* Panel */}
          <div className="fixed bottom-16 left-4 z-[100] w-80 rounded-xl border border-border bg-card shadow-2xl overflow-hidden">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-border bg-secondary/30 px-4 py-3">
              <div>
                <p className="text-sm font-medium text-foreground">Rollen-Ansicht</p>
                <p className="text-[10px] text-muted-foreground">Wechsle die Perspektive — Sidebar & Einstellungen passen sich an</p>
              </div>
              <button onClick={() => setIsOpen(false)} className="text-muted-foreground hover:text-foreground">
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Profiles */}
            <div className="p-2 space-y-1">
              {DEV_PROFILES.map((profile) => {
                const isActive = currentProfileId === profile.id
                return (
                  <button
                    key={profile.id}
                    onClick={() => switchProfile(profile)}
                    className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-all ${
                      isActive
                        ? 'bg-primary/8 ring-1 ring-primary/30'
                        : 'hover:bg-secondary/70'
                    }`}
                  >
                    <div
                      className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
                      style={{ background: profile.color }}
                    >
                      {profile.initials}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-1.5">
                        <p className={`text-sm font-medium ${isActive ? 'text-primary' : 'text-foreground'}`}>
                          {profile.label}
                        </p>
                        {isActive && <Check className="h-3.5 w-3.5 text-primary" />}
                      </div>
                      <p className="text-[10px] text-muted-foreground truncate">{profile.description}</p>
                    </div>
                  </button>
                )
              })}
            </div>

            {/* Footer hint */}
            <div className="border-t border-border px-4 py-2 bg-secondary/20">
              <p className="text-[10px] text-muted-foreground text-center">
                Dev-Tool — nur sichtbar im Design-Modus
              </p>
            </div>
          </div>
        </>
      )}
    </>
  )
}
