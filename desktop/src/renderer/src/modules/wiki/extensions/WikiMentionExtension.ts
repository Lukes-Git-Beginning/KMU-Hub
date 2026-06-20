/**
 * TipTap node extension for @mentions of team members.
 *
 * Patterned after the chat MentionAutocomplete UX, but as a TipTap atomic
 * inline node so the mention is a single non-editable token. Rendered as
 * `<span data-mention-id>@Name</span>`; the id survives serialisation and the
 * read-view sanitiser.
 */
import { Node, mergeAttributes } from '@tiptap/core'

export const WIKI_MENTION_CLASS =
  'wiki-mention rounded bg-info-light px-1 py-0.5 text-info font-medium'

export const WikiMention = Node.create({
  name: 'wikiMention',
  group: 'inline',
  inline: true,
  atom: true,
  selectable: true,

  addAttributes() {
    return {
      userId: {
        default: null,
        parseHTML: (el) => el.getAttribute('data-mention-id'),
        renderHTML: (attrs) =>
          attrs.userId ? { 'data-mention-id': attrs.userId } : {},
      },
      label: {
        default: '',
        parseHTML: (el) => (el.textContent ?? '').replace(/^@/, ''),
        renderHTML: () => ({}),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'span[data-mention-id]' }]
  },

  renderHTML({ HTMLAttributes, node }) {
    return [
      'span',
      mergeAttributes(HTMLAttributes, { class: WIKI_MENTION_CLASS }),
      `@${String(node.attrs.label || '')}`,
    ]
  },

  renderText({ node }) {
    return `@${node.attrs.label}`
  },
})

export default WikiMention
