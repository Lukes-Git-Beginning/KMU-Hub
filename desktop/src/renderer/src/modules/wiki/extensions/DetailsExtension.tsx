/**
 * Details / Toggle block (WP-2) — a collapsible section with an editable summary
 * line and a rich body.
 *
 * Editor: a React NodeView shows a chevron toggle, an editable summary input and
 * the body (NodeViewContent), which collapses when closed.
 * Read view: serialised as native `<details><summary>…</summary>…</details>` so
 * the browser handles the toggle for free (no JS, DOMPurify just needs the two
 * tags + the `open` attribute allow-listed).
 */
import { useState } from 'react'
import { Node, mergeAttributes } from '@tiptap/core'
import { ReactNodeViewRenderer, NodeViewWrapper, NodeViewContent } from '@tiptap/react'
import type { NodeViewProps } from '@tiptap/react'
import { ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

function DetailsView({ node, updateAttributes }: NodeViewProps) {
  const { t } = useTranslation()
  const summary = (node.attrs.summary as string) ?? ''
  // The editor keeps its own open state so the body stays reachable for editing
  // regardless of how the document was saved.
  const [open, setOpen] = useState<boolean>(node.attrs.open !== false)

  const toggle = () => {
    const next = !open
    setOpen(next)
    updateAttributes({ open: next })
  }

  return (
    <NodeViewWrapper className="wiki-details" data-details="" data-open={open ? 'true' : 'false'}>
      <div className="wiki-details-head" contentEditable={false}>
        <button
          type="button"
          onClick={toggle}
          className={`wiki-details-toggle${open ? ' is-open' : ''}`}
          aria-label={open ? t('wiki.block.toggle.collapse') : t('wiki.block.toggle.expand')}
        >
          <ChevronRight className="h-4 w-4" />
        </button>
        <input
          value={summary}
          onChange={(e) => updateAttributes({ summary: e.target.value })}
          placeholder={t('wiki.block.toggle.summaryPlaceholder')}
          className="wiki-details-summary-input"
          aria-label={t('wiki.block.toggle.summaryLabel')}
        />
      </div>
      <NodeViewContent className={`wiki-details-body${open ? '' : ' is-hidden'}`} />
    </NodeViewWrapper>
  )
}

export const DetailsBlock = Node.create({
  name: 'detailsBlock',
  group: 'block',
  content: 'block+',
  defining: true,

  addAttributes() {
    return {
      summary: {
        default: '',
        parseHTML: (el) => el.querySelector('summary')?.textContent ?? '',
        renderHTML: () => ({}),
      },
      open: {
        default: true,
        parseHTML: (el) => el.hasAttribute('open'),
        renderHTML: () => ({}),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'details[data-details]' }]
  },

  renderHTML({ node, HTMLAttributes }) {
    const attrs = mergeAttributes(HTMLAttributes, {
      'data-details': '',
      class: 'wiki-details',
    }) as Record<string, unknown>
    if (node.attrs.open) attrs.open = 'open'
    return [
      'details',
      attrs,
      ['summary', { class: 'wiki-details-summary' }, node.attrs.summary || ''],
      ['div', { 'data-details-content': '', class: 'wiki-details-body' }, 0],
    ]
  },

  addNodeView() {
    return ReactNodeViewRenderer(DetailsView)
  },
})

export default DetailsBlock
