/**
 * PDFPreviewPanel — Inline-Vorschau eines Rechnungs-/Beleg-PDFs (finanzen P2.5c).
 *
 * Rendert die ECHTEN Belegdaten (Kopf, Positionen, Summen) statt grauer
 * Platzhalter. Herunterladen liefert das echte PDF (MSW). Drucken öffnet den
 * Druckdialog (Electron → „Als PDF speichern"). Zoom + Vollbild funktionieren.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Download, Printer, ZoomIn, ZoomOut, Maximize2 } from 'lucide-react'
import { Dialog, DialogContent } from '@/components/ui/dialog'

export interface PreviewLineItem {
  description: string
  quantity: number | string
  unit_price: number | string
  tax_rate?: number | string
  line_total: number | string
}

interface PDFPreviewPanelProps {
  heading?: string
  number: string
  customerName: string
  customerAddress?: string
  date: string
  lineItems: PreviewLineItem[]
  net: number | string
  tax: number | string
  gross: number | string
  currency?: string
  onDownload?: () => void
}

export function PDFPreviewPanel(props: PDFPreviewPanelProps) {
  const { t } = useTranslation()
  const [zoom, setZoom] = useState(1)
  const [fullscreen, setFullscreen] = useState(false)
  const currency = props.currency ?? 'EUR'
  const money = (v: number | string) =>
    `${Number(v ?? 0).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${currency}`

  const handlePrint = () => {
    const rows = props.lineItems
      .map(
        (it) =>
          `<tr><td>${it.quantity}×</td><td>${it.description}</td><td style="text-align:right">${money(it.line_total)}</td></tr>`,
      )
      .join('')
    const html = `<!doctype html><html><head><meta charset="utf-8"><title>${props.number}</title>
      <style>body{font-family:Helvetica,Arial,sans-serif;color:#1a1a1a;padding:48px;font-size:13px}
      h1{font-size:22px;margin:0 0 4px}table{width:100%;border-collapse:collapse;margin:24px 0}
      td,th{padding:6px 4px;border-bottom:1px solid #eee;text-align:left}
      .tot{text-align:right;margin-top:12px}.tot div{margin:2px 0}.g{font-weight:700;font-size:15px}</style></head>
      <body><p style="color:#888;margin:0">Zentria GmbH · Cosmi</p>
      <h1>${props.heading ?? 'RECHNUNG'}</h1>
      <p style="font-family:monospace">${props.number}</p>
      <p>${props.customerName}${props.customerAddress ? '<br>' + props.customerAddress : ''}</p>
      <p style="color:#888">${props.date}</p>
      <table><thead><tr><th>Menge</th><th>Beschreibung</th><th style="text-align:right">Betrag</th></tr></thead><tbody>${rows}</tbody></table>
      <div class="tot"><div>Netto: ${money(props.net)}</div><div>MwSt: ${money(props.tax)}</div><div class="g">Gesamt: ${money(props.gross)}</div></div>
      </body></html>`
    try {
      const iframe = document.createElement('iframe')
      iframe.style.cssText = 'position:fixed;right:0;bottom:0;width:0;height:0;border:0'
      document.body.appendChild(iframe)
      const doc = iframe.contentWindow?.document
      if (!doc) return
      doc.open(); doc.write(html); doc.close()
      iframe.contentWindow?.focus()
      window.setTimeout(() => {
        iframe.contentWindow?.print()
        window.setTimeout(() => document.body.removeChild(iframe), 1000)
      }, 250)
    } catch {
      /* Druck nicht verfügbar (z. B. headless) — Download bleibt der Weg zum PDF. */
    }
  }

  const Document = ({ scale }: { scale: number }) => (
    <div className="overflow-auto rounded-md border border-border bg-white" style={{ maxHeight: fullscreen ? '78vh' : 360 }}>
      <div
        className="mx-auto space-y-5 p-6 text-gray-800"
        style={{ width: 480, transform: `scale(${scale})`, transformOrigin: 'top center' }}
      >
        <div className="flex items-start justify-between">
          <div>
            <p className="text-[9px] text-gray-400">Zentria GmbH · Cosmi</p>
            <p className="text-[9px] text-gray-400">Bahnhofstraße 42</p>
            <p className="text-[9px] text-gray-400">8001 Zürich</p>
          </div>
          <div className="text-right">
            <p className="text-sm font-bold">{props.heading ?? 'RECHNUNG'}</p>
            <p className="font-mono text-[10px] text-gray-500">{props.number}</p>
            <p className="mt-1 text-[10px] text-gray-500">{props.date}</p>
          </div>
        </div>

        <div className="border-l-2 border-gray-300 pl-3">
          <p className="text-[9px] text-gray-400">{t('finanzen.pdf.invoiceTo')}:</p>
          <p className="text-[11px] font-medium text-gray-700">{props.customerName}</p>
          {props.customerAddress && <p className="text-[10px] text-gray-500">{props.customerAddress}</p>}
        </div>

        <table className="w-full text-[9px]">
          <thead>
            <tr className="border-b border-gray-200 text-gray-500">
              <th className="py-1 text-left font-medium">Menge</th>
              <th className="py-1 text-left font-medium">Beschreibung</th>
              <th className="py-1 text-right font-medium">Betrag</th>
            </tr>
          </thead>
          <tbody>
            {props.lineItems.map((it, i) => (
              <tr key={i} className="border-b border-gray-100">
                <td className="py-1 text-gray-600">{it.quantity}×</td>
                <td className="py-1 text-gray-700">{it.description}</td>
                <td className="py-1 text-right text-gray-700">{money(it.line_total)}</td>
              </tr>
            ))}
          </tbody>
        </table>

        <div className="flex justify-end">
          <div className="w-44 space-y-1 text-[10px]">
            <div className="flex justify-between text-gray-500"><span>Netto</span><span>{money(props.net)}</span></div>
            <div className="flex justify-between text-gray-500"><span>MwSt</span><span>{money(props.tax)}</span></div>
            <div className="flex justify-between border-t border-gray-200 pt-1 text-[11px] font-bold text-gray-800"><span>Gesamt</span><span>{money(props.gross)}</span></div>
          </div>
        </div>
      </div>
    </div>
  )

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{t('finanzen.pdf.preview', { defaultValue: 'PDF-Vorschau' })}</h4>
        <div className="flex items-center gap-1">
          <button onClick={() => setZoom((z) => Math.max(0.6, z - 0.2))} className="rounded p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground" title={t('finanzen.pdf.zoomOut', { defaultValue: 'Verkleinern' })}>
            <ZoomOut className="h-3.5 w-3.5" />
          </button>
          <span className="text-[10px] tabular-nums text-muted-foreground">{Math.round(zoom * 100)}%</span>
          <button onClick={() => setZoom((z) => Math.min(1.6, z + 0.2))} className="rounded p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground" title={t('finanzen.pdf.zoomIn', { defaultValue: 'Vergrößern' })}>
            <ZoomIn className="h-3.5 w-3.5" />
          </button>
          <span className="mx-0.5 text-border">|</span>
          <button onClick={() => setFullscreen(true)} className="rounded p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground" title={t('finanzen.pdf.fullscreen')}>
            <Maximize2 className="h-3.5 w-3.5" />
          </button>
          <button onClick={handlePrint} className="rounded p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground" title={t('finanzen.pdf.print')}>
            <Printer className="h-3.5 w-3.5" />
          </button>
          <button onClick={props.onDownload} className="rounded p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground" title={t('common.download')}>
            <Download className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      <Document scale={zoom} />

      <Dialog open={fullscreen} onOpenChange={setFullscreen}>
        <DialogContent className="max-w-2xl">
          <Document scale={1.2} />
        </DialogContent>
      </Dialog>
    </div>
  )
}
