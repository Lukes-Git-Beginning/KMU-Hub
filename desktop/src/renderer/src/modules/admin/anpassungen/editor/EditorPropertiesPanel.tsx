/**
 * EditorPropertiesPanel (Modul-Editor v1, E-2) — the right rail.
 *
 * E-2 renders the section intro + a "pick an element" empty state (this empty
 * state stays real in the final product — it's what you see before selecting a
 * specific field/term). E-3 mounts the actual trio editors (CustomFieldsTab /
 * BegriffeTab / ValueSetsTab, module-filtered + draft-wired) below the intro.
 */
import { useTranslation } from 'react-i18next'
import { MousePointerClick, Type, ListChecks, SquareStack } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { EditorSection } from './EditorTrioNav'

const SECTION_META: Record<EditorSection, { icon: LucideIcon; titleKey: string; descKey: string }> = {
  felder: {
    icon: SquareStack,
    titleKey: 'customization.editor.nav.fields',
    descKey: 'customization.editor.props.fieldsDesc',
  },
  begriffe: {
    icon: Type,
    titleKey: 'customization.editor.nav.terms',
    descKey: 'customization.editor.props.termsDesc',
  },
  wertelisten: {
    icon: ListChecks,
    titleKey: 'customization.editor.nav.valueSets',
    descKey: 'customization.editor.props.valueSetsDesc',
  },
}

export function EditorPropertiesPanel({
  section,
}: {
  section: EditorSection | null
}): React.ReactElement {
  const { t } = useTranslation()

  if (!section) {
    return (
      <aside className="flex w-[320px] shrink-0 flex-col border-l bg-background">
        <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <MousePointerClick className="h-5 w-5" aria-hidden="true" />
          </div>
          <p className="text-sm text-muted-foreground">{t('customization.editor.props.empty')}</p>
        </div>
      </aside>
    )
  }

  const meta = SECTION_META[section]
  const Icon = meta.icon

  return (
    <aside className="flex w-[320px] shrink-0 flex-col border-l bg-background">
      <div className="border-b px-4 py-3.5">
        <div className="flex items-center gap-2">
          <Icon className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
          <h3 className="text-sm font-semibold text-foreground">{t(meta.titleKey)}</h3>
        </div>
        <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{t(meta.descKey)}</p>
      </div>
      <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
        <p className="text-sm text-muted-foreground">{t('customization.editor.props.pickElement')}</p>
      </div>
    </aside>
  )
}
