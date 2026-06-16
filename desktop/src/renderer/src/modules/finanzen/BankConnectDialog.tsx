/**
 * BankConnectDialog — simulierter FinAPI/PSD2-Bankverbindungs-Flow (P2.5e-Fix).
 *
 * Drei Schritte: Bank auswählen → bei der Bank anmelden → Verbindungsaufbau.
 * Keine echte Bank-Anbindung (Demo); am Ende wird das Konto über den
 * stateful connect-Handler verbunden. Echte Anbindung folgt mit FinAPI (P5).
 */
import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Landmark, Search, ChevronRight, Lock, ShieldCheck, Loader2, ArrowLeft } from 'lucide-react'
import { toast } from 'sonner'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import type { BankAccount } from '@/types/finance-types'
import { useConnectBankAccount } from '@/api/hooks/useFinance'

// Gängige DACH-Banken für die Auswahl (Demo).
const BANKS = [
  'Sparkasse',
  'Volksbank Raiffeisenbank',
  'Deutsche Bank',
  'Commerzbank',
  'ING',
  'DKB',
  'Postbank',
  'HypoVereinsbank',
  'comdirect',
  'N26',
  'Targobank',
  'Consorsbank',
]

const AVATAR_COLORS = [
  'bg-rose-500/15 text-rose-600',
  'bg-sky-500/15 text-sky-600',
  'bg-amber-500/15 text-amber-600',
  'bg-emerald-500/15 text-emerald-600',
  'bg-violet-500/15 text-violet-600',
  'bg-orange-500/15 text-orange-600',
]
const colorFor = (name: string) =>
  AVATAR_COLORS[name.charCodeAt(0) % AVATAR_COLORS.length]

interface Props {
  account: BankAccount
  open: boolean
  onClose: () => void
}

export function BankConnectDialog({ account, open, onClose }: Props) {
  const { t } = useTranslation()
  const connectMutation = useConnectBankAccount()
  const [step, setStep] = useState<'select' | 'login' | 'connecting'>('select')
  const [search, setSearch] = useState('')
  const [selectedBank, setSelectedBank] = useState<string | null>(null)
  const [loginName, setLoginName] = useState('')
  const [pin, setPin] = useState('')

  // Beim Öffnen: Karten-Bank als Default-Auswahl, Felder zurücksetzen.
  useEffect(() => {
    if (open) {
      const matched = BANKS.find((b) => account.bankName.toLowerCase().includes(b.toLowerCase()))
      setSelectedBank(matched ?? null)
      setStep('select')
      setSearch('')
      setLoginName('')
      setPin('')
    }
  }, [open, account.bankName])

  const filteredBanks = useMemo(
    () => BANKS.filter((b) => b.toLowerCase().includes(search.toLowerCase())),
    [search],
  )

  const handleConnect = () => {
    setStep('connecting')
    // FinAPI-Weiterleitung + Freigabe simulieren, dann Konto verbinden.
    setTimeout(() => {
      connectMutation.mutate(account.id, {
        onSuccess: () => { toast.success(t('finanzen.banking.connected')); onClose() },
        onError: (err) => { toast.error(err.message); onClose() },
      })
    }, 1600)
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Landmark className="h-5 w-5 text-primary" />
            {t('finanzen.banking.connectTitle')}
          </DialogTitle>
        </DialogHeader>

        {/* Step 1: Bank auswählen */}
        {step === 'select' && (
          <div className="space-y-3">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                autoFocus
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('finanzen.banking.searchBank')}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="max-h-72 space-y-1 overflow-y-auto">
              {filteredBanks.map((bank) => (
                <button
                  key={bank}
                  type="button"
                  onClick={() => { setSelectedBank(bank); setStep('login') }}
                  className={`group flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors ${
                    selectedBank === bank ? 'border-primary bg-primary/5' : 'border-border hover:bg-accent/40'
                  }`}
                >
                  <span className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-xs font-semibold ${colorFor(bank)}`}>
                    {bank.charAt(0)}
                  </span>
                  <span className="flex-1 text-sm font-medium text-foreground">{bank}</span>
                  <ChevronRight className="h-4 w-4 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
                </button>
              ))}
              {filteredBanks.length === 0 && (
                <p className="py-6 text-center text-xs text-muted-foreground">{t('finanzen.banking.noBankFound')}</p>
              )}
            </div>
          </div>
        )}

        {/* Step 2: Login */}
        {step === 'login' && selectedBank && (
          <div className="space-y-4">
            <div className="flex items-center gap-3 rounded-lg border border-border bg-secondary/30 px-3 py-2.5">
              <span className={`flex h-9 w-9 items-center justify-center rounded-lg text-sm font-semibold ${colorFor(selectedBank)}`}>
                {selectedBank.charAt(0)}
              </span>
              <div>
                <p className="text-sm font-medium text-foreground">{selectedBank}</p>
                <p className="text-[11px] text-muted-foreground">{account.iban}</p>
              </div>
            </div>

            <div className="space-y-2.5">
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-foreground">{t('finanzen.banking.loginName')}</span>
                <input
                  type="text"
                  value={loginName}
                  onChange={(e) => setLoginName(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-foreground">{t('finanzen.banking.pin')}</span>
                <input
                  type="password"
                  value={pin}
                  onChange={(e) => setPin(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </label>
            </div>

            <div className="flex items-start gap-2 rounded-md bg-secondary/50 px-3 py-2">
              <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0 text-success" />
              <p className="text-[11px] text-muted-foreground">{t('finanzen.banking.psd2Hint')}</p>
            </div>
            <p className="text-[10px] text-warning">{t('finanzen.banking.demoHint')}</p>

            <div className="flex justify-between pt-1">
              <Button variant="outline" size="sm" onClick={() => setStep('select')}>
                <ArrowLeft className="mr-1.5 h-4 w-4" />
                {t('common.back')}
              </Button>
              <Button size="sm" onClick={handleConnect} disabled={!loginName || !pin}>
                <Lock className="mr-1.5 h-4 w-4" />
                {t('finanzen.banking.secureConnect')}
              </Button>
            </div>
          </div>
        )}

        {/* Step 3: Verbindungsaufbau */}
        {step === 'connecting' && (
          <div className="flex flex-col items-center gap-3 py-10">
            <Loader2 className="h-7 w-7 animate-spin text-primary" />
            <p className="text-sm font-medium text-foreground">{t('finanzen.banking.connecting')}</p>
            <p className="text-xs text-muted-foreground">{selectedBank}</p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
