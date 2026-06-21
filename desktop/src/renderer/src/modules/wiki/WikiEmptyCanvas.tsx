/**
 * Empty article state (WP-5) — a small joy moment for a blank page. Custom SVG
 * (no emoji): a sheet with placeholder lines and an accent spark, warm wording,
 * and a single clear call to action.
 */
import { useTranslation } from 'react-i18next'
import { PenLine } from 'lucide-react'

export function WikiEmptyCanvas({ onStart }: { onStart: () => void }) {
  const { t } = useTranslation()

  return (
    <div className="flex flex-col items-center justify-center gap-4 py-16 text-center">
      <svg viewBox="0 0 120 120" className="h-28 w-28" role="img" aria-hidden="true">
        {/* soft glow */}
        <circle cx="58" cy="60" r="46" className="fill-primary/[0.06]" />
        {/* page */}
        <rect
          x="36"
          y="30"
          width="46"
          height="60"
          rx="6"
          className="fill-card stroke-border"
          strokeWidth="1.5"
        />
        {/* placeholder text lines */}
        <g className="stroke-muted-foreground/30" strokeWidth="2.5" strokeLinecap="round">
          <line x1="45" y1="45" x2="73" y2="45" />
          <line x1="45" y1="54" x2="68" y2="54" />
          <line x1="45" y1="63" x2="73" y2="63" />
          <line x1="45" y1="72" x2="61" y2="72" />
        </g>
        {/* accent spark — a fresh start */}
        <path
          d="M88 34 l2.4 -6 l2.4 6 l6 2.4 l-6 2.4 l-2.4 6 l-2.4 -6 l-6 -2.4 z"
          className="fill-primary"
        />
      </svg>

      <div className="space-y-1">
        <p className="text-base font-semibold text-foreground">{t('wiki.empty.title')}</p>
        <p className="mx-auto max-w-xs text-sm text-muted-foreground">{t('wiki.empty.hint')}</p>
      </div>

      <button
        onClick={onStart}
        className="mt-1 inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
      >
        <PenLine className="h-4 w-4" />
        {t('wiki.article.startWriting')}
      </button>
    </div>
  )
}
