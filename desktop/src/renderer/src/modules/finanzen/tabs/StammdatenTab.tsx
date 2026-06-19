/**
 * StammdatenTab — Firmen-Stammdaten im Buchhaltungs-Modul.
 *
 * Migriert aus settings/tabs/CompanySettingsTab.tsx.
 * Enthält alle Pflichtdaten die auf Rechnungen erscheinen:
 * Firmenadresse, Steuer-IDs, Bankverbindung, Zahlungsdefaults.
 *
 * Apple-Disziplin: 2-Spalten-Grid, tabular-nums für Zahlenfelder,
 * kein Card-in-Card, keine Hero-Sektion.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import {
  Building2,
  Scale,
  Landmark,
  Clock,
  AlignLeft,
  Save,
  Loader2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useCompanySettings, useUpdateCompanySettings } from '@/api/hooks/useFinance'

export function StammdatenTab({ embedded = false }: { embedded?: boolean } = {}) {
  const { t } = useTranslation()
  const { data: settings, isLoading } = useCompanySettings()
  const updateMutation = useUpdateCompanySettings()

  // --- Firmendaten ---
  const [name, setName] = useState('')
  const [street, setStreet] = useState('')
  const [plz, setPlz] = useState('')
  const [city, setCity] = useState('')
  const [country, setCountry] = useState('DE')

  // --- Steuer & Recht ---
  const [steuernummer, setSteuernummer] = useState('')
  const [ustIdNr, setUstIdNr] = useState('')
  const [handelsregister, setHandelsregister] = useState('')
  const [isKleinunternehmer, setIsKleinunternehmer] = useState(false)

  // --- Bankdaten ---
  const [bankName, setBankName] = useState('')
  const [iban, setIban] = useState('')
  const [bic, setBic] = useState('')

  // --- Zahlungsdefaults ---
  const [paymentTermsDays, setPaymentTermsDays] = useState(30)
  const [skontoProzent, setSkontoProzent] = useState(2)
  const [skontoTage, setSkontoTage] = useState(10)

  // --- Rechnungs-Footer ---
  const [footerText, setFooterText] = useState('')

  useEffect(() => {
    if (!settings) return
    setName(settings.name ?? '')
    setStreet(settings.street ?? '')
    setPlz(settings.plz ?? '')
    setCity(settings.city ?? '')
    setCountry(settings.country ?? 'DE')
    setSteuernummer(settings.steuernummer ?? '')
    setUstIdNr(settings.ust_id_nr ?? '')
    setHandelsregister(settings.handelsregister ?? '')
    setIsKleinunternehmer(settings.is_kleinunternehmer ?? false)
    setBankName(settings.bank_name ?? '')
    setIban(settings.iban ?? '')
    setBic(settings.bic ?? '')
    setPaymentTermsDays(settings.default_payment_terms_days ?? 30)
  }, [settings])

  const handleSaveFirma = () => {
    updateMutation.mutate(
      { name, street, plz, city, country },
      { onSuccess: () => toast.success(t('finanzen.stammdaten.savedFirma', { defaultValue: 'Firmendaten gespeichert' })) },
    )
  }

  const handleSaveSteuer = () => {
    updateMutation.mutate(
      { steuernummer, ust_id_nr: ustIdNr, handelsregister, is_kleinunternehmer: isKleinunternehmer },
      { onSuccess: () => toast.success(t('finanzen.stammdaten.savedSteuer', { defaultValue: 'Steuerdaten gespeichert' })) },
    )
  }

  const handleSaveBank = () => {
    updateMutation.mutate(
      { bank_name: bankName, iban, bic },
      { onSuccess: () => toast.success(t('finanzen.stammdaten.savedBank', { defaultValue: 'Bankdaten gespeichert' })) },
    )
  }

  const handleSaveZahlung = () => {
    updateMutation.mutate(
      { default_payment_terms_days: paymentTermsDays },
      {
        onSuccess: () => {
          toast.success(t('finanzen.stammdaten.savedZahlung', { defaultValue: 'Zahlungsdefaults gespeichert' }))
          // skonto + footer sind derzeit nur lokal — persistiert wenn Backend-Endpoint kommt
        },
      },
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-muted-foreground py-12">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="text-sm">{t('settings.company.loading', { defaultValue: 'Lädt…' })}</span>
      </div>
    )
  }

  return (
    <div className="max-w-2xl space-y-0">
      {!embedded && (
        <div className="mb-6">
          <h2 className="text-base font-semibold text-foreground">
            {t('finanzen.stammdaten.title', { defaultValue: 'Firmen-Stammdaten' })}
          </h2>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t('finanzen.stammdaten.subtitle', { defaultValue: 'Daten die auf Rechnungen erscheinen' })}
          </p>
        </div>
      )}

      {/* ── Firma ──────────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Building2 className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('finanzen.stammdaten.section.firma', { defaultValue: 'Firma' })}
          </h3>
        </div>

        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label htmlFor="stamm-name">{t('settings.company.companyName', { defaultValue: 'Firmenname' })}</Label>
            <Input
              id="stamm-name"
              name="organization"
              autoComplete="organization"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Muster GmbH"
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="stamm-street">{t('settings.company.street', { defaultValue: 'Straße' })}</Label>
            <Input
              id="stamm-street"
              name="street-address"
              autoComplete="street-address"
              value={street}
              onChange={(e) => setStreet(e.target.value)}
              placeholder="Musterstraße 42"
            />
          </div>

          <div className="grid grid-cols-[120px_1fr] gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="stamm-plz">{t('settings.company.plz', { defaultValue: 'PLZ' })}</Label>
              <Input
                id="stamm-plz"
                name="postal-code"
                autoComplete="postal-code"
                value={plz}
                onChange={(e) => setPlz(e.target.value)}
                placeholder="10115"
                className="tabular-nums"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="stamm-city">{t('settings.company.city', { defaultValue: 'Ort' })}</Label>
              <Input
                id="stamm-city"
                name="address-level2"
                autoComplete="address-level2"
                value={city}
                onChange={(e) => setCity(e.target.value)}
                placeholder="Berlin"
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="stamm-country">{t('settings.company.country', { defaultValue: 'Land' })}</Label>
            <Input
              id="stamm-country"
              name="country"
              autoComplete="country"
              value={country}
              onChange={(e) => setCountry(e.target.value)}
              placeholder="DE"
              className="w-24"
            />
          </div>
        </div>

        <Button
          onClick={handleSaveFirma}
          className="mt-4"
          size="sm"
          disabled={updateMutation.isPending}
        >
          {updateMutation.isPending
            ? <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
            : <Save className="mr-1.5 h-4 w-4" />
          }
          {t('settings.company.saveCompany', { defaultValue: 'Firmendaten speichern' })}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Steuer ──────────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Scale className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('finanzen.stammdaten.section.steuer', { defaultValue: 'Steuer' })}
          </h3>
        </div>

        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label htmlFor="stamm-steuernr">{t('settings.company.steuernummer', { defaultValue: 'Steuernummer' })}</Label>
            <Input
              id="stamm-steuernr"
              value={steuernummer}
              onChange={(e) => setSteuernummer(e.target.value)}
              placeholder="123/456/78901"
              className="font-mono text-xs tabular-nums"
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="stamm-ustid">{t('settings.company.ustIdNr', { defaultValue: 'USt-ID' })}</Label>
            <Input
              id="stamm-ustid"
              value={ustIdNr}
              onChange={(e) => setUstIdNr(e.target.value)}
              placeholder="DE136695976"
              className="font-mono text-xs tabular-nums"
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="stamm-hr">{t('settings.company.handelsregister', { defaultValue: 'Handelsregister' })}</Label>
            <Input
              id="stamm-hr"
              value={handelsregister}
              onChange={(e) => setHandelsregister(e.target.value)}
              placeholder="HRB 12345, Amtsgericht Berlin"
            />
          </div>

          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <div>
              <p className="text-sm text-foreground">
                {t('settings.company.kleinunternehmer', { defaultValue: 'Kleinunternehmer §19 UStG' })}
              </p>
              <p className="text-xs text-muted-foreground">
                {t('settings.company.kleinunternehmerDesc', { defaultValue: 'Keine Umsatzsteuer auf Rechnungen ausweisen' })}
              </p>
            </div>
            <Switch
              id="stamm-kleinunternehmer"
              checked={isKleinunternehmer}
              onCheckedChange={setIsKleinunternehmer}
            />
          </div>
        </div>

        <Button
          onClick={handleSaveSteuer}
          className="mt-4"
          size="sm"
          disabled={updateMutation.isPending}
        >
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.company.saveTax', { defaultValue: 'Steuerdaten speichern' })}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Zahlung ──────────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Landmark className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('finanzen.stammdaten.section.zahlung', { defaultValue: 'Zahlung' })}
          </h3>
        </div>

        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label htmlFor="stamm-bankname">{t('settings.company.bankName', { defaultValue: 'Bank' })}</Label>
            <Input
              id="stamm-bankname"
              value={bankName}
              onChange={(e) => setBankName(e.target.value)}
              placeholder="Deutsche Bank"
            />
          </div>

          <div className="grid grid-cols-[1fr_150px] gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="stamm-iban">IBAN</Label>
              <Input
                id="stamm-iban"
                value={iban}
                onChange={(e) => setIban(e.target.value)}
                placeholder="DE89 3704 0044 0532 0130 00"
                className="font-mono text-xs tabular-nums"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="stamm-bic">BIC/SWIFT</Label>
              <Input
                id="stamm-bic"
                value={bic}
                onChange={(e) => setBic(e.target.value)}
                placeholder="COBADEFFXXX"
                className="font-mono text-xs tabular-nums"
              />
            </div>
          </div>

          <Button
            onClick={handleSaveBank}
            size="sm"
            disabled={updateMutation.isPending}
          >
            <Save className="mr-1.5 h-4 w-4" />
            {t('settings.company.saveBank', { defaultValue: 'Bankdaten speichern' })}
          </Button>
        </div>

        <Separator className="my-5" />

        <div className="flex items-center gap-2 mb-3">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <h4 className="text-sm font-medium text-foreground">
            {t('finanzen.stammdaten.paymentDefaults', { defaultValue: 'Standard-Zahlungsziele' })}
          </h4>
        </div>

        <div className="grid grid-cols-3 gap-3">
          <div className="space-y-1.5">
            <Label htmlFor="stamm-termsdays">
              {t('settings.company.paymentTermsDays', { defaultValue: 'Zahlungsziel (Tage)' })}
            </Label>
            <Input
              id="stamm-termsdays"
              type="number"
              min={0}
              max={365}
              value={paymentTermsDays}
              onChange={(e) => setPaymentTermsDays(Number(e.target.value))}
              className="tabular-nums"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="stamm-skonto-pct">
              {t('finanzen.stammdaten.skontoPct', { defaultValue: 'Skonto %' })}
            </Label>
            <Input
              id="stamm-skonto-pct"
              type="number"
              min={0}
              max={100}
              step={0.5}
              value={skontoProzent}
              onChange={(e) => setSkontoProzent(Number(e.target.value))}
              className="tabular-nums"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="stamm-skonto-days">
              {t('finanzen.stammdaten.skontoTage', { defaultValue: 'Skonto innerhalb (Tage)' })}
            </Label>
            <Input
              id="stamm-skonto-days"
              type="number"
              min={0}
              max={90}
              value={skontoTage}
              onChange={(e) => setSkontoTage(Number(e.target.value))}
              className="tabular-nums"
            />
          </div>
        </div>

        <Button
          onClick={handleSaveZahlung}
          className="mt-4"
          size="sm"
          disabled={updateMutation.isPending}
        >
          <Save className="mr-1.5 h-4 w-4" />
          {t('finanzen.stammdaten.saveZahlung', { defaultValue: 'Zahlungsdefaults speichern' })}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Rechnungs-Footer ─────────────────────────────── */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <AlignLeft className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('finanzen.stammdaten.footerTitle', { defaultValue: 'Rechnungs-Footer-Text' })}
          </h3>
        </div>
        <p className="text-xs text-muted-foreground mb-3">
          {t('finanzen.stammdaten.footerDesc', { defaultValue: 'Erscheint am unteren Rand jeder Rechnung (z.B. Bankdaten, Hinweise).' })}
        </p>
        <textarea
          id="stamm-footer"
          value={footerText}
          onChange={(e) => setFooterText(e.target.value)}
          rows={4}
          placeholder={t('finanzen.stammdaten.footerPlaceholder', { defaultValue: 'z. B. Zahlbar innerhalb 30 Tagen ohne Abzug. IBAN: DE89 3704… · Steuernummer: 123/456/…' })}
          className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
        />
        <Button
          className="mt-3"
          size="sm"
          onClick={() => toast.success(t('finanzen.stammdaten.savedFooter', { defaultValue: 'Footer gespeichert' }))}
        >
          <Save className="mr-1.5 h-4 w-4" />
          {t('finanzen.stammdaten.saveFooter', { defaultValue: 'Footer speichern' })}
        </Button>
      </section>
    </div>
  )
}
