import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Copy, Link2, Lock } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { ReportDocument, ReportShareToken } from '@/api/berichte-types'
import { useCreateReportShare } from '@/api/hooks/useBerichte'

interface ShareLinkDialogProps {
  doc: ReportDocument
  open: boolean
  onClose: () => void
}

type ExpiryChoice = '30' | '90' | 'never'
const EXPIRY_CHOICES: ExpiryChoice[] = ['30', '90', 'never']

/** Build the demo share URL for a token (real public host is Luke's gap). */
export function shareUrl(token: string): string {
  return `cosmi://share/report/${token}`
}

/**
 * R-5c — generate an external read-only share link for a released report.
 * Demo-only: the token is stored FE-side; the actual unauthenticated public
 * page is a backend gap (Luke).
 */
export function ShareLinkDialog({ doc, open, onClose }: ShareLinkDialogProps) {
  const { t } = useTranslation()
  const create = useCreateReportShare()
  const [expiry, setExpiry] = useState<ExpiryChoice>('30')
  const [usePassword, setUsePassword] = useState(false)
  const [password, setPassword] = useState('')
  const [created, setCreated] = useState<ReportShareToken | null>(null)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (open) {
      setExpiry('30')
      setUsePassword(false)
      setPassword('')
      setCreated(null)
      setCopied(false)
    }
  }, [open])

  const handleCreate = () => {
    create.mutate(
      {
        documentId: doc.id,
        input: {
          expires_in_days: expiry === 'never' ? null : Number(expiry),
          password: usePassword ? password : undefined,
        },
      },
      {
        onSuccess: (res) => setCreated(res.share),
        onError: (err) => toast.error((err as Error).message),
      },
    )
  }

  const handleCopy = async () => {
    if (!created) return
    try {
      await navigator.clipboard.writeText(shareUrl(created.token))
      setCopied(true)
      toast.success(t('berichte.docs.share.linkCopied'))
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error(t('berichte.docs.share.copyError'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-md gap-0 p-0">
        <DialogHeader className="border-b border-border px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
              <Link2 className="h-4 w-4 text-primary" />
            </div>
            <div className="min-w-0">
              <DialogTitle className="text-sm">{t('berichte.docs.share.linkTitle')}</DialogTitle>
              <DialogDescription className="truncate">{doc.title}</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        {created ? (
          <div className="space-y-4 px-6 py-5">
            <p className="text-xs text-muted-foreground">{t('berichte.docs.share.linkReady')}</p>
            <div className="flex items-center gap-2 rounded-lg border border-border bg-secondary/40 px-3 py-2">
              <Link2 className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
              <code className="min-w-0 flex-1 truncate font-mono text-xs text-foreground">
                {shareUrl(created.token)}
              </code>
              <button
                type="button"
                onClick={handleCopy}
                className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
              >
                {copied ? (
                  <Check className="h-3 w-3 text-success" />
                ) : (
                  <Copy className="h-3 w-3" />
                )}
                {copied ? t('berichte.docs.share.copied') : t('berichte.docs.share.copy')}
              </button>
            </div>
            <p className="rounded-lg border border-info/25 bg-info-light px-3 py-2 text-xs text-foreground">
              {t('berichte.docs.share.demoNote')}
            </p>
          </div>
        ) : (
          <div className="space-y-5 px-6 py-5">
            {/* Expiry */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t('berichte.docs.share.expiry')}
              </label>
              <div className="flex gap-2">
                {EXPIRY_CHOICES.map((c) => (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setExpiry(c)}
                    className={`flex-1 rounded-lg border px-3 py-2 text-xs transition-colors ${
                      expiry === c
                        ? 'border-primary bg-primary-light text-primary'
                        : 'border-border text-muted-foreground hover:bg-secondary'
                    }`}
                  >
                    {t(`berichte.docs.share.expiry_${c}`)}
                  </button>
                ))}
              </div>
            </div>

            {/* Password */}
            <div>
              <label className="flex items-center justify-between">
                <span className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                  <Lock className="h-3 w-3" />
                  {t('berichte.docs.share.password')}
                </span>
                <button
                  type="button"
                  onClick={() => setUsePassword((v) => !v)}
                  aria-pressed={usePassword}
                  aria-label={t('berichte.docs.share.password')}
                  className={`flex h-5 w-9 items-center rounded-full p-0.5 transition-colors ${
                    usePassword ? 'bg-primary' : 'bg-secondary'
                  }`}
                >
                  <span
                    className={`h-4 w-4 rounded-full bg-white shadow transition-transform ${
                      usePassword ? 'translate-x-4' : 'translate-x-0'
                    }`}
                  />
                </button>
              </label>
              {usePassword && (
                <input
                  type="text"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t('berichte.docs.share.passwordPlaceholder')}
                  className="mt-2 w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              )}
            </div>
          </div>
        )}

        <DialogFooter className="border-t border-border px-6 py-4">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary"
          >
            {created ? t('berichte.docs.share.done') : t('berichte.docs.share.cancel')}
          </button>
          {!created && (
            <button
              type="button"
              onClick={handleCreate}
              disabled={create.isPending || (usePassword && !password.trim())}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-60"
            >
              <Link2 className="h-4 w-4" />
              {t('berichte.docs.share.create')}
            </button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
