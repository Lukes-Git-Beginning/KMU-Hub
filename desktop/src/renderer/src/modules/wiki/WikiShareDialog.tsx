import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Copy, Check, Globe, Lock, Users, Loader2, Trash2, ExternalLink, Clock } from 'lucide-react'
import { toast } from 'sonner'
import type { WikiArticle } from '@/types/wiki'
import { useShareTokens, useCreateShareToken, useRevokeShareToken } from '@/api/hooks/useWiki'
import { useWikiStore } from '@/stores/wiki'
import { useWikiSettingsStore } from '@/stores/wikiSettings'
import { useHasCapability } from '@/hooks/useCapability'
import { formatDate as libFormatDate } from '@/lib/format'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Access level options
// ---------------------------------------------------------------------------

const accessOptionKeys = [
  { id: 'private', icon: Lock, labelKey: 'wiki.share.accessPrivate', descKey: 'wiki.share.accessPrivateDesc', permissions: ['read'] },
  { id: 'team', icon: Users, labelKey: 'wiki.share.accessTeam', descKey: 'wiki.share.accessTeamDesc', permissions: ['read'] },
  { id: 'public', icon: Globe, labelKey: 'wiki.share.accessPublic', descKey: 'wiki.share.accessPublicDesc', permissions: ['read', 'public'] },
]

function tokenAccess(permissions: string[]): { icon: typeof Users; labelKey: string } {
  return permissions.includes('public')
    ? { icon: Globe, labelKey: 'wiki.share.accessPublic' }
    : { icon: Users, labelKey: 'wiki.share.accessTeam' }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiShareDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  article: WikiArticle | null
}

