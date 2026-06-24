/**
 * BrandingAdminHubTab — tenant branding (A-4).
 *
 * The dedicated, richer branding surface (extracted from the IT tab): workspace
 * name, logo, square icon, and an accent colour constrained to the Cosmi swatch
 * palette (no theme break). A live preview shows the result. Persisted to
 * localStorage as a mock; the real logo upload (→ S3) is Luke's track 🔒.
 */
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Upload, Check, RotateCcw, ImageOff } from 'lucide-react'
import { toast } from 'sonner'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { SWATCH_COLORS } from '@/lib/swatch-colors'

const LS = { name: 'cosmi:brand:name', logo: 'cosmi:brand:logo', icon: 'cosmi:brand:icon', accent: 'cosmi:brand:accent' }
const DEFAULT_ACCENT = '#10B981'
const MAX_BYTES = 512 * 1024

function read(key: string): string {
  try { return localStorage.getItem(key) ?? '' } catch { return '' }
}

export default function BrandingAdminHubTab() {
  const { t } = useTranslation()
  const logoRef = useRef<HTMLInputElement>(null)
  const iconRef = useRef<HTMLInputElement>(null)

  const [name, setName] = useState<string>(() => read(LS.name) || 'Cosmi')
  const [logo, setLogo] = useState<string>(() => read(LS.logo))
  const [icon, setIcon] = useState<string>(() => read(LS.icon))
  const [accent, setAccent] = useState<string>(() => read(LS.accent) || DEFAULT_ACCENT)

  const onUpload = (e: React.ChangeEvent<HTMLInputElement>, set: (v: string) => void) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    if (file.size > MAX_BYTES) { toast.error(t('admin.branding.fileTooLarge')); return }
    const reader = new FileReader()
    reader.onload = (ev) => set((ev.target?.result as string) ?? '')
    reader.readAsDataURL(file)
  }

  const save = () => {
    try {
      localStorage.setItem(LS.name, name.trim())
      localStorage.setItem(LS.logo, logo)
      localStorage.setItem(LS.icon, icon)
      localStorage.setItem(LS.accent, accent)
      document.documentElement.style.setProperty('--brand-accent', accent)
    } catch { /* QuotaExceeded — non-critical in demo */ }
    toast.success(t('admin.branding.saved'))
  }

  const reset = () => {
    setName('Cosmi'); setLogo(''); setIcon(''); setAccent(DEFAULT_ACCENT)
    try {
      Object.values(LS).forEach((k) => localStorage.removeItem(k))
      document.documentElement.style.removeProperty('--brand-accent')
    } catch { /* noop */ }
    toast.success(t('admin.branding.resetDone'))
  }

  return (
    <div className="px-6 py-6">
      <div className="mb-5">
        <h2 className="text-base font-semibold text-foreground">{t('admin.branding.title')}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">{t('admin.branding.subtitle')}</p>
      </div>

      <div className="grid gap-8 lg:grid-cols-2">
        {/* ── Form ── */}
        <div className="max-w-md space-y-6">
          {/* Workspace name */}
          <div className="space-y-1.5">
            <label htmlFor="brand-name" className="text-sm font-medium text-foreground">{t('admin.branding.name')}</label>
            <Input id="brand-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Cosmi" />
          </div>

          {/* Logo */}
          <UploadRow
            label={t('admin.branding.logo')}
            hint={t('admin.branding.logoHint')}
            preview={logo}
            square={false}
            onPick={() => logoRef.current?.click()}
            onRemove={logo ? () => setLogo('') : undefined}
            pickLabel={t('admin.branding.upload')}
            removeLabel={t('admin.branding.remove')}
          />
          <input ref={logoRef} type="file" accept="image/*" className="hidden" onChange={(e) => onUpload(e, setLogo)} />

          {/* Icon */}
          <UploadRow
            label={t('admin.branding.icon')}
            hint={t('admin.branding.iconHint')}
            preview={icon}
            square
            onPick={() => iconRef.current?.click()}
            onRemove={icon ? () => setIcon('') : undefined}
            pickLabel={t('admin.branding.upload')}
            removeLabel={t('admin.branding.remove')}
          />
          <input ref={iconRef} type="file" accept="image/*" className="hidden" onChange={(e) => onUpload(e, setIcon)} />

          {/* Accent colour */}
          <div className="space-y-2">
            <p className="text-sm font-medium text-foreground">{t('admin.branding.accent')}</p>
            <p className="text-xs text-muted-foreground">{t('admin.branding.accentHint')}</p>
            <div className="flex flex-wrap gap-2 pt-1">
              {SWATCH_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => setAccent(c)}
                  aria-label={c}
                  aria-pressed={accent.toLowerCase() === c.toLowerCase()}
                  className={`flex h-7 w-7 items-center justify-center rounded-full border transition-transform hover:scale-110 ${
                    accent.toLowerCase() === c.toLowerCase() ? 'border-foreground' : 'border-border'
                  }`}
                  style={{ backgroundColor: c }}
                >
                  {accent.toLowerCase() === c.toLowerCase() && <Check className="h-3.5 w-3.5 text-white" aria-hidden="true" />}
                </button>
              ))}
            </div>
          </div>

          {/* Actions */}
          <div className="flex gap-2 pt-2">
            <Button onClick={save}>{t('admin.branding.save')}</Button>
            <Button variant="outline" onClick={reset}>
              <RotateCcw className="mr-1.5 h-4 w-4" aria-hidden="true" />
              {t('admin.branding.reset')}
            </Button>
          </div>
        </div>

        {/* ── Live preview ── */}
        <div>
          <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">{t('admin.branding.preview')}</p>
          <BrandPreview name={name} logo={logo} icon={icon} accent={accent} appLabel={t('admin.branding.previewApp')} ctaLabel={t('admin.branding.previewCta')} />
        </div>
      </div>
    </div>
  )
}

