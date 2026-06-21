/**
 * Callout block (WP-2) — an emphasised note box with four variants
 * (info / warning / tip / recommendation), mirroring the berichte CALLOUT_STYLE
 * palette but wiki-appropriate.
 *
 * Editor: a React NodeView renders the variant icon + an inline variant switcher
 * and an editable body (NodeViewContent).
 * Read view: serialised as `<div data-callout data-variant="…">…</div>`; the
 * shared `.wiki-callout` CSS colours it and draws the icon via a CSS mask, so no
 * SVG ever has to pass through DOMPurify.
 */
import { Node, mergeAttributes } from '@tiptap/core'
import { ReactNodeViewRenderer, NodeViewWrapper, NodeViewContent } from '@tiptap/react'
import type { NodeViewProps } from '@tiptap/react'
import { Info, AlertTriangle, Lightbulb, Sparkles, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export type CalloutVariant = 'info' | 'warning' | 'tip' | 'recommendation'

export const CALLOUT_VARIANTS: { variant: CalloutVariant; icon: LucideIcon }[] = [
  { variant: 'info', icon: Info },
  { variant: 'warning', icon: AlertTriangle },
  { variant: 'tip', icon: Lightbulb },
  { variant: 'recommendation', icon: Sparkles },
]

function CalloutView({ node, updateAttributes }: NodeViewProps) {
  const { t } = useTranslation()
  const variant = (node.attrs.variant as CalloutVariant) ?? 'info'

  // The variant icon is drawn by CSS (.wiki-callout::before mask) so it stays
  // identical in editor and read view — here we only render the switcher chrome.
  return (
    <NodeViewWrapper className="wiki-callout" data-variant={variant} data-callout="">
      <div className="wiki-callout-switch" contentEditable={false}>
        {CALLOUT_VARIANTS.map(({ variant: v, icon: VIcon }) => (
          <button
            key={v}
            type="button"
            onClick={() => updateAttributes({ variant: v })}
            className={`wiki-callout-switch-btn${v === variant ? ' is-active' : ''}`}
            data-variant={v}
            title={t(`wiki.block.callout.${v}`)}
            aria-label={t(`wiki.block.callout.${v}`)}
          >
            <VIcon className="h-3.5 w-3.5" />
          </button>
        ))}
      </div>
      <NodeViewContent className="wiki-callout-body" />
    </NodeViewWrapper>
  )
}

export const Callout = Node.create({
  name: 'callout',
  group: 'block',
  content: 'block+',
  defining: true,

  addAttributes() {
    return {
      variant: {
        default: 'info',
        parseHTML: (el) => el.getAttribute('data-variant') ?? 'info',
        renderHTML: (attrs) => ({ 'data-variant': attrs.variant }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-callout]' }]
  },

  renderHTML({ HTMLAttributes }) {
    return [
      'div',
      mergeAttributes(HTMLAttributes, { 'data-callout': '', class: 'wiki-callout' }),
      0,
    ]
  },

  addNodeView() {
    return ReactNodeViewRenderer(CalloutView)
  },
})

export default Callout
