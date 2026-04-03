/**
 * Password Expiry Dialog — shown when the user's password has expired.
 *
 * This dialog blocks interaction until the password is changed.
 * It reads from the settings store and auto-shows when password is expired.
 */
import { useState } from 'react'
import { Shield, Eye, EyeOff } from 'lucide-react'
import { toast } from 'sonner'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
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
  // eslint-disable-next-line react-hooks/purity -- reading current date to compute password expiry status
  const daysSinceChange = lastChanged ? Math.floor((Date.now() - lastChanged.getTime()) / (1000 * 60 * 60 * 24)) : 0
  const isExpired = expiryDays > 0 && daysSinceChange >= expiryDays

  if (!isExpired) return null

  const handleSubmit = () => {
    if (!currentPw || !newPw) return
    if (newPw !== confirmPw) {
      toast.error('Passwörter stimmen nicht überein')
      return
    }
    if (newPw.length < 8) {
      toast.error('Passwort muss mindestens 8 Zeichen haben')
      return
    }
    updateSecurity({ passwordLastChanged: new Date().toISOString() })
    toast.success('Passwort erfolgreich geändert')
    setCurrentPw('')
    setNewPw('')
    setConfirmPw('')
  }

  return (
    <Dialog open={isExpired}>
      <DialogContent className="gap-0 p-0 max-w-md [&>button:last-child]:hidden">
        <DialogHeader className="px-6 pt-6 pb-0">
          <div className="flex items-center gap-3 mb-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-error/10">
              <Shield className="h-6 w-6 text-error" />
            </div>
            <div>
              <DialogTitle className="text-lg font-semibold text-foreground">Passwort abgelaufen</DialogTitle>
              <DialogDescription>
                Dein Passwort muss geändert werden bevor du fortfahren kannst.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-3 px-6 pb-2">
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
            <label className="block text-sm font-medium text-foreground">Passwort bestätigen</label>
            <Input
              type="password"
              value={confirmPw}
              onChange={(e) => setConfirmPw(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
            />
          </div>
        </div>

        <div className="px-6 pb-6 pt-2">
          <Button
            onClick={handleSubmit}
            className="w-full"
            disabled={!currentPw || !newPw || !confirmPw}
          >
            Passwort ändern
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
