/**
 * Figure block (WP-2) — an image with an editorial caption.
 *
 * Editor: a React NodeView. Empty figures show an inline URL field; once a
 * source is set it renders the image plus an editable caption line.
 * Read view: serialised as `<figure><img><figcaption>…</figcaption></figure>`
 * (figure/figcaption allow-listed in the wiki sanitizer).
 */
import { useState } from 'react'
import { Node, mergeAttributes } from '@tiptap/core'
import { ReactNodeViewRenderer, NodeViewWrapper } from '@tiptap/react'
import type { NodeViewProps } from '@tiptap/react'
import { ImagePlus } from 'lucide-react'
import { useTranslation } from 'react-i18next'

function FigureView({ node, updateAttributes }: NodeViewProps) {
  const { t } = useTranslation()
  const src = (node.attrs.src as string) ?? ''
  const alt = (node.attrs.alt as string) ?? ''
  const caption = (node.attrs.caption as string) ?? ''
  const [draftUrl, setDraftUrl] = useState('')

  if (!src) {
    return (
      <NodeViewWrapper className="wiki-figure" data-figure="">
        <div className="wiki-figure-empty" contentEditable={false}>
          <ImagePlus className="h-6 w-6 text-muted-foreground/50" />
          <span className="text-xs text-muted-foreground">{t('wiki.block.image.hint')}</span>
          <div className="flex w-full max-w-md items-center gap-2">
            <input
              value={draftUrl}
              onChange={(e) => setDraftUrl(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && draftUrl.trim()) updateAttributes({ src: draftUrl.trim() })
              }}
              placeholder={t('wiki.block.image.urlPlaceholder')}
              className="h-8 min-w-0 flex-1 rounded-md border border-border bg-card px-2 text-xs outline-none focus:border-primary"
            />
            <button
              type="button"
              onClick={() => draftUrl.trim() && updateAttributes({ src: draftUrl.trim() })}
              disabled={!draftUrl.trim()}
              className="h-8 shrink-0 rounded-md bg-primary px-3 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
            >
              {t('wiki.block.image.insert')}
            </button>
          </div>
        </div>
      </NodeViewWrapper>
    )
  }

  return (
    <NodeViewWrapper className="wiki-figure" data-figure="">
      <img src={src} alt={alt} className="wiki-figure-img" />
      <input
        value={caption}
        onChange={(e) => updateAttributes({ caption: e.target.value })}
        placeholder={t('wiki.block.image.captionPlaceholder')}
        className="wiki-figure-caption-input"
        contentEditable={false}
        aria-label={t('wiki.block.image.captionLabel')}
      />
    </NodeViewWrapper>
  )
}

export const FigureImage = Node.create({
  name: 'figureImage',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      src: { default: '' },
      alt: { default: '' },
      caption: { default: '' },
    }
  },

  parseHTML() {
    return [{ tag: 'figure[data-figure]' }]
  },

  renderHTML({ node, HTMLAttributes }) {
    const children: unknown[] = [
      ['img', { src: node.attrs.src, alt: node.attrs.alt, class: 'wiki-figure-img' }],
    ]
    if (node.attrs.caption) {
      children.push(['figcaption', { class: 'wiki-figure-caption' }, node.attrs.caption])
    }
    return [
      'figure',
      mergeAttributes(HTMLAttributes, { 'data-figure': '', class: 'wiki-figure' }),
      ...children,
    ] as unknown as [string, Record<string, unknown>, ...unknown[]]
  },

  addNodeView() {
    return ReactNodeViewRenderer(FigureView)
  },
})

export default FigureImage
