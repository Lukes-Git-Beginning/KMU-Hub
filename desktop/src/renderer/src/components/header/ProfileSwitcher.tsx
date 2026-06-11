import { useState, useRef, useEffect } from 'react'
import { Check, ChevronDown, Plus, Settings } from 'lucide-react'
import { useProfileStore } from '@/stores/profile'
import { cn } from '@/lib/cn'
import { useTranslation } from 'react-i18next'

export function ProfileSwitcher() {
  const { t } = useTranslation()
  const profiles = useProfileStore((s) => s.profiles)
  const activeProfileId = useProfileStore((s) => s.activeProfileId)
  const switchProfile = useProfileStore((s) => s.switchProfile)
  const createProfile = useProfileStore((s) => s.createProfile)

  const activeProfile = profiles.find((p) => p.id === activeProfileId) ?? profiles[0]

  const [isOpen, setIsOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

   
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false)
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () =>
        document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  const handleSwitch = (profileId: string) => {
    switchProfile(profileId)
    setIsOpen(false)
  }

  const handleCreateProfile = () => {
    const count = profiles.length + 1
    createProfile({
      name: t('header.profileSwitcher.newProfileName', { count }),
      description: t('header.profileSwitcher.customProfileDescription'),
      icon: '\uD83D\uDCCC',
      color: '#6366f1',
      isDefault: false,
    })
  }

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-1 px-2 py-1.5 rounded-lg hover:bg-accent transition-colors"
      >
        {activeProfile?.icon ? (
          <span className="text-base">{activeProfile.icon}</span>
        ) : (
          <Settings className="h-4 w-4 text-muted-foreground" />
        )}
        <ChevronDown
          className={cn(
            'h-3.5 w-3.5 text-muted-foreground transition-transform',
            isOpen && 'rotate-180',
          )}
        />
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-80 bg-card border border-border rounded-lg shadow-xl z-50 overflow-hidden">
          {/* Header */}
          <div className="px-4 py-3 border-b border-border">
            <h3 className="font-semibold text-foreground">
              {t('header.profileSwitcher.title')}
            </h3>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t('header.profileSwitcher.quickSwitchHint')}
            </p>
          </div>

          {/* Profile List */}
          <div className="max-h-96 overflow-y-auto">
            {profiles.map((profile, index) => (
              <button
                key={profile.id}
                onClick={() => handleSwitch(profile.id)}
                className={cn(
                  'w-full px-4 py-3 flex items-start gap-3 hover:bg-accent transition-colors border-l-2',
                  profile.id === activeProfileId
                    ? 'border-primary bg-primary/5'
                    : 'border-transparent',
                )}
              >
                <span className="text-2xl shrink-0">{profile.icon}</span>

                <div className="flex-1 text-left">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-foreground">
                      {profile.name}
                    </span>
                    {profile.isDefault && <span className="text-xs">&#11088;</span>}
                    {index < 9 && (
                      <span className="text-xs text-muted-foreground ml-auto">
                        Ctrl+{index + 1}
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5 line-clamp-1">
                    {profile.description}
                  </p>
                </div>

                {profile.id === activeProfileId && (
                  <Check className="h-4 w-4 text-primary shrink-0" />
                )}
              </button>
            ))}
          </div>

          {/* Actions */}
          <div className="border-t border-border p-3 space-y-2">
            <button
              onClick={handleCreateProfile}
              className="w-full px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors flex items-center justify-center gap-2 text-sm font-medium"
            >
              <Plus className="h-4 w-4" />
              {t('header.profileSwitcher.createNewProfile')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
