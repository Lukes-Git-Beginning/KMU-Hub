/**
 * PrivacySettingsTab — Persönliche Datenschutz-Einstellungen.
 *
 * Phase 4: Nur noch persönlicher Inhalt.
 * Tenant-Einstellungen (Tracking org-weit, Retention, Cookie-Defaults)
 * sind in Admin-Hub → Sicherheit → Datenschutz (PrivacyAdminTab).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Download, Trash2, ShieldCheck, FileDown } from 'lucide-react'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/shared'

export function PrivacySettingsTab() {
  const { t } = useTranslation()
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [isExporting, setIsExporting] = useState(false)

  const handleExportData = () => {
    setIsExporting(true)
    setTimeout(() => {
      setIsExporting(false)
      toast.success(t('settings.privacy.exportPreparing'))
    }, 2000)
  }

  const handleDeleteRequest = () => {
    toast.success(t('settings.privacy.deleteRequestSubmitted'))
    setShowDeleteConfirm(false)
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">
        {t('settings.privacy.personal.title', { defaultValue: 'Meine Daten' })}
      </h2>
      <p className="text-sm text-muted-foreground mb-6">
        {t('settings.privacy.subtitle', { defaultValue: 'Deine persönlichen DSGVO-Rechte (Art. 15 + Art. 17).' })}
      </p>

      {/* GDPR info */}
      <div className="rounded-lg border border-primary/20 bg-primary/5 p-4 mb-8">
        <div className="flex items-center gap-2 mb-2">
          <ShieldCheck className="h-5 w-5 text-primary" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.privacy.gdpr.title')}</h3>
        </div>
        <p className="text-xs text-muted-foreground leading-relaxed">
          {t('settings.privacy.gdpr.desc')}
        </p>
      </div>

      {/* Datenexport (Art. 15) */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <FileDown className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('settings.privacy.personal.exportTitle', { defaultValue: 'Meine Daten exportieren' })}
          </h3>
        </div>
        <p className="text-xs text-muted-foreground mb-3">
          {t('settings.privacy.export.desc', { defaultValue: 'Exportiere eine Kopie aller deiner persönlichen Daten (DSGVO Art. 15).' })}
        </p>
        <div className="flex items-center gap-3 rounded-lg border border-border bg-card p-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <Download className="h-5 w-5 text-primary" />
          </div>
          <div className="flex-1">
            <p className="text-sm font-medium text-foreground">{t('settings.privacy.export.fullExport')}</p>
            <p className="text-xs text-muted-foreground">{t('settings.privacy.export.fullExportHint')}</p>
          </div>
          <Button onClick={handleExportData} disabled={isExporting} size="sm">
            {isExporting
              ? t('settings.privacy.export.preparing')
              : t('settings.privacy.export.requestButton')
            }
          </Button>
        </div>
      </section>

      {/* Account-Löschung (Art. 17) */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Trash2 className="h-4 w-4 text-error" />
          <h3 className="text-sm font-medium text-error">
            {t('settings.privacy.personal.deleteTitle', { defaultValue: 'Account löschen' })}
          </h3>
        </div>
        <div className="rounded-lg border border-error/20 bg-error/5 p-4">
          <p className="text-sm text-foreground mb-2">
            {t('settings.privacy.deleteAccount.requestTitle')}
          </p>
          <p className="text-xs text-muted-foreground mb-4">
            {t('settings.privacy.deleteAccount.requestDesc')}
          </p>
          <Button variant="destructive" size="sm" onClick={() => setShowDeleteConfirm(true)}>
            <Trash2 className="mr-1.5 h-4 w-4" />
            {t('settings.privacy.deleteAccount.requestButton')}
          </Button>
        </div>
      </section>

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title={t('settings.privacy.deleteAccount.confirmTitle')}
        description={t('settings.privacy.deleteAccount.confirmDesc')}
        confirmLabel={t('settings.privacy.deleteAccount.confirmButton')}
        variant="destructive"
        onConfirm={handleDeleteRequest}
      />
    </div>
  )
}
