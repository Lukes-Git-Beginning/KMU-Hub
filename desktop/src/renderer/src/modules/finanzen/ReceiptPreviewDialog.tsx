/**
 * ReceiptPreviewDialog — zeigt den an eine Ausgabe angehängten Beleg (finanzen P2).
 *
 * Gerade hochgeladene Belege liegen als Blob im Session-Cache (Bild → <img>,
 * PDF → <iframe>, beides per CSP-erlaubtem blob:). Seed-/serverseitige Belege
 * haben (noch) keinen Blob → Platzhalter mit Dateiname + Hinweis. Beim
 * Backend-Anbindung lädt die Vorschau die Beleg-URL vom Server.
 */
import { useTranslation } from 'react-i18next'
import { FileText } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { Expense } from '@/stores/finance'
import { getReceipt } from './lib/receipt-cache'

interface ReceiptPreviewDialogProps {
  expense: Expense | null
  onClose: () => void
}

export function ReceiptPreviewDialog({ expense, onClose }: ReceiptPreviewDialogProps) {
  const { t } = useTranslation()
  const cached = expense ? getReceipt(expense.id) : undefined
  const name = cached?.name ?? expense?.receiptName ?? ''
  const isImage = cached?.mimeType.startsWith('image/')

  return (
    <Dialog open={!!expense} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="truncate">{name || t('buchhaltung.form.receipt')}</DialogTitle>
        </DialogHeader>

        {cached ? (
          isImage ? (
            <img
              src={cached.url}
              alt={name}
              className="max-h-[60vh] w-full rounded-lg border border-border object-contain"
            />
          ) : (
            <iframe
              src={cached.url}
              title={name}
              className="h-[60vh] w-full rounded-lg border border-border"
            />
          )
        ) : (
          <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border bg-secondary/30 px-6 py-12 text-center">
            <FileText className="h-10 w-10 text-muted-foreground" aria-hidden="true" />
            <p className="text-sm font-medium text-foreground">{name}</p>
            <p className="max-w-xs text-xs text-muted-foreground">
              {t('buchhaltung.receiptPreview.serverNote')}
            </p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
