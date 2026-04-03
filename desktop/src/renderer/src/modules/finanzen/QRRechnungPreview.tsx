/**
 * QRRechnungPreview — Swiss QR-bill (QR-Rechnung) preview component.
 *
 * Renders a standards-compliant visual preview of a Swiss QR-bill payment slip.
 * Layout follows SIX Swiss Payment Standards (ISO 20022).
 * Mock data for design — backend generates actual QR codes via API.
 */
import { useState } from 'react'
import {
  QrCode,
  Copy,
  Download,
  Printer,
  Check,
  Info,
  Eye,
  EyeOff,
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

interface QRBillData {
  // Creditor (Zahlungsempfaenger)
  creditorName: string
  creditorStreet: string
  creditorZipCity: string
  creditorCountry: string
  creditorIBAN: string
  // Debtor (Zahlungspflichtiger)
  debtorName: string
  debtorStreet: string
  debtorZipCity: string
  debtorCountry: string
  // Payment
  amount: string
  currency: 'CHF' | 'EUR'
  reference: string
  referenceType: 'QRR' | 'SCOR' | 'NON'
  additionalInfo: string
  // Invoice link
  invoiceNumber: string
}

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

const mockBillData: QRBillData = {
  creditorName: 'Cosmi AG',
  creditorStreet: 'Bahnhofstrasse 42',
  creditorZipCity: '8001 Zürich',
  creditorCountry: 'CH',
  creditorIBAN: 'CH93 0076 2011 6238 5295 7',
  debtorName: 'Muster AG',
  debtorStreet: 'Industrieweg 12',
  debtorZipCity: '3000 Bern',
  debtorCountry: 'CH',
  amount: '12 450.00',
  currency: 'CHF',
  reference: '21 00000 00003 13947 14300 09017',
  referenceType: 'QRR',
  additionalInfo: 'Rechnung RE-2026-003',
  invoiceNumber: 'RE-2026-003',
}

// ---------------------------------------------------------------------------
// QR-Bill Visual Component (inline)
// ---------------------------------------------------------------------------

function QRBillSlip({ data }: { data: QRBillData }) {
  return (
    <div className="border-2 border-foreground/20 rounded bg-white text-black">
      {/* Top dashed separator */}
      <div className="border-b-2 border-dashed border-foreground/20 px-1 py-0.5 flex items-center justify-between">
        <span className="text-[8px] text-gray-400">Vor dem Einzahlen abzutrennen</span>
        <span className="text-[8px] text-gray-400">&#9986;</span>
      </div>

      <div className="flex">
        {/* Left: Empfangsschein */}
        <div className="w-[200px] border-r-2 border-dashed border-foreground/20 p-3 space-y-2">
          <p className="text-[11px] font-bold">Empfangsschein</p>

          <div>
            <p className="text-[8px] font-bold">Konto / Zahlbar an</p>
            <p className="text-[9px]">{data.creditorIBAN}</p>
            <p className="text-[9px]">{data.creditorName}</p>
            <p className="text-[9px]">{data.creditorStreet}</p>
            <p className="text-[9px]">{data.creditorZipCity}</p>
          </div>

          <div>
            <p className="text-[8px] font-bold">Referenz</p>
            <p className="text-[9px] font-mono">{data.reference}</p>
          </div>

          <div>
            <p className="text-[8px] font-bold">Zahlbar durch</p>
            <p className="text-[9px]">{data.debtorName}</p>
            <p className="text-[9px]">{data.debtorStreet}</p>
            <p className="text-[9px]">{data.debtorZipCity}</p>
          </div>

          <div className="flex items-end justify-between pt-1">
            <div>
              <p className="text-[8px] font-bold">Waehrung</p>
              <p className="text-[10px]">{data.currency}</p>
            </div>
            <div>
              <p className="text-[8px] font-bold">Betrag</p>
              <p className="text-[10px]">{data.amount}</p>
            </div>
          </div>

          <div className="pt-1 border-t border-gray-200">
            <p className="text-[8px] font-bold">Annahmestelle</p>
          </div>
        </div>

        {/* Right: Zahlteil */}
        <div className="flex-1 p-3 space-y-2">
          <p className="text-[11px] font-bold">Zahlteil</p>

          <div className="flex gap-3">
            {/* QR Code placeholder */}
            <div className="flex h-[120px] w-[120px] shrink-0 items-center justify-center rounded border-2 border-gray-300 bg-gray-50">
              <div className="flex flex-col items-center gap-1">
                <QrCode className="h-12 w-12 text-gray-400" />
                <span className="text-[7px] text-gray-400">Swiss QR Code</span>
              </div>
            </div>

            {/* Payment details */}
            <div className="flex-1 space-y-2">
              <div className="flex items-end gap-4">
                <div>
                  <p className="text-[8px] font-bold">Waehrung</p>
                  <p className="text-[11px]">{data.currency}</p>
                </div>
                <div>
                  <p className="text-[8px] font-bold">Betrag</p>
                  <p className="text-[11px]">{data.amount}</p>
                </div>
              </div>

              <div>
                <p className="text-[8px] font-bold">Konto / Zahlbar an</p>
                <p className="text-[9px]">{data.creditorIBAN}</p>
                <p className="text-[9px]">{data.creditorName}</p>
                <p className="text-[9px]">{data.creditorStreet}</p>
                <p className="text-[9px]">{data.creditorZipCity}</p>
              </div>

              <div>
                <p className="text-[8px] font-bold">Referenz</p>
                <p className="text-[9px] font-mono">{data.reference}</p>
              </div>

              <div>
                <p className="text-[8px] font-bold">Zusätzliche Informationen</p>
                <p className="text-[9px]">{data.additionalInfo}</p>
              </div>

              <div>
                <p className="text-[8px] font-bold">Zahlbar durch</p>
                <p className="text-[9px]">{data.debtorName}</p>
                <p className="text-[9px]">{data.debtorStreet}</p>
                <p className="text-[9px]">{data.debtorZipCity}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Dialog Component
// ---------------------------------------------------------------------------

interface QRRechnungPreviewProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  invoiceNumber?: string
}

export function QRRechnungPreview({
  open,
  onOpenChange,
  invoiceNumber,
}: QRRechnungPreviewProps) {
  const [showDetails, setShowDetails] = useState(true)
  const data = { ...mockBillData, invoiceNumber: invoiceNumber ?? mockBillData.invoiceNumber }

  const handleCopyIBAN = () => {
    navigator.clipboard?.writeText(data.creditorIBAN.replace(/\s/g, ''))
    toast.success('IBAN kopiert')
  }

  const handleCopyReference = () => {
    navigator.clipboard?.writeText(data.reference.replace(/\s/g, ''))
    toast.success('Referenz kopiert')
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <QrCode className="h-5 w-5 text-primary" />
            QR-Rechnung — {data.invoiceNumber}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Info banner */}
          <div className="flex items-start gap-2 rounded-md bg-info-light/50 px-3 py-2">
            <Info className="h-3.5 w-3.5 text-info mt-0.5 shrink-0" />
            <p className="text-[11px] text-info">
              Swiss QR-bill nach SIX-Standard. Der QR-Code wird vom Backend generiert
              und enthaelt alle Zahlungsinformationen für E-Banking.
            </p>
          </div>

          {/* QR-Bill preview */}
          <div className="rounded-lg border border-border p-4 bg-secondary/20">
            <QRBillSlip data={data} />
          </div>

          {/* Payment details */}
          <div className="flex items-center justify-between">
            <button
              onClick={() => setShowDetails(!showDetails)}
              className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              {showDetails ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
              {showDetails ? 'Details ausblenden' : 'Details anzeigen'}
            </button>
          </div>

          {showDetails && (
            <div className="rounded-lg border border-border p-3 space-y-2">
              <div className="grid grid-cols-[110px_1fr_24px] gap-2 items-center">
                <span className="text-[11px] text-muted-foreground">IBAN</span>
                <span className="text-sm font-mono text-foreground">{data.creditorIBAN}</span>
                <button onClick={handleCopyIBAN} className="p-0.5 text-muted-foreground hover:text-primary transition-colors">
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </div>
              <div className="grid grid-cols-[110px_1fr_24px] gap-2 items-center">
                <span className="text-[11px] text-muted-foreground">Referenz</span>
                <span className="text-sm font-mono text-foreground">{data.reference}</span>
                <button onClick={handleCopyReference} className="p-0.5 text-muted-foreground hover:text-primary transition-colors">
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </div>
              <div className="grid grid-cols-[110px_1fr] gap-2 items-center">
                <span className="text-[11px] text-muted-foreground">Referenztyp</span>
                <span className="text-sm text-foreground">
                  {data.referenceType === 'QRR' ? 'QR-Referenz' : data.referenceType === 'SCOR' ? 'Creditor Reference' : 'Ohne Referenz'}
                </span>
              </div>
              <div className="grid grid-cols-[110px_1fr] gap-2 items-center">
                <span className="text-[11px] text-muted-foreground">Betrag</span>
                <span className="text-sm font-medium text-foreground">{data.currency} {data.amount}</span>
              </div>
              <div className="grid grid-cols-[110px_1fr] gap-2 items-center">
                <span className="text-[11px] text-muted-foreground">Empfänger</span>
                <span className="text-sm text-foreground">{data.creditorName}, {data.creditorZipCity}</span>
              </div>
              <div className="grid grid-cols-[110px_1fr] gap-2 items-center">
                <span className="text-[11px] text-muted-foreground">Zahler</span>
                <span className="text-sm text-foreground">{data.debtorName}, {data.debtorZipCity}</span>
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center justify-end gap-2">
            <button
              onClick={() => toast.success('QR-Rechnung als PDF heruntergeladen')}
              className="flex items-center gap-1.5 rounded-md border border-border px-3 py-2 text-xs text-foreground hover:bg-accent transition-colors"
            >
              <Download className="h-3.5 w-3.5" />
              PDF herunterladen
            </button>
            <button
              onClick={() => toast.success('QR-Rechnung wird gedruckt...')}
              className="flex items-center gap-1.5 rounded-md border border-border px-3 py-2 text-xs text-foreground hover:bg-accent transition-colors"
            >
              <Printer className="h-3.5 w-3.5" />
              Drucken
            </button>
            <button
              onClick={() => {
                onOpenChange(false)
                toast.success('QR-Rechnung an Rechnung angehaengt')
              }}
              className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <Check className="h-3.5 w-3.5" />
              An Rechnung anfügen
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Inline indicator (for use in invoice lists/details)
// ---------------------------------------------------------------------------

interface QRBillIndicatorProps {
  hasQRBill: boolean
  invoiceNumber: string
  onPreview?: () => void
}

export function QRBillIndicator({ hasQRBill, invoiceNumber: _invoiceNumber, onPreview }: QRBillIndicatorProps) {
  if (!hasQRBill) return null
  return (
    <button
      onClick={onPreview}
      className="flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary hover:bg-primary/20 transition-colors"
      title="QR-Rechnung anzeigen"
    >
      <QrCode className="h-3 w-3" />
      QR-Bill
    </button>
  )
}
