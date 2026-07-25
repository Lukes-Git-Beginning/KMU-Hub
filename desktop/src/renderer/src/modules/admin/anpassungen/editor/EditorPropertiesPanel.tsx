/**
 * EditorPropertiesPanel (Modul-Editor v1, E-2) — the right rail.
 *
 * E-2 renders the section intro + a "pick an element" empty state (this empty
 * state stays real in the final product — it's what you see before selecting a
 * specific field/term). E-3 mounts the actual trio editors (CustomFieldsTab /
 * BegriffeTab / ValueSetsTab, module-filtered + draft-wired) below the intro.
 */
import { useTranslation } from 'react-i18next'
import { MousePointerClick, Type, ListChecks, SquareStack, Layers, MousePointer2, Copy, PanelLeft } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { EditorSection } from './EditorTrioNav'
import { BegriffePanel } from './BegriffePanel'
import { WertelistenPanel } from './WertelistenPanel'
import { FelderPanel } from './FelderPanel'
import { BereichePanel } from './BereichePanel'

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
  bereiche: {
    icon: Layers,
    titleKey: 'customization.editor.nav.areas',
    descKey: 'customization.editor.props.areasDesc',
  },
}

export function EditorPropertiesPanel({
  section,
  moduleKey,
}: {
  section: EditorSection | null
  moduleKey: string
}): React.ReactElement {
  const { t } = useTranslation()

  if (!section) {
    // Context inspector: edit-in-place is the primary interaction now, so the
    // default panel state teaches HOW to edit directly in the preview and points
    // to the left rail for the tools that aren't in-place (fields / value-sets /
    // areas). Selecting a section replaces this with that section's editor.
    const steps: { icon: LucideIcon; text: string }[] = [
      { icon: MousePointer2, text: t('customization.editor.inspector.step1') },
      { icon: Copy, text: t('customization.editor.inspector.step2') },
      { icon: PanelLeft, text: t('customization.editor.inspector.step3') },
    ]
    return (
      <aside className="flex w-[320px] shrink-0 flex-col border-l bg-background">
        <div className="border-b px-4 py-3.5">
          <div className="flex items-center gap-2">
            <MousePointerClick className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
            <h3 className="text-sm font-semibold text-foreground">{t('customization.editor.inspector.title')}</h3>
          </div>
          <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{t('customization.editor.inspector.subtitle')}</p>
        </div>
        <ol className="flex flex-col gap-3 px-4 py-4">
          {steps.map(({ icon: Icon, text }, i) => (
            <li key={i} className="flex items-start gap-3">
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-[var(--accent-1)]/10 text-[var(--accent-1)]">
                <Icon className="h-3.5 w-3.5" aria-hidden="true" />
              </span>
              <p className="pt-0.5 text-sm leading-relaxed text-muted-foreground">{text}</p>
            </li>
          ))}
        </ol>
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
      {section === 'begriffe' ? (
        <BegriffePanel moduleKey={moduleKey} />
      ) : section === 'wertelisten' ? (
        <WertelistenPanel moduleKey={moduleKey} />
      ) : section === 'felder' ? (
        <FelderPanel moduleKey={moduleKey} />
      ) : section === 'bereiche' ? (
        <BereichePanel moduleKey={moduleKey} />
      ) : (
        <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
          <p className="text-sm text-muted-foreground">{t('customization.editor.props.pickElement')}</p>
        </div>
      )}
    </aside>
  )
}
