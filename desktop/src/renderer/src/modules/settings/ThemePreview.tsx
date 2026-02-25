/**
 * ThemePreview — Live mini-app preview showing current theme combination.
 * Renders a small mockup matching the current nav layout + palette + dark mode
 * so every switch is reflected instantly.
 */
import { useUIStore, type ColorTheme } from '@/stores/ui'

const PALETTE_LABELS: Record<ColorTheme, string> = {
  graphit: 'Graphit',
  sand: 'Sand',
  ozean: 'Ozean',
}

export function ThemePreview({ className = '' }: { className?: string }) {
  const theme = useUIStore((s) => s.theme)
  const colorTheme = useUIStore((s) => s.colorTheme)
  const navLayout = useUIStore((s) => s.navLayout)
  const accentIntensity = useUIStore((s) => s.accentIntensity)

  const isDark = theme === 'dark' || (theme === 'auto' && window.matchMedia('(prefers-color-scheme: dark)').matches)

  const layoutLabel =
    navLayout === 'sidebar' ? 'Sidebar' :
    navLayout === 'dock' ? 'Dock' :
    navLayout === 'topnav' ? 'Top-Nav' : 'Kompakt'

  return (
    <div className={`rounded-xl border border-border overflow-hidden ${className}`}>
      {/* Label bar */}
      <div className="flex items-center justify-between px-4 py-2.5 bg-secondary/50 border-b border-border">
        <span className="text-xs font-medium text-foreground">Live-Vorschau</span>
        <div className="flex items-center gap-2">
          <span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
            {PALETTE_LABELS[colorTheme]}
          </span>
          <span className="inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
            {isDark ? 'Dunkel' : 'Hell'}
          </span>
          <span className="inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
            {layoutLabel}
          </span>
          {accentIntensity === 'vivid' && (
            <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium"
              style={{ background: 'var(--accent-1-light)', color: 'var(--accent-1)' }}>
              Lebendig
            </span>
          )}
        </div>
      </div>

      {/* Mini App Mockup */}
      <div className="flex h-44 relative" style={{ background: 'var(--background)' }}>
        {/* Sidebar (sidebar + classic layouts) */}
        {(navLayout === 'sidebar' || navLayout === 'classic') && (
          <div
            className="shrink-0 border-r flex flex-col items-center gap-2.5 py-3"
            style={{
              width: navLayout === 'classic' ? 40 : 56,
              background: 'var(--sidebar)',
              borderColor: 'var(--border)',
            }}
          >
            <div
              className="h-7 w-7 rounded-lg flex items-center justify-center text-[10px] font-bold"
              style={{ background: 'var(--primary)', color: 'var(--primary-foreground)' }}
            >
              K
            </div>
            {[0, 1, 2, 3].map((i) => (
              <div
                key={i}
                className="rounded-md transition-colors"
                style={{
                  width: navLayout === 'classic' ? 18 : 20,
                  height: navLayout === 'classic' ? 18 : 20,
                  background: i === 0 ? 'var(--sidebar-active)' : 'transparent',
                  border: i === 0 ? '1px solid var(--primary)' : '1px solid transparent',
                }}
              />
            ))}
          </div>
        )}

        {/* Main content area */}
        <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
          {/* Mini header (all layouts except dock get a full header) */}
          {navLayout !== 'dock' && (
            <div
              className="h-9 shrink-0 border-b flex items-center gap-2 px-3"
              style={{
                background: 'var(--header-background)',
                borderColor: 'var(--border)',
              }}
            >
              <div
                className="h-2.5 w-16 rounded-full"
                style={{ background: 'var(--foreground)', opacity: 0.7 }}
              />
              <div className="flex-1" />
              <div
                className="h-5 w-14 rounded-md text-[8px] font-medium flex items-center justify-center"
                style={{ background: 'var(--primary)', color: 'var(--primary-foreground)' }}
              >
                Aktion
              </div>
            </div>
          )}

          {/* Dock: mini top bar */}
          {navLayout === 'dock' && (
            <div
              className="h-7 shrink-0 border-b flex items-center gap-2 px-3"
              style={{
                background: 'var(--header-background)',
                borderColor: 'var(--border)',
              }}
            >
              <div className="h-2 w-12 rounded-full" style={{ background: 'var(--foreground)', opacity: 0.5 }} />
              <div className="flex-1" />
              <div className="h-3 w-3 rounded-full" style={{ background: 'var(--primary)', opacity: 0.6 }} />
            </div>
          )}

          {/* TopNav: tab strip below header */}
          {navLayout === 'topnav' && (
            <div
              className="h-6 shrink-0 border-b flex items-center gap-1.5 px-3"
              style={{
                background: 'var(--card)',
                borderColor: 'var(--border)',
                opacity: 0.8,
              }}
            >
              {['CRM', 'Mail', 'Proj.'].map((label, i) => (
                <div
                  key={label}
                  className="h-4 rounded-md px-1.5 flex items-center text-[6px] font-medium"
                  style={{
                    background: i === 0 ? 'var(--primary)' : 'transparent',
                    color: i === 0 ? 'var(--primary-foreground)' : 'var(--muted-foreground)',
                  }}
                >
                  {label}
                </div>
              ))}
            </div>
          )}

          {/* Mini content */}
          <div className="flex-1 p-2.5 overflow-hidden">
            {/* Stat cards row */}
            <div className="flex gap-2 mb-2.5">
              {[
                { label: 'Umsatz', value: '42.8k', color: 'var(--primary)' },
                { label: 'Kunden', value: '184', color: 'var(--accent-1)' },
                { label: 'Projekte', value: '12', color: 'var(--accent-2)' },
              ].map((stat) => (
                <div
                  key={stat.label}
                  className="flex-1 rounded-lg border p-1.5"
                  style={{
                    background: 'var(--card)',
                    borderColor: 'var(--border)',
                    boxShadow: 'var(--shadow-card)',
                  }}
                >
                  <p className="text-[7px] text-muted-foreground leading-none mb-0.5">{stat.label}</p>
                  <p className="text-[10px] font-semibold leading-none" style={{ color: stat.color }}>
                    {stat.value}
                  </p>
                </div>
              ))}
            </div>

            {/* Mini table */}
            <div
              className="rounded-lg border overflow-hidden"
              style={{
                background: 'var(--card)',
                borderColor: 'var(--border)',
                boxShadow: 'var(--shadow-card)',
              }}
            >
              <div
                className="flex items-center gap-2 px-2 py-1 border-b"
                style={{
                  background: 'var(--secondary)',
                  borderColor: 'var(--border)',
                }}
              >
                <div className="h-1.5 w-12 rounded-full bg-muted-foreground/40" />
                <div className="h-1.5 w-8 rounded-full bg-muted-foreground/40" />
                <div className="flex-1" />
                <div className="h-1.5 w-6 rounded-full bg-muted-foreground/40" />
              </div>
              {[0, 1, 2].map((i) => (
                <div
                  key={i}
                  className="flex items-center gap-2 px-2 py-1 border-b last:border-b-0"
                  style={{ borderColor: 'var(--border)' }}
                >
                  <div
                    className="h-4 w-4 rounded-full shrink-0"
                    style={{ background: i === 0 ? 'var(--primary)' : i === 1 ? 'var(--accent-1)' : 'var(--accent-2)', opacity: 0.7 }}
                  />
                  <div className="h-1.5 rounded-full bg-foreground/30" style={{ width: `${50 + i * 12}px` }} />
                  <div className="flex-1" />
                  <div
                    className="h-3.5 px-1.5 rounded-full text-[6px] font-medium flex items-center"
                    style={{
                      background: i === 0 ? 'var(--success-light)' : i === 1 ? 'var(--warning-light)' : 'var(--info-light)',
                      color: i === 0 ? 'var(--success)' : i === 1 ? 'var(--warning)' : 'var(--info)',
                    }}
                  >
                    {i === 0 ? 'Aktiv' : i === 1 ? 'Offen' : 'Neu'}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Dock: floating bottom bar */}
          {navLayout === 'dock' && (
            <div className="flex justify-center pb-2">
              <div
                className="flex items-center gap-2 rounded-full px-4 py-1.5 border"
                style={{
                  background: 'var(--card)',
                  borderColor: 'var(--border)',
                  boxShadow: 'var(--shadow-card)',
                }}
              >
                {[0, 1, 2, 3].map((i) => (
                  <div
                    key={i}
                    className="h-4 w-4 rounded-md"
                    style={{
                      background: i === 0 ? 'var(--primary)' : 'var(--muted-foreground)',
                      opacity: i === 0 ? 1 : 0.3,
                    }}
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
