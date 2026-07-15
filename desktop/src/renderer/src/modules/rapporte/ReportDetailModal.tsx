import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  AlertTriangle,
  Camera,
  CheckCircle2,
  Clock,
  FileDown,
  HardHat,
  Package,
  Pencil,
  Send,
  ShieldCheck,
  Thermometer,
  Trash2,
  XCircle,
} from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import type { FieldReport } from '@/stores/rapporte'
import { useRapporteTenantStore } from '@/stores/rapporteTenant'
import SignatureCanvas from './SignatureCanvas'
import {
  weatherIcons,
  weatherLabelKeys,
  projectColors,
  approvalBadgeStyles,
  approvalLabelKeys,
  calcNetHours,
  formatDate,
} from './rapporte-shared'
import { buildReportPdf, downloadBlob, csvDateStamp } from './rapporte-export'

/**
 * ReportDetailModal — zentriertes Cosmi-Detailfenster für einen Tagesrapport
 * (ersetzt das frühere Slide-over-DetailPanel). Wetter/Zeiten/Arbeiter/
 * Tätigkeiten/Material/Fotos/Unterschrift/Approval + echter PDF-Export
 * (Markt-Layout) und tenant-gesteuerte Unterschrift-Pflicht vor dem
 * Einreichen (HERO/ToolTime-Muster).
 */
