import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Clock, Copy, Eye, Link2, Lock, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { ReportDocument } from '@/api/berichte-types'
import {
  useCreateReportShare,
  useReportShares,
  useRevokeReportShare,
} from '@/api/hooks/useBerichte'
import { formatDate } from '@/lib/format'

interface ShareLinkDialogProps {
  doc: ReportDocument
  open: boolean
  onClose: () => void
}

// The server caps every link at 365 days, so there is no "never" here.
type ExpiryChoice = '30' | '90' | '365'
const EXPIRY_CHOICES: ExpiryChoice[] = ['30', '90', '365']

/** Build the demo share URL for a token (real public host is Luke's gap). */
export function shareUrl(token: string): string {
  return `cosmi://share/report/${token}`
}

/**
 * R-5c / B6-5 — manage external read-only share links for a released report:
 * list active links (expiry, views, revoke) and mint new ones. The server keeps
 * only a hash of the token, so a new link is shown once, right after creation;
 * the list cannot hand it out again.
 * Demo-only — the actual unauthenticated public page is a backend gap (Luke).
 */
export function ShareLinkDialog({ doc, open, onClose }: ShareLinkDialogProps) {
  const { t } = useTranslation()
  const { data } = useReportShares(open ? doc.id : '')
  const create = useCreateReportShare()
  const revoke = useRevokeReportShare(doc.id)
  const [expiry, setExpiry] = useState<ExpiryChoice>('30')
  const [usePassword, setUsePassword] = useState(false)
  const [password, setPassword] = useState('')
  const [fresh, setFresh] = useState<{ id: string; url: string } | null>(null)
  const [copied, setCopied] = useState(false)

  const shares = data?.shares ?? []

  useEffect(() => {
    if (open) {
      setExpiry('30')
      setUsePassword(false)
      setPassword('')
      setFresh(null)
      setCopied(false)
    }
  }, [open])

  const handleCreate = () => {
    create.mutate(
      {
        documentId: doc.id,
        input: {
          expires_in_days: Number(expiry),
          password: usePassword ? password : undefined,
        },
      },
      {
        onSuccess: ({ share }) => {
          toast.success(t('berichte.docs.share.created'))
          setFresh(share.token ? { id: share.id, url: shareUrl(share.token) } : null)
          setCopied(false)
          setUsePassword(false)
          setPassword('')
        },
        onError: (err) => toast.error((err as Error).message),
      },
    )
  }

  const handleCopy = async (url: string) => {
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
      toast.success(t('berichte.docs.share.linkCopied'))
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error(t('berichte.docs.share.copyError'))
    }
  }

  const handleRevoke = (id: string) => {
    revoke.mutate(id, {
      onSuccess: () => {
        if (fresh?.id === id) setFresh(null)
        toast.success(t('berichte.docs.share.revoked'))
      },
      onError: (err) => toast.error((err as Error).message),
    })
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-lg gap-0 p-0">
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

        <div className="max-h-[70vh] space-y-5 overflow-y-auto px-6 py-5">
          {/* The link just made — the only moment it can be copied */}
          {fresh && (
            <div className="rounded-lg border border-primary/30 bg-primary-light px-3 py-3">
              <p className="mb-2 text-xs font-medium text-foreground">
                {t('berichte.docs.share.newLinkReady')}
              </p>
              <div className="flex items-center gap-2">
                <code className="min-w-0 flex-1 truncate rounded-md bg-card px-2 py-1.5 font-mono text-xs text-foreground">
                  {fresh.url}
                </code>
                <button
                  type="button"
                  onClick={() => handleCopy(fresh.url)}
                  className="flex shrink-0 items-center gap-1.5 rounded-md bg-primary px-2.5 py-1.5 text-xs text-primary-foreground transition-colors hover:bg-button-primary-hover"
                >
                  {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                  {t('berichte.docs.share.copy')}
                </button>
              </div>
              <p className="mt-2 text-[11px] text-muted-foreground">
                {t('berichte.docs.share.onceHint')}
              </p>
            </div>
          )}

          {/* Active links */}
          <div>
            <p className="mb-2 text-xs font-medium text-muted-foreground">
              {t('berichte.docs.share.active')}
            </p>
            {shares.length === 0 ? (
              <p className="rounded-lg border border-border bg-card px-4 py-3 text-xs text-muted-foreground">
                {t('berichte.docs.share.none')}
              </p>
            ) : (
              <ul className="space-y-2">
                {shares.map((s) => (
                  <li
                    key={s.id}
                    className="rounded-lg border border-border bg-card px-3 py-2.5"
                  >
                    <div className="flex items-center gap-2">
                      <Link2 className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                      <span className="min-w-0 flex-1 truncate text-xs text-foreground">
                        {t('berichte.docs.share.createdOn', { date: formatDate(s.created_at) })}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleRevoke(s.id)}
                        disabled={revoke.isPending}
                        aria-label={t('berichte.docs.share.revoke')}
                        className="shrink-0 rounded-md p-1 text-muted-foreground transition-colors hover:bg-error-light hover:text-error disabled:opacity-50"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                    <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 pl-5.5 text-[11px] text-muted-foreground">
                      <span className="inline-flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {s.expires_at
                          ? t('berichte.docs.share.expiresOn', { date: formatDate(s.expires_at) })
                          : t('berichte.docs.share.neverExpires')}
                      </span>
                      <span className="inline-flex items-center gap-1">
                        <Eye className="h-3 w-3" />
                        {t('berichte.docs.share.views', { count: s.view_count })}
                      </span>
                      {s.has_password && (
                        <span className="inline-flex items-center gap-1">
                          <Lock className="h-3 w-3" />
                          {t('berichte.docs.share.protected')}
                        </span>
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* Create a new link */}
          <div className="space-y-4 border-t border-border-muted pt-4">
            <p className="text-xs font-medium text-muted-foreground">
              {t('berichte.docs.share.newLink')}
            </p>
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

            <p className="rounded-lg border border-info/25 bg-info-light px-3 py-2 text-xs text-foreground">
              {t('berichte.docs.share.demoNote')}
            </p>
          </div>
        </div>

        <DialogFooter className="border-t border-border px-6 py-4">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary"
          >
            {t('berichte.docs.share.done')}
          </button>
          <button
            type="button"
            onClick={handleCreate}
            disabled={create.isPending || (usePassword && !password.trim())}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-60"
          >
            <Link2 className="h-4 w-4" />
            {t('berichte.docs.share.create')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
