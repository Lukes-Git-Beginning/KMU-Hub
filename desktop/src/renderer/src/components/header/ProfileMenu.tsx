import { useState, useRef, useEffect } from 'react'
import { Settings, HelpCircle, ChevronDown } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { useAuthStore } from '@/stores/auth'
import { cn } from '@/lib/cn'

export function ProfileMenu() {
  const [isOpen, setIsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)

  const firstName = user?.firstName ?? 'User'
  const lastName = user?.lastName ?? ''
  const email = user?.email ?? ''
  const initials = `${firstName.charAt(0)}${lastName.charAt(0)}`.toUpperCase()
  const role =
    user?.roles?.includes('admin')
      ? 'Administrator'
      : user?.roles?.includes('manager')
        ? 'Projektleiter'
        : 'Mitarbeiter'

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () =>
        document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  const handleSettings = () => {
    navigate('/einstellungen')
    setIsOpen(false)
  }

  return (
    <div ref={menuRef} className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 md:gap-3 pl-2 md:pl-4 border-l border-border hover:opacity-80 transition-opacity"
      >
        <div className="hidden md:block text-right">
          <p className="text-sm text-foreground">
            {firstName} {lastName}
          </p>
          <p className="text-xs text-muted-foreground">{role}</p>
        </div>
        <Avatar className="ring-2 ring-transparent hover:ring-primary transition-all">
          <AvatarFallback>{initials}</AvatarFallback>
        </Avatar>
        <ChevronDown
          className={cn(
            'h-4 w-4 text-muted-foreground transition-transform duration-300',
            isOpen && 'rotate-180',
          )}
        />
      </button>

      {isOpen && (
        <>
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />

          <div className="absolute top-full right-0 mt-2 w-80 md:w-96 bg-card border border-border rounded-lg shadow-xl z-50 overflow-hidden">
            {/* User Info */}
            <button
              onClick={() => {
                navigate('/profil')
                setIsOpen(false)
              }}
              className="p-4 border-b border-border bg-gradient-to-br from-primary/10 to-card hover:from-primary/20 transition-all w-full text-left"
            >
              <div className="flex items-center gap-3">
                <Avatar className="h-12 w-12 ring-2 ring-primary">
                  <AvatarFallback className="text-lg">{initials}</AvatarFallback>
                </Avatar>
                <div>
                  <p className="font-medium text-foreground">
                    {firstName} {lastName}
                  </p>
                  <p className="text-sm text-muted-foreground">{email}</p>
                </div>
              </div>
            </button>

            {/* Action Buttons */}
            <div className="p-3 space-y-2">
              <button
                onClick={handleSettings}
                className="w-full flex items-center gap-3 px-3 py-2 text-foreground hover:bg-accent rounded-lg transition-colors"
              >
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
                  <Settings className="h-4 w-4 text-primary" />
                </div>
                <span className="text-sm">Einstellungen</span>
              </button>

              <button
                onClick={() => setIsOpen(false)}
                className="w-full flex items-center gap-3 px-3 py-2 text-foreground hover:bg-accent rounded-lg transition-colors"
              >
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500/10">
                  <HelpCircle className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                </div>
                <span className="text-sm">Hilfe & Support</span>
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