export function ReportDetailModal({
  report,
  onClose,
  onDelete,
  onUpdate,
}: {
  report: FieldReport | null
  onClose: () => void
  onDelete: (id: string) => void
  onUpdate: (id: string, updates: Partial<FieldReport>) => void
}) {
  const { t } = useTranslation()
  const [showSignaturePad, setShowSignaturePad] = useState(false)
  const requireSignature = useRapporteTenantStore((s) => s.requireSignature)

  const WeatherIcon = report ? weatherIcons[report.weather] : Thermometer
  const netHours = report ? calcNetHours(report.workStart, report.workEnd, report.breakMinutes) : ''
  const signatureBlocked = !!report && requireSignature && report.signatureStatus !== 'signed'

  const handlePdfExport = () => {
    if (!report) return
    const blob = buildReportPdf(report, {
      title: t('rapporte.pdf.title'),
      time: t('rapporte.pdf.time'),
      workers: t('rapporte.pdf.workers'),
      activities: t('rapporte.pdf.activities'),
      materials: t('rapporte.pdf.materials'),
      notes: t('rapporte.pdf.notes'),
      signature: t('rapporte.pdf.signature'),
      signed: t('rapporte.pdf.signed'),
      unsigned: t('rapporte.pdf.unsigned'),
      status: t('rapporte.pdf.status'),
    })
    downloadBlob(blob, `rapport-${report.date}-${csvDateStamp()}.pdf`)
    toast.success(t('rapporte.detail.pdfDownloaded'))
  }

  return (
    <DetailModal
      open={!!report}
      title={report ? formatDate(report.date) : undefined}
      subtitle={report?.projectName}
      onClose={onClose}
      badge={
        report ? (
          <span className={`ml-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${approvalBadgeStyles[report.approvalStatus]}`}>
            {report.approvalStatus === 'approved' && <CheckCircle2 className="h-3 w-3" />}
            {report.approvalStatus === 'rejected' && <XCircle className="h-3 w-3" />}
            {report.approvalStatus === 'submitted' && <Send className="h-3 w-3" />}
            {t(approvalLabelKeys[report.approvalStatus])}
          </span>
        ) : undefined
      }
      footer={
        report ? (
          <div className="flex items-center gap-2">
            <button
              onClick={() => onDelete(report.id)}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
            >
              <Trash2 className="h-3.5 w-3.5" />
              {t('common.delete')}
            </button>
            <button
              onClick={handlePdfExport}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <FileDown className="h-3.5 w-3.5" />
              {t('rapporte.detail.pdfExport')}
            </button>
          </div>
        ) : undefined
      }
    >
      {report && (
        <div className="space-y-5">
          {/* Header info */}
          <div className="flex items-start justify-between">
            <div>
              <span className={`inline-block rounded-full px-2.5 py-0.5 text-[10px] font-medium mb-1 ${projectColors[report.projectId] ?? 'bg-secondary text-muted-foreground'}`}>
                {report.projectName}
              </span>
              <p className="text-xs text-muted-foreground">{t('rapporte.detail.author', { name: report.author })}</p>
            </div>
            {report.signatureStatus === 'pending' && (
              <span className="flex items-center gap-1 rounded-full bg-warning-light text-warning px-2 py-0.5 text-[10px] font-medium">
                <AlertTriangle className="h-3 w-3" />
                {t('rapporte.detail.pending')}
              </span>
            )}
            {report.signatureStatus === 'signed' && (
              <span className="flex items-center gap-1 rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium">
                {t('rapporte.detail.signed')}
              </span>
            )}
          </div>

          {/* Weather block */}
          <div className="rounded-lg border border-border p-3 flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-secondary">
              <WeatherIcon className="h-5 w-5 text-foreground" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{t(weatherLabelKeys[report.weather])}</p>
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <Thermometer className="h-3 w-3" />
                {report.temperature}°C
              </p>
            </div>
          </div>

          {/* Work time block */}
          <div className="grid grid-cols-4 gap-2">
            <div className="rounded-lg border border-border bg-secondary/30 p-2.5 text-center">
              <p className="text-[10px] text-muted-foreground mb-0.5">{t('rapporte.detail.start')}</p>
              <p className="text-sm font-medium text-foreground">{report.workStart}</p>
            </div>
            <div className="rounded-lg border border-border bg-secondary/30 p-2.5 text-center">
              <p className="text-[10px] text-muted-foreground mb-0.5">{t('rapporte.detail.end')}</p>
              <p className="text-sm font-medium text-foreground">{report.workEnd}</p>
            </div>
            <div className="rounded-lg border border-border bg-secondary/30 p-2.5 text-center">
              <p className="text-[10px] text-muted-foreground mb-0.5">{t('rapporte.detail.break')}</p>
              <p className="text-sm font-medium text-foreground">{report.breakMinutes} min</p>
            </div>
            <div className="rounded-lg border border-border bg-primary-light p-2.5 text-center">
              <p className="text-[10px] text-primary mb-0.5">{t('rapporte.detail.net')}</p>
              <p className="text-sm font-semibold text-primary">{netHours}</p>
            </div>
          </div>

          {/* Workers table (hidden while the API delivers no worker rows) */}
          {report.workers.length > 0 && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
              <HardHat className="h-3 w-3" />
              {t('rapporte.detail.workersTitle', { count: report.workers.length })}
            </h4>
            <div className="rounded-lg border border-border overflow-hidden">
              <table className="w-full text-xs">
                <thead>
                  <tr className="bg-secondary/30">
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">{t('rapporte.detail.tableName')}</th>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">{t('rapporte.detail.tableRole')}</th>
                    <th className="px-3 py-2 text-right font-medium text-muted-foreground">{t('rapporte.detail.tableHours')}</th>
                  </tr>
                </thead>
                <tbody>
                  {report.workers.map((w, i) => (
                    <tr key={i} className="border-t border-border-muted">
                      <td className="px-3 py-2 text-foreground">{w.name}</td>
                      <td className="px-3 py-2 text-muted-foreground">{w.role}</td>
                      <td className="px-3 py-2 text-foreground text-right tabular-nums">{w.hours}h</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
          )}

          {/* Activities */}
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              {t('rapporte.detail.activitiesTitle', { count: report.activities.length })}
            </h4>
            <div className="space-y-1.5">
              {report.activities.map((a, i) => (
                <div key={i} className="flex items-start gap-2 rounded-md border border-border-muted px-3 py-2">
                  <div className="flex-1">
                    <p className="text-xs text-foreground">{a.description}</p>
                  </div>
                  <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground shrink-0">
                    {a.category}
                  </span>
                </div>
              ))}
            </div>
          </section>

          {/* Materials table */}
          {report.materials.length > 0 && (
            <section>
              <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
                <Package className="h-3 w-3" />
                {t('rapporte.detail.materialTitle', { count: report.materials.length })}
              </h4>
              <div className="rounded-lg border border-border overflow-hidden">
                <table className="w-full text-xs">
                  <thead>
                    <tr className="bg-secondary/30">
                      <th className="px-3 py-2 text-left font-medium text-muted-foreground">{t('rapporte.detail.tableArticle')}</th>
                      <th className="px-3 py-2 text-right font-medium text-muted-foreground">{t('rapporte.detail.tableQuantity')}</th>
                      <th className="px-3 py-2 text-left font-medium text-muted-foreground pl-3">{t('rapporte.detail.tableUnit')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {report.materials.map((m, i) => (
                      <tr key={i} className="border-t border-border-muted">
                        <td className="px-3 py-2 text-foreground">{m.article}</td>
                        <td className="px-3 py-2 text-muted-foreground text-right tabular-nums">{m.quantity}</td>
                        <td className="px-3 py-2 text-muted-foreground pl-3">{m.unit}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          )}

          {/* Photos */}
          {report.photos.length > 0 && (
            <section>
              <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
                <Camera className="h-3 w-3" />
                {t('rapporte.detail.photosTitle', { count: report.photos.length })}
              </h4>
              <div className="grid grid-cols-2 gap-2">
                {report.photos.map((photo, idx) => (
                  <div key={photo.id} className="rounded-lg border border-border bg-secondary/30 overflow-hidden">
                    <div className="relative aspect-[4/3] bg-secondary flex items-center justify-center">
                      <Camera className="h-8 w-8 text-muted-foreground opacity-20" />
                      <div className="absolute top-2 left-2 flex h-6 w-6 items-center justify-center rounded-full bg-foreground/70 text-[10px] font-bold text-background">
                        {idx + 1}
                      </div>
                    </div>
                    <div className="px-2 py-1.5">
                      <p className="text-[10px] text-muted-foreground line-clamp-2">{photo.caption || t('rapporte.detail.noCaption')}</p>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          )}

          {/* Signature section */}
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              {t('rapporte.detail.signatureTitle')}
            </h4>
            {report.signatureDataUrl ? (
              <div className="rounded-lg border border-border bg-white p-3">
                <img
                  src={report.signatureDataUrl}
                  alt={t('rapporte.detail.signatureAlt')}
                  loading="lazy"
                  decoding="async"
                  className="max-h-24 mx-auto"
                />
                <p className="text-[10px] text-success text-center mt-1 flex items-center justify-center gap-1">
                  <CheckCircle2 className="h-3 w-3" />
                  {t('rapporte.detail.signatureSigned')}
                </p>
              </div>
            ) : showSignaturePad ? (
              <div className="rounded-lg border border-border p-3 space-y-2">
                <SignatureCanvas
                  onSave={(dataUrl) => {
                    onUpdate(report.id, {
                      signatureDataUrl: dataUrl,
                      signatureStatus: 'signed',
                    })
                    setShowSignaturePad(false)
                    toast.success(t('rapporte.detail.signatureSaved'))
                  }}
                />
                <button
                  onClick={() => setShowSignaturePad(false)}
                  className="w-full rounded-lg border border-border py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                >
                  {t('common.cancel')}
                </button>
              </div>
            ) : (
              <button
                onClick={() => setShowSignaturePad(true)}
                className="w-full rounded-lg border-2 border-dashed border-border p-4 flex items-center justify-center gap-2 text-muted-foreground hover:bg-secondary/50 hover:border-primary/30 transition-colors"
              >
                <Pencil className="h-4 w-4" />
                <span className="text-xs font-medium">{t('rapporte.detail.signatureCapture')}</span>
              </button>
            )}
          </section>

          {/* Approval workflow */}
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
              <ShieldCheck className="h-3 w-3" />
              {t('rapporte.detail.approvalTitle')}
            </h4>
            {report.approvalStatus === 'draft' && (
              <div className="space-y-2">
                {/* Tenant policy: signature required before submitting (HERO/ToolTime) */}
                {signatureBlocked && (
                  <div className="flex items-start gap-2 rounded-lg border border-warning/40 bg-warning-light p-3">
                    <AlertTriangle className="h-4 w-4 text-warning shrink-0 mt-0.5" />
                    <p className="text-xs text-warning">{t('rapporte.detail.signatureRequiredHint')}</p>
                  </div>
                )}
                <button
                  onClick={() => {
                    if (signatureBlocked) {
                      toast.error(t('rapporte.detail.signatureRequiredError'))
                      return
                    }
                    onUpdate(report.id, { approvalStatus: 'submitted' })
                    toast.success(t('rapporte.detail.submitted'))
                  }}
                  disabled={signatureBlocked}
                  className="w-full flex items-center justify-center gap-2 rounded-lg border border-primary bg-primary-light px-3 py-2.5 text-xs font-medium text-primary hover:bg-primary hover:text-primary-foreground transition-colors disabled:opacity-50 disabled:hover:bg-primary-light disabled:hover:text-primary"
                >
                  <Send className="h-3.5 w-3.5" />
                  {t('rapporte.detail.submitApproval')}
                </button>
              </div>
            )}
            {report.approvalStatus === 'submitted' && (
              <div className="rounded-lg border border-info bg-info-light p-3 flex items-center gap-2">
                <Clock className="h-4 w-4 text-info shrink-0" />
                <div>
                  <p className="text-xs font-medium text-info">{t('rapporte.detail.waitingApproval')}</p>
                  <p className="text-[10px] text-info/70 mt-0.5">{t('rapporte.detail.waitingDescription')}</p>
                </div>
              </div>
            )}
            {report.approvalStatus === 'approved' && (
              <div className="rounded-lg border border-success bg-success-light p-3 flex items-center gap-2">
                <CheckCircle2 className="h-4 w-4 text-success shrink-0" />
                <div>
                  <p className="text-xs font-medium text-success">{t('rapporte.detail.approved')}</p>
                  {report.approvedBy && (
                    <p className="text-[10px] text-success/70 mt-0.5">{t('rapporte.detail.approvedBy', { name: report.approvedBy })}</p>
                  )}
                </div>
              </div>
            )}
            {report.approvalStatus === 'rejected' && (
              <div className="space-y-2">
                <div className="rounded-lg border border-error bg-error-light p-3 flex items-start gap-2">
                  <XCircle className="h-4 w-4 text-error shrink-0 mt-0.5" />
                  <div>
                    <p className="text-xs font-medium text-error">{t('rapporte.detail.rejected')}</p>
                    {report.approvalComment && (
                      <p className="text-[10px] text-error/80 mt-0.5">{t('rapporte.detail.rejectedReason', { reason: report.approvalComment })}</p>
                    )}
                    {report.approvedBy && (
                      <p className="text-[10px] text-error/60 mt-0.5">{t('rapporte.detail.rejectedBy', { name: report.approvedBy })}</p>
                    )}
                  </div>
                </div>
                <button
                  onClick={() => {
                    onUpdate(report.id, { approvalStatus: 'submitted', approvalComment: undefined })
                    toast.success(t('rapporte.detail.resubmitted'))
                  }}
                  className="w-full flex items-center justify-center gap-2 rounded-lg border border-border px-3 py-2 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  <Send className="h-3.5 w-3.5" />
                  {t('rapporte.detail.resubmit')}
                </button>
              </div>
            )}
          </section>

          {/* Notes */}
          {report.notes && (
            <section>
              <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
                {t('rapporte.detail.notesTitle')}
              </h4>
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <p className="text-xs text-foreground leading-relaxed">{report.notes}</p>
              </div>
            </section>
          )}
        </div>
      )}
    </DetailModal>
  )
}
