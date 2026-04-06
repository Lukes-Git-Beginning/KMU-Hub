import { useUIStore } from '@/stores/ui'
import type { NavLayout } from '@/stores/ui'
import { cn } from '@/lib'
import { useTranslation } from 'react-i18next'

interface LayoutSwitcherProps {
  className?: string
}

/** Mini SVG thumbnails for each layout */
function LayoutPreview({ layout, active }: { layout: NavLayout; active: boolean }) {
  const accent = active ? 'var(--primary)' : 'var(--muted-foreground)'
  const muted = active ? 'var(--primary-light, rgba(79,70,229,0.1))' : 'var(--muted)'
  const bg = 'var(--card)'

  return (
    <svg viewBox="0 0 80 56" className="w-full h-auto" fill="none">
      <rect x="0.5" y="0.5" width="79" height="55" rx="6" fill={bg} stroke={active ? accent : 'var(--border)'} strokeWidth="1" />
      {layout === 'sidebar' && (
        <>
          <rect x="2" y="2" width="18" height="52" rx="4" fill={muted} />
          <rect x="6" y="8" width="10" height="2" rx="1" fill={accent} />
          <rect x="6" y="14" width="10" height="2" rx="1" fill={accent} opacity="0.4" />
          <rect x="6" y="20" width="10" height="2" rx="1" fill={accent} opacity="0.4" />
          <rect x="24" y="6" width="50" height="6" rx="2" fill={muted} />
          <rect x="24" y="16" width="50" height="34" rx="2" fill={muted} opacity="0.5" />
        </>
      )}
      {layout === 'dock' && (
        <>
          <rect x="4" y="4" width="72" height="38" rx="3" fill={muted} opacity="0.5" />
          <rect x="16" y="46" width="48" height="8" rx="4" fill={muted} />
          <circle cx="28" cy="50" r="2.5" fill={accent} />
          <circle cx="38" cy="50" r="2.5" fill={accent} opacity="0.4" />
          <circle cx="48" cy="50" r="2.5" fill={accent} opacity="0.4" />
          <circle cx="58" cy="50" r="2.5" fill={accent} opacity="0.4" />
        </>
      )}
      {layout === 'topnav' && (
        <>
          <rect x="2" y="2" width="76" height="12" rx="4" fill={muted} />
          <rect x="8" y="6" width="12" height="4" rx="1.5" fill={accent} />
          <rect x="24" y="6" width="12" height="4" rx="1.5" fill={accent} opacity="0.4" />
          <rect x="40" y="6" width="12" height="4" rx="1.5" fill={accent} opacity="0.4" />
          <rect x="4" y="18" width="72" height="34" rx="2" fill={muted} opacity="0.5" />
        </>
      )}
      {layout === 'classic' && (
        <>
          <rect x="2" y="2" width="14" height="52" rx="3" fill={muted} />
          <rect x="5" y="8" width="8" height="2" rx="1" fill={accent} />
          <rect x="5" y="14" width="8" height="2" rx="1" fill={accent} opacity="0.4" />
          <rect x="5" y="20" width="8" height="2" rx="1" fill={accent} opacity="0.4" />
          <rect x="20" y="4" width="56" height="48" rx="2" fill={muted} opacity="0.5" />
        </>
      )}
    </svg>
  )
}

export function LayoutSwitcher({ className }: LayoutSwitcherProps) {
  const { t } = useTranslation()
  const navLayout = useUIStore((s) => s.navLayout)
  const setNavLayout = useUIStore((s) => s.setNavLayout)

  const layouts: Array<{ id: NavLayout; label: string; desc: string }> = [
    { id: 'sidebar', label: t('shared.layoutSwitcher.sidebar.label'), desc: t('shared.layoutSwitcher.sidebar.desc') },
    { id: 'dock', label: t('shared.layoutSwitcher.dock.label'), desc: t('shared.layoutSwitcher.dock.desc') },
    { id: 'topnav', label: t('shared.layoutSwitcher.topnav.label'), desc: t('shared.layoutSwitcher.topnav.desc') },
    { id: 'classic', label: t('shared.layoutSwitcher.classic.label'), desc: t('shared.layoutSwitcher.classic.desc') },
  ]

  return (
    <div className={cn('grid grid-cols-4 gap-3', className)}>
      {layouts.map((l) => {
        const active = navLayout === l.id
        return (
          <button
            key={l.id}
            onClick={() => setNavLayout(l.id)}
            className={cn(
              'group relative rounded-xl border-2 p-2 text-center transition-all duration-200',
              active
                ? 'border-primary ring-2 ring-primary/20 bg-primary-light'
                : 'border-border bg-card hover:border-muted-foreground/30'
            )}
          >
            <LayoutPreview layout={l.id} active={active} />
            <p className="mt-2 text-xs font-medium text-foreground">{l.label}</p>
            <p className="text-[10px] text-muted-foreground leading-tight">{l.desc}</p>
          </button>
        )
      })}
    </div>
  )
}
