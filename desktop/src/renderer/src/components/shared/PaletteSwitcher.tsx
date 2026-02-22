import { useUIStore } from '@/stores/ui'
import type { ColorTheme } from '@/stores/ui'
import { cn } from '@/lib'
import { Check } from 'lucide-react'

interface PaletteSwitcherProps {
  className?: string
}

const palettes: Array<{
  id: ColorTheme
  label: string
  desc: string
  colors: { primary: string; accent1: string; accent2: string; bg: string }
}> = [
  {
    id: 'graphit',
    label: 'Graphit',
    desc: 'Modern, klar, Indigo-Akzente',
    colors: { primary: '#4f46e5', accent1: '#f97316', accent2: '#10b981', bg: '#f4f4f6' },
  },
  {
    id: 'sand',
    label: 'Sand',
    desc: 'Warm, klassisch, Teal-Akzente',
    colors: { primary: '#1e7e74', accent1: '#d97706', accent2: '#0d9488', bg: '#e8e3dd' },
  },
  {
    id: 'ozean',
    label: 'Ozean',
    desc: 'Frisch, leicht, Cyan-Akzente',
    colors: { primary: '#0891b2', accent1: '#f43f5e', accent2: '#6366f1', bg: '#f0f7f7' },
  },
]

export function PaletteSwitcher({ className }: PaletteSwitcherProps) {
  const colorTheme = useUIStore((s) => s.colorTheme)
  const setColorTheme = useUIStore((s) => s.setColorTheme)

  return (
    <div className={cn('grid grid-cols-3 gap-3', className)}>
      {palettes.map((p) => {
        const active = colorTheme === p.id
        return (
          <button
            key={p.id}
            onClick={() => setColorTheme(p.id)}
            className={cn(
              'relative rounded-xl border-2 p-4 text-center transition-all duration-200',
              active
                ? 'border-primary ring-2 ring-primary/20 bg-primary-light'
                : 'border-border bg-card hover:border-muted-foreground/30'
            )}
          >
            {active && (
              <span className="absolute top-2 right-2">
                <Check className="h-4 w-4 text-primary" />
              </span>
            )}

            {/* Color circles */}
            <div className="flex items-center justify-center gap-1.5 mb-3">
              <span
                className="h-6 w-6 rounded-full ring-2 ring-white shadow-sm"
                style={{ backgroundColor: p.colors.primary }}
              />
              <span
                className="h-4 w-4 rounded-full ring-2 ring-white shadow-sm"
                style={{ backgroundColor: p.colors.accent1 }}
              />
              <span
                className="h-4 w-4 rounded-full ring-2 ring-white shadow-sm"
                style={{ backgroundColor: p.colors.accent2 }}
              />
            </div>

            {/* Background preview bar */}
            <div
              className="mx-auto mb-2.5 h-3 w-full rounded-full"
              style={{ backgroundColor: p.colors.bg }}
            />

            <p className="text-sm font-medium text-foreground">{p.label}</p>
            <p className="text-[10px] text-muted-foreground mt-0.5 leading-tight">{p.desc}</p>
          </button>
        )
      })}
    </div>
  )
}
