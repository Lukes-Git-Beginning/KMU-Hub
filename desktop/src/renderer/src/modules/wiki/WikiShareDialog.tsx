import { useState } from 'react'
import { Link, Copy, Check, Globe, Lock, Users } from 'lucide-react'
import { toast } from 'sonner'
import type { WikiArticle } from '@/types/wiki'
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

const accessOptions = [
  { id: 'private', icon: Lock, label: 'Privat', description: 'Nur du kannst sehen und bearbeiten' },
  { id: 'team', icon: Users, label: 'Team', description: 'Alle Teammitglieder können sehen' },
  { id: 'public', icon: Globe, label: 'Öffentlich', description: 'Jeder im Unternehmen kann sehen' },
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
  const [access, setAccess] = useState('team')
  const [copied, setCopied] = useState(false)

  if (!article) return null

  const shareLink = `cosmi://wiki/${article.slug}`

  const handleCopy = () => {
    navigator.clipboard.writeText(shareLink).catch(() => {})
    setCopied(true)
    toast.success('Link kopiert')
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Artikel teilen</DialogTitle>
          <DialogDescription>
            Lege fest, wer "{article.title}" sehen kann.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Access levels */}
          <div className="space-y-1.5">
            {accessOptions.map((opt) => {
              const Icon = opt.icon
              const isActive = access === opt.id
              return (
                <button
                  key={opt.id}
                  onClick={() => setAccess(opt.id)}
                  className={`flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors ${
                    isActive
                      ? 'border-primary bg-primary/5'
                      : 'border-border hover:bg-accent'
                  }`}
                >
                  <Icon className={`h-4 w-4 shrink-0 ${isActive ? 'text-primary' : 'text-muted-foreground'}`} />
                  <div className="min-w-0">
                    <p className={`text-sm font-medium ${isActive ? 'text-primary' : 'text-foreground'}`}>{opt.label}</p>
                    <p className="text-[11px] text-muted-foreground">{opt.description}</p>
                  </div>
                </button>
              )
            })}
          </div>

          {/* Share link */}
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">Direktlink</label>
            <div className="flex gap-1.5">
              <div className="flex flex-1 items-center gap-2 rounded-md border border-border bg-secondary/50 px-3 py-1.5">
                <Link className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                <span className="text-xs text-foreground truncate">{shareLink}</span>
              </div>
              <button
                onClick={handleCopy}
                className="rounded-md border border-border px-2.5 py-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
              >
                {copied ? (
                  <Check className="h-3.5 w-3.5 text-success" />
                ) : (
                  <Copy className="h-3.5 w-3.5" />
                )}
              </button>
            </div>
          </div>

          {/* Done button */}
          <div className="flex justify-end">
            <button
              onClick={() => onOpenChange(false)}
              className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              Fertig
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
