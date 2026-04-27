/**
 * ITAdminHubTab — Admin-Hub IT-Tab.
 *
 * Oben: Tenant-Konfiguration (zentraler vs. verteilter Modus).
 * Dann: Branding (Logo + Akzent-Farbe).
 * Dann: Bestehender ITAdminTab-Inhalt (E-Mail, Sicherheits-Policy, Berechtigungen, Signatur).
 *
 * Die Settings-Variante (settings/tabs/ITAdminTab) wird in Phase 4 gelöscht.
 * Bis dahin bleibt sie als Backwards-Compat-Stub.
 */
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ITAdminTab } from '@/modules/settings/tabs/ITAdminTab'

function readLocalStorage(key: string): string {
  try { return localStorage.getItem(key) ?? '' } catch { return '' }
}

export default function ITAdminHubTab() {
  const { t } = useTranslation()
  const logoInputRef = useRef<HTMLInputElement>(null)

  // Logo preview: reads from localStorage on mount, updated after upload
  const [logoPreview, setLogoPreview] = useState<string>(() => readLocalStorage('cosmi:brand:logo'))

  // Accent color: reads from localStorage on mount
  const [savedAccent] = useState<string>(() => readLocalStorage('cosmi:brand:accent'))

  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = (ev) => {
      const b64 = ev.target?.result as string
      try {
        localStorage.setItem('cosmi:brand:logo', b64)
      } catch {
        // QuotaExceededError — non-critical
      }
      setLogoPreview(b64)
      toast.success(t('admin.it.branding.logoUploaded'))
    }
    reader.readAsDataURL(file)
  }

  const handleAccentColor = (e: React.ChangeEvent<HTMLInputElement>) => {
    document.documentElement.style.setProperty('--brand-accent', e.target.value)
    try {
      localStorage.setItem('cosmi:brand:accent', e.target.value)
    } catch {
      // Non-critical
    }
  }

  const handleAccentReset = () => {
    document.documentElement.style.removeProperty('--brand-accent')
    try {
      localStorage.removeItem('cosmi:brand:accent')
    } catch {
      // Non-critical
    }
    toast.success(t('admin.it.branding.accentResetConfirm'))
  }

  return (
    <div className="px-6 py-6 max-w-3xl space-y-10">
      {/* ── Branding ─────────────────────────────────── */}
      <section>
        <h2 className="text-sm font-semibold text-foreground mb-0.5">
          {t('admin.it.branding.title')}
        </h2>
        <p className="text-xs text-muted-foreground mb-4">
          {t('admin.it.branding.subtitle')}
        </p>

        <div className="space-y-4">
          {/* Logo upload */}
          <div className="flex items-center gap-4">
            <div className="flex h-14 w-14 items-center justify-center rounded-lg border border-border bg-secondary overflow-hidden">
              {logoPreview ? (
                <img src={logoPreview} alt="Firmen-Logo" className="h-full w-full object-contain" />
              ) : (
                <span className="text-xs text-muted-foreground">Logo</span>
              )}
            </div>
            <div>
              <input
                ref={logoInputRef}
                type="file"
                accept="image/*"
                onChange={handleLogoUpload}
                className="hidden"
              />
              <button
                onClick={() => logoInputRef.current?.click()}
                className="rounded-md border border-border bg-card px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary transition-colors"
              >
                {t('admin.it.branding.logoUpload')}
              </button>
              <p className="text-[10px] text-muted-foreground mt-1">PNG, SVG empfohlen · max. 512 KB</p>
            </div>
          </div>

          {/* Accent color */}
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <input
                type="color"
                defaultValue={savedAccent || '#1e7e74'}
                onChange={handleAccentColor}
                className="h-11 w-11 cursor-pointer rounded border border-border bg-transparent"
                aria-label={t('admin.it.branding.accentColor')}
              />
              <div>
                <p className="text-sm font-medium text-foreground">{t('admin.it.branding.accentColor')}</p>
                <p className="text-xs text-muted-foreground">Setzt CSS-Variable --brand-accent global</p>
              </div>
            </div>
            <button
              onClick={handleAccentReset}
              className="ml-auto rounded-md border border-border px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
            >
              {t('admin.it.branding.accentReset')}
            </button>
          </div>
        </div>
      </section>

      <div className="border-t border-border" />

      {/* ── Bestehender IT-Admin-Inhalt ───────────────── */}
      <section>
        <ITAdminTab />
      </section>
    </div>
  )
}
