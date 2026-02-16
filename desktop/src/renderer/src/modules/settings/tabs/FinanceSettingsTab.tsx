import { useState } from 'react'
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
import { Save, Building2, Landmark, Receipt, FileSpreadsheet } from 'lucide-react'
import { toast } from 'sonner'
import { useSettingsStore } from '@/stores/settings'

export function FinanceSettingsTab() {
  const { finance, updateFinance } = useSettingsStore()

  const [companyName, setCompanyName] = useState(finance.companyName)
  const [companyAddress, setCompanyAddress] = useState(finance.companyAddress)
  const [bankName, setBankName] = useState(finance.bankName)
  const [iban, setIban] = useState(finance.iban)
  const [bic, setBic] = useState(finance.bic)
  const [vatNumber, setVatNumber] = useState(finance.vatNumber)
  const [defaultVatRate, setDefaultVatRate] = useState(finance.defaultVatRate)
  const [invoicePrefix, setInvoicePrefix] = useState(finance.invoicePrefix)
  const [nextInvoiceNumber, setNextInvoiceNumber] = useState(finance.nextInvoiceNumber)
  const [defaultPaymentTerms, setDefaultPaymentTerms] = useState(finance.defaultPaymentTerms)
  const [datevClient, setDatevClient] = useState(finance.datevClientNumber)
  const [datevConsultant, setDatevConsultant] = useState(finance.datevConsultantNumber)

  const handleSaveCompany = () => {
    updateFinance({ companyName, companyAddress, vatNumber })
    toast.success('Firmendaten gespeichert')
  }

  const handleSaveBank = () => {
    updateFinance({ bankName, iban, bic })
    toast.success('Bankdaten gespeichert')
  }

  const handleSaveInvoice = () => {
    updateFinance({ defaultVatRate, invoicePrefix, nextInvoiceNumber, defaultPaymentTerms })
    toast.success('Rechnungseinstellungen gespeichert')
  }

  const handleSaveDatev = () => {
    updateFinance({ datevClientNumber: datevClient, datevConsultantNumber: datevConsultant })
    toast.success('DATEV-Einstellungen gespeichert')
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">Buchhaltung</h2>
      <p className="text-sm text-muted-foreground mb-6">Firmen-Info, Bankdaten, Rechnungsnummern und DATEV konfigurieren</p>

      {/* Company info */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Building2 className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Firmendaten</h3>
        </div>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label>Firmenname</Label>
            <Input value={companyName} onChange={(e) => setCompanyName(e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>Adresse</Label>
            <Input value={companyAddress} onChange={(e) => setCompanyAddress(e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>MwSt-Nummer</Label>
            <Input value={vatNumber} onChange={(e) => setVatNumber(e.target.value)} placeholder="CHE-xxx.xxx.xxx MWST" />
          </div>
        </div>
        <Button onClick={handleSaveCompany} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          Firmendaten speichern
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* Bank */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Landmark className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Bankverbindung</h3>
        </div>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label>Bank</Label>
            <Input value={bankName} onChange={(e) => setBankName(e.target.value)} />
          </div>
          <div className="grid grid-cols-[1fr_150px] gap-3">
            <div className="space-y-1.5">
              <Label>IBAN</Label>
              <Input value={iban} onChange={(e) => setIban(e.target.value)} className="font-mono text-xs" />
            </div>
            <div className="space-y-1.5">
              <Label>BIC/SWIFT</Label>
              <Input value={bic} onChange={(e) => setBic(e.target.value)} className="font-mono text-xs" />
            </div>
          </div>
        </div>
        <Button onClick={handleSaveBank} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          Bankdaten speichern
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* Invoice settings */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Receipt className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Rechnungseinstellungen</h3>
        </div>
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Rechnungs-Präfix</Label>
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

      {/* DATEV */}
      <section>
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
    </div>
  )
}
