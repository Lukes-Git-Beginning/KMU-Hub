import { useState, useEffect } from 'react'
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

export function FinanceSettingsTab() {
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
    toast.success('Rechnungseinstellungen gespeichert')
  }

  const handleSaveDatev = () => {
    updateFinance({ datevClientNumber: datevClient, datevConsultantNumber: datevConsultant })
    toast.success('DATEV-Einstellungen gespeichert')
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
      { onSuccess: () => toast.success('Mahnwesen-Konfiguration gespeichert') },
    )
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">Buchhaltung</h2>
      <p className="text-sm text-muted-foreground mb-6">
        Rechnungsnummern, DATEV-Export und Mahnwesen konfigurieren
      </p>

      {/* ── Invoice settings ─────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Receipt className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Rechnungseinstellungen</h3>
        </div>
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Rechnungs-Praefix</Label>
              <Input value={invoicePrefix} onChange={(e) => setInvoicePrefix(e.target.value)} placeholder="RE-" />
            </div>
            <div className="space-y-1.5">
              <Label>Nächste Nummer</Label>
              <Input type="number" value={nextInvoiceNumber} onChange={(e) => setNextInvoiceNumber(Number(e.target.value))} />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Standard MwSt-Satz</Label>
              <Select value={String(defaultVatRate)} onValueChange={(v) => setDefaultVatRate(Number(v))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="0">0% (Befreit)</SelectItem>
                  <SelectItem value="2.6">2.6% (Reduziert)</SelectItem>
                  <SelectItem value="3.8">3.8% (Sondersatz)</SelectItem>
                  <SelectItem value="7.7">7.7% (Reduziert alt)</SelectItem>
                  <SelectItem value="8.1">8.1% (Normal)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Zahlungsbedingungen</Label>
              <Select value={defaultPaymentTerms} onValueChange={setDefaultPaymentTerms}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="Sofort fällig">Sofort fällig</SelectItem>
                  <SelectItem value="10 Tage netto">10 Tage netto</SelectItem>
                  <SelectItem value="30 Tage netto">30 Tage netto</SelectItem>
                  <SelectItem value="60 Tage netto">60 Tage netto</SelectItem>
                  <SelectItem value="10 Tage 2% Skonto, 30 Tage netto">10/2%, 30 netto</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-secondary/30 p-3">
            <p className="text-xs text-muted-foreground">
              Vorschau nächste Rechnung: <span className="font-mono font-medium text-foreground">{invoicePrefix}{nextInvoiceNumber}</span>
            </p>
          </div>
        </div>
        <Button onClick={handleSaveInvoice} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          Rechnungseinstellungen speichern
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── DATEV ────────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <FileSpreadsheet className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">DATEV-Export</h3>
        </div>
        <p className="text-xs text-muted-foreground mb-3">Für den automatischen Export an deinen Steuerberater</p>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <Label>Mandanten-Nummer</Label>
            <Input value={datevClient} onChange={(e) => setDatevClient(e.target.value)} placeholder="z.B. 12345" className="font-mono" />
          </div>
          <div className="space-y-1.5">
            <Label>Berater-Nummer</Label>
            <Input value={datevConsultant} onChange={(e) => setDatevConsultant(e.target.value)} placeholder="z.B. 67890" className="font-mono" />
          </div>
        </div>
        <Button onClick={handleSaveDatev} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          DATEV speichern
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Mahnwesen ────────────────────────────────── */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Gavel className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Mahnwesen</h3>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          Konfiguriere die drei Mahnstufen für überfällige Rechnungen
        </p>

        {dunningLoading ? (
          <div className="flex items-center gap-2 text-muted-foreground py-4">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span className="text-sm">Mahnkonfiguration wird geladen...</span>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              {/* Stufe 1 */}
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-2 mb-3">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-warning-light text-warning text-xs font-bold">1</span>
                  <span className="text-sm font-medium text-foreground">Zahlungserinnerung</span>
                </div>
                <div className="space-y-2">
                  <div className="space-y-1">
                    <Label className="text-xs">Tage nach Fälligkeit</Label>
                    <Input type="number" min={1} value={l1Days} onChange={(e) => setL1Days(Number(e.target.value))} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Mahngebühr (CHF)</Label>
                    <Input value={l1Fee} onChange={(e) => setL1Fee(e.target.value)} className="font-mono" />
                  </div>
                </div>
              </div>

              {/* Stufe 2 */}
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-2 mb-3">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-warning-light text-warning text-xs font-bold">2</span>
                  <span className="text-sm font-medium text-foreground">2. Mahnung</span>
                </div>
                <div className="space-y-2">
                  <div className="space-y-1">
                    <Label className="text-xs">Tage nach Stufe 1</Label>
                    <Input type="number" min={1} value={l2Days} onChange={(e) => setL2Days(Number(e.target.value))} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Mahngebühr (CHF)</Label>
                    <Input value={l2Fee} onChange={(e) => setL2Fee(e.target.value)} className="font-mono" />
                  </div>
                </div>
              </div>

              {/* Stufe 3 */}
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-2 mb-3">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-error-light text-destructive text-xs font-bold">3</span>
                  <span className="text-sm font-medium text-foreground">Letzte Mahnung</span>
                </div>
                <div className="space-y-2">
                  <div className="space-y-1">
                    <Label className="text-xs">Tage nach Stufe 2</Label>
                    <Input type="number" min={1} value={l3Days} onChange={(e) => setL3Days(Number(e.target.value))} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Mahngebühr (CHF)</Label>
                    <Input value={l3Fee} onChange={(e) => setL3Fee(e.target.value)} className="font-mono" />
                  </div>
                </div>
              </div>
            </div>

            <Button onClick={handleSaveDunning} className="mt-4" size="sm" disabled={dunningMutation.isPending}>
              <Save className="mr-1.5 h-4 w-4" />
              Mahnwesen speichern
            </Button>
          </>
        )}
      </section>
    </div>
  )
}
