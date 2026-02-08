import { useState, useRef, useEffect } from 'react'
import { Check, ChevronDown, Plus } from 'lucide-react'
import { useProfileStore, type WorkProfile } from '@/stores/profile'
import { cn } from '@/lib/cn'

export function ProfileSwitcher() {
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
      name: `Profil ${count}`,
      description: 'Benutzerdefiniertes Profil',
      icon: '\uD83D\uDCCC',
      color: '#6366f1',
      isDefault: false,
    })
  }

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-2 rounded-lg hover:bg-accent transition-colors"
      >
        <span className="text-lg">{activeProfile?.icon ?? '\u2699\uFE0F'}</span>
        <div className="hidden md:flex flex-col items-start">
          <span className="text-xs text-muted-foreground">Profil</span>
          <span className="text-sm font-medium text-foreground">
            {activeProfile?.name}
          </span>
        </div>
        <ChevronDown
          className={cn(
            'h-4 w-4 text-muted-foreground transition-transform',
            isOpen && 'rotate-180',
          )}
        />
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-80 bg-card border border-border rounded-lg shadow-xl z-50 overflow-hidden">
          {/* Header */}
          <div className="px-4 py-3 border-b border-border">
            <h3 className="font-semibold text-foreground">
              Arbeitsprofile wechseln
            </h3>
            <p className="text-xs text-muted-foreground mt-0.5">
              Ctrl + Shift + [1-9] fuer Schnellwechsel
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
              Neues Profil erstellen
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
