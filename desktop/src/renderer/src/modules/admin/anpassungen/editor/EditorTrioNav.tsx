/**
 * EditorTrioNav (Modul-Editor v1, E-2) — the left rail: the three customization
 * dimensions of the v1 trio (Felder / Begriffe / Wertelisten). Each row shows a
 * live change-count badge sourced from the draft session. Selecting a section
 * drives the properties panel on the right. E-3 fills each section with the real
 * editor content.
 */
import { useTranslation } from 'react-i18next'
import { Type, ListChecks, SquareStack, Layers } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib'
import { useDraftConfig } from './DraftConfigProvider'

export type EditorSection = 'felder' | 'begriffe' | 'wertelisten' | 'bereiche'

interface SectionDef {
  key: EditorSection
  labelKey: string
  icon: LucideIcon
}

const SECTIONS: SectionDef[] = [
  { key: 'felder', labelKey: 'customization.editor.nav.fields', icon: SquareStack },
  { key: 'begriffe', labelKey: 'customization.editor.nav.terms', icon: Type },
  { key: 'wertelisten', labelKey: 'customization.editor.nav.valueSets', icon: ListChecks },
  { key: 'bereiche', labelKey: 'customization.editor.nav.areas', icon: Layers },
]

export function EditorTrioNav({
  active,
  onSelect,
}: {
  active: EditorSection | null
  onSelect: (section: EditorSection) => void
}): React.ReactElement {
  const { t } = useTranslation()
  const { labels, valueSets, customFields, moduleAreas, moduleKey } = useDraftConfig()

  const labelCount = Object.values(labels).reduce((acc, m) => acc + Object.keys(m).length, 0)
  const counts: Record<EditorSection, number> = {
    felder: Object.keys(customFields).length,
    begriffe: labelCount,
    wertelisten: Object.keys(valueSets).length,
    bereiche: Object.keys(moduleAreas[moduleKey] ?? {}).length,
  }

  return (
    <nav
      aria-label={t('customization.editor.nav.label')}
      className="flex w-[240px] shrink-0 flex-col gap-0.5 border-r bg-muted/30 p-2"
    >
      <p className="px-2 pb-1.5 pt-1 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        {t('customization.editor.nav.label')}
      </p>
      {SECTIONS.map(({ key, labelKey, icon: Icon }) => {
        const isActive = active === key
        const count = counts[key]
        return (
          <button
            key={key}
            type="button"
            onClick={() => onSelect(key)}
            aria-current={isActive ? 'true' : undefined}
            className={cn(
              'group flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm transition-colors',
              isActive
                ? 'bg-background font-medium text-foreground shadow-sm'
                : 'text-muted-foreground hover:bg-background/60 hover:text-foreground',
            )}
          >
            <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span className="min-w-0 flex-1 truncate text-left">{t(labelKey)}</span>
            {count > 0 && (
              <span className="shrink-0 rounded-full bg-amber-500/15 px-1.5 text-[11px] font-semibold tabular-nums text-amber-600 dark:text-amber-400">
                {count}
              </span>
            )}
          </button>
        )
      })}
    </nav>
  )
}
