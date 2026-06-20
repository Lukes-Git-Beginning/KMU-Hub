import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Copy, Check, Globe, Lock, Users, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import type { WikiArticle } from '@/types/wiki'
import type { WikiShareToken } from '@/api/wiki-types'
import { useCreateShareToken } from '@/api/hooks/useWiki'
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
  const [token, setToken] = useState<WikiShareToken | null>(null)
  const createShareToken = useCreateShareToken()

  // Reset the generated link when the dialog targets a different article.
  const articleId = article?.id
  useEffect(() => {
    setToken(null)
    setAccess('team')
    setCopied(false)
  }, [articleId])

  if (!article) return null

  const shareLink = token ? `cosmi://wiki/share/${token.token}` : ''

  const handleSelectAccess = (id: string) => {
    setAccess(id)
    // The link encodes the chosen permissions — invalidate it on change.
    setToken(null)
    setCopied(false)
  }

  const handleCreate = async () => {
    const opt = accessOptionKeys.find((o) => o.id === access)
    try {
      const result = await createShareToken.mutateAsync({
        articleId: article.id,
        permissions: opt?.permissions ?? ['read'],
      })
      setToken(result)
    } catch {
      toast.error(t('wiki.share.createError'))
    }
  }

  const handleCopy = () => {
    if (!shareLink) return
    navigator.clipboard.writeText(shareLink).catch(() => {})
    setCopied(true)
    toast.success(t('wiki.share.linkCopied'))
    setTimeout(() => setCopied(false), 2000)
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
          {/* Access levels */}
          <div className="space-y-1.5">
            {accessOptionKeys.map((opt) => {
              const Icon = opt.icon
              const isActive = access === opt.id
              return (
                <button
                  key={opt.id}
                  onClick={() => handleSelectAccess(opt.id)}
                  className={`flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors ${
                    isActive
                      ? 'border-primary bg-primary/5'
                      : 'border-border hover:bg-accent'
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
          </div>

          {/* Share link */}
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('wiki.share.directLink')}</label>
            {token ? (
              <div className="flex gap-1.5">
                <div className="flex flex-1 items-center gap-2 rounded-md border border-border bg-secondary/50 px-3 py-1.5">
                  <Link className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                  <span className="text-xs text-foreground truncate">{shareLink}</span>
                </div>
                <button
                  onClick={handleCopy}
                  className="rounded-md border border-border px-2.5 py-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
                  title={t('wiki.share.copy')}
                >
                  {copied ? (
                    <Check className="h-3.5 w-3.5 text-success" />
                  ) : (
                    <Copy className="h-3.5 w-3.5" />
                  )}
                </button>
              </div>
            ) : (
              <button
                onClick={handleCreate}
                disabled={createShareToken.isPending}
                className="flex w-full items-center justify-center gap-2 rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-accent transition-colors disabled:opacity-50"
              >
                {createShareToken.isPending ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Link className="h-3.5 w-3.5" />
                )}
                {createShareToken.isPending ? t('wiki.share.creating') : t('wiki.share.create')}
              </button>
            )}
          </div>

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
