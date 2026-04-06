/**
 * ShareLinkDialog — generate a shareable link for a file with options.
 *
 * Options: expiry date, password protection, permission (view/download).
 * Generates a mock link with copy-to-clipboard.
 */
import { useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Link2, Copy, Check, Shield, Calendar, RefreshCw, Eye, Download } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'

interface ShareLinkDialogProps {
  open: boolean
  onClose: () => void
  fileName: string
  fileId: string
}

function generateMockLink(fileId: string): string {
  const token = Math.random().toString(36).substring(2, 10)
  return `https://cosmi.zentria.tech/s/${fileId.substring(0, 8)}-${token}`
}

export function ShareLinkDialog({ open, onClose, fileName, fileId }: ShareLinkDialogProps) {
  const { t } = useTranslation()
  const [link, setLink] = useState(() => generateMockLink(fileId))
  const [copied, setCopied] = useState(false)
  const [expiryDays, setExpiryDays] = useState<string>('7')
  const [passwordEnabled, setPasswordEnabled] = useState(false)
  const [password, setPassword] = useState('')
  const [permission, setPermission] = useState<'view' | 'download'>('view')

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(link)
      setCopied(true)
      toast.success(t('dokumente.shareLink.linkCopied'))
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error(t('dokumente.shareLink.copyFailed'))
    }
  }, [link])

  const handleRegenerate = () => {
    setLink(generateMockLink(fileId))
    setCopied(false)
    toast.success(t('dokumente.shareLink.linkRegenerated'))
  }

   
  const expiryDate = useMemo(() => {
    const d = new Date()
    d.setDate(d.getDate() + parseInt(expiryDays || '7'))
    return d
  }, [expiryDays])

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-md">
        {/* Header */}
        <DialogHeader className="border-b border-border px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10">
              <Link2 className="h-4.5 w-4.5 text-primary" />
            </div>
            <div>
              <DialogTitle className="text-sm font-semibold text-foreground">{t('dokumente.shareLink.title')}</DialogTitle>
              <DialogDescription className="text-xs text-muted-foreground truncate max-w-[280px]">{fileName}</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        {/* Content */}
        <div className="px-6 py-4 space-y-4">
          {/* Generated link */}
          <div>
            <label className="text-xs font-medium text-muted-foreground mb-1.5 block">{t('dokumente.shareLink.shareLink')}</label>
            <div className="flex gap-2">
              <input
                type="text"
                readOnly
                value={link}
                className="flex-1 rounded-lg border border-border bg-muted/50 px-3 py-2 text-sm text-foreground font-mono text-xs"
              />
              <button
                onClick={handleCopy}
                aria-label={t('dokumente.shareLink.copyLinkAria')}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm font-medium hover:bg-muted transition-colors"
              >
                {copied ? (
                  <Check className="h-4 w-4 text-green-500" />
                ) : (
                  <Copy className="h-4 w-4 text-muted-foreground" />
                )}
              </button>
              <button
                onClick={handleRegenerate}
                className="rounded-lg border border-border p-2 text-muted-foreground hover:bg-muted transition-colors"
                aria-label={t('dokumente.shareLink.regenerateAria')}
              >
                <RefreshCw className="h-4 w-4" />
              </button>
            </div>
          </div>

          {/* Expiry */}
          <div>
            <label className="flex items-center gap-2 text-xs font-medium text-muted-foreground mb-1.5">
              <Calendar className="h-3.5 w-3.5" />
              {t('dokumente.shareLink.expiryDate')}
            </label>
            <select
              value={expiryDays}
              onChange={(e) => setExpiryDays(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
            >
              <option value="1">{t('dokumente.shareLink.expiry1Day')}</option>
              <option value="7">{t('dokumente.shareLink.expiry7Days')}</option>
              <option value="14">{t('dokumente.shareLink.expiry14Days')}</option>
              <option value="30">{t('dokumente.shareLink.expiry30Days')}</option>
              <option value="90">{t('dokumente.shareLink.expiry90Days')}</option>
              <option value="0">{t('dokumente.shareLink.expiryNone')}</option>
            </select>
            {expiryDays !== '0' && (
              <p className="text-[10px] text-muted-foreground mt-1">
                {t('dokumente.shareLink.expiresOn', { date: expiryDate.toLocaleDateString('de-DE') })}
              </p>
            )}
          </div>

          {/* Permission */}
          <div>
            <label className="text-xs font-medium text-muted-foreground mb-1.5 block">{t('dokumente.shareLink.permission')}</label>
            <div className="flex gap-2">
              <button
                onClick={() => setPermission('view')}
                className={`flex flex-1 items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors ${
                  permission === 'view'
                    ? 'border-primary bg-primary/10 text-primary font-medium'
                    : 'border-border text-muted-foreground hover:bg-muted'
                }`}
              >
                <Eye className="h-4 w-4" />
                {t('dokumente.shareLink.viewOnly')}
              </button>
              <button
                onClick={() => setPermission('download')}
                className={`flex flex-1 items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors ${
                  permission === 'download'
                    ? 'border-primary bg-primary/10 text-primary font-medium'
                    : 'border-border text-muted-foreground hover:bg-muted'
                }`}
              >
                <Download className="h-4 w-4" />
                {t('dokumente.shareLink.downloadAllowed')}
              </button>
            </div>
          </div>

          {/* Password */}
          <div>
            <label className="flex items-center gap-2 text-xs font-medium text-muted-foreground mb-1.5">
              <Shield className="h-3.5 w-3.5" />
              {t('dokumente.shareLink.passwordProtection')}
            </label>
            <div className="flex items-center gap-3">
              <button
                onClick={() => setPasswordEnabled(!passwordEnabled)}
                className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                  passwordEnabled ? 'bg-primary' : 'bg-muted'
                }`}
              >
                <span
                  className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                    passwordEnabled ? 'translate-x-4.5' : 'translate-x-0.5'
                  }`}
                />
              </button>
              {passwordEnabled && (
                <input
                  type="text"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t('dokumente.shareLink.passwordPlaceholder')}
                  className="flex-1 rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground"
                />
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <DialogFooter className="border-t border-border px-6 py-3">
          <button
            onClick={onClose}
            className="rounded-lg px-4 py-2 text-sm text-muted-foreground hover:bg-muted transition-colors"
          >
            {t('common.close')}
          </button>
          <button
            onClick={() => {
              handleCopy()
              onClose()
            }}
            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            {t('dokumente.shareLink.copyAndClose')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
