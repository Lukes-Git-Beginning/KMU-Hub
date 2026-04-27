import { useUIStore } from '@/stores/ui'
import type { ColorTheme } from '@/stores/ui'
import { cn } from '@/lib'
import { Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface PaletteSwitcherProps {
  className?: string
}

type PaletteColors = { primary: string; accent1: string; accent2: string; bg: string }

const paletteColors: Record<ColorTheme, PaletteColors> = {
  graphit:    { primary: '#1e7e74', accent1: '#f97316', accent2: '#10b981', bg: '#f4f4f6' },
  sand:       { primary: '#1e7e74', accent1: '#d97706', accent2: '#0d9488', bg: '#e8e3dd' },
  ozean:      { primary: '#0891b2', accent1: '#f43f5e', accent2: '#6366f1', bg: '#f0f7f7' },
  lavendel:   { primary: '#7c3aed', accent1: '#ec4899', accent2: '#06b6d4', bg: '#f3f0f9' },
  wald:       { primary: '#16a34a', accent1: '#ca8a04', accent2: '#0d9488', bg: '#f0f5f0' },
  rose:       { primary: '#e11d48', accent1: '#d946ef', accent2: '#f59e0b', bg: '#fdf2f4' },
  mitternacht:{ primary: '#2563eb', accent1: '#8b5cf6', accent2: '#14b8a6', bg: '#f0f4ff' },
  terrakotta: { primary: '#c2410c', accent1: '#b45309', accent2: '#059669', bg: '#faf5f0' },
}

export function PaletteSwitcher({ className }: PaletteSwitcherProps) {
  const { t } = useTranslation()
  const colorTheme = useUIStore((s) => s.colorTheme)
  const setColorTheme = useUIStore((s) => s.setColorTheme)

  const palettes: Array<{ id: ColorTheme; label: string; desc: string; colors: PaletteColors }> = [
    { id: 'graphit',     label: t('shared.paletteSwitcher.graphit.label'),     desc: t('shared.paletteSwitcher.graphit.desc'),     colors: paletteColors.graphit },
    { id: 'sand',        label: t('shared.paletteSwitcher.sand.label'),        desc: t('shared.paletteSwitcher.sand.desc'),        colors: paletteColors.sand },
    { id: 'ozean',       label: t('shared.paletteSwitcher.ozean.label'),       desc: t('shared.paletteSwitcher.ozean.desc'),       colors: paletteColors.ozean },
    { id: 'lavendel',    label: t('shared.paletteSwitcher.lavendel.label'),    desc: t('shared.paletteSwitcher.lavendel.desc'),    colors: paletteColors.lavendel },
    { id: 'wald',        label: t('shared.paletteSwitcher.wald.label'),        desc: t('shared.paletteSwitcher.wald.desc'),        colors: paletteColors.wald },
    { id: 'rose',        label: t('shared.paletteSwitcher.rose.label'),        desc: t('shared.paletteSwitcher.rose.desc'),        colors: paletteColors.rose },
    { id: 'mitternacht', label: t('shared.paletteSwitcher.mitternacht.label'), desc: t('shared.paletteSwitcher.mitternacht.desc'), colors: paletteColors.mitternacht },
    { id: 'terrakotta',  label: t('shared.paletteSwitcher.terrakotta.label'),  desc: t('shared.paletteSwitcher.terrakotta.desc'),  colors: paletteColors.terrakotta },
  ]

  return (
    <div className={cn('grid grid-cols-4 gap-3', className)}>
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
