import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Download, Trash2, ShieldCheck, Cookie, FileDown } from 'lucide-react'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/shared'

export function PrivacySettingsTab() {
  const { t } = useTranslation()
  const [analyticsEnabled, setAnalyticsEnabled] = useState(true)
  const [crashReportsEnabled, setCrashReportsEnabled] = useState(true)
  const [usageDataEnabled, setUsageDataEnabled] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [isExporting, setIsExporting] = useState(false)

  const handleExportData = () => {
    setIsExporting(true)
    setTimeout(() => {
      setIsExporting(false)
      toast.success('Datenexport wird vorbereitet. Du erhältst eine E-Mail mit dem Download-Link.')
    }, 2000)
  }

  const handleDeleteRequest = () => {
    toast.success('Löschantrag eingereicht. Du erhältst eine Bestätigung per E-Mail innerhalb von 72 Stunden.')
    setShowDeleteConfirm(false)
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">Datenschutz</h2>
      <p className="text-sm text-muted-foreground mb-6">DSGVO-Einstellungen, Datenexport und Datenlösch-Antraege</p>

      {/* GDPR info */}
      <div className="rounded-lg border border-primary/20 bg-primary/5 p-4 mb-8">
        <div className="flex items-center gap-2 mb-2">
          <ShieldCheck className="h-5 w-5 text-primary" />
          <h3 className="text-sm font-medium text-foreground">DSGVO-konform</h3>
        </div>
        <p className="text-xs text-muted-foreground leading-relaxed">
          Cosmi speichert alle Daten auf EU-Servern (Hetzner, Deutschland). Deine Daten werden nie an Dritte
          außerhalb der EU übermittelt. Du hast jederzeit das Recht auf Auskunft, Berichtigung und Löschung
          deiner personenbezogenen Daten.
        </p>
      </div>

      {/* Cookie / tracking preferences */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Cookie className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Tracking-Einstellungen</h3>
        </div>

        <div className="space-y-3">
          <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
            <div>
              <p className="text-sm text-foreground">Analytik</p>
              <p className="text-xs text-muted-foreground">Anonyme Nutzungsstatistiken zur Verbesserung der App</p>
            </div>
            <Switch checked={analyticsEnabled} onCheckedChange={setAnalyticsEnabled} />
          </div>

          <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
            <div>
              <p className="text-sm text-foreground">Absturz-Berichte</p>
              <p className="text-xs text-muted-foreground">Automatische Fehlerberichte an unser Team senden</p>
            </div>
            <Switch checked={crashReportsEnabled} onCheckedChange={setCrashReportsEnabled} />
          </div>

          <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
            <div>
              <p className="text-sm text-foreground">Detaillierte Nutzungsdaten</p>
              <p className="text-xs text-muted-foreground">Erweiterte Feature-Nutzung und Klick-Muster</p>
            </div>
            <Switch checked={usageDataEnabled} onCheckedChange={setUsageDataEnabled} />
          </div>
        </div>

        <Button
          variant="outline"
          size="sm"
          className="mt-3"
          onClick={() => toast.success('Tracking-Einstellungen gespeichert')}
        >
          Einstellungen speichern
        </Button>
      </section>

      {/* Data export */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <FileDown className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Daten exportieren</h3>
        </div>
        <p className="text-xs text-muted-foreground mb-3">
          Fordere eine Kopie aller deiner personenbezogenen Daten an (Art. 20 DSGVO).
          Der Export umfasst Profil, Kontakte, E-Mails, Dokumente und Aktivitäten.
        </p>
        <div className="flex items-center gap-3 rounded-lg border border-border bg-card p-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <Download className="h-5 w-5 text-primary" />
          </div>
          <div className="flex-1">
            <p className="text-sm font-medium text-foreground">Kompletter Datenexport</p>
            <p className="text-xs text-muted-foreground">Wird als ZIP-Datei per E-Mail zugestellt (kann bis zu 24h dauern)</p>
          </div>
          <Button onClick={handleExportData} disabled={isExporting} size="sm">
            {isExporting ? 'Wird vorbereitet...' : 'Export anfordern'}
          </Button>
        </div>
      </section>

      {/* Account deletion */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Trash2 className="h-4 w-4 text-error" />
          <h3 className="text-sm font-medium text-error">Konto löschen</h3>
        </div>
        <div className="rounded-lg border border-error/20 bg-error/5 p-4">
          <p className="text-sm text-foreground mb-2">Löschantrag stellen (Art. 17 DSGVO)</p>
          <p className="text-xs text-muted-foreground mb-4">
            Alle deine personenbezogenen Daten werden innerhalb von 30 Tagen unwiderruflich gelöscht.
            Gemeinsam genutzte Daten (Projekte, Team-Dokumente) bleiben für andere Nutzer erhalten,
            werden aber anonymisiert.
          </p>
          <Button variant="destructive" size="sm" onClick={() => setShowDeleteConfirm(true)}>
            <Trash2 className="mr-1.5 h-4 w-4" />
            Konto-Löschung beantragen
          </Button>
        </div>
      </section>

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title="Konto wirklich löschen?"
        description="Dieser Vorgang kann nicht rückgängig gemacht werden. Alle deine persönlichen Daten werden innerhalb von 30 Tagen unwiderruflich gelöscht. Du erhältst eine Bestätigungs-E-Mail."
        confirmLabel="Ja, Konto löschen"
        variant="destructive"
        onConfirm={handleDeleteRequest}
      />
    </div>
  )
}
