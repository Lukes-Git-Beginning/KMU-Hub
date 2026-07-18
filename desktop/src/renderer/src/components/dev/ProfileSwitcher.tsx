/**
 * Floating dev tool for switching between role profiles and business profiles.
 *
 * Shows a small button in the bottom-left corner. Clicking opens
 * a panel with:
 * - the RBAC preset demo identities (7 roles + multi-role combo, from
 *   mocks/data/rbac DEMO_PROFILES — switching reloads effective permissions)
 * - 10 business/industry profiles (module visibility)
 * - "Show all modules" dev toggle
 *
 * Only visible when DEV_BYPASS_AUTH is enabled.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Users, ChevronUp, Check, X, Building2, Eye, RotateCcw } from 'lucide-react'
import { useAuthStore } from '@/stores/auth'
import { roleLabelKey } from '@/config/roles'
import { DEMO_PROFILES, ROLE_DEFS, setDemoSessionUserId, type DemoProfile } from '@/mocks/data/rbac'
import { BUSINESS_PROFILES, type BusinessProfileId } from '@/config/business-profiles'
import { useProfileStore } from '@/stores/profile'

type PanelTab = 'roles' | 'business'

export function ProfileSwitcher() {
  const { t } = useTranslation()
  const [isOpen, setIsOpen] = useState(false)
  const [tab, setTab] = useState<PanelTab>('roles')
  const currentUser = useAuthStore((s) => s.user)

  const businessProfileId = useProfileStore((s) => s.businessProfileId)
  const devShowAll = useProfileStore((s) => s.devShowAllModules)
  const setBusinessProfile = useProfileStore((s) => s.setBusinessProfile)
  const toggleDevShowAll = useProfileStore((s) => s.toggleDevShowAll)

  const currentProfile = DEMO_PROFILES.find((p) => p.user.id === currentUser?.id)

  const currentBusinessProfile = BUSINESS_PROFILES.find((p) => p.id === businessProfileId)

  const switchRoleProfile = (profile: DemoProfile) => {
    // Order matters: the MSW session must know the new account before the
    // permissions store (auth-store subscription) refetches /auth/me/permissions.
    setDemoSessionUserId(profile.user.id)
    useAuthStore.setState({
      user: profile.user,
      isAuthenticated: true,
    })
  }

  /** Translated role chain of a profile, e.g. "Teamleiter + HR-Admin". */
  const roleChain = (profile: DemoProfile) =>
    profile.user.roles.map((r) => t(roleLabelKey(r))).join(' + ')

  const switchBusinessProfile = (id: BusinessProfileId | null) => {
    setBusinessProfile(id)
  }

  return (
    <>
      {/* Floating trigger button — bottom-RIGHT so it never covers the
          sidebar's bottom nav entries (Modul-Einstellungen). */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="fixed bottom-4 right-4 z-[100] flex items-center gap-2 rounded-full border border-border bg-card px-3 py-2 shadow-lg hover:shadow-xl transition-all hover:scale-105 glass-elevated"
        title={t('devTools.switchProfileTooltip')}
      >
        {currentProfile ? (
          <div
            className="flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-bold text-white"
            style={{ background: ROLE_DEFS[currentProfile.roleId].color }}
          >
            {currentProfile.initials}
          </div>
        ) : (
          <Users className="h-4 w-4 text-muted-foreground" />
        )}
        <span className="text-xs font-medium text-foreground hidden sm:inline">
          {currentProfile ? roleChain(currentProfile) : t('devTools.noProfile')}
        </span>
        {currentBusinessProfile && (
          <span className="text-xs text-muted-foreground hidden sm:inline">
            | {currentBusinessProfile.emoji}
          </span>
        )}
        <ChevronUp className={`h-3.5 w-3.5 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {/* Profile panel */}
      {isOpen && (
        <>
          {/* Backdrop */}
          <div className="fixed inset-0 z-[99]" onClick={() => setIsOpen(false)} />

          {/* Panel */}
          <div className="fixed bottom-16 right-4 z-[100] w-80 rounded-xl border border-border bg-card shadow-2xl overflow-hidden glass-elevated">
            {/* Header with tabs */}
            <div className="border-b border-border bg-secondary/30">
              <div className="flex items-center justify-between px-4 pt-3 pb-1">
                <p className="text-sm font-medium text-foreground">{t('devTools.title')}</p>
                <button onClick={() => setIsOpen(false)} className="text-muted-foreground hover:text-foreground">
                  <X className="h-4 w-4" />
                </button>
              </div>
              <div className="flex px-2 gap-1 pb-2">
                <button
                  onClick={() => setTab('roles')}
                  className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                    tab === 'roles' ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  <Users className="h-3.5 w-3.5" />
                  {t('devTools.tab.roles')}
                </button>
                <button
                  onClick={() => setTab('business')}
                  className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                    tab === 'business' ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  <Building2 className="h-3.5 w-3.5" />
                  {t('devTools.tab.industry')}
                </button>
              </div>
            </div>

            {/* Tab content */}
            <div className="max-h-80 overflow-y-auto">
              {tab === 'roles' ? (
                <div className="p-2 space-y-1">
                  {DEMO_PROFILES.map((profile) => {
                    const isActive = currentProfile?.user.id === profile.user.id
                    return (
                      <button
                        key={profile.user.id}
                        onClick={() => switchRoleProfile(profile)}
                        className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-all ${
                          isActive
                            ? 'bg-primary/8 ring-1 ring-primary/30'
                            : 'hover:bg-secondary/70'
                        }`}
                      >
                        <div
                          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
                          style={{ background: ROLE_DEFS[profile.roleId].color }}
                        >
                          {profile.initials}
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-1.5">
                            <p className={`text-sm font-medium ${isActive ? 'text-primary' : 'text-foreground'}`}>
                              {roleChain(profile)}
                            </p>
                            {isActive && <Check className="h-3.5 w-3.5 text-primary" />}
                          </div>
                          <p className="text-[10px] text-muted-foreground truncate">
                            {profile.user.firstName} {profile.user.lastName} · {t(`rbac.roles.${profile.roleId}.description`)}
                          </p>
                        </div>
                      </button>
                    )
                  })}
                </div>
              ) : (
                <div className="p-2 space-y-1">
                  {/* No profile option */}
                  <button
                    onClick={() => switchBusinessProfile(null)}
                    className={`flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-all ${
                      !businessProfileId
                        ? 'bg-primary/8 ring-1 ring-primary/30'
                        : 'hover:bg-secondary/70'
                    }`}
                  >
                    <span className="text-xl w-8 text-center shrink-0">&#x2699;&#xFE0F;</span>
                    <div className="flex-1 min-w-0">
                      <p className={`text-sm font-medium ${!businessProfileId ? 'text-primary' : 'text-foreground'}`}>
                        {t('devTools.noProfileAllModules')}
                      </p>
                    </div>
                    {!businessProfileId && <Check className="h-3.5 w-3.5 text-primary shrink-0" />}
                  </button>

                  {BUSINESS_PROFILES.map((profile) => {
                    const isActive = businessProfileId === profile.id
                    return (
                      <button
                        key={profile.id}
                        onClick={() => switchBusinessProfile(profile.id)}
                        className={`flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-all ${
                          isActive
                            ? 'bg-primary/8 ring-1 ring-primary/30'
                            : 'hover:bg-secondary/70'
                        }`}
                      >
                        <span className="text-xl w-8 text-center shrink-0">{profile.emoji}</span>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-1.5">
                            <p className={`text-sm font-medium ${isActive ? 'text-primary' : 'text-foreground'}`}>
                              {profile.name}
                            </p>
                            {isActive && <Check className="h-3.5 w-3.5 text-primary shrink-0" />}
                          </div>
                          <p className="text-[10px] text-muted-foreground truncate">
                            {profile.examples.slice(0, 3).join(', ')}
                          </p>
                        </div>
                      </button>
                    )
                  })}
                </div>
              )}
            </div>

            {/* Footer with dev toggle + reset */}
            <div className="border-t border-border px-4 py-2.5 bg-secondary/20 space-y-2">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={devShowAll}
                  onChange={toggleDevShowAll}
                  className="rounded border-border"
                />
                <Eye className="h-3.5 w-3.5 text-muted-foreground" />
                <span className="text-[11px] text-muted-foreground">
                  {t('devTools.showAllModules')}
                </span>
              </label>
              <button
                onClick={() => {
                  const keys = Object.keys(localStorage).filter((k) => k.startsWith('cosmi-'))
                  keys.forEach((k) => localStorage.removeItem(k))
                  window.location.reload()
                }}
                className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-[11px] text-destructive hover:bg-destructive/10 transition-colors"
              >
                <RotateCcw className="h-3.5 w-3.5" />
                {t('devTools.resetDemo')}
              </button>
            </div>
          </div>
        </>
      )}
    </>
  )
}
