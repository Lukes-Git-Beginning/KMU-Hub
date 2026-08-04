/**
 * EditorTrioNav (Modul-Editor v1, E-2) — the left rail: the three customization
 * dimensions of the v1 trio (Felder / Begriffe / Wertelisten). Each row shows a
 * live change-count badge sourced from the draft session. Selecting a section
 * drives the properties panel on the right. E-3 fills each section with the real
 * editor content.
 */
import { useTranslation } from 'react-i18next'
import { Type, ListChecks, SquareStack, Layers, BarChart3, Inbox } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib'
import { useDraftConfig } from './DraftConfigProvider'
import { getEditorModule } from './editorModules'

import type { EditorFocusSection } from '@/components/customization/EditorSurface'

/** The rail's sections. Defined in EditorSurface so modules can react to the
 *  selected one (useEditorFocusEffect) without importing from the editor. */
export type EditorSection = EditorFocusSection

interface SectionDef {
  key: EditorSection
  labelKey: string
  icon: LucideIcon
  /** When set, the section only shows for modules where this capability is true. */
  requires?: (moduleKey: string) => boolean
}

const SECTIONS: SectionDef[] = [
  { key: 'felder', labelKey: 'customization.editor.nav.fields', icon: SquareStack },
  { key: 'begriffe', labelKey: 'customization.editor.nav.terms', icon: Type },
  { key: 'wertelisten', labelKey: 'customization.editor.nav.valueSets', icon: ListChecks },
  { key: 'bereiche', labelKey: 'customization.editor.nav.areas', icon: Layers },
  { key: 'statistik', labelKey: 'customization.editor.nav.statistics', icon: BarChart3 },
  {
    key: 'kanäle',
    labelKey: 'customization.editor.nav.channels',
    icon: Inbox,
    requires: (moduleKey) => !!getEditorModule(moduleKey)?.intake,
  },
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
  // Stat-widget toggles live in moduleAreas under a `stat:` prefix — split them
  // out so the Bereiche and Statistik badges each count only their own keys.
  const areaKeys = Object.keys(moduleAreas[moduleKey] ?? {})
  const counts: Record<EditorSection, number> = {
    felder: Object.keys(customFields).length,
    begriffe: labelCount,
    wertelisten: Object.keys(valueSets).length,
    bereiche: areaKeys.filter((k) => !k.startsWith('stat:')).length,
    statistik: areaKeys.filter((k) => k.startsWith('stat:')).length,
    // Channels are a functional tenant toggle (not a draft dimension) → no badge.
    'kanäle': 0,
  }

  const sections = SECTIONS.filter((s) => !s.requires || s.requires(moduleKey))

  return (
    <nav
      aria-label={t('customization.editor.nav.label')}
      className="flex w-[240px] shrink-0 flex-col gap-0.5 border-r bg-muted/30 p-2"
    >
      <p className="px-2 pb-1.5 pt-1 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        {t('customization.editor.nav.label')}
      </p>
      {sections.map(({ key, labelKey, icon: Icon }) => {
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
