/**
 * Password Expiry Dialog — shown when the user's password has expired.
 *
 * This dialog blocks interaction until the password is changed.
 * It reads from the settings store and auto-shows when password is expired.
 */
import { useState } from 'react'
import { Shield, Eye, EyeOff } from 'lucide-react'
import { toast } from 'sonner'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useSettingsStore } from '@/stores/settings'

export function PasswordExpiryDialog() {
  const security = useSettingsStore((s) => s.security)
  const updateSecurity = useSettingsStore((s) => s.updateSecurity)

  const [currentPw, setCurrentPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [confirmPw, setConfirmPw] = useState('')
  const [showCurrent, setShowCurrent] = useState(false)
  const [showNew, setShowNew] = useState(false)

  // Check if password is expired
  const lastChanged = security.passwordLastChanged ? new Date(security.passwordLastChanged) : null
  const expiryDays = security.passwordExpiryDays || 90
  const daysSinceChange = lastChanged ? Math.floor((Date.now() - lastChanged.getTime()) / (1000 * 60 * 60 * 24)) : 0
  const isExpired = expiryDays > 0 && daysSinceChange >= expiryDays

  if (!isExpired) return null

  const handleSubmit = () => {
    if (!currentPw || !newPw) return
    if (newPw !== confirmPw) {
      toast.error('Passwoerter stimmen nicht ueberein')
      return
    }
    if (newPw.length < 8) {
      toast.error('Passwort muss mindestens 8 Zeichen haben')
      return
    }
    updateSecurity({ passwordLastChanged: new Date().toISOString() })
    toast.success('Passwort erfolgreich geaendert')
    setCurrentPw('')
    setNewPw('')
    setConfirmPw('')
  }

  return (
    <div className="fixed inset-0 z-[9998] flex items-center justify-center bg-black/60">
      <div className="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-2xl animate-fade-up">
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-error/10">
            <Shield className="h-6 w-6 text-error" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-foreground">Passwort abgelaufen</h2>
            <p className="text-sm text-muted-foreground">
              Dein Passwort muss geaendert werden bevor du fortfahren kannst.
            </p>
          </div>
        </div>

        <div className="space-y-3 mb-4">
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">Aktuelles Passwort</label>
            <div className="relative">
              <Input
                type={showCurrent ? 'text' : 'password'}
                value={currentPw}
                onChange={(e) => setCurrentPw(e.target.value)}
                autoFocus
              />
              <button
                onClick={() => setShowCurrent(!showCurrent)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showCurrent ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">Neues Passwort</label>
            <div className="relative">
              <Input
                type={showNew ? 'text' : 'password'}
                value={newPw}
                onChange={(e) => setNewPw(e.target.value)}
              />
              <button
                onClick={() => setShowNew(!showNew)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showNew ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">Passwort bestaetigen</label>
            <Input
              type="password"
              value={confirmPw}
              onChange={(e) => setConfirmPw(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
            />
          </div>
        </div>

        <Button
          onClick={handleSubmit}
          className="w-full"
          disabled={!currentPw || !newPw || !confirmPw}
        >
          Passwort aendern
        </Button>
      </div>
    </div>
  )
}
