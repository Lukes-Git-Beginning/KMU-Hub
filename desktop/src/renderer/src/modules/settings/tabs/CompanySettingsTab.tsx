import { useState, useEffect } from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import {
  Save,
  Building2,
  Scale,
  Landmark,
  Palette,
  Loader2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useCompanySettings, useUpdateCompanySettings } from '@/api/hooks/useFinance'

export function CompanySettingsTab() {
  const { data: settings, isLoading } = useCompanySettings()
  const updateMutation = useUpdateCompanySettings()

  // --- Firmendaten ---
  const [name, setName] = useState('')
  const [street, setStreet] = useState('')
  const [plz, setPlz] = useState('')
  const [city, setCity] = useState('')
  const [country, setCountry] = useState('CH')

  // --- Steuer & Recht ---
  const [steuernummer, setSteuernummer] = useState('')
  const [ustIdNr, setUstIdNr] = useState('')
  const [handelsregister, setHandelsregister] = useState('')
  const [isKleinunternehmer, setIsKleinunternehmer] = useState(false)

  // --- Bankdaten ---
  const [bankName, setBankName] = useState('')
  const [iban, setIban] = useState('')
  const [bic, setBic] = useState('')

  // --- Defaults ---
  const [logoUrl, setLogoUrl] = useState('')
  const [accentColor, setAccentColor] = useState('#1e7e74')
  const [paymentTermsDays, setPaymentTermsDays] = useState(30)
  const [quoteValidityDays, setQuoteValidityDays] = useState(30)

   
  useEffect(() => {
    if (!settings) return
    // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
    setName(settings.name ?? '')
    setStreet(settings.street ?? '')
    setPlz(settings.plz ?? '')
    setCity(settings.city ?? '')
    setCountry(settings.country ?? 'CH')
    setSteuernummer(settings.steuernummer ?? '')
    setUstIdNr(settings.ust_id_nr ?? '')
    setHandelsregister(settings.handelsregister ?? '')
    setIsKleinunternehmer(settings.is_kleinunternehmer ?? false)
    setBankName(settings.bank_name ?? '')
    setIban(settings.iban ?? '')
    setBic(settings.bic ?? '')
    setLogoUrl(settings.logo_url ?? '')
    setAccentColor(settings.accent_color ?? '#1e7e74')
    setPaymentTermsDays(settings.default_payment_terms_days ?? 30)
    setQuoteValidityDays(settings.default_quote_validity_days ?? 30)
  }, [settings])

  const handleSaveCompany = () => {
    updateMutation.mutate(
      { name, street, plz, city, country },
      { onSuccess: () => toast.success('Firmendaten gespeichert') },
    )
  }

  const handleSaveTax = () => {
    updateMutation.mutate(
      {
        steuernummer,
        ust_id_nr: ustIdNr,
        handelsregister,
        is_kleinunternehmer: isKleinunternehmer,
      },
      { onSuccess: () => toast.success('Steuer & Recht gespeichert') },
    )
  }

  const handleSaveBank = () => {
    updateMutation.mutate(
      { bank_name: bankName, iban, bic },
      { onSuccess: () => toast.success('Bankdaten gespeichert') },
    )
  }

  const handleSaveDefaults = () => {
    updateMutation.mutate(
      {
        logo_url: logoUrl || undefined,
        accent_color: accentColor || undefined,
        default_payment_terms_days: paymentTermsDays,
        default_quote_validity_days: quoteValidityDays,
      },
      { onSuccess: () => toast.success('Standardwerte gespeichert') },
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-muted-foreground py-12">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="text-sm">Firmendaten werden geladen...</span>
      </div>
    )
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">Firma</h2>
      <p className="text-sm text-muted-foreground mb-6">
        Stammdaten, Steuerinformationen und Bankverbindung verwalten
      </p>

      {/* ── Firmendaten ──────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Building2 className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Firmendaten</h3>
        </div>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label>Firmenname</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Muster GmbH" />
          </div>
          <div className="space-y-1.5">
            <Label>Strasse</Label>
            <Input value={street} onChange={(e) => setStreet(e.target.value)} placeholder="Bahnhofstrasse 42" />
          </div>
          <div className="grid grid-cols-[120px_1fr] gap-3">
            <div className="space-y-1.5">
              <Label>PLZ</Label>
              <Input value={plz} onChange={(e) => setPlz(e.target.value)} placeholder="8001" />
            </div>
            <div className="space-y-1.5">
              <Label>Ort</Label>
              <Input value={city} onChange={(e) => setCity(e.target.value)} placeholder="Zürich" />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label>Land</Label>
            <Input value={country} onChange={(e) => setCountry(e.target.value)} placeholder="CH" />
          </div>
        </div>
        <Button onClick={handleSaveCompany} className="mt-4" size="sm" disabled={updateMutation.isPending}>
          <Save className="mr-1.5 h-4 w-4" />
          Firmendaten speichern
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Steuer & Recht ───────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Scale className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Steuer & Recht</h3>
        </div>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label>Steuernummer</Label>
            <Input
              value={steuernummer}
              onChange={(e) => setSteuernummer(e.target.value)}
              placeholder="123/456/78901"
              className="font-mono text-xs"
            />
          </div>
          <div className="space-y-1.5">
            <Label>USt-IdNr.</Label>
            <Input
              value={ustIdNr}
              onChange={(e) => setUstIdNr(e.target.value)}
              placeholder="CHE-123.456.789 MWST"
              className="font-mono text-xs"
            />
          </div>
          <div className="space-y-1.5">
            <Label>Handelsregister</Label>
            <Input
              value={handelsregister}
              onChange={(e) => setHandelsregister(e.target.value)}
              placeholder="HRB 12345, Amtsgericht Zürich"
            />
          </div>
          <div className="flex items-center justify-between rounded-lg border border-border bg-card p-3">
            <div>
              <p className="text-sm text-foreground">Kleinunternehmer-Regelung</p>
              <p className="text-xs text-muted-foreground">Keine MwSt-Ausweisung auf Rechnungen</p>
            </div>
            <Switch checked={isKleinunternehmer} onCheckedChange={setIsKleinunternehmer} />
          </div>
        </div>
        <Button onClick={handleSaveTax} className="mt-4" size="sm" disabled={updateMutation.isPending}>
          <Save className="mr-1.5 h-4 w-4" />
          Steuer & Recht speichern
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Bankdaten ────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Landmark className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Bankverbindung</h3>
        </div>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label>Bank</Label>
            <Input value={bankName} onChange={(e) => setBankName(e.target.value)} placeholder="Zuercher Kantonalbank" />
          </div>
          <div className="grid grid-cols-[1fr_150px] gap-3">
            <div className="space-y-1.5">
              <Label>IBAN</Label>
              <Input value={iban} onChange={(e) => setIban(e.target.value)} className="font-mono text-xs" placeholder="CH93 0070 0110 0000 5000 6" />
            </div>
            <div className="space-y-1.5">
              <Label>BIC/SWIFT</Label>
              <Input value={bic} onChange={(e) => setBic(e.target.value)} className="font-mono text-xs" placeholder="ZKBKCHZZ80A" />
            </div>
          </div>
        </div>
        <Button onClick={handleSaveBank} className="mt-4" size="sm" disabled={updateMutation.isPending}>
          <Save className="mr-1.5 h-4 w-4" />
          Bankdaten speichern
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Defaults ─────────────────────────────────── */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Palette className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Standardwerte</h3>
        </div>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label>Logo-URL</Label>
            <Input value={logoUrl} onChange={(e) => setLogoUrl(e.target.value)} placeholder="https://firma.ch/logo.png" />
          </div>
          <div className="space-y-1.5">
            <Label>Akzentfarbe</Label>
            <div className="flex items-center gap-3">
              <input
                type="color"
                value={accentColor}
                onChange={(e) => setAccentColor(e.target.value)}
                className="h-9 w-9 rounded border border-border cursor-pointer"
              />
              <Input value={accentColor} onChange={(e) => setAccentColor(e.target.value)} className="font-mono text-xs w-32" />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Standard-Zahlungsziel (Tage)</Label>
              <Input
                type="number"
                min={0}
                value={paymentTermsDays}
                onChange={(e) => setPaymentTermsDays(Number(e.target.value))}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Angebots-Gültigkeit (Tage)</Label>
              <Input
                type="number"
                min={0}
                value={quoteValidityDays}
                onChange={(e) => setQuoteValidityDays(Number(e.target.value))}
              />
            </div>
          </div>
        </div>
        <Button onClick={handleSaveDefaults} className="mt-4" size="sm" disabled={updateMutation.isPending}>
          <Save className="mr-1.5 h-4 w-4" />
          Standardwerte speichern
        </Button>
      </section>
    </div>
  )
}
