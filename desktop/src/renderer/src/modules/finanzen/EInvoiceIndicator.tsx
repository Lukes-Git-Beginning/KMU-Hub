/**
 * EInvoiceIndicator — ZUGFeRD / XRechnung compliance indicator.
 *
 * Shows the e-invoicing standard compliance level of an invoice.
 * ZUGFeRD (Zentraler User Guide des Forums elektronische Rechnung Deutschland)
 * supports 3 profiles: MINIMUM, BASIC, COMFORT, EXTENDED.
 * XRechnung is the German public sector standard (EN 16931).
 * Mock data for design — backend generates actual XML payloads.
 */
import { useState } from 'react'
import {
  FileCheck2,
  Shield,
  Info,
  CheckCircle2,
  AlertTriangle,
  X,
  Download,
  Zap,
  Building2,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type EInvoiceStandard = 'zugferd' | 'xrechnung' | 'none'
type ZUGFeRDProfile = 'MINIMUM' | 'BASIC' | 'COMFORT' | 'EXTENDED'

interface EInvoiceStatus {
  standard: EInvoiceStandard
  profile?: ZUGFeRDProfile
  version?: string
  valid: boolean
  errors: string[]
  warnings: string[]
  xmlEmbedded: boolean
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

const standardConfig: Record<
  EInvoiceStandard,
  { label: string; fullName: string; color: string; bg: string }
> = {
  zugferd: {
    label: 'ZUGFeRD',
    fullName: 'Zentraler User Guide des Forums elektronische Rechnung Deutschland',
    color: 'text-primary',
    bg: 'bg-primary/10',
  },
  xrechnung: {
    label: 'XRechnung',
    fullName: 'Deutscher Standard für öffentliche Auftraggeber (EN 16931)',
    color: 'text-info',
    bg: 'bg-info-light',
  },
  none: {
    label: 'Standard',
    fullName: 'Keine E-Rechnung',
    color: 'text-muted-foreground',
    bg: 'bg-secondary',
  },
}

const profileDescriptions: Record<ZUGFeRDProfile, string> = {
  MINIMUM: 'Minimale strukturierte Daten — nur Rechnungskopf',
  BASIC: 'Grundlegende Rechnungsdaten — Positionen ohne Details',
  COMFORT: 'Erweiterte Daten — vollständige Positionen + Steuerdetails',
  EXTENDED: 'Maximale Details — Liefer-/Leistungsdaten, Rabatte, Zuschläge',
}

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

const mockStatuses: Record<string, EInvoiceStatus> = {
  'RE-2026-003': {
    standard: 'zugferd',
    profile: 'COMFORT',
    version: '2.3',
    valid: true,
    errors: [],
    warnings: ['Lieferdatum nicht angegeben — empfohlen für COMFORT-Profil'],
    xmlEmbedded: true,
  },
  'RE-2026-008': {
    standard: 'xrechnung',
    version: '3.0',
    valid: true,
    errors: [],
    warnings: [],
    xmlEmbedded: true,
  },
  'RE-2026-012': {
    standard: 'zugferd',
    profile: 'BASIC',
    version: '2.3',
    valid: false,
    errors: ['Pflichtfeld BT-31 (Seller VAT Identifier) fehlt'],
    warnings: ['Zahlungsbedingungen nicht maschinenlesbar'],
    xmlEmbedded: false,
  },
  'RE-2026-015': {
    standard: 'none',
    valid: true,
    errors: [],
    warnings: [],
    xmlEmbedded: false,
  },
}

// ---------------------------------------------------------------------------
// Badge Component (inline)
// ---------------------------------------------------------------------------

interface EInvoiceBadgeProps {
  invoiceNumber: string
  onClick?: () => void
}

export function EInvoiceBadge({ invoiceNumber, onClick }: EInvoiceBadgeProps) {
  const status = mockStatuses[invoiceNumber]
  if (!status || status.standard === 'none') return null

  const cfg = standardConfig[status.standard]

  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium transition-colors hover:opacity-80 ${cfg.bg} ${cfg.color}`}
      title={`${cfg.label}${status.profile ? ` ${status.profile}` : ''}`}
    >
      <FileCheck2 className="h-3 w-3" />
      {cfg.label}
      {status.profile && <span className="opacity-70">{status.profile}</span>}
      {!status.valid && <AlertTriangle className="h-2.5 w-2.5 text-warning" />}
    </button>
  )
}

// ---------------------------------------------------------------------------
// Detail Dialog
// ---------------------------------------------------------------------------

interface EInvoiceDetailDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  invoiceNumber: string
}

export function EInvoiceDetailDialog({
  open,
  onOpenChange,
  invoiceNumber,
}: EInvoiceDetailDialogProps) {
  const status = mockStatuses[invoiceNumber] ?? {
    standard: 'none' as const,
    valid: true,
    errors: [],
    warnings: [],
    xmlEmbedded: false,
  }

  const [selectedProfile, setSelectedProfile] = useState<ZUGFeRDProfile>(
    status.profile ?? 'COMFORT',
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FileCheck2 className="h-5 w-5 text-primary" />
            E-Rechnung — {invoiceNumber}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Standard selector */}
          <div>
            <p className="text-xs font-medium text-foreground mb-2">E-Rechnungsstandard</p>
            <div className="grid grid-cols-3 gap-2">
              {(Object.entries(standardConfig) as [EInvoiceStandard, typeof standardConfig.zugferd][]).map(
                ([key, s]) => (
                  <div
                    key={key}
                    className={`rounded-lg border p-3 text-center ${
                      status.standard === key
                        ? 'border-primary bg-primary/5'
                        : 'border-border'
                    }`}
                  >
                    <div
                      className={`mx-auto mb-1.5 flex h-8 w-8 items-center justify-center rounded-full ${s.bg}`}
                    >
                      {key === 'zugferd' ? (
                        <Zap className={`h-4 w-4 ${s.color}`} />
                      ) : key === 'xrechnung' ? (
                        <Building2 className={`h-4 w-4 ${s.color}`} />
                      ) : (
                        <FileCheck2 className={`h-4 w-4 ${s.color}`} />
                      )}
                    </div>
                    <p className={`text-xs font-medium ${s.color}`}>{s.label}</p>
                    <p className="text-[9px] text-muted-foreground mt-0.5">
                      {key === 'zugferd'
                        ? 'PDF + XML'
                        : key === 'xrechnung'
                          ? 'Reines XML'
                          : 'Nur PDF'}
                    </p>
                    {status.standard === key && (
                      <CheckCircle2 className="mx-auto mt-1 h-3.5 w-3.5 text-primary" />
                    )}
                  </div>
                ),
              )}
            </div>
          </div>

          {/* ZUGFeRD Profile selector */}
          {status.standard === 'zugferd' && (
            <div>
              <p className="text-xs font-medium text-foreground mb-2">ZUGFeRD-Profil</p>
              <div className="space-y-1.5">
                {(
                  ['MINIMUM', 'BASIC', 'COMFORT', 'EXTENDED'] as ZUGFeRDProfile[]
                ).map((profile) => (
                  <button
                    key={profile}
                    onClick={() => setSelectedProfile(profile)}
                    className={`flex w-full items-center gap-3 rounded-md border px-3 py-2 text-left transition-colors ${
                      selectedProfile === profile
                        ? 'border-primary bg-primary/5'
                        : 'border-border hover:bg-accent/50'
                    }`}
                  >
                    <div
                      className={`flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-bold ${
                        selectedProfile === profile
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-secondary text-muted-foreground'
                      }`}
                    >
                      {profile[0]}
                    </div>
                    <div className="flex-1">
                      <p className="text-sm font-medium text-foreground">{profile}</p>
                      <p className="text-[10px] text-muted-foreground">
                        {profileDescriptions[profile]}
                      </p>
                    </div>
                    {status.profile === profile && (
                      <span className="rounded-full bg-primary/10 px-2 py-0.5 text-[9px] font-medium text-primary">
                        Aktiv
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Version + XML status */}
          {status.standard !== 'none' && (
            <div className="grid grid-cols-2 gap-3">
              <div className="rounded-md border border-border p-2.5">
                <p className="text-[10px] text-muted-foreground">Version</p>
                <p className="text-sm font-medium text-foreground">
                  {status.version ?? '—'}
                </p>
              </div>
              <div className="rounded-md border border-border p-2.5">
                <p className="text-[10px] text-muted-foreground">XML eingebettet</p>
                <div className="flex items-center gap-1.5 mt-0.5">
                  {status.xmlEmbedded ? (
                    <>
                      <CheckCircle2 className="h-4 w-4 text-success" />
                      <span className="text-sm font-medium text-success">Ja</span>
                    </>
                  ) : (
                    <>
                      <X className="h-4 w-4 text-muted-foreground" />
                      <span className="text-sm font-medium text-muted-foreground">Nein</span>
                    </>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Validation results */}
          {status.standard !== 'none' && (
            <div>
              <p className="text-xs font-medium text-foreground mb-2">Validierung</p>
              <div
                className={`rounded-md border p-3 ${
                  status.valid
                    ? 'border-success/30 bg-success-light/20'
                    : 'border-error/30 bg-error-light/20'
                }`}
              >
                <div className="flex items-center gap-2 mb-2">
                  {status.valid ? (
                    <>
                      <CheckCircle2 className="h-4 w-4 text-success" />
                      <span className="text-sm font-medium text-success">Gültig</span>
                    </>
                  ) : (
                    <>
                      <AlertTriangle className="h-4 w-4 text-error" />
                      <span className="text-sm font-medium text-error">Ungültig</span>
                    </>
                  )}
                </div>

                {status.errors.length > 0 && (
                  <div className="space-y-1 mb-2">
                    {status.errors.map((err, i) => (
                      <div key={i} className="flex items-start gap-1.5 text-[11px] text-error">
                        <X className="h-3 w-3 mt-0.5 shrink-0" />
                        {err}
                      </div>
                    ))}
                  </div>
                )}

                {status.warnings.length > 0 && (
                  <div className="space-y-1">
                    {status.warnings.map((warn, i) => (
                      <div key={i} className="flex items-start gap-1.5 text-[11px] text-warning">
                        <AlertTriangle className="h-3 w-3 mt-0.5 shrink-0" />
                        {warn}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Info */}
          <div className="flex items-start gap-2 rounded-md bg-secondary/50 px-3 py-2">
            <Info className="h-3.5 w-3.5 text-muted-foreground mt-0.5 shrink-0" />
            <p className="text-[11px] text-muted-foreground">
              {status.standard === 'zugferd'
                ? 'ZUGFeRD-Rechnungen enthalten eine maschinenlesbare XML-Datei (factur-x.xml) eingebettet in das PDF. Kompatibel mit Buchhaltungssoftware und DATEV.'
                : status.standard === 'xrechnung'
                  ? 'XRechnung ist für öffentliche Auftraggeber in Deutschland Pflicht (seit 27.11.2020). Reines XML-Format nach EN 16931.'
                  : 'Standard-PDF ohne maschinenlesbare Daten. Für automatische Verarbeitung wird ZUGFeRD oder XRechnung empfohlen.'}
            </p>
          </div>

          {/* Actions */}
          {status.standard !== 'none' && (
            <div className="flex items-center justify-end gap-2">
              <button
                onClick={() => toast.success('XML-Datei heruntergeladen')}
                className="flex items-center gap-1.5 rounded-md border border-border px-3 py-2 text-xs text-foreground hover:bg-accent transition-colors"
              >
                <Download className="h-3.5 w-3.5" />
                XML herunterladen
              </button>
              <button
                onClick={() => toast.success('Validierung gestartet...')}
                className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <Shield className="h-3.5 w-3.5" />
                Erneut validieren
              </button>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