export function WikiShareDialog({ open, onOpenChange, article }: WikiShareDialogProps) {
  const { t } = useTranslation()
  const [access, setAccess] = useState('team')
  const [copied, setCopied] = useState(false)
  const shareDefault = useWikiSettingsStore((s) => s.shareDefault)
  const setSelectedArticle = useWikiStore((s) => s.setSelectedArticle)
  const canShareToken = useHasCapability('wiki:share_token:create')

  const articleId = article?.id ?? ''
  const { data: tokens = [], isLoading: tokensLoading } = useShareTokens(articleId)
  const createShareToken = useCreateShareToken()
  const revokeShareToken = useRevokeShareToken()

  useEffect(() => {
    setAccess(shareDefault === 'public' ? 'public' : 'team')
    setCopied(false)
  }, [articleId, shareDefault])

  if (!article) return null

  // Internal team reference — opens the article in the Cosmi app (no cosmi:// scheme).
  const internalLink = `${window.location.origin}/#/wiki?a=${article.id}`

  const handleCreate = async () => {
    const opt = accessOptionKeys.find((o) => o.id === access)
    try {
      await createShareToken.mutateAsync({
        articleId: article.id,
        permissions: opt?.permissions ?? ['read'],
      })
      toast.success(t('wiki.share.created'))
    } catch {
      toast.error(t('wiki.share.createError'))
    }
  }

  const handleRevoke = async (tokenId: string) => {
    try {
      await revokeShareToken.mutateAsync({ tokenId, articleId: article.id })
      toast.success(t('wiki.share.revoked'))
    } catch {
      toast.error(t('wiki.share.revokeError'))
    }
  }

  const handleCopy = () => {
    navigator.clipboard.writeText(internalLink).catch(() => {})
    setCopied(true)
    toast.success(t('wiki.share.linkCopied'))
    setTimeout(() => setCopied(false), 2000)
  }

  const handleOpenInApp = () => {
    setSelectedArticle(article.id)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('wiki.share.title')}</DialogTitle>
          <DialogDescription>
            {t('wiki.share.description', { title: article.title })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Internal link */}
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('wiki.share.internalLink')}</label>
            <div className="flex gap-1.5">
              <div className="flex flex-1 items-center gap-2 rounded-md border border-border bg-secondary/50 px-3 py-1.5">
                <Link className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                <span className="truncate text-xs text-foreground">{internalLink}</span>
              </div>
              <button
                onClick={handleCopy}
                className="rounded-md border border-border px-2.5 py-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
                title={t('wiki.share.copy')}
              >
                {copied ? <Check className="h-3.5 w-3.5 text-success" /> : <Copy className="h-3.5 w-3.5" />}
              </button>
            </div>
            <button
              onClick={handleOpenInApp}
              className="mt-1.5 inline-flex items-center gap-1 text-[11px] text-primary hover:underline"
            >
              <ExternalLink className="h-3 w-3" />
              {t('wiki.share.openInApp')}
            </button>
          </div>

          {/* Access level + token management — only with share_token:create */}
          {canShareToken && (
            <>
              <div className="space-y-1.5">
                <label className="block text-xs font-medium text-muted-foreground">{t('wiki.share.accessLevel')}</label>
                {accessOptionKeys.map((opt) => {
                  const Icon = opt.icon
                  const isActive = access === opt.id
                  return (
                    <button
                      key={opt.id}
                      onClick={() => setAccess(opt.id)}
                      className={`flex w-full items-center gap-3 rounded-lg border px-3 py-2 text-left transition-colors ${
                        isActive ? 'border-primary bg-primary/5' : 'border-border hover:bg-accent'
                      }`}
                    >
                      <Icon className={`h-4 w-4 shrink-0 ${isActive ? 'text-primary' : 'text-muted-foreground'}`} />
                      <div className="min-w-0">
                        <p className={`text-sm font-medium ${isActive ? 'text-primary' : 'text-foreground'}`}>{t(opt.labelKey)}</p>
                        <p className="text-[11px] text-muted-foreground">{t(opt.descKey)}</p>
                      </div>
                    </button>
                  )
                })}
                <button
                  onClick={handleCreate}
                  disabled={createShareToken.isPending}
                  className="flex w-full items-center justify-center gap-2 rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-accent transition-colors disabled:opacity-50"
                >
                  {createShareToken.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Link className="h-3.5 w-3.5" />}
                  {createShareToken.isPending ? t('wiki.share.creating') : t('wiki.share.create')}
                </button>
              </div>

              {/* Active share tokens */}
              <div>
                <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                  {t('wiki.share.activeTokens')}
                  <span className="ml-1 text-muted-foreground/60">({tokens.length})</span>
                </label>
                {tokensLoading ? (
                  <div className="space-y-1.5">
                    {[1, 2].map((i) => (
                      <div key={i} className="h-10 rounded-md bg-muted animate-pulse" />
                    ))}
                  </div>
                ) : tokens.length === 0 ? (
                  <p className="rounded-md border border-dashed border-border px-3 py-3 text-center text-[11px] text-muted-foreground">
                    {t('wiki.share.noTokens')}
                  </p>
                ) : (
                  <div className="space-y-1">
                    {tokens.map((tok) => {
                      const acc = tokenAccess(tok.permissions)
                      const Icon = acc.icon
                      const revoking =
                        revokeShareToken.isPending && revokeShareToken.variables?.tokenId === tok.id
                      return (
                        <div
                          key={tok.id}
                          className="group flex items-center gap-2.5 rounded-md border border-border bg-card/40 px-2.5 py-1.5"
                        >
                          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-info-light text-info">
                            <Icon className="h-3.5 w-3.5" />
                          </span>
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-xs font-medium text-foreground">{t(acc.labelKey)}</p>
                            <p className="flex items-center gap-1 text-[10px] text-muted-foreground">
                              <Clock className="h-2.5 w-2.5" />
                              {libFormatDate(tok.created_at)}
                            </p>
                          </div>
                          <button
                            onClick={() => handleRevoke(tok.id)}
                            disabled={revoking}
                            className="shrink-0 rounded p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-destructive disabled:opacity-100"
                            title={t('wiki.share.revoke')}
                            aria-label={t('wiki.share.revoke')}
                          >
                            {revoking ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
                          </button>
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            </>
          )}

          {/* Done button */}
          <div className="flex justify-end">
            <button
              onClick={() => onOpenChange(false)}
              className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              {t('wiki.share.done')}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
