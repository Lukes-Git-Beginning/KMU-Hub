import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Save, Receipt, FileSpreadsheet, Gavel, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { useSettingsStore } from '@/stores/settings'
import { useDunningConfig, useUpdateDunningConfig } from '@/api/hooks/useFinance'

export function FinanceSettingsTab({ embedded = false }: { embedded?: boolean } = {}) {
  const { t } = useTranslation()
  const { finance, updateFinance } = useSettingsStore()

  const [defaultVatRate, setDefaultVatRate] = useState(finance.defaultVatRate)
  const [invoicePrefix, setInvoicePrefix] = useState(finance.invoicePrefix)
  const [nextInvoiceNumber, setNextInvoiceNumber] = useState(finance.nextInvoiceNumber)
  const [defaultPaymentTerms, setDefaultPaymentTerms] = useState(finance.defaultPaymentTerms)
  const [datevClient, setDatevClient] = useState(finance.datevClientNumber)
  const [datevConsultant, setDatevConsultant] = useState(finance.datevConsultantNumber)

  // --- Dunning Config (from API) ---
  const { data: dunningConfig, isLoading: dunningLoading } = useDunningConfig()
  const dunningMutation = useUpdateDunningConfig()

  const [l1Days, setL1Days] = useState(14)
  const [l1Fee, setL1Fee] = useState('5.00')
  const [l2Days, setL2Days] = useState(14)
  const [l2Fee, setL2Fee] = useState('10.00')
  const [l3Days, setL3Days] = useState(14)
  const [l3Fee, setL3Fee] = useState('20.00')

   
  useEffect(() => {
    if (!dunningConfig) return
    // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
    setL1Days(dunningConfig.level1_days_after_due ?? 14)
    setL1Fee(dunningConfig.level1_fee ?? '5.00')
    setL2Days(dunningConfig.level2_days_after_level1 ?? 14)
    setL2Fee(dunningConfig.level2_fee ?? '10.00')
    setL3Days(dunningConfig.level3_days_after_level2 ?? 14)
    setL3Fee(dunningConfig.level3_fee ?? '20.00')
  }, [dunningConfig])

  const handleSaveInvoice = () => {
    updateFinance({ defaultVatRate, invoicePrefix, nextInvoiceNumber, defaultPaymentTerms })
    toast.success(t('settings.finance.savedInvoice'))
  }

  const handleSaveDatev = () => {
    updateFinance({ datevClientNumber: datevClient, datevConsultantNumber: datevConsultant })
    toast.success(t('settings.finance.savedDatev'))
  }

  const handleSaveDunning = () => {
    dunningMutation.mutate(
      {
        level1_days_after_due: l1Days,
        level1_fee: l1Fee,
        level2_days_after_level1: l2Days,
        level2_fee: l2Fee,
        level3_days_after_level2: l3Days,
        level3_fee: l3Fee,
      },
      { onSuccess: () => toast.success(t('settings.finance.savedDunning')) },
    )
  }

  return (
    <div className="max-w-2xl">
      {!embedded && (
        <>
          <h2 className="text-foreground mb-1">{t('settings.finance.title')}</h2>
          <p className="text-sm text-muted-foreground mb-6">
            {t('settings.finance.subtitle')}
          </p>
        </>
      )}

      {/* ── Invoice settings ─────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Receipt className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.finance.sectionInvoice')}</h3>
        </div>
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('settings.finance.invoicePrefix')}</Label>
              <Input value={invoicePrefix} onChange={(e) => setInvoicePrefix(e.target.value)} placeholder="RE-" />
            </div>
            <div className="space-y-1.5">
              <Label>{t('settings.finance.nextNumber')}</Label>
              <Input type="number" value={nextInvoiceNumber} onChange={(e) => setNextInvoiceNumber(Number(e.target.value))} />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('settings.finance.vatRate')}</Label>
              <Select value={String(defaultVatRate)} onValueChange={(v) => setDefaultVatRate(Number(v))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="0">{t('settings.finance.vat.exempt')}</SelectItem>
                  <SelectItem value="2.6">{t('settings.finance.vat.reduced1')}</SelectItem>
                  <SelectItem value="3.8">{t('settings.finance.vat.special')}</SelectItem>
                  <SelectItem value="7.7">{t('settings.finance.vat.reducedOld')}</SelectItem>
                  <SelectItem value="8.1">{t('settings.finance.vat.normal')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('settings.finance.paymentTerms')}</Label>
              <Select value={defaultPaymentTerms} onValueChange={setDefaultPaymentTerms}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="Sofort fällig">{t('settings.finance.terms.immediate')}</SelectItem>
                  <SelectItem value="10 Tage netto">{t('settings.finance.terms.10days')}</SelectItem>
                  <SelectItem value="30 Tage netto">{t('settings.finance.terms.30days')}</SelectItem>
                  <SelectItem value="60 Tage netto">{t('settings.finance.terms.60days')}</SelectItem>
                  <SelectItem value="10 Tage 2% Skonto, 30 Tage netto">{t('settings.finance.terms.skonto')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-secondary/30 p-3">
            <p className="text-xs text-muted-foreground">
              {t('settings.finance.invoicePreview')}: <span className="font-mono font-medium text-foreground">{invoicePrefix}{nextInvoiceNumber}</span>
            </p>
          </div>
        </div>
        <Button onClick={handleSaveInvoice} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.finance.saveInvoice')}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── DATEV ────────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <FileSpreadsheet className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.finance.sectionDatev')}</h3>
        </div>
        <p className="text-xs text-muted-foreground mb-3">{t('settings.finance.datevDesc')}</p>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <Label>{t('settings.finance.datevClientNr')}</Label>
            <Input value={datevClient} onChange={(e) => setDatevClient(e.target.value)} placeholder="z.B. 12345" className="font-mono" />
          </div>
          <div className="space-y-1.5">
            <Label>{t('settings.finance.datevConsultantNr')}</Label>
            <Input value={datevConsultant} onChange={(e) => setDatevConsultant(e.target.value)} placeholder="z.B. 67890" className="font-mono" />
          </div>
        </div>
        <Button onClick={handleSaveDatev} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.finance.saveDatev')}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Mahnwesen ────────────────────────────────── */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Gavel className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.finance.sectionDunning')}</h3>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          {t('settings.finance.dunningDesc')}
        </p>

        {dunningLoading ? (
          <div className="flex items-center gap-2 text-muted-foreground py-4">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span className="text-sm">{t('settings.finance.dunningLoading')}</span>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              {/* Stufe 1 */}
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-2 mb-3">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-warning-light text-warning text-xs font-bold">1</span>
                  <span className="text-sm font-medium text-foreground">{t('settings.finance.dunning.level1Name')}</span>
                </div>
                <div className="space-y-2">
                  <div className="space-y-1">
                    <Label className="text-xs">{t('settings.finance.dunning.daysAfterDue')}</Label>
                    <Input type="number" min={1} value={l1Days} onChange={(e) => setL1Days(Number(e.target.value))} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">{t('settings.finance.dunning.fee')}</Label>
                    <Input value={l1Fee} onChange={(e) => setL1Fee(e.target.value)} className="font-mono" />
                  </div>
                </div>
              </div>

              {/* Stufe 2 */}
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-2 mb-3">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-warning-light text-warning text-xs font-bold">2</span>
                  <span className="text-sm font-medium text-foreground">{t('settings.finance.dunning.level2Name')}</span>
                </div>
                <div className="space-y-2">
                  <div className="space-y-1">
                    <Label className="text-xs">{t('settings.finance.dunning.daysAfterLevel1')}</Label>
                    <Input type="number" min={1} value={l2Days} onChange={(e) => setL2Days(Number(e.target.value))} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">{t('settings.finance.dunning.fee')}</Label>
                    <Input value={l2Fee} onChange={(e) => setL2Fee(e.target.value)} className="font-mono" />
                  </div>
                </div>
              </div>

              {/* Stufe 3 */}
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-2 mb-3">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-error-light text-destructive text-xs font-bold">3</span>
                  <span className="text-sm font-medium text-foreground">{t('settings.finance.dunning.level3Name')}</span>
                </div>
                <div className="space-y-2">
                  <div className="space-y-1">
                    <Label className="text-xs">{t('settings.finance.dunning.daysAfterLevel2')}</Label>
                    <Input type="number" min={1} value={l3Days} onChange={(e) => setL3Days(Number(e.target.value))} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">{t('settings.finance.dunning.fee')}</Label>
                    <Input value={l3Fee} onChange={(e) => setL3Fee(e.target.value)} className="font-mono" />
                  </div>
                </div>
              </div>
            </div>

            <Button onClick={handleSaveDunning} className="mt-4" size="sm" disabled={dunningMutation.isPending}>
              <Save className="mr-1.5 h-4 w-4" />
              {t('settings.finance.saveDunning')}
            </Button>
          </>
        )}
      </section>
    </div>
  )
}
