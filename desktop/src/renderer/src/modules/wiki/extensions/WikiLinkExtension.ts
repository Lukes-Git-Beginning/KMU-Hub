/**
 * TipTap node extension for internal wiki links: `[[Article title]]`.
 *
 * Rendered as an atomic inline `<a data-wiki-link>` carrying the target article
 * id + slug as data attributes so it survives HTML serialisation and the
 * read-view DOMPurify pass. Clicks are handled by the host (edit mode via the
 * editor's handleClickOn, read mode via a delegated handler on the article body).
 */
import { Node, mergeAttributes } from '@tiptap/core'

export const WIKI_LINK_CLASS =
  'wiki-link rounded bg-primary/10 px-1 py-0.5 text-primary font-medium no-underline cursor-pointer hover:bg-primary/20'

export const WikiLink = Node.create({
  name: 'wikiLink',
  group: 'inline',
  inline: true,
  atom: true,
  selectable: true,

  addAttributes() {
    return {
      articleId: {
        default: null,
        parseHTML: (el) => el.getAttribute('data-article-id'),
        renderHTML: (attrs) =>
          attrs.articleId ? { 'data-article-id': attrs.articleId } : {},
      },
      slug: {
        default: null,
        parseHTML: (el) => el.getAttribute('data-slug'),
        renderHTML: (attrs) => (attrs.slug ? { 'data-slug': attrs.slug } : {}),
      },
      label: {
        default: '',
        parseHTML: (el) => el.textContent ?? '',
        renderHTML: () => ({}),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'a[data-wiki-link]' }]
  },

  renderHTML({ HTMLAttributes, node }) {
    return [
      'a',
      mergeAttributes(HTMLAttributes, {
        'data-wiki-link': '',
        class: WIKI_LINK_CLASS,
      }),
      String(node.attrs.label || node.attrs.slug || 'Artikel'),
    ]
  },

  renderText({ node }) {
    return `[[${node.attrs.label}]]`
  },
})

export default WikiLink
