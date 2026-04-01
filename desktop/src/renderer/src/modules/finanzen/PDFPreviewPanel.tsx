/**
 * PDFPreviewPanel — Inline preview for invoice PDFs.
 *
 * Shows a styled placeholder representing a generated PDF document.
 * Backend swap: replace mock with actual PDF render via iframe/embed.
 */
import {
  Download,
  Printer,
  ZoomIn,
  ZoomOut,
  ChevronLeft,
  ChevronRight,
  Maximize2,
} from 'lucide-react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface PDFPreviewPanelProps {
  invoiceNumber: string
  customerName: string
  date: string
  amount: string
  onDownload?: () => void
}

export function PDFPreviewPanel({
  invoiceNumber,
  customerName,
  date,
  amount,
  onDownload,
}: PDFPreviewPanelProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
          PDF-Vorschau
        </h4>
        <div className="flex items-center gap-1">
          <button
            onClick={() => toast.success('Vollbild-Vorschau geöffnet')}
            className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
            title="Vollbild"
          >
            <Maximize2 className="h-3.5 w-3.5" />
          </button>
          <button
            onClick={() => toast.success('PDF wird gedruckt...')}
            className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
            title="Drucken"
          >
            <Printer className="h-3.5 w-3.5" />
          </button>
          <button
            onClick={onDownload ?? (() => toast.success('PDF heruntergeladen'))}
            className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
            title="Herunterladen"
          >
            <Download className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      {/* Mock PDF document */}
      <div className="rounded-md border border-border bg-white shadow-sm overflow-hidden">
        {/* Page content mock */}
        <div className="p-6 space-y-6 min-h-[320px]">
          {/* Header */}
          <div className="flex items-start justify-between">
            <div>
              <div className="h-8 w-32 rounded bg-gray-200 mb-2" />
              <p className="text-[9px] text-gray-400">Cosmi AG</p>
              <p className="text-[9px] text-gray-400">Bahnhofstrasse 42</p>
              <p className="text-[9px] text-gray-400">8001 Zürich</p>
            </div>
            <div className="text-right">
              <p className="text-sm font-bold text-gray-800">RECHNUNG</p>
              <p className="text-[10px] text-gray-500 font-mono">{invoiceNumber}</p>
              <p className="text-[10px] text-gray-500 mt-1">Datum: {date}</p>
            </div>
          </div>

          {/* Customer block */}
          <div className="border-l-2 border-gray-300 pl-3">
            <p className="text-[9px] text-gray-400">Rechnung an:</p>
            <p className="text-[10px] text-gray-700 font-medium">{customerName}</p>
            <div className="h-2.5 w-28 rounded bg-gray-100 mt-1" />
            <div className="h-2.5 w-24 rounded bg-gray-100 mt-0.5" />
          </div>

          {/* Line items mock */}
          <div>
            <div className="grid grid-cols-[2rem_1fr_50px_60px_60px] gap-1 pb-1 border-b border-gray-200">
              <span className="text-[7px] font-medium text-gray-500">Pos.</span>
              <span className="text-[7px] font-medium text-gray-500">Beschreibung</span>
              <span className="text-[7px] font-medium text-gray-500 text-right">Menge</span>
              <span className="text-[7px] font-medium text-gray-500 text-right">Preis</span>
              <span className="text-[7px] font-medium text-gray-500 text-right">Gesamt</span>
            </div>
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                className="grid grid-cols-[2rem_1fr_50px_60px_60px] gap-1 py-1 border-b border-gray-100"
              >
                <span className="text-[8px] text-gray-500">{i}</span>
                <div className="h-2.5 rounded bg-gray-100" style={{ width: `${60 + i * 15}%` }} />
                <div className="h-2.5 w-6 rounded bg-gray-100 ml-auto" />
                <div className="h-2.5 w-10 rounded bg-gray-100 ml-auto" />
                <div className="h-2.5 w-10 rounded bg-gray-100 ml-auto" />
              </div>
            ))}
          </div>

          {/* Totals */}
          <div className="flex justify-end">
            <div className="w-40 space-y-1">
              <div className="flex justify-between">
                <span className="text-[8px] text-gray-500">Netto</span>
                <div className="h-2 w-12 rounded bg-gray-100" />
              </div>
              <div className="flex justify-between">
                <span className="text-[8px] text-gray-500">MwSt</span>
                <div className="h-2 w-10 rounded bg-gray-100" />
              </div>
              <div className="flex justify-between border-t border-gray-200 pt-1">
                <span className="text-[9px] font-bold text-gray-700">Gesamt</span>
                <span className="text-[9px] font-bold text-gray-800">{amount}</span>
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="border-t border-gray-200 pt-2 flex justify-between">
            <div className="h-2 w-44 rounded bg-gray-50" />
            <div className="h-2 w-32 rounded bg-gray-50" />
          </div>
        </div>

        {/* Page controls */}
        <div className="flex items-center justify-center gap-3 border-t border-border bg-secondary/30 px-3 py-1.5">
          <button className="rounded p-0.5 text-muted-foreground hover:text-foreground" disabled>
            <ChevronLeft className="h-3.5 w-3.5" />
          </button>
          <span className="text-[10px] text-muted-foreground">Seite 1 von 1</span>
          <button className="rounded p-0.5 text-muted-foreground hover:text-foreground" disabled>
            <ChevronRight className="h-3.5 w-3.5" />
          </button>
          <span className="text-[10px] text-border">|</span>
          <button className="rounded p-0.5 text-muted-foreground hover:text-foreground">
            <ZoomOut className="h-3 w-3" />
          </button>
          <span className="text-[10px] text-muted-foreground">100%</span>
          <button className="rounded p-0.5 text-muted-foreground hover:text-foreground">
            <ZoomIn className="h-3 w-3" />
          </button>
        </div>
      </div>
    </div>
  )
}
