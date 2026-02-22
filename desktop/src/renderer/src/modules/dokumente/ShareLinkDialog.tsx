/**
 * ShareLinkDialog — generate a shareable link for a file with options.
 *
 * Options: expiry date, password protection, permission (view/download).
 * Generates a mock link with copy-to-clipboard.
 */
import { useState, useCallback } from 'react'
import { Link2, Copy, Check, Shield, Calendar, RefreshCw, Eye, Download } from 'lucide-react'
import { toast } from 'sonner'

interface ShareLinkDialogProps {
  open: boolean
  onClose: () => void
  fileName: string
  fileId: string
}

function generateMockLink(fileId: string): string {
  const token = Math.random().toString(36).substring(2, 10)
  return `https://kmuhub.app/s/${fileId.substring(0, 8)}-${token}`
}

export function ShareLinkDialog({ open, onClose, fileName, fileId }: ShareLinkDialogProps) {
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
      toast.success('Link kopiert')
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error('Kopieren fehlgeschlagen')
    }
  }, [link])

  const handleRegenerate = () => {
    setLink(generateMockLink(fileId))
    setCopied(false)
    toast.success('Neuer Link generiert')
  }

  if (!open) return null

  const expiryDate = new Date()
  expiryDate.setDate(expiryDate.getDate() + parseInt(expiryDays || '7'))

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="mx-4 w-full max-w-md rounded-xl border border-border bg-card shadow-xl">
        {/* Header */}
        <div className="border-b border-border px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10">
              <Link2 className="h-4.5 w-4.5 text-primary" />
            </div>
            <div>
              <h2 className="text-sm font-semibold text-foreground">Link teilen</h2>
              <p className="text-xs text-muted-foreground truncate max-w-[280px]">{fileName}</p>
            </div>
          </div>
        </div>

        {/* Content */}
        <div className="px-6 py-4 space-y-4">
          {/* Generated link */}
          <div>
            <label className="text-xs font-medium text-muted-foreground mb-1.5 block">Freigabe-Link</label>
            <div className="flex gap-2">
              <input
                type="text"
                readOnly
                value={link}
                className="flex-1 rounded-lg border border-border bg-muted/50 px-3 py-2 text-sm text-foreground font-mono text-xs"
              />
              <button
                onClick={handleCopy}
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
                title="Neuen Link generieren"
              >
                <RefreshCw className="h-4 w-4" />
              </button>
            </div>
          </div>

          {/* Expiry */}
          <div>
            <label className="flex items-center gap-2 text-xs font-medium text-muted-foreground mb-1.5">
              <Calendar className="h-3.5 w-3.5" />
              Ablaufdatum
            </label>
            <select
              value={expiryDays}
              onChange={(e) => setExpiryDays(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
            >
              <option value="1">1 Tag</option>
              <option value="7">7 Tage</option>
              <option value="14">14 Tage</option>
              <option value="30">30 Tage</option>
              <option value="90">90 Tage</option>
              <option value="0">Kein Ablauf</option>
            </select>
            {expiryDays !== '0' && (
              <p className="text-[10px] text-muted-foreground mt-1">
                Laueft ab am {expiryDate.toLocaleDateString('de-DE')}
              </p>
            )}
          </div>

          {/* Permission */}
          <div>
            <label className="text-xs font-medium text-muted-foreground mb-1.5 block">Berechtigung</label>
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
                Nur ansehen
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
                Download erlaubt
              </button>
            </div>
          </div>

          {/* Password */}
          <div>
            <label className="flex items-center gap-2 text-xs font-medium text-muted-foreground mb-1.5">
              <Shield className="h-3.5 w-3.5" />
              Passwortschutz
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
                  placeholder="Passwort eingeben..."
                  className="flex-1 rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground"
                />
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 border-t border-border px-6 py-3">
          <button
            onClick={onClose}
            className="rounded-lg px-4 py-2 text-sm text-muted-foreground hover:bg-muted transition-colors"
          >
            Schliessen
          </button>
          <button
            onClick={() => {
              handleCopy()
              onClose()
            }}
            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            Link kopieren & schliessen
          </button>
        </div>
      </div>
    </div>
  )
}
