/**
 * Article identity (WP-3) — icon + cover for a wiki article.
 *
 * Icons are lucide names from a curated palette (Custom-SVG, never emojis); a
 * missing icon falls back to the title initial in a tinted badge. Covers are
 * either an image URL or a `grad:<id>` token resolved to a curated gradient, so
 * a premium banner works without an upload backend.
 */
import {
  BookOpen,
  FileText,
  Rocket,
  Lightbulb,
  Target,
  Users,
  Building2,
  Briefcase,
  ShieldCheck,
  Receipt,
  Wrench,
  GraduationCap,
  Megaphone,
  Compass,
  Star,
  Flag,
  Calendar,
  Database,
  type LucideIcon,
} from 'lucide-react'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Icon palette
// ---------------------------------------------------------------------------

export const WIKI_ICON_MAP: Record<string, LucideIcon> = {
  BookOpen,
  FileText,
  Rocket,
  Lightbulb,
  Target,
  Users,
  Building2,
  Briefcase,
  ShieldCheck,
  Receipt,
  Wrench,
  GraduationCap,
  Megaphone,
  Compass,
  Star,
  Flag,
  Calendar,
  Database,
}

export const WIKI_ICON_NAMES = Object.keys(WIKI_ICON_MAP)

// ---------------------------------------------------------------------------
// Cover gallery — curated gradients (premium, editorial, not generic purple)
// ---------------------------------------------------------------------------

export const WIKI_COVERS: { id: string; gradient: string }[] = [
  { id: 'aurora', gradient: 'linear-gradient(135deg, #4f46e5 0%, #0ea5e9 100%)' },
  { id: 'sunset', gradient: 'linear-gradient(135deg, #e11d48 0%, #f59e0b 100%)' },
  { id: 'ocean', gradient: 'linear-gradient(135deg, #0891b2 0%, #1e3a8a 100%)' },
  { id: 'forest', gradient: 'linear-gradient(135deg, #10b981 0%, #065f46 100%)' },
  { id: 'dusk', gradient: 'linear-gradient(135deg, #334155 0%, #1e1b4b 100%)' },
  { id: 'meadow', gradient: 'linear-gradient(135deg, #84cc16 0%, #16a34a 100%)' },
  { id: 'ember', gradient: 'linear-gradient(135deg, #f97316 0%, #b91c1c 100%)' },
  { id: 'slate', gradient: 'linear-gradient(135deg, #64748b 0%, #334155 100%)' },
]

const WIKI_COVER_MAP: Record<string, string> = Object.fromEntries(
  WIKI_COVERS.map((c) => [c.id, c.gradient]),
)

export type ResolvedCover =
  | { type: 'gradient'; value: string }
  | { type: 'image'; value: string }
  | { type: 'none' }

/** Map a stored cover value (image URL or `grad:<id>`) to a render descriptor. */
export function resolveCover(coverUrl?: string): ResolvedCover {
  if (!coverUrl) return { type: 'none' }
  if (coverUrl.startsWith('grad:')) {
    const grad = WIKI_COVER_MAP[coverUrl.slice(5)]
    return grad ? { type: 'gradient', value: grad } : { type: 'none' }
  }
  return { type: 'image', value: coverUrl }
}

/** The CSS background for a cover token, for inline style usage. */
export function coverBackground(coverUrl?: string): string | undefined {
  const c = resolveCover(coverUrl)
  if (c.type === 'gradient') return c.value
  if (c.type === 'image') return `center / cover no-repeat url("${c.value}")`
  return undefined
}

export function articleInitial(title: string): string {
  const ch = title.trim()[0]
  return ch ? ch.toUpperCase() : '?'
}

// ---------------------------------------------------------------------------
// Article icon badge — icon or title initial
// ---------------------------------------------------------------------------

type IconSize = 'sm' | 'md' | 'lg'

const SIZE_CLS: Record<IconSize, { box: string; icon: string; text: string }> = {
  sm: { box: 'h-5 w-5 rounded-md', icon: 'h-3 w-3', text: 'text-[10px]' },
  md: { box: 'h-9 w-9 rounded-lg', icon: 'h-5 w-5', text: 'text-sm' },
  lg: { box: 'h-14 w-14 rounded-xl', icon: 'h-7 w-7', text: 'text-xl' },
}

export function WikiArticleIcon({
  icon,
  title,
  size = 'md',
  className,
}: {
  icon?: string
  title: string
  size?: IconSize
  className?: string
}) {
  const cls = SIZE_CLS[size]
  const Icon = icon ? WIKI_ICON_MAP[icon] : undefined

  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center justify-center bg-primary/10 font-semibold text-primary',
        cls.box,
        className,
      )}
      aria-hidden="true"
    >
      {Icon ? (
        <Icon className={cls.icon} />
      ) : (
        <span className={cls.text}>{articleInitial(title)}</span>
      )}
    </span>
  )
}