function UploadRow({
  label, hint, preview, square, onPick, onRemove, pickLabel, removeLabel,
}: {
  label: string; hint: string; preview: string; square: boolean
  onPick: () => void; onRemove?: () => void; pickLabel: string; removeLabel: string
}) {
  return (
    <div className="space-y-1.5">
      <p className="text-sm font-medium text-foreground">{label}</p>
      <div className="flex items-center gap-4">
        <div
          className={`flex items-center justify-center overflow-hidden border border-border bg-secondary ${
            square ? 'h-14 w-14 rounded-xl' : 'h-14 w-24 rounded-lg'
          }`}
        >
          {preview ? (
            <img src={preview} alt="" className="h-full w-full object-contain" />
          ) : (
            <ImageOff className="h-5 w-5 text-muted-foreground/50" aria-hidden="true" />
          )}
        </div>
        <div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={onPick}>
              <Upload className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              {pickLabel}
            </Button>
            {onRemove && (
              <Button variant="ghost" size="sm" onClick={onRemove} className="text-muted-foreground">
                {removeLabel}
              </Button>
            )}
          </div>
          <p className="mt-1 text-[11px] text-muted-foreground">{hint}</p>
        </div>
      </div>
    </div>
  )
}

/** Faux app-shell mockup so branding choices read at a glance. */
function BrandPreview({
  name, logo, icon, accent, appLabel, ctaLabel,
}: {
  name: string; logo: string; icon: string; accent: string; appLabel: string; ctaLabel: string
}) {
  const initial = (name.trim()[0] ?? 'C').toUpperCase()
  return (
    <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
      {/* Top bar */}
      <div className="flex items-center gap-2.5 border-b border-border px-4 py-3">
        <div
          className="flex h-7 w-7 items-center justify-center overflow-hidden rounded-lg text-xs font-bold text-white"
          style={{ backgroundColor: accent }}
        >
          {icon ? <img src={icon} alt="" className="h-full w-full object-contain" /> : initial}
        </div>
        {logo ? (
          <img src={logo} alt="" className="h-5 max-w-[140px] object-contain" />
        ) : (
          <span className="truncate text-sm font-semibold text-foreground">{name || 'Cosmi'}</span>
        )}
      </div>
      {/* Body: faux sidebar + content */}
      <div className="flex">
        <div className="w-32 shrink-0 space-y-1 border-r border-border p-3">
          <div className="flex items-center gap-2 rounded-md px-2 py-1.5" style={{ backgroundColor: `${accent}1a` }}>
            <span className="h-2 w-2 rounded-full" style={{ backgroundColor: accent }} />
            <span className="text-xs font-medium" style={{ color: accent }}>{appLabel}</span>
          </div>
          <div className="flex items-center gap-2 px-2 py-1.5">
            <span className="h-2 w-2 rounded-full bg-muted-foreground/40" />
            <span className="h-2 w-12 rounded bg-secondary" />
          </div>
          <div className="flex items-center gap-2 px-2 py-1.5">
            <span className="h-2 w-2 rounded-full bg-muted-foreground/40" />
            <span className="h-2 w-10 rounded bg-secondary" />
          </div>
        </div>
        <div className="flex-1 space-y-3 p-4">
          <div className="h-2.5 w-24 rounded bg-secondary" />
          <div className="h-2 w-full rounded bg-secondary/70" />
          <div className="h-2 w-4/5 rounded bg-secondary/70" />
          <button
            type="button"
            className="mt-1 rounded-lg px-3 py-1.5 text-xs font-medium text-white"
            style={{ backgroundColor: accent }}
          >
            {ctaLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
