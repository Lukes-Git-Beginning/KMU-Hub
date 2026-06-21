/**
 * wiki block registry — wires the shared document engine into the wiki module.
 *
 * Phase B: the core blocks, but the text + heading blocks are overridden with
 * frameless, page-like editors so a wiki article reads as flowing long-form
 * prose rather than a stack of boxed cards (PB-2). The special elements
 * (callout, image, divider) keep their subtle frames from the core registry.
 *
 * Wiki-specific blocks (toggle, code, simple table, attachment) land in the next
 * batch as additional definitions appended here — the engine and every other
 * surface stay untouched.
 */
import { useTranslation } from 'react-i18next'
import { Heading1, Heading2 } from 'lucide-react'
import { RichTextEditor } from '@/components/shared/RichTextEditor'
import {
  buildRegistry,
  createCoreBlockDefs,
  type BlockEditProps,
  type BlockRegistry,
  type BlockTypeDef,
  type HeadingBlock,
  type TextBlock,
} from '@/components/shared/document'

/* ------------------------- frameless long-form text ----------------------- */

function WikiTextEdit({ block, onPatch }: BlockEditProps<TextBlock>) {
  const { t } = useTranslation()
  return (
    <RichTextEditor
      frameless
      content={block.html}
      onChange={(html) => onPatch({ html })}
      minHeight="1.75rem"
      placeholder={t('document.block.text.placeholder', { defaultValue: 'Text schreiben…' })}
    />
  )
}

/* ---------------------------- frameless heading --------------------------- */

function WikiHeadingEdit({ block, onPatch }: BlockEditProps<HeadingBlock>) {
  const { t } = useTranslation()
  const isH1 = block.level === 1
  return (
    <div className="group/wh relative">
      <input
        value={block.text}
        onChange={(e) => onPatch({ text: e.target.value })}
        placeholder={t('document.block.heading.placeholder', { defaultValue: 'Überschrift' })}
        className={`w-full border-0 bg-transparent px-0 text-foreground placeholder:text-muted-foreground/40 focus:outline-none ${
          isH1
            ? 'report-serif text-[1.9rem] font-semibold leading-tight tracking-tight'
            : 'text-xl font-semibold tracking-tight'
        }`}
      />
      {/* Level toggle — hidden until the heading is hovered or focused. */}
      <div className="absolute -top-2 right-0 hidden items-center overflow-hidden rounded-md border border-border bg-card shadow-sm group-hover/wh:flex group-focus-within/wh:flex">
        {([1, 2] as const).map((lvl) => {
          const Icon = lvl === 1 ? Heading1 : Heading2
          return (
            <button
              key={lvl}
              type="button"
              onClick={() => onPatch({ level: lvl })}
              className={`p-1 ${block.level === lvl ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'}`}
              aria-label={`H${lvl}`}
            >
              <Icon className="h-3.5 w-3.5" />
            </button>
          )
        })}
      </div>
    </div>
  )
}

/* -------------------------------- registry -------------------------------- */

// Core blocks, keyed so we can override text + heading and order the menu.
const core = Object.fromEntries(createCoreBlockDefs().map((d) => [d.type, d])) as Record<
  string,
  BlockTypeDef
>

// Keep each core block's icon/label/factory/read-view; swap in the frameless edit.
const wikiText: BlockTypeDef = { ...core.text, Edit: WikiTextEdit as BlockTypeDef['Edit'] }
const wikiHeading: BlockTypeDef = { ...core.heading, Edit: WikiHeadingEdit as BlockTypeDef['Edit'] }

/**
 * Insert-menu order: prose first (the writer's default), then structure and the
 * special inline elements. Columns/layout come from the engine itself.
 */
export const wikiBlockRegistry: BlockRegistry = buildRegistry([
  wikiText,
  wikiHeading,
  core.bullet,
  core.callout,
  core.image,
  core.divider,
])
